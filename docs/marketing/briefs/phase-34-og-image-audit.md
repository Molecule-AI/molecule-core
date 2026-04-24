# Phase 34 Blog Posts — og_image Audit
**Date:** 2026-04-23
**Auditor:** SEO Analyst
**Status:** 🔴 OPEN — needs Social Media Brand image generation + frontmatter fix

---

## Summary

All Phase 34 blog posts (GA April 30, 2026) are missing `og_image` frontmatter. Phase 34 brief flags partner-api-keys as HIGH priority; audit shows ALL FOUR posts are affected.

---

## Post-by-Post Status

### 1. `docs/blog/2026-04-23-partner-api-keys/index.md` — 🔴 HIGH
- **og_image:** MISSING
- **og_title:** "Ship Partner Integrations Faster with Programmatic Org Management" ✅ (68 chars, limit 70)
- **og_description:** "Partner API Keys: scoped, rate-limited, revocable API keys for programmatic org management. Built for marketplaces, CI/CD, and automation platforms." ✅ (146 chars, limit 155)
- **Asset path needed:** `docs/assets/blog/2026-04-23-partner-api-keys/og.png` (1200×630 PNG)
- **Image brief:** Dark tech, API key / mol_pk_ prefix visualization, marketplace/CI-CD context, Molecule AI branding

### 2. `docs/blog/2026-04-23-tool-trace-observability/index.md` — 🟡 MEDIUM
- **og_image:** MISSING
- **og_title:** "AI Agent Observability Without the Overhead" ✅ (45 chars)
- **og_description:** "See every tool your agent called — inputs, outputs, timing — in every A2A response. Parallel traces handled correctly. No sampling overhead." ✅ (140 chars)
- **Asset path needed:** `docs/assets/blog/2026-04-23-tool-trace-observability/og.png` (1200×630 PNG)
- **Image brief:** Dark tech, terminal/trace output aesthetic, tool call visualization, Molecule AI branding

### 3. `docs/blog/2026-04-23-platform-instructions-governance/index.md` — 🟡 MEDIUM
- **og_image:** MISSING
- **og_title:** "Govern Your AI Fleet at the System Prompt Level" ✅ (51 chars)
- **og_description:** "Platform Instructions: global and workspace-scoped rules prepended to the system prompt. Governance before the first turn, not after." ✅ (144 chars)
- **Asset path needed:** `docs/assets/blog/2026-04-23-platform-instructions-governance/og.png` (1200×630 PNG)
- **Image brief:** Dark tech, governance/policy visual (shield, hierarchy tree), enterprise feel, Molecule AI branding

### 4. `docs/blog/2026-04-23-tool-trace-platform-instructions/index.md` — 🟡 MEDIUM
- **og_image:** MISSING
- **og_title:** "Tool Trace + Platform Instructions: Full Visibility and Policy-Level Governance" ✅ (76 chars)
- **og_description:** "Tool-level observability in every A2A response meets system-prompt governance. Two enterprise-grade features, shipped together." ✅ (135 chars)
- **Asset path needed:** `docs/assets/blog/2026-04-23-tool-trace-platform-instructions/og.png` (1200×630 PNG)
- **Image brief:** Dark tech, two-panel or combined view: trace output + governance rule panel, enterprise, Molecule AI branding

---

## OG Image Spec (all posts)

| Property | Value |
|---|---|
| Dimensions | 1200×630px |
| Format | PNG |
| Style | Dark tech aesthetic (#0a0a0f–#111827 background), MCP teal accent (#00D4FF) |
| Font | Bold sans-serif title, smaller sans-serif subtitle |
| Branding | Molecule AI logo or text mark (subtle, corner) |
| Layout | Clean card/panel; no busy backgrounds |

---

## OG Meta Trims Needed (post-merge fixes)

After PRs #1923/#1922 merge, the MCP server list blog post meta descriptions need trimming:

| Field | Current | Current chars | Limit | Proposed |
|---|---|---|---|---|
| `og_description` | "Find the right MCP server for your AI agent workflow. Full list of reference servers, official integrations, server frameworks, and community registries — with Molecule AI compatibility notes." | 192 | 155 | "Find the right MCP server for your AI agent workflow. Full list of reference servers, official integrations, server frameworks, and community registries with Molecule AI compatibility." |
| `description` | "A practical guide to the Model Context Protocol ecosystem — finding the right MCP server for your use case, which ones integrate with Molecule AI, and how to evaluate servers before you commit." | 193 | 155 | "A practical guide to the Model Context Protocol ecosystem — finding the right MCP server for your use case and which ones integrate with Molecule AI." |

Both proposed trims are under limit and preserve core meaning.

---

## Landing Page SEO (PR #9)
- `Header.astro` logo `<img alt="">` — PR body claims fix but diff shows no change. Unresolved.
- og:image, og:title, hreflang — not changed in PR #9 diff. No action needed from this PR.

---

## Action Items

| Priority | Owner | Action |
|---|---|---|
| HIGH | Social Media Brand | Generate OG images (1200×630 PNG) for all 4 Phase 34 posts |
| HIGH | SEO Analyst | Add `og_image` frontmatter to all 4 posts pointing to generated assets |
| MEDIUM | SEO Analyst | Post-merge: trim MCP server list `og_description` and `description` |
| LOW | Landingpage team | Clarify Header.astro logo alt fix status for PR #9 |
