You're on a 5-minute research orchestration pulse. Coordinate your
research team (Market Analyst, Technical Researcher, Competitive Intelligence).
Keep them busy with real research, not idle between eco-watch fires.

1. SCAN TEAM STATE:
   curl -s http://host.docker.internal:8080/workspaces | \
     python3 -c "import json,sys
   names = {'Market Analyst','Technical Researcher','Competitive Intelligence'}
   for w in json.load(sys.stdin):
     if w.get('name') in names and w.get('status')=='online':
       print(f\"{w['name']:25} busy={'Y' if w.get('active_tasks',0)>0 else 'N'}\")"

2. CHECK RESEARCH BACKLOG:
   - gh issue list --repo ${GITHUB_REPO} --state open --label research --json number,title
   - search_memory "research-question" — questions from PM waiting for an answer
   - Questions you yourself stashed from eco-watch reflection

3. DISPATCH (max 2 A2A per pulse — research is slow):
   - Market sizing / user research / pricing → Market Analyst
   - Framework / SDK / MCP evaluation / protocol research → Technical Researcher
   - Competitor feature tracking / roadmap diffs → Competitive Intelligence
   delegate_task format: "Research <topic>. Report in <N> words. When done, send
     audit_summary to PM with category=research, severity=info, top_recommendation=<one-liner>."

4. REVIEW completed research from last 5 min:
   If a subordinate finished, summarize their output and route the summary to PM
   via delegate_task with audit_summary metadata.

5. REPORT:
   commit_memory "research-pulse HH:MM — dispatched <N>, reviewed <M>, idle <K>".

HARD RULES:
- Max 2 A2A sends per pulse.
- If the eco-watch cron is currently in flight (fires at :08 and :38), SKIP this
  pulse entirely — don't collide with your own deep-work task.
- Don't dispatch to a busy researcher.
- Under 60 seconds wall-clock per pulse.
- If all 3 researchers are idle AND backlog is empty → write "research-clean HH:MM"
  to memory and stop. No busy work.
