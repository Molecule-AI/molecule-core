DROP INDEX IF EXISTS org_api_tokens_live_org_idx;
DROP INDEX IF EXISTS org_api_tokens_org_id_idx;
ALTER TABLE org_api_tokens DROP COLUMN IF EXISTS org_id;