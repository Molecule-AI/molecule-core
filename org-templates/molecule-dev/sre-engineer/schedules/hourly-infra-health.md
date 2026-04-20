IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues, runbooks before starting work.

Hourly infrastructure health check. Execute ALL steps:                                                                                                                        +
                                                                                                                                                                              +
1. CI STATUS — check recent workflow runs:                                                                                                                                    +
   gh run list --repo Molecule-AI/molecule-core --limit 5 --json status,conclusion,name,createdAt                                                                             +
   If any failed, investigate and fix or file issue.                                                                                                                          +
                                                                                                                                                                              +
2. MULTI-REPO ISSUE SCAN — check open issues across key repos:                                                                                                                +
   For each repo: molecule-core, molecule-controlplane, molecule-ai-workspace-runtime, molecule-tenant-proxy, molecule-ci, molecule-app, docs, landingpage, molecule-ai-status+
   gh issue list --repo Molecule-AI/<repo> --state open --json number,title,createdAt                                                                                         +
   Flag any issue older than 48h with no assignee or comment. If it's in your domain (CI, Docker, migrations, deploy), pick it up.                                            +
                                                                                                                                                                              +
3. MULTI-REPO PR SCAN — check open PRs across key repos:                                                                                                                      +
   For each repo above: gh pr list --repo Molecule-AI/<repo> --state open                                                                                                     +
   Check CI status. Flag any PR with failing CI or no reviews after 24h.                                                                                                      +
                                                                                                                                                                              +
4. DOCKER IMAGES — verify platform and workspace images are current:                                                                                                          +
   Check ghcr.io/molecule-ai/* image tags, compare with latest commits.                                                                                                       +
                                                                                                                                                                              +
5. MIGRATION SEQUENCE — verify no gaps:                                                                                                                                       +
   ls platform/migrations/*.up.sql | tail -5                                                                                                                                  +
   Check numbering is sequential, no duplicates.                                                                                                                              +
                                                                                                                                                                              +
6. INFRASTRUCTURE STATUS:                                                                                                                                                     +
   - Platform API: curl -sI https://api.moleculesai.app/health (Railway)                                                                                                      +
   - Staging API: curl -sI https://staging-api.moleculesai.app/health (Railway)                                                                                               +
   - Canvas: curl -sI https://app.moleculesai.app (Vercel)                                                                                                                    +
   - Docs: curl -sI https://doc.moleculesai.app (Vercel)                                                                                                                      +
   NOTE: We are on Railway now, NOT Fly.io. Do not probe any *.fly.dev URLs.                                                                                                  +
                                                                                                                                                                              +
7. INTERNAL REPO CHECK:                                                                                                                                                       +
   gh issue list --repo Molecule-AI/internal --state open                                                                                                                     +
   gh pr list --repo Molecule-AI/internal --state open                                                                                                                        +
   Check Molecule-AI/internal for any new runbooks, security findings, or roadmap updates relevant to infra.                                                                  +
                                                                                                                                                                              +
Report findings with specific issue numbers, file paths, and proposed fixes.
