---
name: careful-mode
description: Refuse or warn before destructive irreversible commands (rm -rf, force push, DROP TABLE, gh pr close, gh issue close, mass DELETE). Inspired by gstack's /careful and /freeze. Activate at the start of any cron tick or when about to write to shared resources.
---

# careful-mode

Cron has merge authority + commit authority. That is enough rope to do permanent damage. This skill is the seatbelt.

## Activate when

- The hourly cron tick starts
- About to call `gh pr merge` / `gh pr close` / `gh issue close`
- About to push to a branch other than your own draft
- About to run `git push --force` for any reason
- About to run `rm -rf` on anything inside the repo
- About to issue `DROP TABLE` / `TRUNCATE` / `DELETE FROM ... WHERE` without a known small WHERE

## Categories

### REFUSE — hard stop

- `git push --force` to `main`, `master`, or any protected branch
- `gh pr merge` on a PR that:
  - has CI failing
  - has `state: draft`
  - has unresolved review comments from a non-bot author
  - was created in the same conversation context (need 1 tick of distance)
- `git reset --hard` against a branch that has commits I haven't seen pushed to a remote
- `rm -rf` against any path matching `**/migrations/**`, `.git/`, `~/.molecule/`, or repo root
- `DROP TABLE`, `TRUNCATE TABLE` against any table in the molecule schema
- `DELETE FROM workspaces` without a `WHERE id = $known_uuid` clause

### WARN — proceed only with explicit confirmation in the prompt

- `gh pr close` on a PR not authored by me
- `gh issue close` on any issue
- `git push --force-with-lease` (safer than `--force`, still requires care)
- `rm -rf node_modules / dist /` (safe, but worth a one-line "yes I meant this")
- `chmod -R` on anything outside the current PR's diff
- Mass curl-DELETE loops over `/workspaces` (the cleanup-rogue-workspaces.sh pattern is OK but document the prefix)

### ALLOW

- Anything against `/tmp/`, the agent's own scratch dir, or test artifacts
- Reads of any kind
- Standard merges via `gh pr merge --merge --delete-branch` once the gates pass
- Single-row updates / deletes with explicit WHERE on a known-uuid

## Freeze mode

When debugging a tricky issue, lock edits to one directory. Example invocation:

```
careful-mode freeze platform/internal/handlers/
# now any Edit/Write outside that path refuses
careful-mode unfreeze
```

This is conceptually like gstack's `/freeze` — prevents accidental scope creep when an agent is spelunking.

## How to honor this skill

The skill is enforced by the AGENT, not by the harness. When making a tool call that lands in the REFUSE / WARN list, the agent must:

1. Stop
2. State the exact command + which list it falls under
3. Explain why this case is or isn't safe
4. For WARN, ask for explicit user confirmation
5. For REFUSE, decline and propose a safer alternative

## Why this exists

The cron has merge authority. gstack documented several near-misses where Claude wiped working directories or force-pushed to main. We avoid those by making the rules explicit and machine-readable, applied at the start of every tick.
