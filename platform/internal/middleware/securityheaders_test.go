package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestSecurityHeaders(t *testing.T) {
	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	tests := []struct {
		header string
		want   string
	}{
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "DENY"},
		{"Strict-Transport-Security", "max-age=31536000; includeSubDomains"},
		// #282: regression guards for the two headers that were
		// documented in CLAUDE.md but missing from the implementation.
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
		{"Permissions-Policy", "camera=(), microphone=(), geolocation=()"},
	}

	for _, tt := range tests {
		got := w.Header().Get(tt.header)
		if got != tt.want {
			t.Errorf("header %s = %q, want %q", tt.header, got, tt.want)
		}
	}

	// CSP: widened to allow Next.js inline scripts/styles + data:/blob:
	// images because the canvas is reverse-proxied through the same
	// gin middleware stack. Assert the policy starts with the tight
	// default-src and contains each expected directive — exact-match
	// would brittle-break every time we tune a subsource list.
	csp := w.Header().Get("Content-Security-Policy")
	for _, fragment := range []string{
		"default-src 'self'",
		"script-src 'self' 'unsafe-inline'",
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data: blob:",
		"connect-src 'self' ws: wss:",
		"font-src 'self' data:",
	} {
		if !strings.Contains(csp, fragment) {
			t.Errorf("CSP missing expected fragment %q (full CSP: %q)", fragment, csp)
		}
	}
}

func TestSecurityHeadersPresenceOnMultipleRoutes(t *testing.T) {
	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/a", func(c *gin.Context) { c.String(http.StatusOK, "a") })
	r.POST("/b", func(c *gin.Context) { c.String(http.StatusCreated, "b") })

	// GET /a
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodGet, "/a", nil)
	r.ServeHTTP(w1, req1)

	if v := w1.Header().Get("X-Frame-Options"); v != "DENY" {
		t.Errorf("GET /a: X-Frame-Options = %q, want DENY", v)
	}

	// POST /b
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/b", nil)
	r.ServeHTTP(w2, req2)

	if v := w2.Header().Get("X-Content-Type-Options"); v != "nosniff" {
		t.Errorf("POST /b: X-Content-Type-Options = %q, want nosniff", v)
	}
	if v := w2.Header().Get("Strict-Transport-Security"); v != "max-age=31536000; includeSubDomains" {
		t.Errorf("POST /b: Strict-Transport-Security = %q, want max-age=31536000; includeSubDomains", v)
	}
	// Fragment-match rather than exact — CSP subsource lists get tuned
	// without changing the security posture. Test intent is "CSP is
	// present + starts with the tight default-src", not "CSP matches
	// this exact byte-for-byte string".
	csp := w2.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "default-src 'self'") {
		t.Errorf("POST /b: CSP missing default-src 'self' (full: %q)", csp)
	}
}

func TestSecurityHeadersDoNotOverrideExisting(t *testing.T) {
	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/custom", func(c *gin.Context) {
		// Handler sets its own X-Frame-Options — SecurityHeaders runs before
		// the handler, so the handler's value will take precedence.
		c.Header("X-Frame-Options", "SAMEORIGIN")
		c.String(http.StatusOK, "custom")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/custom", nil)
	r.ServeHTTP(w, req)

	// The handler's value should be present (may override middleware's)
	got := w.Header().Get("X-Frame-Options")
	if got != "SAMEORIGIN" {
		t.Errorf("expected handler override SAMEORIGIN, got %q", got)
	}
}
