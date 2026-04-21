IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues, runbooks before starting work.

Independent work cycle for molecule-ai-workspace-runtime. Find work, write code, push, open PR, return to staging. FULL CYCLE REQUIRED.

STEP 1 — CHECK CURRENT STATE:
  cd /workspace/repo
  If NOT on staging: your previous work may not be pushed. Push it first:
    git fetch origin staging && git rebase origin/staging
    git push origin $(git branch --show-current)
    gh pr create --base staging --title "fix: description" --body "description" 2>/dev/null || true
    git checkout staging && git pull origin staging

STEP 2 — FIND WORK:
  gh issue list --repo Molecule-AI/molecule-ai-workspace-runtime --state open --json number,title,labels,assignees --jq '.[] | select(.assignees | length == 0) | "#\(.number) \(.title)"'
  Also: gh issue list --repo Molecule-AI/molecule-core --state open --json number,title,labels,assignees --jq '.[] | select(.assignees | length == 0) | select(.title | test("runtime|adapter|executor|workspace-template|a2a|heartbeat|preflight"; "i")) | "#\(.number) \(.title)"'

STEP 3 — SELF-ASSIGN:
  gh issue edit <NUMBER> --repo Molecule-AI/<repo> --add-assignee @me

STEP 4 — WRITE CODE:
  git checkout -b fix/issue-N-description
  Write code. Run tests.
  git add && git commit -m "fix(runtime): description (closes #N)"

STEP 5 — PUSH + OPEN PR:
  git fetch origin staging && git rebase origin/staging
  git push origin <branch>
  gh pr create --base staging --title "fix(runtime): description" --body "Closes #N"

STEP 6 — RETURN TO STAGING:
  git checkout staging && git pull origin staging
  This is MANDATORY. Do not stay on feature branch.

RULES: All PRs target staging. Rebase before push. Merge-commits only.
