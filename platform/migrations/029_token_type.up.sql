-- #684 — token type distinction: 'workspace' vs 'admin'
--
-- Before this migration AdminAuth called ValidateAnyToken, which accepted ANY
-- live token regardless of which workspace it was issued to. That meant a
-- workspace agent bearer could hit /bundles/import, /events, /org/import, etc.
-- by presenting its own workspace token.
--
-- Fix: introduce a token_type column. IssueToken continues to produce
-- 'workspace' tokens (scoped to an agent). IssueAdminToken produces 'admin'
-- tokens (platform-wide, not scoped to a workspace). ValidateAnyToken (used
-- by AdminAuth) now filters WHERE token_type = 'admin', so workspace bearers
-- are unconditionally rejected on admin routes.
--
-- Existing rows default to 'workspace'. Any token issued before this migration
-- by the test-token endpoint (dev/CI only) must be re-issued — the endpoint
-- was updated to call IssueAdminToken instead.

-- Make workspace_id nullable so admin tokens (not bound to any workspace) can
-- be stored in the same table. The NOT NULL constraint on existing 'workspace'
-- rows is preserved by the CHECK constraint below.
ALTER TABLE workspace_auth_tokens
    ALTER COLUMN workspace_id DROP NOT NULL;

ALTER TABLE workspace_auth_tokens
    ADD COLUMN IF NOT EXISTS token_type TEXT NOT NULL DEFAULT 'workspace';

-- CHECK constraint validates accepted values and enforces that workspace tokens
-- always carry a workspace_id while admin tokens must have workspace_id = NULL.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'workspace_auth_tokens_token_type_check'
          AND conrelid = 'workspace_auth_tokens'::regclass
    ) THEN
        ALTER TABLE workspace_auth_tokens
            ADD CONSTRAINT workspace_auth_tokens_token_type_check
            CHECK (token_type IN ('workspace', 'admin'));
    END IF;
    -- workspace tokens MUST have a workspace_id; admin tokens MUST NOT.
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'workspace_auth_tokens_scope_check'
          AND conrelid = 'workspace_auth_tokens'::regclass
    ) THEN
        ALTER TABLE workspace_auth_tokens
            ADD CONSTRAINT workspace_auth_tokens_scope_check
            CHECK (
                (token_type = 'workspace' AND workspace_id IS NOT NULL) OR
                (token_type = 'admin'     AND workspace_id IS NULL)
            );
    END IF;
END $$;
