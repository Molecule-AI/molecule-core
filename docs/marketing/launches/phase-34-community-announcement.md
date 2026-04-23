# Phase 34 Community Announcement
**Campaign:** Phase 34 GA (April 30, 2026)
**Owner:** Community Manager
**Issue:** [Molecule-AI/molecule-core#1836](https://github.com/Molecule-AI/molecule-core/issues/1836)
**Status:** ✅ Draft complete — ready for April 30 publish

---

```
🚀 Phase 34 shipped!

If you've been running agents in production and wondering what they actually
did under the hood — today we shipped the visibility to answer that.
Four platform-level features, no SDK required.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

What's new

🔍 Tool Trace — finally, see what your agents are doing
──────────────────────────────────────────────
Every A2A response now carries a tool_trace[] array in Message.metadata.
For every tool call: the tool name, the input you sent, and a snippet
of the output — in the order they happened, with run_id pairing so
parallel calls are traced correctly.

If you've ever spent time reverse-engineering agent behavior from final
outputs alone, Tool Trace is for you.

Capped at 200 entries per task (prevents runaway loops). Stored in
activity_logs. Zero extra infrastructure — it's in the response
envelope your agent already produces.

Docs: docs.moleculesai.app/blog/ai-agent-observability-without-overhead
PR: #1686 (merged 2026-04-23)


📋 Platform Instructions — system prompt for your whole org
──────────────────────────────────────────────
Org admins can now configure workspace-level instructions via:
PUT /cp/platform-instructions

Think of it as a system prompt that applies to every agent in your org —
no need to update config files per workspace. Set it once, it applies
to every agent at startup.

Example rules:
  "Never commit directly to main — always open a PR first."
  "Tag all security-sensitive operations with a audit tag."
  "Confirm before running destructive commands in production."

Rules prepend to each agent's effective system prompt, before the
agent reads its own config. Governance happens before execution,
not after an incident.

Docs: docs.moleculesai.app/blog/platform-instructions-governance


🔑 Partner API Keys (GA April 30) — first agent platform with a
first-class partner provisioning API
──────────────────────────────────────────────
Marketplaces, CI/CD pipelines, and platform builders: you can now
provision and manage Molecule AI orgs entirely via API, using scoped
revocable tokens with the mol_pk_* prefix.

Ephemeral test orgs per PR:
  PUT /cp/orgs → run your test suite → DELETE → billing stops.

Tokens are scoped to the orgs they're authorized for, rate-limited
(60 req/min default, configurable per key), and fully audit-logged.
Revocation is immediate — no grace period.

GA: April 30, 2026. Docs and early access requests:
docs.moleculesai.app/blog/partner-api-keys


☁️ SaaS Federation v2 — improved multi-org federation
──────────────────────────────────────────────
Better cross-tenant isolation, cleaner org lifecycle management, and
tighter alignment with the Partner API Keys infrastructure. If you're
operating multiple orgs — for partners, internal teams, or separate
products — the federation improvements make multi-org setups more
robust at scale.

Docs: docs.moleculesai.app/guides/external-workspace-quickstart


━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Quick start

Tool Trace: live now. Inspect message.metadata.tool_trace[] in any
A2A response — no SDK, no activation step needed.
Walkthrough: docs.moleculesai.app/blog/ai-agent-observability-without-overhead

Platform Instructions: PUT /cp/platform-instructions (org admin only)
Walkthrough: docs.moleculesai.app/blog/platform-instructions-governance

Partner API Keys: GA April 30.
Docs: docs.moleculesai.app/blog/partner-api-keys

SaaS Fed v2: docs.moleculesai.app/guides/external-workspace-quickstart


━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Try it & tell us what you think

Questions on any of these? Reply here — DevRel and the platform team
are watching.

Tool Trace feedback especially welcome — we want to know what you'd
use the trace data for. Drop it in GitHub Discussions:
github.com/Molecule-AI/molecule-core/discussions

Full Phase 34 blog coverage on staging:
  docs.moleculesai.app/blog/tool-trace-platform-instructions (overview)
  docs.moleculesai.app/blog/ai-agent-observability-without-overhead (Tool Trace)
  docs.moleculesai.app/blog/platform-instructions-governance (Platform Instructions)
  docs.moleculesai.app/blog/partner-api-keys (Partner API Keys)
  docs.moleculesai.app/guides/external-workspace-quickstart (SaaS Fed v2)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

*Phase 34 — production-grade AI agents. Nothing to bolt on.*
```

---

## Publishing checklist

- [ ] Discord `#announcements` + `#general` — April 30, ~09:00 PT
- [ ] Slack `#announcements` — same time
- [ ] Blog must be live before announcement post goes out
- [ ] Partner API Keys: do not claim it's live before April 30
- [ ] Do NOT name design partners in community copy
- [ ] Monitor threads for 2 hours after posting; route DevRel questions accordingly