#!/usr/bin/env python3
"""
demo.py — Org-Scoped API Keys Demo
===================================
PR #1105 (molecule-core) | Issue: org-api-keys-launch campaign

Demonstrates the org-scoped token API:
  1. List existing org tokens (GET /org/tokens)
  2. Mint a new token with name, scope, expiry (POST /org/tokens)
  3. Verify token works with a scoped workspace call (POST /workspaces/:id/artifacts)
  4. Verify cross-org access is rejected
  5. Revoke a token (DELETE /org/tokens/:id)

Requirements: pip install requests

Usage:
  export PLATFORM_URL=https://your-deployment.moleculesai.app
  export ORG_TOKEN=your-org-scoped-token  # must be admin-level
  python demo.py

────────────────────────────────────────────────────────────────────────────
"""

from __future__ import annotations

import os, textwrap, time

try:
    import requests
except ImportError:
    raise SystemExit("pip install requests  # HTTP client for Molecule AI API")


PLATFORM_URL = os.environ.get("PLATFORM_URL", "https://your-deployment.moleculesai.app")
ORG_TOKEN    = os.environ.get("ORG_TOKEN",    "your-org-scoped-token")


# ─────────────────────────────────────────────────────────────────────────────
# Utilities
# ─────────────────────────────────────────────────────────────────────────────

def is_live_platform() -> bool:
    """Return True only when credentials point to a real deployment."""
    if "your-deployment" in PLATFORM_URL:
        return False
    if PLATFORM_URL.startswith("http://") and "localhost" not in PLATFORM_URL:
        return False
    if ORG_TOKEN in ("", "your-org-scoped-token", "demo-token"):
        return False
    return True


def divider(title: str) -> None:
    d = "═" * 68
    print(f"\n  {d}")
    print(f"  {title}")
    print(f"  {d}\n")


# ─────────────────────────────────────────────────────────────────────────────
# API client
# ─────────────────────────────────────────────────────────────────────────────

class OrgTokenClient:
    def __init__(self, platform_url: str, org_token: str):
        self.base = platform_url.rstrip("/")
        self.hdrs = {"Authorization": f"Bearer {org_token}", "Content-Type": "application/json"}

    def _url(self, path: str) -> str:
        return f"{self.base}{path}"

    def _req(self, method: str, path: str, **kwargs) -> requests.Response:
        r = requests.request(method, self._url(path), headers=self.hdrs, timeout=15, **kwargs)
        r.raise_for_status()
        return r

    # GET /org/tokens — list all org tokens
    def list_tokens(self) -> dict:
        return self._req("GET", "/org/tokens").json()

    # POST /org/tokens — mint a new org token
    def mint_token(self, name: str, scope: str = "read",
                   expires_in_days: int = 30) -> dict:
        return self._req("POST", "/org/tokens", json={
            "name": name,
            "scope": scope,
            "expires_in_days": expires_in_days,
        }).json()

    # DELETE /org/tokens/:id — revoke a token
    def revoke_token(self, token_id: str) -> requests.Response:
        return self._req("DELETE", f"/org/tokens/{token_id}")

    # POST /workspaces/:id/artifacts — scoped workspace call
    def create_workspace_artifact(self, workspace_id: str, name: str) -> dict:
        return self._req("POST", f"/workspaces/{workspace_id}/artifacts", json={
            "name": name,
            "description": f"Created via org token at {time.strftime('%Y-%m-%dT%H:%M:%SZ')}",
        }).json()


# ─────────────────────────────────────────────────────────────────────────────
# Simulated responses (no live platform needed)
# ─────────────────────────────────────────────────────────────────────────────

def simulate_list_tokens() -> dict:
    return {
        "tokens": [
            {"id": "tok_abc001", "name": "slack-integration",  "scope": "read",  "created_at": "2026-04-20T09:00:00Z", "last_used": "2026-04-23T08:45:00Z"},
            {"id": "tok_abc002", "name": "monitoring-script",   "scope": "read",  "created_at": "2026-04-21T14:00:00Z", "last_used": None},
            {"id": "tok_abc003", "name": "ci-pipeline-key",     "scope": "write", "created_at": "2026-04-23T10:00:00Z", "last_used": None},
        ],
        "total": 3,
    }

def simulate_mint_token(name: str = "demo-token") -> dict:
    return {
        "id": "tok_xyz789",
        "name": name,
        "token": "org_tk_demo_abc123xyz456def789abc000",
        "scope": "write",
        "expires_at": "2026-04-30T00:00:00Z",
        "created_at": "2026-04-23T12:00:00Z",
        "message": "Save this token — it cannot be retrieved again.",
    }

def simulate_create_artifact(workspace_id: str, name: str) -> dict:
    return {
        "id": "art_demo_001",
        "workspace_id": workspace_id,
        "cf_repo_name": name,
        "cf_namespace": "molecule-prod",
        "remote_url": "https://artifacts.cloudflare.net/git/" + name,
        "created_at": "2026-04-23T12:00:00Z",
    }


# ─────────────────────────────────────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────────────────────────────────────

def main():
    is_live = is_live_platform()
    client = OrgTokenClient(PLATFORM_URL, ORG_TOKEN) if is_live else None

    print("""
    ╔══════════════════════════════════════════════════════════════════════╗
    ║      Org-Scoped API Keys Demo — PR #1105 (molecule-core)            ║
    ║                                                                      ║
    ║  Every Molecule AI organization can mint scoped API keys:           ║
    ║  name them, set expiry, scope them read/write, revoke instantly.  ║
    ║  No shared global admin keys. No workspace-level token sharing.    ║
    ╚══════════════════════════════════════════════════════════════════════╝
    """)

    # ── Step 1 ────────────────────────────────────────────────────────────
    divider("Step 1 — List org tokens")
    print("  GET /org/tokens")
    print()
    if is_live:
        print("  → calling live platform...")
        try:
            resp = client.list_tokens()
        except Exception as e:
            print(f"  ✗ Error: {e}")
    else:
        resp = simulate_list_tokens()
        print("  ✓ Active tokens:")
        for tok in resp["tokens"]:
            print(f"    [{tok['scope']:5}] {tok['name']:25} last_used={tok['last_used'] or 'never'}")

    # ── Step 2 ────────────────────────────────────────────────────────────
    divider("Step 2 — Mint a new org token")
    print("  POST /org/tokens")
    print("  Body: {\"name\": \"ci-pipeline-key\", \"scope\": \"write\", \"expires_in_days\": 90}")
    print()
    if is_live:
        print("  → calling live platform...")
        try:
            resp = client.mint_token(name="ci-pipeline-key", scope="write", expires_in_days=90)
            print(f"  ✓ Token minted:")
            print(f"    token : {resp['token']}")
            print(f"    expires_at : {resp['expires_at']}")
        except Exception as e:
            print(f"  ✗ Error: {e}")
    else:
        resp = simulate_mint_token("ci-pipeline-key")
        print(f"  ✓ Token minted (plaintext shown ONCE — save it now):")
        print(f"    token     : {resp['token']}")
        print(f"    scope     : {resp['scope']}")
        print(f"    expires_at: {resp['expires_at']}")
        print(f"    message   : {resp['message']}")

    # ── Step 3 ────────────────────────────────────────────────────────────
    divider("Step 3 — Use org token for scoped workspace call")
    print("  POST /workspaces/:id/artifacts")
    print("  (org token scopes access to all workspaces in this org)")
    print()
    if is_live:
        print("  → calling live platform...")
        try:
            resp = client.create_workspace_artifact("ws-demo-001", "ci-deploy-snapshots")
            print(f"  ✓ Artifact created: {resp['cf_repo_name']}")
        except Exception as e:
            print(f"  ✗ Error: {e}")
    else:
        resp = simulate_create_artifact("ws-demo-001", "ci-deploy-snapshots")
        print(f"  ✓ CI can attach an Artifacts repo with org token:")
        print(f"    workspace_id : {resp['workspace_id']}")
        print(f"    cf_repo_name : {resp['cf_repo_name']}")
        print(f"    remote_url   : {resp['remote_url']}")

    # ── Step 4 ────────────────────────────────────────────────────────────
    divider("Step 4 — Cross-org access is blocked")
    print("  The org token only grants access within its own organization.")
    print("  Any call to a workspace in a different org returns 403.")
    print()
    if is_live:
        print("  → calling cross-org workspace with same token...")
        try:
            resp = requests.get(
                f"{PLATFORM_URL}/workspaces/ws-external-org/executions",
                headers={"Authorization": f"Bearer {ORG_TOKEN}"},
                timeout=15,
            )
            if resp.status_code == 403:
                print("  ✓ 403 Forbidden — org boundary enforced correctly")
            else:
                print(f"  ⚠ Unexpected status: {resp.status_code}")
        except Exception as e:
            print(f"  ✗ Error: {e}")
    else:
        print("  Simulated cross-org rejection:")
        print("  curl -H 'Authorization: Bearer $ORG_TOKEN' \\")
        print("       $PLATFORM/workspaces/ws-external-org/executions")
        print()
        print("  → {\"error\": \"unauthorized\", \"message\": \"token org does not match target workspace org\"}")
        print("  ✓ Org boundary enforced — cross-org access is blocked")

    # ── Step 5 ────────────────────────────────────────────────────────────
    divider("Step 5 — Revoke a token")
    print("  DELETE /org/tokens/:id")
    print("  Revocation is immediate — no grace period, no cached sessions.")
    print()
    if is_live:
        print("  → calling live platform...")
        try:
            client.revoke_token("tok_xyz789")
            print("  ✓ Token revoked successfully")
        except Exception as e:
            print(f"  ✗ Error: {e}")
    else:
        print("  Simulated revocation of tok_xyz789 (ci-pipeline-key):")
        print("  DELETE /org/tokens/tok_xyz789")
        print()
        print("  → 204 No Content")
        print("  ✓ Token is dead immediately — no grace period, no cached sessions")

    # ── Architecture ────────────────────────────────────────────────────────
    divider("Architecture Summary")
    print(textwrap.dedent("""\
      POST /org/tokens
        → validates caller is org admin
        → generates crypto-random token value (shown once)
        → stores token hash + metadata (NOT plaintext)
        → returns plaintext token to caller exactly once

      GET /org/tokens
        → lists tokens: id, name, scope, created_at, last_used
        → never returns token plaintext (only stored as hash)

      DELETE /org/tokens/:id
        → soft-deletes token row (revoked_at set)
        → next request with that token → 401 immediately

      Org token in requests:
        → Authorization: Bearer <token> (same as workspace tokens)
        → Platform resolves token → org_id → workspace org check
        → cross-org call → 403 Forbidden

      Security model:
      • Plaintext token shown exactly once at mint time
      • Server stores only SHA-256 hash
      • Org-level scoping — can access any workspace in same org
      • Instant revocation — no grace period
      • Audit log: created_at, last_used tracked per token
    """))

    divider("Reference")
    print("  Handler  : workspace-server/internal/handlers/org_tokens.go")
    print("  Tests    : workspace-server/internal/handlers/org_tokens_test.go")
    print("  DB table : org_tokens")
    print("  Frontend : canvas/src/components/settings/OrgTokensTab.tsx")
    print()
    print("  Set PLATFORM_URL + ORG_TOKEN to run against a live platform.")
    print("  Demo path: docs/marketing/devrel/demos/org-api-keys/")

if __name__ == "__main__":
    main()


# ─────────────────────────────────────────────────────────────────────────────
# Key design notes
# ─────────────────────────────────────────────────────────────────────────────
"""
Org-scoped tokens vs workspace tokens:

  Workspace token (existing):
    → grants access to ONE specific workspace
    → issued at workspace creation time
    → tied to workspace lifecycle

  Org token (new — PR #1105):
    → grants access to ALL workspaces in ONE org
    → mintable/revokable by org admin from Canvas
    → scoped read|write
    → expiry: 1–365 days
    → plaintext shown once, SHA-256 hash stored
    → revoke = immediate 401 on next use
"""
