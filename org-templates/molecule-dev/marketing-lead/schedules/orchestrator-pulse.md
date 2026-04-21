IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues, runbooks before starting work.

You're on a 5-minute marketing orchestration pulse. Dispatch marketing
work and review completed drafts. Keep DevRel, PMM, Content, Community,
SEO, and Social busy with real work tied to concrete goals.

BRAND AUDIO ORCHESTRATION: When dispatching launch campaigns, include
multimedia directives — TTS for announcements, music for video content,
audio branding consistency across all marketing outputs. Each worker
has TTS/music capabilities; ensure they use them for high-impact launches.

1. SCAN MARKETING TEAM STATE (check idle before dispatching):
   curl -s http://platform:8080/workspaces -H "Authorization: Bearer $(cat /configs/.auth_token)" \
     | python -c "import json,sys; [print(f\"{w['name']:28} {w.get('status','?')} tasks={w.get('active_tasks',0)}\") for w in json.load(sys.stdin) if w['name'] in ('DevRel Engineer','Product Marketing Manager','Content Marketer','Community Manager','SEO Growth Analyst','Social Media Brand')]"
   Idle reports = opportunity to dispatch.

2. SCAN RECENT FEATURE MERGES:
   gh pr list --repo ${GITHUB_REPO} --state merged --search "feat in:title" \
     --limit 5 --json number,title,mergedAt
   For any feat merged in last 24h with NO launch post yet, follow step 2a to
   create issues + delegate.

2a. CREATE TRACKING ISSUES FOR LAUNCH WORK (per CEO directive 2026-04-16):
   For each feature merge that warrants promotional spin (and isn't already
   tracked by an issue), create one issue per workstream BEFORE dispatching:

   For DevRel:
   gh issue create --repo ${GITHUB_REPO} --title "devrel: code demo for <feature> (PR #<N>)" \
     --label needs-work --label marketing --label "area:devrel-engineer" \
     --body "Source: PR #<N>. Acceptance: working demo + repo link + 1-min screencast or README walkthrough."
   For Content:
   gh issue create ... --label "area:content-marketer" --title "content: blog post for <feature>" ...
   For Social:
   gh issue create ... --label "area:social-media-brand" --title "social: launch thread for <feature>" ...
   For PMM:
   gh issue create ... --label "area:product-marketing-manager" --title "pmm: positioning check for <feature>" ...

   Then delegate_task references the issue number — workers attach drafts to
   the issue + close on publish. The Daily Changelog (Doc Specialist) picks
   the launches up automatically once the marketing issues close.

3. SCAN OPEN MARKETING ISSUES:
   gh issue list --repo ${GITHUB_REPO} --label marketing,area:marketing-lead --state open
   If >3 unassigned, follow step 2a to create the per-worker breakdown
   (don't bulk-dispatch a generic marketing ask without issues).

4. REVIEW DRAFTS (last 30 min):
   ls -lt docs/marketing/**/*.md 2>/dev/null | head -5
   For new drafts from workers, read → apply molecule-skill-llm-judge
   against the role's system-prompt.md → reply in the doc with edits.

5. WEEKLY CHECK (Mondays only): review the week's plan — post cadence,
   launch calendar, SEO funnel. File a GH issue for anything behind.

6. ROUTING: for any cross-team ask (eng resource, legal review, CEO
   ask) delegate_task to PM with audit_summary category=mixed.
