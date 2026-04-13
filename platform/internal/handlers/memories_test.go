package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

// ---------- MemoriesHandler: Commit ----------

func TestMemoriesCommit_Local_Success(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoriesHandler()

	mock.ExpectQuery("INSERT INTO agent_memories").
		WithArgs("ws-1", "The answer is 42", "LOCAL", "general").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("mem-1"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}
	body := `{"content":"The answer is 42","scope":"LOCAL"}`
	c.Request = httptest.NewRequest("POST", "/", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Commit(c)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["id"] != "mem-1" {
		t.Errorf("expected id mem-1, got %v", resp["id"])
	}
	if resp["scope"] != "LOCAL" {
		t.Errorf("expected scope LOCAL, got %v", resp["scope"])
	}
}

func TestMemoriesCommit_Global_AsRoot(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoriesHandler()

	// Root workspace — no parent
	mock.ExpectQuery("SELECT parent_id FROM workspaces WHERE id").
		WithArgs("root-ws").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(nil))

	mock.ExpectQuery("INSERT INTO agent_memories").
		WithArgs("root-ws", "global fact", "GLOBAL", "general").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("mem-global"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "root-ws"}}
	body := `{"content":"global fact","scope":"GLOBAL"}`
	c.Request = httptest.NewRequest("POST", "/", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Commit(c)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMemoriesCommit_Global_ForbiddenForChild(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoriesHandler()

	// Child workspace — has parent
	parentID := "parent-ws"
	mock.ExpectQuery("SELECT parent_id FROM workspaces WHERE id").
		WithArgs("child-ws").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(&parentID))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "child-ws"}}
	body := `{"content":"global fact","scope":"GLOBAL"}`
	c.Request = httptest.NewRequest("POST", "/", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Commit(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMemoriesCommit_InvalidScope(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoriesHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}
	body := `{"content":"fact","scope":"PRIVATE"}`
	c.Request = httptest.NewRequest("POST", "/", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Commit(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestMemoriesCommit_MissingFields(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoriesHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}
	c.Request = httptest.NewRequest("POST", "/", bytes.NewBufferString(`{"content":"fact"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Commit(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ---------- MemoriesHandler: Search ----------

func TestMemoriesSearch_LocalScope(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoriesHandler()

	// Parent lookup
	mock.ExpectQuery("SELECT parent_id FROM workspaces WHERE id").
		WithArgs("ws-1").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(nil))

	rows := sqlmock.NewRows([]string{"id", "workspace_id", "content", "scope", "namespace", "created_at"}).
		AddRow("mem-1", "ws-1", "local memory", "LOCAL", "general", "2024-01-01T00:00:00Z")

	mock.ExpectQuery("SELECT id, workspace_id, content, scope, namespace, created_at FROM agent_memories WHERE workspace_id").
		WillReturnRows(rows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}
	c.Request = httptest.NewRequest("GET", "/memories?scope=LOCAL", nil)
	c.Request.URL.RawQuery = "scope=LOCAL"

	handler.Search(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result []interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if len(result) != 1 {
		t.Errorf("expected 1 memory, got %d", len(result))
	}
}

func TestMemoriesSearch_GlobalScope(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoriesHandler()

	// Parent lookup
	mock.ExpectQuery("SELECT parent_id FROM workspaces WHERE id").
		WithArgs("ws-1").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(nil))

	rows := sqlmock.NewRows([]string{"id", "workspace_id", "content", "scope", "namespace", "created_at"}).
		AddRow("mem-g1", "root-ws", "global knowledge", "GLOBAL", "general", "2024-01-01T00:00:00Z")

	mock.ExpectQuery("SELECT id, workspace_id, content, scope, namespace, created_at FROM agent_memories WHERE scope = 'GLOBAL'").
		WillReturnRows(rows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}
	c.Request = httptest.NewRequest("GET", "/memories?scope=GLOBAL", nil)
	c.Request.URL.RawQuery = "scope=GLOBAL"

	handler.Search(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMemoriesSearch_DefaultScope_WithQuery(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoriesHandler()

	mock.ExpectQuery("SELECT parent_id FROM workspaces WHERE id").
		WithArgs("ws-1").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(nil))

	rows := sqlmock.NewRows([]string{"id", "workspace_id", "content", "scope", "namespace", "created_at"})

	mock.ExpectQuery("SELECT id, workspace_id, content, scope, namespace, created_at FROM agent_memories WHERE workspace_id").
		WillReturnRows(rows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}
	c.Request = httptest.NewRequest("GET", "/memories?q=answer", nil)
	c.Request.URL.RawQuery = "q=answer"

	handler.Search(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestMemoriesSearch_TeamScope_AsChild(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoriesHandler()

	parentID := "parent-ws"
	mock.ExpectQuery("SELECT parent_id FROM workspaces WHERE id").
		WithArgs("child-ws").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(&parentID))

	rows := sqlmock.NewRows([]string{"id", "workspace_id", "content", "scope", "namespace", "created_at"}).
		AddRow("mem-t1", "sibling-ws", "team info", "TEAM", "general", "2024-01-01T00:00:00Z")

	mock.ExpectQuery("SELECT m.id, m.workspace_id, m.content, m.scope, m.namespace, m.created_at").
		WillReturnRows(rows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "child-ws"}}
	c.Request = httptest.NewRequest("GET", "/memories?scope=TEAM", nil)
	c.Request.URL.RawQuery = "scope=TEAM"

	handler.Search(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------- MemoriesHandler: Delete ----------

func TestMemoriesDelete_Success(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoriesHandler()

	mock.ExpectExec("DELETE FROM agent_memories WHERE id").
		WithArgs("mem-del", "ws-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}, {Key: "memoryId", Value: "mem-del"}}
	c.Request = httptest.NewRequest("DELETE", "/", nil)

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

func TestMemoriesDelete_NotFound(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoriesHandler()

	mock.ExpectExec("DELETE FROM agent_memories WHERE id").
		WithArgs("mem-none", "ws-1").
		WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows affected

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}, {Key: "memoryId", Value: "mem-none"}}
	c.Request = httptest.NewRequest("DELETE", "/", nil)

	handler.Delete(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// ---------- nextArg helper ----------

func TestNextArg(t *testing.T) {
	if nextArg(0) != "$1" {
		t.Errorf("expected $1")
	}
	if nextArg(2) != "$3" {
		t.Errorf("expected $3")
	}
}

// ---------- MemoryHandler (workspace key-value store) ----------

func TestMemoryHandler_List_Empty(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoryHandler()

	mock.ExpectQuery("SELECT key, value, expires_at, updated_at FROM workspace_memory").
		WithArgs("ws-1").
		WillReturnRows(sqlmock.NewRows([]string{"key", "value", "expires_at", "updated_at"}))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-1/memory", nil)

	handler.List(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestMemoryHandler_Get_NotFound(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoryHandler()

	mock.ExpectQuery("SELECT key, value, expires_at, updated_at FROM workspace_memory").
		WithArgs("ws-1", "missing-key").
		WillReturnError(sql.ErrNoRows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}, {Key: "key", Value: "missing-key"}}
	c.Request = httptest.NewRequest("GET", "/", nil)

	handler.Get(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// ---------- MemoriesHandler: namespace + FTS (migration 017) ----------

func TestMemoriesCommit_WithNamespace(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoriesHandler()

	mock.ExpectQuery("INSERT INTO agent_memories").
		WithArgs("ws-1", "API route table", "LOCAL", "reference").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("mem-ns-1"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}
	body := `{"content":"API route table","scope":"LOCAL","namespace":"reference"}`
	c.Request = httptest.NewRequest("POST", "/", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Commit(c)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["namespace"] != "reference" {
		t.Errorf("expected namespace reference, got %v", resp["namespace"])
	}
}

func TestMemoriesCommit_NamespaceTooLong(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoriesHandler()

	long := strings.Repeat("a", 51)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}
	body := `{"content":"x","scope":"LOCAL","namespace":"` + long + `"}`
	c.Request = httptest.NewRequest("POST", "/", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Commit(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for over-long namespace, got %d", w.Code)
	}
}

func TestMemoriesSearch_FTSForMultiCharQuery(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoriesHandler()

	mock.ExpectQuery("SELECT parent_id FROM workspaces WHERE id").
		WithArgs("ws-1").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(nil))

	// The FTS path uses content_tsv @@ plainto_tsquery and ts_rank ordering.
	// sqlmock matches the regex substring against the actual SQL.
	rows := sqlmock.NewRows([]string{"id", "workspace_id", "content", "scope", "namespace", "created_at"}).
		AddRow("mem-fts-1", "ws-1", "canvas zinc theme convention", "LOCAL", "general", "2024-01-01T00:00:00Z")
	mock.ExpectQuery("content_tsv @@ plainto_tsquery").
		WillReturnRows(rows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}
	c.Request = httptest.NewRequest("GET", "/memories?q=zinc+theme", nil)
	c.Request.URL.RawQuery = "q=zinc+theme"

	handler.Search(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if len(result) != 1 || result[0]["namespace"] != "general" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestMemoriesSearch_ILIKEFallbackForSingleChar(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoriesHandler()

	mock.ExpectQuery("SELECT parent_id FROM workspaces WHERE id").
		WithArgs("ws-1").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(nil))

	// Single-char query bypasses FTS (tsvector tokenises single chars to
	// nothing in 'english' config) and falls back to ILIKE.
	rows := sqlmock.NewRows([]string{"id", "workspace_id", "content", "scope", "namespace", "created_at"})
	mock.ExpectQuery("content ILIKE").
		WillReturnRows(rows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}
	c.Request = httptest.NewRequest("GET", "/memories?q=a", nil)
	c.Request.URL.RawQuery = "q=a"

	handler.Search(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMemoriesSearch_NamespaceFilter(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoriesHandler()

	mock.ExpectQuery("SELECT parent_id FROM workspaces WHERE id").
		WithArgs("ws-1").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(nil))

	// Namespace filter composes with the default scope query.
	rows := sqlmock.NewRows([]string{"id", "workspace_id", "content", "scope", "namespace", "created_at"}).
		AddRow("mem-proc-1", "ws-1", "how to restart agents", "LOCAL", "procedures", "2024-01-01T00:00:00Z")
	mock.ExpectQuery("AND namespace =").
		WillReturnRows(rows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}
	c.Request = httptest.NewRequest("GET", "/memories?namespace=procedures", nil)
	c.Request.URL.RawQuery = "namespace=procedures"

	handler.Search(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	if len(result) != 1 || result[0]["namespace"] != "procedures" {
		t.Errorf("unexpected result: %v", result)
	}
}
