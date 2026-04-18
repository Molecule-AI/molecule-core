"""Tests for tools/audit.py — RBAC, audit logging, and workspace roles.

Loads the *real* module via importlib to bypass the conftest mock for
tools.audit, so every test exercises the actual implementation.
"""

from __future__ import annotations

import os
import importlib.util
import os
import json
import os
import sys
from types import ModuleType
from unittest.mock import MagicMock, patch

import os
import pytest


# ---------------------------------------------------------------------------
# Fixture — load the real tools.audit module
# ---------------------------------------------------------------------------

@pytest.fixture
def real_audit(monkeypatch, tmp_path):
    """Load the real tools/audit.py, bypassing the conftest mock."""
    # Remove mocks so the real module is loaded fresh
    monkeypatch.delitem(sys.modules, "builtin_tools.audit", raising=False)
    monkeypatch.delitem(sys.modules, "builtin_tools.compliance", raising=False)

    # Point audit log at a temp file so tests don't hit the filesystem
    monkeypatch.setenv("AUDIT_LOG_PATH", str(tmp_path / "audit.jsonl"))
    monkeypatch.setenv("WORKSPACE_ID", "test-ws")

    spec = importlib.util.spec_from_file_location(
        "builtin_tools.audit",
        os.path.join(os.path.dirname(__file__), "..", "builtin_tools/audit.py"),
    )
    mod = importlib.util.module_from_spec(spec)
    monkeypatch.setitem(sys.modules, "builtin_tools.audit", mod)
    spec.loader.exec_module(mod)

    # Re-read env vars into the module-level constants (they are read at import)
    mod.AUDIT_LOG_PATH = str(tmp_path / "audit.jsonl")
    mod.WORKSPACE_ID = "test-ws"

    return mod


# ---------------------------------------------------------------------------
# check_permission — built-in roles
# ---------------------------------------------------------------------------

class TestCheckPermissionBuiltinRoles:

    def test_check_permission_admin(self, real_audit):
        """admin shortcircuits and returns True for any action."""
        mod = real_audit
        assert mod.check_permission("delegate", ["admin"]) is True
        assert mod.check_permission("approve", ["admin"]) is True
        assert mod.check_permission("memory.read", ["admin"]) is True
        assert mod.check_permission("memory.write", ["admin"]) is True
        assert mod.check_permission("totally_unknown_action", ["admin"]) is True

    def test_check_permission_operator(self, real_audit):
        """operator has delegate, approve, memory.read, memory.write."""
        mod = real_audit
        assert mod.check_permission("delegate", ["operator"]) is True
        assert mod.check_permission("approve", ["operator"]) is True
        assert mod.check_permission("memory.read", ["operator"]) is True
        assert mod.check_permission("memory.write", ["operator"]) is True
        assert mod.check_permission("rbac.deny", ["operator"]) is False

    def test_check_permission_read_only(self, real_audit):
        """read-only has only memory.read; no delegation or approval."""
        mod = real_audit
        assert mod.check_permission("memory.read", ["read-only"]) is True
        assert mod.check_permission("delegate", ["read-only"]) is False
        assert mod.check_permission("approve", ["read-only"]) is False
        assert mod.check_permission("memory.write", ["read-only"]) is False

    def test_check_permission_no_delegation(self, real_audit):
        """no-delegation cannot delegate, but can approve and write memory."""
        mod = real_audit
        assert mod.check_permission("delegate", ["no-delegation"]) is False
        assert mod.check_permission("approve", ["no-delegation"]) is True
        assert mod.check_permission("memory.read", ["no-delegation"]) is True
        assert mod.check_permission("memory.write", ["no-delegation"]) is True

    def test_check_permission_no_approval(self, real_audit):
        """no-approval cannot approve, but can delegate and write memory."""
        mod = real_audit
        assert mod.check_permission("approve", ["no-approval"]) is False
        assert mod.check_permission("delegate", ["no-approval"]) is True
        assert mod.check_permission("memory.read", ["no-approval"]) is True
        assert mod.check_permission("memory.write", ["no-approval"]) is True

    def test_check_permission_memory_readonly(self, real_audit):
        """memory-readonly can only read memory."""
        mod = real_audit
        assert mod.check_permission("memory.read", ["memory-readonly"]) is True
        assert mod.check_permission("memory.write", ["memory-readonly"]) is False
        assert mod.check_permission("delegate", ["memory-readonly"]) is False
        assert mod.check_permission("approve", ["memory-readonly"]) is False


# ---------------------------------------------------------------------------
# check_permission — custom roles
# ---------------------------------------------------------------------------

class TestCheckPermissionCustomRoles:

    def test_check_permission_custom_roles(self, real_audit):
        """A role defined in custom_permissions is respected."""
        mod = real_audit
        custom = {"developer": ["deploy", "memory.read"]}
        assert mod.check_permission("deploy", ["developer"], custom) is True
        assert mod.check_permission("memory.read", ["developer"], custom) is True

    def test_check_permission_custom_role_no_builtin_fallthrough(self, real_audit):
        """Custom role with custom_permissions does NOT fall through to built-ins.

        'operator' is also a built-in role, but if it appears in custom_permissions
        with a restricted list, the custom list is the complete permission set.
        """
        mod = real_audit
        # Override 'operator' to only allow memory.read via custom_permissions
        custom = {"operator": ["memory.read"]}
        # memory.read is in the custom list — allowed
        assert mod.check_permission("memory.read", ["operator"], custom) is True
        # delegate is in the built-in operator set but NOT in the custom list
        # — must be denied because custom entry is definitive
        assert mod.check_permission("delegate", ["operator"], custom) is False

    def test_check_permission_unknown_role(self, real_audit):
        """A role that exists neither in built-ins nor custom_permissions returns False."""
        mod = real_audit
        assert mod.check_permission("delegate", ["ghost-role"]) is False
        assert mod.check_permission("approve", ["phantom", "specter"]) is False

    def test_check_permission_empty_roles(self, real_audit):
        """An empty roles list always returns False."""
        mod = real_audit
        assert mod.check_permission("delegate", []) is False
        assert mod.check_permission("memory.read", []) is False


# ---------------------------------------------------------------------------
# log_event
# ---------------------------------------------------------------------------

class TestLogEvent:

    def test_log_event_writes_json_line(self, real_audit, tmp_path):
        """log_event appends a valid JSON line to the audit file."""
        mod = real_audit
        mod.log_event(
            event_type="delegation",
            action="delegate",
            resource="billing-agent",
            outcome="success",
        )
        log_file = tmp_path / "audit.jsonl"
        assert log_file.exists(), "audit file was not created"
        lines = log_file.read_text(encoding="utf-8").strip().splitlines()
        assert len(lines) == 1
        event = json.loads(lines[0])
        assert event["event_type"] == "delegation"
        assert event["action"] == "delegate"
        assert event["resource"] == "billing-agent"
        assert event["outcome"] == "success"
        assert "timestamp" in event
        assert "trace_id" in event
        assert "workspace_id" in event

    def test_log_event_returns_trace_id(self, real_audit):
        """log_event returns the trace_id string."""
        mod = real_audit
        result = mod.log_event(
            event_type="rbac",
            action="rbac.deny",
            resource="memory-scope",
            outcome="denied",
        )
        assert isinstance(result, str)
        assert len(result) > 0

    def test_log_event_custom_trace_id(self, real_audit, tmp_path):
        """log_event uses the caller-supplied trace_id."""
        mod = real_audit
        supplied_id = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
        returned_id = mod.log_event(
            event_type="approval",
            action="approve",
            resource="deploy",
            outcome="granted",
            trace_id=supplied_id,
        )
        assert returned_id == supplied_id
        log_file = tmp_path / "audit.jsonl"
        event = json.loads(log_file.read_text().strip())
        assert event["trace_id"] == supplied_id

    def test_log_event_actor_default(self, real_audit, tmp_path):
        """actor defaults to WORKSPACE_ID when not supplied."""
        mod = real_audit
        mod.WORKSPACE_ID = "test-ws"
        mod.log_event(
            event_type="memory",
            action="memory.read",
            resource="global-scope",
            outcome="success",
        )
        log_file = tmp_path / "audit.jsonl"
        event = json.loads(log_file.read_text().strip())
        assert event["actor"] == "test-ws"

    def test_log_event_extra_fields(self, real_audit, tmp_path):
        """Extra kwargs are written to the JSON; built-in keys cannot be overridden.

        The built-in key 'workspace_id' is set automatically by the module
        (not a function parameter), so passing it via **extra exercises the
        "built-in keys are not overridable" guard in log_event.
        """
        mod = real_audit
        mod.WORKSPACE_ID = "real-ws"
        # 'workspace_id' is a built-in event key — must not be overwritten by extra
        mod.log_event(
            event_type="delegation",
            action="delegate",
            resource="target-ws",
            outcome="success",
            attempt=3,
            target_workspace_id="target-ws",
            workspace_id="SHOULD-NOT-APPEAR",  # built-in key override attempt
        )
        log_file = tmp_path / "audit.jsonl"
        event = json.loads(log_file.read_text().strip())
        # Extra fields present
        assert event["attempt"] == 3
        assert event["target_workspace_id"] == "target-ws"
        # Built-in 'workspace_id' is NOT overridden by the extra kwarg
        assert event["workspace_id"] == "real-ws"

    def test_log_event_write_failure_does_not_raise(self, real_audit, tmp_path, monkeypatch):
        """If the file write fails (e.g. fsync raises), only a WARNING is logged; no exception."""
        mod = real_audit
        import os as _os
        monkeypatch.setattr(_os, "fsync", lambda fd: (_ for _ in ()).throw(OSError("disk full")))
        # Must not raise
        mod.log_event(
            event_type="memory",
            action="memory.write",
            resource="scope",
            outcome="failure",
        )


# ---------------------------------------------------------------------------
# get_workspace_roles
# ---------------------------------------------------------------------------

class TestGetWorkspaceRoles:

    def test_get_workspace_roles_config_available(self, real_audit, monkeypatch):
        """Returns roles and allowed_actions from the workspace config."""
        mod = real_audit

        # Build a minimal config mock
        mock_rbac = MagicMock()
        mock_rbac.roles = ["operator", "read-only"]
        mock_rbac.allowed_actions = {"developer": ["deploy"]}
        mock_cfg = MagicMock()
        mock_cfg.rbac = mock_rbac

        mock_config_mod = ModuleType("config")
        mock_config_mod.load_config = MagicMock(return_value=mock_cfg)
        monkeypatch.setitem(sys.modules, "config", mock_config_mod)

        # Clear the lru_cache so our new mock is used
        mod._load_workspace_config.cache_clear()
        try:
            roles, allowed_actions = mod.get_workspace_roles()
            assert roles == ["operator", "read-only"]
            assert allowed_actions == {"developer": ["deploy"]}
        finally:
            mod._load_workspace_config.cache_clear()

    def test_get_workspace_roles_config_unavailable(self, real_audit, monkeypatch):
        """Falls back to (['operator'], {}) when config cannot be loaded."""
        mod = real_audit

        # Make load_config raise
        mock_config_mod = ModuleType("config")
        mock_config_mod.load_config = MagicMock(side_effect=RuntimeError("config missing"))
        monkeypatch.setitem(sys.modules, "config", mock_config_mod)

        mod._load_workspace_config.cache_clear()
        try:
            roles, allowed_actions = mod.get_workspace_roles()
            assert roles == ["operator"]
            assert allowed_actions == {}
        finally:
            mod._load_workspace_config.cache_clear()
