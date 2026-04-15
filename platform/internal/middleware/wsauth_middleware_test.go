package middleware

import (
	"crypto/sha256"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

// ────────────────────────────────────────────────────────────────────────────
// WorkspaceAuth middleware tests (covers findings C4, C8 and the full
// per-workspace bearer-token contract).
//
// WorkspaceAuth calls wsauth.HasAnyLiveToken to decide whether to enforce:
//   - 0 live tokens → fail-open (bootstrap / rolling upgrade)
//   - ≥1 live token → Authorization: Bearer <token> required and validated
// ────────────────────────────────────────────────────────────────────────────

// hasLiveTokenQuery is the SQL fragment matched by sqlmock for HasAnyLiveToken.
const hasLiveTokenQuery = "SELECT COUNT.*FROM workspace_auth_tokens.*workspace_id"

// hasAnyLiveTokenGlobalQuery is matched for HasAnyLiveTokenGlobal.
const hasAnyLiveTokenGlobalQuery = "SELECT COUNT.*FROM workspace_auth_tokens"

// validateTokenQuery is matched for ValidateToken (SELECT).
const validateTokenSelectQuery = "SELECT id, workspace_id.*FROM workspace_auth_tokens.*token_hash"

// validateAnyTokenQuery is matched for ValidateAnyToken (SELECT).
const validateAnyTokenSelectQuery = "SELECT id.*FROM workspace_auth_tokens.*token_hash"

// validateTokenUpdateQuery is matched for the best-effort last_used_at UPDATE.
const validateTokenUpdateQuery = "UPDATE workspace_auth_tokens SET last_used_at"

// newWorkspaceAuthRouter builds a minimal gin router that applies WorkspaceAuth
// to a single GET /workspaces/:id/test route, returning 200 on success.
func newWorkspaceAuthRouter(db sqlmock.Sqlmock, realDB interface{ Close() error }) *gin.Engine {
	_ = db  // unused directly; sqlmock intercepts calls via the *sql.DB pointer
	r := gin.New()
	// We need the *sql.DB, not the mock. The caller passes mockDB via the
	// test-local var — this helper is only used to build the router topology.
	return r
}

// TestWorkspaceAuth_FailOpen_NoTokens — C4/C8: when a workspace has no live
// token on file (first boot / pre-Phase-30), the middleware must let the
// request through so in-flight agents are not bricked during rolling upgrades.
func TestWorkspaceAuth_FailOpen_NoTokens(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	// HasAnyLiveToken returns 0 — no tokens yet.
	mock.ExpectQuery(hasLiveTokenQuery).
		WithArgs("ws-bootstrap").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	r := gin.New()
	r.GET("/workspaces/:id/test", WorkspaceAuth(mockDB), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/workspaces/ws-bootstrap/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("fail-open (no tokens): expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestWorkspaceAuth_C4_C8_NoBearer_Returns401 — C4/C8 critical path:
// when a workspace has live tokens and the caller sends NO bearer token,
// the middleware must return 401.  This was the confirmed attack vector —
// unauthenticated POSTs to /delegations/:id/update and /memories succeeded.
func TestWorkspaceAuth_C4_C8_NoBearer_Returns401(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	// HasAnyLiveToken returns 1 — workspace is token-enrolled.
	mock.ExpectQuery(hasLiveTokenQuery).
		WithArgs("ws-enrolled").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	r := gin.New()
	r.POST("/workspaces/:id/delegations/:delegation_id/update",
		WorkspaceAuth(mockDB),
		func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	// C4 attack: no Authorization header.
	req, _ := http.NewRequest(http.MethodPost,
		"/workspaces/ws-enrolled/delegations/del-1/update", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("C4 no-bearer: expected 401, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestWorkspaceAuth_C8_MemoriesCommit_NoBearer_Returns401 tests specifically
// the C8 vector: POST /workspaces/:id/memories without auth.
func TestWorkspaceAuth_C8_MemoriesCommit_NoBearer_Returns401(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	mock.ExpectQuery(hasLiveTokenQuery).
		WithArgs("ws-memory-target").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	r := gin.New()
	r.POST("/workspaces/:id/memories",
		WorkspaceAuth(mockDB),
		func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost,
		"/workspaces/ws-memory-target/memories", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("C8 no-bearer: expected 401, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestWorkspaceAuth_InvalidBearer_Returns401 — wrong token must be rejected.
func TestWorkspaceAuth_InvalidBearer_Returns401(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	// HasAnyLiveToken: tokens exist.
	mock.ExpectQuery(hasLiveTokenQuery).
		WithArgs("ws-enrolled").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// ValidateToken: hash doesn't match any live row.
	wrongToken := "wrong-token-value"
	wrongHash := sha256.Sum256([]byte(wrongToken))
	mock.ExpectQuery(validateTokenSelectQuery).
		WithArgs(wrongHash[:]).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workspace_id"})) // empty → ErrNoRows

	r := gin.New()
	r.GET("/workspaces/:id/test",
		WorkspaceAuth(mockDB),
		func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/workspaces/ws-enrolled/test", nil)
	req.Header.Set("Authorization", "Bearer "+wrongToken)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("invalid bearer: expected 401, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestWorkspaceAuth_ValidBearer_Passes — correct token for the right workspace
// must be accepted and the handler reached (200).
func TestWorkspaceAuth_ValidBearer_Passes(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	testToken := "valid-workspace-bearer-token-abc123"
	tokenHash := sha256.Sum256([]byte(testToken))

	// HasAnyLiveToken: workspace has tokens.
	mock.ExpectQuery(hasLiveTokenQuery).
		WithArgs("ws-enrolled").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// ValidateToken SELECT — returns matching token_id + workspace_id.
	mock.ExpectQuery(validateTokenSelectQuery).
		WithArgs(tokenHash[:]).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workspace_id"}).
			AddRow("tok-1", "ws-enrolled"))

	// Best-effort last_used_at UPDATE (ignored on error, but we expect it).
	mock.ExpectExec(validateTokenUpdateQuery).
		WithArgs("tok-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := gin.New()
	r.POST("/workspaces/:id/memories",
		WorkspaceAuth(mockDB),
		func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/workspaces/ws-enrolled/memories", nil)
	req.Header.Set("Authorization", "Bearer "+testToken)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("valid bearer: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestWorkspaceAuth_WrongWorkspace_Returns401 — token valid for workspace A must
// not authenticate workspace B (cross-workspace token replay attack).
func TestWorkspaceAuth_WrongWorkspace_Returns401(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	tokenForA := "token-issued-to-workspace-a"
	tokenHash := sha256.Sum256([]byte(tokenForA))

	// URL targets workspace-b but the token was issued to workspace-a.
	mock.ExpectQuery(hasLiveTokenQuery).
		WithArgs("workspace-b").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// ValidateToken SELECT returns workspace-a from DB.
	mock.ExpectQuery(validateTokenSelectQuery).
		WithArgs(tokenHash[:]).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workspace_id"}).
			AddRow("tok-a", "workspace-a")) // workspace mismatch!

	r := gin.New()
	r.GET("/workspaces/:id/test",
		WorkspaceAuth(mockDB),
		func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/workspaces/workspace-b/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenForA)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("cross-workspace replay: expected 401, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// AdminAuth middleware tests (covers findings C10, C11 and the
// global bearer-token contract for /admin/secrets, /settings/secrets).
// ────────────────────────────────────────────────────────────────────────────

// ── Issue #168 regression — canvas session-cookie extension ──────────────────
//
// PR #167 gated PUT /canvas/viewport, GET /events/:workspaceId,
// GET /bundles/export/:id, and POST /bundles/import behind AdminAuth (Bearer
// only). Canvas uses credentials:"include" without an Authorization header, so
// every one of those routes 401'd. The fix: AdminAuth also accepts the token
// via a "mcp_session" cookie. Bearer takes precedence; cookie is the fallback.
//
// Three tests:
//   1. Bearer path still works (regression guard)
//   2. Session cookie path works (new canvas path)
//   3. No credentials at all → 401

// TestAdminAuth_Issue168_BearerValid verifies the existing Authorization:Bearer
// path is not disturbed by the cookie extension.
func TestAdminAuth_Issue168_BearerValid(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	tok := "canvas-bearer-regression-token"
	h := sha256.Sum256([]byte(tok))

	mock.ExpectQuery(hasAnyLiveTokenGlobalQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(validateAnyTokenSelectQuery).
		WithArgs(h[:]).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("tok-canvas-1"))
	mock.ExpectExec(validateTokenUpdateQuery).
		WithArgs("tok-canvas-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := gin.New()
	r.PUT("/canvas/viewport", AdminAuth(mockDB), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/canvas/viewport", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("#168 bearer regression: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestAdminAuth_Issue168_SessionCookieValid verifies that a valid token carried
// in the mcp_session cookie is accepted, allowing the canvas to call
// AdminAuth-gated routes with credentials:"include".
func TestAdminAuth_Issue168_SessionCookieValid(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	tok := "canvas-session-cookie-token"
	h := sha256.Sum256([]byte(tok))

	mock.ExpectQuery(hasAnyLiveTokenGlobalQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(validateAnyTokenSelectQuery).
		WithArgs(h[:]).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("tok-canvas-2"))
	mock.ExpectExec(validateTokenUpdateQuery).
		WithArgs("tok-canvas-2").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := gin.New()
	r.GET("/bundles/export/:id", AdminAuth(mockDB), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/bundles/export/ws-123", nil)
	// No Authorization header — canvas sends the token via cookie.
	req.AddCookie(&http.Cookie{Name: "mcp_session", Value: tok})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("#168 session cookie: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestAdminAuth_Issue168_NoCreds_Returns401 verifies that a request with neither
// Authorization header nor mcp_session cookie is rejected with 401.
func TestAdminAuth_Issue168_NoCreds_Returns401(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	mock.ExpectQuery(hasAnyLiveTokenGlobalQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	r := gin.New()
	r.PUT("/canvas/viewport", AdminAuth(mockDB), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/canvas/viewport", nil)
	// Deliberately: no Authorization header, no mcp_session cookie.
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("#168 no-creds: expected 401, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestAdminAuth_FailOpen_NoTokensGlobally — C10/C11: on a fresh install (no
// live tokens anywhere) the middleware must let the request through so existing
// deployments keep working during the Phase-30 rollout.
func TestAdminAuth_FailOpen_NoTokensGlobally(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	// HasAnyLiveTokenGlobal returns 0 — fresh install.
	mock.ExpectQuery(hasAnyLiveTokenGlobalQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	r := gin.New()
	r.GET("/admin/secrets", AdminAuth(mockDB), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/secrets", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("C10 fail-open (no global tokens): expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestAdminAuth_C10_NoBearer_Returns401 — C10 critical path: when at least
// one workspace has tokens, GET /admin/secrets without a bearer → 401.
func TestAdminAuth_C10_NoBearer_Returns401(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	// HasAnyLiveTokenGlobal returns 1 — platform has at least one enrolled workspace.
	mock.ExpectQuery(hasAnyLiveTokenGlobalQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	r := gin.New()
	r.GET("/admin/secrets", AdminAuth(mockDB), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"secrets": []string{"GITHUB_TOKEN", "CLAUDE_CODE_OAUTH_TOKEN"}})
	})

	w := httptest.NewRecorder()
	// C10 attack: no Authorization header — must not leak secrets.
	req, _ := http.NewRequest(http.MethodGet, "/admin/secrets", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("C10 no-bearer: expected 401, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestAdminAuth_C11_PostNoBearer_Returns401 — C11 critical path: env poisoning
// via POST /admin/secrets without auth must be rejected.
func TestAdminAuth_C11_PostNoBearer_Returns401(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	mock.ExpectQuery(hasAnyLiveTokenGlobalQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	r := gin.New()
	r.POST("/admin/secrets", AdminAuth(mockDB), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/admin/secrets", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("C11 POST no-bearer: expected 401, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestAdminAuth_C11_DeleteNoBearer_Returns401 — C11: DELETE /admin/secrets/:key
// without auth must be rejected (env poisoning → RCE on agent restart).
func TestAdminAuth_C11_DeleteNoBearer_Returns401(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	mock.ExpectQuery(hasAnyLiveTokenGlobalQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	r := gin.New()
	r.DELETE("/admin/secrets/:key", AdminAuth(mockDB), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/admin/secrets/CLAUDE_CODE_OAUTH_TOKEN", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("C11 DELETE no-bearer: expected 401, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestAdminAuth_ValidBearer_Passes — a valid bearer token (from any workspace)
// must be accepted for admin routes.
func TestAdminAuth_ValidBearer_Passes(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	adminToken := "admin-bearer-token-from-any-workspace"
	tokenHash := sha256.Sum256([]byte(adminToken))

	// HasAnyLiveTokenGlobal: tokens exist.
	mock.ExpectQuery(hasAnyLiveTokenGlobalQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// ValidateAnyToken SELECT — token matches a live row.
	mock.ExpectQuery(validateAnyTokenSelectQuery).
		WithArgs(tokenHash[:]).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("tok-admin-1"))

	// Best-effort last_used_at UPDATE.
	mock.ExpectExec(validateTokenUpdateQuery).
		WithArgs("tok-admin-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := gin.New()
	r.GET("/admin/secrets", AdminAuth(mockDB), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/secrets", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("C10 valid bearer: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestAdminAuth_InvalidBearer_Returns401 — wrong token must not grant admin access.
func TestAdminAuth_InvalidBearer_Returns401(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	wrongToken := "completely-wrong-token"
	wrongHash := sha256.Sum256([]byte(wrongToken))

	mock.ExpectQuery(hasAnyLiveTokenGlobalQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// ValidateAnyToken SELECT — no matching row.
	mock.ExpectQuery(validateAnyTokenSelectQuery).
		WithArgs(wrongHash[:]).
		WillReturnRows(sqlmock.NewRows([]string{"id"})) // empty → ErrNoRows

	r := gin.New()
	r.GET("/admin/secrets", AdminAuth(mockDB), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/secrets", nil)
	req.Header.Set("Authorization", "Bearer "+wrongToken)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("C10 invalid bearer: expected 401, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Issue #120 regression — unauthenticated PATCH /workspaces/:id
//
// Before PR #125, PATCH /workspaces/:id was registered outside the wsAdmin
// group and did NOT enforce AdminAuth.  An attacker could change workspace
// name, tier, parent_id, runtime, or workspace_dir without any token.
// Security Auditor confirmed the live exploit:
//   curl -X PATCH .../workspaces/00000000-.../  -d '{"name":"probe"}' → 200
//
// This test asserts AdminAuth applied to the PATCH route blocks unauthenticated
// requests — the route-level fix in router.go is the enforcement point.
// ────────────────────────────────────────────────────────────────────────────

// TestAdminAuth_Issue120_PatchWorkspace_NoBearer_Returns401 documents the #120
// attack vector and verifies that AdminAuth returns 401 for PATCH without a token.
func TestAdminAuth_Issue120_PatchWorkspace_NoBearer_Returns401(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	// HasAnyLiveTokenGlobal returns 1 — at least one workspace is token-enrolled.
	mock.ExpectQuery(hasAnyLiveTokenGlobalQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	r := gin.New()
	// Mirror the PR #125 router change: PATCH is inside the wsAdmin AdminAuth group.
	r.PATCH("/workspaces/:id", AdminAuth(mockDB), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "updated"})
	})

	w := httptest.NewRecorder()
	// #120 attack: no Authorization header on PATCH.
	req, _ := http.NewRequest(http.MethodPatch,
		"/workspaces/00000000-0000-0000-0000-000000000000",
		nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("#120 PATCH no-bearer: expected 401, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}
