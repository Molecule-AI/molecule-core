# Molecule AI — SEO Keyword Briefs

> Active campaigns. Each section is self-contained. Stale sections should be marked `Status: superseded` rather than deleted.

---

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

---

# ⚠️ PRE-POSITIONING — Do Not Publish Until Feature Ships

# External Workspaces Polling Mode — SEO Keyword Brief

**⚠️ Status:** PRE-POSITIONING. Do NOT create blog content or commit SEO-optimized copy until `molecule-external-agent` Python library ships on PyPI and `/docs/guides/external-workspaces-polling.md` is published. Feature is in draft (2026-04-21).
**Campaign:** Phase 30 extension — External Workspaces Polling Mode
**Date:** 2026-04-21
**Owner:** SEO Analyst
**Status:** Pre-positioning brief — feature not yet shipped

## Background (read before writing)

External workspaces (Phase 30) require agents to expose a public HTTP endpoint. This works for servers but fails for laptops behind NAT, corporate firewalls, or consumer ISPs. Polling mode flips the pattern: the agent makes outbound HTTPS calls only — long-polling an inbox endpoint for inbound messages, posting outbound A2A the same way. No port forwarding. No public URL. No NAT holes. Ships as `molecule-external-agent` on PyPI + `/docs/guides/external-workspaces-polling.md`.

**Design doc:** `Molecule-AI/internal/product/external-workspaces-polling.md` (SHA draft, 2026-04-21)

## Primary Keywords (P0)

| Keyword | Intent | Target | Notes |
|---------|--------|--------|-------|
| `run AI agent on laptop` | Informational / How-to | Blog H1 — exact match | Core user story: "anyone with a laptop and Python 3.10+ can be running an external workspace in < 5 minutes" |
| `external AI agent` | Informational / Product | Blog H2 + meta description | Key differentiator vs. cloud-only platforms |
| `AI agent behind firewall` | Informational / How-to | Blog body + anchor | Solves the exact problem competitors can't |

## Secondary Keywords (P1)

| Keyword | Intent | Target | Notes |
|---------|--------|--------|-------|
| `remote AI agent without VPN` | Informational | Blog body | Core value prop — no infrastructure required |
| `laptop AI agent` | Informational / Long-tail | Blog body | Casual/daily-driver user persona |
| `AI agent cross-network` | Informational | Blog body | Cross-cloud A2A angle |
| `self-hosted AI agent laptop` | Comparison / How-to | Blog body | Self-hosted angle from laptop persona |

## Keyword Strategy

- **P0 kw `run AI agent on laptop`** — this is the head term. Exact-match H1 required. Strong search intent from developers who want to run agents on personal hardware without cloud dependency or infrastructure setup. Competitive gap: most "run AI agent locally" content focuses on Ollama/LM Studio (local model inference), not agent platform. This is a distinct angle.
- **P0 kw `external AI agent`** — product-category kw. Correlates with Phase 30 Remote Workspaces. Do not conflate with "remote agent" (which could mean cloud VM). Use `external` precisely to mean "not on-platform, registered via A2A."
- **P0 kw `AI agent behind firewall`** — solves a real pain. Enterprise/dev users blocked by corporate networks are a high-value audience. Content should demonstrate the specific problem and the 3-line `pip install + molecli ws create + chat` solution.
- Competitive landscape: most AI agent platforms assume cloud-hosted agents. "Laptop agent" and "behind firewall" angles have low competition from other agent platform vendors — this is a genuine SEO gap.

## Content Angle

Lead with the problem: "Your AI agent can't run on a laptop because it needs a public URL." Pivot to the solution: "Now it doesn't." Sub-5-minute setup story is the hook. Code sample: `pip install molecule-external-agent && molecli ws create`. Show the agent appearing on the Canvas with a purple REMOTE badge.

## Blog Post Candidate Slug

`docs/blog/2026-04-XX-external-workspaces-polling/index.md` (slug: `run-ai-agent-on-your-laptop`)

## Confirmed Deliverables (pending ship)

- **Blog post:** `docs/blog/2026-04-XX-external-workspaces-polling/index.md` (slug: `run-ai-agent-on-your-laptop`)
- **Keyword brief:** this entry
- **Trigger:** ship `molecule-external-agent` on PyPI + publish `/docs/guides/external-workspaces-polling.md`

## SEO Analyst Note

**Wait to publish.** This brief is pre-positioning only. Publishing content before the feature ships creates 404 risk if the feature is delayed or renamed. Add to content pipeline when engineering confirms ship date.
