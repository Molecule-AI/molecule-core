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

// newAdminMemoriesHandler is a test helper that returns an AdminMemoriesHandler.
func newAdminMemoriesHandler() *AdminMemoriesHandler {
	return NewAdminMemoriesHandler()
}

// adminPost builds a POST /admin/memories/import request.
func adminPost(t *testing.T, h *AdminMemoriesHandler, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/admin/memories/import", bytes.NewReader(b))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Import(c)
	return w
}

// adminGet builds a GET /admin/memories/export request.
func adminGet(t *testing.T, h *AdminMemoriesHandler) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin/memories/export", nil)
	h.Export(c)
	return w
}

// ─────────────────────────────────────────────────────────────────────────────
// Export tests
// ─────────────────────────────────────────────────────────────────────────────

func TestAdminMemories_Export_Success(t *testing.T) {
	mock := setupTestDB(t)
	h := newAdminMemoriesHandler()

	now := time.Now().UTC().Truncate(time.Second)
	rows := sqlmock.NewRows([]string{"id", "content", "scope", "namespace", "created_at", "workspace_name"}).
		AddRow("mem-1", "hello world", "LOCAL", "ws-1", now, "my-workspace").
		AddRow("mem-2", "another fact", "TEAM", "ws-1", now, "my-workspace")

	mock.ExpectQuery("SELECT am.id, am.content, am.scope, am.namespace, am.created_at,").
		WillReturnRows(rows)

	w := adminGet(t, h)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var memories []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &memories); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(memories) != 2 {
		t.Errorf("expected 2 memories, got %d", len(memories))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestAdminMemories_Export_Empty(t *testing.T) {
	mock := setupTestDB(t)
	h := newAdminMemoriesHandler()

	rows := sqlmock.NewRows([]string{"id", "content", "scope", "namespace", "created_at", "workspace_name"})
	mock.ExpectQuery("SELECT am.id, am.content, am.scope, am.namespace, am.created_at,").
		WillReturnRows(rows)

	w := adminGet(t, h)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var memories []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &memories); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(memories) != 0 {
		t.Errorf("expected 0 memories, got %d", len(memories))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestAdminMemories_Export_QueryError(t *testing.T) {
	mock := setupTestDB(t)
	h := newAdminMemoriesHandler()

	mock.ExpectQuery("SELECT am.id, am.content, am.scope, am.namespace, am.created_at,").
		WillReturnError(sql.ErrConnDone)

	w := adminGet(t, h)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestAdminMemories_Export_RedactsSecrets(t *testing.T) {
	mock := setupTestDB(t)
	h := newAdminMemoriesHandler()

	// Content with a secret pattern. Export must call redactSecrets and return
	// the redacted form, not the raw credential.
	secretContent := "Remember to use OPENAI_API_KEY=sk-1234567890abcdefgh for the model"
	redacted, _ := redactSecrets("my-workspace", secretContent)

	now := time.Now().UTC().Truncate(time.Second)
	rows := sqlmock.NewRows([]string{"id", "content", "scope", "namespace", "created_at", "workspace_name"}).
		AddRow("mem-secret", secretContent, "LOCAL", "my-workspace", now, "my-workspace")

	mock.ExpectQuery("SELECT am.id, am.content, am.scope, am.namespace, am.created_at,").
		WillReturnRows(rows)

	w := adminGet(t, h)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var memories []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &memories); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(memories) != 1 {
		t.Fatalf("expected 1 memory, got %d", len(memories))
	}
	// The exported content must be the REDACTED version, not the raw secret.
	if content, ok := memories[0]["content"].(string); ok {
		if content == secretContent {
			t.Errorf("Export returned raw secret %q — F1084 regression: redactSecrets not called", secretContent)
		}
		if content != redacted {
			t.Errorf("Export content = %q, want redacted %q", content, redacted)
		}
		// Confirm the redacted version doesn't contain the raw key fragment.
		if len(content) > 10 && content == "OPENAI_API_KEY=[REDACTED:" {
			t.Errorf("redaction appears incomplete: %q", content)
		}
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Import tests
// ─────────────────────────────────────────────────────────────────────────────

func TestAdminMemories_Import_Success(t *testing.T) {
	mock := setupTestDB(t)
	h := newAdminMemoriesHandler()

	// Workspace lookup returns one row.
	mock.ExpectQuery("SELECT id FROM workspaces WHERE name = \\$1").
		WithArgs("my-workspace").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("ws-uuid-1"))

	// Duplicate check returns false.
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("ws-uuid-1", sqlmock.AnyArg(), "LOCAL").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	// Insert succeeds. Handler uses 4-arg INSERT when created_at is absent.
	mock.ExpectExec("INSERT INTO agent_memories").
		WithArgs("ws-uuid-1", sqlmock.AnyArg(), "LOCAL", "general").
		WillReturnResult(sqlmock.NewResult(1, 1))

	w := adminPost(t, h, []map[string]interface{}{
		{
			"content":        "important fact",
			"scope":         "LOCAL",
			"workspace_name": "my-workspace",
		},
	})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["imported"].(float64) != 1 {
		t.Errorf("imported = %v, want 1", resp["imported"])
	}
	if resp["skipped"].(float64) != 0 {
		t.Errorf("skipped = %v, want 0", resp["skipped"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestAdminMemories_Import_InvalidJSON(t *testing.T) {
	_ = setupTestDB(t)
	h := newAdminMemoriesHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/admin/memories/import", bytes.NewReader([]byte("not json")))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Import(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminMemories_Import_WorkspaceNotFound_SkipsEntry(t *testing.T) {
	mock := setupTestDB(t)
	h := newAdminMemoriesHandler()

	// Workspace lookup returns no rows.
	mock.ExpectQuery("SELECT id FROM workspaces WHERE name = \\$1").
		WithArgs("ghost-workspace").
		WillReturnError(sql.ErrNoRows)

	w := adminPost(t, h, []map[string]interface{}{
		{
			"content":        "some fact",
			"scope":         "LOCAL",
			"workspace_name": "ghost-workspace",
		},
	})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["skipped"].(float64) != 1 {
		t.Errorf("skipped = %v, want 1 (workspace not found)", resp["skipped"])
	}
	if resp["imported"].(float64) != 0 {
		t.Errorf("imported = %v, want 0", resp["imported"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestAdminMemories_Import_DuplicateSkipped(t *testing.T) {
	mock := setupTestDB(t)
	h := newAdminMemoriesHandler()

	// Workspace lookup succeeds.
	mock.ExpectQuery("SELECT id FROM workspaces WHERE name = \\$1").
		WithArgs("my-workspace").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("ws-uuid-1"))

	// Duplicate check returns true → entry is skipped.
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("ws-uuid-1", sqlmock.AnyArg(), "LOCAL").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	w := adminPost(t, h, []map[string]interface{}{
		{
			"content":        "already stored fact",
			"scope":         "LOCAL",
			"workspace_name": "my-workspace",
		},
	})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["skipped"].(float64) != 1 {
		t.Errorf("skipped = %v, want 1 (duplicate)", resp["skipped"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestAdminMemories_Import_RedactsSecretsBeforeDedup verifies F1085 (#1132):
// redactSecrets is called BEFORE the deduplication check so that two backups
// with the same original secret each get the same placeholder and dedup works.
// The DB dedup query must receive the REDACTED content, not the raw credential.
func TestAdminMemories_Import_RedactsSecretsBeforeDedup(t *testing.T) {
	mock := setupTestDB(t)
	h := newAdminMemoriesHandler()

	rawContent := "the key is OPENAI_API_KEY=sk-1234567890abcdefgh"
	redacted, changed := redactSecrets("my-workspace", rawContent)
	if !changed {
		t.Fatalf("precondition: redactSecrets must change the test content")
	}

	// Workspace lookup.
	mock.ExpectQuery("SELECT id FROM workspaces WHERE name = \\$1").
		WithArgs("my-workspace").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("ws-uuid-1"))

	// Dedup check — the sqlmock must be set up for the REDACTED content,
	// because Import calls redactSecrets before running the dedup query.
	// If redactSecrets is not called, the mock would match on rawContent instead.
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("ws-uuid-1", redacted, "LOCAL").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	// Insert — receives the redacted content (not raw). Handler uses the
	// 4-arg INSERT when created_at is absent from the payload.
	mock.ExpectExec("INSERT INTO agent_memories").
		WithArgs("ws-uuid-1", redacted, "LOCAL", "general").
		WillReturnResult(sqlmock.NewResult(1, 1))

	w := adminPost(t, h, []map[string]interface{}{
		{
			"content":        rawContent,
			"scope":         "LOCAL",
			"workspace_name": "my-workspace",
		},
	})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["imported"].(float64) != 1 {
		t.Errorf("imported = %v, want 1", resp["imported"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v (F1085 regression: redactSecrets not called before dedup)", err)
	}
}

func TestAdminMemories_Import_PreservesCreatedAt(t *testing.T) {
	mock := setupTestDB(t)
	h := newAdminMemoriesHandler()

	origTime := "2026-01-15T10:30:00Z"

	// Workspace lookup.
	mock.ExpectQuery("SELECT id FROM workspaces WHERE name = \\$1").
		WithArgs("my-workspace").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("ws-uuid-1"))

	// Dedup check.
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("ws-uuid-1", sqlmock.AnyArg(), "LOCAL").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	// Insert with created_at — must use the 5-arg INSERT.
	mock.ExpectExec("INSERT INTO agent_memories").
		WithArgs("ws-uuid-1", sqlmock.AnyArg(), "LOCAL", "general", origTime).
		WillReturnResult(sqlmock.NewResult(1, 1))

	w := adminPost(t, h, []map[string]interface{}{
		{
			"content":        "a fact",
			"scope":         "LOCAL",
			"workspace_name": "my-workspace",
			"created_at":    origTime,
		},
	})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["imported"].(float64) != 1 {
		t.Errorf("imported = %v, want 1", resp["imported"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}
