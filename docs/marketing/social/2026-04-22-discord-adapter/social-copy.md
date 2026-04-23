# Discord Adapter — Social Copy
Campaign: Discord Adapter | Source: `docs/blog/2026-04-21-discord-adapter/`
Publish day: 2026-04-22 (Day 2)
Status: ✅ APPROVED — PMM proxy review 2026-04-22 (Marketing Lead approval confirmed)
Blog URL: `https://docs.molecule.ai/blog/2026-04-21-discord-adapter`

---
## Campaign Angle

**Problem:** Traditional Discord bot setup requires a Developer Portal app, OAuth2, Gateway connection, intents configuration, and rate limit handling — significant overhead before the agent can say hello.

**Solution:** Molecule AI Discord adapter uses two standard Discord primitives only: an Incoming Webhook and an Interactions Endpoint URL. No bot account, no OAuth flow, no Gateway. One webhook URL — that's the only credential.

**Audience:** DevOps engineers, internal tooling teams, community managers running AI agent workflows.

---

## X (Twitter) — 3-post thread

### Post 1 — Problem framing
```
Setting up a Discord bot for your AI agent typically requires:
- Developer Portal app
- OAuth2 flow
- Gateway connection
- Intent configuration
- Rate limit handling

That's before it can say "hello" in a channel.

There's a simpler path.
```

### Post 2 — The webhook approach
```
Molecule AI's Discord adapter: two primitives, no bot account.

One Incoming Webhook → your agent sends to any Discord channel.
One Interactions Endpoint URL → slash commands route to your agent.

No Gateway. No OAuth. No polling.

Connect a workspace to Discord in under a minute.
```

### Post 3 — Use cases
```
Your AI agent can now receive `/ask` in a Discord channel.

Real use cases:
→ Incident triage: agent runs checks, posts formatted status to #incidents
→ CI/CD coordination: build results + rollback recommendations to DevOps channel
→ Community management: `/support` routes to sub-agent, returns answer to Discord

Slash commands are the interface. The agent decides what to do.
```

---

## X (Twitter) — Alternative single post (240 chars)
```
Traditional Discord bot setup:
Developer Portal → OAuth2 → Gateway → intents → rate limits → finally, hello.

Molecule AI Discord adapter:
Incoming Webhook → done.

Agents receive `/slash` commands and post to any channel. No bot account required.
```

---

## LinkedIn — Single post (~160 words)

```
Connecting an AI agent to Discord used to mean building a bot. Developer Portal app, OAuth2, Gateway connection, intent configuration — that's a non-trivial project before your agent can post a single message.

Molecule AI's Discord adapter takes a different approach: two standard Discord primitives, no bot account required.

An Incoming Webhook is your only credential. Your workspace agent can send messages directly to any Discord channel — long responses are auto-split into Discord-safe chunks. Slash commands route to the agent via the Interactions Endpoint URL. No polling, no Gateway, no message-reading permissions.

The setup takes under a minute: create a webhook, paste it into the Channels tab in Canvas, done.

Use cases the adapter enables: incident triage agents that post to #incidents, CI/CD agents that relay build results to DevOps channels, community management agents that receive `/support` and route to sub-agents.

One webhook URL. Full AI agent capability in Discord.

#Discord #AIAgents #DevOps #MoleculeAI #InternalTools
```

---

## Reddit — r/LocalLLaMA

**Title:** Molecule AI Discord adapter — no bot account, no OAuth, just a webhook URL

**Body:**
Most Discord integrations for AI agents require building a full bot: Developer Portal app, OAuth2 flow, Gateway connection, intent configuration. That's a significant chunk of work before your agent can say hello.

The Molecule AI Discord adapter works differently. You create an Incoming Webhook from any Discord channel — one URL, generated from the channel's Integrations settings. That's the only credential. Your workspace agent can now send messages to that channel. Long responses are auto-split into Discord-safe chunks.

Inbound slash commands route to your agent via the Interactions Endpoint URL. No polling, no Gateway, no message-reading permissions.

The setup: Channel Settings → Integrations → Webhooks → New Webhook → paste into Canvas → done. Under a minute.

Real-world use cases: incident triage agents posting to #incidents, CI/CD agents relaying build results to DevOps channels, community management agents receiving `/support` and routing to sub-agents.

Full details in the [launch post](https://docs.molecule.ai/blog/2026-04-21-discord-adapter). GitHub: [PR #656](https://github.com/Molecule-AI/molecule-core/pull/656).

---

## Hacker News — Show HN

**Title:** One webhook URL to connect an AI agent to Discord — no bot account, no OAuth

```
We shipped a Discord adapter for Molecule AI workspaces.

The traditional approach to Discord + AI agents requires: Developer Portal app, OAuth2, Gateway connection, intent configuration, rate limit handling. That's a non-trivial project.

The Molecule AI approach: Incoming Webhook + Interactions Endpoint URL. No bot account, no OAuth flow, no Gateway.

Outbound: workspace agent → Discord channel (auto-splits long messages)
Inbound: `/slash` commands → agent → back to Discord

Setup: create webhook in Discord → paste into Canvas Channels tab → done.

Use cases: incident triage, CI/CD coordination, community management via slash commands.

PR: molecule-core#656
Blog: docs.molecule.ai/blog/2026-04-21-discord-adapter
```

---

## Image suggestions

| Post | Image |
|---|---|
| X Post 1 (Problem) | Comparison card: "Traditional bot setup" (6 steps, complex) vs "Molecule AI" (1 webhook URL) |
| X Post 2 (Solution) | Canvas screenshot: Discord channel connected, showing "Connected" status |
| X Post 3 (Use cases) | 3-item list graphic: Incident triage / CI-CD / Community management |
| LinkedIn | Canvas screenshot or quote card: "One webhook URL. That's the only credential." |
| Reddit | Same comparison card as X Post 1 |
| HN | No image — technical, text-only |

---

## Coordination

- **Day 1 (2026-04-21):** Chrome DevTools MCP — blocked on social credentials
- **Day 2 (2026-04-22):** Discord Adapter — this file, post after credentials confirmed
- **Day 3 (2026-04-23):** Org-Scoped API Keys — `docs/marketing/social/2026-04-23-org-api-keys/social-copy.md`
- **Day 4 (2026-04-24):** EC2 Console Output — approved in `docs/marketing/social/2026-04-24-ec2-console-output/social-copy.md`
- **Day 5 (2026-04-25):** Cloudflare Artifacts — approved, coordinate with Marketing Lead

*Approved by PMM proxy 2026-04-22. Ready for Social Media Brand agent to execute posting.*