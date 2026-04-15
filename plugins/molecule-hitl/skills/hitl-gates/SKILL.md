---
name: hitl-gates
description: "Gate irreversible actions behind a human approval request. Use when an async callable (tool, method, or standalone function) performs a destructive or public action: deployment, deletion, outbound message, or issue/PR creation. Prevents unattended agents from shipping destructive work."
---

# HITL Gates

Human-in-the-loop gates for any async callable. Wraps the `@requires_approval`
decorator and `pause_task` / `resume_task` tools from
`builtin_tools/hitl.py`, which are already present in every runtime image.
This skill is the opt-in policy layer that tells an agent *when* to call
them — the Python implementation is always available; only workspaces that
install this plugin consult the policy.

## When to use a gate

Always, before any of these classes of action:

| Class | Examples |
|---|---|
| **Deployment** | `fly deploy`, `docker push`, kubectl apply, Vercel deploy |
| **Irreversible filesystem** | `rm -rf`, `git push --force`, DB `DROP TABLE`, `TRUNCATE` |
| **Public / external message** | Opening a GitHub issue or PR, posting to Slack, sending an email, posting on social media |
| **Production mutation** | Database migration against prod, secret rotation, cache invalidation that affects users |
| **Cross-workspace destructive** | Deleting another agent's memories, removing another workspace, cancelling another agent's delegations |

Reversible, scoped-to-self actions (editing local files, running tests,
reading documentation, saving memories to your own namespace) do **not**
need a gate.

## Usage — decorator form

For any async callable you own, wrap it in `@requires_approval`:

```python
from builtin_tools.hitl import requires_approval

@requires_approval(
    action="deploy_production",
    reason="Fly deploy to molecule-cp — affects all tenants",
    timeout=300,
    bypass_roles=["operator"],
)
async def deploy_fly_machine(app: str, image: str) -> dict:
    ...
```

What happens at call time:

1. The decorator fires `notify_humans(action, reason)` via the channels
   configured under `hitl:` in `config.yaml` (dashboard approval + optional
   Slack/email).
2. The caller's task is paused until a human clicks approve/deny or the
   `timeout` expires.
3. Timeout → rejected → raises `HITLRejectedError`. Caller handles it.
4. Approved → the wrapped function runs normally.
5. If the caller's role is in `bypass_roles`, the gate is skipped entirely
   (useful for an `operator` role that's already human-driven).

## Usage — explicit pause/resume

For cases where the decorator pattern is awkward (multi-step workflows
where the pause point is dynamic), use the pause/resume tools directly:

```python
from builtin_tools.hitl import pause_task, resume_task

task_id = await pause_task(
    task_id="deploy-abc",
    reason="About to run destructive migration 0042",
    timeout=600,
)
# External signal wakes us up:
#   - dashboard click
#   - another agent calling resume_task("deploy-abc", decision="approved")
#   - timeout → resumes with decision="timeout"
outcome = await resume_task(task_id)  # blocks until resolved
if outcome.decision != "approved":
    return {"status": "cancelled", "reason": outcome.decision}
```

## Configuration

Add to `config.yaml`:

```yaml
hitl:
  channels:
    - type: dashboard          # always on — uses the platform approval API
    - type: slack
      webhook_url: ${SLACK_HITL_WEBHOOK}
  default_timeout: 300         # seconds
  bypass_roles: [operator]     # roles that skip the gate entirely
```

Secrets referenced via `${ENV_VAR}` come from the workspace's secrets
store (set via `POST /workspaces/:id/secrets`).

## Anti-patterns

- **Don't** wrap read-only tools. A gate on `read_file` just annoys humans.
- **Don't** call `request_approval` from inside a cron tick — the human
  can't approve in time and the tick times out. Cron-fired actions should
  defer destructive steps to a follow-up task the human can approve.
- **Don't** rely on `molecule-careful-bash` + HITL together for the same
  action. HITL is the policy layer; careful-bash is the harness-level
  safety net. Pick one per call site or they double-prompt.
- **Don't** set a `timeout` shorter than ~60s. Humans need time to see the
  notification and context-switch.

## Test plan

1. Install this plugin on a workspace: `POST /workspaces/:id/plugins` with
   `{"source": "builtin://molecule-hitl"}`.
2. Configure `hitl.channels` + `bypass_roles` in the workspace's
   `config.yaml`.
3. Ask the agent to perform a gated action; verify a pending approval
   appears in `GET /approvals/pending`.
4. Approve via the canvas approval banner; verify the agent resumes and
   completes the action.
5. Deny via the canvas; verify the agent raises `HITLRejectedError` and
   responds with a graceful cancellation.

## Related

- `builtin_tools/hitl.py` — the implementation this plugin activates
- `builtin_tools/approval.py` — the lower-level approval store
- `molecule-careful-bash` — harness-level bash REFUSE list (complementary,
  not a replacement for HITL on non-bash actions)
- Issue #257 — the proposal that led to this plugin
