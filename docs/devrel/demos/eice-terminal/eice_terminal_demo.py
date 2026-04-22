#!/usr/bin/env python3
"""
EICE Terminal Demo — Molecule AI workspace-server PR #1533

This script demonstrates how the EC2 Instance Connect Endpoint (EICE)
terminal feature works from a client perspective.

In production this is triggered automatically when you open the Terminal
tab on a CP-provisioned workspace. This script breaks the feature down so
you can:
  1. See the exact AWS API calls being made
  2. Understand how the ephemeral keypair flow works
  3. Verify your tenant IAM and EIC Endpoint wiring independently

Prerequisites:
  - aws-cli v2 installed and configured (needs ec2-instance-connect permissions)
  - Python 3.10+ with `pip install websockets boto3`
  - A CP-provisioned workspace with instance_id populated (or a mock instance)

Usage:
  python3 eice_terminal_demo.py [--workspace-id <id>] [--instance-id <i-xxx>]
                                [--region <region>] [--dry-run]

  --dry-run  Print the EIC commands that WOULD be executed without running them.
             Useful to verify the flow before a real workspace is available.

Environment variables (alternative to flags):
  INSTANCE_ID     EC2 instance ID of the workspace
  AWS_REGION      AWS region
  WORKSPACE_ID    Molecule workspace ID (used only in the print output)
"""

import argparse
import subprocess
import tempfile
import os
import sys
import time
import shutil

try:
    import websockets
except ImportError:
    websockets = None

try:
    import boto3
    import botocore.exceptions
except ImportError:
    boto3 = None


# ---------------------------------------------------------------------------
# Step 1 — parse arguments / env
# ---------------------------------------------------------------------------

def get_args():
    parser = argparse.ArgumentParser(description="EICE Terminal demo (PR #1533)")
    parser.add_argument("--workspace-id",  default=os.environ.get("WORKSPACE_ID", "ws-demo-001"))
    parser.add_argument("--instance-id",   default=os.environ.get("INSTANCE_ID", "i-demo-0123456789abcdef0"))
    parser.add_argument("--region",        default=os.environ.get("AWS_REGION", "us-east-2"))
    parser.add_argument("--os-user",       default=os.environ.get("OS_USER", "ubuntu"))
    parser.add_argument("--dry-run",       action="store_true",
                        help="Print the commands that would run without executing them")
    parser.add_argument("--server-url",     default=os.environ.get("SERVER_URL", "wss://app.molecule.ai"),
                        help="Workspace server WebSocket endpoint")
    return parser.parse_args()


# ---------------------------------------------------------------------------
# Step 2 — generate ephemeral keypair (never written to disk permanently)
# ---------------------------------------------------------------------------

def generate_ephemeral_keypair(key_dir: str):
    """Generate an Ed25519 keypair in key_dir. Returns (private_key_path, public_key_str)."""
    key_path = os.path.join(key_dir, "id")
    if shutil.which("ssh-keygen") is None:
        # No ssh-keygen in this environment — generate a mock key for dry-run
        # In production, this always runs on the workspace-server host which has it.
        mock_pub = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIH虚构假公钥-仅用于无法运行ssh-keygen的环境演示 Molecule-Demo-Key"
        # Write a dummy private key so the path check below doesn't fail
        with open(key_path, "w") as f:
            f.write("-----BEGIN OPENSSH PRIVATE KEY-----\nMOCK_KEY_FOR_DRY_RUN\n-----END OPENSSH PRIVATE KEY-----\n")
        with open(key_path + ".pub", "w") as f:
            f.write(mock_pub + "\n")
        return key_path, mock_pub
    result = subprocess.run(
        ["ssh-keygen", "-t", "ed25519", "-f", key_path, "-N", "", "-q", "-C", "molecule-terminal"],
        capture_output=True, text=True, check=True
    )
    with open(key_path + ".pub") as f:
        pub_key = f.read().strip()
    return key_path, pub_key


# ---------------------------------------------------------------------------
# Step 3 — send SSH public key to EIC (push ephemeral key)
# ---------------------------------------------------------------------------

def send_ssh_public_key(instance_id: str, region: str, os_user: str, pub_key: str, dry_run: bool):
    """
    Calls: aws ec2-instance-connect send-ssh-public-key
    This pushes the ephemeral public key into the instance's metadata.
    The instance's sshd will accept the paired private key for 60 seconds.
    """
    cmd = [
        "aws", "ec2-instance-connect", "send-ssh-public-key",
        "--instance-id", instance_id,
        "--region", region,
        "--instance-os-user", os_user,
        "--ssh-public-key", pub_key,
    ]
    print(f"\n[Step 1] Push ephemeral SSH public key to instance via EIC")
    print(f"  Command: {' '.join(cmd)}")
    if dry_run:
        print("  → DRY RUN — would call EIC to register key (valid 60s)")
        return True
    result = subprocess.run(cmd, capture_output=True, text=True)
    if result.returncode != 0:
        if dry_run and shutil.which("aws") is None:
            print(f"  ✓ DRY RUN — would call EIC (aws-cli not in PATH, expected in this demo env)")
            return True
        print(f"  ✗ send-ssh-public-key failed:\n    {result.stderr.strip()}")
        return False
    print("  ✓ Key accepted by instance metadata (valid 60s)")
    return True


# ---------------------------------------------------------------------------
# Step 4 — open EIC tunnel
# ---------------------------------------------------------------------------

def open_eic_tunnel(instance_id: str, region: str, dry_run: bool) -> int:
    """
    Calls: aws ec2-instance-connect open-tunnel --instance-id <id> --local-port 0
    Returns the local port the tunnel is listening on.

    This establishes a TLS tunnel from localhost to the instance's port 22
    through the EIC Endpoint in the workspace VPC. No port 22 needed in SG.
    """
    cmd = [
        "aws", "ec2-instance-connect", "open-tunnel",
        "--instance-id", instance_id,
        "--region", region,
    ]
    print(f"\n[Step 2] Open EIC tunnel to instance port 22")
    print(f"  Command: {' '.join(cmd)}")
    if dry_run:
        # Simulate a random free port
        tunnel_port = 54321
        print(f"  → DRY RUN — would open tunnel on localhost:{tunnel_port}")
        return tunnel_port

    # Run tunnel in background
    proc = subprocess.Popen(
        cmd + ["--local-port", "0"],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    # Parse port from stdout: "Tunnel ready on port 54321"
    tunnel_port = None
    for _ in range(30):  # wait up to 30s for tunnel to establish
        line = proc.stdout.readline()
        if not line:
            break
        line = line.decode().strip()
        if "port" in line.lower():
            # Try to extract port number
            import re
            m = re.search(r"\d{5}", line)
            if m:
                tunnel_port = int(m.group())
                break
        if "error" in line.lower() or "fail" in line.lower():
            print(f"  ✗ Tunnel open failed: {line}")
            return None

    if tunnel_port is None:
        proc.kill()
        print("  ✗ Could not determine tunnel port")
        return None

    print(f"  ✓ Tunnel listening on localhost:{tunnel_port}")
    return tunnel_port


# ---------------------------------------------------------------------------
# Step 5 — connect over SSH
# ---------------------------------------------------------------------------

def ssh_connect(tunnel_port: int, private_key_path: str, os_user: str, dry_run: bool):
    """Open an interactive SSH session through the tunnel."""
    cmd = [
        "ssh",
        "-i", private_key_path,
        "-o", "StrictHostKeyChecking=no",
        "-o", "UserKnownHostsFile=/dev/null",
        "-o", "ServerAliveInterval=30",
        "-p", str(tunnel_port),
        f"{os_user}@127.0.0.1",
    ]
    print(f"\n[Step 3] SSH to workspace container via tunnel")
    print(f"  Command: {' '.join(cmd)}")
    if dry_run:
        print("  → DRY RUN — would open interactive SSH session")
        return

    # Replace current process with SSH — interactive terminal
    os.execvp("ssh", cmd)


# ---------------------------------------------------------------------------
# Production WebSocket path (canvas tab → molecule-server)
# ---------------------------------------------------------------------------

async def ws_terminal_session(workspace_id: str, server_url: str):
    """
    In production, the canvas Terminal tab connects directly to the
    workspace-server via WebSocket. The handler (terminal.go) performs
    all three steps above (key push, tunnel, SSH) on the server side
    and bridges PTY ↔ WebSocket.

    This function demonstrates the client-side WebSocket interaction.
    """
    if websockets is None:
        print("Install websockets: pip install websockets")
        return

    endpoint = f"{server_url}/api/workspaces/{workspace_id}/terminal"
    print(f"\n[Canvas path] WebSocket → {endpoint}")
    print("  The server handles EIC key push + tunnel + SSH internally.")
    print("  PTY bytes flow: EC2 sshd → molecule-server PTY bridge → WebSocket → browser")
    print("  (To verify: open browser DevTools → Network → WS, filter /terminal)")

    # This would block on an interactive session — not runnable in demo mode
    # async with websockets.connect(endpoint) as ws:
    #     while True:
    #         data = await ws.recv()
    #         sys.stdout.buffer.write(data)
    #         sys.stdout.flush()


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    args = get_args()

    print("=" * 60)
    print("  EICE Terminal Demo  |  PR #1533  |  molecule-core")
    print("=" * 60)
    print(f"\n  workspace : {args.workspace_id}")
    print(f"  instance  : {args.instance_id}")
    print(f"  region    : {args.region}")
    print(f"  os-user   : {args.os_user}")
    print(f"  dry-run   : {args.dry_run}")

    # Check dependencies
    missing = []
    for cmd in ["ssh-keygen", "aws"]:
        if shutil.which(cmd) is None:
            missing.append(cmd)
    if missing:
        print(f"\n⚠ Missing commands: {missing} — install before running.")

    # Step 0 — check instance
    print(f"\n[Pre-flight] Verify instance exists in {args.region}")
    if boto3:
        try:
            ec2 = boto3.client("ec2", region_name=args.region)
            resp = ec2.describe_instances(InstanceIds=[args.instance_id])
            state = resp["Reservations"][0]["Instances"][0]["State"]["Name"]
            print(f"  ✓ Instance {args.instance_id} is {state}")
        except Exception as e:
            print(f"  ⚠ describe_instances: {e}")
    else:
        print("  ⚠ boto3 not installed — skipping AWS SDK check")

    # Step 1 — keypair
    print(f"\n[Step 0] Generate ephemeral Ed25519 keypair")
    with tempfile.TemporaryDirectory() as key_dir:
        key_path, pub_key = generate_ephemeral_keypair(key_dir)
        print(f"  ✓ Generated keypair in {key_dir} (auto-cleaned on exit)")

        # Show the public key (this is what gets pushed to EIC)
        print(f"\n  Public key (pushed to instance metadata):")
        for line in pub_key.split("\n"):
            print(f"    {line}")

        if not args.dry_run:
            print("\n  Key stored in temp dir — will be deleted when demo ends.")
            print("  (In production: temp dir is always used, keys never hit ~/.ssh)")

        # Step 1 — EIC send key
        ok = send_ssh_public_key(args.instance_id, args.region, args.os_user, pub_key, args.dry_run)
        if not ok:
            print("\n✗ EIC key push failed — check IAM policy on molecule-cp user.")
            print("  Required: ec2-instance-connect:SendSSHPublicKey + OpenTunnel")
            print("  See: docs/infra/workspace-terminal.md")
            sys.exit(1)

        # Step 2 — tunnel
        tunnel_port = open_eic_tunnel(args.instance_id, args.region, args.dry_run)
        if tunnel_port is None:
            print("\n✗ EIC tunnel failed — check EIC Endpoint in workspace VPC.")
            print("  See: docs/infra/workspace-terminal.md")
            sys.exit(1)

        # Step 3 — WebSocket path note
        import asyncio
        asyncio.get_event_loop().run_until_complete(
            ws_terminal_session(args.workspace_id, args.server_url)
        )

        # Step 4 — SSH (interactive — replaces process in non-dry-run)
        ssh_connect(tunnel_port, key_path, args.os_user, args.dry_run)


if __name__ == "__main__":
    main()