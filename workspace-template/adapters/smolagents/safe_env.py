"""Denylist-based environment sanitization for smolagents (issue #826 — C3 CRITICAL).

This module provides a simple denylist approach: well-known secret variable
names plus ``*_API_KEY`` and ``*_TOKEN`` suffix patterns are stripped before
env is passed to agent-executed code.

For a stricter allowlist-based alternative that only passes explicitly-safe
variables through, see :mod:`adapters.smolagents.env_sanitize`.

Usage::

    from adapters.smolagents.safe_env import make_safe_env

    executor = LocalPythonExecutor(...)
    # Pass only the sanitised env to the subprocess / exec context:
    safe = make_safe_env()
"""

import copy
import os

# Named API keys and tokens known to be used by smolagents / LLM clients.
# These are removed regardless of the suffix-pattern below.
SMOLAGENTS_ENV_DENYLIST: frozenset = frozenset(
    {
        "OPENAI_API_KEY",
        "ANTHROPIC_API_KEY",
        "GROQ_API_KEY",
        "CEREBRAS_API_KEY",
        "QIANFAN_API_KEY",
        "LANGFUSE_SECRET_KEY",
        "LANGFUSE_PUBLIC_KEY",
        "HF_TOKEN",
    }
)


def make_safe_env() -> dict:
    """Return a sanitised copy of ``os.environ`` with secrets removed.

    Removes any key that:
    - Is in :data:`SMOLAGENTS_ENV_DENYLIST`, OR
    - Ends with ``_API_KEY``, OR
    - Ends with ``_TOKEN``

    ``os.environ`` is **never mutated** — a fresh ``dict`` copy is returned.

    Returns
    -------
    dict
        A copy of the current environment with secret keys removed.
    """
    env = copy.copy(dict(os.environ))
    for key in list(env.keys()):
        if (
            key in SMOLAGENTS_ENV_DENYLIST
            or key.endswith("_API_KEY")
            or key.endswith("_TOKEN")
        ):
            del env[key]
    return env
