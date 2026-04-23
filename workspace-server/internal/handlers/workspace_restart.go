package handlers

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/models"
	"github.com/gin-gonic/gin"
)

// restartMu prevents concurrent RestartByID calls for the same workspace
var restartMu sync.Map // map[workspaceID]*sync.Mutex

// isParentPaused checks if any ancestor of the workspace is paused.
func isParentPaused(ctx context.Context, workspaceID string) (bool, string) {
	var parentID *string
	db.DB.QueryRowContext(ctx, `SELECT parent_id FROM workspaces WHERE id = $1`, workspaceID).Scan(&parentID)
	if parentID == nil {
		return false, ""
	}
	var parentStatus, parentName string
	err := db.DB.QueryRowContext(ctx,
		`SELECT status, name FROM workspaces WHERE id = $1`, *parentID,
	).Scan(&parentStatus, &parentName)
	if err != nil {
		return false, ""
	}
	if parentStatus == "paused" {
		return true, parentName
	}
	// Check grandparent recursively
	return isParentPaused(ctx, *parentID)
}

// Restart handles POST /workspaces/:id/restart
// Works for offline, failed, or degraded workspaces. Stops any existing container, then re-provisions.
func (h *WorkspaceHandler) Restart(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	var status, wsName, dbRuntime string
	var tier int
	err := db.DB.QueryRowContext(ctx,
		`SELECT status, name, tier, COALESCE(runtime, 'langgraph') FROM workspaces WHERE id = $1`, id,
	).Scan(&status, &wsName, &tier, &dbRuntime)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lookup failed"})
		return
	}
	// Block restart if any ancestor is paused — must resume parent first
	if paused, parentName := isParentPaused(ctx, id); paused {
		c.JSON(http.StatusConflict, gin.H{"error": "parent workspace \"" + parentName + "\" is paused — resume it first"})
		return
	}

	// SaaS mode: cpProv handles workspace EC2 lifecycle. Self-hosted mode:
	// provisioner handles local Docker containers. At least one must be
	// available — previously only `provisioner` was checked, which broke
	// restart entirely on every SaaS tenant (the workspace EC2 couldn't
	// be terminated + relaunched, the endpoint 503'd on every try).
	if h.provisioner == nil && h.cpProv == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "provisioner not available"})
		return
	}

	// Read runtime from container's config.yaml before stopping. Docker-
	// only: in SaaS mode the workspace runs on a remote EC2 and we can't
	// exec into it, so we trust the DB value (user updates runtime via
	// the Config tab which writes through to both the DB and the container).
	containerRuntime := dbRuntime
	if h.provisioner != nil {
		containerName := configDirName(id) // ws-{id[:12]}
		if cfgBytes, readErr := h.provisioner.ExecRead(ctx, containerName, "/configs/config.yaml"); readErr == nil {
			for _, line := range strings.Split(string(cfgBytes), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "runtime:") {
					parsed := strings.TrimSpace(strings.TrimPrefix(line, "runtime:"))
					if parsed != "" && parsed != containerRuntime {
						log.Printf("Restart: runtime changed in config.yaml %q→%q for %s", containerRuntime, parsed, wsName)
						containerRuntime = parsed
						db.DB.ExecContext(ctx, `UPDATE workspaces SET runtime = $1 WHERE id = $2`, containerRuntime, id)
					}
					break
				}
			}
		}
	}

	// Stop existing container / terminate existing EC2. Pick the matching
	// stop path. CPProvisioner.Stop calls DELETE /cp/workspaces/:id to
	// terminate the workspace EC2; the subsequent provision call launches
	// a fresh one with the latest secrets + config.
	if h.provisioner != nil {
		_ = h.provisioner.Stop(ctx, id)
	} else if h.cpProv != nil {
		if err := h.cpProv.Stop(ctx, id); err != nil {
			log.Printf("Restart: cpProv.Stop(%s) failed: %v (continuing to reprovision)", id, err)
		}
	}

	// Reset to provisioning
	db.DB.ExecContext(ctx,
		`UPDATE workspaces SET status = 'provisioning', url = '', updated_at = now() WHERE id = $1`, id)
	h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_PROVISIONING", id, map[string]interface{}{
		"name": wsName,
		"tier": tier,
	})

	// Read template from request body or try to find matching config
	var body struct {
		Template      string `json:"template"`
		ApplyTemplate bool   `json:"apply_template"` // force re-apply runtime-default template (e.g. after runtime change)
		Reset         bool   `json:"reset"`          // #12: discard claude-sessions volume before restart
		RebuildConfig bool   `json:"rebuild_config"` // #239: re-render config volume from org-template source (recovery path when volume was destroyed)
	}
	// REAL-BUG fix: previously a malformed JSON body silently produced zero values,
	// which could mask a caller bug (e.g. apply_template:true sent as a string)
	// and corrupt restart state. Reject obviously broken bodies with 400 instead.
	// An entirely empty body remains valid (all fields are optional and zero values
	// are the documented defaults — ShouldBindJSON returns io.EOF for empty bodies
	// which we tolerate).
	if err := c.ShouldBindJSON(&body); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Resolve template path in priority order:
	// 1. Explicit template from request body
	// 2. Runtime-specific default template (e.g. claude-code-default/)
	// 3. Name-based match in templates directory
	// 4. No template — the volume already has configs from previous run
	var templatePath string
	var configFiles map[string][]byte
	configLabel := "existing-volume"

	template := body.Template
	if template == "" {
		template = findTemplateByName(h.configsDir, wsName)
	}
	if template != "" {
		candidatePath, resolveErr := resolveInsideRoot(h.configsDir, template)
		if resolveErr != nil {
			log.Printf("Restart: invalid template %q: %v — proceeding without it", template, resolveErr)
		} else if _, err := os.Stat(candidatePath); err == nil {
			templatePath = candidatePath
			configLabel = template
		} else {
			log.Printf("Restart: template %q dir not found — proceeding without it", template)
		}
	}

	// #239: rebuild_config=true — try org-templates as last-resort source so a
	// workspace with a destroyed config volume can self-recover without admin
	// intervention. Only fires when no other template was resolved above.
	if templatePath == "" && body.RebuildConfig {
		if p, label := resolveOrgTemplate(h.configsDir, wsName); p != "" {
			templatePath = p
			configLabel = label
			log.Printf("Restart: rebuild_config — using org-template %s for %s (%s)", label, wsName, id)
		}
	}

	if templatePath == "" {
		log.Printf("Restart: reusing existing config volume for %s (%s)", wsName, id)
	} else {
		log.Printf("Restart: using template %s for %s (%s)", templatePath, wsName, id)
	}

	payload := models.CreateWorkspacePayload{Name: wsName, Tier: tier, Runtime: containerRuntime}
	log.Printf("Restart: workspace %s (%s) runtime=%q", wsName, id, containerRuntime)

	// Apply runtime-default template ONLY when explicitly requested via "apply_template": true.
	// Use case: runtime was changed via Config tab — need new runtime's base files.
	// Normal restarts preserve existing config volume (user's model, skills, prompts).
	if templatePath == "" && body.ApplyTemplate && dbRuntime != "" {
		runtimeTemplate := filepath.Join(h.configsDir, dbRuntime+"-default")
		if _, err := os.Stat(runtimeTemplate); err == nil {
			templatePath = runtimeTemplate
			configLabel = dbRuntime + "-default"
			log.Printf("Restart: applying template %s (runtime change)", configLabel)
		}
	}

	// #12: ?reset=true (or body.Reset) discards the claude-sessions volume
	// before restart, giving the agent a clean /root/.claude/sessions dir.
	resetClaudeSession := c.Query("reset") == "true" || body.Reset
	if resetClaudeSession {
		log.Printf("Restart: reset=true — will discard claude-sessions volume for %s (%s)", wsName, id)
	}

	// Capture restart-context data BEFORE provisionWorkspaceOpts flips
	// last_heartbeat_at with the new session. Issue #19 Layer 1.
	restartData := loadRestartContextData(ctx, id)

	// Dispatch to the correct provisioner. provisionWorkspaceOpts is the
	// Docker path; provisionWorkspaceCP is the SaaS path. The Create
	// handler already branches this way; Restart now mirrors it.
	if h.cpProv != nil {
		go h.provisionWorkspaceCP(id, templatePath, configFiles, payload)
	} else {
		go h.provisionWorkspaceOpts(id, templatePath, configFiles, payload, resetClaudeSession)
	}
	go h.sendRestartContext(id, restartData)

	c.JSON(http.StatusOK, gin.H{"status": "provisioning", "config_dir": configLabel, "reset_session": resetClaudeSession})
}

// Hibernate handles POST /workspaces/:id/hibernate
// Manually puts a running workspace into hibernation — useful for immediate
// cost savings without waiting for the idle timer. The workspace auto-wakes
// on the next incoming A2A message/send.
func (h *WorkspaceHandler) Hibernate(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	var wsName string
	var tier, activeTasks int
	err := db.DB.QueryRowContext(ctx,
		`SELECT name, tier, active_tasks FROM workspaces WHERE id = $1 AND status IN ('online', 'degraded')`, id,
	).Scan(&wsName, &tier, &activeTasks)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found or not in a hibernatable state (must be online or degraded)"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lookup failed"})
		return
	}

	// #822: reject hibernation when active tasks are in flight unless caller
	// passes ?force=true. Prevents operator from accidentally killing a
	// mid-task agent.
	if activeTasks > 0 && c.Query("force") != "true" {
		c.JSON(http.StatusConflict, gin.H{
			"error":        "workspace has active tasks; use ?force=true to terminate them",
			"active_tasks": activeTasks,
		})
		return
	}
	if activeTasks > 0 {
		log.Printf("[WARN] force-hibernating workspace %s (%s) with %d active tasks", id, wsName, activeTasks)
	}

	h.HibernateWorkspace(ctx, id)
	c.JSON(http.StatusOK, gin.H{"status": "hibernated"})
}

// HibernateWorkspace stops the container and sets the workspace status to
// 'hibernated'. Called by the hibernation monitor when a workspace has had
// active_tasks == 0 for longer than its configured hibernation_idle_minutes.
// Hibernated workspaces auto-wake on the next incoming A2A message.
//
// TOCTOU safety (#819): the three-step pattern below is atomic at the DB level.
//
//  1. Atomic claim: a single UPDATE WHERE locks the row by transitioning
//     status → 'hibernating', gated on status IN ('online','degraded') AND
//     active_tasks = 0.  If any concurrent caller (another goroutine, the
//     idle-timer, or a manual API call) already claimed the row, or if tasks
//     arrived since the caller decided to hibernate, rowsAffected == 0 and
//     this function returns immediately without stopping anything.
//
//  2. provisioner.Stop: safe to call now because status == 'hibernating';
//     the routing layer rejects new tasks for non-online/degraded workspaces,
//     so no new task can be dispatched between step 1 and step 2.
//
//  3. Final UPDATE to 'hibernated': records the completed hibernation.
func (h *WorkspaceHandler) HibernateWorkspace(ctx context.Context, workspaceID string) {
	// ── Step 1: Atomic claim ──────────────────────────────────────────────────
	// The UPDATE acts as a DB-level advisory lock: only one concurrent caller
	// can transition the row from online/degraded → hibernating.  The
	// active_tasks = 0 predicate ensures we never interrupt a running task.
	result, err := db.DB.ExecContext(ctx, `
		UPDATE workspaces
		SET    status = 'hibernating', updated_at = now()
		WHERE  id = $1
		  AND  status IN ('online', 'degraded')
		  AND  active_tasks = 0`, workspaceID)
	if err != nil {
		log.Printf("Hibernate: atomic claim failed for %s: %v", workspaceID, err)
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// Either already hibernating/hibernated/paused/removed, or active_tasks > 0 —
		// safe to abort without side-effects.
		return
	}

	// Fetch name/tier for logging and event broadcast (after the claim, so we
	// can use a simple SELECT without a status guard).
	var wsName string
	var tier int
	if scanErr := db.DB.QueryRowContext(ctx,
		`SELECT name, tier FROM workspaces WHERE id = $1`, workspaceID,
	).Scan(&wsName, &tier); scanErr != nil {
		wsName = workspaceID // fallback for log messages
	}

	// ── Step 2: Stop the container ────────────────────────────────────────────
	// Status is now 'hibernating'; the router rejects new task routing here, so
	// there is no race window between claiming the row and stopping the container.
	log.Printf("Hibernate: stopping container for %s (%s)", wsName, workspaceID)
	if h.stopFnOverride != nil {
		h.stopFnOverride(ctx, workspaceID)
	} else if h.provisioner != nil {
		_ = h.provisioner.Stop(ctx, workspaceID)
	}

	// ── Step 3: Mark fully hibernated ─────────────────────────────────────────
	if _, err = db.DB.ExecContext(ctx,
		`UPDATE workspaces SET status = 'hibernated', url = '', updated_at = now() WHERE id = $1`,
		workspaceID); err != nil {
		log.Printf("Hibernate: failed to mark hibernated for %s: %v", workspaceID, err)
		return
	}

	db.ClearWorkspaceKeys(ctx, workspaceID)
	h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_HIBERNATED", workspaceID, map[string]interface{}{
		"name": wsName,
		"tier": tier,
	})
	log.Printf("Hibernate: workspace %s (%s) is now hibernated", wsName, workspaceID)
}

// RestartByID restarts a workspace by ID — for programmatic use (e.g., auto-restart after secret change).
func (h *WorkspaceHandler) RestartByID(workspaceID string) {
	if h.provisioner == nil {
		return
	}

	// Per-workspace mutex — skip if already restarting (last-write-wins)
	mu, _ := restartMu.LoadOrStore(workspaceID, &sync.Mutex{})
	wsMu := mu.(*sync.Mutex)
	if !wsMu.TryLock() {
		log.Printf("Auto-restart: skipping %s — restart already in progress", workspaceID)
		return
	}
	defer wsMu.Unlock()

	ctx := context.Background()

	var wsName, status, dbRuntime string
	var tier int
	err := db.DB.QueryRowContext(ctx,
		`SELECT name, status, tier, COALESCE(runtime, 'langgraph') FROM workspaces WHERE id = $1 AND status NOT IN ('removed', 'paused', 'hibernated')`, workspaceID,
	).Scan(&wsName, &status, &tier, &dbRuntime)
	if err != nil {
		return // includes paused/hibernated — don't auto-restart those
	}

	// Don't auto-restart external workspaces (no Docker container)
	if dbRuntime == "external" {
		return
	}

	// Don't auto-restart if any ancestor is paused
	if paused, _ := isParentPaused(ctx, workspaceID); paused {
		return
	}

	// If still provisioning, brief wait so container exists for Stop()
	if status == "provisioning" {
		log.Printf("Auto-restart: interrupting provisioning for %s (%s)", wsName, workspaceID)
		time.Sleep(10 * time.Second)
	}

	log.Printf("Auto-restart: restarting %s (%s) runtime=%q (was: %s)", wsName, workspaceID, dbRuntime, status)

	_ = h.provisioner.Stop(ctx, workspaceID)

	db.DB.ExecContext(ctx,
		`UPDATE workspaces SET status = 'provisioning', url = '', updated_at = now() WHERE id = $1`, workspaceID)
	h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_PROVISIONING", workspaceID, map[string]interface{}{
		"name": wsName, "tier": tier,
	})

	// Runtime from DB — no more config file parsing
	payload := models.CreateWorkspacePayload{Name: wsName, Tier: tier, Runtime: dbRuntime}

	// Snapshot restart-context data before the new session overwrites
	// last_heartbeat_at. Issue #19 Layer 1.
	restartData := loadRestartContextData(ctx, workspaceID)

	// On auto-restart, do NOT re-apply templates — preserve existing config volume.
	go h.provisionWorkspace(workspaceID, "", nil, payload)
	go h.sendRestartContext(workspaceID, restartData)
}

// Pause handles POST /workspaces/:id/pause
// Stops the container and sets status to 'paused'. The workspace remains on the canvas
// but won't receive heartbeats, won't be auto-restarted, and won't consume resources.
// Config volume is preserved — resume will re-provision with the same config.
func (h *WorkspaceHandler) Pause(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	var status, wsName string
	err := db.DB.QueryRowContext(ctx,
		`SELECT status, name FROM workspaces WHERE id = $1 AND status NOT IN ('removed', 'paused')`, id,
	).Scan(&status, &wsName)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found or already paused"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lookup failed"})
		return
	}

	// Collect this workspace + all descendants to pause
	toPause := []struct{ id, name string }{{id, wsName}}
	rows, _ := db.DB.QueryContext(ctx,
		`WITH RECURSIVE descendants AS (
			SELECT id, name FROM workspaces WHERE parent_id = $1 AND status NOT IN ('removed', 'paused')
			UNION ALL
			SELECT w.id, w.name FROM workspaces w JOIN descendants d ON w.parent_id = d.id WHERE w.status NOT IN ('removed', 'paused')
		) SELECT id, name FROM descendants`, id)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var cid, cname string
			if rows.Scan(&cid, &cname) == nil {
				toPause = append(toPause, struct{ id, name string }{cid, cname})
			}
		}
	}

	// Stop containers and mark all as paused
	for _, ws := range toPause {
		if h.provisioner != nil {
			_ = h.provisioner.Stop(ctx, ws.id)
		}
		db.DB.ExecContext(ctx,
			`UPDATE workspaces SET status = 'paused', url = '', updated_at = now() WHERE id = $1`, ws.id)
		db.ClearWorkspaceKeys(ctx, ws.id)
		h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_PAUSED", ws.id, map[string]interface{}{
			"name": ws.name,
		})
	}

	log.Printf("Paused workspace %s (%s) + %d children", wsName, id, len(toPause)-1)
	c.JSON(http.StatusOK, gin.H{"status": "paused", "paused_count": len(toPause)})
}

// Resume handles POST /workspaces/:id/resume
// Re-provisions a paused workspace. Config volume is preserved from before the pause.
func (h *WorkspaceHandler) Resume(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	var wsName, dbRuntime string
	var tier int
	err := db.DB.QueryRowContext(ctx,
		`SELECT name, tier, COALESCE(runtime, 'langgraph') FROM workspaces WHERE id = $1 AND status = 'paused'`, id,
	).Scan(&wsName, &tier, &dbRuntime)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found or not paused"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lookup failed"})
		return
	}

	// Accept either provisioner (Docker self-hosted OR CP SaaS). See the
	// same guard in Restart above for context — Resume previously 503'd
	// on every SaaS tenant.
	if h.provisioner == nil && h.cpProv == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "provisioner not available"})
		return
	}

	// Block resume if any ancestor is still paused — must resume from the top down
	if paused, parentName := isParentPaused(ctx, id); paused {
		c.JSON(http.StatusConflict, gin.H{"error": "parent workspace \"" + parentName + "\" is paused — resume it first"})
		return
	}

	// Collect this workspace + all paused descendants to resume
	type wsInfo struct {
		id, name, runtime string
		tier              int
	}
	toResume := []wsInfo{{id, wsName, dbRuntime, tier}}
	rows, _ := db.DB.QueryContext(ctx,
		`WITH RECURSIVE descendants AS (
			SELECT id, name, tier, COALESCE(runtime, 'langgraph') AS runtime FROM workspaces WHERE parent_id = $1 AND status = 'paused'
			UNION ALL
			SELECT w.id, w.name, w.tier, COALESCE(w.runtime, 'langgraph') FROM workspaces w JOIN descendants d ON w.parent_id = d.id WHERE w.status = 'paused'
		) SELECT id, name, tier, runtime FROM descendants`, id)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var ws wsInfo
			if rows.Scan(&ws.id, &ws.name, &ws.tier, &ws.runtime) == nil {
				toResume = append(toResume, ws)
			}
		}
	}

	// Re-provision all
	for _, ws := range toResume {
		db.DB.ExecContext(ctx,
			`UPDATE workspaces SET status = 'provisioning', updated_at = now() WHERE id = $1`, ws.id)
		h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_PROVISIONING", ws.id, map[string]interface{}{
			"name": ws.name, "tier": ws.tier,
		})
		payload := models.CreateWorkspacePayload{Name: ws.name, Tier: ws.tier, Runtime: ws.runtime}
		// Dispatch to the matching provisioner (mirrors the Create +
		// Restart branching). SaaS tenants use cpProv; self-hosted Docker
		// uses provisioner via provisionWorkspaceOpts.
		if h.cpProv != nil {
			go h.provisionWorkspaceCP(ws.id, "", nil, payload)
		} else {
			go h.provisionWorkspace(ws.id, "", nil, payload)
		}
	}

	log.Printf("Resuming workspace %s (%s) + %d children", wsName, id, len(toResume)-1)
	c.JSON(http.StatusOK, gin.H{"status": "provisioning", "resumed_count": len(toResume)})
}
