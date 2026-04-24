package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/handlers"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/middleware"
	"github.com/gin-gonic/gin"
)

// buildTestTokenEngine builds a minimal Gin engine containing only the
// test-token route with AdminAuth middleware — the same registration that
// router.go now uses. Allows us to verify the auth gate is enforced at the
// HTTP layer without spinning up the full Setup() dependency graph.
func buildTestTokenEngine(t *testing.T) gin.IRouter {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	tokh := handlers.NewAdminTestTokenHandler()
	r.GET("/admin/workspaces/:id/test-token", middleware.AdminAuth(db.DB), tokh.GetTestToken)
	return r
}

// setupRouterTestDB initialises db.DB with a sqlmock connection and returns
// the mock controller. Restores db.DB on test cleanup.
func setupRouterTestDB(t *testing.T) sqlmock.Sqlmock {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	prev := db.DB
	db.DB = mockDB
	t.Cleanup(func() {
		db.DB = prev
		mockDB.Close()
	})
	return mock
}

// TestTestTokenRoute_RequiresAdminAuth_WhenTokensExist verifies that once the
// platform has at least one live token, the test-token endpoint returns 401
// for callers that provide no Authorization header. This is the core security
// property added by the fix — without AdminAuth in the router the request
// would reach the handler and mint a new bearer for any workspace UUID.
func TestTestTokenRoute_RequiresAdminAuth_WhenTokensExist(t *testing.T) {
	t.Setenv("MOLECULE_ENV", "development") // enable the handler itself
	// Explicit ADMIN_TOKEN so AdminAuth's dev-mode fail-open branch
	// (middleware/devmode.go::isDevModeFailOpen) does NOT fire — we're
	// testing the production-like security property that once any
	// workspace token exists, an unauthenticated request is rejected.
	// Setting ADMIN_TOKEN is the operator's opt-in to #684 closure and
	// is what hosted SaaS tenants always have set.
	t.Setenv("ADMIN_TOKEN", "test-admin-secret-not-presented-by-caller")
	mock := setupRouterTestDB(t)

	// HasAnyLiveTokenGlobal: platform has one enrolled workspace.
	mock.ExpectQuery("SELECT COUNT.*FROM workspace_auth_tokens").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	r := buildTestTokenEngine(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin/workspaces/ws-target/test-token", nil)
	// No Authorization header — should be rejected by AdminAuth.
	r.(http.Handler).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when tokens exist and no auth header, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock expectations not met: %v", err)
	}
}

// TestTestTokenRoute_FailOpenOnFreshInstall verifies that AdminAuth is
// fail-open on a fresh install (HasAnyLiveTokenGlobal == 0), so the test-token
// bootstrap path still works before the first workspace has registered.
func TestTestTokenRoute_FailOpenOnFreshInstall(t *testing.T) {
	t.Setenv("MOLECULE_ENV", "development")
	mock := setupRouterTestDB(t)

	// HasAnyLiveTokenGlobal: no tokens yet — fresh install.
	mock.ExpectQuery("SELECT COUNT.*FROM workspace_auth_tokens").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// Handler's own DB queries: workspace existence check + token insert.
	mock.ExpectQuery("SELECT id FROM workspaces WHERE id =").
		WithArgs("ws-bootstrap").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("ws-bootstrap"))
	mock.ExpectExec("INSERT INTO workspace_auth_tokens").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := buildTestTokenEngine(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin/workspaces/ws-bootstrap/test-token", nil)
	r.(http.Handler).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 on fresh install (fail-open), got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock expectations not met: %v", err)
	}
}
