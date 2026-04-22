# Workspace Terminal over EIC + SSH

Tracking: [molecule-core#1528](https://github.com/Molecule-AI/molecule-core/issues/1528)

## Problem

Canvas's Terminal tab calls `workspace-server /workspaces/:id/terminal` which tries `docker.ContainerInspect` on the tenant's local Docker daemon. That works for locally-provisioned workspaces, but CP-provisioned (SaaS) workspaces run on **separate EC2 instances** — the tenant has no path to their Docker. Users see "Failed to connect — is the workspace container running?" while `STATUS: online` because A2A heartbeats come from the remote instance independently.

## Chosen approach: EC2 Instance Connect + SSH

`ec2-instance-connect:SendSSHPublicKey` pushes an ephemeral SSH public key (valid 60s) into the instance's metadata. A short-lived SSH connection uses the matching private key, runs `docker exec -it ws-<id> /bin/bash`, and bridges stdin/stdout to the canvas WebSocket.

### Why not SSM Session Manager

SSM would be the "right" answer in a mature infra but requires:
- An IAM instance profile with `AmazonSSMManagedInstanceCore` on every workspace EC2 (currently none have one — `aws ssm describe-instance-information` returns an empty list across the fleet)
- SSM agent on the AMI (already present on AL2023/Ubuntu, but unverified)
- Outbound to `ssm.*.amazonaws.com` (current VPC config unknown)

EIC short-circuits all three. The existing `molecule-cp` IAM user picks up a small policy addition and we're done — no per-instance identity to bootstrap.

### Comparison

| Axis | EIC + SSH | SSM Session Manager |
|---|---|---|
| Uses existing `molecule-cp` creds | Yes | No — needs instance profile |
| AMI changes | None (EIC in OS since AL2 2019+, Ubuntu 20.04+) | Verify agent present |
| Infra changes | IAM policy + security group | IAM role + instance profile + maybe NAT/VPCe |
| Audit | CloudTrail for `SendSSHPublicKey` | CloudTrail + SSM session logs (richer) |
| Rotation | Every session (60s key lifetime) | Managed by AWS |
| Compliance story | "SSH with per-session keys, CloudTrailed" | "SSM Session Manager with recording available" |

Pick SSM later if compliance needs session recording. For now EIC is strictly less work.

## Data flow

```
[Canvas]                [Tenant workspace-server]                    [Workspace EC2]
   │                             │                                         │
   │ WS /workspaces/:id/terminal │                                         │
   ├────────────────────────────▶│                                         │
   │                             │ SELECT instance_id                      │
   │                             │ FROM workspaces WHERE id=:id            │
   │                             │                                         │
   │                             │ ec2:DescribeInstances(instance_id)      │
   │                             │ → public_dns, availability_zone, az     │
   │                             │                                         │
   │                             │ ec2-instance-connect:SendSSHPublicKey   │
   │                             │   target: instance_id                   │
   │                             │   os_user: ec2-user|ubuntu              │
   │                             │   public_key: ephemeral (ed25519)       │
   │                             │                                         │
   │                             │ ssh ec2-user@public_dns                 │
   │                             │     -o StrictHostKeyChecking=no         │
   │                             ├────────────────────────────────────────▶│
   │                             │                                         │
   │                             │ docker exec -it ws-<id> /bin/bash       │
   │                             ├────────────────────────────────────────▶│
   │                             │                                         │
   │◀───── stdout bridge ────────┤◀──────────── stdout ────────────────────┤
   │───── stdin bridge ─────────▶│───────────── stdin ─────────────────────▶│
```

`instance_id` is persisted on provision by migration `038_workspace_instance_id`. Terminal handler branches on `instance_id IS NOT NULL`.

## Topology (verified from molecule-controlplane code)

- Workspaces launch in a **shared workspace VPC** (`p.VPCID`), not the tenant's VPC
- Each workspace gets its own SG created by `createPerTenantSG("workspace", <ws-short>, workspaceIngressRules())`
- Current `workspaceIngressRules()` opens only `8000/tcp` from `0.0.0.0/0` — no port 22
- CP already tags every workspace instance with `Role=workspace` (+ `WorkspaceID`, `Runtime`, `SGID`, `ManagedBy=molecule-cp`)

Because tenant EC2 and workspace EC2 are in **different VPCs**, a direct SG CIDR rule for port 22 is awkward (would require VPC peering + tenant-CIDR bookkeeping). **EIC Endpoint** is the natural fit — it's a VPC resource that acts as a TLS tunnel to any instance in its VPC, keyed on IAM permissions rather than source CIDR.

## IAM policy addition for `molecule-cp`

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "DescribeInstancesForTerminalResolution",
      "Effect": "Allow",
      "Action": ["ec2:DescribeInstances"],
      "Resource": "*"
    },
    {
      "Sid": "PushEphemeralSSHKeyToWorkspaceInstances",
      "Effect": "Allow",
      "Action": [
        "ec2-instance-connect:SendSSHPublicKey",
        "ec2-instance-connect:OpenTunnel"
      ],
      "Resource": "arn:aws:ec2:*:*:instance/*",
      "Condition": {
        "StringEquals": {
          "aws:ResourceTag/Role": "workspace"
        }
      }
    }
  ]
}
```

Tag key is **`Role`** (capitalized) — CP already sets this at launch in `ec2.go:1126`. No CP change needed for the policy's scoping to work fleet-wide.

## EIC Endpoint (one-time setup in the workspace VPC)

```bash
aws ec2 create-instance-connect-endpoint \
  --subnet-id <any-subnet-in-workspace-vpc> \
  --security-group-ids <sg-id-allowing-egress-only> \
  --tag-specifications 'ResourceType=instance-connect-endpoint,Tags=[{Key=Name,Value=molecule-workspace-eic}]'
```

One endpoint per workspace VPC. Free for the resource (pay only for data transferred). Replaces both "open port 22 in every SG" and "establish VPC peering for tenant→workspace SSH" — no change to `workspaceIngressRules()` needed, no change to tenant VPC routing needed.

## Alternative: direct SG rule (not recommended)

If you really want direct SSH instead of EIC Endpoint:

1. Add `22/tcp` to `workspaceIngressRules()` in `molecule-controlplane`, sourced from the tenant VPC's CIDR
2. Establish VPC peering between tenant VPC and workspace VPC
3. Update the route tables on both sides

Three more failure modes + ongoing bookkeeping per tenant. Skip unless you have a specific reason EIC Endpoint doesn't fit.

## Key lifetime

- ed25519 keypair generated per-session in the terminal handler
- Public half pushed via `SendSSHPublicKey` (valid 60s)
- Private half held in-memory only, discarded when the WS closes
- No keys on disk, no rotation cron, no secrets rotation debt

## Failure modes + their user-visible messages

| Condition | Message | Actionable? |
|---|---|---|
| `instance_id IS NULL` (local workspace) | Falls through to current local-Docker handler | n/a — existing behavior |
| `instance_id` set, DescribeInstances returns nothing | "workspace instance no longer exists — recreate the workspace" | Yes |
| `SendSSHPublicKey` 403 | "tenant lacks EIC permission — contact your admin" | Yes (requires IAM fix) |
| SSH connect timeout | "tenant cannot reach workspace instance — check security group" | Yes (SG fix) |
| `docker exec` fails (no container) | "workspace container is not running — try restart" | Yes (normal ops) |

## Rollout checklist

### 1. Infra prep (one-time)

- [ ] Add IAM policy above to `molecule-cp` user (tag key is `Role`, already set by CP at launch — no CP change needed)
- [ ] Create one EIC Endpoint in the workspace VPC (see command above)
- [ ] No change to `workspaceIngressRules()` — EIC Endpoint bypasses SG ingress

### 2. Tenant code (this repo)

- [ ] PR 1 (this one): migration `038_workspace_instance_id` + persist instance_id on CP provision
- [ ] PR 2 (follow-up): terminal handler EIC + SSH branch + tests

### 3. Verification

- [ ] After PR 1 merges + deploys, provision a new CP workspace → verify `SELECT instance_id FROM workspaces` returns the EC2 id
- [ ] After PR 2 merges + deploys, open Terminal tab on a CP workspace → bash prompt appears
- [ ] Intentionally terminate the EC2 → Terminal tab shows the "instance no longer exists" message
- [ ] Pull the `ec2-instance-connect:OpenTunnel` action from molecule-cp temporarily → Terminal shows "tenant lacks EIC permission"

## Future work (not in scope)

- Session recording for compliance → SSM migration with instance profile
- Multi-user concurrent terminals → connection pooling per workspace
- Terminal for workspaces behind a private NAT with no EIC route → fall back to SSM
