# Launch Brief: Org-Scoped API Keys
**PR:** [#1105](https://github.com/Molecule-AI/molecule-core/pull/1105) — `feat(auth): org-scoped API keys`
**Merged:** 2026-04-20
**Owner:** PMM | **Status:** DRAFT — routing to Content Marketer

---

## Problem

Everyday development and integrations required full-admin tokens (`ADMIN_TOKEN`). There was no way to issue a token scoped to a specific org — you either got full access or nothing. For platform teams sharing tokens across tools, this was a silent security risk and a governance gap enterprise buyers flag in security reviews.

---

## Solution

User-minted full-admin tokens replace `ADMIN_TOKEN` for everyday use, with org-level scoping and a canvas UI tab for token management. Admins can now issue, rotate, and revoke tokens with the minimum required scope — org only, no global access.

---

## 3 Core Claims

1. **Scoped by default.** Org-level bearer tokens replace shared admin keys. Workspace A's token cannot hit Workspace B — enforced at the protocol level (Phase 30.1 auth model).
2. **Self-service token management.** Canvas UI tab lets admins issue, rotate, and revoke tokens without touching infra config.
3. **Enterprise procurement-ready.** Org scoping closes the gap that security reviewers flag in eval questionnaires — no more "one global key for everything."

---

## Target Developer

- **Indie devs / small teams** who want to rotate tokens without redeploying
- **Platform teams** integrating Molecule AI into multi-tenant tooling
- **Enterprise security reviewers** who require scoped auth before purchase

---

## CTA

"Replace your shared admin key. Issue org-scoped tokens from the canvas." → Docs link: TBD (confirm routing)

---

## Coverage Decision (from Content Marketer, 2026-04-21)

**No standalone blog post needed.** Folds into Phase 30 secure-by-design narrative. Social copy at `campaigns/org-api-keys-launch/social-copy.md` is the right level of coverage.

---

## Positioning Alignment

- Strengthens Phase 30.1 auth narrative (`X-Workspace-ID` + per-workspace tokens)
- Directly addresses the "governance" concern surfaced in enterprise positioning
- No competitor has a clear org-scoped token story — potential differentiation angle

---

## Open Questions

- [x] Does this need a dedicated blog post? → No (Content Marketer confirmed)
- [ ] Does the canvas UI tab have a public GA date?
- [ ] CTA doc link — confirm docs routing before publish

---

*PMM — route social copy to Social Media Brand once canvas UI tab is GA.*
