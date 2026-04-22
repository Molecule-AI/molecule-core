# SEO Brief: Phase 30 — Remote Workspaces / SaaS Federation
**Issue:** #1126
**Date:** 2026-04-20 (updated 2026-04-21)
**Author:** SEO Analyst
**Campaign:** Phase 30 Remote Workspaces
**Status:** BRIEF DRAFT — pending PMM positioning review

---

## 1. Context

Phase 30 ships per-workspace bearer tokens, unified fleet visibility, and remote agent registration for heterogeneous AI agent fleets spanning laptops, cloud VMs, CI/CD pipelines, on-premise servers, and SaaS integrations.

**Already published:**
- Blog post: `docs/blog/2026-04-20-remote-workspaces/index.md`
  - Title: "One Canvas, Every Agent: Remote AI Agents and Fleet Visibility on Molecule AI"
  - Covers: fleet visibility problem, bearer token security model, agent registration, heartbeat, org placement

**This brief:** Additional SEO content needed to support the launch and capture long-tail informational queries.

---

## 2. Target Keywords

| Keyword | Intent | Difficulty | Priority |
|---|---|---|---|
| `remote AI agent deployment` | Informational | Low | High |
| `self-hosted AI agents platform` | Informational / Commercial | Medium | High |
| `AI agent SaaS federation` | Informational | Low | Medium |
| `cross-network AI orchestration` | Informational | Low | Medium |
| `federated AI agents` | Informational | Low | Medium |
| `AI agent fleet management` | Informational / Transactional | Medium | High |
| `self-host Claude Code agents` | Informational | Low | High |
| `multi-cloud AI agent platform` | Commercial | Medium | Medium |
| `remote AI agent canvas` | Navigational | Low | Medium |

**Primary angle:** `remote AI agent deployment` + `self-hosted AI agents platform` — these capture the developer audience searching for how to deploy agents outside a single cloud/VPS.

---

## 3. Content Gap Analysis

### Already covered (blog post):
- Fleet visibility problem framing
- Bearer token security model
- Agent registration flow
- Heartbeat mechanism
- Org placement

### Missing for SEO:
| Gap | Content type | Priority | Rationale |
|---|---|---|---|
| Step-by-step: register a remote agent | Tutorial / How-to | High | High search intent, procedural |
| Self-hosted remote agents setup | Tutorial / How-to | High | Complements `self-hosted AI agents platform` kw |
| Remote agent vs Docker workspace | Comparison / FAQ | Medium | Common confusion point |
| Cross-network A2A walkthrough | Tutorial | Medium | Technical audience |
| Remote agent on fly machines | Tutorial | Medium | Specific infra angle |

---

## 4. Content Recommendation

**This is a docs play, not a landing page play.**

Search intent for `remote AI agent deployment` and `self-hosted AI agents platform` is overwhelmingly informational/how-to. Developers searching these terms want to understand the problem and evaluate solutions — they want setup guides, not marketing copy.

**Recommended content sequence:**

1. **Expand existing blog post** — add a "Step-by-Step: Register a Remote Agent" section with code/config examples to capture procedural search queries
2. **New tutorial: "Register a Remote Agent on Molecule AI"** — a focused how-to targeting `remote AI agent deployment` + `register AI agent with Molecule AI`
3. **New tutorial: "Self-Hosted AI Agents with Molecule AI"** — targeting `self-hosted AI agents platform`, covers Docker, Fly Machines, bare metal
4. **Update: `docs/agent-runtime/workspace-runtime.md`** — add remote agents section with bearer token setup
5. **Update: `docs/guides/external-agent-registration.md`** — if exists, audit for Phase 30 coverage; if not, create

---

## 5. Docs Pages to Update Post-Launch

| Page | Update needed |
|---|---|
| `docs/agent-runtime/workspace-runtime.md` | Add remote agent registration, bearer token setup, heartbeat config |
| `docs/agent-runtime/agent-card.md` | Confirm agent card covers external agent registration |
| `docs/api-protocol/registry-and-heartbeat.md` | Confirm heartbeat covers external agents (30s interval noted in blog) |
| `docs/guides/external-agent-registration.md` | Create if missing — step-by-step for registering CI/CD agents, laptop agents, cloud VMs |
| `docs/quickstart.md` | Add remote agent path alongside Docker/Fly Machines |
| `docs/index.md` | Add Remote Agents to product features list |

---

## 6. PMM Positioning Review Needed

The issue #1126 acceptance criteria specifies: "Coordinate with PMM (issue #1116) on positioning language."

**Questions for PMM:**
1. **Primary message:** "One canvas, every agent" (fleet visibility) or "Deploy agents anywhere, manage them from one place" (deployment flexibility)?
2. **Competitive framing:** How does Phase 30 compare to LangChain Agents + LangServe, CrewAI remote executors, or OpenAI's agent SDK? Any positioning lines to own?
3. **Audience priority:** Is the primary buyer/evaluator an infra lead, a developer, or a platform team? This affects keyword targeting and content tone.
4. **Pricing/availability:** Is Phase 30 live for all tiers or a specific plan? Affects CTA language.

---

## 7. Action Items

| # | Action | Owner | Status |
|---|---|---|---|
| 1 | Keyword research (this brief) | SEO Analyst | ✅ Draft done |
<<<<<<< HEAD
| 2 | PMM positioning review | PMM (issue #1116) | ⏸ Holding — PMM Slack: "Phase 30 position holding" |
| 3 | Expand blog post with step-by-step | Content Marketer | ⏸ Pending PMM |
| 4 | Draft tutorial: "Register a Remote Agent" | SEO Analyst | ✅ Done — `docs/tutorials/register-remote-agent.md`, pushed to molecule-core@main |
| 5 | Draft tutorial: "Self-Hosted AI Agents" | SEO Analyst | ✅ Done — `docs/tutorials/self-hosted-ai-agents.md`, pushed to molecule-core@main |
| 6 | Update workspace-runtime.md | DevRel | ✅ Done — remote agent registration section already on main |
| 7 | Audit/create external-agent-registration.md | DevRel | ✅ Done — already on main, full coverage |
| 8 | Update quickstart.md + docs/index.md | DevRel | ✅ Done — Remote Agent path in quickstart; docs/index.md updated with Remote Agents feature card + blog links |
=======
| 2 | PMM positioning review | PMM (issue #1116) | ⏸ Pending |
| 3 | Expand blog post with step-by-step | Content Marketer | ⏸ Pending PMM |
| 4 | Draft tutorial: "Register a Remote Agent" | Content Marketer | ⏸ Pending |
| 5 | Draft tutorial: "Self-Hosted AI Agents" | Content Marketer | ⏸ Pending |
| 6 | Update workspace-runtime.md | DevRel | ⏸ Flag to DevRel |
| 7 | Audit/create external-agent-registration.md | DevRel | ⏸ Flag to DevRel |
| 8 | Update quickstart.md | DevRel | ⏸ Flag to DevRel |
>>>>>>> origin/staging

---

## 8. Campaign Assets

**Blog post URL (live):** `https://github.com/Molecule-AI/molecule-core/blob/main/docs/blog/2026-04-20-remote-workspaces/index.md`

**Internal links to add once tutorials are published:**
- Blog post → Remote Agent tutorial
- Quickstart → Remote Agent section
- Agent Card docs → remote registration section
- External Agent tutorial → A2A cross-network walkthrough

---

*Draft by SEO Analyst 2026-04-21 — pending PMM positioning review*
