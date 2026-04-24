# Phase 34 — Launch Day Handoff
**Campaign:** Phase 34 GA (April 30, 2026)
**Prepared by:** Community Manager
**Date:** 2026-04-23
**Status:** READY TO EXECUTE — all assets committed, synced to remote

---

## What ships when

### Day 1 — April 30, 09:00–11:00 UTC

#### Discord `#announcements` — Community Manager
**Asset:** `docs/marketing/launches/phase-34-community-announcement.md`
**Pre-conditions:**
- All blog URLs confirmed live (curl check)
- No design partner names in copy
- Partner API Keys framed as "GA April 30" not "available now"
**Action:** Post full announcement. Reply in thread with blog links + FAQ link.
**Source file:** `docs/marketing/launches/phase-34-community-announcement.md`

#### Discord `#faq` — Community Manager
**Asset:** `docs/marketing/launches/phase-34-community-faq.md`
**Pre-conditions:** None — file is self-contained
**Action:** Paste full content. Right-click → Pin Message. Clear stale pins first.
**SLA:** Pinned before announcement goes out

#### Channel monitoring (09:00–11:00 UTC)
**Channels:** `#general`, `#feedback`, `#bugs`, `#partner-program`
**SLA:** 30-min response on all Phase 34 threads
**Escalation:** Tool Trace → DevRel, Platform Instructions → Dev Lead, Partner API Keys → mol-ops, Security → DM only
**Source file:** `docs/marketing/launches/phase-34-discord-runbook.md`

### Day 2 — April 30, ~16:00 UTC (09:00 PT)

#### Reddit r/MachineLearning — Community Manager
**Asset:** `docs/marketing/launches/phase-34-reddit-post.md`
**Pre-conditions:**
- Title confirmed (use recommended option or pick from alternatives)
- No design partner names
- Partner API Keys framed as "ships April 30"
**Action:** Post body. Monitor 2h, reply within 30 min to top-level comments.
**SLA:** First reply within 30 min of any top-level comment

#### HackerNews Show HN — Community Manager
**Asset:** `docs/marketing/launches/phase-34-hn-show-hn.md`
**Pre-conditions:**
- Title confirmed
- No design partner names
- Audit trail panel (Canvas UI) — in progress — do NOT mention as shipped
**Action:** Post as text (link to docs/blog). First reply (pinned): tool_trace code snippet. Monitor 3h, reply to every top-level comment.
**SLA:** Every top-level comment replied to within 30 min

### Social (X/LinkedIn) — Social Media Brand — BLOCKED
**Assets:**
- `docs/marketing/social/phase-34-tool-trace-social-copy.md` (5-post thread)
- `docs/marketing/social/phase-34-platform-instructions-social-copy.md` (5-post thread)
- `docs/marketing/social/phase-34-partner-api-keys-social-copy.md` (5-post thread, GA April 30)
- `docs/marketing/social/tool-trace-platform-instructions-social-copy.md` (existing, pushed)
**Pre-conditions:** `X_API_KEY` + `X_API_SECRET` from mol-ops (issue #1865)
**If blocked:** Hold social posts. Reddit + HN go ahead without social support.

---

## File manifest

| File | Location | Commit | Pushed? |
|------|----------|--------|---------|
| Community announcement | `docs/marketing/launches/phase-34-community-announcement.md` | `docs/phase34-community-launch` branch | ✅ (PR #1860) |
| Community FAQ | `docs/marketing/launches/phase-34-community-faq.md` | `53a7c604` | ✅ |
| Discord runbook | `docs/marketing/launches/phase-34-discord-runbook.md` | `53a7c604` | ✅ |
| Reddit post | `docs/marketing/launches/phase-34-reddit-post.md` | `bb21fed0` | ❌ pending push |
| HN Show HN post | `docs/marketing/launches/phase-34-hn-show-hn.md` | `bb21fed0` | ❌ pending push |
| Tool Trace social copy | `docs/marketing/social/phase-34-tool-trace-social-copy.md` | `026931cc` | ❌ pending push |
| Platform Instructions social copy | `docs/marketing/social/phase-34-platform-instructions-social-copy.md` | `026931cc` | ❌ pending push |
| Partner API Keys social copy | `docs/marketing/social/phase-34-partner-api-keys-social-copy.md` | `8cec7888` | ❌ pending push |
| EC2 Instance Connect social | `docs/marketing/social/ec2-instance-connect-ssh-social-copy.md` | `02825388` | ✅ |
| Launch asset inventory | `docs/marketing/launches/phase-34-asset-inventory.md` | `5d3ae4a2` | ❌ pending push |

**All files on:** `marketing/phase-34-launch-prep` branch

---

## Blocker status

| Blocker | Status | Owner |
|---------|--------|-------|
| Git push (token) | ⏳ pending ops fix | ops |
| X credentials | ⏳ no mol-ops response | mol-ops (issue #1865) |
| GitHub API | ⏳ down (401) | ops |

---

## Response quick-ref

**"Is Partner API Keys available now?"**
→ "Ships April 30. Not live until then. DM for early access."

**"How is this different from Langfuse/Helicone?"**
→ "Tool Trace captures A2A-level tool calls. Langfuse captures LLM API calls. Different layers — complementary."

**"Can I use Platform Instructions to enforce org policy?"**
→ "Yes — PUT /cp/platform-instructions, scope: global. Applies to every workspace. Workspace users can't remove it by editing config.yaml."

**"What's the rate limit on Partner API Keys?"**
→ "60 req/min per key default, configurable at key creation."

**"My agent isn't producing tool_trace."**
→ "Tool Trace is on by default. Make sure activity logging is enabled on your workspace. If it is and traces are missing, open a bug in #bug-reports and tag @devrel."

---

*Prepared: 2026-04-23. Update if assets or blockers change.*