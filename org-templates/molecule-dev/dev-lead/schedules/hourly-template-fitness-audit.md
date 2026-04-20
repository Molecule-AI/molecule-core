IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues, runbooks before starting work.

Daily audit of `org-templates/molecule-dev/`. Catches drift, stale prompts,
missing schedules, and gaps that block the team-runs-24/7 goal. Symptom
of prior incident (issue #85): cron scheduler died silently for 10+ hours
and nobody noticed because no one was watching template fitness.

1. CHECK SCHEDULES ARE FIRING:
   For every workspace_schedule in the platform DB:
   curl -s http://host.docker.internal:8080/workspaces/<id>/schedules
   Compare last_run_at to now() vs cron interval. Anything more than 2x
   the interval behind = STALE. File issue against platform.

2. CHECK SYSTEM PROMPTS ARE FRESH:
   cd /workspace/repo
   for f in org-templates/molecule-dev/*/system-prompt.md; do
     echo "$(git log -1 --format='%ar' -- "$f") $f"
   done
   Anything not touched in 30+ days might be stale relative to recent
   platform changes. Spot-check vs CLAUDE.md and recent merges.

3. CHECK ROLES HAVE PLUGINS THEY NEED:
   yq '.workspaces[] | (.name, .plugins)' org-templates/molecule-dev/org.yaml
   (or python+yaml). Roles inherit defaults; flag any role that should
   plausibly have role-specific extras (compare role description vs
   plugins list).

4. CHECK CRONS COVER THE EVOLUTION LEVERS:
   The team must keep evolving plugins, template, channels, watchlist.
   Verify schedules exist for: ecosystem-watch (Research Lead),
   plugin-curation (Technical Researcher), template-fitness (you,
   this cron), channel-expansion (DevOps).
   Any missing? File issue.

5. CHECK CHANNELS:
   Today only PM has telegram. Should any other role have a channel?
   (Security Auditor → email on critical findings; DevOps → Slack on
    build breaks; etc.) File issue if a channel gap is meaningful.

6. ROUTING: delegate_task to PM with audit_summary metadata
   (category=template, severity=…, issues=[…], top_recommendation=…).
7. If everything is fit and current, PM-message one-line "clean".
