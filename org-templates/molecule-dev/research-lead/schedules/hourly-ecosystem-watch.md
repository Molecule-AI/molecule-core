IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues, runbooks before starting work.

Daily survey for new agent-infra / AI-agent projects worth tracking.

1. Pull docs/ecosystem-watch.md to know what's already tracked.
2. Browse the web for last 24h:
   - github.com/trending?since=daily&language=python (and typescript, go)
   - HN front page, anything about agent frameworks
   - Twitter/X mentions of new agent SDKs, MCP servers, frameworks
3. Cross-reference: skip anything already in ecosystem-watch.md.
4. For each genuinely new + relevant project (1-3 max per day):
   - Add an entry under "## Entries" using the existing template
     (Pitch / Shape / Overlap / Differentiation / Worth borrowing /
      Terminology collisions / Signals to react to / Last reviewed + stars)
   - Keep each entry ≤200 words.
5. If a finding suggests a concrete improvement to plugins/, workspace-template/,
   or org-templates/, file a GH issue (`gh issue create`) with the proposal.
6. Commit additions to a branch named chore/eco-watch-YYYY-MM-DD. PUSH it
   (per the repo "always raise PR" policy) and open a PR.
7. Routing: delegate_task to PM with summary
   (audit_summary metadata: category=research, severity=info,
    issues=[<gh issue numbers>], top_recommendation=<one-liner>).
8. If nothing notable today, skip the commit and PM-message a one-line "clean".
