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
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/wsauth"
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
	//
	// Query errors also abort provisioning: an agent that starts without its
	// secrets fails at task time with opaque auth errors. Better to mark the
	// workspace failed immediately so the operator retries/investigates.
	globalRows, globalErr := db.DB.QueryContext(ctx,
		`SELECT key, encrypted_value, encryption_version FROM global_secrets`)
	if globalErr != nil {
		log.Printf("Provisioner: FATAL — global secrets query failed for workspace %s: %v — aborting provision", workspaceID, globalErr)
		h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_PROVISION_FAILED", workspaceID, map[string]interface{}{
			"error": fmt.Sprintf("global secrets query failed: %v", globalErr),
		})
		db.DB.ExecContext(ctx, `UPDATE workspaces SET status = 'failed', updated_at = now() WHERE id = $1`, workspaceID)
		return
	}
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
	if iterErr := globalRows.Err(); iterErr != nil {
		log.Printf("Provisioner: FATAL — global secrets iteration error for workspace %s: %v — aborting provision", workspaceID, iterErr)
		h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_PROVISION_FAILED", workspaceID, map[string]interface{}{
			"error": fmt.Sprintf("global secrets iteration failed: %v", iterErr),
		})
		db.DB.ExecContext(ctx, `UPDATE workspaces SET status = 'failed', updated_at = now() WHERE id = $1`, workspaceID)
		return
	}

	// 2. Workspace-specific secrets (override globals with same key)
	rows, err := db.DB.QueryContext(ctx,
		`SELECT key, encrypted_value, encryption_version FROM workspace_secrets WHERE workspace_id = $1`, workspaceID)
	if err != nil {
		log.Printf("Provisioner: FATAL — workspace secrets query failed for workspace %s: %v — aborting provision", workspaceID, err)
		h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_PROVISION_FAILED", workspaceID, map[string]interface{}{
			"error": fmt.Sprintf("workspace secrets query failed: %v", err),
		})
		db.DB.ExecContext(ctx, `UPDATE workspaces SET status = 'failed', updated_at = now() WHERE id = $1`, workspaceID)
		return
	}
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
	if iterErr := rows.Err(); iterErr != nil {
		log.Printf("Provisioner: FATAL — workspace secrets iteration error for workspace %s: %v — aborting provision", workspaceID, iterErr)
		h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_PROVISION_FAILED", workspaceID, map[string]interface{}{
			"error": fmt.Sprintf("workspace secrets iteration failed: %v", iterErr),
		})
		db.DB.ExecContext(ctx, `UPDATE workspaces SET status = 'failed', updated_at = now() WHERE id = $1`, workspaceID)
		return
	}

	pluginsPath, _ := filepath.Abs(filepath.Join(h.configsDir, "..", "plugins"))
	awarenessNamespace := h.loadAwarenessNamespace(ctx, workspaceID)

	// Per-agent git identity (Option 3 of agent-separation rollout).
	// Sets GIT_AUTHOR_* / GIT_COMMITTER_* so commits from each workspace
	// carry a distinct author in `git log` / `git blame` — instead of
	// every agent appearing as whoever the shared PAT belongs to. PR +
	// issue authorship is still tied to GITHUB_TOKEN (shared PAT); that
	// gets solved by the GitHub App migration (Option 1, follow-up PR).
	// Runs after secret loads so an operator can still override via a
	// workspace_secret named GIT_AUTHOR_NAME if they want custom identity.
	applyAgentGitIdentity(envVars, payload.Name)

	// Plugin extension point: run any registered EnvMutators (e.g.
	// github-app-auth, vault-secrets) AFTER built-in identity injection so
	// plugins can override or augment GIT_AUTHOR_*, GITHUB_TOKEN, etc.
	// A failure here aborts provisioning — a missing GitHub App token
	// would manifest later as opaque "git push 401" loops, and the agent
	// never recovers. Failing fast here surfaces the cause to the operator.
	if err := h.envMutators.Run(ctx, workspaceID, envVars); err != nil {
		log.Printf("Provisioner: env mutator chain failed for %s: %v", workspaceID, err)
		h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_PROVISION_FAILED", workspaceID, map[string]interface{}{
			"error": err.Error(),
		})
		if _, dbErr := db.DB.ExecContext(ctx,
			`UPDATE workspaces SET status = 'failed', last_sample_error = $2, updated_at = now() WHERE id = $1`,
			workspaceID, err.Error()); dbErr != nil {
			log.Printf("Provisioner: failed to mark workspace %s as failed after mutator error: %v", workspaceID, dbErr)
		}
		return
	}

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

	// Issue/rotate the workspace auth token and inject the plaintext into the
	// config volume so the workspace always has a valid bearer credential on
	// disk, even after a container rebuild wiped the volume (issue #418).
	//
	// We must rotate (revoke-then-issue) rather than reuse because the DB only
	// stores sha256(plaintext) — we cannot reconstruct the original token to
	// write it back. The new plaintext is written into /configs/.auth_token via
	// WriteFilesToContainer, which runs immediately after ContainerStart and
	// wins the race against the Python adapter's startup time (~1-2 s).
	h.issueAndInjectToken(ctx, workspaceID, &cfg)

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

// issueAndInjectToken rotates the workspace auth token and injects the
// plaintext into cfg.ConfigFiles[".auth_token"] so it is written into the
// /configs volume by WriteFilesToContainer immediately after the container
// starts (issue #418: container rebuild wipes /configs/.auth_token).
//
// Rotation strategy: since the DB only stores sha256(plaintext) we can never
// recover an existing token. We revoke all live tokens first and issue a
// fresh one. On any error the injection is skipped and a warning is logged;
// provisioning continues — the workspace will get 401 on its first heartbeat
// and can recover on the next restart.
func (h *WorkspaceHandler) issueAndInjectToken(ctx context.Context, workspaceID string, cfg *provisioner.WorkspaceConfig) {
	// Revoke any existing live tokens. If this fails we bail out rather than
	// issuing a second live token whose plaintext we can't also deliver.
	if err := wsauth.RevokeAllForWorkspace(ctx, db.DB, workspaceID); err != nil {
		log.Printf("Provisioner: failed to revoke existing tokens for %s: %v — skipping auth-token injection", workspaceID, err)
		return
	}

	token, err := wsauth.IssueToken(ctx, db.DB, workspaceID)
	if err != nil {
		log.Printf("Provisioner: failed to issue auth token for %s: %v — skipping auth-token injection", workspaceID, err)
		return
	}

	if cfg.ConfigFiles == nil {
		cfg.ConfigFiles = make(map[string][]byte)
	}
	cfg.ConfigFiles[".auth_token"] = []byte(token)
	log.Printf("Provisioner: injected fresh auth token for workspace %s into config volume", workspaceID)
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

// resolveOrgTemplate looks for a matching role directory under
// configsDir/org-templates/ and returns the absolute path and a short label
// ("org-templates/<dir>"). Used by the restart handler's rebuild_config path
// (#239) so a workspace can recover from a destroyed config volume without
// admin intervention.
// Returns ("", "") when no match is found.
func resolveOrgTemplate(configsDir, wsName string) (path, label string) {
	orgDir := filepath.Join(configsDir, "org-templates")
	match := findTemplateByName(orgDir, wsName)
	if match == "" {
		return "", ""
	}
	full := filepath.Join(orgDir, match)
	if _, err := os.Stat(full); err != nil {
		return "", ""
	}
	return full, "org-templates/" + match
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

// knownRuntimes is the allowlist of runtime strings the provisioner will
// accept. Unknown values are coerced to the default ("langgraph") instead
// of being splatted into filepath.Join + config.yaml templating, which
// closes both the YAML-injection vector (#241) where an attacker could
// smuggle `initial_prompt: run id && curl …` through a crafted runtime
// string, and the path-traversal oracle where `runtime: ../../sensitive`
// probed host directories for existence.
//
// Keep in sync with workspace/build-all.sh — adding a new
// runtime means bumping both this list and the Docker image tags.
var knownRuntimes = map[string]struct{}{
	"langgraph":   {},
	"claude-code": {},
	"openclaw":    {},
	"crewai":      {},
	"autogen":     {},
	"deepagents":  {},
	"hermes":      {},
	"codex":       {},
}

// yamlQuote emits a YAML double-quoted scalar that safely contains any
// input string. Newlines + carriage returns are stripped first so we
// never need the multi-line block form, and fmt.Sprintf %q produces a
// Go-syntax quoted string whose escape rules are a strict subset of
// YAML's double-quoted scalar — colons, hashes, braces, and every other
// YAML metacharacter are safe inside it.
//
// Empty input → `""` (explicit empty scalar) which YAML readers accept
// cleanly; the alternative of emitting raw %s could leak a trailing
// newline from a prior line if the caller forgot a \n separator.
func yamlQuote(s string) string {
	clean := strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), "\r", "")
	return fmt.Sprintf("%q", clean)
}

// sanitizeRuntime coerces a payload runtime string to a known entry.
// Empty strings → the default. Unknown strings also → the default,
// with a log so operators can notice typos or attack attempts.
func sanitizeRuntime(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "langgraph"
	}
	if _, ok := knownRuntimes[raw]; ok {
		return raw
	}
	log.Printf("provisioner: rejected unknown runtime %q, falling back to langgraph", raw)
	return "langgraph"
}

// ensureDefaultConfig generates minimal config files in memory for workspaces without a template.
// Returns a map of filename → content to be written into the container's /configs volume.
func (h *WorkspaceHandler) ensureDefaultConfig(workspaceID string, payload models.CreateWorkspacePayload) map[string][]byte {
	files := make(map[string][]byte)

	// Determine runtime — pass through the allowlist so an attacker
	// can't smuggle `initial_prompt: …` or a path-traversal oracle
	// via a crafted runtime string (#241).
	runtime := sanitizeRuntime(payload.Runtime)

	// Generate a minimal config.yaml
	model := payload.Model
	if model == "" {
		if runtime == "claude-code" {
			model = "sonnet"
		} else {
			model = "anthropic:claude-opus-4-7"
		}
	}

	// Sanitize name/role/model for YAML safety — always double-quote so
	// a crafted value with a newline or colon can't terminate the scalar
	// and inject an arbitrary key into the generated config. runtime is
	// already allowlisted above so it does not need quoting.
	//
	// Pattern: strip newlines (unrepresentable in a double-quoted YAML
	// scalar without escaping), then emit via %q which produces a Go-
	// syntax quoted string — valid YAML double-quoted scalar because
	// the character sets overlap for this field-value shape.
	quoteName := yamlQuote(payload.Name)
	quoteRole := yamlQuote(payload.Role)
	quoteModel := yamlQuote(model)
	configYAML := fmt.Sprintf("name: %s\ndescription: %s\nversion: 1.0.0\ntier: %d\nruntime: %s\n",
		quoteName, quoteRole, payload.Tier, runtime)

	// Model always at top level — config.py reads raw["model"] for all runtimes.
	configYAML += fmt.Sprintf("model: %s\n", quoteModel)

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

// loadWorkspaceSecrets loads global + workspace-specific secrets into a map.
// Returns nil map + error string on any query, iteration, or decrypt failure.
// Shared by both Docker and control plane provisioning paths to avoid
// duplication. Query errors abort loading — an agent provisioned without its
// secrets fails at task time with opaque auth errors.
func loadWorkspaceSecrets(ctx context.Context, workspaceID string) (map[string]string, string) {
	envVars := map[string]string{}
	globalRows, globalErr := db.DB.QueryContext(ctx,
		`SELECT key, encrypted_value, encryption_version FROM global_secrets`)
	if globalErr != nil {
		return nil, fmt.Sprintf("global secrets query failed: %v", globalErr)
	}
	defer globalRows.Close()
	for globalRows.Next() {
		var k string
		var v []byte
		var ver int
		if globalRows.Scan(&k, &v, &ver) == nil {
			decrypted, decErr := crypto.DecryptVersioned(v, ver)
			if decErr != nil {
				return nil, fmt.Sprintf("cannot decrypt global secret %s: %v", k, decErr)
			}
			envVars[k] = string(decrypted)
		}
	}
	if iterErr := globalRows.Err(); iterErr != nil {
		return nil, fmt.Sprintf("global secrets iteration failed: %v", iterErr)
	}
	wsRows, err := db.DB.QueryContext(ctx,
		`SELECT key, encrypted_value, encryption_version FROM workspace_secrets WHERE workspace_id = $1`, workspaceID)
	if err != nil {
		return nil, fmt.Sprintf("workspace secrets query failed: %v", err)
	}
	defer wsRows.Close()
	for wsRows.Next() {
		var k string
		var v []byte
		var ver int
		if wsRows.Scan(&k, &v, &ver) == nil {
			decrypted, decErr := crypto.DecryptVersioned(v, ver)
			if decErr != nil {
				return nil, fmt.Sprintf("cannot decrypt workspace secret %s: %v", k, decErr)
			}
			envVars[k] = string(decrypted)
		}
	}
	if iterErr := wsRows.Err(); iterErr != nil {
		return nil, fmt.Sprintf("workspace secrets iteration failed: %v", iterErr)
	}
	return envVars, ""
}

// provisionWorkspaceCP provisions a workspace via the control plane API.
func (h *WorkspaceHandler) provisionWorkspaceCP(workspaceID, templatePath string, configFiles map[string][]byte, payload models.CreateWorkspacePayload) {
	ctx, cancel := context.WithTimeout(context.Background(), provisioner.ProvisionTimeout)
	defer cancel()

	envVars, decryptErr := loadWorkspaceSecrets(ctx, workspaceID)
	if decryptErr != "" {
		log.Printf("CPProvisioner: %s for %s", decryptErr, workspaceID)
		db.DB.ExecContext(ctx, `UPDATE workspaces SET status = 'failed', last_sample_error = $2, updated_at = now() WHERE id = $1`,
			workspaceID, decryptErr)
		return
	}

	applyAgentGitIdentity(envVars, payload.Name)
	if err := h.envMutators.Run(ctx, workspaceID, envVars); err != nil {
		log.Printf("CPProvisioner: env mutator failed for %s: %v", workspaceID, err)
		db.DB.ExecContext(ctx, `UPDATE workspaces SET status = 'failed', last_sample_error = $2, updated_at = now() WHERE id = $1`,
			workspaceID, err.Error())
		return
	}

	cfg := provisioner.WorkspaceConfig{
		WorkspaceID: workspaceID,
		Tier:        payload.Tier,
		Runtime:     payload.Runtime,
		EnvVars:     envVars,
		PlatformURL: h.platformURL,
	}

	machineID, err := h.cpProv.Start(ctx, cfg)
	if err != nil {
		log.Printf("CPProvisioner: failed to start workspace %s: %v", workspaceID, err)
		h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_PROVISION_FAILED", workspaceID, map[string]interface{}{
			"error": err.Error(),
		})
		db.DB.ExecContext(ctx, `UPDATE workspaces SET status = 'failed', last_sample_error = $2, updated_at = now() WHERE id = $1`,
			workspaceID, err.Error())
		return
	}

	log.Printf("CPProvisioner: workspace %s started as machine %s via control plane", workspaceID, machineID)
	// Issue token so the agent can authenticate on boot
	token, tokenErr := wsauth.IssueToken(ctx, db.DB, workspaceID)
	if tokenErr != nil {
		log.Printf("CPProvisioner: failed to issue token for %s: %v", workspaceID, tokenErr)
	} else {
		log.Printf("CPProvisioner: issued auth token for workspace %s (prefix: %s...)", workspaceID, token[:8])
	}
}
