package handlers

// org_import.go — workspace tree creation during org template import.
// Contains createWorkspaceTree (recursive provisioning) and countWorkspaces.

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/channels"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/crypto"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/models"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/provisioner"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/scheduler"
	"github.com/google/uuid"
)
// createWorkspaceTree recursively materialises an OrgWorkspace (and its
// descendants) into the workspaces + canvas_layouts tables and kicks off
// Docker provisioning. absX/absY are THIS workspace's absolute canvas
// coordinates — roots inherit them from ws.Canvas, children receive
// parent.abs + childSlotInGrid(index, siblingSizes) computed by the
// caller. Storing already-absolute coords means a child that is itself
// a parent can simply compound the grid without any per-call math.
// relX / relY are THIS workspace's position RELATIVE to its parent's
// absolute origin (i.e. childSlotInGrid output for children; 0,0 for
// roots since a root's absolute IS its relative). The broadcast
// payload ships relative coords so the canvas can drop the node
// straight into the parent's child-coordinate space without doing a
// canvas-wide absolute-position walk.
func (h *OrgHandler) createWorkspaceTree(ws OrgWorkspace, parentID *string, absX, absY, relX, relY float64, defaults OrgDefaults, orgBaseDir string, results *[]map[string]interface{}, provisionSem chan struct{}) error {
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
			model = "anthropic:claude-opus-4-7"
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

	// Expand ${VAR} references in workspace_dir against the org's .env files
	// before validation. Without this, a template that ships
	// `workspace_dir: ${WORKSPACE_DIR}` (so each operator can pick the host
	// path to bind-mount) reaches validateWorkspaceDir as the literal
	// "${WORKSPACE_DIR}" string and fails with "must be an absolute path".
	// Other fields (channel config, prompts) already go through expandWithEnv;
	// workspace_dir was the last hold-out.
	if ws.WorkspaceDir != "" {
		ws.WorkspaceDir = expandWithEnv(ws.WorkspaceDir, loadWorkspaceEnv(orgBaseDir, ws.FilesDir))
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

	// Org-template imports default to expanded so children render
	// visually nested inside their parent — matches the user's mental
	// model ("all children should be in front of its parent"). The
	// topology rescue heuristic lays any children whose YAML coords
	// fall outside the computed parent bbox into a tidy 2-column grid
	// (see canvas-topology.ts), so imports don't spray the viewport.
	initialCollapsed := false

	_, err := db.DB.ExecContext(ctx, `
		INSERT INTO workspaces (id, name, role, tier, runtime, awareness_namespace, status, parent_id, workspace_dir, workspace_access)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, id, ws.Name, role, tier, runtime, awarenessNS, "provisioning", parentID, workspaceDir, workspaceAccess)
	if err != nil {
		log.Printf("Org import: failed to create %s: %v", ws.Name, err)
		return fmt.Errorf("failed to create %s: %w", ws.Name, err)
	}

	// Canvas layout — absX/absY were computed by the caller using the
	// subtree-aware grid (childSlotInGrid) so a nested-parent child
	// doesn't clip into its siblings. Raw YAML canvas coords are only
	// consulted at the root: many templates predate the nested-parent
	// model and author them as a flat horizontal row (y=180, x=100..1220),
	// which overlaps chaotically once the cards render inside a parent
	// container.
	//
	// `collapsed` lives on canvas_layouts (005_canvas_layouts.sql), not
	// on workspaces; the UI-only flag is intentionally decoupled from
	// the workspace row.
	if _, err := db.DB.ExecContext(ctx, `INSERT INTO canvas_layouts (workspace_id, x, y, collapsed) VALUES ($1, $2, $3, $4)`, id, absX, absY, initialCollapsed); err != nil {
		log.Printf("Org import: canvas layout insert failed for %s: %v", ws.Name, err)
	}

	// Broadcast — include runtime so the canvas pill renders the right
	// badge immediately instead of "unknown". parent_id + x/y let the
	// canvas's org-deploy animation spawn the child from the parent's
	// current coords and tween into its reserved slot, instead of
	// landing in a default grid position first and snapping on the
	// next hydrate.
	payload := map[string]interface{}{
		"name": ws.Name, "tier": tier, "runtime": runtime,
		// Parent-relative coords — the canvas's React Flow node uses
		// these as the node's position when parent_id is set (React
		// Flow treats node.position as parent-relative when the node
		// has a parentId). For roots, relX/relY == absX/absY.
		"x": relX, "y": relY,
	}
	if parentID != nil {
		payload["parent_id"] = *parentID
	}
	h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_PROVISIONING", id, payload)

	// Seed initial memories from workspace config or defaults (issue #1050).
	// Per-workspace initial_memories override defaults; if workspace has none,
	// fall back to defaults.initial_memories.
	wsMemories := ws.InitialMemories
	if len(wsMemories) == 0 {
		wsMemories = defaults.InitialMemories
	}
	seedInitialMemories(ctx, id, wsMemories, awarenessNS)

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
		// boot (see workspace/heartbeat.py + config.py).
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
				idleInterval = 600 // same default as workspace/config.py
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

		// #1084: limit concurrent Docker provisioning via semaphore.
		provisionSem <- struct{}{} // acquire
		go func(wID, tPath string, cFiles map[string][]byte, p models.CreateWorkspacePayload) {
			defer func() { <-provisionSem }() // release
			h.workspace.provisionWorkspace(wID, tPath, cFiles, p)
		}(id, templatePath, configFiles, payload)
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
		// #722: surface the error rather than silently using time.Time{} (zero)
		// which lib/pq stores as 0001-01-01 and may confuse the fire query.
		nextRun, nextRunErr := scheduler.ComputeNextRun(sched.CronExpr, tz, time.Now())
		if nextRunErr != nil {
			log.Printf("Org import: invalid cron expression for schedule '%s' on %s: %v — skipping insert",
				sched.Name, ws.Name, nextRunErr)
			continue
		}
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
	// Children's abs coords = this.abs + childSlotInGrid(index, siblingSizes),
	// with sibling sizes computed by sizeOfSubtree so a nested-parent
	// child claims a bigger grid slot than a leaf sibling — no slot
	// clipping across mixed leaf / parent siblings.
	if len(ws.Children) > 0 {
		siblingSizes := make([]nodeSize, len(ws.Children))
		for i, c := range ws.Children {
			siblingSizes[i] = sizeOfSubtree(c)
		}
		for i, child := range ws.Children {
			slotX, slotY := childSlotInGrid(i, siblingSizes)
			childAbsX := absX + slotX
			childAbsY := absY + slotY
			// slotX/slotY are already parent-relative — that's
			// exactly what childSlotInGrid returns.
			if err := h.createWorkspaceTree(child, &id, childAbsX, childAbsY, slotX, slotY, defaults, orgBaseDir, results, provisionSem); err != nil {
				return err
			}
			time.Sleep(workspaceCreatePacingMs * time.Millisecond)
		}
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
