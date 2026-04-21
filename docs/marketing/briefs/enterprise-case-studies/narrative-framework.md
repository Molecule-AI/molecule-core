# Enterprise Case Study Narrative Framework — GH#1405
**Owner:** Marketing Lead | **Research Lead:** Customer candidates
**Status:** NARRATIVE DRAFT — awaiting Research Lead candidate customers
**Purpose:** Pre-built narrative structure to accelerate case study writing once candidates arrive

---

## Competitive Context

**CrewAI's moat:** 18 named enterprise logos (IBM, PwC, NTT DATA, PepsiCo, RBC, DocuSign + 12). Their reference customer wall is a real sales asset — it answers "who else uses this?" before the first demo.

**Molecule AI's gap:** Zero named case studies. Even the Remote Workspaces blog quote ("mid-stage SaaS company, infrastructure lead") is anonymized. This is a GTM credibility blocker at the enterprise layer.

**Goal:** 2–3 case studies before Phase 30 GTM close. Priority: named > quote > anonymized.

---

## Narrative Themes (Pick 1 Per Case Study)

### Theme 1: Fleet Visibility at Scale
**Angle:** "We had agents in five different places. Molecule AI gave us one canvas."
**Buyer:** VP Engineering, Platform team, DevOps Lead
**CTA:** /docs/guides/remote-workspaces

**Customer profile:**
- Multiple agent deployment environments (cloud + on-prem, or multi-cloud)
- Existing problem: can't see what agents are doing across the fleet
- Existing workaround: custom dashboards stitching together logs from different systems

**Narrative arc:**
1. Problem: agents are distributed and invisible
2. Discovery: Phase 30 remote agent registration
3. Implementation: external workspace setup, A2A across network boundaries
4. Outcome: unified canvas, audit trail, governance

**Metrics to capture (directionally acceptable):**
- Reduction in agent monitoring overhead
- Time to onboard new agent type
- Number of environments unified under one canvas

---

### Theme 2: CI/CD + Automated Quality Gates
**Angle:** "Our AI agent runs Lighthouse on every PR. No human in the loop."
**Buyer:** Engineering Manager, QA Lead, Platform Engineer
**CTA:** /blog/chrome-devtools-mcp (Lighthouse audit use case)

**Customer profile:**
- Existing CI/CD pipeline (GitHub Actions, CircleCI, etc.)
- Already using or evaluating AI agents
- Problem: manual QA bottlenecks, UI regression catching late
- Existing workaround: manual Lighthouse runs, human code review for UI

**Narrative arc:**
1. Problem: UI regression catches late, manual QA bottleneck
2. Discovery: Chrome DevTools MCP + Molecule AI governance
3. Implementation: agent workspace registered externally, MCP tools wired to CI
4. Outcome: automated Lighthouse on every PR, agent reports score, flags regressions

**Metrics to capture:**
- Lighthouse score improvement tracking
- Time saved per PR cycle
- Number of regressions caught pre-deploy

---

### Theme 3: Enterprise Compliance + Governance
**Angle:** "Our security team asked for audit trails. Molecule AI had them before we finished the question."
**Buyer:** Security Lead, Compliance Officer, CISO, Platform Security
**CTA:** /blog/chrome-devtools-mcp (governance section), /docs/guides/org-api-keys

**Customer profile:**
- Regulated environment (finance, healthcare, legal, government)
- Active AI agent adoption with security/compliance scrutiny
- Problem: agents operating without attributable audit trails
- Existing workaround: no agent audit trail, or manual logging

**Narrative arc:**
1. Problem: agents operating without audit attribution — can't answer "which agent accessed what, when"
2. Discovery: org API keys + per-workspace bearer tokens
3. Implementation: org API key setup, agent registration, audit log integration
4. Outcome: full attribution on every agent action, instant revocation capability

**Metrics to capture:**
- Audit log completeness (% of agent actions captured)
- Time to revoke a compromised credential
- Compliance audit pass rate

---

### Theme 4: Multi-Agent Orchestration Across Teams
**Angle:** "Our PM agent talks to our dev agent, which talks to our ops agent — across two clouds."
**Buyer:** CTO, Head of Platform, AI/ML Lead
**CTA:** /docs/guides/remote-workspaces, /docs/architecture/a2a

**Customer profile:**
- Multiple specialized agents (PM, dev, ops, research)
- Agents on different networks/cloud environments
- Problem: agents can't communicate across environments, or comms require brittle webhook chains
- Existing workaround: shared Slack channels, manual handoffs

**Narrative arc:**
1. Problem: multi-agent workflows require agents to share context manually
2. Discovery: A2A across network boundaries, remote agent registration
3. Implementation: external workspaces on each environment, A2A proxy routing
4. Outcome: agents coordinate autonomously across cloud boundaries

**Metrics to capture:**
- Reduction in manual handoff time
- Agent coordination latency
- Number of agent-to-agent workflows live

---

## Case Study Format (1-page)

```
---
title: "[Customer Name] + Molecule AI: [Headline Outcome]"
date: TBD
slug: TBD
description: "[One sentence: what the customer achieved]"
tags: [enterprise, TBD]
---

# [Customer Name] + Molecule AI: [Headline Outcome]

> "[Pull quote — specific, not vague.]"
> — [Title], [Customer Name]

## The Challenge

[2-3 paragraphs. What problem were they trying to solve? What existing approach was failing or missing?]

## The Solution

[2-3 paragraphs. What did they build with Molecule AI? Deployment model, agent types, integrations.]

## The Results

- **[Metric 1]:** [Directional or precise]
- **[Metric 2]:** [Directional or precise]
- **[Metric 3]:** [Directional or precise]

[If named customer: quote. If anonymized: "A [vertical] [company type]..."]

## About [Customer Name]

[2-3 sentences. Company size, industry, what they do. Anonymized if not cleared.]

## About Molecule AI

[1 paragraph. Phase 30 positioning: agents that run anywhere, visible on your canvas, governed from one place. Link to relevant docs.]
```

---

## Writing Prompts per Theme (for Research Lead candidate brief)

When Research Lead returns candidate customer info, ask:
1. **Named vs. anonymized?** (Named > quote > anonymized)
2. **What's the one-sentence outcome?**
3. **Which theme does this fit?** (Fleet, CI/CD, Compliance, Multi-agent)
4. **Deployment model:** self-hosted, cloud, hybrid?
5. **What can we say publicly?** (Metrics, quote, or narrative only?)
6. **Legal/comms cleared?** (Required before publish)

---

## Where These Will Live

| Format | Location | Owner |
|--------|----------|-------|
| Full case study | `/docs/blog/YYYY-MM-DD-[slug]/index.md` | Marketing Lead |
| LinkedIn post | Social Media Brand queue | Marketing Lead approves copy |
| Sales one-pager | `/docs/sales/[slug].md` | Marketing Lead |
| Competitive battlecard | `/docs/marketing/competitive/case-studies-vs-crewai.md` | Marketing Lead |

---

*Marketing Lead narrative framework — 2026-04-21. Awaiting Research Lead customer candidates to fill in templates.*
