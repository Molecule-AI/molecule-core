package handlers

// Tests for per-workspace budget_limit field and A2A enforcement (#541).
//
// Coverage:
//   - GET /workspaces/:id includes budget_limit (nil when unset, int when set)
//   - GET /workspaces/:id includes monthly_spend
//   - POST /workspaces creates workspace with budget_limit
//   - PATCH /workspaces/:id updates budget_limit (nil clears the ceiling)
//   - A2A proxy returns 429 when monthly_spend >= budget_limit
//   - A2A proxy passes through when monthly_spend < budget_limit
//   - A2A proxy passes through when budget_limit is NULL (no limit)
//   - A2A proxy fail-open on DB error during budget check

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

// wsColumns is the canonical column list for scanWorkspaceRow tests.
var wsColumns = []string{
	"id", "name", "role", "tier", "status", "agent_card", "url",
	"parent_id", "active_tasks", "last_error_rate", "last_sample_error",
	"uptime_seconds", "current_task", "runtime", "workspace_dir", "x", "y", "collapsed",
	"budget_limit", "monthly_spend",
}

// ==================== GET — budget_limit serialisation ====================

// TestWorkspaceBudget_Get_NilLimit verifies that budget_limit is null in the
// JSON response when the DB column IS NULL (no ceiling configured).
func TestWorkspaceBudget_Get_NilLimit(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewWorkspaceHandler(newTestBroadcaster(), nil, "http://localhost:8080", t.TempDir())

	mock.ExpectQuery("SELECT w.id, w.name").
		WithArgs("ws-nobudget").
		WillReturnRows(sqlmock.NewRows(wsColumns).
			AddRow("ws-nobudget", "Free Agent", "worker", 1, "online",
				[]byte(`{}`), "http://localhost:9001",
				nil, 0, 0.0, "", 0, "", "langgraph", "",
				0.0, 0.0, false,
				nil, // budget_limit NULL
				0))  // monthly_spend 0

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-nobudget"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-nobudget", nil)
	handler.Get(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["budget_limit"] != nil {
		t.Errorf("expected budget_limit=nil, got %v", resp["budget_limit"])
	}
	if resp["monthly_spend"] != float64(0) {
		t.Errorf("expected monthly_spend=0, got %v", resp["monthly_spend"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock expectations not met: %v", err)
	}
}

// TestWorkspaceBudget_Get_WithLimit verifies that a non-NULL budget_limit is
// returned as the correct integer value (USD cents) in the response.
func TestWorkspaceBudget_Get_WithLimit(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewWorkspaceHandler(newTestBroadcaster(), nil, "http://localhost:8080", t.TempDir())

	mock.ExpectQuery("SELECT w.id, w.name").
		WithArgs("ws-limited").
		WillReturnRows(sqlmock.NewRows(wsColumns).
			AddRow("ws-limited", "Capped Agent", "worker", 1, "online",
				[]byte(`{}`), "http://localhost:9002",
				nil, 0, 0.0, "", 0, "", "langgraph", "",
				0.0, 0.0, false,
				int64(500),  // budget_limit = $5.00
				int64(123))) // monthly_spend = $1.23

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-limited"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-limited", nil)
	handler.Get(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["budget_limit"] != float64(500) {
		t.Errorf("expected budget_limit=500, got %v", resp["budget_limit"])
	}
	if resp["monthly_spend"] != float64(123) {
		t.Errorf("expected monthly_spend=123, got %v", resp["monthly_spend"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock expectations not met: %v", err)
	}
}

// ==================== POST — create with budget_limit ====================

// TestWorkspaceBudget_Create_WithLimit verifies that POST /workspaces with
// a budget_limit passes the value as the 10th INSERT parameter ($10).
func TestWorkspaceBudget_Create_WithLimit(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewWorkspaceHandler(newTestBroadcaster(), nil, "http://localhost:8080", t.TempDir())

	budgetVal := int64(1000) // $10.00
	mock.ExpectExec("INSERT INTO workspaces").
		WithArgs(
			sqlmock.AnyArg(), // id
			"Budgeted Agent", // name
			nil,              // role
			1,                // tier
			"langgraph",      // runtime
			sqlmock.AnyArg(), // awareness_namespace
			(*string)(nil),   // parent_id
			nil,              // workspace_dir
			"none",           // workspace_access
			&budgetVal,       // budget_limit
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO canvas_layouts").
		WithArgs(sqlmock.AnyArg(), float64(0), float64(0)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO structure_events").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"name":"Budgeted Agent","budget_limit":1000}`
	c.Request = httptest.NewRequest("POST", "/workspaces", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	handler.Create(c)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock expectations not met: %v", err)
	}
}

// ==================== PATCH — update budget_limit ====================

// TestWorkspaceBudget_Update_SetLimit verifies that PATCH /workspaces/:id with
// budget_limit=500 issues an UPDATE workspaces SET budget_limit = 500.
func TestWorkspaceBudget_Update_SetLimit(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewWorkspaceHandler(newTestBroadcaster(), nil, "http://localhost:8080", t.TempDir())

	// Existence probe
	mock.ExpectQuery("SELECT EXISTS.*workspaces WHERE id").
		WithArgs("ws-upd-budget").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	// budget_limit UPDATE
	mock.ExpectExec("UPDATE workspaces SET budget_limit").
		WithArgs("ws-upd-budget", int64(500)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-upd-budget"}}
	body := `{"budget_limit":500}`
	c.Request = httptest.NewRequest("PATCH", "/workspaces/ws-upd-budget", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	handler.Update(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock expectations not met: %v", err)
	}
}

// TestWorkspaceBudget_Update_ClearLimit verifies that PATCH /workspaces/:id
// with budget_limit=null issues an UPDATE with NULL, clearing the ceiling.
func TestWorkspaceBudget_Update_ClearLimit(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewWorkspaceHandler(newTestBroadcaster(), nil, "http://localhost:8080", t.TempDir())

	mock.ExpectQuery("SELECT EXISTS.*workspaces WHERE id").
		WithArgs("ws-clear-budget").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	// NULL clears the budget ceiling
	mock.ExpectExec("UPDATE workspaces SET budget_limit").
		WithArgs("ws-clear-budget", nil).
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-clear-budget"}}
	body := `{"budget_limit":null}`
	c.Request = httptest.NewRequest("PATCH", "/workspaces/ws-clear-budget", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	handler.Update(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock expectations not met: %v", err)
	}
}

// ==================== A2A enforcement ====================

// TestWorkspaceBudget_A2A_ExceededReturns429 verifies that the A2A proxy
// returns HTTP 429 {"error":"workspace budget limit exceeded"} when
// monthly_spend equals budget_limit.
func TestWorkspaceBudget_A2A_ExceededReturns429(t *testing.T) {
	mock := setupTestDB(t)
	mr := setupTestRedis(t)
	handler := NewWorkspaceHandler(newTestBroadcaster(), nil, "http://localhost:8080", t.TempDir())

	// Cache a URL so resolveAgentURL doesn't need a DB query after budget check
	mr.Set(fmt.Sprintf("ws:%s:url", "ws-over-budget"), "http://localhost:9999")

	// Budget check query: spend = limit → exceeded
	mock.ExpectQuery("SELECT budget_limit, COALESCE").
		WithArgs("ws-over-budget").
		WillReturnRows(sqlmock.NewRows([]string{"budget_limit", "monthly_spend"}).
			AddRow(int64(500), int64(500)))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-over-budget"}}
	body := `{"message":{"role":"user","parts":[{"text":"hello"}]}}`
	c.Request = httptest.NewRequest("POST", "/workspaces/ws-over-budget/a2a", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	handler.ProxyA2A(c)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 when budget exceeded, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "workspace budget limit exceeded" {
		t.Errorf("expected 'workspace budget limit exceeded', got %v", resp["error"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock expectations not met: %v", err)
	}
}

// TestWorkspaceBudget_A2A_AboveLimitReturns429 verifies 429 when spend > limit.
func TestWorkspaceBudget_A2A_AboveLimitReturns429(t *testing.T) {
	mock := setupTestDB(t)
	mr := setupTestRedis(t)
	handler := NewWorkspaceHandler(newTestBroadcaster(), nil, "http://localhost:8080", t.TempDir())

	mr.Set(fmt.Sprintf("ws:%s:url", "ws-way-over"), "http://localhost:9999")

	// spend > limit
	mock.ExpectQuery("SELECT budget_limit, COALESCE").
		WithArgs("ws-way-over").
		WillReturnRows(sqlmock.NewRows([]string{"budget_limit", "monthly_spend"}).
			AddRow(int64(100), int64(9999)))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-way-over"}}
	body := `{"message":{"role":"user","parts":[{"text":"test"}]}}`
	c.Request = httptest.NewRequest("POST", "/workspaces/ws-way-over/a2a", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	handler.ProxyA2A(c)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 when spend > limit, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock expectations not met: %v", err)
	}
}

// TestWorkspaceBudget_A2A_UnderLimitPassesThrough verifies that A2A calls
// succeed normally when monthly_spend is below budget_limit.
func TestWorkspaceBudget_A2A_UnderLimitPassesThrough(t *testing.T) {
	mock := setupTestDB(t)
	mr := setupTestRedis(t)
	handler := NewWorkspaceHandler(newTestBroadcaster(), nil, "http://localhost:8080", t.TempDir())

	// Stand up a minimal mock agent that returns a valid A2A response
	agentServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":"1","result":{"status":"ok"}}`)
	}))
	defer agentServer.Close()

	mr.Set(fmt.Sprintf("ws:%s:url", "ws-under-budget"), agentServer.URL)

	// Budget check: spend (100) < limit (500) → pass-through
	mock.ExpectQuery("SELECT budget_limit, COALESCE").
		WithArgs("ws-under-budget").
		WillReturnRows(sqlmock.NewRows([]string{"budget_limit", "monthly_spend"}).
			AddRow(int64(500), int64(100)))

	// Activity log INSERT from logA2ASuccess
	mock.ExpectExec("INSERT INTO activity_logs").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-under-budget"}}
	body := `{"jsonrpc":"2.0","id":"1","method":"message/send","params":{"message":{"role":"user","parts":[{"text":"hello"}]}}}`
	c.Request = httptest.NewRequest("POST", "/workspaces/ws-under-budget/a2a", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	handler.ProxyA2A(c)

	// Give the async logA2ASuccess goroutine a moment to fire
	time.Sleep(50 * time.Millisecond)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when under budget, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock expectations not met: %v", err)
	}
}

// TestWorkspaceBudget_A2A_NilLimitPassesThrough verifies that when
// budget_limit IS NULL (no ceiling set), A2A calls pass through unconditionally.
func TestWorkspaceBudget_A2A_NilLimitPassesThrough(t *testing.T) {
	mock := setupTestDB(t)
	mr := setupTestRedis(t)
	handler := NewWorkspaceHandler(newTestBroadcaster(), nil, "http://localhost:8080", t.TempDir())

	agentServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":"2","result":{"status":"ok"}}`)
	}))
	defer agentServer.Close()

	mr.Set(fmt.Sprintf("ws:%s:url", "ws-no-limit"), agentServer.URL)

	// budget_limit NULL → no enforcement regardless of monthly_spend
	mock.ExpectQuery("SELECT budget_limit, COALESCE").
		WithArgs("ws-no-limit").
		WillReturnRows(sqlmock.NewRows([]string{"budget_limit", "monthly_spend"}).
			AddRow(nil, int64(999999))) // huge spend but no limit set

	mock.ExpectExec("INSERT INTO activity_logs").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-no-limit"}}
	body := `{"jsonrpc":"2.0","id":"2","method":"message/send","params":{"message":{"role":"user","parts":[{"text":"hi"}]}}}`
	c.Request = httptest.NewRequest("POST", "/workspaces/ws-no-limit/a2a", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	handler.ProxyA2A(c)

	time.Sleep(50 * time.Millisecond)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when no limit set, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock expectations not met: %v", err)
	}
}

// TestWorkspaceBudget_A2A_DBErrorFailOpen verifies that a DB error during the
// budget check is fail-open — the request proceeds rather than being blocked.
func TestWorkspaceBudget_A2A_DBErrorFailOpen(t *testing.T) {
	mock := setupTestDB(t)
	mr := setupTestRedis(t)
	handler := NewWorkspaceHandler(newTestBroadcaster(), nil, "http://localhost:8080", t.TempDir())

	agentServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":"3","result":{"status":"ok"}}`)
	}))
	defer agentServer.Close()

	mr.Set(fmt.Sprintf("ws:%s:url", "ws-db-err-budget"), agentServer.URL)

	// Budget check fails with DB error → fail-open (request proceeds)
	mock.ExpectQuery("SELECT budget_limit, COALESCE").
		WithArgs("ws-db-err-budget").
		WillReturnError(sql.ErrConnDone)

	mock.ExpectExec("INSERT INTO activity_logs").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-db-err-budget"}}
	body := `{"jsonrpc":"2.0","id":"3","method":"message/send","params":{"message":{"role":"user","parts":[{"text":"fail-open test"}]}}}`
	c.Request = httptest.NewRequest("POST", "/workspaces/ws-db-err-budget/a2a", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	handler.ProxyA2A(c)

	time.Sleep(50 * time.Millisecond)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 on DB error (fail-open), got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock expectations not met: %v", err)
	}
}
