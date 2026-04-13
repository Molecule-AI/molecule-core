"""Validator for org-templates/<name>/org.yaml.

An **org template** defines a hierarchical team of workspaces — typically a
PM with research + dev branches, each with their own children. The platform
instantiates the whole tree on ``POST /org/import``.

Schema (matches ``platform/internal/handlers/org.go::OrgWorkspace``):

.. code-block:: yaml

    name: Molecule AI Dev Team
    description: AI agent company for building Molecule AI
    defaults:                    # inherited by every workspace unless overridden
      runtime: claude-code
      tier: 2
      required_env: [CLAUDE_CODE_OAUTH_TOKEN]
      initial_prompt: |
        ...
    workspaces:
      - name: PM
        role: Project Manager
        tier: 3
        files_dir: pm
        channels:                # optional social channel configs
          - type: telegram
            config: {bot_token: ${TELEGRAM_BOT_TOKEN}}
            enabled: true
        workspace_access: read_only   # #65: none | read_only | read_write
        children:
          - name: Research Lead
            ...

This module catches schema errors before ``POST /org/import`` so authors
don't burn platform cycles on typos.
"""
from __future__ import annotations

from pathlib import Path
from typing import Any

import yaml

from .channel import validate_channel_config
from .workspace import SUPPORTED_RUNTIMES, ValidationError


# Workspace-access values — mirrors the CHECK constraint in
# platform/migrations/019_workspace_access.up.sql. #65.
_WORKSPACE_ACCESS_VALUES = frozenset({"none", "read_only", "read_write"})


def _validate_workspace_node(
    node: Any,
    path: str,
    file_ref: str,
    errors: list[ValidationError],
) -> None:
    """Recursively validate a single workspace node (and its children)."""
    if not isinstance(node, dict):
        errors.append(ValidationError(file_ref, f"{path}: must be an object, got {type(node).__name__}"))
        return

    # Required
    if not node.get("name"):
        errors.append(ValidationError(file_ref, f"{path}: missing required field 'name'"))

    # Tier (optional)
    if "tier" in node and node["tier"] not in (1, 2, 3):
        errors.append(
            ValidationError(file_ref, f"{path}: tier must be 1, 2, or 3; got {node['tier']!r}")
        )

    # Runtime (optional — inherited from defaults)
    runtime = node.get("runtime")
    if runtime and runtime not in SUPPORTED_RUNTIMES:
        errors.append(
            ValidationError(
                file_ref,
                f"{path}: runtime={runtime!r} — must be one of {sorted(SUPPORTED_RUNTIMES)}",
            )
        )

    # workspace_access (#65)
    access = node.get("workspace_access")
    if access is not None and access not in _WORKSPACE_ACCESS_VALUES:
        errors.append(
            ValidationError(
                file_ref,
                f"{path}: workspace_access={access!r} — must be one of {sorted(_WORKSPACE_ACCESS_VALUES)}",
            )
        )
    if access in ("read_only", "read_write") and not node.get("workspace_dir"):
        errors.append(
            ValidationError(
                file_ref,
                f"{path}: workspace_access={access!r} requires workspace_dir to be set",
            )
        )

    # Channels (optional list)
    channels = node.get("channels")
    if channels is not None:
        if not isinstance(channels, list):
            errors.append(ValidationError(file_ref, f"{path}.channels: must be a list"))
        else:
            for i, ch in enumerate(channels):
                if not isinstance(ch, dict):
                    errors.append(
                        ValidationError(file_ref, f"{path}.channels[{i}]: must be an object")
                    )
                    continue
                # Delegate to channel validator — single source of truth for channel schema.
                ch_ref = f"{file_ref}:{path}.channels[{i}]"
                errors.extend(validate_channel_config(ch, ch_ref))

    # Schedules (optional list)
    schedules = node.get("schedules")
    if schedules is not None:
        if not isinstance(schedules, list):
            errors.append(ValidationError(file_ref, f"{path}.schedules: must be a list"))
        else:
            for i, sch in enumerate(schedules):
                if not isinstance(sch, dict):
                    errors.append(
                        ValidationError(file_ref, f"{path}.schedules[{i}]: must be an object")
                    )
                    continue
                if not sch.get("cron_expr"):
                    errors.append(
                        ValidationError(
                            file_ref, f"{path}.schedules[{i}]: missing 'cron_expr'"
                        )
                    )
                if not sch.get("prompt"):
                    errors.append(
                        ValidationError(
                            file_ref, f"{path}.schedules[{i}]: missing 'prompt'"
                        )
                    )

    # Plugins (optional list of strings)
    plugins = node.get("plugins")
    if plugins is not None:
        if not isinstance(plugins, list) or not all(isinstance(p, str) for p in plugins):
            errors.append(ValidationError(file_ref, f"{path}.plugins: must be a list of strings"))

    # External workspaces must declare a URL
    if node.get("external") and not node.get("url"):
        errors.append(
            ValidationError(file_ref, f"{path}: external=true requires url to be set")
        )

    # Recurse into children
    children = node.get("children")
    if children is not None:
        if not isinstance(children, list):
            errors.append(ValidationError(file_ref, f"{path}.children: must be a list"))
        else:
            for i, child in enumerate(children):
                cname = child.get("name", "?") if isinstance(child, dict) else "?"
                _validate_workspace_node(
                    child, f"{path}.children[{i}:{cname}]", file_ref, errors
                )


def validate_org_template(path: Path) -> list[ValidationError]:
    """Validate an org-template directory (must contain org.yaml)."""
    errors: list[ValidationError] = []

    org_yaml = path / "org.yaml"
    if not org_yaml.exists():
        errors.append(ValidationError(str(org_yaml), "missing org.yaml"))
        return errors

    try:
        org = yaml.safe_load(org_yaml.read_text()) or {}
    except yaml.YAMLError as exc:
        errors.append(ValidationError(str(org_yaml), f"invalid YAML: {exc}"))
        return errors

    if not isinstance(org, dict):
        errors.append(ValidationError(str(org_yaml), "org.yaml must be a YAML object"))
        return errors

    if not org.get("name"):
        errors.append(ValidationError(str(org_yaml), "missing required field: name"))

    # defaults block (optional but common)
    defaults = org.get("defaults")
    if defaults is not None and not isinstance(defaults, dict):
        errors.append(ValidationError(str(org_yaml), "defaults must be an object"))

    workspaces = org.get("workspaces")
    if not workspaces:
        errors.append(ValidationError(str(org_yaml), "missing required field: workspaces (non-empty list)"))
    elif not isinstance(workspaces, list):
        errors.append(ValidationError(str(org_yaml), "workspaces must be a list"))
    else:
        for i, ws in enumerate(workspaces):
            _validate_workspace_node(ws, f"workspaces[{i}:{ws.get('name','?') if isinstance(ws, dict) else '?'}]", str(org_yaml), errors)

    return errors


__all__ = ["validate_org_template"]
