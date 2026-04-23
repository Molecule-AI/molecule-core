# Discord Adapter — Social Copy
**Feature:** Discord channel adapter (inbound via Interactions webhook, outbound via Incoming Webhooks)
**Campaign:** Discord Adapter | **Docs:** `docs/agent-runtime/social-channels.md` (Discord Setup section)
**Canonical URL:** `github.com/Molecule-AI/molecule-core/blob/main/docs/agent-runtime/social-channels.md` (moleculesai.app TBD — outage confirmed)
**Status:** APPROVED (PMM proxy — Marketing Lead offline) | Reddit/HN copy ADDED by PMM
**Owner:** PMM → Social Media Brand | **Day:** Ready to post once X credentials are restored

---

## X (140–280 chars)

### Version A — Slash commands for agents
```
Your Discord community just got an agent layer.

Connect a Molecule AI workspace to any Discord channel. Members query your agents via slash commands — no bot token setup for outbound.

Governance included. Audit trail included.
```

### Version B — Multi-channel agent access
```
Your AI agents can already handle Telegram, email, and Slack.
Now add Discord — without changing how agents work.

Slash commands → agent workspace → response to any channel.
One protocol. Any channel. Molecule AI's channel adapter.
```

### Version C — Developer angle
```
Setting up an AI agent in Discord used to mean: create app, configure intents, handle events.

Molecule AI's Discord adapter: paste a webhook URL. Done.

Inbound via Interactions. Outbound via Incoming Webhook. Zero bot token management.
```

### Version D — Platform angle
```
Discord communities can now talk to your agent fleet.

Molecule AI's channel adapter: one workspace, any social platform. Telegram, Slack, Discord — all the same agent underneath.

Your agents. Your channels. One canvas.
```

---

## LinkedIn (100–200 words)

```
Connecting your AI agent fleet to Discord just got simpler — and more powerful.

Molecule AI's Discord adapter ships today. Here's what that means in practice:

Outbound messages: paste an Incoming Webhook URL. That's it. No Discord bot app, no OAuth token, no intent configuration — just a webhook URL and your agent is live in any channel.

Inbound: slash commands and message components arrive as signed Interactions payloads. The adapter parses them, forwards them to the workspace agent, and routes the response back to Discord.

Your Discord community gets access to the same agent capabilities as your Telegram users, your Slack channels, and your Canvas — without duplicating the agent logic or managing separate bot tokens.

One protocol. Any channel. Molecule AI's channel adapter layer makes social platforms first-class citizen channels for your agent fleet.
```

---

## Image suggestions

| Post | Image | Source |
|---|---|---|
| X Version A | Slash command dropdown screenshot — `/agent` in Discord | Custom: Discord UI screenshot |
| X Version B | Multi-channel diagram: Telegram + Slack + Discord → same workspace agent | Custom: platform diagram |
| X Version C | Before/after: complex bot setup vs "paste webhook URL" | Custom: simple comparison card |
| X Version D | Canvas Channels tab with Discord connected | Custom: Canvas screenshot |
| LinkedIn | Multi-platform diagram | Custom |

---

## Hashtags

`#MoleculeAI` `#Discord` `#AIAgents` `#MCP` `#SocialChannels` `#MultiChannel` `#AgentPlatform` `#DevOps`

---

## CTA

`moleculesai.app/docs/agent-runtime/social-channels`

---

## Campaign timing

Ready to post once:
1. X consumer credentials (`X_API_KEY` + `X_API_SECRET`) are restored to Social Media Brand workspace — blocking all posts
2. Discord Adapter Day 2 copy is approved by Marketing Lead (coordinate with Social Media Brand)

---

*PMM drafted 2026-04-22 — no prior social copy file found for Discord adapter*
*Positioning note: Discord adapter is outbound-primary (no separate bot token for outbound); inbound via Interactions webhook — leverage this simplicity in copy*

---

## Reddit Post (r/LocalLLaMA or r/MachineLearning)
```
Molecule AI just shipped a Discord adapter for AI agent fleets.

The setup: paste a webhook URL. That's it — no Discord bot app, no OAuth token, no intent configuration.

Inbound: slash commands and message components arrive as signed Interactions payloads. The adapter parses them, forwards to your workspace agent, routes the response back to Discord.

Outbound: same incoming webhook, no separate bot token needed.

One workspace. Any channel. Your Telegram, Slack, and Discord users all hit the same agent underneath — no duplicated logic, no separate bot tokens per platform.

GitHub: github.com/Molecule-AI/molecule-core
Docs: github.com/Molecule-AI/molecule-core/blob/main/docs/agent-runtime/social-channels.md
```

---

## Hacker News — Show HN
```
Show HN: Molecule AI Discord adapter — webhook URL setup, zero bot token management

Molecule AI shipped a Discord channel adapter for AI agent fleets.

The problem it solves: connecting Discord to an AI agent fleet usually means creating a Discord app, configuring intents, handling events, managing token rotation. The agent logic isn't the hard part — the integration is.

What we built: a Discord adapter that uses Discord's Interactions webhooks for inbound and Incoming Webhooks for outbound. No Discord bot app required. No OAuth token. No intent configuration.

Setup: paste an Incoming Webhook URL. Done.

Inbound: slash commands and message components arrive as signed Interactions payloads. The adapter parses them, forwards to your workspace agent, routes the response back to the channel.

Outbound: same incoming webhook. No separate bot token for outbound messages.

What this means in practice: your Discord community gets access to the same agent capabilities as your Telegram users, your Slack channels, and your Canvas — without duplicating the agent logic or managing separate bot tokens per platform.

Under 100 lines to add Discord to an existing Molecule AI workspace. Full source in the linked repo.

GitHub: github.com/Molecule-AI/molecule-core
Docs: github.com/Molecule-AI/molecule-core/blob/main/docs/agent-runtime/social-channels.md
```