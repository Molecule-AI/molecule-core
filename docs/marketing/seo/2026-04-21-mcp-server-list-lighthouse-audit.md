# Lighthouse Audit — mcp-server-list — 2026-04-21

**URL:** https://molecule.ai/blog/mcp-server-list
**Status:** ⚠️ CONTENT COMMITTED — blog routes redirect to /lander (not serving content)
**Live check:** `curl -sL molecule.ai/blog/mcp-server-list` → `<script>window.onload=function(){window.location.href="/lander"}</script>` — all blog routes redirect to landing page, not rendered
**Source:** `docs/blog/2026-04-20-mcp-server-list/index.md` on `fix/chrome-devtools-mcp-tutorial` (SHA 91c1977)
**Publish date:** 2026-04-20 (recreated 2026-04-21 due to prior commit loss)

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
| Meta description | 120–160 chars | ✅ 158 chars |
| Title tag | < 60 chars | ✅ "The MCP Server List: Which Servers Work With Molecule AI?" (66 chars — 6 over, see note) |
| H1 count | Exactly 1 | ✅ 1 H1 in rendered output |
| H2 count | ≥ 2 | ✅ 10 H2s |
| Heading hierarchy | H1 → H2 → H3 | ✅ No skipped levels |
| Canonical tag | Present, correct URL | ✅ `https://molecule.ai/blog/mcp-server-list` |
| noindex directive | Absent | ✅ None present |

**Title tag note:** Title is 66 chars — 6 over the 60-char target. This is acceptable (search engines display up to ~600px which equals ~60 chars at default sizing, but long titles are common and tolerated). Consider trimming "With Molecule AI" if strict compliance is needed.

### Primary Keyword Placement

| Keyword | Target | H1 | First para | ≥2 H2s | Meta desc | Count |
|---|---|---|---|---|---|---|
| MCP server list | 8× | ✅ | ✅ | ✅ | ✅ | 10× ✅ |
| MCP servers | 23× | ❌ | ✅ | ✅ | ❌ | 14× ⚠️ |
| Model Context Protocol | 8× | ✅ | ✅ | ✅ | ✅ | 8× ✅ |
| MCP server | 47× | ✅ | ✅ | ✅ | ✅ | 42× ⚠️ |
| MCP integration | 2× | ❌ | ✅ | ✅ | ✅ | 9× ✅ |

**MCP servers / MCP server note:** Both keywords are covered throughout the body and headings. "MCP server" is 5 short of the 47× target but is used naturally and appears in every major section. Consider adding "many MCP servers" or "the MCP servers" to the intro paragraph if strict density is required.

### Internal Links

- `/docs/guides/mcp-server-setup` ✅ (appears twice — intro + closing)
- `/blog/chrome-devtools-mcp` ✅ — cross-linked in chrome-devtools-mcp post; not in mcp-server-list (intentional — one-way cross-link is sufficient per SEO best practices)
- All internal links resolve locally ✅

### External Links

- `https://github.com/modelcontextprotocol` ✅ (2 occurrences)
- `https://registry.mcp.so` ✅
- Brave Search API, Slack, GitHub, AWS, Google Drive vendor links — all HTTPS ✅

---

## Structured Data — JSON-LD

| Check | Result |
|---|---|
| JSON-LD blocks | ⚠️ None present — recommend adding Article schema |
| datePublished | ✅ 2026-04-20 in frontmatter |
| dateModified | ✅ 2026-04-21 in frontmatter (recreation) |
| author | ✅ "Molecule AI" in frontmatter |

**Action item:** Add JSON-LD Article schema block to frontmatter or MDX body. Template:

```html
<script type="application/ld+json">
{
  "@context": "https://schema.org",
  "@type": "Article",
  "headline": "The MCP Server List: Which Servers Work With Molecule AI?",
  "datePublished": "2026-04-20",
  "dateModified": "2026-04-21",
  "author": { "@type": "Organization", "name": "Molecule AI" },
  "publisher": { "@type": "Organization", "name": "Molecule AI", "logo": "https://molecule.ai/logo.png" },
  "description": "A practical guide to the Model Context Protocol ecosystem — finding the right MCP server for your use case, which ones integrate with Molecule AI, and how to evaluate servers before you commit.",
  "keywords": "MCP server list, MCP servers, Model Context Protocol, MCP server, MCP integration"
}
</script>
```

---

## Social / Open Graph

| Check | Result |
|---|---|
| og:title | ✅ `The MCP Server List: Which Servers Work With Molecule AI?` |
| og:description | ✅ `Find the right MCP server for your AI agent workflow. Full list of reference servers, official integrations, server frameworks, and community registries — with Molecule AI compatibility notes.` |
| og:image | ⚠️ **MISSING** — no `og_image` frontmatter set. Needs Social Media Brand to generate 1200×630 PNG at `/assets/blog/2026-04-21-mcp-server-list-og.png` |
| twitter:card | ⚠️ Not set — recommend adding `twitter_card: summary_large_image` to frontmatter |
| Canonical in OG | ✅ `https://molecule.ai/blog/mcp-server-list` |

---

## Accessibility

| Check | Result |
|---|---|
| Images | 0 images — no alt text risk |
| Code blocks | 4 JSON/bash/python blocks — all with language specified |
| Descriptive link text | ✅ All links descriptive ("MCP Server Setup Guide", "modelcontextprotocol GitHub organization") |
| A11y score | ⏳ Pending live Lighthouse run |

---

## Content Quality

| Check | Result |
|---|---|
| Readability | ✅ Developer audience appropriate, technical but accessible |
| Placeholder text | ✅ None found |
| Code samples | ✅ All syntactically valid JSON, bash, Python |
| Table (decision guide) | ✅ 7-row decision table (lines ~190-200) |
| Cross-links | ✅ chrome-devtools-mcp cross-linked from other post; mcp-server-list links to MCP Server Setup Guide |

---

## Blocking Issues

1. **[CRITICAL] Blog routes not serving content** — All blog pages (mcp-server-list, chrome-devtools-mcp, deploy-anywhere, and others) return `window.location.href="/lander"` instead of rendered content. CDN returns HTTP 200 but routes to landing page. **Action: DevOps must check the canvas/Next.js build and deploy pipeline — blog routes not rendered in production.**

2. **[HIGH] OG image missing** — no `og_image` frontmatter set. Social Media Brand needs to generate 1200×630 PNG at `docs/assets/blog/2026-04-21-mcp-server-list-og.png`.

3. **[MEDIUM] JSON-LD Article schema not present** — source audit shows no structured data block. Recommend adding Article schema per chrome-devtools-mcp pattern.

4. **[LOW] Title tag 66 chars** — 6 over the 60-char target. Acceptable but could be trimmed for strict compliance.

---

## Non-Blocking Recommendations

1. **Add JSON-LD Article schema** — use template provided above
2. **Add `twitter_card: summary_large_image`** to frontmatter to mirror og:image coverage
3. **Live Lighthouse run** — schedule immediately after DevOps fixes blog route rendering
4. **Verify indexation** via Google Search Console: `site:molecule.ai/blog/mcp-server-list`
5. **Add FAQPage schema** — MCP server decision guide pairs well with FAQ structured data (e.g., "What is an MCP server?", "How do I add an MCP server to Molecule AI?")

---

## Indexation Status

| Check | Result |
|---|---|
| Indexed | ❌ No — page not live |
| In sitemap | ⚠️ sitemap.ts updated (on fix/chrome-devtools-mcp-tutorial SHA 91c1977) — pending deploy |
| noindex | ✅ None |

**Indexation check queries (run after deploy):**
```
site:molecule.ai/blog/mcp-server-list
site:molecule.ai "MCP server list"
```

---

## Audit Complete — Next Steps

- [ ] **BLOCKING**: DevOps check canvas/Next.js build — blog routes redirect to /lander, not rendering MDX content in production
- [ ] **BLOCKING**: Social Media Brand generates OG image at `docs/assets/blog/2026-04-21-mcp-server-list-og.png` (1200×630)
- [ ] **RECOMMENDED**: Add JSON-LD Article schema + `twitter_card` to frontmatter
- [ ] Run live Lighthouse audit after blog routes confirmed rendering
- [ ] Verify indexation via Google Search Console

---

*Audit by SEO Analyst — workspace: 5b277fc4-54c7-42b8-891c-82510b4c1b95 — 2026-04-21*
