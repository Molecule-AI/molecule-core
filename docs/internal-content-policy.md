# Internal content policy

The `Molecule-AI/molecule-monorepo` repo is **public**. Anything internal
(positioning, competitive briefs, sales playbooks, PMM/press drip, draft
campaigns, raw research notes, ops runbooks, retrospectives) lives in
**`Molecule-AI/internal`**.

This page is the canonical decision tree.

## Quick decision

> *"I'm an agent (or human) about to write a markdown file. Where does it go?"*

| If the artifact is… | Put it in… |
|---|---|
| Competitive brief, market analysis, raw research notes | `Molecule-AI/internal/research/` |
| PMM positioning draft, sales playbook, press release pre-publish | `Molecule-AI/internal/marketing/` |
| Draft campaign asset (still iterating, not yet customer-visible) | `Molecule-AI/internal/marketing/campaigns/` |
| Roadmap discussion, planning doc, retrospective | `Molecule-AI/internal/PLAN.md` or `Molecule-AI/internal/retrospectives/` |
| Runbook, ops procedure, incident postmortem | `Molecule-AI/internal/runbooks/` |
| **Public-ready** blog post (final draft, ready to ship to docs site) | `Molecule-AI/molecule-monorepo/docs/blog/` |
| **Public-ready** tutorial / quickstart | `Molecule-AI/molecule-monorepo/docs/tutorials/` |
| Public DevRel content (code samples, demos for users) | `Molecule-AI/molecule-monorepo/docs/devrel/` |
| API reference, architecture docs for external developers | `Molecule-AI/molecule-monorepo/docs/api/` |
| Code, tests, infrastructure | wherever is appropriate inside this repo |

**Rule of thumb:** *"Would I be comfortable if a competitor / journalist / customer
read this verbatim today?"* — yes → `monorepo/docs/`. No / not yet → `internal/`.

## Why

This repo is publicly indexable. Anything pushed here is permanently in git
history, search-engine indexed, and accessible to anyone who clones. Past
incidents (audit 2026-04-23) found:

- Competitive teardowns of CrewAI / Paperclip / VoltAgent at root `/research/`
- 45 marketing artifacts at root `/marketing/` including `pmm/positioning.md`,
  `press/launch.md`, `sales/enablement.md`
- 31 draft campaign files at `/docs/marketing/`
- Junk temp files at root: `comment-1172.json`, `tick-reflections-temp.md`

All migrated to `internal/from-monorepo-2026-04-23/` for curator triage.

## Enforcement

Three layers, all required:

1. **`.gitignore`** — blocks the directories at `git add` time. Quietest
   layer; doesn't fire if someone uses `git add -f`.
2. **CI workflow `block-internal-paths.yml`** — fails any PR that adds a
   forbidden path. Mechanical backstop. Cannot be bypassed without editing
   the workflow + PR review.
3. **Agent prompts** — `SHARED_RULES.md` rule (in
   `molecule-ai-org-template-molecule-dev`) tells every agent role to
   write internal content to `Molecule-AI/internal` directly via `gh repo
   clone` + commit + PR. This is the prevention-at-source layer.

If you're hitting the CI gate and your file genuinely belongs in this repo,
edit `FORBIDDEN_PATTERNS` in the workflow with reviewer signoff. Don't
work around the gate by renaming files.

## How to write to the internal repo (for agents)

```bash
# One-time clone (idempotent — re-running is a no-op)
mkdir -p ~/repos
test -d ~/repos/internal || gh repo clone Molecule-AI/internal ~/repos/internal

cd ~/repos/internal
git pull origin main
mkdir -p research
cat > research/<slug>.md <<EOF
# <title>

…content…
EOF

git checkout -b <agent-role>/research-<slug>
git add research/<slug>.md
git commit -m "research: add <slug>"
git push -u origin HEAD
gh pr create --base main --fill
```

Yes, this is more steps than `cd molecule-monorepo && git add research/foo.md`.
That cost is intentional: the friction is the point. Public space and
internal space are different products with different audiences and
different durability guarantees.
