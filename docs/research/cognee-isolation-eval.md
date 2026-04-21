# Cognee Workspace Isolation Evaluation

**Date:** 2026-04-20
**Issue:** Molecule-AI/molecule-core#1146
**Status:** Preliminary — needs deeper architecture review

## Summary

Cognee (Apache-2.0, by Topoteretes UG) is an open-source AI memory engine with a shipped MCP component. It has direct overlap with Molecule AI's Phase 9 hierarchical memory architecture.

## Workspace Isolation Assessment

**Signal: Partial/Positive**

Cognee's GitHub README explicitly lists "agentic user/tenant isolation, traceability, OTEL collector, audit traits" as a core architectural feature.

This is a positive signal. However:
- The README mention does not specify the technical mechanism (namespace-level separation? separate vector DB instances per tenant? row-level security in a shared DB?)
- The cognee-mcp MCP component's handling of multi-workspace contexts is not documented in the surface-level readme

**Verdict:** Cognee claims tenant isolation. Further due diligence required before treating this as confirmed.

## Next Steps

1. **Deep-dive into cognee architecture docs** — check if isolation is enforced at the storage layer (separate DB/collection per workspace), application layer (row-level), or both
2. **Test cognee-mcp with a multi-workspace scenario** — the MCP tool interface should reveal whether workspace_id is a first-class parameter
3. **Check cognee's GitHub issues/discussions** — any community reports of cross-tenant data leakage?
4. **Evaluate migration path** — if Cognee is adopted, what's involved in migrating existing Phase 9 work?

## Recommendation

Proceed with Phase 9 build-vs-buy review. Cognee is a credible candidate — isolation is claimed but mechanism needs verification. The Phase 9 halt stands until this is resolved.

## Sources

- https://github.com/topoteretes/cognee (README, 2026-04-20)
- /workspace/repo/research/cognee-memo.md
