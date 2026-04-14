# Loop discipline — process notes

## 2026-04-14 — gstack-inspired cron upgrades

Five new skills added under `.claude/skills/` (inspired by garrytan/gstack):

- **`cross-vendor-review`** — second-model adversarial review for noteworthy PRs (auth, billing, data deletion, migrations). Catches the 15–30% of bugs single-model review misses.
- **`careful-mode`** — REFUSE/WARN/ALLOW lists for destructive commands. Active at the start of every cron tick. Refuses force-push to main, blocks merging draft PRs, prevents `rm -rf` outside scratch dirs.
- **`cron-learnings`** — per-project JSONL of operational learnings. End each cron tick by appending 1–3 lines; start the next tick by replaying the last 20.
- **`cron-retro`** — weekly retrospective auto-posted as a GitHub issue. Sunday 23:07 local. Tracks PR count, time-to-merge, gate failures, code-review severity trends.
- **`llm-judge`** — cheap LLM-as-judge eval to catch "agent shipped the wrong thing" — the failure mode unit tests miss.

Two crons govern this:
- **Hourly triage** (`:17` past each hour) — Step 0 activates careful-mode + replays cron-learnings; Step 2 supplements run code-review and (for noteworthy PRs) cross-vendor-review; Step 4 issue-pickup runs llm-judge before marking ready; Step 5 appends cron-learnings.
- **Weekly retro** (Sunday `23:07`) — invokes cron-retro skill, posts a GitHub issue.

Both crons are session-only per the runtime; re-invoke in a new session if needed.

## Rule: a "skipped" PR must have a comment explaining the skip

When the hourly maintenance loop skips a PR for any reason — CI red,
conflicting, merge dirty, missing tests, design drift — the FIRST skip
in a session must leave a PR comment with the specific blocker and the
exact fix the author needs to apply. Subsequent skips of the same PR
(SHA unchanged) can be silent.

The failure mode this rule prevents: silently skipping a PR for many
hours under a vague reason ("blocked / no CI / conflicting") without
ever telling the author what they need to do. The PR sits indefinitely
because the author has no comment to act on.

Concrete check at the top of each loop:
- For every "known-blocked" PR I'm about to silently skip, verify there
  is a bot/me comment on the PR newer than the PR's head SHA that names
  the specific blocker. If not, that PR isn't actually blocked on the
  author — it's blocked on me writing the comment.

Caught 2026-04-13 on PR #114 (skipped 6+ loops with no comment).
