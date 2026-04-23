# PR #1686 Positioning Brief: Tool Trace + Platform Instructions

**Source:** PR #1686 — `feat: tool trace + platform instructions`
**Date:** 2026-04-23
**Author:** PMM
**Status:** Draft — for internal review before announcement

---

## Target Buyer

**Primary:** Platform Engineering / DevOps leads (80% of value)
**Secondary:** Enterprise IT / Security Governance leads (Platform Instructions)

Platform teams own the agent runtime and are the first to get paged when an agent goes off-script. They need built-in observability, not bolt-on stitching. Enterprise IT and compliance teams care about the governance angle — system-prompt rules that enforce behavior before an agent runs, not after it has already done something unintended.

---

## Primary Value Prop

> **Tool Trace** gives every A2A response a complete, run_id-paired execution record — so platform teams can trace what every agent actually did, without wiring up a third-party SDK.

> **Platform Instructions** lets workspace admins enforce system-prompt rules at startup — so governance happens before the agent runs, not after an incident.

---

## Competitive Angle

**vs. Langfuse / Helicone / separate observability pipelines:**
Third-party LLM observability tools require instrumentation in every agent: SDK installs, API key management, proxy configuration, and a separate vendor relationship. Tool Trace ships the execution record inside every A2A message and stores it in `activity_logs` — no extra pipeline, no separate pane of glass. For teams already on Molecule, it's zero-lift observability.

Langfuse/Helicone remain stronger for *cross-platform, multi-model* observability (tracking OpenAI + Anthropic + self-hosted in one view). That's not Molecule's fight. The positioning here is: "If you're already running agents on Molecule, you already have enterprise-grade trace — turn it on, don't integrate it."

**vs. Hermes native tool tracing:**
Hermes traces individual model calls. Tool Trace traces *agent behavior* — the A2A-level sequence of tool calls and responses across the full task lifecycle. Different layer of the stack. Tool Trace is additive, not competitive.

**vs. policy-as-code tools (OPA, Sentinel):**
Platform Instructions enforces behavioral guardrails at the system-prompt level. Policy engines enforce runtime resource access. They complement; Platform Instructions is earlier in the chain (pre-execution vs. during-execution).

---

## Key Differentiator

Tool Trace and Platform Instructions are **platform-native** — not plugins, not third-party SDKs, not configuration-as-code you have to maintain. They live where the agent runs: inside the workspace startup path and inside every A2A message envelope. There's nothing to install, no API key to rotate, no version drift to manage when the agent framework updates.

Third-party observability and governance tooling always has a lag between "agent framework ships a new behavior" and "our integration captures it." Native trace and prompt-level instructions have no lag — they are the platform.

---

## Objection Handlers

**O1: "We already use Datadog / Langfuse / Splunk for this."**
That's fine for cross-platform, multi-model environments. Tool Trace captures *A2A-level* agent behavior — tool calls, input/output previews, run_id-paired sequences — that generic LLM observability pipelines typically miss or flatten. Think of it as your Molecule-specific layer inside your existing observability stack. It doesn't replace Datadog; it enriches it.

**O2: "Why enforce system-prompt rules at the platform level instead of in code?"**
Because code changes require a deployment, and governance that requires a deployment is governance that only happens at the next release cycle. Platform Instructions are workspace-scoped rules that take effect at startup — a platform team or IT admin can update agent behavior without touching application code or triggering a redeploy. Speed of governance matters.

---

## Overlap / Conflict Notes

| Existing Feature | Relationship |
|-----------------|--------------|
| Org-scoped API keys (#1105) | Different layer: API key auth vs. agent behavior/prompt. Tool Trace traces what agents *do* with the keys; org keys control *who gets* the keys. Not cannibalization — complementary. |
| Audit trail visualization panel (#759) | Tool Trace is the raw execution record; the audit trail panel is the compliance UI on top of it. Tool Trace feeds the audit trail. Not competitive — dependency. |
| Snapshot secret scrubber (#977) | Both platform observability. Secret scrubber is about data posture; Tool Trace is about behavior. No conflict. |

**Cannibalization risk: LOW.** Tool Trace and Platform Instructions occupy the observability/governance vertical that existing features touch from different angles — no direct overlap, strong adjacency.

---

## CTA

**For platform teams:** "Enable activity log tracing for your workspace — every A2A task now has a complete execution record, no SDK required."
**For enterprise IT:** "Set workspace-level system prompt rules to enforce behavioral guardrails before agents run. No code deploy required."
**Combined anchor:** "Molecule gives you observability and governance as platform primitives — not afterthought integrations."

---

## Recommended Announcement Angle

Lead with the platform-native story, not the feature list. The headline is: *"Molecule agents now come with built-in execution tracing and governance — nothing to integrate."* Avoid leading with "Tool Trace" as a feature name in top-level copy; use "execution tracing" or "agent observability" for broader appeal.
