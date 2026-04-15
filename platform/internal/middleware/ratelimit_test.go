package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// newTestLimiter spins up a tiny limiter with a 2-token/5s budget so tests can
// exhaust + recover without real-time delays.
func newTestLimiter(t *testing.T) (*RateLimiter, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	rl := NewRateLimiter(2, 5*time.Second, ctx)
	r := gin.New()
	r.Use(rl.Middleware())
	r.GET("/x", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	return rl, r
}

// TestRateLimit_HeadersPresentOnAllowedRequest covers issue #105 — every
// response (not just 429s) must carry the X-RateLimit-* triplet so clients
// can back off proactively.
func TestRateLimit_HeadersPresentOnAllowedRequest(t *testing.T) {
	_, r := newTestLimiter(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))

	if got := w.Header().Get("X-RateLimit-Limit"); got != "2" {
		t.Errorf("X-RateLimit-Limit = %q, want 2", got)
	}
	if got := w.Header().Get("X-RateLimit-Remaining"); got != "1" {
		t.Errorf("X-RateLimit-Remaining = %q, want 1", got)
	}
	reset, err := strconv.Atoi(w.Header().Get("X-RateLimit-Reset"))
	if err != nil || reset < 0 || reset > 5 {
		t.Errorf("X-RateLimit-Reset = %q, want 0-5", w.Header().Get("X-RateLimit-Reset"))
	}
}

// TestRateLimit_RetryAfterOn429 — throttled responses must carry Retry-After
// per RFC 6585, so curl/fetch clients back off the exact required window.
func TestRateLimit_RetryAfterOn429(t *testing.T) {
	_, r := newTestLimiter(t)
	// Burn through both tokens.
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: want 200, got %d", i+1, w.Code)
		}
	}
	// Third should 429.
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("3rd request: want 429, got %d", w.Code)
	}
	if got := w.Header().Get("Retry-After"); got == "" {
		t.Error("missing Retry-After header on 429")
	}
	if got := w.Header().Get("X-RateLimit-Remaining"); got != "0" {
		t.Errorf("X-RateLimit-Remaining = %q on 429, want 0", got)
	}
}
