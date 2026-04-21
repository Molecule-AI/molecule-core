You have no active task. Proactively check infrastructure health:

1. Check CI status: `gh run list --repo Molecule-AI/molecule-core --limit 5 --json conclusion,name`
2. Check for migration issues: `ls platform/migrations/*.up.sql | tail -5` — verify sequential numbering
3. Check Docker image freshness: `docker images --format "{{.Repository}}:{{.Tag}} {{.CreatedSince}}" | grep workspace`
4. Check for open infra issues: `gh issue list --repo Molecule-AI/molecule-core --label infra --state open --limit 5`
5. If nothing queued, audit Dockerfile reproducibility or CI workflow security (pinned actions, no floating tags)

Pick ONE item, fix it. Under 90 seconds.
