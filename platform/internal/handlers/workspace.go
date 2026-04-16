package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/events"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/middleware"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/models"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/provisioner"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/wsauth"
	"github.com/Molecule-AI/molecule-monorepo/platform/pkg/provisionhook"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/google/uuid"
)

type WorkspaceHandler struct {
	broadcaster *events.Broadcaster
	provisioner *provisioner.Provisioner
	cpProv      *provisioner.CPProvisioner
	platformURL string
	configsDir  string // path to workspace-configs-templates/ (for reading templates)
	// envMutators runs registered EnvMutator plugins right before
	// container Start, after built-in secret loads. Nil = no plugins
	// registered; Registry.Run handles a nil receiver as a no-op so the
	// hot path stays a single nil-pointer compare.
	envMutators *provisionhook.Registry
}

func NewWorkspaceHandler(b *events.Broadcaster, p *provisioner.Provisioner, platformURL, configsDir string) *WorkspaceHandler {
	return &WorkspaceHandler{
		broadcaster: b,
		provisioner: p,
		platformURL: platformURL,
		configsDir:  configsDir,
	}
}

// SetCPProvisioner wires the control plane provisioner for SaaS tenants.
// Auto-activated when MOLECULE_ORG_ID is set (no manual config needed).
func (h *WorkspaceHandler) SetCPProvisioner(cp *provisioner.CPProvisioner) {
	h.cpProv = cp
}

// SetEnvMutators wires a provisionhook.Registry into the handler. Plugins
// living in separate repos register on the same Registry instance during
// boot (see cmd/server/main.go) and main.go calls this setter once before
// router.Setup. Re-callable for tests but not safe under concurrent
// provisions — only invoke during single-threaded init.
func (h *WorkspaceHandler) SetEnvMutators(r *provisionhook.Registry) {
	h.envMutators = r
}

// Create handles POST /workspaces
func (h *WorkspaceHandler) Create(c *gin.Context) {
	var payload models.CreateWorkspacePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := uuid.New().String()
	awarenessNamespace := workspaceAwarenessNamespace(id)
	if payload.Tier == 0 {
		payload.Tier = 1
	}

	// Detect runtime from template config.yaml if not specified in request.
	// Must happen before DB insert so the correct runtime is persisted.
	if payload.Runtime == "" && payload.Template != "" {
		// #226: payload.Template is attacker-controllable. resolveInsideRoot
		// rejects absolute paths and any ".." that escapes configsDir so the
		// provisioner can't be pointed at host directories.
		candidatePath, resolveErr := resolveInsideRoot(h.configsDir, payload.Template)
		if resolveErr != nil {
			log.Printf("Create: invalid template path %q: %v", payload.Template, resolveErr)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template"})
			return
		}
		cfgData, readErr := os.ReadFile(filepath.Join(candidatePath, "config.yaml"))
		if readErr != nil {
			log.Printf("Create: could not read config.yaml for template %q: %v", payload.Template, readErr)
		}
		for _, line := range strings.Split(string(cfgData), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "runtime:") {
				payload.Runtime = strings.TrimSpace(strings.TrimPrefix(line, "runtime:"))
				break
			}
		}
	}
	if payload.Runtime == "" {
		payload.Runtime = "langgraph"
	}

	ctx := c.Request.Context()

	// Convert empty role to NULL
	var role interface{}
	if payload.Role != "" {
		role = payload.Role
	}

	// Validate and convert workspace_dir
	var workspaceDir interface{}
	if payload.WorkspaceDir != "" {
		if err := validateWorkspaceDir(payload.WorkspaceDir); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		workspaceDir = payload.WorkspaceDir
	}

	// #65: validate workspace_access, default to "none".
	workspaceAccess := payload.WorkspaceAccess
	if workspaceAccess == "" {
		workspaceAccess = provisioner.WorkspaceAccessNone
	}
	if err := provisioner.ValidateWorkspaceAccess(workspaceAccess, payload.WorkspaceDir); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Insert workspace with runtime persisted in DB
	_, err := db.DB.ExecContext(ctx, `
		INSERT INTO workspaces (id, name, role, tier, runtime, awareness_namespace, status, parent_id, workspace_dir, workspace_access)
		VALUES ($1, $2, $3, $4, $5, $6, 'provisioning', $7, $8, $9)
	`, id, payload.Name, role, payload.Tier, payload.Runtime, awarenessNamespace, payload.ParentID, workspaceDir, workspaceAccess)
	if err != nil {
		log.Printf("Create workspace error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create workspace"})
		return
	}

	// Insert canvas layout — non-fatal: workspace can be dragged into position later
	if _, err := db.DB.ExecContext(ctx, `
		INSERT INTO canvas_layouts (workspace_id, x, y) VALUES ($1, $2, $3)
	`, id, payload.Canvas.X, payload.Canvas.Y); err != nil {
		log.Printf("Create: canvas layout insert failed for %s (workspace will appear at 0,0): %v", id, err)
	}

	// Broadcast provisioning event
	h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_PROVISIONING", id, map[string]interface{}{
		"name": payload.Name,
		"tier": payload.Tier,
	})

	// External workspaces: no container provisioning — just set the URL and mark online
	if payload.External {
		if payload.URL != "" {
			db.DB.ExecContext(ctx, `UPDATE workspaces SET url = $1, status = 'online', updated_at = now() WHERE id = $2`, payload.URL, id)
			if err := db.CacheURL(ctx, id, payload.URL); err != nil {
				log.Printf("External workspace: failed to cache URL for %s: %v", id, err)
			}
		} else {
			db.DB.ExecContext(ctx, `UPDATE workspaces SET status = 'online', updated_at = now() WHERE id = $1`, id)
		}
		h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_ONLINE", id, map[string]interface{}{
			"name": payload.Name, "external": true,
		})
		log.Printf("Created external workspace %s (%s) at %s", payload.Name, id, payload.URL)
		c.JSON(http.StatusCreated, gin.H{
			"id":       id,
			"status":   "online",
			"external": true,
		})
		return
	}

	// Resolve template config — needed for both Docker provisioning and
	// config-only persistence (tenant SaaS without Docker).
	var templatePath string
	var configFiles map[string][]byte
	if payload.Template != "" {
		candidatePath, resolveErr := resolveInsideRoot(h.configsDir, payload.Template)
		if resolveErr != nil {
			log.Printf("Create provision: rejecting template %q: %v", payload.Template, resolveErr)
			return
		}
		if _, err := os.Stat(candidatePath); err == nil {
			templatePath = candidatePath
		} else {
			log.Printf("Create: template %q not found, falling back for %s", payload.Template, payload.Name)
			safeRuntime := sanitizeRuntime(payload.Runtime)
			runtimeDefault := filepath.Join(h.configsDir, safeRuntime+"-default")
			if _, err := os.Stat(runtimeDefault); err == nil {
				templatePath = runtimeDefault
			} else {
				configFiles = h.ensureDefaultConfig(id, payload)
			}
		}
	} else {
		configFiles = h.ensureDefaultConfig(id, payload)
	}

	// Auto-provision — pick backend: control plane (SaaS) or Docker (self-hosted)
	if h.cpProv != nil {
		go h.provisionWorkspaceCP(id, templatePath, configFiles, payload)
	} else if h.provisioner != nil {
		go h.provisionWorkspace(id, templatePath, configFiles, payload)
	} else {
		// No Docker available (SaaS tenant). Persist basic config as JSON
		// so the Config tab shows the correct runtime/model/name. Then mark
		// the workspace as failed with a clear message.
		cfgJSON := fmt.Sprintf(`{"name":%q,"runtime":%q,"tier":%d,"template":%q}`,
			payload.Name, payload.Runtime, payload.Tier, payload.Template)
		db.DB.ExecContext(ctx, `
			INSERT INTO workspace_config (workspace_id, data) VALUES ($1, $2::jsonb)
			ON CONFLICT (workspace_id) DO UPDATE SET data = $2::jsonb
		`, id, cfgJSON)
		db.DB.ExecContext(ctx,
			`UPDATE workspaces SET status = 'failed', last_sample_error = 'Docker not available — workspace containers require a Docker daemon or external provisioning.', updated_at = now() WHERE id = $1`, id)
		h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_PROVISION_FAILED", id, map[string]interface{}{
			"error": "Docker not available on this platform instance",
		})
		log.Printf("Create: no Docker daemon — workspace %s config persisted, marked failed", id)
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":                  id,
		"status":              "provisioning",
		"awareness_namespace": awarenessNamespace,
		"workspace_access":    workspaceAccess,
	})
}

// scanWorkspaceRow is a helper to scan workspace+layout rows into a clean JSON map.
func scanWorkspaceRow(rows interface {
	Scan(dest ...interface{}) error
}) (map[string]interface{}, error) {
	var id, name, role, status, url, sampleError, currentTask, runtime, workspaceDir string
	var tier, activeTasks, uptimeSeconds int
	var errorRate, x, y float64
	var collapsed bool
	var parentID *string
	var agentCard []byte

	err := rows.Scan(&id, &name, &role, &tier, &status, &agentCard, &url,
		&parentID, &activeTasks, &errorRate, &sampleError, &uptimeSeconds,
		&currentTask, &runtime, &workspaceDir, &x, &y, &collapsed)
	if err != nil {
		return nil, err
	}

	ws := map[string]interface{}{
		"id":                id,
		"name":              name,
		"tier":              tier,
		"status":            status,
		"url":               url,
		"parent_id":         parentID,
		"active_tasks":      activeTasks,
		"last_error_rate":   errorRate,
		"last_sample_error": sampleError,
		"uptime_seconds":    uptimeSeconds,
		"current_task":      currentTask,
		"runtime":           runtime,
		"workspace_dir":     nilIfEmpty(workspaceDir),
		"x":                 x,
		"y":                 y,
		"collapsed":         collapsed,
	}

	// Only include non-empty values
	if role != "" {
		ws["role"] = role
	} else {
		ws["role"] = nil
	}

	// Parse agent_card as raw JSON
	if len(agentCard) > 0 && string(agentCard) != "null" {
		ws["agent_card"] = json.RawMessage(agentCard)
	} else {
		ws["agent_card"] = nil
	}

	return ws, nil
}

const workspaceListQuery = `
	SELECT w.id, w.name, COALESCE(w.role, ''), w.tier, w.status,
		   COALESCE(w.agent_card, 'null'::jsonb), COALESCE(w.url, ''),
		   w.parent_id, w.active_tasks, w.last_error_rate,
		   COALESCE(w.last_sample_error, ''), w.uptime_seconds,
		   COALESCE(w.current_task, ''), COALESCE(w.runtime, 'langgraph'),
		   COALESCE(w.workspace_dir, ''),
		   COALESCE(cl.x, 0), COALESCE(cl.y, 0), COALESCE(cl.collapsed, false)
	FROM workspaces w
	LEFT JOIN canvas_layouts cl ON cl.workspace_id = w.id
	WHERE w.status != 'removed'
	ORDER BY w.created_at`

// List handles GET /workspaces
func (h *WorkspaceHandler) List(c *gin.Context) {
	rows, err := db.DB.QueryContext(c.Request.Context(), workspaceListQuery)
	if err != nil {
		log.Printf("List workspaces error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()

	workspaces := make([]map[string]interface{}, 0)
	for rows.Next() {
		ws, err := scanWorkspaceRow(rows)
		if err != nil {
			log.Printf("List scan error: %v", err)
			continue
		}
		workspaces = append(workspaces, ws)
	}
	if err := rows.Err(); err != nil {
		log.Printf("List rows error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query iteration failed"})
		return
	}

	c.JSON(http.StatusOK, workspaces)
}

// Get handles GET /workspaces/:id
func (h *WorkspaceHandler) Get(c *gin.Context) {
	id := c.Param("id")

	row := db.DB.QueryRowContext(c.Request.Context(), `
		SELECT w.id, w.name, COALESCE(w.role, ''), w.tier, w.status,
			   COALESCE(w.agent_card, 'null'::jsonb), COALESCE(w.url, ''),
			   w.parent_id, w.active_tasks, w.last_error_rate,
			   COALESCE(w.last_sample_error, ''), w.uptime_seconds,
			   COALESCE(w.current_task, ''), COALESCE(w.runtime, 'langgraph'),
			   COALESCE(w.workspace_dir, ''),
			   COALESCE(cl.x, 0), COALESCE(cl.y, 0), COALESCE(cl.collapsed, false)
		FROM workspaces w
		LEFT JOIN canvas_layouts cl ON cl.workspace_id = w.id
		WHERE w.id = $1
	`, id)

	ws, err := scanWorkspaceRow(row)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found"})
		return
	}
	if err != nil {
		log.Printf("Get workspace error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}

	c.JSON(http.StatusOK, ws)
}

// State handles GET /workspaces/:id/state — minimal status payload for
// remote-agent polling (Phase 30.4). Returns `{status, paused, deleted,
// workspace_id}` so a remote agent can detect pause/resume/delete
// without needing WebSocket reachability from the platform.
//
// Auth: Phase 30.1 bearer token required when the workspace has any
// live token on file; legacy workspaces grandfathered. Uses the same
// fail-closed posture as secrets.Values — polling this cadence with
// unauth'd callers would be a trivial DoS / workspace-status-scanner
// otherwise.
//
// The endpoint is deliberately NOT merged with GET /workspaces/:id:
// that handler is optimized for canvas (returns config, agent_card,
// position, …) and is unauthenticated by design. State is the
// agent-machinery polling path — tight, token-gated, cache-friendly.
func (h *WorkspaceHandler) State(c *gin.Context) {
	workspaceID := c.Param("id")
	ctx := c.Request.Context()

	// Auth gate — same shape as secrets.Values (Phase 30.2). Fail-closed
	// on DB errors because the caller is about to poll this at ~60s
	// cadence; letting unauth'd callers through on a hiccup turns this
	// into a workspace-status scanner.
	hasLive, hlErr := wsauth.HasAnyLiveToken(ctx, db.DB, workspaceID)
	if hlErr != nil {
		log.Printf("wsauth: HasAnyLiveToken(%s) failed for workspace.State: %v", workspaceID, hlErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "auth check failed"})
		return
	}
	if hasLive {
		tok := wsauth.BearerTokenFromHeader(c.GetHeader("Authorization"))
		if tok == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing workspace auth token"})
			return
		}
		if err := wsauth.ValidateToken(ctx, db.DB, workspaceID, tok); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid workspace auth token"})
			return
		}
	}

	var status string
	err := db.DB.QueryRowContext(ctx, `
		SELECT status
		FROM workspaces
		WHERE id = $1
	`, workspaceID).Scan(&status)
	if err == sql.ErrNoRows {
		// A deleted workspace row no longer exists — remote agent should
		// interpret 404 as "shut yourself down" (our pause path uses
		// status='removed' but keeps the row; a 404 here means the
		// workspace was hard-deleted out from under the agent).
		c.JSON(http.StatusNotFound, gin.H{
			"workspace_id": workspaceID,
			"deleted":      true,
		})
		return
	}
	if err != nil {
		log.Printf("workspace.State query error for %s: %v", workspaceID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}

	// Two delete paths: hard-delete (sql.ErrNoRows above → 404) AND
	// soft-delete (status='removed' → also return 404 here so the SDK
	// doesn't have to remember "is it 200 with deleted=true OR 404 with
	// deleted=true?"). Same shape, same status code, same flag set.
	if status == "removed" {
		c.JSON(http.StatusNotFound, gin.H{
			"workspace_id": workspaceID,
			"status":       "removed",
			"deleted":      true,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"workspace_id": workspaceID,
		"status":       status,
		"paused":       status == "paused",
		"deleted":      false,
	})
}

// sensitiveUpdateFields gates the #120/#138 field-level auth check inside
// Update. Any key in this set requires a valid bearer token even when the
// rest of the route is open — tier is a resource-escalation vector,
// parent_id rewrites the A2A hierarchy, runtime swaps the container image
// on next restart, workspace_dir redirects host bind-mounts. Cosmetic
// fields (name, role, x, y, canvas) do not appear here and pass through
// unauthenticated so canvas drag-reposition and inline rename keep working.
var sensitiveUpdateFields = map[string]struct{}{
	"tier":          {},
	"parent_id":     {},
	"runtime":       {},
	"workspace_dir": {},
}

// Update handles PATCH /workspaces/:id
func (h *WorkspaceHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// #138 field-level authz: PATCH /workspaces/:id is on the open router so
	// canvas drag-reposition (cookie-based, no bearer token) keeps working,
	// BUT the sensitive fields below require a valid bearer via the usual
	// admin-token check. Lazy-bootstrap: if no live admin tokens exist at all
	// (fresh install) the check is a no-op and everyone passes through.
	for field := range body {
		if _, sensitive := sensitiveUpdateFields[field]; !sensitive {
			continue
		}
		hasLive, hlErr := wsauth.HasAnyLiveTokenGlobal(ctx, db.DB)
		if hlErr != nil {
			log.Printf("wsauth: Update HasAnyLiveTokenGlobal failed: %v — allowing request", hlErr)
			break
		}
		if !hasLive {
			break // fresh install — fail-open
		}
		tok := wsauth.BearerTokenFromHeader(c.GetHeader("Authorization"))
		if tok == "" {
			if middleware.IsSameOriginCanvas(c) {
				break // tenant canvas — trusted same-origin
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": "admin auth required for field: " + field})
			return
		}
		if err := wsauth.ValidateAnyToken(ctx, db.DB, tok); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid admin auth token"})
			return
		}
		break // one successful validation covers the whole body
	}

	// #120: guard — return 404 for nonexistent workspace IDs instead of
	// silently applying zero-row UPDATEs and returning 200.
	var exists bool
	if err := db.DB.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM workspaces WHERE id = $1)`, id,
	).Scan(&exists); err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found"})
		return
	}

	if name, ok := body["name"]; ok {
		if _, err := db.DB.ExecContext(ctx, `UPDATE workspaces SET name = $2, updated_at = now() WHERE id = $1`, id, name); err != nil {
			log.Printf("Update name error for %s: %v", id, err)
		}
	}
	if role, ok := body["role"]; ok {
		if _, err := db.DB.ExecContext(ctx, `UPDATE workspaces SET role = $2, updated_at = now() WHERE id = $1`, id, role); err != nil {
			log.Printf("Update role error for %s: %v", id, err)
		}
	}
	if tier, ok := body["tier"]; ok {
		if _, err := db.DB.ExecContext(ctx, `UPDATE workspaces SET tier = $2, updated_at = now() WHERE id = $1`, id, tier); err != nil {
			log.Printf("Update tier error for %s: %v", id, err)
		}
	}
	if parentID, ok := body["parent_id"]; ok {
		if _, err := db.DB.ExecContext(ctx, `UPDATE workspaces SET parent_id = $2, updated_at = now() WHERE id = $1`, id, parentID); err != nil {
			log.Printf("Update parent_id error for %s: %v", id, err)
		}
	}
	if runtime, ok := body["runtime"]; ok {
		if _, err := db.DB.ExecContext(ctx, `UPDATE workspaces SET runtime = $2, updated_at = now() WHERE id = $1`, id, runtime); err != nil {
			log.Printf("Update runtime error for %s: %v", id, err)
		}
	}
	needsRestart := false
	if wsDir, ok := body["workspace_dir"]; ok {
		// Allow null to clear workspace_dir
		if wsDir != nil {
			if dirStr, isStr := wsDir.(string); isStr && dirStr != "" {
				if err := validateWorkspaceDir(dirStr); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
			}
		}
		if _, err := db.DB.ExecContext(ctx, `UPDATE workspaces SET workspace_dir = $2, updated_at = now() WHERE id = $1`, id, wsDir); err != nil {
			log.Printf("Update workspace_dir error for %s: %v", id, err)
		}
		needsRestart = true
	}

	// Update canvas position if both x and y provided
	if x, xOk := body["x"]; xOk {
		if y, yOk := body["y"]; yOk {
			if _, err := db.DB.ExecContext(ctx, `
				INSERT INTO canvas_layouts (workspace_id, x, y)
				VALUES ($1, $2, $3)
				ON CONFLICT (workspace_id) DO UPDATE SET x = EXCLUDED.x, y = EXCLUDED.y
			`, id, x, y); err != nil {
				log.Printf("Update position error for %s: %v", id, err)
			}
		}
	}

	resp := gin.H{"status": "updated"}
	if needsRestart {
		resp["needs_restart"] = true
	}
	c.JSON(http.StatusOK, resp)
}

// validateWorkspaceDir checks that a workspace_dir path is safe to bind-mount.
func validateWorkspaceDir(dir string) error {
	if !filepath.IsAbs(dir) {
		return fmt.Errorf("workspace_dir must be an absolute path")
	}
	if strings.Contains(dir, "..") {
		return fmt.Errorf("workspace_dir must not contain '..'")
	}
	// Reject system-critical paths
	clean := filepath.Clean(dir)
	for _, blocked := range []string{"/etc", "/var", "/proc", "/sys", "/dev", "/boot", "/sbin", "/bin", "/lib", "/usr"} {
		if clean == blocked || strings.HasPrefix(clean, blocked+"/") {
			return fmt.Errorf("workspace_dir must not be a system path (%s)", blocked)
		}
	}
	return nil
}

// Delete handles DELETE /workspaces/:id
// If the workspace has children (is a team), cascade deletes all sub-workspaces.
// Use ?confirm=true to actually delete (otherwise returns children list for confirmation).
func (h *WorkspaceHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()
	confirm := c.Query("confirm") == "true"

	// Check for children
	rows, err := db.DB.QueryContext(ctx,
		`SELECT id, name FROM workspaces WHERE parent_id = $1 AND status != 'removed'`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check children"})
		return
	}
	defer rows.Close()

	var children []map[string]string
	for rows.Next() {
		var childID, childName string
		if rows.Scan(&childID, &childName) == nil {
			children = append(children, map[string]string{"id": childID, "name": childName})
		}
	}
	if err := rows.Err(); err != nil {
		log.Printf("Delete: child rows error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check children"})
		return
	}

	// If has children and not confirmed, return children list for confirmation.
	// Uses HTTP 409 Conflict (not 200) so `curl --fail`, `fetch().ok`, and any
	// client that treats HTTP 4xx as an error surfaces the confirmation
	// requirement. Body shape unchanged so the canvas UI's parser keeps
	// working. Fixes #88.
	if len(children) > 0 && !confirm {
		c.JSON(http.StatusConflict, gin.H{
			"status":         "confirmation_required",
			"message":        "This workspace has sub-workspaces. Delete with ?confirm=true to cascade delete.",
			"children":       children,
			"children_count": len(children),
		})
		return
	}

	// Cascade delete: collect ALL descendants (not just direct children) via
	// recursive CTE, then stop each container and remove each volume.
	// Previous bug: only direct children's containers were stopped, leaving
	// grandchildren as orphan running containers after a cascade delete.
	descendantIDs := []string{}
	if len(children) > 0 {
		descRows, err := db.DB.QueryContext(ctx, `
			WITH RECURSIVE descendants AS (
				SELECT id FROM workspaces WHERE parent_id = $1 AND status != 'removed'
				UNION ALL
				SELECT w.id FROM workspaces w JOIN descendants d ON w.parent_id = d.id WHERE w.status != 'removed'
			)
			SELECT id FROM descendants
		`, id)
		if err != nil {
			log.Printf("Delete: descendant query error for %s: %v", id, err)
		} else {
			for descRows.Next() {
				var descID string
				if descRows.Scan(&descID) == nil {
					descendantIDs = append(descendantIDs, descID)
				}
			}
			descRows.Close()
		}
	}

	// #73 fix: mark rows 'removed' in the DB FIRST, BEFORE stopping containers
	// or removing volumes. Previously the sequence was stop → update-status,
	// which left a gap where:
	//   - the container's last pre-teardown heartbeat could resurrect the row
	//     via the register-handler UPSERT (now also guarded in #73)
	//   - the liveness monitor could observe 'online' status + expired Redis
	//     TTL and trigger RestartByID, recreating a container we're trying
	//     to destroy
	// Marking 'removed' first makes both of those paths no-op via their
	// existing `status NOT IN ('removed', ...)` guards.
	allIDs := append([]string{id}, descendantIDs...)
	if _, err := db.DB.ExecContext(ctx,
		`UPDATE workspaces SET status = 'removed', updated_at = now() WHERE id = ANY($1::uuid[])`,
		pq.Array(allIDs)); err != nil {
		log.Printf("Delete status update error for %s: %v", id, err)
	}
	if _, err := db.DB.ExecContext(ctx,
		`DELETE FROM canvas_layouts WHERE workspace_id = ANY($1::uuid[])`,
		pq.Array(allIDs)); err != nil {
		log.Printf("Delete canvas_layouts error for %s: %v", id, err)
	}
	// Revoke all auth tokens for the deleted workspaces. Once the workspace is
	// gone its tokens are meaningless; leaving them alive would keep
	// HasAnyLiveTokenGlobal = true even after the platform is otherwise empty,
	// which prevents AdminAuth from returning to fail-open and breaks the E2E
	// test's count-zero assertion (and local re-run cleanup).
	if _, err := db.DB.ExecContext(ctx,
		`UPDATE workspace_auth_tokens SET revoked_at = now()
		 WHERE workspace_id = ANY($1::uuid[]) AND revoked_at IS NULL`,
		pq.Array(allIDs)); err != nil {
		log.Printf("Delete token revocation error for %s: %v", id, err)
	}

	// Now stop containers + remove volumes for all descendants (any depth).
	// Any concurrent heartbeat / registration / liveness-triggered restart
	// will see status='removed' and bail out early.
	for _, descID := range descendantIDs {
		if h.provisioner != nil {
			h.provisioner.Stop(ctx, descID)
			if err := h.provisioner.RemoveVolume(ctx, descID); err != nil {
				log.Printf("Delete descendant %s volume removal warning: %v", descID, err)
			}
		}
		db.ClearWorkspaceKeys(ctx, descID)
		h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_REMOVED", descID, map[string]interface{}{})
	}

	// Stop + remove volume for the workspace itself
	if h.provisioner != nil {
		h.provisioner.Stop(ctx, id)
		if err := h.provisioner.RemoveVolume(ctx, id); err != nil {
			log.Printf("Delete %s volume removal warning: %v", id, err)
		}
	}
	db.ClearWorkspaceKeys(ctx, id)

	h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_REMOVED", id, map[string]interface{}{
		"cascade_deleted": len(descendantIDs),
	})

	c.JSON(http.StatusOK, gin.H{"status": "removed", "cascade_deleted": len(descendantIDs)})
}
