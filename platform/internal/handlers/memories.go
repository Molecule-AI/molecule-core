package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/registry"
	"github.com/gin-gonic/gin"
)

// defaultMemoryNamespace is used when a caller omits the field on POST or
// when querying for memories written before migration 017. Matches the
// column default in platform/migrations/017_memories_fts_namespace.up.sql.
const defaultMemoryNamespace = "general"

// memoryFTSMinQueryLen is the shortest query length that gets Postgres
// full-text search treatment. Anything shorter uses ILIKE because
// tsvector requires at least one token and single characters tokenise
// to nothing in the 'english' config.
const memoryFTSMinQueryLen = 2

type MemoriesHandler struct{}

func NewMemoriesHandler() *MemoriesHandler {
	return &MemoriesHandler{}
}

// Commit handles POST /workspaces/:id/memories
// Stores a memory fact with a scope (LOCAL, TEAM, GLOBAL) and an optional
// namespace (defaults to "general"). Namespaces implement the Holaboss
// knowledge/{facts,procedures,blockers,reference}/ pattern so agents can
// file and recall memories by category.
func (h *MemoriesHandler) Commit(c *gin.Context) {
	workspaceID := c.Param("id")
	ctx := c.Request.Context()

	var body struct {
		Content   string `json:"content" binding:"required"`
		Scope     string `json:"scope" binding:"required"` // LOCAL, TEAM, GLOBAL
		Namespace string `json:"namespace,omitempty"`      // optional; defaults to "general"
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if body.Scope != "LOCAL" && body.Scope != "TEAM" && body.Scope != "GLOBAL" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "scope must be LOCAL, TEAM, or GLOBAL"})
		return
	}

	namespace := body.Namespace
	if namespace == "" {
		namespace = defaultMemoryNamespace
	}
	if len(namespace) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "namespace must be <= 50 characters"})
		return
	}

	// GLOBAL scope: only root workspaces (no parent) can write
	if body.Scope == "GLOBAL" {
		var parentID *string
		db.DB.QueryRowContext(ctx, `SELECT parent_id FROM workspaces WHERE id = $1`, workspaceID).Scan(&parentID)
		if parentID != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "only root workspaces can write GLOBAL memories"})
			return
		}
	}

	var memoryID string
	err := db.DB.QueryRowContext(ctx, `
		INSERT INTO agent_memories (workspace_id, content, scope, namespace)
		VALUES ($1, $2, $3, $4) RETURNING id
	`, workspaceID, body.Content, body.Scope, namespace).Scan(&memoryID)
	if err != nil {
		log.Printf("Commit memory error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store memory"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": memoryID, "scope": body.Scope, "namespace": namespace})
}

// Search handles GET /workspaces/:id/memories
// Searches memories visible to the requesting workspace.
//
// Supports:
//   - ?scope=LOCAL|TEAM|GLOBAL for access-control slicing
//   - ?q=... full-text search (ts_rank ordered) when len>=memoryFTSMinQueryLen;
//     falls back to ILIKE for shorter strings
//   - ?namespace=... additional filter on the Holaboss-style namespace tag
func (h *MemoriesHandler) Search(c *gin.Context) {
	workspaceID := c.Param("id")
	scope := c.DefaultQuery("scope", "")
	query := c.DefaultQuery("q", "")
	namespace := c.DefaultQuery("namespace", "")
	ctx := c.Request.Context()

	// Get workspace info for access control
	var parentID *string
	db.DB.QueryRowContext(ctx, `SELECT parent_id FROM workspaces WHERE id = $1`, workspaceID).Scan(&parentID)

	// Build query based on scope and access rules
	var sqlQuery string
	var args []interface{}

	switch scope {
	case "LOCAL":
		// Only this workspace's memories
		sqlQuery = `SELECT id, workspace_id, content, scope, namespace, created_at FROM agent_memories WHERE workspace_id = $1 AND scope = 'LOCAL'`
		args = []interface{}{workspaceID}

	case "TEAM":
		// Team = self + parent + siblings (same parent_id)
		if parentID != nil {
			// Child workspace: team is parent + siblings sharing same parent_id
			sqlQuery = `SELECT m.id, m.workspace_id, m.content, m.scope, m.namespace, m.created_at
				FROM agent_memories m
				JOIN workspaces w ON w.id = m.workspace_id
				WHERE m.scope = 'TEAM' AND w.status != 'removed'
				AND (w.parent_id = $1 OR w.id = $1)`
			args = []interface{}{*parentID}
		} else {
			// Root workspace: team is self + direct children only
			sqlQuery = `SELECT m.id, m.workspace_id, m.content, m.scope, m.namespace, m.created_at
				FROM agent_memories m
				JOIN workspaces w ON w.id = m.workspace_id
				WHERE m.scope = 'TEAM' AND w.status != 'removed'
				AND (w.parent_id = $1 OR w.id = $1)`
			args = []interface{}{workspaceID}
		}

	case "GLOBAL":
		// All GLOBAL memories (readable by everyone)
		sqlQuery = `SELECT id, workspace_id, content, scope, namespace, created_at FROM agent_memories WHERE scope = 'GLOBAL'`
		args = []interface{}{}

	default:
		// All accessible memories
		sqlQuery = `SELECT id, workspace_id, content, scope, namespace, created_at FROM agent_memories WHERE workspace_id = $1`
		args = []interface{}{workspaceID}
	}

	// Namespace filter (optional) — applies regardless of scope.
	if namespace != "" {
		sqlQuery += ` AND namespace = ` + nextArg(len(args))
		args = append(args, namespace)
	}

	// Text search: FTS with ts_rank ordering for multi-char queries,
	// ILIKE fallback for 1-char and empty-after-tokenization edge cases.
	// ILIKE path is preserved as the secondary ORDER BY tie-breaker is
	// still created_at DESC so empty-tsvector rows don't leak to the top.
	ftsActive := false
	if len(query) >= memoryFTSMinQueryLen {
		sqlQuery += ` AND content_tsv @@ plainto_tsquery('english', ` + nextArg(len(args)) + `)`
		args = append(args, query)
		ftsActive = true
	} else if query != "" {
		sqlQuery += ` AND content ILIKE ` + nextArg(len(args))
		args = append(args, "%"+query+"%")
	}

	if ftsActive {
		// Rank FTS hits first, tie-break by recency.
		sqlQuery += ` ORDER BY ts_rank(content_tsv, plainto_tsquery('english', ` + nextArg(len(args)) + `)) DESC, created_at DESC`
		args = append(args, query)
	} else {
		sqlQuery += ` ORDER BY created_at DESC`
	}
	sqlQuery += ` LIMIT 50`

	rows, err := db.DB.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		log.Printf("Search memories error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
		return
	}
	defer rows.Close()

	memories := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id, wsID, content, memScope, memNS, createdAt string
		if rows.Scan(&id, &wsID, &content, &memScope, &memNS, &createdAt) != nil {
			continue
		}

		// Access control check for TEAM scope
		if memScope == "TEAM" && wsID != workspaceID {
			if !registry.CanCommunicate(workspaceID, wsID) {
				continue // Skip memories from workspaces we can't reach
			}
		}

		memories = append(memories, map[string]interface{}{
			"id":           id,
			"workspace_id": wsID,
			"content":      content,
			"scope":        memScope,
			"namespace":    memNS,
			"created_at":   createdAt,
		})
	}

	c.JSON(http.StatusOK, memories)
}

// Delete handles DELETE /workspaces/:id/memories/:memoryId
func (h *MemoriesHandler) Delete(c *gin.Context) {
	workspaceID := c.Param("id")
	memoryID := c.Param("memoryId")
	ctx := c.Request.Context()

	result, err := db.DB.ExecContext(ctx,
		`DELETE FROM agent_memories WHERE id = $1 AND workspace_id = $2`, memoryID, workspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "memory not found or not owned by this workspace"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func nextArg(current int) string {
	return fmt.Sprintf("$%d", current+1)
}
