IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues, runbooks before starting work.

Independent work cycle for molecule-core (Go + Canvas). Find work, write code, push, open PR, return to staging. FULL CYCLE REQUIRED.

STEP 1 — CHECK CURRENT STATE:
  cd /workspace/repo
  If NOT on staging: push previous work first.
    git fetch origin staging && git rebase origin/staging
    git push origin $(git branch --show-current)
    gh pr create --base staging --title "fix: description" --body "description" 2>/dev/null || true
    git checkout staging && git pull origin staging

STEP 2 — FIND WORK (prefer cross-cutting issues):
  gh issue list --repo Molecule-AI/molecule-core --state open --json number,title,labels,assignees --jq '.[] | select(.assignees | length == 0) | select(.title | test("fullstack|api.*canvas|websocket|endpoint.*ui|handler.*component"; "i")) | "#\(.number) \(.title)"'
  Also pick up any issue that touches both platform/ and canvas/.

STEP 3 — SELF-ASSIGN:
  gh issue edit <NUMBER> --repo Molecule-AI/molecule-core --add-assignee @me

STEP 4 — WRITE CODE:
  git checkout -b fix/issue-N-description
  Write code on BOTH sides if needed.
  Run tests:
    cd workspace-server && go test -race ./...
    cd ../canvas && npm test && npm run build
  git add && git commit -m "fix: description (closes #N)"

STEP 5 — PUSH + OPEN PR:
  git fetch origin staging && git rebase origin/staging
  git push origin <branch>
  gh pr create --base staging --title "fix: description" --body "Closes #N"

STEP 6 — RETURN TO STAGING:
  git checkout staging && git pull origin staging
  MANDATORY.

RULES: All PRs target staging. Both test suites must pass. Merge-commits only.
