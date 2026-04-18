"""Tests for canonical namespace helpers."""

from policies.namespaces import resolve_awareness_namespace, workspace_awareness_namespace


def test_workspace_awareness_namespace_is_stable():
    assert workspace_awareness_namespace("ws-123") == "workspace:ws-123"
    assert workspace_awareness_namespace("  ws-123  ") == "workspace:ws-123"
    assert workspace_awareness_namespace("") == "workspace:unknown"


def test_resolve_awareness_namespace_prefers_configured_value():
    assert resolve_awareness_namespace("ws-123", "custom:ns") == "custom:ns"
    assert resolve_awareness_namespace("ws-123", "  custom:ns  ") == "custom:ns"
    assert resolve_awareness_namespace("ws-123", "") == "workspace:ws-123"
