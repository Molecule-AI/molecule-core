# Phase 30 Lighthouse Audit Checklist

> Run this checklist at **48h post-publish** for each Phase 30 blog post. First run, then follow-up at **2 weeks**. Each item must be checked per post — do not batch visually.

**Posts to audit:**
1. `docs/blog/2026-04-17-deploy-anywhere/` — Fly Machines backend launch ⚠️ Source audit done, live Lighthouse pending deploy
2. `docs/blog/2026-04-20-chrome-devtools-mcp/` — Browser automation via MCP ✅ **SOURCE AUDIT COMPLETE** (see `2026-04-21-chrome-devtools-mcp-lighthouse-audit.md`). BLOCKED: page not deployed, OG image missing.
3. `docs/blog/2026-04-20-remote-workspaces/` — Remote Agent Workspaces (PR #1157, merged 2026-04-20T23:56Z) ✅ RESOLVED
4. `docs/blog/2026-04-20-agent-billing-attribution/` — Billing Attribution (pending — brief ready)
5. `docs/blog/2026-04-20-fly-machines-ai-hosting/` — Fly Machines blog (pending — brief ready)
6. `docs/blog/2026-04-21-cross-network-federation/` — Cross-Network Federation (pending — brief ready)
7. `docs/blog/2026-04-21-remote-agent-platform/` — Remote Agent Platform anchor (pending — brief ready)

---

## 1. Performance — Core Web Vitals

| Metric | Target | How to measure |
|--------|--------|----------------|
| LCP (Largest Contentful Paint) | < 2.5s | Lighthouse > Performance > LCP |
| CLS (Cumulative Layout Shift) | < 0.1 | Lighthouse > Performance > CLS |
| INP (Interaction to Next Paint) | < 200ms | Lighthouse > Performance > INP |
| Lighthouse Performance score | ≥ 90 | Lighthouse > Performance score |

**CLS risk areas to check manually:**
- [ ] Code blocks (```bash, ```json, ```python) — verify font size matches body text or has explicit width
- [ ] Images — all have explicit `width` and `height` attributes
- [ ] No layout shifts from async-loaded content

---

## 2. SEO — On-Page

| Check | Target | Notes |
|-------|--------|-------|
| Meta description | Present, 120–160 chars | Check source |
| Meta description | Under 160 chars | Character count |
| Title tag | Under 60 chars | Check source |
| H1 present | Exactly 1 per page | Check source |
| H2 count | At least 2 | Internal keyword vessels |
| Heading hierarchy | Logical H1 → H2 → H3 | No skipped levels |
| Canonical tag | Present, correct URL | `<link rel="canonical">` |
| noindex directive | **Absent** | Must NOT be present |
| `robots.txt` crawlable | Page accessible to Googlebot | Fetch as Googlebot |

**Primary keyword placement:**
- [ ] In H1 (exact match preferred)
- [ ] In first paragraph (first 100 words)
- [ ] In at least 2 H2 headings
- [ ] In meta description

**Internal links:**
- [ ] At least 2 internal links per post
- [ ] All internal links resolve (no 404s)
- [ ] External links open in new tab where appropriate

---

## 3. Structured Data — JSON-LD

| Check | Expected |
|-------|----------|
| JSON-LD blocks present | ✅ Present |
| `HowTo` schema (tutorial posts) | If setup/how-to structure |
| `Article` schema (narrative posts) | author, datePublished, dateModified |
| `TechArticle` schema | For technical reference posts |
| Schema valid | No JSON-LD parse errors |
| `@type` matches post type | Check against post structure |

**Rich snippet opportunities:**
- [ ] `HowTo` schema on setup/registration steps
- [ ] `Article` schema with author + datePublished
- [ ] `Comparison` schema on competitor comparison tables

---

## 4. Accessibility

| Check | Target | Lighthouse A11y score |
|-------|--------|----------------------|
| Lighthouse Accessibility score | ≥ 95 | Run Lighthouse > Accessibility |
| Color contrast | WCAG AA (4.5:1 for body text) | Inspect elements |
| `<th scope="col">` / `<th scope="row">` | Present in all tables | Check source |
| All images have `alt` text | 100% coverage | Lighthouse > Accessibility |
| Descriptive link text | Not "click here" or "read more" | Check each link |

---

## 5. Social / Open Graph

| Check | Expected |
|-------|----------|
| OG title tag | `<meta property="og:title">` present |
| OG description | `<meta property="og:description">` present |
| OG image | `<meta property="og:image">` present, 1200×630 |
| Twitter card | `<meta name="twitter:card">` present |
| Canonical URL | Correct absolute URL in OG tags |

---

## 6. Indexation

| Check | Tool |
|-------|------|
| Page is indexed | `site:molecule.ai/blog/<slug>` in Google |
| Page discoverable via sitemap | URL present in sitemap.xml |
| No `noindex` on page | Check HTML source |
| Fetch as Googlebot | Google Search Console > URL Inspection |

**Indexation spot-check queries (run at 48h):**
```
site:molecule.ai/blog/chrome-devtools-mcp
site:molecule.ai/blog/deploy-anywhere
site:molecule.ai/blog/remote-workspaces
site:molecule.ai/blog/agent-billing-attribution
site:molecule.ai/blog/fly-machines-ai-hosting
site:molecule.ai/blog/cross-network-federation
site:molecule.ai/blog/remote-agent-platform
```

---

## 7. Content Quality

| Check | Notes |
|-------|-------|
| Readability | Flesch-Kincaid appropriate for audience (dev/ops readers) |
| No placeholder text | No "TODO", "FIXME", "[insert]" |
| Code samples runnable | All code blocks syntactically valid |
| Links to external resources work | Test all external links |
| External links open in new tab | UX best practice |

---

## 8. Phase 30 Keyword Coverage — Per Post

### Chrome DevTools MCP (`chrome-devtools-mcp`)
Primary: `MCP browser automation`, `AI agent browser control`, `browser automation AI agents`, `MCP governance layer`
- [ ] All 4 P0 keywords present
- [ ] JSON-LD `HowTo` + `SoftwareApplication` present
- [ ] GH link correct: `github.com/ChromeDevTools/chrome-devtools-mcp`
- [ ] Comparison table: raw MCP vs Molecule AI

### Deploy Anywhere (`deploy-anywhere`)
Primary: `fly machines AI hosting`, `AI agent deployment platform`
- [ ] Both P0 keywords present
- [ ] Comparison table (Docker vs Fly vs Control Plane) present
- [ ] Phase 30 cross-link to remote-workspaces

### Remote Workspaces (`remote-workspaces`)
Primary: `remote workspaces AI`, `heterogeneous fleet visibility`, `per-workspace bearer tokens`
- [ ] All 3 Phase 30 keywords present
- [ ] Getting Started → external-agent-registration link
- [ ] JSON-LD TechArticle schema present

### Billing Attribution (once published)
Primary: `AI agent billing attribution`, `per-workspace AI cost tracking`, `AI agent cost allocation`
- [ ] All P0 keywords present
- [ ] Cross-link to cost-tracking.md

### Fly Machines AI Hosting (once published)
Primary: `fly machines AI hosting`, `remote AI agent platform`, `self-hosted AI agent platform`
- [ ] All P0 keywords present
- [ ] Cross-link to remote-workspaces.md

### Cross-Network Federation (once published)
Primary: `AI agent cross-network`, `AI agent fleet management`, `multi-tenant AI agents`
- [ ] All P0 keywords present
- [ ] Cross-link to billing attribution blog

### Remote Agent Platform (once published)
Primary: `remote AI agent platform`, `self-hosted AI agent platform`, `AI agent workspace isolation`
- [ ] All P0 keywords present
- [ ] Hub blog — bidirectional links to all other Phase 30 blogs

---

## 9. Reporting Template

```
## Lighthouse Audit — <POST SLUG> — <DATE>

URL: https://molecule.ai/blog/<slug>

### Scores
- Performance: __/100
- SEO: __/100
- Accessibility: __/100
- Best Practices: __/100

### Core Web Vitals
- LCP: __ms (target <2500ms)
- CLS: __ (target <0.1)
- INP: __ms (target <200ms)

### Passed
- [ ]

### Failed / Needs Fix
- [ ]

### Blocking Issues
1.

### Non-blocking Recommendations
1.

### Indexation Status
- Indexed: Yes / No
- In sitemap: Yes / No
```

---

*Maintained by SEO Analyst. Update when posts publish.*
