package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/Molecule-AI/molecule-monorepo/platform/internal/wsauth"
	"github.com/gin-gonic/gin"
)

// TokenHandler exposes user-facing token management for workspaces.
// Routes: GET/POST/DELETE /workspaces/:id/tokens (behind WorkspaceAuth).
type TokenHandler struct{}

func NewTokenHandler() *TokenHandler {
	return &TokenHandler{}
}

type tokenListItem struct {
	ID        string     `json:"id"`
	Prefix    string     `json:"prefix"`
	CreatedAt time.Time  `json:"created_at"`
	LastUsed  *time.Time `json:"last_used_at"`
}

// List returns non-revoked tokens for the workspace (prefix + metadata only,
// never the plaintext or hash).
func (h *TokenHandler) List(c *gin.Context) {
	workspaceID := c.Param("id")

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT id, prefix, created_at, last_used_at
		FROM workspace_auth_tokens
		WHERE workspace_id = $1 AND revoked_at IS NULL
		ORDER BY created_at DESC
	`, workspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tokens"})
		return
	}
	defer rows.Close()

	tokens := []tokenListItem{}
	for rows.Next() {
		var t tokenListItem
		if err := rows.Scan(&t.ID, &t.Prefix, &t.CreatedAt, &t.LastUsed); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan token"})
			return
		}
		tokens = append(tokens, t)
	}

	c.JSON(http.StatusOK, gin.H{
		"tokens": tokens,
		"count":  len(tokens),
	})
}

// Create mints a new token for the workspace. The plaintext is returned
// exactly once in the response — it cannot be recovered afterwards.
func (h *TokenHandler) Create(c *gin.Context) {
	workspaceID := c.Param("id")

	token, err := wsauth.IssueToken(c.Request.Context(), db.DB, workspaceID)
	if err != nil {
		log.Printf("tokens: issue failed for %s: %v", workspaceID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create token"})
		return
	}

	log.Printf("tokens: issued new token for workspace %s", workspaceID)

	c.JSON(http.StatusCreated, gin.H{
		"auth_token":   token,
		"workspace_id": workspaceID,
		"message":      "Save this token now — it cannot be retrieved again.",
	})
}

// Revoke invalidates a specific token by ID. The token ID is the database
// row ID visible from List, not the plaintext token itself.
func (h *TokenHandler) Revoke(c *gin.Context) {
	workspaceID := c.Param("id")
	tokenID := c.Param("tokenId")

	result, err := db.DB.ExecContext(c.Request.Context(), `
		UPDATE workspace_auth_tokens
		SET revoked_at = now()
		WHERE id = $1 AND workspace_id = $2 AND revoked_at IS NULL
	`, tokenID, workspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke token"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "token not found or already revoked"})
		return
	}

	log.Printf("tokens: revoked token %s for workspace %s", tokenID, workspaceID)
	c.JSON(http.StatusOK, gin.H{"status": "revoked"})
}
