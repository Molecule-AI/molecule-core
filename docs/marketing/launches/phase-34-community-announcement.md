# Phase 34 Community Announcement
**Campaign:** Phase 34 GA (April 30, 2026)
**Owner:** Community Manager
**Issue:** [Molecule-AI/molecule-core#1836](https://github.com/Molecule-AI/molecule-core/issues/1836)
**Status:** ✅ Draft complete — review before publish

---

```
🚀 Phase 34 shipped!

Four features dropped today that make Molecule AI meaningfully better for
teams running agents in production. Let's dig in.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

What's new

🔍 Tool Trace — see exactly what your agent did
──────────────────────────────────────────────
Every A2A response now includes a tool_trace[] array in the response
metadata. For each tool call, you get: tool name, input params, output
preview — in order, with run_id pairing for parallel calls.

If you've ever spent an hour reverse-engineering an agent's behavior from
final outputs, you'll understand why this matters. Tool Trace is in every
response. No SDK, no sidecar, no sampling.

Enable activity logging on your workspace and every task gets a full
execution record, stored in activity_logs.

Docs: docs.moleculesai.app/blog/ai-agent-observability-without-overhead (live on staging)
PR: #1686 (merged 2026-04-23)


📋 Platform Instructions — governance before the agent runs
──────────────────────────────────────────────
Org admins can now configure system-level instructions via API:
POST /cp/platform-instructions

Think of it as a system prompt for your entire org. You can inject shared
context, enforce behavioral rules, or set compliance guardrails — and they
take effect at workspace startup, before the agent reads its own config.

"Never commit to main without a review."
"Tag all security-sensitive operations."
"Don't write outside the project directory."

These rules prepend to every agent's effective system prompt in your org,
regardless of individual workspace config. Governance happens before
execution, not after an incident.

Docs: docs.moleculesai.app/blog/platform-instructions-governance (live on staging)


🔑 Partner API Keys (GA April 30)
──────────────────────────────────────────────
Programmatic org provisioning via API — no browser, no manual handoff.

CI/CD pipelines, marketplace integrations, platform builders: you can now
create and manage Molecule AI orgs entirely via API using scoped,
revocable tokens with the mol_pk_* prefix.

Common pattern: ephemeral test orgs per PR.
  POST /cp/orgs → run your test suite → DELETE → billing stops.

Default rate limit: 60 req/min per key (configurable). Keys are scoped,
irreversibly revocable, and logged in the audit trail.

This is, to our knowledge, the first partner provisioning API of its kind
in the agent platform space.

Docs: docs.moleculesai.app/blog/partner-api-keys (live on staging)
GA: April 30, 2026


☁️ SaaS Federation v2
──────────────────────────────────────────────
Federation improvements for multi-org deployments: better cross-tenant
isolation, improved org lifecycle management, and tighter alignment with
the Partner API Keys infrastructure.

If you're running multiple orgs — for partners, for internal teams, or
for different products — the improvements to federation architecture make
it more robust to operate at scale.

Docs: docs.moleculesai.app/guides/external-workspace-quickstart (live on staging)


━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Quick start

Tool Trace is live now — enabled by default for all workspaces.
Check your A2A responses for message.metadata.tool_trace[].

Platform Instructions: POST /cp/platform-instructions (org admin only)
Docs: docs.moleculesai.app/blog/platform-instructions-governance (live on staging)

Partner API Keys: GA April 30 — docs.moleculesai.app/blog/partner-api-keys
Apply for early access via GitHub Discussions if you want to test ahead.

SaaS Fed v2: docs.moleculesai.app/guides/external-workspace-quickstart (live on staging)


━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Try it / tell us what you think

Questions on any of these? Reply here — DevRel and the platform team are
monitoring.

Got feedback on Tool Trace? We want to know what you'd use it for.
Drop it in GitHub Discussions: github.com/Molecule-AI/molecule-core/discussions

Full blog coverage on staging:
- docs.moleculesai.app/blog/tool-trace-platform-instructions (combined overview)
- docs.moleculesai.app/blog/ai-agent-observability-without-overhead (Tool Trace deep-dive)
- docs.moleculesai.app/blog/platform-instructions-governance (Platform Instructions)
- docs.moleculesai.app/blog/partner-api-keys (Partner API Keys)
- docs.moleculesai.app/guides/external-workspace-quickstart (SaaS Fed v2)

Full messaging matrix: docs/marketing/briefs/phase34-positioning.md

Documentation across all four features is linked above — if something
is missing or unclear, open an issue and tag @community.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

*Phase 34 — production-grade AI agents. Nothing to bolt on.*
```

---

## Publishing notes

- **Post on:** Discord `#announcements` + `#general`, Slack `#announcements`
- **Timing:** April 30, 2026 GA day, ~09:00 Pacific
- **Blog must be live first** — coordinate with Content Marketer before posting
- **Partner API Keys:** Do NOT claim it's live before April 30. Frame as "GA April 30."
- **Do NOT name specific design partners** in community posts
- **Engagement:** Monitor threads for 2 hours after posting; route technical questions to DevRel