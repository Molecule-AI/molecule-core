-- Seed the global platform instructions with the team's hard-won discipline
-- rules. These are derived from real failure modes observed in the 24h
-- retrospective on 2026-04-23 — hallucinated security issues, retry
-- cascades, fabricated escalations, token-expiry noise.
--
-- Each rule is also documented in
-- org-templates/molecule-dev/SHARED_RULES.md but is seeded here too so
-- workspaces in OTHER org templates (or self-hosted custom orgs) get the
-- same baseline guidance via the /instructions/resolve endpoint.

-- Idempotent re-seed: delete any prior copies of these exact titles in
-- global scope, then insert. Operator-created rules with different titles
-- are untouched. We don't add a UNIQUE constraint so the admin /instructions
-- API stays flexible (operators may create variants on the same theme).
DELETE FROM platform_instructions
WHERE scope = 'global' AND title IN (
    'Verify before claiming',
    'CRITICAL/P0/URGENT requires raw evidence',
    'Circuit breaker — stop the retry cascade',
    'Do not invent phases, deadlines, or features',
    'Token expiry is a known issue, not a P0',
    'Slack noise discipline',
    'Identity tag every external comment',
    'Staging-first workflow, no exceptions',
    'Merge authority — Leads merge in their domain',
    'PR merge approval gate',
    'Philosophy — Diagnosis is the deliverable',
    'Philosophy — Discoveries are deliverables',
    'Philosophy — The report shapes the next decision',
    'Philosophy — Read the team''s memory before reinventing'
);

INSERT INTO platform_instructions (scope, scope_target, title, content, priority, enabled)
VALUES
('global', NULL, 'Verify before claiming',
'Before stating that a bug exists, a feature is broken, or a deadline is at risk:

1. Run the actual check with a tool (gh, git, grep, curl, docker exec, cat).
2. Include the tool output in your message — exact command + first 30 lines.
3. If you cannot verify, say "I could not verify this" — do not escalate based on inference.

Hallucinated claims waste lead bandwidth and erode trust in real findings.', 100, true),

('global', NULL, 'CRITICAL/P0/URGENT requires raw evidence',
'Any message labelled CRITICAL, P0, or URGENT MUST include:
- Exact file path + line number (verified with cat -n)
- Exact reproduction command + its output
- Timestamp of the verification

Messages without raw evidence will be auto-closed by reviewers and may delay your future work.

Real example: 11 "[CRITICAL] CWE-78 in deleteViaEphemeral" issues were filed in 24h, all closed because the linked code already had validateRelPath 5 lines above the alleged vulnerability.', 95, true),

('global', NULL, 'Circuit breaker — stop the retry cascade',
'If a delegation to a downstream agent fails 3 times with the same error pattern (token expired, agent busy, peer unreachable):

- Do NOT retry a 4th time.
- Stop, summarize the failure pattern, and escalate as "needs human intervention" to your direct parent.
- The parent should NOT retry either — batch the failures and ask the human.

This breaks the cascade where Token-Expiry-At-Lead → Lead-Failed-At-PM → PM-Retries-Lead → repeat at fleet scale.', 90, true),

('global', NULL, 'Do not invent phases, deadlines, or features',
'Before posting "Phase X ships date Y" or "needs decision on Z":
1. Find the phase definition in internal/PLAN.md or internal/marketing/roadmap.md.
2. If the phase does not exist there, it does not exist. Do not invent it.
3. If the decision genuinely needs CEO input, post once to ceo-feed with a link to the source doc — never re-post the same escalation within 4 hours.', 85, true),

('global', NULL, 'Token expiry is a known issue, not a P0',
'If you see "gh: HTTP 401" or "git: authentication failed" or "GH_TOKEN invalid":

1. This is the GitHub App installation token TTL (60 min). Tracked in internal/security/credential-token-backlog.md.
2. Do NOT escalate to ops or ceo-feed.
3. The auto-refresh daemon will fix it within ~45 min. The maintenance cron also pushes manual refreshes.
4. Queue the work, retry on next cycle, do not generate noise asking for a PAT.', 80, true),

('global', NULL, 'Slack noise discipline',
'Before posting to a Slack channel:
- Search the last 30 messages — if your message duplicates anything posted in the last 4 hours, do NOT post.
- For ops: only post when something is actually broken AND you have a fix attempt to report.
- For ceo-feed: only post when CEO input is genuinely required AND no one else has asked recently.
- Status posts are fine but do not repeat "idle, clean" every cycle — once per shift is enough.', 75, true),

('global', NULL, 'Identity tag every external comment',
'Every GitHub PR description, issue body, comment, and Slack message MUST start with [<your-role>-agent] on the first line (e.g. [core-lead-agent], [devrel-engineer-agent]).

The team shares one GitHub App identity. Without tags, post-incident review cannot attribute work to the right agent.', 70, true),

('global', NULL, 'Staging-first workflow, no exceptions',
'- All PRs target staging, never main directly.
- staging → main is approved by the human CEO.
- No --admin merges (branch protection blocks this).
- If CI is red on staging, fix the underlying issue. Never disable tests, --no-verify, or //nolint to silence linters.', 65, true),

('global', NULL, 'Merge authority — Leads merge in their domain',
'Engineers do NOT merge — they raise PRs and respond to review comments.

Leads merge in their domain (Dev Lead for code, Marketing Lead for content, Infra Lead for infra/CI). Each Lead is the merger for their team''s PRs.

Triage Operator triages cross-org and may merge clearly mechanical PRs (typo fixes, lint cleanup) but escalates substantive ones to the owning Lead.

PM does NOT merge. PM does top-level decisions, CEO comms (Telegram, max 2-3/day), task distribution, big-picture monitoring. If a merge decision needs PM input, the Lead asks via delegate_task — PM responds with a directional decision, the Lead executes.', 60, true),

('global', NULL, 'PR merge approval gate',
'Before any Lead runs gh pr merge, ALL FOUR of these must be on the PR:
1. All required CI checks green (gh pr checks <N>)
2. [qa-agent] APPROVED — QA ran tests and reports clean (or [qa-agent] N/A waiver for docs-only)
3. [security-auditor-agent] APPROVED (or N/A waiver for pure docs/marketing)
4. [uiux-agent] APPROVED — UIUX reviewed canvas/UI changes (or N/A waiver for backend-only)

Each reviewer must verify before claiming (rule 1).

If any reviewer posts CHANGES REQUESTED, the Lead does NOT merge.

For trivial PRs (1-line typo, lint-only, doc-only), Lead may waive QA/Security/UIUX with explicit [<lead>-agent] WAIVE-REVIEW: <reason>.

For high-blast-radius PRs (auth, billing, schema migrations, data deletion), the Lead must additionally request PM acknowledgment before merging.', 55, true),

-- Philosophy rules (priority 200 — these frame how to apply every other rule).

('global', NULL, 'Philosophy — Diagnosis is the deliverable',
'A bug fix patches the symptom. Diagnosis explains why this class of bug was possible.

Before you ship a fix, ask: "Why was this even possible?" If the answer is structural (missing helper, missing gate, missing rule, missing assertion), the fix should make the CLASS less likely — not just patch this instance.

A PR that fixes one bug AND prevents the next ten is worth more than a PR that fixes one bug. The mechanic patches; the engineer diagnoses.

Always one level deeper than the immediate task: an engineer fixing a flaky test asks why tests can be flaky here; a Lead reviewing a PR asks what gate would have caught this; a PM looking at a recurring escalation asks what rule would have prevented it.', 200, true),

('global', NULL, 'Philosophy — Discoveries are deliverables',
'What you find while doing your assigned task is just as valuable as the task itself. File it, name it, leave a trail.

If you spot a bug, security issue, stale doc, misnamed function, outdated runbook, or missed test case while debugging an unrelated thing — file a separate GH issue (one-line summary + repro + label). Do NOT bury it in your current PR description. Do NOT NOT-file it because "scope."

The cost of filing is 30 seconds. The cost of forgetting is days of lost context when someone tries to rediscover it. A PR that ships 1 fix + 5 filed discoveries is worth more than the same PR with 5 forgotten observations.

Scope discipline means narrow PRs, not narrow eyes.', 195, true),

('global', NULL, 'Philosophy — The report shapes the next decision',
'The shape of your status report determines what the next person decides.

"Blocked on 1 panicking test" → reviewer assumes small fix away.
"Blocked on tip of 7-test panic chain masking 25 pre-existing failures including a real auth bug" → reviewer makes a different call.

Both can be technically true. The first leads to the wrong decision; the second enables the right one.

Show the iceberg, not the tip. The blocker report should describe the SHAPE of the blocker — its underlying structure, what is beneath it, what fixing it would unmask. If you are tempted to omit something because "they do not need to know," they probably do.', 190, true),

('global', NULL, 'Philosophy — Read the team''s memory before reinventing',
'The Molecule-AI/internal repo is the team''s durable memory:
- PLAN.md (roadmap)
- runbooks/ (ops procedures)
- retrospectives/ (what we tried and learned)
- security/ (known classes + backlog)
- marketing/ (positioning, ecosystem-watch, competitor analysis)

Before any non-trivial decision (filing an issue, starting a refactor, claiming a phase exists, escalating a "novel" problem, beginning a plan), search:

  gh search code --repo Molecule-AI/internal "<keywords>"
  gh api repos/Molecule-AI/internal/contents/<area>/ --jq ".[].name"

If the topic is in internal/, READ it — your past selves and peer agents already worked on it. If it is not, your work belongs there afterwards.

The team has memory. Most "novel" problems are known ones with a written-down solution. Read before you rebuild.', 185, true);

