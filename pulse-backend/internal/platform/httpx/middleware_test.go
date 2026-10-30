package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOriginPolicy(t *testing.T) {
	p := NewOriginPolicy([]string{"http://localhost:5173", "https://pulseponics*.vercel.app"})
	cases := map[string]bool{
		"http://localhost:5173":                   true,
		"https://pulseponics.vercel.app":          true,
		"https://pulseponics-git-main.vercel.app": true,
		"https://evil.vercel.app":                 false,
		"https://pulseponics.vercel.app.evil.com": false,
		"http://localhost:3000":                   false,
		"":                                        false,
	}
	for origin, want := range cases {
		if got := p.Allowed(origin); got != want {
			t.Errorf("Allowed(%q) = %v, want %v", origin, got, want)
		}
	}
}

func TestCORSPreflight(t *testing.T) {
	p := NewOriginPolicy([]string{"https://app.example.com"})
	h := p.CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("preflight must not reach the handler")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/devices", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Error("allowed origin not echoed")
	}
	if rec.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Error("allow-headers missing")
	}
}

func TestClientIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "172.18.0.5:41234"
	req.Header.Set("X-Forwarded-For", "6.6.6.6, 203.0.113.9")

	if got := ClientIP(req, false); got != "172.18.0.5" {
		t.Errorf("untrusted proxy: got %s", got)
	}
	if got := ClientIP(req, true); got != "203.0.113.9" {
		t.Errorf("trusted proxy should use the right-most entry: got %s", got)
	}
}
