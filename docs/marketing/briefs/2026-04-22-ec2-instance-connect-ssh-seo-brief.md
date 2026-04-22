# SEO Brief: Phase 30 — EC2 Instance Connect SSH
**Issue:** (to fill in by SEO Analyst)
**Date:** 2026-04-22
**Author:** Content Marketer (SEO gap filing)
**Campaign:** EC2 Instance Connect SSH Terminal Support
**KD (SEO Analyst):** EC2 Instance Connect ~25-40 | AI agent SSH access ~10-25 | SSH bastion host alternative ~20-35

---

## 1. Context

Phase 30 ships EC2 Instance Connect Endpoint integration for CP-provisioned workspaces. The Terminal tab in Canvas now connects via a signed, ephemeral SSH tunnel — no inbound SSH ports, no bastion host, no long-lived credentials. CloudTrail records every tunnel open event.

**Published:**
- Blog post: `docs/blog/2026-04-22-ec2-instance-connect-ssh/index.md`
  - Title: "EC2 Instance Connect SSH: Shell Access Without Opening Inbound Ports"
  - Covers: the bastion/inbound-SSH problem, EICE tunnel flow, IAM setup checklist, CloudTrail audit trail, getting started guide

**This brief:** SEO targeting for the EC2 Instance Connect SSH campaign — targets platform engineers and DevOps teams who need secure shell access to AI agent workspaces on EC2.

---

## 2. Target Keywords

| Keyword | Intent | Difficulty | Priority |
|---|---|---|---|
| `EC2 Instance Connect` | Informational | Medium | High |
| `AI agent SSH access` | Informational | Low | High |
| `AI agent on EC2 without SSH keys` | Informational | Low | High |
| `SSH AI agent platform` | Commercial | Medium | High |
| `EC2 Instance Connect Endpoint tutorial` | Informational | Medium | High |
| `secure shell access EC2 AI agent` | Informational | Low | Medium |
| `SSH bastion host alternative` | Informational | Low | Medium |
| `AI agent DevOps shell access` | Informational | Low | Medium |

**Primary angle:** `EC2 Instance Connect Endpoint tutorial` + `AI agent SSH access` — captures platform engineers looking for how to get shell access to agent workspaces without the security overhead of inbound SSH or bastion hosts.

**Secondary angle:** `SSH bastion host alternative` — captures DevOps teams already running bastion hosts and evaluating alternatives.

---

## 3. Content Angle

The use case is operational: teams running Molecule AI on EC2 need a way for operators (debugging a workspace, auditing configuration) to get shell access without permanently exposing port 22. The security story is the differentiator: every tunnel open event is recorded in CloudTrail, linked to the IAM principal who initiated it.

**SEO angle:** Target platform engineers and DevOps teams searching for EC2 Instance Connect setup guides (IAM permissions, VPC endpoint creation, tenant image configuration). Also capture the "bastion host" searchers who are evaluating a better approach.

---

## 4. Query Patterns to Capture

- "EC2 Instance Connect Endpoint setup" (how-to, high intent)
- "EC2 Instance Connect IAM permissions" (configuration, medium intent)
- "AI agent platform shell access" (use-case, low competition)
- "SSH without inbound ports EC2" (specific technical query)
- "AWS ephemeral SSH tunnel" (alternative phrasing)

---

## 5. Recommended Content

1. **Tutorial page (existing):** `docs/tutorials/workspace-terminal-ieee.md` — covers the full setup with diagnostic guidance for incomplete IAM wiring. Good canonical for `EC2 Instance Connect tutorial`.

2. **Comparison reference:** Add a "vs. bastion host" section to the tutorial or blog post. Targets the ~30% of operators who have tried bastion hosts and are actively looking for alternatives.

---

*SEO gap filed by Content Marketer 2026-04-22. SEO Analyst: please validate keyword difficulty scores and add to keywords.md. Also note: "SSH" keyword is now covered in EC2 Instance Connect SSH content — consider adding to overall keywords tracking.*