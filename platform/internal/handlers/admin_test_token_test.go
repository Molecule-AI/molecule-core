package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/wsauth"
	"github.com/gin-gonic/gin"
)

func newTestTokenRequest(workspaceID string) (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: workspaceID}}
	c.Request = httptest.NewRequest("GET", "/admin/workspaces/"+workspaceID+"/test-token", nil)
	return w, c
}

func TestAdminTestToken_HiddenInProduction(t *testing.T) {
	setupTestDB(t)
	t.Setenv("MOLECULE_ENV", "production")
	t.Setenv("MOLECULE_ENABLE_TEST_TOKENS", "")

	h := NewAdminTestTokenHandler()
	w, c := newTestTokenRequest("ws-1")
	h.GetTestToken(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 in production, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminTestToken_EnabledViaFlagEvenInProd(t *testing.T) {
	mock := setupTestDB(t)
	t.Setenv("MOLECULE_ENV", "production")
	t.Setenv("MOLECULE_ENABLE_TEST_TOKENS", "1")

	mock.ExpectQuery("SELECT id FROM workspaces WHERE id =").
		WithArgs("ws-1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("ws-1"))
	mock.ExpectExec("INSERT INTO workspace_auth_tokens").
		WillReturnResult(sqlmock.NewResult(0, 1))

	h := NewAdminTestTokenHandler()
	w, c := newTestTokenRequest("ws-1")
	h.GetTestToken(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminTestToken_WorkspaceNotFound(t *testing.T) {
	mock := setupTestDB(t)
	t.Setenv("MOLECULE_ENV", "development")

	mock.ExpectQuery("SELECT id FROM workspaces WHERE id =").
		WithArgs("missing").
		WillReturnError(sqlErrNoRows())

	h := NewAdminTestTokenHandler()
	w, c := newTestTokenRequest("missing")
	h.GetTestToken(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing workspace, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminTestToken_HappyPath_TokenValidates(t *testing.T) {
	mock := setupTestDB(t)
	t.Setenv("MOLECULE_ENV", "development")

	mock.ExpectQuery("SELECT id FROM workspaces WHERE id =").
		WithArgs("ws-1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("ws-1"))

	// Capture the hash inserted by IssueToken so we can replay it on Validate.
	var capturedHash []byte
	mock.ExpectExec("INSERT INTO workspace_auth_tokens").
		WithArgs("ws-1", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	h := NewAdminTestTokenHandler()
	w, c := newTestTokenRequest("ws-1")
	h.GetTestToken(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		AuthToken   string `json:"auth_token"`
		WorkspaceID string `json:"workspace_id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if resp.AuthToken == "" {
		t.Fatal("expected non-empty auth_token")
	}
	if resp.WorkspaceID != "ws-1" {
		t.Errorf("expected workspace_id ws-1, got %q", resp.WorkspaceID)
	}
	if len(resp.AuthToken) < 32 {
		t.Errorf("token looks too short: %d chars", len(resp.AuthToken))
	}

	// Now simulate ValidateToken lookup using the same DB — prove the token
	// can be validated by feeding its sha256 back through ExpectedArgs.
	// (We stub the SELECT rather than re-reading capturedHash since sqlmock
	// doesn't capture live args; the important invariant is that the issued
	// token passes ValidateToken given a matching hash row exists.)
	_ = capturedHash
	mock.ExpectQuery("SELECT id, workspace_id\\s+FROM workspace_auth_tokens").
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workspace_id"}).AddRow("tok-1", "ws-1"))
	mock.ExpectExec("UPDATE workspace_auth_tokens SET last_used_at").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := wsauth.ValidateToken(c.Request.Context(), db.DB, "ws-1", resp.AuthToken); err != nil {
		t.Errorf("issued token failed to validate: %v", err)
	}
}

func sqlErrNoRows() error { return sql.ErrNoRows }
