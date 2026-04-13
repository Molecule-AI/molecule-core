package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

// ==================== POST /workspaces/:id/restart — additional coverage ====================

func TestRestartHandler_WorkspaceNotFoundReturns404(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	mock.ExpectQuery("SELECT status, name, tier, COALESCE").
		WithArgs("ws-nonexistent").
		WillReturnError(sql.ErrNoRows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-nonexistent"}}
	c.Request = httptest.NewRequest("POST", "/workspaces/ws-nonexistent/restart", nil)

	handler.Restart(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestRestartHandler_DBConnectionError(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	mock.ExpectQuery("SELECT status, name, tier, COALESCE").
		WithArgs("ws-conn-err").
		WillReturnError(sql.ErrConnDone)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-conn-err"}}
	c.Request = httptest.NewRequest("POST", "/workspaces/ws-conn-err/restart", nil)

	handler.Restart(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestRestartHandler_AncestorPausedBlocksRestart(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	// Lookup workspace
	mock.ExpectQuery("SELECT status, name, tier, COALESCE").
		WithArgs("ws-grandchild").
		WillReturnRows(sqlmock.NewRows([]string{"status", "name", "tier", "runtime"}).
			AddRow("offline", "Grandchild Agent", 1, "langgraph"))

	// isParentPaused: get parent_id of grandchild -> child
	mock.ExpectQuery("SELECT parent_id FROM workspaces WHERE id =").
		WithArgs("ws-grandchild").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow("ws-mid"))

	// isParentPaused: check child's status (online, not paused)
	mock.ExpectQuery("SELECT status, name FROM workspaces WHERE id =").
		WithArgs("ws-mid").
		WillReturnRows(sqlmock.NewRows([]string{"status", "name"}).AddRow("online", "Middle Agent"))

	// Recursive: isParentPaused for ws-mid -> ws-root
	mock.ExpectQuery("SELECT parent_id FROM workspaces WHERE id =").
		WithArgs("ws-mid").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow("ws-root"))

	// isParentPaused: ws-root is paused
	mock.ExpectQuery("SELECT status, name FROM workspaces WHERE id =").
		WithArgs("ws-root").
		WillReturnRows(sqlmock.NewRows([]string{"status", "name"}).AddRow("paused", "Root Agent"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-grandchild"}}
	c.Request = httptest.NewRequest("POST", "/workspaces/ws-grandchild/restart", nil)

	handler.Restart(c)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if errMsg, ok := resp["error"].(string); !ok || !strings.Contains(errMsg, "paused") {
		t.Errorf("expected error about paused grandparent, got %v", resp["error"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestRestartHandler_NilProvisionerReturns503(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	mock.ExpectQuery("SELECT status, name, tier, COALESCE").
		WithArgs("ws-no-prov").
		WillReturnRows(sqlmock.NewRows([]string{"status", "name", "tier", "runtime"}).
			AddRow("offline", "Test Agent", 1, "langgraph"))

	// isParentPaused: no parent
	mock.ExpectQuery("SELECT parent_id FROM workspaces WHERE id =").
		WithArgs("ws-no-prov").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-no-prov"}}
	c.Request = httptest.NewRequest("POST", "/workspaces/ws-no-prov/restart", nil)

	handler.Restart(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// ==================== POST /workspaces/:id/pause — additional coverage ====================

func TestPauseHandler_WorkspaceNotFoundReturns404(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	mock.ExpectQuery("SELECT status, name FROM workspaces WHERE id =").
		WithArgs("ws-pause-gone").
		WillReturnError(sql.ErrNoRows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-pause-gone"}}
	c.Request = httptest.NewRequest("POST", "/workspaces/ws-pause-gone/pause", nil)

	handler.Pause(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestPauseHandler_DBConnectionError(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	mock.ExpectQuery("SELECT status, name FROM workspaces WHERE id =").
		WithArgs("ws-pause-dberr").
		WillReturnError(sql.ErrConnDone)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-pause-dberr"}}
	c.Request = httptest.NewRequest("POST", "/workspaces/ws-pause-dberr/pause", nil)

	handler.Pause(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestPauseHandler_SuccessNoChildren(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	mock.ExpectQuery("SELECT status, name FROM workspaces WHERE id =").
		WithArgs("ws-pause-ok").
		WillReturnRows(sqlmock.NewRows([]string{"status", "name"}).AddRow("online", "Agent A"))

	mock.ExpectQuery("WITH RECURSIVE descendants").
		WithArgs("ws-pause-ok").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}))

	mock.ExpectExec("UPDATE workspaces SET status = 'paused'").
		WithArgs("ws-pause-ok").
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec("INSERT INTO structure_events").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-pause-ok"}}
	c.Request = httptest.NewRequest("POST", "/workspaces/ws-pause-ok/pause", nil)

	handler.Pause(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "paused" {
		t.Errorf("expected status 'paused', got %v", resp["status"])
	}
	if count, ok := resp["paused_count"].(float64); !ok || count != 1 {
		t.Errorf("expected paused_count 1, got %v", resp["paused_count"])
	}
}

// ==================== POST /workspaces/:id/resume — additional coverage ====================

func TestResumeHandler_NotPausedReturns404(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	mock.ExpectQuery("SELECT name, tier, COALESCE").
		WithArgs("ws-resume-notpaused").
		WillReturnError(sql.ErrNoRows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-resume-notpaused"}}
	c.Request = httptest.NewRequest("POST", "/workspaces/ws-resume-notpaused/resume", nil)

	handler.Resume(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestResumeHandler_DBConnectionError(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	mock.ExpectQuery("SELECT name, tier, COALESCE").
		WithArgs("ws-resume-dberr").
		WillReturnError(sql.ErrConnDone)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-resume-dberr"}}
	c.Request = httptest.NewRequest("POST", "/workspaces/ws-resume-dberr/resume", nil)

	handler.Resume(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestResumeHandler_NilProvisionerReturns503(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	mock.ExpectQuery("SELECT name, tier, COALESCE").
		WithArgs("ws-resume-noprov").
		WillReturnRows(sqlmock.NewRows([]string{"name", "tier", "runtime"}).
			AddRow("Test Agent", 1, "langgraph"))

	// provisioner nil check happens BEFORE isParentPaused, so no parent query expected

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-resume-noprov"}}
	c.Request = httptest.NewRequest("POST", "/workspaces/ws-resume-noprov/resume", nil)

	handler.Resume(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// Note: TestResumeHandler_ParentPausedBlocksResume requires a non-nil provisioner
// (Resume checks provisioner before isParentPaused). This is covered in
// handlers_additional_test.go's integration-style tests.
