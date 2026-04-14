# Agent operating rules — auto-loaded into every workspace

These are the discipline rules the Molecule AI orchestrator applies to
itself. The `molecule-guardrails` plugin installs them so every agent
workspace inherits the same posture.

The rules apply to every conversation, automated cron tick, and every
subagent the orchestrator spawns inside this workspace.

## Why these exist

Skills are opt-in (the agent has to remember to invoke them). Hooks are
ambient (the harness enforces them on every matching event). This rules
file documents both in one place so the agent knows what's enforced
versus what's available on demand.

## Hooks active in this workspace

The following ambient guardrails fire automatically, configured in
`.claude/settings.json`. When a hook blocks a tool call, the response
will include a `permissionDecisionReason` — read it before retrying.

| Hook | Event | Effect |
|------|-------|--------|
| `pre-bash-careful.sh` | PreToolUse:Bash | REFUSES `git push --force` to main, `rm -rf` at root/HOME, `DROP TABLE` against prod. WARNs on `--force-with-lease`, `gh pr/issue close`. |
| `pre-edit-freeze.sh` | PreToolUse:Edit/Write | Blocks edits outside the path in `.claude/freeze` if that file exists. Set scope: `echo platform/internal > .claude/freeze`. Unlock: `rm .claude/freeze`. |
| `session-start-context.sh` | SessionStart | Auto-loads recent cron-learnings, freeze status, open PR/issue counts. |
| `post-edit-audit.sh` | PostToolUse:Edit/Write | Appends every edit to `.claude/audit.jsonl` (gitignored). |
| `user-prompt-tag.sh` | UserPromptSubmit | Injects warning when prompt mentions force-push / drop-table / "delete all" / etc. |

## Skills active in this workspace

These are documented in `.claude/skills/*/SKILL.md`. Invoke explicitly
via the `Skill` tool — they are NOT auto-applied.

- `code-review` — full 16-criteria rubric on a diff
- `cross-vendor-review` — adversarial second-model review for noteworthy PRs
- `careful-mode` — the doc backing the bash hook above
- `cron-learnings` — defines the JSONL operational-memory format
- `llm-judge` — score whether a deliverable addresses the request
- `update-docs` — sync repo docs after merges

## Slash commands

- `/triage` — full PR-triage cycle (gates 1-7 + code-review + merge)
- `/retro` — weekly retrospective (PRs, issues, gate failures, trends)

## Standing rules (inviolable)

- Never push directly to main — use feat/fix/chore/docs branches
- Merge-commits only (`gh pr merge --merge`) — never `--squash` / `--rebase`
- Dark theme only (no white/light CSS classes)
- No native browser dialogs (`confirm` / `alert` / `prompt`) — use a `ConfirmDialog` component
- Delegate through PM, never bypass hierarchy
- Only PM mounts the repo; other agents get isolated volumes

## Operational discipline

1. **Read recent learnings before reviewing PRs.** Open the JSONL referenced by the `cron-learnings` skill, read the last 20 lines. Patterns recur.
2. **Treat docs PRs that touch CLAUDE.md/PLAN.md as ALWAYS noteworthy.** Those are the agent-facing source of truth. Run `code-review` skill at minimum.
3. **After any cron tick, write a 1-line reflection** to `.claude/per-tick-reflections.md` (gitignored).
4. **Use `/freeze` when debugging** to prevent scope creep across files unrelated to the bug.
