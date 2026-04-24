# Competitive Intelligence — Molecule AI
**Last generated:** 2026-04-23 (from ecosystem-watch.md snapshot, updated 2026-04-22)
**Maintenance:** PMM cron diffs `ecosystem-watch.md` competitor-snapshot block and updates this file when `notable_changes` change.
**Source:** Molecule-AI/internal `ecosystem-watch.md`

---

## Snapshot Overview

| Competitor | Threat | Stars | Version | Last Shipped |
|---|---|---|---|---|
| Paperclip | Medium | 54.8k | v2026.416.0 | Apr 16 |
| OpenAI Agents SDK | High | 14k | v0.14.1 | Apr 15 |
| OpenAI Codex Agent | High | N/A | Apr 17 launch | Apr 17 |
| CrewAI | High | 48k | 1.14.3a2 | Apr 21 |
| Google ADK | High | 19k | v2.0.0b1 | Apr 22 |
| Google Vertex AI Agent Builder | High | N/A | GA (built-in) | Apr 22 |
| Microsoft Agent Framework | High | 9.5k | python-1.1.0 | Apr 21 |
| LangGraph | Medium | 29k | v1.1.9 | Apr 21 |
| Dify | Medium | 60k | v1.13.3 | Apr 22 |
| VoltAgent | Medium | 8.2k | @voltagent/server-hono@2.0.11 | Apr 22 |

---

## HIGH THREAT

### OpenAI Agents SDK — `openai/openai-agents-python`
**Threat:** High | **Stars:** 14k | **Version:** v0.14.1

**Notable changes (2026-04-22):**
v0.14.1 (Apr 15) patches tracing export on top of v0.14.0's SandboxAgent beta — persistent isolated workspaces, snapshot/resume, sandbox memory directly competing with workspace lifecycle model. No new release since Apr 15 (7 days).

**Molecule AI gap:** SandboxAgent workspaces directly overlap our `workspace-template/` lifecycle. If OpenAI Codex agents become the default "agent build path," Molecule AI becomes a deployment option, not the platform.

**Recommended action:** Monitor for v0.14.x stable release. Assess whether our Docker/ Fly Machine workspace backends compete on cold-start speed with SandboxAgent.

---

### OpenAI Codex Agent — `openai/codex`
**Threat:** High | **Stars:** N/A | **Version:** Launched Apr 17 2026

**Notable changes (2026-04-22):**
Relaunched Apr 17 2026 (HN #2, 769 pts): full autonomous agent product — parallel subagent orchestration, cross-session project memory, autonomous self-wake scheduling, macOS computer control, inline image generation. Distinct threat surface from openai-agents-sdk; directly overlaps workspace lifecycle, agent_memories, workspace_schedules.

**Molecule AI gap:** Cross-session memory + autonomous scheduling are our Phase 22 (cron) and Phase 9 (hierarchical memory). Codex ships them as a product. Molecule AI needs to ship them as features and own the developer-accessible version.

**Recommended action:** DevRel should benchmark Codex memory vs. our `commit_memory` + `workspace_schedules` stack. Document the Molecule AI alternative as a developer-accessible, self-hosted path.

---

### CrewAI — `crewAIInc/crewAI`
**Threat:** High | **Stars:** 48k | **Version:** 1.14.3a2

**Notable changes (2026-04-22):**
v1.14.2 (Apr 17) confirmed Crew Studio is real — node-and-edge drag-and-drop canvas. AMP Factory self-hosted: on-prem/VPC, K8s, FedRAMP High. A2A spec v0.3.0 first-class — zero-shim interop with Molecule AI confirmed. **LangGraph, AutoGen, CrewAI, Claude, OpenAI Agents as tool integrations via Composio** (MIT-adjacent, ~18k ⭐).

**CrewAI observability:** Third-party integrations only — LangSmith, Weights & Biases, custom callbacks. Manual instrumentation required. No structured tool-call-level trace inside A2A response.

**Molecule AI gap:** No governance-layer canvas. Team-role primitives are internal to a Crew, not org-scoped. Tool Trace + Platform Instructions fills the governance gap CrewAI doesn't have.

**Recommended action:** Own "A2A-native governance" as the differentiator. CrewAI competes on canvas; we compete on platform.

---

### Google ADK — `google/adk-python`
**Threat:** High | **Stars:** 19k | **Version:** v2.0.0b1

**Notable changes (2026-04-22):**
v2.0.0b1 (Apr 22 2026): FULL Workflow graph orchestration core — NodeRunner per-node execution isolation, DefaultNodeScheduler, graph-based execution engine GA. HITL resume via event reconstruction. Security fix: RCE vulnerability in nested YAML configs patched. Threat escalated: graph workflow + node isolation is direct overlap with our workspace delivery pipeline.

**Molecule AI gap:** Phase 12 (DAG workflow orchestration) is the direct counter. Google ADK ships it; we don't have it yet.

**Recommended action:** Prioritize Phase 12 DAG/workflow builder to directly counter Vertex AI Agent Builder's built-in flow builder. EC2 Instance Connect terminal is the differentiator against ADK — no ADK equivalent.

---

### Google Vertex AI Agent Builder — `google-cloud-aiplatform/vertex-ai-samples`
**Threat:** High | **Stars:** N/A | **Version:** GA (built-in with Vertex AI enterprise seats)

**Notable changes (2026-04-22):**
Built-in flow builder shipped with Vertex AI enterprise seats — zero incremental cost for GCP shops. Procurement objection, not a feature comparison. ADK v2.0.0b1 released same day — workflow graph + NodeRunner isolation.

**Molecule AI gap:** Vertex AI Agent Builder is included with enterprise seats — no separate purchase required for GCP customers. The objection is "we already have this." Phase 30 remote workspaces + EC2 Instance Connect terminal are the structural differentiators: direct EC2 access vs. managed Vertex service. Phase 12 DAG/workflow builder is the product counter.

**Critical note (per issue #1862):** Vertex AI Agent Builder is a **procurement objection, not a feature comparison**. Don't engage on features — reframe to: "Vertex AI Agent Builder is a managed service. Molecule AI is the agent runtime that runs on your infrastructure." The self-hosted + multi-backend story (Phase 30) is the answer.

**Recommended action:** Add Vertex AI Agent Builder to competitive monitoring at critical tier. Ensure Phase 30 remote workspaces messaging explicitly addresses the GCP managed service vs. self-hosted cost difference. Prioritize Phase 12 DAG/workflow builder to close the product gap.

---

### Microsoft Agent Framework — `microsoft/agent-framework`
**Threat:** High | **Stars:** 9.5k | **Version:** python-1.1.0

**Notable changes (2026-04-22):**
python-1.1.0 (Apr 21 2026): A2A metadata propagation across Message/Artifact/Task/event types. Foundry V2 hosted agents. AG-UI forwardedProps exposed to agents/tools via session metadata. GeminiChatClient added. FileCheckpointStorage BREAKING change (restricted pickle). AG-UI SSE endpoint gap remains. Process Framework GA still Q2 2026.

**Molecule AI gap:** AG-UI is a real spec for agent-UI communication. If it gains adoption, our Canvas competes with it. Document our Canvas feature set vs. AG-UI as a differentiator.

---

## MEDIUM THREAT

### LangGraph — `langchain-ai/langgraph`
**Threat:** Medium | **Stars:** 29k | **Version:** v1.1.9

**Notable changes (2026-04-22):**
v1.1.9 (Apr 21 2026) patches core. v1.1.6 (Apr 10) ships LangGraph 2.0 declarative guardrail nodes. langgraph-cli v0.4.22 (Apr 16) adds deploy source tracking. LangGraph Cloud hosted execution competes with our scheduler.

**LangGraph observability:** LangSmith integration — SDK-level instrumentation (`from langsmith import trace`), cross-platform multi-model traces. Requires active LangSmith account and separate vendor relationship.

**Molecule AI differentiator:** Tool Trace is A2A-level agent behavior (tool call sequences, run_id pairing) — LangSmith tracks model-level tokens; Tool Trace tracks agent behavior. LangGraph Cloud competes with our scheduler; our governance layer (Platform Instructions + Tool Trace) is what they don't have.

**LangGraph A2A status:** PRs #6645, #7113, #7205 (still in review as of Apr 22) — protocol layer only, no governance. PR #7205 adds DNS-AID agent discovery utilities. ⚠️ VERIFY: PRs #7113 and #7205 not independently confirmed OPEN this cycle — blog QA flagged same.

---

### Paperclip — `paperclipai/paperclip`
**Threat:** Medium | **Stars:** 54.8k | **Version:** v2026.416.0

**Notable changes (2026-04-22):**
v2026.416.0 (Apr 16) ships execution policies (HiTL multi-stage reviewer/approver routing) + chat threads (assistant-ui per-issue inline). Threat level MEDIUM confirmed per 2026-04-20 deep-dive — no architectural change to concurrency model.

**Molecule AI gap:** HITL routing is our Phase 8 (Human-in-the-Loop Approvals) and Paperclip HiTL (April 2026). Our differentiation: org-chart-based approval routing vs. per-task reviewer assignment.

---

### Dify — `difyai/dify`
**Threat:** Medium | **Stars:** 60k | **Version:** v1.13.3

**Notable changes (2026-04-22):**
v1.13.3 (Apr 22 2026) — patch. A2A spec compliance, multi-tenant managed service.

**Molecule AI gap:** Dify is self-hostable and has strong self-hosted adoption. Our self-hosted story needs to be clearer than "runs anywhere Docker runs." Phase 30 remote workspaces is the answer.

---

### VoltAgent — `voltagent/voltagent`
**Threat:** Medium | **Stars:** 8.2k | **Version:** @voltagent/server-hono@2.0.11

**Notable changes (2026-04-22):**
@voltagent/server-hono@2.0.11 (Apr 22 2026) — hotfix. A2A agent card endpoints.

**Molecule AI gap:** VoltAgent is an emerging player. Monitor for enterprise adoption.

---

## Update Log

| Date | Action | Trigger |
|---|---|---|
| 2026-04-23 | Created from ecosystem-watch.md snapshot | PMM cycle |
| 2026-04-23 | Added Google Vertex AI Agent Builder (critical tier) | Issue #1862 competitive brief |

---

*Source: `ecosystem-watch.md` competitor-snapshot block (Molecule-AI/internal), updated 2026-04-22*
*Generated by: PMM cron*
*Next update: when ecosystem-watch.md `date` + `notable_changes` fields change, or Google Vertex AI Agent Builder pricing/feature update*