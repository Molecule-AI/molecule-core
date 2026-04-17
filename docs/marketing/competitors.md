# Competitor Tracker

> **Auto-maintained by PMM cron** — diffs `docs/ecosystem-watch.md` on schedule
> to detect version bumps, threat escalations, and notable changes.
>
> Source of truth for competitor state: `docs/ecosystem-watch.md#competitor-snapshot`
> Full narrative analysis: `docs/ecosystem-watch.md#entries`
>
> **Last updated:** 2026-04-17 (bootstrap — subsequent updates by PMM cron)

---

## High-Threat Competitors

Platforms that directly substitute for or significantly erode Molecule AI's market position.

| Competitor | Version | Stars | Threat Signal | Updated |
|---|---|---|---|---|
| [OpenAI Agents SDK](https://github.com/openai/openai-agents-python) | v0.14.1 | 14k | v0.14.1 SandboxAgent beta — persistent isolated workspaces, snapshot/resume, sandbox memory; directly competes with our workspace lifecycle | 2026-04-17 |
| [CrewAI](https://github.com/crewAIInc/crewAI) | v1.14.1 | 48k | 1.4B agentic automations, 60% Fortune 500 adoption, $18M Insight-led round; CrewAI Enterprise SaaS targeting our enterprise segment | 2026-04-17 |
| [Google ADK](https://github.com/google/adk-python) | v1.30.0 | 19k | v1.30.0 adds Auth Provider registry; full Google agent stack (ADK + Gemini CLI + adk-web DevUI + Scion harness) = largest platform risk | 2026-04-17 |
| [Microsoft Agent Framework](https://github.com/microsoft/agent-framework) | python-1.0.1 | 9.5k | v1.0 GA (official AutoGen successor); SOC 2/HIPAA compliance; .NET + Python; Process Framework GA in Q2 2026 | 2026-04-17 |

---

## Medium-Threat Competitors

Significant overlap in adjacent space; active watch required.

| Competitor | Version | Stars | Notes | Updated |
|---|---|---|---|---|
| [Paperclip](https://github.com/paperclipai/paperclip) | v2026.416.0 | 54.8k | Downgraded HIGH→MEDIUM (deep-dive #571): no A2A, no visual canvas on roadmap; single-process task DAG only; brand/framing threat ("zero-human companies"), not a technical substitute. Only gap vs Molecule AI: per-workspace budget limits (#541). | 2026-04-17 |
| [Dify](https://github.com/langgenius/dify) | v1.13.3 | 60k | v1.14.0 RC adds Human Input node; $30M Pre-A ($180M val); no-code positioning targets business users, not our developer audience | 2026-04-17 |
| [LangGraph](https://github.com/langchain-ai/langgraph) | v1.1.6 | 29k | CLI v0.4.22 Apr 16; LangGraph Cloud hosted execution competes with our scheduler | 2026-04-17 |
| [VoltAgent](https://github.com/VoltAgent/voltagent) | server-elysia@2.0.7 | 8.2k | VoltOps Console = closest Canvas analogue in TypeScript ecosystem | 2026-04-17 |
| [n8n](https://github.com/n8n-io/n8n) | v2.17.2 | 50k | n8n 2.0 enterprise AI Agent nodes + RBAC + 400+ channel integrations | 2026-04-17 |
| [Claude Code Routines](https://code.claude.com/docs/en/routines) | cloud-feature | — | Apr 14 2026 launch: Anthropic-hosted cron + GitHub-event-triggered Claude Code sessions | 2026-04-17 |
| [Scion](https://github.com/GoogleCloudPlatform/scion) | active | early | GCP experimental container-per-agent harness (Apr 8 2026); escalation risk to HIGH if productized | 2026-04-17 |
| [Multica](https://github.com/multica-ai/multica) | active | 12.8k | Positioned as Claude Managed Agents alternative; local daemon + central backend with skill compounding | 2026-04-17 |
| [Cline](https://github.com/cline/cline) | active | 44k | Primary user-overlap with our Claude Code workspace; developers who outgrow Cline convert to Molecule AI | 2026-04-17 |
| [ClawRun](https://github.com/clawrun-sh/clawrun) | active | 84 | Closest architectural match tracked (sandbox/heartbeat/snapshot-resume/channels/cost-tracking); early stage but actively shipped | 2026-04-17 |
| [Gemini CLI](https://github.com/google-gemini/gemini-cli) | v0.38.1 | 101k | Runtime candidate for our workspace adapter; elevated to MEDIUM as part of Google's full agent stack | 2026-04-17 |

---

## Low-Threat Competitors

Tools, infra layers, single-agent products, or projects we use — not direct substitutes.

| Competitor | Version | Stars | Role | Updated |
|---|---|---|---|---|
| [Hermes Agent](https://github.com/NousResearch/hermes-agent) | v0.10.0 | 61k | v0.10.0 (Apr 16) Tool Gateway launch; personal AI single-user shape | 2026-04-17 |
| [gstack](https://github.com/garrytan/gstack) | active | 70k | Sequential single-session Claude Code persona-switching; no multi-agent infra | 2026-04-17 |
| [claude-mem](https://github.com/thedotmack/claude-mem) | active | 56k | Memory addon; 56k ⭐ signals demand gap we need to close in agent_memories | 2026-04-17 |
| [Flowise](https://github.com/FlowiseAI/Flowise) | flowise@3.1.2 | 30k | Acquired by Workday (Aug 2025); v3.1.2 security hardening; narrowed to HR/finance enterprise | 2026-04-17 |
| [OpenHands](https://github.com/All-Hands-AI/OpenHands) | v1.6.0 | 47k | SWE-Bench top; v1.6.0 (Mar 30); single-agent software engineer only | 2026-04-17 |
| [Temporal](https://github.com/temporalio/temporal) | v1.30.4 | 13k | Durable execution infra we integrate; $5B valuation, not a competitor | 2026-04-17 |
| [Chrome DevTools MCP](https://github.com/ChromeDevTools/chrome-devtools-mcp) | active | 35.5k | Browser MCP we adopt (issue #540); 23-tool surface | 2026-04-17 |
| [AgentScope](https://github.com/modelscope/agentscope) | v1.0.18 | 23.8k | Alibaba/ModelScope framework; MCP integration; no deployment layer | 2026-04-17 |
| [Composio](https://github.com/composio-dev/composio) | active | 18k | Tool integration library; potential skill-pack dependency | 2026-04-17 |
| [Archon](https://github.com/coleam00/Archon) | v0.3.6 | 18.1k | YAML-DAG coding workflow; reference design for workspace delivery pipelines | 2026-04-17 |
| [Skills CLI](https://github.com/vercel-labs/skills) | active | 14.2k | Vercel agentskills.io CLI; aligning plugins/ = free distribution channel | 2026-04-17 |
| [Holaboss](https://github.com/holaboss-ai/holaboss-ai) | active | 1.7k | Desktop AI employee; terminology collisions (workspace/SKILL.md) | 2026-04-17 |
| [Tencent AI-Infra-Guard](https://github.com/Tencent/AI-Infra-Guard) | v4.1.3 | 3.5k | Security scanner; use as MCP + plugin registry compliance checklist | 2026-04-17 |
| [Plannotator](https://github.com/backnotprop/plannotator) | v0.17.10 | 4.3k | HITL plan annotation UX; reference for improving approvals API schema | 2026-04-17 |
| [open-multi-agent](https://github.com/JackChen-me/open-multi-agent) | v1.1.0 | 5.7k | TypeScript goal-to-DAG library; ephemeral, no identity | 2026-04-17 |
| [Open Agents (Vercel)](https://github.com/vercel-labs/open-agents) | active | 2.2k | Reference app; snapshot-based VM resumption pattern worth borrowing | 2026-04-17 |
| [GenericAgent](https://github.com/lsdefine/GenericAgent) | v1.0 | 2.1k | Self-evolving skill tree; four-tier memory taxonomy worth borrowing | 2026-04-17 |
| [OpenSRE](https://github.com/Tracer-Cloud/opensre) | active | 900 | AI SRE toolkit; potential DevOps workspace skill-pack source | 2026-04-17 |
| [AMD GAIA](https://github.com/amd/gaia) | v0.17.2 | 1.2k | Hardware-locked (AMD Ryzen AI 300+); not general-purpose | 2026-04-17 |

---

## Watchlist — Escalation Signals

The following events would require immediate threat-level re-assessment:

| Competitor | Watch Signal | Current Level | Escalates To |
|---|---|---|---|
| Paperclip | Ships persistent agent memory | MEDIUM | HIGH — 54.8k ⭐ head-start |
| Paperclip | Ships visual org-chart canvas | MEDIUM | HIGH — direct Canvas competitor |
| Scion | Google productizes as managed GCP service | MEDIUM | HIGH |
| VoltAgent | VoltOps Console adds visual org-chart topology | MEDIUM | HIGH |
| Google ADK | ADK + Vertex AI becomes hosted managed platform | HIGH | CRITICAL |
| OpenAI Agents SDK | Inter-sandbox A2A across process boundaries | HIGH | CRITICAL |
| ClawRun | Adds A2A or multi-agent coordination | MEDIUM | HIGH |
| gstack | Adds multi-session/parallel execution | LOW | HIGH — 70k ⭐ head-start |
| Claude Code Routines | Adds A2A between routine sessions | MEDIUM | HIGH — Anthropic distribution |

---

## Recently Changed (last 30 days)

> PMM cron updates this section automatically when `notable_changes` or `version` fields change.

| Date | Competitor | Change |
|---|---|---|
| 2026-04-17 | **Paperclip** | Threat downgraded HIGH→MEDIUM (deep-dive #571): no A2A, no canvas, brand threat only |
| 2026-04-17 | **Paperclip** | v2026.416.0 — execution policies + chat threads for agent transcripts |
| 2026-04-17 | **Hermes Agent** | v0.10.0 — Tool Gateway (web search, image gen, TTS, browser automation) |
| 2026-04-16 | **LangGraph CLI** | v0.4.22 — deploy source tracking |
| 2026-04-15 | **OpenAI Agents SDK** | v0.14.1 — tracing patch on top of Sandbox Agents beta |
| 2026-04-15 | **Gemini CLI** | v0.38.1 — stability patch |
| 2026-04-14 | **Flowise** | v3.1.2 — security hardening (CORS, credential leaks) |
| 2026-04-14 | **Claude Code Routines** | Launched — Anthropic-hosted cron-triggered Claude Code sessions |
| 2026-04-13 | **Google ADK** | v1.30.0 — Auth Provider + Parameter Manager + Gemma 4 support |
| 2026-04-11 | **VoltAgent** | server-elysia@2.0.7 — A2A agent card URL fix |
| 2026-04-10 | **LangGraph** | v1.1.6 — declarative guardrail nodes (LangGraph 2.0 GA) |
| 2026-04-10 | **Temporal** | v1.30.4 — CVE-2026-5724 security patch |
| 2026-04-10 | **Microsoft Agent Framework** | python-1.0.1 — FileCheckpointStorage security hardening |
| 2026-04-08 | **Scion** | Launched — GCP container-per-agent experimental harness |
| 2026-04-08 | **CrewAI** | v1.14.1 — async checkpoint TUI browser |
