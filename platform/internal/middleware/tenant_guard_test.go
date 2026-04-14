package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// helper: build a router with TenantGuard configured to `orgID` and two
// representative routes — a regular API route and two allowlisted ones.
func newGuardedRouter(orgID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(TenantGuardWithOrgID(orgID))
	r.GET("/health", func(c *gin.Context) { c.String(200, "ok") })
	r.GET("/metrics", func(c *gin.Context) { c.String(200, "metrics") })
	r.GET("/workspaces", func(c *gin.Context) { c.String(200, "workspaces") })
	return r
}

func doRequest(r *gin.Engine, path, orgIDHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", path, nil)
	if orgIDHeader != "" {
		req.Header.Set("X-Molecule-Org-Id", orgIDHeader)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// MOLECULE_ORG_ID unset → passthrough. Existing self-hosted behavior preserved.
func TestTenantGuard_UnsetIsPassthrough(t *testing.T) {
	r := newGuardedRouter("")
	for _, path := range []string{"/health", "/metrics", "/workspaces"} {
		if w := doRequest(r, path, ""); w.Code != 200 {
			t.Errorf("%s: expected 200 with guard disabled, got %d", path, w.Code)
		}
	}
}

// Set + matching header → 200.
func TestTenantGuard_MatchingHeader(t *testing.T) {
	r := newGuardedRouter("org-abc")
	if w := doRequest(r, "/workspaces", "org-abc"); w.Code != 200 {
		t.Errorf("matching header: expected 200, got %d", w.Code)
	}
}

// Set + mismatching header → 404 (not 403 — don't leak tenant existence).
func TestTenantGuard_MismatchedHeaderIs404(t *testing.T) {
	r := newGuardedRouter("org-abc")
	w := doRequest(r, "/workspaces", "org-xyz")
	if w.Code != 404 {
		t.Errorf("mismatched header: expected 404, got %d", w.Code)
	}
	if w.Body.String() != "" {
		// Bouncing via AbortWithStatus leaves an empty body, which is what we
		// want — no response body means no tenant fingerprint.
		t.Errorf("expected empty body on 404, got %q", w.Body.String())
	}
}

// Set + missing header → 404.
func TestTenantGuard_MissingHeaderIs404(t *testing.T) {
	r := newGuardedRouter("org-abc")
	if w := doRequest(r, "/workspaces", ""); w.Code != 404 {
		t.Errorf("missing header: expected 404, got %d", w.Code)
	}
}

// Allowlisted paths bypass the guard even in tenant mode — required for health
// probes (Fly Machines checks) and Prometheus scrape.
func TestTenantGuard_AllowlistBypassesCheck(t *testing.T) {
	r := newGuardedRouter("org-abc")
	for _, path := range []string{"/health", "/metrics"} {
		w := doRequest(r, path, "") // no header
		if w.Code != 200 {
			t.Errorf("%s: allowlisted path should return 200 without header, got %d", path, w.Code)
		}
	}
}

// The allowlist is exact-match, not prefix. "/health/debug" must NOT bypass.
func TestTenantGuard_AllowlistIsExactMatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(TenantGuardWithOrgID("org-abc"))
	r.GET("/health/debug", func(c *gin.Context) { c.String(200, "debug") })

	req := httptest.NewRequest("GET", "/health/debug", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected /health/debug to be guarded (404), got %d", w.Code)
	}
}
