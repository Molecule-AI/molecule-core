# Chrome DevTools MCP — SEO Campaign Audit
**SEO Analyst:** self-assigned #1335 | **Date:** 2026-04-21
**Blog:** `docs/blog/2026-04-20-chrome-devtools-mcp/index.md`
**Status:** ✅ AUDIT COMPLETE — fixes applied directly, pending push auth to commit

---

## Keyword Gap Analysis

| Keyword | Target | Before | After | Status |
|---|---|---|---|---|
| AI agent browser control | P0 | 0 hits | ~6 uses | ✅ FIXED |
| MCP browser automation | P0 | 2 hits | ~7 uses | ✅ FIXED |
| browser automation governance | P1 | 0 hits | ~3 uses | ✅ FIXED |
| Chrome DevTools MCP | — | 22 hits | 22 hits | ✅ Already strong |

**Verdict:** All P0/P1 keywords now meaningfully integrated into the blog post copy. Natural density, no keyword stuffing.

---

## Technical SEO Fixes

### Frontmatter — FIXED
| Field | Before | After |
|---|---|---|
| `canonical` | Missing | Added |
| `og:title` | Missing | Added |
| `og:description` | Missing | Added |
| `og:image` | Missing | Added |
| `twitter:card` | Missing | Added |
| `twitter:title` | Missing | Added |
| `twitter:description` | Missing | Added |
| `author` | Missing | Added |

### Heading Structure — FIXED
| Issue | Fix |
|---|---|
| 4× H1 tags (code blocks) | ✅ Reduced to 1 H1, 6 H2s with anchor IDs |
| H2 anchor IDs | ✅ All H2s have anchor slugs |

### Structured Data — FIXED
- Article JSON-LD schema added (supports Google Discover)

---

## Infrastructure

| Asset | Status | Notes |
|---|---|---|
| robots.txt | ✅ Verified | Already present |
| sitemap.ts | ✅ Created | `canvas/src/app/sitemap.ts` — auto-generates sitemap |
| OG image template | ❌ Not created | Social Media Brand to create — see below |

---

## Post-Publish Checklist

- [ ] **Social Media Brand:** Create 1200×630 OG image template for Chrome DevTools MCP blog post
- [ ] **DevRel Engineer:** Write "headless Chrome MCP" tutorial (high interlink opportunity)
- [ ] **Content Marketer:** Write "MCP server list" explainer (interlink opportunity)
- [ ] **SEO Analyst:** Run Lighthouse audit after blog deploy to staging
- [ ] **SEO Analyst:** Submit blog to Google News (if applicable)
- [ ] **SEO Analyst:** Verify canonical URL resolves correctly after deploy

---

## Open Items for Marketing Lead

1. **Push auth:** Marketing Lead workspace cannot push to `molecule-ai/molecule-core` — `ghs_` token lacks git push scope. Fixes in this audit cannot be committed without classic PAT (`ghp_`) or bot write access. Filed as blocker.
2. **OG image:** Social Media Brand owns this. No template exists yet.
3. **Lighthouse audit:** Requires blog to be live on staging/prod. Schedule after first deploy.

---

## Interlink Opportunities

| Target Page | Anchor Text | Rationale |
|---|---|---|
| Headless Chrome MCP tutorial (DevRel) | "Chrome DevTools MCP" | Top-of-funnel from tutorial |
| MCP server list (Content Marketer) | "Chrome DevTools MCP" | List context, high authority |
| Security blog | "MCP governance layer" | Security-aware audience |

---

*Audit completed by SEO Analyst — Marketing Lead verified against repo on 2026-04-21*
