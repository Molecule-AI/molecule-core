---
name: cron-learnings
description: At the end of every cron tick, append 1-3 lines of operational learnings (what worked, what surprised, what should change next tick) to a per-project JSONL. Replay at start of next tick. Inspired by gstack's /learn skill.
---

# cron-learnings

Each tick, the cron does a lot of work. Half the lessons are forgotten by the next tick. This skill is the compounding layer.

## Storage

Per-project file at:
```
~/.claude/projects/<sanitized-project-path>/memory/cron-learnings.jsonl
```

For molecule-monorepo, that's:
```
~/.claude/projects/-Users-hongming-Documents-GitHub-molecule-monorepo/memory/cron-learnings.jsonl
```

One JSON object per line:
```json
{"ts": "2026-04-14T05:17:00Z", "tick_id": "5939aa3f-001", "category": "gate-fail", "summary": "Gate 4 (security) flagged token!=secret in PR #28; requireInternalAPISecret needs subtle.ConstantTimeCompare", "next_action": "When reviewing auth-gate code, grep for `subtle.ConstantTimeCompare`. Flag plain == on tokens."}
```

Categories:
- `gate-fail` — a verification gate caught something
- `mechanical-fix` — fixed a gate failure on-branch
- `false-positive` — a code-review finding turned out to be wrong; record so we don't keep flagging it
- `tool-error` — an MCP tool / CLI flaked; note the workaround
- `repo-state` — something about the repo's state that next tick should know
- `pattern` — a cross-PR pattern worth remembering (e.g., "every cron loop adds itself as `noreply@anthropic.com`; reviewers OK with it")

## When to write

End of every cron tick (Step 5 of the cron prompt). 1-3 lines max — be terse.

## When to read

Start of every cron tick. Read the last 20 lines (most recent first) before Step 1. Use them to:
- Skip false-positive paths the previous tick flagged
- Apply learned patterns (e.g., "PR #28 found INTERNAL_API_SECRET missing from .env.example — when reviewing future security PRs, always check .env.example sync as a first move")
- Avoid re-litigating decided design choices

## Pruning

Cap at 500 lines. When exceeded, the next write also drops the oldest 100 lines. The point is recent operational memory, not an audit log.

## Format discipline

- One line per event
- ASCII-only for grep-friendliness
- No PII, no tokens, no URLs with auth
- `summary` is what HAPPENED; `next_action` is what FUTURE-YOU should DO
- If you can't think of a concrete next_action, it's not worth logging

## Why this exists

gstack's `/learn` showed that AI sessions repeatedly make the same mistakes because the lessons live only in the conversation that produced them. Writing them to disk lets every tick start with the accumulated wisdom of every prior tick, at zero cost. The awareness MCP we have is fine for cross-session human/agent memory — this file is specifically for the cron's own automation.
