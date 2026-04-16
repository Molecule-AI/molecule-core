package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestCanvasProxy_StripsAuthorizationHeader verifies that workspace bearer
// tokens are NOT forwarded to the canvas Next.js server (issue #451 / N2).
// A compromised or unpatched Next.js route could echo the token back to an
// attacker; stripping it at the proxy layer is the safe default.
func TestCanvasProxy_StripsAuthorizationHeader(t *testing.T) {
	var capturedAuth string

	// Stand up a tiny upstream that records what headers it received.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.NoRoute(newCanvasProxy(upstream.URL))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/some-canvas-page", nil)
	req.Header.Set("Authorization", "Bearer ws-secret-token")
	engine.ServeHTTP(w, req)

	if capturedAuth != "" {
		t.Errorf("Authorization header must not reach canvas upstream, got %q", capturedAuth)
	}
}

// TestCanvasProxy_StripsCookieHeader verifies that session cookies are not
// forwarded to the canvas Next.js server (same rationale as Authorization).
func TestCanvasProxy_StripsCookieHeader(t *testing.T) {
	var capturedCookie string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCookie = r.Header.Get("Cookie")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.NoRoute(newCanvasProxy(upstream.URL))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/canvas-route", nil)
	req.Header.Set("Cookie", "session=abc123; auth=secret")
	engine.ServeHTTP(w, req)

	if capturedCookie != "" {
		t.Errorf("Cookie header must not reach canvas upstream, got %q", capturedCookie)
	}
}

// TestCanvasProxy_ForwardsOtherHeaders verifies that non-credential headers
// (e.g. Accept, X-Request-ID) still reach the upstream — stripping is
// surgical, not a blanket header wipe.
func TestCanvasProxy_ForwardsOtherHeaders(t *testing.T) {
	var capturedAccept, capturedRequestID string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAccept = r.Header.Get("Accept")
		capturedRequestID = r.Header.Get("X-Request-Id")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.NoRoute(newCanvasProxy(upstream.URL))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/page", nil)
	req.Header.Set("Accept", "text/html")
	req.Header.Set("X-Request-Id", "trace-abc")
	req.Header.Set("Authorization", "Bearer should-be-stripped")
	engine.ServeHTTP(w, req)

	if capturedAccept != "text/html" {
		t.Errorf("Accept header should be forwarded, got %q", capturedAccept)
	}
	if capturedRequestID != "trace-abc" {
		t.Errorf("X-Request-Id should be forwarded, got %q", capturedRequestID)
	}
}

// TestCanvasProxy_NoBothCredentialHeaders is the combined regression: a request
// carrying both Authorization AND Cookie must have both stripped.
func TestCanvasProxy_NoBothCredentialHeaders(t *testing.T) {
	var gotAuth, gotCookie string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotCookie = r.Header.Get("Cookie")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.NoRoute(newCanvasProxy(upstream.URL))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard", nil)
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("Cookie", "sid=xyz")
	engine.ServeHTTP(w, req)

	if gotAuth != "" || gotCookie != "" {
		t.Errorf("both credential headers must be stripped: Authorization=%q Cookie=%q", gotAuth, gotCookie)
	}
}
