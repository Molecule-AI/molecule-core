IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues, runbooks before starting work.

Hourly infrastructure health check. Execute ALL steps:

1. CI STATUS — check recent workflow runs across ALL org repos:
   for repo in molecule-core molecule-controlplane molecule-app molecule-tenant-proxy molecule-ai-workspace-runtime docs molecule-ci; do
     gh run list --repo Molecule-AI/$repo --limit 3 --json status,conclusion,name,createdAt 2>/dev/null
   done
   If any failed, investigate and fix or file issue.

2. DEPENDABOT CHECK — review dependency update PRs:
   for repo in molecule-core molecule-controlplane molecule-app molecule-tenant-proxy docs; do
     gh pr list --repo Molecule-AI/$repo --state open --label dependencies --json number,title --limit 3 2>/dev/null
   done
   Approve safe minor/patch updates. Flag breaking major updates.

3. MULTI-REPO ISSUE SCAN:
   For each repo: molecule-core, molecule-controlplane, molecule-ai-workspace-runtime,
   molecule-tenant-proxy, molecule-ci, molecule-app, docs, landingpage, molecule-ai-status
   gh issue list --repo Molecule-AI/<repo> --state open --json number,title,createdAt
   Flag any issue older than 48h with no assignee. Pick up if in your domain.

4. MULTI-REPO PR SCAN:
   Check open PRs across key repos. Flag PRs with failing CI or no reviews after 24h.

5. DOCKER IMAGES:
   Check ghcr.io/molecule-ai/* image tags, compare with latest commits.

6. MIGRATION SEQUENCE:
   ls platform/migrations/*.up.sql | tail -5
   Check numbering sequential, no duplicates.

7. INFRASTRUCTURE STATUS:
   - Platform API: curl -sI https://api.moleculesai.app/health (Railway)
   - Staging API: curl -sI https://staging-api.moleculesai.app/health (Railway)
   - Canvas: curl -sI https://app.moleculesai.app (Vercel)
   - Docs: curl -sI https://doc.moleculesai.app (Vercel)
   NOTE: We are on Railway now, NOT Fly.io.

8. INTERNAL REPO CHECK:
   gh issue list --repo Molecule-AI/internal --state open
   Check for new runbooks, security findings, or roadmap updates.

NOTE: Platform Engineer handles molecule-ai-status, molecule-ci, and shared workflows.
Coordinate — you focus on live infra health; Platform Engineer on CI pipeline + Dependabot.

Report findings with specific issue numbers, file paths, and proposed fixes.
