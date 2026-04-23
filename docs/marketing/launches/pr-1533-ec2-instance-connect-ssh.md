# Positioning Brief: EC2 Instance Connect SSH
**PR:** [#1533](https://github.com/Molecule-AI/molecule-core/pull/1533) — `feat(terminal): remote path via aws ec2-instance-connect + pty`
**Merged:** 2026-04-22
**Owner:** PMM | **Status:** APPROVED — routing to team

---

## Situation

When workspace provisioning moved from local Docker to the SaaS control plane (Fly Machines / EC2), a gap opened: Docker workspaces had a canvas terminal tab. SaaS-provisioned EC2 workspaces didn't — there was no path to exec into a cloud VM from the browser without a public IP, pre-configured SSH keys, or a bastion host.

PR #1533 closes that gap using **EC2 Instance Connect Endpoint (EICE)** — a purpose-built AWS service for IAM-authenticated, key-free SSH access to instances, including those in private subnets.

---

## Problem Statement

Getting a terminal into a SaaS-provisioned EC2 workspace requires infrastructure that most users don't have set up. The options available before this PR:

| Option | What's needed | Works for agents? |
|--------|---------------|---------------------|
| Direct SSH | Public IP + keypair + key distribution | No — no public IP on private-subnet EC2s |
| Bastion host | Separate EC2 + SSH config + key for bastion | No — extra infra, adds attack surface |
| SSM Session Manager | SSM agent installed + IAM profile + session document | Partially — requires pre-config per instance |
| EC2 Instance Connect CLI | `aws ec2-instance-connect ssh` — but must be run from a machine with the right IAM | Designed for humans, not agent runtimes |

For an agent runtime that spins up workspaces dynamically, none of these are acceptable. EC2 Instance Connect via EICE is the right fit: it requires only IAM permissions and a VPC Endpoint (already available in the SaaS VPC), and the session is initiated server-side by the platform — not by the agent's laptop.

---

## Solution

CP-provisioned workspaces (those with an `instance_id` in the workspaces table) get a terminal tab in the canvas automatically. The platform handles the EICE handshake and proxies the PTY over the WebSocket — the user sees a fully interactive terminal with no configuration required.

```
User opens terminal tab in canvas
  → platform checks workspace.instance_id
  → instance_id found → spawn aws ec2-instance-connect ssh --connection-type eice
  → PTY bridged to canvas WebSocket
  → user gets interactive shell in < 3 seconds
```

---

## Core Claims

### Claim 1: No SSH keys, no bastion, no public IP

EC2 Instance Connect pushes a temporary RSA key to the instance metadata via the AWS API, valid for 60 seconds. The session uses that key — no pre-shared key on disk, no key rotation to manage, no key distribution to instances. The platform initiates the connection; users never touch an SSH key.

### Claim 2: Private subnet instances work out of the box

EICE (EC2 Instance Connect Endpoint) routes the connection through AWS's internal network — no internet egress, no public IP, no ingress security group rules. The only requirement is a VPC Endpoint for EC2 Instance Connect in the same VPC as the target instance. The SaaS VPC already has this.

### Claim 3: Zero per-user configuration

The terminal tab appears for every CP-provisioned workspace automatically. No IAM role setup by the user, no SSM configuration, no bastion. The platform's IAM credentials (the same ones used to provision the instance) are used for EICE — the user doesn't need to know anything about AWS IAM policies to get a shell.

---

## Target Audience

**Primary:** DevOps and platform engineers managing SaaS-provisioned workspaces on EC2. They want browser-based terminal access without SSH key overhead. They likely already have IAM roles set up for their AWS environment and will recognise EICE as the right primitive.

**Secondary:** Enterprise security reviewers evaluating Molecule AI's SaaS offering. The ability to connect to cloud VMs via IAM — not shared SSH keys — is a meaningful signal. It aligns with the enterprise governance narrative and per-workspace auth token story.

**Not the audience:** Self-hosted users (Docker workspaces already have terminal via `docker exec`). The value proposition is SaaS/Control Plane-specific.

---

## Competitive Angle

EC2 Instance Connect integration for browser-based terminal access is not documented for any competitor:

- **LangGraph**: No terminal integration. Users who want shell access to provisioned resources must SSH manually or use SSM Session Manager via the AWS CLI.
- **CrewAI**: No cloud VM terminal story. Enterprise tier has SaaS management UI, but no browser-based shell access.
- **AutoGen (Microsoft)**: No EC2 integration documented. Relies on user-managed infrastructure.
- **Custom/self-rolled agent platforms**: Must implement EICE or SSM themselves. Molecule AI ships it as a product feature.

This is an uncontested claim for the AWS-aligned segment. It belongs in press briefings and analyst conversations as a concrete example of the SaaS control plane doing work users would otherwise have to do themselves.

---

## Messaging Tier

**Feature tier: Enhancement** (not a standalone product launch)

EC2 Instance Connect SSH is a meaningful UX improvement to the SaaS workspace experience. It belongs in:
- Phase 30 remote workspaces narrative as "SaaS terminal access"
- SaaS onboarding copy ("your EC2 workspace has a terminal tab — no SSH keys needed")
- Release notes (not a press release)

**Do not frame as:**
- A new standalone product
- A replacement for local Docker terminal
- A competitor-specific feature (lead with the benefit, not the AWS integration)

---

## Taglines

Primary: *"Your SaaS workspace has a terminal tab. No SSH keys required."*

Secondary: *"Connect to any EC2 workspace from the canvas — IAM-authorized, no bastion, no public IP."*

Fallback (technical): *"CP-provisioned workspaces get browser-based terminal via AWS EC2 Instance Connect Endpoint. No keypair on disk. No bastion. No configuration."*

---

## Channel Coverage

| Channel | Asset | Owner | Status |
|---------|-------|-------|--------|
| Blog post | "How to access your EC2 workspace terminal from the canvas" | Content Marketer | Blocked: needs DevRel code demo first |
| Social launch thread | 5 posts: problem → solution → claim 1 → claim 2 → CTA | Social Media Brand | Blocked: awaiting blog post + code demo |
| Code demo | Working example: open canvas → click terminal → interact with EC2 workspace | DevRel Engineer | Needs assignment (#1545) |
| Docs | `docs/infra/workspace-terminal.md` | DevRel Engineer | ✅ Shipped in PR #1533 |

**Coverage decision:** Blog post + social thread. Not a standalone campaign. Frame as "SaaS workspace terminal" within the Phase 30 remote workspaces narrative.

---

## Positioning Alignment

- **Phase 30 remote workspaces**: EICE terminal completes the remote workspace UX — agents register, accept tasks, and now also have a terminal, all without leaving the canvas
- **Per-workspace auth tokens**: The same IAM-scoped credentials that authorize A2A also authorize EICE — the platform manages the credential lifecycle, not the user
- **Enterprise governance**: No SSH keys means no orphaned keys in AWS IAM. Connection authorization via IAM is auditable in CloudTrail. This is a governance argument as much as a UX argument.

---

## Open Questions

- [x] Does the terminal UI expose EC2 Instance Connect as a distinct connection type? → No — seamless; the platform handles it transparently
- [x] Is there a docs page? → Yes: `docs/infra/workspace-terminal.md` (shipped in PR #1533)
- [ ] Social Media Brand: confirm launch thread length (5 posts recommended)
- [ ] Confirm EICE VPC Endpoint is present in the SaaS production VPC (DevOps/ops check)

---

## Sign-off

- [x] PMM positioning: approved
- [ ] Marketing Lead: pending
- [ ] DevRel: needs assignment (#1545)
- [ ] Content Marketer: blocked on DevRel code demo

---

*PMM — routing to DevRel (#1545 code demo) → Content Marketer (#1546 blog) → Social Media Brand (#1547 launch thread). Close when all routed.*