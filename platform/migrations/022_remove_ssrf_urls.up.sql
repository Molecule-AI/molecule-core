-- C6 security remediation: remove workspace registrations with SSRF-capable URLs.
--
-- Clears the URL column (not the workspace row) for any registered workspace
-- whose URL targets a link-local or RFC-1918 address range that could be used
-- to probe cloud metadata services (169.254.169.254) or internal infrastructure.
--
-- 127.0.0.1 is intentionally NOT cleared — Docker-provisioned workspaces
-- register with http://127.0.0.1:<port> URLs, which are legitimate.
--
-- Run condition: safe to re-run (UPDATE ... WHERE url ~ '...' only matches SSRF rows).
UPDATE workspaces
SET    url = '', updated_at = now()
WHERE  status != 'removed'
  AND  url ~ '^https?://(169\.254\.|10\.|172\.(1[6-9]|2[0-9]|3[01])\.|192\.168\.)';
