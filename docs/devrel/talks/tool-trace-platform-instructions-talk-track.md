# Talk Track: Tool Trace + Platform Instructions
**Format:** 5-minute conference talk / DevRel livestream  
**Audience:** Platform engineers, DevOps leads, enterprise IT / compliance  
**Features:** Tool Trace (Phase 34) + Platform Instructions (Phase 34)  
**Last updated:** 2026-04-23

---

## Pre-talk checklist

- [ ] Terminal open, `$MOL_WS_TOKEN` exported
- [ ] Browser tab open: `https://docs.molecule.ai/platform/tool-trace`
- [ ] Code snippets from the appendix pre-loaded in editor or notes
- [ ] Slide deck advanced to title slide
- [ ] Chat window ready to paste the curl snippet at CTA

---

## [0:00–0:30] Hook — "Your agent ran for 20 minutes. Here's everything it did."

**[SLIDE: Title — "Agent Observability Built In"]**

> Your agent finishes. You get the output. Maybe it looks right.
>
> But do you actually know what it did? Which tools it called? What it sent to the web search API? Whether it wrote to a file it shouldn't have touched?
>
> Most platforms give you the output. That's it. You're left reverse-engineering a 20-minute run from the final answer.

**[PAUSE — let that land]**

> Two new features in Molecule's Phase 34 close that gap.
>
> **Tool Trace** gives you the full execution record — every tool call, every input, every output preview — and it's in every single A2A response with zero setup.
>
> **Platform Instructions** lets you set behavioral rules for every agent in your org before any of that runs — so you're not just auditing after the fact, you're governing up front.
>
> Let me show you both in five minutes.

**[SLIDE TRANSITION: "Part 1: Tool Trace"]**

---

## [0:30–2:30] Tool Trace — live demo walkthrough

### [0:30–1:00] Show the response structure

**[SCREEN: Switch to editor / terminal — show JSON Snippet A]**

> Here's what a Molecule A2A response looks like now.

```json
{
  "metadata": {
    "tool_trace": [
      {
        "tool_name": "web_search",
        "input": { "query": "molecule ai agent platform benchmarks" },
        "output_preview": "Molecule AI ranked #1 in agent coordination latency..."
      },
      {
        "tool_name": "write_file",
        "input": { "path": "research/benchmarks.md", "content": "..." },
        "output_preview": "File written successfully (2,847 bytes)"
      },
      {
        "tool_name": "bash",
        "input": { "command": "python analyze.py research/benchmarks.md" },
        "output_preview": "Analysis complete. 3 insights extracted."
      }
    ]
  }
}
```

> This is `message.metadata.tool_trace`. It's a chronological list of every tool the agent called during that task.
>
> `tool_name` — what it called.  
> `input` — exactly what it sent.  
> `output_preview` — the first chunk of what came back.
>
> Three tools. A web search, a file write, a bash execution. You can see the query. You can see the file path. You can see what the script returned. In the response envelope, right now, for every agent on every plan.

**[SPEAKER NOTE: Point at each field while naming it. Slow down on `input` — that's the field people most want to see.]**

> And here's the key thing I want you to hear: **this is in every A2A response. No SDK to install. No extra API calls. No observability pipeline to configure. You already have it.**

### [1:00–1:30] Show parallel call handling with run_id

**[SCREEN: Show JSON Snippet B]**

> For agents that run tools in parallel — and Molecule agents do, they'll fire multiple MCP tools concurrently — you need a way to pair each call's start event with its end event when they interleave.

```json
{
  "tool_name": "grep",
  "input": { "pattern": "TODO", "path": "src/" },
  "output_preview": "47 matches found across 12 files",
  "run_id": "a3f9b2c1"
}
```

> That's what `run_id` is for. Every concurrent call gets a unique run_id. The trace engine pairs start and end by that ID, so concurrent calls don't get scrambled in the trace.
>
> If you're running agents against large codebases — parallelizing searches across directories, for example — this is what keeps the trace readable.

### [1:30–2:00] Stored and queryable

**[SCREEN: Show SQL Snippet / mental model — Snippet C]**

> The trace doesn't disappear when the HTTP response closes. It's persisted to the `activity_logs` table in a JSONB column called `tool_trace`.

```sql
-- Example: find every bash call this week that touched production
SELECT
  id,
  created_at,
  tool_trace
FROM activity_logs
WHERE tool_trace @> '[{"tool_name": "bash"}]'
  AND created_at > now() - interval '7 days';
```

> JSONB — so you can query inside the array. Filter by `tool_name`, inspect inputs, build audit reports, pipe into your SIEM. The trace is capped at 200 entries per response to keep log size predictable.
>
> This is the difference between a five-minute diagnosis and a two-hour investigation when something goes wrong in a multi-agent workflow.

**[SLIDE TRANSITION: "Part 2: Platform Instructions"]**

---

## [1:30–3:00] Platform Instructions — live demo

### [2:00–2:30] The curl — set org-wide instructions

**[SCREEN: Switch to terminal — live curl, or show Snippet D]**

> Now let's talk about the governance side. Platform Instructions lets a workspace admin set system-level rules that apply to every agent in the org. One API call.

```bash
curl -X PUT https://api.molecule.ai/cp/platform-instructions \
  -H "Authorization: Bearer $MOL_WS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "instructions": "Always respond in English. Tag every response with the originating workspace ID. Do not execute destructive operations (DELETE, DROP, rm -rf) without explicit confirmation."
  }'
```

> Three rules: language, tagging, and a safety constraint on destructive ops. One PUT. Done.
>
> Now every agent turn in this org has those rules baked into its system prompt before it reasons about anything.

**[SPEAKER NOTE: If running live, actually execute the curl and show the 200 response. If not, show the pre-baked response from Snippet D2.]**

### [2:30–3:00] The key framing — this is not a filter

> Here's the framing I want you to take away from this.
>
> **This isn't a filter. It's part of what the agent IS when it starts.** The instructions land in the system prompt before the first token of reasoning. There's no lag between "I updated the rule" and "the rule is in effect." The next agent turn runs with the new instruction. No deployment cycle required.

**[SLIDE: "Platform Instructions vs. Policy-as-Code"]**

> Some of you are thinking: "we already have OPA" or "we use Sentinel for this."
>
> Great — keep them. They solve a different problem. OPA and Sentinel enforce runtime resource access: what APIs the agent can call, what it's allowed to touch. Platform Instructions enforces *behavioral guardrails* — what the agent is instructed to do before it even starts deciding.
>
> They're complementary. Platform Instructions is earlier in the chain. Pre-execution, not mid-execution.
>
> For compliance teams, that distinction matters a lot. A rule that lives in the system prompt can't be bypassed by a misconfigured policy engine. It's in the prompt.

**[SLIDE TRANSITION: "The Loop"]**

---

## [3:00–3:30] The combined governance loop

**[SLIDE: Diagram — Platform Instructions → Agent Turn → Tool Trace]**

```
[Platform Instructions]  →  agent turn  →  [Tool Trace]
  "no destructive ops"                      "bash: rm -rf → blocked, 0 bytes"
  "tag with workspace ID"                   "write_file: tagged ✓"
```

> This is one slide and it's the whole point.
>
> **Platform Instructions** sets what your agent knows and is instructed to do going in.  
> **Tool Trace** proves what it actually did coming out.
>
> Write the policy once. Enforce it everywhere. Trace every execution.
>
> For teams managing agent fleets at scale — especially in compliance-sensitive environments — this is the observability and governance stack that used to require integrating three separate tools. It now ships as part of the platform.

**[SLIDE TRANSITION: "Get Started"]**

---

## [3:30–4:30] CTA + Q&A setup

**[SLIDE: "Get started today"]**

> Two things you can do in the next ten minutes.
>
> **Tool Trace:** zero config. It's already in every A2A response. Open your next agent run, look at `message.metadata.tool_trace`, and the trace is there.
>
> **Platform Instructions:** one PUT call. I'm dropping the curl snippet in chat right now.

**[ACTION: Paste Snippet D into chat/Discord/Slack]**

> If you're running agents in a compliance-sensitive environment, set a baseline instruction today. Takes thirty seconds. Applies to everything.
>
> Quick note on scope: Platform Instructions supports both global — org-wide, every workspace — and workspace-scoped overrides, so individual teams can layer their own context on top of the org baseline without conflicting with it.
>
> Both features are live as part of Phase 34. Docs links are in the slides. Happy to take questions.

**[SLIDE: Q&A — docs.molecule.ai/platform/tool-trace | docs.molecule.ai/platform/platform-instructions]**

> What are you building with agents that this would most change? Let's talk.

---

## Objection handling (speaker prep — not scripted)

**"We already use Langfuse / Datadog / Splunk for this."**  
Tool Trace captures A2A-level agent behavior — tool call sequences, input/output previews, run_id-paired parallel calls — at a layer that generic LLM observability pipelines typically miss or flatten. Think of it as your Molecule-specific layer inside your existing stack, not a replacement for it. It enriches Datadog; it doesn't replace it.

**"Why not just instrument this ourselves?"**  
You can. But instrumentation has version drift — every time the agent framework ships a new behavior, your instrumentation has to catch up. Tool Trace is native to the platform. It has no lag and no version drift to manage.

**"Why enforce system-prompt rules at the platform level instead of in application code?"**  
Code changes require a deployment. Governance that requires a deployment is governance that only happens at the next release cycle. Platform Instructions take effect at startup — an IT admin can update agent behavior without touching application code or triggering a redeploy. Speed of governance matters.

**"Can individual teams override Platform Instructions?"**  
Yes — workspace-scoped instructions are additive on top of the global baseline. You set the floor org-wide, teams add their own context. No team can remove the global rules; they can only add to them.

---

## Appendix: Code snippets to have ready

Paste these into your editor or notes app before going live. All examples are safe to show publicly — no real tokens.

---

### Snippet A — Tool Trace: basic A2A response

```json
{
  "metadata": {
    "tool_trace": [
      {
        "tool_name": "web_search",
        "input": { "query": "molecule ai agent platform benchmarks" },
        "output_preview": "Molecule AI ranked #1 in agent coordination latency..."
      },
      {
        "tool_name": "write_file",
        "input": { "path": "research/benchmarks.md", "content": "..." },
        "output_preview": "File written successfully (2,847 bytes)"
      },
      {
        "tool_name": "bash",
        "input": { "command": "python analyze.py research/benchmarks.md" },
        "output_preview": "Analysis complete. 3 insights extracted."
      }
    ]
  }
}
```

---

### Snippet B — Tool Trace: parallel call with run_id

```json
{
  "tool_name": "grep",
  "input": { "pattern": "TODO", "path": "src/" },
  "output_preview": "47 matches found across 12 files",
  "run_id": "a3f9b2c1"
}
```

---

### Snippet C — activity_logs query (mental model)

```sql
-- Find every bash call this week that touched any path
SELECT
  id,
  created_at,
  tool_trace
FROM activity_logs
WHERE tool_trace @> '[{"tool_name": "bash"}]'
  AND created_at > now() - interval '7 days';
```

```sql
-- Find every write_file call to paths under /prod
SELECT
  id,
  created_at,
  entry->>'tool_name'     AS tool,
  entry->'input'->>'path' AS file_path
FROM activity_logs,
  jsonb_array_elements(tool_trace) AS entry
WHERE entry->>'tool_name' = 'write_file'
  AND entry->'input'->>'path' LIKE '/prod/%';
```

---

### Snippet D — Platform Instructions: PUT (set)

```bash
curl -X PUT https://api.molecule.ai/cp/platform-instructions \
  -H "Authorization: Bearer $MOL_WS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "instructions": "Always respond in English. Tag every response with the originating workspace ID. Do not execute destructive operations (DELETE, DROP, rm -rf) without explicit confirmation."
  }'
```

---

### Snippet D2 — Platform Instructions: expected 200 response

```json
{
  "status": "ok",
  "scope": "global",
  "updated_at": "2026-04-23T14:00:00Z"
}
```

---

### Snippet E — Platform Instructions: GET (read current)

```bash
curl https://api.molecule.ai/cp/platform-instructions \
  -H "Authorization: Bearer $MOL_WS_TOKEN"
```

---

### Snippet F — Combined diagram (for whiteboard or slide)

```
[Platform Instructions]  →  agent turn  →  [Tool Trace]
  "no destructive ops"                      "bash: rm -rf → blocked, 0 bytes"
  "tag with workspace ID"                   "write_file: tagged ✓"
  "English only"                            "response_language: en ✓"
```

---

## Docs + links (for slides / chat)

- Tool Trace docs: `https://docs.molecule.ai/platform/tool-trace`
- Platform Instructions docs: `https://docs.molecule.ai/platform/platform-instructions`
- Phase 34 release notes: `https://docs.molecule.ai/changelog/phase-34`
- Blog post: `https://molecule.ai/blog/agent-observability-tool-trace-platform-instructions`
