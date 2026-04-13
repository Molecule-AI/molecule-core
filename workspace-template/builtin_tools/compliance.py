"""OWASP Top 10 for Agentic Applications compliance enforcement (Dec 2025).

Enable via config.yaml::

    compliance:
      mode: owasp_agentic
      prompt_injection: detect   # detect | block
      max_tool_calls_per_task: 50
      max_task_duration_seconds: 300

When ``mode`` is absent or empty, this module is a no-op — no overhead, no
behaviour change.  This makes it safe to import unconditionally.

Coverage
--------

OA-01 Prompt Injection (``sanitize_input``)
  Scans user-supplied text for instruction-override patterns, role-hijacking
  attempts, system-prompt delimiter injection, and known jailbreak keywords.

  - ``detect`` (default): log an audit event, return the original text so
    the agent still processes the input.  Operators are alerted without
    breaking legitimate use-cases that happen to contain trigger words.

  - ``block``: raise ``PromptInjectionError`` before the agent sees the text.

OA-03 Excessive Agency (``check_agency_limits``)
  Tracks the number of tool calls and wall-clock time elapsed per task.
  When a limit is exceeded, ``ExcessiveAgencyError`` is raised.  The caller
  (``a2a_executor.py``) catches it and terminates the task gracefully.

OA-02 / OA-06 Insecure Output / Sensitive Data Exposure (``redact_pii``)
  Scans agent output for credit-card numbers, SSNs, API keys, AWS access
  keys, and e-mail addresses.  Detected values are replaced with
  ``[REDACTED:<type>]`` tokens before the response reaches the caller.
  An audit event records the PII types found (not the values themselves).

  Note on streaming: ``redact_pii`` is applied to the *final accumulated
  text* before the terminal ``Message`` event is emitted.  Token-by-token
  SSE artifacts that have already been sent to streaming clients are not
  retroactively redacted.  For full streaming redaction, integrate
  ``redact_pii`` at the ``TaskArtifactUpdateEvent`` level.

Compliance posture report (``get_compliance_posture``)
  Returns the current effective compliance configuration as a plain ``dict``
  suitable for a health or audit endpoint, letting operators verify that the
  correct settings are active without reading config files.
"""

from __future__ import annotations

import logging
import re
import time
import uuid
from dataclasses import dataclass, field
from typing import Any

from builtin_tools.audit import log_event

logger = logging.getLogger(__name__)


# ---------------------------------------------------------------------------
# Public exceptions
# ---------------------------------------------------------------------------


class PromptInjectionError(ValueError):
    """Raised when prompt injection is detected and ``prompt_injection=block``."""


class ExcessiveAgencyError(RuntimeError):
    """Raised when the tool-call count or task-duration limit is exceeded."""


# ---------------------------------------------------------------------------
# OA-01 — Prompt Injection detection
# ---------------------------------------------------------------------------

#: Compiled patterns matched against normalised (lowercased + collapsed) input.
#: Add workspace-specific patterns in config if needed.
_INJECTION_PATTERNS: list[tuple[re.Pattern[str], str]] = [
    # Instruction override
    (re.compile(r"ignore\s+(all\s+)?previous\s+instructions?", re.I), "instruction_override"),
    (re.compile(r"disregard\s+(all\s+)?previous", re.I), "instruction_override"),
    (re.compile(r"forget\s+(all\s+)?previous", re.I), "instruction_override"),
    (re.compile(r"override\s+(your\s+)?(instructions?|guidelines?|rules?)", re.I), "instruction_override"),
    # Role hijacking
    (re.compile(r"you\s+are\s+now\s+\w", re.I), "role_hijack"),
    (re.compile(r"act\s+as\s+(a\s+)?(new\s+|different\s+|unrestricted\s+)", re.I), "role_hijack"),
    (re.compile(r"roleplay\s+as", re.I), "role_hijack"),
    (re.compile(r"pretend\s+(you\s+are|to\s+be)\b", re.I), "role_hijack"),
    (re.compile(r"from\s+now\s+on\s+(you\s+are|act\s+as)", re.I), "role_hijack"),
    # System-prompt delimiter injection (LLM-specific tokens)
    (re.compile(r"<\|?\s*(system|im_start|im_end|endoftext)\s*\|?>", re.I), "delimiter_injection"),
    (re.compile(r"\[INST\]|\[/INST\]|\[\[SYS\]\]|\[\[/SYS\]\]", re.I), "delimiter_injection"),
    (re.compile(r"<</SYS>>|<<SYS>>", re.I), "delimiter_injection"),
    # DAN / jailbreak keywords
    (re.compile(r"\bDAN\b.{0,30}(mode|now|enabled|activated)", re.I), "jailbreak"),
    (re.compile(r"do\s+anything\s+now", re.I), "jailbreak"),
    (re.compile(r"\bjailbreak\b", re.I), "jailbreak"),
    (re.compile(r"developer\s+mode\s+(enabled|on)", re.I), "jailbreak"),
    # Prompt exfiltration
    (re.compile(r"(repeat|print|output|show|reveal|display)\s+(your\s+)?(system\s+prompt|initial\s+instructions?)", re.I), "prompt_exfiltration"),
    (re.compile(r"what\s+(are\s+)?your\s+(instructions?|system\s+prompt)", re.I), "prompt_exfiltration"),
]


def detect_prompt_injection(text: str) -> list[tuple[str, str]]:
    """Return a list of ``(pattern_description, category)`` for each match.

    Args:
        text: Raw user input to scan.

    Returns:
        List of ``(matched_pattern, category)`` tuples; empty means clean.
    """
    matches: list[tuple[str, str]] = []
    for pattern, category in _INJECTION_PATTERNS:
        m = pattern.search(text)
        if m:
            matches.append((m.group(0)[:80], category))
    return matches


def sanitize_input(
    text: str,
    *,
    prompt_injection_mode: str = "detect",
    context_id: str = "",
) -> str:
    """Check *text* for prompt injection and enforce the configured response.

    Args:
        text:                   User-supplied input to the agent.
        prompt_injection_mode:  ``"detect"`` or ``"block"``.
        context_id:             Task/context identifier for audit correlation.

    Returns:
        The original *text* unchanged (``detect`` mode always returns input).

    Raises:
        :class:`PromptInjectionError`: only when ``prompt_injection_mode="block"``
            and at least one injection pattern is matched.
    """
    matches = detect_prompt_injection(text)
    if not matches:
        return text

    categories = list({cat for _, cat in matches})
    trace_id = str(uuid.uuid4())

    log_event(
        event_type="compliance",
        action="prompt_injection.detect",
        resource="user_input",
        outcome="detected" if prompt_injection_mode == "detect" else "blocked",
        trace_id=trace_id,
        context_id=context_id,
        categories=categories,
        match_count=len(matches),
        # Log category + truncated match, never the full raw text (OA-06)
        matches=[{"category": cat, "snippet": snippet} for snippet, cat in matches[:5]],
    )

    if prompt_injection_mode == "block":
        raise PromptInjectionError(
            f"Prompt injection detected ({', '.join(categories)}). "
            "Request blocked by compliance policy."
        )

    # detect mode — log and continue
    logger.warning(
        "Prompt injection patterns detected (context_id=%s, categories=%s) — "
        "passing to agent in detect mode",
        context_id,
        categories,
    )
    return text


# ---------------------------------------------------------------------------
# OA-03 — Excessive Agency
# ---------------------------------------------------------------------------


@dataclass
class AgencyTracker:
    """Per-task mutable state for excessive-agency enforcement.

    Instantiate once per ``execute()`` call and pass to
    :func:`check_agency_limits` at each tool-start event.
    """

    max_tool_calls: int = 50
    max_duration_seconds: float = 300.0
    tool_call_count: int = field(default=0, init=False)
    start_time: float = field(default_factory=time.monotonic, init=False)

    def on_tool_call(self, tool_name: str = "", context_id: str = "") -> None:
        """Increment counter and enforce limits.

        Raises:
            :class:`ExcessiveAgencyError`: if either limit is exceeded.
        """
        self.tool_call_count += 1
        elapsed = time.monotonic() - self.start_time

        if self.tool_call_count > self.max_tool_calls:
            log_event(
                event_type="compliance",
                action="excessive_agency.tool_limit",
                resource=tool_name or "unknown_tool",
                outcome="blocked",
                context_id=context_id,
                tool_call_count=self.tool_call_count,
                limit=self.max_tool_calls,
                elapsed_seconds=round(elapsed, 2),
            )
            raise ExcessiveAgencyError(
                f"Tool call limit exceeded: {self.tool_call_count} calls > "
                f"max {self.max_tool_calls} per task"
            )

        if elapsed > self.max_duration_seconds:
            log_event(
                event_type="compliance",
                action="excessive_agency.duration_limit",
                resource=tool_name or "unknown_tool",
                outcome="blocked",
                context_id=context_id,
                tool_call_count=self.tool_call_count,
                elapsed_seconds=round(elapsed, 2),
                limit_seconds=self.max_duration_seconds,
            )
            raise ExcessiveAgencyError(
                f"Task duration limit exceeded: {elapsed:.0f}s > "
                f"max {self.max_duration_seconds:.0f}s per task"
            )


# ---------------------------------------------------------------------------
# OA-02 / OA-06 — PII redaction
# ---------------------------------------------------------------------------

#: ``(compiled_pattern, replacement_token)`` pairs applied in order.
#: The replacement tokens are SIEM-friendly: ``[REDACTED:type]``.
_PII_PATTERNS: list[tuple[re.Pattern[str], str]] = [
    # Formatted credit cards:  XXXX-XXXX-XXXX-XXXX  or  XXXX XXXX XXXX XXXX
    (re.compile(r"\b\d{4}[\s\-]\d{4}[\s\-]\d{4}[\s\-]\d{4}\b"), "[REDACTED:credit_card]"),
    # US Social Security Numbers:  XXX-XX-XXXX
    (re.compile(r"\b\d{3}-\d{2}-\d{4}\b"), "[REDACTED:ssn]"),
    # OpenAI-style keys: sk-... (≥ 32 chars after prefix)
    (re.compile(r"\bsk-[A-Za-z0-9_\-]{32,}\b"), "[REDACTED:api_key]"),
    # Generic API/secret keys with common prefixes
    (re.compile(r"\b(?:sk|pk|api|secret|token|auth)[-_][A-Za-z0-9_\-]{20,}\b", re.I), "[REDACTED:api_key]"),
    # AWS Access Key IDs
    (re.compile(r"\bAKIA[0-9A-Z]{16}\b"), "[REDACTED:aws_key]"),
    # GitHub personal access tokens
    (re.compile(r"\bghp_[A-Za-z0-9]{36}\b"), "[REDACTED:github_token]"),
    # Email addresses
    (re.compile(r"\b[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}\b"), "[REDACTED:email]"),
]


def redact_pii(text: str) -> tuple[str, list[str]]:
    """Redact PII from *text* and return ``(redacted_text, pii_types_found)``.

    Each unique PII type is reported at most once in ``pii_types_found``.
    The replacement tokens (``[REDACTED:type]``) are SIEM-indexable and
    preserve the structural context of the output while hiding sensitive data.

    Args:
        text: Agent output text to scan.

    Returns:
        Tuple of ``(redacted_text, list_of_pii_type_strings)``.  The list is
        empty when no PII is detected (the common case).

    Examples::

        >>> redacted, types = redact_pii("Call me at test@example.com sk-abc123...")
        >>> "email" in types
        True
        >>> "[REDACTED:email]" in redacted
        True
    """
    found: list[str] = []
    result = text
    for pattern, replacement in _PII_PATTERNS:
        new_result = pattern.sub(replacement, result)
        if new_result != result:
            # Extract type from "[REDACTED:type]"
            pii_type = replacement[len("[REDACTED:"):-1]
            if pii_type not in found:
                found.append(pii_type)
            result = new_result
    return result, found


# ---------------------------------------------------------------------------
# Compliance posture report
# ---------------------------------------------------------------------------


def get_compliance_posture() -> dict[str, Any]:
    """Return the current compliance configuration as a serialisable dict.

    Loads ``WorkspaceConfig`` lazily (cached) and returns a snapshot of the
    active compliance settings.  Safe to call from a health endpoint.

    Returns a dict with these keys::

        {
          "compliance_mode": "owasp_agentic" | "",
          "enabled": true | false,
          "prompt_injection": "detect" | "block",
          "max_tool_calls_per_task": 50,
          "max_task_duration_seconds": 300,
          "pii_redaction_enabled": true,
          "security_scan_mode": "warn" | "block" | "off",
          "rbac_roles": ["operator"],
        }
    """
    try:
        from builtin_tools.audit import _load_workspace_config
        cfg = _load_workspace_config()
    except Exception:
        cfg = None

    if cfg is None:
        return {
            "compliance_mode": "",
            "enabled": False,
            "prompt_injection": "detect",
            "max_tool_calls_per_task": 50,
            "max_task_duration_seconds": 300,
            "pii_redaction_enabled": False,
            "security_scan_mode": "warn",
            "rbac_roles": [],
            "note": "config unavailable",
        }

    c = cfg.compliance
    enabled = c.mode == "owasp_agentic"
    return {
        "compliance_mode": c.mode,
        "enabled": enabled,
        "prompt_injection": c.prompt_injection,
        "max_tool_calls_per_task": c.max_tool_calls_per_task,
        "max_task_duration_seconds": c.max_task_duration_seconds,
        # PII redaction is active whenever compliance mode is on
        "pii_redaction_enabled": enabled,
        "security_scan_mode": cfg.security_scan.mode,
        "rbac_roles": list(cfg.rbac.roles),
    }
