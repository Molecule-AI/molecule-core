# Tutorial: Add a Browser Terminal to CP-Provisioned Workspaces

PR [#1533](https://github.com/Molecule-AI/molecule-core/pull/1533) landed remote workspace terminal support for Cloud Provisioning (CP)-provisioned workspaces using AWS EC2 Instance Connect Endpoint (EICE). This tutorial explains the data flow and how to consume it from a client.

## How it works

When a workspace is CP-provisioned, the `instance_id` of the backing EC2 instance is stored in the `workspaces` table. The terminal handler detects this and routes the WebSocket connection through EICE instead of the local Docker path:

```
Canvas UI  (WebSocket)
  → molecule-server
    → aws ec2-instance-connect ssh --connection-type eice --instance-id <id> --
      → EC2 Instance Connect Endpoint
        → EC2 instance (ec2-user)
          → docker exec -it <workspace-id> /bin/bash
```

The `aws-cli v2` handles the EICE WebSocket handshake and signed credential injection. No native SDK needed server-side.

## Runnable snippet: open a terminal session

```python
import asyncio
import websockets

async def terminal_session(workspace_id: str, server_url: str = "wss://app.molecule.ai"):
    endpoint = f"{server_url}/api/workspaces/{workspace_id}/terminal"
    async with websockets.connect(endpoint) as ws:
        while True:
            data = await ws.recv()
            print(data, end="", flush=True)
```

## What to add to your client

1. **Detect the CP path** — no action needed on your end. The server checks `instance_id` in the DB and routes automatically.
2. **Grant IAM permissions** — the tenant's `molecule-cp` role needs `ec2-instance-connect:SendSSHPublicKey` + `ec2-instance-connect:OpenTunnel` with a tag condition `aws:ResourceTag/Role=workspace`. See [docs/infra/workspace-terminal.md](https://github.com/Molecule-AI/molecule-core/blob/main/docs/infra/workspace-terminal.md) for the full policy JSON.
3. **Open the Terminal tab** — the canvas `Terminal` tab connects to `/api/workspaces/{id}/terminal`. It works identically for both local Docker and CP workspaces once the IAM wiring is complete.

## Design trade-offs

| | EICE | SSM Session Manager |
|---|---|---|
| Outbound connection | AWS-managed, no inbound ports | AWS-managed, no inbound ports |
| Auth | IAM-based, short-lived key injection | IAM role + IAM Auth plugin |
| Dependency | aws-cli v2 in tenant image (~1MB) | Session Manager plugin |
| Latency | Direct tunnel, lower | Relay-based |

EICE was chosen because the signed-key flow works with a standard `ssh` binary rather than requiring a plugin. The tenant image adds ~1MB via `apk add aws-cli openssh-client`.

## Verification checklist

- [ ] `aws-cli --version` returns a non-error in the workspace container
- [ ] IAM policy attached to `molecule-cp` includes both EICE actions
- [ ] EIC Endpoint exists in the workspace VPC
- [ ] New CP-provisioned workspace has non-null `instance_id` in DB: `SELECT instance_id FROM workspaces WHERE id = '<id>'`
- [ ] Terminal tab shows bash prompt within 5 seconds

If EICE isn't wired yet, the terminal shows a hint pointing at the [design doc](https://github.com/Molecule-AI/molecule-core/blob/main/docs/infra/workspace-terminal.md) instead of a blank screen.