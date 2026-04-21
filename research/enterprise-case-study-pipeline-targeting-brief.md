# Enterprise Case Study Pipeline Targeting Brief

**Source:** GH#1398 CrewAI Enterprise Strategy + GH#1405 Enterprise Case Studies
**Author:** Research Lead
**Date:** 2026-04-21
**Status:** DRAFT — for Sales/CS review

---

## Purpose

Identify which existing Molecule AI pipeline contacts to prioritize for enterprise case study reference clearance outreach. Based on: (1) CrewAI enterprise target verticals and roles, (2) Molecule AI's existing pipeline signals, (3) reference clearance likelihood by segment.

---

## What We're Competing Against

**CrewAI's 18 named enterprise logos** (GH#1398):
IBM, PwC, NTT DATA, PepsiCo, RBC, DocuSign + 12 others

**CrewAI's target enterprise profile:**
- **Verticals:** Financial services, enterprise software, manufacturing, professional services
- **Roles:** VP Engineering, Director of Developer Productivity, Chief AI Officer, Head of Platform Engineering
- **Use case:** Multi-agent pipelines for internal tooling, code generation at scale, document processing, customer service automation
- **Deployment:** Dedicated VPC (AMP Factory), SSO-gated, enterprise procurement

---

## Molecule AI's Counter-Positioning Advantage

For each CrewAI target persona, identify Molecule AI's differentiation:

| CrewAI Target | Molecule AI Advantage | Who to Approach |
|---------------|----------------------|-----------------|
| **VP Engineering / Platform** | Remote runtime: agent compute where data lives, not on CrewAI's cloud | Platform engineering leads with data residency concerns |
| **Director of Developer Productivity** | Org-scoped API keys + audit logs: governance without sacrificing autonomy | Dev productivity teams at regulated enterprises |
| **Head of AI / CAIO** | Multi-tenant SaaS: no infra to manage, A2A protocol works across fleet | AI offices evaluating build-vs-buy |
| **Enterprise Sales (inbound)** | Docker + Remote mixed fleet: same Canvas, same auth, two runtimes | Companies already running self-hosted AI infra |

---

## Priority Outreach Segments

### Tier 1 — Highest clearance likelihood, strongest narrative

**1. Data engineering teams on AWS/GCP using Remote Workspaces**
- *Why:* Already referenced in Phase 30 sales enablement ("raw data never touches Molecule AI platform")
- *Use case:* Data pipeline agents, ETL automation, data processing
- *Deployment:* Remote Runtime (self-managed AWS/GCP compute)
- *Clearance likelihood:* HIGH — customer self-selected as security-conscious; likely contractually clear for technical reference
- *Approach:* Ask for technical reference call + use case quote. Anonymize if named clearance fails.

**2. Enterprise platform teams evaluating AI governance**
- *Why:* Org-scoped API keys + audit logs are a differentiator vs. CrewAI's developer-tool model
- *Use case:* Agent fleet governance, MCP plugin allowlists, compliance reporting
- *Deployment:* Hybrid (Canvas + Remote)
- *Clearance likelihood:* MEDIUM-HIGH — governance buyers are often more comfortable with references

**3. AI-first startups / mid-market companies with active dev teams**
- *Why:* Faster sales cycle, more likely to have named contacts willing to go on record
- *Use case:* Multi-agent development pipelines, autonomous code review, CI/CD integration
- *Deployment:* Molecule AI Cloud or self-hosted
- *Clearance likelihood:* MEDIUM — faster to close, but may lack enterprise legal process

### Tier 2 — Valuable but harder to clear

**4. Financial services / regulated enterprises (matching CrewAI's IBM/PwC/RBC profile)**
- *Why:* Same vertical as CrewAI's confirmed wins — strongest competitive displacement narrative
- *Use case:* Compliance automation, document processing, internal tooling
- *Clearance likelihood:* LOW in near term (FedRAMP, SOC 2, internal legal review) — start outreach now but expect 6–8 weeks

---

## Recommended First Move

**Approach the AWS data engineering team first** (Tier 1, #1 above):
- Anonymized reference already exists in sales materials — customer is presumably aware they may be referenced
- Technical use case is documented (pipeline agents, AWS, Remote Runtime)
- Self-selected for data security narrative — strongest Molecule AI proof point
- Clearance: start with CS contact asking for "technical reference call" before mentioning public use

**Script for CS initial outreach:**
> "We're preparing a technical case study for our Phase 30 launch and we'd love to feature the work your team is doing with [use case]. This would be a short [named/anonymized — their choice] overview of what you deployed and the outcome. Legal clearance typically takes 2–3 weeks — we're starting now so we're ready for launch. Would your contact be open to a 20-minute call with our marketing team?"

---

## What to Capture on the Call

For each reference candidate, collect:
1. **Named customer** (company + contact name + title) OR explicit anonymization approval
2. **Use case:** What problem, what Molecule AI features, how many agents/users
3. **Deployment model:** Cloud / self-hosted / hybrid; backend infrastructure
4. **Outcome metric:** Even directional ("reduced X by ~70%") is useful
5. **Quote:** 1–2 sentences on what problem they solved and why they chose Molecule AI
6. **Approval:** Email confirmation from legal or contact for marketing to reference

---

## Next Steps

- [ ] CS to pull list of all pipeline contacts with "data engineering," "platform engineering," or "AI governance" in role/company description
- [ ] CS to identify which contacts are on AWS or have data residency requirements (highest fit)
- [ ] Draft outreach email template (use script above)
- [ ] Begin legal clearance process for Tier 1 candidate this week
