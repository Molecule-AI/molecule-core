# Phase 34 GA Launch SEO Brief
**Campaign:** Phase 34 GA — GA date April 30, 2026
**Author:** SEO Analyst (5b277fc4)
**Date:** 2026-04-23
**Status:** Draft — for April 30 launch readiness review
**Web search:** Unavailable at time of writing (API error); keyword estimates based on internal knowledge + prior research cycles.

> **Note on posts:** Three Phase 34 posts are already published (2026-04-23). This brief covers on-page SEO gaps, keyword cluster confirmation, and launch-day checklist for the April 30 GA push. Live post slugs differ from brief drafts — confirmed actual slugs used throughout.

---

## 1. Published Phase 34 Posts

| Post | Slug | Status | og_image |
|------|------|--------|----------|
| "AI Agent Observability Without the Overhead" | `ai-agent-observability-without-overhead` | ✅ Published 2026-04-23 | ✅ `/assets/blog/2026-04-23-tool-trace-observability/og.png` |
| "Govern Your AI Fleet at the System Prompt Level" | `govern-ai-fleet-system-prompt-level` | ✅ Published 2026-04-23 | ✅ `/assets/blog/2026-04-23-platform-instructions-governance/og.png` |
| "Ship Partner Integrations Faster with Programmatic Org Management" | `partner-api-keys` | ✅ Published 2026-04-23 | ❌ **MISSING** — action required |

---

## 2. Keyword Clusters

### Cluster A: AI Agent Observability

**Primary keyword:** `AI agent observability`
**Intent:** Informational / Commercial investigation
**Est. volume signal:** Medium — active category with growing search behavior as agent adoption matures
**Competition:** Arize AI, LangChain, Honeycomb, Humanloop — mostly LLM-level observability; **no dominant agent-tool-level result yet**. First-mover opportunity.

| Keyword | Intent | Priority | Notes |
|---------|--------|----------|-------|
| `AI agent observability` | Commercial / Informational | **P0** | Head term. Must own this by GA. |
| `AI agent debugging` | How-to / Informational | **P0** | Developer audience, high intent. |
| `AI agent tool tracing` | Informational | **P1** | Precise; near-zero competition. Own it. |
| `LLM tool call logging` | Informational | **P1** | Overlaps with observability cluster. Cross-link target. |
| `AI agent action audit` | Informational / Commercial | **P1** | Compliance angle for enterprise buyers. |
| `agentic AI debugging tools` | Informational | **P2** | Long-tail; build content around "debugging tools" list format. |
| `AI agent execution trace` | Informational | **P2** | Technical audience. Code-adjacent queries. |
| `A2A protocol observability` | Informational | **P2** | Ownable — A2A-specific angle competitors can't claim. |

**LSI keywords for `ai-agent-observability-without-overhead` post:**
`tool trace`, `run_id pairing`, `parallel tool calls`, `A2A response metadata`, `agent debugging workflow`

---

### Cluster B: Partner API Provisioning / Agent Platform API

**Primary keyword:** `partner API provisioning`
**Intent:** Commercial investigation — enterprise B2B
**Est. volume signal:** Low-medium — niche but high-value clicks from enterprise buyers
**Competition:** No dedicated "agent platform partner API" content. Generic API management content (Stronghold, Moesif, 3scale) does not address the agent-platform context.

| Keyword | Intent | Priority | Notes |
|---------|--------|----------|-------|
| `partner API provisioning` | Commercial | **P0** | Core enterprise angle. |
| `agent platform API` | Commercial / Informational | **P0** | Broad platform adoption term. |
| `programmatic org management` | Informational | **P1** | Already in post description. |
| `API key management platform` | Commercial | **P1** | Enterprise buyer keyword. |
| `multi-tenant API provisioning` | Informational | **P1** | Technical enterprise audience. |
| `reseller API integration` | Commercial | **P2** | Marketplace/reseller angle. |
| `CI/CD API automation` | How-to | **P2** | DevOps angle for partner keys. |

**LSI keywords for `partner-api-keys` post:**
`scoped API keys`, `rate-limited API keys`, `revocable API keys`, `marketplace integration`, `org provisioning API`

---

## 3. On-Page SEO Briefs

> **Note:** The post at `docs/blog/2026-04-23-tool-trace-platform-instructions.md` was not found. The two actual observability/governance posts are at `docs/blog/2026-04-23-tool-trace-observability/` and `docs/blog/2026-04-23-platform-instructions-governance/`. Briefs below reference actual live posts.

### 3a. `ai-agent-observability-without-overhead` — Tool Trace

| Element | Current state | Recommendation |
|---------|---------------|----------------|
| **Title tag** | "AI Agent Observability Without the Overhead" (46 chars) | ✅ Acceptable — within ≤60. Consider appending "— Molecule AI" if SERP CTR needs boost. |
| **Meta description** | "Tool Trace gives every A2A response a structured record of every tool call — inputs, output previews, run_id-paired parallel traces. No sampling, no sidecar, no guesswork." (~190 chars) | ⚠️ Trim to ≤155 chars: "Tool Trace: see every tool your AI agent calls — inputs, outputs, timing — in every A2A response. No sampling, no sidecar. →" (152 chars) |
| **H1** | "AI Agent Observability Without the Overhead" | ✅ Keep. |
| **Slug** | `ai-agent-observability-without-overhead` | ✅ Good. Descriptive, readable. |
| **og_image** | ✅ Present at `/assets/blog/2026-04-23-tool-trace-observability/og.png` | ✅ Confirmed. |
| **Internal links** | Not reviewed — check for links to `chrome-devtools-mcp`, `remote-workspaces`, `org-scoped-api-keys` | Add at least 3 internal links in body. |
| **keywords frontmatter** | ✅ Present | ✅ Expand to explicitly include `LLM tool call logging` and `A2A protocol observability`. |

**Recommended frontmatter additions:**
```yaml
keywords: [AI agent observability, tool trace debugging, Claude agent debugging, agent audit trail,
  parallel tool call trace, run_id pairing, AI agent monitoring, DevOps agent observability,
  LLM tool call logging, A2A protocol observability]
```

---

### 3b. `partner-api-keys` — Partner API Keys

| Element | Current state | Recommendation |
|---------|---------------|----------------|
| **Title tag** | "Ship Partner Integrations Faster with Programmatic Org Management" (~71 chars) | ⚠️ Trim to ≤60: "Partner API Keys: Programmatic Org Management for AI Platforms" (57 chars) |
| **Meta description** | "Partner API Keys let marketplace resellers, CI/CD pipelines, and automation tools create and manage Molecule AI orgs via API — no browser session required." (~189 chars) | ⚠️ Trim to ≤155 chars: "Partner API Keys for AI platforms: scoped, rate-limited, revocable keys for marketplace, CI/CD, and reseller integrations. →" (143 chars) |
| **H1** | "Ship Partner Integrations Faster with Programmatic Org Management" | ✅ Keep or shorten to match title tag. |
| **Slug** | `partner-api-keys` | ✅ Good. |
| **og_image** | ❌ **MISSING** | 🚨 **Action required before GA April 30.** Social Media Brand to create 1200×630 OG image. |
| **og_title** | "Ship Partner Integrations Faster with Programmatic Org Management" | ⚠️ Trim to ≤60 or rewrite. |
| **og_description** | "Partner API Keys: scoped, rate-limited, revocable API keys for programmatic org management. Built for marketplaces, CI/CD, and automation platforms." (153 chars) | ⚠️ Trim to ≤97 chars: "Partner API Keys: scoped, rate-limited, revocable. For AI marketplaces and CI/CD." (89 chars) |
| **keywords frontmatter** | ✅ Present | ✅ Expand to include `partner API provisioning`, `agent platform API`. |

**Recommended frontmatter fixes:**
```yaml
title: "Partner API Keys: Programmatic Org Management for AI Platforms"
og_title: "Partner API Keys for AI Platforms"
og_description: "Partner API Keys: scoped, rate-limited, revocable. For AI marketplaces and CI/CD."
og_image: /assets/blog/2026-04-23-partner-api-keys/og.png
keywords: [partner API keys, programmatic org management, marketplace integration, CI/CD automation,
  Molecule AI API, reseller integration, org provisioning API, partner API provisioning,
  agent platform API, scoped API keys]
```

---

## 4. Cross-Linking Opportunities (April 30 Launch)

The three Phase 34 posts have not been cross-linked to each other or to Phase 30 posts. Inject these links at GA:

| From | To | Anchor text | Priority |
|------|----|-------------|----------|
| `ai-agent-observability-without-overhead` | `partner-api-keys` | "org-scoped API keys" or "programmatic org management" | High |
| `ai-agent-observability-without-overhead` | `a2a-v1-agent-platform` | "A2A response metadata" | High |
| `ai-agent-observability-without-overhead` | `org-scoped-api-keys` (docs) | "audit trail" cross-link | Medium |
| `govern-ai-fleet-system-prompt-level` | `ai-agent-observability-without-overhead` | "agent observability" | High |
| `govern-ai-fleet-system-prompt-level` | `partner-api-keys` | "workspace-level governance" or "platform instructions" | Medium |
| `partner-api-keys` | `org-scoped-api-keys` (docs) | "org-scoped API keys" | Medium |
| `partner-api-keys` | `govern-ai-fleet-system-prompt-level` | "workspace-scoped rules" | Medium |
| All 3 Phase 34 posts | `remote-workspaces` | "remote agent platform" | Medium |

**Phase 30 → Phase 34 backward links** (inject into existing Phase 30 posts):
| From | To | Anchor |
|------|----|--------|
| `a2a-v1-agent-platform` | `ai-agent-observability-without-overhead` | "tool_trace[] metadata" or "agent observability" |
| `org-scoped-api-keys` (docs) | `partner-api-keys` | "Partner API Keys" |
| `remote-workspaces` | `ai-agent-observability-without-overhead` | "agent observability tooling" |

---

## 5. Core Web Vitals Notes

| Post | LCP | INP | CLS | Notes |
|------|-----|-----|-----|-------|
| `ai-agent-observability-without-overhead` | TBD post-deploy | TBD post-deploy | TBD post-deploy | Code examples increase LCP risk if not lazy-loaded. |
| `govern-ai-fleet-system-prompt-level` | TBD post-deploy | TBD post-deploy | TBD post-deploy | Table of global vs. workspace-scoped rules — use semantic HTML tables. |
| `partner-api-keys` | TBD post-deploy | TBD post-deploy | TBD post-deploy | No heavy assets expected. Monitor if OG image causes CLS on load. |

**CWV actions before/after GA:**
1. Lighthouse audit all 3 posts within 48h of GA (2026-05-02)
2. Confirm `partner-api-keys` OG image dimensions (1200×630) to prevent CLS
3. Code blocks: ensure syntax highlighting is lazy-loaded, not render-blocking
4. Tables: wrap in semantic `<table>` with explicit `width` to prevent layout shift

---

## 6. April 30 GA Launch Checklist

| # | Action | Owner | Status |
|---|--------|-------|--------|
| 1 | Create OG image for `partner-api-keys` (1200×630) | Social Media Brand | 🚨 **Missing — block GA** |
| 2 | Trim title tag on `partner-api-keys` to ≤60 chars | Content Marketer | 🚨 Needs update |
| 3 | Trim meta description on `partner-api-keys` to ≤155 chars | Content Marketer | 🚨 Needs update |
| 4 | Trim og_description on `partner-api-keys` to ≤97 chars | Content Marketer | 🚨 Needs update |
| 5 | Add `partner-api-keys` og_image path to frontmatter | Content Marketer | 🚨 Blocked on #1 |
| 6 | Expand keywords frontmatter on all 3 posts (see Section 3) | Content Marketer | Pending |
| 7 | Inject cross-links between Phase 34 posts (Section 4) | Content Marketer | Pending |
| 8 | Inject backward links from Phase 30 posts to Phase 34 | Content Marketer | Pending |
| 9 | Trim meta description on `ai-agent-observability-without-overhead` | Content Marketer | Pending |
| 10 | Lighthouse audit all 3 posts — post-GA (2026-05-02) | SEO Analyst | Scheduled |
| 11 | Verify sitemap entries for all 3 posts | DevRel | Pending |
| 12 | Verify JSON-LD schema (Article or HowTo) on all 3 posts | SEO Analyst | To verify |

---

## 7. Priority Order

1. **🚨 og_image for `partner-api-keys`** — only hard blocker. Everything else can go out at GA and be fixed post-launch.
2. **`partner-api-keys` meta title + description trim** — low effort, high SERP impact.
3. **Cross-linking all 3 Phase 34 posts** — maximum SEO lift from existing content.
4. **Phase 30 → Phase 34 backward links** — amplifies Phase 34 authority.
5. **Lighthouse audits** — 48h post-GA.

---

*Brief maintained by SEO Analyst (5b277fc4). Web search unavailable at time of writing — keyword volume estimates are directional. Update with live SERP data when API is restored.*
