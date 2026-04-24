package handlers

import (
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/scheduler"
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
			model = "anthropic:claude-opus-4-7"
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
			model = "anthropic:claude-opus-4-7"
		}
	}
	if model != "anthropic:claude-opus-4-7" {
		t.Errorf("deepagents with empty model should get 'anthropic:claude-opus-4-7', got %q", model)
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
			model = "anthropic:claude-opus-4-7"
		}
	}
	if model != "anthropic:claude-opus-4-7" {
		t.Errorf("langgraph with empty model should get 'anthropic:claude-opus-4-7', got %q", model)
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

func eqStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestPlugins_UnionWithDefaults(t *testing.T) {
	got := mergePlugins([]string{"a", "b"}, []string{"c"})
	want := []string{"a", "b", "c"}
	if !eqStringSlice(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestPlugins_DedupesDuplicates(t *testing.T) {
	got := mergePlugins([]string{"a", "b"}, []string{"b", "c"})
	want := []string{"a", "b", "c"}
	if !eqStringSlice(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestPlugins_OptOutWithBang(t *testing.T) {
	got := mergePlugins([]string{"a", "b", "c"}, []string{"!b", "d"})
	want := []string{"a", "c", "d"}
	if !eqStringSlice(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestPlugins_OptOutWithDash(t *testing.T) {
	got := mergePlugins([]string{"a", "b"}, []string{"-a"})
	want := []string{"b"}
	if !eqStringSlice(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// ==================== category_routing (issue #51) ====================

func TestCategoryRouting_ParsedFromOrgYaml(t *testing.T) {
	raw := `
name: Test Org
defaults:
  runtime: claude-code
  category_routing:
    security: [Backend Engineer, DevOps Engineer]
    ui: [Frontend Engineer]
    infra: [DevOps Engineer]
workspaces:
  - name: PM
    role: Project Manager
    category_routing:
      performance: [Backend Engineer]
`
	var tmpl OrgTemplate
	if err := yaml.Unmarshal([]byte(raw), &tmpl); err != nil {
		t.Fatalf("yaml parse failed: %v", err)
	}
	if got := tmpl.Defaults.CategoryRouting["security"]; len(got) != 2 || got[0] != "Backend Engineer" {
		t.Errorf("defaults.category_routing.security wrong: %v", got)
	}
	if got := tmpl.Defaults.CategoryRouting["ui"]; len(got) != 1 || got[0] != "Frontend Engineer" {
		t.Errorf("defaults.category_routing.ui wrong: %v", got)
	}
	if len(tmpl.Workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(tmpl.Workspaces))
	}
	if got := tmpl.Workspaces[0].CategoryRouting["performance"]; len(got) != 1 || got[0] != "Backend Engineer" {
		t.Errorf("ws.category_routing.performance wrong: %v", got)
	}
}

func TestCategoryRouting_UnionWithDefaults(t *testing.T) {
	defaults := map[string][]string{
		"security": {"Backend Engineer", "DevOps"},
		"ui":       {"Frontend Engineer"},
		"infra":    {"DevOps"},
	}
	ws := map[string][]string{
		"performance": {"Backend Engineer"}, // new key, added
		"ui":          {"Designer"},          // override-replace existing key
		"infra":       {},                    // empty → drop
	}
	got := mergeCategoryRouting(defaults, ws)

	if v := got["security"]; len(v) != 2 || v[0] != "Backend Engineer" || v[1] != "DevOps" {
		t.Errorf("security should be inherited from defaults unchanged, got %v", v)
	}
	if v := got["ui"]; len(v) != 1 || v[0] != "Designer" {
		t.Errorf("ui should be replaced by ws value, got %v", v)
	}
	if _, ok := got["infra"]; ok {
		t.Errorf("infra should be dropped (empty ws list), got %v", got["infra"])
	}
	if v := got["performance"]; len(v) != 1 || v[0] != "Backend Engineer" {
		t.Errorf("performance should be added from ws, got %v", v)
	}
}

func TestCategoryRouting_RenderedIntoWorkspaceConfig(t *testing.T) {
	routing := map[string][]string{
		"security": {"Backend Engineer", "DevOps"},
		"ui":       {"Frontend Engineer"},
	}
	block, err := renderCategoryRoutingYAML(routing)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if block == "" {
		t.Fatal("expected non-empty block")
	}
	// Must parse as valid YAML when concatenated with a base config
	combined := "name: Test\nruntime: claude-code\n" + block
	var parsed map[string]interface{}
	if err := yaml.Unmarshal([]byte(combined), &parsed); err != nil {
		t.Fatalf("rendered YAML is invalid: %v\n---\n%s", err, combined)
	}
	cr, ok := parsed["category_routing"].(map[string]interface{})
	if !ok {
		t.Fatalf("category_routing not a map in parsed config: %T", parsed["category_routing"])
	}
	sec, ok := cr["security"].([]interface{})
	if !ok || len(sec) != 2 {
		t.Fatalf("security routing wrong shape: %v", cr["security"])
	}
	if sec[0] != "Backend Engineer" || sec[1] != "DevOps" {
		t.Errorf("security roles wrong: %v", sec)
	}
	ui, ok := cr["ui"].([]interface{})
	if !ok || len(ui) != 1 || ui[0] != "Frontend Engineer" {
		t.Errorf("ui roles wrong: %v", cr["ui"])
	}
	// Output should be deterministic (keys sorted) — security < ui
	if strings.Index(block, "security") > strings.Index(block, "ui") {
		t.Errorf("expected sorted keys (security before ui), got:\n%s", block)
	}
}

// YAML-reserved characters in role names must be escaped by the YAML library.
// Regression guard for the earlier hand-rolled JSON-as-YAML implementation.
func TestCategoryRouting_EscapesYAMLSpecials(t *testing.T) {
	routing := map[string][]string{
		"security": {"Role: with colon", `Role "with quotes"`, "Role\nwith newline"},
	}
	block, err := renderCategoryRoutingYAML(routing)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	var parsed map[string]interface{}
	if err := yaml.Unmarshal([]byte(block), &parsed); err != nil {
		t.Fatalf("rendered YAML is invalid for special chars: %v\n---\n%s", err, block)
	}
	cr := parsed["category_routing"].(map[string]interface{})
	roles := cr["security"].([]interface{})
	if len(roles) != 3 || roles[0] != "Role: with colon" {
		t.Errorf("special-char roles did not round-trip: %v", roles)
	}
}

// appendYAMLBlock must guarantee a newline boundary between existing buffer and
// the new block so downstream parsers see two separate top-level keys.
func TestAppendYAMLBlock_NewlineGuard(t *testing.T) {
	cases := []struct {
		name     string
		existing string
		block    string
	}{
		{"existing ends without newline", "name: foo", "category_routing:\n  a: [b]\n"},
		{"existing ends with newline", "name: foo\n", "category_routing:\n  a: [b]\n"},
		{"empty existing", "", "category_routing:\n  a: [b]\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := appendYAMLBlock([]byte(tc.existing), tc.block)
			var parsed map[string]interface{}
			if err := yaml.Unmarshal(got, &parsed); err != nil {
				t.Fatalf("appended YAML invalid: %v\n---\n%s", err, string(got))
			}
			if _, ok := parsed["category_routing"]; !ok {
				t.Errorf("expected top-level category_routing key, got: %v", parsed)
			}
		})
	}
}

func TestCategoryRouting_EmptyRendersNothing(t *testing.T) {
	got, err := renderCategoryRoutingYAML(nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty render for nil routing, got %q", got)
	}
	got, err = renderCategoryRoutingYAML(map[string][]string{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty render for empty map, got %q", got)
	}
}

func TestPlugins_BackwardCompat(t *testing.T) {
	// Re-listing defaults in per-workspace plugins still yields the same list
	// (dedupe keeps behavior stable for existing org.yaml files).
	got := mergePlugins([]string{"a", "b"}, []string{"a", "b", "c"})
	want := []string{"a", "b", "c"}
	if !eqStringSlice(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// ── TestOrgImport_ScheduleComputeError (#722 Bug 2) ───────────────────────────
//
// The org importer previously used `nextRun, _ := scheduler.ComputeNextRun(...)`,
// discarding the error and passing time.Time{} (zero value) to the INSERT.
// After fix #722 it surfaces the error and skips the INSERT via `continue`.
//
// This test verifies that the inputs an org.yaml schedule can supply (bad cron
// expression, invalid timezone) DO cause ComputeNextRun to return a non-nil
// error — confirming that the fix is meaningful and the skip path is reachable.

func TestOrgImport_ScheduleComputeError(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name     string
		cronExpr string
		tz       string
	}{
		{
			name:     "invalid cron expression",
			cronExpr: "not-a-cron-expr",
			tz:       "UTC",
		},
		{
			name:     "invalid timezone",
			cronExpr: "0 9 * * 1",
			tz:       "Not/A/Valid/Timezone",
		},
		{
			name:     "both invalid",
			cronExpr: "every monday",
			tz:       "Moon/Far_Side",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := scheduler.ComputeNextRun(tc.cronExpr, tc.tz, now)
			if err == nil {
				t.Errorf("ComputeNextRun(%q, %q) returned nil error — "+
					"org importer would silently insert zero next_run_at; #722 fix requires non-nil",
					tc.cronExpr, tc.tz)
			}
		})
	}
}

// ============================================================================
// Org env-preflight aggregation (collectOrgEnv)
// ============================================================================

func TestCollectOrgEnv_UnionAcrossLevels(t *testing.T) {
	tmpl := &OrgTemplate{
		RequiredEnv:    []string{"ANTHROPIC_API_KEY"},
		RecommendedEnv: []string{"SLACK_WEBHOOK_URL"},
		Workspaces: []OrgWorkspace{
			{
				Name:         "Root",
				RequiredEnv:  []string{"GITHUB_TOKEN"},
				Children: []OrgWorkspace{
					{
						Name:           "Leaf",
						RequiredEnv:    []string{"OPENROUTER_API_KEY"},
						RecommendedEnv: []string{"DISCORD_WEBHOOK_URL"},
					},
				},
			},
		},
	}
	req, rec := collectOrgEnv(tmpl)
	// Required is the union of top-level + root + leaf.
	wantReq := []string{"ANTHROPIC_API_KEY", "GITHUB_TOKEN", "OPENROUTER_API_KEY"}
	if !stringSlicesEqual(req, wantReq) {
		t.Errorf("required mismatch: got %v, want %v", req, wantReq)
	}
	wantRec := []string{"DISCORD_WEBHOOK_URL", "SLACK_WEBHOOK_URL"}
	if !stringSlicesEqual(rec, wantRec) {
		t.Errorf("recommended mismatch: got %v, want %v", rec, wantRec)
	}
}

func TestCollectOrgEnv_RequiredWinsOverRecommended(t *testing.T) {
	// Same key declared at one layer as recommended and another as
	// required MUST surface only on the required side — a required
	// declaration is strictly stricter than a recommended one, and
	// listing it in both tiers would confuse the preflight modal.
	tmpl := &OrgTemplate{
		RecommendedEnv: []string{"API_KEY"},
		Workspaces: []OrgWorkspace{
			{Name: "X", RequiredEnv: []string{"API_KEY"}},
		},
	}
	req, rec := collectOrgEnv(tmpl)
	if len(req) != 1 || req[0] != "API_KEY" {
		t.Errorf("required should contain API_KEY, got %v", req)
	}
	for _, k := range rec {
		if k == "API_KEY" {
			t.Errorf("API_KEY must not appear in recommended once required elsewhere")
		}
	}
}

func TestCollectOrgEnv_Dedup(t *testing.T) {
	// Same key declared twice at different levels should appear once.
	tmpl := &OrgTemplate{
		RequiredEnv: []string{"K", "K"},
		Workspaces: []OrgWorkspace{
			{Name: "A", RequiredEnv: []string{"K"}},
			{Name: "B", RequiredEnv: []string{"K"}, Children: []OrgWorkspace{
				{Name: "C", RequiredEnv: []string{"K"}},
			}},
		},
	}
	req, _ := collectOrgEnv(tmpl)
	if len(req) != 1 || req[0] != "K" {
		t.Errorf("dedup failed: got %v, want [K]", req)
	}
}

func TestCollectOrgEnv_Empty(t *testing.T) {
	tmpl := &OrgTemplate{}
	req, rec := collectOrgEnv(tmpl)
	if len(req) != 0 || len(rec) != 0 {
		t.Errorf("empty template should produce empty slices, got req=%v rec=%v", req, rec)
	}
}

// stringSlicesEqual checks ordered equality — collectOrgEnv sorts its
// output so callers can do deterministic comparisons.
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestCollectOrgEnv_RequiredWinsOnSameStruct(t *testing.T) {
	// The same key declared required AND recommended on the SAME
	// workspace node (rare but legal to parse) must still dedup
	// correctly and end up required-only.
	tmpl := &OrgTemplate{
		Workspaces: []OrgWorkspace{
			{
				Name:           "X",
				RequiredEnv:    []string{"API_KEY"},
				RecommendedEnv: []string{"API_KEY"},
			},
		},
	}
	req, rec := collectOrgEnv(tmpl)
	if len(req) != 1 || req[0] != "API_KEY" {
		t.Errorf("required should contain API_KEY once, got %v", req)
	}
	for _, k := range rec {
		if k == "API_KEY" {
			t.Errorf("API_KEY must not appear in recommended when also required on same struct")
		}
	}
}

func TestCollectOrgEnv_RejectsInvalidNames(t *testing.T) {
	// Names failing envVarNamePattern (lowercase, traversal, whitespace,
	// shell metachars) must be dropped silently — the log line is not
	// asserted here; the output slice assertion is enough to prove the
	// filter fires.
	tmpl := &OrgTemplate{
		RequiredEnv: []string{
			"VALID_ONE",
			"lowercase_bad",
			"../../etc/passwd",
			"name with spaces",
			"WITH-DASH",
			"'; DROP TABLE users;--",
			"",
			"A", // single char — still valid per regex
		},
	}
	req, _ := collectOrgEnv(tmpl)
	if !stringSlicesEqual(req, []string{"A", "VALID_ONE"}) {
		t.Errorf("expected only valid names, got %v", req)
	}
}
