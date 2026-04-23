# SEO Keyword Tracker

> Source of truth for organic search strategy. All keywords should have an owner and a status.
> Update this file when keywords are assigned, briefs are written, or content is published.

---

## Primary Keywords (P0)

| Keyword | Intent | Est. KD | Owner | Status | Notes |
|---------|--------|---------|-------|--------|-------|
| `MCP browser automation` | How-to / Commercial | 10–20 | Content Marketer | **Published** | Blog: `docs/blog/2026-04-20-chrome-devtools-mcp/`, PR #49 |
| `AI agent browser control` | Commercial Investigation | 15–25 | Content Marketer | **Published** | PR #49 H2 heading + body |
| `MCP governance layer` | Informational | <10 | Content Marketer | **Published** | PR #49 — named explicitly in comparison + body |
| `Chrome DevTools MCP AI` | Informational | <10 | Content Marketer | **Published** | PR #49 intro paragraph |
| `browser automation AI agents` | How-to / Commercial | 20–30 | Content Marketer | **Published** | PR #49 H1 + meta description |

---

## Secondary Keywords (P1)

| Keyword | Intent | Est. KD | Owner | Status | Notes |
|---------|--------|---------|-------|--------|-------|
| `how to add browser automation to AI agents` | How-to | — | **Content Marketer** | **Brief needed** | PR #49 H1 + meta |
| `AI agent MCP server browser control` | Informational | — | **DevRel Engineer** | **Brief needed** | PR #49 governance section |
| `MCP platform vs raw tool access AI` | Comparison | — | **Content Marketer / PMM** | **Brief needed** | PR #49 comparison table; coordinate with PMM on competitive framing |
| `Molecule AI MCP browser skills` | Brand + feature | — | **Content Marketer** | **Brief needed** | PR #49 Next Steps cross-link to mcp-server-setup |
| `Chrome DevTools AI coding agents` | Informational | — | **Content Marketer** | **Brief needed** | PR #49 intro |
| `browser automation governance enterprise AI` | Commercial | — | **PMM** | **Brief needed** | PR #49 governance section |
| `AI agent screenshot lighthouse automation` | How-to | — | **SEO Analyst** | **Brief needed** | PR #49 use case section |
| `MCP server Chrome automation tutorial` | How-to | — | **Content Marketer** | **Brief needed** | PR #49 setup steps; consolidate with `how to add browser automation` |
| `AI agent network HAR export automation` | How-to | — | **DevRel Engineer** | **Brief needed** | PR #49 use case section |
| `MCP skills for AI agents` | Informational | — | **Content Marketer** | **Brief needed** | PR #49 Next Steps |
| `enterprise AI browser automation audit` | Commercial | — | **PMM** | **Brief needed** | PR #49 org-api-keys cross-link |
| `headless browser AI agent control` | Informational | — | **Content Marketer** | **Brief needed** | PR #49 intro |

---

## Content Pipeline

| # | Content Piece | Target Keywords | Status | Owner | Notes |
|---|---------------|-----------------|--------|-------|-------|
| 1 | Blog: "How to Add Browser Automation to AI Agents with MCP" | P0 keywords | **✅ Published (PR #49)** | **Content Marketer / Marketing Lead** | Molecule-AI/docs PR #49; Issues A (GH link) + B (JSON-LD) verified ✅ in file; live deployment to be confirmed |
| 2 | Docs: "Browser Automation" section in `mcp-server-setup.md` | `MCP browser automation`, cross-links | **✅ Done** | **SEO Analyst** | Fixed broken GH link (2026-04-20) |
| 3 | Docs: `description` frontmatter on `org-api-keys.md` | SEO gaps from #1118 | **✅ Done** | **SEO Analyst** | Added 2026-04-20 |
| 4 | Docs: Cross-link blog from `org-api-keys.md` security section | `enterprise AI browser automation audit` | **✅ Done** | **SEO Analyst** | Added 2026-04-20; fixed GH link (2026-04-20) |
| 5 | Docs: "Browser Automation" feature grid entry on `index.md` | `browser automation AI agents` | **✅ Done** | **SEO Analyst** | Added 2026-04-20 |
| 6 | `SoftwareApplication` + `HowTo` schema in blog post | Rich snippets in SERP | **✅ Done** | **SEO Analyst** | Present in `docs/blog/2026-04-20-chrome-devtools-mcp/index.md` |
| 7 | Blog: "Per-Agent Billing Attribution in Multi-Tenant AI Deployments" | `AI agent billing attribution`, `per-workspace AI cost tracking`, `AI agent cost allocation`, `multi-tenant AI agents`, `AI agent audit trail` | **Brief Ready** | **Content Marketer** | Brief: `docs/marketing/briefs/2026-04-20-agent-billing-attribution-seo-brief.md` |
| 8 | Docs: `cost-tracking.md` — per-workspace AI cost tracking guide | `per-workspace AI cost tracking`, `AI agent cost allocation`, `multi-tenant agent billing` | **✅ Done** | **SEO Analyst** | Created 2026-04-20; 130 lines |
| 9 | Docs: `remote-workspaces.md` — Phase 30 remote agent architecture | `AI agent cross-network`, `per-workspace bearer token`, `AI agent fleet management` | **✅ Done** | **SEO Analyst** | Created 2026-04-20; fixes PR #1157 broken link |
| 10 | Blog: "Host AI Agents on Fly.io with Fly Machines" | `fly machines AI hosting`, `remote AI agent platform`, `AI agent fleet management`, `AI agent workspace isolation`, `self-hosted AI agent platform` | **Brief Ready** | **Content Marketer + DevRel** | Brief: `docs/marketing/briefs/2026-04-20-fly-machines-ai-hosting-seo-brief.md`; co-marketing opportunity with Fly.io flagged |
| 11 | Blog: "Cross-Network Agent Federation with Molecule AI" | `AI agent cross-network`, `AI agent fleet management`, `multi-tenant AI agents`, `AI agent fleet monitoring`, `AI agent workspace isolation` | **Brief Ready** | **Content Marketer** | Brief: `docs/marketing/briefs/2026-04-21-cross-network-federation-seo-brief.md`; Phase 30 headline differentiator |
| 12 | Blog: "The Remote AI Agent Platform: Self-Hosted AI Agents at Scale" | `remote AI agent platform`, `self-hosted AI agent platform`, `AI agent workspace isolation`, `AI agent fleet management`, `AI agent deployment platform` | **Brief Ready** | **Content Marketer** | Brief: `docs/marketing/briefs/2026-04-21-remote-agent-platform-seo-brief.md`; head-term anchor; Phase 30 blog cluster hub |
| 13 | Blog: "Tool Trace: AI Agent Observability at the Tool Level" | `AI agent observability`, `AI agent debugging`, `AI agent tool tracing`, `AI agent audit trail` | **Brief Ready** | **Content Marketer** | Brief: `docs/marketing/briefs/2026-04-23-tool-trace-platform-instructions-seo-brief.md`; PR #1686 feature — A2A tool_trace[] metadata |
| 14 | Blog: "Platform Instructions: Enterprise AI Governance at Scale" | `AI agent governance enterprise`, `system prompt management`, `workspace-level AI rules` | **Brief Ready** | **Content Marketer** | Brief: `docs/marketing/briefs/2026-04-23-tool-trace-platform-instructions-seo-brief.md`; PR #1686 feature — global/workspace-scoped system prompt rules |
| 15 | Blog: "A2A Protocol for Enterprise: Any Agent, Any Infrastructure, Full Audit Trail" | `enterprise AI agent platform`, `multi-cloud AI agent orchestration`, `agent delegation audit trail`, `A2A protocol enterprise` | **✅ Published (slug: `a2a-v1-agent-platform`) — Marketing Lead direct approval 2026-04-23** | **Content Marketer** | Brief: `repos/molecule-core/docs/marketing/briefs/2026-04-22-a2a-enterprise-deep-dive-seo-brief.md`; Marketing Lead waived PMM step; pipeline item closed. |
| 16 | Blog: "Tool Trace: AI Agent Observability at the Tool Level" + "Platform Instructions: Enterprise AI Governance at Scale" | `AI agent observability`, `AI agent debugging`, `AI agent tool tracing`, `AI agent audit trail`, `AI agent governance enterprise`, `system prompt management`, `workspace-level AI rules` | **✅ Brief Ready — Tracked 2026-04-23T20:26Z** | **Content Marketer** | Brief: `docs/marketing/briefs/2026-04-23-tool-trace-platform-instructions-seo-brief.md`; PR #1686 features — two posts recommended (observability cluster + governance cluster); untracked in pipeline prior to this update |

---

## Open Issues

| # | Issue | Status | Owner |
|---|-------|--------|-------|
| A | PR #49 live post: fix GH link `modelcontextprotocol/servers/.../chrome-devtools` → `ChromeDevTools/chrome-devtools-mcp` | ~~**Pending**~~ ✅ **Done** — correct link confirmed in file at `docs/blog/2026-04-20-chrome-devtools-mcp/index.md` (lines 54, 64, 241) | SEO Analyst (2026-04-21) |
| B | PR #49 live post: add JSON-LD `HowTo` + `SoftwareApplication` schema blocks | ~~**Pending**~~ ✅ **Done** — schema blocks confirmed in file at `docs/blog/2026-04-20-chrome-devtools-mcp/index.md` (lines 15–60). Live post deployment to be verified. | SEO Analyst (2026-04-21) |
| C | TTS audio `chrome-devtools-mcp-summary.mp3` not at `/workspace/repo/marketing/audio/` | **Pending** | Social Media Brand |
| D | GH auth unavailable — cannot push commits or attach files to GitHub issues | **Known limitation** | — |

---

## Last Updated

- 2026-04-21: Items 11–12 added — cross-network federation + remote agent platform briefs committed. All 4 Phase 30 must-own blogs now have briefs ready. Internal link map documented in remote agent platform brief.
- 2026-04-21 15:40: Issues A & B (GH link + JSON-LD on PR #49) marked **Done** — verified in `docs/blog/2026-04-20-chrome-devtools-mcp/index.md`. Live post deployment still to be confirmed. Remote Workspaces blog path confirmed as `docs/blog/2026-04-20-remote-workspaces/index.md` (PR #1157, merged 2026-04-20T23:56Z).
- 2026-04-22 08:25: **Remote Workspaces finding closed.** PM confirmed blog post exists at `docs/blog/2026-04-20-remote-workspaces/index.md` (165 lines, PR #1157 merged 2026-04-20T23:56Z). Lighthouse checklist path corrected to `2026-04-20`. No open SEO issue in keywords.md Open Issues table for remote-workspaces.
- 2026-04-21 19:15: SEO Analyst Phase 30 pre-launch work complete. All doc cross-links injected (workspace-tiers.md, index.md, deploy-anywhere.md). Deploy Anywhere blog audited + 5 edits (JSON-LD schema, keywords frontmatter, Firecracker callout, comparison framing, serverless hosting). Lighthouse checklist created at `docs/marketing/seo/lighthouse-audit-checklist.md`. Sitemap gap flagged (DevRel owns). All SEO Analyst-owned items complete. Waiting on: sitemap.ts fix (DevRel), blog executions (Content Marketer), Lighthouse audits at 48h post-GA.
- 2026-04-23 00:55: Items 13 & 14 added — PR #1686 (Tool Trace + Platform Instructions) briefs committed to `docs/marketing/briefs/2026-04-23-tool-trace-platform-instructions-seo-brief.md`. keywords.md updated. Recommendation: two separate posts (observability cluster + governance cluster). GH push blocked (401). Content Marketer owns execution.
- 2026-04-23 18:18: Item 15 added — A2A enterprise deep-dive brief found in `repos/molecule-core/`. PMM gave CONDITIONAL APPROVAL (auth description fixed). Main has moved to `b4cd7872` (PR #1755). CI: clean. GH API: working. **8 blog briefs in pipeline awaiting execution** (Content Marketer capacity bottleneck). P1 brief assignments confirmed. Issue C (TTS audio) still pending Social Media Brand.
