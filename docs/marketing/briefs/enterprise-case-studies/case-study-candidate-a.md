---
title: "How a US Enterprise Data Team Runs AI Agents on Their Own AWS Account"
date: 2026-04-25
slug: data-engineering-team-aws-remote-runtime
description: "A data engineering team needed AI agents that could access their AWS infrastructure without routing sensitive data through a third-party platform. Here's how they solved it."
tags: [enterprise, remote-runtime, aws, data-pipelines, governance]
---

# How a US Enterprise Data Team Runs AI Agents on Their Own AWS Account

> *"A data engineering team is currently using this for a pipeline agent running in their own AWS account — raw data never touches the Molecule AI platform."*
>
> — Phase 30 Sales Enablement Materials

---

## The Challenge

Data engineering teams working with sensitive or regulated data face a common constraint: AI agent platforms that route all traffic through their infrastructure. For teams operating under compliance requirements — SOC 2, HIPAA, financial data handling — that's a blocker. Third-party infrastructure means your data crosses a boundary your security team needs to sign off on.

The alternative has traditionally been building and maintaining your own agent infrastructure: the orchestration layer, the runtime, the observability stack. That's a significant engineering investment just to keep data on-premises.

---

## The Solution

This team runs their AI pipeline agents on their own AWS account — EC2 and ECS compute they already manage — with Molecule AI as the orchestration and governance layer.

The agents connect to the Molecule AI canvas via Phase 30 Remote Workspaces. They register as external workspaces, appear in the fleet view alongside the team's other agents, and communicate via A2A — but the compute runs on AWS infrastructure the team controls.

**What this means in practice:**

- Every API call the agent makes runs in the team's VPC, not through Molecule AI's infrastructure
- Org-scoped API keys tie every agent action to a named integration in the audit log
- MCP-compatible data tools connect directly to internal systems the agent needs to access
- The canvas gives the platform team fleet-wide visibility without requiring agents to run inside the platform's own runtime

**The data residency result:** raw data processed by the pipeline agents never crosses onto Molecule AI's infrastructure. The platform manages orchestration, governance, and observability. The compute handles data.

---

## The Results

- **Data residency:** Pipeline agent compute runs entirely on the team's own AWS VPC — raw data never touches Molecule AI's platform
- **Governance without overhead:** Org-scoped API keys give the security team full audit attribution on every agent action without requiring agents to run inside the platform's runtime
- **Fleet visibility:** All agents — local and remote — appear in a single canvas view with status indicators, activity feeds, and workspace-level logs
- **MCP-compatible tooling:** Data tools that speak MCP connect to the pipeline agents through Molecule AI's MCP bridge, without custom integration code

---

## What "Raw Data Never Touches the Platform" Actually Means

For most AI agent platforms, "running in your cloud" means the agent runs on the platform's infrastructure in your cloud account. The platform's control plane still sees all traffic.

With Molecule AI Phase 30 Remote Workspaces, the distinction is structural. External workspaces register with the platform's canvas and A2A routing — but the agent's compute and data access happen on infrastructure the team owns and controls. The platform sees orchestration metadata (which agent, which workspace, which action) but not the data the agent processes.

This is the difference between a platform that runs agents *for* you and a platform that orchestrates agents *you* run.

---

## About This Deployment

| | |
|---|---|
| **Deployment model** | Remote Runtime — self-hosted AWS (EC2/ECS) |
| **Platform features used** | Remote Workspaces, A2A, org-scoped API keys, MCP bridge |
| **Data tools** | MCP-compatible internal data connectors |
| **Compliance context** | SOC 2 / financial data handling |
| **Agent type** | Autonomous data pipeline agents (ETL, data processing) |

---

*This case study is published with anonymized data. A named version is in progress. If you'd like to discuss how Remote Workspaces could work for your infrastructure, [reach out to our team](https://molecule.ai/contact).*
