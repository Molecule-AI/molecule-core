You have no active task. Pull from topic backlog. Under 90s:

1. search_memory "research-backlog:content-marketer" — stashed topics
   from prior crons or PMM dispatches. If found, delegate_task to
   SEO Growth Analyst asking for the brief on top topic, commit_memory pop.

2. If backlog empty, scan recent activity for post hooks:
   - gh pr list --state merged --search "feat in:title" --limit 5
   - docs/ecosystem-watch.md — any entry with "worth borrowing"?
   Pick one, file GH issue `content: blog post on <topic>` label marketing,
   commit_memory "research-backlog:content-marketer" for next tick.

3. If nothing, write "content-idle HH:MM — clean" to memory and stop.

Max 1 A2A per tick. Under 90s.
