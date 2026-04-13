package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

// ==================== normalizeName ====================

func TestNormalizeName(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"My Agent", "my-agent"},
		{"SEO Agent", "seo-agent"},
		{"hello_world", "hello_world"},
		{"Agent v2.0", "agent-v20"},
		{"UPPER CASE", "upper-case"},
		{"a-b-c", "a-b-c"},
		{"../hack", "hack"},
		{"", "unnamed"},
		{"$$$", "unnamed"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			result := normalizeName(tc.input)
			if result != tc.expected {
				t.Errorf("normalizeName(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

// ==================== generateDefaultConfig ====================

func TestGenerateDefaultConfig_WithFiles(t *testing.T) {
	files := map[string]string{
		"system-prompt.md":          "# System prompt",
		"rules.md":                  "# Rules",
		"skills/search/SKILL.md":    "Search skill",
		"skills/review/SKILL.md":    "Review skill",
		"skills/review/templates.md": "Templates",
	}

	cfg := generateDefaultConfig("Test Agent", files)

	if !strings.Contains(cfg, "name: Test Agent") {
		t.Error("config should contain agent name")
	}
	if !strings.Contains(cfg, "tier: 1") {
		t.Error("config should default to tier 1")
	}
	// Should detect prompt files
	if !strings.Contains(cfg, "system-prompt.md") {
		t.Error("config should include system-prompt.md as prompt file")
	}
	if !strings.Contains(cfg, "rules.md") {
		t.Error("config should include rules.md as prompt file")
	}
	// Should detect skills
	if !strings.Contains(cfg, "search") {
		t.Error("config should include 'search' skill")
	}
	if !strings.Contains(cfg, "review") {
		t.Error("config should include 'review' skill")
	}
}

func TestGenerateDefaultConfig_Empty(t *testing.T) {
	files := map[string]string{
		"data/something.json": `{"key": "value"}`,
	}

	cfg := generateDefaultConfig("Empty Agent", files)

	if !strings.Contains(cfg, "name: Empty Agent") {
		t.Error("config should contain agent name")
	}
	// Should fallback to default prompt file
	if !strings.Contains(cfg, "system-prompt.md") {
		t.Error("config should include default system-prompt.md")
	}
}

// ==================== writeFiles ====================

func TestWriteFiles_Success(t *testing.T) {
	tmpDir := t.TempDir()
	files := map[string]string{
		"config.yaml":            "name: Test\n",
		"skills/test/SKILL.md":   "# Test Skill",
		"system-prompt.md":       "# System Prompt",
	}

	if err := writeFiles(tmpDir, files); err != nil {
		t.Fatalf("writeFiles failed: %v", err)
	}

	// Verify files exist
	for path, expectedContent := range files {
		data, err := os.ReadFile(filepath.Join(tmpDir, path))
		if err != nil {
			t.Errorf("expected file %s to exist: %v", path, err)
			continue
		}
		if string(data) != expectedContent {
			t.Errorf("file %s content mismatch: got %q, want %q", path, string(data), expectedContent)
		}
	}
}

func TestWriteFiles_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	files := map[string]string{
		"../escape.txt": "malicious",
	}

	err := writeFiles(tmpDir, files)
	if err == nil {
		t.Error("expected error for path traversal")
	}
}

// ==================== POST /templates/import ====================

func TestImport_Success(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)

	tmpDir := t.TempDir()
	handler := NewTemplatesHandler(tmpDir, nil)

	body := `{
		"name": "New Agent",
		"files": {
			"system-prompt.md": "# System Prompt\nYou are a helpful assistant",
			"skills/test/SKILL.md": "# Test Skill"
		}
	}`

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/templates/import", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Import(c)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "imported" {
		t.Errorf("expected status 'imported', got %v", resp["status"])
	}
	if resp["id"] != "new-agent" {
		t.Errorf("expected id 'new-agent', got %v", resp["id"])
	}

	// Verify config.yaml was auto-generated
	if _, err := os.Stat(filepath.Join(tmpDir, "new-agent", "config.yaml")); os.IsNotExist(err) {
		t.Error("config.yaml should have been auto-generated")
	}
}

func TestImport_MissingName(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)

	handler := NewTemplatesHandler(t.TempDir(), nil)

	body := `{"files": {"test.md": "content"}}`

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/templates/import", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Import(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestImport_TooManyFiles(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)

	handler := NewTemplatesHandler(t.TempDir(), nil)

	files := make(map[string]string)
	for i := 0; i <= maxUploadFiles; i++ {
		files[fmt.Sprintf("file%d.txt", i)] = "content"
	}

	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"name":  "Big Agent",
		"files": files,
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/templates/import", bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Import(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestImport_AlreadyExists(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)

	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "existing-agent"), 0755)

	handler := NewTemplatesHandler(tmpDir, nil)

	body := `{"name": "Existing Agent", "files": {"test.md": "content"}}`

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/templates/import", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Import(c)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestImport_WithConfigYaml(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)

	tmpDir := t.TempDir()
	handler := NewTemplatesHandler(tmpDir, nil)

	body := `{
		"name": "Custom Agent",
		"files": {
			"config.yaml": "name: Custom Agent\ntier: 3\nmodel: gpt-4",
			"system-prompt.md": "# You are custom"
		}
	}`

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/templates/import", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Import(c)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Verify the provided config.yaml was used (not auto-generated)
	data, err := os.ReadFile(filepath.Join(tmpDir, "custom-agent", "config.yaml"))
	if err != nil {
		t.Fatalf("failed to read config.yaml: %v", err)
	}
	if !strings.Contains(string(data), "tier: 3") {
		t.Error("config.yaml should contain provided tier: 3, not auto-generated content")
	}
}

// ==================== PUT /workspaces/:id/files (ReplaceFiles) ====================

func TestReplaceFiles_MissingBody(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)

	handler := NewTemplatesHandler(t.TempDir(), nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}
	c.Request = httptest.NewRequest("PUT", "/workspaces/ws-1/files", bytes.NewBufferString("{}"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ReplaceFiles(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReplaceFiles_TooManyFiles(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)

	handler := NewTemplatesHandler(t.TempDir(), nil)

	files := make(map[string]string)
	for i := 0; i <= maxUploadFiles; i++ {
		files[fmt.Sprintf("file%d.txt", i)] = "content"
	}
	bodyBytes, _ := json.Marshal(map[string]interface{}{"files": files})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}
	c.Request = httptest.NewRequest("PUT", "/workspaces/ws-1/files", bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ReplaceFiles(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReplaceFiles_WorkspaceNotFound(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	handler := NewTemplatesHandler(t.TempDir(), nil)

	mock.ExpectQuery("SELECT name FROM workspaces WHERE id =").
		WithArgs("ws-rf-nf").
		WillReturnError(sql.ErrNoRows)

	body := `{"files": {"test.md": "content"}}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-rf-nf"}}
	c.Request = httptest.NewRequest("PUT", "/workspaces/ws-rf-nf/files", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ReplaceFiles(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestReplaceFiles_PathTraversal(t *testing.T) {
	mock := setupTestDB(t)
	setupTestRedis(t)

	handler := NewTemplatesHandler(t.TempDir(), nil)

	mock.ExpectQuery("SELECT name FROM workspaces WHERE id =").
		WithArgs("ws-rf-pt").
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("Test Agent"))

	body := `{"files": {"../../../etc/passwd": "malicious"}}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "ws-rf-pt"}}
	c.Request = httptest.NewRequest("PUT", "/workspaces/ws-rf-pt/files", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ReplaceFiles(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}
