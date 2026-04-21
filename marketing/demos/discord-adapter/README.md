# Discord Adapter Demo — Storyboard + Walkthrough
> Source: PR #1209 — `docs(marketing): Discord adapter launch visual assets`
> Token unavailable — demo prepared locally, ready to push when token clears

---

## What the Discord Adapter Does

The Discord adapter lets Molecule AI agents send and receive messages via Discord — using **Incoming Webhooks** for outbound and **Discord Interactions** (slash commands) for inbound. No bot token required for basic setup.

---

## Architecture Overview

```
Molecule AI Agent
    │
    │ SendMessage() → Discord Webhook URL
    ▼
Discord Channel
    │
    │ Slash command → /status, /help, /delegate
    ▼
Molecule AI Canvas ← agent activity
```

---

## Storyboard: Discord Adapter Setup + Demo

### Scene 1: Channel Configuration

**Narrator:** "Add Discord as a channel in Canvas. You only need a webhook URL — no bot token for outbound-only."

**Screen:** Canvas → Workspace → Config → Channels → Add Discord

**Config block:**
```yaml
channels:
  - type: discord
    enabled: true
    webhook_url: "https://discord.com/api/webhooks/123456789/abcdefgh"
    # outbound: SendMessage via webhook
    # inbound: Discord Interactions (slash commands) → ParseWebhook
```

**Key fields:**
- `webhook_url` — Discord Incoming Webhook (get from Discord channel → Edit Channel → Integrations → Create Webhook)
- No bot token needed for outbound-only setup
- For slash commands (inbound): configure Interactions URL in Discord Developer Portal

---

### Scene 2: Outbound Message

**Narrator:** "The agent sends messages via Discord webhook. Long messages are auto-split at 2000 chars."

**Demo:**
```python
# Agent sends a report to the #deployments channel
await channel.send_message(
    config={"webhook_url": "https://discord.com/api/webhooks/123456789/abcdefgh"},
    chat_id="ignored",
    text="✅ Deployment complete: v2.1.4 → production (3 agents, 12 tasks)"
)
# Discord receives: 204 No Content
```

**What happens:**
1. `SendMessage` POSTs JSON `{"content": "..."}` to the webhook URL
2. Discord delivers the message to the configured channel
3. Error if webhook URL invalid → `fmt.Errorf("discord: webhook returned %d")`

---

### Scene 3: Slash Command Inbound

**Narrator:** "Slash commands arrive as Discord Interactions POSTs. ParseWebhook extracts the command name, options, and user."

**Demo:**
```python
# User types in Discord: /status production
# Discord POSTs to your Interactions endpoint:
payload = {
  "type": 2,
  "id": "1234567890",
  "data": {
    "name": "status",
    "options": [{"name": "env", "value": "production"}]
  },
  "member": {"user": {"id": "987654321", "username": "alice"}},
  "channel_id": "111222333",
  "token": "interaction_token_here"
}

inbound = adapter.parse_webhook(payload)
# Returns:
InboundMessage(
    chat_id="111222333",
    user_id="987654321",
    username="alice",
    text="/status production",
    message_id="1234567890",
    metadata={"platform": "discord", "interaction_token": "..."}
)
```

**Key behaviors:**
- Type 1 (PING) → returns `nil, nil` — your handler sends `{"type":1}` to Discord
- Type 2 (slash command) / Type 3 (button) → extracts text as `/command option1 option2`
- Prefers `member.user` (guild) over `user` (DM)
- 1 MiB body cap for DoS protection

---

### Scene 4: /status Command Flow

**Narrator:** "The Community Manager agent handles /status in Discord. Users get real-time status without leaving Discord."

**Demo flow:**
```
Discord user → /status production
  → Discord POSTs to Molecule AI Interactions URL
  → ParseWebhook extracts: text="/status production", user=alice
  → Molecule AI agent processes: checks deployment status
  → SendMessage posts result to Discord webhook
  → Alice sees the response in Discord
```

**Agent-side code:**
```python
message = await channel.receive_message()
if message.text.startswith("/status"):
    env = message.text.split()[1] if len(message.text.split()) > 1 else "prod"
    status = await check_deployment_status(env)
    await channel.send_message(
        config=channel.config,
        chat_id=message.chat_id,
        text=f"**{env}**: {status.summary} | {status.uptime}% uptime | Last deploy: {status.last_deploy}"
    )
```

---

### Scene 5: Long Message Split

**Narrator:** "If the agent generates a long report, Discord adapter splits it into 2000-char chunks automatically."

**Demo:**
```python
long_text = "A" * 3500  # exceeds 2000-char Discord limit
chunks = split_message(long_text, 2000)
# Result: ["AAAA... (2000 chars)", "AAAA... (1500 chars)"]
# Each chunk posted as separate webhook call
# Discord shows them as consecutive messages (threaded if channel permits)
```

---

### Scene 6: Error Handling

**Narrator:** "Webhook errors surface clearly — but webhook tokens are never logged."

**Code point:**
```python
# Note from discord.go line 88-91:
# "Do NOT wrap err — the url.Error from http.Client.Do includes the
#  full Discord webhook token. Wrapping would propagate it into logs."
# → Returns: fmt.Errorf("discord: HTTP request failed")
```

**Known error codes:**
- `400` — malformed payload
- `404` — invalid webhook URL
- `429` — rate limited (retry after header)
- `500` — Discord server error (transient)

---

## Visual Assets Needed (from PR #1209 brief)

| Asset | Dimensions | Description |
|---|---|---|
| `discord-molecule-logo-combo.png` | 1200×800 | Discord + Molecule AI logo combo, dark #1E1E2E background, radial blurple glow, tagline |
| `discord-slack-command-mockup.png` | 1200×900 | Full Discord UI mockup with slash-command interaction, agent response with cyan status badge |
| `discord-community-signal-flow.png` | 1200×600 | Routing flow diagram: Discord → Community Manager → Security |

---

## 60-Second Screencast Outline

| Timestamp | Scene |
|---|---|
| 0:00–0:05 | Title card: "Discord Adapter — Live in Molecule AI" |
| 0:05–0:15 | Canvas channel config: add Discord with webhook URL |
| 0:15–0:30 | Simulated /status command in Discord UI → agent response |
| 0:30–0:45 | Flow diagram overlay: Discord → Community Manager → response |
| 0:45–0:55 | Key capabilities callout: slash commands, long message split, no bot token |
| 0:55–1:00 | CTA: "Try it in your workspace → docs link" |

---

## Social Copy (X/LinkedIn)

**Technical:**
> The Discord adapter is live. Your Molecule AI agents can now receive /slash commands and post status updates directly to Discord — no bot token required, just a webhook URL. Open-core, runs anywhere. → [docs link]

**Community:**
> Community Manager + Discord = live support without switching tools. Your users ask /status in Discord; your agents answer from Canvas. Built on the Molecule AI channel adapter framework.

---

*Demo prepared locally. Push pending token availability.*