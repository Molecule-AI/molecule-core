---
title: "EC2 Instance Connect SSH: Shell Access Without Opening Inbound Ports"
date: 2026-04-22
slug: ec2-instance-connect-ssh
description: "Access a shell on CP-provisioned workspaces without opening inbound SSH ports. Molecule AI integrates AWS EC2 Instance Connect Endpoint — the Terminal tab connects via a signed, ephemeral tunnel."
og_image: /docs/assets/blog/2026-04-22-ec2-instance-connect-ssh-og.png
tags: [EC2, terminal, SSH, self-hosted, DevOps, security, Canvas]
keywords: [EC2 Instance Connect, AI agent SSH access, SSH bastion host alternative, AI agent terminal, EC2 SSH]
---

<script type="application/ld+json">
{
  "@context": "https://schema.org",
  "@type": "Article",
  "headline": "EC2 Instance Connect SSH: Shell Access Without Opening Inbound Ports",
  "description": "Access a shell on CP-provisioned workspaces without opening inbound SSH ports. Molecule AI integrates AWS EC2 Instance Connect Endpoint with ephemeral signed tunnels.",
  "author": { "@type": "Organization", "name": "Molecule AI" },
  "datePublished": "2026-04-22",
  "publisher": {
    "@type": "Organization",
    "name": "Molecule AI",
    "logo": { "@type": "ImageObject", "url": "https://molecule.ai/logo.png" }
  }
}
</script>

# EC2 Instance Connect SSH: Shell Access Without Opening Inbound Ports


Traditional SSH to a cloud instance requires one of two setups:

- **Inbound port 22 open** on a security group — effective, but a permanent attack surface
- **Bastion host / jump box** — adds latency, complexity, and another credential surface to manage

EIC Endpoint takes a different approach. Instead of opening a door, it establishes a *signed, time-limited tunnel* from the AWS platform side. The connection is initiated by the AWS API, not by an inbound request — there is nothing to port-scan or brute-force from the internet.

For Molecule AI, this means the Terminal tab in Canvas connects to a CP workspace via a tunnel that:

- Requires no inbound SSH port on the instance's security group
- Uses short-lived, automatically-rotated credentials (AWS handles this under the hood)
- Closes when the session ends — no persistent tunnel to manage
- Works for any instance in a VPC that has an EIC Endpoint configured

The result is a secure shell that feels like a local terminal, without the operational overhead of maintaining a jump box or exposing port 22.

## How It Works in Molecule AI

When you open the Terminal tab in Canvas for a CP-provisioned workspace, the platform determines which path to use:

```
HandleConnect(workspace)
├── SELECT instance_id FROM workspaces WHERE id = workspace_id
├── instance_id IS NULL  →  local Docker path (existing behavior, unchanged)
└── instance_id IS SET   →  handleRemoteConnect
                            └── aws ec2-instance-connect ssh \
                                  --connection-type eice \
                                  --instance-id <instance-id> \
                                  --os-user ec2-user \
                                  -- docker exec -it <container-id> /bin/bash
                            └── PTY (creack/pty) wraps the session for TTY semantics
                            └── PTY ↔ Canvas WebSocket bridge

> **Note:** PR [#1531](https://github.com/Molecule-AI/molecule-core/pull/1531) persists the `instance_id` on the workspace record at provisioning time — so the Terminal tab can route to the EIC Endpoint path without a separate lookup at session open.
```

The `sshCommandFactory` is a configurable variable — the tests stub it to avoid spawning real `aws-cli` processes during CI, but in production it produces the command above. The subprocess bridges the PTY to the Canvas WebSocket, so the browser terminal and the remote bash session stay in sync.

Context cancellation is bidirectional: closing the WebSocket terminates the SSH process, and an SSH exit closes the WebSocket. No orphaned processes.

## Why DevOps Teams Care

CP-provisioned workspaces run in your own VPC. Without EIC Endpoint, reaching them from the Canvas UI required either inbound SSH (with all its management overhead) or a VPN/bastion setup. Neither is ideal for a platform that teams are expected to self-serve.

EIC Endpoint gives your platform operators a secure, audited way to open a shell on any workspace — from Canvas, without leaving the browser. The SSH session shows up in your AWS CloudTrail logs as an `ec2-instance-connect:OpenTunnel` event, linked to the IAM principal that initiated it.

This matters for several scenarios:

**Debugging an agent's environment** — The agent reports success, but something in the configuration looks wrong. Open the Terminal tab, run `env | grep MOLECULE`, inspect the filesystem. No separate SSH client needed.

**Verifying provisioning state** — Before handing a workspace to a team, an operator can confirm the right IAM role, security group, and runtime image are attached — by checking from inside the instance, not from the outside.

**Incident response** — If an agent goes off-script or a workspace becomes unresponsive, a DevOps engineer can get a shell without escalating a ticket to infra. The same IAM credentials that provisioned the workspace authorize the tunnel; no separate sudo access to manage.

## IAM Configuration Checklist

EIC Endpoint requires two IAM permissions on the `molecule-cp` role — the provisioning principal for CP workspaces:

```json
{
  "Effect": "Allow",
  "Action": [
    "ec2-instance-connect:SendSSHPublicKey",
    "ec2-instance-connect:OpenTunnel"
  ],
  "Condition": {
    "StringEquals": {
      "aws:ResourceTag/Role": "workspace"
    }
  }
}
```

The `aws:ResourceTag/Role=workspace` condition scopes the permission to instances tagged with the workspace role — instances that Molecule AI provisioned. It does not grant access to other EC2 instances in the same account.

You also need one **EIC Endpoint** in the workspace VPC, associated with the subnet(s) where workspaces run. The endpoint is a regional, VPC-scoped resource — create it once per VPC, not per instance.

If the Terminal tab shows a "check tenant AWS CLI + IAM" hint instead of a bash prompt, the EIC wiring is incomplete. Verify:

- [ ] `molecule-cp` IAM role has `ec2-instance-connect:SendSSHPublicKey` and `ec2-instance-connect:OpenTunnel`
- [ ] Condition tag `Role=workspace` is attached to the workspace instance(s)
- [ ] An EIC Endpoint exists in the workspace VPC, in the same Availability Zone as the instance
- [ ] Tenant image includes `aws-cli` and `openssh-client` (included in the default Molecule AI tenant image via `apk add`)

## Getting Started

For teams using the Molecule AI hosted SaaS, EIC Endpoint terminal access is enabled automatically — no IAM configuration needed on your end.

For self-hosted deployments on EC2:

1. Add the IAM policy above to your `molecule-cp` role
2. Tag workspace instances with `Role=workspace`
3. Create an EIC Endpoint in the workspace VPC (one time per VPC)
4. Open the Terminal tab on any CP-provisioned workspace in Canvas

For platform operators: the `sshCommandFactory` variable in `workspace-server/internal/handlers/terminal.go` is the injection point for testing. Swap it for a stub in unit tests to avoid spawning real `aws-cli` processes — the PR ships `TestSshCommandFactory_BuildsEICCommand` as a regression guard on the argv shape.

---

*Molecule AI is open source. EC2 Instance Connect Endpoint support shipped in PR [#1533](https://github.com/Molecule-AI/molecule-core/pull/1533). The design is documented in `docs/infra/workspace-terminal.md`.*