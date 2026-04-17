"""Secret-scrubbing utilities for workspace runtime (#834 — C2).

Provides ``_redact_secrets()`` applied at every ``commit_memory`` call site
to prevent API keys and tokens from being persisted verbatim in the
memories table.

Design notes
------------
- **Allowlist of known prefixes** (``sk-``, ``ghp_``, etc.) cover the most
  dangerous tokens because they are unambiguous.
- **Contextual pattern** covers generic high-entropy values that appear
  immediately after assignment keywords (``key=``, ``token=``, ``secret=``,
  ``password=``, ``api_key=``).  The keyword is preserved in the output so
  log lines remain readable; only the value is redacted.
- **Idempotent**: the replacement token ``[REDACTED]`` does not match any
  of the patterns, so calling ``_redact_secrets`` twice is safe.
- **No false-positive risk on normal prose**: all patterns require either
  a well-known prefix (``AKIA``, ``ghp_``, ``sk-``) or both a keyword and
  ≥ 40 base64/alphanumeric chars — ordinary English words never match.

Relationship to ``compliance.redact_pii``
------------------------------------------
``redact_pii`` handles PII (emails, SSNs, credit cards) and uses typed
tokens ``[REDACTED:type]`` for SIEM indexing.  ``_redact_secrets`` is
narrowly scoped to API credentials and uses the plain ``[REDACTED]`` token
because the exact secret type is not important at the storage layer —
what matters is that no credential value ever reaches the database.
"""

from __future__ import annotations

import re
from typing import List

# ---------------------------------------------------------------------------
# Replacement sentinel
# ---------------------------------------------------------------------------

#: Replacement token — deliberately plain so downstream readers do not need
#: to parse structured tokens.  Does not match any scrub pattern (idempotent).
REDACTED: str = "[REDACTED]"

# ---------------------------------------------------------------------------
# Patterns
# ---------------------------------------------------------------------------

# Patterns that identify secret values by their well-known prefix.
# Ordered from most specific to least specific.
_BARE_PATTERNS: List[re.Pattern] = [
    # OpenAI / Anthropic-style keys: sk-<20+ alnum/hyphen/underscore chars>
    # Covers: sk-<key>, sk-ant-<key>, sk-proj-<key>, etc.
    re.compile(r"\bsk-[A-Za-z0-9_-]{20,}\b"),
    # GitHub classic personal access token
    re.compile(r"\bghp_[A-Za-z0-9]{36}\b"),
    # GitHub server-to-server token
    re.compile(r"\bghs_[A-Za-z0-9]{36}\b"),
    # GitHub fine-grained personal access token
    re.compile(r"\bgithub_pat_[A-Za-z0-9_]{82}\b"),
    # AWS access key ID
    re.compile(r"\bAKIA[0-9A-Z]{16}\b"),
]

# Contextual pattern: keyword= followed by a high-entropy value.
#
# Group 1 captures the keyword + equals sign so it is preserved in the
# replacement — "api_key=[REDACTED]" is more informative than "[REDACTED]".
#
# The value charset [A-Za-z0-9+/] covers base64 and common token alphabets.
# The minimum length of 40 chars prevents false-positives on short values.
_CONTEXTUAL_RE: re.Pattern = re.compile(
    r"(?i)"
    r"((?:api_key|key|token|secret|password)\s*=\s*)"
    r"([A-Za-z0-9+/]{40,}={0,2})"
)


# ---------------------------------------------------------------------------
# Public API
# ---------------------------------------------------------------------------


def _redact_secrets(content: str) -> str:
    """Scrub known secret patterns from *content*, replacing with ``[REDACTED]``.

    Parameters
    ----------
    content:
        Raw string to scrub — typically a ``commit_memory`` payload.

    Returns
    -------
    str
        Copy of *content* with secrets replaced.  If no secrets are found,
        the original string is returned unchanged.  Calling this function
        on already-redacted content is safe (idempotent).

    Examples::

        >>> _redact_secrets("token is sk-abc1234567890123456789012345")
        'token is [REDACTED]'

        >>> _redact_secrets("api_key=" + "A" * 45)
        'api_key=[REDACTED]'

        >>> _redact_secrets("The answer is 42.")
        'The answer is 42.'

        >>> _redact_secrets("[REDACTED]")
        '[REDACTED]'
    """
    result = content

    # Apply prefix-based patterns first (most unambiguous)
    for pattern in _BARE_PATTERNS:
        result = pattern.sub(REDACTED, result)

    # Apply contextual pattern — preserve keyword, replace only the value
    result = _CONTEXTUAL_RE.sub(r"\1" + REDACTED, result)

    return result
