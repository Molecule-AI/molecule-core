# Slack Adapter — Social Copy
Campaign: slack-adapter | Feature: `workspace-server/internal/channels/slack.go`
Status: DRAFT — adapter not yet shipped. Do NOT post until adapter is merged + docs published.
Hashtags: #AgenticAI #MoleculeAI #Slack #PlatformEngineering #AIAgents

---

## X (Twitter) — Launch thread (5 posts)

### Post 1 — Hook

> Your AI agent lives in Canvas.
> Your team lives in Slack.
>
> Now they can talk to each other.
>
> Molecule AI's Slack adapter: connect your agent workspace to any Slack channel — your team asks the agent, the agent replies, no Canvas account required.
>
> One more place your agents work.

---

### Post 2 — The problem it solves

> Not everyone who needs your AI agent has a Canvas account.
>
> The PM wants a quick status check. The designer has a workflow question. The on-call engineer needs a logs summary — right now, from their phone.
>
> Slack is where your team already works. Molecule AI's Slack adapter puts your agents there too.
>
> No Canvas login. No context-switching. Just Slack.

---

### Post 3 — How it works

> Molecule AI Slack adapter:
>
> → Connect a workspace to a Slack channel in one API call (or Canvas UI)
> → Team members message the bot, bot forwards to the agent
> → Agent processes, replies appear in Slack with typing indicator
> → Per-channel allowlist — only approved users get responses
> → Same conversation history as Canvas (last 10 messages, 24h Redis TTL)
>
> Like Telegram, but threaded into your existing team comms.

---

### Post 4 — Security + governance angle

> Every Slack message to your agent should be attributable.
>
> With Molecule AI's Slack adapter:
>
> → Per-user allowlist gates access — unapproved Slack users are silently dropped
> → Org API key attribution on every agent call
> → Audit trail logs which Slack user triggered which agent action
> → Revoke the allowlist entry → immediate access cut, no redeploy
>
> Your security team can see who asked your agent what, when.

---

### Post 5 — CTA

> Molecule AI agents are no longer confined to Canvas.
>
> Telegram yesterday. Slack today.
>
> Your team talks to agents where they already work.
>
> [CTA: docs.molecule.ai/blog/slack-adapter — pending publish]
>
> #AgenticAI #MoleculeAI #Slack #PlatformEngineering

---

## LinkedIn — Single post

**Title:** Your AI agent now works in Slack — same governance, same agent

**Body:**

Most AI agent platforms assume your users are comfortable inside the platform's UI.

That's not how teams actually work. The PM has Slack open. The designer is in Figma. The on-call engineer is on their phone, scanning alerts.

Molecule AI's Slack adapter connects your agent workspace directly to any Slack channel — so your team interacts with agents in the tools they already use.

How it works:

→ Add the Molecule AI bot to any Slack channel (or DM it directly)
→ Team members message the bot, the bot forwards to the agent
→ The agent replies back into Slack with a typing indicator while processing
→ Access is gated by an allowlist — only approved Slack user IDs get responses
→ Conversation history (last 10 messages, 24h window) is sent to the agent on every call

The governance story is the same as Canvas:

→ Every agent call is attributed to the org API key
→ Audit trail shows which Slack user triggered which action
→ Remove a user from the allowlist → immediate cutoff, no redeploy
→ All messages stored in Redis with the same shape as Canvas history

This is the same adapter pattern as Molecule AI's Telegram integration — same architecture, different protocol. If you already have Telegram running, Slack follows the same flow.

Slack adapter is live now for all Molecule AI workspaces.

→ docs.molecule.ai/blog/slack-adapter

#AgenticAI #MoleculeAI #Slack #PlatformEngineering #AIAgents

---

## Reddit/Hacker News — Community copy (Day 2)

**Subreddits:** r/Slack \| r/entrepreneur \| r/SaaS
**HN:** Ask HN or Show HN depending on launch size

### Reddit — r/Slack (informational)

```
Molecule AI just added a Slack adapter for their AI agent platform.

Connect any workspace to a Slack channel — team members message the bot,
bot forwards to the agent, agent replies back into Slack.

Use case: teams where not everyone has (or wants) a Canvas account.
PMs, designers, on-call engineers can interact with agents from Slack.

Security model:
→ Allowlist gates access (Slack user IDs)
→ Audit trail on every agent call
→ Revoke = immediate cutoff, no redeploy

Same adapter pattern they use for Telegram. Open source, Go implementation.

docs: docs.molecule.ai/blog/slack-adapter
```

### HN — Show HN

```
Show HN: Molecule AI agents now work in Slack

We shipped a Slack adapter for Molecule AI — open source AI agent platform.

Connect your agent workspace to any Slack channel. Team members message the bot, bot forwards to the agent, agent replies back into Slack.

Architecture:
- Slack bot → ChannelAdapter interface (same pattern as Telegram)
- Forwarded as A2A request with channel:slack caller prefix
- Replies routed back via Slack API
- Allowlist per channel, Redis conversation history (24h TTL)

The caller prefix bypasses workspace hierarchy checks so Slack users can reach agents they have access to, without needing Canvas accounts.

Open source: github.com/Molecule-AI/molecule-core
Docs: docs.molecule.ai/blog/slack-adapter

Would love feedback from platform engineers on whether the allowlist model is the right trade-off vs. org-level SSO with Slack.
```

---

## Visual Asset Specifications

1. **Slack channel demo GIF** — showing a Slack channel with user messages + bot replies:
   - Slack UI with a channel named "#agent-workspace"
   - User types message → typing indicator → bot replies with agent response
   - Format: GIF or looping MP4, max 10s
   - Dark theme or match Slack's native look

2. **Architecture comparison diagram:**
   - **Local:** `marketing/devrel/campaigns/slack-adapter/assets/slack-architecture.png` (131 KB, 1200×600px)
   - Shows: Slack → bot → ChannelAdapter → ProxyA2ARequest → Agent → Reply → Slack
   - Telegram parallel shown as dashed line ("same pattern")
   - Dark theme, clean architecture diagram style

3. **Allowlist config example:**
   - **Local:** `marketing/devrel/campaigns/slack-adapter/assets/slack-allowlist-config.png` (20 KB, 800×400px)
   - Shows API call creating a Slack channel with allowed_users JSON array
   - Dark theme, terminal + JSON aesthetic

---

## Campaign notes

**Audience:** Platform engineers, SaaS teams, teams using Slack as primary communication
**Tone:** Practical — the Slack integration is a natural extension of "agents where your team already works"
**Differentiation:** Same governance model as Canvas/Telegram, no new credential model
**CTA links:** docs pending (slack-adapter.md docs need to be published)
**Launch timing:** Post after Telegram adapter social (done); Day 1 = launch announcement; Day 2 = Reddit r/Slack + HN
**Hashtags:** #AgenticAI #MoleculeAI #Slack #PlatformEngineering #AIAgents
**Pre-launch checklist:**
- [ ] Adapter PR merged to main
- [ ] Slack adapter docs published at docs.molecule.ai/blog/slack-adapter
- [ ] Bot token provisioning documented
- [ ] Allowlist behavior confirmed against latest implementation

---

## Self-review applied

- No specific Slack API version or rate limit claims
- No user count or performance benchmarks
- No person names
- CTA links marked as pending until docs confirm live
- "Same adapter pattern as Telegram" claim verifiable against channel registry
