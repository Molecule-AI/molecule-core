# Social Copy — Post: Heterogeneous Fleet Visibility Guide
> Guide: `docs/blog/2026-04-21-fleet-visibility/index.md` | Slug: `heterogeneous-agent-fleet-visibility`
> Platform: X (Twitter) thread + LinkedIn post | Status: READY

---

## X Thread — 4 posts

### Post 1 — Hook
> Your laptop agent, your AWS agent, and your on-prem agent — all on the same canvas.
> That's not a feature. That's how the platform normalizes runtime differences structurally.

### Post 2 — What the canvas actually shows
> Docker agent or remote agent — Canvas shows the same fields:
> — Status: online / degraded / offline
> — Current task + active task count
> — Error rate
> — Activity log (one schema, every runtime)
> The REMOTE badge is the only visual difference.

### Post 3 — Why it's architectural
> Cosmetic fleet visibility: query each runtime separately, merge in the browser.
> Problem: merge breaks when one runtime is slow. No single source of truth.
> Molecule AI's approach: every workspace implements the same /state contract.
> The platform stores the state. Canvas reads the state. Same schema everywhere.

### Post 4 — What the REMOTE badge signals
> The purple REMOTE badge means one thing: this agent needs the platform A2A proxy for inbound traffic.
> It doesn't mean less monitoring, less capability, or less security.
> It means the agent can't receive inbound connections — so the platform proxies.
> The activity log doesn't distinguish. The monitoring doesn't change.

---

## LinkedIn Post

**Heterogeneous fleet visibility sounds like a dashboard feature. It's not — it's architectural.**

The common approach to "seeing all your agents in one place" is a dashboard that queries each runtime separately and merges the results in the browser. Docker agents here, cloud agents there, on-prem agents somewhere else. The merge logic lives in the frontend.

That approach has two failure modes: the merge breaks when one runtime is slow or unreachable, and there's no single source of truth for what the fleet actually looks like right now.

Molecule AI's normalization is deeper. Every workspace — Docker or remote — implements the same state contract:

```
GET /workspaces/:id/state
→ {
    "workspace_id": "ws-abc",
    "status": "online",
    "current_task": "running research query",
    "active_tasks": 2,
    "error_rate": 0.0
}
```

This contract is the same regardless of where the agent runs. The platform stores the state. Canvas reads from the platform. If the platform is unreachable, Canvas shows stale data with a warning — but the logic doesn't branch based on runtime type.

What this means in practice: if your on-premises agent goes offline, you open Canvas. You see the last-seen timestamp, the error rate, and the activity log. You don't open a terminal on the server.

The heterogeneous fleet — Docker agents on the platform, remote agents on your laptop, cloud agents on AWS, on-prem agents in your data center — appears in one view, in one format, queryable by workspace, time range, or actor. No cross-referencing between separate runtime logs.

→ [Heterogeneous Fleet Visibility guide](https://moleculesai.app/blog/heterogeneous-agent-fleet-visibility)

---

## Platform Notes
- X thread: thread start at 9am PST
- LinkedIn: post same day at 10am PST
- CTA links: update before posting
- Pairs well with the Per-Workspace Bearer Tokens guide for a "security + visibility" content pairing