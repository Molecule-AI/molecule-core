#!/usr/bin/env python3
"""
Gemini CLI runtime adapter — live demo
Molecule AI | feat(adapters): add gemini-cli runtime adapter (#379)

Spins up a gemini-cli workspace, sends a task via the A2A proxy,
prints the reply, then tears down the workspace.

Usage:
    pip install httpx
    export PLATFORM_URL=http://localhost:8080
    export PLATFORM_TOKEN=<admin-bearer-token>
    export GEMINI_API_KEY=<your-google-ai-studio-key>
    python demo.py

No API keys are ever hardcoded or logged.
"""

import os
import sys
import time
import uuid

try:
    import httpx
except ImportError:
    print("Missing dependency: pip install httpx")
    sys.exit(1)

# ── Config (all from environment — no hardcoded values) ──────────────────────
PLATFORM_URL   = os.environ.get("PLATFORM_URL", "").rstrip("/")
PLATFORM_TOKEN = os.environ.get("PLATFORM_TOKEN", "")
GEMINI_API_KEY = os.environ.get("GEMINI_API_KEY", "")

MISSING = [k for k, v in {
    "PLATFORM_URL": PLATFORM_URL,
    "PLATFORM_TOKEN": PLATFORM_TOKEN,
    "GEMINI_API_KEY": GEMINI_API_KEY,
}.items() if not v]
if MISSING:
    print(f"Missing required env vars: {', '.join(MISSING)}")
    sys.exit(1)

HEADERS = {
    "Authorization": f"Bearer {PLATFORM_TOKEN}",
    "Content-Type": "application/json",
}

TASK = (
    "List the three biggest advantages of Google Gemini 2.5 Pro "
    "over GPT-4o for agentic coding tasks. One sentence each."
)


# ── Helpers ───────────────────────────────────────────────────────────────────

def step(n: int, msg: str) -> None:
    print(f"\n\033[1;34m[{n}]\033[0m {msg}")


def die(msg: str) -> None:
    print(f"\n\033[1;31m✗\033[0m {msg}")
    sys.exit(1)


def api(method: str, path: str, **kwargs) -> dict:
    """Make an authenticated request; exit on non-2xx."""
    url = f"{PLATFORM_URL}{path}"
    with httpx.Client(timeout=kwargs.pop("timeout", 30)) as client:
        resp = getattr(client, method)(url, headers=HEADERS, **kwargs)
    if resp.status_code not in (200, 201, 204):
        die(f"HTTP {resp.status_code} {method.upper()} {path}: {resp.text[:300]}")
    return resp.json() if resp.content else {}


# ── Main ─────────────────────────────────────────────────────────────────────

def main() -> None:
    workspace_id: str | None = None

    try:
        # 1. Create the gemini-cli workspace
        step(1, "Creating gemini-cli workspace...")
        ws = api("post", "/workspaces", json={
            "name": "gemini-cli-demo",
            "role": "Molecule AI gemini-cli adapter demo",
            "runtime": "gemini-cli",
            "runtime_config": {
                "model": "gemini-2.5-flash",   # flash: faster boot for demo purposes
                "timeout": 0,
            },
            "tier": 2,  # 2 GB / 2 vCPU
        })
        workspace_id = ws["id"]
        print(f"  created  id={workspace_id}")

        # 2. Inject GEMINI_API_KEY as a workspace-scoped secret
        step(2, "Storing GEMINI_API_KEY as workspace secret (value never logged)...")
        api("put", f"/workspaces/{workspace_id}/secrets",
            json={"key": "GEMINI_API_KEY", "value": GEMINI_API_KEY})
        print("  secret stored")

        # 3. Wait for the workspace container to boot and register
        step(3, "Waiting for workspace to come online (up to 90 s)...")
        for attempt in range(30):
            ws = api("get", f"/workspaces/{workspace_id}", timeout=10)
            status = ws.get("status", "unknown")
            print(f"  {status:12s} ({attempt + 1}/30)", end="\r", flush=True)
            if status == "online":
                print(f"\n  online in ~{attempt * 3} s")
                break
            if status in ("failed", "error"):
                die(f"workspace entered error state: {status}")
            time.sleep(3)
        else:
            die("timed out waiting for 'online' status")

        # 4. Send a task via the A2A proxy (JSON-RPC 2.0 over HTTP)
        step(4, "Sending task via A2A proxy...")
        print(f'  Task: "{TASK}"')
        result = api(
            "post",
            f"/workspaces/{workspace_id}/a2a",
            json={
                "jsonrpc": "2.0",
                "id": str(uuid.uuid4()),
                "method": "message/send",
                "params": {
                    "message": {
                        "role": "user",
                        "parts": [{"kind": "text", "text": TASK}],
                    }
                },
            },
            timeout=120,  # agent may take a moment to reason
        )

        # 5. Extract the text reply from the A2A response envelope
        step(5, "Gemini CLI agent reply:")
        try:
            parts = result["result"]["status"]["message"]["parts"]
            reply = "\n".join(
                p["text"] for p in parts if p.get("kind") == "text"
            )
        except (KeyError, TypeError):
            reply = str(result)

        print()
        for line in reply.splitlines():
            print(f"  {line}")
        print()

    finally:
        # 6. Always clean up — even if an earlier step failed
        if workspace_id:
            step(6, "Deleting demo workspace...")
            api("delete", f"/workspaces/{workspace_id}", timeout=15)
            print("  workspace deleted")

    print("\033[1;32mDemo complete.\033[0m\n")


if __name__ == "__main__":
    main()
