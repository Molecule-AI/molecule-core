"""Canonical namespace helpers for workspace-scoped resources."""

from __future__ import annotations


def workspace_awareness_namespace(workspace_id: str) -> str:
    """Return the default awareness namespace for a workspace."""
    workspace_id = workspace_id.strip()
    return f"workspace:{workspace_id}" if workspace_id else "workspace:unknown"


def resolve_awareness_namespace(
    workspace_id: str,
    configured_namespace: str | None = None,
) -> str:
    """Return the configured namespace, or the workspace default when unset."""
    namespace = (configured_namespace or "").strip()
    return namespace or workspace_awareness_namespace(workspace_id)
