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

// orgImportScheduleSQL is the upsert executed for every schedule during
// org/import. Extracted to a const so TestImport_OrgScheduleSQLShape can
// assert its shape without regex-scanning org.go (issue #24 follow-up).
//
// Guarantees, in one statement:
//   - INSERT new rows with source='template'
//   - On (workspace_id, name) collision, only refresh template-source rows
//     (runtime-added schedules are preserved across re-imports)
//   - No DELETE — removal is out of scope (additive semantics)
const orgImportScheduleSQL = `
INSERT INTO workspace_schedules (workspace_id, name, cron_expr, timezone, prompt, enabled, next_run_at, source)
VALUES ($1, $2, $3, $4, $5, $6, $7, 'template')
ON CONFLICT (workspace_id, name) DO UPDATE
    SET cron_expr   = EXCLUDED.cron_expr,
        timezone    = EXCLUDED.timezone,
        prompt      = EXCLUDED.prompt,
        enabled     = EXCLUDED.enabled,
        next_run_at = EXCLUDED.next_run_at,
        updated_at  = now()
    WHERE workspace_schedules.source = 'template'
`

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
	Runtime       string   `yaml:"runtime" json:"runtime"`
	Tier          int      `yaml:"tier" json:"tier"`
	Model         string   `yaml:"model" json:"model"`
	Plugins       []string `yaml:"plugins" json:"plugins"`
	InitialPrompt string   `yaml:"initial_prompt" json:"initial_prompt"`
	// InitialPromptFile is a file ref alternative to InitialPrompt. Path is
	// resolved relative to the workspace's files_dir (or the org base dir
	// when used at defaults level — defaults don't have their own files_dir,
	// so the file must live at the org root). Inline InitialPrompt wins
	// when both are set.
	InitialPromptFile string `yaml:"initial_prompt_file" json:"initial_prompt_file"`
	// IdlePrompt / IdleIntervalSeconds are the workspace-default idle-loop
	// body and cadence (see workspace-template/heartbeat.py). They were
	// previously dropped by the org importer because the struct didn't
	// declare them — causing live configs to boot without idle_prompts
	// even when org.yaml had them. Phase 1 scalability work adds both
	// inline + file-ref forms.
	IdlePrompt           string `yaml:"idle_prompt" json:"idle_prompt"`
	IdlePromptFile       string `yaml:"idle_prompt_file" json:"idle_prompt_file"`
	IdleIntervalSeconds  int    `yaml:"idle_interval_seconds" json:"idle_interval_seconds"`
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
	// PromptFile is a file ref alternative to inline Prompt. Path is
	// resolved relative to the workspace's files_dir. Inline Prompt wins
	// when both are set. Scalability: hourly/weekly cron prompts are the
	// largest text bodies in org.yaml (~1-5 KB each); externalizing them
	// cuts the file by ~62%.
	PromptFile string `yaml:"prompt_file" json:"prompt_file"`
	Enabled    *bool  `yaml:"enabled" json:"enabled"`
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
	Name     string `yaml:"name" json:"name"`
	Role     string `yaml:"role" json:"role"`
	Runtime  string `yaml:"runtime" json:"runtime"`
	Tier     int    `yaml:"tier" json:"tier"`
	Template string `yaml:"template" json:"template"`
	FilesDir string `yaml:"files_dir" json:"files_dir"`
	// SystemPrompt is an inline override. Normally each role's system-prompt.md
	// lives at `<files_dir>/system-prompt.md` and is copied via the files_dir
	// template-copy step; inline overrides that path for ad-hoc workspaces.
	SystemPrompt    string   `yaml:"system_prompt" json:"system_prompt"`
	Model           string   `yaml:"model" json:"model"`
	WorkspaceDir    string   `yaml:"workspace_dir" json:"workspace_dir"`
	WorkspaceAccess string   `yaml:"workspace_access" json:"workspace_access"` // #65: "none" (default), "read_only", "read_write"
	Plugins         []string `yaml:"plugins" json:"plugins"`
	// InitialPrompt is the one-shot boot prompt. Agents run this once on first
	// start; the body often clones the repo, reads CLAUDE.md + system-prompt,
	// and commits conventions to memory. InitialPromptFile is the file-ref
	// alternative — read at import time from `<files_dir>/<InitialPromptFile>`.
	// Inline wins when both are set.
	InitialPrompt     string `yaml:"initial_prompt" json:"initial_prompt"`
	InitialPromptFile string `yaml:"initial_prompt_file" json:"initial_prompt_file"`
	// IdlePrompt / IdleIntervalSeconds drive the idle-loop reflection
	// pattern (#205). When IdlePrompt is non-empty, the workspace self-sends
	// this prompt every IdleIntervalSeconds while heartbeat.active_tasks == 0.
	// Both fields were previously dropped by the org importer (struct didn't
	// declare them); Phase 1 scalability PR adds them so engineer + researcher
	// idle loops propagate correctly from org.yaml → /configs/config.yaml.
	// IdlePromptFile is the file-ref alternative — same semantics as
	// InitialPromptFile. Inline wins when both are set.
	IdlePrompt          string `yaml:"idle_prompt" json:"idle_prompt"`
	IdlePromptFile      string `yaml:"idle_prompt_file" json:"idle_prompt_file"`
	IdleIntervalSeconds int    `yaml:"idle_interval_seconds" json:"idle_interval_seconds"`
	// CategoryRouting extends/overrides defaults.category_routing per-workspace.
	// Merge semantics: workspace keys replace defaults' value for the same key
	// (empty list drops the category entirely); new keys are added. See
	// mergeCategoryRouting.
	CategoryRouting map[string][]string `yaml:"category_routing" json:"category_routing"`
	Schedules       []OrgSchedule       `yaml:"schedules" json:"schedules"`
	Channels        []OrgChannel        `yaml:"channels" json:"channels"`
	External        bool                `yaml:"external" json:"external"`
	URL             string              `yaml:"url" json:"url"`
	Canvas          struct {
		X float64 `yaml:"x" json:"x"`
		Y float64 `yaml:"y" json:"y"`
	} `yaml:"canvas" json:"canvas"`
	Children []OrgWorkspace `yaml:"children" json:"children"`
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
		templateDir := filepath.Join(h.orgDir, e.Name())
		orgFile := filepath.Join(templateDir, "org.yaml")
		data, err := os.ReadFile(orgFile)
		if err != nil {
			// Try org.yml
			orgFile = filepath.Join(templateDir, "org.yml")
			data, err = os.ReadFile(orgFile)
			if err != nil {
				continue
			}
		}
		// Expand !include directives before unmarshal so templates that
		// split across team/role files still report an accurate workspace
		// count on the /org/templates listing.
		if expanded, err := resolveYAMLIncludes(data, templateDir); err == nil {
			data = expanded
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
		// Reject traversal attempts — `dir` must resolve inside h.orgDir.
		// Without this, `dir: "../../../etc"` gets joined into h.orgDir and
		// filepath.Join's lexical cleanup resolves it outside the root,
		// letting an unauthenticated caller probe arbitrary filesystem paths.
		resolved, err := resolveInsideRoot(h.orgDir, body.Dir)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid dir: %v", err)})
			return
		}
		orgBaseDir = resolved
		orgFile := filepath.Join(orgBaseDir, "org.yaml")
		data, err := os.ReadFile(orgFile)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("org template not found: %s", body.Dir)})
			return
		}
		// Expand !include directives before unmarshal. Splits org.yaml
		// into per-team or per-role files; Phase 3 of the scalability
		// refactor. Fails loudly on missing / cyclic / escaping includes.
		expanded, err := resolveYAMLIncludes(data, orgBaseDir)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("!include expansion failed: %v", err)})
			return
		}
		if err := yaml.Unmarshal(expanded, &tmpl); err != nil {
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
			// `template` comes from the uploaded YAML — treat as untrusted.
			// Only accept paths that stay inside h.configsDir.
			if tp, err := resolveInsideRoot(h.configsDir, ws.Template); err == nil {
				if _, statErr := os.Stat(tp); statErr == nil {
					templatePath = tp
				}
			}
		}
		if templatePath == "" {
			// #241: sanitizeRuntime() allowlists the runtime string so a
			// crafted org.yaml cannot use it as a path-traversal oracle.
			safeRuntime := sanitizeRuntime(runtime)
			runtimeDefault := filepath.Join(h.configsDir, safeRuntime+"-default")
			if _, err := os.Stat(runtimeDefault); err == nil {
				templatePath = runtimeDefault
			}
		}

		// Always generate default config.yaml (runtime, model, tier, etc.)
		configFiles := h.workspace.ensureDefaultConfig(id, payload)

		// Copy files_dir contents on top (system-prompt.md, CLAUDE.md, skills/, etc.)
		// Uses templatePath for CopyTemplateToContainer — runs AFTER configFiles are written
		if ws.FilesDir != "" && orgBaseDir != "" {
			// `files_dir` also comes from untrusted YAML. Join inside orgBaseDir
			// (already validated above) and reject anything that escapes.
			if filesPath, err := resolveInsideRoot(orgBaseDir, ws.FilesDir); err == nil {
				if info, statErr := os.Stat(filesPath); statErr == nil && info.IsDir() {
					templatePath = filesPath
				}
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

		// Resolve initial_prompt — inline wins, then file-ref, then defaults
		// (inline → file → defaults.inline → defaults.file). File refs are
		// rooted at <orgBaseDir>/<files_dir>/ per resolvePromptRef semantics.
		initialPrompt, err := resolvePromptRef(ws.InitialPrompt, ws.InitialPromptFile, orgBaseDir, ws.FilesDir)
		if err != nil {
			log.Printf("Org import: failed to resolve initial_prompt for %s: %v", ws.Name, err)
		}
		if initialPrompt == "" {
			// Fall back to defaults. Defaults live at the org root, so they
			// resolve with empty filesDir (relative to orgBaseDir).
			var defaultErr error
			initialPrompt, defaultErr = resolvePromptRef(defaults.InitialPrompt, defaults.InitialPromptFile, orgBaseDir, "")
			if defaultErr != nil {
				log.Printf("Org import: failed to resolve defaults.initial_prompt for %s: %v", ws.Name, defaultErr)
			}
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

		// Resolve idle_prompt — same precedence (ws inline → ws file → defaults).
		// Inject into config.yaml alongside idle_interval_seconds so the
		// workspace's heartbeat loop picks up the idle-reflection cadence on
		// boot (see workspace-template/heartbeat.py + config.py).
		idlePrompt, err := resolvePromptRef(ws.IdlePrompt, ws.IdlePromptFile, orgBaseDir, ws.FilesDir)
		if err != nil {
			log.Printf("Org import: failed to resolve idle_prompt for %s: %v", ws.Name, err)
		}
		if idlePrompt == "" {
			var defaultErr error
			idlePrompt, defaultErr = resolvePromptRef(defaults.IdlePrompt, defaults.IdlePromptFile, orgBaseDir, "")
			if defaultErr != nil {
				log.Printf("Org import: failed to resolve defaults.idle_prompt for %s: %v", ws.Name, defaultErr)
			}
		}
		idleInterval := ws.IdleIntervalSeconds
		if idleInterval == 0 {
			idleInterval = defaults.IdleIntervalSeconds
		}
		if idlePrompt != "" {
			if configFiles == nil {
				configFiles = map[string][]byte{}
			}
			trimmed := strings.TrimSpace(idlePrompt)
			lines := strings.Split(trimmed, "\n")
			for i, line := range lines {
				lines[i] = strings.TrimRight(line, " \t")
			}
			indented := strings.Join(lines, "\n  ")
			// idle_interval_seconds belongs with idle_prompt — empty idle_prompt
			// means the idle loop never fires regardless of interval, so we
			// only emit interval when there's a body to go with it.
			if idleInterval <= 0 {
				idleInterval = 600 // same default as workspace-template/config.py
			}
			block := fmt.Sprintf("idle_interval_seconds: %d\nidle_prompt: |\n  %s\n", idleInterval, indented)
			configFiles["config.yaml"] = appendYAMLBlock(configFiles["config.yaml"], block)
			log.Printf("Org import: injected idle_prompt (%d chars, interval=%ds) into config.yaml for %s", len(trimmed), idleInterval, ws.Name)
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

	// Insert schedules if defined. Resolve each schedule's prompt body from
	// either inline `prompt:` or `prompt_file:` (file ref relative to the
	// workspace's files_dir). Inline wins; empty prompt after resolution is
	// a configuration error (cron with no body would never do anything).
	for _, sched := range ws.Schedules {
		tz := sched.Timezone
		if tz == "" {
			tz = "UTC"
		}
		enabled := true
		if sched.Enabled != nil {
			enabled = *sched.Enabled
		}
		prompt, promptErr := resolvePromptRef(sched.Prompt, sched.PromptFile, orgBaseDir, ws.FilesDir)
		if promptErr != nil {
			log.Printf("Org import: failed to resolve prompt for schedule '%s' on %s: %v — skipping insert", sched.Name, ws.Name, promptErr)
			continue
		}
		if prompt == "" {
			log.Printf("Org import: schedule '%s' on %s has empty prompt (neither prompt nor prompt_file set) — skipping insert", sched.Name, ws.Name)
			continue
		}
		nextRun, _ := scheduler.ComputeNextRun(sched.CronExpr, tz, time.Now())
		if _, err := db.DB.ExecContext(context.Background(), orgImportScheduleSQL,
			id, sched.Name, sched.CronExpr, tz, prompt, enabled, nextRun); err != nil {
			log.Printf("Org import: failed to upsert schedule '%s' for %s: %v", sched.Name, ws.Name, err)
		} else {
			log.Printf("Org import: schedule '%s' (%s, %d chars) upserted for %s (source=template)", sched.Name, sched.CronExpr, len(prompt), ws.Name)
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

// resolvePromptRef reads a prompt body from either an inline string or a
// file ref relative to the workspace's files_dir. Inline always wins when
// both are non-empty (caller-provided inline is more authoritative than a
// file path that may not exist yet during dev loops).
//
// File resolution:
//   - `<orgBaseDir>/<filesDir>/<fileRef>` when filesDir is non-empty
//   - `<orgBaseDir>/<fileRef>` when filesDir is empty (defaults-level refs)
//
// Both paths go through resolveInsideRoot so a crafted fileRef can't escape
// the org template directory via traversal (same defense the files_dir
// copy-step uses).
//
// Returns (resolved body, error). If both inline and fileRef are empty,
// returns ("", nil) — caller decides whether that's a problem.
func resolvePromptRef(inline, fileRef, orgBaseDir, filesDir string) (string, error) {
	if inline != "" {
		return inline, nil
	}
	if fileRef == "" {
		return "", nil
	}
	if orgBaseDir == "" {
		// Inline-only template (POST /org/import with a raw Template in the
		// JSON body, not a dir). File refs can't be resolved — surface the
		// problem rather than silently returning empty.
		return "", fmt.Errorf("prompt_file %q requires a dir-based org template (no orgBaseDir in inline-template mode)", fileRef)
	}
	searchRoot := orgBaseDir
	if filesDir != "" {
		p, err := resolveInsideRoot(orgBaseDir, filesDir)
		if err != nil {
			return "", fmt.Errorf("invalid files_dir %q: %w", filesDir, err)
		}
		searchRoot = p
	}
	abs, err := resolveInsideRoot(searchRoot, fileRef)
	if err != nil {
		return "", fmt.Errorf("invalid prompt_file %q: %w", fileRef, err)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return "", fmt.Errorf("read prompt_file %q: %w", fileRef, err)
	}
	return string(data), nil
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

// resolveInsideRoot joins `userPath` onto `root` and ensures the lexically
// cleaned result stays inside root. Rejects absolute paths outright and
// anything containing ".." that would escape the root.
//
// Both arguments are resolved to absolute paths via filepath.Abs before the
// prefix check so a root passed as a relative path still works correctly.
// Follows Go's standard pattern for SSRF-class path sanitization; using
// strings.HasPrefix on an absolute-path pair plus the separator guard rejects
// sibling directories that share a prefix (e.g. "/foo" vs "/foobar").
func resolveInsideRoot(root, userPath string) (string, error) {
	if userPath == "" {
		return "", fmt.Errorf("path is empty")
	}
	if filepath.IsAbs(userPath) {
		return "", fmt.Errorf("absolute paths are not allowed")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("root abs: %w", err)
	}
	joined := filepath.Join(absRoot, userPath)
	absJoined, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("joined abs: %w", err)
	}
	// Allow exact-root match (rare but valid) and any descendant.
	if absJoined != absRoot && !strings.HasPrefix(absJoined, absRoot+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes root")
	}
	return absJoined, nil
}
