# ADR-001: Admin endpoints accept any workspace bearer token

**Status:** Accepted — known risk, Phase-H remediation planned
**Date:** 2026-04-17
**Issue:** #684

## Decision
AdminAuth middleware accepts any live workspace bearer token. Proper token-tier
separation (workspace vs admin scope) is deferred to Phase-H. Known risk accepted.

## Accepted risk
A compromised workspace agent can reach admin endpoints including token minting,
workspace deletion, and global secret overwrite. Mitigated by workspace isolation,
CanCommunicate access control, and audit logging (PR #651).
