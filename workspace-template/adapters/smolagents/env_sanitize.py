"""Allowlist-based environment sanitization for smolagents (#826 — C3 CRITICAL).

Security model
--------------
We use an **allowlist** (not a denylist) — only variables explicitly
enumerated as safe are passed through to agent-executed code.  Any key not
on the list is silently dropped.

This is intentionally strict: adding a new safe variable is a deliberate
engineering act that surfaces in code review, rather than hoping a regex
denylist catches every new secret name.

Thread safety
-------------
``SafeLocalPythonExecutor.__call__`` mutates ``os.environ`` temporarily.
``_ENV_PATCH_LOCK`` serialises concurrent calls so simultaneous executions
do not see each other's env patches.

Extending the allowlist
-----------------------
Set ``SMOLAGENTS_ENV_EXTRA_ALLOWLIST`` to a comma-separated list of
additional uppercase env var names that should be passed through.  This is
intended for workspace-specific non-secret variables (e.g. ``WORKSPACE_ID``
that you know are safe):

    SMOLAGENTS_ENV_EXTRA_ALLOWLIST="MY_COMPANY_ENV,REGION"

Never add secret names here — use workspace secrets injection instead.
"""

from __future__ import annotations

import os
import threading
from typing import Any, Dict, List, Optional

# ---------------------------------------------------------------------------
# Allowlist configuration
# ---------------------------------------------------------------------------

# Core safe env variables — non-secret system and runtime variables that
# agent code may legitimately need (e.g. PATH for subprocess-free tools,
# PYTHONPATH for module resolution, TZ for datetime ops).
_SAFE_ENV_ALLOWLIST: frozenset = frozenset(
    [
        # Shell / system fundamentals
        "PATH",
        "HOME",
        "USER",
        "LOGNAME",
        "SHELL",
        "TERM",
        "TZ",
        "TMPDIR",
        "TEMP",
        "TMP",
        # Language / locale
        "LANG",
        "LANGUAGE",
        "LC_ALL",
        "LC_CTYPE",
        "LC_MESSAGES",
        "LC_NUMERIC",
        "LC_TIME",
        # Python runtime
        "PYTHONPATH",
        "PYTHONHOME",
        "PYTHONDONTWRITEBYTECODE",
        "PYTHONUNBUFFERED",
        "PYTHONIOENCODING",
        # Molecule workspace non-secret identity vars
        "WORKSPACE_ID",
        "WORKSPACE_NAME",
        "PLATFORM_URL",
    ]
)

# Imports permanently excluded from the executor's authorized list.
# These are well-known sandbox-escape vectors.
_BANNED_IMPORTS: frozenset = frozenset(
    ["subprocess", "socket", "ctypes", "importlib", "importlib.util"]
)

# Baseline imports every SafeLocalPythonExecutor allows — pure-computation
# modules with no I/O escape surface.
_BASELINE_SAFE_IMPORTS: List[str] = [
    "math",
    "json",
    "re",
    "datetime",
    "collections",
    "itertools",
    "functools",
    "typing",
    "string",
    "textwrap",
    "decimal",
    "fractions",
    "statistics",
    "random",
    "hashlib",
    "base64",
    "urllib.parse",
    "copy",
    "dataclasses",
    "enum",
    "abc",
    "io",
]

# Thread lock for env patching
_ENV_PATCH_LOCK = threading.Lock()


# ---------------------------------------------------------------------------
# Public API
# ---------------------------------------------------------------------------


def make_safe_env(
    extra_allowed: Optional[List[str]] = None,
) -> Dict[str, str]:
    """Return a *copy* of the environment containing only allowlisted keys.

    ``os.environ`` is **never mutated** by this function.

    Parameters
    ----------
    extra_allowed:
        Additional variable names to include beyond the built-in allowlist.
        Also merged with the ``SMOLAGENTS_ENV_EXTRA_ALLOWLIST`` env var.

    Returns
    -------
    dict
        A copy of ``os.environ`` filtered to allowlisted keys only.
        Keys not on the list are silently dropped.
    """
    allowed = set(_SAFE_ENV_ALLOWLIST)

    # Merge caller-provided extras
    if extra_allowed:
        allowed.update(k.upper() for k in extra_allowed)

    # Merge env-var-configured extras
    env_extra = os.environ.get("SMOLAGENTS_ENV_EXTRA_ALLOWLIST", "")
    if env_extra:
        for key in env_extra.split(","):
            key = key.strip().upper()
            if key:
                allowed.add(key)

    return {k: v for k, v in os.environ.items() if k in allowed}


class SafeLocalPythonExecutor:
    """Allowlist-gated wrapper around smolagents ``LocalPythonExecutor``.

    Guarantees that agent-generated code cannot read secret environment
    variables (``ANTHROPIC_API_KEY``, ``GH_TOKEN``, ``DATABASE_URL``, etc.)
    because they are absent from ``os.environ`` during execution.

    Parameters
    ----------
    additional_imports:
        Extra module names to allow beyond ``_BASELINE_SAFE_IMPORTS``.
        ``_BANNED_IMPORTS`` takes precedence — listed names are silently
        removed.
    extra_allowed_env:
        Extra variable names to pass through beyond the core allowlist.
    _inner:
        Inject a mock ``LocalPythonExecutor`` for tests.  When ``None``,
        the real smolagents executor is constructed lazily.
    """

    def __init__(
        self,
        additional_imports: Optional[List[str]] = None,
        extra_allowed_env: Optional[List[str]] = None,
        *,
        _inner: Any = None,
    ) -> None:
        # Compute final import list (baseline + extras − banned)
        combined = list(_BASELINE_SAFE_IMPORTS)
        if additional_imports:
            for imp in additional_imports:
                if imp not in _BANNED_IMPORTS:
                    combined.append(imp)

        self._authorized_imports: List[str] = combined
        self._extra_allowed_env: Optional[List[str]] = extra_allowed_env
        self._inner = _inner  # may be None until first call

    def _get_inner(self) -> Any:
        """Lazy-construct the real executor on first use (avoids import errors in tests)."""
        if self._inner is None:
            from smolagents import LocalPythonExecutor  # type: ignore[import]

            self._inner = LocalPythonExecutor(
                additional_authorized_imports=self._authorized_imports
            )
        return self._inner

    def __call__(self, code: str, *args: Any, **kwargs: Any) -> Any:
        """Execute ``code`` with only allowlisted env vars visible.

        All keys not on the allowlist are removed from ``os.environ`` for
        the duration of execution and restored afterward, even on exception.
        The lock ensures thread safety across concurrent calls.
        """
        safe_env = make_safe_env(self._extra_allowed_env)
        inner = self._get_inner()

        with _ENV_PATCH_LOCK:
            # Snapshot full current env
            original_env = dict(os.environ)
            # Remove everything not in the safe set
            keys_to_remove = [k for k in os.environ if k not in safe_env]
            for k in keys_to_remove:
                del os.environ[k]
            try:
                return inner(code, *args, **kwargs)
            finally:
                # Always restore
                os.environ.clear()
                os.environ.update(original_env)
