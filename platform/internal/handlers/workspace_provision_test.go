package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/models"
)

// ==================== workspaceAwarenessNamespace ====================

func TestWorkspaceAwarenessNamespace(t *testing.T) {
	tests := []struct {
		workspaceID string
		expected    string
	}{
		{"ws-123", "workspace:ws-123"},
		{"abc-def-ghi", "workspace:abc-def-ghi"},
		{"", "workspace:"},
	}

	for _, tt := range tests {
		t.Run(tt.workspaceID, func(t *testing.T) {
			result := workspaceAwarenessNamespace(tt.workspaceID)
			if result != tt.expected {
				t.Errorf("workspaceAwarenessNamespace(%q) = %q, want %q", tt.workspaceID, result, tt.expected)
			}
		})
	}
}

// ==================== configDirName ====================

func TestConfigDirName(t *testing.T) {
	tests := []struct {
		workspaceID string
		expected    string
	}{
		{"abc-def-ghi", "ws-abc-def-ghi"},
		{"abcdefghijklmnop", "ws-abcdefghijkl"}, // truncated at 12
		{"short", "ws-short"},
		{"123456789012", "ws-123456789012"}, // exactly 12
		{"1234567890123", "ws-123456789012"}, // 13 chars, truncated
	}

	for _, tt := range tests {
		t.Run(tt.workspaceID, func(t *testing.T) {
			result := configDirName(tt.workspaceID)
			if result != tt.expected {
				t.Errorf("configDirName(%q) = %q, want %q", tt.workspaceID, result, tt.expected)
			}
		})
	}
}

// ==================== findTemplateByName ====================

func TestFindTemplateByName_ByDirName(t *testing.T) {
	tmpDir := t.TempDir()

	// Create template dirs
	os.MkdirAll(filepath.Join(tmpDir, "seo-agent"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "data-analyst"), 0755)

	result := findTemplateByName(tmpDir, "SEO Agent")
	if result != "seo-agent" {
		t.Errorf("expected 'seo-agent', got %q", result)
	}

	result = findTemplateByName(tmpDir, "Data Analyst")
	if result != "data-analyst" {
		t.Errorf("expected 'data-analyst', got %q", result)
	}
}

func TestFindTemplateByName_ByConfigYAML(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a template dir with a different name than the workspace
	templateDir := filepath.Join(tmpDir, "org-pm")
	os.MkdirAll(templateDir, 0755)
	os.WriteFile(filepath.Join(templateDir, "config.yaml"), []byte("name: Project Manager\nversion: 1.0\n"), 0644)

	result := findTemplateByName(tmpDir, "Project Manager")
	if result != "org-pm" {
		t.Errorf("expected 'org-pm', got %q", result)
	}
}

func TestFindTemplateByName_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	result := findTemplateByName(tmpDir, "Nonexistent Agent")
	if result != "" {
		t.Errorf("expected empty string for missing template, got %q", result)
	}
}

func TestFindTemplateByName_SkipsWsPrefix(t *testing.T) {
	tmpDir := t.TempDir()

	// Dirs starting with "ws-" are workspace instance dirs, should be skipped in YAML search
	wsDir := filepath.Join(tmpDir, "ws-12345678")
	os.MkdirAll(wsDir, 0755)
	os.WriteFile(filepath.Join(wsDir, "config.yaml"), []byte("name: Test Agent\n"), 0644)

	result := findTemplateByName(tmpDir, "Test Agent")
	if result != "" {
		t.Errorf("expected empty string (ws- dirs should be skipped), got %q", result)
	}
}

func TestFindTemplateByName_InvalidDir(t *testing.T) {
	result := findTemplateByName("/nonexistent/path", "Any Agent")
	if result != "" {
		t.Errorf("expected empty string for invalid dir, got %q", result)
	}
}

// ==================== ensureDefaultConfig ====================

func TestEnsureDefaultConfig_LangGraph(t *testing.T) {
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	payload := models.CreateWorkspacePayload{
		Name:    "Test Agent",
		Tier:    1,
		Runtime: "langgraph",
	}

	files := handler.ensureDefaultConfig("ws-test-123", payload)

	configYAML, ok := files["config.yaml"]
	if !ok {
		t.Fatal("expected config.yaml in generated files")
	}

	content := string(configYAML)
	if !contains(content, "name: Test Agent") {
		t.Errorf("config.yaml missing name, got:\n%s", content)
	}
	if !contains(content, "runtime: langgraph") {
		t.Errorf("config.yaml missing runtime, got:\n%s", content)
	}
	if !contains(content, "tier: 1") {
		t.Errorf("config.yaml missing tier, got:\n%s", content)
	}
	if !contains(content, "model: anthropic:claude-sonnet-4-6") {
		t.Errorf("config.yaml should use default langgraph model, got:\n%s", content)
	}
}

func TestEnsureDefaultConfig_ClaudeCode(t *testing.T) {
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	payload := models.CreateWorkspacePayload{
		Name:    "Code Agent",
		Tier:    2,
		Runtime: "claude-code",
	}

	files := handler.ensureDefaultConfig("ws-code-123", payload)

	configYAML, ok := files["config.yaml"]
	if !ok {
		t.Fatal("expected config.yaml in generated files")
	}

	content := string(configYAML)
	if !contains(content, "runtime: claude-code") {
		t.Errorf("config.yaml missing runtime, got:\n%s", content)
	}
	if !contains(content, "model: sonnet") {
		t.Errorf("config.yaml should use default claude-code model, got:\n%s", content)
	}
	if !contains(content, "runtime_config:") {
		t.Errorf("config.yaml should have runtime_config section for claude-code, got:\n%s", content)
	}
	if !contains(content, "required_env:") {
		t.Errorf("config.yaml should have required_env for claude-code, got:\n%s", content)
	}
	if !contains(content, "CLAUDE_CODE_OAUTH_TOKEN") {
		t.Errorf("config.yaml should require CLAUDE_CODE_OAUTH_TOKEN, got:\n%s", content)
	}
	// Should NOT have .auth-token file
	if _, ok := files[".auth-token"]; ok {
		t.Error("claude-code should not generate .auth-token file — use env vars via secrets API")
	}
}

func TestEnsureDefaultConfig_CustomModel(t *testing.T) {
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	payload := models.CreateWorkspacePayload{
		Name:    "Custom Agent",
		Tier:    1,
		Runtime: "langgraph",
		Model:   "gpt-4o",
	}

	files := handler.ensureDefaultConfig("ws-custom", payload)

	configYAML := string(files["config.yaml"])
	if !contains(configYAML, "model: gpt-4o") {
		t.Errorf("config.yaml should use custom model, got:\n%s", configYAML)
	}
}

func TestEnsureDefaultConfig_SpecialCharsInName(t *testing.T) {
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	payload := models.CreateWorkspacePayload{
		Name:    "Agent: With Special #Chars",
		Role:    "worker: {advanced}",
		Tier:    1,
		Runtime: "langgraph",
	}

	files := handler.ensureDefaultConfig("ws-special", payload)

	configYAML := string(files["config.yaml"])
	// Names with special chars should be quoted
	if !contains(configYAML, fmt.Sprintf("%q", "Agent: With Special #Chars")) {
		t.Errorf("config.yaml should quote name with special chars, got:\n%s", configYAML)
	}
}

func TestEnsureDefaultConfig_OpenClawGetsRuntimeConfig(t *testing.T) {
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	payload := models.CreateWorkspacePayload{
		Name:    "OpenClaw Agent",
		Tier:    1,
		Runtime: "openclaw",
		Model:   "openai:gpt-4o",
	}

	files := handler.ensureDefaultConfig("ws-openclaw", payload)
	configYAML := string(files["config.yaml"])
	if !contains(configYAML, "runtime_config:") {
		t.Errorf("openclaw should have runtime_config, got:\n%s", configYAML)
	}
	if !contains(configYAML, "model: openai:gpt-4o") {
		t.Errorf("model should be at top level, got:\n%s", configYAML)
	}
}

func TestEnsureDefaultConfig_CrewAIGetsRuntimeConfig(t *testing.T) {
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	payload := models.CreateWorkspacePayload{
		Name:    "CrewAI Agent",
		Tier:    1,
		Runtime: "crewai",
	}

	files := handler.ensureDefaultConfig("ws-crewai", payload)
	configYAML := string(files["config.yaml"])
	if !contains(configYAML, "runtime_config:") {
		t.Errorf("crewai should have runtime_config, got:\n%s", configYAML)
	}
	// crewai falls into the default case — runtime_config with timeout only, no required_env
	if !contains(configYAML, "timeout: 0") {
		t.Errorf("crewai should have timeout in runtime_config, got:\n%s", configYAML)
	}
}

func TestEnsureDefaultConfig_EmptyRuntimeDefaultsToLangGraph(t *testing.T) {
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	payload := models.CreateWorkspacePayload{
		Name: "Default Agent",
		Tier: 1,
	}

	files := handler.ensureDefaultConfig("ws-empty-rt", payload)
	configYAML := string(files["config.yaml"])
	if !contains(configYAML, "runtime: langgraph") {
		t.Errorf("empty runtime should default to langgraph, got:\n%s", configYAML)
	}
	if !contains(configYAML, "model: anthropic:claude-sonnet-4-6") {
		t.Errorf("langgraph default model should be anthropic, got:\n%s", configYAML)
	}
}

func TestEnsureDefaultConfig_EmptyNameAndRole(t *testing.T) {
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	payload := models.CreateWorkspacePayload{
		Tier:    1,
		Runtime: "langgraph",
	}

	files := handler.ensureDefaultConfig("ws-empty-name", payload)
	configYAML := string(files["config.yaml"])
	// Should not panic — empty name/role produce valid YAML
	if !contains(configYAML, "name: ") {
		t.Errorf("config.yaml should have name field, got:\n%s", configYAML)
	}
	if !contains(configYAML, "runtime: langgraph") {
		t.Errorf("config.yaml should have runtime, got:\n%s", configYAML)
	}
}

func TestEnsureDefaultConfig_DeepAgents(t *testing.T) {
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	payload := models.CreateWorkspacePayload{
		Name:    "Deep Agent",
		Tier:    2,
		Runtime: "deepagents",
		Model:   "google_genai:gemini-2.5-flash",
	}

	files := handler.ensureDefaultConfig("ws-deep", payload)

	configYAML := string(files["config.yaml"])
	if !contains(configYAML, "runtime: deepagents") {
		t.Errorf("config.yaml missing runtime, got:\n%s", configYAML)
	}
	if !contains(configYAML, "model: google_genai:gemini-2.5-flash") {
		t.Errorf("config.yaml should have model at top level, got:\n%s", configYAML)
	}
	// deepagents should NOT have runtime_config block
	if contains(configYAML, "runtime_config:") {
		t.Errorf("config.yaml should NOT have runtime_config for deepagents, got:\n%s", configYAML)
	}
	// Should NOT have auth token
	if _, ok := files[".auth-token"]; ok {
		t.Error("deepagents should not get .auth-token")
	}
}

func TestEnsureDefaultConfig_ModelAlwaysTopLevel(t *testing.T) {
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	for _, runtime := range []string{"langgraph", "deepagents", "claude-code"} {
		t.Run(runtime, func(t *testing.T) {
			payload := models.CreateWorkspacePayload{
				Name:    "Agent",
				Tier:    1,
				Runtime: runtime,
				Model:   "test-model",
			}
			files := handler.ensureDefaultConfig("ws-"+runtime, payload)
			configYAML := string(files["config.yaml"])
			if !contains(configYAML, "model: test-model") {
				t.Errorf("config.yaml missing top-level model for runtime %s, got:\n%s", runtime, configYAML)
			}
		})
	}
}

// ==================== buildProvisionerConfig ====================

func TestBuildProvisionerConfig_BasicFields(t *testing.T) {
	broadcaster := newTestBroadcaster()
	tmpDir := t.TempDir()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", tmpDir)

	templatePath := filepath.Join(tmpDir, "template")
	pluginsPath := t.TempDir()
	cfg := handler.buildProvisionerConfig(
		"ws-basic",
		templatePath,
		map[string][]byte{"config.yaml": []byte("name: test")},
		models.CreateWorkspacePayload{Tier: 1, Runtime: "langgraph"},
		map[string]string{"API_KEY": "secret"},
		pluginsPath,
		"workspace:ws-basic",
	)

	if cfg.WorkspaceID != "ws-basic" {
		t.Errorf("expected WorkspaceID 'ws-basic', got %q", cfg.WorkspaceID)
	}
	if cfg.Tier != 1 {
		t.Errorf("expected Tier 1, got %d", cfg.Tier)
	}
	if cfg.Runtime != "langgraph" {
		t.Errorf("expected Runtime 'langgraph', got %q", cfg.Runtime)
	}
	if cfg.PlatformURL != "http://localhost:8080" {
		t.Errorf("expected PlatformURL 'http://localhost:8080', got %q", cfg.PlatformURL)
	}
	if cfg.AwarenessNamespace != "workspace:ws-basic" {
		t.Errorf("expected AwarenessNamespace 'workspace:ws-basic', got %q", cfg.AwarenessNamespace)
	}
	if cfg.PluginsPath != pluginsPath {
		t.Errorf("expected PluginsPath %q, got %q", pluginsPath, cfg.PluginsPath)
	}
	if cfg.EnvVars["API_KEY"] != "secret" {
		t.Errorf("expected EnvVars to include API_KEY, got %v", cfg.EnvVars)
	}
	if cfg.TemplatePath != templatePath {
		t.Errorf("expected TemplatePath %q, got %q", templatePath, cfg.TemplatePath)
	}
}

func TestBuildProvisionerConfig_WorkspacePathFromEnv(t *testing.T) {
	broadcaster := newTestBroadcaster()
	handler := NewWorkspaceHandler(broadcaster, nil, "http://localhost:8080", t.TempDir())

	workspaceDir := t.TempDir()
	t.Setenv("WORKSPACE_DIR", workspaceDir)
	t.Setenv("AWARENESS_URL", "http://awareness:37800")

	pluginsPath := t.TempDir()
	cfg := handler.buildProvisionerConfig(
		"ws-env",
		"",
		nil,
		models.CreateWorkspacePayload{Tier: 2, Runtime: "claude-code"},
		nil,
		pluginsPath,
		"workspace:ws-env",
	)

	if cfg.WorkspacePath != workspaceDir {
		t.Errorf("expected WorkspacePath from env, got %q", cfg.WorkspacePath)
	}
	if cfg.AwarenessURL != "http://awareness:37800" {
		t.Errorf("expected AwarenessURL from env, got %q", cfg.AwarenessURL)
	}
}

// contains is a helper for substring matching in tests
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
