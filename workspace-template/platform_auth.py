"""Workspace auth-token store (Phase 30.1).

Single source of truth for this workspace's authentication token. The
token is issued by the platform on the first successful
``POST /registry/register`` call and travels with every subsequent
heartbeat / update-card / (later) secrets-pull / A2A request.

The token is persisted to ``<configs>/.auth_token`` so it survives
restarts — we only expect to receive it once from the platform, since
``/registry/register`` no-ops token issuance for workspaces that already
have one on file.

Storage:
    ${CONFIGS_DIR}/.auth_token        # 0600, one line, no trailing newline

Callers interact with three functions:
    :func:`get_token`   — returns the cached token or None
    :func:`save_token`  — persists a freshly-issued token
    :func:`auth_headers`— builds the Authorization header dict for httpx
"""
from __future__ import annotations

import logging
import os
from pathlib import Path

logger = logging.getLogger(__name__)

# In-process cache so we don't hit disk on every heartbeat. The heartbeat
# loop fires on a short interval and reading a tiny file 10x per minute
# is wasteful. The file is the durable copy; this var is the hot path.
_cached_token: str | None = None


def _token_file() -> Path:
    """Path to the on-disk token file. Respects CONFIGS_DIR, falls back
    to /configs for the default container layout."""
    return Path(os.environ.get("CONFIGS_DIR", "/configs")) / ".auth_token"


def get_token() -> str | None:
    """Return the cached token, reading it from disk on first call."""
    global _cached_token
    if _cached_token is not None:
        return _cached_token
    path = _token_file()
    if not path.exists():
        return None
    try:
        tok = path.read_text().strip()
    except OSError as exc:
        logger.warning("platform_auth: failed to read %s: %s", path, exc)
        return None
    if not tok:
        return None
    _cached_token = tok
    return tok


def save_token(token: str) -> None:
    """Persist a newly-issued token. Creates the file with 0600 mode.

    Idempotent — if an identical token is already on disk we skip the
    write so we don't churn the file's mtime or trigger spurious
    filesystem watchers."""
    global _cached_token
    token = token.strip()
    if not token:
        raise ValueError("platform_auth: refusing to save empty token")
    if get_token() == token:
        return
    path = _token_file()
    path.parent.mkdir(parents=True, exist_ok=True)
    # Write + chmod before assigning — if the chmod fails we don't want
    # a world-readable copy of the token sitting around.
    path.write_text(token)
    try:
        os.chmod(path, 0o600)
    except OSError as exc:
        logger.warning("platform_auth: chmod 0600 on %s failed: %s", path, exc)
    _cached_token = token


def auth_headers() -> dict[str, str]:
    """Return a header dict to merge into httpx calls. Empty if no token
    is available yet — callers send the request as-is and the platform's
    heartbeat handler grandfathers pre-token workspaces through until
    their next /registry/register issues one."""
    tok = get_token()
    if not tok:
        return {}
    return {"Authorization": f"Bearer {tok}"}


def clear_cache() -> None:
    """Reset the in-memory cache. Used by tests that write fresh token
    files between cases."""
    global _cached_token
    _cached_token = None
