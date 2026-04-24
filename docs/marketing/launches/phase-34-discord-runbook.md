# Phase 34 — Discord Posting Runbook
**Campaign:** Phase 34 GA (April 30, 2026)
**Owner:** Community Manager
**Use:** Step-by-step guide for the Apr 30 launch day Discord post + monitoring cadence
**Date:** 2026-04-23

---

## Launch day overview

| Time | Action |
|------|--------|
| -15 min | Pre-launch checklist (below) |
| 09:00 UTC | Post announcement in `#announcements` |
| 09:00 UTC | Pin FAQ in `#faq` |
| 09:00–11:00 UTC | Monitor `#general` + `#feedback` (2h active watch) |
| Day 2 | Post Reddit r/MachineLearning + HN Show HN |

---

## Pre-launch checklist (15 min before — ~08:45 UTC)

Complete before posting anything:

- [ ] **Blog confirmed live.** Check `docs.moleculesai.app/blog/ai-agent-observability-without-overhead` is accessible. All four Phase 34 blog posts must be live before the announcement goes out.
- [ ] **Announcement file verified.** Confirm `docs/marketing/launches/phase-34-community-announcement.md` is on `main` or `staging` and the CTA links are accurate. No broken links.
- [ ] **FAQ file verified.** Confirm `docs/marketing/launches/phase-34-community-faq.md` is on branch and all Q&As are accurate. Links to docs URLs.
- [ ] **X credentials status checked.** Issue #1865 — if mol-ops has provided `X_API_KEY` + `X_API_SECRET`, note this in the announcement post. If not, do not post external social (Reddit/HN) until Day 2 when credentials are confirmed.
- [ ] **DevRel on standby.** Confirm DevRel is monitoring `#devrel` or their preferred channel for how-to routing. Post in their channel: "Phase 34 launching 09:00 UTC — FYI."
- [ ] **Bug-report channel monitored.** Confirm `#bug-reports` is being watched by the platform team.
- [ ] **Announcement CTA checked.** Confirm the GitHub Discussions link in the announcement is correct (`github.com/Molecule-AI/molecule-core/discussions`).
- [ ] **Escalation path clear.** If something goes wrong (wrong links, wrong claims, toxic thread), you know how to reach the platform team fast.

---

## Step 1 — Post in `#announcements` (09:00 UTC)

Copy the content from `docs/marketing/launches/phase-34-community-announcement.md`. Post as a single message block (or a few messages for readability — Discord has a 2000-char message limit).

**Before posting:**
- Confirm all doc links resolve correctly (run a quick curl to each staging URL)
- Confirm Partner API Keys is framed as "GA April 30" — do not say "available now"
- Confirm no design partner names are in the copy

**Post format:** Plain text with emoji formatting (Discord-native). Use the content as-is from the announcement file.

**After posting:** Drop a thread under the announcement in `#announcements` with the FAQ link:
```
📋 FAQ — answers to the top community questions:
docs.moleculesai.app/blog/phase-34-faq  ← (link to pinned message in #faq)
```

---

## Step 2 — Pin FAQ in `#faq`

1. Go to `#faq` channel
2. Paste the full content from `docs/marketing/launches/phase-34-community-faq.md` as a message
3. Pin the message (right-click → Pin Message)
4. Confirm the pin is visible

The FAQ should be pinned before the announcement goes out so people can find it immediately. If the channel already has old pins, clear stale ones and pin the Phase 34 FAQ.

**Day 2 update:** After the announcement settles, consider pinning the announcement itself in `#announcements` so it stays at the top of the channel.

---

## Step 3 — Monitor `#general` and `#feedback` (09:00–11:00 UTC)

**SLA: respond within 30 minutes of any Phase 34 reply.**

Set a 30-min reminder to check the channels. Assign yourself to the channels if your workspace supports channel monitoring.

**Response template for questions you can answer:**
```
Hey — good question. [1-2 sentence answer]. The full details are in our docs: [link]. Let me know if that doesn't answer it!
```

**Response template for questions you can't answer:**
```
Great question — let me check with the platform team and get back to you. [Tag DevRel or PM as appropriate.]
```

---

## Step 4 — Triage inbound (ongoing, first 2h critical)

Route all incoming replies to the right place:

| Type | Route to | How |
|------|----------|-----|
| How-to / setup questions | DevRel | Tag `@devrel` in the channel or DM them |
| Bugs / something broke | `#bug-reports` | Move thread or copy the report to `#bug-reports` with a link back |
| Feature requests | PM | Note in a thread, DM to PM with summary |
| Security / vulnerability | Security team | Do not post publicly. DM Security directly, do not route through channel |
| Press / media inquiries | Marketing Lead | Do not engage publicly. Escalate immediately |

**For the first 2h:** Stay on top of replies. A 30-min response gap is noticeable. After the initial surge, check every 60 min.

---

## Step 5 — Day 2: Reddit + HN (April 30, ~09:00 PT / 16:00 UTC)

**Reddit:**
1. Post at `r/MachineLearning` using `docs/marketing/community/phase34-reddit-post.md`
2. Title: "Built agent execution tracing into the platform — no SDK, no sidecar, no sampling"
3. Monitor for 2h, reply to top-level comments within 30 min
4. Do not mention specific design partners

**HackerNews:**
1. Post using `docs/marketing/community/phase34-hn-post.md`
2. Use "Show HN:" prefix in title
3. Post as text link (link to docs/blog, not a product page)
4. First-reply comment (pinned): short technical context — post a code snippet from the tool_trace example
5. Monitor for 3h, reply to every top-level comment

**Before either post:** Confirm X credentials are available if Social Media Brand is posting from a workspace. If still blocked, coordinate with Marketing Lead on whether to post without X or wait.

---

## Escalation — what to do if something goes wrong

**Wrong links in announcement:**
- Edit the announcement message in `#announcements` immediately
- Post a correction thread

**Toxic or spam thread:**
- Do not engage
- Tag Marketing Lead immediately with the thread link

**Security issue reported publicly:**
- Do not confirm or deny in channel
- DM Security team immediately, do not route through channel

**Platform outage on launch day:**
- Pause all community posts
- Post an update in `#announcements` when there's something to share
- Coordinate with Marketing Lead before posting anything about the outage

---

## Files reference

| File | Location |
|------|----------|
| Community announcement | `docs/marketing/launches/phase-34-community-announcement.md` |
| FAQ | `docs/marketing/launches/phase-34-community-faq.md` |
| Reddit post | `docs/marketing/community/phase34-reddit-post.md` |
| HN post | `docs/marketing/community/phase34-hn-post.md` |
| EC2 Instance Connect social | `docs/marketing/social/ec2-instance-connect-ssh-social-copy.md` |
| Phase 34 positioning | `docs/marketing/briefs/phase34-positioning.md` |

**Last updated:** 2026-04-23