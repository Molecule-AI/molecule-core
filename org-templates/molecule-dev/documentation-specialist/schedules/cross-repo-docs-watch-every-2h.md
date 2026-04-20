IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues, runbooks before starting work.

Cross-repo docs watch. Fire every 2 hours. Mandate: keep documentation in
lockstep with the entire Molecule-AI/* GitHub org (40+ repos), NOT just
molecule-core. Updates that match repository state are owned by Doc Specialist
alone — no marketing approval needed. Marketing only enters the picture for
promotional spin on top of factual changes (e.g. blog post for a major release).

## 1. SETUP — record the cycle window

```bash
LAST_TICK=$(recall_memory "doc-watch-last-tick" 2>/dev/null || echo '2 hours ago')
NOW_TS=$(date -u +%Y-%m-%dT%H:%M:%SZ)
echo "Window: $LAST_TICK → $NOW_TS"
```

## 2. ENUMERATE every Molecule-AI repo (live list, don't trust the prior cache)

```bash
gh repo list Molecule-AI --limit 60 --json name,description,updatedAt,visibility \
  > /tmp/org-repos.json
```

Filter to repos that received commits since LAST_TICK — those are the ones
worth scanning. (Skipping idle repos keeps the cycle bounded.)

## 3. PER-REPO: list merged PRs in the window

For each repo with recent activity:
```bash
gh pr list --repo Molecule-AI/<repo> --state merged \
  --search "merged:>=${LAST_TICK}" \
  --json number,title,mergedAt,files \
  --limit 20
```

For each merged PR, check `files`:
- Touches a public API (`platform/internal/handlers/`, `platform/internal/router/`) → docs site `api-reference.mdx` likely needs update.
- Touches a template repo (`workspace-configs-templates/*`, standalone template repo) → docs site `org-template.mdx` or `concepts.mdx`.
- Touches a plugin repo → docs site `plugins.mdx` (and the plugin repo's own README).
- Touches a channel adapter (`platform/internal/channels/`, e.g. the new `lark.go` or `slack.go`) → docs site `channels.mdx`.
- Touches a schedule / cron / workflow → docs site `schedules.mdx`.
- Touches `migrations/` → docs site `architecture.mdx` schema section + a callout in the daily changelog.
- Touches CI (`*.yml` in `.github/workflows/`) → typically internal-only; skip unless it changes a publicly-documented release/deploy flow.
- Touches `controlplane/` (PRIVATE repo) → update `controlplane/README.md` and `controlplane/PLAN.md`. **NEVER mention controlplane internals in public docs site.** Per privacy rule.

## 4. WRITE THE DOCS PR

For each docs gap discovered:
1. Branch in the docs site repo: `docs/<short-topic>-from-pr-<repo>-<number>` (e.g. `docs/lark-channel-from-core-480`)
2. Edit the relevant MDX file. Include:
   - 1-paragraph what-changed prose
   - The new/changed config syntax in a fenced code block
   - A working example
   - Cross-link to the PR that introduced it (`See [#480](...)` etc.)
3. Run `npm run build` locally (the docs site is a Next.js app — link checker + MDX parse run during build). Skip the PR if build fails; fix the docs first.
4. Open PR with title `docs(<area>): pair PR <repo>#<n> — <topic>` and body referencing the originating PR. **Always branch + PR — never commit to main on any repo.**

## 5. TERMINOLOGY DRIFT CHECK

Quick grep on the merged PRs' diffs for any new concept names. Compare to:
```bash
recall_memory "canonical-terminology" 2>/dev/null
```
If the PR introduces a NEW term that wasn't in your terminology memory, add it.
If the PR uses a SYNONYM of an existing term, file a fix-up PR to align with
the canonical name and update the terminology memory in same cycle.

## 6. STUB BACKFILL — opportunistic

If you finished the per-PR pairings with cycle time to spare, pick the
oldest "Coming soon" stub from the docs site and backfill it. Track
remaining stubs in memory under `stubs-pending` so the next tick picks the
next-oldest, not the same one twice.

## 7. MEMORY UPDATE — end of cycle

```python
commit_memory(
  key="doc-watch-last-tick",
  value=NOW_TS,
)
commit_memory(
  key=f"doc-watch-cycle-{NOW_TS[:13]}",
  value={
    "repos_scanned": [...],
    "prs_paired": [{"repo": r, "pr": n, "docs_pr": dp} for ...],
    "terminology_drift_caught": [...],
    "stubs_backfilled": [...],
    "deferred_to_next_cycle": [...],
  },
)
```

## 8. ESCALATION

- **Marketing handoff**: only when a PR represents a customer-facing
  feature launch worth blog-post coverage. Use `delegate_task` to
  Marketing Lead with a link to your docs PR + a one-liner of why it's
  notable. Don't ask marketing for routine docs updates — those are
  yours alone per CEO directive 2026-04-16.
- **Cross-team blockers**: if a PR is so undocumentable that you need
  the original engineer's input (private API, complex behavior), use
  `delegate_task` to Dev Lead asking for a clarifying comment on the
  source PR.
- **Privacy violations**: if you spot a public PR that leaks
  controlplane internals (file paths, internal endpoints, schema
  details), open a Critical issue on molecule-controlplane and
  IMMEDIATELY notify Security Auditor via A2A.

## DEFINITION OF DONE FOR THIS CYCLE

- Memory updated with `doc-watch-last-tick`
- Every PR merged in the window has either: a paired docs PR open, OR a memory
  note explaining why it didn't need one (CI-only, internal refactor, etc.)
- No tools/files touched on `main` directly (always branch + PR)
- Activity log entry summarising the cycle's output (PR count, docs PR URLs)

6. INTERNAL DOCS REPO — Molecule-AI/internal (added 2026-04-18):
   This is the team's private knowledge base. You own keeping it current:
   - PLAN.md — product roadmap. Update when phases complete or priorities shift.
   - known-issues.md — update when issues are resolved or new ones discovered.
   - runbooks/ — operational playbooks. Update when infra changes (e.g. Fly.io → Railway migration).
   - security/ — threat models and findings. Sync with Security Auditor's audit outputs.
   - retrospectives/ — session retrospectives. Add entries after major incidents or milestones.
   - ecosystem-watch.md, ecosystem-research-outcomes.md — sync with Research Lead outputs.

   Every 2h check:
   gh pr list --repo Molecule-AI/internal --state open --json number,title
   gh api repos/Molecule-AI/internal/commits --jq '.[0:3] | .[] | "\(.sha[:8]) \(.commit.message | split("\n") | first)"'
   If internal docs are stale vs actual platform state (e.g. still reference Fly.io), open a PR to fix.
   NEVER copy internal content to public repos (molecule-core, docs). Privacy rule applies.
