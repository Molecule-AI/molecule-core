"""RemoteAgentClient — blocking HTTP client for the Phase 30 remote-agent flow.

The client is deliberately dependency-light (``requests`` only) so a remote
agent author can drop it into any runtime. All methods correspond 1:1 to
a Phase 30 endpoint:

* :py:meth:`register`        → ``POST /registry/register``           (30.1)
* :py:meth:`pull_secrets`    → ``GET  /workspaces/:id/secrets/values`` (30.2)
* :py:meth:`poll_state`      → ``GET  /workspaces/:id/state``         (30.4)
* :py:meth:`heartbeat`       → ``POST /registry/heartbeat``           (30.1)
* :py:meth:`run_heartbeat_loop` — drives heartbeat + state-poll on a timer,
  returns when the platform reports the workspace paused or deleted.

No inbound A2A server is bundled here yet — that requires hosting an HTTP
endpoint the platform's proxy can reach, which is network-dependent. A
future 30.8b iteration will add an optional ``start_a2a_server()`` helper.
"""
from __future__ import annotations

import logging
import os
import stat
import subprocess
import tarfile
import time
import uuid
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

import requests

logger = logging.getLogger(__name__)

# Polling cadence defaults. Chosen to align with the platform's 60-second
# Redis TTL — one heartbeat per minute keeps the TTL refreshed; state-poll
# at the same cadence is cheap (tiny GET) and gives ≤60s reaction time on
# pause / delete. Overridable via RemoteAgentClient constructor kwargs.
DEFAULT_HEARTBEAT_INTERVAL = 30.0    # seconds
DEFAULT_STATE_POLL_INTERVAL = 30.0   # seconds

# Phase 30.6 — sibling URL cache TTL. Cached URLs expire after this many
# seconds, forcing a re-discovery call. Short enough that a sibling that
# moved (restart with new port) is picked up quickly; long enough that
# we don't hit the discovery endpoint on every A2A call.
DEFAULT_URL_CACHE_TTL = 300.0        # 5 minutes


def _safe_extract_tar(tf: tarfile.TarFile, dest: Path) -> None:
    """Extract a tarfile, refusing entries that would escape `dest`
    and silently skipping symlinks/hardlinks.

    Tar archives can include ``..`` and absolute paths. Without explicit
    rejection we'd risk the classic "tar slip" CVE — a malicious plugin
    source could overwrite the agent's home files. We resolve every
    entry's target to an absolute path, verify it lives inside dest, and
    extract one-by-one so the symlink-skip actually takes effect (a bare
    ``extractall`` would still write the symlinks we marked as skipped).
    """
    dest_abs = dest.resolve()
    for member in tf.getmembers():
        # Symlinks and hardlinks could point outside the staged tree;
        # skip them entirely (matches the platform-side tar producer).
        if member.issym() or member.islnk():
            continue
        target = (dest / member.name).resolve()
        try:
            target.relative_to(dest_abs)
        except ValueError as exc:
            raise ValueError(
                f"refusing tar entry escaping dest: {member.name!r} -> {target}"
            ) from exc
        tf.extract(member, dest)


def _rmtree_quiet(path: Path) -> None:
    """rm -rf <path> swallowing missing-file errors. Used for atomic
    install rollback where we sometimes call this on a non-existent
    staging dir."""
    import shutil
    try:
        shutil.rmtree(path)
    except FileNotFoundError:
        pass
    except Exception as exc:
        logger.warning("rmtree(%s) failed: %s", path, exc)


@dataclass
class WorkspaceState:
    """Snapshot of a remote workspace's platform-side state."""
    workspace_id: str
    status: str         # "online" / "paused" / "degraded" / "removed" / "offline" / ...
    paused: bool
    deleted: bool

    @property
    def should_stop(self) -> bool:
        """True when the agent should exit its run loop — platform has
        paused or hard-deleted the workspace. The agent can be restarted
        later; we just don't want to keep heartbeating against a dead row.
        """
        return self.paused or self.deleted


@dataclass
class PeerInfo:
    """A sibling or parent workspace that this agent can communicate with."""
    id: str
    name: str
    url: str
    role: str = ""
    tier: int = 2
    status: str = "unknown"
    agent_card: dict[str, Any] = field(default_factory=dict)


class RemoteAgentClient:
    """Blocking HTTP client for a Phase 30 remote agent.

    Args:
        workspace_id: UUID of the workspace this agent represents. The
            workspace row must exist on the platform (created via
            ``POST /workspaces`` or ``POST /org/import``) — the agent
            claims that identity when it registers.
        platform_url: Base URL of the platform, e.g.
            ``https://molecule.example.com``. No trailing slash; the
            client adds paths.
        agent_card: A2A agent card payload. Minimal: ``{"name": "..."}``.
            Full schema matches what an in-container agent would report
            (skills list, capabilities, etc.).
        reported_url: Optional externally-reachable URL at which siblings
            can call this agent's A2A endpoint. If omitted, the agent is
            reachable only via the platform's proxy (which won't be able
            to initiate calls to the agent either — that's a limitation
            of remote agents today, resolved by 30.6/30.7 or by exposing
            an inbound endpoint yourself).
        token_dir: Where to cache the workspace auth token on disk.
            Defaults to ``~/.molecule/<workspace_id>/``. Created with
            0700 permissions if missing.
        heartbeat_interval: Seconds between heartbeats in the run loop.
        state_poll_interval: Seconds between state polls in the run loop.
    """

    def __init__(
        self,
        workspace_id: str,
        platform_url: str,
        agent_card: dict[str, Any] | None = None,
        reported_url: str = "",
        token_dir: Path | None = None,
        heartbeat_interval: float = DEFAULT_HEARTBEAT_INTERVAL,
        state_poll_interval: float = DEFAULT_STATE_POLL_INTERVAL,
        url_cache_ttl: float = DEFAULT_URL_CACHE_TTL,
        session: requests.Session | None = None,
    ) -> None:
        self.workspace_id = workspace_id
        self.platform_url = platform_url.rstrip("/")
        self.agent_card = agent_card or {"name": f"remote-agent-{workspace_id[:8]}"}
        self.reported_url = reported_url
        self.heartbeat_interval = heartbeat_interval
        self.state_poll_interval = state_poll_interval
        self.url_cache_ttl = url_cache_ttl
        # Phase 30.6 — sibling URL cache keyed by workspace id. Values are
        # (url, expires_at_unix_seconds). Process-memory only; we re-fetch
        # on restart because agent lifetimes are short enough that
        # persisting doesn't buy much.
        self._url_cache: dict[str, tuple[str, float]] = {}
        self._session = session or requests.Session()
        self._token_dir = token_dir or (
            Path.home() / ".molecule" / workspace_id
        )
        self._token: str | None = None
        self._start_time = time.time()

    # ------------------------------------------------------------------
    # Token persistence
    # ------------------------------------------------------------------

    @property
    def token_file(self) -> Path:
        return self._token_dir / ".auth_token"

    def load_token(self) -> str | None:
        """Load a cached token from disk if present. Populates the
        in-memory cache on success."""
        if self._token is not None:
            return self._token
        if not self.token_file.exists():
            return None
        try:
            tok = self.token_file.read_text().strip()
        except OSError as exc:
            logger.warning("failed to read %s: %s", self.token_file, exc)
            return None
        if not tok:
            return None
        self._token = tok
        return tok

    def save_token(self, token: str) -> None:
        """Persist a freshly-issued token to disk. Creates the parent
        directory with 0700 and the file with 0600 to keep the credential
        off other users' prying eyes."""
        token = token.strip()
        if not token:
            raise ValueError("refusing to save empty token")
        self._token_dir.mkdir(parents=True, exist_ok=True)
        try:
            os.chmod(self._token_dir, 0o700)
        except OSError:
            pass  # non-fatal — best-effort on unusual filesystems
        self.token_file.write_text(token)
        try:
            os.chmod(self.token_file, 0o600)
        except OSError:
            pass
        self._token = token

    def _auth_headers(self) -> dict[str, str]:
        tok = self.load_token()
        if not tok:
            return {}
        return {"Authorization": f"Bearer {tok}"}

    # ------------------------------------------------------------------
    # Endpoints
    # ------------------------------------------------------------------

    def register(self) -> str:
        """Register with the platform and cache the issued auth token.

        Returns the token (also persisted to disk). If the platform has
        already issued a token for this workspace (identified by the
        cached file), register will still succeed but the response will
        not include a new ``auth_token`` — the client keeps using the
        on-disk copy.

        Raises :class:`requests.HTTPError` on non-2xx responses.
        """
        # The platform's RegisterPayload requires a non-empty url. A remote
        # agent that doesn't expose inbound A2A yet still needs a placeholder
        # — we use "remote://no-inbound" so the platform can distinguish it
        # from a real HTTP URL and not try to reach the agent.
        reported = self.reported_url or "remote://no-inbound"
        resp = self._session.post(
            f"{self.platform_url}/registry/register",
            json={
                "id": self.workspace_id,
                "url": reported,
                "agent_card": self.agent_card,
            },
            timeout=10.0,
        )
        resp.raise_for_status()
        data = resp.json()
        tok = data.get("auth_token", "")
        if tok:
            self.save_token(tok)
            logger.info("registered and saved new token (prefix=%s…)", tok[:8])
        else:
            # Already-tokened workspace — keep using the cached one.
            existing = self.load_token()
            if not existing:
                logger.warning(
                    "register returned no auth_token and no cached token exists — "
                    "authenticated calls will 401 until a token is minted"
                )
        return self._token or ""

    def pull_secrets(self) -> dict[str, str]:
        """Fetch the merged decrypted secrets via the 30.2 endpoint.

        Returns an empty dict when the platform has no secrets configured
        for this workspace. Raises on network errors and on 401 (which
        means the token is missing / invalid — call :py:meth:`register`
        first).
        """
        resp = self._session.get(
            f"{self.platform_url}/workspaces/{self.workspace_id}/secrets/values",
            headers=self._auth_headers(),
            timeout=10.0,
        )
        resp.raise_for_status()
        return resp.json() or {}

    def poll_state(self) -> WorkspaceState | None:
        """Fetch the workspace's current state via the 30.4 endpoint.

        Returns None if the platform returns 404 with ``{"deleted": true}``
        (workspace hard-deleted) — callers typically exit their run loop
        in that case. Raises on other HTTP errors.
        """
        resp = self._session.get(
            f"{self.platform_url}/workspaces/{self.workspace_id}/state",
            headers=self._auth_headers(),
            timeout=10.0,
        )
        if resp.status_code == 404:
            # Platform signals hard-delete via 404 + deleted:true
            return WorkspaceState(
                workspace_id=self.workspace_id,
                status="removed",
                paused=False,
                deleted=True,
            )
        resp.raise_for_status()
        data = resp.json()
        return WorkspaceState(
            workspace_id=self.workspace_id,
            status=str(data.get("status", "unknown")),
            paused=bool(data.get("paused", False)),
            deleted=bool(data.get("deleted", False)),
        )

    def heartbeat(
        self,
        current_task: str = "",
        active_tasks: int = 0,
        error_rate: float = 0.0,
        sample_error: str = "",
    ) -> None:
        """Send a single heartbeat. Safe to call repeatedly — the platform
        treats it as idempotent state-refresh. Raises on non-2xx."""
        uptime = int(time.time() - self._start_time)
        resp = self._session.post(
            f"{self.platform_url}/registry/heartbeat",
            headers=self._auth_headers(),
            json={
                "workspace_id": self.workspace_id,
                "current_task": current_task,
                "active_tasks": active_tasks,
                "error_rate": error_rate,
                "sample_error": sample_error,
                "uptime_seconds": uptime,
            },
            timeout=10.0,
        )
        resp.raise_for_status()

    # ------------------------------------------------------------------
    # Peer discovery + cache (Phase 30.6)
    # ------------------------------------------------------------------

    def get_peers(self) -> list[PeerInfo]:
        """Fetch the list of peer workspaces this agent can communicate with.

        Hits ``GET /registry/:id/peers`` with the bearer token. The returned
        list includes siblings (same parent) and, if applicable, the parent.
        Each peer's URL is seeded into the local cache so subsequent calls
        to :py:meth:`discover_peer` short-circuit without hitting the
        platform.

        Raises on 401 (stale/missing token → call :py:meth:`register`) and
        other non-2xx.
        """
        resp = self._session.get(
            f"{self.platform_url}/registry/{self.workspace_id}/peers",
            headers={
                **self._auth_headers(),
                "X-Workspace-ID": self.workspace_id,
            },
            timeout=10.0,
        )
        resp.raise_for_status()
        data = resp.json() or []
        peers: list[PeerInfo] = []
        now = time.time()
        for row in data:
            pid = str(row.get("id", ""))
            url = str(row.get("url", ""))
            if not pid:
                continue
            peer = PeerInfo(
                id=pid,
                name=str(row.get("name", "")),
                url=url,
                role=str(row.get("role", "")),
                tier=int(row.get("tier", 2) or 2),
                status=str(row.get("status", "unknown")),
                agent_card=row.get("agent_card") or {},
            )
            peers.append(peer)
            # Seed the cache so a subsequent call_peer doesn't need a
            # discover round-trip. Only cache HTTP-shaped URLs; skip the
            # "remote://no-inbound" placeholder and empty strings.
            if url.startswith(("http://", "https://")):
                self._url_cache[pid] = (url, now + self.url_cache_ttl)
        return peers

    def discover_peer(self, target_id: str) -> str | None:
        """Resolve a peer's URL, using the cache when fresh.

        Returns the URL string, or None if the platform has no usable URL
        for this target. On 401/403 the caller should re-authenticate or
        verify the hierarchy rule; those are raised as ``HTTPError``.

        Cache semantics: a cached entry is returned immediately if its TTL
        hasn't expired; otherwise the platform is hit and the cache
        refreshed. Call :py:meth:`invalidate_peer_url` to drop an entry
        that was stale (connection error, 5xx) so the next discover
        re-fetches instead of returning the dead URL again.
        """
        cached = self._url_cache.get(target_id)
        if cached is not None:
            url, expires_at = cached
            if time.time() < expires_at:
                return url
            # Expired — drop and fall through to refresh
            self._url_cache.pop(target_id, None)

        resp = self._session.get(
            f"{self.platform_url}/registry/discover/{target_id}",
            headers={
                **self._auth_headers(),
                "X-Workspace-ID": self.workspace_id,
            },
            timeout=10.0,
        )
        if resp.status_code == 404:
            return None
        resp.raise_for_status()
        url = str((resp.json() or {}).get("url", ""))
        if url.startswith(("http://", "https://")):
            self._url_cache[target_id] = (url, time.time() + self.url_cache_ttl)
            return url
        return None

    def invalidate_peer_url(self, target_id: str) -> None:
        """Drop a peer's cached URL. Call this after a direct-call failure
        so the next call_peer performs a fresh discover."""
        self._url_cache.pop(target_id, None)

    def call_peer(
        self,
        target_id: str,
        message: str,
        prefer_direct: bool = True,
    ) -> dict[str, Any]:
        """Send an A2A ``message/send`` to a peer.

        Preferred path (``prefer_direct=True``, default):
            1. Resolve target URL via :py:meth:`discover_peer` (cache-hot
               path when we've seen this peer before).
            2. POST the JSON-RPC envelope directly to the peer's URL.
            3. On connection error / 5xx, invalidate the cache and retry
               via the platform proxy — graceful fallback so a stale URL
               doesn't brick inter-agent communication.

        Proxy-only path (``prefer_direct=False``):
            Always routes through ``POST /workspaces/:id/a2a`` — useful
            when both agents are behind NAT and can't reach each other
            directly, but the platform can reach both.

        Returns the full JSON-RPC response dict so callers can inspect
        ``result`` vs ``error`` without us flattening the envelope.
        """
        body = {
            "jsonrpc": "2.0",
            "id": str(uuid.uuid4()),
            "method": "message/send",
            "params": {
                "message": {
                    "role": "user",
                    "messageId": str(uuid.uuid4()),
                    "parts": [{"kind": "text", "text": message}],
                }
            },
        }
        headers = {
            **self._auth_headers(),
            "X-Workspace-ID": self.workspace_id,
            "Content-Type": "application/json",
        }

        if prefer_direct:
            url = self.discover_peer(target_id)
            if url:
                try:
                    resp = self._session.post(url, json=body, headers=headers, timeout=30.0)
                    resp.raise_for_status()
                    return resp.json()
                except Exception as exc:
                    logger.warning(
                        "direct A2A to %s (%s) failed: %s — invalidating cache, falling back to proxy",
                        target_id, url, exc,
                    )
                    self.invalidate_peer_url(target_id)

        # Proxy fallback (or prefer_direct=False)
        resp = self._session.post(
            f"{self.platform_url}/workspaces/{target_id}/a2a",
            json=body, headers=headers, timeout=30.0,
        )
        resp.raise_for_status()
        return resp.json()

    # ------------------------------------------------------------------
    # Plugin install (Phase 30.3)
    # ------------------------------------------------------------------

    @property
    def plugins_dir(self) -> Path:
        """Where pulled plugins are unpacked. Lives under the same
        per-workspace directory as the auth token (``~/.molecule/<id>/``)
        so a single rm cleans the agent's local state."""
        return self._token_dir / "plugins"

    def install_plugin(
        self,
        name: str,
        source: str | None = None,
        run_setup_sh: bool = True,
        report_to_platform: bool = True,
    ) -> Path:
        """Pull a plugin tarball from the platform, unpack it, optionally
        run its ``setup.sh``, and report success.

        Phase 30.3 contract:

        1. Stream ``GET /workspaces/:id/plugins/:name/download[?source=…]``
        2. Atomically extract into ``~/.molecule/<workspace>/plugins/<name>/``
           via a sibling-tempdir + rename so a partial extract never
           leaves the directory in a half-installed state.
        3. If ``setup.sh`` exists in the unpacked tree, run it (bash) so
           pip/npm deps land on the agent's machine. Failures from
           ``setup.sh`` are logged but don't prevent the install record
           — the agent author can re-run setup manually.
        4. POST ``/workspaces/:id/plugins`` with the source string so the
           platform's ``workspace_plugins`` table reflects the install.

        Returns the path to the unpacked plugin directory.
        Raises ``requests.HTTPError`` on download failure (401 / 404 / etc.).
        """
        target = self.plugins_dir / name
        staging = self.plugins_dir / f".staging-{name}-{uuid.uuid4().hex[:8]}"
        self.plugins_dir.mkdir(parents=True, exist_ok=True)

        url = f"{self.platform_url}/workspaces/{self.workspace_id}/plugins/{name}/download"
        params: dict[str, str] = {}
        if source:
            params["source"] = source

        # We pull the whole tarball into memory before extracting. The
        # platform side (PLUGIN_INSTALL_MAX_DIR_BYTES) caps plugin size
        # at 100 MiB by default, which is comfortable to hold in process
        # memory and lets us use tarfile's seekable r:gz mode (the
        # streaming r|gz mode is sequential-only and breaks on tarballs
        # whose entries weren't sorted by name, which we don't enforce).
        # If the cap is ever raised above ~500 MiB, switch to a temp
        # file: tarfile.open(fileobj=open(temp, "rb"), mode="r:gz").
        with self._session.get(
            url, headers=self._auth_headers(), params=params,
            timeout=60.0,
        ) as resp:
            resp.raise_for_status()
            staging.mkdir(parents=True)
            try:
                import io as _io
                with tarfile.open(fileobj=_io.BytesIO(resp.content), mode="r:gz") as tf:
                    _safe_extract_tar(tf, staging)
            except Exception:
                # Roll back the partial extract
                _rmtree_quiet(staging)
                raise

        # Atomic swap: remove old dir if present, rename staging into place.
        if target.exists():
            _rmtree_quiet(target)
        staging.rename(target)
        logger.info("plugin %s unpacked to %s", name, target)

        # 3. setup.sh — best-effort. We never raise on its failure because
        # the plugin files are now correctly installed; setup is just for
        # heavy deps that the agent author can rerun manually.
        if run_setup_sh:
            setup = target / "setup.sh"
            if setup.is_file():
                try:
                    proc = subprocess.run(
                        ["bash", str(setup)],
                        cwd=str(target),
                        capture_output=True, text=True, timeout=120,
                    )
                    if proc.returncode == 0:
                        logger.info("plugin %s setup.sh ok", name)
                    else:
                        logger.warning(
                            "plugin %s setup.sh exit=%d stderr=%s",
                            name, proc.returncode, proc.stderr[:200],
                        )
                except subprocess.TimeoutExpired:
                    logger.warning("plugin %s setup.sh timed out (120s)", name)
                except FileNotFoundError:
                    logger.warning("plugin %s setup.sh present but bash not found", name)

        # 4. Report to platform — write a workspace_plugins row so List
        # reflects the install. Best-effort; the local files are correct
        # regardless of whether this POST succeeds.
        if report_to_platform:
            try:
                report_source = source or f"local://{name}"
                self._session.post(
                    f"{self.platform_url}/workspaces/{self.workspace_id}/plugins",
                    headers=self._auth_headers(),
                    json={"source": report_source},
                    timeout=10.0,
                )
            except Exception as exc:
                logger.warning("plugin %s install record POST failed: %s", name, exc)

        return target

    # ------------------------------------------------------------------
    # Run loop
    # ------------------------------------------------------------------

    def run_heartbeat_loop(
        self,
        max_iterations: int | None = None,
        task_supplier: "callable | None" = None,
    ) -> str:
        """Drive heartbeat + state-poll on a timer. Returns the terminal
        status when the loop exits (``"paused"``, ``"removed"``, or
        ``"max_iterations"``).

        Args:
            max_iterations: Stop after N loop iterations. None = run until
                the workspace is paused / deleted. Useful for tests and
                smoke scripts.
            task_supplier: Optional zero-arg callable returning a dict
                ``{"current_task": str, "active_tasks": int}`` fetched
                each iteration. Lets the agent report what it's doing.

        The loop sends one heartbeat + one state poll per iteration; the
        next iteration sleeps for ``heartbeat_interval`` seconds. Errors
        from either call are logged and the loop continues — we deliberately
        do NOT re-raise because a transient platform hiccup shouldn't take
        a remote agent offline.
        """
        i = 0
        while True:
            if max_iterations is not None and i >= max_iterations:
                return "max_iterations"
            i += 1

            report: dict[str, Any] = {}
            if task_supplier is not None:
                try:
                    report = task_supplier() or {}
                except Exception as exc:
                    logger.warning("task_supplier raised: %s", exc)

            try:
                self.heartbeat(
                    current_task=str(report.get("current_task", "")),
                    active_tasks=int(report.get("active_tasks", 0)),
                )
            except Exception as exc:
                logger.warning("heartbeat failed: %s — continuing", exc)

            try:
                state = self.poll_state()
            except Exception as exc:
                logger.warning("state poll failed: %s — continuing", exc)
                state = None

            if state is not None and state.should_stop:
                logger.info(
                    "platform reports workspace %s (paused=%s deleted=%s) — exiting",
                    state.status, state.paused, state.deleted,
                )
                return state.status

            time.sleep(self.heartbeat_interval)


__all__ = [
    "RemoteAgentClient",
    "WorkspaceState",
    "PeerInfo",
    "DEFAULT_HEARTBEAT_INTERVAL",
    "DEFAULT_STATE_POLL_INTERVAL",
    "DEFAULT_URL_CACHE_TTL",
]
