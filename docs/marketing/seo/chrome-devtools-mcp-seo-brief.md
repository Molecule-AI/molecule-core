# SEO Brief: Chrome DevTools MCP Campaign
**Status:** Draft v1 — SEO Analyst
**Date:** 2026-04-21
**Campaign:** Chrome DevTools MCP
**Keyword targets (from social-copy.md):** `AI agent browser control`, `MCP browser automation`, `browser automation governance`, `Chrome DevTools MCP`

---

## 1. SEO Audit: Existing Blog Post (2026-04-17-deploy-anywhere)

### Strengths
- ✅ Frontmatter present: `title`, `date`, `slug`, `description`, `tags`
- ✅ H2/H3 heading hierarchy is clean and scannable
- ✅ Internal links present (`/docs/quickstart`)
- ✅ MDX frontmatter description field feeds `<meta name="description">`
- ✅ One PR link anchor to external authority content
- ✅ Table used correctly for structured comparison data

### Gaps Identified
- ❌ **No Open Graph meta tags** in frontmatter — no `og:title`, `og:description`, `og:image`, `og:url`
- ❌ **No Twitter card meta** — no `twitter:card`, `twitter:title`, `twitter:description`
- ❌ **No canonical URL** in frontmatter — duplicate content risk
- ❌ **No `keywords` meta** — search engines still honour this for context signals
- ❌ **No structured data (JSON-LD)** — no `Article` or `TechArticle` schema
- ❌ **No anchor-ID links** on H2 headings — can't be deep-linked from social/external
- ❌ **No H1 inside body** — the rendered page H1 comes from frontmatter `title` only; if the MDX processor doesn't render it, there is no visible H1 in the body
- ❌ **Images lack alt text** — all images in blog posts should have descriptive alt text for accessibility + image search
- ❌ **No related posts / backlinks internal** — the post links out but nothing links back

### Recommended Per-Post Fixes (applies to all future posts too)
```yaml
# Add to frontmatter
keywords: [fly.io, AI agent deployment, multi-cloud, container backend]
canonical: https://molecule.ai/blog/deploy-anywhere
og_image: /assets/blog/2026-04-17-deploy-anywhere-og.png
twitter_card: summary_large_image
```

---

## 2. Chrome DevTools MCP — Keyword Gap Analysis

### Target Keywords & Intent

| Keyword | Intent | Difficulty Signal | Priority |
|---|---|---|---|
| `Chrome DevTools MCP` | Informational / discovery | Medium — niche but growing | **P0** |
| `MCP browser automation` | Informational + comparison | Medium — emerging category | **P0** |
| `AI agent browser control` | Informational | High competition | **P1** |
| `browser automation governance` | Informational / pain-point | Low competition | **P1** |
| `MCP protocol browser` | Informational | Medium | **P2** |
| `headless Chrome MCP` | Informational / how-to | Medium | **P2** |
| `AI browser agent enterprise` | Commercial investigation | High competition | **P2** |

### Topical Gap Map

| What competitors rank for | Does Molecule AI rank? | Action |
|---|---|---|
| "Chrome DevTools MCP" blog posts | Not yet — first-mover opportunity | Draft and publish ASAP |
| "MCP browser automation" comparison pages | Not yet | Create dedicated comparison/architecture doc |
| "browser automation governance" / enterprise framing | Not yet | Use as P1 differentiator in blog post |
| "headless browser AI agent" tutorials | Not yet | DevRel to write tutorial, SEO to interlink |
| Related: "MCP server list", "MCP protocol explained" | Not yet | Content Marketer to write explainer; interlink |

---

## 3. Recommended On-Site Structure for Chrome DevTools MCP Blog Post

### Frontmatter Template (copy for new post)
```yaml
---
title: "Browser Control for AI Agents: How Chrome DevTools MCP Works"
date: 2026-04-21
slug: chrome-devtools-mcp
description: "Learn how the Chrome DevTools MCP protocol gives AI agents secure, governance-ready browser control — without the Puppeteer dependency. Includes code sample and architecture diagram."
tags: [MCP, browser-automation, AI-agents, chrome, governance]
keywords: [Chrome DevTools MCP, MCP browser automation, AI agent browser control, browser automation governance, headless Chrome MCP]
canonical: https://molecule.ai/blog/chrome-devtools-mcp
og_title: "Browser Control for AI Agents: Chrome DevTools MCP"
og_description: "Secure, scalable browser control for AI agent teams using the Chrome DevTools MCP protocol. Enterprise governance built in."
og_image: /assets/blog/2026-04-21-chrome-devtools-mcp-og.png
twitter_card: summary_large_image
author: Molecule AI
---
```

### Recommended Heading Structure (H1→H2→H3)
```
H1: Browser Control for AI Agents: How Chrome DevTools MCP Works
  H2: What is the Chrome DevTools MCP Protocol?
    H3: How it differs from Puppeteer / Playwright
  H2: Why AI Agents Need Governance-Aware Browser Control
    H3: Credential scoping
    H3: Session isolation per workspace
    H3: Audit trails for browser actions
  H2: Architecture: Chrome DevTools MCP in Molecule AI
  H2: Code Sample — AI Agent Controls a Browser Tab
  H2: Enterprise Use Cases
  H2: Getting Started
```

### Internal Link Strategy
- Link from existing posts: `2026-04-17-deploy-anywhere` → new post (shared `platform` + `AI-agents` tag)
- Link from quickstart.md if browser runtime is documented there
- New post → `/docs/architecture/architecture` (control plane governance section)

### External Link / Authority Signals
- Link to official MCP spec (modelcontextprotocol.io)
- Link to Chrome DevTools protocol docs
- Link to any W3C browser automation specs

---

## 4. Technical SEO — Missing Infrastructure

| Issue | File / Location | Severity | Action |
|---|---|---|---|
| No `robots.txt` | `canvas/public/robots.txt` | High | Create — allow all, point to sitemap |
| No `sitemap.xml` | Next.js sitemap route | High | Add `app/sitemap.ts` per Next.js 15 |
| No JSON-LD on blog posts | MDX frontmatter | Medium | Add `schema` field → render as `<script type="application/ld+json">` |
| Generic layout metadata | `canvas/src/app/layout.tsx` | Medium | Add `generateMetadata` per blog slug |
| No OG image template | Design needed | Medium | Create 1200×630 OG template for blog |
| Lighthouse score unknown | N/A | Medium | Run Lighthouse audit on deployed blog |

---

## 5. Lighthouse Audit Checklist (post-deploy)

- [ ] **Performance**: LCP < 2.5s, FID < 100ms, CLS < 0.1
- [ ] **Accessibility**: All images have alt text, colour contrast ≥ 4.5:1, semantic HTML
- [ ] **Best Practices**: HTTPS (on production), no console errors, no deprecated APIs
- [ ] **SEO**: Meta description present, title < 60 chars, H1 present, no blocked resources

---

## 6. Off-Site SEO Recommendations

| Channel | Action | Owner |
|---|---|---|
| GitHub | Pin repo, add `topics: mcp, ai-agents, browser-automation` | DevRel |
| Hacker News | Submit when blog post goes live (not blogspam — add context) | SEO + Marketing Lead |
| Google Search Console | Submit sitemap, check coverage report post-publish | SEO |
| MDN / Chrome Extensions gallery | Investigate if listing makes sense | DevRel |

---

## 7. Dependencies & Next Steps

| Task | Owner | Status |
|---|---|---|
| Create `robots.txt` in `canvas/public/` | SEO Analyst | Pending |
| Add `app/sitemap.ts` for Next.js 15 sitemap | DevRel / Frontend | Pending |
| Write Chrome DevTools MCP blog post (1200–1500 words) | Content Marketer | Pending |
| Add OG image template (1200×630) | Social Media Brand | Pending |
| Run Lighthouse on deployed blog | SEO Analyst | Pending |
| Wire up per-post `generateMetadata` in Next.js app | DevRel / Frontend | Pending |

---

*Brief maintained by SEO Analyst (workspace: 5b277fc4-54c7-42b8-891c-82510b4c1b95). Update on each campaign cycle.*
