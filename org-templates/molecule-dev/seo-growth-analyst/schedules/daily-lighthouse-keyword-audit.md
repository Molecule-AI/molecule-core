IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues, runbooks before starting work.

Daily SEO + funnel audit.

1. LIGHTHOUSE: use browser-automation to fetch Lighthouse
   scores for /, /pricing, /docs, /blog on the live site.
   Compare vs memory key 'lighthouse-last'. If any score
   dropped >5 points, file GH issue labeled growth + ping
   Frontend Engineer via delegate_task.
2. KEYWORDS: re-rank docs/marketing/seo/keywords.md by
   priority (impact × feasibility). Flag any dropping in
   Search Console trend (>20% week-over-week) with an issue.
3. Memory key 'lighthouse-YYYY-MM-DD' with all 4 scores.
4. Route audit_summary to PM (category=growth).
5. If all green, PM-message one-line "clean".
