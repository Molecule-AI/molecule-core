package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

// ==================== GET /workspaces/:id ====================

func TestWorkspaceGet_Success(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	columns := []string{
		"id", "name", "role", "tier", "status", "agent_card", "url",
		"parent_id", "active_tasks", "last_error_rate", "last_sample_error",
		"uptime_seconds", "current_task", "runtime", "workspace_dir", "x", "y", "collapsed",
		"budget_limit", "monthly_spend",
	}
	mock.ExpectQuery("SELECT w.id, w.name").
		WithArgs("cccccccc-0001-0000-0000-000000000000").
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow("cccccccc-0001-0000-0000-000000000000", "My Agent", "worker", 1, "online", []byte(`{"name":"test"}`),
				"http://localhost:8001", nil, 2, 0.05, "", 3600, "working", "langgraph",
				"", 10.0, 20.0, false,
				nil, 0))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "cccccccc-0001-0000-0000-000000000000"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-get-1", nil)

	handler.Get(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["name"] != "My Agent" {
		t.Errorf("expected name 'My Agent', got %v", resp["name"])
	}
	if resp["status"] != "online" {
		t.Errorf("expected status 'online', got %v", resp["status"])
	}
	if resp["runtime"] != "langgraph" {
		t.Errorf("expected runtime 'langgraph', got %v", resp["runtime"])
	}
	if resp["current_task"] != "working" {
		t.Errorf("expected current_task 'working', got %v", resp["current_task"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestWorkspaceGet_NotFound(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	mock.ExpectQuery("SELECT w.id, w.name").
		WithArgs("cccccccc-0002-0000-0000-000000000000").
		WillReturnError(sql.ErrNoRows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "cccccccc-0002-0000-0000-000000000000"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-nonexistent", nil)

	handler.Get(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestWorkspaceGet_DBError(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	mock.ExpectQuery("SELECT w.id, w.name").
		WithArgs("cccccccc-0003-0000-0000-000000000000").
		WillReturnError(sql.ErrConnDone)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "cccccccc-0003-0000-0000-000000000000"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-dberr", nil)

	handler.Get(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// ==================== POST /workspaces (Create) ====================

func TestWorkspaceCreate_BadJSON(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Missing required "name" field
	body := `{"tier":1}`
	c.Request = httptest.NewRequest("POST", "/workspaces", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWorkspaceCreate_DBInsertError(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	// Transaction begins, workspace INSERT fails, transaction is rolled back.
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO workspaces").
		WithArgs(sqlmock.AnyArg(), "Failing Agent", nil, 1, "langgraph", sqlmock.AnyArg(), (*string)(nil), nil, "none", (*int64)(nil)).
		WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"name":"Failing Agent"}`
	c.Request = httptest.NewRequest("POST", "/workspaces", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestWorkspaceCreate_DefaultsApplied(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	// Transaction wraps the workspace INSERT (no secrets in this request).
	mock.ExpectBegin()
	// Expect workspace INSERT with defaulted tier=1, runtime="langgraph"
	mock.ExpectExec("INSERT INTO workspaces").
		WithArgs(sqlmock.AnyArg(), "Default Agent", nil, 1, "langgraph", sqlmock.AnyArg(), (*string)(nil), nil, "none", (*int64)(nil)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	// Expect canvas_layouts INSERT (x=0, y=0 — defaults)
	mock.ExpectExec("INSERT INTO canvas_layouts").
		WithArgs(sqlmock.AnyArg(), float64(0), float64(0)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect RecordAndBroadcast INSERT
	mock.ExpectExec("INSERT INTO structure_events").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"name":"Default Agent"}`
	c.Request = httptest.NewRequest("POST", "/workspaces", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["status"] != "provisioning" {
		t.Errorf("expected status 'provisioning', got %v", resp["status"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestWorkspaceCreate_WithSecrets_Persists asserts that secrets in the create
// payload are written to workspace_secrets inside the same transaction as the
// workspace row, and that the handler returns 201.
func TestWorkspaceCreate_WithSecrets_Persists(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	// External workspace: simplest code path — no provisioner goroutine.
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO workspaces").
		WithArgs(sqlmock.AnyArg(), "Hermes Agent", nil, 1, "hermes", sqlmock.AnyArg(), (*string)(nil), nil, "none", (*int64)(nil)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	// Secret inserted inside the same transaction.
	mock.ExpectExec("INSERT INTO workspace_secrets").
		WithArgs(sqlmock.AnyArg(), "HERMES_API_KEY", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	// canvas_layouts (non-fatal, outside tx)
	mock.ExpectExec("INSERT INTO canvas_layouts").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"name":"Hermes Agent","runtime":"hermes","external":true,"secrets":{"HERMES_API_KEY":"sk-test-123"}}`
	c.Request = httptest.NewRequest("POST", "/workspaces", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestWorkspaceCreate_SecretPersistFails_RollsBack asserts that a DB error
// while persisting a secret causes the entire transaction to roll back and
// the handler to return 500.  The workspace row must NOT be committed.
func TestWorkspaceCreate_SecretPersistFails_RollsBack(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO workspaces").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO workspace_secrets").
		WillReturnError(sql.ErrConnDone) // DB failure while writing secret
	mock.ExpectRollback() // workspace insert must be rolled back

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"name":"Rollback Agent","secrets":{"OPENAI_API_KEY":"sk-fail"}}`
	c.Request = httptest.NewRequest("POST", "/workspaces", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestWorkspaceCreate_EmptySecrets_OK asserts that an empty secrets map (or
// no secrets key at all) creates the workspace normally without touching
// workspace_secrets.
func TestWorkspaceCreate_EmptySecrets_OK(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO workspaces").
		WillReturnResult(sqlmock.NewResult(0, 1))
	// No ExpectExec for workspace_secrets — empty map must be a no-op.
	mock.ExpectCommit()
	mock.ExpectExec("INSERT INTO canvas_layouts").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"name":"No Secrets Agent","external":true,"secrets":{}}`
	c.Request = httptest.NewRequest("POST", "/workspaces", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// ==================== GET /workspaces (List) ====================

func TestWorkspaceList_Empty(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	mock.ExpectQuery("SELECT w.id, w.name").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "role", "tier", "status", "agent_card", "url",
			"parent_id", "active_tasks", "last_error_rate", "last_sample_error",
			"uptime_seconds", "current_task", "runtime", "workspace_dir", "x", "y", "collapsed",
			"budget_limit", "monthly_spend",
		}))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/workspaces", nil)

	handler.List(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("expected 0 workspaces, got %d", len(resp))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestWorkspaceList_DBError(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	mock.ExpectQuery("SELECT w.id, w.name").
		WillReturnError(sql.ErrConnDone)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/workspaces", nil)

	handler.List(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// ==================== PATCH /workspaces/:id (Update) ====================

func TestWorkspaceUpdate_BadJSON(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "cccccccc-0004-0000-0000-000000000000"}}
	c.Request = httptest.NewRequest("PATCH", "/workspaces/ws-upd", bytes.NewBufferString("not json"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Update(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWorkspaceUpdate_MultipleFields(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	// #125: existence probe fires once before any field update.
	mock.ExpectQuery("SELECT EXISTS.*workspaces WHERE id").
		WithArgs("cccccccc-0005-0000-0000-000000000000").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	// Expect name, role, and tier updates
	mock.ExpectExec("UPDATE workspaces SET name").
		WithArgs("cccccccc-0005-0000-0000-000000000000", "Updated Agent").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE workspaces SET role").
		WithArgs("cccccccc-0005-0000-0000-000000000000", "manager").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE workspaces SET tier").
		WithArgs("cccccccc-0005-0000-0000-000000000000", float64(3)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "cccccccc-0005-0000-0000-000000000000"}}

	body := `{"name":"Updated Agent","role":"manager","tier":3}`
	c.Request = httptest.NewRequest("PATCH", "/workspaces/ws-multi", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Update(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["status"] != "updated" {
		t.Errorf("expected status 'updated', got %v", resp["status"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestWorkspaceUpdate_RuntimeField(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	mock.ExpectQuery("SELECT EXISTS.*workspaces WHERE id").
		WithArgs("cccccccc-0006-0000-0000-000000000000").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec("UPDATE workspaces SET runtime").
		WithArgs("cccccccc-0006-0000-0000-000000000000", "claude-code").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "cccccccc-0006-0000-0000-000000000000"}}

	body := `{"runtime":"claude-code"}`
	c.Request = httptest.NewRequest("PATCH", "/workspaces/ws-rt", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Update(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// ==================== DELETE /workspaces/:id ====================

func TestWorkspaceDelete_ConfirmationRequired(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	// Children query returns 2 children
	mock.ExpectQuery("SELECT id, name FROM workspaces WHERE parent_id").
		WithArgs("cccccccc-0007-0000-0000-000000000000").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow("cccccccc-0008-0000-0000-000000000000", "Child One").
			AddRow("cccccccc-0009-0000-0000-000000000000", "Child Two"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "cccccccc-0007-0000-0000-000000000000"}}
	// No ?confirm=true
	c.Request = httptest.NewRequest("DELETE", "/workspaces/ws-parent", nil)

	handler.Delete(c)

	// #88: confirmation required now returns 409 Conflict (not 200) so
	// curl --fail / fetch().ok / any HTTP-status-aware client surfaces
	// the confirmation requirement instead of silently treating it as
	// success.
	if w.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["status"] != "confirmation_required" {
		t.Errorf("expected status 'confirmation_required', got %v", resp["status"])
	}
	if resp["children_count"] != float64(2) {
		t.Errorf("expected children_count 2, got %v", resp["children_count"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestWorkspaceDelete_CascadeWithChildren(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	// Children query returns 1 child
	mock.ExpectQuery("SELECT id, name FROM workspaces WHERE parent_id").
		WithArgs("cccccccc-000a-0000-0000-000000000000").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow("cccccccc-000b-0000-0000-000000000000", "Child Agent"))

	// Descendant CTE query returns the recursive set (1 descendant: ws-child-del)
	mock.ExpectQuery("WITH RECURSIVE descendants").
		WithArgs("cccccccc-000a-0000-0000-000000000000").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("cccccccc-000b-0000-0000-000000000000"))

	// #73: single batch UPDATE covering [self + descendants] BEFORE stopping
	// containers (prevents heartbeat/restart resurrection races).
	mock.ExpectExec("UPDATE workspaces SET status = 'removed'").
		WillReturnResult(sqlmock.NewResult(2, 2))
	// Batch canvas_layouts DELETE for the same id set.
	mock.ExpectExec("DELETE FROM canvas_layouts WHERE workspace_id = ANY").
		WillReturnResult(sqlmock.NewResult(2, 2))
	// Token revocation: once a workspace is gone its auth tokens are meaningless.
	mock.ExpectExec("UPDATE workspace_auth_tokens SET revoked_at").
		WillReturnResult(sqlmock.NewResult(0, 2))
	// Broadcast for child WORKSPACE_REMOVED (fires during the descendant loop).
	mock.ExpectExec("INSERT INTO structure_events").
		WillReturnResult(sqlmock.NewResult(0, 1))
	// Broadcast for parent WORKSPACE_REMOVED (fires after parent cleanup).
	mock.ExpectExec("INSERT INTO structure_events").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "cccccccc-000a-0000-0000-000000000000"}}
	c.Request = httptest.NewRequest("DELETE", "/workspaces/ws-parent-del?confirm=true", nil)

	handler.Delete(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["status"] != "removed" {
		t.Errorf("expected status 'removed', got %v", resp["status"])
	}
	if resp["cascade_deleted"] != float64(1) {
		t.Errorf("expected cascade_deleted 1, got %v", resp["cascade_deleted"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestWorkspaceDelete_ChildrenQueryError(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	mock.ExpectQuery("SELECT id, name FROM workspaces WHERE parent_id").
		WithArgs("cccccccc-000c-0000-0000-000000000000").
		WillReturnError(sql.ErrConnDone)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "cccccccc-000c-0000-0000-000000000000"}}
	c.Request = httptest.NewRequest("DELETE", "/workspaces/ws-err-del?confirm=true", nil)

	handler.Delete(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// ==================== Phase 30.4 — State polling ====================

const stateWsID = "550e8400-e29b-41d4-a716-446655440000"

func stateReq(w *httptest.ResponseRecorder, auth string) *gin.Context {
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: stateWsID}}
	req := httptest.NewRequest("GET", "/workspaces/"+stateWsID+"/state", nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	c.Request = req
	return c
}

func TestWorkspaceState_LegacyGrandfatheredOnlineStatus(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", "/tmp")

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM workspace_auth_tokens`).
		WithArgs(stateWsID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT status\s+FROM workspaces\s+WHERE id`).
		WithArgs(stateWsID).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("online"))

	w := httptest.NewRecorder()
	c := stateReq(w, "")
	handler.State(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["status"] != "online" || body["paused"] != false || body["deleted"] != false {
		t.Errorf("unexpected body: %+v", body)
	}
}

func TestWorkspaceState_PausedDetected(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", "/tmp")

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM workspace_auth_tokens`).
		WithArgs(stateWsID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT status\s+FROM workspaces\s+WHERE id`).
		WithArgs(stateWsID).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("paused"))

	w := httptest.NewRecorder()
	c := stateReq(w, "")
	handler.State(c)

	var body map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["paused"] != true {
		t.Errorf("paused flag should be true when status=paused; body=%v", body)
	}
}

func TestWorkspaceState_DeletedRowReturns404WithDeletedFlag(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", "/tmp")

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM workspace_auth_tokens`).
		WithArgs(stateWsID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT status\s+FROM workspaces\s+WHERE id`).
		WithArgs(stateWsID).
		WillReturnError(sql.ErrNoRows)

	w := httptest.NewRecorder()
	c := stateReq(w, "")
	handler.State(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for hard-deleted row, got %d", w.Code)
	}
	var body map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["deleted"] != true {
		t.Errorf("deleted flag should be true on 404; body=%+v", body)
	}
}

func TestWorkspaceState_MissingTokenWhenOnFile(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", "/tmp")

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM workspace_auth_tokens`).
		WithArgs(stateWsID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	w := httptest.NewRecorder()
	c := stateReq(w, "")
	handler.State(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when token required and absent, got %d", w.Code)
	}
}

func TestWorkspaceState_ValidTokenReturnsStatus(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", "/tmp")

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM workspace_auth_tokens`).
		WithArgs(stateWsID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT t\.id, t\.workspace_id.*FROM workspace_auth_tokens t.*JOIN workspaces`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workspace_id"}).AddRow("t1", stateWsID))
	mock.ExpectExec(`UPDATE workspace_auth_tokens SET last_used_at`).
		WithArgs("t1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT status\s+FROM workspaces\s+WHERE id`).
		WithArgs(stateWsID).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("degraded"))

	w := httptest.NewRecorder()
	c := stateReq(w, "Bearer good")
	handler.State(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["status"] != "degraded" {
		t.Errorf("status should be 'degraded', got %v", body["status"])
	}
}

// ── #138 field-level auth tests ─────────────────────────────────────────────
// Cosmetic PATCH (name/x/y/role) stays open so canvas drag-reposition works
// without a bearer token. Sensitive fields (tier/parent_id/runtime/
// workspace_dir) require a valid admin bearer once any live token exists.

func TestWorkspaceUpdate_CosmeticField_NoBearer_FailOpen_NoTokens(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	// Body contains only cosmetic field → no wsauth probe ever fires.
	mock.ExpectQuery("SELECT EXISTS.*workspaces WHERE id").
		WithArgs("cccccccc-000d-0000-0000-000000000000").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec("UPDATE workspaces SET name").
		WithArgs("cccccccc-000d-0000-0000-000000000000", "Cosmetic").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "cccccccc-000d-0000-0000-000000000000"}}
	c.Request = httptest.NewRequest("PATCH", "/workspaces/ws-cosmetic",
		bytes.NewBufferString(`{"name":"Cosmetic"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	handler.Update(c)

	if w.Code != http.StatusOK {
		t.Errorf("cosmetic PATCH (no bearer) should pass; got %d: %s", w.Code, w.Body.String())
	}
}

// TestWorkspaceUpdate_SensitiveField_AuthEnforcedByMiddleware documents the #680 fix:
// auth for PATCH /workspaces/:id is now enforced by WorkspaceAuth middleware (router
// layer), not inside the handler. The handler processes sensitive fields (tier,
// parent_id, runtime, workspace_dir) directly — WorkspaceAuth has already verified
// the caller holds a valid bearer token for this specific workspace before the handler
// runs. No in-handler wsauth DB probe fires.
func TestWorkspaceUpdate_SensitiveField_AuthEnforcedByMiddleware(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	// No workspace_auth_tokens query expected — auth is middleware's responsibility.
	mock.ExpectQuery("SELECT EXISTS.*workspaces WHERE id").
		WithArgs("cccccccc-000e-0000-0000-000000000000").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec("UPDATE workspaces SET tier").
		WithArgs("cccccccc-000e-0000-0000-000000000000", float64(3)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "cccccccc-000e-0000-0000-000000000000"}}
	c.Request = httptest.NewRequest("PATCH", "/workspaces/cccccccc-000e-0000-0000-000000000000",
		bytes.NewBufferString(`{"tier":3}`))
	c.Request.Header.Set("Content-Type", "application/json")
	// WorkspaceAuth middleware would have validated the bearer before this runs.
	handler.Update(c)

	if w.Code != http.StatusOK {
		t.Errorf("sensitive PATCH (auth at middleware): got %d, want 200: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// ==================== #611 Security Auditor regressions ====================

// TestWorkspaceGet_FinancialFieldsStripped verifies that GET /workspaces/:id
// does NOT expose budget_limit or monthly_spend. The endpoint is on the open
// router — any caller with a UUID would otherwise read billing data. (#611 Fix 2)
func TestWorkspaceGet_FinancialFieldsStripped(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	columns := []string{
		"id", "name", "role", "tier", "status", "agent_card", "url",
		"parent_id", "active_tasks", "last_error_rate", "last_sample_error",
		"uptime_seconds", "current_task", "runtime", "workspace_dir", "x", "y", "collapsed",
		"budget_limit", "monthly_spend",
	}
	// Populate with non-zero financial values to confirm they are stripped.
	mock.ExpectQuery("SELECT w.id, w.name").
		WithArgs("cccccccc-0010-0000-0000-000000000000").
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow("cccccccc-0010-0000-0000-000000000000", "Finance Test", "worker", 1, "online", []byte(`{}`),
				"http://localhost:9001", nil, 0, 0.0, "", 0, "", "langgraph",
				"", 0.0, 0.0, false,
				int64(50000), int64(12500))) // budget_limit=500 USD, spend=125 USD

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "cccccccc-0010-0000-0000-000000000000"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-fin-1", nil)

	handler.Get(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if _, present := resp["budget_limit"]; present {
		t.Errorf("budget_limit must not appear in GET /workspaces/:id response (got %v)", resp["budget_limit"])
	}
	if _, present := resp["monthly_spend"]; present {
		t.Errorf("monthly_spend must not appear in GET /workspaces/:id response (got %v)", resp["monthly_spend"])
	}
	// Sanity-check that normal fields are still present.
	if resp["name"] != "Finance Test" {
		t.Errorf("expected name 'Finance Test', got %v", resp["name"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// ==================== #934 Security Auditor regressions ====================

// TestWorkspaceGet_OperationalFieldsStripped verifies that GET /workspaces/:id
// does NOT expose any of the six operational fields that were identified as
// leaking internal network topology and runtime state to unauthenticated
// callers. (#934)
func TestWorkspaceGet_OperationalFieldsStripped(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	columns := []string{
		"id", "name", "role", "tier", "status", "agent_card", "url",
		"parent_id", "active_tasks", "last_error_rate", "last_sample_error",
		"uptime_seconds", "current_task", "runtime", "workspace_dir", "x", "y", "collapsed",
		"budget_limit", "monthly_spend",
	}
	mock.ExpectQuery("SELECT w.id, w.name").
		WithArgs("cccccccc-0020-0000-0000-000000000000").
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow(
				"cccccccc-0020-0000-0000-000000000000", "Ops Test", "worker", 1, "online",
				[]byte(`{"name":"ops-agent"}`),
				"http://172.17.0.5:8080",       // url — must be stripped
				nil,                             // parent_id
				3,                               // active_tasks — must be retained
				0.12,                            // last_error_rate — must be stripped
				"connection refused",            // last_sample_error — must be stripped
				3600,                            // uptime_seconds — must be stripped
				"Summarising quarterly reports", // current_task — must be stripped
				"langgraph",
				"/host/data/workspaces/cccc", // workspace_dir — must be stripped
				100.0, 200.0, false,
				nil, int64(0),
			))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "cccccccc-0020-0000-0000-000000000000"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/cccccccc-0020-0000-0000-000000000000", nil)

	handler.Get(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// These six fields must be absent from the open endpoint response.
	stripped := []string{"url", "current_task", "uptime_seconds", "last_error_rate", "last_sample_error", "workspace_dir"}
	for _, field := range stripped {
		if _, present := resp[field]; present {
			t.Errorf("field %q must not appear in GET /workspaces/:id response (got %v)", field, resp[field])
		}
	}

	// Core fields required for canvas rendering must still be present.
	retained := map[string]interface{}{
		"id":          "cccccccc-0020-0000-0000-000000000000",
		"name":        "Ops Test",
		"status":      "online",
		"active_tasks": float64(3),
		"runtime":     "langgraph",
	}
	for field, want := range retained {
		if got := resp[field]; got != want {
			t.Errorf("field %q: got %v, want %v", field, got, want)
		}
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestWorkspaceGet_AgentCardURLScrubbed verifies that the "url" key inside
// agent_card is removed before the response is written.  The agent_card url
// holds the same internal container address as the top-level url field —
// both must be absent from the open endpoint. (#934)
func TestWorkspaceGet_AgentCardURLScrubbed(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	columns := []string{
		"id", "name", "role", "tier", "status", "agent_card", "url",
		"parent_id", "active_tasks", "last_error_rate", "last_sample_error",
		"uptime_seconds", "current_task", "runtime", "workspace_dir", "x", "y", "collapsed",
		"budget_limit", "monthly_spend",
	}
	// agent_card contains a "url" key pointing to the internal container address.
	agentCard := []byte(`{"name":"scrub-agent","url":"http://172.17.0.7:8080","version":"1.0","skills":[]}`)
	mock.ExpectQuery("SELECT w.id, w.name").
		WithArgs("cccccccc-0021-0000-0000-000000000000").
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow(
				"cccccccc-0021-0000-0000-000000000000", "Card Scrub Test", "worker", 1, "online",
				agentCard,
				"http://172.17.0.7:8080",
				nil, 0, 0.0, "", 0, "", "langgraph", "",
				0.0, 0.0, false,
				nil, int64(0),
			))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "cccccccc-0021-0000-0000-000000000000"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/cccccccc-0021-0000-0000-000000000000", nil)

	handler.Get(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Top-level url must be stripped.
	if _, present := resp["url"]; present {
		t.Errorf("top-level url must not appear in open GET /workspaces/:id response")
	}

	// agent_card must be present but must NOT contain a "url" key.
	rawCard, ok := resp["agent_card"]
	if !ok {
		t.Fatal("agent_card must be present in response")
	}
	// Gin serialises json.RawMessage inline — re-marshal and unmarshal to inspect.
	cardBytes, err := json.Marshal(rawCard)
	if err != nil {
		t.Fatalf("failed to re-marshal agent_card: %v", err)
	}
	var card map[string]interface{}
	if err := json.Unmarshal(cardBytes, &card); err != nil {
		t.Fatalf("failed to parse agent_card: %v", err)
	}
	if _, present := card["url"]; present {
		t.Errorf("agent_card.url must be scrubbed from open GET /workspaces/:id response (got %v)", card["url"])
	}
	// Other agent_card fields should be intact.
	if card["name"] != "scrub-agent" {
		t.Errorf("agent_card.name: got %v, want scrub-agent", card["name"])
	}
	if card["version"] != "1.0" {
		t.Errorf("agent_card.version: got %v, want 1.0", card["version"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestWorkspaceUpdate_BudgetLimitIgnored verifies that including budget_limit
// in a PATCH /workspaces/:id body does NOT trigger a DB write. The only write
// path for budget_limit is PATCH /workspaces/:id/budget (AdminAuth-gated).
// Any workspace bearer must not be able to self-clear its spending ceiling.
// (#611 Fix 1)
func TestWorkspaceUpdate_BudgetLimitIgnored(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	// Only the existence probe fires — no UPDATE for budget_limit.
	mock.ExpectQuery("SELECT EXISTS.*workspaces WHERE id").
		WithArgs("cccccccc-0011-0000-0000-000000000000").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	// name update is the only expected write
	mock.ExpectExec("UPDATE workspaces SET name").
		WithArgs("cccccccc-0011-0000-0000-000000000000", "Safe Name").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "cccccccc-0011-0000-0000-000000000000"}}
	// Send budget_limit alongside an innocuous field.
	body := `{"name":"Safe Name","budget_limit":null}`
	c.Request = httptest.NewRequest("PATCH", "/workspaces/ws-budget-test",
		bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Update(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// sqlmock will fail if any unexpected DB call was made (e.g. for budget_limit).
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected DB call — budget_limit must not be written via Update: %v", err)
	}
}

// TestWorkspaceUpdate_BudgetLimitOnly_Ignored verifies that a body containing
// ONLY budget_limit results in no DB writes at all (besides the existence probe).
func TestWorkspaceUpdate_BudgetLimitOnly_Ignored(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	mock.ExpectQuery("SELECT EXISTS.*workspaces WHERE id").
		WithArgs("cccccccc-0012-0000-0000-000000000000").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	// No UPDATE expected — budget_limit must be silently skipped.

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "cccccccc-0012-0000-0000-000000000000"}}
	c.Request = httptest.NewRequest("PATCH", "/workspaces/ws-budget-only",
		bytes.NewBufferString(`{"budget_limit":999999}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Update(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected DB call for budget_limit: %v", err)
	}
}
