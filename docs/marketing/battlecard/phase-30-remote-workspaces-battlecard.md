# Phase 30 — Remote Workspaces Competitive Battlecard
**Feature:** Remote workspaces (Fly Machines, EC2 Instance Connect SSH, any SSH target) + fleet visibility canvas
**Status:** PMM DRAFT | **Date:** 2026-04-23
**Phase:** 30 | **Owner:** PMM
**Campaign:** Phase 30 Remote Workspaces | **GA date:** 2026-04-20

---
## Competitive Context

Phase 30 shipped remote agent execution across heterogeneous backends — Fly Machines, EC2 Instance Connect SSH, and any SSH-accessible target — under a single canvas with full fleet visibility. No competitor matches this combination of deployment flexibility, governance layer, and unified observability.

**Competitor landscape:**

| Competitor | Remote Agent Execution | Fleet Visibility Canvas | A2A-Native Cross-Backend | Org API Keys + Audit |
|---|---|---|---|---|
| LangGraph Cloud | ❌ SaaS-only | ❌ | ❌ (PRs in review) | ❌ |
| CrewAI | ❌ (Docker exec, local only) | ❌ (Crew Studio canvas, per-crew) | ✅ (v0.3.0, no org scope) | ❌ |
| Dify | ✅ (self-hosted, any VM) | ❌ (no fleet view) | ✅ (A2A spec) | ❌ |
| Google ADK | ❌ (local/graph only) | ❌ | ❌ | ❌ |
| Microsoft AutoGen | ✅ (open source, any host) | ❌ | ❌ | ❌ |
| **Molecule AI Phase 30** | **✅ 3 backends, 1 canvas** | **✅ Fleet canvas** | **✅ A2A-native** | **✅ Org keys + audit** |

---

## Feature-by-Feature Battlecard

### 1. Multi-Backend Deployment

**Buyer question:** "Can I run agents on Fly Machines, EC2, and my own servers — without separate tooling?"

| | Molecule AI Phase 30 | LangGraph Cloud | CrewAI | Dify |
|---|---|---|---|---|
| Fly Machines | ✅ | ❌ | ❌ | ❌ |
| EC2 Instance Connect SSH | ✅ (no SSH keys to manage) | ❌ | ❌ | ❌ |
| Any SSH target | ✅ | ❌ | ❌ | ✅ |
| Single canvas — all backends | ✅ | ❌ | ❌ | ❌ |
| A2A routing across backends | ✅ | ❌ | ❌ | ❌ |
| Unified fleet dashboard | ✅ | ❌ | ❌ | ❌ |

**Molecule AI counter:** "LangGraph Cloud is a SaaS platform. CrewAI and Dify are single-backend. Molecule AI is the only agent platform where Fly Machines, EC2, and bare-metal run under the same org hierarchy — same auth, same A2A, same canvas."

**EC2 Instance Connect SSH differentiator:** Agents on EC2 without SSH key management. Browser-based IAM authentication via EC2 Instance Connect. No SSH keys to rotate, no bastion hosts to maintain. No competitor has this.

---

### 2. Fleet Visibility — One Canvas, Every Agent

**Buyer question:** "Can I see my whole agent fleet — across all environments — in one view?"

| | Molecule AI Phase 30 | LangGraph Cloud | CrewAI | Dify |
|---|---|---|---|---|
| Org-wide fleet view | ✅ — Canvas shows full org hierarchy | ❌ | ❌ | ❌ |
| Per-workspace role assignment | ✅ — admin / editor / viewer | ❌ | ❌ | ❌ |
| Live agent status across backends | ✅ | ❌ | ❌ | ❌ |
| A2A peer graph visible | ✅ | ❌ | ❌ | ❌ |
| Fleet-wide audit log | ✅ | ❌ | ❌ | ❌ |

**Molecule AI counter:** "CrewAI has Crew Studio — a canvas for one crew. Molecule AI's Canvas shows your whole org. That's the difference between managing a team and managing a platform."

**From Phase 30 positioning brief (Content Marketer, 2026-04-22):** "Fleet visibility by default" is the approved differentiator. "One canvas, every agent" is the approved social headline.

---

### 3. A2A-Native Cross-Backend Communication

**Buyer question:** "Can agents on different backends communicate with each other using standard protocol?"

| | Molecule AI Phase 30 | LangGraph Cloud | CrewAI | Dify |
|---|---|---|---|---|
| A2A v1.0 native | ✅ (since Phase 1) | ❌ (PRs in review) | ✅ (v0.3.0) | ✅ (spec compliance) |
| Org hierarchy = routing model | ✅ | ❌ | ❌ | ❌ |
| Platform never in message path | ✅ (peer-to-peer) | ❌ | ❌ | ❌ |
| Per-workspace tokens at every route | ✅ | ❌ | ❌ | ❌ |
| Cross-backend A2A delegation | ✅ | ❌ | ❌ | ❌ |

**Molecule AI counter:** "A2A is becoming table stakes. Molecule AI shipped A2A-native before the Linux Foundation ratified the standard. LangGraph's A2A implementation is still in review. CrewAI has A2A v0.3.0 — but without org-level governance."

**A2A governance differentiator:** CrewAI A2A is crew-scoped. Molecule AI A2A is org-scoped. "A2A is solved. A2A governance is not." — approved copy from Phase 30 positioning brief.

---

### 4. Org API Keys + Audit Trail

**Buyer question:** "Can I attribute every agent action to an org, workspace, and API key?"

| | Molecule AI Phase 30 | LangGraph Cloud | CrewAI | Dify |
|---|---|---|---|---|
| Org-level API keys | ✅ | ❌ (per-seat SaaS only) | ❌ | ❌ |
| Per-workspace tokens | ✅ (`mol_ws_*`) | ❌ | ❌ | ❌ |
| Audit log (agent action attribution) | ✅ | ❌ | ❌ | ❌ |
| Instant key revocation | ✅ | ❌ | ❌ | ❌ |
| Workspace-level isolation | ✅ | ❌ (per-seat) | ❌ (per-crew) | ❌ |

**Molecule AI counter:** "LangGraph Cloud bills per seat. CrewAI charges per crew. Molecule AI charges per org — and gives you the API keys to run your platform."

**From approved Phase 30 copy:** "Org API keys. Audit trail. Instant revocation." — confirmed as safe CTA language per Content Marketer.

---

### 5. Self-Hosted / Remote Execution Flexibility

**Buyer question:** "Can I run Molecule AI on my own infrastructure, behind my own firewall?"

| | Molecule AI Phase 30 | LangGraph Cloud | CrewAI | Dify |
|---|---|---|---|---|
| Self-hosted | ✅ (Docker, any SSH target) | ❌ | ✅ (open source) | ✅ |
| Remote agent registration | ✅ (workspace registration) | ❌ | ❌ | ❌ |
| Fly Machines backend | ✅ | ❌ | ❌ | ❌ |
| EC2 Instance Connect SSH | ✅ | ❌ | ❌ | ❌ |
| Remote agents under org governance | ✅ | ❌ | ❌ | ❌ |

**Dify comparison:** Dify is self-hostable with Docker compose and supports remote execution. But Dify has no fleet canvas, no A2A-native cross-backend routing, and no org API key governance. Running Dify across EC2 and Fly would require separate deployments.

**Molecule AI counter:** "Dify runs anywhere Docker runs. Molecule AI runs anywhere Docker runs — then connects Fly Machines, EC2, and bare-metal into one fleet canvas with one audit log."

---

## Positioning Claims

**Lead claim:** "Molecule AI is the **first** agent platform with fleet-wide visibility across heterogeneous backends — Fly Machines, EC2 Instance Connect SSH, and any SSH target — under a single canvas with org-level governance."

**Supporting claims:**
1. **"One canvas, every agent"** — fleet visibility is the approved headline per Content Marketer positioning brief (2026-04-22)
2. **"Deploy agents anywhere, manage them from one place"** — deployment flexibility + fleet control is the SEO-approved sub-message
3. **A2A-native since Phase 1 (2025)** — two years before Linux Foundation ratified A2A v1.0
4. **"EC2 without SSH keys"** — EC2 Instance Connect uses browser-based IAM; no key rotation, no bastion hosts
5. **"A2A is solved. A2A governance is not."** — approved competitive framing from Phase 30 positioning brief

**Risks to monitor:**
- Google ADK v2.0 ships graph workflow GA → Phase 12 DAG builder becomes priority
- LangGraph Cloud adds org hierarchy → update "only" framing
- Dify ships fleet canvas → update "only" framing

---

## Language to Avoid

- ~~"Only platform with fleet visibility"~~ — Dify and others could ship
- ~~Benchmark numbers (cold-start latency, etc.)~~ — unconfirmed
- ~~"Available on all plans"~~ — pricing tier not confirmed by PM
- ~~"Better than [competitor]"~~ — use specific feature comparisons only

---

## Update Triggers

| Event | Action |
|---|---|
| Google ADK v2.0 ships graph workflow GA | Update Google ADK row; flag Phase 12 priority |
| LangGraph Cloud adds org hierarchy | Update LangGraph row; remove "org hierarchy" claim |
| Dify ships fleet canvas | Update Dify row; update lead claim |
| Phase 12 DAG builder ships | Link Phase 12 battlecard |
| Phase 34 GA (Apr 30) | Add Partner API Keys cross-sell to Phase 30 copy |

---

## Cross-Campaign Linkage

**Phase 34 GA (April 30, 2026):**
Phase 30 workspace isolation (`mol_ws_*`) + Phase 34 partner scoping (`mol_pk_*`) = **first agent platform with layered token scoping and a first-class partner provisioning API.**

**Phase 30 campaigns with approved copy:**
- Chrome DevTools MCP → Phase 30 Day 1, fleet canvas visual
- Cloudflare Artifacts → Phase 30 catch-up, git-native remote storage
- Fly Deploy Anywhere → Phase 30 catch-up, 3 backends
- EC2 Instance Connect SSH → Phase 30 Day 4, EC2 without SSH keys
- Org-Scoped API Keys → Phase 30 Day 5, audit trail + revocation
- MCP Server List → Phase 30 Day 1, MCP + fleet governance

**A2A Enterprise Deep-Dive:**
Phase 30 A2A + Org hierarchy = routing model. Phase 34 Partner API Keys = platform builder story. Together: "Molecule AI is infrastructure your platform builds on."

---

## Proof Points (for sales + content)

| Proof point | Source | Where to use |
|---|---|---|
| Fly Machines + EC2 + SSH under one canvas | Phase 30 GA docs | All Phase 30 copy |
| A2A-native since Phase 1 (2025) | PLAN.md Phase 1 | A2A copy, competitive claims |
| 23,300 GitHub stars on A2A v1.0 ratification | Linux Foundation, March 12 2026 | All A2A copy |
| EC2 Instance Connect — no SSH keys | PR #1637 | EC2 Console Output + SSH copy |
| Org API key audit trail | Phase 30 approved copy | Org API Keys copy |
| Zero-shim A2A interop with CrewAI | ecosystem-watch.md (2026-04-22) | Competitive claims |

---

*PMM draft 2026-04-23 — Phase 30 Remote Workspaces battlecard*
*Source material: Phase 30 positioning brief (Content Marketer, 2026-04-22), A2A v1.0 reference story brief (PMM, 2026-04-23), Phase 30 social copy (approved)*
*GA date: 2026-04-20 per phase30-launch-calendar.md*
