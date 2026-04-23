package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/events"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/provisioner"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type TeamHandler struct {
	broadcaster *events.Broadcaster
	provisioner *provisioner.Provisioner
	platformURL string
	configsDir  string
}

func NewTeamHandler(b *events.Broadcaster, p *provisioner.Provisioner, platformURL, configsDir string) *TeamHandler {
	return &TeamHandler{
		broadcaster: b,
		provisioner: p,
		platformURL: platformURL,
		configsDir:  configsDir,
	}
}

// Expand handles POST /workspaces/:id/expand
// Reads sub_workspaces from the workspace's config and provisions child workspaces.
func (h *TeamHandler) Expand(c *gin.Context) {
	parentID := c.Param("id")
	ctx := c.Request.Context()

	// Verify workspace exists and is online
	var name string
	var tier int
	var status string
	err := db.DB.QueryRowContext(ctx,
		`SELECT name, tier, status FROM workspaces WHERE id = $1`, parentID,
	).Scan(&name, &tier, &status)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lookup failed"})
		return
	}

	// Find the workspace's config to get sub_workspaces
	templateDir := findTemplateDirByName(h.configsDir, name)
	if templateDir == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no config found for workspace"})
		return
	}

	configData, err := os.ReadFile(filepath.Join(templateDir, "config.yaml"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read config"})
		return
	}

	var config struct {
		SubWorkspaces []struct {
			Config string `yaml:"config"`
			Name   string `yaml:"name"`
			Role   string `yaml:"role"`
		} `yaml:"sub_workspaces"`
	}
	if err := yaml.Unmarshal(configData, &config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse config"})
		return
	}

	if len(config.SubWorkspaces) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workspace has no sub_workspaces defined in config"})
		return
	}

	// Create child workspaces
	children := make([]map[string]interface{}, 0)
	for _, sub := range config.SubWorkspaces {
		childID := uuid.New().String()
		childName := sub.Name
		if childName == "" {
			childName = sub.Config
		}

		_, err := db.DB.ExecContext(ctx, `
			INSERT INTO workspaces (id, name, role, tier, status, parent_id)
			VALUES ($1, $2, $3, $4, 'provisioning', $5)
		`, childID, childName, nilStr(sub.Role), tier, parentID)
		if err != nil {
			log.Printf("Expand: failed to create child %s: %v", childName, err)
			continue
		}

		// Insert canvas layout (offset from parent)
		if _, err := db.DB.ExecContext(ctx, `
			INSERT INTO canvas_layouts (workspace_id, x, y) VALUES ($1, $2, $3)
		`, childID, 0, 0); err != nil {
			log.Printf("Team expand: failed to insert layout for child %s: %v", childID, err)
		}

		h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_PROVISIONING", childID, map[string]interface{}{
			"name":      childName,
			"tier":      tier,
			"parent_id": parentID,
		})

		// Provision if template exists
		if h.provisioner != nil && sub.Config != "" {
			templatePath := filepath.Join(h.configsDir, sub.Config)
			if _, err := os.Stat(templatePath); err == nil {
				pluginsPath, _ := filepath.Abs(filepath.Join(h.configsDir, "..", "plugins"))
				go func(wID, tPath, pPath string, t int) {
					provCtx, cancel := context.WithTimeout(context.Background(), provisioner.ProvisionTimeout)
					defer cancel()
					cfg := provisioner.WorkspaceConfig{
						WorkspaceID:  wID,
						TemplatePath: tPath,
						PluginsPath:  pPath,
						Tier:         t,
						EnvVars:      map[string]string{"PARENT_ID": parentID},
						PlatformURL:  h.platformURL,
					}
					if _, err := h.provisioner.Start(provCtx, cfg); err != nil {
						log.Printf("Expand: provision failed for %s: %v", wID, err)
					}
				}(childID, templatePath, pluginsPath, tier)
			}
		}

		children = append(children, map[string]interface{}{
			"id":   childID,
			"name": childName,
			"role": sub.Role,
		})
	}

	// Mark parent as expanded
	h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_EXPANDED", parentID, map[string]interface{}{
		"children": children,
	})

	c.JSON(http.StatusOK, gin.H{
		"status":   "expanded",
		"children": children,
	})
}

// Collapse handles POST /workspaces/:id/collapse
// Stops and removes all child workspaces.
func (h *TeamHandler) Collapse(c *gin.Context) {
	parentID := c.Param("id")
	ctx := c.Request.Context()

	// Find children
	rows, err := db.DB.QueryContext(ctx,
		`SELECT id, name FROM workspaces WHERE parent_id = $1 AND status != 'removed'`, parentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query children"})
		return
	}
	defer rows.Close()

	removed := make([]string, 0)
	for rows.Next() {
		var childID, childName string
		if rows.Scan(&childID, &childName) != nil {
			continue
		}

		// Stop container if provisioner available.
		// best-effort: cleanup path, container may already be gone or unreachable.
		if h.provisioner != nil {
			_ = h.provisioner.Stop(ctx, childID)
		}

		// Mark as removed
		if _, err := db.DB.ExecContext(ctx,
			`UPDATE workspaces SET status = 'removed', updated_at = now() WHERE id = $1`, childID); err != nil {
			log.Printf("Team collapse: failed to remove workspace %s: %v", childID, err)
		}
		if _, err := db.DB.ExecContext(ctx,
			`DELETE FROM canvas_layouts WHERE workspace_id = $1`, childID); err != nil {
			log.Printf("Team collapse: failed to delete layout for %s: %v", childID, err)
		}

		h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_REMOVED", childID, map[string]interface{}{})

		removed = append(removed, childName)
	}

	h.broadcaster.RecordAndBroadcast(ctx, "WORKSPACE_COLLAPSED", parentID, map[string]interface{}{
		"removed_children": removed,
	})

	c.JSON(http.StatusOK, gin.H{
		"status":  "collapsed",
		"removed": removed,
	})
}

func nilStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func findTemplateDirByName(configsDir, name string) string {
	normalized := normalizeName(name)

	candidate := filepath.Join(configsDir, normalized)
	if _, err := os.Stat(filepath.Join(candidate, "config.yaml")); err == nil {
		return candidate
	}

	// Fall back to scanning all dirs
	entries, err := os.ReadDir(configsDir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		cfgPath := filepath.Join(configsDir, e.Name(), "config.yaml")
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			continue
		}
		var cfg struct {
			Name string `yaml:"name"`
		}
		if json.Unmarshal(data, &cfg) == nil && cfg.Name == name {
			return filepath.Join(configsDir, e.Name())
		}
		// Try yaml unmarshal too
		if yaml.Unmarshal(data, &cfg) == nil && cfg.Name == name {
			return filepath.Join(configsDir, e.Name())
		}
	}
	return ""
}
