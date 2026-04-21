package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/gin-gonic/gin"
)

// newAdminMemoriesHandler is a test helper that constructs an AdminMemoriesHandler.
func newAdminMemoriesHandler(t *testing.T, mock sqlmock.Sqlmock) *AdminMemoriesHandler {
	t.Helper()
	_ = mock // surfaced for callers that need to set expectations
	return NewAdminMemoriesHandler()
}

// ---------- Export ----------

// TestAdminMemoriesExport_Empty verifies that Export returns 200 with an
// empty JSON array when no memories exist in the DB.
func TestAdminMemoriesExport_Empty(t *testing.T) {
	mock := setupTestDB(t)
	h := newAdminMemoriesHandler(t, mock)

	mock.ExpectQuery("SELECT am.id, am.content, am.scope, am.namespace, am.created_at,").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "content", "scope", "namespace", "created_at", "workspace_name",
		}))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin/memories/export", nil)

	h.Export(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 memories, got %d", len(result))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestAdminMemoriesExport_MultipleMemories verifies that Export joins
// agent_memories with workspaces and returns the correct JSON fields.
func TestAdminMemoriesExport_MultipleMemories(t *testing.T) {
	mock := setupTestDB(t)
	h := newAdminMemoriesHandler(t, mock)

	cols := []string{"id", "content", "scope", "namespace", "created_at", "workspace_name"}
	createdAt := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)
	mock.ExpectQuery("SELECT am.id, am.content, am.scope, am.namespace, am.created_at,").
		WillReturnRows(sqlmock.NewRows(cols).
			AddRow("mem-001", "remember the config", "local", "general", createdAt, "ws-alpha").
			AddRow("mem-002", "use TLS", "global", "security", createdAt.Add(time.Hour), "ws-beta"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin/memories/export", nil)

	h.Export(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 memories, got %d", len(result))
	}
	if result[0]["id"] != "mem-001" {
		t.Errorf("expected id 'mem-001', got %v", result[0]["id"])
	}
	if result[0]["scope"] != "local" {
		t.Errorf("expected scope 'local', got %v", result[0]["scope"])
	}
	if result[0]["workspace_name"] != "ws-alpha" {
		t.Errorf("expected workspace_name 'ws-alpha', got %v", result[0]["workspace_name"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestAdminMemoriesExport_QueryError_Returns500 verifies that a DB query
// error causes Export to return 500.
func TestAdminMemoriesExport_QueryError_Returns500(t *testing.T) {
	mock := setupTestDB(t)
	h := newAdminMemoriesHandler(t, mock)

	mock.ExpectQuery("SELECT am.id, am.content, am.scope, am.namespace, am.created_at,").
		WillReturnError(errors.New("db: connection refused"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin/memories/export", nil)

	h.Export(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on DB query error, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestAdminMemoriesExport_RowsErr_Returns500 verifies that a rows.Err()
// set during iteration causes Export to return 500.
func TestAdminMemoriesExport_RowsErr_Returns500(t *testing.T) {
	mock := setupTestDB(t)
	h := newAdminMemoriesHandler(t, mock)

	// Inject a row-level error at index 0 (same technique as checkpoints_test.go).
	cols := []string{"id", "content", "scope", "namespace", "created_at", "workspace_name"}
	createdAt := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)
	mock.ExpectQuery("SELECT am.id, am.content, am.scope, am.namespace, am.created_at,").
		WillReturnRows(sqlmock.NewRows(cols).
			AddRow("mem-001", "some content", "local", "general", createdAt, "ws-a").
			RowError(0, errors.New("storage fault")))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin/memories/export", nil)

	h.Export(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on rows.Err(), got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// ---------- Import ----------

// TestAdminMemoriesImport_InvalidJSON_Returns400 verifies that a malformed
// request body causes Import to return 400.
func TestAdminMemoriesImport_InvalidJSON_Returns400(t *testing.T) {
	mock := setupTestDB(t)
	h := newAdminMemoriesHandler(t, mock)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/admin/memories/import",
		bytes.NewBufferString("{ not valid json }"))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Import(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 on invalid JSON, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestAdminMemoriesImport_EmptyArray_ReturnsAllZeros verifies that an empty
// array body returns all counts at zero.
func TestAdminMemoriesImport_EmptyArray_ReturnsAllZeros(t *testing.T) {
	mock := setupTestDB(t)
	h := newAdminMemoriesHandler(t, mock)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/admin/memories/import",
		bytes.NewBufferString("[]"))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Import(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["imported"] != float64(0) {
		t.Errorf("expected imported=0, got %v", resp["imported"])
	}
	if resp["skipped"] != float64(0) {
		t.Errorf("expected skipped=0, got %v", resp["skipped"])
	}
	if resp["errors"] != float64(0) {
		t.Errorf("expected errors=0, got %v", resp["errors"])
	}
	if resp["total"] != float64(0) {
		t.Errorf("expected total=0, got %v", resp["total"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestAdminMemoriesImport_WorkspaceNotFound_Skips verifies that an entry
// whose workspace name does not exist in workspaces is counted as skipped.
func TestAdminMemoriesImport_WorkspaceNotFound_Skips(t *testing.T) {
	mock := setupTestDB(t)
	h := newAdminMemoriesHandler(t, mock)

	// Workspace lookup returns no rows → workspace not found.
	mock.ExpectQuery("SELECT id FROM workspaces WHERE name = \\$1 LIMIT 1").
		WithArgs("nonexistent-ws").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := []map[string]interface{}{
		{"content": "some memory", "scope": "local", "namespace": "general",
			"workspace_name": "nonexistent-ws"},
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/admin/memories/import",
		bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Import(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["imported"] != float64(0) {
		t.Errorf("expected imported=0, got %v", resp["imported"])
	}
	if resp["skipped"] != float64(1) {
		t.Errorf("expected skipped=1, got %v", resp["skipped"])
	}
	if resp["errors"] != float64(0) {
		t.Errorf("expected errors=0, got %v", resp["errors"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestAdminMemoriesImport_Duplicate_Skips verifies that an entry that
// already exists (same workspace_id + content + scope) is counted as skipped.
func TestAdminMemoriesImport_Duplicate_Skips(t *testing.T) {
	mock := setupTestDB(t)
	h := newAdminMemoriesHandler(t, mock)

	// Workspace lookup succeeds.
	mock.ExpectQuery("SELECT id FROM workspaces WHERE name = \\$1 LIMIT 1").
		WithArgs("ws-alpha").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("ws-001"))

	// Duplicate check returns true.
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("ws-001", "remember the config", "local").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := []map[string]interface{}{
		{"content": "remember the config", "scope": "local", "namespace": "general",
			"workspace_name": "ws-alpha"},
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/admin/memories/import",
		bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Import(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["imported"] != float64(0) {
		t.Errorf("expected imported=0 for duplicate, got %v", resp["imported"])
	}
	if resp["skipped"] != float64(1) {
		t.Errorf("expected skipped=1 for duplicate, got %v", resp["skipped"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestAdminMemoriesImport_NewMemory_Inserts verifies that a non-duplicate
// entry with a valid workspace is inserted and counted as imported.
func TestAdminMemoriesImport_NewMemory_Inserts(t *testing.T) {
	mock := setupTestDB(t)
	h := newAdminMemoriesHandler(t, mock)

	// Workspace lookup succeeds.
	mock.ExpectQuery("SELECT id FROM workspaces WHERE name = \\$1 LIMIT 1").
		WithArgs("ws-alpha").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("ws-001"))

	// Duplicate check returns false.
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("ws-001", "remember the config", "local").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	// Insert without created_at (empty string).
	mock.ExpectExec("INSERT INTO agent_memories").
		WithArgs("ws-001", "remember the config", "local", "general").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := []map[string]interface{}{
		{"content": "remember the config", "scope": "local", "namespace": "general",
			"workspace_name": "ws-alpha"},
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/admin/memories/import",
		bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Import(c)

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
	if resp["errors"] != float64(0) {
		t.Errorf("expected errors=0, got %v", resp["errors"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestAdminMemoriesImport_PreservesCreatedAt verifies that when
// CreatedAt is provided (RFC3339 string), the original timestamp is
// preserved in the INSERT.
func TestAdminMemoriesImport_PreservesCreatedAt(t *testing.T) {
	mock := setupTestDB(t)
	h := newAdminMemoriesHandler(t, mock)

	mock.ExpectQuery("SELECT id FROM workspaces WHERE name = \\$1 LIMIT 1").
		WithArgs("ws-alpha").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("ws-001"))

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("ws-001", "remember the config", "local").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	// Insert with created_at preserved.
	mock.ExpectExec("INSERT INTO agent_memories").
		WithArgs("ws-001", "remember the config", "local", "general", "2026-01-15T09:00:00Z").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := []map[string]interface{}{
		{"content": "remember the config", "scope": "local", "namespace": "general",
			"workspace_name": "ws-alpha", "created_at": "2026-01-15T09:00:00Z"},
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/admin/memories/import",
		bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Import(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["imported"] != float64(1) {
		t.Errorf("expected imported=1, got %v", resp["imported"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestAdminMemoriesImport_InsertError_ErrorsCount verifies that a DB insert
// error increments the errors counter (not imported or skipped).
func TestAdminMemoriesImport_InsertError_ErrorsCount(t *testing.T) {
	mock := setupTestDB(t)
	h := newAdminMemoriesHandler(t, mock)

	mock.ExpectQuery("SELECT id FROM workspaces WHERE name = \\$1 LIMIT 1").
		WithArgs("ws-alpha").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("ws-001"))

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("ws-001", "remember the config", "local").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	mock.ExpectExec("INSERT INTO agent_memories").
		WithArgs("ws-001", "remember the config", "local", "general").
		WillReturnError(errors.New("db: unique constraint violation"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := []map[string]interface{}{
		{"content": "remember the config", "scope": "local", "namespace": "general",
			"workspace_name": "ws-alpha"},
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/admin/memories/import",
		bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Import(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (errors counted internally), got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["imported"] != float64(0) {
		t.Errorf("expected imported=0 on insert error, got %v", resp["imported"])
	}
	if resp["errors"] != float64(1) {
		t.Errorf("expected errors=1 on insert error, got %v", resp["errors"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestAdminMemoriesImport_DefaultNamespace verifies that when namespace is
// empty, "general" is used as the default.
func TestAdminMemoriesImport_DefaultNamespace(t *testing.T) {
	mock := setupTestDB(t)
	h := newAdminMemoriesHandler(t, mock)

	mock.ExpectQuery("SELECT id FROM workspaces WHERE name = \\$1 LIMIT 1").
		WithArgs("ws-alpha").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("ws-001"))

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("ws-001", "some content", "local").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	// Namespace defaults to "general".
	mock.ExpectExec("INSERT INTO agent_memories").
		WithArgs("ws-001", "some content", "local", "general").
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := []map[string]interface{}{
		{"content": "some content", "scope": "local",
			"workspace_name": "ws-alpha"},
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/admin/memories/import",
		bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Import(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["imported"] != float64(1) {
		t.Errorf("expected imported=1, got %v", resp["imported"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}
