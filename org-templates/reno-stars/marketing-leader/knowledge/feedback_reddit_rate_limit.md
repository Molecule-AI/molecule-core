---
name: Reddit comment rate limit — pace posts
description: Reddit rate-limits comments after ~4-5 rapid posts; space them out to avoid 9+ minute cooldowns
type: feedback
---

Reddit's web UI rate-limits comments aggressively. Posting 4 comments back-to-back from the Reno Stars account on 2026-04-07 triggered "Rate limit exceeded. Please wait 564 seconds and try again" (~9.4 minutes).

**Why:** Reddit treats burst commenting as spam-like behavior. The cooldown is long enough to derail a "publish all approved replies" run.

**How to apply:**
- When publishing multiple Reddit replies in one session, space them 60–90 seconds apart from the start (use a sleep between posts), not back-to-back.
- If the rate limit hits anyway, wait the full duration Reddit reports (don't retry early — it'll extend the cooldown).
- For social-media-engage cron runs that publish many approved replies: pace the publishing phase, not just the searching phase.
- Consider updating `prompts/social-media-engage.md` Phase 1 to add a "wait 60s between Reddit comments" instruction.
