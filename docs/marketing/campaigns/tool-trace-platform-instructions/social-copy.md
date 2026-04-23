# Tool Trace + Platform Instructions — Social Copy
**Campaign:** Tool Trace + Platform Instructions | **Demo:** `docs/marketing/devrel/demos/tool-trace-platform-instructions-demo.md`
**Source:** PR #1686 (merged 2026-04-23)
**Owner:** DevRel → PMM → Social Media Brand | **Launch:** Coordinated with PR #1686 merge
**Status:** DRAFT — pending PMM review

---

## X (140–280 chars)

### Version A — Platform operator framing
```
Platform operators: you can now see exactly what every agent did.

Tool trace: every tool call — bash, file reads, web search — captured with input and output preview.

Plus: inject rules into every agent's system prompt at startup. No template edits. No deploys.
```

### Version B — Observability angle
```
What did your AI agents actually do last night?

Tool trace answers that — stored in activity logs, GIN-indexed, queryable.

Plus Platform Instructions: inject operator rules at startup, see them in every run.

Observability + control, built in.
```

### Version C — Security/audit framing
```
Every bash call. Every file read. Every web search.

Tool trace captures it all — input, output preview, paired by run ID. Stored. Queryable.

Platform Instructions let you inject guardrails before the agent starts.

Your security team called.
```

### Version D — Developer angle
```
Two new platform primitives in Molecule AI:

1. Tool Trace — every tool call logged with input/output preview
2. Platform Instructions — inject rules into agent system prompts at startup

Both ship today. Check the demo.
```

---

## LinkedIn

### Version A
```
Every AI agent is a black box — until now.

Molecule AI just shipped two platform-level observability features:

**Tool Trace** captures every tool call an agent makes — bash, file reads, web searches — stored in activity logs with input and output preview. Parallel calls are paired via run ID. GIN-indexed for fast cross-agent queries.

**Platform Instructions** lets platform operators inject rules into every agent's system prompt at startup — globally or per workspace — without editing templates or coordinating deploys.

Together: you can now see exactly what agents did, and guide what they should do next.

Demo linked in comments.
```

### Version B
```
Platform teams: if you're running AI agents in production and you can't answer "what did it just do?" — that's a governance gap.

Molecule AI's new Tool Trace feature makes every agent run auditable. Every tool call, its input, and a short output preview — stored in activity_logs.tool_trace, queryable across your entire fleet.

And if you need to inject guardrails — cost limits, compliance rules, style guidelines — Platform Instructions lets you do that at startup, without touching a single template file.

Observability and control, not bolted on after the fact.

Demo + API reference in comments.
```

---

## Reddit (r/LocalLLaMA / r/ClaudeAI / r/aiassistance)

### Version
```
Molecule AI just shipped two features that solve real platform-scale problems:

**Tool Trace** — every tool call (bash, read_file, web search, etc.) an agent makes is logged with input and a 300-char output preview, stored in activity_logs. Parallel calls are paired by run ID. Capped at 200 entries per run. Fast GIN-indexed queries across your whole fleet.

**Platform Instructions** — inject rules into every agent's system prompt at startup. Global scope (all workspaces) or per-workspace. No template edits, no deploys. Just POST to /instructions.

Both are now in production. Demo link below.
```

---

## Hook / Discussion prompt (internal)
```
Pitch: "We just shipped Tool Trace + Platform Instructions. These are the two features that make Molecule AI viable for regulated/enterprise environments — full audit trail + operator control. 

Would love to do a short thread on the observability story. Can we get a quote from the platform team on what's coming next (token expiry, role scoping)?"
```
