---
name: llm-judge
description: Evaluate whether a Molecule AI agent's output (a PR, a delegation result, a generated config) actually addresses the original request. Cheap LLM-as-judge gate that catches "wrong answer to right question" — the failure mode unit tests miss. Inspired by gstack's tier-3 LLM-as-judge test infra.
---

# llm-judge

Unit tests verify the code RAN. They don't verify it did the RIGHT THING for the customer's actual request. This skill closes that gap.

## When to invoke

After a Molecule AI agent (PM, Dev Lead, QA, etc.) produces a deliverable:
- A PR they opened in response to an issue
- A delegation result (response to an A2A `message/send`)
- A generated config or template
- A code review they posted

Specifically: when a worker agent comes back with "done", before we believe them.

## Inputs

1. The ORIGINAL request — the issue body, the user message, the delegation prompt
2. The DELIVERABLE — the diff, the response text, the generated artifact
3. ACCEPTANCE CRITERIA if explicit (often in the issue body)

## How to evaluate

Send to a small fast model (Haiku, GPT-mini, Gemini Flash):

```
You are an evaluator. Below is a customer request and the deliverable
the AI agent produced. Rate, on a 0-5 scale, how well the deliverable
addresses the original request. Then list the top 3 reasons for the score.

REQUEST:
<paste original>

DELIVERABLE:
<paste artifact>

ACCEPTANCE CRITERIA (if any):
<paste>

Output JSON:
{
  "score": 0..5,
  "addresses_request": true|false,
  "missing": ["...", "..."],
  "wrong": ["...", "..."],
  "reasons": ["...", "...", "..."]
}
```

## Decision

| Score | Action |
|---|---|
| 5 | Accept — log to telemetry |
| 4 | Accept with note — file a follow-up issue for the gap if material |
| 3 | Send back to the agent for revision with the judge's "missing" list |
| 0–2 | Reject. Escalate to CEO. Likely the agent misunderstood the task — fixing the prompt > fixing the deliverable |

## Cost

Tier-3 (Haiku-class): ~$0.001 per eval. Even at 100 evals/day = $0.10/day. Negligible.

## Where to plug it in

- **Cron Step 4 (issue pickup)**: after a draft PR is opened by a subagent, run llm-judge against the issue body. Mark the PR ready ONLY if score >= 4.
- **A2A delegation in workspaces**: optionally enable per-org. PM gets the worker's response, runs llm-judge, only forwards to the next stage if accepted.
- **Manual**: `npm run skill:llm-judge -- --request <file> --deliverable <file>`

## Why this exists

gstack runs LLM-as-judge as a test-tier ($0.15 per eval, ~30s). Our worker agents produce many more deliverables per day than gstack's single-session model — making the eval cheaper and more frequent matches our scale. The failure mode this catches — "agent shipped the wrong thing" — is invisible to unit tests AND to code-review skills (both verify the code, not the intent).
