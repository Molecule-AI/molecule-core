IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues, runbooks before starting work.

Weekly survey of channel integrations (Telegram, Slack, Discord, email,
webhooks). The team should grow its external comms surface where useful,
not stay locked at "PM-only Telegram".

1. INVENTORY:
   yq '.workspaces[] | {name: .name, channels: .channels}' \
     org-templates/molecule-dev/org.yaml 2>/dev/null
   (or python+yaml). List which roles have which channels.
2. PLATFORM CAPABILITY CHECK:
   grep -rE "channel|telegram|slack|discord|webhook" \
     platform/internal/handlers/ --include="*.go" -l
   What channel types does the platform actually support today?
3. GAP ANALYSIS:
   - PM has Telegram → can the user reach OTHER roles directly?
   - Security Auditor: would email-on-critical-finding help?
   - DevOps Engineer: would Slack-on-CI-break help?
   - Any role that produces high-value asynchronous output but the
     user has to poll memory to see it?
4. EXTERNAL: are there channel platforms we should consider adding?
   (Discord for community, GitHub Discussions for product, etc.)
5. For the top 1-2 gaps, file a GH issue:
   - "Channel proposal: <type> for <role>" with rationale, integration
     sketch, secret requirements (e.g. SLACK_BOT_TOKEN as global secret).
6. ROUTING: delegate_task to PM with audit_summary metadata
   (category=channels, issues=[…], top_recommendation=…).
7. If no gap this week, PM-message a one-line "clean".
