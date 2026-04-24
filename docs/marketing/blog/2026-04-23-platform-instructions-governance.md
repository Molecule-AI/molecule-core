---
title: "Govern Your Entire AI Fleet at the System Prompt Level"
date: 2026-04-23
slug: govern-ai-fleet-system-prompt-level
description: "Platform Instructions lets enterprise IT enforce org-wide and workspace-scoped policy rules at the system prompt level — before the first agent turn executes."
og_title: "Govern Your Entire AI Fleet at the System Prompt Level"
og_description: "Global rules. Workspace rules. Prepended to the system prompt at startup. Governance before the first turn, not after."
og_image: /docs/assets/blog/2026-04-23-platform-instructions-governance-og.png
tags: [governance, platform-instructions, enterprise, security, it-governance, system-prompt, policy, a2a]
keywords: [AI fleet governance, enterprise AI policy, system prompt governance, AI agent compliance, platform instructions, workspace policy enforcement, enterprise AI security]
canonical: https://docs.molecule.ai/blog/govern-ai-fleet-system-prompt-level
---

<script type="application/ld+json">
{
  "@context": "https://schema.org",
  "@type": "Article",
  "headline": "Govern Your Entire AI Fleet at the System Prompt Level",
  "description": "Platform Instructions lets enterprise IT enforce org-wide and workspace-scoped policy rules at the system prompt level.",
  "author": { "@type": "Organization", "name": "Molecule AI" },
  "datePublished": "2026-04-23",
  "publisher": { "@type": "Organization", "name": "Molecule AI", "logo": { "@type": "ImageObject", "url": "https://molecule.ai/logo.png" } }
}
</script>

# Govern Your Entire AI Fleet at the System Prompt Level

The moment an AI agent goes into production, the governance question stops being theoretical. Which tools can it call? What data can it write to? Are there constraints that apply to every turn, not just the ones where someone remembered to add a guardrail?

Most platforms answer with post-hoc filtering — a rule that checks outputs after the agent has already decided what to do. Platform Instructions takes a different approach: governance at the source, before the first token is generated. Rules are prepended to the system prompt at workspace startup, shaping what the agent is instructed to do from the very first turn. The agent doesn't receive them as a filter. It receives them as context.

## Global and Workspace Scopes

Platform Instructions supports two scoping levels:

- **Global** — applied to every workspace in your organization. One rule, enforced everywhere across your entire AI fleet.
- **Workspace** — applied to a specific workspace only. Fine-grained control without global impact.

When a workspace starts, Molecule AI resolves all applicable instructions — combining global rules with workspace-specific ones — and prepends them to the agent's system prompt. The distinction matters: a filter can be worked around; a system prompt instruction shapes the agent's reasoning from the ground up.

## The API

Platform Instructions are managed via a REST API:

```bash
# Create a global instruction (org-admin token required)
POST /instructions
Authorization: Bearer <org-admin-token>
Content-Type: application/json

{
  "scope": "global",
  "content": "Before invoking any tool that writes to external storage, confirm the target path is within the org-approved sandbox directory. Reject and report if not."
}

# Create a workspace-scoped instruction
POST /instructions
Authorization: Bearer <workspace-admin-token>
Content-Type: application/json

{
  "scope": "workspace",
  "workspace_id": "ws_01HXKM3T8PRQN4ZW7XYVD2EJ5A",
  "content": "This workspace handles customer PII. Redact all PII fields in tool outputs before writing to external systems."
}

# Retrieve resolved instructions for a workspace
# (wsAuth-gated — workspace can only read its own)
GET /workspaces/ws_01HXKM3T8PRQN4ZW7XYVD2EJ5A/instructions/resolve
Authorization: Bearer <workspace_token>
```

The resolve endpoint is gated by wsAuth — the calling workspace's own token. Workspaces cannot enumerate or retrieve instructions from other workspaces. There is no cross-workspace read-back. Each instruction is capped at 8KB of content. Resolved instruction sets are fetched once at startup and cached, so governance is enforced without adding latency to individual agent turns.

## Enforcement Before Execution

The architectural difference from post-hoc policy is timing. A post-hoc filter evaluates after the agent decides what to do. Platform Instructions are in the system prompt before the agent decides anything.

For regulated environments where the requirement is "prevent bad behavior" not "flag bad behavior," that distinction is everything. A compliance team that requires PII redaction doesn't want the agent to write raw PII and redact it on the way out — they want the agent to reason about redaction as part of its core task framing. Platform Instructions makes that possible.

## Enterprise Security: ACLs That Match the Requirement

Platform Instructions are enterprise-only because enterprise governance requires enterprise-grade access control:

- **Global instructions** are managed by org admins — not workspace owners
- **Workspace instructions** are managed by workspace admins within their own scope only
- **Resolve endpoint** requires wsAuth — a workspace cannot retrieve another workspace's instructions
- **No cross-workspace enumeration** — the API returns nothing to callers outside the owning scope

For IT governance teams, this is the access control surface the compliance review demands: policy lives at the org level, is enforced at the workspace level, and cannot be read or modified by the agents it governs.

## Get Started

Platform Instructions are available on **Enterprise plans**. To get started:

- Contact your account team or visit your workspace settings
- Define your first global instruction via POST /instructions
- Assign workspace-scoped instructions to specific workspaces
- Verify resolved instructions via GET /workspaces/{id}/instructions/resolve

For a complete governance picture, combine Platform Instructions with Tool Trace — see exactly which tools were called and what inputs were passed, alongside the policy that governed them.

---

*Molecule AI is open source. Platform Instructions shipped in Phase 34 (2026-04-23). Enterprise plans include org-scoped governance, wsAuth-gated resolve endpoints, and full instruction audit logs.*
