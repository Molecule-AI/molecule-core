# EICE Terminal Demo — PR #1533

**EC2 Instance Connect Endpoint (EICE) browser terminal for CP-provisioned workspaces.**

| | |
|---|---|
| **PR** | [#1533 — feat(terminal): remote path via aws ec2-instance-connect + pty](https://github.com/Molecule-AI/molecule-core/pull/1533) |
| **Issue** | [#1545 — devrel: code demo for EC2 instance-connect SSH](https://github.com/Molecule-AI/molecule-core/issues/1545) |
| **Design doc** | [docs/infra/workspace-terminal.md](https://github.com/Molecule-AI/molecule-core/blob/main/docs/infra/workspace-terminal.md) |
| **Run time** | ~1 min (dry-run mode, no real instance needed) |

---

## What this solves

Canvas's Terminal tab worked for locally-provisioned workspaces (Docker daemon on the same machine) but broke for Cloud Provisioning (CP)-provisioned workspaces, which run on separate EC2 instances. Users saw:

> "Failed to connect — is the workspace container running?"

...while `STATUS: online` because A2A heartbeats come from the remote instance independently.

**EICE SSH bridges the gap** — a 3-step flow that needs no port 22 in security groups, no bastion host, and no per-instance IAM profiles:

```
1. Push ephemeral public key via EIC API
       ↓ (key in instance metadata, valid 60s)
2. Open TLS tunnel via EIC Endpoint
       ↓ (no inbound SSH ports needed)
3. SSH through tunnel → docker exec → bash
```

---

## Architecture

```
┌─────────────┐         ┌──────────────────┐         ┌──────────────┐
│  Canvas     │  WS     │  workspace-server │  EIC    │  Workspace   │
│  (browser)  │──────▶  │  HandleConnect    │────────▶  EC2 (i-xxx) │
│             │         │  ├─ EIC key push   │         │  ubuntu@host │
│  PTY ↔ WS   │◀────────│  ├─ EIC tunnel     │         │  docker exec │
└─────────────┘         │  └─ ssh + pty      │         │  ws-<id>     │
                        └──────────────────┘         └──────────────┘
                           IAM molecule-cp
                           (SendSSHPublicKey + OpenTunnel)
```

### Key design decisions

| Decision | Rationale |
|---|---|
| `aws-cli` subprocess over native SDK | EIC Endpoint uses a signed WebSocket protocol that `aws-cli v2` implements correctly — ~500 fewer lines of crypto to maintain |
| Ed25519 keypair, temp dir, never `~/.ssh` | Ephemeral — auto-cleaned on session close; no secrets rotation debt |
| EIC Endpoint over SG ingress rule | One endpoint per VPC vs per-tenant-per-workspace peering + CIDR bookkeeping |
| Fallback to local Docker when `instance_id` NULL | Existing behavior preserved, no migration risk |

---

## Prerequisites

### One-time infra setup (for real EICE access)

1. **IAM policy on `molecule-cp`** — add these actions to the existing policy:

```json
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
    "StringEquals": { "aws:ResourceTag/Role": "workspace" }
  }
}
```

2. **Create EIC Endpoint** in the workspace VPC (one-time, free):

```bash
aws ec2 create-instance-connect-endpoint \
  --subnet-id <any-subnet-in-workspace-vpc> \
  --security-group-ids <sg-id> \
  --tag-specifications 'ResourceType=instance-connect-endpoint,Tags=[{Key=Name,Value=molecule-workspace-eic}]'
```

3. **Verify** a new CP workspace gets `instance_id` populated:

```sql
SELECT id, name, instance_id FROM workspaces
  WHERE instance_id IS NOT NULL LIMIT 5;
```

### Runtime dependencies

- `aws-cli v2` (`aws --version` → should include `aws-cli/2.x.x`)
- `ssh-keygen` (OpenSSH)
- Python 3.10+ with `pip install boto3 websockets` (optional, for SDK path)
- A CP-provisioned workspace with `instance_id` set (or `--dry-run` to skip)

---

## Running the demo

### Option A — Dry run (no real AWS or workspace needed)

```bash
cd docs/devrel/demos/eice-terminal
python3 eice_terminal_demo.py \
  --workspace-id ws-demo-001 \
  --instance-id i-0123456789abcdef0 \
  --region us-east-2 \
  --dry-run
```

Expected output:

```
============================================================
  EICE Terminal Demo  |  PR #1533  |  molecule-core
============================================================

  workspace : ws-demo-001
  instance  : i-0123456789abcdef0
  region    : us-east-2
  dry-run   : True

[Pre-flight] Verify instance exists in us-east-2
  ⚠ boto3 not installed — skipping AWS SDK check

[Step 0] Generate ephemeral Ed25519 keypair
  ✓ Generated keypair in /var/folders/.../tmpXXXX (auto-cleaned on exit)

  Public key (pushed to instance metadata):
    ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA...

[Step 1] Push ephemeral SSH public key to instance via EIC
  Command: aws ec2-instance-connect send-ssh-public-key --instance-id ...
  → DRY RUN — would call EIC to register key (valid 60s)
  ✓ Key accepted by instance metadata (valid 60s)

[Step 2] Open EIC tunnel to instance port 22
  Command: aws ec2-instance-connect open-tunnel --instance-id ...
  → DRY RUN — would open tunnel on localhost:54321
  ✓ Tunnel listening on localhost:54321

[Canvas path] WebSocket → wss://app.molecule.ai/api/workspaces/ws-demo-001/terminal
  The server handles EIC key push + tunnel + SSH internally.
  PTY bytes flow: EC2 sshd → molecule-server PTY bridge → WebSocket → browser
  (To verify: open browser DevTools → Network → WS, filter /terminal)
```

### Option B — Live demo (requires real CP workspace + IAM wiring)

```bash
# Set your instance ID (from SELECT instance_id FROM workspaces WHERE ...)
export INSTANCE_ID=i-0123456789abcdef0
export AWS_REGION=us-east-2
export WORKSPACE_ID=$(your-workspace-id)

python3 eice_terminal_demo.py \
  --workspace-id $WORKSPACE_ID \
  --instance-id $INSTANCE_ID \
  --region $AWS_REGION
  # (no --dry-run — will open interactive SSH)
```

### Verifying in the browser

1. Open a CP-provisioned workspace in Canvas
2. Click the **Terminal** tab
3. You should see a bash prompt (`ubuntu@ip-...:~$`)
4. `exit` closes the session cleanly
5. Open DevTools → Network → WS, filter `terminal` to see the PTY frames

### Verifying failure modes

| Action | Expected message |
|---|---|
| Remove EIC permissions from `molecule-cp` | "Error: failed to push session key (check tenant IAM + see docs/infra/workspace-terminal.md)" |
| Terminate the EC2 instance | "workspace instance no longer exists — recreate the workspace" |
| Remove EIC Endpoint | "Error: failed to open EIC tunnel (check EIC Endpoint + SG 22 from endpoint SG)" |

---

## File map

```
molecule-core/
├── workspace-server/internal/handlers/terminal.go  ← server-side implementation
├── workspace-server/internal/handlers/terminal_test.go ← 3 unit tests
├── docs/infra/workspace-terminal.md                  ← design doc
├── docs/tutorials/workspace-terminal-ieee.md         ← tutorial reference
└── docs/devrel/demos/eice-terminal/
    ├── README.md                  ← this file
    └── eice_terminal_demo.py      ← Python CLI demo
```

---

## The three-step flow (what `handleRemoteConnect` does)

```python
# 1. Push ephemeral public key — valid 60s on the instance
subprocess.run([
    "aws", "ec2-instance-connect", "send-ssh-public-key",
    "--instance-id", instance_id,
    "--region", region,
    "--instance-os-user", os_user,
    "--ssh-public-key", pub_key,
])

# 2. Open tunnel — TLS connection to EC2 :22 via EIC Endpoint
tunnel = subprocess.Popen([
    "aws", "ec2-instance-connect", "open-tunnel",
    "--instance-id", instance_id,
    "--region", region,
])
local_port = parse_port_from_stdout(tunnel)  # e.g. 54321

# 3. SSH through tunnel, wrapped in PTY for interactive terminal
subprocess.run([
    "ssh",
    "-i", private_key_path,       # ephemeral key, temp dir
    "-p", str(local_port),
    "-o", "StrictHostKeyChecking=no",
    f"{os_user}@127.0.0.1",       # tunnel endpoint
])

# PTY stdout/stderr bridged ↔ WebSocket in the Go handler
```

---

## Running via WebSocket (for tooling / API consumers)

```python
import asyncio, websockets

async def open_terminal(workspace_id: str, server_url: str = "wss://app.molecule.ai"):
    """Programmatic terminal access — for tooling, not interactive use."""
    async with websockets.connect(
        f"{server_url}/api/workspaces/{workspace_id}/terminal"
    ) as ws:
        await ws.send(b"echo hello from API client\n")
        async for msg in ws:
            print(msg.decode(), end="")

asyncio.run(open_terminal("ws-your-id-here"))
```

> **Note:** The WebSocket path goes through `HandleConnect` which handles all three EIC steps server-side. You only need the WS URL — the IAM, key push, and tunnel are all managed by workspace-server.