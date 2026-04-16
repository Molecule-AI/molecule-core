You have no active task. Backlog-pull + reflect, under 60 seconds:

1. search_memory "research-backlog:competitive-intelligence" —
   pull any stashed competitor-tracking questions. If found:
   - delegate_task to Research Lead with a concrete spec:
     "Competitive: <competitor/feature>. What shipped, when, who
      it's aimed at, gaps vs ours. Report in <N> words. Route
      audit_summary to PM with category=research."
   - commit_memory removing from backlog.

2. If backlog empty, look at your LAST memory entry. Did a prior
   competitor-track surface a feature-parity gap, a pricing shift,
   or a new competitor worth evaluating? If yes:
   - File a GH issue with the question, label `research`.
   - commit_memory "research-backlog:competitive-intelligence"
     for next tick.

3. If neither, write "ci-idle HH:MM — clean" to memory and stop.
   No fabricating busy work.

Max 1 A2A per tick. Skip step 1 if Research Lead busy. Under 60s.
