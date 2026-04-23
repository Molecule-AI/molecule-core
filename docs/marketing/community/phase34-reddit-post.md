# Phase 34 Launch — Reddit r/MachineLearning Post

**Suggested title:** `Tool Trace is the feature I didn't know I needed until I used it`

**Self-post body:**

---

I've been building with [Molecule](https://github.com/Molecule-AI/molecule) for the past few months, and Phase 34 just shipped — wanted to share what's actually useful rather than what's marketable.

**The feature I'm writing about: Tool Trace**

When an agent decides to call a tool in Molecule, the reasoning behind that call is now surfaced in `Message.metadata.tool_trace[]`. It's not a dashboard. It's not a UI. It's a structured array of `{tool, reasoning, input, output}` objects you can log, inspect, and build on.

Here's what the output looks like in practice:

```python
message = await session.send("Deploy my workspace to staging")
print(message.metadata.tool_trace)
# [
#   {
#     "tool": "workspace_deploy",
#     "reasoning": "User said 'deploy to staging'. Checking env var for target.",
#     "input": {"env": "staging"},
#     "output": {"status": "ok", "deployment_id": "dep_8xf92"}
#   },
#   {
#     "tool": "notify_slack",
#     "reasoning": "Deploy succeeded. Sending confirmation to #eng-alerts.",
#     "input": {"channel": "#eng-alerts", "msg": "staging deploy done"},
#     "output": {"ts": "1776960000"}
#   }
# ]
```

This is alpha — `tool_trace` is present when the model emits it, which means quality depends on the model. Larger models like Claude Sonnet 4 do it consistently. Smaller models may be inconsistent. That's honest.

**What Platform Instructions does (the governance layer)**

If you're running a multi-tenant deployment, Platform Instructions lets you push org-wide defaults to every agent in your org:

```json
{
  "type": "instruction",
  "instruction": "Always tag resources with cost_center before provisioning.",
  "priority": "required"
}
```

Agents inherit these at session start. It's not magic — it's a structured override that lets you enforce guardrails without patching agent prompts every release.

**Partner API Keys (mol_pk_*)**

Scoped tokens for programmatic org provisioning. If you're building a product on top of Molecule's API, you can now provision workspaces under your org with a token scoped to `provision:write` only — no admin access needed. The keys support tiered rate limits (9/29/99 USD flat-rate tiers, details on the pricing page).

**SaaS Federation v2**

Multi-tenant control plane. You can now issue agent identities that cross org boundaries — think of it as a trust mesh between partner orgs where agents can delegate to each other under explicit policy. This one's genuinely new and the docs are the best place to understand the trust model.

**What to try first**

If you're evaluating Molecule for agentic workflows, I'd start with the Tool Trace — it's the most immediately actionable. Log a session, look at the trace, and ask yourself whether the reasoning chain matches what you'd expect a good developer to think. If it does, the rest of the stack probably works well for you.

Links: [Docs](https://docs.molecule.ai) · [GitHub](https://github.com/Molecule-AI/molecule) · [Changelog](https://docs.molecule.ai/changelog)

---

*Self-post — no affiliate links, no spam. Posting because this is the kind of detail I wish had been available when I was evaluating agent frameworks.*
