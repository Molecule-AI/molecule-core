package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

// ==================== validateRelPath ====================

func TestValidateRelPath_Valid(t *testing.T) {
	cases := []string{
		"config.yaml",
		"skills/my-skill/SKILL.md",
		"system-prompt.md",
		"a/b/c.txt",
	}
	for _, tc := range cases {
		if err := validateRelPath(tc); err != nil {
			t.Errorf("expected valid path %q, got error: %v", tc, err)
		}
	}
}

func TestValidateRelPath_Invalid(t *testing.T) {
	cases := []string{
		"../etc/passwd",
		"../../secrets",
		"/absolute/path",
	}
	for _, tc := range cases {
		if err := validateRelPath(tc); err == nil {
			t.Errorf("expected error for path %q, got nil", tc)
		}
	}
}

// ==================== GET /templates ====================

func TestTemplatesList_EmptyDir(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)

	tmpDir := t.TempDir()
	handler := NewTemplatesHandler(tmpDir, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/templates", nil)

	handler.List(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp []templateSummary
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("expected empty list, got %d items", len(resp))
	}
}

func TestTemplatesList_WithTemplates(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)

	tmpDir := t.TempDir()

	// Create a template directory with config.yaml
	tmplDir := filepath.Join(tmpDir, "test-agent")
	os.MkdirAll(tmplDir, 0755)
	configYaml := `name: Test Agent
description: A test agent
tier: 2
model: anthropic:claude-sonnet-4-20250514
skills:
  - web-search
  - code-review
`
	os.WriteFile(filepath.Join(tmplDir, "config.yaml"), []byte(configYaml), 0644)

	// Create a non-directory file (should be skipped)
	os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# readme"), 0644)

	// Create a directory without config.yaml (should be skipped)
	os.MkdirAll(filepath.Join(tmpDir, "no-config"), 0755)

	handler := NewTemplatesHandler(tmpDir, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/templates", nil)

	handler.List(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp []templateSummary
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("expected 1 template, got %d", len(resp))
	}
	if resp[0].ID != "test-agent" {
		t.Errorf("expected ID 'test-agent', got %q", resp[0].ID)
	}
	if resp[0].Name != "Test Agent" {
		t.Errorf("expected Name 'Test Agent', got %q", resp[0].Name)
	}
	if resp[0].Tier != 2 {
		t.Errorf("expected Tier 2, got %d", resp[0].Tier)
	}
	if resp[0].SkillCount != 2 {
		t.Errorf("expected SkillCount 2, got %d", resp[0].SkillCount)
	}
}

func TestTemplatesList_NonexistentDir(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)

	handler := NewTemplatesHandler("/nonexistent/path/to/templates", nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/templates", nil)

	handler.List(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp []templateSummary
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp) != 0 {
		t.Errorf("expected empty list, got %d items", len(resp))
	}
}

// ==================== GET /workspaces/:id/files ====================

func TestListFiles_InvalidRoot(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	handler := NewTemplatesHandler(t.TempDir(), nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-1/files?root=/etc", nil)
	// Need to set query params
	c.Request.URL.RawQuery = "root=/etc"

	handler.ListFiles(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	// Verify no DB call was made (early return before DB query)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestListFiles_WorkspaceNotFound(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	handler := NewTemplatesHandler(t.TempDir(), nil)

	mock.ExpectQuery("SELECT name FROM workspaces WHERE id =").
		WithArgs("ws-nonexist").
		WillReturnError(sql.ErrNoRows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-nonexist"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-nonexist/files", nil)

	handler.ListFiles(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestListFiles_FallbackToHost_NoTemplate(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	tmpDir := t.TempDir()
	handler := NewTemplatesHandler(tmpDir, nil) // nil docker = no container

	mock.ExpectQuery("SELECT name FROM workspaces WHERE id =").
		WithArgs("ws-fallback").
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("Unknown Agent"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-fallback"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-fallback/files", nil)

	handler.ListFiles(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Should return empty list
	var resp []interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp) != 0 {
		t.Errorf("expected empty file list, got %d items", len(resp))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestListFiles_FallbackToHost_WithTemplate(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	tmpDir := t.TempDir()
	// Create a template matching the workspace name
	tmplDir := filepath.Join(tmpDir, "test-agent")
	os.MkdirAll(tmplDir, 0755)
	os.WriteFile(filepath.Join(tmplDir, "config.yaml"), []byte("name: Test Agent\n"), 0644)
	os.WriteFile(filepath.Join(tmplDir, "system-prompt.md"), []byte("# prompt"), 0644)

	handler := NewTemplatesHandler(tmpDir, nil)

	mock.ExpectQuery("SELECT name FROM workspaces WHERE id =").
		WithArgs("ws-tmpl").
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("Test Agent"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-tmpl"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-tmpl/files", nil)

	handler.ListFiles(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp) < 2 {
		t.Errorf("expected at least 2 files, got %d", len(resp))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// ==================== GET /workspaces/:id/files/*path ====================

func TestReadFile_PathTraversal(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)

	handler := NewTemplatesHandler(t.TempDir(), nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "id", Value: "ws-1"},
		{Key: "path", Value: "/../../../etc/passwd"},
	}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-1/files/../../../etc/passwd", nil)

	handler.ReadFile(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReadFile_InvalidRoot(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)

	handler := NewTemplatesHandler(t.TempDir(), nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "id", Value: "ws-1"},
		{Key: "path", Value: "/config.yaml"},
	}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-1/files/config.yaml?root=/tmp", nil)
	c.Request.URL.RawQuery = "root=/tmp"

	handler.ReadFile(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReadFile_WorkspaceNotFound(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	handler := NewTemplatesHandler(t.TempDir(), nil)

	mock.ExpectQuery("SELECT name FROM workspaces WHERE id =").
		WithArgs("ws-nf").
		WillReturnError(sql.ErrNoRows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "id", Value: "ws-nf"},
		{Key: "path", Value: "/config.yaml"},
	}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-nf/files/config.yaml", nil)

	handler.ReadFile(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestReadFile_FallbackToHost_Success(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	tmpDir := t.TempDir()
	tmplDir := filepath.Join(tmpDir, "reader-agent")
	os.MkdirAll(tmplDir, 0755)
	os.WriteFile(filepath.Join(tmplDir, "config.yaml"), []byte("name: Reader Agent\ntier: 1\n"), 0644)

	handler := NewTemplatesHandler(tmpDir, nil)

	mock.ExpectQuery("SELECT name FROM workspaces WHERE id =").
		WithArgs("ws-read").
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("Reader Agent"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "id", Value: "ws-read"},
		{Key: "path", Value: "/config.yaml"},
	}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-read/files/config.yaml", nil)

	handler.ReadFile(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["path"] != "config.yaml" {
		t.Errorf("expected path 'config.yaml', got %v", resp["path"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestReadFile_FallbackToHost_NotFound(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	tmpDir := t.TempDir()
	handler := NewTemplatesHandler(tmpDir, nil)

	mock.ExpectQuery("SELECT name FROM workspaces WHERE id =").
		WithArgs("ws-nofile").
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("No File Agent"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "id", Value: "ws-nofile"},
		{Key: "path", Value: "/nonexistent.txt"},
	}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-nofile/files/nonexistent.txt", nil)

	handler.ReadFile(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// ==================== PUT /workspaces/:id/files/*path ====================

func TestWriteFile_PathTraversal(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)

	handler := NewTemplatesHandler(t.TempDir(), nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "id", Value: "ws-1"},
		{Key: "path", Value: "/../../../etc/shadow"},
	}
	body := `{"content": "malicious"}`
	c.Request = httptest.NewRequest("PUT", "/workspaces/ws-1/files/../../../etc/shadow",
		strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.WriteFile(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWriteFile_InvalidBody(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)

	handler := NewTemplatesHandler(t.TempDir(), nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "id", Value: "ws-1"},
		{Key: "path", Value: "/config.yaml"},
	}
	c.Request = httptest.NewRequest("PUT", "/workspaces/ws-1/files/config.yaml",
		strings.NewReader("not json"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.WriteFile(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWriteFile_WorkspaceNotFound(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	handler := NewTemplatesHandler(t.TempDir(), nil)

	mock.ExpectQuery("SELECT name FROM workspaces WHERE id =").
		WithArgs("ws-wf-nf").
		WillReturnError(sql.ErrNoRows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "id", Value: "ws-wf-nf"},
		{Key: "path", Value: "/config.yaml"},
	}
	body := `{"content": "name: test"}`
	c.Request = httptest.NewRequest("PUT", "/workspaces/ws-wf-nf/files/config.yaml",
		strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.WriteFile(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// ==================== DELETE /workspaces/:id/files/*path ====================

func TestDeleteFile_PathTraversal(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)

	handler := NewTemplatesHandler(t.TempDir(), nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "id", Value: "ws-1"},
		{Key: "path", Value: "/../../../etc/passwd"},
	}
	c.Request = httptest.NewRequest("DELETE", "/workspaces/ws-1/files/../../../etc/passwd", nil)

	handler.DeleteFile(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteFile_WorkspaceNotFound(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	handler := NewTemplatesHandler(t.TempDir(), nil)

	mock.ExpectQuery("SELECT name FROM workspaces WHERE id =").
		WithArgs("ws-del-nf").
		WillReturnError(sql.ErrNoRows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "id", Value: "ws-del-nf"},
		{Key: "path", Value: "/old-file.txt"},
	}
	c.Request = httptest.NewRequest("DELETE", "/workspaces/ws-del-nf/files/old-file.txt", nil)

	handler.DeleteFile(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// ==================== GET /workspaces/:id/shared-context ====================

func TestSharedContext_WorkspaceNotFound(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	handler := NewTemplatesHandler(t.TempDir(), nil)

	mock.ExpectQuery("SELECT name FROM workspaces WHERE id =").
		WithArgs("ws-sc-nf").
		WillReturnError(sql.ErrNoRows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-sc-nf"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-sc-nf/shared-context", nil)

	handler.SharedContext(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestSharedContext_NoTemplate(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	tmpDir := t.TempDir()
	handler := NewTemplatesHandler(tmpDir, nil) // no docker

	mock.ExpectQuery("SELECT name FROM workspaces WHERE id =").
		WithArgs("ws-sc-nt").
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("Unknown Agent"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-sc-nt"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-sc-nt/shared-context", nil)

	handler.SharedContext(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Should return empty array
	var resp []interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp) != 0 {
		t.Errorf("expected empty list, got %d items", len(resp))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestSharedContext_WithFiles(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	tmpDir := t.TempDir()
	tmplDir := filepath.Join(tmpDir, "ctx-agent")
	os.MkdirAll(tmplDir, 0755)
	os.WriteFile(filepath.Join(tmplDir, "config.yaml"), []byte("name: Ctx Agent\nshared_context:\n  - rules.md\n  - style.md\n"), 0644)
	os.WriteFile(filepath.Join(tmplDir, "rules.md"), []byte("# Rules\nBe nice"), 0644)
	os.WriteFile(filepath.Join(tmplDir, "style.md"), []byte("# Style\nBe clear"), 0644)

	handler := NewTemplatesHandler(tmpDir, nil)

	mock.ExpectQuery("SELECT name FROM workspaces WHERE id =").
		WithArgs("ws-sc-ok").
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("Ctx Agent"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-sc-ok"}}
	c.Request = httptest.NewRequest("GET", "/workspaces/ws-sc-ok/shared-context", nil)

	handler.SharedContext(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp) != 2 {
		t.Fatalf("expected 2 context files, got %d", len(resp))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// ==================== resolveTemplateDir ====================

func TestResolveTemplateDir_ByNormalizedName(t *testing.T) {
	tmpDir := t.TempDir()
	tmplDir := filepath.Join(tmpDir, "my-agent")
	os.MkdirAll(tmplDir, 0755)

	handler := NewTemplatesHandler(tmpDir, nil)
	result := handler.resolveTemplateDir("My Agent")

	if result != tmplDir {
		t.Errorf("expected %q, got %q", tmplDir, result)
	}
}

func TestResolveTemplateDir_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	handler := NewTemplatesHandler(tmpDir, nil)
	result := handler.resolveTemplateDir("Nonexistent Agent")

	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}
