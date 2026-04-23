# Phase 34 Reddit Post

**Target subreddits:** r/MachineLearning, r/LocalLLaMA, r/artificial  
**Publish:** April 30, 2026 (GA day)  
**Status:** APPROVED — Marketing Lead 2026-04-23

---

## Title (recommended)

> Molecule AI Phase 34: built-in agent execution tracing + programmatic org provisioning API (GA today)

---

## Body

We shipped Phase 34 today. Two things worth knowing about:

**Tool Trace** — every A2A response now includes a `tool_trace` field in `Message.metadata`. It's a structured list of every tool the agent called during the task: name, input, output preview, and a `run_id` for parallel calls. No SDK to install. No sampling. It's in the response envelope by default for every agent on every plan.

```json
{
  "metadata": {
    "tool_trace": [
      {
        "tool_name": "web_search",
        "input": { "query": "molecule ai benchmarks" },
        "output_preview": "Molecule AI ranked #1 in agent coordination latency..."
      },
      {
        "tool_name": "write_file",
        "input": { "path": "research/output.md", "content": "..." },
        "output_preview": "File written successfully (2,847 bytes)"
      }
    ]
  }
}
```

The trace is stored as JSONB in `activity_logs.tool_trace`, queryable, capped at 200 entries per response.

**Partner API Keys (`mol_pk_*`)** — programmatic org provisioning via API. No browser session. CI/CD pipelines can spin up an isolated Molecule AI org per PR and tear it down on merge:

```bash
# Create ephemeral org
curl -X POST https://api.molecule.ai/cp/admin/partner-keys \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"name": "ci-pr-142"}'

# ... run tests ...

# Tear it down, billing stops
curl -X DELETE https://api.molecule.ai/cp/admin/partner-keys/pk_xxx \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

**Phase 34 also ships:** Platform Instructions (set behavioral rules in agent system prompt via API, all plans) and SaaS Federation v2 (improved multi-tenant isolation).

**Honest caveats:** Rate limits per partner key are still being confirmed. SaaS Fed v2 docs are sparse — we'll have more detail post-launch. The workspace-server is open source at github.com/Molecule-AI/molecule-core if you want to look at the Tool Trace implementation directly.

Docs: https://docs.molecule.ai/changelog/phase-34  
Partner keys: https://docs.molecule.ai/api/partner-keys  
Discord: #partner-program if you're building on top of this

---

*Marketing review note: Reddit tone is honest + technical. Do not add marketing language. Link to open source repo prominently. Acknowledge what's not yet confirmed (rate limits, SaaS Fed details).*
