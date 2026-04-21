package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/gin-gonic/gin"
)

// AdminMemoriesHandler provides bulk export/import of agent memories for
// backup and restore across Docker rebuilds (issue #1051).
type AdminMemoriesHandler struct{}

// NewAdminMemoriesHandler constructs the handler.
func NewAdminMemoriesHandler() *AdminMemoriesHandler {
	return &AdminMemoriesHandler{}
}

// memoryExportEntry is the JSON shape for a single exported memory.
type memoryExportEntry struct {
	ID            string    `json:"id"`
	Content       string    `json:"content"`
	Scope         string    `json:"scope"`
	Namespace     string    `json:"namespace"`
	CreatedAt     time.Time `json:"created_at"`
	WorkspaceName string    `json:"workspace_name"`
}

// Export handles GET /admin/memories/export
// Returns all agent memories joined with workspace name so the dump is
// human-readable and can be re-imported after workspaces are re-provisioned
// (UUIDs change, names stay stable).
func (h *AdminMemoriesHandler) Export(c *gin.Context) {
	ctx := c.Request.Context()

	rows, err := db.DB.QueryContext(ctx, `
		SELECT am.id, am.content, am.scope, am.namespace, am.created_at,
		       w.name AS workspace_name
		FROM agent_memories am
		JOIN workspaces w ON am.workspace_id = w.id
		ORDER BY am.created_at
	`)
	if err != nil {
		log.Printf("admin/memories/export: query error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "export query failed"})
		return
	}
	defer rows.Close()

	memories := make([]memoryExportEntry, 0)
	for rows.Next() {
		var m memoryExportEntry
		if err := rows.Scan(&m.ID, &m.Content, &m.Scope, &m.Namespace, &m.CreatedAt, &m.WorkspaceName); err != nil {
			log.Printf("admin/memories/export: scan error: %v", err)
			continue
		}
		memories = append(memories, m)
	}
	if err := rows.Err(); err != nil {
		log.Printf("admin/memories/export: rows error: %v", err)
	}

	c.JSON(http.StatusOK, memories)
}

// memoryImportEntry is the JSON shape accepted on import. Matches export format.
type memoryImportEntry struct {
	Content       string `json:"content"`
	Scope         string `json:"scope"`
	Namespace     string `json:"namespace"`
	CreatedAt     string `json:"created_at"` // RFC3339 string, preserved on insert
	WorkspaceName string `json:"workspace_name"`
}

// Import handles POST /admin/memories/import
// Accepts a JSON array of memories (same format as export). Matches each
// workspace by name (not UUID). Skips duplicates where workspace_id + content
// + scope already exist. Returns counts of imported and skipped entries.
func (h *AdminMemoriesHandler) Import(c *gin.Context) {
	ctx := c.Request.Context()

	var entries []memoryImportEntry
	if err := c.ShouldBindJSON(&entries); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON: " + err.Error()})
		return
	}

	imported := 0
	skipped := 0
	errors := 0

	for _, entry := range entries {
		// 1. Resolve workspace by name
		var workspaceID string
		err := db.DB.QueryRowContext(ctx,
			`SELECT id FROM workspaces WHERE name = $1 LIMIT 1`,
			entry.WorkspaceName,
		).Scan(&workspaceID)
		if err != nil {
			log.Printf("admin/memories/import: workspace %q not found, skipping", entry.WorkspaceName)
			skipped++
			continue
		}

		// 2. Check for duplicate (same workspace + content + scope)
		var exists bool
		err = db.DB.QueryRowContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM agent_memories WHERE workspace_id = $1 AND content = $2 AND scope = $3)`,
			workspaceID, entry.Content, entry.Scope,
		).Scan(&exists)
		if err != nil {
			log.Printf("admin/memories/import: duplicate check error for workspace %q: %v", entry.WorkspaceName, err)
			errors++
			continue
		}
		if exists {
			skipped++
			continue
		}

		// 3. Insert the memory, preserving original created_at if provided
		namespace := entry.Namespace
		if namespace == "" {
			namespace = "general"
		}

		if entry.CreatedAt != "" {
			_, err = db.DB.ExecContext(ctx,
				`INSERT INTO agent_memories (workspace_id, content, scope, namespace, created_at) VALUES ($1, $2, $3, $4, $5)`,
				workspaceID, entry.Content, entry.Scope, namespace, entry.CreatedAt,
			)
		} else {
			_, err = db.DB.ExecContext(ctx,
				`INSERT INTO agent_memories (workspace_id, content, scope, namespace) VALUES ($1, $2, $3, $4)`,
				workspaceID, entry.Content, entry.Scope, namespace,
			)
		}
		if err != nil {
			log.Printf("admin/memories/import: insert error for workspace %q: %v", entry.WorkspaceName, err)
			errors++
			continue
		}
		imported++
	}

	c.JSON(http.StatusOK, gin.H{
		"imported": imported,
		"skipped":  skipped,
		"errors":   errors,
		"total":    len(entries),
	})
}
