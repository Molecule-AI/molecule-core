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
	}
	mock.ExpectQuery("SELECT w.id, w.name").
		WithArgs("ws-get-1").
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow("ws-get-1", "My Agent", "worker", 1, "online", []byte(`{"name":"test"}`),
				"http://localhost:8001", nil, 2, 0.05, "", 3600, "working", "langgraph",
				"", 10.0, 20.0, false))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-get-1"}}
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
		WithArgs("ws-nonexistent").
		WillReturnError(sql.ErrNoRows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-nonexistent"}}
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
		WithArgs("ws-dberr").
		WillReturnError(sql.ErrConnDone)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-dberr"}}
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

	// Workspace INSERT fails
	mock.ExpectExec("INSERT INTO workspaces").
		WithArgs(sqlmock.AnyArg(), "Failing Agent", nil, 1, "langgraph", sqlmock.AnyArg(), (*string)(nil), nil, "none").
		WillReturnError(sql.ErrConnDone)

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

	// Expect workspace INSERT with defaulted tier=1, runtime="langgraph"
	mock.ExpectExec("INSERT INTO workspaces").
		WithArgs(sqlmock.AnyArg(), "Default Agent", nil, 1, "langgraph", sqlmock.AnyArg(), (*string)(nil), nil, "none").
		WillReturnResult(sqlmock.NewResult(0, 1))

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
	c.Params = gin.Params{{Key: "id", Value: "ws-upd"}}
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

	// Expect name, role, and tier updates
	mock.ExpectExec("UPDATE workspaces SET name").
		WithArgs("ws-multi", "Updated Agent").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE workspaces SET role").
		WithArgs("ws-multi", "manager").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE workspaces SET tier").
		WithArgs("ws-multi", float64(3)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-multi"}}

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

	mock.ExpectExec("UPDATE workspaces SET runtime").
		WithArgs("ws-rt", "claude-code").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-rt"}}

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
		WithArgs("ws-parent").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow("ws-child-1", "Child One").
			AddRow("ws-child-2", "Child Two"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-parent"}}
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
		WithArgs("ws-parent-del").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow("ws-child-del", "Child Agent"))

	// Descendant CTE query returns the recursive set (1 descendant: ws-child-del)
	mock.ExpectQuery("WITH RECURSIVE descendants").
		WithArgs("ws-parent-del").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("ws-child-del"))

	// #73: single batch UPDATE covering [self + descendants] BEFORE stopping
	// containers (prevents heartbeat/restart resurrection races).
	mock.ExpectExec("UPDATE workspaces SET status = 'removed'").
		WillReturnResult(sqlmock.NewResult(2, 2))
	// Batch canvas_layouts DELETE for the same id set.
	mock.ExpectExec("DELETE FROM canvas_layouts WHERE workspace_id = ANY").
		WillReturnResult(sqlmock.NewResult(2, 2))
	// Broadcast for child WORKSPACE_REMOVED (fires during the descendant loop).
	mock.ExpectExec("INSERT INTO structure_events").
		WillReturnResult(sqlmock.NewResult(0, 1))
	// Broadcast for parent WORKSPACE_REMOVED (fires after parent cleanup).
	mock.ExpectExec("INSERT INTO structure_events").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-parent-del"}}
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
		WithArgs("ws-err-del").
		WillReturnError(sql.ErrConnDone)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-err-del"}}
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
	mock.ExpectQuery(`SELECT id, workspace_id FROM workspace_auth_tokens`).
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
