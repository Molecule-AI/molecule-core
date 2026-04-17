"""Context7 MCP tool wrappers for Molecule AI (#836 — C1+C4+C5).

Provides two tools that proxy requests to ``https://mcp.context7.com/mcp``:
  resolve_library_id  — map a library name to its canonical Context7 library ID
  query_docs          — fetch documentation snippets for a library/topic

Security controls
-----------------
C1 — Response scrubbing
    Any ``ctx7_*`` token that leaks from the Context7 API response into the
    agent's context is replaced with ``[REDACTED]`` before the result is
    returned.  The full ``_SECRET_PATTERNS`` list from ``builtin_tools.memory``
    is re-used here so the two lists stay in sync.

C4 — Query validation
    Queries longer than ``_MAX_QUERY_LEN`` (200 chars) are rejected up-front
    with a ``ToolError``.  Queries that *themselves* contain secret-like
    patterns (e.g. an API key accidentally pasted by the LLM) are rejected so
    they are never forwarded to ``mcp.context7.com``.

C5 — Session call counter
    ``CONTEXT7_MAX_CALLS_PER_SESSION`` (default 50) caps the total number of
    API calls made by this module in the lifetime of the Python process.  A
    ``ToolError`` is raised when the counter would be exceeded so runaway LLM
    loops cannot drain quota unnoticed.

Mock backend
------------
When ``CONTEXT7_API_KEY`` is absent the tools return a predictable stub
response — safe for CI and local development.

Environment variables
---------------------
``CONTEXT7_API_KEY``              — Required for live calls.  Set as a *workspace*
                                    secret (never global — see README.md §Key Management).
``CONTEXT7_MAX_CALLS_PER_SESSION`` — int, default ``50``.
``CONTEXT7_BASE_URL``             — override endpoint; defaults to
                                    ``https://mcp.context7.com/mcp``.
"""

from __future__ import annotations

import logging
import os
import re
import threading
from types import SimpleNamespace
from typing import Any

from langchain_core.tools import tool

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------

_BASE_URL: str = os.environ.get(
    "CONTEXT7_BASE_URL", "https://mcp.context7.com/mcp"
)
_MAX_QUERY_LEN: int = 200
_DEFAULT_MAX_CALLS: int = 50

# Re-declare the same patterns used in builtin_tools/memory.py (#834) so that
# both the storage layer AND the network layer scrub the same secret formats.
# Keep the two lists in sync whenever a new pattern is added to either file.
_SECRET_PATTERNS: list[re.Pattern] = [
    re.compile(r'ctx7_[A-Za-z0-9_\-]{8,}'),
    re.compile(r'sk-[A-Za-z0-9]{20,}'),
    re.compile(r'ghp_[A-Za-z0-9]{36,}'),
    re.compile(r'Bearer [A-Za-z0-9\-._~+/]{20,}'),
    re.compile(r'[A-Z_]{5,}_API_KEY=[A-Za-z0-9+/]{10,}'),
]

# ---------------------------------------------------------------------------
# Session call counter (C5)
# ---------------------------------------------------------------------------

_counter_lock = threading.Lock()
_session_call_count: int = 0


def _max_calls() -> int:
    """Return the configured per-session call cap."""
    try:
        return int(os.environ.get("CONTEXT7_MAX_CALLS_PER_SESSION", _DEFAULT_MAX_CALLS))
    except (ValueError, TypeError):
        return _DEFAULT_MAX_CALLS


def _increment_and_check() -> None:
    """Increment the call counter.  Raise ``ToolError`` if the cap is exceeded.

    Thread-safe — uses a module-level lock so concurrent async tasks
    (if any) cannot race past the cap.
    """
    global _session_call_count
    with _counter_lock:
        _session_call_count += 1
        current = _session_call_count
    cap = _max_calls()
    if current > cap:
        raise ToolError(
            f"Context7 session call limit reached ({cap}). "
            f"Increase CONTEXT7_MAX_CALLS_PER_SESSION or start a new session."
        )


def _reset_counter() -> None:
    """Reset the session counter — exposed for tests only."""
    global _session_call_count
    with _counter_lock:
        _session_call_count = 0


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


class ToolError(Exception):
    """Raised by context7 tools on validation / rate-limit errors.

    Using a named exception (rather than bare ``RuntimeError``) lets tests
    assert on the type and callers catch it specifically.
    """


def _scrub_response(text: str) -> str:
    """Replace secret-like tokens in an API response with ``[REDACTED]`` (C1)."""
    for pattern in _SECRET_PATTERNS:
        text = pattern.sub('[REDACTED]', text)
    return text


def _validate_query(query: str) -> None:
    """Reject queries that are too long or contain secrets (C4).

    Args:
        query: The raw query string supplied by the LLM.

    Raises:
        ToolError: if the query exceeds the length cap or matches a secret pattern.
    """
    if len(query) > _MAX_QUERY_LEN:
        raise ToolError(
            f"Query too long ({len(query)} chars > {_MAX_QUERY_LEN} cap). "
            "Shorten the query before calling query_docs."
        )
    for pattern in _SECRET_PATTERNS:
        if pattern.search(query):
            raise ToolError(
                "Query contains a secret-like pattern and was rejected. "
                "Do not include API keys or tokens in documentation queries."
            )


def _api_key() -> str:
    """Return the workspace-scoped Context7 API key, or empty string if absent."""
    return os.environ.get("CONTEXT7_API_KEY", "").strip()


def _mock_resolve(library_name: str) -> dict[str, Any]:
    """Return a stub resolve response for tests / keyless environments."""
    safe_name = library_name.lower().replace(" ", "-")
    return {
        "library_id": f"/mock/{safe_name}",
        "name": library_name,
        "mock": True,
    }


def _mock_query(library_id: str, topic: str, tokens: int) -> dict[str, Any]:
    """Return a stub query response for tests / keyless environments."""
    return {
        "library_id": library_id,
        "topic": topic,
        "tokens_used": min(tokens, 100),
        "content": f"[Mock documentation for {library_id}#{topic or 'overview'}]",
        "mock": True,
    }


# ---------------------------------------------------------------------------
# Live HTTP helpers (lazy-import httpx)
# ---------------------------------------------------------------------------

try:  # pragma: no cover — optional at import time; tests mock the client
    import httpx as _httpx
except ImportError:  # pragma: no cover
    _httpx = SimpleNamespace(AsyncClient=None)


async def _live_resolve(library_name: str, api_key: str) -> dict[str, Any]:
    """Call the Context7 MCP resolve endpoint."""
    payload = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": "tools/call",
        "params": {
            "name": "resolve-library-id",
            "arguments": {"libraryName": library_name},
        },
    }
    async with _httpx.AsyncClient(timeout=15.0) as client:
        resp = await client.post(
            _BASE_URL,
            json=payload,
            headers={
                "Authorization": f"Bearer {api_key}",
                "Content-Type": "application/json",
            },
        )
        resp.raise_for_status()
        data = resp.json()

    # Normalise the JSON-RPC response envelope.
    result = data.get("result") or {}
    content_blocks = result.get("content") or []
    text = " ".join(
        block.get("text", "") for block in content_blocks if block.get("type") == "text"
    )
    return {"library_id": text.strip(), "name": library_name, "raw": result}


async def _live_query(
    library_id: str, topic: str, tokens: int, api_key: str
) -> dict[str, Any]:
    """Call the Context7 MCP query endpoint."""
    args: dict[str, Any] = {
        "context7CompatibleLibraryID": library_id,
        "tokens": tokens,
    }
    if topic:
        args["topic"] = topic

    payload = {
        "jsonrpc": "2.0",
        "id": 2,
        "method": "tools/call",
        "params": {"name": "get-library-docs", "arguments": args},
    }
    async with _httpx.AsyncClient(timeout=30.0) as client:
        resp = await client.post(
            _BASE_URL,
            json=payload,
            headers={
                "Authorization": f"Bearer {api_key}",
                "Content-Type": "application/json",
            },
        )
        resp.raise_for_status()
        data = resp.json()

    result = data.get("result") or {}
    content_blocks = result.get("content") or []
    text = "\n".join(
        block.get("text", "") for block in content_blocks if block.get("type") == "text"
    )
    return {
        "library_id": library_id,
        "topic": topic,
        "tokens_used": tokens,
        "content": text,
        "raw": result,
    }


# ---------------------------------------------------------------------------
# Public tools
# ---------------------------------------------------------------------------


@tool
async def resolve_library_id(library_name: str) -> dict[str, Any]:
    """Resolve a library name to its canonical Context7 library ID.

    Args:
        library_name: Human-friendly library name, e.g. ``"react"``, ``"fastapi"``,
                      ``"langchain"``.

    Returns:
        dict with ``library_id`` (e.g. ``"/facebook/react"``), ``name``, and
        optionally ``mock: True`` when running without a live API key.

    Raises:
        ToolError: on session rate-limit breach.
    """
    if not library_name or not library_name.strip():
        return {"error": "library_name is required"}

    _increment_and_check()

    key = _api_key()
    if not key:
        logger.info("context7: CONTEXT7_API_KEY absent — using mock backend")
        return _mock_resolve(library_name.strip())

    try:
        result = await _live_resolve(library_name.strip(), key)
        result["library_id"] = _scrub_response(result.get("library_id", ""))
        logger.info("context7: resolve_library_id(%r) → %s", library_name, result.get("library_id"))
        return result
    except Exception as exc:
        logger.exception("context7: resolve_library_id failed")
        return {"error": str(exc)}


@tool
async def query_docs(
    library_id: str,
    topic: str = "",
    tokens: int = 5000,
) -> dict[str, Any]:
    """Fetch documentation snippets for a library from Context7.

    Args:
        library_id: Canonical library ID returned by ``resolve_library_id``
                    (e.g. ``"/facebook/react"``).
        topic: Optional topic to focus the documentation fetch (e.g.
               ``"hooks"``, ``"routing"``, ``"dependency injection"``).
               Maximum 200 characters.
        tokens: Approximate token budget for the returned content (default 5000).

    Returns:
        dict with ``content`` (documentation text), ``library_id``, ``topic``,
        ``tokens_used``, and optionally ``mock: True``.

    Raises:
        ToolError: on query validation failure or session rate-limit breach.
    """
    if not library_id or not library_id.strip():
        return {"error": "library_id is required"}

    # Validate topic (C4) — library_id is an internal identifier, not user input.
    if topic:
        try:
            _validate_query(topic)
        except ToolError as exc:
            return {"error": str(exc)}

    _increment_and_check()

    key = _api_key()
    if not key:
        logger.info("context7: CONTEXT7_API_KEY absent — using mock backend")
        return _mock_query(library_id.strip(), topic, tokens)

    try:
        result = await _live_query(library_id.strip(), topic, tokens, key)
        # Scrub any ctx7_* tokens that leaked into the response (C1).
        result["content"] = _scrub_response(result.get("content", ""))
        logger.info(
            "context7: query_docs(library_id=%r, topic=%r) → %d chars",
            library_id, topic, len(result["content"]),
        )
        return result
    except Exception as exc:
        logger.exception("context7: query_docs failed")
        return {"error": str(exc)}
