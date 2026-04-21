# Cognee Architecture Deep-Dive — Workspace Isolation

**Date:** 2026-04-20
**Issue:** Molecule-AI/molecule-core#1146
**Research by:** Research Lead
**Status:** Complete

---

## Executive Summary

Cognee has **dataset-level isolation primitives** but **no storage-layer enforcement** and **no native `workspace_id` support** in its MCP tool interface. Cross-workspace isolation is caller-controlled, not enforced by the storage layer.

---

## Isolation Layer Analysis

| Layer | Mechanism | Enforced? | Risk |
|-------|-----------|-----------|------|
| Storage (Postgres) | No RLS, no schema namespacing | ❌ None | High |
| App — dataset | `dataset_name` passed per tool call | ⚠️ Caller-controlled | Medium |
| App — user | `get_default_user()` internal resolver only | ⚠️ Soft | Medium |
| MCP `workspace_id` param | Not present in cognee-mcp interface | ❌ N/A | High |

---

## Key Findings

1. **Storage layer:** No Postgres row-level security (RLS), no schema-level tenant separation. Any admin with DB access can read any tenant's data.

2. **Dataset isolation:** Cognee uses `dataset_name` as a logical namespace, but it's passed by the caller per tool call — not enforced server-side. A misconfigured or malicious caller could read/write across datasets.

3. **MCP interface:** `cognee-mcp` does not expose `workspace_id` as a first-class parameter. Workspaces would need to be mapped to dataset names externally.

4. **User isolation:** `get_default_user()` resolves users internally without verifiable enforcement at the data layer.

---

## Migration Implications

Adopting Cognee as the memory substrate requires an **auth bridge**:

- The bridge wraps cognee-mcp and injects `workspace_id` → `dataset_name` mapping
- All tool calls are routed through the bridge, which enforces tenant context
- Estimated effort: **~100–200 LOC** for the MCP proxy wrapper
- This is a pragmatic path — the bridge provides the isolation Cognee's storage layer lacks

---

## Recommendation

**Attempt the auth bridge prototype first (1–2 days of engineering):**
1. Build MCP proxy that maps workspace_id to dataset_name on each call
2. Validate that cross-workspace calls are correctly rejected
3. If clean → adopt Cognee for Phase 9
4. If complex → build native with storage-layer enforcement

**Do not proceed with Phase 9 proprietary memory investment until bridge prototype is evaluated.**

---

## Sources

- Cognee GitHub: https://github.com/topoteretes/cognee
- Preliminary eval: /workspace/repo/docs/research/cognee-isolation-eval.md
