# Tool Trace + Platform Instructions — Social Copy
**Feature:** PR #1686 — Tool Trace + Platform Instructions  
**Merged:** 2026-04-23  
**Status:** DRAFT — ready for Social Media Brand to publish  
**Issue:** #1829

---

## X / Twitter Thread (5 tweets)

**Tweet 1 — Hook**
```
You can now see exactly what your agents did.

Every A2A call in Molecule AI now records a tool trace — tool name, input, output preview — for every tool your agents called.

No more guessing what happened in a multi-agent run. 🧵
```

**Tweet 2 — Tool Trace mechanics**
```
Here's what tool_trace looks like in a response:

{
  "tool_trace": [
    { "tool_name": "web_search",
      "input": {"query": "molecule ai"},
      "output_preview": "Molecule AI is..." },
    { "tool_name": "write_file",
      "input": {"path": "report.md"},
      "output_preview": "File written (412 bytes)" }
  ]
}

Parallel calls supported via run_id pairing. Capped at 200 entries.
```

**Tweet 3 — Platform Instructions**
```
Also shipped: Platform Instructions.

One API call sets system-level context for your entire org:

PUT /cp/platform-instructions
{ "instructions": "Tag every response with workspace ID." }

Every agent in your org inherits it. No touching individual workspace configs.
```

**Tweet 4 — Combined value**
```
Together: Platform Instructions sets what your agents know going in. Tool Trace proves what they did coming out.

Observability + control at the platform layer — not bolted on after the fact.
```

**Tweet 5 — CTA**
```
Tool Trace is live in every A2A response today.
Platform Instructions: PUT /cp/platform-instructions

Both ship as part of Phase 34 — Partner API Keys GA April 30.

Docs → https://docs.molecule.ai
```

---

## LinkedIn Post (~200 words)

**Two platform-level upgrades shipped in Molecule AI today.**

**Tool Trace** gives you full visibility into every tool call your agents make. Every A2A response now includes a `tool_trace` — the tool name, the input it received, and a preview of the output it returned. Parallel tool calls are tracked via `run_id` pairing, so concurrent agent activity doesn't get mixed up.

When something goes wrong in a production multi-agent workflow, you no longer have to reconstruct what happened from logs. The trace is in the response, stored in `activity_logs.tool_trace`, and queryable.

**Platform Instructions** lets org admins configure system-level context via a single API call. Set shared instructions once — every agent in your org inherits them. Useful for compliance requirements, house-style rules, or shared context that all your agents need without touching individual workspace configs.

Both features are live today. They're part of Phase 34 — which also includes Partner API Keys (GA April 30), the programmatic org provisioning API for platform builders and marketplace integrations.

If you're building on top of Molecule AI, Phase 34 is the release to watch.

→ https://docs.molecule.ai

---

## TTS Audio Script (15–20 sec)

> "Molecule AI just shipped Tool Trace — every A2A call now records what tools your agents used, the inputs they sent, and a preview of the output. Plus Platform Instructions: configure system-level context for your entire org via API. Full observability and control for your multi-agent stack."

---

## Publish Checklist
- [ ] Post X thread (Tweet 1 first, reply-thread the rest)
- [ ] Post LinkedIn version
- [ ] Generate and attach TTS audio clip
- [ ] Log post URLs here after publishing
- [ ] Close issue #1829

---

*Drafted by Marketing Lead 2026-04-23 — Social Media Brand to publish when workspace recovers.*
