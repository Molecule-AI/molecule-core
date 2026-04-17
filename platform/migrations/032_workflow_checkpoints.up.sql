-- 032_workflow_checkpoints.up.sql
--
-- Temporal checkpoint persistence layer (#788 / parent #583).
-- Stores step-level progress for long-running Temporal workflows so they can
-- resume after a crash or restart without replaying already-completed steps.
-- Each (workspace_id, workflow_id, step_index) triple is unique; an ON CONFLICT
-- upsert updates the step_name and payload in-place.

CREATE TABLE IF NOT EXISTS workflow_checkpoints (
  id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  workflow_id  TEXT        NOT NULL,
  step_name    TEXT        NOT NULL,
  step_index   INT         NOT NULL,
  completed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  payload      JSONB,
  UNIQUE(workspace_id, workflow_id, step_index)
);

CREATE INDEX IF NOT EXISTS idx_wf_checkpoints_workspace
  ON workflow_checkpoints(workspace_id, workflow_id);
