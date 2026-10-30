package httpx

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"pulse-backend/internal/platform/apperr"
)

// responseRecorder captures the status code for the request log.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// Hijack is required for WebSocket upgrades: embedding hides the underlying
// writer's http.Hijacker implementation, so it is forwarded explicitly.
func (r *responseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("responseRecorder: underlying ResponseWriter does not support hijacking")
	}
	return hijacker.Hijack()
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("%s %s -> %d (%s)", r.Method, r.URL.Path, rec.statusCode, time.Since(start).Round(time.Microsecond))
	})
}

// Recover turns a panic in a handler into a 500 instead of killing the connection.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler {
					panic(v)
				}
				log.Printf("panic in %s %s: %v", r.Method, r.URL.Path, v)
				WriteJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// OriginPolicy decides which browser origins may call the API and open the
// WebSocket. Patterns are exact origins ("https://pulseponics.vercel.app"),
// globs ("https://*.vercel.app"), or "*" for any origin.
type OriginPolicy struct {
	patterns []string
}

func NewOriginPolicy(patterns []string) *OriginPolicy {
	return &OriginPolicy{patterns: patterns}
}

func (p *OriginPolicy) Allowed(origin string) bool {
	if origin == "" {
		return false
	}
	for _, pattern := range p.patterns {
		if pattern == "*" || pattern == origin {
			return true
		}
		if strings.Contains(pattern, "*") {
			if ok, _ := path.Match(pattern, origin); ok {
				return true
			}
		}
	}
	return false
}

// CORS answers preflight requests and adds CORS headers for allowed origins.
// Auth uses the Authorization header (no cookies), so credentials mode is off.
func (p *OriginPolicy) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		preflight := r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != ""

		if origin != "" {
			w.Header().Add("Vary", "Origin")
		}
		if p.Allowed(origin) {
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			if preflight {
				h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Device-Key")
				h.Set("Access-Control-Max-Age", "600")
			}
		}
		if preflight {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ClientIP returns the caller's IP. Behind Caddy (trustProxy = true) that is
// the right-most X-Forwarded-For entry, which Caddy appends itself and a
// client cannot forge.
func ClientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if ip := strings.TrimSpace(parts[len(parts)-1]); ip != "" {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// RateLimiter is a per-IP token bucket, used on the auth endpoints to slow
// down password guessing. State is in memory, which is fine for one instance.
type RateLimiter struct {
	mu         sync.Mutex
	visitors   map[string]*visitor
	limit      rate.Limit
	burst      int
	trustProxy bool
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func NewRateLimiter(every time.Duration, burst int, trustProxy bool) *RateLimiter {
	rl := &RateLimiter{
		visitors:   make(map[string]*visitor),
		limit:      rate.Every(every),
		burst:      burst,
		trustProxy: trustProxy,
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	v, ok := rl.visitors[ip]
	if !ok {
		v = &visitor{limiter: rate.NewLimiter(rl.limit, rl.burst)}
		rl.visitors[ip] = v
	}
	v.lastSeen = time.Now()
	return v.limiter.Allow()
}

func (rl *RateLimiter) cleanup() {
	for range time.Tick(time.Minute) {
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > 10*time.Minute {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Limit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !rl.allow(ClientIP(r, rl.trustProxy)) {
			WriteError(w, r, apperr.TooManyRequests("too many requests, try again in a minute"))
			return
		}
		next(w, r)
	}
}
