# Issue #1831 — PMM Positioning Brief: Tool Trace + Platform Instructions
**Source:** PR #1686 (feat: tool trace + platform instructions, merged 2026-04-23)
**Status:** READY — cleared for Content Marketer and Social Media Brand use
**Owner:** PMM | **Date:** 2026-04-23
**Gate for:** Issue #1835 (blog post), Issue #1829 (social thread), Phase 34 GA (April 30, 2026)

---

## PART 1 — TOOL TRACE

### What it is

Tool Trace ships a `tool_trace[]` array in every A2A `Message.metadata`. Each entry records:
- **`tool`** — tool name (e.g. `Write`, `Bash`, `Grep`)
- **`input`** — exact parameters passed
- **`output_preview`** — first ~200 chars of result (readable at scale)
- **`run_id`** — pairs concurrent parallel calls so traces don't merge

Entries are written to `activity_logs.tool_trace` as JSONB. Platform-native: ships inside the A2A response, no SDK, no sidecar, no instrumentation required.

### Who it's for

**Primary:** Platform engineers, DevOps leads, SREs who get paged when agents break in production.

**Secondary:** Enterprise IT debugging teams, developer advocates demoing agent reliability to prospective buyers.

### Primary value prop

> "Every A2A task now comes with a complete, structured execution record — so you know exactly what your agent did, without wiring up a third-party SDK."

### Recommended messaging angle

**Lead with developer productivity / debugging, not observability infrastructure.**

The 2am pager scenario is the most resonant entry point: *"You know the final output. Now you know exactly how it got there."* This speaks directly to the platform engineering persona and lands without requiring them to understand A2A internals.

**Use these phrases in top-level copy:** "execution tracing," "agent observability," "know what your agent did"
**Avoid leading with:** "tool_trace" as a feature name, "Message.metadata" anywhere in copy, "JSONB" anywhere in copy

### Competitive differentiation

| Competitor | Their approach | Molecule AI advantage |
|-----------|---------------|----------------------|
| Langfuse / Helicone / Braintrust | Third-party SDK, separate pipeline, sampling-based | Tool Trace ships inside every A2A response — zero config, no SDK, no pipeline |
| Datadog / Splunk | Generic LLM observability, requires instrumentation | Molecule-specific layer — captures A2A-level agent behavior (tool call sequences) that generic LLM pipelines miss or flatten |
| OpenTelemetry | Standard for distributed tracing, requires agent instrumentation | Platform-native, no instrumentation needed; captures at A2A layer not model-call layer |
| Hermes (Molecule native) | Traces individual model calls | Tool Trace traces *agent behavior* — tool call sequences, not model tokens. Complementary, not competitive. |

**Counter-framing for sellers:**
> "Langfuse and Helicone are great for cross-platform, multi-model observability. Tool Trace is your Molecule-specific layer inside your existing stack — and it ships with every response, no instrumentation required."

### HN/Reddit framing

**Do:** Lead with developer experience. *"Tool Trace ships today in Molecule AI. Every agent turn now includes a structured record of every tool called — inputs, output previews, run_id-paired for parallel calls."* Be honest: beta feature.

**Do NOT:** Claim this is GA. Don't say "generally available" or "production-ready by default." Say "now in beta" or "shipping today."

---

## PART 2 — PLATFORM INSTRUCTIONS

### What it is

Platform Instructions is a governance layer that prepends workspace-scoped rules to the system prompt at workspace startup. Rules take effect *before* the first agent turn — shaping what the agent is instructed to do from the start, not filtering outputs after the fact.

Two scoping levels:
- **Global** — applied to every workspace in the org. One rule, enforced everywhere.
- **Workspace** — applied to a specific workspace only. Fine-grained control without global impact.

Policy updates take effect at the next workspace restart. No code change, no application redeploy, no agent restart required.

### Who it's for

**Primary:** Enterprise IT, Security/Compliance leads, CISO office.

**Secondary:** Platform Engineering leads who need to enforce org-wide guardrails without touching agent code.

### Primary value prop

> "Governance before the first token is generated — not after an incident."

### Recommended messaging angle

**Lead with pre-execution governance and compliance, not admin control.**

The core contrast that resonates with enterprise buyers: most governance tools are *reactive* (filter outputs after the agent has already acted). Platform Instructions is *proactive* (shapes behavior before the first turn). Frame it as the difference between a guard on the exit door and a guard at the entrance.

**Use these phrases:** "governance before agents run," "pre-execution guardrails," "policy without a deployment," "shape agent behavior at the system prompt level"

**Avoid leading with:** "admin control," "workspace policies," or anything that sounds like IT bureaucracy. The value is speed and safety, not restriction.

### Competitive differentiation

| Competitor | Their approach | Molecule AI advantage |
|-----------|---------------|----------------------|
| OPA / Sentinel (policy-as-code) | Runtime resource access enforcement | Platform Instructions fires earlier in the chain (pre-execution vs. during-execution). Complementary, not competitive. |
| Output filtering (most platforms) | Post-hoc filtering of agent outputs | Platform Instructions shapes behavior *before* the model generates anything. Proactive vs. reactive. |
| Custom system prompt injection | Requires code changes per workspace | Platform Instructions is workspace-scoped config — no code, no redeploy. |
| CrewAI / LangGraph | No equivalent feature | First-mover at the agent orchestration layer. |

**Counter-framing for sellers:**
> "Policy engines like OPA and Sentinel are strong for runtime resource access. Platform Instructions works upstream — at the system prompt level, before the first token is generated. Use both together."

### HN/Reddit framing

**Do:** Frame as "the missing governance layer for production agents." Emphasize that governance that only works after the fact is governance that failed.

**Do NOT:** Overclaim compliance certifications. Do not compare directly to OPA/Sentinel — say "complements runtime policy engines," not "replaces them." Do not publish specific policy examples until PM confirms which are GA-ready.

---

## PART 3 — COMBINED NARRATIVE

### The Phase 34 headline story

**"Molecule agents come with built-in execution tracing and pre-execution governance — nothing to bolt on."**

These two features form a coherent governance + observability stack:

- **Platform Instructions** → governs what agents do *before* they run (proactive)
- **Tool Trace** → records what agents *actually did* after they run (reactive)

Together: *"governance before, observability after. Nothing leaves production unaccounted for."*

This is the Phase 34 headline theme. Use it in blog post intros, social copy, and launch announcement.

### Feature relationship diagram

```
Platform Instructions          Tool Trace
(pre-execution)               (post-execution)
      ↓                              ↓
Shapes agent behavior    ←  Records agent behavior
before first turn             after every task

Combined: proactive governance + complete execution record
```

### Phase 34 cross-sell (for sellers)

> "Phase 30 gave you per-workspace auth tokens (`mol_ws_*`). Phase 34 gives you platform-native observability (`Tool Trace`) and governance (`Platform Instructions`). Together: enterprise-ready from day one."

### Partner API Keys linkage

Issue #1831 does NOT cover Partner API Keys (that's Issue #1831's sister issue). But for combined Phase 34 copy, the full stack story is:

1. **Partner API Keys** → provision + manage orgs via API
2. **Platform Instructions** → govern agent behavior per tenant/org
3. **Tool Trace** → full observability per tenant/org
4. **SaaS Federation v2** → multi-tenant isolation

*Reference: "an early design partner" — no real partner name in copy until PM confirms.*

---

## PART 4 — COPY GUARDRAILS

### Required disclaimers

| Feature | Required disclaimer | When to use |
|---------|-------------------|-------------|
| Tool Trace | "Now in beta" | All external copy (HN, Reddit, LinkedIn, blog) |
| Platform Instructions | "Now in beta" | All external copy |
| Both | Do NOT say "GA," "generally available," or "production-ready by default" | Hard stop — do not use in any copy |

### Prohibited framings — DO NOT USE

| Prohibited framing | Why | Replace with |
|-------------------|-----|-------------|
| "Tool Trace is like Langfuse" | Not equivalent — different layer, different mechanism | "Built-in execution tracing, no SDK required" |
| "Platform Instructions replaces OPA/Sentinel" | Complementary, not competitive | "Works alongside runtime policy engines" |
| Any specific policy rule examples as GA-ready | PM not confirmed | "Enforce any workspace-level behavior rule at the system prompt level" |
| "Acme Corp" in published copy | Placeholder only | "an early design partner" |
| Rate limits or pricing for Partner API Keys | Not confirmed by PM | "Invite-only private access — apply at [partner contact]" |
| `mol_pk_test_*` sandbox keys | Post-GA only | Do not mention |
| "GA" or "generally available" for any Phase 34 feature | Not confirmed | "Now in beta," "shipping today," "now available" |
| "Message.metadata" or "JSONB" | Internal implementation detail | Never in external copy |

### Required framing — MUST USE

- Lead with the developer experience, not the feature name (say "execution tracing" not "Tool Trace" in headlines)
- Lead with "governance before agents run" not "admin control" for Platform Instructions
- Partner API Keys: lead with CI/CD ephemeral org lifecycle story as the hook (per PM copy guardrail)
- All Phase 34 external copy: include Phase 30 cross-link ("built on Phase 30 workspace isolation")

### Label guidance

| Feature | Label | Safe alternatives |
|---------|-------|------------------|
| Tool Trace | **BETA** | "now in beta," "shipping today," "in early access" |
| Platform Instructions | **BETA** | "now in beta," "shipping today," "in early access" |
| Partner API Keys | **INVITE-ONLY BETA** | "invite-only private access," "apply for early access" |
| SaaS Federation v2 | **BETA** (pending PM confirmation) | "now in beta" — do not publish until PM confirms scope and label |

---

## PART 5 — ASSET CHECKLIST

| Asset | Owner | Status | Notes |
|-------|-------|--------|-------|
| Issue #1831 PMM positioning brief (this doc) | PMM | ✅ READY | Cleared for Content Marketer + Social Media Brand use |
| Tool Trace blog post (`2026-04-23-tool-trace-observability/`) | Content Marketer | ✅ Staged | "AI Agent Observability Without the Overhead" — aligned with brief |
| Platform Instructions blog post (`2026-04-23-platform-instructions-governance/`) | Content Marketer | ✅ Staged | "Govern Your AI Fleet at the System Prompt Level" — aligned with brief |
| Phase 34 combined blog (`2026-04-23-tool-trace-platform-instructions/`) | Content Marketer | ✅ Staged | PR #1799 |
| Social copy (X + LinkedIn) | Social Media Brand | ⏳ Awaiting brief | This doc is the brief — ready to execute |
| Issue #1831 close | PMM | ✅ Ready to close | Positioning locked; blog posts verified aligned |

---

*Status: READY — gate cleared. Content Marketer and Social Media Brand may proceed.*
*PMM: close Issue #1831 after confirming Content Marketer has received this brief.*
