package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

// ==================== GET /workspaces/:id/memory (List) ====================

func TestMemoryList_Success(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoryHandler()

	now := time.Now()
	rows := sqlmock.NewRows([]string{"key", "value", "expires_at", "updated_at"}).
		AddRow("api-key", []byte(`"sk-123"`), nil, now).
		AddRow("count", []byte(`42`), nil, now)

	mock.ExpectQuery("SELECT key, value, expires_at, updated_at").
		WithArgs("ws-mem-1").
		WillReturnRows(rows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-mem-1"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-mem-1/memory", nil)

	handler.List(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []MemoryEntry
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("expected 2 entries, got %d", len(resp))
	}
	if resp[0].Key != "api-key" {
		t.Errorf("expected key 'api-key', got %q", resp[0].Key)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestMemoryList_Empty(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoryHandler()

	mock.ExpectQuery("SELECT key, value, expires_at, updated_at").
		WithArgs("ws-empty").
		WillReturnRows(sqlmock.NewRows([]string{"key", "value", "expires_at", "updated_at"}))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-empty"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-empty/memory", nil)

	handler.List(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp []MemoryEntry
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp) != 0 {
		t.Errorf("expected empty list, got %d entries", len(resp))
	}
}

func TestMemoryList_DBError(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoryHandler()

	mock.ExpectQuery("SELECT key, value, expires_at, updated_at").
		WithArgs("ws-dberr").
		WillReturnError(sql.ErrConnDone)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-dberr"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-dberr/memory", nil)

	handler.List(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// ==================== GET /workspaces/:id/memory/:key (Get) ====================

func TestMemoryGet_Success(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoryHandler()

	now := time.Now()
	mock.ExpectQuery("SELECT key, value, expires_at, updated_at").
		WithArgs("ws-get", "api-key").
		WillReturnRows(sqlmock.NewRows([]string{"key", "value", "expires_at", "updated_at"}).
			AddRow("api-key", []byte(`"sk-123"`), nil, now))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "id", Value: "ws-get"},
		{Key: "key", Value: "api-key"},
	}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-get/memory/api-key", nil)

	handler.Get(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp MemoryEntry
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Key != "api-key" {
		t.Errorf("expected key 'api-key', got %q", resp.Key)
	}
}

func TestMemoryGet_NotFound(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoryHandler()

	mock.ExpectQuery("SELECT key, value, expires_at, updated_at").
		WithArgs("ws-nf", "missing-key").
		WillReturnError(sql.ErrNoRows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "id", Value: "ws-nf"},
		{Key: "key", Value: "missing-key"},
	}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-nf/memory/missing-key", nil)

	handler.Get(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMemoryGet_DBError(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoryHandler()

	mock.ExpectQuery("SELECT key, value, expires_at, updated_at").
		WithArgs("ws-err", "key").
		WillReturnError(sql.ErrConnDone)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "id", Value: "ws-err"},
		{Key: "key", Value: "key"},
	}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-err/memory/key", nil)

	handler.Get(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// ==================== POST /workspaces/:id/memory (Set) ====================

func TestMemorySet_Success(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoryHandler()

	mock.ExpectExec("INSERT INTO workspace_memory").
		WillReturnResult(sqlmock.NewResult(0, 1))

	body := `{"key":"counter","value":42}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-set"}}
	c.Request = httptest.NewRequest("POST", "/workspaces/ws-set/memory", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Set(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["key"] != "counter" {
		t.Errorf("expected key 'counter', got %v", resp["key"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestMemorySet_WithTTL(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoryHandler()

	mock.ExpectExec("INSERT INTO workspace_memory").
		WillReturnResult(sqlmock.NewResult(0, 1))

	body := `{"key":"temp","value":"ephemeral","ttl_seconds":3600}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-ttl"}}
	c.Request = httptest.NewRequest("POST", "/workspaces/ws-ttl/memory", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Set(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMemorySet_MissingKey(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoryHandler()

	body := `{"value":"no-key"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-nokey"}}
	c.Request = httptest.NewRequest("POST", "/workspaces/ws-nokey/memory", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Set(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMemorySet_InvalidJSON(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoryHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-bad"}}
	c.Request = httptest.NewRequest("POST", "/workspaces/ws-bad/memory", bytes.NewBufferString("not json"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Set(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMemorySet_DBError(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoryHandler()

	mock.ExpectExec("INSERT INTO workspace_memory").
		WillReturnError(sql.ErrConnDone)

	body := `{"key":"fail","value":"oops"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-set-err"}}
	c.Request = httptest.NewRequest("POST", "/workspaces/ws-set-err/memory", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Set(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

// ==================== DELETE /workspaces/:id/memory/:key ====================

func TestMemoryDelete_Success(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoryHandler()

	mock.ExpectExec("DELETE FROM workspace_memory").
		WithArgs("ws-del", "old-key").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "id", Value: "ws-del"},
		{Key: "key", Value: "old-key"},
	}
	c.Request = httptest.NewRequest("DELETE", "/workspaces/ws-del/memory/old-key", nil)

	handler.Delete(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "deleted" {
		t.Errorf("expected status 'deleted', got %v", resp["status"])
	}
}

func TestMemoryDelete_DBError(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoryHandler()

	mock.ExpectExec("DELETE FROM workspace_memory").
		WithArgs("ws-del-err", "key").
		WillReturnError(sql.ErrConnDone)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "id", Value: "ws-del-err"},
		{Key: "key", Value: "key"},
	}
	c.Request = httptest.NewRequest("DELETE", "/workspaces/ws-del-err/memory/key", nil)

	handler.Delete(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
