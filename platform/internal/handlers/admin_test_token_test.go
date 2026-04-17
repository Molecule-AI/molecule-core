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

	// #684: IssueAdminToken inserts with NULL workspace_id, so only hash + prefix
	// are positional args. token_type = 'admin' is a literal in the SQL.
	mock.ExpectExec("INSERT INTO workspace_auth_tokens").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
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

	// Prove the issued admin token passes ValidateAnyToken (AdminAuth path).
	// Stub the SELECT so sqlmock returns a matching row with token_type='admin'.
	mock.ExpectQuery("SELECT id.*FROM workspace_auth_tokens.*token_type = 'admin'").
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("tok-1"))
	mock.ExpectExec("UPDATE workspace_auth_tokens SET last_used_at").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := wsauth.ValidateAnyToken(c.Request.Context(), db.DB, resp.AuthToken); err != nil {
		t.Errorf("issued admin token failed ValidateAnyToken: %v", err)
	}
}

func sqlErrNoRows() error { return sql.ErrNoRows }
