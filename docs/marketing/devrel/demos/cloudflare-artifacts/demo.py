#!/usr/bin/env python3
"""
demo.py — Cloudflare Artifacts Demo
====================================
Issue: #1479 | Source: PR #641 (`feat: workspace git artifacts (Cloudflare)`)
Handler: workspace-server/internal/handlers/artifacts.go

Demonstrates the Cloudflare Artifacts integration:
1. Attach a CF Artifacts Git repo to a workspace (POST /workspaces/:id/artifacts)
2. Mint a short-lived git credential (POST /workspaces/:id/artifacts/token)
3. Clone, commit, push from the agent
4. Fork the repo before a risky experiment

Requirements: pip install requests

Usage:
  export PLATFORM_URL=https://your-deployment.moleculesai.app
  export WORKSPACE_TOKEN=your-workspace-token
  export WORKSPACE_ID=your-workspace-id
  python demo.py

────────────────────────────────────────────────────────────────────────────
"""

from __future__ import annotations

import os, shutil, subprocess, tempfile, textwrap, time
from dataclasses import dataclass
from typing import Optional

try:
    import requests
except ImportError:
    raise SystemExit("pip install requests  # HTTP client for Molecule AI API")


PLATFORM_URL    = os.environ.get("PLATFORM_URL",    "https://your-deployment.moleculesai.app")
WORKSPACE_TOKEN = os.environ.get("WORKSPACE_TOKEN", "your-workspace-token")
WORKSPACE_ID    = os.environ.get("WORKSPACE_ID",     "ws-demo-001")


# ─────────────────────────────────────────────────────────────────────────────
# Utilities
# ─────────────────────────────────────────────────────────────────────────────

def is_live_platform() -> bool:
    """Return True only when credentials point to a real deployment."""
    if "your-deployment" in PLATFORM_URL:
        return False
    if PLATFORM_URL.startswith("http://") and "localhost" not in PLATFORM_URL and "platform" not in PLATFORM_URL:
        return False
    if WORKSPACE_TOKEN in ("", "your-workspace-token", "demo-token"):
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

class ArtifactsClient:
    def __init__(self, platform_url: str, workspace_token: str, workspace_id: str):
        self.base = platform_url.rstrip("/")
        self.hdrs = {"Authorization": f"Bearer {workspace_token}", "Content-Type": "application/json"}
        self.ws   = workspace_id

    def _url(self, path: str) -> str:
        return f"{self.base}/workspaces/{self.ws}{path}"

    def _post(self, path: str, data: Optional[dict] = None) -> requests.Response:
        r = requests.post(self._url(path), headers=self.hdrs, json=data or {}, timeout=15)
        r.raise_for_status()
        return r

    def _get(self, path: str) -> requests.Response:
        r = requests.get(self._url(path), headers=self.hdrs, timeout=15)
        r.raise_for_status()
        return r

    def _delete(self, path: str) -> requests.Response:
        r = requests.delete(self._url(path), headers=self.hdrs, timeout=15)
        r.raise_for_status()
        return r

    # POST /workspaces/:id/artifacts — create/link a CF Artifacts repo
    def attach_repo(self, name: str = "", description: str = "",
                    import_url: str = "", read_only: bool = False) -> dict:
        payload = {"description": description, "read_only": read_only}
        if name:
            payload["name"] = name
        if import_url:
            payload["import_url"] = import_url  # must be https://
        return self._post("/artifacts", payload).json()

    # GET /workspaces/:id/artifacts — get linked repo info
    def get_repo(self) -> dict:
        return self._get("/artifacts").json()

    # POST /workspaces/:id/artifacts/token — mint a short-lived git credential
    def mint_token(self, scope: str = "write", ttl: int = 3600) -> dict:
        return self._post("/artifacts/token", {"scope": scope, "ttl": ttl}).json()

    # POST /workspaces/:id/artifacts/fork — fork the workspace's primary repo
    def fork_repo(self, name: str, description: str = "",
                  default_branch_only: bool = True) -> dict:
        return self._post("/artifacts/fork", {
            "name": name,
            "description": description,
            "default_branch_only": default_branch_only,
        }).json()

    # DELETE /workspaces/:id/artifacts — detach the linked repo
    def detach_repo(self) -> None:
        self._delete("/artifacts")


# ─────────────────────────────────────────────────────────────────────────────
# Git helpers
# ─────────────────────────────────────────────────────────────────────────────

@dataclass
class GitResult:
    success: bool
    stdout: str
    stderr: str
    command: str

def run_git(cwd: str, *args: str) -> GitResult:
    cmd = ["git", "-C", cwd] + list(args)
    r = subprocess.run(cmd, capture_output=True, text=True)
    return GitResult(success=r.returncode == 0, stdout=r.stdout,
                     stderr=r.stderr, command=" ".join(cmd))


def clone_and_push(clone_url: str, repo_dir: str,
                   commit_msg: str, file_content: str,
                   file_path: str = "AGENT_SNAPSHOT.md") -> GitResult:
    """Clone, write file, commit, push. Cleans up temp dir on exit."""
    tmpdir = tempfile.mkdtemp(prefix="cf-artifacts-")
    try:
        r = run_git(tmpdir, "clone", "--quiet", clone_url, repo_dir)
        if not r.success:
            return r
        target = os.path.join(tmpdir, repo_dir)
        with open(os.path.join(target, file_path), "w") as f:
            f.write(file_content)
        run_git(target, "config", "user.email", "agent@molecule.ai")
        run_git(target, "config", "user.name",  "Molecule AI Agent")
        run_git(target, "add", file_path)
        r = run_git(target, "commit", "-m", commit_msg)
        if not r.success:
            return r
        return run_git(target, "push", "-q", "origin", "HEAD")
    finally:
        shutil.rmtree(tmpdir, ignore_errors=True)


# ─────────────────────────────────────────────────────────────────────────────
# Simulated responses (no live platform needed)
# ─────────────────────────────────────────────────────────────────────────────

def simulate_attach_repo() -> dict:
    return {
        "id": "wa_abc123", "workspace_id": WORKSPACE_ID,
        "cf_repo_name": "molecule-ws-demo", "cf_namespace": "molecule-prod",
        "remote_url": "https://artifacts.cloudflare.net/git/molecule-ws-demo",
        "description": "Demo workspace", "created_at": "2026-04-23T10:00:00Z",
    }

def simulate_mint_token() -> dict:
    return {
        "token_id": "tok_xyz789", "token": "cf_tok_demo_abc123xyz",
        "scope": "write", "expires_at": "2026-04-23T11:00:00Z",
        "clone_url": "https://x:cf_tok_demo_abc123xyz@artifacts.cloudflare.net/git/molecule-ws-demo.git",
        "message": "Save this token — it cannot be retrieved again.",
    }

def simulate_fork_repo() -> dict:
    return {
        "fork": {"name": "molecule-ws-demo/experiment", "namespace": "molecule-prod",
                 "remote_url": "https://artifacts.cloudflare.net/git/molecule-ws-demo-experiment"},
        "object_count": 14,
        "remote_url": "https://artifacts.cloudflare.net/git/molecule-ws-demo-experiment",
    }


# ─────────────────────────────────────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────────────────────────────────────

def main():
    is_live = is_live_platform()

    print("""
    ╔══════════════════════════════════════════════════════════════════════╗
    ║       Cloudflare Artifacts Demo — PR #641 (molecule-core)             ║
    ║                                                                      ║
    ║  Every Molecule AI workspace can have its own Git repo on            ║
    ║  Cloudflare's edge — versioned snapshots, isolated forks,           ║
    ║  short-lived credentials.                                             ║
    ╚══════════════════════════════════════════════════════════════════════╝
    """)

    # ── Step 1 ────────────────────────────────────────────────────────────
    divider("Step 1 — Attach a Cloudflare Artifacts repo")
    print("  POST /workspaces/:id/artifacts")
    print("  Body: {\"name\": \"agent-demo\", \"description\": \"Demo workspace\"}")
    print()
    if is_live:
        print("  → calling live platform...")
        try:
            result = ArtifactsClient(PLATFORM_URL, WORKSPACE_TOKEN, WORKSPACE_ID).attach_repo(
                name="agent-demo", description="Demo workspace")
            print(f"  ✓ Repo created: {result['cf_repo_name']}")
        except Exception as e:
            print(f"  ✗ Error: {e}")
    else:
        result = simulate_attach_repo()
        print(f"  ✓ Repo linked:")
        print(f"    cf_repo_name : {result['cf_repo_name']}")
        print(f"    cf_namespace : {result['cf_namespace']}")
        print(f"    remote_url   : {result['remote_url']}")

    # ── Step 2 ────────────────────────────────────────────────────────────
    divider("Step 2 — Mint a short-lived Git credential")
    print("  POST /workspaces/:id/artifacts/token")
    print("  Body: {\"scope\": \"write\", \"ttl\": 3600}")
    print()
    if is_live:
        print("  → calling live platform...")
        try:
            token_resp = ArtifactsClient(PLATFORM_URL, WORKSPACE_TOKEN, WORKSPACE_ID).mint_token()
            clone_url = token_resp["clone_url"]
            print(f"  ✓ Token minted (expires: {token_resp['expires_at']})")
            print(f"    clone_url: {clone_url[:60]}...")
        except Exception as e:
            print(f"  ✗ Error: {e}")
            clone_url = None
    else:
        token_resp = simulate_mint_token()
        clone_url = token_resp["clone_url"]
        print(f"  ✓ Token minted:")
        print(f"    token_id   : {token_resp['token_id']}")
        print(f"    scope      : {token_resp['scope']}")
        print(f"    expires_at : {token_resp['expires_at']}")
        print(f"    clone_url  : {token_resp['clone_url'][:60]}...")

    # ── Step 3 ────────────────────────────────────────────────────────────
    divider("Step 3 — Git clone, write, commit, push")
    print("  The agent uses the clone_url from Step 2:")
    print()
    print("    git clone <clone_url> demo-workspace")
    print("    # write AGENT_SNAPSHOT.md")
    print("    git add AGENT_SNAPSHOT.md")
    print("    git commit -m 'feat: agent run snapshot'")
    print("    git push origin HEAD")
    print()
    print("  Every agent run becomes a Git commit — versioned, auditable,")
    print("  and fork-able before a risky experiment.")
    print()
    if is_live and clone_url:
        snapshot = f"# Agent Run — {time.strftime('%Y-%m-%d %H:%M UTC')}\nWorkspace: {WORKSPACE_ID}\n"
        r = clone_and_push(clone_url, "demo-workspace", "feat: demo agent run snapshot", snapshot)
        if r.success:
            print(f"  ✓ Committed and pushed.")
        else:
            print(f"  ✗ Git error: {r.stderr.strip()}")
    else:
        print("  ⚠ SKIPPED (set PLATFORM_URL + WORKSPACE_TOKEN for live git ops)")

    # ── Step 4 ────────────────────────────────────────────────────────────
    divider("Step 4 — Fork before a risky experiment")
    print("  POST /workspaces/:id/artifacts/fork")
    print("  Body: {\"name\": \"agent-demo/experiment\", \"default_branch_only\": true}")
    print()
    print("  A fork is an isolated copy. Main stays clean.")
    print("  If the experiment succeeds: merge back. If it fails: discard fork.")
    print()
    if is_live:
        print("  → calling live platform...")
        try:
            fork_resp = ArtifactsClient(PLATFORM_URL, WORKSPACE_TOKEN, WORKSPACE_ID).fork_repo(
                name="agent-demo/experiment",
                description="Fork for experimental auth strategy",
                default_branch_only=True)
            print(f"  ✓ Fork created ({fork_resp.get('object_count','?')} objects)")
            print(f"    fork_url: {fork_resp.get('remote_url','')}")
        except Exception as e:
            print(f"  ✗ Error: {e}")
    else:
        fork_resp = simulate_fork_repo()
        print(f"  ✓ Fork created:")
        print(f"    name         : {fork_resp['fork']['name']}")
        print(f"    object_count : {fork_resp['object_count']}")
        print(f"    remote_url   : {fork_resp['remote_url']}")

    # ── Architecture ────────────────────────────────────────────────────────
    divider("Architecture Summary")
    print(textwrap.dedent("""\
      POST /workspaces/:id/artifacts
        → CF Artifacts API: CreateRepo / ImportRepo
        → workspace_artifacts DB row (credentials stripped — never persisted)

      POST /workspaces/:id/artifacts/token
        → CF API: CreateToken (short-lived, scoped)
        → Returns: token (shown once) + clone_url
        → Credential never stored server-side

      Agent side
        → git clone <clone_url>
        → git commit (agent run snapshot)
        → git push (pushed to CF edge git — fast global reads)

      POST /workspaces/:id/artifacts/fork
        → CF API: ForkRepo
        → NOT recorded in workspace_artifacts — caller owns the fork

      Security model:
      • CF_ARTIFACTS_API_TOKEN stored in platform env, not in DB
      • Repo credentials stripped before DB persistence (stripCredentials)
      • Per-call token minting — each git op uses a short-lived credential
      • Token scope: read | write, max TTL 7 days
      • Import URLs must be https:// (SSRF protection in artifacts.go:168)
    """))

    divider("Reference")
    print("  Handler : workspace-server/internal/handlers/artifacts.go")
    print("  Tests   : workspace-server/internal/handlers/artifacts_test.go")
    print("  DB table: workspace_artifacts")
    print("  Env vars: CF_ARTIFACTS_API_TOKEN, CF_ARTIFACTS_NAMESPACE")
    print()
    print("  Set PLATFORM_URL + WORKSPACE_TOKEN to run against a live platform.")
    print("  Demo path: docs/marketing/devrel/demos/cloudflare-artifacts/")


if __name__ == "__main__":
    main()