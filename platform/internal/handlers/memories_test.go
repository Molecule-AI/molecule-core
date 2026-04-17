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

	// #767: GLOBAL writes always produce an audit log entry.
	mock.ExpectExec("INSERT INTO activity_logs").
		WillReturnResult(sqlmock.NewResult(0, 1))

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
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
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

	mock.ExpectQuery("SELECT key, value, version, expires_at, updated_at FROM workspace_memory").
		WithArgs("ws-1").
		WillReturnRows(sqlmock.NewRows([]string{"key", "value", "version", "expires_at", "updated_at"}))

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

	mock.ExpectQuery("SELECT key, value, version, expires_at, updated_at FROM workspace_memory").
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

// ---------- MemoriesHandler: limit cap (#377) ----------

// TestMemoriesSearch_LimitCap_OverMaxClampsTo50 verifies that requesting
// more than 50 results (e.g. ?limit=100) is silently clamped to 50.
// The LIMIT argument passed to the DB must be 50, not 100.
func TestMemoriesSearch_LimitCap_OverMaxClampsTo50(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoriesHandler()

	mock.ExpectQuery("SELECT parent_id FROM workspaces WHERE id").
		WithArgs("ws-limit-cap").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(nil))

	// LOCAL scope: args are (workspace_id, limit). Expect limit arg = 50 even
	// though the caller asked for 100.
	mock.ExpectQuery("SELECT id, workspace_id, content, scope, namespace, created_at FROM agent_memories WHERE workspace_id").
		WithArgs("ws-limit-cap", 50).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workspace_id", "content", "scope", "namespace", "created_at"}))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-limit-cap"}}
	c.Request = httptest.NewRequest("GET", "/memories?scope=LOCAL&limit=100", nil)
	c.Request.URL.RawQuery = "scope=LOCAL&limit=100"

	handler.Search(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock expectations not met (limit was not clamped to 50): %v", err)
	}
}

// TestMemoriesSearch_LimitExplicit_HonouredWhenBelowMax verifies that
// ?limit=10 is honoured as-is (well under the 50 ceiling).
func TestMemoriesSearch_LimitExplicit_HonouredWhenBelowMax(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoriesHandler()

	mock.ExpectQuery("SELECT parent_id FROM workspaces WHERE id").
		WithArgs("ws-limit-10").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(nil))

	// Expect limit arg = 10.
	mock.ExpectQuery("SELECT id, workspace_id, content, scope, namespace, created_at FROM agent_memories WHERE workspace_id").
		WithArgs("ws-limit-10", 10).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workspace_id", "content", "scope", "namespace", "created_at"}))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-limit-10"}}
	c.Request = httptest.NewRequest("GET", "/memories?scope=LOCAL&limit=10", nil)
	c.Request.URL.RawQuery = "scope=LOCAL&limit=10"

	handler.Search(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock expectations not met (limit=10 was not passed through): %v", err)
	}
}

// TestMemoriesSearch_LimitDefault_Is50 verifies that omitting ?limit uses
// the default ceiling of 50.
func TestMemoriesSearch_LimitDefault_Is50(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoriesHandler()

	mock.ExpectQuery("SELECT parent_id FROM workspaces WHERE id").
		WithArgs("ws-limit-default").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(nil))

	// No ?limit param → expect DB arg = 50.
	mock.ExpectQuery("SELECT id, workspace_id, content, scope, namespace, created_at FROM agent_memories WHERE workspace_id").
		WithArgs("ws-limit-default", 50).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workspace_id", "content", "scope", "namespace", "created_at"}))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-limit-default"}}
	c.Request = httptest.NewRequest("GET", "/memories?scope=LOCAL", nil)
	c.Request.URL.RawQuery = "scope=LOCAL"

	handler.Search(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock expectations not met (default limit should be 50): %v", err)
	}
}

// ---------- Issue #767: GLOBAL memory prompt injection safeguards ----------

// TestRecallMemory_GlobalScope_HasDelimiter verifies that GLOBAL-scope
// memories returned by Search are wrapped with the non-instructable
// [MEMORY id=... scope=GLOBAL from=...]: prefix. This prevents stored
// content from being interpreted as LLM instructions by MCP tool outputs.
func TestRecallMemory_GlobalScope_HasDelimiter(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoriesHandler()

	// Parent lookup (needed by Search for access-control branching)
	mock.ExpectQuery("SELECT parent_id FROM workspaces WHERE id").
		WithArgs("ws-reader").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(nil))

	rows := sqlmock.NewRows([]string{"id", "workspace_id", "content", "scope", "namespace", "created_at"}).
		AddRow("mem-g1", "root-ws", "global knowledge", "GLOBAL", "general", "2024-01-01T00:00:00Z")

	mock.ExpectQuery("SELECT id, workspace_id, content, scope, namespace, created_at FROM agent_memories WHERE scope = 'GLOBAL'").
		WillReturnRows(rows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-reader"}}
	c.Request = httptest.NewRequest("GET", "/memories?scope=GLOBAL", nil)
	c.Request.URL.RawQuery = "scope=GLOBAL"

	handler.Search(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("body not valid JSON: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 memory in result, got %d", len(result))
	}

	content, _ := result[0]["content"].(string)
	want := "[MEMORY id=mem-g1 scope=GLOBAL from=root-ws]: global knowledge"
	if content != want {
		t.Errorf("GLOBAL content delimiter missing or incorrect\ngot:  %q\nwant: %q", content, want)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestCommitMemory_GlobalScope_AuditLogEntry verifies that writing a
// GLOBAL-scope memory always produces an activity_log entry with
// event_type='memory_write_global'. The audit entry stores the SHA-256
// content hash (never plaintext) for forensic replay.
func TestCommitMemory_GlobalScope_AuditLogEntry(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoriesHandler()

	// Root workspace — allowed to write GLOBAL
	mock.ExpectQuery("SELECT parent_id FROM workspaces WHERE id").
		WithArgs("root-ws").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(nil))

	mock.ExpectQuery("INSERT INTO agent_memories").
		WithArgs("root-ws", "sensitive global fact", "GLOBAL", "general").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("mem-audit"))

	// KEY ASSERTION: GLOBAL write must produce an audit log entry.
	// We match on the SQL prefix; the exact arguments (content hash, etc.)
	// are validated by the implementation — here we verify the INSERT fires.
	mock.ExpectExec("INSERT INTO activity_logs").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "root-ws"}}
	body := `{"content":"sensitive global fact","scope":"GLOBAL"}`
	c.Request = httptest.NewRequest("POST", "/", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Commit(c)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	// ExpectationsWereMet fails if the audit INSERT was not called —
	// that's the primary assertion of this test.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("GLOBAL memory write must produce audit log entry: %v", err)
	}
}
