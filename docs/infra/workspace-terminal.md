# Workspace Terminal over EIC + SSH

Tracking: [molecule-core#1528](https://github.com/Molecule-AI/molecule-core/issues/1528) (resolved 2026-04-22)

**Status: live in prod** on hongmingwang tenant as of 2026-04-22. Verified end-to-end against the Hermes workspace EC2.

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

## Rollout (verified recipe)

Each AWS account (staging + prod, etc.) needs this once. The CP repo
ships `scripts/bootstrap-eic-terminal.sh` that automates everything
below — what's here is what the script does, in case you want to run
the steps by hand or audit it.

### 1. Infra (one-shot)

```bash
# From molecule-controlplane checkout (needs IAM admin creds):
./scripts/bootstrap-eic-terminal.sh <workspace-vpc-id> <region>
```

Creates (idempotent):
- EC2 Instance Connect **service-linked role** (`AWSServiceRoleForEC2InstanceConnect`)
- **Managed IAM policy** `MoleculeEICTerminal` (DescribeInstances + SendSSHPublicKey + OpenTunnel + CreateInstanceConnectEndpoint + DescribeInstanceConnectEndpoints)
- **IAM role + instance profile** `MoleculeTenantEICRole` / `MoleculeTenantEICProfile` (attach the managed policy) — this replaces env-var AWS creds on tenant EC2s
- **EIC Endpoint** in the workspace VPC (uses the default VPC SG for egress, which is all EIC Endpoint needs)

Script prints the endpoint SG id + profile name to set on the CP:

```
EIC_ENDPOINT_SG_ID=sg-xxxxxx
EC2_TENANT_IAM_PROFILE=MoleculeTenantEICProfile
```

### 2. CP config + redeploy

Set those two env vars on the CP service (Railway dashboard or equivalent). On redeploy, [molecule-controlplane#227](https://github.com/Molecule-AI/molecule-controlplane/pull/227) ensures every **newly-provisioned** workspace + tenant SG auto-carries a `22/tcp` ingress rule sourced from the EIC Endpoint SG.

### 3. Tenant env vars (every tenant EC2)

The tenant workspace-server container needs these env vars to verify session cookies and reach the CP. Missing any of these produces a working-looking tenant whose canvas cold-loads with `401 admin auth required` on every call — which is what broke the hongmingwang tenant on 2026-04-22 before these were set.

| Env var | Value | What breaks if missing |
|---|---|---|
| `CP_UPSTREAM_URL` | `https://api.moleculesai.app` (or your CP) | `/cp/*` paths fall through to Next.js 404 → canvas `AuthGate` infinite-redirects on login, hits browser's 431 header-limit |
| `MOLECULE_ORG_SLUG` | tenant slug, e.g. `hongmingwang` | `verifiedCPSession` returns false — session cookie never validates, every API call 401s with "admin auth required" |
| `MOLECULE_ORG_ID` | UUID of the tenant org | `tenant_guard` middleware 404s all non-`/cp/*` routes |
| `AWS_REGION` | e.g. `us-east-2` | `aws ec2-instance-connect` subprocesses default to `us-east-1` and can't find instances |

Tenants launched by CP should have `MOLECULE_ORG_ID` + `MOLECULE_ORG_SLUG` injected from the `organizations` row at provision time. If you find a tenant where these are missing, that's a CP provisioner bug, not operator error.

AWS creds are NOT on this list because the instance profile (`MoleculeTenantEICProfile` from step 1) provides them via IMDSv2 — aws-cli inside the tenant container picks them up automatically. If you still see `AWS_ACCESS_KEY_ID` env vars on a tenant, strip them and rely on the profile.

### 4. Backfill existing instances

Pre-existing SGs need one-time ingress added. The bootstrap script's final output includes this loop with the real SG id substituted; shown here for visibility — **replace `<EIC_ENDPOINT_SG_ID>` with the `sg-…` value step 1 printed**:

```bash
EIC_SG=<EIC_ENDPOINT_SG_ID>  # from step 1 output

for sg in $(aws ec2 describe-security-groups --region us-east-2 \
    --filters 'Name=tag:ManagedBy,Values=molecule-cp' \
    --query 'SecurityGroups[].GroupId' --output text | tr '\t' '\n'); do
  aws ec2 authorize-security-group-ingress --region us-east-2 \
    --group-id "$sg" --protocol tcp --port 22 --source-group "$EIC_SG" \
    2>&1 | grep -v DuplicatePermission || true
done
```

Note the `| tr '\t' '\n'` — aws-cli `--output text` tab-separates values within a row, which can concatenate all SG ids into a single word that breaks the for loop. Splitting to newlines is a no-op on well-behaved output and a fix on the concatenated case.

### 5. Tenant code (this monorepo)

Already merged:
- [#1531](https://github.com/Molecule-AI/molecule-core/pull/1531) — migration `038_workspace_instance_id` + persist on CP provision
- [#1533](https://github.com/Molecule-AI/molecule-core/pull/1533) — terminal handler remote branch (EIC open-tunnel + ssh + pty)

Tenant image (`ghcr.io/molecule-ai/platform-tenant:latest`) ships with `aws-cli` + `openssh-client` as of 2026-04-22.

### 6. Verification (how to confirm after deploy)

- Provision a fresh CP workspace → `SELECT instance_id FROM workspaces WHERE id = ?` is non-null
- Open canvas Terminal on that workspace → bash prompt (`ubuntu@ip-...`)
- Terminate the workspace EC2 manually → Terminal shows "EIC tunnel didn't come up"
- Temporarily remove `ec2-instance-connect:OpenTunnel` from `MoleculeEICTerminal` → Terminal shows "failed to push session key"

### Existing-workspace backfill of `instance_id`

Migrations run on tenant boot, but pre-existing workspace rows have NULL `instance_id`. The CP provisioner only writes `instance_id` on NEW provisions; old workspaces need:

```sql
-- Inside the tenant DB
UPDATE workspaces SET instance_id = '<i-xxx from DescribeInstances by tag WorkspaceID>', updated_at = now()
WHERE id = '<workspace-uuid>';
```

For a whole fleet, join CP's workspace table with the DescribeInstances result by `WorkspaceID` tag and batch-UPDATE.

## Future work (not in scope)

- Session recording for compliance → SSM migration with instance profile
- Multi-user concurrent terminals → connection pooling per workspace
- Terminal for workspaces behind a private NAT with no EIC route → fall back to SSM
