IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues, runbooks before starting work.

Independent work cycle for CI, status, internal. Be productive every tick.

STEP 1 — CI HEALTH CHECK (across ALL org repos):
  gh repo list Molecule-AI --limit 60 --json name -q '.[].name' | while read repo; do
    FAILED=$(gh run list --repo Molecule-AI/$repo --status failure --limit 1 --json databaseId -q '.[].databaseId' 2>/dev/null)
    if [ -n "$FAILED" ]; then
      echo "FAILING CI: Molecule-AI/$repo — run $FAILED"
    fi
  done

STEP 2 — DEPENDABOT CHECK:
  for repo in molecule-core molecule-controlplane molecule-app molecule-tenant-proxy docs; do
    gh pr list --repo Molecule-AI/$repo --state open --label dependencies --json number,title --limit 3
  done
  Review and approve safe dependency updates.

STEP 3 — STATUS PAGE ACCURACY:
  curl -sI -o /dev/null -w "%{http_code}" https://status.moleculesai.app
  Cross-check Upptime monitors against actual service endpoints.

STEP 4 — FIND WORK:
  gh issue list --repo Molecule-AI/molecule-ci --state open --label needs-work --json number,title --limit 3
  gh issue list --repo Molecule-AI/molecule-ai-status --state open --label needs-work --json number,title --limit 3
  gh issue list --repo Molecule-AI/internal --state open --label needs-work --json number,title --limit 3

STEP 5 — If CI is broken, fix it. Branch, commit, push, PR. Return to staging.

RULES: CI health is #1 priority. Pin action versions. No secrets in logs.
