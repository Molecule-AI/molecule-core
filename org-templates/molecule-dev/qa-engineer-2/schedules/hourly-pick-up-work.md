IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues, runbooks before starting work.

Independent QA cycle for molecule-controlplane + molecule-tenant-proxy. FULL CYCLE REQUIRED.

STEP 1 — RUN TEST SUITES:
  for repo in molecule-controlplane molecule-tenant-proxy; do
    echo "=== $repo ==="
    cd /workspace/repos/$repo && git pull 2>/dev/null || true
    go test -race ./... 2>&1 | tail -20
  done

STEP 2 — PR REVIEW FOR TEST COVERAGE:
  for repo in molecule-controlplane molecule-tenant-proxy; do
    gh pr list --repo Molecule-AI/$repo --state open --json number,title,files --limit 5
  done
  For each PR: check if changed files have corresponding test updates.
  Leave review comments for coverage gaps.

STEP 3 — FIND QA WORK:
  for repo in molecule-controlplane molecule-tenant-proxy; do
    gh issue list --repo Molecule-AI/$repo --state open \
      --label needs-work --json number,title --limit 3
  done
  Pick highest-priority test improvement. Self-assign, branch, implement.

STEP 4 — WRITE TESTS:
  git checkout -b test/issue-N-description
  Write integration/regression tests.
  git add && git commit -m "test: description (closes #N)"

STEP 5 — PUSH + OPEN PR:
  git push origin <branch>
  gh pr create --base staging --title "test: description" --body "Closes #N"

STEP 6 — RETURN TO STAGING:
  git checkout staging && git pull origin staging

RULES: All tests must pass. Coverage must not decrease. Flaky = fix immediately.
