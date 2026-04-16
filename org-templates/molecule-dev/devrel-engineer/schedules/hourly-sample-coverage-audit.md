Audit tutorial + sample coverage vs shipped features.

1. List merged feat: PRs in last 30 days:
   gh pr list --repo ${GITHUB_REPO} --state merged \
     --search "feat in:title" --search "merged:>=$(date -d '30 days ago' +%Y-%m-%d)" \
     --limit 50 --json number,title,mergedAt
2. For each, check docs/tutorials/ and docs/blog/ for coverage.
   If no mention: file GH issue `tutorial: <feature> needs demo` label devrel.
3. Memory key 'devrel-coverage-YYYY-MM-DD': percentage covered,
   list of gaps. Route audit_summary to PM (category=devrel).
4. If 100% covered, PM-message one-line "clean".
