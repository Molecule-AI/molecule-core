# Talk Track — Tool Trace + Platform Instructions
**Title:** *Observability and Governance as Platform Primitives*
**Duration:** 5 minutes | **Format:** Conference / meetup / sales demo
**Source:** PR #1686 (`feat: tool trace + platform instructions`)
**Canonical positioning:** `docs/marketing/briefs/2026-04-23-pr1686-tool-trace-platform-instructions-positioning.md`

> **Assumptions:** Audience of 20–80 platform engineers or enterprise IT. Mixed experience with AI agents. 5-min slot, no live demo required — code traces and screenshots only.

---

## Pre-roll (0:00–0:10)

**Slide:** Full-bleed dark canvas screenshot — Molecule AI workspace graph with two agent nodes connected by an A2A arrow. Activity log panel open on the right showing tool call entries.

Narration:
> "Every agent platform has the same problem at scale: you can't actually see what your agents are doing."

**[Cue: advance on "doing."]**

---

## The Problem (0:10–0:50)

**Slide:** Two-column terminal comparison — left: verbose curl + manual grep; right: clean Agent Activity Report output.

Narration:
> "You have twenty agents running in production. One of them made a call you can't explain. That usually comes up the week before a compliance review."
>
> "Most agent platforms give you two options: bolt on a third-party observability SDK, or fly blind. Bolt-on means instrumentation in every agent, API key management, proxy config, and a vendor relationship you have to maintain. And by the time you finish integrating it, your agents have already shipped."

**[Cue: "already shipped" — advance.]**

---

## Tool Trace — Zero-Lift Observability (0:50–2:10)

**Slide:** Code block — simplified A2A response metadata excerpt with `tool_trace` array. Three entries: web_search, summarize_text, write_to_file. Each entry shows `{tool, input[:500], output_preview[:300], run_id}`.

Narration:
> "Tool Trace ships inside every A2A response your agents generate — no SDK, no pipeline, no separate pane of glass."
>
> "It's a list of every tool the agent called: the tool name, the input — sanitized, truncated at 500 characters — and a 300-character output preview. No secrets, just enough to verify what happened."
>
> "LangGraph agents can run multiple tools at the same time. Here's where run-ID becomes critical. When two tools fire concurrently, the platform records both start events before either end event. Without run-ID, the output previews would overwrite each other in a simple list. The run-ID key pairs each output back to its matching tool entry."
>
> "The platform stores the trace in `activity_logs` — a JSONB column with a GIN index, queryable via the activity endpoint. You can pull it in real time, or stream it into your SIEM for long-term audit storage."
>
> "If you're already running on Molecule, you already have this — it's on by default."

**[Cue: "on by default" — advance.]**

---

## Platform Instructions — Pre-Execution Governance (2:10–3:30)

**Slide:** Canvas screenshot — Platform Instructions editor panel. Three rules visible: "no file writes outside /workspace", "require approval before git push", "mask PII in tool outputs". Rule status indicators green.

Narration:
> "Observability tells you what an agent did after it happened. What if you could govern what it does *before* it runs?"
>
> "Platform Instructions are workspace-level system prompt rules set by an admin — they take effect at workspace startup, before any agent prompt is processed."
>
> "A platform team or IT admin can enforce behavioral guardrails without touching application code or triggering a redeploy. And because they live in the platform startup path, there's no version drift when your agent framework updates."
>
> "Think of it as governance at the speed of platform configuration — not the speed of code deployment."

**[Cue: "code deployment" — advance.]**

---

## How They Work Together (3:30–4:10)

**Slide:** Architecture diagram — two agent nodes + Molecule AI Platform box. Left panel: Platform Instructions arrow into agent startup path. Right panel: Tool Trace arrow out of agent task completion into activity log.

Narration:
> "Tool Trace and Platform Instructions are two sides of the same primitive: platform-native observability and governance."
>
> "Platform Instructions govern behavior before the agent runs. Tool Trace records what actually happened after. Together they give you a complete audit chain — pre-execution policy, post-execution trace — without a single third-party integration."
>
> "That's the platform-native story: not bolt-on stitching, not afterthought integration. Built in from the start."

**[Cue: "from the start" — advance.]**

---

## Close + CTA (4:10–5:00)

**Slide:** Dark card — two-line headline, two bullet CTAs.

  **Headline:**
  > *Molecule agents come with built-in execution tracing and governance — nothing to integrate.*

  **CTAs:**
  > Enable activity log tracing → every A2A task now has a complete execution record.
  > Set workspace-level Platform Instructions → governance without a code deploy.

Narration:
> "Tool Trace and Platform Instructions are available now on all Molecule AI deployments."
>
> "If you're evaluating agent platforms, this is the difference worth measuring: not just what your agents can do, but whether you can actually see and govern it."
>
> "Molecule AI — observability and governance as platform primitives."

**[Cue: last word — hold 2 seconds — lights up Q&A.]**

---

## Key Numbers to Land (memorize)

| Fact | Value |
|------|-------|
| Input truncation | 500 chars |
| Output preview | 300 chars |
| Max trace entries | 200 (cap prevents runaway loops) |
| Activity log storage | JSONB + GIN index |
| Instruction max size | 8 KB (DB CHECK constraint) |
| Default session TTL | 3600s (1 hour) |

## Objection Matrix

| Objection | Response |
|-----------|----------|
| "We already use Datadog / Langfuse." | Good for cross-platform multi-model. Tool Trace captures A2A-level agent behavior those pipelines miss or flatten. Think of it as your Molecule-specific enrichment layer. |
| "Why not just write governance in code?" | Code changes require a deployment. Platform Instructions take effect at workspace startup — a platform team or IT admin can update behavior without touching application code or triggering a redeploy. |
| "Is this on by default?" | Yes. Tool Trace is on for every A2A response. Platform Instructions requires an admin to define rules — off by default, but trivially enabled per workspace. |

## Competitive Frame

| Competitor | Angle |
|-----------|-------|
| Langfuse / Helicone | Bolt-on SDK, separate pipeline, version drift risk. Tool Trace: zero-lift, platform-native. |
| Hermes native tool tracing | Hermes traces model calls. Tool Trace traces agent *behavior* — A2A-level tool call sequences. Additive, not competitive. |
| OPA / Sentinel | Policy engines enforce runtime resource access. Platform Instructions enforces at the system-prompt level — pre-execution, earlier in the chain. Complementary. |
| Roll-your-own audit logging | Custom code must be maintained across agent framework updates. Platform-native trace has no version drift. |

## Files

- Canonical positioning: `docs/marketing/briefs/2026-04-23-pr1686-tool-trace-platform-instructions-positioning.md`
- Demo package: `docs/marketing/devrel/demos/tool-trace-demo/`
- PR: `github.com/Molecule-AI/molecule-core/pull/1686`
- Phase 34 launch brief: `docs/seo/pipeline15-phase34-mcp-briefs.md`
