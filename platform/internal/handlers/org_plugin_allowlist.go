package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/gin-gonic/gin"
)

// resolveOrgID returns the effective org ID for a workspace: the parent_id
// when one exists, or the workspace's own ID when it is the org root.
// Returns an empty string if the workspace is not found.
func resolveOrgID(ctx context.Context, workspaceID string) (string, error) {
	var parentID sql.NullString
	err := db.DB.QueryRowContext(ctx,
		`SELECT parent_id FROM workspaces WHERE id = $1`,
		workspaceID,
	).Scan(&parentID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if parentID.Valid && parentID.String != "" {
		return parentID.String, nil
	}
	return workspaceID, nil
}

// checkOrgPluginAllowlist returns (true, reason) when the plugin is blocked
// by the org's allowlist, or (false, "") when the install is permitted.
//
// Semantics:
//   - No allowlist rows for this org → allow-all (backward compat).
//   - Allowlist exists and plugin is on it → allowed.
//   - Allowlist exists and plugin is NOT on it → blocked (403).
//   - DB errors → fail-open with a log (don't block installs on DB hiccup).
func checkOrgPluginAllowlist(ctx context.Context, workspaceID, pluginName string) (blocked bool, reason string) {
	orgID, err := resolveOrgID(ctx, workspaceID)
	if err != nil {
		log.Printf("allowlist: resolveOrgID(%s) failed: %v — allowing install", workspaceID, err)
		return false, ""
	}
	if orgID == "" {
		return false, "" // workspace not found; let later checks handle it
	}

	var allowed bool
	err = db.DB.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM org_plugin_allowlist
			WHERE org_id = $1 AND plugin_name = $2
		)
	`, orgID, pluginName).Scan(&allowed)
	if err != nil {
		log.Printf("allowlist: existence check failed (org=%s plugin=%s): %v — allowing install", orgID, pluginName, err)
		return false, ""
	}
	if allowed {
		return false, "" // explicitly on the allowlist
	}

	// Check whether an allowlist exists at all. Empty allowlist = allow-all.
	var count int
	if err := db.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM org_plugin_allowlist WHERE org_id = $1`,
		orgID,
	).Scan(&count); err != nil {
		log.Printf("allowlist: count check failed (org=%s): %v — allowing install", orgID, err)
		return false, ""
	}
	if count == 0 {
		return false, "" // no allowlist configured — allow-all
	}

	return true, fmt.Sprintf("plugin %q is not in the org allowlist", pluginName)
}

// OrgPluginAllowlistHandler manages the per-org plugin governance registry.
type OrgPluginAllowlistHandler struct{}

// NewOrgPluginAllowlistHandler constructs an OrgPluginAllowlistHandler.
func NewOrgPluginAllowlistHandler() *OrgPluginAllowlistHandler {
	return &OrgPluginAllowlistHandler{}
}

// allowlistEntry is the JSON shape for a single allowlist record.
type allowlistEntry struct {
	PluginName string    `json:"plugin_name"`
	EnabledBy  string    `json:"enabled_by"`
	EnabledAt  time.Time `json:"enabled_at"`
}

// putAllowlistRequest is the request body for PUT /orgs/:id/plugins/allowlist.
// Plugins holds the complete desired allowlist; the handler replaces the
// current entries atomically. An empty slice clears the allowlist (allow-all).
type putAllowlistRequest struct {
	Plugins   []string `json:"plugins"`
	EnabledBy string   `json:"enabled_by"` // workspace ID of the admin performing the change
}

// GetAllowlist handles GET /orgs/:id/plugins/allowlist.
//
// Returns the current allowlist for the org workspace identified by :id.
// An empty array means no allowlist is configured (allow-all). Auth: AdminAuth.
func (h *OrgPluginAllowlistHandler) GetAllowlist(c *gin.Context) {
	orgID := c.Param("id")
	ctx := c.Request.Context()

	// Verify the org workspace exists.
	var exists bool
	if err := db.DB.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM workspaces WHERE id = $1)`,
		orgID,
	).Scan(&exists); err != nil {
		log.Printf("allowlist: org check failed for %s: %v", orgID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify org"})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "org not found"})
		return
	}

	rows, err := db.DB.QueryContext(ctx, `
		SELECT plugin_name, enabled_by, enabled_at
		FROM org_plugin_allowlist
		WHERE org_id = $1
		ORDER BY plugin_name
	`, orgID)
	if err != nil {
		log.Printf("allowlist: query failed for org %s: %v", orgID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch allowlist"})
		return
	}
	defer rows.Close()

	entries := make([]allowlistEntry, 0)
	for rows.Next() {
		var e allowlistEntry
		if err := rows.Scan(&e.PluginName, &e.EnabledBy, &e.EnabledAt); err != nil {
			log.Printf("allowlist: scan error for org %s: %v", orgID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read allowlist"})
			return
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		log.Printf("allowlist: rows error for org %s: %v", orgID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read allowlist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"org_id":   orgID,
		"plugins":  entries,
		"allow_all": len(entries) == 0,
	})
}

// PutAllowlist handles PUT /orgs/:id/plugins/allowlist.
//
// Replaces the org's allowlist atomically with the supplied plugin names.
// Sending an empty plugins array clears the allowlist (reverts to allow-all).
// Auth: AdminAuth.
func (h *OrgPluginAllowlistHandler) PutAllowlist(c *gin.Context) {
	orgID := c.Param("id")
	ctx := c.Request.Context()

	var req putAllowlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.EnabledBy == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "enabled_by is required"})
		return
	}

	// Validate each plugin name for safety before touching the DB.
	for _, name := range req.Plugins {
		if err := validatePluginName(name); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":       "invalid plugin name",
				"plugin_name": name,
				"detail":      err.Error(),
			})
			return
		}
	}

	// Verify the org workspace exists.
	var exists bool
	if err := db.DB.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM workspaces WHERE id = $1)`,
		orgID,
	).Scan(&exists); err != nil {
		log.Printf("allowlist: org check failed for %s: %v", orgID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify org"})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "org not found"})
		return
	}

	// Replace atomically: delete all current entries, then insert the new set.
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("allowlist: begin tx failed for org %s: %v", orgID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer tx.Rollback() //nolint:errcheck // superseded by Commit on success path

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM org_plugin_allowlist WHERE org_id = $1`,
		orgID,
	); err != nil {
		log.Printf("allowlist: delete failed for org %s: %v", orgID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update allowlist"})
		return
	}

	for _, name := range req.Plugins {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO org_plugin_allowlist (org_id, plugin_name, enabled_by)
			VALUES ($1, $2, $3)
			ON CONFLICT (org_id, plugin_name) DO NOTHING
		`, orgID, name, req.EnabledBy); err != nil {
			log.Printf("allowlist: insert %q failed for org %s: %v", name, orgID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update allowlist"})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("allowlist: commit failed for org %s: %v", orgID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit allowlist update"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"org_id":    orgID,
		"plugins":   req.Plugins,
		"allow_all": len(req.Plugins) == 0,
	})
}
