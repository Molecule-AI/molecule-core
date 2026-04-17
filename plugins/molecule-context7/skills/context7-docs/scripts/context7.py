"""Context7 MCP tool wrappers for Molecule AI (#836 — C1+C4+C5).

Provides two tools that proxy requests to ``https://mcp.context7.com/mcp``:
  resolve_library_id  — map a library name to its canonical Context7 library ID
  query_docs          — fetch documentation snippets for a library/topic

Security controls
-----------------
C1 — Response scrubbing (two layers, applied in order)
    Layer 1 — *injection scrub*: HTML-injection tags (``<script>``, ``<iframe>``,
    ``<object>``, ``<embed>``, ``<form>``, ``<input>``) and prompt-injection role
    markers (lines starting with ``SYSTEM:``, ``HUMAN:``, ``ASSISTANT:``,
    ``[INST]``, ``<|im_start|>``) are removed from API responses before the
    result is returned to the agent.  Replaced with
    ``[content removed by security wrapper]``.

    Layer 2 — *secret scrub*: Any ``ctx7_*`` token or other known-secret
    pattern that leaks from the Context7 API response is replaced with
    ``[REDACTED]``.  The ``_SECRET_PATTERNS`` list mirrors
    ``builtin_tools/security.py`` so both the storage layer and the network
    layer cover the same formats.

C4 — Query length cap
    Topics longer than ``_MAX_QUERY_LEN`` (500 chars) are truncated before
    being forwarded to context7.com.  A ``WARNING`` log line is emitted when
    truncation occurs.  Queries that *themselves* contain secret-like patterns
    (e.g. an API key accidentally pasted by the LLM) are rejected with a
    ``ToolError`` so secrets never reach the external API.

C5 — Per-workspace session call limit
    A per-workspace call counter (keyed on ``WORKSPACE_ID``) caps total
    context7 API calls at ``CONTEXT7_MAX_CALLS_PER_SESSION`` (default 20)
    for the lifetime of the Python process.  After the cap is reached the
    tool returns an error: restart the workspace container to reset the
    counter.

Mock backend
------------
When ``CONTEXT7_API_KEY`` is absent the tools return a predictable stub
response — safe for CI and local development.

Environment variables
---------------------
``CONTEXT7_API_KEY``               — Required for live calls.  Set as a *workspace*
                                     secret (never global — see README.md §Key Management).
``CONTEXT7_MAX_CALLS_PER_SESSION`` — int, default ``20``.
``CONTEXT7_BASE_URL``              — override endpoint; defaults to
                                     ``https://mcp.context7.com/mcp``.
``WORKSPACE_ID``                   — injected by the platform; used to key
                                     the per-workspace call counter.
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

# C4: maximum topic length before truncation.
_MAX_QUERY_LEN: int = 500

# C5: default per-workspace call cap.
_DEFAULT_MAX_CALLS: int = 20

# Replacement sentinel for removed injection content (C1 layer 1).
_REMOVED_MARKER: str = "[content removed by security wrapper]"

# ---------------------------------------------------------------------------
# C1 — Injection scrubbing patterns (layer 1: HTML + prompt-injection)
# ---------------------------------------------------------------------------

# <script> blocks — match across newlines so multi-line scripts are caught.
_SCRIPT_RE: re.Pattern = re.compile(
    r"<script[^>]*>.*?</script>", re.IGNORECASE | re.DOTALL
)

# Dangerous HTML tags whose opening tag alone is sufficient to warrant removal.
# We remove the opening tag; the closing tag (e.g. </iframe>) is harmless text.
_DANGEROUS_TAG_RE: re.Pattern = re.compile(
    r"<(?:iframe|object|embed|form|input)(?:\s[^>]*)?>",
    re.IGNORECASE,
)

# Prompt-injection role markers: lines that start with these tokens could
# trick the agent into treating external doc content as system instructions.
#
# Two sub-patterns:
#   1. Role keywords (SYSTEM/HUMAN/ASSISTANT/[INST]) must be followed by a
#      space or colon separator before their content.
#   2. ChatML <|im_start|> is immediately followed by the role name with no
#      separator, so it only requires one or more subsequent characters.
_PROMPT_INJECTION_RE: re.Pattern = re.compile(
    r"^(?:(?:SYSTEM|HUMAN|ASSISTANT|\[INST\])[ :].+|<\|im_start\|>.+)$",
    re.MULTILINE | re.IGNORECASE,
)

# ---------------------------------------------------------------------------
# C1 — Secret scrubbing patterns (layer 2: credential tokens)
#
# Re-declares the same patterns from builtin_tools/security.py (#834) so that
# both the storage layer AND the network layer scrub the same formats.
# Keep in sync when adding new patterns to either file.
# ---------------------------------------------------------------------------

_SECRET_PATTERNS: list[re.Pattern] = [
    re.compile(r"ctx7_[A-Za-z0-9_\-]{8,}"),
    re.compile(r"sk-[A-Za-z0-9]{20,}"),
    re.compile(r"ghp_[A-Za-z0-9]{36,}"),
    re.compile(r"Bearer [A-Za-z0-9\-._~+/]{20,}"),
    re.compile(r"[A-Z_]{5,}_API_KEY=[A-Za-z0-9+/]{10,}"),
]

# ---------------------------------------------------------------------------
# C5 — Per-workspace session call counter
# ---------------------------------------------------------------------------

_counter_lock = threading.Lock()
# Dict keyed on workspace ID (WORKSPACE_ID env var, or "default" in tests).
_session_counters: dict[str, int] = {}


def _workspace_key() -> str:
    """Return the counter key for the current workspace."""
    return os.environ.get("WORKSPACE_ID", "default")


def _max_calls() -> int:
    """Return the configured per-session call cap."""
    try:
        return int(
            os.environ.get("CONTEXT7_MAX_CALLS_PER_SESSION", _DEFAULT_MAX_CALLS)
        )
    except (ValueError, TypeError):
        return _DEFAULT_MAX_CALLS


def _increment_and_check() -> None:
    """Increment the per-workspace counter and raise if the cap is exceeded (C5).

    Thread-safe — uses a module-level lock so concurrent async tasks cannot
    race past the cap.

    Raises:
        ToolError: when the call count for this workspace exceeds the cap.
    """
    key = _workspace_key()
    with _counter_lock:
        _session_counters[key] = _session_counters.get(key, 0) + 1
        current = _session_counters[key]
    cap = _max_calls()
    if current > cap:
        raise ToolError(
            f"context7 session call limit reached ({cap}/session)"
            " \u2014 restart workspace to reset"
        )


def _reset_counter(workspace_key: str | None = None) -> None:
    """Reset session counter(s) — exposed for tests only.

    Args:
        workspace_key: If given, reset only that workspace's counter.
                       If ``None``, reset all counters.
    """
    with _counter_lock:
        if workspace_key is not None:
            _session_counters.pop(workspace_key, None)
        else:
            _session_counters.clear()


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


class ToolError(Exception):
    """Raised by context7 tools on validation / rate-limit errors.

    Using a named exception (rather than bare ``RuntimeError``) lets tests
    assert on the type and callers catch it specifically.
    """


def _scrub_injection(text: str) -> str:
    """Strip HTML-injection and prompt-injection markers from *text* (C1 layer 1).

    Patterns removed (replaced with ``[content removed by security wrapper]``):
    - ``<script>`` blocks (including content, DOTALL)
    - Opening tags for: ``<iframe>``, ``<object>``, ``<embed>``, ``<form>``,
      ``<input>``
    - Lines beginning with ``SYSTEM:``, ``HUMAN:``, ``ASSISTANT:``,
      ``[INST]``, ``<|im_start|>``

    Args:
        text: Raw string from the Context7 API response.

    Returns:
        Sanitised copy of *text*.  If nothing matched, the original string is
        returned unchanged.
    """
    text = _SCRIPT_RE.sub(_REMOVED_MARKER, text)
    text = _DANGEROUS_TAG_RE.sub(_REMOVED_MARKER, text)
    text = _PROMPT_INJECTION_RE.sub(_REMOVED_MARKER, text)
    return text


def _scrub_response(text: str) -> str:
    """Replace secret-like tokens in an API response with ``[REDACTED]`` (C1 layer 2)."""
    for pattern in _SECRET_PATTERNS:
        text = pattern.sub("[REDACTED]", text)
    return text


def _sanitize_result(text: str) -> str:
    """Apply the full two-layer C1 sanitisation pipeline to a tool result.

    Order: injection scrub first (removes structure), then secret scrub
    (replaces credential tokens in what remains).
    """
    return _scrub_response(_scrub_injection(text))


def _cap_query(query: str) -> str:
    """Truncate *query* to ``_MAX_QUERY_LEN`` chars if necessary (C4).

    Logs a WARNING when truncation occurs so operators can see that a query
    was shortened without surfacing an error to the agent.

    Args:
        query: The raw topic string supplied by the LLM.

    Returns:
        ``query`` unchanged if ``len(query) <= _MAX_QUERY_LEN``, otherwise
        ``query[:_MAX_QUERY_LEN]``.
    """
    if len(query) > _MAX_QUERY_LEN:
        logger.warning(
            "context7: query truncated from %d to %d chars (C4 query cap)",
            len(query),
            _MAX_QUERY_LEN,
        )
        return query[:_MAX_QUERY_LEN]
    return query


def _validate_query(query: str) -> None:
    """Reject queries that contain secret-like patterns (C4 secret guard).

    Length enforcement is handled separately by ``_cap_query`` (truncation).
    This function only inspects the *content* of the query.

    Args:
        query: The (possibly already truncated) query string.

    Raises:
        ToolError: if the query matches a known secret pattern.
    """
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
        result["library_id"] = _sanitize_result(result.get("library_id", ""))
        logger.info(
            "context7: resolve_library_id(%r) → %s",
            library_name,
            result.get("library_id"),
        )
        return result
    except ToolError:
        raise
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
               Truncated to 500 characters if longer (C4).
        tokens: Approximate token budget for the returned content (default 5000).

    Returns:
        dict with ``content`` (documentation text), ``library_id``, ``topic``,
        ``tokens_used``, and optionally ``mock: True``.

    Raises:
        ToolError: on query secret-check failure or session rate-limit breach.
    """
    if not library_id or not library_id.strip():
        return {"error": "library_id is required"}

    # C4 — apply length cap then secret check (library_id is internal, not capped).
    if topic:
        topic = _cap_query(topic)
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
        # C1 — apply full two-layer sanitisation to the response content.
        result["content"] = _sanitize_result(result.get("content", ""))
        logger.info(
            "context7: query_docs(library_id=%r, topic=%r) → %d chars",
            library_id,
            topic,
            len(result["content"]),
        )
        return result
    except ToolError:
        raise
    except Exception as exc:
        logger.exception("context7: query_docs failed")
        return {"error": str(exc)}
