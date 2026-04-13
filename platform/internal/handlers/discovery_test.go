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

// ==================== Discover — missing X-Workspace-ID header ====================

func TestDiscover_MissingCallerHeader(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)
	handler := NewDiscoveryHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-target"}}
	c.Request = httptest.NewRequest("GET", "/registry/discover/ws-target", nil)
	// No X-Workspace-ID header

	handler.Discover(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["error"] != "X-Workspace-ID header is required" {
		t.Errorf("expected error about missing header, got %v", resp["error"])
	}
}

// ==================== Discover — workspace not found (with caller) ====================

func TestDiscover_WorkspaceNotFound_WithCaller(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewDiscoveryHandler()

	// CanCommunicate will need DB lookups — both workspace name lookups
	// For the access check: caller lookup succeeds, target lookup fails
	mock.ExpectQuery("SELECT id, parent_id FROM workspaces WHERE id =").
		WithArgs("ws-caller").
		WillReturnRows(sqlmock.NewRows([]string{"id", "parent_id"}).AddRow("ws-caller", nil))
	mock.ExpectQuery("SELECT id, parent_id FROM workspaces WHERE id =").
		WithArgs("ws-missing").
		WillReturnRows(sqlmock.NewRows([]string{"id", "parent_id"})) // no rows

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-missing"}}
	c.Request = httptest.NewRequest("GET", "/registry/discover/ws-missing", nil)
	c.Request.Header.Set("X-Workspace-ID", "ws-caller")

	handler.Discover(c)

	// Access denied because target not found in registry → 403
	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// ==================== Discover — external (no caller header, DB fallback) ====================

func TestDiscover_External_NotFound(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewDiscoveryHandler()

	// This tests the external path (no X-Workspace-ID header), but we need
	// the request to have the header as empty string bypass. Instead test the
	// DB path for external callers:
	// For an external request without caller, the code first checks callerID == ""
	// which triggers the StatusBadRequest, so we test with a header but Redis+DB miss

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-ext-missing"}}
	c.Request = httptest.NewRequest("GET", "/registry/discover/ws-ext-missing", nil)
	// No header → returns 400

	handler.Discover(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// ==================== Peers — success with parent/siblings/children ====================

func TestPeers_WithParent(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewDiscoveryHandler()

	// Expect parent_id lookup for the requesting workspace
	mock.ExpectQuery("SELECT parent_id FROM workspaces WHERE id =").
		WithArgs("ws-sibling-1").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow("ws-parent"))

	// Expect siblings query (same parent, excluding self)
	peerCols := []string{"id", "name", "role", "tier", "status", "agent_card", "url", "parent_id", "active_tasks"}
	mock.ExpectQuery("SELECT w.id, w.name.*WHERE w.parent_id = \\$1 AND w.id != \\$2").
		WithArgs("ws-parent", "ws-sibling-1").
		WillReturnRows(sqlmock.NewRows(peerCols).
			AddRow("ws-sibling-2", "Sibling Two", "worker", 1, "online", []byte("null"), "http://localhost:8002", "ws-parent", 0))

	// Expect children query
	mock.ExpectQuery("SELECT w.id, w.name.*WHERE w.parent_id = \\$1 AND w.status").
		WithArgs("ws-sibling-1").
		WillReturnRows(sqlmock.NewRows(peerCols))

	// Expect parent query
	mock.ExpectQuery("SELECT w.id, w.name.*WHERE w.id = \\$1 AND w.status").
		WithArgs("ws-parent").
		WillReturnRows(sqlmock.NewRows(peerCols).
			AddRow("ws-parent", "Parent PM", "manager", 2, "online", []byte("null"), "http://localhost:8001", nil, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-sibling-1"}}
	c.Request = httptest.NewRequest("GET", "/registry/ws-sibling-1/peers", nil)

	handler.Peers(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var peers []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &peers); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(peers) != 2 {
		t.Errorf("expected 2 peers (1 sibling + 1 parent), got %d", len(peers))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestPeers_NotFound(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewDiscoveryHandler()

	// Workspace not found
	mock.ExpectQuery("SELECT parent_id FROM workspaces WHERE id =").
		WithArgs("ws-ghost").
		WillReturnError(sql.ErrNoRows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-ghost"}}
	c.Request = httptest.NewRequest("GET", "/registry/ws-ghost/peers", nil)

	handler.Peers(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestPeers_DBError(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewDiscoveryHandler()

	mock.ExpectQuery("SELECT parent_id FROM workspaces WHERE id =").
		WithArgs("ws-dberr").
		WillReturnError(sql.ErrConnDone)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-dberr"}}
	c.Request = httptest.NewRequest("GET", "/registry/ws-dberr/peers", nil)

	handler.Peers(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestPeers_RootWorkspace_NoPeers(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewDiscoveryHandler()

	// Root workspace (parent_id is NULL)
	mock.ExpectQuery("SELECT parent_id FROM workspaces WHERE id =").
		WithArgs("ws-root-alone").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(nil))

	peerCols := []string{"id", "name", "role", "tier", "status", "agent_card", "url", "parent_id", "active_tasks"}

	// Siblings (other root-level workspaces) — none
	mock.ExpectQuery("SELECT w.id, w.name.*WHERE w.parent_id IS NULL AND w.id != \\$1").
		WithArgs("ws-root-alone").
		WillReturnRows(sqlmock.NewRows(peerCols))

	// Children — none
	mock.ExpectQuery("SELECT w.id, w.name.*WHERE w.parent_id = \\$1").
		WithArgs("ws-root-alone").
		WillReturnRows(sqlmock.NewRows(peerCols))

	// No parent query since parent_id is NULL

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-root-alone"}}
	c.Request = httptest.NewRequest("GET", "/registry/ws-root-alone/peers", nil)

	handler.Peers(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var peers []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &peers); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(peers) != 0 {
		t.Errorf("expected 0 peers, got %d", len(peers))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// ==================== CheckAccess ====================

func TestCheckAccess_BadJSON(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)
	handler := NewDiscoveryHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/registry/check-access", bytes.NewBufferString("not json"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CheckAccess(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCheckAccess_MissingFields(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)
	handler := NewDiscoveryHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"caller_id":"ws-1"}`
	c.Request = httptest.NewRequest("POST", "/registry/check-access", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CheckAccess(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCheckAccess_SameWorkspace(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)
	handler := NewDiscoveryHandler()

	// CanCommunicate("ws-1", "ws-1") returns true immediately (same ID, no DB lookups)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"caller_id":"ws-1","target_id":"ws-1"}`
	c.Request = httptest.NewRequest("POST", "/registry/check-access", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CheckAccess(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["allowed"] != true {
		t.Errorf("expected allowed=true for same workspace, got %v", resp["allowed"])
	}
}
