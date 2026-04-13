---
name: "Approve" via chat means run the relevant cron now, not wait for next schedule
description: When the user replies "approve" to a cron prompt in this conversation (vs. via Telegram to the bot), spawn the cron subprocess immediately instead of just flipping JSON state and waiting up to 6h
type: feedback
---

When the user is talking to me directly (via Telegram or terminal) and says "approve" to a pending cron prompt, they expect immediate action — not "I flipped the status to approved, the next 6h cron run will pick it up."

**Why:** On 2026-04-07 the user approved `post_20260406_210000` and I just flipped pending-posts.json status to `approved`. They corrected me: "so when I say approve, just run it in sub proccess not another 6 hour wait time." The cron's approval flow is designed for *passive* approval (the cron itself reads pending state on the next run), but when the user is *actively* engaging with me, the wait is friction.

**How to apply:**
1. Flip the pending state to `approved` (or whatever the cron expects) so the cron's idempotency still works.
2. **Then immediately spawn the relevant cron as a background subprocess** with `POSTER_MODE=publish_only` (or the equivalent override env var for that cron). The pattern:
   ```bash
   POSTER_MODE=publish_only nohup /Users/renostars/.local/bin/claude --print --dangerously-skip-permissions \
     --add-dir '/Users/renostars/.claude' \
     --add-dir '/Users/renostars/.openclaw/workspace' \
     --add-dir '/Users/renostars/reno-star-business-intelligent' \
     -p "$(cat '/Users/renostars/reno-star-business-intelligent/prompts/<job>.md')

[OVERRIDE: PUBLISH_ONLY]" \
     >> ~/reno-star-business-intelligent/data/cron-logs/<job>.stdout.log \
     2>> ~/reno-star-business-intelligent/data/cron-logs/<job>.stderr.log &
   ```
   The override is passed two ways (env var + inline marker) so the model honors it reliably. Use `run_in_background: true` on the Bash tool so this session stays free.
3. Tell the user the subprocess pid and where the log is, so they can tail it if they want.
4. The subprocess will run independently and ping them via Telegram when done — same as a normal cron run.

**Why publish_only mode matters:** Without it, the cron does its full Phase-0-trend-research → Phase-1-publish → Phase-2-draft-new → Phase-3-ping-for-approval flow. So a manual approval triggers another draft + another approval prompt — infinite loop. The override skips Phase 0 and Phase 2 so the worker only does the publish step. This is documented in `~/reno-star-business-intelligent/prompts/social-media-poster.md` under "Mode Override: PUBLISH_ONLY". Add the same override to other approval-style cron prompts as needed.

**Applies to all approval-style prompts** from social-media-poster, social-media-engage, social-media-monitor, seo-builder, seo-weekly-report, etc. Not just the social media poster. Each prompt needs its own override block; check the prompt before assuming `POSTER_MODE=publish_only` exists for that cron.

**Caveat:** If the cron is already running (check `pgrep -f '<prompt-filename>.md'`), don't spawn a duplicate — the running instance is mid-draft and you should let it finish, then ask the user how to handle the duplicate.
