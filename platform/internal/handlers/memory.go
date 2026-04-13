package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/db"
	"github.com/gin-gonic/gin"
)

type MemoryEntry struct {
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	ExpiresAt *time.Time      `json:"expires_at,omitempty"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type MemoryHandler struct{}

func NewMemoryHandler() *MemoryHandler { return &MemoryHandler{} }

// List handles GET /workspaces/:id/memory
func (h *MemoryHandler) List(c *gin.Context) {
	workspaceID := c.Param("id")

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT key, value, expires_at, updated_at
		FROM workspace_memory
		WHERE workspace_id = $1 AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY key
	`, workspaceID)
	if err != nil {
		log.Printf("Memory list error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()

	entries := make([]MemoryEntry, 0)
	for rows.Next() {
		var entry MemoryEntry
		var value []byte
		if err := rows.Scan(&entry.Key, &value, &entry.ExpiresAt, &entry.UpdatedAt); err != nil {
			log.Printf("Memory list scan error: %v", err)
			continue
		}
		entry.Value = json.RawMessage(value)
		entries = append(entries, entry)
	}

	c.JSON(http.StatusOK, entries)
}

// Get handles GET /workspaces/:id/memory/:key
func (h *MemoryHandler) Get(c *gin.Context) {
	workspaceID := c.Param("id")
	key := c.Param("key")

	var entry MemoryEntry
	var value []byte
	err := db.DB.QueryRowContext(c.Request.Context(), `
		SELECT key, value, expires_at, updated_at
		FROM workspace_memory
		WHERE workspace_id = $1 AND key = $2 AND (expires_at IS NULL OR expires_at > NOW())
	`, workspaceID, key).Scan(&entry.Key, &value, &entry.ExpiresAt, &entry.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
		return
	}
	if err != nil {
		log.Printf("Memory get error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}

	entry.Value = json.RawMessage(value)
	c.JSON(http.StatusOK, entry)
}

// Set handles POST /workspaces/:id/memory
func (h *MemoryHandler) Set(c *gin.Context) {
	workspaceID := c.Param("id")

	var body struct {
		Key        string          `json:"key"`
		Value      json.RawMessage `json:"value"`
		TTLSeconds *int            `json:"ttl_seconds"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if body.Key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
		return
	}

	var expiresAt *time.Time
	if body.TTLSeconds != nil {
		t := time.Now().Add(time.Duration(*body.TTLSeconds) * time.Second)
		expiresAt = &t
	}

	_, err := db.DB.ExecContext(c.Request.Context(), `
		INSERT INTO workspace_memory(id, workspace_id, key, value, expires_at, updated_at)
		VALUES(gen_random_uuid(), $1, $2, $3::jsonb, $4, NOW())
		ON CONFLICT(workspace_id, key) DO UPDATE
		SET value = $3::jsonb, expires_at = $4, updated_at = NOW()
	`, workspaceID, body.Key, string(body.Value), expiresAt)
	if err != nil {
		log.Printf("Memory set error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set memory"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "key": body.Key})
}

// Delete handles DELETE /workspaces/:id/memory/:key
func (h *MemoryHandler) Delete(c *gin.Context) {
	workspaceID := c.Param("id")
	key := c.Param("key")

	_, err := db.DB.ExecContext(c.Request.Context(), `
		DELETE FROM workspace_memory WHERE workspace_id = $1 AND key = $2
	`, workspaceID, key)
	if err != nil {
		log.Printf("Memory delete error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
