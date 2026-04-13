package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/gin-gonic/gin"
)

func TestSessionSearchReturnsActivityAndMemory(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	broadcaster := newTestBroadcaster()
	handler := NewActivityHandler(broadcaster)

	rows := sqlmock.NewRows([]string{
		"kind", "id", "workspace_id", "label", "content", "method", "status", "request_body", "response_body", "created_at",
	}).
		AddRow("activity", "act-1", "ws-123", "task_update", "Working on docs", "POST", "ok", `{"task":"docs"}`, `{"ok":true}`, time.Now()).
		AddRow("activity", "act-2", "ws-123", "skill_promotion", "Promoted repeatable workflow", "memory/skill-promotion", "ok", `{"promote_to_skill":true}`, `{"id":"mem-2"}`, time.Now()).
		AddRow("memory", "mem-1", "ws-123", "TEAM", "remember the docs path", "", "", nil, nil, time.Now())

	mock.ExpectQuery("WITH session_items AS").
		WithArgs("ws-123", "%docs%", 50).
		WillReturnRows(rows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-123/session-search?q=docs", bytes.NewBufferString(""))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "ws-123"}}

	handler.SessionSearch(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp) != 3 {
		t.Fatalf("expected 3 results, got %d", len(resp))
	}
	if resp[0]["kind"] != "activity" || resp[1]["kind"] != "activity" || resp[2]["kind"] != "memory" {
		t.Fatalf("unexpected result kinds: %#v", resp)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// ---------- Activity List source filter ----------

func TestActivityList_SourceCanvas(t *testing.T) {
	mock := setupTestDB(t)
	broadcaster := newTestBroadcaster()
	handler := NewActivityHandler(broadcaster)

	// Expect query with "source_id IS NULL"
	mock.ExpectQuery(`SELECT .+ FROM activity_logs WHERE workspace_id = .+ AND source_id IS NULL`).
		WithArgs("ws-1", 100).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "workspace_id", "activity_type", "source_id", "target_id",
			"method", "summary", "request_body", "response_body",
			"duration_ms", "status", "error_detail", "created_at",
		}))

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-1/activity?source=canvas", nil)
	handler.List(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestActivityList_SourceAgent(t *testing.T) {
	mock := setupTestDB(t)
	broadcaster := newTestBroadcaster()
	handler := NewActivityHandler(broadcaster)

	// Expect query with "source_id IS NOT NULL"
	mock.ExpectQuery(`SELECT .+ FROM activity_logs WHERE workspace_id = .+ AND source_id IS NOT NULL`).
		WithArgs("ws-1", 100).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "workspace_id", "activity_type", "source_id", "target_id",
			"method", "summary", "request_body", "response_body",
			"duration_ms", "status", "error_detail", "created_at",
		}))

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-1/activity?source=agent", nil)
	handler.List(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestActivityList_SourceInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	broadcaster := newTestBroadcaster()
	handler := NewActivityHandler(broadcaster)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-1/activity?source=bogus", nil)
	handler.List(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid source, got %d", w.Code)
	}
}

func TestActivityList_SourceWithType(t *testing.T) {
	mock := setupTestDB(t)
	broadcaster := newTestBroadcaster()
	handler := NewActivityHandler(broadcaster)

	// Both type and source filters
	mock.ExpectQuery(`SELECT .+ FROM activity_logs WHERE workspace_id = .+ AND activity_type = .+ AND source_id IS NULL`).
		WithArgs("ws-1", "a2a_receive", 100).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "workspace_id", "activity_type", "source_id", "target_id",
			"method", "summary", "request_body", "response_body",
			"duration_ms", "status", "error_detail", "created_at",
		}))

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-1/activity?type=a2a_receive&source=canvas", nil)
	handler.List(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// ---------- Activity type allowlist (#125: memory_write added) ----------

func TestActivityReport_AcceptsMemoryWriteType(t *testing.T) {
	mockDB, mock, _ := sqlmock.New()
	defer mockDB.Close()
	db.DB = mockDB

	mock.ExpectExec(`INSERT INTO activity_logs`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	broadcaster := newTestBroadcaster()
	handler := NewActivityHandler(broadcaster)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-mem"}}
	body := `{"workspace_id":"ws-mem","activity_type":"memory_write","summary":"[LOCAL] x","status":"ok"}`
	c.Request = httptest.NewRequest("POST", "/workspaces/ws-mem/activity", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	handler.Report(c)

	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Errorf("memory_write should be accepted; got %d: %s", w.Code, w.Body.String())
	}
}

func TestActivityReport_RejectsUnknownType(t *testing.T) {
	mockDB, _, _ := sqlmock.New()
	defer mockDB.Close()
	db.DB = mockDB

	broadcaster := newTestBroadcaster()
	handler := NewActivityHandler(broadcaster)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-x"}}
	body := `{"workspace_id":"ws-x","activity_type":"made_up_type","summary":"x","status":"ok"}`
	c.Request = httptest.NewRequest("POST", "/workspaces/ws-x/activity", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	handler.Report(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("unknown type should 400; got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "memory_write") {
		t.Errorf("error message should list valid types including memory_write; got %s", w.Body.String())
	}
}
