"""Validator for social-channel configurations embedded in org.yaml / direct API payloads.

The platform's Go channel adapters (``platform/internal/channels/``) are the
authoritative implementations (Telegram first, Slack/Discord/WhatsApp on the
roadmap). This module provides a Python-side schema check for the YAML /
JSON blob that users write — so authors catch misspelled fields before the
platform rejects them.

Shape (matches ``platform/internal/handlers/channels.go``):

.. code-block:: yaml

    type: telegram
    config:
      bot_token: ${TELEGRAM_BOT_TOKEN}   # platform-resolved env var
      chat_id: ${TELEGRAM_CHAT_ID}
    enabled: true                        # default true

Supported types track what the platform knows about via the channel
adapter registry. Keep in sync with ``channels.ChannelAdapter.Type()``.
"""
from __future__ import annotations

from pathlib import Path
from typing import Any

import yaml

from .workspace import ValidationError


# Channel types the platform has adapters for, as of today. New adapters
# (slack, discord, whatsapp) are welcome additions — update this set when
# the corresponding Go adapter lands.
SUPPORTED_CHANNEL_TYPES = frozenset({"telegram"})

# Per-type required config keys. Empty tuple = no required keys (for
# adapters that accept zero config).
_REQUIRED_KEYS: dict[str, tuple[str, ...]] = {
    "telegram": ("bot_token",),
}


def validate_channel_config(
    cfg: dict[str, Any], file_ref: str = "<channel>"
) -> list[ValidationError]:
    """Validate a single channel config dict (not a file)."""
    errors: list[ValidationError] = []

    ch_type = cfg.get("type")
    if not ch_type:
        errors.append(ValidationError(file_ref, "missing required field: type"))
        return errors
    if ch_type not in SUPPORTED_CHANNEL_TYPES:
        errors.append(
            ValidationError(
                file_ref,
                f"type={ch_type!r} — must be one of {sorted(SUPPORTED_CHANNEL_TYPES)}",
            )
        )
        return errors

    config = cfg.get("config")
    if config is not None and not isinstance(config, dict):
        errors.append(ValidationError(file_ref, f"config must be an object; got {type(config).__name__}"))
        return errors

    required = _REQUIRED_KEYS.get(ch_type, ())
    for key in required:
        if not config or key not in config:
            errors.append(
                ValidationError(file_ref, f"config.{key} is required for type={ch_type!r}")
            )

    if "enabled" in cfg and not isinstance(cfg["enabled"], bool):
        errors.append(ValidationError(file_ref, f"enabled must be a boolean; got {type(cfg['enabled']).__name__}"))

    return errors


def validate_channel_file(path: Path) -> list[ValidationError]:
    """Validate a YAML / JSON file containing a channel config or a list of them."""
    if not path.exists():
        return [ValidationError(str(path), "file does not exist")]

    try:
        doc = yaml.safe_load(path.read_text())
    except yaml.YAMLError as exc:
        return [ValidationError(str(path), f"invalid YAML / JSON: {exc}")]

    if doc is None:
        return [ValidationError(str(path), "file is empty")]

    errors: list[ValidationError] = []
    if isinstance(doc, list):
        for i, entry in enumerate(doc):
            if not isinstance(entry, dict):
                errors.append(ValidationError(str(path), f"[{i}]: entry must be an object"))
                continue
            errors.extend(validate_channel_config(entry, f"{path}[{i}]"))
    elif isinstance(doc, dict):
        errors.extend(validate_channel_config(doc, str(path)))
    else:
        errors.append(ValidationError(str(path), f"top-level must be a channel object or list; got {type(doc).__name__}"))
    return errors


__all__ = [
    "SUPPORTED_CHANNEL_TYPES",
    "validate_channel_config",
    "validate_channel_file",
]
