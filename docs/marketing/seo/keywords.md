<<<<<<< HEAD
# Molecule AI — SEO Keyword Briefs

> Active campaigns. Each section is self-contained. Stale sections should be marked `Status: superseded` rather than deleted.

---

=======
>>>>>>> origin/staging
# Chrome DevTools MCP — SEO Keyword Brief

**Campaign:** Phase 30 Chrome DevTools MCP SEO launch
**Date:** 2026-04-20
**Owner:** Marketing Lead + SEO Analyst
**Status:** Keywords confirmed — content live

## Primary Keywords (P0)

| Keyword | Intent | Target |
|---------|--------|--------|
| `MCP browser automation` | Informational / Tutorial | Blog post H1 + first 100 words |
| `Chrome DevTools MCP` | Informational / Product | Blog post H2 + meta description |

## Secondary Keywords (P1)

| Keyword | Intent | Target |
|---------|--------|--------|
| `AI agent browser control` | Informational | Blog body sections |
| `MCP protocol tutorial` | Tutorial / How-to | Blog post anchor sections |

## Keyword Strategy

- **P0 keywords** are locked. Both must appear in the blog post title, H1, and first 100 words.
- **P1 keywords** should appear naturally in body content and subheadings.
- Avoid generic marketing language in headings — this is a developer audience.

## Confirmed Deliverables

- **Brief:** `docs/marketing/briefs/2026-04-20-chrome-devtools-mcp-seo-brief.md`
<<<<<<< HEAD
- **Blog post:** `docs/blog/2026-04-20-chrome-devtools-mcp/index.md`
  > Note: brief originally referenced `docs/marketing/blog/...` path; actual shipped path is `docs/blog/...`. Both paths are live. Confirm canonical URL with DevRel.

## SEO Analyst Note

Chrome DevTools MCP blog H1 ("Browser Automation Meets Production Standards") does not contain a P0 keyword verbatim. Recommend adding "MCP browser automation" as a subtitle or alt-H1 to improve exact-match signal.

---

# Phase 30 Remote Workspaces GA — SEO Keyword Brief

**Campaign:** Phase 30 Remote Workspaces General Availability
**Date:** 2026-04-20
**Owner:** SEO Analyst
**Status:** Keywords confirmed — content live (GH#1126)

## Primary Keywords (P0)

| Keyword | Intent | Target |
|---------|--------|--------|
| `remote AI agent deployment` | How-to / Comparison | Blog post H1 + first 100 words |
| `self-hosted AI agent platform` | Informational / Comparison | Blog H2, meta description |
| `run AI agent on laptop` | Informational / Long-tail | Blog body, anchor links |

## Secondary Keywords (P1)

| Keyword | Intent | Target |
|---------|--------|--------|
| `AI agent multi-cloud orchestration` | Informational | Blog body sections |
| `federated AI agents` | Informational / Glossary | Blog body, architecture docs |
| `Molecule AI remote workspaces` | Brand + Product | Guide H1, blog H2 |

## Keyword Strategy

- **P0 keywords** are locked for the GA blog post. "Remote workspaces" is implicit in all Phase 30 content — do not use generic phrasing like "external agents" or "external runtime" in H1s.
- **P1 kw `federated AI agents`** aligns with PLAN.md Phase 30 framing. Use in body only — competitive landscape for this term is growing.
- Avoid "SaaS federation" in headings — low search intent, conflates two concepts.

## Confirmed Deliverables

- **GA blog post:** `docs/blog/2026-04-20-remote-workspaces/index.md` (slug: `remote-workspaces-ga`)
- **Decision guide blog:** `docs/blog/2026-04-20-container-vs-remote/index.md`
- **Remote Workspaces guide:** `docs/guides/remote-workspaces.md`
- **Remote Workspaces FAQ:** `docs/guides/remote-workspaces-faq.md`

## SEO Analyst Note

No dedicated landing page confirmed yet — coordinate with PMM (GH#1116) to determine whether a Phase 30 product page exists at `moleculesai.app/remote-workspaces`. If so, add a `landing-page` entry to this brief targeting the P0 keywords above.

---

# Phase 30 Container vs. Remote — SEO Keyword Brief

**Campaign:** Phase 30 — Container vs. Remote decision guide
**Date:** 2026-04-20
**Owner:** SEO Analyst
**Status:** Keywords confirmed — content live (GH#1126)

## Primary Keywords (P0)

| Keyword | Intent | Target |
|---------|--------|--------|
| `container vs remote AI agents` | Comparison / Decision | Blog post H1 (exact match preferred) |
| `AI agent runtime comparison` | Informational | Blog H2, meta description |

## Secondary Keywords (P1)

| Keyword | Intent | Target |
|---------|--------|--------|
| `AI agent fleet management` | Informational | Blog body |
| `Molecule AI remote workspaces` | Brand + Product | Blog body, CTA links |

## Keyword Strategy

- **P0 kw `container vs remote AI agents`** — this is an exact-match head term. The H1 "Container or Remote? How to Choose Your Agent Runtime in Molecule AI" is close but not exact. Consider adding "container vs remote AI agents" as a subtitle or intro paragraph lead.
- No dedicated brief file exists in `docs/marketing/briefs/` — brief is satisfied by this entry.

## Confirmed Deliverables

- **Blog post:** `docs/blog/2026-04-20-container-vs-remote/index.md` (slug: `container-vs-remote`)

---

# Phase 30 Secure by Design — SEO Keyword Brief

**Campaign:** Phase 30 auth hardening (org API keys, session auth, tenant isolation)
**Date:** 2026-04-20
**Owner:** SEO Analyst
**Status:** Keywords confirmed — content live (GH#1126)

## Primary Keywords (P0)

| Keyword | Intent | Target |
|---------|--------|--------|
| `AI agent org API keys` | Informational / How-to | Blog post H1 + first 100 words |
| `AI agent multi-tenant security` | Informational | Blog H2, meta description |

## Secondary Keywords (P1)

| Keyword | Intent | Target |
|---------|--------|--------|
| `AI agent audit trail` | Informational | Blog body sections |
| `multi-tenant AI platform` | Comparison | Blog body |

## Keyword Strategy

- **P0 kw `AI agent org API keys`** — this is a niche but high-intent product kw. The blog post's H1 focuses on "Secure by Design" framing rather than leading with this term. Surface `org API keys` in the first 100 words and in a visible subheading.
- Competitive landscape for `multi-tenant AI platform security` is growing — this brief positions Molecule AI before the field saturates.

## Confirmed Deliverables

- **Blog post:** `docs/blog/2026-04-20-secure-by-design/index.md` (slug: `beta-auth-hardening`)

---

# Same-Origin Canvas Fetches (/cp/* proxy) — SEO Keyword Brief

**Campaign:** Phase 30 technical architecture documentation
**Date:** 2026-04-20
**Owner:** SEO Analyst
**Status:** Keywords confirmed — content live (GH#1126)

## Primary Keywords (P0)

| Keyword | Intent | Target |
|---------|--------|--------|
| `Molecule AI Canvas` | Brand / Informational | Guide H1 |
| `AI agent canvas dashboard` | Informational | Guide H2, meta description |

## Secondary Keywords (P1)

| Keyword | Intent | Target |
|---------|--------|--------|
| `reverse proxy AI platform` | Technical / How-to | Guide body |
| `same-origin API proxy` | Technical | Guide body |

## Keyword Strategy

- This is primarily a technical reference guide, not an organic acquisition target. P0 keywords are brand-adjacent.
- **Action required:** Add a `description:` frontmatter field to `docs/guides/same-origin-canvas-fetches.md` before publishing. Currently missing — search engines will auto-generate from first paragraph. Recommended: *"Learn how Molecule AI's /cp/* reverse proxy lets Canvas make same-origin browser API calls to both tenant and control plane backends — without CORS or cookie domain issues."*

## Confirmed Deliverables

- **Guide:** `docs/guides/same-origin-canvas-fetches.md`
=======
- **Blog post:** `docs/marketing/blog/2026-04-20-how-to-add-browser-automation-to-ai-agents-with-mcp.md`

## SEO Analyst Note

SEO Analyst reported 6 campaign actions complete. File paths `docs/blog/...` and `docs/marketing/seo/keywords.md` — the latter is now confirmed real (this file). The `docs/blog/...` path has been superseded by the confirmed `docs/marketing/blog/...` location. All other SEO Analyst deliverables should be verified before treating as complete.
>>>>>>> origin/staging
