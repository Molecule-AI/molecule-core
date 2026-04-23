---
title: "EC2 Workspace Terminal: No SSH Keys, No Bastion, No Config"
slug: ec2-instance-connect-ssh
date: 2026-04-23
authors: [molecule-ai]
tags: [ec2, ssh, devops, mcp, terminal, aws]
description: "Molecule AI's SaaS workspaces now include a browser terminal backed by EC2 Instance Connect. IAM-authorized, private-subnet-compatible, zero SSH key management."
og_image: /assets/blog/2026-04-23-ec2-ssh/og.png
---

When Molecule AI launched Docker-based workspaces, a built-in terminal tab came along for free — the container runtime made it easy to attach a PTY and proxy it to the browser. EC2 workspaces were a different story. Getting a shell meant reaching for one of the standard playbooks: distribute an SSH key pair, stand up a bastion host, or enroll the instance in AWS Systems Manager Session Manager. Each approach works, but none of them is frictionless. SSH key distribution turns into a rotation problem the moment a team grows past two people. Bastions add infrastructure to maintain and a hop that every operator has to remember. SSM requires the SSM agent, an instance profile with the right policies, and a region-aware endpoint — reasonable for a dedicated platform team, but steep overhead for a developer who just wants to inspect a running workspace.

The gap between Docker workspaces (click, get a shell) and EC2 workspaces (file a ticket, get a key) was real, and it showed up in support requests every week.

## A terminal tab that just appears

As of Phase 30 (PR #1533, merged 2026-04-22), every CP-provisioned EC2 workspace has a Terminal tab in the canvas. No setup step, no opt-in toggle, no per-user configuration. The tab appears because the platform now handles the entire connection lifecycle on your behalf, using AWS EC2 Instance Connect Endpoint (EICE).

Here is what happens in the roughly three seconds between clicking the tab and seeing a prompt:

```
Browser                   Molecule Platform              AWS
  │                              │                         │
  │── open terminal tab ────────>│                         │
  │                              │── generate RSA keypair  │
  │                              │── push public key ─────>│ EC2 Instance Connect API
  │                              │   (60-second TTL)       │   (instance metadata)
  │                              │── open EICE tunnel ────>│ EC2 Instance Connect Endpoint
  │<── WebSocket (PTY) ──────────│<────── TCP/22 ──────────│──> EC2 instance
  │                              │                         │
  interactive shell ready        │                         │
```

The platform generates a fresh RSA key pair per session, pushes the public half to the instance via the EC2 Instance Connect API, and opens a proxied PTY over WebSocket. The temporary key expires after 60 seconds regardless — by the time it does, the SSH handshake has already completed and the session is live on the persistent WebSocket connection.

## No keys to manage

The RSA key pair is ephemeral and scoped to a single session. There is no long-lived private key to rotate, no `~/.ssh/authorized_keys` file to audit, and no shared credential that accumulates across team members. Each terminal session gets its own key, uses it once, and discards it. If a session is never opened, no key is ever created.

## Private subnets, no internet egress required

EICE routes the SSH connection through AWS's internal network rather than the public internet. EC2 instances that live in private subnets — no public IP, no internet gateway route — are fully reachable. The only requirement is that an EC2 Instance Connect Endpoint exists in the VPC, which Molecule's provisioning layer creates automatically for CP-managed workspaces. Your instances do not need inbound rules on port 22 from the internet, and they do not need to reach out to any external endpoint to register themselves.

## IAM is the control plane

Authorization flows through the platform's IAM credentials, not through a shared secret or a separate access-control layer bolted onto SSH. The EICE handshake is an AWS API call — it appears in CloudTrail like any other IAM-authorized action. If you need to answer "who opened a terminal session to workspace X at time T," the answer is in your CloudTrail logs, attributed to the platform role, correlated with whatever session identity your IdP provides to Molecule.

This is a meaningful difference from the traditional bastion-plus-shared-key model, where SSH access control and your IAM policy boundary are two separate things that drift apart over time. With EICE, there is one authorization path, and it is the same path that governs the rest of your AWS estate.

## Getting started

The terminal tab is available now on all EC2 workspaces provisioned through the Molecule platform. No action is required to enable it. For a full description of the connection architecture, networking prerequisites, and troubleshooting guidance, see the [workspace terminal documentation](../../infra/workspace-terminal.md).
