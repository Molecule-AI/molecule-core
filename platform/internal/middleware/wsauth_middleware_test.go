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
// Since PR #357 (#351 fix) the middleware enforces strictly: every request
// under /workspaces/:id/* must carry a valid bearer token — no fail-open,
// no grace period, no existence check.
// ────────────────────────────────────────────────────────────────────────────

// hasAnyLiveTokenGlobalQuery is matched for HasAnyLiveTokenGlobal.
const hasAnyLiveTokenGlobalQuery = "SELECT COUNT.*FROM workspace_auth_tokens"

// validateTokenQuery is matched for ValidateToken (SELECT).
const validateTokenSelectQuery = "SELECT id, workspace_id.*FROM workspace_auth_tokens.*token_hash"

// validateAnyTokenQuery is matched for ValidateAnyToken (SELECT).
// #684: the query now filters token_type = 'admin' so workspace tokens cannot
// satisfy AdminAuth. No workspace JOIN needed (admin tokens have NULL workspace_id).
const validateAnyTokenSelectQuery = "SELECT id.*FROM workspace_auth_tokens.*token_type = 'admin'"

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

// TestWorkspaceAuth_351_NoBearer_Returns401 — strict contract: every request
// under /workspaces/:id/* must carry a valid bearer, period. No fail-open,
// no grace period, no existence check. The middleware goes straight to
// ValidateToken and 401s when the bearer is missing — no DB calls happen.
// #351 closed the last fail-open hole (zombie workspaces with no tokens);
// this test pins the strict behaviour.
func TestWorkspaceAuth_351_NoBearer_Returns401_NoDBCalls(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	// NO expected queries — strict path short-circuits before any DB call.

	r := gin.New()
	r.GET("/workspaces/:id/secrets", WorkspaceAuth(mockDB), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// Any UUID — fake, legit, zombie — all must 401 without a bearer.
	for _, id := range []string{
		"00000000-0000-0000-0000-000000000000", // fake UUID (#318 scenario)
		"ffffffff-ffff-ffff-ffff-ffffffffffff", // zombie test-artifact (#351)
		"ws-bootstrap",                         // legitimate pre-token (was grace-period)
	} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/workspaces/"+id+"/secrets", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("no-bearer %q: expected 401, got %d: %s", id, w.Code, w.Body.String())
		}
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

	// #351: no DB calls expected — strict path short-circuits at missing bearer.

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

	// #351: no DB calls expected — strict path short-circuits at missing bearer.

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

	// ValidateToken SELECT returns workspace-a from DB — strict middleware
	// catches the mismatch via the workspace-binding check in wsauth.ValidateToken.
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
// Issue #170 regression — unauthenticated DELETE /workspaces/:id/secrets/:key
//
// Before this fix, the route was registered as:
//   r.DELETE("/workspaces/:id/secrets/:key", sech.Delete)
// on the bare Gin router — no auth at all.  Any caller could delete a secret
// AND trigger a forced workspace restart (the handler calls go restartFunc(id)
// on every successful delete).  CWE-306.
//
// The fix: move the route inside the wsAuth group so it matches all other
// /workspaces/:id/secrets mutations (POST + PUT are already auth-gated).
// ────────────────────────────────────────────────────────────────────────────

// TestWorkspaceAuth_Issue170_SecretDelete_NoBearer_Returns401 is the primary
// regression test: when the workspace has live tokens, a DELETE /secrets/:key
// without a bearer token must be rejected with 401.
func TestWorkspaceAuth_Issue170_SecretDelete_NoBearer_Returns401(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	// #351: strict path — no DB calls on missing bearer.

	r := gin.New()
	// Mirror the fix: DELETE /secrets/:key is inside the wsAuth group.
	r.DELETE("/workspaces/:id/secrets/:key",
		WorkspaceAuth(mockDB),
		func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "deleted"}) })

	w := httptest.NewRecorder()
	// #170 attack: no Authorization header.
	req, _ := http.NewRequest(http.MethodDelete,
		"/workspaces/ws-secret-owner/secrets/OPENAI_API_KEY", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("#170 secret delete no-bearer: expected 401, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestWorkspaceAuth_Issue170_SecretDelete_NoTokensStillRejected — #351: the
// former "fail-open for tokenless workspaces" path is gone. Even a legitimate
// pre-Phase-30.1 workspace now must present a bearer; otherwise 401. This
// closes the zombie-workspace secret-enumeration vector.
func TestWorkspaceAuth_Issue170_SecretDelete_NoTokensStillRejected(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	// #351: no DB calls on missing bearer.

	r := gin.New()
	r.DELETE("/workspaces/:id/secrets/:key",
		WorkspaceAuth(mockDB),
		func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "deleted"}) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete,
		"/workspaces/ws-legacy/secrets/SOME_KEY", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("#351 tokenless-no-bearer: expected 401, got %d: %s", w.Code, w.Body.String())
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

// ────────────────────────────────────────────────────────────────────────────
// Issue #180 regression — unauthenticated GET /approvals/pending
//
// GET /approvals/pending was registered on the open router (no middleware)
// and returned all pending approvals across every workspace to any caller,
// with no token required.
// Attack vector confirmed by Security Auditor:
//   curl http://host/approvals/pending → 200 with full cross-workspace list
//
// Fixed by adding inline AdminAuth to the route registration in router.go.
// This test asserts the gate blocks unauthenticated reads.
// ────────────────────────────────────────────────────────────────────────────

// TestAdminAuth_Issue180_ApprovalsListing_NoBearer_Returns401 documents the #180
// attack vector and verifies that AdminAuth returns 401 for GET without a token.
func TestAdminAuth_Issue180_ApprovalsListing_NoBearer_Returns401(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	// HasAnyLiveTokenGlobal returns 1 — at least one workspace is token-enrolled.
	mock.ExpectQuery(hasAnyLiveTokenGlobalQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	r := gin.New()
	// Mirror the router.go fix: GET /approvals/pending is behind AdminAuth.
	r.GET("/approvals/pending", AdminAuth(mockDB), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"approvals": []interface{}{}})
	})

	w := httptest.NewRecorder()
	// #180 attack: no Authorization header on GET /approvals/pending.
	req, _ := http.NewRequest(http.MethodGet, "/approvals/pending", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("#180 GET /approvals/pending no-bearer: expected 401, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestAdminAuth_Issue180_ApprovalsListing_FailOpen_NoTokens documents the
// fail-open contract: on a fresh install (no tokens anywhere), the middleware
// must not block the canvas from polling /approvals/pending.
func TestAdminAuth_Issue180_ApprovalsListing_FailOpen_NoTokens(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	// HasAnyLiveTokenGlobal returns 0 — fresh install, no tokens yet.
	mock.ExpectQuery(hasAnyLiveTokenGlobalQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	r := gin.New()
	r.GET("/approvals/pending", AdminAuth(mockDB), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"approvals": []interface{}{}})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/approvals/pending", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("#180 fail-open (no tokens): expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

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

// ── CanvasOrBearer (#168) ────────────────────────────────────────────────────
// Narrow softer variant of AdminAuth used ONLY on PUT /canvas/viewport.
// Accepts bearer or a matching Origin header. MUST NOT be used anywhere a
// forged request would leak data or create resources.

func TestCanvasOrBearer_NoTokens_FailOpen(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer mockDB.Close()

	mock.ExpectQuery(hasAnyLiveTokenGlobalQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	r := gin.New()
	r.PUT("/canvas/viewport", CanvasOrBearer(mockDB), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/canvas/viewport", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("bootstrap fail-open: got %d, want 200 (%s)", w.Code, w.Body.String())
	}
}

func TestCanvasOrBearer_TokensExist_NoCreds_Returns401(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer mockDB.Close()

	mock.ExpectQuery(hasAnyLiveTokenGlobalQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	r := gin.New()
	r.PUT("/canvas/viewport", CanvasOrBearer(mockDB), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/canvas/viewport", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("no creds: got %d, want 401", w.Code)
	}
}

func TestCanvasOrBearer_TokensExist_CanvasOrigin_Passes(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer mockDB.Close()

	mock.ExpectQuery(hasAnyLiveTokenGlobalQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	t.Setenv("CORS_ORIGINS", "https://acme.moleculesai.app,https://bob.moleculesai.app")

	r := gin.New()
	r.PUT("/canvas/viewport", CanvasOrBearer(mockDB), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/canvas/viewport", nil)
	req.Header.Set("Origin", "https://acme.moleculesai.app")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("canvas origin: got %d, want 200 (%s)", w.Code, w.Body.String())
	}
}

func TestCanvasOrBearer_TokensExist_WrongOrigin_Returns401(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer mockDB.Close()

	mock.ExpectQuery(hasAnyLiveTokenGlobalQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	t.Setenv("CORS_ORIGINS", "https://acme.moleculesai.app")

	r := gin.New()
	r.PUT("/canvas/viewport", CanvasOrBearer(mockDB), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/canvas/viewport", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("wrong origin: got %d, want 401", w.Code)
	}
}

func TestCanvasOriginAllowed_EmptyOriginRejected(t *testing.T) {
	if canvasOriginAllowed("") {
		t.Error("empty Origin must not pass")
	}
}

func TestCanvasOriginAllowed_LocalhostDefault(t *testing.T) {
	t.Setenv("CORS_ORIGINS", "")
	if !canvasOriginAllowed("http://localhost:3000") {
		t.Error("localhost:3000 should be allowed by default")
	}
	if canvasOriginAllowed("http://evil.example.com") {
		t.Error("random origin should not be allowed")
	}
}

// ── Issue #623 regression ─────────────────────────────────────────────────────
// AdminAuth must NOT accept forged Origin headers. Any container on the Docker
// network can set Origin: http://localhost:3000 without a bearer token, which
// previously bypassed AdminAuth on ALL admin-gated routes. (#623, dup #626)

// TestAdminAuth_623_ForgedOrigin_Returns401 — the main regression test:
// a request with a matching CORS origin but no bearer token must be rejected.
func TestAdminAuth_623_ForgedOrigin_Returns401(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer mockDB.Close()

	// Platform has live tokens — AdminAuth is active.
	mock.ExpectQuery(hasAnyLiveTokenGlobalQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	t.Setenv("CORS_ORIGINS", "http://localhost:3000")

	r := gin.New()
	r.GET("/settings/secrets", AdminAuth(mockDB), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"secrets": []string{"OPENAI_API_KEY"}})
	})

	w := httptest.NewRecorder()
	// #623 attack: forge the canvas Origin header — no bearer token.
	req, _ := http.NewRequest(http.MethodGet, "/settings/secrets", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("#623 forged Origin bypass: expected 401, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestAdminAuth_623_ForgedCORSOrigin_Returns401 — variant: attacker uses the
// tenant-domain CORS origin from CORS_ORIGINS (not just localhost).
func TestAdminAuth_623_ForgedCORSOrigin_Returns401(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer mockDB.Close()

	mock.ExpectQuery(hasAnyLiveTokenGlobalQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	t.Setenv("CORS_ORIGINS", "https://acme.moleculesai.app")

	r := gin.New()
	r.GET("/admin/secrets", AdminAuth(mockDB), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin/secrets", nil)
	req.Header.Set("Origin", "https://acme.moleculesai.app")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("#623 forged tenant Origin: expected 401, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestAdminAuth_623_ValidBearer_WithOrigin_Passes — bearer + matching Origin
// should still work (the Origin is irrelevant once the bearer validates).
func TestAdminAuth_623_ValidBearer_WithOrigin_Passes(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer mockDB.Close()

	goodToken := "valid-bearer-token-xyz"
	tokenHash := sha256.Sum256([]byte(goodToken))

	mock.ExpectQuery(hasAnyLiveTokenGlobalQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(validateAnyTokenSelectQuery).
		WithArgs(tokenHash[:]).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("tok-1"))
	mock.ExpectExec(validateTokenUpdateQuery).
		WithArgs("tok-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	t.Setenv("CORS_ORIGINS", "http://localhost:3000")

	r := gin.New()
	r.GET("/settings/secrets", AdminAuth(mockDB), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/settings/secrets", nil)
	req.Header.Set("Authorization", "Bearer "+goodToken)
	req.Header.Set("Origin", "http://localhost:3000") // present but irrelevant
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("bearer+origin: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}
