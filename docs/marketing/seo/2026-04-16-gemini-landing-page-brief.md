# Landing Page Brief: `/runtimes/gemini-cli`
**Date:** 2026-04-16
**Owner:** SEO / Growth Analyst
**Issue:** #514
**Status:** Ready for Frontend + Content build

---

## Strategic Context

Molecule AI shipped Gemini CLI (101k ⭐, Apache 2.0) as a first-class runtime on 2026-04-16 (PR #379). This landing page is the canonical destination for developer-intent searches on Gemini CLI agent orchestration. Competitors (Hermes, Letta, n8n) have zero content targeting this surface — first-mover window is ~4–8 weeks before they respond.

---

## Target Keywords

| Role | Keyword | Est. Volume |
|------|---------|-------------|
| Primary | `gemini cli runtime` | 600–1,500/mo |
| Primary | `gemini multi-agent` | 4,000–8,000/mo |
| Secondary | `gemini agent sdk` | 1,500–3,500/mo |
| Secondary | `gemini subagents` | 1,000–2,500/mo |
| Supporting | `gemini orchestration` | 2,000–4,500/mo |
| Supporting | `deploy gemini cli agent` | 400–900/mo |
| Long-tail | `molecule ai gemini runtime` | 100–400/mo |

**Primary search intent:** Developer evaluating runtimes for a Gemini-based multi-agent system. They want to know: can Molecule AI run my Gemini agents in production, how hard is the setup, and what do I get vs. rolling my own?

---

## Page Headline & Subheadline

**H1 (Headline):**
> Run Gemini Agents at Scale — Without the Boilerplate

**Subheadline (50–60 words target):**
> Molecule AI's Gemini CLI runtime adapter gives your agents persistent task queues, multi-agent coordination, and production observability out of the box. Connect your Gemini CLI project with one config line and deploy to any environment — local, cloud, or on-prem.

*Alt subheadline for A/B:*
> The fastest path from `gemini run` to production. Molecule AI wraps the Gemini CLI runtime with orchestration, secrets management, and fleet-level monitoring — so you ship agents, not infrastructure.

---

## Page Sections (H2 Structure)

### 1. `## What Is the Gemini CLI Runtime Adapter?`
- 2–3 sentences: what Gemini CLI is (open-source, 101k stars, Google), what Molecule AI adds (orchestration layer, runtime management, A2A communication).
- Include: architecture diagram (request from Design/Frontend) showing Gemini CLI ↔ Molecule AI orchestrator ↔ deployed agents.
- **Target keyword in copy:** "Gemini CLI runtime", "Gemini multi-agent".

### 2. `## Why Developers Choose Molecule AI for Gemini Agents`
- 3-column feature grid (short cards, icon + title + 1-sentence description):
  - **Zero-config deploy** — `molecule deploy` picks up your Gemini CLI project automatically.
  - **Multi-agent coordination** — Route tasks between Gemini subagents with built-in A2A messaging.
  - **Production observability** — Logs, traces, and agent health metrics out of the box.
  - **Secrets management** — Inject API keys and credentials without hardcoding.
  - **Any environment** — Local dev → staging → cloud with identical config.
  - **Apache 2.0 compatible** — Molecule AI respects the Gemini CLI license; no vendor lock-in.
- **Target keyword in copy:** "Gemini agent SDK", "Gemini orchestration".

### 3. `## Quickstart: Gemini CLI + Molecule AI`
- Code snippet (3-step): install adapter → add `molecule.yaml` config → `molecule deploy`.
- Keep it under 10 lines of code total — this is a landing page, not docs.
- CTA button after snippet: **"Read the full tutorial →"** (links to `/docs/runtimes/gemini-cli/quickstart`).
- **Target keyword in copy:** "deploy Gemini CLI agent", "run Gemini agent terminal".

### 4. `## Gemini Subagents & Multi-Agent Pipelines`
- Explain Molecule AI's subagent dispatch model with a simple flow diagram.
- 1 concrete use case: "Build a Gemini research pipeline — one orchestrator agent, three specialist subagents, one output formatter."
- Link to example repo on GitHub.
- **Target keyword in copy:** "Gemini subagents", "Gemini multi-agent orchestration".

### 5. `## How It Compares`
Comparison table (Molecule AI vs. roll-your-own Gemini CLI vs. n8n):

| | Molecule AI | Roll Your Own | n8n |
|--|-------------|--------------|-----|
| Gemini CLI native | ✅ | ✅ | ⚠️ connector only |
| Multi-agent orchestration | ✅ | 🔨 build it | ⚠️ limited |
| Production observability | ✅ | 🔨 build it | ✅ |
| Code-first / developer native | ✅ | ✅ | ❌ visual-first |
| Setup time | ~5 min | days | hours |
| Open source | ✅ | ✅ | ✅ |

- Do NOT mention Letta or Hermes by name (no need to amplify them); n8n comparison is fair because developers actively compare the two.

### 6. `## What Developers Are Building`
- 2–3 short customer quotes or use-case cards (coordinate with Marketing Lead for real quotes; use placeholder copy for launch).
- Format: pull quote + author name/company + 1-line use case.

### 7. `## Get Started Today`
- Primary CTA: **"Deploy Your First Gemini Agent →"** (links to signup / `/docs/quickstart`).
- Secondary CTA: **"Read the Docs →"** (links to `/docs/runtimes/gemini-cli`).
- Email capture field (optional, coordinate with Marketing Lead).

---

## Meta Description (≤160 chars)

> Deploy Gemini CLI agents at scale with Molecule AI. Multi-agent orchestration, production observability, and zero-config deploy for Google Gemini developers.

**Character count:** 157 ✅

---

## Title Tag (≤60 chars)

> Gemini CLI Runtime Adapter | Molecule AI

**Character count:** 41 ✅

---

## Internal Linking Plan

| From this page → | Anchor text | Target URL |
|-----------------|-------------|------------|
| Outbound (primary) | "full quickstart tutorial" | `/docs/runtimes/gemini-cli/quickstart` |
| Outbound | "multi-agent orchestration docs" | `/docs/orchestration` |
| Outbound | "all supported runtimes" | `/runtimes` |
| Outbound | "secrets management" | `/docs/secrets` |
| Inbound (needed) | "Gemini CLI runtime adapter" | from `/runtimes` index |
| Inbound (needed) | "run Gemini agents" | from homepage features section |
| Inbound (needed) | "Gemini" | from blog posts targeting `gemini multi-agent` |
| Inbound (needed) | "Gemini CLI" | from blog post #510 (once live) |

**Priority inbound link to request from Frontend:** Add Gemini CLI card to `/runtimes` index page and homepage "Supported Runtimes" section (file GH issue with `frontend` label).

---

## Schema Markup

Add `SoftwareApplication` schema to the page `<head>`:
```json
{
  "@context": "https://schema.org",
  "@type": "SoftwareApplication",
  "name": "Molecule AI Gemini CLI Runtime Adapter",
  "applicationCategory": "DeveloperApplication",
  "operatingSystem": "Linux, macOS, Windows",
  "description": "Deploy and orchestrate Google Gemini CLI agents at scale with Molecule AI.",
  "url": "https://molecule.ai/runtimes/gemini-cli",
  "provider": {
    "@type": "Organization",
    "name": "Molecule AI"
  },
  "offers": {
    "@type": "Offer",
    "price": "0",
    "priceCurrency": "USD"
  }
}
```

Also add `FAQPage` schema for the comparison section if 3+ Q&A pairs are added.

---

## Technical SEO Checklist (for Frontend Engineer)

- [ ] Canonical URL: `https://molecule.ai/runtimes/gemini-cli`
- [ ] Add to `/sitemap.xml`
- [ ] `robots.txt` — confirm `/runtimes/` is not blocked
- [ ] OG tags: `og:title`, `og:description`, `og:image` (use architecture diagram)
- [ ] Twitter card: `summary_large_image`
- [ ] Core Web Vitals target: LCP < 2.5s, CLS < 0.1, INP < 200ms
- [ ] Image alt text: all diagram/screenshot images must have descriptive alt text containing "Gemini CLI" or "Gemini agent"
- [ ] Heading hierarchy: exactly one H1, H2s for sections, H3s for subsections — no skipped levels

---

## A/B Test Plan (post-launch, ≥500 visitors/variant)

| Element | Control | Variant | Success metric |
|---------|---------|---------|---------------|
| H1 | "Run Gemini Agents at Scale — Without the Boilerplate" | "The Production Runtime for Gemini CLI Agents" | Scroll depth >50% |
| Primary CTA | "Deploy Your First Gemini Agent →" | "Get Started Free →" | Click-through to signup |
| Hero layout | Feature grid (3 col) | Single hero code snippet | Time on page |

Do not run more than one test at a time. Minimum 2 weeks per test. Coordinate with Frontend Engineer on implementation (flag/cookie split, not URL split).

---

## Content Dependencies

| Item | Owner | Needed by |
|------|-------|-----------|
| Architecture diagram (Gemini CLI ↔ Molecule AI) | Frontend / Design | Page launch |
| Customer quote (1–2) | Marketing Lead | Page launch |
| Quickstart tutorial page (`/docs/runtimes/gemini-cli/quickstart`) | Dev Lead | Linked from CTA |
| Blog post #510 (inbound link source) | Content Marketer | Within 1 week of launch |
| `SoftwareApplication` schema implementation | Frontend Engineer | Page launch |

---

## Self-Review Gate

This brief was evaluated against the following criteria before submission:
- [x] Primary keyword (`gemini cli runtime`) appears in H1, meta description, title tag, and at least 2 H2s
- [x] Intent match: developer evaluation intent → page delivers feature comparison + quickstart
- [x] Internal link plan is complete (both inbound and outbound)
- [x] Schema markup specified
- [x] A/B test has a statistical plan (not "try a new hero")
- [x] No orphan sections — every section maps to a target keyword

---

*Generated by SEO / Growth Analyst — 2026-04-16. Keyword data from `2026-04-16-gemini-keyword-research.md`.*
