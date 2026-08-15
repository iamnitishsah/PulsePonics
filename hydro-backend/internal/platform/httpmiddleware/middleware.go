package httpmiddleware

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

// responseRecorder wraps http.ResponseWriter so we can capture the status
// code that was written — the stdlib ResponseWriter doesn't expose this
// after the fact, so we intercept WriteHeader() to record it ourselves.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// Hijack forwards to the underlying ResponseWriter's Hijack method, if it
// has one. This is required for WebSocket upgrades to work: gorilla/
// websocket needs to take over the raw TCP connection from the HTTP
// server, which it does via the http.Hijacker interface. Without this
// method, wrapping the ResponseWriter (as we do for logging) hides that
// capability even though the underlying writer supports it — Go only sees
// the interface methods responseRecorder itself declares, not ones it
// inherits transitively through an embedded field's dynamic type.
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

		duration := time.Since(start)
		log.Printf("%s %s -> %d (%s)", r.Method, r.URL.Path, rec.statusCode, duration)
	})
}
