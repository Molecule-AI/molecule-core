---
title: "Browser Terminal for EC2 Workspaces — No SSH Key Management Required"
date: 2026-04-22
slug: ec2-instance-connect-terminal
description: "Molecule AI workspaces running on EC2 now get a browser-based terminal tab with zero credential infrastructure. Here's how EC2 Instance Connect Endpoint replaced bastion hosts, VPN tunnels, and ~/.ssh/ bookkeeping."
tags: [platform, EC2, security, SSH, devops]
---

Molecule AI's Terminal tab — the one inside the Canvas workspace UI — used to stop working the moment a workspace landed on an EC2 instance instead of a local Docker daemon. The fix shipped in [PR #1533](https://github.com/Molecule-AI/molecule-core/pull/1533) and uses a single AWS primitive to bridge the gap: **EC2 Instance Connect Endpoint (EICE)**.

This post covers what EICE is, why it beats the alternatives, and what you need to do to get the terminal working on your EC2-provisioned workspaces.

## The problem: Docker path doesn't reach EC2

Canvas connects to `/workspaces/:id/terminal` on the workspace-server. For locally-provisioned workspaces, the handler runs `docker exec -it <container> /bin/bash` against the local Docker daemon — no network involved.

For Cloud Provisioning (CP)-provisioned workspaces, the workspace runs on a separate EC2 instance in the Molecule control plane's VPC. Your tenant's Docker daemon has no path to it. Users saw:

> Failed to connect — is the workspace container running?

...while the workspace showed `STATUS: online` (A2A heartbeats route independently).

## The old solutions all had trade-offs

| Approach | Problem |
|---|---|
| Open port 22 in workspace SG, tenant SSH in | Requires VPC peering or CIDR allow-listing per tenant. Ongoing bookkeeping. |
| SSM Session Manager | Needs an IAM instance profile on every workspace EC2 — none exist today. SSM agent status unverified. Outbound to `ssm.*.amazonaws.com` depends on VPC config. |
| Bastion host | Operationally heavy. Another machine to maintain, patch, rotate. |

EICE solves all three:

- Uses the existing `molecule-cp` IAM user (no per-instance profile needed)
- No inbound port 22 in any security group
- EIC Endpoint is a VPC-scoped resource, not per-instance — one endpoint covers the whole workspace fleet
- Key lifetime is 60 seconds, key lives in instance metadata, private key never touches disk

## How EICE SSH works

The flow has three steps, all handled by the workspace-server when you open the Terminal tab:

```
1. Generate ephemeral Ed25519 keypair (temp dir, auto-cleaned on close)
2. Push public key to instance metadata via EIC API
       ↓  (valid 60 seconds)
3. Open TLS tunnel via EIC Endpoint (port 22 on the instance)
       ↓
4. SSH over the tunnel → docker exec → bash
```

The PTY (pseudo-terminal) is bridged to the Canvas WebSocket, so the terminal behaves like a normal interactive shell — tab completion, ANSI colors, history all work.

### The IAM policy you need (one-time)

The `molecule-cp` IAM user already exists. Add this to its policy:

```json
{
  "Sid": "WorkspaceTerminalEICE",
  "Effect": "Allow",
  "Action": [
    "ec2:DescribeInstances",
    "ec2-instance-connect:SendSSHPublicKey",
    "ec2-instance-connect:OpenTunnel"
  ],
  "Resource": "arn:aws:ec2:*:*:instance/*",
  "Condition": {
    "StringEquals": { "aws:ResourceTag/Role": "workspace" }
  }
}
```

The tag condition `aws:ResourceTag/Role=workspace` scopes the permission to workspace EC2s only. Molecule's control plane sets this tag at launch — no CP changes needed.

### One EIC Endpoint per workspace VPC (also one-time)

```bash
aws ec2 create-instance-connect-endpoint \
  --subnet-id <any-subnet-in-the-workspace-vpc> \
  --security-group-ids <sg-id-for-egress-only> \
  --tag-specifications 'ResourceType=instance-connect-endpoint,Tags=[{Key=Name,Value=molecule-workspace-eic}]'
```

One endpoint. Free to create. Pay only for data transferred through it. Replaces "open port 22 in every workspace SG."

## Verifying it's working

In Canvas, open a CP-provisioned workspace and click **Terminal**. You should see a bash prompt within 5 seconds:

```
ubuntu@ip-10-0-1-42:~$ echo $HOSTNAME
ws-a1b2c3d4
ubuntu@ip-10-0-1-42:~$ docker ps
CONTAINER ID   IMAGE                         COMMAND    CREATED
a1b2c3d4e5f6   ghcr.io/molecule-ai/...      "/bin/sh"  2 hours ago
```

To watch the frames: open browser DevTools → Network → WS, filter `/terminal`. You'll see binary PTY data flowing bidirectionally.

## Failure modes and what they mean

| What you see | What it means | Fix |
|---|---|---|
| `Error: failed to push session key (check tenant IAM)` | `molecule-cp` lacks EIC permissions | Add the IAM policy above |
| `Error: failed to open EIC tunnel` | EIC Endpoint not created, or SG doesn't allow outbound from the endpoint | Create the endpoint; check endpoint SG egress |
| `workspace instance no longer exists` | EC2 was terminated | Recreate the workspace |
| Bash prompt, but no `docker` binary | Workspace running as native process, not container | `ls /var/run/docker.sock` — if absent, workspace uses the direct process model |

## Self-hosted EC2 deployments

If you're running Molecule AI on your own AWS account with CP provisioning, the setup is identical: add the IAM policy to your `molecule-cp` user, create one EIC Endpoint in the workspace VPC, and the Terminal tab starts working for all existing and future CP workspaces.

The key difference from hosted: you control the `molecule-cp` credentials and the VPC topology. The EIC flow is the same either way.

## Architecture summary

```
Canvas UI  (WebSocket /terminal)
    ↓
workspace-server
    ├─ generate ephemeral keypair (temp dir)
    ├─ aws ec2-instance-connect send-ssh-public-key  (push to metadata, 60s TTL)
    ├─ aws ec2-instance-connect open-tunnel  (TLS → EC2 :22 via EIC Endpoint)
    └─ ssh -p <port> ubuntu@127.0.0.1  (PTY ↔ WebSocket bridge)
          ↓
      EC2 workspace instance (ec2-user, docker exec ws-<id> /bin/bash)
```

No keys on disk. No bastion host. No VPN. No per-instance IAM profiles. The setup takes about 10 minutes if your `molecule-cp` user already has basic EC2 permissions — add the EIC actions, create one endpoint, done.

For the full design doc and rollout checklist, see [docs/infra/workspace-terminal.md](https://github.com/Molecule-AI/molecule-core/blob/main/docs/infra/workspace-terminal.md) in the Molecule Core repo.