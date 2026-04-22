# EC2 Instance Connect SSH — Social Copy
Campaign: ec2-instance-connect-ssh | Blog: `docs/blog/2026-04-22-ec2-instance-connect-ssh/`
Slug: `ec2-instance-connect-ssh`
Publish day: TBD — coordinate with Marketing Lead
Assets: OG image at `docs/assets/blog/2026-04-22-a2a-enterprise-og.png` (share with A2A campaign if single-image budget)

---
**NOTE:** Copy ready for human social media execution. X credentials blocked in all agent workspaces.

---

## X (Twitter) — Primary thread (5 posts)

### Post 1 — Hook
When an AI agent workspace fails to boot, you need a shell.

Before EC2 Instance Connect Endpoint: open an inbound SSH port, or set up a bastion host. Neither fits a self-serve platform where the person debugging isn't always the person who owns the networking.

Now: open the Terminal tab. Bash prompt appears. Nothing to configure.

→ https://docs.molecule.ai/blog/ec2-instance-connect-ssh

---

### Post 2 — What changed
Molecule AI's Terminal tab now routes through AWS EC2 Instance Connect Endpoint — for CP-provisioned workspaces.

The tunnel is signed by the AWS API on demand. It exists only for the duration of the session. No inbound ports. No long-lived credentials.

Platform engineers get a shell. Security teams get CloudTrail records.

→ https://docs.molecule.ai/blog/ec2-instance-connect-ssh

---

### Post 3 — How it works (technical)
The path:

Canvas WebSocket → molecule-server → aws ec2-instance-connect ssh
→ EC2 Instance Connect Endpoint → EC2 instance → docker exec bash

aws-cli v2 handles the EICE WebSocket handshake and signed credential injection.
No native SDK needed. The tenant image adds ~1MB via apk.

Standard SSH. AWS-signed. Ephemeral.

→ https://docs.molecule.ai/blog/ec2-instance-connect-ssh

---

### Post 4 — IAM setup (the part that matters)
EIC Endpoint requires two IAM permissions on the molecule-cp role:

→ ec2-instance-connect:SendSSHPublicKey
→ ec2-instance-connect:OpenTunnel

Both scoped with aws:ResourceTag/Role=workspace — so the permission applies only to instances Molecule AI provisioned.

One EIC Endpoint per VPC. Create it once.

Full IAM policy JSON in the docs:

→ https://github.com/Molecule-AI/molecule-core/blob/main/docs/infra/workspace-terminal.md

---

### Post 5 — CTA
CP-provisioned workspaces on EC2? The Terminal tab now Just Works.

No bastion host. No inbound SSH. No SSH key rotation.

Canvas → Terminal tab → bash prompt.

Docs + IAM setup guide:

→ https://docs.molecule.ai/blog/ec2-instance-connect-ssh

---

## LinkedIn — Single post

**Title:** The last reason to open an inbound SSH port for your AI agent platform

**Body:**

For teams running AI agent platforms on AWS EC2, shell access has always been the awkward problem.

Option A: open inbound port 22. Permanent attack surface. Security team wants a review. The review takes three weeks.

Option B: bastion host. Now you have another infrastructure component to provision, credential-rotate, and audit. Also, latency.

EC2 Instance Connect Endpoint takes a different approach. Instead of opening a door from the outside, it establishes a signed tunnel initiated by the AWS API — on demand, scoped to a single session, automatically closed when you're done.

Molecule AI's Terminal tab now uses EIC Endpoint for all CP-provisioned workspaces. Open the tab, get a bash prompt. The platform handles the signing. CloudTrail records the OpenTunnel event linked to the IAM principal that initiated it.

For platform engineers, this means faster time-to-debug when a workspace fails to boot. For security teams, it means audit trails without permanent SSH exposure. For the platform itself, it means no bastion host to maintain.

The tradeoff: each VPC needs one EIC Endpoint, and the tenant image needs aws-cli v2 (~1MB). Both are one-time setup costs.

If you're running agent workspaces on EC2 and your operators are still SSHing in via a bastion, this is the upgrade.

→ [Read the docs](https://docs.molecule.ai/blog/ec2-instance-connect-ssh)
→ [IAM setup guide](https://github.com/Molecule-AI/molecule-core/blob/main/docs/infra/workspace-terminal.md)

---

## Campaign notes

**Audience:** Platform engineers (X), DevOps / infra leads (LinkedIn)
**Tone:** Technical and practical. Lead with the operational pain, not the AWS feature. The story is: no bastion, no inbound SSH, audit trail, faster debugging.
**Differentiation:** Secure shell access for CP workspaces — not a feature comparison with other platforms, but an operational improvement for Molecule AI self-hosted deployments.
**Suggested image:** Share the A2A Enterprise OG image or create a dedicated one showing the EICE tunnel path: Canvas → AWS API → EIC Endpoint → EC2 instance → bash. Dark theme, mint accents.
**Hashtags:** #AWS #EC2 #AIPlatform #MoleculeAI #DevOps #PlatformEngineering
**Coordination:** Publish after EC2 Instance Connect SSH blog post is live. Coordinate with Marketing Lead on timing. Suggested spacing: Day 3-4 of launch week (after Chrome DevTools MCP Day 1 and A2A Enterprise Day 1).