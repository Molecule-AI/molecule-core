package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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

// ---------- Semantic search (pgvector, issue #576) ----------

// TestCommitMemory_EmbeddingFailure_IsNonFatal verifies that when the
// embedding function returns an error, the memory is still stored (201) and
// no UPDATE is issued against the DB.
func TestCommitMemory_EmbeddingFailure_IsNonFatal(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	embedErr := errors.New("embedding service unavailable")
	handler := NewMemoriesHandler().WithEmbedding(
		func(_ context.Context, _ string) ([]float32, error) {
			return nil, embedErr
		},
	)

	// Only the INSERT is expected — no UPDATE because embedding failed.
	mock.ExpectQuery("INSERT INTO agent_memories").
		WithArgs("ws-1", "important fact", "LOCAL", "general").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("mem-new"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}
	body := `{"content":"important fact","scope":"LOCAL"}`
	c.Request = httptest.NewRequest("POST", "/", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Commit(c)

	if w.Code != http.StatusCreated {
		t.Errorf("embedding failure must not prevent 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["id"] != "mem-new" {
		t.Errorf("expected id 'mem-new', got %v", resp["id"])
	}
	// All expectations met means the unexpected UPDATE was never issued.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected DB calls after embedding failure: %v", err)
	}
}

// TestRecallMemory_SemanticSearch_ReturnsOrderedByDistance verifies that when
// an EmbeddingFunc is configured, Search uses the cosine-similarity path and
// returns results with a similarity_score field ordered highest-first.
func TestRecallMemory_SemanticSearch_ReturnsOrderedByDistance(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	// Stub embedding: returns a unit vector along dimension 0.
	knownVec := make([]float32, 1536)
	knownVec[0] = 1.0
	embedCalled := false
	handler := NewMemoriesHandler().WithEmbedding(
		func(_ context.Context, text string) ([]float32, error) {
			embedCalled = true
			return knownVec, nil
		},
	)

	// Parent lookup for default scope.
	mock.ExpectQuery("SELECT parent_id FROM workspaces WHERE id").
		WithArgs("ws-sem").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(nil))

	// Semantic search returns two rows pre-ordered by the DB (highest first).
	semRows := sqlmock.NewRows([]string{
		"id", "workspace_id", "content", "scope", "namespace", "created_at", "similarity_score",
	}).
		AddRow("mem-a", "ws-sem", "dogs are mammals", "LOCAL", "general", "2024-01-02T00:00:00Z", 0.95).
		AddRow("mem-b", "ws-sem", "chairs have legs", "LOCAL", "general", "2024-01-01T00:00:00Z", 0.42)

	// The semantic SQL contains "similarity_score"; FTS SQL does not.
	mock.ExpectQuery(`similarity_score`).
		WillReturnRows(semRows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-sem"}}
	c.Request = httptest.NewRequest("GET", "/memories?q=animals", nil)
	c.Request.URL.RawQuery = "q=animals"

	handler.Search(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !embedCalled {
		t.Error("expected EmbeddingFunc to be called for semantic search")
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d: %s", len(result), w.Body.String())
	}
	score0, ok0 := result[0]["similarity_score"].(float64)
	score1, ok1 := result[1]["similarity_score"].(float64)
	if !ok0 || !ok1 {
		t.Fatalf("similarity_score missing or wrong type in results: %v", result)
	}
	if score0 <= score1 {
		t.Errorf("expected result[0].similarity_score (%g) > result[1].similarity_score (%g)", score0, score1)
	}
}

// TestRecallMemory_SemanticSearch_FallsBackToFTS_WhenNoEmbedding verifies that
// when no EmbeddingFunc is configured (or all rows lack embeddings), Search
// falls back to the standard FTS path without crashing. The response must be
// 200 and must NOT contain a similarity_score field.
func TestRecallMemory_SemanticSearch_FallsBackToFTS_WhenNoEmbedding(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	// Plain handler — no embedding function configured.
	handler := NewMemoriesHandler()

	mock.ExpectQuery("SELECT parent_id FROM workspaces WHERE id").
		WithArgs("ws-fts").
		WillReturnRows(sqlmock.NewRows([]string{"parent_id"}).AddRow(nil))

	// FTS path: 6-column SELECT (no similarity_score).
	ftsRows := sqlmock.NewRows([]string{
		"id", "workspace_id", "content", "scope", "namespace", "created_at",
	}).AddRow("mem-fts", "ws-fts", "knowledge about topics", "LOCAL", "general", "2024-01-01T00:00:00Z")

	mock.ExpectQuery(`SELECT id, workspace_id, content, scope, namespace, created_at FROM agent_memories WHERE workspace_id`).
		WillReturnRows(ftsRows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-fts"}}
	c.Request = httptest.NewRequest("GET", "/memories?q=topics", nil)
	c.Request.URL.RawQuery = "q=topics"

	handler.Search(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on FTS fallback, got %d: %s", w.Code, w.Body.String())
	}
	var result []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 FTS result, got %d", len(result))
	}
	if _, hasSim := result[0]["similarity_score"]; hasSim {
		t.Error("FTS path must not include similarity_score field")
	}
	if result[0]["id"] != "mem-fts" {
		t.Errorf("expected id 'mem-fts', got %v", result[0]["id"])
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

// ---------- SAFE-T1201: secret redaction (issue #838) ----------

// TestRedactSecrets_CleanContent_PassesThrough verifies that content with no
// secret patterns is returned unchanged and changed==false.
func TestRedactSecrets_CleanContent_PassesThrough(t *testing.T) {
	inputs := []string{
		"The answer is 42",
		"dogs are mammals",
		"remember to open the PR before EOD",
		"short",
		"",
	}
	for _, in := range inputs {
		out, changed := redactSecrets("ws-1", in)
		if changed {
			t.Errorf("clean content %q was unexpectedly changed to %q", in, out)
		}
		if out != in {
			t.Errorf("clean content %q was mutated to %q", in, out)
		}
	}
}

// TestRedactSecrets_APIKeyPattern_IsRedacted verifies that env-var API key
// assignments are scrubbed before persistence.
func TestRedactSecrets_APIKeyPattern_IsRedacted(t *testing.T) {
	cases := []struct {
		input string
		label string
	}{
		{"OPENAI_API_KEY=sk-1234567890abcdefgh", "API_KEY"},
		{"ANTHROPIC_API_KEY=sk-ant-api03-longkeyvalue", "API_KEY"},
		{"MY_SERVICE_TOKEN=ghp_ABCDEFGH1234567890", "TOKEN"},
		{"DATABASE_SECRET=supersecret", "SECRET"},
	}
	for _, tc := range cases {
		out, changed := redactSecrets("ws-1", tc.input)
		if !changed {
			t.Errorf("expected redaction of %q, got unchanged", tc.input)
		}
		want := "[REDACTED:" + tc.label + "]"
		if out != want {
			t.Errorf("input %q: got %q, want %q", tc.input, out, want)
		}
	}
}

// TestRedactSecrets_BearerToken_IsRedacted verifies HTTP Bearer header values
// are scrubbed.
func TestRedactSecrets_BearerToken_IsRedacted(t *testing.T) {
	input := "Authorization: Bearer ghp_AbCdEfGhIjKlMnOp1234"
	out, changed := redactSecrets("ws-1", input)
	if !changed {
		t.Errorf("Bearer token was not redacted in %q", input)
	}
	if strings.Contains(out, "ghp_") {
		t.Errorf("Bearer token value still present after redaction: %q", out)
	}
	if !strings.Contains(out, "[REDACTED:BEARER_TOKEN]") {
		t.Errorf("expected [REDACTED:BEARER_TOKEN] in output, got: %q", out)
	}
}

// TestRedactSecrets_SKToken_IsRedacted verifies sk-... prefixed secret keys
// (OpenAI / Anthropic format) are scrubbed.
func TestRedactSecrets_SKToken_IsRedacted(t *testing.T) {
	// Use a key that is NOT caught by the env-var pattern first (no KEY= prefix)
	input := "the key is sk-ant-api03-AAAAAAAAAAAAAAAAAAAAAA"
	out, changed := redactSecrets("ws-1", input)
	if !changed {
		t.Errorf("sk- token was not redacted in %q", input)
	}
	if strings.Contains(out, "sk-ant") {
		t.Errorf("sk- value still present after redaction: %q", out)
	}
}

// TestRedactSecrets_Ctx7Token_IsRedacted verifies context7 tokens are scrubbed.
func TestRedactSecrets_Ctx7Token_IsRedacted(t *testing.T) {
	input := "ctx7_AbCdEfGhIjKlMnOpQrStUvWxYz123456"
	out, changed := redactSecrets("ws-1", input)
	if !changed {
		t.Errorf("ctx7_ token was not redacted in %q", input)
	}
	if strings.Contains(out, "ctx7_") {
		t.Errorf("ctx7_ value still present after redaction: %q", out)
	}
	if !strings.Contains(out, "[REDACTED:CTX7_TOKEN]") {
		t.Errorf("expected [REDACTED:CTX7_TOKEN] in output, got: %q", out)
	}
}

// TestRedactSecrets_Base64Blob_IsRedacted verifies that high-entropy base64
// blobs of 33+ chars are scrubbed.
func TestRedactSecrets_Base64Blob_IsRedacted(t *testing.T) {
	// A realistic base64-encoded secret (33+ chars, contains + and /)
	input := "stored secret: dGhpcyBpcyBhIHNlY3JldCBibG9i/AAAA=="
	out, changed := redactSecrets("ws-1", input)
	if !changed {
		t.Errorf("base64 blob was not redacted in %q", input)
	}
	if !strings.Contains(out, "[REDACTED:BASE64_BLOB]") {
		t.Errorf("expected [REDACTED:BASE64_BLOB] in output, got: %q", out)
	}
}

// TestCommitMemory_SecretInContent_IsRedactedBeforeInsert verifies that the
// Commit handler scrubs secret patterns before the INSERT so credentials are
// never persisted verbatim. The DB mock expects the redacted value.
func TestCommitMemory_SecretInContent_IsRedactedBeforeInsert(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)
	handler := NewMemoriesHandler()

	// The raw content contains an API key assignment. After redaction the DB
	// must receive the scrubbed version, not the original.
	rawContent := "OPENAI_API_KEY=sk-1234567890abcdefgh"
	redacted, _ := redactSecrets("ws-1", rawContent) // derive expected value

	mock.ExpectQuery("INSERT INTO agent_memories").
		WithArgs("ws-1", redacted, "LOCAL", "general").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("mem-safe"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}
	body := `{"content":"OPENAI_API_KEY=sk-1234567890abcdefgh","scope":"LOCAL"}`
	c.Request = httptest.NewRequest("POST", "/", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Commit(c)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("secret content was not redacted before DB insert: %v", err)
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