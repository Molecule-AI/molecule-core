package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/middleware"
	"github.com/gin-gonic/gin"
)

// buildOrgTemplatesEngine builds a minimal Gin engine with only the
// /org/templates route and AdminAuth middleware — same registration as router.go.
func buildOrgTemplatesEngine(t *testing.T) gin.IRouter {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Stub handler: if auth passes, return 200 with an empty JSON array.
	r.GET("/org/templates", middleware.AdminAuth(db.DB), func(c *gin.Context) {
		c.JSON(http.StatusOK, []interface{}{})
	})
	return r
}

// TestOrgTemplatesRoute_RequiresAdminAuth_WhenTokensExist verifies that
// GET /org/templates returns 401 for unauthenticated callers once the platform
// has at least one live workspace token. Prior to #686 this route had no auth
// middleware, allowing anyone on the Docker network to enumerate org names and
// workspace counts without credentials.
func TestOrgTemplatesRoute_RequiresAdminAuth_WhenTokensExist(t *testing.T) {
	mock := setupRouterTestDB(t)

	// HasAnyLiveTokenGlobal: platform has one enrolled workspace.
	mock.ExpectQuery("SELECT COUNT.*FROM workspace_auth_tokens").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	r := buildOrgTemplatesEngine(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/org/templates", nil)
	// No Authorization header — AdminAuth must reject.
	r.(http.Handler).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when tokens exist and no auth header, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock expectations not met: %v", err)
	}
}

// TestOrgTemplatesRoute_FailOpenOnFreshInstall verifies that AdminAuth is
// fail-open on a fresh install (HasAnyLiveTokenGlobal == 0), so the bootstrap
// flow can still query templates before the first workspace registers.
func TestOrgTemplatesRoute_FailOpenOnFreshInstall(t *testing.T) {
	mock := setupRouterTestDB(t)

	// HasAnyLiveTokenGlobal: no tokens yet — fresh install.
	mock.ExpectQuery("SELECT COUNT.*FROM workspace_auth_tokens").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	r := buildOrgTemplatesEngine(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/org/templates", nil)
	r.(http.Handler).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 on fresh install (fail-open), got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock expectations not met: %v", err)
	}
}
