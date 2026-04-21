-- Add org_id to org_api_tokens for org-scoped authorization.
--
-- org_id is the workspace UUID of the org root (the workspace whose
-- parent_id IS NULL and which is the ancestor of all org workspaces).
-- This lets requireCallerOwnsOrg authorize org-token holders against
-- specific org workspaces without relying on the misnamed created_by
-- column (which holds WorkOS user_ids, not workspace references).
--
-- org_id is NULL for:
--   - Pre-migration tokens (backfill deferred — safe, auth defaults
--     to bypass for legacy tokens anyway).
--   - Tokens minted via ADMIN_TOKEN bootstrap (no session context).
--
-- New tokens minted via OrgTokenHandler.Create always receive org_id
-- from the request context's org workspace resolution.

ALTER TABLE org_api_tokens ADD COLUMN IF NOT EXISTS org_id UUID;

-- Primary index for the auth path: requireCallerOwnsOrg looks up
-- (id, org_id) for every org-token request.
CREATE INDEX IF NOT EXISTS org_api_tokens_org_id_idx
    ON org_api_tokens (id, org_id);

-- Partial index: only index rows that have org_id (auth queries filter
-- by org_id, not by id alone).
CREATE INDEX IF NOT EXISTS org_api_tokens_live_org_idx
    ON org_api_tokens (org_id)
    WHERE org_id IS NOT NULL;