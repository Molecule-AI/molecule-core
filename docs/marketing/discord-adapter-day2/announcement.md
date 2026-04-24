# Discord Adapter Day 2 — Community Copy

> Posted 2026-04-21. Discord adapter launched Day 1; Day 2 covers Reddit, Hacker News.
> Blog URL: https://docs.molecule.ai/blog/discord-adapter-launch
> PR: https://github.com/Molecule-AI/molecule-core/pull/656

---

## Reddit r/LocalLLaMA

**Title:** Molecule AI now connects to Discord via a webhook — no bot account, no Gateway, no OAuth

```
Molecule AI workspaces can now send messages to Discord and receive slash commands using only a webhook URL. No Discord Developer Portal, no intents, no bot token — just an inbound webhook and your agent is in the channel.

Built it as a proof-of-concept to keep our own team workflow on Discord without the overhead of a full bot app. Figured other people might want the same thing.

The adapter uses Discord's built-in webhook delivery for outbound + slash command reception. No polling. No Gateway connection. Works behind NAT — the agent initiates all outbound connections to the platform, which proxies to Discord.

Here's the architecture gist:
- Outbound: POST to Discord webhook URL (standard, no auth beyond the URL token)
- Inbound: Discord delivers slash command payloads to a platform endpoint; platform fans out to the relevant workspace via A2A
- No Discord bot app required. No Developer Portal setup.

If your team lives in Discord and you want an AI agent that can post summaries, respond to /ask commands, and route alerts — it's now a webhook URL and a config line.

Demo repo and docs: https://github.com/Molecule-AI/molecule-core/tree/main/docs/blog/2026-04-21-discord-adapter

Happy to answer questions about the adapter design.
```

**Tags:** `discord`, `mcp`, `molecule-ai`, `webhook`, `ai-agents`

---

## Reddit r/MachineLearning

**Title:** Show HN: Molecule AI Discord adapter — AI agents in Discord via webhook, no bot account needed

```
Show HN: Molecule AI Discord adapter — webhook-only, no Gateway connection required

HN: built a Discord integration for Molecule AI workspaces that requires zero bot app setup. It's just a webhook URL and an agent config.

The problem: Discord bot integrations typically require a Developer Portal app, OAuth flow, Gateway connection management, intent configuration, and rate limit handling. That's a meaningful chunk of work before your agent can say hello.

The approach: use Discord's native webhook delivery for inbound slash commands (no Gateway) and standard webhook POST for outbound messages. The platform acts as a proxy — Discord delivers to the platform endpoint, the platform routes to the relevant workspace via A2A. Works behind NAT since the agent initiates outbound connections.

No bot token. No intents. No Gateway.

Code: https://github.com/Molecule-AI/molecule-core/tree/main/docs/blog/2026-04-21-discord-adapter
Launch post: https://docs.molecule.ai/blog/discord-adapter-launch
```

---

## Hacker News

**Title:** Molecule AI — Discord adapter via webhook (no bot account, no Gateway)

**Body:**

Built a Discord integration for Molecule AI workspaces that works with just a webhook URL — no Discord Developer Portal setup, no bot token, no Gateway connection.

**Why**

Our own team lives in Discord. We wanted a lightweight way to have an AI agent respond to slash commands and post updates without the overhead of a full bot app. Realized Discord's native webhook primitives cover both inbound (slash command delivery) and outbound (channel messages) if you proxy through a platform endpoint.

**How it works**

- Outbound: agent POSTs to a Discord webhook URL (standard, URL contains the auth token)
- Inbound: Discord delivers slash command payloads to a platform endpoint; platform fans out to the relevant workspace via A2A
- No bot account required. No Gateway. Works behind NAT — the agent only initiates outbound connections.

The adapter lives in the MCP server (`mcp-server/src/tools/channels/discord.go`) alongside Telegram and other channel adapters. Each workspace configures its own Discord channel with a webhook URL.

**Links**

- Docs: https://docs.molecule.ai/blog/discord-adapter-launch
- Code + examples: https://github.com/Molecule-AI/molecule-core/tree/main/docs/blog/2026-04-21-discord-adapter
- PR: https://github.com/Molecule-AI/molecule-core/pull/656
