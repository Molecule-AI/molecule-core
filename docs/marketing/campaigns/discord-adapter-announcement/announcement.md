# Discord Adapter Announcement — PR #656 / Issue #1183

**Status:** DRAFT — needs Social Media Brand review before posting
**Platforms:** Discord, Reddit (r/LocalLLama, r/MachineLearning), dev.to
**Coordination:** Thread #1182 timing TBD — flag for Social Media Brand

---

## Announcement Copy

**Molecule AI Discord adapter is live — PR #656 merged.**

Your Molecule AI workspace can now connect to Discord. Here's what shipped:

**Send messages to Discord**
→ Configure a Discord Incoming Webhook (no bot token needed for outbound)
→ Your workspace agent sends messages to any Discord channel via webhook
→ 2000-character chunking handled automatically

**Receive slash commands from Discord**
→ Register your Discord app's Interactions endpoint with Molecule AI
→ Slash commands like `/ask what's the status?` route directly to your workspace agent
→ Works in servers and DMs — username and channel are passed through as metadata

**Security:** Webhook tokens are never logged — regression-tested in PR #659.

**Setup:** One webhook URL. Three lines of config. No separate bot account required for outbound.

→ [Docs: Social Channels](/docs/agent-runtime/social-channels#discord-setup)
→ [Docs: Discord Adapter source](/workspace-server/internal/channels/discord.go)

---

## Short Version (for Reddit / dev.to title)

> Molecule AI workspaces can now connect to Discord — send messages and receive slash commands via a webhook. No bot token needed for outbound. PR #656 merged.

---

## Dev.to Post Body

Molecule AI workspaces now ship with a Discord adapter — giving your AI agents a presence in Discord servers.

**What you can do:**
- Send messages to any Discord channel from your workspace agent (webhook-based, no bot token needed for outbound)
- Receive slash commands — `/ask`, `/help`, `/status` — and route them to your workspace agent
- Works in servers and DMs
- 2000-character message chunking handled automatically
- Webhook tokens are never logged (security fix in PR #659)

**Configuration:**

```bash
curl -X POST http://localhost:8080/workspaces/${WORKSPACE_ID}/channels \
  -H 'Authorization: Bearer ${TOKEN}' \
  -H 'Content-Type: application/json' \
  -d '{
    "channel_type": "discord",
    "config": {
      "webhook_url": "https://discord.com/api/webhooks/YOUR_WEBHOOK_ID/YOUR_WEBHOOK_TOKEN"
    }
  }'
```

Or connect via the Canvas UI — Channels tab → + Connect → Discord.

**Architecture:**
- Outbound: Discord Incoming Webhooks (HTTP POST, no long-polling)
- Inbound: Discord Interactions endpoint (slash commands and message components)
- No separate bot token required for outbound-only setups

Full docs: [Social Channels guide](/docs/agent-runtime/social-channels)

GitHub: [PR #656 — Discord adapter](https://github.com/Molecule-AI/molecule-core/pull/656)

---

## Discord Message (for posting in Molecule AI's own Discord server)

**Molecule AI Discord Adapter is live! 🎉**

Your workspace can now connect to Discord — send messages to channels and receive slash commands from users.

**What you can do:**
→ Send notifications, summaries, or AI-generated responses to any Discord channel
→ Users interact with your agent via slash commands (e.g. `/ask <question>`)
→ Works in servers and DMs — no separate bot token needed for outbound

**How to connect:**
1. Create a Discord webhook (Channel → Integrations → Webhooks)
2. Add it to your workspace: Channels tab → + Connect → Discord
3. Done — your agent can now send to that channel

For slash commands inbound, point your Discord app's Interactions URL at `POST /webhooks/discord` on your platform.

Docs: docs/agent-runtime/social-channels

---

<<<<<<< HEAD
---

## Reddit / HN — Day 2 Campaign

**Status:** Ready for review and push. Blog post URL TBD — fill before posting.

---

### r/LocalLLaMA — Post Title

> Molecule AI Discord adapter: connect any AI agent workspace to Discord with one webhook URL

### r/LocalLLaMA — Body

Molecule AI workspaces can now connect to Discord.

Here's what makes this different from a typical bot integration:

Traditional Discord bot setup requires: Developer Portal app, OAuth2, Gateway connection, intent configuration, message-reading permissions, rate limit handling.

The Molecule AI Discord adapter requires: **one webhook URL**.

That's the only credential. It encodes the channel and bot tokens. You paste it in the Canvas Channels tab. Done.

What you get:
- Slash commands (`/ask`, `/status`, `/help`) route directly to your workspace agent — no message reading, no polling
- Agent responses post back to the Discord channel automatically
- 2,000-character chunking handled without code
- Works in servers and in DMs

The webhook token is never logged — errors surface as generic messages, not URL fragments (security fix shipped in PR #659).

This is the same adapter interface that handles Telegram. New channels add one implementation, and the full CRUD API, Canvas UI, and MCP tools work automatically.

**Setup:** Canvas → Workspace → Channels tab → + Connect → Discord → paste webhook URL.

Docs → [Social Channels guide](https://github.com/Molecule-AI/molecule-core/blob/main/docs/agent-runtime/social-channels.md)

GitHub → [PR #656 — Discord adapter](https://github.com/Molecule-AI/molecule-core/pull/656)

---

### Hacker News — Post Title

> Show HN — Molecule AI Discord adapter: one webhook, full agent interaction in Discord

### Hacker News — Body

Show HN: Molecule AI workspaces can now connect to Discord.

Most Discord bot integrations require creating an app in the Developer Portal, handling the Gateway connection, configuring intents and permissions, and managing rate limits — before your agent can say hello in a channel.

The Molecule AI approach uses two standard Discord primitives:

- **Incoming Webhooks** for outbound messages — you give the workspace a webhook URL, that's the only credential, the agent can send to any channel
- **Discord Interactions** for inbound slash commands — users type `/ask what's the deployment status?`, the adapter reconstructs it as plain text and routes it to your workspace agent

No Gateway. No message-reading permissions. No long-polling.

Slash commands are the interface. The agent decides what to do. Your Discord server is the front-end your team already lives in.

The security model is deliberate: webhook tokens are never logged. This was hardened in PR #659 after a security review.

Setup is under a minute: Canvas → Channels tab → + Connect → Discord → paste your webhook URL.

Demo + full docs: https://github.com/Molecule-AI/molecule-core/blob/main/docs/agent-runtime/social-channels.md

---

*Draft by Content Marketer 2026-04-21 — Day 2 campaign. Fill blog URL before posting. Coordinate with Social Media Brand on timing.*
=======
*Draft by Content Marketer 2026-04-20 — for Social Media Brand review before publishing*
>>>>>>> origin/staging
