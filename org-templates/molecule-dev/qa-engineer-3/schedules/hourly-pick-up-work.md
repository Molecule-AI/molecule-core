IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues, runbooks before starting work.

Independent QA cycle for molecule-app + docs. FULL CYCLE REQUIRED.

STEP 1 — RUN TEST SUITES:
  echo "=== molecule-app ==="
  cd /workspace/repos/molecule-app && git pull 2>/dev/null || true
  npm test 2>&1 | tail -20
  npm run build 2>&1 | tail -10
  echo "=== docs ==="
  cd /workspace/repos/docs && git pull 2>/dev/null || true
  npm run build 2>&1 | tail -10

STEP 2 — PR REVIEW:
  for repo in molecule-app docs; do
    gh pr list --repo Molecule-AI/$repo --state open --json number,title,files --limit 5
  done
  Check each PR for test coverage, accessibility, dark theme compliance.

STEP 3 — E2E TEST MAINTENANCE:
  Run Playwright tests if configured. Fix flaky tests immediately.

STEP 4 — FIND QA WORK:
  for repo in molecule-app docs; do
    gh issue list --repo Molecule-AI/$repo --state open \
      --label needs-work --json number,title --limit 3
  done

STEP 5 — WRITE TESTS:
  git checkout -b test/issue-N-description
  Write E2E/component tests.
  git add && git commit -m "test: description (closes #N)"
  git push origin <branch>
  gh pr create --base staging --title "test: description" --body "Closes #N"

STEP 6 — RETURN TO STAGING.

RULES: Build must pass. Accessibility checks. Dark theme only. Link integrity.
