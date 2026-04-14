-- Add `source` column to workspace_schedules so the org-import path can
-- distinguish rows it owns ('template') from rows created via the runtime
-- API/Canvas ('runtime'). DB is the source of truth; org/import is now
-- additive — it only INSERTs missing template rows and only UPDATEs rows
-- where source = 'template'. Runtime-added schedules survive re-imports.

ALTER TABLE workspace_schedules
    ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'runtime'
    CHECK (source IN ('template', 'runtime'));

COMMENT ON COLUMN workspace_schedules.source IS
    'template = seeded by org/import (refreshable); runtime = created via Canvas/API (preserved across re-imports)';

-- Required so org-import can use ON CONFLICT (workspace_id, name) DO UPDATE.
-- Schedules within a single workspace are uniquely identified by name.
CREATE UNIQUE INDEX IF NOT EXISTS idx_schedules_workspace_name
    ON workspace_schedules(workspace_id, name);
