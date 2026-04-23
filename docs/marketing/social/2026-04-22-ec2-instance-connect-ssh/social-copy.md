# EC2 Instance Connect SSH — Social Copy
Campaign: ec2-instance-connect-ssh | PR: molecule-core#1533
Publish day: 2026-04-22 (today)
Assets: `marketing/devrel/campaigns/ec2-instance-connect-ssh/assets/`
Status: Draft — pending Marketing Lead approval + credential availability

---

## X (Twitter) — Primary thread (5 posts)

### Post 1 — Hook

> Your AI agent has a workspace on an EC2 instance.
>
> How do you get a shell inside it right now?
>
> Old answer: copy the IP, find the key, `ssh -i key.pem ec2-user@X.X.X.X`, hope your
> security group is right.
>
> New answer: click Terminal in Canvas.
>
> Molecule AI now speaks AWS EC2 Instance Connect.

---

### Post 2 — The problem it solves

> SSH into a cloud agent workspace sounds simple.
>
> It's not.
>
> → Instance IP changes on restart
> → Key management across your whole agent fleet
> → Security group rules you have to get right every time
> → No audit trail on who SSH'd in and when
>
> EC2 Instance Connect handles all of it. Molecule AI wires it up so
> your agent workspace is one Terminal tab away.

---

### Post 3 — How it works

> Molecule AI + EC2 Instance Connect:
>
> → Workspace provisioned in your VPC, instance_id stored
> → Click Terminal tab in Canvas → WebSocket opens
> → Platform calls `aws ec2-instance-connect ssh` under the hood
> → EIC Endpoint opens a tunnel, STS pushes a temporary key
> → PTY bridges directly to the Canvas terminal
>
> No keys to manage. No IP to find. No security group dance.
> One click.

---

### Post 4 — Security angle

> Every SSH access to a cloud agent workspace should be attributable.
>
> With EC2 Instance Connect:
>
> → IAM policy gates access (condition: `Role=workspace` tag)
> → STS temporary key, auto-expires
> → EIC audit log shows which principal requested the tunnel
> → No long-lived SSH keys anywhere
>
> Your security team will appreciate this.

---

### Post 5 — CTA

> EC2 Instance Connect SSH is live in Molecule AI (PR #1533).
>
> Provision a CP-managed workspace → open the Terminal tab → you're in.
>
> If you're still `ssh -i key.pem` into your agent fleet — there's a better way.
>
> [CTA: docs.molecule.ai/infra/workspace-terminal — pending docs publish]
> #AIAgents #MoleculeAI #AWS #DevOps #PlatformEngineering

---

## LinkedIn — Single post

**Title:** We gave AI agents their own terminal tab — powered by AWS EC2 Instance Connect

**Body:**

Getting a shell inside a cloud-hosted AI agent used to mean: find the instance IP, locate the SSH key, configure the security group, run `ssh`, hope nothing broke.

That's now one click inside Molecule AI.

We shipped EC2 Instance Connect SSH integration (PR #1533). Here's what changed:

**The old flow:**
Copy the EC2 IP → find the SSH key → configure the security group to allow port 22 → `ssh -i key.pem ec2-user@X.X.X.X` → verify you're connected

**The new flow:**
Provision a workspace in Canvas → click Terminal → you have a bash prompt

What makes this possible is AWS EC2 Instance Connect. The platform stores the `instance_id` from provisioning, calls `aws ec2-instance-connect ssh --connection-type eice` on your behalf, and the EIC Endpoint opens a tunnel with an STS-pushed temporary key. The PTY bridges straight into the Canvas Terminal tab.

Why this matters beyond convenience:

→ No long-lived SSH keys to manage or rotate
→ IAM policy controls access (condition on `aws:ResourceTag/Role=workspace`)
→ EIC audit log gives you provenance on every tunnel open event
→ Temporary keys auto-expire

Your agent workspaces are now as easy to access as your browser tab — with better audit trails than a manually managed SSH key rotation process.

EC2 Instance Connect SSH is live now for all CP-provisioned workspaces.

---

## Visual Asset Specifications

1. **Terminal demo GIF** — Canvas Terminal tab showing bash prompt inside an EC2 workspace:
   - Canvas UI with a workspace node selected
   - Terminal tab open, showing `ec2-user@ip-10-0-x-x:~$` prompt
   - Optional: running `whoami` or `hostname` to show EC2 context
   - Format: GIF or looping MP4, max 10s
   - Dark theme, molecule navy background

2. **Architecture diagram** (optional for LI):
   - Canvas (browser) → WebSocket → Platform (Go) → `aws ec2-instance-connect ssh` → EIC Endpoint → EC2 Instance
   - Shows the tunnel path for audience who wants to understand the mechanism

---

## Campaign notes

**Audience:** DevOps, platform engineers, ML infrastructure teams running agents in AWS
**Tone:** Practical — the IAM/audit story is the differentiator for security-conscious buyers; the "one click" story is the differentiator for developer audience
**Differentiation:** No manual SSH key management vs. traditional bastion host approach
**Hashtags:** #AIAgents #MoleculeAI #AWS #EC2InstanceConnect #PlatformEngineering #DevOps
**CTA links:** docs pending (workspace-terminal.md docs need to be published)

---

## Self-review applied

- No timeline claims ("today", "just shipped", etc.) beyond what's confirmed in PR state
- No person names
- No benchmarks or performance claims
- CTA links marked as pending until docs confirm live