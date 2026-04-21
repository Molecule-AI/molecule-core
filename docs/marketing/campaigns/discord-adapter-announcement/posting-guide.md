# Posting Guide — Discord Adapter Announcement (Day 2 Campaign)
## Issue #1183 | PR #656 merged | Day 2 community push

**Status:** Blog live on `main` (slug: `discord-adapter-launch`). Reddit/HN Day 2 copy in `announcement.md`. Hero image ready.

---

## Copy Sources

- **Reddit / HN copy:** `announcement.md` → sections "Reddit / HN — Day 2 Campaign"
- **Hero image:** `marketing/devrel/campaigns/discord-adapter-launch/assets/discord-adapter-hero.png`
- **Social copy:** `social-copy.md`
- **Dev.to post body:** see section 3 below

---

## 1. Reddit — r/LocalLlama

**Why:** Active developer community for AI agent tooling. MCP + agent-channel integrations are on-topic.
**Platform:** Reddit
**Credentials:** `REDDIT_CLIENT_ID` + `REDDIT_CLIENT_SECRET` (Social Media Brand)
**When:** 12pm PT on publish day (same day as HN)

**Title:**
> Molecule AI Discord adapter: connect any AI agent workspace to Discord with one webhook URL

**Body:** Use "Reddit / HN — Day 2 Campaign / r/LocalLLaMA — Body" section from `announcement.md`.
Link: `[BLOG_URL]` → fill with live blog URL before posting. Fallback: `https://github.com/Molecule-AI/molecule-core/pull/656`

---

## 2. Reddit — r/MachineLearning

**Why:** Broader AI/ML developer audience.
**Platform:** Reddit
**Credentials:** Same as above
**When:** 1pm PT (30 min after r/LocalLlama)

**Title:**
> Molecule AI Discord adapter: one webhook, full agent interaction in Discord

**Note:** Trim the architecture paragraph. Lead with "what it does" before "how it works."
Use the r/LocalLlama body from `announcement.md` as source, trim to ~200 words.

---

## 3. Hacker News

**Why:** Technical early-adopters, developer tooling audience.
**Platform:** https://news.ycombinator.com/submit
**Credentials:** Hacker News account (team member submits manually)
**When:** 11am UTC on publish day

**Title:**
> Show HN — Molecule AI Discord adapter: one webhook, full agent interaction in Discord

**Body:** Use "Reddit / HN — Day 2 Campaign / Hacker News — Body" section from `announcement.md`.
Link: `[BLOG_URL]` → same as above.

HN-specific rules:
- 2–3 paragraphs, no fluff
- Be specific ("A2A protocol", "workspace auth tokens" signal technical depth)
- Don't hard-sell
- Close with "(I'm [NAME] from the Molecule AI team — AMA)"
- Upvote your own post once after submitting

---

## 4. dev.to

**Why:** Developer blogging platform, strong AI/agent audience.
**API:** `POST https://dev.to/api/articles` with `DEV_TO_API_KEY`
**Credentials:** `DEV_TO_API_KEY` (Social Media Brand)

**Frontmatter:**
```yaml
---
title: "Molecule AI Discord Adapter: Slash Commands + Outbound Webhooks for AI Agents"
published: true
tag_list: "AI, Python, MCP, Discord, Bots, AgenticAI"
---
```

**Body:**

Molecule AI workspaces can now connect to Discord.

Here's what makes this different from a typical bot integration:

Traditional Discord bot setup requires: Developer Portal app, OAuth2, Gateway connection, intent configuration, message-reading permissions, rate limit handling.

The Molecule AI Discord adapter requires: **one webhook URL.**

That's the only credential. It encodes the channel and bot tokens. You paste it in the Canvas Channels tab. Done.

What you get:
- Slash commands (`/ask`, `/status`, `/help`) route directly to your workspace agent
- Agent responses post back to the Discord channel automatically
- 2,000-character chunking handled without code
- Works in servers and in DMs
- Webhook tokens are never logged (security fix in PR #659)

This is the same adapter interface that handles Telegram. New channels add one implementation, and the full CRUD API, Canvas UI, and MCP tools work automatically.

**Setup:** Canvas → Workspace → Channels tab → + Connect → Discord → paste your webhook URL.

Docs → [Social Channels guide](https://github.com/Molecule-AI/molecule-core/blob/main/docs/agent-runtime/social-channels.md)

GitHub → [PR #656 — Discord adapter](https://github.com/Molecule-AI/molecule-core/pull/656)

---

## 5. Molecule AI Discord Server (#announcements)

**Server:** https://discord.com/invite/molecule-ai
**Channel:** `#announcements`
**Credentials:** Discord account with post permissions

**Copy:**

> **Molecule AI Discord Adapter is live! 🎉**
>
> Your workspace can now connect to Discord — send messages to channels and receive slash commands from users.
>
> **What you can do:**
> → Send notifications, summaries, or AI-generated responses to any Discord channel
> → Users interact with your agent via slash commands (e.g. `/ask <question>`)
> → Works in servers and DMs — no separate bot token needed for outbound
>
> **How to connect:**
> 1. Create a Discord webhook (Channel → Integrations → Webhooks)
> 2. Add it to your workspace: Channels tab → + Connect → Discord
> 3. Done
>
> For slash commands inbound, point your Discord app's Interactions URL at `POST /webhooks/discord` on your platform.
>
> Docs: [Social Channels guide](https://github.com/Molecule-AI/molecule-core/blob/main/docs/agent-runtime/social-channels.md)

---

## Coordination Checklist

Before posting Day 2:
- [ ] Fill `[BLOG_URL]` placeholder in announcement.md Reddit/HN copy → live blog URL
- [ ] Confirm Discord adapter blog post is on `main` at `docs/blog/2026-04-21-discord-adapter/`
- [ ] Coordinate Reddit/HN timing: HN first (11am UTC), r/LocalLlama (12pm PT), r/MachineLearning (1pm PT)
- [ ] Social Media Brand posts Reddit/HN — owns timing + credentials
- [ ] DevRel posts dev.to — needs `DEV_TO_API_KEY`
- [ ] Community posts in Molecule AI Discord #announcements

---

## What Was Already Done

- [x] Blog post live on `main` (slug: `discord-adapter-launch`)
- [x] Reddit r/LocalLlama + r/MachineLearning copy drafted (`announcement.md`)
- [x] Hacker News post body drafted (`announcement.md`)
- [x] dev.to post body drafted (this file, section 4)
- [x] Hero image ready (`discord-adapter-hero.png`, 1200×630)
- [x] All committed to `staging` and pushed

---

*Updated 2026-04-21 by Content Marketer — Day 2 campaign prep*
