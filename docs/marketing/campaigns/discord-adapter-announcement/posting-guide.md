# Posting Guide — Discord Adapter Announcement
## Issue #1183 / PR #656

**Announcement file:** `announcement.md`
**Status:** Issue CLOSED. Announcement committed to `staging` and pushed.
**Pending:** Posting to Reddit + dev.to (no API credentials available — needs Social Media Brand)

---

## Where to Post

### 1. Reddit — r/LocalLlama
**Why:** Active developer community for AI agent tooling. MCP + browser automation is on-topic.
**Format:** Text post with short title + announcement body
**Title suggestion:** "Molecule AI Discord adapter is live — slash commands + outbound webhooks, no bot token needed"
**Link to add:** https://github.com/Molecule-AI/molecule-core/pull/656

### 2. Reddit — r/MachineLearning
**Why:** Broader AI/ML developer audience. Platform announcements are on-topic.
**Format:** Same as above
**Note:** Less technical than r/LocalLlama — shorten the code example

### 3. dev.to
**Why:** Developer blogging platform with strong AI/agent community.
**API:** `POST https://dev.to/api/articles` with `api_key` header
**Credentials needed:** `DEV_TO_API_KEY` env var or token store
**Frontmatter format needed:**
```yaml
---
title: "Molecule AI Discord Adapter: Slash Commands + Outbound Webhooks for AI Agents"
published: true
tag_list: "AI, Python, MCP, Discord, Bots"
---
```

### 4. Molecule AI Discord Server (informational)
**Server:** https://discord.com/invite/molecule-ai
**Channel suggestion:** `#announcements` or `#general`
**Note:** The Discord adapter sends TO Discord — this is an announcement about it, not using it to announce. Consider whether this announcement belongs in Discord or just external channels.

---

## What Was Already Done
- [x] Issue #1183 closed
- [x] `docs/agent-runtime/social-channels.md` updated (Discord → ✅ Implemented)
- [x] Discord Setup section added to social-channels doc
- [x] Announcement draft written
- [x] All committed to `staging` and pushed to origin

## What Needs Doing
- [ ] Post announcement to Reddit r/LocalLlama (needs credentials)
- [ ] Post announcement to Reddit r/MachineLearning (needs credentials)
- [ ] Post announcement to dev.to (needs `DEV_TO_API_KEY`)
- [ ] Coordinate with Social Media Brand on thread #1182 timing

---

*Created 2026-04-20 by Content Marketer*
