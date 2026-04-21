-- Add org_id column to org_api_tokens for Phase 32 multi-org isolation.
-- Tokens without org_id are pre-fix tokens (beta); requireCallerOwnsOrg
-- treats them as un-owned and denies access. Follow-up: capture org_id at
-- token creation time so all tokens carry their org anchor.
--
-- org_id references the root workspace that acts as the org anchor —
-- same pattern used by org_plugin_allowlist.org_id.
ALTER TABLE org_api_tokens
    ADD COLUMN org_id UUID REFERENCES workspaces(id) ON DELETE SET NULL;
