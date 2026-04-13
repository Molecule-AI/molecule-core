"""Tests for the molecule_agent Phase 30.8 remote-agent client.

The client is pure HTTP — we mock the network via ``requests_mock``-style
monkey-patching of ``requests.Session.get`` / ``.post`` instead of pulling
in a third-party mock library.
"""
from __future__ import annotations

import stat
import time
from pathlib import Path
from typing import Any
from unittest.mock import MagicMock

import pytest

from molecule_agent import PeerInfo, RemoteAgentClient, WorkspaceState


# ---------------------------------------------------------------------------
# FakeResponse / FakeSession — minimal stand-ins for requests
# ---------------------------------------------------------------------------


class FakeResponse:
    def __init__(self, status_code: int = 200, json_body: Any = None, text: str = ""):
        self.status_code = status_code
        self._json = json_body
        self.text = text

    def json(self) -> Any:
        return self._json

    def raise_for_status(self) -> None:
        if self.status_code >= 400:
            import requests
            raise requests.HTTPError(f"HTTP {self.status_code}")


@pytest.fixture
def tmp_token_dir(tmp_path: Path) -> Path:
    return tmp_path / "molecule-token-cache"


@pytest.fixture
def client(tmp_token_dir: Path) -> RemoteAgentClient:
    session = MagicMock()
    return RemoteAgentClient(
        workspace_id="ws-abc-123",
        platform_url="http://platform.test",
        agent_card={"name": "test-agent"},
        token_dir=tmp_token_dir,
        session=session,
    )


# ---------------------------------------------------------------------------
# Token persistence
# ---------------------------------------------------------------------------


def test_save_and_load_token_roundtrip(client: RemoteAgentClient, tmp_token_dir: Path):
    client.save_token("secret-token-abc")
    assert client.token_file.exists()
    # File must be 0600 so other local users can't read the credential.
    mode = stat.S_IMODE(client.token_file.stat().st_mode)
    assert mode == 0o600, f"expected 0600, got 0o{mode:o}"
    assert client.load_token() == "secret-token-abc"


def test_save_empty_token_rejected(client: RemoteAgentClient):
    with pytest.raises(ValueError):
        client.save_token("")
    with pytest.raises(ValueError):
        client.save_token("   ")


def test_load_token_returns_none_when_absent(client: RemoteAgentClient):
    assert client.load_token() is None


def test_load_token_returns_none_when_file_empty(client: RemoteAgentClient, tmp_token_dir: Path):
    tmp_token_dir.mkdir(parents=True, exist_ok=True)
    (tmp_token_dir / ".auth_token").write_text("")
    assert client.load_token() is None


def test_token_dir_default_is_under_home(tmp_path: Path):
    # Just verifies the default path shape — we don't want to actually
    # write to $HOME during tests.
    c = RemoteAgentClient(
        workspace_id="ws-xyz",
        platform_url="http://p",
    )
    assert "ws-xyz" in str(c.token_file)
    assert ".molecule" in str(c.token_file)


# ---------------------------------------------------------------------------
# register()
# ---------------------------------------------------------------------------


def test_register_saves_token_when_issued(client: RemoteAgentClient):
    client._session.post.return_value = FakeResponse(
        200, {"status": "registered", "auth_token": "fresh-token-xyz"}
    )

    tok = client.register()

    assert tok == "fresh-token-xyz"
    assert client.load_token() == "fresh-token-xyz"
    # Verify call shape
    url, kwargs = client._session.post.call_args[0][0], client._session.post.call_args[1]
    assert url == "http://platform.test/registry/register"
    assert kwargs["json"]["id"] == "ws-abc-123"
    assert kwargs["json"]["agent_card"] == {"name": "test-agent"}


def test_register_keeps_cached_token_when_platform_omits(client: RemoteAgentClient):
    # Simulate re-register of an already-tokened workspace: platform returns
    # no auth_token, SDK must keep using the cached one.
    client.save_token("cached-from-earlier")
    client._session.post.return_value = FakeResponse(200, {"status": "registered"})

    tok = client.register()
    assert tok == "cached-from-earlier"


def test_register_http_error_propagates(client: RemoteAgentClient):
    client._session.post.return_value = FakeResponse(500, {"error": "boom"})
    with pytest.raises(Exception):
        client.register()


# ---------------------------------------------------------------------------
# pull_secrets()
# ---------------------------------------------------------------------------


def test_pull_secrets_sends_bearer_token(client: RemoteAgentClient):
    client.save_token("tok-for-secrets")
    client._session.get.return_value = FakeResponse(200, {"API_KEY": "v1", "DB_URL": "v2"})

    out = client.pull_secrets()

    assert out == {"API_KEY": "v1", "DB_URL": "v2"}
    url, kwargs = client._session.get.call_args[0][0], client._session.get.call_args[1]
    assert url == "http://platform.test/workspaces/ws-abc-123/secrets/values"
    assert kwargs["headers"]["Authorization"] == "Bearer tok-for-secrets"


def test_pull_secrets_empty_body_yields_empty_dict(client: RemoteAgentClient):
    client.save_token("t")
    client._session.get.return_value = FakeResponse(200, None)
    assert client.pull_secrets() == {}


def test_pull_secrets_401_raises(client: RemoteAgentClient):
    client.save_token("t")
    client._session.get.return_value = FakeResponse(401, {"error": "missing token"})
    with pytest.raises(Exception):
        client.pull_secrets()


# ---------------------------------------------------------------------------
# poll_state()
# ---------------------------------------------------------------------------


def test_poll_state_returns_normal_state(client: RemoteAgentClient):
    client.save_token("t")
    client._session.get.return_value = FakeResponse(
        200, {"workspace_id": "ws-abc-123", "status": "online", "paused": False, "deleted": False}
    )

    state = client.poll_state()

    assert state is not None
    assert state.status == "online"
    assert state.paused is False
    assert state.deleted is False
    assert state.should_stop is False


def test_poll_state_detects_paused(client: RemoteAgentClient):
    client.save_token("t")
    client._session.get.return_value = FakeResponse(
        200, {"workspace_id": "ws-abc-123", "status": "paused", "paused": True, "deleted": False}
    )
    state = client.poll_state()
    assert state.should_stop is True


def test_poll_state_404_means_deleted(client: RemoteAgentClient):
    client.save_token("t")
    client._session.get.return_value = FakeResponse(404, {"deleted": True})

    state = client.poll_state()

    assert state is not None
    assert state.deleted is True
    assert state.should_stop is True


def test_poll_state_server_error_raises(client: RemoteAgentClient):
    client.save_token("t")
    client._session.get.return_value = FakeResponse(500, {"error": "boom"})
    with pytest.raises(Exception):
        client.poll_state()


# ---------------------------------------------------------------------------
# heartbeat()
# ---------------------------------------------------------------------------


def test_heartbeat_sends_full_payload(client: RemoteAgentClient):
    client.save_token("t")
    client._session.post.return_value = FakeResponse(200, {"status": "ok"})

    client.heartbeat(current_task="indexing", active_tasks=1, error_rate=0.1, sample_error="err")

    url = client._session.post.call_args[0][0]
    kwargs = client._session.post.call_args[1]
    assert url == "http://platform.test/registry/heartbeat"
    body = kwargs["json"]
    assert body["workspace_id"] == "ws-abc-123"
    assert body["current_task"] == "indexing"
    assert body["active_tasks"] == 1
    assert body["error_rate"] == 0.1
    assert body["sample_error"] == "err"
    assert "uptime_seconds" in body
    assert kwargs["headers"]["Authorization"] == "Bearer t"


# ---------------------------------------------------------------------------
# run_heartbeat_loop()
# ---------------------------------------------------------------------------


def test_run_loop_exits_on_max_iterations(client: RemoteAgentClient, monkeypatch):
    # Stub sleep so the test doesn't actually wait
    import molecule_agent.client as mod
    monkeypatch.setattr(mod.time, "sleep", lambda s: None)

    client.save_token("t")
    client._session.post.return_value = FakeResponse(200, {"status": "ok"})
    client._session.get.return_value = FakeResponse(
        200, {"status": "online", "paused": False, "deleted": False}
    )

    terminal = client.run_heartbeat_loop(max_iterations=3)

    assert terminal == "max_iterations"
    # 3 heartbeats + 3 state polls
    assert client._session.post.call_count == 3
    assert client._session.get.call_count == 3


def test_run_loop_exits_on_paused(client: RemoteAgentClient, monkeypatch):
    import molecule_agent.client as mod
    monkeypatch.setattr(mod.time, "sleep", lambda s: None)

    client.save_token("t")
    client._session.post.return_value = FakeResponse(200, {"status": "ok"})
    # First iteration: online. Second: paused.
    responses = [
        FakeResponse(200, {"status": "online", "paused": False, "deleted": False}),
        FakeResponse(200, {"status": "paused", "paused": True, "deleted": False}),
    ]
    client._session.get.side_effect = responses

    terminal = client.run_heartbeat_loop(max_iterations=10)

    assert terminal == "paused"
    assert client._session.post.call_count == 2
    assert client._session.get.call_count == 2


def test_run_loop_exits_on_deleted_404(client: RemoteAgentClient, monkeypatch):
    import molecule_agent.client as mod
    monkeypatch.setattr(mod.time, "sleep", lambda s: None)

    client.save_token("t")
    client._session.post.return_value = FakeResponse(200, {"status": "ok"})
    client._session.get.return_value = FakeResponse(404, {"deleted": True})

    terminal = client.run_heartbeat_loop(max_iterations=10)

    assert terminal == "removed"
    assert client._session.get.call_count == 1


def test_run_loop_continues_through_transient_errors(client: RemoteAgentClient, monkeypatch):
    """Network hiccups must log-and-continue, never crash the loop."""
    import molecule_agent.client as mod
    monkeypatch.setattr(mod.time, "sleep", lambda s: None)

    client.save_token("t")

    # Heartbeat fails on iter 1, succeeds on iter 2
    client._session.post.side_effect = [
        ConnectionError("flaky net"),
        FakeResponse(200, {"status": "ok"}),
    ]
    # State poll returns online both times
    client._session.get.return_value = FakeResponse(
        200, {"status": "online", "paused": False, "deleted": False}
    )

    terminal = client.run_heartbeat_loop(max_iterations=2)
    assert terminal == "max_iterations"
    # Both iterations completed despite the first post failing
    assert client._session.post.call_count == 2


def test_run_loop_task_supplier_reported(client: RemoteAgentClient, monkeypatch):
    import molecule_agent.client as mod
    monkeypatch.setattr(mod.time, "sleep", lambda s: None)

    client.save_token("t")
    client._session.post.return_value = FakeResponse(200, {"status": "ok"})
    client._session.get.return_value = FakeResponse(
        200, {"status": "online", "paused": False, "deleted": False}
    )

    reports = [{"current_task": "step-1", "active_tasks": 1}]

    client.run_heartbeat_loop(max_iterations=1, task_supplier=lambda: reports[0])

    body = client._session.post.call_args[1]["json"]
    assert body["current_task"] == "step-1"
    assert body["active_tasks"] == 1


# ---------------------------------------------------------------------------
# WorkspaceState dataclass
# ---------------------------------------------------------------------------


def test_workspace_state_should_stop_semantics():
    assert WorkspaceState("w", "online", False, False).should_stop is False
    assert WorkspaceState("w", "degraded", False, False).should_stop is False
    assert WorkspaceState("w", "paused", True, False).should_stop is True
    assert WorkspaceState("w", "removed", False, True).should_stop is True


# ---------------------------------------------------------------------------
# Phase 30.6 — sibling URL cache + call_peer
# ---------------------------------------------------------------------------

def test_get_peers_seeds_cache(client: RemoteAgentClient):
    client.save_token("t")
    client._session.get.return_value = FakeResponse(200, [
        {"id": "sibling-1", "name": "Research", "url": "http://10.0.0.5:8000", "role": "researcher", "tier": 2, "status": "online"},
        {"id": "sibling-2", "name": "Dev",      "url": "http://10.0.0.6:8000", "role": "developer",  "tier": 2, "status": "online"},
    ])

    peers = client.get_peers()

    assert len(peers) == 2
    assert peers[0].id == "sibling-1"
    assert peers[0].name == "Research"
    assert peers[0].url == "http://10.0.0.5:8000"
    # Cache seeded for both
    assert client._url_cache["sibling-1"][0] == "http://10.0.0.5:8000"
    assert client._url_cache["sibling-2"][0] == "http://10.0.0.6:8000"
    # Request included bearer + X-Workspace-ID
    headers = client._session.get.call_args[1]["headers"]
    assert headers["Authorization"] == "Bearer t"
    assert headers["X-Workspace-ID"] == "ws-abc-123"


def test_get_peers_skips_non_http_urls_in_cache(client: RemoteAgentClient):
    """Cache seed only accepts http(s); the 'remote://no-inbound' placeholder
    for remote agents without inbound servers must not poison the cache."""
    client.save_token("t")
    client._session.get.return_value = FakeResponse(200, [
        {"id": "sib-remote", "name": "Remote", "url": "remote://no-inbound"},
        {"id": "sib-http",   "name": "HTTP",   "url": "http://192.168.1.7:8000"},
    ])

    client.get_peers()

    assert "sib-remote" not in client._url_cache
    assert "sib-http" in client._url_cache


def test_discover_peer_cache_hit(client: RemoteAgentClient):
    client._url_cache["sib-x"] = ("http://cached.url:8000", time.time() + 60)

    url = client.discover_peer("sib-x")

    assert url == "http://cached.url:8000"
    # No network call
    client._session.get.assert_not_called()


def test_discover_peer_cache_miss_hits_platform(client: RemoteAgentClient):
    client.save_token("t")
    client._session.get.return_value = FakeResponse(
        200, {"id": "sib-y", "url": "http://fresh.url:8000", "name": "Y"}
    )

    url = client.discover_peer("sib-y")

    assert url == "http://fresh.url:8000"
    assert client._url_cache["sib-y"][0] == "http://fresh.url:8000"
    # Request used discover endpoint
    called_url = client._session.get.call_args[0][0]
    assert "/registry/discover/sib-y" in called_url


def test_discover_peer_expired_cache_refreshes(client: RemoteAgentClient, monkeypatch):
    # Cache entry already expired
    client._url_cache["sib-stale"] = ("http://stale.url", time.time() - 10)
    client.save_token("t")
    client._session.get.return_value = FakeResponse(
        200, {"url": "http://fresh.url:9000"}
    )

    url = client.discover_peer("sib-stale")

    assert url == "http://fresh.url:9000"
    # Cache replaced with fresh entry
    assert client._url_cache["sib-stale"][0] == "http://fresh.url:9000"


def test_discover_peer_404_returns_none(client: RemoteAgentClient):
    client.save_token("t")
    client._session.get.return_value = FakeResponse(404, {"error": "not found"})
    assert client.discover_peer("missing") is None


def test_invalidate_peer_url_drops_cache_entry(client: RemoteAgentClient):
    client._url_cache["sib-x"] = ("http://x", time.time() + 100)
    client.invalidate_peer_url("sib-x")
    assert "sib-x" not in client._url_cache
    # Idempotent — second call is safe
    client.invalidate_peer_url("sib-x")


def test_call_peer_direct_path_on_cache_hit(client: RemoteAgentClient):
    client.save_token("t")
    client._url_cache["sib"] = ("http://direct.peer:8000", time.time() + 60)

    client._session.post.return_value = FakeResponse(
        200, {"jsonrpc": "2.0", "id": "x", "result": {"ok": True}}
    )

    out = client.call_peer("sib", "hello sibling")

    assert out["result"]["ok"] is True
    # Exactly ONE post: direct to the cached URL, not through proxy
    assert client._session.post.call_count == 1
    called_url = client._session.post.call_args[0][0]
    assert called_url == "http://direct.peer:8000"
    body = client._session.post.call_args[1]["json"]
    assert body["method"] == "message/send"
    assert body["params"]["message"]["parts"][0]["text"] == "hello sibling"
    headers = client._session.post.call_args[1]["headers"]
    assert headers["X-Workspace-ID"] == "ws-abc-123"


def test_call_peer_falls_back_to_proxy_on_direct_error(client: RemoteAgentClient):
    client.save_token("t")
    client._url_cache["sib"] = ("http://dead.peer:8000", time.time() + 60)

    # First post (direct): connection error. Second post (proxy): success.
    client._session.post.side_effect = [
        ConnectionError("unreachable"),
        FakeResponse(200, {"jsonrpc": "2.0", "result": {"via": "proxy"}}),
    ]

    out = client.call_peer("sib", "hello")

    assert out["result"]["via"] == "proxy"
    assert client._session.post.call_count == 2
    # Direct URL was invalidated so next call re-discovers
    assert "sib" not in client._url_cache
    # Second call went to /workspaces/sib/a2a
    proxy_url = client._session.post.call_args_list[1][0][0]
    assert "/workspaces/sib/a2a" in proxy_url


def test_call_peer_proxy_only_when_prefer_direct_false(client: RemoteAgentClient):
    client.save_token("t")
    client._url_cache["sib"] = ("http://direct.peer:8000", time.time() + 60)

    client._session.post.return_value = FakeResponse(
        200, {"jsonrpc": "2.0", "result": {"via": "proxy-only"}}
    )

    client.call_peer("sib", "hello", prefer_direct=False)

    # Exactly one post — went straight to proxy despite cache hit
    assert client._session.post.call_count == 1
    assert "/workspaces/sib/a2a" in client._session.post.call_args[0][0]


def test_call_peer_no_cached_url_uses_discover_then_direct(client: RemoteAgentClient):
    """Fresh call: no cache entry → discover via GET, then direct POST to the
    returned URL. Tests the full discover-then-call sequence in one shot."""
    client.save_token("t")
    # discover returns a URL
    client._session.get.return_value = FakeResponse(
        200, {"url": "http://newly-discovered:9000"}
    )
    # direct post succeeds
    client._session.post.return_value = FakeResponse(
        200, {"jsonrpc": "2.0", "result": {"ok": True}}
    )

    out = client.call_peer("new-sib", "hi")

    assert out["result"]["ok"] is True
    assert client._url_cache["new-sib"][0] == "http://newly-discovered:9000"
    called_url = client._session.post.call_args[0][0]
    assert called_url == "http://newly-discovered:9000"


def test_peer_info_dataclass_defaults():
    p = PeerInfo(id="x", name="y", url="http://z")
    assert p.role == ""
    assert p.tier == 2
    assert p.status == "unknown"
    assert p.agent_card == {}


# ---------------------------------------------------------------------------
# Phase 30.3 — install_plugin
# ---------------------------------------------------------------------------

import io
import tarfile

from molecule_agent.client import _safe_extract_tar


def _make_tarball(files: dict[str, bytes]) -> bytes:
    """Build a gzipped tarball in memory from a {name: content} dict."""
    buf = io.BytesIO()
    with tarfile.open(fileobj=buf, mode="w:gz") as tf:
        for name, content in files.items():
            info = tarfile.TarInfo(name=name)
            info.size = len(content)
            info.mode = 0o644
            tf.addfile(info, io.BytesIO(content))
    return buf.getvalue()


class _StreamingResp:
    """requests-shaped response with .content + .iter_content + context-manager.

    install_plugin switched from streaming reads to .content (we hold the
    full <=100MiB tarball in memory before extract — see client.py comment),
    but we keep iter_content available for any future test that wants to
    exercise a streaming path.
    """
    def __init__(self, status: int, body: bytes):
        self.status_code = status
        self._body = body
        self.content = body  # used by .content readers (install_plugin today)
    def __enter__(self): return self
    def __exit__(self, *a): return None
    def raise_for_status(self):
        if self.status_code >= 400:
            import requests
            raise requests.HTTPError(f"HTTP {self.status_code}")
    def iter_content(self, chunk_size=64*1024):
        i = 0
        while i < len(self._body):
            yield self._body[i:i+chunk_size]
            i += chunk_size
            i += chunk_size


def test_install_plugin_unpacks_into_per_workspace_dir(client: RemoteAgentClient, tmp_path):
    client.save_token("t")
    tarball = _make_tarball({
        "plugin.yaml": b"name: hello\nversion: 1.0.0\n",
        "rules.md":    b"some rules\n",
        "skills/x/SKILL.md": b"---\nname: x\n---\n",
    })

    # Stub out the streaming GET (used inside `with`)
    def fake_get(url, headers=None, params=None, stream=False, timeout=None):
        assert "/plugins/hello/download" in url
        assert headers["Authorization"] == "Bearer t"
        return _StreamingResp(200, tarball)
    client._session.get.side_effect = fake_get
    # POST install record — also stubbed
    client._session.post.return_value = FakeResponse(200, {"status": "installed"})

    target = client.install_plugin("hello")

    assert target.exists()
    assert (target / "plugin.yaml").read_bytes() == b"name: hello\nversion: 1.0.0\n"
    assert (target / "skills" / "x" / "SKILL.md").read_text().startswith("---\nname: x\n")
    # Atomic-rename means no .staging-* leftover
    assert not any(p.name.startswith(".staging-") for p in client.plugins_dir.iterdir())
    # Reported the install
    post_url = client._session.post.call_args[0][0]
    assert post_url.endswith(f"/workspaces/{client.workspace_id}/plugins")


def test_install_plugin_passes_source_query_when_given(client: RemoteAgentClient):
    client.save_token("t")
    tarball = _make_tarball({"plugin.yaml": b"name: gh\nversion: 0.1.0\n"})
    captured = {}
    def fake_get(url, headers=None, params=None, stream=False, timeout=None):
        captured["url"] = url
        captured["params"] = params
        return _StreamingResp(200, tarball)
    client._session.get.side_effect = fake_get
    client._session.post.return_value = FakeResponse(200, {})

    client.install_plugin("gh", source="github://acme/my-plugin")
    assert captured["params"] == {"source": "github://acme/my-plugin"}


def test_install_plugin_atomic_rollback_on_corrupt_tarball(client: RemoteAgentClient):
    client.save_token("t")
    # Truncated gzip — tarfile.open will raise
    client._session.get.side_effect = lambda *a, **k: _StreamingResp(200, b"not a gzip")
    client._session.post.return_value = FakeResponse(200, {})

    import pytest as _pytest
    with _pytest.raises(Exception):
        client.install_plugin("broken")
    # No .staging-* dir lingering, no half-installed plugin dir
    assert not list(client.plugins_dir.iterdir()) if client.plugins_dir.exists() else True


def test_install_plugin_overwrites_existing(client: RemoteAgentClient):
    client.save_token("t")
    # Pre-populate an old version
    old_dir = client.plugins_dir / "rotateme"
    old_dir.mkdir(parents=True)
    (old_dir / "old-marker").write_text("old")

    new_tarball = _make_tarball({
        "plugin.yaml": b"name: rotateme\nversion: 2.0.0\n",
        "new-marker": b"new",
    })
    client._session.get.side_effect = lambda *a, **k: _StreamingResp(200, new_tarball)
    client._session.post.return_value = FakeResponse(200, {})

    client.install_plugin("rotateme")
    assert not (client.plugins_dir / "rotateme" / "old-marker").exists()
    assert (client.plugins_dir / "rotateme" / "new-marker").read_text() == "new"


def test_install_plugin_runs_setup_sh_when_present(client: RemoteAgentClient, tmp_path):
    client.save_token("t")
    # setup.sh that drops a sentinel file we can verify
    sentinel = tmp_path / "ran"
    setup_script = f"#!/bin/bash\nset -e\ntouch {sentinel}\n".encode()
    tarball = _make_tarball({
        "plugin.yaml": b"name: withsetup\n",
        "setup.sh":    setup_script,
    })
    client._session.get.side_effect = lambda *a, **k: _StreamingResp(200, tarball)
    client._session.post.return_value = FakeResponse(200, {})

    client.install_plugin("withsetup")

    # setup.sh extracted with 0644 perms (tar default), so script execution
    # depends on bash interpreting the file contents. The bash invocation
    # runs without the +x bit because we call `bash <setup>` not `<setup>`.
    assert sentinel.exists(), "setup.sh did not run"


def test_install_plugin_skips_setup_when_disabled(client: RemoteAgentClient, tmp_path):
    client.save_token("t")
    sentinel = tmp_path / "should-not-exist"
    tarball = _make_tarball({
        "setup.sh": f"#!/bin/bash\ntouch {sentinel}\n".encode(),
    })
    client._session.get.side_effect = lambda *a, **k: _StreamingResp(200, tarball)
    client._session.post.return_value = FakeResponse(200, {})

    client.install_plugin("nosetup", run_setup_sh=False)
    assert not sentinel.exists()


def test_install_plugin_skips_platform_report_when_disabled(client: RemoteAgentClient):
    client.save_token("t")
    tarball = _make_tarball({"plugin.yaml": b"name: silent\n"})
    client._session.get.side_effect = lambda *a, **k: _StreamingResp(200, tarball)

    client.install_plugin("silent", report_to_platform=False)
    # POST never called when report disabled
    client._session.post.assert_not_called()


def test_install_plugin_404_raises_with_useful_url(client: RemoteAgentClient):
    client.save_token("t")
    client._session.get.side_effect = lambda *a, **k: _StreamingResp(404, b"")
    import pytest as _pytest
    with _pytest.raises(Exception):
        client.install_plugin("missing")


# ---------------------------------------------------------------------------
# _safe_extract_tar
# ---------------------------------------------------------------------------

def test_safe_extract_rejects_path_traversal(tmp_path: Path):
    """Tar slip CVE: an entry named '../escape' must be rejected."""
    buf = io.BytesIO()
    with tarfile.open(fileobj=buf, mode="w") as tf:
        info = tarfile.TarInfo(name="../escape.txt")
        data = b"oops"
        info.size = len(data)
        tf.addfile(info, io.BytesIO(data))
    buf.seek(0)
    with tarfile.open(fileobj=buf, mode="r") as tf:
        import pytest as _pytest
        with _pytest.raises(ValueError, match="refusing tar entry escaping"):
            _safe_extract_tar(tf, tmp_path)


def test_safe_extract_rejects_absolute_paths(tmp_path: Path):
    buf = io.BytesIO()
    with tarfile.open(fileobj=buf, mode="w") as tf:
        info = tarfile.TarInfo(name="/etc/passwd")
        data = b"oops"
        info.size = len(data)
        tf.addfile(info, io.BytesIO(data))
    buf.seek(0)
    with tarfile.open(fileobj=buf, mode="r") as tf:
        import pytest as _pytest
        with _pytest.raises(ValueError):
            _safe_extract_tar(tf, tmp_path)


def test_safe_extract_skips_symlinks_silently(tmp_path: Path):
    buf = io.BytesIO()
    with tarfile.open(fileobj=buf, mode="w") as tf:
        sym = tarfile.TarInfo(name="link.lnk")
        sym.type = tarfile.SYMTYPE
        sym.linkname = "/etc/passwd"
        tf.addfile(sym)
        # Plus a normal file alongside
        info = tarfile.TarInfo(name="real.md")
        data = b"ok"
        info.size = len(data)
        tf.addfile(info, io.BytesIO(data))
    buf.seek(0)
    with tarfile.open(fileobj=buf, mode="r") as tf:
        _safe_extract_tar(tf, tmp_path)
    assert (tmp_path / "real.md").exists()
    assert not (tmp_path / "link.lnk").exists()
