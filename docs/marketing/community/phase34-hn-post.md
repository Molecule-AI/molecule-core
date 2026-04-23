# Phase 34 Launch — Hacker News Show HN Post

**Title:** `Molecule Phase 34: Tool Trace, Platform Instructions, Partner API Keys, SaaS Federation v2`

**HN post body (2-3 sentences + link):**

> Show HN: Phase 34 of [Molecule](https://github.com/Molecule-AI/molecule) (open-source agentic framework) is now live. Highlights: Tool Trace surfaces per-call reasoning in Message.metadata, Platform Instructions lets orgs push governance rules to every agent, and Partner API Keys enable scoped programmatic provisioning. Link to full docs in comment.

---

**First-reply comment (technical "more context"):**

---

Here's what each feature actually does, with enough detail to evaluate whether it's relevant to your work.

**Tool Trace (`Message.metadata.tool_trace`)**

This is the most immediately useful feature for anyone debugging or auditing agent behavior. Every tool call now emits a structured record:

```json
{
  "tool": "workspace_deploy",
  "reasoning": "User said 'deploy to staging'. Checking env var for target.",
  "input": {"env": "staging"},
  "output": {"status": "ok", "deployment_id": "dep_8xf92"}
}
```

`reasoning` is the model's own explanation for why it chose that tool — not post-hoc rationalization. Currently depends on model quality (Claude Sonnet 4 does it consistently; smaller models vary). Works in both single-agent and multi-agent delegation flows.

**Platform Instructions**

Org-level governance rules that agents inherit at session start:

```json
{
  "type": "instruction",
  "instruction": "Tag all provisioned resources with cost_center tag.",
  "priority": "required"
}
```

`priority: "required"` agents cannot override; `priority: "preferred"` they can. This is for multi-tenant deployments where you need to enforce compliance without patching prompts per release. The instructions are stored server-side — they travel with the org, not the session.

**Partner API Keys (`mol_pk_*`)**

Scoped tokens for programmatic org provisioning. Key capabilities:
- Provision workspaces under your org without admin credentials
- Token-scoped: `provision:write`, `read:only`, or custom per key
- Tiered rate limits by flat-rate tier (docs cover the specifics)
- Keys are org-level, not user-level — useful for product integrations

**SaaS Federation v2**

Cross-org agent identity and delegation. Org A can define a trust policy allowing Org B's agents to act within a defined scope. This is the most architecturally novel piece — the trust model is policy-driven, not pairwise-keyed. The [docs](https://docs.molecule.ai) explain it better than I can in a comment.

**What we're honest about**

- Tool Trace quality depends on the model. Not all models produce consistent reasoning chains.
- Platform Instructions is a governance tool, not a sandbox. It enforces organizational policy, not security boundaries.
- SaaS Fed v2 is new — the trust model is documented, but production hardening is ongoing.

Repo: [github.com/Molecule-AI/molecule](https://github.com/Molecule-AI/molecule)
Docs: [docs.molecule.ai](https://docs.molecule.ai)

Happy to answer questions about specific use cases.
