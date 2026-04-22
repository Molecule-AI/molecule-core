# Discord Adapter Launch — Social Copy
Campaign: discord-adapter-launch | PR: molecule-core#1209
Publish day: TBD — coordinate with Marketing Lead
Assets: visual assets at marketing/devrel/campaigns/discord-adapter-launch/assets/

---

## X (Twitter) — Primary thread (5 posts)

### Post 1 — Hook
Your team is already in Discord.

Your AI agent is in Molecule AI.

Why are you switching between two tools to talk to your own infrastructure?

Discord adapter for Molecule AI: connect any agent workspace to a Discord channel.
Slash commands in. Agent responses out.

---

### Post 2 — Setup simplicity
Most Discord bot integrations require:
→ Create a bot in the Developer Portal
→ Set up OAuth2
→ Handle the Gateway
→ Manage intents and permissions

Molecule AI's Discord adapter requires:
→ One webhook URL

That's it. The webhook encodes the channel and bot credentials. You paste it in Canvas. You're done.

---

### Post 3 — How it works (technical)
The Discord adapter uses two standard Discord features:

→ Incoming Webhooks for outbound messages (agent → Discord)
→ Discord Interactions for inbound slash commands (Discord → agent)

No polling. No Gateway. No message-reading permissions.

Users type `/ask what's our deployment status?` — the adapter reconstructs that as plain text, the agent responds, the response goes back to the channel.

---

### Post 4 — Hierarchy use case
In Molecule AI, a Community Manager agent receives the slash command, delegates to the right sub-agent, and returns the answer to Discord.

The routing is invisible to the Discord user.

Discord → Community Manager → (Security Auditor | QA Engineer | PM) → Discord

Your whole agent team, accessible from a Discord server your team already lives in.

---

### Post 5 — CTA
Discord adapter for Molecule AI is live.

If your team runs standups, triage, and deployments in Discord — your AI agents can be in the same room.

Connect a workspace in two minutes. Start with a slash command.

→ https://docs.molecule.ai/blog/discord-adapter

---

## LinkedIn — Single post

**Title:** We put our AI agents in Discord — here's why that's a bigger deal than it sounds

**Body:**

Every AI agent platform eventually gets asked the same question: "can we talk to it from where our team already communicates?"

For a lot of teams, that place is Discord. Not as a notification sink — as a working interface.

We just shipped a Discord adapter for Molecule AI. Here's what made it interesting to build:

The naive approach is a Discord bot with message reading permissions, OAuth flows, Gateway connections, and rate limit handling. That's a lot of surface area, and it requires permissions that workspace policies often don't grant.

The Molecule AI approach is two standard Discord primitives:

→ Incoming Webhooks for outbound messages. You give us a webhook URL. That's the only credential. It encodes the channel and bot credentials. You paste it in Canvas. Done.

→ Discord Interactions for inbound slash commands. Users type `/ask what's our deployment status?`. We parse the command and options from the signed JSON payload. The agent receives it as plain text. The response goes back to the channel.

No polling. No Gateway. No special permissions.

What this unlocks: your whole agent hierarchy, accessible from a Discord server your team already lives in. A Community Manager agent receives the slash command, routes to the right sub-agent (Security Auditor, QA, PM), and returns the answer. The routing is invisible to the Discord user.

If your team runs standups, incident triage, or deployment coordination in Discord — your AI agents are now in the same room.

Discord adapter is live now. Connect a workspace in the Channels tab.

---

## Campaign notes

**Audience:** DevOps, platform engineers, developer teams already in Discord
**Tone:** Practical, technical credibility. Not hype — the simplicity of the webhook setup is the story.
**Differentiation:** Zero-boilerplate Discord integration vs. traditional bot setup complexity
**Use case pairing:** X → slash commands as the interface (developer-friendly), LinkedIn → team workflow integration (manager/lead audience)
**Hashtags:** #Discord #AIAgents #AgenticAI #MoleculeAI #PlatformEngineering
**Assets:** visual assets at `docs/marketing/campaigns/discord-adapter-launch/assets/`:
  - discord-adapter-hero.png (1200x630)
  - discord-adapter-social-card.png (1080x1080)
**Coordination:** Publish after blog post is live. Coordinate with Social Media Brand queue.
