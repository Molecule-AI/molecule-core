# Battlecard: Molecule AI vs. LangGraph A2A
**Created by:** Content Marketer
**Date:** 2026-04-22
**Source:** PMM competitive research brief (issue-14-competitive-research-brief.md)
**Use:** Competitive calls, sales enablement, comparison pages
**Review:** PMM to approve before distribution

---

## Tagline
*LangGraph A2A connects agents. Molecule AI A2A connects agents with full control.*

---

## The Question Prospects Ask
*"We're evaluating LangGraph for our multi-agent setup. How does Molecule AI compare?"*

---

## Three-Sentence Governance Contrast

LangGraph's A2A implementation (PRs [#6645](https://github.com/langchain-ai/langgraph/pull/6645) + [#7113](https://github.com/langchain-ai/langgraph/pull/7113)) adds A2A client capability — agents can send and receive tasks across endpoints. But LangGraph's implementation has no org-level governance layer: no workspace-scoped token enforcement, no immutable audit attribution per org key, and no revocation model. Molecule AI's A2A ships with all three — org API key attribution on every cross-agent call, per-workspace bearer tokens, and instant revocation — so compliance teams can answer "which agent accessed which workspace, and what did it do with the data?"

---

## Capability Comparison

| | LangGraph A2A | Molecule AI A2A |
|---|---|---|
| Connect agents | ✅ | ✅ |
| A2A client (send + receive) | ✅ | ✅ |
| Cross-infrastructure discovery | Via DNS-AID (not merged) | ✅ Via workspace registry |
| Org API key attribution | ❌ | ✅ |
| Per-workspace bearer tokens | ❌ | ✅ |
| Audit trail on every call | ❌ | ✅ |
| Instant revocation | ❌ | ✅ |
| Cross-network A2A | ❌ | ✅ |
| Fleet visibility | ❌ | ✅ Canvas overlay |

---

## Competitive Frame

**If the prospect is technical:**
> "LangGraph's A2A is solid protocol work. What we're hearing from enterprise teams is that 'can agents talk to each other' is the starting question, not the ending one. The question compliance teams ask is: can you show me which agent called which, what it did, and can you revoke access without a redeploy? That's where Molecule AI's org API key layer makes a difference."

**If the prospect is a buyer/enterprise:**
> "Both platforms can do agent-to-agent communication. The difference is what's in the audit log. Molecule AI attributes every A2A call to an org API key — so you can trace the delegation chain, export it for compliance, and revoke access instantly. If that's on your checklist, we're the only platform that has it."

**If the prospect is already using LangGraph:**
> "You don't have to replace LangGraph — Molecule AI workspaces can call LangGraph agents via A2A. What you get is the governance layer on top of whatever agents are already running. The Molecule AI canvas becomes the unified control plane for your whole agent fleet."

---

## Key PRs to Know

- LangGraph [#6645](https://github.com/langchain-ai/langgraph/pull/6645): A2A server (inbound) — open, no governance
- LangGraph [#7113](https://github.com/langchain-ai/langgraph/pull/7113): A2A client (outbound) — open, no governance
- LangGraph [#7205](https://github.com/langchain-ai/langgraph/pull/7205): DNS-AID discovery — auto-closed, not a threat

---

## Objection Handling

**"LangGraph is more mature"**
> "LangGraph's A2A PRs are open and advancing, but they're still experimental — there's no governance layer, no revocation, and no audit trail in the current implementation. Molecule AI shipped A2A with governance in Phase 30 (2026-04-20). Maturity in the agent framework doesn't translate to maturity in enterprise controls."

**"We can add governance ourselves"**
> "You can — but it's the hard part. Org-level attribution, audit chain verification, and revocation across a multi-workspace fleet is non-trivial to build correctly. Molecule AI ships it out of the box."

**"We don't need that level of governance"**
> "For internal dev agents, you might not. But if any of your agents touch customer data, production systems, or regulated environments — the compliance review will ask for it. Better to have it and not need it."

---

*PMM review required before sales distribution. GH API blocked — filed as markdown record 2026-04-22.*
