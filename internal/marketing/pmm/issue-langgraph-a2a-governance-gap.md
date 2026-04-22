# GH Issue Draft — LangGraph A2A Governance Gap

**Status:** READY TO FILE (GH API 401 — filed as markdown record)
**Created by:** Content Marketer, 2026-04-22
**PMM source:** issue-14-competitive-research-brief.md (Track 1 WATCH finding)
**Labels:** `marketing` `pmm: positioning update` `langgraph`

---

**Proposed title:** `pmm: LangGraph A2A governance gap — content + battlecard opportunity`

**Proposed body:**

## PMM finding (2026-04-21)

LangGraph's A2A implementation (PRs [#6645](https://github.com/langchain-ai/langgraph/pull/6645) + [#7113](https://github.com/langchain-ai/langgraph/pull/7113)) adds A2A client capability — inbound and outbound — but has **no governance layer**.

**The gap:**
- LangGraph: no workspace-scoped token enforcement, no immutable audit attribution, no revocation model
- Molecule AI: org API key attribution on every cross-agent call, per-workspace bearer tokens, instant revocation

**Two-line contrast:** *"LangGraph A2A connects agents. Molecule AI A2A connects agents with full control."*

## Content action items

1. **Battlecard snippet** — 3-sentence governance comparison, ready for sales team to use in competitive calls
2. **Blog post / comparison page** — "Why A2A Native Isn't Enough Without Governance" — targets enterprise buyers evaluating A2A platforms; query: "LangGraph A2A audit trail"
3. **Social copy update** — add LangGraph governance gap callout to A2A Enterprise social thread (already done in staging commit 12d32f7)

## Labels
`marketing` `pmm: positioning update` `langgraph`

## Owner
Content Marketer — battlecard + blog post; Sales Enablement — snippet distribution
