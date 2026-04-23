# Tool Trace — Positioning Brief
## Phase 34 | GA: April 30, 2026

> **Status:** READY — blocks Social Media Brand social copy | **Owner:** DevRel
> **Source:** PR #1686 (`molecule-core`) | **Conflicting PRs:** LangGraph A2A (open, moat intact — verified by PMM)
> **Conflicting narrative risk:** A2A v1 narrative (issue #1286) — pending Content Marketer assignment

---

## What It Is

Tool Trace is an observability layer built into every A2A response. Every tool call an agent makes — which tool, what inputs, what output preview — is captured and stored in `activity_logs.tool_trace` (JSONB + GIN index). Admins query via `GET /workspaces/:id/activity`.

The 200-entry cap per A2A turn prevents runaway-loop log bloat. Parallel tool calls are paired via shared `run_id`. Output previews are truncated at 200 chars.

---

## Positioning Statement

> *"Debug your agents the way you debug your code — with full instrumented traces, not LLM output guessing."*

Tool Trace is the observability layer for AI agents. It answers: "What did my agent actually do, and why did it do it?" — without re-running the conversation.

---

## Target Audiences (priority order)

| Audience | Use case | Angle |
|---|---|---|
| **Platform engineers / DevOps** | Production debugging, on-call | "Instrumented, not guessed" |
| **Developers building on Molecule AI** | Agent behavior verification | "See exactly what ran" |
| **Compliance / security teams** | Audit trails, regulatory requirements | "Every tool call logged, stored in your org" |
| **AI/ML engineers** | Agent evaluation, regression testing | "Trace = ground truth for what the agent did" |

---

## Competitive Differentiation

| Alternative | What they do | Tool Trace advantage |
|---|---|---|
| **LangChain callbacks** | Log LLM calls + tool calls to LangSmith/LangFuse | Tool Trace is built into the A2A protocol layer, not per-LLM; stored in your org's DB, not a third-party SaaS |
| **OpenTelemetry traces** | Distributed tracing for microservices | OTEL is for services, not agents — no concept of tool-level instrumentation of LLM tool calls |
| **Anthropic/OpenAI conversation logs** | LLM API logs | LLM logs only show prompts/responses, not the tool calls the agent makes |
| **Custom instrumentation** | Roll your own logging | Molecule AI has it on by default — no SDK changes required, no custom code |

**Key differentiator:** Tool Trace is protocol-native. It lives in the A2A response metadata and the org's activity_logs. No agent-code changes, no SDK additions, no third-party dependency.

---

## Messaging Pillars

1. **"Instrumented, not guessed"** — You see exactly which tools ran, in what order, with what inputs/outputs. No more reading LLM output to understand agent behavior.

2. **"Audit-ready by default"** — Every tool call is logged in your org's database. Stored in `activity_logs.tool_trace` JSONB. Queryable via API. Redaction is configurable.

3. **"Parallel tool calls handled correctly"** — `run_id` pairs start/end events across concurrent tool calls. Not merged into a confusing single timeline.

4. **"Built in, not bolted on"** — No SDK changes, no third-party SaaS, no custom instrumentation required. Tool Trace ships with every A2A response.

---

## Key Copy Angles (for social)

### Angle 1: Developer debugging (primary)
```
Your agent did something unexpected.

Tool Trace: see exactly which tools ran, with what inputs, what came back.

No more reading tea leaves in LLM output.
→ moleculesai.app/docs/tool-trace
```

### Angle 2: Compliance / audit
```
Regulators want to know: what did your AI agents do, and why?

Tool Trace: per-tool call history, input/output logging, stored in your org's activity_logs.

Audit-ready agent behavior records, built into the platform.
→ moleculesai.app/docs/tool-trace
```

### Angle 3: Platform engineer
```
Running AI agents in production means you're now an SRE for your agents.

Tool Trace gives you the observability to debug, audit, and improve — the same way you'd handle any production service.

Built into every A2A response. Stored in your org. Queryable via API.
→ moleculesai.app/docs/tool-trace
```

---

## Use Cases (for blog / longer-form content)

1. **Debug unexpected agent behavior** — Agent took a wrong action? Query the tool trace, see exactly which tool returned what output that led to the decision.

2. **Verify agent claims** — Agent says "I checked the config file." Tool trace shows `mcp__files__read` on `config.yaml` → output = `api_version: v2`. Verified.

3. **Audit trail for compliance** — SOC 2 / GDPR audit: show the full tool call sequence for any agent session.

4. **Agent regression testing** — Run agent with known inputs → compare tool trace to expected sequence. Catch behavioral regressions without LLM output comparison.

5. **Internal dashboards** — Query `activity_logs` for tool usage patterns across the org. See which tools are used most, which return errors, where latency lives.

---

## Objection Handlers

**"Doesn't this bloat every A2A response?"**
> The `tool_trace` is in `Message.metadata` and stored server-side in `activity_logs`. It's not sent to the LLM on every turn. The 200-entry cap prevents unbounded growth. This is not the same as appending verbose logs to every response.

**"We can just add LangChain callbacks for this."**
> LangChain callbacks work for LLM-level tracing. Tool Trace captures tool-level calls across any agent runtime — not just LLMs. It's protocol-native, not SDK-dependent.

**"Why not just use OpenTelemetry?"**
> OpenTelemetry is for distributed systems tracing. Tool Trace instruments AI agent tool calls — a fundamentally different signal. OTEL can't tell you which MCP tool an agent called with what arguments.

**"I don't need this for internal agents."**
> Every production workload needs debugging when things go wrong. Tool Trace turns "why did it do that?" from an LLM-output archaeology problem into a structured query.

---

## Content Status

| Asset | Status | Notes |
|---|---|---|
| Talk-track (this feature) | ✅ Done | `phase34-talk-track.md` |
| Social copy | ✅ Done | `phase34-social-copy.md` X-C1, X-C2 |
| Screencast storyboard | ✅ Done | PR #1878 |
| TTS narration | ✅ Done | `narration.txt` |
| **Positioning brief (this doc)** | ✅ Done | Blocks Social Media Brand |
| **Blog post (observability)** | ⏳ PMM blocked | In PR #1799 — PM owns |
| **X launch post** | ⏳ Brief done, awaiting Social Media Brand | Ready to execute |
| LinkedIn post | ⏳ Brief done, awaiting Social Media Brand | Ready to execute |

---

*Ready for Social Media Brand execution. GA: April 30, 2026.*