"""Safe send_message wrapper for smolagents (issue #827 — C1 HIGH).

Prevents social-engineering attacks where agent-generated content could
impersonate platform messages, inject HTML, or flood the user chat.

Guarantees
----------
1. Every message is prefixed with ``[smolagents]`` so recipients can
   attribute it to the agent and cannot be mistaken for platform UI.
2. Truncated to 2000 characters to prevent log/UI floods.
3. HTML entities (``<``, ``>``, ``&``, ``"``, ``'``) are escaped so
   rendered UIs that interpret HTML cannot be injected into.

Usage::

    from adapters.smolagents.send_message_wrapper import safe_send_message

    safe_send_message("Hello world", send_fn=platform_client.send)
"""

from __future__ import annotations

import html
import logging

logger = logging.getLogger(__name__)

# Maximum character length for the *user-visible* portion of the message
# (label prefix does not count toward this cap).
_MAX_TEXT_LEN: int = 2000

# Label prepended to every outbound message.
_LABEL: str = "[smolagents]"


def safe_send_message(text: str, send_fn) -> None:
    """Sanitise *text* and deliver it via *send_fn*.

    Parameters
    ----------
    text:
        The raw message text produced by the agent.
    send_fn:
        Callable that delivers the message (e.g. ``platform_client.send``
        or a WebSocket broadcast function). Called with the final,
        sanitised string as its sole positional argument.

    Side effects
    ------------
    - Logs a warning when truncation occurs.
    - Logs a debug entry with the final payload length.
    """
    if not isinstance(text, str):
        text = str(text)

    # Strip HTML entities to prevent injection into rendered UIs.
    sanitised = html.escape(text, quote=True)

    # Truncate to cap (before adding label so cap applies to content).
    if len(sanitised) > _MAX_TEXT_LEN:
        logger.warning(
            "safe_send_message: truncating message from %d to %d chars",
            len(sanitised),
            _MAX_TEXT_LEN,
        )
        sanitised = sanitised[:_MAX_TEXT_LEN]

    payload = f"{_LABEL} {sanitised}"

    logger.debug("safe_send_message: delivering %d-char payload", len(payload))
    send_fn(payload)
