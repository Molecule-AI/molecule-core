# Phase 34 SEO Brief — GA April 30, 2026
**Author:** SEO Analyst
**Date:** 2026-04-23
**Status:** Draft — for PMM review before GA

---

## Product Summary

Phase 34 ships four features:
1. **Tool Trace** — structured, chronological record of every tool call in every A2A response (`Message.metadata.tool_trace`)
2. **Platform Instructions** — global/workspace-scoped governance rules prepended to the system prompt at workspace startup
3. **Partner API Keys** — scoped, rate-limited, revocable bearer tokens (`mol_pk_`) for marketplace resellers, CI/CD pipelines, and automation platforms
4. **Audit Chain Verification** — cryptographic audit chain verification (referenced from Phase 30 audit trail)

Published posts (4):
- `docs/blog/2026-04-23-tool-trace-observability/`
- `docs/blog/2026-04-23-tool-trace-platform-instructions/`
- `docs/blog/2026-04-23-partner-api-keys/`
- `docs/blog/2026-04-23-platform-instructions-governance/`

---

## Keyword Strategy

### Primary Keyword Recommendation

| Keyword | Justification |
|---|---|
| **`AI agent observability`** | Highest strategic value. Targets the platform engineering + DevOps buyer who needs to monitor, debug, and audit agent behavior in production. "Observability" is the established enterprise framing — not "tool trace" or "audit trail" which are feature-specific. Intent: commercial/informational. Confirmed by Phase 34 post title: *AI Agent Observability Without the Overhead*. |

**Why not "agent observability platform" or "AI agent debugging"?**
- "AI agent observability" is the root intent — broader reach, matches both "how do I observe agents?" and "what tools exist for agent monitoring?"
- "Platform" adds noise and reduces CTR for technical buyers who want a feature, not a vendor category
- "Debugging" skews informational/developer rather than enterprise procurement

### Secondary / LSI Keywords

| Keyword | Intent | Rationale |
|---|---|---|
| `AI agent debugging` | Informational | Developer-aligned variant; captured by the Tool Trace post |
| `agent governance policy` | Commercial | Targets IT/security buyers; Platform Instructions post covers this |
| `partner API keys` | Commercial | Brand-adjacent; unique differentiation — no competitor has "Partner API Keys" as a named feature |
| `AI agent audit trail` | Informational | Connects Tool Trace + org-scoped API keys from Phase 30; captures compliance searchers |
| `AI agent monitoring` | Informational | Broader than observability; Platform Instructions + Tool Trace post covers both |

---

## On-Page SEO

### Recommended Page Title (≤60 chars)

```
AI Agent Observability — Tool Trace & Governance | Molecule AI
```

**Count:** 57 characters. ✅

> Alternative: `AI Agent Observability & Fleet Governance | Molecule AI` (58 chars)

### Recommended Meta Description (≤160 chars)

```
Tool Trace shows every agent tool call in real-time. Platform Instructions enforces governance at the system prompt level. Enterprise AI observability, built in. → Learn more
```

**Count:** 150 characters. ✅

---

## H1 + H2 Structure (Landing Page)

```
H1: AI Agent Observability Without the Overhead

H2: What Tool Trace Captures
     — tool, input, output_preview, run_id

H2: Parallel Calls, Traced Correctly
     — run_id pairing for concurrent tool calls

H2: Built In, Not Bolt-On
     — no sidecar, no sampling, no new infrastructure

H2: Platform Instructions: Governance Before the First Turn
     — global + workspace scopes

H2: Enterprise-Grade Auditability
     — org API key attribution → tool trace → complete chain

H2: Get Started
     — docs links, plan availability
```

---

## Content Gaps vs. SERP Competitors

### "AI agent observability" — Top 3 Competitor Gaps

**Competitors:** LangSmith (CipherLabs/LangChain), Honeycomb AI Observability, Arize AI

| Gap | Competitor Has | Molecule AI Current State |
|---|---|---|
| **"Why observability" opener** | All three open with a pain/problem statement ("Agents fail in production silently") | Tool Trace post jumps straight into the feature — no intro framing |
| **Quickstart / "Getting started in 3 steps"** | LangSmith has a 5-min quickstart | No equivalent section; "Get Started" is thin |
| **Benchmarks / performance data** | Honeycomb publishes latency/effort benchmarks | No performance claims |
| **Use-case drill-down by persona** | Arize has separate pages per use case (LLM ops, compliance, debugging) | Single post, all personas collapsed |

**Quick fix:** Add a 2-3 sentence intro to the Tool Trace post that names the pain before describing the solution.

---

### "Partner API keys" — Top 3 Competitor Gaps

**Competitors:** Stripe (partner connections), GitHub Apps (token-based integrations), Zapier (OAuth partner model)

| Gap | Competitor Has | Molecule AI Current State |
|---|---|---|
| **Token prefix explanation** | Stripe/Stripe Docs: `sk_live_`/`rk_live_` documented with security rationale | `mol_pk_` prefix shown but not explained ("what makes this key different from a regular API key?") |
| **Enterprise pricing/plan badge visible** | Stripe shows plan tier on each API feature page | Phase 34 posts hide plan availability in footer |
| **Marketplace-specific copy** | GitHub Apps: dedicated "Building a GitHub App" guide with marketplace framing | Partner API Keys post uses generic CI/CD framing; marketplace reseller section is thin |

**Quick fix:** Add a "Why `mol_pk_`?" callout explaining the token type distinction from org-scoped keys.

---

## Recommended Internal Links (from existing docs)

| From Phase 34 Post | Target | Anchor Text | Rationale |
|---|---|---|---|
| Tool Trace observability | `/docs/blog/2026-04-21-org-scoped-api-keys/` | "org-scoped API key audit trail" | Closes the trace chain: org key → workspace → agent → tool calls |
| Tool Trace observability | `/docs/api-protocol/a2a-protocol.md` | "A2A protocol documentation" | Deep-link for implementers who want the full spec |
| Platform Instructions | `/blog/ai-agent-observability-without-overhead/` | "Tool Trace" | Cross-link between the two Phase 34 features |
| Partner API Keys | `/docs/blog/2026-04-21-org-scoped-api-keys/` | "org-scoped API keys" | Contrast/differentiation section in Partner API Keys post |

---

## Lighthouse / Core Web Vitals Note

> **Note:** Lighthouse audit requires a live staging URL. The docs site build/deploy pipeline was not accessible from this environment (`/docs/.vitepress/` not found, staging URL not in `package.json`). A live CWV audit should be run by DevRel or the deploy pipeline owner before GA.

### Known CWV Risk Flags (from Phase 30 audit pattern)

- **`og_image` missing** on all 4 Phase 34 posts — this will cause Facebook/LinkedIn card fallbacks, which may be penalized in social sharing previews. Fix: add `og_image` field to all 4 frontmatter blocks.
- **No images with alt text** in Phase 34 posts — current posts have no images, so alt text is not a current risk, but if any diagrams are added before GA, alt text is required.
- **JSON-LD schema present** on all 4 posts — ✅ structurally correct `Article` schema with `headline`, `description`, `author`, `datePublished`, `publisher`.

---

## Quick Wins Checklist (applies to all Phase 34 posts)

| # | Fix | Files |
|---|---|---|
| ☐ | Add `og_image` to frontmatter (e.g. `og_image: /assets/phase34-tool-trace.png`) | All 4 Phase 34 posts |
| ☐ | Internal link: Partner API Keys → org-scoped API keys post | `docs/blog/2026-04-23-partner-api-keys.md` |
| ☐ | Internal link: Tool Trace platform instructions → Tool Trace observability post | `docs/blog/2026-04-23-tool-trace-platform-instructions/index.md` |
| ☐ | Add 2-sentence "why" opener to Tool Trace observability post | `docs/blog/2026-04-23-tool-trace-observability/index.md` |
| ☐ | Add "Why `mol_pk_`?" callout to Partner API Keys post | `docs/blog/2026-04-23-partner-api-keys.md` |

---

## Top 3 Keyword Recommendations (Summary)

1. **`AI agent observability`** — primary, commercial/informational, enterprise platform buyer
2. **`AI agent debugging`** — secondary, informational, developer/ops persona
3. **`partner API keys`** — secondary, commercial, brand-differentiated, marketplace/CI-CD buyer
