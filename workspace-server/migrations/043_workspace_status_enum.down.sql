-- 043_workspace_status_enum.down.sql
--
-- Reverse 043_workspace_status_enum.up.sql: convert workspaces.status
-- back to plain TEXT and drop the workspace_status enum type.

BEGIN;

ALTER TABLE workspaces
    ALTER COLUMN status DROP DEFAULT;

ALTER TABLE workspaces
    ALTER COLUMN status TYPE TEXT USING status::TEXT;

ALTER TABLE workspaces
    ALTER COLUMN status SET DEFAULT 'provisioning';

DROP TYPE workspace_status;

COMMIT;
