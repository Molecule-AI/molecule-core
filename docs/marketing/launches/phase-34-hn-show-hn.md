# Phase 34 — Hacker News Show HN Post

**Target:** news.ycombinator.com  
**Publish:** April 30, 2026 (GA day)  
**Status:** APPROVED — Marketing Lead 2026-04-23

---

## Title

> Show HN: Molecule AI – every agent tool call now logged in A2A response (no SDK, GA today)

---

## Body (~150 words)

We shipped Phase 34 today. The two things most relevant to HN:

**Tool Trace**: every A2A response now includes a structured `tool_trace` in `Message.metadata` — tool name, input, output preview, `run_id` for concurrent calls. Zero config, no sidecar, no sampling. In every response by default. Stored as JSONB, queryable. workspace-server is open source: https://github.com/Molecule-AI/molecule-core

**Partner API Keys (`mol_pk_*`)**: programmatic org provisioning. `POST /cp/admin/partner-keys` creates an isolated Molecule AI org. `DELETE` tears it down and stops billing. Built for CI/CD ephemeral-org-per-PR patterns and marketplace resellers.

Also shipping: Platform Instructions (behavioral rules injected into agent system prompt via API, no code deploy needed) and SaaS Federation v2.

Changelog: https://docs.molecule.ai/changelog/phase-34

---

## Pre-brief: Anticipated HN objections

**"How is this different from Langfuse / Helicone?"**  
Tool Trace is A2A-native and zero-config — it captures the agent-behavior layer (which tools were called, in what order, with what inputs) in every response, without instrumentation. Langfuse and Helicone are cross-platform LLM observability pipelines — stronger for multi-model, multi-provider environments. They're complementary. Tool Trace doesn't replace Langfuse; it enriches it with the A2A-specific layer Langfuse typically misses.

**"Is this open source?"**  
workspace-server is MIT-licensed at github.com/Molecule-AI/molecule-core. The controlplane (which handles billing, org provisioning, and partner keys) is proprietary.

**"What's the pricing?"**  
Tool Trace: all plans, no extra cost.  
Platform Instructions: all plans, no extra cost.  
Partner API Keys: GA today — pricing details at docs.molecule.ai/api/partner-keys. Rate limits are being confirmed post-launch.

**"Why build this instead of just using OpenTelemetry?"**  
OpenTelemetry traces the infrastructure layer. Tool Trace traces agent *behavior* — the sequence of tool decisions an agent made during a task. Different layer, different granularity. You can export Tool Trace data into an OpenTelemetry pipeline if you want.

**"Show the code"**  
`workspace-server/internal/handlers/instructions.go` — Platform Instructions handler (no tier gate, all plans confirmed).  
`Message.metadata.tool_trace` — in every A2A response from today.

---

*Marketing review note: HN audience is skeptical and technical. No hype. Lead with the open-source link. Acknowledge what's proprietary. Be specific about what's confirmed vs. TBD.*
