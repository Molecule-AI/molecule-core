# SaaS Federation v2 — What Shipped Research Note

**Date:** 2026-04-23  
**Author:** Marketing Lead (direct code search)  
**Status:** NO IMPLEMENTATION FOUND IN molecule-core

---

## Finding

**SaaS Federation v2 implementation is NOT present in `molecule-core`.**

Searched for:
- Files/functions containing `federation`, `FedV2`, `fed_v2`, `SaasFed`, `saas-fed`
- PR #1613 references in any migration, handler, or docs file
- `docs/tutorials/saas-federation` — does not exist
- `docs/marketing/launches/pr-1613-saas-federation-v2.md` — does not exist

No results found for any of the above.

## Implication

The Phase 34 messaging matrix (`docs/marketing/briefs/phase34-messaging-matrix.md`) already flags this:

> ⚠️ WARNING: SaaS Federation v2 is listed in Issue #1836 as a Phase 34 feature, but no PMM positioning brief or blog post exists for it yet. Do NOT draft community copy for this feature until PM confirms: (a) what it actually ships, (b) the GA/beta/alpha label, and (c) the primary use case narrative.

This note confirms: the implementation either lives in the private `molecule-controlplane` repo, or has not shipped to `molecule-core` yet.

## Recommendation

**The SaaS Federation v2 battlecard CANNOT be written** without PM confirmation of:
1. What the feature actually does (technical description)
2. GA / beta / alpha label
3. Primary use case (one sentence)
4. Whether the implementation is in `molecule-controlplane` (private) vs. `molecule-core`

## Marketing Impact

- SaaS Fed v2 is mentioned in Phase 34 community announcement (`docs/marketing/launches/phase-34-community-announcement.md`) as a bullet item — keep vague ("improved multi-org federation") until PM confirms specifics
- The Phase 34 community FAQ at `docs/marketing/launches/phase-34-community-faq.md` has one question about SaaS Fed v2 with a brief "improved reliability and org boundary controls" answer — acceptable placeholder until PM confirms
- Do NOT write a dedicated blog post or battlecard until PM confirms details

## Parking Status

Phase 32/34 SaaS Fed v2 battlecard: **PARKED** pending PM confirmation. Not a current blocker for April 30 GA launch — Tool Trace, Platform Instructions, and Partner API Keys launch content is complete.

---

*Marketing Lead 2026-04-23. Escalated to PM via status report.*
