# Phase 34 Community FAQ

*Top-10 anticipated questions from the Molecule community, Phase 34 launch (April 30, 2026).*
*Prepared for DevRel + Support pre-brief. PM partner name TBD — swap "Acme Corp" placeholder when confirmed.*

---

## Q1: What exactly is Tool Trace and when did it exist before?

Tool Trace surfaces per-call reasoning in `Message.metadata.tool_trace`. Before Phase 34, you could infer tool calls from logs, but there was no structured record of *why* the agent chose a particular tool or what reasoning drove the decision. Tool Trace adds a structured array:

```python
message = await session.send("Deploy my workspace to staging")
for entry in message.metadata.tool_trace:
    print(entry["tool"], "→", entry["reasoning"])
# workspace_deploy → User said 'deploy to staging'. Checking env var for target.
# notify_slack → Deploy succeeded. Sending confirmation to #eng-alerts.
```

**When it doesn't appear:** Tool Trace is model-dependent. Not all models produce consistent reasoning chains. Claude Sonnet 4 does it reliably. Smaller models may emit partial or no traces. This is documented as alpha.

---

## Q2: Is Tool Trace a security boundary?

**No.** Tool Trace is observability, not enforcement. The `reasoning` field is the model's own description of its decision — it's not parsed or acted upon by the runtime. If you need to *prevent* an agent from calling a specific tool, use Platform Instructions or WorkspaceAuth.

Use Tool Trace for: debugging, auditing, building dashboards, evaluating model quality.
Do not use it for: access control, security enforcement, compliance certification.

---

## Q3: What are Platform Instructions and how do they differ from system prompts?

System prompts are per-session, managed by the developer. Platform Instructions are org-wide rules pushed by an admin that agents inherit at session start — they travel with the org, not the session.

```json
{
  "type": "instruction",
  "instruction": "Always tag resources with cost_center before provisioning.",
  "priority": "required"
}
```

Priority levels:
- `required`: agent cannot override. The runtime enforces it.
- `preferred`: agent may override with justification. The runtime logs the deviation.

Platform Instructions are stored server-side and applied to every new session in the org. They don't require redeploying or patching prompts.

**Difference from system prompts:**

| | System Prompt | Platform Instruction |
|---|---|---|
| Scope | Per session | Org-wide |
| Management | Developer | Org admin |
| Override | Full control | `required` = no; `preferred` = with reason |
| Storage | Application code | Server-side, org config |

---

## Q4: How do Partner API Keys work? Can I restrict them to specific workspaces?

Yes. Partner API Keys (`mol_pk_*`) are scoped tokens issued at the org level for programmatic provisioning. Each key has a capability scope:

- `provision:write` — create/read/delete workspaces under the org
- `read:only` — read-only access to org resources
- Custom: define a scope per key

Keys are org-level, not user-level. You can issue multiple keys for different integrations — revoke one without affecting others.

```bash
# Example: provision a workspace with a scoped key
curl -X POST https://api.molecule.ai/v1/workspaces \
  -H "Authorization: Bearer mol_pk_provision_only_xxxxx" \
  -d '{"name": "acme-corp-build", "template": "standard"}'
```

Rate limits are tiered by flat-rate plan ($9 / $29 / $99 USD/month). Details are on the pricing page.

---

## Q5: What's the difference between Partner API Keys and regular API keys?

Regular API keys are user-level credentials tied to a specific user account. Partner API Keys are org-level scoped tokens designed for product integrations — they allow a partner (like Acme Corp) to build on top of Molecule's API without exposing a human user's credentials.

| | Regular API Key | Partner API Key |
|---|---|---|
| Scope | User-level | Org-level |
| Use case | Developer auth | Product integrations |
| Provisioning | No | Yes |
| Revocation | Per user | Per key (granular) |

---

## Q6: What is SaaS Federation v2 and when would I use it?

SaaS Federation v2 enables cross-org agent identity and delegation. Org A can define a trust policy allowing Org B's agents to act within a defined scope — without sharing credentials.

**Use cases:**
- **Partner integrations:** Your product is used by multiple enterprise customers. Federation lets agents in each customer's org act within their own org under their own identity.
- **Multi-tenant SaaS:** You run a platform where end-user agents need to delegate to your backend agents — Federation provides the trust mesh without pairwise key sharing.
- **Compliance isolation:** Each org's agents operate under their org's policy, not a shared policy.

The trust model is policy-driven. You define what Org B's agents can do within Org A, and the runtime enforces it. The [docs](https://docs.molecule.ai) cover the full trust model with examples.

---

## Q7: How does Platform Instructions interact with SaaS Federation v2?

Platform Instructions are org-scoped — they apply within a single org. Federation v2 is about cross-org delegation. When an agent in Org A delegates to an agent in Org B:

1. Org A's Platform Instructions apply to the delegating session.
2. Org B's Platform Instructions apply to the receiving session.
3. The trust policy (defined during Federation setup) determines what the delegated agent can do in Org A's context.

If Org A trusts Org B, the receiving agent operates under Org A's instructions *within the scope defined by the trust policy*. Instructions outside that scope are not automatically granted.

---

## Q8: Is Phase 34 production-ready? What's alpha vs beta?

| Feature | Status | Notes |
|---|---|---|
| Tool Trace | Alpha | Model-dependent. Large models (Claude Sonnet 4) reliable; smaller models vary. |
| Platform Instructions | Beta | Functional. `required`/`preferred` priority implemented. |
| Partner API Keys | Beta | Production-ready for org provisioning. Rate limits apply. |
| SaaS Federation v2 | Alpha | Trust model documented. Production hardening ongoing. |

Tool Trace and SaaS Fed v2 are marked alpha because the underlying behavior depends on model quality (Tool Trace) and production hardening under real multi-org trust policies (Fed v2). Platform Instructions and Partner API Keys are beta — stable APIs, documented behavior.

---

## Q9: How do I migrate from Phase 33 to Phase 34? Are there breaking changes?

No breaking changes in Phase 34. The new fields (`Message.metadata.tool_trace`, Platform Instructions, Partner API Keys) are additive — existing integrations continue to work without modification.

**Migration steps:**
1. Update your SDK to the Phase 34 client version.
2. Tool Trace: read `message.metadata.tool_trace` if present; fall back gracefully if absent.
3. Platform Instructions: set up org-level instructions in the admin console (docs have the walkthrough).
4. Partner API Keys: generate new keys in the org settings; existing keys remain valid.
5. SaaS Fed v2: Federation setup is opt-in — existing orgs are unaffected until explicitly configured.

---

## Q10: Where do I report bugs or get help with Phase 34 features?

- **Bugs:** Open an issue on [github.com/Molecule-AI/molecule](https://github.com/Molecule-AI/molecule) with the Phase 34 label. Include: SDK version, model used, and a minimal repro if possible.
- **Questions:** [GitHub Discussions](https://github.com/Molecule-AI/molecule/discussions) — monitored by the team. For urgent production issues, use the support channel in your org's Slack/Discord.
- **Documentation:** [docs.molecule.ai/changelog](https://docs.molecule.ai/changelog) for full feature docs.
- **Feature requests:** GitHub Discussions — use the "feature request" category.

For Platform Instructions and Partner API Keys specifically: the docs include runnable examples. Tool Trace examples are in the [agent observability guide](https://docs.molecule.ai).

---

*Partner name placeholder "Acme Corp" — swap with confirmed partner name before cross-posting.*
*Document version: 1.0 — Phase 34 GA, April 30, 2026.*
