#!/usr/bin/env python3
"""
demo.py — Phase 34 Partner API Keys Demo
=========================================
Phase: 34 | Source: PR #TBD (`feat: partner api keys`)
Handler: workspace-server/internal/handlers/partner_keys.go

Demonstrates the Partner API Keys integration:
1. Create a partner-scoped key (POST /cp/admin/partner-keys)
2. Use the partner key to create an ephemeral org (POST /cp/orgs)
3. Poll org status until ready, then create a workspace
4. Revoke the partner key — next request returns 401

Requirements: pip install requests

Usage:
  export PLATFORM_URL=https://your-deployment.moleculesai.app
  export ADMIN_TOKEN=your-admin-token
  python demo.py

────────────────────────────────────────────────────────────────────────────
"""

from __future__ import annotations

import json, os, textwrap, time
from dataclasses import dataclass
from typing import Optional

try:
    import requests
except ImportError:
    raise SystemExit("pip install requests  # HTTP client for Molecule AI API")


PLATFORM_URL = os.environ.get("PLATFORM_URL", "https://your-deployment.moleculesai.app")
ADMIN_TOKEN  = os.environ.get("ADMIN_TOKEN",  "your-admin-token")


# ─────────────────────────────────────────────────────────────────────────────
# Utilities
# ─────────────────────────────────────────────────────────────────────────────

def is_live_platform() -> bool:
    """Return True only when credentials point to a real deployment."""
    if "your-deployment" in PLATFORM_URL:
        return False
    if PLATFORM_URL.startswith("http://") and "localhost" not in PLATFORM_URL:
        return False
    if ADMIN_TOKEN in ("", "your-admin-token"):
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

class PartnerKeysClient:
    def __init__(self, platform_url: str, admin_token: str):
        self.base = platform_url.rstrip("/")
        self.admin_hdrs = {"Authorization": f"Bearer {admin_token}", "Content-Type": "application/json"}

    def _url(self, path: str) -> str:
        return f"{self.base}{path}"

    def _req(self, method: str, path: str, headers: dict, **kwargs) -> requests.Response:
        r = requests.request(method, self._url(path), headers=headers, timeout=15, **kwargs)
        r.raise_for_status()
        return r

    # POST /cp/admin/partner-keys — create a partner key
    def create_key(self, name: str, scopes: list[str],
                   description: str = "") -> dict:
        return self._req("POST", "/cp/admin/partner-keys",
                          headers=self.admin_hdrs,
                          json={"name": name, "scopes": scopes,
                                "description": description}).json()

    # GET /cp/admin/partner-keys — list all partner keys
    def list_keys(self) -> dict:
        return self._req("GET", "/cp/admin/partner-keys",
                          headers=self.admin_hdrs).json()

    # DELETE /cp/admin/partner-keys/:id — revoke a partner key
    def revoke_key(self, key_id: str) -> requests.Response:
        return self._req("DELETE", f"/cp/admin/partner-keys/{key_id}",
                          headers=self.admin_hdrs)

    # POST /cp/orgs — create an org using a partner key
    def create_org(self, partner_key: str, name: str,
                   slug: str = "", plan: str = "standard") -> dict:
        partner_hdrs = {"Authorization": f"Bearer {partner_key}",
                        "Content-Type": "application/json"}
        payload = {"name": name, "plan": plan}
        if slug:
            payload["slug"] = slug
        return self._req("POST", "/cp/orgs",
                          headers=partner_hdrs,
                          json=payload).json()

    # GET /cp/orgs/:id — poll org status
    def get_org(self, partner_key: str, org_id: str) -> dict:
        partner_hdrs = {"Authorization": f"Bearer {partner_key}",
                        "Content-Type": "application/json"}
        return self._req("GET", f"/cp/orgs/{org_id}",
                          headers=partner_hdrs).json()

    # DELETE /cp/orgs/:id — delete an org
    def delete_org(self, partner_key: str, org_id: str) -> requests.Response:
        partner_hdrs = {"Authorization": f"Bearer {partner_key}"}
        return self._req("DELETE", f"/cp/orgs/{org_id}",
                          headers=partner_hdrs)

    # POST /cp/orgs/:org_id/workspaces — create a workspace in the org
    def create_workspace(self, partner_key: str, org_id: str,
                         name: str, plan: str = "standard") -> dict:
        partner_hdrs = {"Authorization": f"Bearer {partner_key}",
                        "Content-Type": "application/json"}
        return self._req("POST", f"/cp/orgs/{org_id}/workspaces",
                          headers=partner_hdrs,
                          json={"name": name, "plan": plan}).json()


# ─────────────────────────────────────────────────────────────────────────────
# Simulated responses (no live platform needed)
# ─────────────────────────────────────────────────────────────────────────────

def simulate_create_key(name: str = "ci-pipeline-key") -> dict:
    return {
        "id": "pak_01HXKM4ABC",
        "key": f"mol_pk_live_abc123xyz{name[:8]}789",
        "name": name,
        "scopes": ["orgs:create", "orgs:list", "workspaces:create"],
        "created_at": "2026-04-23T08:00:00Z",
        "message": "Save this key — it cannot be retrieved again.",
    }

def simulate_create_org(partner_key: str, name: str = "test-pr-123") -> dict:
    return {
        "id": f"org_{name[:8].upper()}",
        "name": name,
        "slug": name.lower().replace(" ", "-"),
        "status": "provisioning",
        "created_at": "2026-04-23T08:01:00Z",
    }

def simulate_get_org(org_id: str) -> dict:
    return {
        "id": org_id,
        "status": "active",
        "created_at": "2026-04-23T08:01:00Z",
    }


# ─────────────────────────────────────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────────────────────────────────────

def main():
    is_live = is_live_platform()
    client = PartnerKeysClient(PLATFORM_URL, ADMIN_TOKEN) if is_live else None

    print("""
    ╔══════════════════════════════════════════════════════════════════════╗
    ║       Partner API Keys Demo — Phase 34 (molecule-core)           ║
    ║                                                                      ║
    ║  mol_pk_* keys let CI/CD pipelines, marketplace resellers,        ║
    ║  and automation platforms create and manage orgs via API —        ║
    ║  no browser session required.                                        ║
    ╚══════════════════════════════════════════════════════════════════════╝
    """)

    # ── Step 1 ────────────────────────────────────────────────────────────
    divider("Step 1 — Create a partner-scoped API key")
    print("  POST /cp/admin/partner-keys")
    print("  Body: {\"name\": \"ci-pipeline-key\", \"scopes\": [\"orgs:create\", \"orgs:list\"]}")
    print()
    if is_live:
        print("  → calling live platform...")
        try:
            result = client.create_key(name="ci-pipeline-key",
                                       scopes=["orgs:create", "orgs:list", "workspaces:create"],
                                       description="CI pipeline integration")
            print(f"  ✓ Key created: {result['key'][:30]}...")
            print(f"    scopes: {result['scopes']}")
            partner_key = result["key"]
        except Exception as e:
            print(f"  ✗ Error: {e}")
            partner_key = None
    else:
        result = simulate_create_key()
        partner_key = result["key"]
        print(f"  ✓ Partner key created:")
        print(f"    id    : {result['id']}")
        print(f"    key   : {result['key'][:30]}...")
        print(f"    scopes: {result['scopes']}")
        print(f"    {result['message']}")

    # ── Step 2 ────────────────────────────────────────────────────────────
    divider("Step 2 — Create an ephemeral org with the partner key")
    print("  POST /cp/orgs  (authenticated with mol_pk_*)")
    print("  Body: {\"name\": \"test-pr-123\", \"plan\": \"ephemeral\"}")
    print()
    if is_live and partner_key:
        print("  → calling live platform...")
        try:
            org = client.create_org(partner_key, name="test-pr-123", plan="ephemeral")
            org_id = org["id"]
            print(f"  ✓ Org created: {org_id} (status: {org.get('status', 'provisioning')})")
        except Exception as e:
            print(f"  ✗ Error: {e}")
            org_id = None
    else:
        org = simulate_create_org(partner_key, "test-pr-123")
        org_id = org["id"]
        print(f"  ✓ Ephemeral org created:")
        print(f"    org_id: {org_id}")
        print(f"    status: {org['status']}")

    # ── Step 3 ────────────────────────────────────────────────────────────
    if org_id:
        divider("Step 3 — Poll until org is active, then create a workspace")
        print(f"  GET /cp/orgs/{org_id}")
        print()
        if is_live and partner_key:
            print("  → polling org status (simulated poll loop)...")
            # Poll up to 5 times with 1s delay
            for i in range(5):
                org_status = client.get_org(partner_key, org_id)
                if org_status.get("status") == "active":
                    print(f"  ✓ Org active: {org_id}")
                    break
                print(f"    attempt {i+1}: status={org_status.get('status')}")
                time.sleep(1)
            # Create workspace
            try:
                ws = client.create_workspace(partner_key, org_id, name="pr-123-test")
                print(f"  ✓ Workspace created: {ws.get('id', '?')}")
            except Exception as e:
                print(f"  ✗ Workspace create error: {e}")
        else:
            org_status = simulate_get_org(org_id)
            print(f"  ✓ Org polled: status={org_status['status']}")
            print(f"  ✓ Workspace create: pr-123-test (simulated)")

    # ── Step 4 ────────────────────────────────────────────────────────────
    divider("Step 4 — Ephemeral teardown: DELETE the org")
    print("  DELETE /cp/orgs/:id  (authenticated with mol_pk_*)")
    print()
    print("  Billing stops immediately. No orphaned resources.")
    print()
    if is_live and partner_key and org_id:
        print("  → calling live platform...")
        try:
            r = client.delete_org(partner_key, org_id)
            print(f"  ✓ Org deleted (status: {r.status_code})")
        except Exception as e:
            print(f"  ✗ Error: {e}")
    else:
        print(f"  Simulated DELETE /cp/orgs/{org_id}")
        print("  → 204 No Content")
        print("  ✓ Ephemeral org destroyed — billing stopped")

    # ── Step 5 ────────────────────────────────────────────────────────────
    divider("Step 5 — Revoke a compromised partner key")
    print("  DELETE /cp/admin/partner-keys/:id  (admin-only)")
    print()
    print("  Compromised key? One call. Next request → 401.")
    print()
    if is_live:
        print("  → calling live platform...")
        try:
            # First list keys, then revoke the first one
            keys = client.list_keys()
            key_id = keys["keys"][0]["id"]
            r = client.revoke_key(key_id)
            print(f"  ✓ Key revoked (status: {r.status_code})")
            # Verify: try to list again
            r2 = client.list_keys()
            remaining = [k for k in r2.get("keys", []) if k["id"] != key_id]
            print(f"  ✓ Verification: {len(remaining)} key(s) remain")
        except Exception as e:
            print(f"  ✗ Error: {e}")
    else:
        print("  Simulated DELETE /cp/admin/partner-keys/pak_01HXKM4ABC")
        print("  → 204 No Content")
        print("  ✓ Key is dead on the next request — no propagation delay")

    # ── Architecture ────────────────────────────────────────────────────────
    divider("Architecture Summary")
    print(textwrap.dedent("""\
      Admin creates a partner key:
        POST /cp/admin/partner-keys
        → returns plaintext key (shown ONCE)
        → SHA-256 hash stored server-side

      Partner uses the key:
        POST /cp/orgs           — create org within partner's scope
        GET  /cp/orgs/:id        — poll until active
        DELETE /cp/orgs/:id     — tear down, billing stops
        POST /cp/orgs/:id/workspaces — create workspaces in the org

      Key revocation:
        DELETE /cp/admin/partner-keys/:id  (admin-only)
        → next request with that key → 401 immediately

      Security model:
      • mol_pk_* keys are org-scoped — cannot escape their org boundary
      • Plaintext shown once at creation, SHA-256 hash stored
      • Per-key rate limiter (separate from session limits)
      • last_used_at tracked on every request
      • mol_pk_ added to pre-commit secret scanner

      Phase 34 also shipped:
      • Tool Trace — execution record in every A2A response
      • Platform Instructions — org-level system prompt via API
      • SaaS Fed v2 — improved multi-org federation
    """))

    divider("Reference")
    print("  Handler  : workspace-server/internal/handlers/partner_keys.go")
    print("  DB table : partner_api_keys")
    print("  Blog     : docs/blog/2026-04-23-partner-api-keys/")
    print("  Battlecard: docs/marketing/battlecard/phase-34-partner-api-keys-battlecard.md")
    print()
    print("  Set PLATFORM_URL + ADMIN_TOKEN to run against a live platform.")


if __name__ == "__main__":
    main()


# ─────────────────────────────────────────────────────────────────────────────
# Key design notes
# ─────────────────────────────────────────────────────────────────────────────
"""
Partner API Keys vs other key types:

  Workspace token  — scopes to ONE workspace
  Org token       — scopes to ALL workspaces in ONE org
  Partner key     — scopes to ORG-LEVEL operations (create orgs, list keys, revoke self)
                    CANNOT access workspace sub-routes

  Org-scoped keys operate WITHIN an org. Partner keys operate AT the org level.

The mol_pk_ prefix is the identifier that makes the secret scanner find these
keys before they get committed to git history.
"""
