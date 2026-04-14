package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/channels"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/crypto"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/events"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/models"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/provisioner"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/scheduler"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// OrgHandler manages org template import/export.
// workspaceCreatePacingMs is the brief delay between sibling workspace creations
// during org import. Prevents overwhelming Docker when creating many containers.
const workspaceCreatePacingMs = 50

type OrgHandler struct {
	workspace   *WorkspaceHandler
	broadcaster *events.Broadcaster
	provisioner *provisioner.Provisioner
	channelMgr  *channels.Manager
	configsDir  string
	orgDir      string // path to org-templates/
}

func NewOrgHandler(wh *WorkspaceHandler, b *events.Broadcaster, p *provisioner.Provisioner, channelMgr *channels.Manager, configsDir, orgDir string) *OrgHandler {
	return &OrgHandler{
		workspace:   wh,
		broadcaster: b,
		provisioner: p,
		channelMgr:  channelMgr,
		configsDir:  configsDir,
		orgDir:      orgDir,
	}
}

// OrgTemplate is the YAML structure for an org hierarchy.
type OrgTemplate struct {
	Name        string            `yaml:"name" json:"name"`
	Description string            `yaml:"description" json:"description"`
	Defaults    OrgDefaults       `yaml:"defaults" json:"defaults"`
	Workspaces  []OrgWorkspace    `yaml:"workspaces" json:"workspaces"`
}

type OrgDefaults struct {
	Runtime       string              `yaml:"runtime" json:"runtime"`
	Tier          int                 `yaml:"tier" json:"tier"`
	Model         string              `yaml:"model" json:"model"`
	Plugins       []string            `yaml:"plugins" json:"plugins"`
	InitialPrompt string              `yaml:"initial_prompt" json:"initial_prompt"`
	// CategoryRouting maps issue/audit category → list of target roles.
	// Per-workspace blocks UNION + override per-key with these defaults.
	// Rendered into each workspace's config.yaml so agent prompts can read it
	// generically (no hardcoded role names in prompts). See issue #51.
	CategoryRouting map[string][]string `yaml:"category_routing" json:"category_routing"`
}

type OrgSchedule struct {
	Name     string `yaml:"name" json:"name"`
	CronExpr string `yaml:"cron_expr" json:"cron_expr"`
	Timezone string `yaml:"timezone" json:"timezone"`
	Prompt   string `yaml:"prompt" json:"prompt"`
	Enabled  *bool  `yaml:"enabled" json:"enabled"`
}

// OrgChannel defines a social channel (Telegram, Slack, etc.) to auto-link
// when the workspace is created. Config values may reference env vars
// using ${VAR_NAME} syntax — useful for keeping bot tokens out of YAML.
type OrgChannel struct {
	Type         string            `yaml:"type" json:"type"`
	Config       map[string]string `yaml:"config" json:"config"`
	AllowedUsers []string          `yaml:"allowed_users" json:"allowed_users"`
	Enabled      *bool             `yaml:"enabled" json:"enabled"`
}

type OrgWorkspace struct {
	Name          string         `yaml:"name" json:"name"`
	Role          string         `yaml:"role" json:"role"`
	Runtime       string         `yaml:"runtime" json:"runtime"`
	Tier          int            `yaml:"tier" json:"tier"`
	Template      string         `yaml:"template" json:"template"`
	FilesDir      string         `yaml:"files_dir" json:"files_dir"`
	SystemPrompt  string         `yaml:"system_prompt" json:"system_prompt"`
	Model         string         `yaml:"model" json:"model"`
	WorkspaceDir    string `yaml:"workspace_dir" json:"workspace_dir"`
	WorkspaceAccess string `yaml:"workspace_access" json:"workspace_access"` // #65: "none" (default), "read_only", "read_write"
	Plugins       []string       `yaml:"plugins" json:"plugins"`
	InitialPrompt string         `yaml:"initial_prompt" json:"initial_prompt"`
	// CategoryRouting extends/overrides defaults.category_routing per-workspace.
	// Merge semantics: workspace keys replace defaults' value for the same key
	// (empty list drops the category entirely); new keys are added. See
	// mergeCategoryRouting.
	CategoryRouting map[string][]string `yaml:"category_routing" json:"category_routing"`
	Schedules     []OrgSchedule  `yaml:"schedules" json:"schedules"`
	Channels      []OrgChannel   `yaml:"channels" json:"channels"`
	External      bool           `yaml:"external" json:"external"`
	URL           string         `yaml:"url" json:"url"`
	Canvas        struct {
		X float64 `yaml:"x" json:"x"`
		Y float64 `yaml:"y" json:"y"`
	} `yaml:"canvas" json:"canvas"`
	Children      []OrgWorkspace `yaml:"children" json:"children"`
}

// ListTemplates handles GET /org/templates — lists available org templates.
func (h *OrgHandler) ListTemplates(c *gin.Context) {
	templates := []map[string]interface{}{}

	entries, err := os.ReadDir(h.orgDir)
	if err != nil {
		c.JSON(http.StatusOK, templates)
		return
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// Look for org.yaml inside the directory
		orgFile := filepath.Join(h.orgDir, e.Name(), "org.yaml")
		data, err := os.ReadFile(orgFile)
		if err != nil {
			// Try org.yml
			orgFile = filepath.Join(h.orgDir, e.Name(), "org.yml")
			data, err = os.ReadFile(orgFile)
			if err != nil {
				continue
			}
		}
		var tmpl OrgTemplate
		if err := yaml.Unmarshal(data, &tmpl); err != nil {
			continue
		}
		count := countWorkspaces(tmpl.Workspaces)
		templates = append(templates, map[string]interface{}{
			"dir":         e.Name(),
			"name":        tmpl.Name,
			"description": tmpl.Description,
			"workspaces":  count,
		})
	}

	c.JSON(http.StatusOK, templates)
}

// Import handles POST /org/import — creates an entire org from a template.
func (h *OrgHandler) Import(c *gin.Context) {
	var body struct {
		Dir      string      `json:"dir"`      // org template directory name
		Template OrgTemplate `json:"template"` // or inline template
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var tmpl OrgTemplate
	var orgBaseDir string // base directory for files_dir resolution

	if body.Dir != "" {
		orgBaseDir = filepath.Join(h.orgDir, body.Dir)
		orgFile := filepath.Join(orgBaseDir, "org.yaml")
		data, err := os.ReadFile(orgFile)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("org template not found: %s", body.Dir)})
			return
		}
		if err := yaml.Unmarshal(data, &tmpl); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid YAML: %v", err)})
			return
		}
	} else if body.Template.Name != "" {
		tmpl = body.Template
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provide 'dir' or 'template'"})
		return
	}

	results := []map[string]interface{}{}
	var createErr error

	// Recursively create workspaces
	for _, ws := range tmpl.Workspaces {
		if err := h.createWorkspaceTree(ws, nil, tmpl.Defaults, orgBaseDir, &results); err != nil {
			createErr = err
			break
		}
	}

	// Hot-reload channel manager once after all channels are inserted
	// (instead of per-workspace, avoiding N redundant DB queries + diffs).
	if h.channelMgr != nil {
		hasAnyChannels := false
		for _, r := range results {
			if _, ok := r["channels"]; ok {
				hasAnyChannels = true
				break
			}
		}
		if hasAnyChannels {
			h.channelMgr.Reload(context.Background())
		}
	}

	status := http.StatusCreated
	resp := gin.H{
		"org":        tmpl.Name,
		"workspaces": results,
		"count":      len(results),
	}
	if createErr != nil {
		status = http.StatusMultiStatus
		resp["error"] = createErr.Error()
	}

	log.Printf("Org import: %s — %d workspaces created", tmpl.Name, len(results))
	c.JSON(status, resp)
}

// createWorkspaceTree recursively creates a workspace and its children.
func (h *OrgHandler) createWorkspaceTree(ws OrgWorkspace, parentID *string, defaults OrgDefaults, orgBaseDir string, results *[]map[string]interface{}) error {
	// Apply defaults
	runtime := ws.Runtime
	if runtime == "" {
		runtime = defaults.Runtime
	}
	if runtime == "" {
		runtime = "langgraph"
	}
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
	tier := ws.Tier
	if tier == 0 {
		tier = defaults.Tier
	}
	if tier == 0 {
		tier = 2
	}

	id := uuid.New().String()
	awarenessNS := workspaceAwarenessNamespace(id)

	var role interface{}
	if ws.Role != "" {
		role = ws.Role
	}

	// Validate and convert workspace_dir to NULL if empty
	var workspaceDir interface{}
	if ws.WorkspaceDir != "" {
		if err := validateWorkspaceDir(ws.WorkspaceDir); err != nil {
			return fmt.Errorf("workspace %s: %w", ws.Name, err)
		}
		workspaceDir = ws.WorkspaceDir
	}

	// #65: validate workspace_access (defaults to "none" when empty).
	workspaceAccess := ws.WorkspaceAccess
	if workspaceAccess == "" {
		workspaceAccess = provisioner.WorkspaceAccessNone
	}
	if err := provisioner.ValidateWorkspaceAccess(workspaceAccess, ws.WorkspaceDir); err != nil {
		return fmt.Errorf("workspace %s: %w", ws.Name, err)
	}

	ctx := context.Background()

	// Insert workspace
	_, err := db.DB.ExecContext(ctx, `
		INSERT INTO workspaces (id, name, role, tier, runtime, awareness_namespace, status, parent_id, workspace_dir, workspace_access)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, id, ws.Name, role, tier, runtime, awarenessNS, "provisioning", parentID, workspaceDir, workspaceAccess)
	if err != nil {
		log.Printf("Org import: failed to create %s: %v", ws.Name, err)
		return fmt.Errorf("failed to create %s: %w", ws.Name, err)
	}

	// Canvas layout with coordinates from YAML
	if _, err := db.DB.ExecContext(ctx, `INSERT INTO canvas_layouts (workspace_id, x, y) VALUES ($1, $2, $3)`, id, ws.Canvas.X, ws.Canvas.Y); err != nil {
		log.Printf("Org import: canvas layout insert failed for %s: %v", ws.Name, err)
	}

	// Broadcast
	h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_PROVISIONING", id, map[string]interface{}{
		"name": ws.Name, "tier": tier,
	})

	// Handle external workspaces
	if ws.External {
		if _, err := db.DB.ExecContext(ctx, `UPDATE workspaces SET status = 'online', url = $1 WHERE id = $2`, ws.URL, id); err != nil {
			log.Printf("Org import: external workspace status update failed for %s: %v", ws.Name, err)
		}
		h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_ONLINE", id, map[string]interface{}{
			"name": ws.Name, "external": true,
		})
	} else if h.provisioner != nil {
		// Provision container
		payload := models.CreateWorkspacePayload{
			Name: ws.Name, Tier: tier, Runtime: runtime, Model: model,
			WorkspaceDir:    ws.WorkspaceDir,
			WorkspaceAccess: workspaceAccess,
		}
		templatePath := ""
		if ws.Template != "" {
			tp := filepath.Join(h.configsDir, ws.Template)
			if _, err := os.Stat(tp); err == nil {
				templatePath = tp
			}
		}
		if templatePath == "" {
			runtimeDefault := filepath.Join(h.configsDir, runtime+"-default")
			if _, err := os.Stat(runtimeDefault); err == nil {
				templatePath = runtimeDefault
			}
		}

		// Always generate default config.yaml (runtime, model, tier, etc.)
		configFiles := h.workspace.ensureDefaultConfig(id, payload)

		// Copy files_dir contents on top (system-prompt.md, CLAUDE.md, skills/, etc.)
		// Uses templatePath for CopyTemplateToContainer — runs AFTER configFiles are written
		if ws.FilesDir != "" && orgBaseDir != "" {
			filesPath := filepath.Join(orgBaseDir, ws.FilesDir)
			if info, err := os.Stat(filesPath); err == nil && info.IsDir() {
				templatePath = filesPath
			}
		}

		// Pre-install plugins: copy from registry into configFiles as plugins/<name>/*.
		// Per-workspace plugins UNION with defaults.plugins (issue #68).
		// A leading "!" or "-" on a per-workspace entry opts that plugin out.
		plugins := mergePlugins(defaults.Plugins, ws.Plugins)
		if len(plugins) > 0 {
			if configFiles == nil {
				configFiles = map[string][]byte{}
			}
			pluginsBase, _ := filepath.Abs(filepath.Join(h.configsDir, "..", "plugins"))
			for _, pluginName := range plugins {
				pluginSrc := filepath.Join(pluginsBase, pluginName)
				if info, err := os.Stat(pluginSrc); err != nil || !info.IsDir() {
					log.Printf("Org import: plugin %s not found at %s, skipping", pluginName, pluginSrc)
					continue
				}
				filepath.Walk(pluginSrc, func(path string, info os.FileInfo, err error) error {
					if err != nil || info.IsDir() {
						return nil
					}
					rel, _ := filepath.Rel(pluginSrc, path)
					data, readErr := os.ReadFile(path)
					if readErr == nil {
						configFiles["plugins/"+pluginName+"/"+rel] = data
					}
					return nil
				})
			}
		}

		// Render category_routing into config.yaml so the agent can read its routing
		// table at runtime without hardcoded role names in prompts (issue #51).
		// Per-workspace keys replace defaults per-key (empty list drops the key);
		// see mergeCategoryRouting for exact semantics.
		routing := mergeCategoryRouting(defaults.CategoryRouting, ws.CategoryRouting)
		if len(routing) > 0 {
			if configFiles == nil {
				configFiles = map[string][]byte{}
			}
			block, err := renderCategoryRoutingYAML(routing)
			if err != nil {
				log.Printf("Org import: failed to render category_routing for %s: %v", ws.Name, err)
			} else {
				configFiles["config.yaml"] = appendYAMLBlock(configFiles["config.yaml"], block)
			}
		}

		// Inject initial_prompt into config.yaml (workspace-level overrides default)
		initialPrompt := ws.InitialPrompt
		if initialPrompt == "" {
			initialPrompt = defaults.InitialPrompt
		}
		if initialPrompt != "" {
			if configFiles == nil {
				configFiles = map[string][]byte{}
			}
			// Append initial_prompt to config.yaml using YAML block scalar.
			// Trim each line to avoid trailing whitespace issues.
			trimmed := strings.TrimSpace(initialPrompt)
			lines := strings.Split(trimmed, "\n")
			for i, line := range lines {
				lines[i] = strings.TrimRight(line, " \t")
			}
			indented := strings.Join(lines, "\n  ")
			configFiles["config.yaml"] = appendYAMLBlock(configFiles["config.yaml"], fmt.Sprintf("initial_prompt: |\n  %s\n", indented))
			log.Printf("Org import: injected initial_prompt (%d chars) into config.yaml for %s", len(trimmed), ws.Name)
		}

		// Inline system_prompt (only if no files_dir provides one)
		if ws.SystemPrompt != "" {
			if configFiles == nil {
				configFiles = map[string][]byte{}
			}
			configFiles["system-prompt.md"] = []byte(ws.SystemPrompt)
		}

		// Inject secrets from .env files as workspace secrets.
		// Resolution: workspace .env → org root .env (workspace overrides org root).
		// Each line: KEY=VALUE → stored as encrypted workspace secret.
		envVars := map[string]string{}
		if orgBaseDir != "" {
			// 1. Org root .env (shared defaults)
			parseEnvFile(filepath.Join(orgBaseDir, ".env"), envVars)
			// 2. Workspace-specific .env (overrides)
			if ws.FilesDir != "" {
				parseEnvFile(filepath.Join(orgBaseDir, ws.FilesDir, ".env"), envVars)
			}
		}
		// Store as workspace secrets via DB (encrypted if key is set, raw otherwise)
		for key, value := range envVars {
			var encrypted []byte
			if crypto.IsEnabled() {
				var err error
				encrypted, err = crypto.Encrypt([]byte(value))
				if err != nil {
					log.Printf("Org import: failed to encrypt secret %s for %s: %v", key, ws.Name, err)
					continue
				}
			} else {
				encrypted = []byte(value) // store raw when encryption disabled
			}
			if _, err := db.DB.ExecContext(ctx, `
				INSERT INTO workspace_secrets (workspace_id, key, encrypted_value)
				VALUES ($1, $2, $3)
				ON CONFLICT (workspace_id, key) DO UPDATE SET encrypted_value = $3, updated_at = now()
			`, id, key, encrypted); err != nil {
				log.Printf("Org import: failed to insert secret %s for %s: %v", key, ws.Name, err)
			}
		}

		go h.workspace.provisionWorkspace(id, templatePath, configFiles, payload)
	}

	// Insert schedules if defined
	for _, sched := range ws.Schedules {
		tz := sched.Timezone
		if tz == "" {
			tz = "UTC"
		}
		enabled := true
		if sched.Enabled != nil {
			enabled = *sched.Enabled
		}
		nextRun, _ := scheduler.ComputeNextRun(sched.CronExpr, tz, time.Now())
		if _, err := db.DB.ExecContext(context.Background(), `
			INSERT INTO workspace_schedules (workspace_id, name, cron_expr, timezone, prompt, enabled, next_run_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, id, sched.Name, sched.CronExpr, tz, sched.Prompt, enabled, nextRun); err != nil {
			log.Printf("Org import: failed to create schedule '%s' for %s: %v", sched.Name, ws.Name, err)
		} else {
			log.Printf("Org import: schedule '%s' (%s) created for %s", sched.Name, sched.CronExpr, ws.Name)
		}
	}

	// Insert channels if defined (Telegram, Slack, etc.). Config values
	// support ${VAR} expansion from .env files. The manager is reloaded
	// once at the end of org import (in Import), not per-workspace.
	channelEnv := loadWorkspaceEnv(orgBaseDir, ws.FilesDir)
	wsChannelsCreated := []string{}
	wsChannelsSkipped := []map[string]string{}
	// skipChannel records a skipped channel with consistent shape across all reasons.
	skipChannel := func(channelType, reason string) {
		wsChannelsSkipped = append(wsChannelsSkipped, map[string]string{
			"workspace": ws.Name,
			"type":      channelType, // empty string when type field was missing
			"reason":    reason,
		})
	}

	for _, ch := range ws.Channels {
		if ch.Type == "" {
			skipChannel("", "empty type")
			log.Printf("Org import: skipping channel with empty type for %s", ws.Name)
			continue
		}
		// Validate adapter exists upfront — fail fast instead of inserting orphan rows
		adapter, ok := channels.GetAdapter(ch.Type)
		if !ok {
			skipChannel(ch.Type, "unknown adapter")
			log.Printf("Org import: skipping %s channel for %s — no adapter registered", ch.Type, ws.Name)
			continue
		}

		expandedConfig := make(map[string]interface{}, len(ch.Config))
		missing := []string{}
		for k, v := range ch.Config {
			expanded := expandWithEnv(v, channelEnv)
			if hasUnresolvedVarRef(v, expanded) {
				missing = append(missing, v)
			}
			expandedConfig[k] = expanded
		}
		if len(missing) > 0 {
			skipChannel(ch.Type, fmt.Sprintf("missing env: %v", missing))
			log.Printf("Org import: skipping %s channel for %s — env vars not set: %v", ch.Type, ws.Name, missing)
			continue
		}

		// Adapter-level config validation
		if err := adapter.ValidateConfig(expandedConfig); err != nil {
			skipChannel(ch.Type, err.Error())
			log.Printf("Org import: skipping %s channel for %s — invalid config: %v", ch.Type, ws.Name, err)
			continue
		}

		configJSON, err := json.Marshal(expandedConfig)
		if err != nil {
			log.Printf("Org import: failed to marshal config for %s channel: %v", ch.Type, err)
			continue
		}
		allowedJSON, err := json.Marshal(ch.AllowedUsers)
		if err != nil {
			log.Printf("Org import: failed to marshal allowed_users for %s channel: %v", ch.Type, err)
			continue
		}
		enabled := true
		if ch.Enabled != nil {
			enabled = *ch.Enabled
		}
		// Idempotent insert — if same workspace+type already exists, update config
		if _, err := db.DB.ExecContext(context.Background(), `
			INSERT INTO workspace_channels (workspace_id, channel_type, channel_config, enabled, allowed_users)
			VALUES ($1, $2, $3::jsonb, $4, $5::jsonb)
			ON CONFLICT (workspace_id, channel_type) DO UPDATE
			SET channel_config = EXCLUDED.channel_config,
			    enabled = EXCLUDED.enabled,
			    allowed_users = EXCLUDED.allowed_users,
			    updated_at = now()
		`, id, ch.Type, string(configJSON), enabled, string(allowedJSON)); err != nil {
			log.Printf("Org import: failed to create %s channel for %s: %v", ch.Type, ws.Name, err)
		} else {
			wsChannelsCreated = append(wsChannelsCreated, ch.Type)
			log.Printf("Org import: %s channel created for %s", ch.Type, ws.Name)
		}
	}

	resultEntry := map[string]interface{}{
		"id":   id,
		"name": ws.Name,
		"tier": tier,
	}
	if len(wsChannelsCreated) > 0 {
		resultEntry["channels"] = wsChannelsCreated
	}
	if len(wsChannelsSkipped) > 0 {
		resultEntry["channels_skipped"] = wsChannelsSkipped
	}
	*results = append(*results, resultEntry)

	// Recurse into children. Brief pacing avoids overwhelming Docker when
	// creating many containers in sequence; container provisioning runs in
	// goroutines so the main createWorkspaceTree returns quickly.
	for _, child := range ws.Children {
		if err := h.createWorkspaceTree(child, &id, defaults, orgBaseDir, results); err != nil {
			return err
		}
		time.Sleep(workspaceCreatePacingMs * time.Millisecond)
	}

	return nil
}

func countWorkspaces(workspaces []OrgWorkspace) int {
	count := len(workspaces)
	for _, ws := range workspaces {
		count += countWorkspaces(ws.Children)
	}
	return count
}

// envVarRefPattern matches actual ${VAR} or $VAR references (not literal $).
// Used to detect unresolved placeholders without false positives like "$5".
var envVarRefPattern = regexp.MustCompile(`\$\{?[A-Za-z_][A-Za-z0-9_]*\}?`)

// hasUnresolvedVarRef returns true if the original string had a ${VAR} or $VAR
// reference that the expanded string didn't fully replace (i.e. the var was unset).
func hasUnresolvedVarRef(original, expanded string) bool {
	if !envVarRefPattern.MatchString(original) {
		return false // no var refs to resolve
	}
	// If expansion produced the same string and that string still has refs, unresolved.
	// If expansion stripped them to "", also unresolved.
	return expanded == "" || envVarRefPattern.MatchString(expanded)
}

// expandWithEnv expands ${VAR} and $VAR references in s using the env map.
// Falls back to the platform process env if a var isn't in the map.
func expandWithEnv(s string, env map[string]string) string {
	return os.Expand(s, func(key string) string {
		if v, ok := env[key]; ok {
			return v
		}
		return os.Getenv(key)
	})
}

// loadWorkspaceEnv reads the org root .env and the workspace-specific .env
// (workspace overrides org root). Used by both secret injection and channel
// config expansion.
func loadWorkspaceEnv(orgBaseDir, filesDir string) map[string]string {
	envVars := map[string]string{}
	if orgBaseDir == "" {
		return envVars
	}
	parseEnvFile(filepath.Join(orgBaseDir, ".env"), envVars)
	if filesDir != "" {
		parseEnvFile(filepath.Join(orgBaseDir, filesDir, ".env"), envVars)
	}
	return envVars
}

// parseEnvFile reads a .env file and adds KEY=VALUE pairs to the map.
// Skips comments (#) and empty lines. Values can be quoted.
func parseEnvFile(path string, out map[string]string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		// Strip surrounding quotes
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}
		if key != "" && value != "" {
			out[key] = value
		}
	}
}

// mergeCategoryRouting unions defaults.category_routing with per-workspace
// category_routing. Workspace-level keys override the default's value for that
// key (the role list is replaced wholesale, not unioned per-key, so a workspace
// can narrow a category — e.g. "infra: [DevOps Only]"). Empty role lists drop
// the category entirely. See issue #51.
func mergeCategoryRouting(defaultRouting, wsRouting map[string][]string) map[string][]string {
	out := map[string][]string{}
	for k, v := range defaultRouting {
		if k == "" || len(v) == 0 {
			continue
		}
		cp := make([]string, len(v))
		copy(cp, v)
		out[k] = cp
	}
	for k, v := range wsRouting {
		if k == "" {
			continue
		}
		if len(v) == 0 {
			// Empty list = explicit "drop this category for this workspace"
			delete(out, k)
			continue
		}
		cp := make([]string, len(v))
		copy(cp, v)
		out[k] = cp
	}
	return out
}

// renderCategoryRoutingYAML emits a deterministic YAML block of the form:
//
//	category_routing:
//	  security: [Backend Engineer, DevOps]
//	  ui: [Frontend Engineer]
//
// Keys are sorted for stable, test-friendly output. Uses yaml.Node + yaml.Marshal
// so role names containing YAML-reserved characters (colons, quotes, unicode line
// separators, etc.) are escaped by the YAML library — no ad-hoc quoting.
func renderCategoryRoutingYAML(routing map[string][]string) (string, error) {
	if len(routing) == 0 {
		return "", nil
	}
	keys := make([]string, 0, len(routing))
	for k := range routing {
		if k == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	inner := &yaml.Node{Kind: yaml.MappingNode}
	for _, k := range keys {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: k}
		valNode := &yaml.Node{Kind: yaml.SequenceNode, Style: yaml.FlowStyle}
		for _, role := range routing[k] {
			valNode.Content = append(valNode.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: role})
		}
		inner.Content = append(inner.Content, keyNode, valNode)
	}
	doc := &yaml.Node{Kind: yaml.MappingNode}
	doc.Content = []*yaml.Node{
		{Kind: yaml.ScalarNode, Value: "category_routing"},
		inner,
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// appendYAMLBlock concatenates a YAML fragment to an existing buffer, guaranteeing
// a newline boundary between them. Upstream code writes config.yaml in fragments
// (base template → category_routing → initial_prompt) and the base isn't
// guaranteed to end in \n, which would merge the last line into the next block.
func appendYAMLBlock(existing []byte, block string) []byte {
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		existing = append(existing, '\n')
	}
	return append(existing, []byte(block)...)
}

// mergePlugins returns the union of defaults and per-workspace plugin lists
// (deduplicated, defaults first). A per-workspace entry starting with "!" or
// "-" opts that plugin OUT of the union. See issue #68.
func mergePlugins(defaultPlugins, wsPlugins []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(defaultPlugins)+len(wsPlugins))
	for _, p := range defaultPlugins {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	for _, p := range wsPlugins {
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, "!") || strings.HasPrefix(p, "-") {
			target := strings.TrimLeft(p, "!-")
			if target == "" {
				continue
			}
			if seen[target] {
				delete(seen, target)
				filtered := out[:0]
				for _, existing := range out {
					if existing != target {
						filtered = append(filtered, existing)
					}
				}
				out = filtered
			}
			continue
		}
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}
