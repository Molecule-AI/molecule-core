# Phase 34 Community FAQ

**Last updated:** 2026-04-23
**GA date:** April 30, 2026

This FAQ covers the four features that shipped in Phase 34: Tool Trace, Platform Instructions, Partner API Keys, and SaaS Federation v2. If your question isn't answered here, open a GitHub Discussion or drop a message in `#general`.

---

## Tool Trace

**Q: What is Tool Trace?**

Tool Trace gives you a complete, ordered record of every tool your agent called during a run — including what it sent in and a preview of what it got back. If you've ever had to guess why an agent did something unexpected, Tool Trace is the answer. No more reading logs line by line trying to reconstruct what happened.

---

**Q: Where exactly is the `tool_trace` field in the response?**

It's in `Message.metadata.tool_trace` on every A2A response. Each entry looks like this:

```json
"tool_trace": [
  {
    "tool_name": "web_search",
    "input": { "query": "molecule ai agent platform" },
    "output_preview": "Molecule AI is a multi-agent..."
  },
  {
    "tool_name": "write_file",
    "input": { "path": "summary.md" },
    "output_preview": "File written (412 bytes)"
  }
]
```

The full trace is also persisted to `activity_logs.tool_trace`, so you can query and audit it after the fact.

---

**Q: Does it handle parallel tool calls correctly?**

Yes. Start and end events for parallel calls are paired via a `run_id`, so concurrent calls don't get interleaved or attributed to the wrong call. If your agent fans out across multiple tools at once, the trace stays clean.

---

**Q: Is Tool Trace on by default? Do I need to flip a flag or configure anything?**

It's on by default — no configuration required. As of Phase 34, `tool_trace` is present in every A2A response. Just look for it in `message.metadata.tool_trace`.

---

**Q: Is there a cap on the number of entries in `tool_trace`?**

Yes, 200 entries per run. This is an intentional guardrail to prevent runaway agent loops from bloating your logs or your stored activity data. If you're hitting the cap regularly, that's usually a signal that the agent loop itself needs attention.

---

**Q: Is Tool Trace available on all plans?**

Yes. Tool Trace is available on all plans — it's a core part of the A2A response, not a premium add-on.

---

## Platform Instructions

**Q: What are Platform Instructions?**

Platform Instructions let org admins configure a system-level instruction set that applies to every agent in the org — automatically, without touching individual workspace configs. Think of it as a shared system prompt for your entire platform: compliance rules, house-style requirements, shared context, anything that should be consistent across every agent you run.

---

**Q: How do I set Platform Instructions for my org?**

Make an authenticated `PUT` request to `/cp/platform-instructions` with your org admin token:

```http
PUT /cp/platform-instructions
Authorization: Bearer <your-org-admin-token>
Content-Type: application/json

{
  "instructions": "Always respond in English. Tag every response with the agent workspace ID."
}
```

That's it. Every agent in the org inherits those instructions immediately.

---

**Q: What's the difference between global scope and workspace scope?**

Global scope (what `PUT /cp/platform-instructions` sets) applies across the entire org — every workspace, every agent. Workspace scope applies only to a specific workspace and its agents. When both are set, they are combined: the platform-level instructions run alongside whatever workspace-specific instructions are configured. Global scope is the right place for anything that must be org-wide; workspace scope is for context that only belongs to one team or product area.

---

**Q: Is Platform Instructions available on all plans?**

Yes. Platform Instructions is available on all plans, including the free tier.

---

## Partner API Keys

**Q: What are Partner API Keys (`mol_pk_*`)?**

Partner API Keys are a programmatic provisioning API for teams building platforms on top of Molecule AI. If you're operating a marketplace, a CI/CD integration, or any multi-tenant product where you need to create and manage Molecule AI orgs on behalf of your customers — this is for you.

Keys use the `mol_pk_*` prefix and are org-scoped: they can't escape the org boundary, so there's no risk of a key leaking access to other orgs.

---

**Q: When do Partner API Keys go GA?**

April 30, 2026. They're available in preview now, but the GA release — with stable API contracts and full docs — lands on April 30.

---

**Q: What can I actually build with Partner API Keys?**

Some concrete examples:

- **Ephemeral test orgs per PR** — spin up a fresh Molecule AI org for every pull request, tear it down when the PR closes, and stop billing on `DELETE`.
- **Multi-tenant SaaS products** — programmatically provision a Molecule AI org per customer when they sign up.
- **CI/CD integrations** — create orgs, run tests, delete orgs, all from your pipeline.
- **Marketplace integrations** — list your product on a marketplace and automate the org-provisioning step of onboarding.

The key primitives are:

```http
POST   /cp/admin/partner-keys        # create a key
DELETE /cp/admin/partner-keys/:id    # delete (and stop billing)
```

---

**Q: How do I get started with Partner API Keys?**

Full docs are at `/docs/api/partner-keys` and go live on April 30. If you want early access before then, drop a message in `#partner-program` on Discord and the team will get you set up.

---

## SaaS Federation v2

**Q: What changed in SaaS Federation v2?**

SaaS Fed v2 is a targeted reliability and correctness improvement for teams running federated Molecule AI across multiple orgs. The headline changes are better reliability under load and cleaner enforcement of org boundaries. If you're operating a federated deployment, check the Phase 34 changelog for the full diff — the improvements are most visible in edge cases that previously caused intermittent boundary leakage or dropped federation events.

---

## General

**Q: Where do I report bugs or unexpected behavior?**

Post in `#bug-reports` on the Molecule AI Discord. Include your org ID, the request or action that triggered the issue, and any error messages or response bodies you're seeing. The team monitors `#bug-reports` actively and will triage from there.

---

**Q: Where is the partner Discord channel?**

Join `#partner-program` on the Molecule AI Discord. That's the right place for Partner API Key access requests, partner integration questions, and feedback from teams building on top of the platform.
