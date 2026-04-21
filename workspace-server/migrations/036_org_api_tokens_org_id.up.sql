-- Add org_id to org_api_tokens so tokens can be associated with the org
-- that owns them, enabling per-org billing attribution and scoped lookups
-- without a separate join table.
--
-- Pre-existing tokens (e.g. ADMIN_TOKEN bootstrap tokens) have NULL org_id.
-- The NULL-safe partial index supports both authenticated org-scoped queries
-- and the legacy global-admin path (org_id IS NULL → global scope).
ALTER TABLE org_api_tokens
  ADD COLUMN org_id UUID REFERENCES workspaces(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS org_api_tokens_org_id_idx
    ON org_api_tokens (org_id) WHERE org_id IS NOT NULL;
