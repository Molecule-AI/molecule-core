# Phase 30 — Sales Enablement Package

> **For:** Sales + Solutions Engineering | **Status:** Draft
> **Purpose:** Equip sellers with competitive battlecards, objection handlers, and demo talking points for Phase 30 Remote Workspaces

---

## Competitive Battlecards

### Battlecard 1: Molecule AI vs. Modal / Railway

**Their pitch:** "We handle the infra so you don't have to."
**Decision-maker's concern:** "You mean I give up control of my data?"

| Dimension | Molecule AI Phase 30 | Modal / Railway |
|---|---|---|
| **Compute ownership** | You own it — run on your laptop, your cloud, on-prem | They own it — serverless, you don't control the machine |
| **Data residency** | Agent compute stays on your infrastructure | Data processed on their infrastructure |
| **Multi-agent coordination** | A2A protocol, Canvas, org-scoped auth | Single-function inference calls |
| **Orchestration layer** | Yes — task dispatch, parent/child relationships | No — just inference |
| **Use case fit** | Agent fleets, coordination, autonomous pipelines | Short-lived inference jobs, batch processing |

**Winning talk track:**
> "Modal and Railway are inference platforms — they run your code on their infrastructure. Molecule AI is an orchestration layer — it runs on yours. If your concern is data residency or keeping compute on-premises, that's a fundamentally different category. We're not competing with Modal. We're solving a different problem."

---

### Battlecard 2: Molecule AI vs. Cursor / Copilot

**Their pitch:** "AI coding assistant built in to your IDE."
**Decision-maker's concern:** "Our team is already using Cursor. Why do we need this?"

| Dimension | Molecule AI Phase 30 | Cursor / Copilot |
|---|---|---|
| **Use case** | Autonomous multi-agent pipelines | One human + one AI pairing |
| **Agent autonomy** | Agents act without a human in the loop | Human drives every decision |
| **Coordination** | A2A, parent/child task dispatch | No coordination layer |
| **Scale** | Fleet of agents, mixed runtimes | Individual developer sessions |
| **Enterprise governance** | Org API keys, audit logs, MCP allowlists | Developer tool, no org-level controls |

**Winning talk track:**
> "Cursor and Copilot are incredible developer tools — one human, one AI, great for coding assistance. Molecule AI is an agent orchestration platform. When you want multiple autonomous agents that coordinate with each other — dispatching tasks, reporting status, working in parallel — that's a different product category. Phase 30 Remote Workspaces means you can run those agents wherever your compute lives. If your roadmap involves multi-agent systems, that's where we come in."

---

### Battlecard 3: Molecule AI vs. CrewAI / Autogen (open-source frameworks)

**Their pitch:** "Build multi-agent systems with open-source Python."
**Decision-maker's concern:** "Why pay for something we can build ourselves?"

| Dimension | Molecule AI Phase 30 | CrewAI / Autogen |
|---|---|---|
| **Operational burden** | Zero — platform manages infra, auth, heartbeat | You manage all of it — servers, scaling, auth |
| **Governance** | Org API keys, MCP allowlists, workspace audit logs | Diy — you build it yourself |
| **Canvas / observability** | Real-time workspace visibility, status, chat | No UI — code and logs only |
| **Deployment model** | Hybrid — container + remote, same org | Self-hosted only |
| **Time to value** | Hours | Weeks (to build the same capability) |
| **Maintenance** | Platform team owns uptime and updates | Your team maintains everything |

**Winning talk track:**
> "CrewAI and Autogen are solid frameworks for prototyping multi-agent systems. The problem is what comes after prototype: who maintains the servers, how do you add auth, where's the observability, how do you govern what agents can do. That's a significant engineering investment before you get to production. Molecule AI gives you the coordination layer on day one. Phase 30 means you can even run the agents on your own infrastructure if that's a requirement. The open-source framework gets you to prototype faster. We get you to production faster."

---

### Battlecard 4: Molecule AI vs. Windsurf / Devin

**Their pitch:** "Autonomous coding agent."
**Decision-maker's concern:** "Autonomous agents sound good but they scare my security team."

| Dimension | Molecule AI Phase 30 | Windsurf / Devin |
|---|---|---|
| **Governance** | MCP allowlists, org API keys, audit trail | No org-level governance model |
| **Browser access** | Chrome DevTools MCP + Molecule AI governance layer | Raw CDP, no control layer |
| **Multi-agent fleet** | Yes — full A2A coordination | Single-agent only |
| **Observability** | Canvas, real-time status, task chat | Developer tool UI only |
| **Enterprise readiness** | SOC 2-ready, org-scoped auth, session tier | Early-stage, not enterprise-hardened |

**Winning talk track:**
> "The autonomous coding agents are getting good — but they're a single-agent paradigm. When you want a fleet of agents, or when your security team needs to control what an agent can do with a browser or an API key, you need a governance layer on top. That's what Molecule AI adds. Phase 30's Chrome DevTools MCP integration, for example, gives an agent browser access through your org's MCP allowlist — with a full audit trail. That's not something you get with a standalone autonomous coding tool."

---

## Objection Responses

### "Our data can't leave our infrastructure."

**Response:**
> "Phase 30 was built for exactly that requirement. Remote Workspaces let you run the agent on your own machine, your own cloud account, your on-premises server. The platform handles orchestration and coordination — the agent compute runs where your data lives. This isn't a workaround. It's the primary deployment model."

**Proof point:** "A data engineering team is currently using this for a pipeline agent running in their own AWS account — raw data never touches the Molecule AI platform."

---

### "This sounds complicated. Our team doesn't want to manage more infrastructure."

**Response:**
> "There's two ways to run it. Container workspaces are fully managed — you don't touch the infra. Remote Workspaces are for when you specifically need the agent to run elsewhere. Most teams use both: managed agents for standard tasks, remote agents for data-locality or environment-specific requirements."

**Proof point:** "The mixed-fleet pattern means you only manage what you need to manage. Canvas shows everything in one view regardless of runtime."

---

### "We already have a team that manages agent infrastructure. Why would we add Molecule AI?"

**Response:**
> "Because you're managing the orchestration layer yourself. Molecule AI replaces the custom coordination code — A2A task dispatch, parent/child relationships, auth, heartbeat, observability. That's nontrivial to build and maintain. We give you the platform; your team focuses on what the agents actually do."

---

### "How is this different from just running agents in Kubernetes?"

**Response:**
> "Kubernetes manages containers. It doesn't manage agent identity, task dispatch, or coordination. With Remote Workspaces, you get the platform layer — Canvas, A2A, org-scoped auth, audit logs — without needing a custom-built orchestration system. The agent still runs on your infra, but it's registered to the platform."

---

### "What's the pricing difference between remote and container workspaces?"

**Response:**
> "At GA launch, remote and container workspaces are priced identically. Future tiers may differentiate on egress or storage, but that's not in the current release. There's no premium for the remote runtime specifically."

---

## Demo Talking Points — Phase 30 (3-minute live demo script)

### Opening (30s)
> "I'm going to show you two things today: how an agent runs on my laptop, and how it coordinates with agents running on the platform — same Canvas, same A2A, same auth."

**Do:** Open Canvas, show one container workspace + one remote workspace both online.

---

### Setup moment (60s)
> "This agent is running on my local machine. I installed it with a single command. It registered with the org and appeared here within 10 seconds. No inbound ports, no VPN — just outbound HTTPS to the platform."

**Do:** Terminal — run `python3 run.py` show registration output, cut to Canvas showing REMOTE badge.

---

### Coordination moment (60s)
> "Now I'm going to dispatch a task from the PM agent — which is running in a container on the platform — to the remote agent on my laptop. Watch Canvas."

**Do:** PM dispatches task, researcher on remote laptop receives and executes, result returned to PM, Canvas shows both active during coordination.

---

### Close (30s)
> "Two runtimes, one Canvas. Same auth, same A2A protocol. Where the agent runs is a deployment choice — not an architectural constraint."

**Do:** Canvas full screen, both agents active. Point to REMOTE badge.

---

## Quick-Start Checklist for Sales Engineers

Before a remote workspace demo, verify:
- [ ] Agent binary installed on demo machine (`curl -sSL https://get.moleculesai.app | bash`)
- [ ] `molecule login --org [customer-org]` authenticated
- [ ] `molecule workspace init --name demo-agent --runtime remote` created
- [ ] Workspace appears in Canvas within 10s of startup
- [ ] REMOTE badge visible on workspace card
- [ ] A2A messages route successfully to/from remote workspace
- [ ] Cloudflare Artifacts repo can be attached (if demoing the feature)

---

## Objection → Champion Mapping

Use this to help your champion build internal arguments:

| Objection | Internal argument to make |
|---|---|
| "Data residency" | Phase 30 is the only platform with remote runtime + data residency |
| "Too complex" | Mixed fleet means you only use remote when you need it |
| "Why not just use Kubernetes" | We handle orchestration — they handle compute |
| "Price" | Remote = container pricing at GA; no premium |
| "Security" | MCP governance + org API keys apply to remote identically |

---

*Drafted by DevRel. Sales Engineers should customize the talk tracks to their own voice before customer calls.*
