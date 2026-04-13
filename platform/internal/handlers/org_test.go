package handlers

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestOrgDefaults_InitialPrompt_YAMLParsing(t *testing.T) {
	raw := `
runtime: claude-code
tier: 2
initial_prompt: |
  Clone the repo and read CLAUDE.md.
  Report ready status.
`
	var defaults OrgDefaults
	if err := yaml.Unmarshal([]byte(raw), &defaults); err != nil {
		t.Fatalf("failed to parse YAML: %v", err)
	}
	if defaults.Runtime != "claude-code" {
		t.Errorf("expected runtime 'claude-code', got %q", defaults.Runtime)
	}
	if !strings.Contains(defaults.InitialPrompt, "Clone the repo") {
		t.Errorf("expected InitialPrompt to contain 'Clone the repo', got %q", defaults.InitialPrompt)
	}
	if !strings.Contains(defaults.InitialPrompt, "Report ready") {
		t.Errorf("expected InitialPrompt to contain 'Report ready', got %q", defaults.InitialPrompt)
	}
}

func TestOrgWorkspace_InitialPrompt_Override(t *testing.T) {
	raw := `
name: Frontend Engineer
role: Next.js canvas
initial_prompt: Custom FE prompt
`
	var ws OrgWorkspace
	if err := yaml.Unmarshal([]byte(raw), &ws); err != nil {
		t.Fatalf("failed to parse YAML: %v", err)
	}
	if ws.InitialPrompt != "Custom FE prompt" {
		t.Errorf("expected 'Custom FE prompt', got %q", ws.InitialPrompt)
	}
}

func TestInitialPrompt_ConfigYAML_Injection(t *testing.T) {
	// Simulate what createWorkspaceTree does: append initial_prompt to config.yaml
	configYAML := "name: Test\nruntime: claude-code\n"
	initialPrompt := "Clone the repo.\nRead CLAUDE.md.\nReport ready."

	trimmed := strings.TrimSpace(initialPrompt)
	lines := strings.Split(trimmed, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	indented := strings.Join(lines, "\n  ")
	result := configYAML + "initial_prompt: |\n  " + indented + "\n"

	// Parse result as YAML to verify it's valid
	var parsed map[string]interface{}
	if err := yaml.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("generated YAML is invalid: %v\n---\n%s", err, result)
	}

	prompt, ok := parsed["initial_prompt"].(string)
	if !ok {
		t.Fatalf("initial_prompt not found or not a string in parsed YAML")
	}
	if !strings.Contains(prompt, "Clone the repo") {
		t.Errorf("expected prompt to contain 'Clone the repo', got %q", prompt)
	}
	if !strings.Contains(prompt, "Read CLAUDE.md") {
		t.Errorf("expected prompt to contain 'Read CLAUDE.md', got %q", prompt)
	}
}

func TestInitialPrompt_ConfigYAML_Empty(t *testing.T) {
	// When initial_prompt is empty, nothing should be appended
	configYAML := "name: Test\nruntime: langgraph\n"
	initialPrompt := ""

	result := configYAML
	if initialPrompt != "" {
		// This block shouldn't execute
		result += "initial_prompt: |\n  " + initialPrompt + "\n"
	}

	var parsed map[string]interface{}
	if err := yaml.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("generated YAML is invalid: %v", err)
	}
	if _, exists := parsed["initial_prompt"]; exists {
		t.Error("initial_prompt should not exist in config when empty")
	}
}

func TestOrgDefaults_Model_YAMLParsing(t *testing.T) {
	raw := `
runtime: deepagents
tier: 2
model: google_genai:gemini-2.5-flash
`
	var defaults OrgDefaults
	if err := yaml.Unmarshal([]byte(raw), &defaults); err != nil {
		t.Fatalf("failed to parse YAML: %v", err)
	}
	if defaults.Model != "google_genai:gemini-2.5-flash" {
		t.Errorf("expected model 'google_genai:gemini-2.5-flash', got %q", defaults.Model)
	}
}

func TestOrgDefaults_Model_Empty(t *testing.T) {
	raw := `
runtime: langgraph
tier: 2
`
	var defaults OrgDefaults
	if err := yaml.Unmarshal([]byte(raw), &defaults); err != nil {
		t.Fatalf("failed to parse YAML: %v", err)
	}
	if defaults.Model != "" {
		t.Errorf("expected empty model when not specified, got %q", defaults.Model)
	}
}

func TestOrgWorkspace_Model_Override(t *testing.T) {
	raw := `
name: Worker
role: coding
model: groq:llama-3.3-70b-versatile
`
	var ws OrgWorkspace
	if err := yaml.Unmarshal([]byte(raw), &ws); err != nil {
		t.Fatalf("failed to parse YAML: %v", err)
	}
	if ws.Model != "groq:llama-3.3-70b-versatile" {
		t.Errorf("expected model 'groq:llama-3.3-70b-versatile', got %q", ws.Model)
	}
}

// ==================== Model Fallback Edge Cases ====================
// These test the cascading fallback: ws.Model → defaults.Model → runtime-specific default
// They verify behavior without a database since createWorkspaceTree requires sqlmock.
// The struct-level tests + ensureDefaultConfig tests cover the full data flow.

func TestOrgDefaults_Model_WorkspaceOverridesDefault(t *testing.T) {
	// When both ws and defaults have a model, ws.Model takes precedence.
	// This verifies the YAML struct correctly captures both values.
	defaultsRaw := `
runtime: deepagents
model: google_genai:gemini-2.5-flash
`
	wsRaw := `
name: Worker
model: groq:llama-3.3-70b-versatile
`
	var defaults OrgDefaults
	if err := yaml.Unmarshal([]byte(defaultsRaw), &defaults); err != nil {
		t.Fatalf("failed to parse defaults: %v", err)
	}
	var ws OrgWorkspace
	if err := yaml.Unmarshal([]byte(wsRaw), &ws); err != nil {
		t.Fatalf("failed to parse workspace: %v", err)
	}

	// Simulate the fallback logic from createWorkspaceTree
	model := ws.Model
	if model == "" {
		model = defaults.Model
	}
	if model != "groq:llama-3.3-70b-versatile" {
		t.Errorf("ws.Model should override defaults.Model, got %q", model)
	}
}

func TestOrgDefaults_Model_FallbackClaudeCode(t *testing.T) {
	// When both ws and defaults models are empty, claude-code runtime → "sonnet"
	var defaults OrgDefaults
	var ws OrgWorkspace

	runtime := "claude-code"
	model := ws.Model
	if model == "" {
		model = defaults.Model
	}
	if model == "" {
		if runtime == "claude-code" {
			model = "sonnet"
		} else {
			model = "anthropic:claude-sonnet-4-6"
		}
	}
	if model != "sonnet" {
		t.Errorf("claude-code with empty model should get 'sonnet', got %q", model)
	}
}

func TestOrgDefaults_Model_FallbackDeepAgents(t *testing.T) {
	// When both ws and defaults models are empty, deepagents runtime → anthropic default
	var defaults OrgDefaults
	var ws OrgWorkspace

	runtime := "deepagents"
	model := ws.Model
	if model == "" {
		model = defaults.Model
	}
	if model == "" {
		if runtime == "claude-code" {
			model = "sonnet"
		} else {
			model = "anthropic:claude-sonnet-4-6"
		}
	}
	if model != "anthropic:claude-sonnet-4-6" {
		t.Errorf("deepagents with empty model should get 'anthropic:claude-sonnet-4-6', got %q", model)
	}
}

func TestOrgDefaults_Model_FallbackLangGraph(t *testing.T) {
	// Langgraph also gets the default anthropic model
	model := ""
	runtime := "langgraph"
	if model == "" {
		if runtime == "claude-code" {
			model = "sonnet"
		} else {
			model = "anthropic:claude-sonnet-4-6"
		}
	}
	if model != "anthropic:claude-sonnet-4-6" {
		t.Errorf("langgraph with empty model should get 'anthropic:claude-sonnet-4-6', got %q", model)
	}
}

func TestOrgDefaults_Model_DefaultsModelUsedWhenWsEmpty(t *testing.T) {
	// ws.Model empty → falls back to defaults.Model
	defaultsRaw := `
model: cerebras:llama3.1-8b
`
	var defaults OrgDefaults
	if err := yaml.Unmarshal([]byte(defaultsRaw), &defaults); err != nil {
		t.Fatalf("failed to parse defaults: %v", err)
	}

	model := "" // ws.Model is empty
	if model == "" {
		model = defaults.Model
	}
	if model != "cerebras:llama3.1-8b" {
		t.Errorf("expected defaults.Model 'cerebras:llama3.1-8b', got %q", model)
	}
}

func TestInitialPrompt_SpecialChars(t *testing.T) {
	// Ensure YAML-special characters in prompt don't break parsing
	initialPrompt := `Run: git clone https://${GITHUB_TOKEN}@github.com/${GITHUB_REPO}.git
Check "config.yaml" for settings
Use key: value pairs`

	configYAML := "name: Test\n"
	trimmed := strings.TrimSpace(initialPrompt)
	lines := strings.Split(trimmed, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	indented := strings.Join(lines, "\n  ")
	result := configYAML + "initial_prompt: |\n  " + indented + "\n"

	var parsed map[string]interface{}
	if err := yaml.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("generated YAML with special chars is invalid: %v\n---\n%s", err, result)
	}

	prompt := parsed["initial_prompt"].(string)
	if !strings.Contains(prompt, "${GITHUB_TOKEN}") {
		t.Error("expected prompt to preserve ${GITHUB_TOKEN}")
	}
	if !strings.Contains(prompt, `"config.yaml"`) {
		t.Error("expected prompt to preserve quoted strings")
	}
}

// ==================== OrgChannel + env expansion tests ====================

func TestOrgChannel_YAMLParsing(t *testing.T) {
	raw := `
name: PM
files_dir: pm
channels:
  - type: telegram
    config:
      bot_token: ${TELEGRAM_BOT_TOKEN}
      chat_id: ${TELEGRAM_CHAT_ID}
    allowed_users: ["123", "456"]
    enabled: true
`
	var ws OrgWorkspace
	if err := yaml.Unmarshal([]byte(raw), &ws); err != nil {
		t.Fatalf("YAML parse failed: %v", err)
	}
	if len(ws.Channels) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(ws.Channels))
	}
	ch := ws.Channels[0]
	if ch.Type != "telegram" {
		t.Errorf("expected type telegram, got %q", ch.Type)
	}
	if ch.Config["bot_token"] != "${TELEGRAM_BOT_TOKEN}" {
		t.Errorf("expected raw ${VAR}, got %q", ch.Config["bot_token"])
	}
	if len(ch.AllowedUsers) != 2 {
		t.Errorf("expected 2 allowed users, got %d", len(ch.AllowedUsers))
	}
	if ch.Enabled == nil || !*ch.Enabled {
		t.Error("expected enabled=true")
	}
}

func TestExpandWithEnv_FromMap(t *testing.T) {
	env := map[string]string{"FOO": "bar", "TOKEN": "abc123"}
	got := expandWithEnv("${FOO}-${TOKEN}", env)
	if got != "bar-abc123" {
		t.Errorf("expected 'bar-abc123', got %q", got)
	}
}

func TestExpandWithEnv_FromProcessEnv(t *testing.T) {
	t.Setenv("EXPAND_TEST_VAR", "process-value")
	got := expandWithEnv("${EXPAND_TEST_VAR}", map[string]string{})
	if got != "process-value" {
		t.Errorf("expected 'process-value', got %q", got)
	}
}

func TestExpandWithEnv_MapOverridesProcess(t *testing.T) {
	t.Setenv("OVERRIDE_VAR", "process")
	got := expandWithEnv("${OVERRIDE_VAR}", map[string]string{"OVERRIDE_VAR": "map"})
	if got != "map" {
		t.Errorf("map should override process env, got %q", got)
	}
}

func TestExpandWithEnv_UnsetVar(t *testing.T) {
	got := expandWithEnv("${DEFINITELY_NOT_SET_XYZ}", map[string]string{})
	if got != "" {
		t.Errorf("unset var should expand to empty, got %q", got)
	}
}

func TestHasUnresolvedVarRef_NoVars(t *testing.T) {
	if hasUnresolvedVarRef("plain text", "plain text") {
		t.Error("plain text should not be flagged")
	}
}

func TestHasUnresolvedVarRef_LiteralDollar(t *testing.T) {
	// "$5" is a literal price, not a var ref — should NOT be flagged
	if hasUnresolvedVarRef("price: $5", "price: $5") {
		t.Error("literal $5 should not be flagged as unresolved")
	}
}

func TestHasUnresolvedVarRef_Resolved(t *testing.T) {
	// Original had ${VAR}, expanded to "value" — fully resolved
	if hasUnresolvedVarRef("${VAR}", "value") {
		t.Error("fully resolved var should not be flagged")
	}
}

func TestHasUnresolvedVarRef_Unresolved(t *testing.T) {
	// Original had ${VAR}, expanded to "" — unresolved
	if !hasUnresolvedVarRef("${VAR}", "") {
		t.Error("unresolved var should be flagged")
	}
}

func TestHasUnresolvedVarRef_DollarVarSyntax(t *testing.T) {
	// $VAR syntax (no braces) — also a real ref
	if !hasUnresolvedVarRef("$MISSING_VAR", "") {
		t.Error("$VAR syntax should be detected as ref when unresolved")
	}
}
