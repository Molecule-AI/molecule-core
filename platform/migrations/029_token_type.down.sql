ALTER TABLE workspace_auth_tokens DROP CONSTRAINT IF EXISTS workspace_auth_tokens_scope_check;
ALTER TABLE workspace_auth_tokens DROP CONSTRAINT IF EXISTS workspace_auth_tokens_token_type_check;
ALTER TABLE workspace_auth_tokens DROP COLUMN IF EXISTS token_type;
-- Note: we cannot safely re-add NOT NULL to workspace_id if admin rows (NULL) exist.
-- Operators should purge admin tokens before rolling back this migration.
