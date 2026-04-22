# Social Copy — Phase 30 Remote Workspaces / SaaS Federation

## Blog Post (Live)
**URL:** `docs/blog/2026-04-20-remote-workspaces/index.md`
**Title:** "One Canvas, Every Agent: Remote AI Agents and Fleet Visibility on Molecule AI"

---

## X / Twitter Thread

**Post 1 (Hook — fleet visibility problem):**
> Your AI agents are scattered across 6 different clouds, 3 VPNs, and someone's laptop.
Each one has its own token. Its own dashboard. Its own on-call rotation.

Molecule AI's Phase 30 ships one canvas that sees all of it.

---

**Post 2 (What it is):**
> Remote agents are now first-class citizens on the Molecule AI canvas.

Register any agent — laptop, cloud VM, CI/CD runner, on-prem server — with a per-workspace bearer token. Send heartbeats every 30s. Done.

The canvas shows a purple REMOTE badge. That's how you know it's running on *your* infra, not ours.

---

**Post 3 (The security model):**
> Here's what "remote agent" means for your security posture:

→ Bearer token issued once at registration, never again
→ Secrets fetched on demand via API — never hardcoded or in env blocks
→ Heartbeat TTL: 90s offline threshold, no silent failures
→ X-Workspace-ID header for cross-network A2A — audit trail on every message

Built for production teams, not demos.

---

**Post 4 (Use cases):**
> What actually runs on remote agents today:

→ CI/CD pipelines that open PRs, run tests, and post results back
→ Laptops that run dev agents between standups
→ On-prem servers that can't be containerized
→ Cloud VMs in other regions — same canvas, different infra

All of them visible from one place.

---

**Post 5 (CTA + tutorial):**
> New tutorial: "Register a Remote Agent on Molecule AI"

6 steps — external workspace, bearer token, heartbeat loop, A2A messaging.
Copy-paste Python example included.

→ [Read the tutorial](https://github.com/Molecule-AI/molecule-core/blob/main/docs/tutorials/register-remote-agent.md)
→ [Full launch post](https://github.com/Molecule-AI/molecule-core/blob/main/docs/blog/2026-04-20-remote-workspaces/index.md)

---

## LinkedIn Post

**Single post:**

We shipped Phase 30 — and the headline is fleet visibility.

If you're running AI agents across multiple environments (and most production teams are), you've probably built custom dashboards to track them, shared tokens that nobody wants to rotate, and lost sleep over whether that agent on the VPN is still alive.

Molecule AI's Remote Agents changes this. Register any agent — laptop, cloud VM, CI/CD runner, on-prem — with a per-workspace bearer token and a 30-second heartbeat. It appears on your canvas with a REMOTE badge. You manage it from there.

The security model is deliberate: tokens shown once, secrets pulled on demand, no long-lived credentials floating around. If an agent goes offline for 90 seconds, the canvas reflects it immediately.

If you've been managing a fleet of agents with a spreadsheet and Slack, this is the upgrade.

→ [Tutorial: Register a Remote Agent](https://github.com/Molecule-AI/molecule-core/blob/main/docs/tutorials/register-remote-agent.md)
→ [Full launch post](https://github.com/Molecule-AI/molecule-core/blob/main/docs/blog/2026-04-20-remote-workspaces/index.md)

#AIagents #fleetmanagement #selfhosted #DevOps #AIAgents

---

## Visual Assets

| Platform | Asset | File |
|---|---|---|
| X (hook) | Fleet diagram | `marketing/assets/phase30-fleet-diagram.png` |
| X (security) | Token lifecycle card | `marketing/devrel/campaigns/phase30-remote-workspaces/assets/token-lifecycle-card.png` |
| LinkedIn | Canvas fleet mockup | `marketing/devrel/campaigns/phase30-remote-workspaces/assets/canvas-fleet-mockup.png` |
| CTA | "One canvas, every agent." + GitHub link | |

---

## Publishing Schedule

| Platform | When | Notes |
|---|---|---|
| X thread | Day of publish, 9am PT | 5 posts, staggered 20-30 min |
| LinkedIn | Day of publish, 11am PT | Same day as X |
| Reddit r/LocalLLaMA | Day of publish, 12pm PT | Angle: fleet management for self-hosted agents |
| Reddit r/MachineLearning | Day of publish, 1pm PT | Angle: multi-cloud agent orchestration |

---

## Keyword Targeting

Primary: `remote AI agent deployment` + `self-hosted AI agents platform`
Secondary: `federated AI agents`, `AI agent fleet management`, `multi-cloud AI agent platform`

Thread posts should organically include "remote agent deployment" and "self-hosted" where natural.

---

*Draft by SEO Analyst 2026-04-21 — coordinating with Content Marketer on blog expansion (Action 3) and Social Media Brand on thread timing (#1182)*