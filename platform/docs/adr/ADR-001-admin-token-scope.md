# ADR-001: Admin endpoints accept any workspace bearer token

**Status:** Accepted — known risk, Phase-H remediation planned  
**Date:** 2026-04-17  
**Issue:** #684

## Context
AdminAuth middleware uses ValidateAnyToken which accepts any live workspace bearer token.
The following admin endpoints are therefore reachable by any compromised workspace agent:
- GET /admin/workspaces/:id/test-token — mint tokens for any workspace
- DELETE /workspaces/:id — delete any workspace
- PUT/POST /settings/secrets — overwrite all global secrets
- GET /admin/github-installation-token — obtain live GitHub App token
- POST /bundles/import, POST /org/import — create rogue workspaces
- GET /events/:workspaceId — read any workspace event log
- PATCH /workspaces/:id/budget — clear any workspace budget

## Decision
Accepted as known risk. A proper token-tier separation (workspace vs admin scope) requires
a schema migration and bootstrap changes tracked in Phase-H. Implementing it as a hotfix
risks breaking existing scrapers and CI tooling.

## Accepted risk
A single compromised workspace agent can achieve full platform takeover via admin endpoints.
Mitigated by: workspace isolation, CanCommunicate access control, and audit logging.

## Phase-H remediation
Add `scope TEXT DEFAULT 'workspace' CHECK (scope IN ('workspace','admin'))` to
workspace_auth_tokens. AdminAuth rejects workspace-scope tokens. Admin tokens issued
only via explicit bootstrap flow. Tracked in phase-h/token-tier-upgrade.
