package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

// ---------- ApprovalsHandler: Create ----------

func TestApprovals_Create_Success(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewApprovalsHandler(broadcaster)

	// Insert approval
	mock.ExpectQuery("INSERT INTO approval_requests").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("appr-1"))

	// RecordAndBroadcast for APPROVAL_REQUESTED
	mock.ExpectExec("INSERT INTO structure_events").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Parent lookup (no parent — nil)
	mock.ExpectQuery("SELECT parent_id FROM workspaces WHERE id").
		WithArgs("ws-1").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(nil))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}
	body := `{"action":"run_script","reason":"need to execute","task_id":"task-99"}`
	c.Request = httptest.NewRequest("POST", "/", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["approval_id"] != "appr-1" {
		t.Errorf("expected approval_id appr-1, got %v", resp["approval_id"])
	}
	if resp["status"] != "pending" {
		t.Errorf("expected status 'pending', got %v", resp["status"])
	}
}

func TestApprovals_Create_WithParentEscalation(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewApprovalsHandler(broadcaster)

	// Insert approval
	mock.ExpectQuery("INSERT INTO approval_requests").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("appr-2"))

	// APPROVAL_REQUESTED broadcast
	mock.ExpectExec("INSERT INTO structure_events").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Parent lookup — returns parent-ws
	parentID := "parent-ws"
	mock.ExpectQuery("SELECT parent_id FROM workspaces WHERE id").
		WithArgs("child-ws").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(&parentID))

	// APPROVAL_ESCALATED broadcast to parent
	mock.ExpectExec("INSERT INTO structure_events").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "child-ws"}}
	body := `{"action":"delete_file","reason":"cleanup"}`
	c.Request = httptest.NewRequest("POST", "/", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestApprovals_Create_MissingAction(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)
	handler := NewApprovalsHandler(newTestBroadcaster())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}
	c.Request = httptest.NewRequest("POST", "/", bytes.NewBufferString(`{"reason":"no action"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ---------- ApprovalsHandler: ListAll ----------

func TestApprovals_ListAll_Empty(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewApprovalsHandler(newTestBroadcaster())

	// Auto-expire stale approvals
	mock.ExpectExec("UPDATE approval_requests SET status = 'denied'").
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Query all pending
	mock.ExpectQuery("SELECT a.id, a.workspace_id, w.name, a.action, a.reason, a.status, a.created_at FROM approval_requests a").
		WillReturnRows(sqlmock.NewRows([]string{"id", "workspace_id", "name", "action", "reason", "status", "created_at"}))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/approvals/pending", nil)

	handler.ListAll(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var result []interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if len(result) != 0 {
		t.Errorf("expected empty list, got %v", result)
	}
}

func TestApprovals_ListAll_WithResults(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewApprovalsHandler(newTestBroadcaster())

	mock.ExpectExec("UPDATE approval_requests SET status = 'denied'").
		WillReturnResult(sqlmock.NewResult(0, 0))

	reason := "needs review"
	rows := sqlmock.NewRows([]string{"id", "workspace_id", "name", "action", "reason", "status", "created_at"}).
		AddRow("appr-1", "ws-1", "Test WS", "run_script", &reason, "pending", "2024-01-01T00:00:00Z").
		AddRow("appr-2", "ws-2", "Test WS 2", "delete_file", nil, "pending", "2024-01-02T00:00:00Z")

	mock.ExpectQuery("SELECT a.id, a.workspace_id, w.name").
		WillReturnRows(rows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/approvals/pending", nil)

	handler.ListAll(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result []interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if len(result) != 2 {
		t.Errorf("expected 2 approvals, got %d", len(result))
	}
}

// ---------- ApprovalsHandler: List ----------

func TestApprovals_List_ForWorkspace(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewApprovalsHandler(newTestBroadcaster())

	decidedBy := "admin"
	decidedAt := "2024-01-01T00:00:00Z"
	taskID := "task-1"
	reason := "safety check"
	rows := sqlmock.NewRows([]string{"id", "task_id", "action", "reason", "status", "decided_by", "decided_at", "created_at"}).
		AddRow("appr-5", &taskID, "run", &reason, "approved", &decidedBy, &decidedAt, "2024-01-01T00:00:00Z")

	mock.ExpectQuery("SELECT id, task_id, action, reason, status, decided_by, decided_at, created_at FROM approval_requests WHERE workspace_id").
		WithArgs("ws-1").
		WillReturnRows(rows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-1/approvals", nil)

	handler.List(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result []interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if len(result) != 1 {
		t.Errorf("expected 1 approval, got %d", len(result))
	}
}

// ---------- ApprovalsHandler: Decide ----------

func TestApprovals_Decide_Approved(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewApprovalsHandler(broadcaster)

	mock.ExpectExec("UPDATE approval_requests SET status").
		WithArgs("approved", "human", "appr-1", "ws-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec("INSERT INTO structure_events").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}, {Key: "approvalId", Value: "appr-1"}}
	c.Request = httptest.NewRequest("POST", "/",
		bytes.NewBufferString(`{"decision":"approved"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Decide(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "approved" {
		t.Errorf("expected status 'approved', got %v", resp["status"])
	}
}

func TestApprovals_Decide_Denied(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewApprovalsHandler(broadcaster)

	mock.ExpectExec("UPDATE approval_requests SET status").
		WithArgs("denied", "supervisor", "appr-2", "ws-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec("INSERT INTO structure_events").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}, {Key: "approvalId", Value: "appr-2"}}
	c.Request = httptest.NewRequest("POST", "/",
		bytes.NewBufferString(`{"decision":"denied","decided_by":"supervisor"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Decide(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestApprovals_Decide_NotFound(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewApprovalsHandler(newTestBroadcaster())

	mock.ExpectExec("UPDATE approval_requests SET status").
		WithArgs("approved", "human", "appr-none", "ws-1").
		WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows affected

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}, {Key: "approvalId", Value: "appr-none"}}
	c.Request = httptest.NewRequest("POST", "/",
		bytes.NewBufferString(`{"decision":"approved"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Decide(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestApprovals_Decide_InvalidDecision(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)
	handler := NewApprovalsHandler(newTestBroadcaster())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}, {Key: "approvalId", Value: "appr-1"}}
	c.Request = httptest.NewRequest("POST", "/",
		bytes.NewBufferString(`{"decision":"maybe"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Decide(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestApprovals_Decide_MissingDecision(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)
	handler := NewApprovalsHandler(newTestBroadcaster())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}, {Key: "approvalId", Value: "appr-1"}}
	c.Request = httptest.NewRequest("POST", "/", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Decide(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
