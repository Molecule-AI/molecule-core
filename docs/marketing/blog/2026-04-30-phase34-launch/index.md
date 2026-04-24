---
title: "Phase 34: Partner API Keys, SaaS Federation v2, Tool Trace, and Platform Instructions"
date: 2026-04-30
slug: phase-34-launch
description: "Phase 34 ships four production features: Partner API Keys for programmatic org management, SaaS Federation v2 for multi-org collaboration, Tool Trace for full observability, and Platform Instructions for governance at the system prompt level."
og_title: "Phase 34: Four Production Features for AI Agent Platforms"
og_description: "Partner API Keys. SaaS Federation v2. Tool Trace. Platform Instructions. Phase 34 ships April 30 — the biggest platform release yet."
og_image: /docs/assets/blog/2026-04-30-phase34-launch-og.png
tags: [phase-34, partner-api, federation, observability, governance, enterprise, tool-trace, platform-instructions, a2a]
keywords: [AI agent platform, Partner API Keys, SaaS Federation, enterprise AI, AI agent observability, agent governance, A2A protocol, production AI]
canonical: https://docs.molecule.ai/blog/phase-34-launch
---

<script type="application/ld+json">
{
  "@context": "https://schema.org",
  "@type": "Article",
  "headline": "Phase 34: Partner API Keys, SaaS Federation v2, Tool Trace, and Platform Instructions",
  "description": "Phase 34 ships four production features for AI agent platforms: Partner API Keys, SaaS Federation v2, Tool Trace, and Platform Instructions.",
  "author": { "@type": "Organization", "name": "Molecule AI" },
  "datePublished": "2026-04-30",
  "publisher": { "@type": "Organization", "name": "Molecule AI", "logo": { "@type": "ImageObject", "url": "https://molecule.ai/logo.png" } }
}
</script>

# Phase 34: Partner API Keys, SaaS Federation v2, Tool Trace, and Platform Instructions

**April 30, 2026 — GA**

Phase 34 is Molecule AI's biggest platform release yet. Four production features ship today: **Partner API Keys** for programmatic org lifecycle management, **SaaS Federation v2** for cross-org collaboration at scale, **Tool Trace** for complete observability into every agent execution, and **Platform Instructions** for governance enforced at the system prompt level.

These aren't add-ons. They're the infrastructure layer that moves AI agent platforms from "it ran" to "it's under control."

---

## Partner API Keys: Programmatic Org Management for Platforms and Resellers

The headline feature of Phase 34 is one that most end users won't see directly — but every platform builder will feel immediately.

When your product needs to create a Molecule AI org on behalf of a customer — a new tenant, a CI environment, a marketplace resale — you shouldn't need a human with a browser. Partner API Keys give marketplace resellers, automation platforms, and CI/CD tooling a scoped, rate-limited, revocable API key — prefixed `mol_pk_` — that authenticates to the partner-facing control plane. No dashboard. No session token. One API call.

```bash
# Mint a Partner API Key (admin-master-key required)
POST /cp/admin/partner-keys
{
  "name": "acme-ci-pipeline",
  "scopes": ["orgs:create", "orgs:list", "workspaces:create"]
}

# Response — key shown ONCE
{
  "id": "pak_01HXKM4...",
  "key": "mol_pk_1a2b3c4d5e6f...",
  "name": "acme-ci-pipeline",
  "scopes": ["orgs:create", "orgs:list", "workspaces:create"],
  "created_at": "2026-04-30T00:00:00Z"
}
```

With the key in hand, a partner automation pipeline drives the full customer onboarding lifecycle programmatically:

```bash
# 1. Create the org
POST /cp/orgs
Authorization: Bearer mol_pk_1a2b3c4d5e6f...
{
  "name": "acme-corp",
  "slug": "acme-corp",
  "plan": "standard"
}

# 2. Poll until the org is provisioned
GET /cp/orgs/org_01HXKM4.../status

# 3. Ship the customer their login link
# → https://app.moleculeai.ai/login?org=acme-corp

# 4. Revoke when the integration ends
DELETE /cp/admin/partner-keys/pak_01HXKM4...
```

Available scopes: `orgs:create`, `orgs:list`, `orgs:delete`, `workspaces:create`, `billing:read`. Each key has independent rate limits — a misbehaving integration hits its own ceiling without affecting other partners or organic traffic. Every call is audited: the log records which key was used, when, and what it did.

**Pricing tiers and rate limits are [coming soon](https://molecule.ai/pricing).** Contact your Molecule AI account team to request key issuance for your integration.

Partner API Keys are available on Partner and Enterprise plans.

---

## SaaS Federation v2: Cross-Org Collaboration Without Credential Sharing

Phase 33 shipped org-scoped API keys — credentials that live at the organization level and reach every workspace in your org. SaaS Federation v2 builds on that foundation to let organizations collaborate across org boundaries, without sharing credentials, without workarounds, and without a separate "enterprise bridge" service to maintain.

Federation v2 introduces a structured trust model between orgs. An org admin grants a named federation trust to a partner org — specifying exactly which workspaces are reachable and which operations are permitted. The trust is scoped, revocable, and auditable. No shared secrets. No bilateral token exchange. No custom middleware.

```bash
# Org A: establish a trust relationship with Org B
POST /orgs/:org_a/federation/trusts
{
  "target_org_id": "org_01HXKM4B...",
  "target_org_name": "Acme Corp",
  "workspace_ids": ["ws_abc123", "ws_def456"],
  "allowed_operations": ["workspaces:read", "channels:read"],
  "expires_at": "2027-04-30T00:00:00Z"
}
```

Once established, agents in Org A can route tasks to Org B's workspaces using Molecule AI's existing A2A protocol — with federation claims embedded in the request metadata so the receiving org can verify the caller's org identity without receiving their org token.

**Why this matters:** In Phase 33, cross-org collaboration either required credential sharing (bad) or manual operator handoff (slow). Federation v2 makes it a first-class, auditable, revocable protocol operation. Your AI fleet can hand off work to a partner's fleet — and you can see exactly what was handed off, when, and whether it was permitted.

SaaS Federation v2 is available on Enterprise plans. Contact your account team to enable federation trusts for your orgs.

---

## Tool Trace: Full Observability Into Every Agent Execution

When your AI agent produces a wrong answer in production, the question isn't "what did the agent say?" — it's "which tool did it call, with what inputs, and what did it get back?"

Most platforms answer the first question. Tool Trace answers all three — for every call, in every response.

Every A2A response from Molecule AI now includes a structured `tool_trace` array in `Message.metadata`. For each tool invocation:

- **`tool`** — the tool name (`Bash`, `Write`, `Read`, `HTTPRequest`, etc.)
- **`input`** — the exact parameters passed
- **`output_preview`** — first ~200 characters of the result
- **`run_id`** — groups concurrent calls so parallel executions don't collapse into a scrambled sequence

```json
{
  "metadata": {
    "tool_trace": [
      {
        "tool": "Bash",
        "input": { "command": "go build ./... && go test ./..." },
        "output_preview": "ok      auth    0.314s\nok      config  0.201s\n--- PASS: TestIntegration (12.3s)",
        "run_id": "01HXKM3T8PRQN4ZW7XYVD2EJ5A"
      },
      {
        "tool": "Read",
        "input": { "file_path": "/workspace/coverage/report.json" },
        "output_preview": "Read 2.1 KB from /workspace/coverage/report.json",
        "run_id": "01HXKM3T8PRQN4ZW7XYVD2EJ5A"
      }
    ],
    "run_id": "01HXKM3T8PRQN4ZW7XYVD2EJ5A"
  }
}
```

Instead of "the agent ran the migration and it failed," you get: `tool_call[2] — Bash — kubectl apply -f migration.sql — returned error 1062: Duplicate entry — 0 rows affected — 23ms.` The diagnosis takes minutes, not hours.

Tool Trace is available on all Molecule AI plans. No sampling, no sidecar service. It's in the A2A response metadata.

---

## Platform Instructions: Governance at the System Prompt Level

Most governance platforms filter outputs *after* an agent decides what to do. Platform Instructions takes a different approach: governance at the source, before the first token is generated.

Platform Instructions are policy rules prepended to the agent's system prompt at workspace startup. The agent doesn't receive them as a filter. It receives them as part of its task framing from the very first turn — which means the policy shapes the agent's reasoning, not just its output.

### Two Scopes, One Model

- **Global** — applied to every workspace in your organization. One rule, enforced everywhere across your AI fleet.
- **Workspace** — applied to a specific workspace only. Fine-grained control without global impact.

When a workspace starts, Molecule AI resolves all applicable instructions and prepends them to the system prompt. The distinction matters: a post-hoc filter can be worked around; a system prompt instruction shapes the agent's reasoning from the ground up.

```bash
# Create a global instruction (org-admin token)
POST /instructions
{
  "scope": "global",
  "content": "Before invoking any tool that writes to external storage, confirm the target path is within the org-approved sandbox directory. Reject and report if not."
}

# Create a workspace-scoped instruction
POST /instructions
{
  "scope": "workspace",
  "workspace_id": "ws_01HXKM3T8PRQN4ZW7XYVD2EJ5A",
  "content": "This workspace handles customer PII. Redact all PII fields in tool outputs before writing to external systems."
}
```

The resolve endpoint is gated by `wsAuth` — a workspace can only read its own instructions. Each instruction is capped at 8KB. Resolved instruction sets are fetched once at startup and cached, so governance doesn't add per-turn latency.

**For regulated environments, this is the architecture compliance reviews demand:** policy lives at the org level, is enforced at the workspace level, and cannot be read or modified by the agents it governs. Combine with Tool Trace to see exactly which tools were called alongside the policy that governed them.

Platform Instructions are available on Enterprise plans.

---

## What This Means for Developers

Phase 34 is an infrastructure release. It doesn't change how agents run — it changes what you can do *after*.

**If you're building a platform on Molecule AI:** Partner API Keys let your marketplace, CI pipeline, or automation tool manage orgs programmatically. Create, provision, audit, revoke — no human in the loop.

**If you're collaborating across orgs:** SaaS Federation v2 makes cross-org agent handoffs a first-class, auditable protocol operation — not a credential-sharing workaround.

**If you're debugging in production:** Tool Trace turns opaque failures into structured traces. You know which tool failed, what it received, what it returned, and how long it took.

**If you're in a regulated industry:** Platform Instructions means governance is in the system prompt before the agent decides anything. Not a filter. Not a separate service. It's part of the workspace contract.

---

## Get Started

- **Partner API Keys** — available on Partner and Enterprise plans. Contact your account team to request key issuance. Pricing tiers [coming soon](https://molecule.ai/pricing).
- **SaaS Federation v2** — available on Enterprise plans. Contact your account team to enable federation trusts.
- **Tool Trace** — available on all plans. See the [A2A Protocol Reference](/docs/api-protocol/a2a-protocol) for the full `Message.metadata` schema.
- **Platform Instructions** — available on Enterprise plans. Configure them in your workspace settings or via the REST API.

[Molecule AI](https://molecule.ai) is open source. Phase 34 ships April 30, 2026.

*Phase 34 ships in PRs #1807 (Tool Trace), #XXXX (SaaS Federation v2), and across the platform control plane.*
