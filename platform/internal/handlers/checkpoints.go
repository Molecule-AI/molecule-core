package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Molecule-AI/molecule-monorepo/platform/internal/checkpoints"
	"github.com/gin-gonic/gin"
)

// CheckpointsHandler exposes three workspace-scoped endpoints for Temporal
// workflow checkpoint persistence (#788 / parent #583):
//
//	POST   /workspaces/:id/checkpoints          — upsert a step checkpoint
//	GET    /workspaces/:id/checkpoints/:wfid    — get the latest checkpoint
//	DELETE /workspaces/:id/checkpoints/:wfid    — clear on clean shutdown
//
// All routes live under the wsAuth Gin group so WorkspaceAuth middleware
// ensures the bearer token matches the :id path parameter.
type CheckpointsHandler struct {
	repo checkpoints.CheckpointRepository
}

// NewCheckpointsHandler creates a CheckpointsHandler backed by the given
// repository.  Pass checkpoints.NewRepository(db.DB) in production; pass a
// mock implementation in tests.
func NewCheckpointsHandler(repo checkpoints.CheckpointRepository) *CheckpointsHandler {
	return &CheckpointsHandler{repo: repo}
}

// Upsert handles POST /workspaces/:id/checkpoints
//
// Body: { "workflow_id", "step_name", "step_index", "payload"? }
//
// Returns 201 on success.
func (h *CheckpointsHandler) Upsert(c *gin.Context) {
	workspaceID := c.Param("id")
	ctx := c.Request.Context()

	var body struct {
		WorkflowID string          `json:"workflow_id" binding:"required"`
		StepName   string          `json:"step_name"   binding:"required"`
		StepIndex  int             `json:"step_index"`
		Payload    json.RawMessage `json:"payload"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Normalise payload: missing → nil → stored as SQL NULL.
	var rawPayload []byte
	if len(body.Payload) > 0 {
		rawPayload = body.Payload
	}

	if err := h.repo.UpsertCheckpoint(ctx, workspaceID, body.WorkflowID, body.StepName, body.StepIndex, rawPayload); err != nil {
		log.Printf("CheckpointsHandler.Upsert error workspace=%s wf=%s: %v", workspaceID, body.WorkflowID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upsert checkpoint"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"workspace_id": workspaceID,
		"workflow_id":  body.WorkflowID,
		"step_name":    body.StepName,
		"step_index":   body.StepIndex,
	})
}

// GetLatest handles GET /workspaces/:id/checkpoints/:wfid
//
// Returns the checkpoint with the highest step_index for the given workflow,
// or 404 when none exists.
func (h *CheckpointsHandler) GetLatest(c *gin.Context) {
	workspaceID := c.Param("id")
	workflowID := c.Param("wfid")
	ctx := c.Request.Context()

	cp, err := h.repo.GetLatestCheckpoint(ctx, workspaceID, workflowID)
	if err != nil {
		log.Printf("CheckpointsHandler.GetLatest error workspace=%s wf=%s: %v", workspaceID, workflowID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get checkpoint"})
		return
	}
	if cp == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no checkpoint found for workflow"})
		return
	}

	c.JSON(http.StatusOK, cp)
}

// Delete handles DELETE /workspaces/:id/checkpoints/:wfid
//
// Removes all checkpoints for the workflow (clean-shutdown path).
// Returns 200 with deleted count on success, 404 if no checkpoint existed.
func (h *CheckpointsHandler) Delete(c *gin.Context) {
	workspaceID := c.Param("id")
	workflowID := c.Param("wfid")
	ctx := c.Request.Context()

	n, err := h.repo.DeleteCheckpoints(ctx, workspaceID, workflowID)
	if err != nil {
		log.Printf("CheckpointsHandler.Delete error workspace=%s wf=%s: %v", workspaceID, workflowID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete checkpoints"})
		return
	}
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "no checkpoint found for workflow"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": n, "workflow_id": workflowID})
}
