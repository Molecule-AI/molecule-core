# Social Queue — BLOCKED (blog not live)

**Status:** ⚠️ PARTIALLY BLOCKED — `docs.molecule.ai` still returning 000. Confirm uptime before firing any CTA links.
**Last updated:** 2026-04-22T21:25 UTC
**Blocker:** docs.molecule.ai not responding. Chrome DevTools MCP blog + EC2 Instance Connect SSH CTA both blocked.
**New entry added:** EC2 Instance Connect SSH (ready to post, publish day today — blocked on docs uptime)
**New entry added:** Slack Adapter (DRAFT — adapter not shipped, DO NOT POST)
**Confirmed live:** Discord Adapter Day 2 (Reddit + HN community copy)

---

## Campaign: Chrome DevTools MCP Day 1 (POST TODAY)

**Blog URL:** https://docs.molecule.ai/blog/chrome-devtools-mcp
**Hashtags:** #MCP #AIAgents #AgenticAI #MoleculeAI

### X Thread — 5 posts

**Post 1** (hook, P0: AI agent browser control)
```
Your AI agent just made a purchase on your behalf.

What did it buy? From where? With which account?

Most agents operate in a black box. Browser DevTools MCP makes the
browser a first-class tool — with org-level audit attribution on every action.

→ docs.molecule.ai/blog/chrome-devtools-mcp
```

**Post 2** (problem framing, P0: MCP browser automation)
```
Browser automation for AI agents usually means: give the agent your credentials,
hope it doesn't go somewhere unexpected, and check the logs after.

That's not a governance model. That's a trust fall.

Molecule AI's MCP governance layer for Chrome DevTools MCP gives you:
→ Which agent accessed which session
→ What it did (navigate, fill, screenshot, submit)
→ Audit trail with org API key attribution

One org API key prefix per integration. Instant revocation.

→ docs.molecule.ai/blog/chrome-devtools-mcp
```

**Post 3** (use case, concrete, P0: browser automation AI agents)
```
Real things teams use Chrome DevTools MCP for in production:

• Automated Lighthouse audits on every PR — agent runs the audit, reports the score, flags regressions
• Visual regression detection — agent screenshots key pages, diffs against baseline, opens tickets on drift
• Auth scraping — agent reads the authenticated state from an existing browser session

The governance layer means your security team can see all three in the audit trail.

→ docs.molecule.ai/blog/chrome-devtools-mcp
```

**Post 4** (competitive/positioning, P0: MCP governance layer)
```
The MCP protocol lets you connect any compatible tool to any compatible agent.

What's been missing: visibility into what the agent actually did with that access.

Molecule AI's MCP governance layer adds:
• Per-action audit logging with org API key attribution
• Token-scoped Chrome sessions — no credential sharing across agents
• Instant revocation without redeployment

→ docs.molecule.ai/blog/chrome-devtools-mcp
```

**Post 5** (CTA)
```
Chrome DevTools MCP launched April 20 as part of Molecule AI Phase 30.

If you're running AI agents that interact with web UIs — there's a governance story
you need to have ready before your security team asks.

→ docs.molecule.ai/blog/chrome-devtools-mcp

#MCP #AIAgents #AgenticAI #MoleculeAI
```

### LinkedIn Post — Single

```
Why your AI agent's browser access needs a governance layer

Your AI agent can use a browser. That's useful. But "useful" isn't a security posture.

When an agent operates inside a browser — filling forms, reading session state,
navigating authenticated flows — most platforms give you two options:
trust it completely, or don't let it near the browser at all.

Molecule AI's Chrome DevTools MCP integration adds a third option:
visibility with control.

Here's what "governance layer" actually means in this context:

→ Every browser action is logged with the org API key prefix that made the call.
   You know which agent touched what session, every time.

→ Chrome sessions are token-scoped. Agent A's session is not Agent B's session.
   No credential cross-contamination.

→ Revocation is instant. One API call, the key stops working, the session closes.
   No redeploy.

→ Audit trails are exportable. Your security team can review them without
   a custom logging pipeline.

This is the difference between "the agent can use a browser" and
"the agent's browser access is auditable, attributable, and revocable."

Chrome DevTools MCP is available now on all Molecule AI deployments.

→ docs.molecule.ai/blog/chrome-devtools-mcp

#MCP #AIAgents #AgenticAI #MoleculeAI #PlatformEngineering
```

---

## Campaign: Org-Scoped API Keys (POST THIS WEEK)

**Blog URL:** https://docs.molecule.ai/blog/org-scoped-api-keys
**Hashtag:** #OrgAPIKeys
**Hashtags:** #AIAgents #AgenticAI #MoleculeAI #Security

### X Thread — 5 posts

**Post 1** (hook — ADMIN_TOKEN SPOF)
```
Your production Molecule AI setup probably has one secret that can do everything: ADMIN_TOKEN.

Rotate it = downtime for every integration that holds a copy.
Leak it = your whole tenant is compromised.

That's not a security posture. That's a single point of failure.

Org-scoped API keys: named, revocable, per-integration.
No shared secrets. No blast radius from one rotation.

→ docs.molecule.ai/blog/org-scoped-api-keys
```

**Post 2** (solution — what org keys give you)
```
Molecule AI org-scoped API keys:

• Mint one key per integration — ci-bot, zapier, monitoring, whatever
• Revoke one key instantly, without touching anything else
• Audit trail shows org:keyId prefix on every call
• Full org scope: manage all workspaces, channels, secrets, templates

ADMIN_TOKEN is still there. But you never need to touch it again.

→ docs.molecule.ai/blog/org-scoped-api-keys
```

**Post 3** (feature — API walkthrough)
```
Mint a key in 30 seconds:

curl -X POST https://moleculesai.app/org/tokens \\
  -H 'Authorization: Bearer <your-org-key>' \\
  -d '{"name": "ci-bot"}'

Got a leaked key?
curl -X DELETE https://moleculesai.app/org/tokens/ci-bot \\
  -H 'Authorization: Bearer <your-org-key>'

The key stops working. The session closes. Nothing else touches.
That's surgical blast-radius control.

→ docs.molecule.ai/blog/org-scoped-api-keys
```

**Post 4** (enterprise angle)
```
Enterprise teams with multiple pipelines, integrations, and contractors:
org-scoped API keys change how you think about credential hygiene.

Every key has a label. Every call carries an org:keyId prefix.
last_used_at updated on every request.

You know exactly which pipeline made which call, every time.
No hunting through logs for the culprit.

→ docs.molecule.ai/blog/org-scoped-api-keys

#OrgAPIKeys
```

**Post 5** (CTA + compliance angle)
```
Org-scoped API keys: the difference between "we trust our integrations"
and "we can verify and revoke what our integrations do."

Named. Revocable. Audit-trail-enabled.
Built into Molecule AI. Not bolted on.

→ docs.molecule.ai/blog/org-scoped-api-keys

#OrgAPIKeys #AIAgents #AgenticAI #MoleculeAI
```

### LinkedIn Post — Single (governance/enterprise angle)

```
The problem with API keys for multi-agent platforms isn't key management.
It's that one key usually means one blast radius.

Rotate the ADMIN_TOKEN to stop a compromised integration —
and you've interrupted every other integration that holds a copy.

Molecule AI's org-scoped API keys solve this with a different model:

Named keys, per integration. Revocable individually, instantly.
Audit trail with org:keyId attribution on every call.
Full org scope — manage all workspaces, channels, secrets, and approvals.

The ADMIN_TOKEN stays functional as a break-glass fallback.
But the day-to-day runs on scoped keys — each one traceable,
each one independently revocable.

For enterprise teams: this is what compliance-ready credential management
looks like for AI agent platforms.

Org-scoped API keys are live now on Molecule AI.

→ docs.molecule.ai/blog/org-scoped-api-keys

#OrgAPIKeys #AgenticAI #AIAgents #MoleculeAI #PlatformEngineering #Security
```

---

## Campaign: Discord Adapter Day 2 — Reddit + HN (READY)

**Source:** `docs/marketing/discord-adapter-day2/announcement.md`
**Blog URL:** `docs.molecule.ai/blog/discord-adapter`

Community copy (not X/LinkedIn) — Reddit r/LocalLLaMA + r/MachineLearning + Hacker News.
File corrected: Discord adapter code path (`mcp-server/...` → `workspace-server/internal/channels/discord.go`).
File corrected: blog URL (`moleculesai.app/...launch` → `docs.molecule.ai/blog/discord-adapter`).

---

## Campaign: EC2 Instance Connect SSH — Post Today (READY)

**Source:** `docs/marketing/social/2026-04-22-ec2-instance-connect-ssh/social-copy.md`
**Blog URL:** `docs.molecule.ai/infra/workspace-terminal` (pending docs publish)
**Publish day:** 2026-04-22 (today)
**Status:** ⚠️ DO NOT POST until docs.molecule.ai is confirmed up and CTA link resolves
**Hashtags:** #AgenticAI #MoleculeAI #AWS #EC2InstanceConnect #PlatformEngineering #DevOps

> **Audit notes (2026-04-22 DevRel tick):**
> - PR #1533 cited correctly (terminal feature PR; #1531 stores instance_id)
> - All technical claims verified against `docs/tutorials/ec2-instance-connect-ssh/index.md`
> - CTA link blocked on docs publish — do not fire until confirmed
> - Visual assets (GIF, architecture diagram) specified in source file — confirm they exist before posting
> - Missing from queue entirely prior to this update — added 2026-04-22

---

## Campaign: Slack Adapter (DRAFT — DO NOT POST until adapter ships)

**Source:** `docs/marketing/campaigns/slack-adapter/social-copy.md`
**Feature:** `workspace-server/internal/channels/slack.go` — status: Planned (not yet shipped)
**Status:** ⚠️ DRAFT — adapter PR not merged, docs not published. Do NOT post.
**Hashtags:** #AgenticAI #MoleculeAI #Slack #PlatformEngineering #AIAgents

**Pre-launch checklist (update as items complete):**
- [ ] Adapter PR merged to main
- [ ] `docs.molecule.ai/blog/slack-adapter` published
- [ ] Slack bot token provisioning documented
- [ ] Allowlist behavior confirmed against implementation

---

## Campaign: Phase 32 Cloud SaaS Launch (DO NOT POST — pending GA)

**Status:** ⚠️ DRAFT — do not post until Phase 32 GA + Stripe Atlas confirmed
**Blog URL:** https://docs.molecule.ai/blog/phase-32-saas-launch (placeholder)

### X — 4 variants (post A–D spaced through launch week)

**Version A** (developer angle)
```
The runtime used to be your problem.

Docker config. Docker socket. Cloud credentials. Network rules.
That's before your agent actually does anything.

Molecule AI Cloud: create an org, get a canvas, launch agents.

No infra to run. No ops to maintain. You focus on what the agents do.

→ moleculesai.app
```

**Version B** (platform engineer — isolation story)
```
Molecule AI Cloud ships per-org Neon database branches and Firecracker microVMs.

Your agents and data are isolated by org — not shared infrastructure with good intentions.

Neon branch-per-org DB = query isolation.
Firecracker microVMs = compute isolation.
Platform handles the rest.

Zero ops. Production isolation.
→ moleculesai.app
```

**Version C** (indie/solo dev — fast to value)
```
Wanted to run AI agents without managing a server.

Signed up for Molecule AI Cloud. Had a canvas with 3 agents running in 15 minutes.
Used remote workspaces to wire in a script running on my Mac.

Zero Docker. Zero cloud config. One org.
→ moleculesai.app
```

**Version D** (A2A/multi-agent)
```
Most agent platforms give you one agent.

Molecule AI gives you an org: a canvas, A2A task dispatch, a secrets store,
and a fleet of heterogeneous agents — running on Docker, Fly.io,
or behind a NAT on your laptop.

Multi-agent orchestration without the infrastructure overhead.
→ moleculesai.app
```

### LinkedIn — Single (launch day)

```
The real cost of an agent platform isn't the agents.

It's the ops underneath: Docker configs, credential management,
network rules, monitoring, on-call rotation for your own infrastructure.

That's the model most agent frameworks sell you on.
You own the runtime. The agents do the work. You run the ops.

Molecule AI Cloud inverts that. You get:

→ A canvas that visualizes your full agent fleet — Docker, Fly.io, remote
→ A2A task dispatch between agents — regardless of where they run
→ A secrets store, org-scoped, with audit attribution
→ Org-level billing, usage metering, and quota controls
→ Neon branch-per-org database isolation
→ Firecracker microVMs for compute isolation

No ops. No own-the-runtime requirement.
Your team focuses on what the agents do, not the infrastructure that runs them.

From laptop to production in one command — or skip the laptop entirely
and use the cloud platform directly.

→ moleculesai.app

#AIAgents #AgenticAI #MoleculeAI #DevOps #PlatformEngineering #SaaS
```

---

---

## Campaign: Phase 33 Cloudflare Tunnel Migration (POST THIS WEEK)

**Status:** ⚠️ DRAFT — do not post until Phase 33 is live and docs published
**Blog URL:** https://docs.molecule.ai/blog/cloudflare-tunnel-migration (pending publish)
**Hashtags:** #AIAgents #AgenticAI #MoleculeAI #Cloudflare #DevOps

### X — 4 posts (spaced through launch week)

**Post 1** (hook — the change)
```
Your AI agent workspace used to connect to the platform through a Cloudflare Tunnel.

In Phase 33: it gets its own public IP.

Outbound tunnel daemon → direct WebSocket. No middleman.
~20–40ms latency reduction. No single dependency on Cloudflare for connectivity.

→ docs.molecule.ai/blog/cloudflare-tunnel-migration
```

**Post 2** (operator angle — what changes for infra teams)
```
Running AI agent workspaces in AWS?

Phase 33 means:
→ Workspaces get public IPs from your VPC subnet
→ Platform manages security group rules (port 443, TLS, JWT)
→ Direct WebSocket — no cloudflared daemon in the container
→ No inbound firewall holes required (same as before, different mechanism)

Migration is automatic on next workspace restart. Nothing to reconfigure.

→ docs.molecule.ai/blog/cloudflare-tunnel-migration
```

**Post 3** (developer angle — what doesn't change)
```
From the agent runtime: nothing changes.

Your code still registers with the platform, receives task dispatch,
runs tools, and talks to model APIs. The transport path is different
— the API contract is identical.

What does change: if you need to reach a workspace directly for
monitoring or health checks, you now have its public IP.
No tunnel hostname required.

→ docs.molecule.ai/blog/cloudflare-tunnel-migration
```

**Post 4** (security + CTA)
```
Every production agent fleet needs a connectivity story.

Phase 33 gives cloud-hosted workspaces: direct paths, platform-managed
security groups, no single dependency on a third-party tunnel provider.

Public IPs, not tunnel hostnames.

→ docs.molecule.ai/blog/cloudflare-tunnel-migration

#AIAgents #AgenticAI #MoleculeAI #DevOps #PlatformEngineering
```

### LinkedIn — Single post

```
The infrastructure story behind how cloud-hosted AI agent workspaces connect to their platform used to involve a third-party tunnel daemon running inside every container.

That changed this week with Phase 33.

Molecule AI cloud workspaces now get public IP addresses from their VPC subnet. The connection from the workspace to the platform is a direct WebSocket — no Cloudflare Tunnel in the path.

Here's what this means in practice:

**For platform operators:** the platform manages the security group rules. You don't open inbound ports or configure firewall rules — that's handled automatically. Workspaces migrate on their next restart cycle with no manual intervention.

**For developer tooling:** scripts and monitoring tools that need to reach a running workspace directly can now use its public IP. Health checks, log scraping, and port forwarding work without a tunnel hostname.

**For latency:** removing the Cloudflare tunnel hop reduces round-trip time by roughly 20–40ms depending on region. Not dramatic, but measurable at agent-fleet scale.

The connection model:
Browser → Platform API → Security Group (port 443, TLS, JWT) → Workspace (direct WebSocket)

No cloudflared daemon. No tunnel hostname. Just a public IP and a direct path.

Phase 33 is live now for all new CP-managed workspace provisions. Existing workspaces migrate on restart.

→ docs.molecule.ai/blog/cloudflare-tunnel-migration

#AIAgents #AgenticAI #MoleculeAI #DevOps #PlatformEngineering #AWS
```

---

*Queue prepared by DevRel 2026-04-22. All copy ready to fire once X_API_KEY + X_API_SECRET land in workspace env.*