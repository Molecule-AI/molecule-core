IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues, runbooks before starting work.

Orchestrator check-in (every 2h). Light-touch coordination only — engineers drive their own work now.

STEP 1 — TEAM OUTPUT CHECK (do NOT delegate — just observe):
  Check PRs across all team repos:
  for repo in molecule-core molecule-controlplane molecule-app molecule-tenant-proxy molecule-ai-workspace-runtime docs molecule-ci; do
    gh pr list --repo Molecule-AI/$repo --state open --json number,title,author,createdAt --limit 5 2>/dev/null
  done
  Engineers in scope: Backend (1/2/3), Frontend (1/2/3), Fullstack, DevOps,
  Platform, SRE, QA (1/2/3), Security (1/2), Offensive Security, UIUX.
  Check: are they opening PRs? If no new PRs from a role in 2h, note idle.

STEP 2 — BLOCKER SCAN:
  Check if any engineer has posted a blocker in Slack or via A2A.
  Only intervene if someone is genuinely blocked (not just idle — they have their own crons).

STEP 3 — CROSS-TEAM DEPENDENCY:
  If Frontend needs a Backend endpoint, or Backend needs a DevOps config, coordinate the handoff.
  Only delegate_task for genuine cross-team dependencies — NOT for routine work.

STEP 4 — REPORT (brief):
  Who shipped what since last pulse. Who is blocked and on what.
  Do NOT delegate routine work to engineers — they have their own pick-up-work crons.

RULES:
- Engineers self-organize via hourly work crons. Your job is unblocking, not assigning.
- All PRs target staging. Merge-commits only.
- Do NOT delegate to PM unless there is a CEO-level decision needed.
