# Phase 34 — Tool Trace + Platform Instructions Social Copy
**Source:** PR #1686 merged to origin/main (2026-04-23)
**Features:** Tool Trace observability + Platform Instructions governance
**Blog posts:** `docs/blog/2026-04-23-tool-trace-observability/index.md`,
                 `docs/blog/2026-04-23-platform-instructions-governance/index.md`
**Status:** APPROVED — PMM reviewed (2026-04-23)
**Owner:** PMM → Social Media Brand | **Launch:** Coordinated with PR #1686 merge

---

## X (140–280 chars)

### Version A — Observability framing
```
Your agent says "I checked the logs." You don't know if it did.

Tool Trace: every A2A response now shows which tools were called,
with what inputs, and what they returned.

No sampling. No sidecar. Built in.
```

### Version B — Governance framing
```
Platform Instructions: write a policy once, enforce it everywhere.

Global or workspace-scoped rules injected into every agent's
system prompt at startup. Before the first turn. No code deploy.

Enterprise governance, shipped.
```

### Version C — Debugging angle
```
Parallel tool calls. Concurrent execution. One failed silently.

Tool Trace gives you run_id-paired traces so you can see
exactly which call failed and what it received.

Debug like it's a real system. Because it is.
```

### Version D — Combined / enterprise framing
```
Phase 34: two enterprise-grade features shipped together.

Tool Trace → see every tool your agent called.
Platform Instructions → govern what it can do before it does it.

Observability + policy. Built into the A2A response itself.
```

### Version E — Security / compliance angle
```
Combine Tool Trace with org-scoped API keys.

Every A2A call now traces back to: which org key → which workspace
→ which agent → which tools → what they returned.

Complete auditability. No guesswork.
```

---

## LinkedIn (100–200 words)

### Version A — Observability lead
```
Debugging a running AI agent in production is still, for most platforms, an exercise in inference. You get the final output. You don't get the tool calls that produced it.

Tool Trace changes that. Every A2A response from Molecule AI now includes a structured, chronological record of every tool the agent invoked — the tool name, the inputs passed to it, and a preview of the output. No sampling. No sidecar collector.

Parallel tool calls are handled correctly via run_id pairing. When the agent fires multiple tool calls simultaneously in a single turn, each entry carries the same run_id so you can trace them independently.

Tool Trace is enabled by default on all plans.

→ moleculesai.app/blog/ai-agent-observability-without-overhead
```

### Version B — Governance lead
```
The moment an AI agent goes into production, the governance question stops being theoretical. Which tools can it call? What data can it write to? Are there constraints that apply to every turn?

Most platforms answer these questions with post-hoc filtering — a rule that evaluates after the agent has already decided what to do. Platform Instructions takes a different approach: governance at the source, before the first token is generated.

Rules are prepended to the system prompt at workspace startup. The agent doesn't receive these rules as a filter — it receives them as part of its core instruction set. A filter can be worked around; a system prompt instruction shapes the agent's reasoning from the ground up.

Platform Instructions are available on Enterprise plans.

→ moleculesai.app/blog/govern-ai-fleet-system-prompt-level
```

### Version C — Combined / enterprise
```
Two questions enterprise teams ask first when evaluating an AI agent platform: what did the agent actually do? And can we enforce policy at the platform level?

Today Molecule AI ships both answers.

Tool Trace: every A2A response includes a structured record of every tool call — inputs, output previews, run_id-paired parallel traces. Built into the response. No sampling, no sidecar.

Platform Instructions: admin-configurable rules injected into every agent's system prompt at startup. Global or workspace-scoped. Before the first turn. Enterprise-only.

Together, they close the last gaps in agent observability and governance.

→ moleculesai.app/blog/tool-trace-platform-instructions
```

---

## Image suggestions per post

| Post | Best image |
|---|---|
| X Version A (Observability) | Terminal / API response showing `tool_trace` JSON in Message.metadata |
| X Version B (Governance) | System prompt screenshot showing `# Platform Instructions` as first section |
| X Version C (Debugging) | Before/after debug comparison: merged log vs run_id-paired traces |
| X Version D (Combined) | Composite: tool_trace JSON + system prompt governance section |
| X Version E (Security) | Audit chain diagram: org key → workspace → agent → tool calls |
| LinkedIn A | Tool trace JSON snippet (clean, formatted) |
| LinkedIn B | System prompt injection diagram (before/after prompt structure) |
| LinkedIn C | Combined feature diagram or composite |

---

## Screencast TTS script

**File:** `docs/devrel/demos/tool-trace-platform-instructions/narration.txt`
**Duration:** ~90 seconds, 5 moments
**Voice:** en-US-AriaNeural (or comparable neutral-professional voice)

**Status:** ✅ TTS script written and internally reviewed

---

## Hashtags

`#MoleculeAI` `#AIAgents` `#AgentObservability` `#EnterpriseSecurity` `#AIGovernance` `#A2A` `#DevOps` `#PlatformEngineering`

---

## UTMs

| Asset | UTM |
|---|---|
| Tool Trace blog | `?utm_source=linkedin&utm_medium=social&utm_campaign=phase34-tool-trace-launch` |
| Platform Instructions blog | `?utm_source=linkedin&utm_medium=social&utm_campaign=phase34-platform-instructions-launch` |
| Combined blog | `?utm_source=linkedin&utm_medium=social&utm_campaign=phase34-combined-launch` |
| Docs landing | `?utm_source=linkedin&utm_medium=social&utm_campaign=phase34-launch` |

---

## Publishing schedule (coordinated with Marketing Lead)

| Asset | Platform | Timing |
|---|---|---|
| Tool Trace TTS screencast | X + LinkedIn | Day 1 (2026-04-23) |
| X Version A (Observability) | X / Twitter | Day 1 afternoon |
| LinkedIn Version A (Observability) | LinkedIn | Day 2 morning |
| X Version B (Governance) | X / Twitter | Day 2 afternoon |
| LinkedIn Version B (Governance) | LinkedIn | Day 3 morning |
| X Version E (Security/audit) | X / Twitter | Day 4 (if engagement is strong) |
| LinkedIn Version C (Combined) | LinkedIn | Day 4–5 |

**Stagger posts by ≥4 hours. Anchor to blog publish time (TBD by Marketing Lead).**
