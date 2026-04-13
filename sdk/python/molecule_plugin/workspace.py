"""Validator for workspace-configs-templates/<name>/config.yaml.

A **workspace template** is a directory the platform copies into a new
workspace's /configs volume at provision time. It contains at minimum a
``config.yaml`` declaring the agent's runtime, model defaults, and env
requirements; optionally ``CLAUDE.md``, ``system-prompt.md``, ``skills/``,
etc.

This module validates the shape of a workspace-template directory so
authors can catch errors before publishing. Called from
``python -m molecule_plugin validate workspace <dir>``.
"""
from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
from typing import Any

import yaml


# Runtimes the platform knows how to provision. Stays aligned with
# provisioner.RuntimeImages in platform/internal/provisioner/provisioner.go.
SUPPORTED_RUNTIMES = frozenset(
    {
        "langgraph",
        "claude-code",
        "claude_code",  # adapter dirs use underscores
        "openclaw",
        "deepagents",
        "crewai",
        "autogen",
    }
)


@dataclass
class ValidationError:
    """Single problem found in a workspace template."""
    file: str
    message: str


def validate_workspace_template(path: Path) -> list[ValidationError]:
    """Validate a workspace-template directory.

    Returns an empty list when the template is well-formed. Each element
    in the returned list is a distinct problem — callers render them as
    a checklist for the author.
    """
    errors: list[ValidationError] = []

    config_path = path / "config.yaml"
    if not config_path.exists():
        errors.append(ValidationError(str(config_path), "missing config.yaml"))
        return errors

    try:
        config = yaml.safe_load(config_path.read_text()) or {}
    except yaml.YAMLError as exc:
        errors.append(ValidationError(str(config_path), f"invalid YAML: {exc}"))
        return errors

    if not isinstance(config, dict):
        errors.append(ValidationError(str(config_path), "config.yaml must be a YAML object"))
        return errors

    # Required top-level fields
    for field in ("name", "runtime"):
        if field not in config or not config[field]:
            errors.append(ValidationError(str(config_path), f"missing required field: {field}"))

    # Runtime must be one the platform knows about
    runtime = config.get("runtime")
    if runtime and runtime not in SUPPORTED_RUNTIMES:
        errors.append(
            ValidationError(
                str(config_path),
                f"runtime={runtime!r} — must be one of: {sorted(SUPPORTED_RUNTIMES)}",
            )
        )

    # Tier is optional but when present must be 1/2/3
    if "tier" in config and config["tier"] not in (1, 2, 3):
        errors.append(
            ValidationError(str(config_path), f"tier must be 1, 2, or 3; got {config['tier']!r}")
        )

    # runtime_config (when present) should be a dict
    rc = config.get("runtime_config")
    if rc is not None and not isinstance(rc, dict):
        errors.append(
            ValidationError(str(config_path), "runtime_config must be an object")
        )
    elif isinstance(rc, dict):
        required_env = rc.get("required_env", [])
        if required_env is not None and not isinstance(required_env, list):
            errors.append(
                ValidationError(
                    str(config_path),
                    "runtime_config.required_env must be a list of env var names",
                )
            )
        timeout = rc.get("timeout")
        if timeout is not None and not isinstance(timeout, (int, float)):
            errors.append(
                ValidationError(
                    str(config_path),
                    f"runtime_config.timeout must be a number; got {type(timeout).__name__}",
                )
            )

    return errors


# Re-exported for type hints in __init__.py
__all__ = ["ValidationError", "SUPPORTED_RUNTIMES", "validate_workspace_template"]
