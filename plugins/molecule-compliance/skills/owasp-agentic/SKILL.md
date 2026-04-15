---
name: owasp-agentic
description: "Enforce OWASP Top 10 for Agentic Applications. Use when a workspace handles untrusted input (user messages, scraped web content, file uploads) or when it would be catastrophic if the agent ran away with unlimited tool calls. Gates prompt injection + excessive agency."
---

# OWASP Agentic Compliance

Opt-in compliance layer that wraps `builtin_tools/compliance.py`. The
Python primitives exist in every runtime image — installing this plugin
activates them via config and documents the policy.

## Coverage

| OWASP ID | Name | Primitive | Default mode |
|---|---|---|---|
| **OA-01** | Prompt Injection | `sanitize_input(text)` | `detect` |
| **OA-03** | Excessive Agency | `check_agency_limits(task_ctx)` | 50 calls / 300s |

## When to install

Install this plugin on any workspace that:
- Accepts free-form user input (chat interfaces, A2A message bodies)
- Scrapes or ingests untrusted web content
- Runs long-horizon tasks where a stuck loop could burn LLM budget
- Must satisfy compliance reviews that cite OWASP Top 10 for AI

## Configuration

Add to `config.yaml`:

```yaml
compliance:
  mode: owasp_agentic
  prompt_injection: detect        # detect → log+pass, block → raise PromptInjectionError
  max_tool_calls_per_task: 50     # OA-03 ceiling
  max_task_duration_seconds: 300  # OA-03 wall-clock ceiling
```

Modes explained:

- **`detect`** (default) — logs an audit event via `audit.log_event` when a
  trigger pattern is found, returns the original text. The agent still
  processes the input. Good for rollout: you see what triggers before
  committing to blocking.
- **`block`** — raises `PromptInjectionError` before the agent sees the
  text. The caller (typically `a2a_executor.py`) catches it and returns a
  400-shaped error to the sender.

## Trigger patterns (OA-01)

`sanitize_input` scans for:
- Instruction-override phrases ("ignore previous instructions", "new system prompt")
- Role-hijacking attempts ("you are now", "act as")
- System-prompt delimiter injection (`</s>`, `<|im_start|>`)
- Known jailbreak keywords (rotating list; update via compliance.py)

False positives on legitimate content are expected in `detect` mode —
that's why it's the default. Only flip to `block` after you've reviewed
audit logs for a week and confirmed the hit rate is low.

## Agency limits (OA-03)

Tracks per-task:
- Number of tool calls (`tool_call_count`)
- Elapsed wall-clock time (`started_at → now`)

When either exceeds the configured ceiling, `check_agency_limits` raises
`ExcessiveAgencyError`. The task terminates gracefully — the caller sees
a final message + `status=failed`.

## Anti-patterns

- **Don't** install on workspaces that only process trusted internal
  input — the overhead isn't worth it.
- **Don't** set `max_tool_calls_per_task` below 20. Many legitimate
  multi-step tasks need 15-30 tool calls; ceilings that low cause false
  terminations.
- **Don't** flip `prompt_injection` to `block` without a rollout period.
- **Don't** rely on this as your only defense — it's a cheap policy
  layer, not a substitute for proper sandboxing of the agent's
  filesystem + network access.

## Related

- `builtin_tools/compliance.py` — the implementation
- `molecule-audit` — audit-log retention for the events this plugin
  generates (OA-01 detections, OA-03 terminations). Install both to get
  a coherent compliance story.
- `molecule-security-scan` — pre-load CVE gate for skill dependencies
  (complements this runtime policy with supply-chain policy).
- Issue #256 — the proposal that led to this plugin split
