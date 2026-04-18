"""Tests for workspace/platform_auth.py (Phase 30.1)."""
from __future__ import annotations

import os
import stat
from pathlib import Path

import pytest

import platform_auth


@pytest.fixture(autouse=True)
def _isolate(tmp_path, monkeypatch):
    """Each test gets its own CONFIGS_DIR and a fresh in-process cache."""
    monkeypatch.setenv("CONFIGS_DIR", str(tmp_path))
    platform_auth.clear_cache()
    yield
    platform_auth.clear_cache()


def test_get_token_returns_none_when_file_absent(tmp_path):
    assert platform_auth.get_token() is None


def test_save_and_get_roundtrip(tmp_path):
    platform_auth.save_token("secret-abc123")
    assert platform_auth.get_token() == "secret-abc123"
    # File contents match exactly, no trailing newline
    assert (tmp_path / ".auth_token").read_text() == "secret-abc123"


def test_saved_file_is_0600(tmp_path):
    platform_auth.save_token("very-secret")
    mode = stat.S_IMODE((tmp_path / ".auth_token").stat().st_mode)
    assert mode == 0o600, f"expected 0600 mode, got 0o{mode:o}"


def test_save_token_strips_whitespace(tmp_path):
    platform_auth.save_token("  padded-token  \n")
    assert platform_auth.get_token() == "padded-token"


def test_save_token_rejects_empty():
    with pytest.raises(ValueError):
        platform_auth.save_token("")
    with pytest.raises(ValueError):
        platform_auth.save_token("   \n")


def test_save_token_idempotent(tmp_path):
    """Saving the same token twice must not change the file's mtime."""
    platform_auth.save_token("stable-token")
    path = tmp_path / ".auth_token"
    first_mtime = path.stat().st_mtime_ns
    # Force cache path to fire; save_token should no-op
    platform_auth.clear_cache()
    platform_auth.save_token("stable-token")
    assert path.stat().st_mtime_ns == first_mtime


def test_save_token_rotation_overwrites(tmp_path):
    platform_auth.save_token("token-v1")
    platform_auth.save_token("token-v2")
    assert platform_auth.get_token() == "token-v2"


def test_auth_headers_when_no_token_is_empty():
    assert platform_auth.auth_headers() == {}


def test_auth_headers_format():
    platform_auth.save_token("hello-world")
    assert platform_auth.auth_headers() == {"Authorization": "Bearer hello-world"}


def test_get_token_caches_after_first_disk_read(tmp_path, monkeypatch):
    path = tmp_path / ".auth_token"
    path.write_text("disk-token")

    # First call populates the cache
    assert platform_auth.get_token() == "disk-token"

    # Now mutate the file behind the cache's back.
    path.write_text("ignored-by-cache")
    # Subsequent calls return the cached value, NOT the new disk content.
    assert platform_auth.get_token() == "disk-token"

    # clear_cache() forces a re-read
    platform_auth.clear_cache()
    assert platform_auth.get_token() == "ignored-by-cache"


def test_get_token_handles_empty_file(tmp_path):
    (tmp_path / ".auth_token").write_text("")
    assert platform_auth.get_token() is None


def test_get_token_handles_whitespace_only_file(tmp_path):
    (tmp_path / ".auth_token").write_text("   \n\n   ")
    assert platform_auth.get_token() is None


def test_configs_dir_respected(tmp_path, monkeypatch):
    alt = tmp_path / "alt-configs"
    alt.mkdir()
    monkeypatch.setenv("CONFIGS_DIR", str(alt))
    platform_auth.clear_cache()
    platform_auth.save_token("where-does-it-land")
    assert (alt / ".auth_token").exists()
    assert not (tmp_path / ".auth_token").exists()


def test_default_configs_dir_fallback(tmp_path, monkeypatch):
    monkeypatch.delenv("CONFIGS_DIR", raising=False)
    # Can't actually write to /configs on a dev laptop, so just verify the
    # path resolution points there. Save will fail gracefully via mkdir+exist_ok.
    platform_auth.clear_cache()
    # We expect _token_file() to resolve under /configs when env is unset.
    path = platform_auth._token_file()
    assert str(path).startswith("/configs")


# ---------------------------------------------------------------------------
# MOLECULE_AUTH_TOKEN env-var bootstrap (EC2 / CP provisioner boot path)
# HIGH #6 fix: CP provisioner injects token in env before launching the
# EC2 instance so the agent can heartbeat without going through registration.
# ---------------------------------------------------------------------------

def test_env_var_used_when_file_absent(tmp_path, monkeypatch):
    """MOLECULE_AUTH_TOKEN is returned when no .auth_token file exists."""
    monkeypatch.setenv("CONFIGS_DIR", str(tmp_path))
    monkeypatch.setenv("MOLECULE_AUTH_TOKEN", "env-boot-token-abc")
    platform_auth.clear_cache()
    assert platform_auth.get_token() == "env-boot-token-abc"


def test_env_var_persisted_to_file(tmp_path, monkeypatch):
    """Token read from env var is immediately persisted to .auth_token file
    so it survives process restarts that don't inherit the env."""
    monkeypatch.setenv("CONFIGS_DIR", str(tmp_path))
    monkeypatch.setenv("MOLECULE_AUTH_TOKEN", "env-boot-token-persisted")
    platform_auth.clear_cache()
    platform_auth.get_token()
    token_file = tmp_path / ".auth_token"
    assert token_file.exists(), ".auth_token must be created on first env-var read"
    assert token_file.read_text() == "env-boot-token-persisted"


def test_file_token_takes_priority_over_env_var(tmp_path, monkeypatch):
    """If a .auth_token file already exists, it takes priority over the env var."""
    token_file = tmp_path / ".auth_token"
    token_file.write_text("file-token-wins")
    monkeypatch.setenv("CONFIGS_DIR", str(tmp_path))
    monkeypatch.setenv("MOLECULE_AUTH_TOKEN", "env-token-loses")
    platform_auth.clear_cache()
    assert platform_auth.get_token() == "file-token-wins"


def test_env_var_absent_returns_none_when_no_file(tmp_path, monkeypatch):
    """Neither file nor env var → get_token() returns None."""
    monkeypatch.setenv("CONFIGS_DIR", str(tmp_path))
    monkeypatch.delenv("MOLECULE_AUTH_TOKEN", raising=False)
    platform_auth.clear_cache()
    assert platform_auth.get_token() is None


def test_env_var_empty_string_treated_as_absent(tmp_path, monkeypatch):
    """An empty MOLECULE_AUTH_TOKEN env var must NOT be used as a token."""
    monkeypatch.setenv("CONFIGS_DIR", str(tmp_path))
    monkeypatch.setenv("MOLECULE_AUTH_TOKEN", "")
    platform_auth.clear_cache()
    assert platform_auth.get_token() is None
