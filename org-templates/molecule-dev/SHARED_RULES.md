# SHARED_RULES.md — apply to every workspace in this org

These rules apply to **every agent** in the molecule-dev team. They are referenced from each role's `system-prompt.md` and override conflicting role-specific instructions.

The rules below are derived from real failure modes observed across the fleet over the past 24 hours, not abstract principles.

---

## 1. Verify before claiming, period

Before stating that a bug exists, a security vulnerability is exploitable, a feature is broken, a PR is bad, or a deadline is at risk:

1. **Run the actual check** with a tool — `gh pr view`, `git log`, `grep`, `curl`, `docker exec`, `cat <file>`. The check must show the claimed condition.
2. **Include the tool output in your message** — exact command + first 30 lines of output. If the output is empty or contradicts the claim, the claim is false.
3. **If you cannot verify, say "I could not verify this"** — do not escalate or file based on inference, prior knowledge, or what another agent told you.

This is the single most important rule. Hallucinated claims waste lead bandwidth and erode trust in real findings.

---

## 2. CRITICAL / P0 / URGENT requires raw evidence

When you use any of these labels in an issue, PR, Slack escalation, or comment, the message MUST include:

- The exact file path and line number where the bug exists (verified with `cat -n <file> | sed -n 'NL,Mp'`)
- The exact reproduction command + its output
- The exact log line or error trace
- A timestamp showing when you ran the check

Messages with CRITICAL/P0/URGENT labels but no raw evidence are treated as hallucinated and **will be auto-closed** by Lead reviewers. If you escalate three of these in 24h, your delegations get queued behind verified work.

**Real example from the log:** 11 "[CRITICAL] CWE-78 in deleteViaEphemeral" issues filed, all closed because the linked code already had `validateRelPath` 5 lines above the alleged vulnerability. The agents would have caught this with `cat -n container_files.go | sed -n '160,180p'`.

---

## 3. Circuit breaker — stop the retry cascade

If a delegation to a downstream agent fails 3 times with the same error pattern (token expired, agent busy, peer unreachable, etc.):

- **Do not retry a 4th time**
- Stop, summarize the failure pattern, and **escalate as "needs human intervention"** to your direct parent (PM for Leads, Lead for engineers)
- The parent should NOT retry either — they should batch the failures and ask the human

This breaks the cascade where Token-Expiry-At-Lead → Lead-Failed-At-PM → PM-Retries-Lead → repeat at 30-cycle scale, which generated 1100+ "X Lead failed" entries in 24h.

---

## 4. Do not invent phases, deadlines, or features

Before posting "Phase 34 ships X date" / "GA on Y" / "first design partner = Z" / "needs PM decision on rate limits, key rotation, partner tiers":

1. Find the phase definition in `internal/PLAN.md` or `internal/marketing/roadmap.md`
2. If the phase doesn't exist there, **it doesn't exist**. Don't invent it. Don't escalate about it.
3. If the decision genuinely needs CEO input, post once to `#ceo-feed` with a link to the source doc — never re-post the same escalation within 4 hours.

---

## 5. Token expiry is a known issue, not a P0

If you see `gh: HTTP 401` or `git: authentication failed` or `GH_TOKEN invalid`:

1. This is the GitHub App installation token TTL (60 min). It's a known recurring issue tracked in `internal/security/credential-token-backlog.md`.
2. **Do not escalate to ops or ceo-feed.** The auto-refresh daemon (when present) will fix it within ~45 min. The maintenance cron also pushes manual refreshes.
3. If you need to push urgently and the token is dead, queue the work in your own task list and retry on next cycle. Don't generate noise asking for a PAT.

---

## 6. Slack noise discipline

**Before posting to a Slack channel:**

- Search the last 30 messages in the channel — if your message duplicates anything posted in the last 4 hours, **don't post**
- For `#ops`: only post when something is actually broken AND you have a fix attempt to report
- For `#ceo-feed`: only post when CEO input is genuinely required AND no one else has asked the same question recently
- For `#engineering`: status posts are fine, but don't repeat "idle, clean" every cycle — once per shift is enough

The 24h log shows multiple "PM not responding to DMs" escalations within minutes of each other. PM was not unresponsive — PM was working.

---

## 7. Identity tag every external comment

Every GitHub PR description, issue body, comment, and Slack message MUST start with `[<your-role>-agent]` on the first line (e.g., `[core-lead-agent]`, `[devrel-engineer-agent]`).

This is required because the team shares one GitHub App identity (`molecule-ai[bot]`). Without tags, post-incident review can't attribute work to the right agent.

---

## 8. Staging-first workflow (no exceptions)

- All PRs target `staging`, never `main` directly
- `staging → main` is approved by the human CEO via their second account
- No `--admin` merges (branch protection now blocks this anyway)
- If CI is red on staging, **fix the underlying issue** — don't disable tests, don't `--no-verify`, don't add `//nolint` to silence linters

---

## 9. Memory and context hygiene

- Use `commit_memory` to record real findings; do not commit "reflections" or "I noticed X" without a tool output backing it
- Memory is shared across the role — your future self will read what you write today
- If a memory turns out to be wrong, delete it via `forget_memory` rather than leaving stale claims around
