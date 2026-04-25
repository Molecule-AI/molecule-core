-- 043_workspace_status_enum.up.sql
--
-- Convert workspaces.status from free-form TEXT to a real Postgres
-- ENUM type. The previous shape (TEXT DEFAULT 'provisioning' with no
-- CHECK constraint, set by 001_workspaces.sql) let any handler write
-- any string, including typos and stale values from older code paths.
-- Locking the value set forces every writer to use one of the agreed
-- states and lets us add a new state (`degraded`, used by the SDK
-- wedge detector landing in this same change) without losing type
-- safety on the column.
--
-- Value set covers every status the production codebase actually writes:
--
--   provisioning  — workspace row exists, container is being created
--                   (initial INSERT default)
--   online        — heartbeat fresh + last response was successful
--   offline       — Redis liveness key expired (ws-side dead) or
--                   the proxy detected an unreachable upstream
--   degraded      — runtime is alive but reporting trouble (heartbeat
--                   error_rate >= 0.5, OR new in this change:
--                   workspace explicitly reported runtime_state="wedged")
--   failed        — provisioning never completed, or workspace marked
--                   itself failed via bundle import / runtime crash
--   removed       — soft-delete tombstone; the row stays so foreign-
--                   key references survive but no operations target it
--
-- Verified before writing this migration that production code in
-- workspace-server/internal/{handlers,registry,bundle} writes only
-- values from this list (test fixtures may write others; tests run
-- against an isolated fixture DB so the cast doesn't affect them).

BEGIN;

CREATE TYPE workspace_status AS ENUM (
    'provisioning',
    'online',
    'offline',
    'degraded',
    'failed',
    'removed'
);

-- The two-step ALTER (DROP DEFAULT then change type then SET DEFAULT)
-- is required because Postgres rejects an ALTER COLUMN TYPE on a
-- column that has a DEFAULT whose expression doesn't match the new
-- type. The intermediate moment with no default is fine — no INSERT
-- happens between these statements inside the same transaction.
--
-- The `USING status::workspace_status` cast is the type-conversion
-- expression Postgres needs when the source and target types aren't
-- assignment-compatible. If any existing row has a status value
-- outside the enum's set, this statement aborts the transaction and
-- the migration leaves the table untouched — that's the correct
-- behavior (we'd want to know about the rogue value before locking
-- the type).
ALTER TABLE workspaces
    ALTER COLUMN status DROP DEFAULT;

ALTER TABLE workspaces
    ALTER COLUMN status TYPE workspace_status USING status::workspace_status;

ALTER TABLE workspaces
    ALTER COLUMN status SET DEFAULT 'provisioning'::workspace_status;

COMMIT;
