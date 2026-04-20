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
)

// ---------- AdminMemoriesHandler: Export ----------

// TestAdminMemoriesExport_RedactsSecrets verifies F1084/#1131: secrets stored
// in agent_memories (e.g. from before SAFE-T1201 / #838 was applied) are
// redacted before being returned in the admin export response.
func TestAdminMemoriesExport_RedactsSecrets(t *testing.T) {
	mock := setupTestDB(t)
	handler := NewAdminMemoriesHandler()

	createdAt, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")

	// The DB contains raw secret-bearing content (pre-redactSecrets write).
	mock.ExpectQuery("SELECT am.id, am.content, am.scope, am.namespace, am.created_at,").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "content", "scope", "namespace", "created_at", "workspace_name",
		}).
			AddRow("mem-1", "API key is sk-ant-...abc123", "LOCAL", "general", createdAt, "agent-1").
			AddRow("mem-2", "Bearer ghp_xxxxxxxxxxxx", "TEAM", "general", createdAt, "agent-2").
			AddRow("mem-3", "OPENAI_API_KEY=sk-...xyz789", "LOCAL", "general", createdAt, "agent-3").
			AddRow("mem-4", " innocent prose only ", "LOCAL", "general", createdAt, "agent-4"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin/memories/export", nil)

	handler.Export(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var results []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &results); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(results) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(results))
	}

	// mem-1: OpenAI sk-ant-... key must be redacted.
	if results[0]["content"] != "[REDACTED:SK_TOKEN]" {
		t.Errorf("mem-1: expected redacted SK_TOKEN, got %q", results[0]["content"])
	}

	// mem-2: GitHub Bearer token must be redacted.
	if results[1]["content"] != "[REDACTED:BEARER_TOKEN]" {
		t.Errorf("mem-2: expected redacted BEARER_TOKEN, got %q", results[1]["content"])
	}

	// mem-3: env-var assignment API key must be redacted.
	if results[2]["content"] != "[REDACTED:API_KEY]" {
		t.Errorf("mem-3: expected redacted API_KEY, got %q", results[2]["content"])
	}

	// mem-4: plain text must be returned unchanged.
	if results[3]["content"] != " innocent prose only " {
		t.Errorf("mem-4: expected unchanged prose, got %q", results[3]["content"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestAdminMemoriesExport_EmptyDb returns empty array, not error.
func TestAdminMemoriesExport_EmptyDb(t *testing.T) {
	mock := setupTestDB(t)
	handler := NewAdminMemoriesHandler()

	mock.ExpectQuery("SELECT am.id, am.content, am.scope, am.namespace, am.created_at,").
		WillReturnError(sql.ErrNoRows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin/memories/export", nil)

	handler.Export(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var results []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &results)
	if len(results) != 0 {
		t.Errorf("expected 0 entries, got %d", len(results))
	}
}

// ---------- AdminMemoriesHandler: Import ----------

// TestAdminMemoriesImport_RedactsBeforeInsert verifies F1085/#1132: imported
// memories have secrets scrubbed by redactSecrets before both the dedup check
// and the actual INSERT so that secrets never land unredacted in agent_memories.
func TestAdminMemoriesImport_RedactsBeforeInsert(t *testing.T) {
	mock := setupTestDB(t)
	handler := NewAdminMemoriesHandler()

	payload := `[{
		"content": "OPENAI_API_KEY=sk-test1234567890abcdef",
		"scope": "LOCAL",
		"namespace": "general",
		"workspace_name": "agent-1"
	}]`

	// Step 1: workspace lookup must succeed.
	mock.ExpectQuery("SELECT id FROM workspaces WHERE name =").
		WithArgs("agent-1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("ws-1"))

	// Step 2: dedup check uses REDACTED content (not the raw secret).
	// The raw content "OPENAI_API_KEY=sk-test..." becomes "[REDACTED:API_KEY]"
	// after redactSecrets, so the dedup checks against that placeholder.
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("ws-1", "[REDACTED:API_KEY]", "LOCAL").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	// Step 3: INSERT uses the redacted content, not the raw secret.
	mock.ExpectExec("INSERT INTO agent_memories").
		WithArgs("ws-1", "[REDACTED:API_KEY]", "LOCAL", "general", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/admin/memories/import",
		bytes.NewBufferString(payload))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Import(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["imported"] != float64(1) {
		t.Errorf("expected imported=1, got %v", resp["imported"])
	}
	if resp["skipped"] != float64(0) {
		t.Errorf("expected skipped=0, got %v", resp["skipped"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestAdminMemoriesImport_WorkspaceNotFound skips gracefully.
func TestAdminMemoriesImport_WorkspaceNotFound(t *testing.T) {
	mock := setupTestDB(t)
	handler := NewAdminMemoriesHandler()

	payload := `[{"content": "some content", "scope": "LOCAL", "workspace_name": "ghost-ws"}]`

	mock.ExpectQuery("SELECT id FROM workspaces WHERE name =").
		WithArgs("ghost-ws").
		WillReturnError(sql.ErrNoRows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/admin/memories/import",
		bytes.NewBufferString(payload))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Import(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["skipped"] != float64(1) {
		t.Errorf("expected skipped=1, got %v", resp["skipped"])
	}
}

// TestAdminMemoriesImport_InvalidJson returns 400.
func TestAdminMemoriesImport_InvalidJson(t *testing.T) {
	setupTestDB(t) // still needed for package-level init
	handler := NewAdminMemoriesHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/admin/memories/import",
		bytes.NewBufferString("not valid json"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Import(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// TestAdminMemoriesImport_CreatedAtPreserved uses 5-arg INSERT.
func TestAdminMemoriesImport_CreatedAtPreserved(t *testing.T) {
	mock := setupTestDB(t)
	handler := NewAdminMemoriesHandler()

	payload := `[{
		"content": "secret token GITHUB_TOKEN=ghp_deadbeef",
		"scope": "TEAM",
		"namespace": "research",
		"created_at": "2026-01-15T10:30:00Z",
		"workspace_name": "agent-2"
	}]`

	mock.ExpectQuery("SELECT id FROM workspaces WHERE name =").
		WithArgs("agent-2").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("ws-2"))

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("ws-2", "[REDACTED:TOKEN]", "TEAM").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	// 5-arg INSERT (with created_at)
	mock.ExpectExec("INSERT INTO agent_memories").
		WithArgs("ws-2", "[REDACTED:TOKEN]", "TEAM", "research", "2026-01-15T10:30:00Z").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/admin/memories/import",
		bytes.NewBufferString(payload))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Import(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestAdminMemoriesImport_DefaultNamespace uses "general" when namespace is empty.
func TestAdminMemoriesImport_DefaultNamespace(t *testing.T) {
	mock := setupTestDB(t)
	handler := NewAdminMemoriesHandler()

	payload := `[{
		"content": "ANTHROPIC_API_KEY=sk-ant-test999",
		"scope": "LOCAL",
		"workspace_name": "agent-3"
	}]`

	mock.ExpectQuery("SELECT id FROM workspaces WHERE name =").
		WithArgs("agent-3").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("ws-3"))

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("ws-3", "[REDACTED:API_KEY]", "LOCAL").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	// Namespace defaults to "general"
	mock.ExpectExec("INSERT INTO agent_memories").
		WithArgs("ws-3", "[REDACTED:API_KEY]", "LOCAL", "general", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/admin/memories/import",
		bytes.NewBufferString(payload))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Import(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
