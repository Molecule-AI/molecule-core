# Phase 34 — Community Response Queue
**Campaign:** Phase 34 GA (April 30, 2026)
**Owner:** Community Manager
**Use:** Ready-to-deploy response templates for launch day inbound questions
**Date:** 2026-04-23
**Status:** ⚠️ HOLD — GA vs Beta label conflict unresolved (PM must confirm)

> ⚠️ **ALL PHASE 34 POSTING ON HOLD.**
> Internal positioning briefs say "BETA" for Tool Trace + Platform Instructions.
> External launch assets say "live now / GA."
> PM must confirm labeling before any Discord announcement, Reddit, or HN post goes live.
> Full conflict documented at `docs/marketing/briefs/phase34-ga-vs-beta-conflict.md`.

---

## Channel Monitoring Priority

| Channel | Priority | SLA | Notes |
|---------|----------|-----|-------|
| `#announcements` | HIGH | 15 min | Launch thread — watch for clarification questions |
| `#faq` | HIGH | 30 min | Pinned — people reading before asking |
| `#general` | MEDIUM | 30 min | Broad questions, how-to |
| `#feedback` | MEDIUM | 30 min | Feature requests — tag @pm |
| `#bug-reports` | HIGH | 15 min | Bugs → tag @devrel immediately |
| `#partner-program` | MEDIUM | 30 min | Early access requests → DM |
| `#devrel` | LOW | 60 min | Cross-team routing only |

---

## Tool Trace Questions

### "Is Tool Trace available now? What plan do I need?"
→ Tool Trace is live for all Molecule AI workspaces. No plan upgrade required. It's in the A2A response by default — check `Message.metadata.tool_trace` on any response.

### "How is this different from Langfuse or Helicone?"
→ Tool Trace captures A2A-level agent behavior — tool calls, inputs, output previews. Langfuse/Helicone capture LLM API calls (prompt → response). They measure different layers. Tool Trace is platform-native, zero-config, free. Langfuse/Helicone are still great if you need cross-platform multi-model observability.

### "My agent isn't producing tool_trace in responses."
→ Tool Trace is on by default for all workspaces. Make sure activity logging is enabled on your workspace. If it is and you're still not seeing traces, open a bug in #bug-reports and tag @devrel.

### "Does Tool Trace cost extra?"
→ No. Tool Trace is included with every Molecule AI plan. No additional cost, no tier restriction.

### "What's the 200-entry cap?"
→ The cap prevents runaway loops from generating unbounded trace data. A task calling more than 200 tools will have entries dropped from the end — you get the first 200, sufficient for virtually any production task. If you're hitting the cap regularly, consider breaking the task into smaller subtasks.

### "Does it work with custom MCP tools?"
→ Yes. Tool Trace records every tool invocation via the agent runtime — built-in and custom MCP tools appear identically in the trace.

### "How do I read the tool trace? Is there a UI?"
→ The raw trace is in `Message.metadata.tool_trace[]` on every A2A response. A Canvas UI for traces is in progress. The `activity_logs` table stores traces as JSONB — query via `GET /workspaces/:id/activity?type=tool_call`.

---

## Platform Instructions Questions

### "How is this different from config.yaml system prompts?"
→ `config.yaml` sets the workspace owner's system prompt. Platform Instructions are rules the platform injects at startup — before the agent reads its own config. Workspace users can't override or remove them by editing `config.yaml`. It's governance, not configuration.

### "Can workspace users remove Platform Instructions?"
→ No. Platform Instructions are injected by the platform at startup. A workspace user cannot remove them by editing `config.yaml` or any workspace setting.

### "Can workspace-level instructions override global ones?"
→ Workspace-level rules are additive to global ones, not overriding. Both apply simultaneously. A workspace rule can explicitly opt out a global rule by prefix (e.g. `!global-rule-name`), but by default both apply.

### "Is Platform Instructions on all plans?"
→ Yes. Platform-native, included with every plan. No additional subscription required.

### "Can Platform Instructions leak my system prompt to other orgs?"
→ No. Platform Instructions are scoped to the workspace that created them. No cross-workspace enumeration is possible. `GET /workspaces/:id/instructions/resolve` requires workspace authentication — a workspace can only read its own instructions.

---

## Partner API Keys Questions

### "Is Partner API Keys available now?"
→ **Not yet.** GA is April 30, 2026. Until then the API is not live. If you want early access for a concrete integration use case, DM me and I'll connect you with the team.

### "Can I use it for ephemeral CI/CD test orgs?"
→ Yes — that's a primary use case. Create org → run test suite → delete org → billing stops. Rate limited at 60 req/min per key (default), configurable. Org creation is async — poll for `status: "active"` before running tests.

### "What's the rate limit?"
→ Default is 60 requests/minute per key, configurable at key creation time. For high-volume CI pipelines, use one key per pipeline runner or request a higher limit when applying for a partner key.

### "Can a partner key access other orgs' data?"
→ Scoped keys can only access the orgs they're authorized for. `org_id` bindings are enforced. Revocation is immediate — no grace period.

### "My key was deleted — are the orgs it created still running?"
→ Yes. Deleting a key does NOT delete the orgs it created. Org deletion is a separate API call (`DELETE /cp/orgs/:slug`). Orgs continue running and accruing billing until explicitly deleted.

---

## SaaS Fed v2 Questions

### "What changed in v2? Is this just a rebrand?"
→ No — it's the multi-tenant control plane architecture. Improved cross-tenant isolation, cleaner org lifecycle management, tighter alignment with Partner API Keys infrastructure. External workspaces benefit from improved heartbeat reliability.

### "Does this affect self-hosted users?"
→ No. SaaS Fed v2 is SaaS-only. Self-hosted users are unaffected.

### "Who does this affect?"
→ Teams running multiple Molecule AI orgs — partners, internal teams, separate products. Single-org users don't need to take action.

---

## General / Escalation

### "This feature doesn't work / I found a bug."
→ Sorry to hear that — let me get the right eyes on this. Can you share what you were trying to do and what happened? I'll flag it in #bug-reports for the platform team.

### "I want early access to [feature]."
→ For Tool Trace + Platform Instructions: already live, no early access needed. For Partner API Keys: DM me with your use case and I'll connect you with the team.

### "Security concern / vulnerability."
→ Do not respond publicly. Take the details DM-only and escalate to Security immediately.

### "Press / media inquiry."
→ Do not engage publicly. DM Marketing Lead immediately.

### "Toxic or spam thread."
→ Do not engage. Screenshot and DM Marketing Lead with link.

---

## Feature Request Logging

When a feature request comes in:
1. Acknowledge in channel ("Love this idea — tagging @pm")
2. DM @pm with: issue description, channel context, number of community members asking similar things
3. Optionally: open a GitHub issue with label "enhancement" and share link in channel

---

*Last updated: 2026-04-23 — hold on GA posting pending PM confirmation*