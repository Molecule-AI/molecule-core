# Phase 34 Launch — SEO Brief
**Date:** 2026-04-23  
**Author:** Marketing Lead (drafted directly — SEO Analyst workspace looping)  
**GA Date:** April 30, 2026  
**Features:** Tool Trace, Platform Instructions, Partner API Keys (`mol_pk_*`), SaaS Fed v2

---

## Keyword Research

### Cluster A — Agent Observability / Tool Tracing

**Primary keyword:** `agent observability`  
Search intent: Informational + commercial investigation. Builders evaluating how to monitor multi-agent systems in production. High technical sophistication. Growing volume as LLM agent frameworks mature (LangSmith, Langfuse, Helicone driving awareness of the category).

**Supporting LSI keywords:**
1. `multi-agent tracing` — captures the A2A-specific layer; low competition, highly specific to Molecule's positioning
2. `LLM tool call logging` — transactional intent, developers searching for how to log specific tool invocations
3. `agent execution trace` — technical variant, aligns directly with `tool_trace` field name

**Competition level:** Medium. LangSmith and Langfuse dominate "LLM observability" broadly. Molecule's differentiator is **A2A-native, zero-integration** tracing — angle into the gap with "agent observability without a third-party SDK."

**Avoid:** "LLM observability" as primary — Langfuse/Datadog own it. Target the agent-behavior layer specifically.

---

### Cluster B — Partner API Provisioning

**Primary keyword:** `agent platform API`  
Search intent: Commercial investigation. Platform engineers and marketplace builders evaluating whether an agent framework exposes programmable org lifecycle management.

**Supporting LSI keywords:**
1. `programmatic org provisioning` — exact-match for the mol_pk_* use case; low competition, high buying intent
2. `multi-tenant agent platform` — captures the reseller/marketplace angle
3. `partner API integration` — broader but captures the ecosystem builder ICP

**Competition level:** Low–medium. No competitor is ranking for "partner API provisioning" in the agent orchestration context — genuine first-mover SEO window ahead of April 30 GA.

---

## On-Page SEO Brief — Tool Trace Blog Post

**Target file:** `docs/marketing/blog/2026-04-23-tool-trace-platform-instructions.md`  
*(Note: file not yet written as of 2026-04-23 — apply these specs when Content Marketer delivers)*

| Element | Recommendation |
|---------|---------------|
| **Title tag** | `Agent Observability Built In: Tool Trace + Platform Instructions` (60 chars) |
| **Meta description** | `Molecule AI now records every tool call your agents make — name, input, output preview — with zero SDK setup. Plus org-level Platform Instructions.` (150 chars) |
| **H1** | `Molecule agents now ship with built-in execution tracing and governance` |
| **Slug** | `/blog/agent-observability-tool-trace-platform-instructions` |
| **OG image** | Generate with feature name + "Built-in. No SDK." tagline |

**Internal linking targets (link FROM new post TO these):**
- `docs/blog/2026-04-21-cloudflare-artifacts/` — cloud-native platform angle
- `docs/blog/2026-04-22-a2a-v1-agent-platform/` — A2A architecture context
- `docs/marketing/launches/pr-1105-org-scoped-api-keys.md` — auth layer context (org keys → tool trace → platform instructions ladder)

**Link TO new post FROM:**
- `docs/blog/2026-04-22-a2a-v1-agent-platform/` — add a "→ See also: Tool Trace for A2A observability" callout
- Any future Partner API Keys post — Tool Trace is a prerequisite story for the partner platform narrative

---

## April 30 Launch SEO Checklist

### Pages needing og:image / meta desc updates
- [ ] `docs/blog/2026-04-21-cloudflare-artifacts/index.md` — og:image path fix already committed (PR #1899); verify meta desc present
- [ ] `docs/blog/2026-04-22-a2a-v1-agent-platform/index.md` — slug updated to `a2a-v1-agent-platform` (pipeline #15 ✅); confirm meta desc ≤155 chars
- [ ] Tool Trace blog post (when written) — apply title/meta from table above
- [ ] Partner API Keys GA announcement page — needs dedicated og:image with `mol_pk_*` branding

### Cross-linking before April 30
1. Add "Phase 34 ships April 30" callout to `docs/blog/2026-04-22-a2a-v1-agent-platform/` sidebar or footer
2. Ensure Discord adapter post links to Phase 34 announcement once it publishes
3. After Tool Trace blog post lands, back-link from A2A v1 post

### Core Web Vitals notes
- Blog template: no known LCP issues from Lighthouse audit (see `docs/marketing/seo/lighthouse-audit-chrome-devtools-mcp-2026-04-22.md` for baseline)
- Watch: TTS audio embeds in new blog posts — lazy-load audio players, don't autoplay on load
- Watch: og:image generation — ensure images are ≤200KB and served via CDN, not inline

---

## Keyword Gaps to Address (Next 30 Days)

| Gap | Recommended Post | Priority |
|-----|-----------------|----------|
| "agent platform for marketplaces" | Partner API Keys deep-dive (after Apr 30 GA) | High |
| "multi-tenant LLM platform" | Case study: ephemeral test orgs per PR | High |
| "MCP server governance" | MCP server list explainer (#1493) | Medium |
| "A2A protocol enterprise" | A2A enterprise deep-dive (#1492) | Medium |

---

*Marketing Lead 2026-04-23. SEO Analyst to review and extend with live keyword volume data when workspace recovers.*
