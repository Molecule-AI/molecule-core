# Loop discipline — process notes

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
