// Package checkpoints provides persistence for Temporal workflow step
// checkpoints (#788 / parent #583).  A checkpoint records that a specific
// workflow step completed successfully so that, after a crash or restart,
// the workflow can query the platform and skip already-completed steps.
package checkpoints

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time"
)

// Checkpoint is the canonical in-memory representation of a persisted step.
type Checkpoint struct {
	ID          string
	WorkspaceID string
	WorkflowID  string
	StepName    string
	StepIndex   int
	CompletedAt time.Time
	Payload     json.RawMessage // nil when no payload was stored
}

// CheckpointRepository defines the persistence operations for workflow
// checkpoints.  Implementations must use parameterised queries and must
// propagate the caller's context to every DB call.
type CheckpointRepository interface {
	// UpsertCheckpoint inserts a new checkpoint or, on conflict for the
	// (workspace_id, workflow_id, step_index) triple, updates the step_name,
	// completed_at, and payload in-place.  payload may be nil.
	UpsertCheckpoint(
		ctx context.Context,
		workspaceID, workflowID, stepName string,
		stepIndex int,
		payload []byte,
	) error

	// GetLatestCheckpoint returns the checkpoint with the highest step_index
	// for the given workflow.  Returns (nil, nil) when no checkpoint exists.
	GetLatestCheckpoint(
		ctx context.Context,
		workspaceID, workflowID string,
	) (*Checkpoint, error)

	// DeleteCheckpoints removes all checkpoints for a workflow.  Used on
	// clean shutdown to release storage.  Returns the number of rows deleted
	// (0 is not an error — the workflow may never have been checkpointed).
	DeleteCheckpoints(
		ctx context.Context,
		workspaceID, workflowID string,
	) (int64, error)
}

// postgresRepository is the production implementation backed by Postgres.
type postgresRepository struct {
	db *sql.DB
}

// NewRepository returns a CheckpointRepository backed by the given database.
// Pass db.DB at server startup; pass a sqlmock DB in tests.
func NewRepository(database *sql.DB) CheckpointRepository {
	return &postgresRepository{db: database}
}

// UpsertCheckpoint inserts or updates a workflow step checkpoint.
//
// SQL contract
// ------------
// ON CONFLICT (workspace_id, workflow_id, step_index) DO UPDATE refreshes
// step_name, completed_at, and payload so that a re-delivered activity (e.g.
// after a Temporal retry) is idempotent.
//
// JSONB safety: payload is converted to string and cast with ::jsonb so that
// lib/pq does not misinterpret []byte as bytea.
func (r *postgresRepository) UpsertCheckpoint(
	ctx context.Context,
	workspaceID, workflowID, stepName string,
	stepIndex int,
	payload []byte,
) error {
	// Normalise payload: nil or empty → SQL NULL via the empty string path.
	payloadStr := "null"
	if len(payload) > 0 {
		payloadStr = string(payload)
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO workflow_checkpoints
		    (workspace_id, workflow_id, step_name, step_index, payload)
		VALUES ($1, $2, $3, $4, $5::jsonb)
		ON CONFLICT (workspace_id, workflow_id, step_index) DO UPDATE
		    SET step_name    = EXCLUDED.step_name,
		        completed_at = now(),
		        payload      = EXCLUDED.payload
	`, workspaceID, workflowID, stepName, stepIndex, payloadStr)
	if err != nil {
		log.Printf("checkpoints: UpsertCheckpoint error workspace=%s wf=%s step=%d: %v",
			workspaceID, workflowID, stepIndex, err)
	}
	return err
}

// GetLatestCheckpoint returns the highest-step_index checkpoint for a workflow.
//
// Returns (nil, nil) when no checkpoint exists (sql.ErrNoRows is consumed).
// Returns (nil, err) on any other database error.
func (r *postgresRepository) GetLatestCheckpoint(
	ctx context.Context,
	workspaceID, workflowID string,
) (*Checkpoint, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, workspace_id, workflow_id, step_name, step_index, completed_at, payload
		FROM workflow_checkpoints
		WHERE workspace_id = $1 AND workflow_id = $2
		ORDER BY step_index DESC
		LIMIT 1
	`, workspaceID, workflowID)
	if err != nil {
		log.Printf("checkpoints: GetLatestCheckpoint query error workspace=%s wf=%s: %v",
			workspaceID, workflowID, err)
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		// rows.Next() returns false on EOF (no rows) and on error. Check
		// rows.Err() to distinguish the two cases.
		if err := rows.Err(); err != nil {
			log.Printf("checkpoints: GetLatestCheckpoint rows.Err workspace=%s wf=%s: %v",
				workspaceID, workflowID, err)
			return nil, err
		}
		// No checkpoint exists for this workflow — not an error.
		return nil, nil
	}

	var cp Checkpoint
	var rawPayload []byte
	if err := rows.Scan(
		&cp.ID, &cp.WorkspaceID, &cp.WorkflowID,
		&cp.StepName, &cp.StepIndex, &cp.CompletedAt, &rawPayload,
	); err != nil {
		log.Printf("checkpoints: GetLatestCheckpoint scan error workspace=%s wf=%s: %v",
			workspaceID, workflowID, err)
		return nil, err
	}
	if len(rawPayload) > 0 {
		cp.Payload = json.RawMessage(rawPayload)
	}

	// Consume any remaining rows and surface iteration errors.
	if err := rows.Err(); err != nil {
		log.Printf("checkpoints: GetLatestCheckpoint rows.Err (post-scan) workspace=%s wf=%s: %v",
			workspaceID, workflowID, err)
		return nil, err
	}

	return &cp, nil
}

// DeleteCheckpoints removes all checkpoints for the given workflow.
// Returns (0, nil) when the workflow had no checkpoints — not an error.
func (r *postgresRepository) DeleteCheckpoints(
	ctx context.Context,
	workspaceID, workflowID string,
) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM workflow_checkpoints
		WHERE workspace_id = $1 AND workflow_id = $2
	`, workspaceID, workflowID)
	if err != nil {
		log.Printf("checkpoints: DeleteCheckpoints error workspace=%s wf=%s: %v",
			workspaceID, workflowID, err)
		return 0, err
	}
	n, _ := result.RowsAffected()
	return n, nil
}
