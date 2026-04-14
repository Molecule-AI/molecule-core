package handlers

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/crypto"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/models"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/provisioner"
)

// provisionWorkspace handles async container deployment with timeout.
func (h *WorkspaceHandler) provisionWorkspace(workspaceID, templatePath string, configFiles map[string][]byte, payload models.CreateWorkspacePayload) {
	h.provisionWorkspaceOpts(workspaceID, templatePath, configFiles, payload, false)
}

// provisionWorkspaceOpts is the workhorse variant of provisionWorkspace that
// accepts extra per-invocation knobs (e.g. resetClaudeSession for issue #12)
// that should NOT be persisted on CreateWorkspacePayload because they're
// request-scoped flags.
func (h *WorkspaceHandler) provisionWorkspaceOpts(workspaceID, templatePath string, configFiles map[string][]byte, payload models.CreateWorkspacePayload, resetClaudeSession bool) {
	ctx, cancel := context.WithTimeout(context.Background(), provisioner.ProvisionTimeout)
	defer cancel()

	// Load global secrets first, then workspace-specific secrets (which override globals).
	envVars := map[string]string{}

	// 1. Global secrets (platform-wide defaults). Uses DecryptVersioned
	// so plaintext rows written before encryption was enabled (#85)
	// keep working. A decrypt failure aborts provisioning — silent skip
	// used to manifest as opaque "missing OAuth token" preflight crashes.
	globalRows, globalErr := db.DB.QueryContext(ctx,
		`SELECT key, encrypted_value, encryption_version FROM global_secrets`)
	if globalErr == nil {
		defer globalRows.Close()
		for globalRows.Next() {
			var k string
			var v []byte
			var ver int
			if globalRows.Scan(&k, &v, &ver) == nil {
				decrypted, decErr := crypto.DecryptVersioned(v, ver)
				if decErr != nil {
					log.Printf("Provisioner: FATAL — failed to decrypt global secret %s (version=%d): %v — aborting provision of workspace %s", k, ver, decErr, workspaceID)
					h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_PROVISION_FAILED", workspaceID, map[string]interface{}{
						"error": fmt.Sprintf("cannot decrypt global secret %s: %v", k, decErr),
					})
					db.DB.ExecContext(ctx, `UPDATE workspaces SET status = 'failed', updated_at = now() WHERE id = $1`, workspaceID)
					return
				}
				envVars[k] = string(decrypted)
			}
		}
	}

	// 2. Workspace-specific secrets (override globals with same key)
	rows, err := db.DB.QueryContext(ctx,
		`SELECT key, encrypted_value, encryption_version FROM workspace_secrets WHERE workspace_id = $1`, workspaceID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var k string
			var v []byte
			var ver int
			if rows.Scan(&k, &v, &ver) == nil {
				decrypted, decErr := crypto.DecryptVersioned(v, ver)
				if decErr != nil {
					log.Printf("Provisioner: FATAL — failed to decrypt workspace secret %s (version=%d) for %s: %v — aborting provision", k, ver, workspaceID, decErr)
					h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_PROVISION_FAILED", workspaceID, map[string]interface{}{
						"error": fmt.Sprintf("cannot decrypt workspace secret %s: %v", k, decErr),
					})
					db.DB.ExecContext(ctx, `UPDATE workspaces SET status = 'failed', updated_at = now() WHERE id = $1`, workspaceID)
					return
				}
				envVars[k] = string(decrypted)
			}
		}
	}

	pluginsPath, _ := filepath.Abs(filepath.Join(h.configsDir, "..", "plugins"))
	awarenessNamespace := h.loadAwarenessNamespace(ctx, workspaceID)
	cfg := h.buildProvisionerConfig(workspaceID, templatePath, configFiles, payload, envVars, pluginsPath, awarenessNamespace)
	cfg.ResetClaudeSession = resetClaudeSession // #12

	// Preflight #17: refuse to start a container we already know will crash on missing config.yaml.
	// When the caller supplies neither a template dir nor in-memory configFiles (the auto-restart
	// path), probe the existing Docker named volume. If it's empty/missing config.yaml, mark the
	// workspace 'failed' instead of handing it to Docker's unless-stopped restart policy, which
	// would otherwise loop forever on FileNotFoundError.
	if srcErr := provisioner.ValidateConfigSource(templatePath, configFiles); srcErr != nil {
		hasConfig, probeErr := h.provisioner.VolumeHasFile(ctx, workspaceID, "config.yaml")
		if probeErr != nil {
			log.Printf("Provisioner: config.yaml preflight probe failed for %s: %v (proceeding)", workspaceID, probeErr)
		} else if !hasConfig {
			msg := fmt.Sprintf("cannot start workspace %s: no config.yaml source and config volume is empty — delete the workspace or provide a template", workspaceID)
			log.Printf("Provisioner: %s", msg)
			if _, dbErr := db.DB.ExecContext(ctx,
				`UPDATE workspaces SET status = 'failed', last_sample_error = $2, updated_at = now() WHERE id = $1`,
				workspaceID, msg); dbErr != nil {
				log.Printf("Provisioner: failed to mark workspace %s as failed: %v", workspaceID, dbErr)
			}
			h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_PROVISION_FAILED", workspaceID, map[string]interface{}{
				"error": msg,
			})
			return
		}
	}

	url, err := h.provisioner.Start(ctx, cfg)
	if err != nil {
		// Persist the error text to last_sample_error so the canvas and
		// GET /workspaces/:id expose something actionable — previously the
		// provision failure was only logged + broadcast, leaving the DB
		// row with an empty last_sample_error. Issue #117.
		log.Printf("Provisioner: failed to start workspace %s: %v", workspaceID, err)
		if _, dbErr := db.DB.ExecContext(ctx,
			`UPDATE workspaces SET status = 'failed', last_sample_error = $2, updated_at = now() WHERE id = $1`,
			workspaceID, err.Error()); dbErr != nil {
			log.Printf("Provisioner: failed to mark workspace %s as failed: %v", workspaceID, dbErr)
		}
		h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_PROVISION_FAILED", workspaceID, map[string]interface{}{
			"error": err.Error(),
		})
	} else if url != "" {
		// Pre-store the host-accessible URL (http://127.0.0.1:<port>) so the A2A proxy can reach the container.
		// The registry's ON CONFLICT preserves URLs starting with http://127.0.0.1 when the agent self-registers.
		if _, dbErr := db.DB.ExecContext(ctx, `UPDATE workspaces SET url = $1 WHERE id = $2`, url, workspaceID); dbErr != nil {
			log.Printf("Provisioner: failed to store URL for %s: %v", workspaceID, dbErr)
		}
		if cacheErr := db.CacheURL(ctx, workspaceID, url); cacheErr != nil {
			log.Printf("Provisioner: failed to cache URL for %s: %v", workspaceID, cacheErr)
		}
		// Also cache the Docker-internal URL for workspace-to-workspace discovery.
		// Containers on molecule-monorepo-net can reach each other by container name.
		internalURL := provisioner.InternalURL(workspaceID)
		if cacheErr := db.CacheInternalURL(ctx, workspaceID, internalURL); cacheErr != nil {
			log.Printf("Provisioner: failed to cache internal URL for %s: %v", workspaceID, cacheErr)
		}
	}
	// On success, the workspace will register via POST /registry/register
	// which transitions status to 'online' and broadcasts WORKSPACE_ONLINE
}

func workspaceAwarenessNamespace(workspaceID string) string {
	return fmt.Sprintf("workspace:%s", workspaceID)
}

func (h *WorkspaceHandler) loadAwarenessNamespace(ctx context.Context, workspaceID string) string {
	var awarenessNamespace string
	err := db.DB.QueryRowContext(ctx, `SELECT COALESCE(awareness_namespace, '') FROM workspaces WHERE id = $1`, workspaceID).Scan(&awarenessNamespace)
	if err != nil || awarenessNamespace == "" {
		return workspaceAwarenessNamespace(workspaceID)
	}
	return awarenessNamespace
}

func (h *WorkspaceHandler) buildProvisionerConfig(
	workspaceID, templatePath string,
	configFiles map[string][]byte,
	payload models.CreateWorkspacePayload,
	envVars map[string]string,
	pluginsPath, awarenessNamespace string,
) provisioner.WorkspaceConfig {
	// Per-workspace workspace_dir takes priority over global WORKSPACE_DIR env var.
	// If neither is set, the provisioner creates an isolated Docker volume.
	//
	// #65: also read workspace_access (DB column) so restart paths preserve
	// the mode set at create/import time. Payload's WorkspaceAccess (if
	// present) wins, matching the existing WorkspaceDir precedence.
	workspacePath := payload.WorkspaceDir
	workspaceAccess := payload.WorkspaceAccess
	if workspacePath == "" || workspaceAccess == "" {
		var dbDir, dbAccess string
		if err := db.DB.QueryRow(
			`SELECT COALESCE(workspace_dir, ''), COALESCE(workspace_access, 'none') FROM workspaces WHERE id = $1`,
			workspaceID,
		).Scan(&dbDir, &dbAccess); err == nil {
			if workspacePath == "" && dbDir != "" {
				workspacePath = dbDir
			}
			if workspaceAccess == "" {
				workspaceAccess = dbAccess
			}
		}
	}
	if workspacePath == "" {
		workspacePath = os.Getenv("WORKSPACE_DIR")
	}
	if workspaceAccess == "" {
		workspaceAccess = provisioner.WorkspaceAccessNone
	}

	return provisioner.WorkspaceConfig{
		WorkspaceID:        workspaceID,
		TemplatePath:       templatePath,
		ConfigFiles:        configFiles,
		PluginsPath:        pluginsPath,
		WorkspacePath:      workspacePath,
		WorkspaceAccess:    workspaceAccess,
		Tier:               payload.Tier,
		Runtime:            payload.Runtime,
		EnvVars:            envVars,
		PlatformURL:        h.platformURL,
		AwarenessURL:       os.Getenv("AWARENESS_URL"),
		AwarenessNamespace: awarenessNamespace,
	}
}

// findTemplateByName looks for a workspace-configs-templates directory matching a name.
func findTemplateByName(configsDir, name string) string {
	entries, err := os.ReadDir(configsDir)
	if err != nil {
		return ""
	}
	// Normalize name: "SEO Agent" → look for "seo-agent"
	normalized := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	for _, e := range entries {
		if e.IsDir() && e.Name() == normalized {
			return e.Name()
		}
	}
	// Also search by config.yaml name field (for templates like org-pm where dir name != workspace name)
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), "ws-") {
			continue
		}
		cfgPath := filepath.Join(configsDir, e.Name(), "config.yaml")
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			continue
		}
		// Quick YAML name extraction (avoids importing yaml parser)
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "name:") {
				cfgName := strings.TrimSpace(strings.TrimPrefix(line, "name:"))
				if strings.EqualFold(cfgName, name) {
					return e.Name()
				}
				break
			}
		}
	}
	return ""
}

// configDirName returns the standard config directory name for a workspace ID.
// Used by resolveConfigDir in templates.go for host-side template resolution.
func configDirName(workspaceID string) string {
	id := workspaceID
	if len(id) > 12 {
		id = id[:12]
	}
	return "ws-" + id
}

// ensureDefaultConfig generates minimal config files in memory for workspaces without a template.
// Returns a map of filename → content to be written into the container's /configs volume.
func (h *WorkspaceHandler) ensureDefaultConfig(workspaceID string, payload models.CreateWorkspacePayload) map[string][]byte {
	files := make(map[string][]byte)

	// Determine runtime
	runtime := payload.Runtime
	if runtime == "" {
		runtime = "langgraph"
	}

	// Generate a minimal config.yaml
	model := payload.Model
	if model == "" {
		if runtime == "claude-code" {
			model = "sonnet"
		} else {
			model = "anthropic:claude-sonnet-4-6"
		}
	}

	// Sanitize name/role for YAML safety — quote values that contain special chars
	safeName := strings.ReplaceAll(strings.ReplaceAll(payload.Name, "\n", " "), "\r", "")
	safeRole := strings.ReplaceAll(strings.ReplaceAll(payload.Role, "\n", " "), "\r", "")
	// Quote if contains YAML-breaking chars
	quoteName := safeName
	quoteRole := safeRole
	for _, special := range []string{":", "#", "'", "\"", "{", "}", "[", "]"} {
		if strings.Contains(safeName, special) {
			quoteName = fmt.Sprintf("%q", safeName)
			break
		}
	}
	for _, special := range []string{":", "#", "'", "\"", "{", "}", "[", "]"} {
		if strings.Contains(safeRole, special) {
			quoteRole = fmt.Sprintf("%q", safeRole)
			break
		}
	}
	configYAML := fmt.Sprintf("name: %s\ndescription: %s\nversion: 1.0.0\ntier: %d\nruntime: %s\n",
		quoteName, quoteRole, payload.Tier, runtime)

	// Model always at top level — config.py reads raw["model"] for all runtimes.
	configYAML += fmt.Sprintf("model: %s\n", model)

	// Add required_env based on runtime — preflight checks these are set via secrets API.
	switch runtime {
	case "claude-code":
		configYAML += "runtime_config:\n  required_env:\n    - CLAUDE_CODE_OAUTH_TOKEN\n  timeout: 0\n"
	case "codex":
		configYAML += "runtime_config:\n  required_env:\n    - OPENAI_API_KEY\n  timeout: 0\n"
	case "langgraph", "deepagents":
		// These runtimes read API keys from env directly, no runtime_config needed.
	default:
		configYAML += "runtime_config:\n  timeout: 0\n"
	}

	files["config.yaml"] = []byte(configYAML)

	log.Printf("Provisioner: generated %d config files for workspace %s (runtime: %s)", len(files), workspaceID, runtime)
	return files
}
