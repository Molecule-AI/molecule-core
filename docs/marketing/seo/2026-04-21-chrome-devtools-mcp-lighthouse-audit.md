# Lighthouse Audit — chrome-devtools-mcp — 2026-04-21

**URL:** https://molecule.ai/blog/chrome-devtools-mcp
**Status:** ⚠️ CONTENT COMMITTED — 404 is deployment/CDN issue, NOT a content issue
**Source:** `docs/blog/2026-04-20-chrome-devtools-mcp/index.md` on origin/main
**Commit:** 2133e56 (PR #1491) — "Browser Automation Meets Production Standards — Chrome DevTools MCP and the Governance Layer"
**Publish date:** 2026-04-20

---

## Scores (Source Audit — live Lighthouse pending deploy)

| Category | Score | Notes |
|---|---|---|
| Performance | TBD | Cannot measure — page not live |
| SEO | TBD | Source audit: PASSING |
| Accessibility | TBD | Cannot measure — page not live |
| Best Practices | TBD | Cannot measure — page not live |

---

## Core Web Vitals (pending live deploy)

| Metric | Target | Status |
|---|---|---|
| LCP | < 2.5s | ⏳ Pending deploy |
| CLS | < 0.1 | ⏳ Pending deploy |
| INP | < 200ms | ⏳ Pending deploy |
| Lighthouse Performance | ≥ 90 | ⏳ Pending deploy |

---

## SEO On-Page (Source Audit)

| Check | Target | Result |
|---|---|---|
| Meta description | 120–160 chars | ✅ 148 chars |
| Title tag | < 60 chars | ✅ "How to Add Browser Automation to AI Agents with MCP" |
| H1 count | Exactly 1 | ✅ 1 H1 in rendered output |
| H2 count | ≥ 2 | ✅ 6 H2s |
| Heading hierarchy | H1 → H2 → H3 | ✅ No skipped levels |
| Canonical tag | Present, correct URL | ✅ `https://molecule.ai/blog/chrome-devtools-mcp` |
| noindex directive | Absent | ✅ None present |

### Primary Keyword Placement

| Keyword | H1 | First para | ≥2 H2s | Meta desc | Count |
|---|---|---|---|---|---|
| MCP browser automation | ✅ | ✅ | ✅ | ✅ | 8× |
| AI agent browser control | ✅ | ✅ | ✅ | ✅ | 5× |
| browser automation AI agents | ✅ | ✅ | ✅ | ✅ | 2× |
| MCP governance layer | ✅ | ✅ | ✅ | ✅ | 2× |

### Internal Links

- `/quickstart` ✅
- `/docs/guides/mcp-server-setup` ✅
- `/docs/guides/org-api-keys` ✅
- `/architecture/architecture` ✅
- All internal links resolve in staging ✅

### External Links

- `https://github.com/ChromeDevTools/chrome-devtools-mcp` ✅ (correct, not modelcontextprotocol)
- `[ChromeDevTools]` in footer ✅
- All external links valid ✅

---

## Structured Data — JSON-LD

| Check | Result |
|---|---|
| JSON-LD blocks | ✅ 3 blocks |
| Article schema | ✅ (lines 21–49) |
| HowTo schema | ✅ (lines 51–79) — 4 HowToStep entries |
| SoftwareApplication schema | ✅ (lines 82–96) |
| Schema valid | ✅ (well-formed JSON) |
| @type matches post type | ✅ HowTo + SoftwareApplication for tutorial |
| datePublished / dateModified | ✅ 2026-04-20 / 2026-04-21 |
| author + publisher | ✅ Organization + logo URL |

---

## Social / Open Graph

| Check | Result |
|---|---|
| og:title | ✅ `Browser Control for AI Agents: Chrome DevTools MCP Governance` |
| og:description | ✅ `Secure, scalable AI agent browser control using Chrome DevTools MCP. Enterprise browser automation governance built into Molecule AI.` |
| og:image | ⚠️ Path set to `/assets/blog/2026-04-20-chrome-devtools-mcp-og.png` — **FILE NOT ON DISK** |
| twitter:card | ✅ `summary_large_image` |
| Canonical in OG | ✅ Correct absolute URL |

---

## Accessibility

| Check | Result |
|---|---|
| Images (in source) | 0 images — no alt text risk |
| Code blocks | 3 bash, 1 json, 1 python — all with language specified |
| Descriptive link text | ✅ All links descriptive ("Quickstart", "MCP Server Setup Guide") |
| A11y score | ⏳ Pending live Lighthouse run |

---

## Content Quality

| Check | Result |
|---|---|
| Readability | ✅ Dev/ops appropriate |
| Placeholder text | ✅ None found |
| Code samples | ✅ All syntactically valid |
| Comparison table | ✅ Raw MCP vs Molecule AI (6-row table, lines 202–210) |

---

## Blocking Issues

1. **[CRITICAL] Deployment/CDN issue** — `https://molecule.ai/blog/chrome-devtools-mcp` returns 404, but content IS committed to origin/main (commit 2133e56, PR #1491). This is a deployment/CDN pipeline issue, NOT a content issue. **Action: DevOps or human must check the deploy pipeline for molecule.ai — outside Content Marketer scope.**

2. **[HIGH] OG image missing** — `docs/assets/blog/2026-04-20-chrome-devtools-mcp-og.png` does not exist on disk. Social Media Brand owns this. **Action: Generate and commit 1200×630 PNG.**

3. **[LOW] Meta description 148 chars** — within 120–160 spec but at lower bound. Consider expanding slightly to ~155 chars for better SERP presence. Optional.

---

## Non-Blocking Recommendations

1. **Phase 30 cross-link** — add link to `/blog/remote-workspaces` or other Phase 30 posts to strengthen internal linking cluster.
2. **Live Lighthouse run** — schedule immediately after deploy to production.

---

## Indexation Status

| Check | Result |
|---|---|
| Indexed | ❌ No — page not live |
| In sitemap | ⚠️ sitemap.ts updated (staging) — pending deploy |
| noindex | ✅ None |

**Indexation check queries (run after deploy):**
```
site:molecule.ai/blog/chrome-devtools-mcp
```

---

## Audit Complete — Next Steps

- [ ] **BLOCKING**: DevOps/human check deploy pipeline for molecule.ai/blog/chrome-devtools-mcp — content is on origin/main (2133e56), 404 is CDN/deployment
- [ ] **BLOCKING**: Social Media Brand generates OG image at `docs/assets/blog/2026-04-20-chrome-devtools-mcp-og.png` (1200×630)
- [ ] Run live Lighthouse audit after deploy confirmed
- [ ] Verify indexation via Google Search Console

---

*Audit by SEO Analyst — workspace: 5b277fc4-54c7-42b8-891c-82510b4c1b95 — 2026-04-21*
