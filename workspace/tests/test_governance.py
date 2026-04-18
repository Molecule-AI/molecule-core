"""Tests for tools/governance.py — GovernanceAdapter and module-level functions.

Loads the real module via importlib to bypass the conftest mock for
tools.governance, exercising actual implementation logic including
graceful degradation when agent-os-kernel is not installed.
"""

from __future__ import annotations

import os
import importlib.util
import os
import sys
from unittest.mock import MagicMock, AsyncMock

import os
import pytest


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _make_config(
    policy_mode="audit",
    enabled=True,
    toolkit="microsoft",
    policy_endpoint="",
    policy_file="",
    blocked_patterns=None,
    max_tool_calls_per_task=50,
):
    cfg = MagicMock()
    cfg.enabled = enabled
    cfg.toolkit = toolkit
    cfg.policy_mode = policy_mode
    cfg.policy_endpoint = policy_endpoint
    cfg.policy_file = policy_file
    cfg.blocked_patterns = blocked_patterns or []
    cfg.max_tool_calls_per_task = max_tool_calls_per_task
    return cfg


def _load_governance_module(monkeypatch, mock_audit, mock_telemetry, with_agent_os=False):
    """Load tools/governance.py fresh, injecting mock dependencies."""
    # Provide mock tools.audit
    tools_mod = MagicMock()
    tools_mod.audit = mock_audit
    monkeypatch.setitem(sys.modules, "tools", tools_mod)
    monkeypatch.setitem(sys.modules, "builtin_tools.audit", mock_audit)
    monkeypatch.setitem(sys.modules, "builtin_tools.telemetry", mock_telemetry)

    if not with_agent_os:
        # Ensure agent_os is NOT installed (graceful degradation)
        monkeypatch.setitem(sys.modules, "agent_os", None)
        monkeypatch.setitem(sys.modules, "agent_os.policies", None)

    monkeypatch.delitem(sys.modules, "builtin_tools.governance", raising=False)
    spec = importlib.util.spec_from_file_location(
        "builtin_tools.governance",
        os.path.join(os.path.dirname(__file__), "..", "builtin_tools", "governance.py"),
    )
    mod = importlib.util.module_from_spec(spec)
    monkeypatch.setitem(sys.modules, "builtin_tools.governance", mod)
    spec.loader.exec_module(mod)
    # Reset global singleton
    mod._adapter = None
    return mod


# ---------------------------------------------------------------------------
# Base fixture (no agent_os toolkit)
# ---------------------------------------------------------------------------


@pytest.fixture
def real_governance(monkeypatch):
    """Load real governance module with no agent_os toolkit available."""
    mock_audit = MagicMock()
    mock_audit.check_permission = MagicMock(return_value=True)
    mock_audit.log_event = MagicMock(return_value="trace-abc")

    mock_telemetry = MagicMock()
    mock_telemetry.get_current_traceparent = MagicMock(return_value="00-abc-def-01")

    mod = _load_governance_module(monkeypatch, mock_audit, mock_telemetry, with_agent_os=False)
    return mod, mock_audit, mock_telemetry


# ---------------------------------------------------------------------------
# Toolkit fixture helper
# ---------------------------------------------------------------------------


def _make_toolkit_mocks():
    """Return (mock_decision, mock_evaluator_instance, MockPolicyEvaluator, mock_agent_os_policies)."""
    mock_decision = MagicMock()
    mock_decision.allowed = True
    mock_decision.reason = "policy_ok"
    mock_decision.evaluator_name = "test-evaluator"

    mock_evaluator_instance = MagicMock()
    mock_evaluator_instance.evaluate = MagicMock(return_value=mock_decision)

    MockPolicyEvaluator = MagicMock(return_value=mock_evaluator_instance)

    mock_agent_os_policies = MagicMock()
    mock_agent_os_policies.PolicyEvaluator = MockPolicyEvaluator

    return mock_decision, mock_evaluator_instance, MockPolicyEvaluator, mock_agent_os_policies


# ---------------------------------------------------------------------------
# Test 1: GovernanceAdapter constructor
# ---------------------------------------------------------------------------


class TestGovernanceAdapterInit:

    def test_governance_adapter_init(self, real_governance):
        """GovernanceAdapter(config) creates adapter with _toolkit_available=False."""
        mod, mock_audit, mock_telemetry = real_governance
        cfg = _make_config()
        adapter = mod.GovernanceAdapter(cfg)
        assert adapter._config is cfg
        assert adapter._evaluator is None
        assert adapter._toolkit_available is False


# ---------------------------------------------------------------------------
# Test 2: _init_evaluator — no toolkit
# ---------------------------------------------------------------------------


class TestInitEvaluatorNoToolkit:

    def test_init_evaluator_no_toolkit(self, real_governance):
        """_init_evaluator() with agent_os not installed logs a warning; _toolkit_available stays False."""
        mod, mock_audit, mock_telemetry = real_governance
        cfg = _make_config()
        adapter = mod.GovernanceAdapter(cfg)

        # Call _init_evaluator — agent_os is None in sys.modules → ImportError
        # Must not raise any exception
        adapter._init_evaluator()

        assert adapter._toolkit_available is False
        assert adapter._evaluator is None


# ---------------------------------------------------------------------------
# Test 3: _init_evaluator — with toolkit
# ---------------------------------------------------------------------------


class TestInitEvaluatorWithToolkit:

    def test_init_evaluator_with_toolkit(self, monkeypatch):
        """_init_evaluator() with agent_os available sets _toolkit_available=True."""
        mock_audit = MagicMock()
        mock_audit.check_permission = MagicMock(return_value=True)
        mock_audit.log_event = MagicMock(return_value="trace-abc")
        mock_telemetry = MagicMock()
        mock_telemetry.get_current_traceparent = MagicMock(return_value="00-abc-def-01")

        mock_decision, mock_evaluator_instance, MockPolicyEvaluator, mock_agent_os_policies = (
            _make_toolkit_mocks()
        )
        monkeypatch.setitem(sys.modules, "agent_os", MagicMock())
        monkeypatch.setitem(sys.modules, "agent_os.policies", mock_agent_os_policies)

        mod = _load_governance_module(
            monkeypatch, mock_audit, mock_telemetry, with_agent_os=True
        )

        cfg = _make_config(policy_mode="strict")
        adapter = mod.GovernanceAdapter(cfg)
        adapter._init_evaluator()

        assert adapter._toolkit_available is True
        assert adapter._evaluator is mock_evaluator_instance


# ---------------------------------------------------------------------------
# Test 4: initialize() — no toolkit → RBAC-only warning
# ---------------------------------------------------------------------------


class TestInitializeRbacOnly:

    @pytest.mark.asyncio
    async def test_initialize_sets_toolkit_available_false(self, real_governance):
        """await adapter.initialize() with no toolkit logs 'RBAC-only mode' warning."""
        mod, mock_audit, mock_telemetry = real_governance
        cfg = _make_config()
        adapter = mod.GovernanceAdapter(cfg)

        import logging
        with patch_logger_warning(mod) as warn_calls:
            await adapter.initialize()

        assert adapter._toolkit_available is False
        # At least one warning about RBAC-only mode
        messages = [str(c) for c in warn_calls]
        assert any("RBAC" in m or "rbac" in m.lower() or "agent-os-kernel" in m for m in messages)


def patch_logger_warning(mod):
    """Context manager that collects logger.warning calls for the module's logger."""
    from unittest.mock import patch as _patch
    recorded = []
    original = mod.logger.warning

    class Collector:
        def __enter__(self):
            mod.logger.warning = lambda msg, *a, **kw: recorded.append(msg % a if a else msg)
            return recorded

        def __exit__(self, *exc):
            mod.logger.warning = original

    return Collector()


# ---------------------------------------------------------------------------
# Tests 5-11: check_permission scenarios
# ---------------------------------------------------------------------------


class TestCheckPermission:

    def test_check_permission_rbac_deny(self, real_governance):
        """audit.check_permission returns False → (False, 'RBAC denied ...')."""
        mod, mock_audit, mock_telemetry = real_governance
        mock_audit.check_permission.return_value = False

        cfg = _make_config()
        adapter = mod.GovernanceAdapter(cfg)

        allowed, reason = adapter.check_permission("memory.write", ["read-only"])
        assert allowed is False
        assert "RBAC denied" in reason
        assert "memory.write" in reason

    def test_check_permission_rbac_allow_no_toolkit(self, real_governance):
        """RBAC allows, toolkit unavailable → (True, 'rbac_allowed')."""
        mod, mock_audit, mock_telemetry = real_governance
        mock_audit.check_permission.return_value = True

        cfg = _make_config(policy_mode="strict")
        adapter = mod.GovernanceAdapter(cfg)
        adapter._toolkit_available = False

        allowed, reason = adapter.check_permission("memory.read", ["operator"])
        assert allowed is True
        assert reason == "rbac_allowed"

    def test_check_permission_audit_mode(self, real_governance):
        """RBAC allows, toolkit available but policy_mode='audit' → (True, 'rbac_allowed')."""
        mod, mock_audit, mock_telemetry = real_governance
        mock_audit.check_permission.return_value = True

        cfg = _make_config(policy_mode="audit")
        adapter = mod.GovernanceAdapter(cfg)
        # Even if we pretend toolkit is available, audit mode bypasses it
        adapter._toolkit_available = True
        mock_evaluator = MagicMock()
        adapter._evaluator = mock_evaluator

        allowed, reason = adapter.check_permission("memory.read", ["operator"])
        assert allowed is True
        assert reason == "rbac_allowed"
        # Evaluator should NOT be called in audit mode
        mock_evaluator.evaluate.assert_not_called()

    def test_check_permission_strict_mode_toolkit_deny(self, monkeypatch):
        """Toolkit denies in strict mode → (False, reason)."""
        mock_audit = MagicMock()
        mock_audit.check_permission = MagicMock(return_value=True)
        mock_audit.log_event = MagicMock(return_value="trace-abc")
        mock_telemetry = MagicMock()
        mock_telemetry.get_current_traceparent = MagicMock(return_value="00-abc-def-01")

        mock_decision, mock_evaluator_instance, MockPolicyEvaluator, mock_agent_os_policies = (
            _make_toolkit_mocks()
        )
        mock_decision.allowed = False
        mock_decision.reason = "policy_denied"

        monkeypatch.setitem(sys.modules, "agent_os", MagicMock())
        monkeypatch.setitem(sys.modules, "agent_os.policies", mock_agent_os_policies)

        mod = _load_governance_module(
            monkeypatch, mock_audit, mock_telemetry, with_agent_os=True
        )

        cfg = _make_config(policy_mode="strict")
        adapter = mod.GovernanceAdapter(cfg)
        adapter._init_evaluator()

        allowed, reason = adapter.check_permission("memory.write", ["operator"])
        assert allowed is False
        assert reason == "policy_denied"

    def test_check_permission_strict_mode_toolkit_allow(self, monkeypatch):
        """Toolkit allows in strict mode → (True, reason)."""
        mock_audit = MagicMock()
        mock_audit.check_permission = MagicMock(return_value=True)
        mock_audit.log_event = MagicMock(return_value="trace-abc")
        mock_telemetry = MagicMock()
        mock_telemetry.get_current_traceparent = MagicMock(return_value="00-abc-def-01")

        mock_decision, mock_evaluator_instance, MockPolicyEvaluator, mock_agent_os_policies = (
            _make_toolkit_mocks()
        )
        mock_decision.allowed = True
        mock_decision.reason = "policy_ok"

        monkeypatch.setitem(sys.modules, "agent_os", MagicMock())
        monkeypatch.setitem(sys.modules, "agent_os.policies", mock_agent_os_policies)

        mod = _load_governance_module(
            monkeypatch, mock_audit, mock_telemetry, with_agent_os=True
        )

        cfg = _make_config(policy_mode="strict")
        adapter = mod.GovernanceAdapter(cfg)
        adapter._init_evaluator()

        allowed, reason = adapter.check_permission("memory.read", ["operator"])
        assert allowed is True
        assert reason == "policy_ok"

    def test_check_permission_permissive_mode_toolkit_deny(self, monkeypatch):
        """Toolkit denies but permissive mode → (True, ...) logs warning."""
        mock_audit = MagicMock()
        mock_audit.check_permission = MagicMock(return_value=True)
        mock_audit.log_event = MagicMock(return_value="trace-abc")
        mock_telemetry = MagicMock()
        mock_telemetry.get_current_traceparent = MagicMock(return_value="00-abc-def-01")

        mock_decision, mock_evaluator_instance, MockPolicyEvaluator, mock_agent_os_policies = (
            _make_toolkit_mocks()
        )
        mock_decision.allowed = False
        mock_decision.reason = "advisory_deny"

        monkeypatch.setitem(sys.modules, "agent_os", MagicMock())
        monkeypatch.setitem(sys.modules, "agent_os.policies", mock_agent_os_policies)

        mod = _load_governance_module(
            monkeypatch, mock_audit, mock_telemetry, with_agent_os=True
        )

        cfg = _make_config(policy_mode="permissive")
        adapter = mod.GovernanceAdapter(cfg)
        adapter._init_evaluator()

        warnings_logged = []
        original_warn = mod.logger.warning
        mod.logger.warning = lambda msg, *a, **kw: warnings_logged.append(msg % a if a else msg)
        try:
            allowed, reason = adapter.check_permission("memory.write", ["operator"])
        finally:
            mod.logger.warning = original_warn

        # In permissive mode, toolkit denial is advisory — action is still allowed
        assert allowed is True
        # A warning was logged about the advisory denial
        assert any("permissive" in w or "advisory" in w or "denied" in w for w in warnings_logged)

    def test_check_permission_toolkit_exception(self, monkeypatch):
        """evaluator.evaluate raises exception → falls back to RBAC result."""
        mock_audit = MagicMock()
        mock_audit.check_permission = MagicMock(return_value=True)
        mock_audit.log_event = MagicMock(return_value="trace-abc")
        mock_telemetry = MagicMock()
        mock_telemetry.get_current_traceparent = MagicMock(return_value="00-abc-def-01")

        mock_decision, mock_evaluator_instance, MockPolicyEvaluator, mock_agent_os_policies = (
            _make_toolkit_mocks()
        )
        mock_evaluator_instance.evaluate.side_effect = RuntimeError("toolkit error")

        monkeypatch.setitem(sys.modules, "agent_os", MagicMock())
        monkeypatch.setitem(sys.modules, "agent_os.policies", mock_agent_os_policies)

        mod = _load_governance_module(
            monkeypatch, mock_audit, mock_telemetry, with_agent_os=True
        )

        cfg = _make_config(policy_mode="strict")
        adapter = mod.GovernanceAdapter(cfg)
        adapter._init_evaluator()

        # Should NOT raise; falls back to RBAC result
        allowed, reason = adapter.check_permission("memory.read", ["operator"])
        assert allowed is True  # RBAC allowed, exception fallback keeps RBAC result
        assert reason == "toolkit_evaluation_error"


# ---------------------------------------------------------------------------
# Tests 12-13: emit()
# ---------------------------------------------------------------------------


class TestEmit:

    def test_emit_calls_audit_log_event(self, real_governance):
        """emit() calls audit.log_event with governance_toolkit and traceparent."""
        mod, mock_audit, mock_telemetry = real_governance
        mock_audit.log_event.return_value = "trace-123"
        mock_telemetry.get_current_traceparent.return_value = "00-trace-parent-01"

        cfg = _make_config(toolkit="microsoft")
        adapter = mod.GovernanceAdapter(cfg)
        adapter._toolkit_available = True

        result = adapter.emit(
            event_type="permission_check",
            action="memory.write",
            resource="scope",
            outcome="allowed",
            actor="test-actor",
        )

        assert result == "trace-123"
        mock_audit.log_event.assert_called_once()
        call_kwargs = mock_audit.log_event.call_args
        # Check traceparent and governance_toolkit are passed
        kwargs = call_kwargs.kwargs if call_kwargs.kwargs else {}
        all_args = {**kwargs}
        # Also check positional → keyword mapping
        if call_kwargs.args:
            # log_event(event_type, action, resource, outcome, **kwargs)
            pass
        assert "governance_toolkit" in all_args or "microsoft" in str(call_kwargs)
        assert "traceparent" in all_args or "00-trace-parent-01" in str(call_kwargs)

    def test_emit_disabled_toolkit_label(self, real_governance):
        """When _toolkit_available=False, governance_toolkit='disabled'."""
        mod, mock_audit, mock_telemetry = real_governance
        mock_audit.log_event.return_value = "trace-456"

        cfg = _make_config(toolkit="microsoft")
        adapter = mod.GovernanceAdapter(cfg)
        adapter._toolkit_available = False  # explicitly disabled

        adapter.emit(
            event_type="permission_check",
            action="memory.read",
            resource="scope",
            outcome="allowed",
        )

        mock_audit.log_event.assert_called_once()
        call_args_str = str(mock_audit.log_event.call_args)
        assert "disabled" in call_args_str


# ---------------------------------------------------------------------------
# Tests 14-15: initialize_governance()
# ---------------------------------------------------------------------------


class TestInitializeGovernance:

    @pytest.mark.asyncio
    async def test_initialize_governance_success(self, real_governance):
        """initialize_governance() sets module _adapter singleton on success."""
        mod, mock_audit, mock_telemetry = real_governance
        assert mod._adapter is None

        cfg = _make_config()
        adapter = await mod.initialize_governance(cfg)

        assert adapter is not None
        assert mod._adapter is adapter
        assert isinstance(adapter, mod.GovernanceAdapter)

    @pytest.mark.asyncio
    async def test_initialize_governance_failure(self, real_governance):
        """initialize_governance() returns None and _adapter stays None on failure."""
        mod, mock_audit, mock_telemetry = real_governance
        assert mod._adapter is None

        cfg = _make_config()
        # Make GovernanceAdapter.initialize raise
        original_init = mod.GovernanceAdapter.initialize

        async def bad_initialize(self):
            raise RuntimeError("init failed")

        mod.GovernanceAdapter.initialize = bad_initialize
        try:
            result = await mod.initialize_governance(cfg)
        finally:
            mod.GovernanceAdapter.initialize = original_init

        assert result is None
        assert mod._adapter is None


# ---------------------------------------------------------------------------
# Test 16: get_governance_adapter()
# ---------------------------------------------------------------------------


class TestGetGovernanceAdapter:

    def test_get_governance_adapter_none_initially(self, real_governance):
        """get_governance_adapter() returns None when _adapter is not set."""
        mod, mock_audit, mock_telemetry = real_governance
        assert mod._adapter is None
        assert mod.get_governance_adapter() is None

    def test_get_governance_adapter_returns_set_adapter(self, real_governance):
        """get_governance_adapter() returns the _adapter after it is set."""
        mod, mock_audit, mock_telemetry = real_governance
        fake_adapter = MagicMock()
        mod._adapter = fake_adapter
        assert mod.get_governance_adapter() is fake_adapter


# ---------------------------------------------------------------------------
# Tests 17-18: check_permission_with_governance()
# ---------------------------------------------------------------------------


class TestCheckPermissionWithGovernance:

    def test_check_permission_with_governance_no_adapter(self, real_governance):
        """_adapter=None → falls through to audit.check_permission."""
        mod, mock_audit, mock_telemetry = real_governance
        mod._adapter = None
        mock_audit.check_permission.return_value = True

        allowed, reason = mod.check_permission_with_governance("memory.read", ["operator"])
        assert allowed is True
        assert reason == "rbac_only"
        mock_audit.check_permission.assert_called_once_with("memory.read", ["operator"], None)

    def test_check_permission_with_governance_with_adapter(self, real_governance):
        """_adapter set → calls adapter.check_permission."""
        mod, mock_audit, mock_telemetry = real_governance
        mock_adapter = MagicMock()
        mock_adapter.check_permission.return_value = (True, "adapter_allowed")
        mod._adapter = mock_adapter

        allowed, reason = mod.check_permission_with_governance(
            "memory.write", ["admin"], None, {"resource": "scope"}
        )
        assert allowed is True
        assert reason == "adapter_allowed"
        mock_adapter.check_permission.assert_called_once_with(
            "memory.write", ["admin"], None, {"resource": "scope"}
        )


# ---------------------------------------------------------------------------
# Tests 19-20: _emit_governance_event()
# ---------------------------------------------------------------------------


class TestEmitGovernanceEvent:

    def test_emit_governance_event_no_adapter(self, real_governance):
        """_adapter=None → _emit_governance_event returns None."""
        mod, mock_audit, mock_telemetry = real_governance
        mod._adapter = None
        result = mod._emit_governance_event(
            event_type="permission_check",
            action="memory.read",
            resource="scope",
            outcome="allowed",
        )
        assert result is None

    def test_emit_governance_event_with_adapter(self, real_governance):
        """_adapter set → calls adapter.emit and returns its result."""
        mod, mock_audit, mock_telemetry = real_governance
        mock_adapter = MagicMock()
        mock_adapter.emit.return_value = "trace-emit-xyz"
        mod._adapter = mock_adapter

        result = mod._emit_governance_event(
            event_type="permission_check",
            action="memory.write",
            resource="scope",
            outcome="denied",
            actor="test-actor",
            trace_id="explicit-trace",
            extra_key="extra_val",
        )
        assert result == "trace-emit-xyz"
        mock_adapter.emit.assert_called_once_with(
            "permission_check",
            "memory.write",
            "scope",
            "denied",
            actor="test-actor",
            trace_id="explicit-trace",
            extra_key="extra_val",
        )


# ---------------------------------------------------------------------------
# Tests for policy_file loading (exercises _init_evaluator branches)
# ---------------------------------------------------------------------------


class TestInitEvaluatorPolicyFile:

    def _setup_with_toolkit(self, monkeypatch):
        mock_audit = MagicMock()
        mock_audit.check_permission = MagicMock(return_value=True)
        mock_audit.log_event = MagicMock(return_value="trace-abc")
        mock_telemetry = MagicMock()
        mock_telemetry.get_current_traceparent = MagicMock(return_value="00-abc-def-01")

        mock_decision, mock_evaluator_instance, MockPolicyEvaluator, mock_agent_os_policies = (
            _make_toolkit_mocks()
        )
        monkeypatch.setitem(sys.modules, "agent_os", MagicMock())
        monkeypatch.setitem(sys.modules, "agent_os.policies", mock_agent_os_policies)

        mod = _load_governance_module(
            monkeypatch, mock_audit, mock_telemetry, with_agent_os=True
        )
        return mod, mock_evaluator_instance, MockPolicyEvaluator

    def test_policy_file_rego_loaded(self, monkeypatch, tmp_path):
        """When policy_file is a .rego file that exists, evaluator.load_rego is called."""
        mod, mock_evaluator_instance, MockPolicyEvaluator = self._setup_with_toolkit(monkeypatch)

        policy_path = tmp_path / "policy.rego"
        policy_path.write_text("package main\ndefault allow = false\n")

        cfg = _make_config(policy_mode="strict", policy_file=str(policy_path))
        adapter = mod.GovernanceAdapter(cfg)
        adapter._init_evaluator()

        assert adapter._toolkit_available is True
        mock_evaluator_instance.load_rego.assert_called_once_with(path=str(policy_path))

    def test_policy_file_nonexistent_logs_warning(self, monkeypatch, tmp_path):
        """Non-existent policy_file logs a warning but does not crash."""
        mod, mock_evaluator_instance, MockPolicyEvaluator = self._setup_with_toolkit(monkeypatch)

        cfg = _make_config(
            policy_mode="strict",
            policy_file=str(tmp_path / "missing.rego"),
        )
        adapter = mod.GovernanceAdapter(cfg)

        warnings = []
        original_warn = mod.logger.warning
        mod.logger.warning = lambda msg, *a, **kw: warnings.append(msg % a if a else msg)
        try:
            adapter._init_evaluator()
        finally:
            mod.logger.warning = original_warn

        # Toolkit still initialised (file load skipped, not a hard failure)
        assert adapter._toolkit_available is True
        assert any("does not exist" in w or "skipping" in w for w in warnings)
        mock_evaluator_instance.load_rego.assert_not_called()

    def test_policy_file_unknown_extension_logs_warning(self, monkeypatch, tmp_path):
        """Unknown policy file extension logs a warning and skips load."""
        mod, mock_evaluator_instance, MockPolicyEvaluator = self._setup_with_toolkit(monkeypatch)

        policy_path = tmp_path / "policy.unknown"
        policy_path.write_text("not a real policy format")

        cfg = _make_config(policy_mode="strict", policy_file=str(policy_path))
        adapter = mod.GovernanceAdapter(cfg)

        warnings = []
        original_warn = mod.logger.warning
        mod.logger.warning = lambda msg, *a, **kw: warnings.append(msg % a if a else msg)
        try:
            adapter._init_evaluator()
        finally:
            mod.logger.warning = original_warn

        assert adapter._toolkit_available is True
        assert any("Unrecognised" in w or "extension" in w for w in warnings)


# ---------------------------------------------------------------------------
# Gap 1: New targeted coverage tests
# ---------------------------------------------------------------------------


class TestGap1InitializeToolkitAvailable:

    @pytest.mark.asyncio
    async def test_initialize_logs_info_when_toolkit_available(self, monkeypatch):
        """Line 72-75: initialize() logs info (not warning) when _toolkit_available=True."""
        mock_audit = MagicMock()
        mock_audit.check_permission = MagicMock(return_value=True)
        mock_audit.log_event = MagicMock(return_value="trace-abc")
        mock_telemetry = MagicMock()
        mock_telemetry.get_current_traceparent = MagicMock(return_value="00-abc-def-01")

        mock_decision, mock_evaluator_instance, MockPolicyEvaluator, mock_agent_os_policies = (
            _make_toolkit_mocks()
        )
        monkeypatch.setitem(sys.modules, "agent_os", MagicMock())
        monkeypatch.setitem(sys.modules, "agent_os.policies", mock_agent_os_policies)

        mod = _load_governance_module(
            monkeypatch, mock_audit, mock_telemetry, with_agent_os=True
        )

        cfg = _make_config(policy_mode="strict")
        adapter = mod.GovernanceAdapter(cfg)

        info_messages = []
        original_info = mod.logger.info
        mod.logger.info = lambda msg, *a, **kw: info_messages.append(msg % a if a else msg)
        try:
            await adapter.initialize()
        finally:
            mod.logger.info = original_info

        assert adapter._toolkit_available is True
        assert any("GovernanceAdapter initialised" in m or "toolkit=" in m for m in info_messages)


class TestGap1PolicyEndpoint:

    def test_policy_endpoint_added_to_kwargs(self, monkeypatch):
        """Line 107: policy_endpoint non-empty → kwargs['endpoint'] set."""
        mock_audit = MagicMock()
        mock_audit.check_permission = MagicMock(return_value=True)
        mock_audit.log_event = MagicMock(return_value="trace-abc")
        mock_telemetry = MagicMock()
        mock_telemetry.get_current_traceparent = MagicMock(return_value="00-abc-def-01")

        mock_decision, mock_evaluator_instance, MockPolicyEvaluator, mock_agent_os_policies = (
            _make_toolkit_mocks()
        )
        monkeypatch.setitem(sys.modules, "agent_os", MagicMock())
        monkeypatch.setitem(sys.modules, "agent_os.policies", mock_agent_os_policies)

        mod = _load_governance_module(
            monkeypatch, mock_audit, mock_telemetry, with_agent_os=True
        )

        cfg = _make_config(policy_mode="strict", policy_endpoint="https://policy.example.com/v1")
        adapter = mod.GovernanceAdapter(cfg)
        adapter._init_evaluator()

        assert adapter._toolkit_available is True
        call_kwargs = MockPolicyEvaluator.call_args.kwargs
        assert call_kwargs.get("endpoint") == "https://policy.example.com/v1"


class TestGap1PolicyFileYamlCedar:

    def _setup_with_toolkit(self, monkeypatch):
        mock_audit = MagicMock()
        mock_audit.check_permission = MagicMock(return_value=True)
        mock_audit.log_event = MagicMock(return_value="trace-abc")
        mock_telemetry = MagicMock()
        mock_telemetry.get_current_traceparent = MagicMock(return_value="00-abc-def-01")

        mock_decision, mock_evaluator_instance, MockPolicyEvaluator, mock_agent_os_policies = (
            _make_toolkit_mocks()
        )
        monkeypatch.setitem(sys.modules, "agent_os", MagicMock())
        monkeypatch.setitem(sys.modules, "agent_os.policies", mock_agent_os_policies)

        mod = _load_governance_module(
            monkeypatch, mock_audit, mock_telemetry, with_agent_os=True
        )
        return mod, mock_evaluator_instance

    def test_policy_file_yaml_loaded(self, monkeypatch, tmp_path):
        """Lines 120-121: .yaml policy file → evaluator.load_yaml called."""
        mod, mock_evaluator_instance = self._setup_with_toolkit(monkeypatch)

        policy_path = tmp_path / "policy.yaml"
        policy_path.write_text("version: 1\n")

        cfg = _make_config(policy_mode="strict", policy_file=str(policy_path))
        adapter = mod.GovernanceAdapter(cfg)
        adapter._init_evaluator()

        assert adapter._toolkit_available is True
        mock_evaluator_instance.load_yaml.assert_called_once_with(path=str(policy_path))

    def test_policy_file_yml_loaded(self, monkeypatch, tmp_path):
        """Lines 120-121: .yml extension also calls load_yaml."""
        mod, mock_evaluator_instance = self._setup_with_toolkit(monkeypatch)

        policy_path = tmp_path / "policy.yml"
        policy_path.write_text("version: 1\n")

        cfg = _make_config(policy_mode="strict", policy_file=str(policy_path))
        adapter = mod.GovernanceAdapter(cfg)
        adapter._init_evaluator()

        assert adapter._toolkit_available is True
        mock_evaluator_instance.load_yaml.assert_called_once_with(path=str(policy_path))

    def test_policy_file_cedar_loaded(self, monkeypatch, tmp_path):
        """Lines 123-124: .cedar policy file → evaluator.load_cedar called."""
        mod, mock_evaluator_instance = self._setup_with_toolkit(monkeypatch)

        policy_path = tmp_path / "policy.cedar"
        policy_path.write_text("permit(principal, action, resource);\n")

        cfg = _make_config(policy_mode="strict", policy_file=str(policy_path))
        adapter = mod.GovernanceAdapter(cfg)
        adapter._init_evaluator()

        assert adapter._toolkit_available is True
        mock_evaluator_instance.load_cedar.assert_called_once_with(path=str(policy_path))


class TestGap1InitEvaluatorGenericException:

    def test_init_evaluator_non_import_error_swallowed(self, monkeypatch):
        """Lines 142-143: PolicyEvaluator() itself raises non-ImportError → logged, toolkit_available=False."""
        mock_audit = MagicMock()
        mock_audit.check_permission = MagicMock(return_value=True)
        mock_audit.log_event = MagicMock(return_value="trace-abc")
        mock_telemetry = MagicMock()
        mock_telemetry.get_current_traceparent = MagicMock(return_value="00-abc-def-01")

        # PolicyEvaluator() raises RuntimeError (not ImportError)
        MockPolicyEvaluator = MagicMock(side_effect=RuntimeError("toolkit init failed"))
        mock_agent_os_policies = MagicMock()
        mock_agent_os_policies.PolicyEvaluator = MockPolicyEvaluator

        monkeypatch.setitem(sys.modules, "agent_os", MagicMock())
        monkeypatch.setitem(sys.modules, "agent_os.policies", mock_agent_os_policies)

        mod = _load_governance_module(
            monkeypatch, mock_audit, mock_telemetry, with_agent_os=True
        )

        cfg = _make_config(policy_mode="strict")
        adapter = mod.GovernanceAdapter(cfg)

        warnings = []
        original_warn = mod.logger.warning
        mod.logger.warning = lambda msg, *a, **kw: warnings.append(msg % a if a else msg)
        try:
            adapter._init_evaluator()
        finally:
            mod.logger.warning = original_warn

        assert adapter._toolkit_available is False
        assert adapter._evaluator is None
        assert any("Failed" in w or "toolkit init failed" in w for w in warnings)


class TestGap1ExtraContextKeys:

    def test_check_permission_extra_context_keys_merged(self, monkeypatch):
        """Lines 206-207: extra context keys beyond base eval_context are merged in."""
        mock_audit = MagicMock()
        mock_audit.check_permission = MagicMock(return_value=True)
        mock_audit.log_event = MagicMock(return_value="trace-abc")
        mock_telemetry = MagicMock()
        mock_telemetry.get_current_traceparent = MagicMock(return_value="00-abc-def-01")

        mock_decision, mock_evaluator_instance, MockPolicyEvaluator, mock_agent_os_policies = (
            _make_toolkit_mocks()
        )
        mock_decision.allowed = True
        mock_decision.reason = "policy_ok"

        monkeypatch.setitem(sys.modules, "agent_os", MagicMock())
        monkeypatch.setitem(sys.modules, "agent_os.policies", mock_agent_os_policies)

        mod = _load_governance_module(
            monkeypatch, mock_audit, mock_telemetry, with_agent_os=True
        )

        cfg = _make_config(policy_mode="strict")
        adapter = mod.GovernanceAdapter(cfg)
        adapter._init_evaluator()

        # Pass context with extra_key not in the base eval_context dict
        context = {"resource": "my-resource", "actor": "user-1", "extra_key": "extra_value"}
        allowed, reason = adapter.check_permission("memory.read", ["operator"], context=context)

        assert allowed is True
        # Verify evaluator.evaluate was called with eval_context containing extra_key
        call_args = mock_evaluator_instance.evaluate.call_args
        eval_ctx = call_args.args[0] if call_args.args else call_args.kwargs.get("eval_context", {})
        assert eval_ctx.get("extra_key") == "extra_value"
