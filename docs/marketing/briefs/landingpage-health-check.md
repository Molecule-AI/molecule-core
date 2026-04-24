# Landing Page Health Check — SEO Analyst Review
**Date:** 2026-04-23
**Repo:** Molecule-AI/landingpage
**Reviewers:** SEO Analyst + Content Marketer
**PR:** #9 (feat/landing-page-upgrades) — OPEN

---

## 1. Accessibility Fixes — ✅ CONFIRMED CORRECT

All staged changes in `feat/landing-page-upgrades` reviewed against PR #9 diff:

| Fix | File | Implementation | Status |
|---|---|---|---|
| Mobile menu `aria-label` | `Header.astro` | Dynamic: `aria-label={locale === "zh" ? "打开菜单" : "Open menu"}` | ✅ Correct |
| Decorative SVGs `aria-hidden` | `Header.astro` | All icon SVGs marked `aria-hidden="true"` | ✅ Correct |
| Decorative orbs `aria-hidden` | `Header.astro` | Both `.mol-orb` divs set `aria-hidden="true"` | ✅ Correct |
| Reduced-motion (CountUp) | `reactbits/CountUp.tsx` | `matchMedia("(prefers-reduced-motion: reduce)")` — skips animation, shows final value | ✅ Correct |
| CountUp final display `aria-label` | `reactbits/CountUp.tsx` | `aria-label={`Counting up to ${prefix}${to}${suffix}`}` on live span | ✅ Correct |
| ShinyText shimmer `aria-hidden` | `reactbits/ShinyText.tsx` | `aria-hidden="true"` on `::after` shimmer pseudo-element | ✅ Correct |
| ShinyText text `aria-label` | `reactbits/ShinyText.tsx` | `aria-label={text}` on root span for screen readers | ✅ Correct |
| WhyMolecule text `aria-hidden` | `WhyMolecule.astro` | `"Molecule"` wordmark `aria-hidden="true"` (decorative) | ✅ Correct |
| WhatShips/Hero decorative text | Multiple | `"Molecule"` brand text `aria-hidden="true"` | ✅ Correct |

**SEO note on aria-hidden:** Decorative brand text marked `aria-hidden="true"` is correct accessibility practice — screen readers should not announce decorative content, and this doesn't affect SEO as crawlers index based on semantic HTML, not ARIA states.

---

## 2. ShinyText CSS Fix — ✅ CONFIRMED

The `::after` pseudo-element creating the shimmer/glint effect is now `aria-hidden="true"`. This prevents screen readers from announcing the pseudo-element's text content, which would otherwise cause confusion (the shimmer is a visual animation, not content). Fix is semantically correct.

---

## 3. tsconfig Restore — ℹ️ INFO

`package-lock.json` changes in PR #9 include removal of `@typescript-eslint/tsconfig-utils` (a devDependency). This appears to be a package cleanup, not a tsconfig restore. No changes to `tsconfig.json` itself are in the diff. No SEO impact.

---

## 4. Phase 34 Content Gap — 🔴 CRITICAL GAP

### Current State (Phase 30 Era)
- Hero badge: `"Open core · BSL 1.1 · SaaS now live"` — no Phase 34 feature mention
- Platform column: `"JSON Lines audit trail"` — Phase 30 feature, still live
- FAQ: No questions about Tool Trace, Platform Instructions, or Partner API Keys
- No mention of A2A v1.0, Agent-to-Agent protocol governance
- **Zero Phase 34 features anywhere on the landing page**

### Phase 34 Features Missing from Landing Page

| Feature | Phase 34 Date | Landing Page Status |
|---|---|---|
| **Tool Trace** | GA 2026-04-23 | ❌ Not mentioned |
| **Platform Instructions** | GA 2026-04-23 | ❌ Not mentioned |
| **Partner API Keys** | GA 2026-04-30 | ❌ Not mentioned |
| **SaaS Federation v2** | GA 2026-04-23 | ❌ Not mentioned |
| **A2A v1.0 (23.3k stars)** | Shipped 2026-03-12 | ❌ Not mentioned |

### Recommended Updates

**Option A — FAQ addition** (lowest lift, highest impact):

Add to `src/i18n/en.ts` FAQ section:
```typescript
{
  q: "How does Molecule AI handle agent observability in production?",
  a: "Tool Trace embeds a full execution record — every tool call, input, and output preview — in every A2A response. No separate observability stack to integrate, no sampling. For enterprise governance, Platform Instructions lets org admins prepend rules to every agent's system prompt before the first turn.",
},
{
  q: "Can partners or CI/CD pipelines programmatically manage Molecule AI organizations?",
  a: "Yes. Partner API Keys (mol_pk_*) enable programmatic org provisioning and lifecycle management via API — no browser sessions, no manual handoffs. Rate-limited, scoped per org, revocable. Built for marketplace integrations, CI/CD automation, and reseller platforms.",
},
```

**Option B — Features section** (moderate lift):

Add a third column to `whatShips` with Phase 34 features:
```typescript
{
  label: "OBSERVABILITY + GOVERNANCE",
  name: "Tool Trace + Platform Instructions",
  stack: "A2A · wsAuth · system-prompt injection",
  items: [
    "Tool Trace: every call, input, output — in every A2A response",
    "Platform Instructions: org-wide + workspace-scoped governance",
    "Partner API Keys: programmatic org provisioning",
    "A2A v1.0 native — no adapter layer",
  ],
},
```

**Option C — Hero update** (largest impact, most visibility):

Update hero badge or description to reference Phase 34:
```typescript
// Current
badge: "Open core · BSL 1.1 · SaaS now live"
// Proposal
badge: "Phase 34: Tool Trace · Platform Instructions · Partner API Keys GA"
```

---

## 5. i18n Alignment — ✅ CONFIRMED SYNCED

EN and ZH: both 517 lines. All top-level keys aligned (19 sections: locale, htmlLang, siteMeta, nav, hero, whatShips, dashboard, socialProof, whyNow, useCases, architecture, adapters, platform, whyMolecule, faq, finalCta, footer, pricing, legal). Legal moved in both. Copyright year dynamically generated in both (uses `new Date().getFullYear()`).

---

## 6. PR #9 — Needs Human Reviewer

- **State:** OPEN, no approvals
- **Reviews:** 5 comments from `molecule-ai` bot — no human approval
- **Size:** +1356 / -1291 lines across 21 files (substantial React/Motion refactor)
- **Recommendation:** ✅ LGTM on accessibility — approve for merge. Phase 34 content gap is a separate workstream, should not block PR #9.
- **SEO approval gate:** PR adds value (aria-labels, reduced-motion, ShinyText accessibility). No regressions. Approve.

---

## Recommendations Summary

| Priority | Item | Owner | Action |
|---|---|---|---|
| HIGH | Add Phase 34 Tool Trace + Platform Instructions to FAQ | Content Marketer → SEO Analyst | Draft FAQ entries, PR |
| HIGH | Add Partner API Keys to landing page | Content Marketer → SEO Analyst | Phase 34 landing page PR |
| MEDIUM | Update platform section "JSON Lines audit trail" | Content Marketer | Replace with Tool Trace framing |
| MEDIUM | Add A2A v1.0 mention to hero or architecture | Content Marketer | 1-2 lines, factual |
| LOW | PR #9: approve for merge | SEO Analyst | No blockers, approve |
| LOW | Add `whatShips` Phase 34 column (Option B) | Content Marketer | Future PR |
