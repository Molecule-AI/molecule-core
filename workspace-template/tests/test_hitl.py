"""Tests for the HITL (Human-In-The-Loop) workflow primitives.

Covers:
- _TaskPauseRegistry: register/resume/timeout/list_paused
- pause_task / resume_task tools: success, timeout, not-found
- @requires_approval decorator: approval granted, denied, RBAC bypass
- HITLConfig loading from workspace config
- Notification helpers: Slack URL construction, email config validation
"""

import asyncio
import importlib.util
import sys
from pathlib import Path
from types import ModuleType
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

ROOT = Path(__file__).resolve().parents[1]


# ---------------------------------------------------------------------------
# Module loader (isolated from conftest mocks)
# ---------------------------------------------------------------------------

def _load_hitl(monkeypatch):
    """Load tools/hitl.py in a fresh namespace with controlled dependencies."""
    # Ensure langchain_core.tools.tool is a no-op decorator
    if "langchain_core" not in sys.modules:
        lc = ModuleType("langchain_core")
        lc_tools = ModuleType("langchain_core.tools")
        lc_tools.tool = lambda f: f
        monkeypatch.setitem(sys.modules, "langchain_core", lc)
        monkeypatch.setitem(sys.modules, "langchain_core.tools", lc_tools)
    else:
        monkeypatch.setattr(sys.modules["langchain_core.tools"], "tool", lambda f: f, raising=False)

    # Stub heavy deps the module imports at top level
    httpx_stub = ModuleType("httpx")
    httpx_stub.AsyncClient = MagicMock()
    monkeypatch.setitem(sys.modules, "httpx", httpx_stub)

    monkeypatch.setenv("PLATFORM_URL", "http://platform.test")
    monkeypatch.setenv("WORKSPACE_ID", "ws-test")

    monkeypatch.setitem(sys.modules, "builtin_tools.audit", MagicMock(
        log_event=MagicMock(return_value="trace-id"),
        check_permission=MagicMock(return_value=True),
        get_workspace_roles=MagicMock(return_value=(["operator"], {})),
    ))
    monkeypatch.setitem(sys.modules, "builtin_tools.approval", MagicMock(
        request_approval=MagicMock(ainvoke=AsyncMock(return_value={"approved": True, "approval_id": "appr-1"})),
    ))

    # Remove any cached hitl module
    monkeypatch.setitem(sys.modules, "builtin_tools.hitl", None)  # force reload
    sys.modules.pop("builtin_tools.hitl", None)

    spec = importlib.util.spec_from_file_location(
        "builtin_tools.hitl", ROOT / "builtin_tools" / "hitl.py"
    )
    mod = importlib.util.module_from_spec(spec)
    monkeypatch.setitem(sys.modules, "builtin_tools.hitl", mod)
    spec.loader.exec_module(mod)
    return mod


# ============================================================================
# _TaskPauseRegistry
# ============================================================================

class TestPauseRegistry:

    def test_register_creates_event(self, monkeypatch):
        mod = _load_hitl(monkeypatch)
        reg = mod._TaskPauseRegistry()
        ev = reg.register("task-1")
        assert not ev.is_set()

    def test_resume_sets_event(self, monkeypatch):
        mod = _load_hitl(monkeypatch)
        reg = mod._TaskPauseRegistry()
        reg.register("task-2")
        result = reg.resume("task-2", {"note": "approved"})
        assert result is True

    def test_resume_unknown_returns_false(self, monkeypatch):
        mod = _load_hitl(monkeypatch)
        reg = mod._TaskPauseRegistry()
        assert reg.resume("nonexistent", {}) is False

    def test_pop_result_returns_stored_payload(self, monkeypatch):
        mod = _load_hitl(monkeypatch)
        reg = mod._TaskPauseRegistry()
        reg.register("task-3")
        reg.resume("task-3", {"data": "hello"})
        r = reg.pop_result("task-3")
        assert r == {"data": "hello"}

    def test_pop_result_missing_returns_empty(self, monkeypatch):
        mod = _load_hitl(monkeypatch)
        reg = mod._TaskPauseRegistry()
        assert reg.pop_result("no-such-task") == {}

    def test_list_paused_only_unset(self, monkeypatch):
        mod = _load_hitl(monkeypatch)
        reg = mod._TaskPauseRegistry()
        reg.register("t-paused")
        reg.register("t-resumed")
        reg.resume("t-resumed", {})
        assert "t-paused" in reg.list_paused()
        assert "t-resumed" not in reg.list_paused()

    def test_cleanup_removes_entries(self, monkeypatch):
        mod = _load_hitl(monkeypatch)
        reg = mod._TaskPauseRegistry()
        reg.register("t-clean")
        reg.cleanup("t-clean")
        assert "t-clean" not in reg.list_paused()
        assert reg.pop_result("t-clean") == {}


# ============================================================================
# pause_task / resume_task tools
# ============================================================================

class TestPauseResumeTool:

    @pytest.mark.asyncio
    async def test_pause_resumes_on_signal(self, monkeypatch):
        mod = _load_hitl(monkeypatch)
        # Override the global registry with a fresh one
        reg = mod._TaskPauseRegistry()
        monkeypatch.setattr(mod, "pause_registry", reg)

        # Schedule a resume signal 50 ms after pause starts
        async def _schedule_resume():
            await asyncio.sleep(0.05)
            reg.resume("task-a", {"note": "human approved"})

        asyncio.create_task(_schedule_resume())

        result = await mod.pause_task("task-a", "waiting for review")

        assert result["resumed"] is True
        assert result["task_id"] == "task-a"

    @pytest.mark.asyncio
    async def test_pause_times_out(self, monkeypatch):
        mod = _load_hitl(monkeypatch)
        reg = mod._TaskPauseRegistry()
        monkeypatch.setattr(mod, "pause_registry", reg)
        # Set a very short timeout via the HITL config
        monkeypatch.setattr(mod, "_load_hitl_config",
                            lambda: mod.HITLConfig(default_timeout=0.05))

        result = await mod.pause_task("task-timeout", "will timeout")

        assert result["resumed"] is False
        assert "error" in result

    @pytest.mark.asyncio
    async def test_resume_task_success(self, monkeypatch):
        mod = _load_hitl(monkeypatch)
        reg = mod._TaskPauseRegistry()
        monkeypatch.setattr(mod, "pause_registry", reg)
        reg.register("task-r")

        result = await mod.resume_task("task-r", "looks good")

        assert result["success"] is True
        assert result["task_id"] == "task-r"

    @pytest.mark.asyncio
    async def test_resume_task_not_found(self, monkeypatch):
        mod = _load_hitl(monkeypatch)
        reg = mod._TaskPauseRegistry()
        monkeypatch.setattr(mod, "pause_registry", reg)

        result = await mod.resume_task("does-not-exist", "")

        assert result["success"] is False
        assert "error" in result

    @pytest.mark.asyncio
    async def test_list_paused_tasks_empty(self, monkeypatch):
        mod = _load_hitl(monkeypatch)
        reg = mod._TaskPauseRegistry()
        monkeypatch.setattr(mod, "pause_registry", reg)

        result = await mod.list_paused_tasks()

        assert result["count"] == 0
        assert result["paused_tasks"] == []

    @pytest.mark.asyncio
    async def test_list_paused_tasks_shows_registered(self, monkeypatch):
        mod = _load_hitl(monkeypatch)
        reg = mod._TaskPauseRegistry()
        monkeypatch.setattr(mod, "pause_registry", reg)
        reg.register("t-show")

        result = await mod.list_paused_tasks()

        assert result["count"] == 1
        assert "t-show" in result["paused_tasks"]


# ============================================================================
# @requires_approval decorator
# ============================================================================

class TestRequiresApproval:

    @pytest.mark.asyncio
    async def test_executes_when_approved(self, monkeypatch):
        mod = _load_hitl(monkeypatch)

        approval_mock = MagicMock()
        approval_mock.ainvoke = AsyncMock(return_value={
            "approved": True, "approval_id": "appr-ok"
        })
        monkeypatch.setitem(
            sys.modules, "builtin_tools.approval",
            MagicMock(request_approval=approval_mock)
        )

        executed = []

        @mod.requires_approval("Run migration")
        async def run_migration(table: str):
            executed.append(table)
            return {"done": True}

        result = await run_migration(table="users")

        assert result == {"done": True}
        assert executed == ["users"]

    @pytest.mark.asyncio
    async def test_blocks_when_denied(self, monkeypatch):
        mod = _load_hitl(monkeypatch)

        approval_mock = MagicMock()
        approval_mock.ainvoke = AsyncMock(return_value={
            "approved": False, "approval_id": "appr-no", "message": "Denied by human"
        })
        monkeypatch.setitem(
            sys.modules, "builtin_tools.approval",
            MagicMock(request_approval=approval_mock)
        )

        executed = []

        @mod.requires_approval("Drop table")
        async def drop_table(table: str):
            executed.append(table)
            return {"done": True}

        result = await drop_table(table="orders")

        assert result["success"] is False
        assert "not approved" in result["error"].lower() or "approved" in result["error"].lower()
        assert executed == []  # Never ran

    @pytest.mark.asyncio
    async def test_bypasses_for_admin_role(self, monkeypatch):
        mod = _load_hitl(monkeypatch)

        # Mock RBAC: workspace has 'admin' role
        audit_mock = MagicMock()
        audit_mock.get_workspace_roles = MagicMock(return_value=(["admin"], {}))
        audit_mock.check_permission = MagicMock(return_value=True)
        audit_mock.log_event = MagicMock(return_value="tid")
        monkeypatch.setitem(sys.modules, "builtin_tools.audit", audit_mock)

        approval_called = []

        approval_mock = MagicMock()
        approval_mock.ainvoke = AsyncMock(side_effect=lambda _: approval_called.append(1) or {"approved": True})
        monkeypatch.setitem(sys.modules, "builtin_tools.approval",
                            MagicMock(request_approval=approval_mock))

        @mod.requires_approval("Danger", bypass_roles=["admin"])
        async def dangerous_op():
            return {"ran": True}

        result = await dangerous_op()

        assert result == {"ran": True}
        assert len(approval_called) == 0  # approval was bypassed

    @pytest.mark.asyncio
    async def test_reason_template_interpolation(self, monkeypatch):
        mod = _load_hitl(monkeypatch)

        captured_reason = []
        async def fake_ainvoke(args):
            captured_reason.append(args["reason"])
            return {"approved": True}

        approval_mock = MagicMock()
        approval_mock.ainvoke = fake_ainvoke
        monkeypatch.setitem(sys.modules, "builtin_tools.approval",
                            MagicMock(request_approval=approval_mock))

        @mod.requires_approval("Delete record",
                               reason_template="Deleting record {record_id} from {table}")
        async def delete_record(record_id: str, table: str):
            return {"deleted": True}

        await delete_record(record_id="42", table="users")

        assert captured_reason == ["Deleting record 42 from users"]

    @pytest.mark.asyncio
    async def test_handles_approval_tool_exception(self, monkeypatch):
        mod = _load_hitl(monkeypatch)

        approval_mock = MagicMock()
        approval_mock.ainvoke = AsyncMock(side_effect=ConnectionError("platform down"))
        monkeypatch.setitem(sys.modules, "builtin_tools.approval",
                            MagicMock(request_approval=approval_mock))

        @mod.requires_approval("Risky op")
        async def risky():
            return {"done": True}

        result = await risky()

        assert result["success"] is False
        assert "error" in result


# ============================================================================
# HITLConfig loading
# ============================================================================

class TestHITLConfig:

    def test_defaults_when_config_unavailable(self, monkeypatch):
        mod = _load_hitl(monkeypatch)
        monkeypatch.setitem(sys.modules, "config",
                            MagicMock(load_config=MagicMock(side_effect=FileNotFoundError)))
        cfg = mod._load_hitl_config()
        assert cfg.default_timeout == 300.0
        assert cfg.bypass_roles == []
        assert any(c.get("type") == "dashboard" for c in cfg.channels)

    def test_loads_from_workspace_config(self, monkeypatch):
        mod = _load_hitl(monkeypatch)

        fake_hitl = mod.HITLConfig(
            channels=[{"type": "slack", "webhook_url": "https://slack.example.com"}],
            default_timeout=120.0,
            bypass_roles=["admin", "superuser"],
        )
        fake_ws_cfg = MagicMock()
        fake_ws_cfg.hitl = fake_hitl

        monkeypatch.setitem(sys.modules, "config",
                            MagicMock(load_config=MagicMock(return_value=fake_ws_cfg)))

        cfg = mod._load_hitl_config()

        assert cfg.default_timeout == 120.0
        assert "admin" in cfg.bypass_roles
        assert cfg.channels[0]["type"] == "slack"


# ============================================================================
# Notification channel helpers
# ============================================================================

class TestNotificationChannels:

    @pytest.mark.asyncio
    async def test_slack_skipped_without_webhook_url(self, monkeypatch):
        mod = _load_hitl(monkeypatch)
        # Should not raise, and should log a warning
        await mod._notify_slack({}, "action", "reason", "appr-1",
                                 "http://platform.test", "ws-test")

    @pytest.mark.asyncio
    async def test_email_skipped_with_missing_config(self, monkeypatch):
        mod = _load_hitl(monkeypatch)
        # Missing smtp_host/from/to — should return without raising
        await mod._notify_email({}, "action", "reason", "appr-1",
                                 "http://platform.test", "ws-test")

    @pytest.mark.asyncio
    async def test_slack_posts_to_webhook(self, monkeypatch):
        mod = _load_hitl(monkeypatch)

        posted = []

        class FakeAsyncClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def post(self, url, json):
                posted.append({"url": url, "payload": json})

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeAsyncClient)

        await mod._notify_slack(
            {"webhook_url": "https://hooks.slack.test/abc"},
            "Delete bucket",
            "Spring cleanup",
            "appr-slack-1",
            "http://platform.test",
            "ws-test",
        )

        assert len(posted) == 1
        assert posted[0]["url"] == "https://hooks.slack.test/abc"
        payload = posted[0]["payload"]
        assert "Delete bucket" in str(payload)
        assert "appr-slack-1" in str(payload)

    @pytest.mark.asyncio
    async def test_notify_channels_ignores_channel_errors(self, monkeypatch):
        mod = _load_hitl(monkeypatch)

        cfg = mod.HITLConfig(channels=[
            {"type": "slack", "webhook_url": "https://hooks.bad.test/fail"},
            {"type": "dashboard"},
        ])

        # Make the slack post raise
        class FailingClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def post(self, url, json): raise ConnectionError("webhook down")

        monkeypatch.setattr(mod.httpx, "AsyncClient", FailingClient)

        # Should not raise — channel errors are swallowed
        await mod._notify_channels("test action", "reason", "appr-x", cfg)

    @pytest.mark.asyncio
    async def test_notify_email_success(self, monkeypatch):
        """_notify_email sends email via SMTP when config is complete."""
        mod = _load_hitl(monkeypatch)

        smtp_calls = []

        class FakeSMTP:
            def __init__(self, host, port):
                smtp_calls.append({"host": host, "port": port})
                self.sent = []

            def __enter__(self):
                return self

            def __exit__(self, *a):
                pass

            def ehlo(self): pass
            def starttls(self): pass

            def login(self, user, pw):
                smtp_calls[-1]["login"] = (user, pw)

            def send_message(self, msg):
                smtp_calls[-1]["msg"] = msg

        async def fake_to_thread(fn, *args, **kwargs):
            fn()

        monkeypatch.setattr(mod.smtplib, "SMTP", FakeSMTP)
        monkeypatch.setattr(mod.asyncio, "to_thread", fake_to_thread)

        cfg = {
            "smtp_host": "smtp.example.com",
            "smtp_port": "587",
            "from": "from@example.com",
            "to": "to@example.com",
            "username": "user@example.com",
            "password": "secret",
        }

        await mod._notify_email(
            cfg, "Deploy prod", "scheduled maintenance", "appr-email-1",
            "http://platform.test", "ws-test",
        )

        assert len(smtp_calls) == 1
        assert smtp_calls[0]["host"] == "smtp.example.com"
        assert smtp_calls[0]["login"] == ("user@example.com", "secret")
        msg = smtp_calls[0]["msg"]
        # The body may be base64-encoded; decode it to check content
        body = msg.get_payload(decode=True).decode("utf-8")
        assert "appr-email-1" in body

    @pytest.mark.asyncio
    async def test_notify_email_missing_config(self, monkeypatch):
        """_notify_email with missing smtp_host logs warning and returns without error."""
        mod = _load_hitl(monkeypatch)

        smtp_called = []

        class FakeSMTP:
            def __init__(self, *a, **kw): smtp_called.append(True)
            def __enter__(self): return self
            def __exit__(self, *a): pass

        monkeypatch.setattr(mod.smtplib, "SMTP", FakeSMTP)

        # Missing smtp_host
        await mod._notify_email(
            {"from": "f@ex.com", "to": "t@ex.com"},
            "action", "reason", "appr-x",
            "http://platform.test", "ws-test",
        )

        assert smtp_called == [], "SMTP should not have been called with missing config"

    @pytest.mark.asyncio
    async def test_notify_channels_email_channel_error_is_swallowed(self, monkeypatch):
        """Exception in email channel notification is caught and logged, not re-raised."""
        mod = _load_hitl(monkeypatch)

        cfg = mod.HITLConfig(channels=[
            {
                "type": "email",
                "smtp_host": "smtp.example.com",
                "from": "a@b.com",
                "to": "c@d.com",
            },
        ])

        async def fake_to_thread(fn, *args, **kwargs):
            raise ConnectionRefusedError("SMTP server down")

        monkeypatch.setattr(mod.asyncio, "to_thread", fake_to_thread)

        # Should NOT raise — email errors are swallowed like slack errors
        await mod._notify_channels("action", "reason", "appr-y", cfg)


# ============================================================================
# HITLConfig — attribute-less raw object (line 77)
# ============================================================================

class TestHITLConfigEdgeCases:

    def test_defaults_when_raw_has_no_channels_attribute(self, monkeypatch):
        """When raw.channels attribute check fails, HITLConfig() defaults are used."""
        mod = _load_hitl(monkeypatch)

        # Return a raw config object whose .hitl attribute has NO .channels attr
        raw_hitl = MagicMock(spec=[])  # spec=[] means NO attributes at all
        fake_ws_cfg = MagicMock()
        fake_ws_cfg.hitl = raw_hitl

        monkeypatch.setitem(
            sys.modules, "config",
            MagicMock(load_config=MagicMock(return_value=fake_ws_cfg))
        )

        cfg = mod._load_hitl_config()

        # Should fall back to defaults safely
        assert cfg.default_timeout == 300.0
        assert cfg.channels == [{"type": "dashboard"}]
        assert cfg.bypass_roles == []


# ============================================================================
# @requires_approval — RBAC bypass exception path
# ============================================================================

class TestRequiresApprovalEdgeCases:

    @pytest.mark.asyncio
    async def test_rbac_bypass_check_exception_proceeds_to_gate(self, monkeypatch):
        """If get_workspace_roles raises, the decorator falls through to the approval gate."""
        mod = _load_hitl(monkeypatch)

        audit_mock = MagicMock()
        audit_mock.get_workspace_roles = MagicMock(side_effect=RuntimeError("rbac unavailable"))
        audit_mock.check_permission = MagicMock(return_value=True)
        audit_mock.log_event = MagicMock(return_value="tid")
        monkeypatch.setitem(sys.modules, "builtin_tools.audit", audit_mock)

        approval_mock = MagicMock()
        approval_mock.ainvoke = AsyncMock(return_value={"approved": True, "approval_id": "a1"})
        monkeypatch.setitem(
            sys.modules, "builtin_tools.approval",
            MagicMock(request_approval=approval_mock),
        )

        @mod.requires_approval("Risky action", bypass_roles=["admin"])
        async def risky_op():
            return {"ran": True}

        # Even though RBAC check raised, approval gate is invoked and fn executes
        result = await risky_op()

        assert result == {"ran": True}
        approval_mock.ainvoke.assert_called_once()


# ============================================================================
# pause_task / resume_task — audit import error paths
# ============================================================================

class TestAuditImportErrors:

    @pytest.mark.asyncio
    async def test_pause_task_audit_import_error(self, monkeypatch):
        """pause_task still completes even if tools.audit import raises."""
        mod = _load_hitl(monkeypatch)

        # Make tools.audit unavailable so the import inside pause_task fails
        monkeypatch.setitem(sys.modules, "builtin_tools.audit", None)

        reg = mod._TaskPauseRegistry()
        monkeypatch.setattr(mod, "pause_registry", reg)

        # Schedule resume quickly so we don't actually wait long
        async def _schedule_resume():
            await asyncio.sleep(0.05)
            reg.resume("audit-err-task", {"ok": True})

        asyncio.create_task(_schedule_resume())

        result = await mod.pause_task("audit-err-task", "audit missing")

        assert result["resumed"] is True
        assert result["task_id"] == "audit-err-task"

    @pytest.mark.asyncio
    async def test_resume_task_audit_import_error(self, monkeypatch):
        """resume_task still works even if tools.audit import raises."""
        mod = _load_hitl(monkeypatch)

        monkeypatch.setitem(sys.modules, "builtin_tools.audit", None)

        reg = mod._TaskPauseRegistry()
        monkeypatch.setattr(mod, "pause_registry", reg)
        reg.register("audit-err-resume")

        result = await mod.resume_task("audit-err-resume", "all good")

        assert result["success"] is True
        assert result["task_id"] == "audit-err-resume"


# ============================================================================
# @requires_approval — reason_template KeyError / IndexError (line 334-335)
# ============================================================================

class TestRequiresApprovalReasonTemplate:

    @pytest.mark.asyncio
    async def test_requires_approval_reason_template_format_keyerror(self, monkeypatch):
        """If reason_template.format(**kwargs) raises KeyError, use raw template."""
        mod = _load_hitl(monkeypatch)

        captured_reason = []

        async def fake_ainvoke(args):
            captured_reason.append(args["reason"])
            return {"approved": True}

        approval_mock = MagicMock()
        approval_mock.ainvoke = fake_ainvoke
        monkeypatch.setitem(sys.modules, "builtin_tools.approval",
                            MagicMock(request_approval=approval_mock))

        # reason_template references {nonexistent_field} which is not in kwargs
        @mod.requires_approval("Delete record",
                               reason_template="Delete {nonexistent_field} from table")
        async def delete_record(record_id: str):
            return {"deleted": True}

        result = await delete_record(record_id="42")

        assert result == {"deleted": True}
        # The raw template should be used when format raises KeyError
        assert captured_reason == ["Delete {nonexistent_field} from table"]


# ============================================================================
# _load_hitl_config — hitl attr is None (line 77)
# ============================================================================

class TestLoadHitlConfigHitlAttrNone:

    def test_load_hitl_config_hitl_attr_none(self, monkeypatch):
        """When cfg.hitl is None, _load_hitl_config returns default HITLConfig()."""
        mod = _load_hitl(monkeypatch)

        mock_cfg = MagicMock()
        mock_cfg.hitl = None
        monkeypatch.setitem(sys.modules, "config",
                            MagicMock(load_config=MagicMock(return_value=mock_cfg)))

        result = mod._load_hitl_config()
        assert isinstance(result, mod.HITLConfig)
        assert result.default_timeout == 300.0
        assert result.bypass_roles == []


# ============================================================================
# Gap 2: pause_task timeout path — audit log_event raises inside except block
# ============================================================================

class TestPauseTaskTimeoutAuditFails:

    @pytest.mark.asyncio
    async def test_pause_task_timeout_audit_log_event_raises(self, monkeypatch):
        """Lines 439-440: audit log_event raises inside timeout handler — except Exception: pass swallows it."""
        mod = _load_hitl(monkeypatch)

        reg = mod._TaskPauseRegistry()
        monkeypatch.setattr(mod, "pause_registry", reg)
        monkeypatch.setattr(mod, "_load_hitl_config",
                            lambda: mod.HITLConfig(default_timeout=0.01))

        # Make tools.audit.log_event raise an exception — only affects the import
        # inside the timeout handler (from builtin_tools.audit import log_event)
        raising_audit = MagicMock()
        raising_audit.log_event = MagicMock(side_effect=RuntimeError("audit exploded"))
        raising_audit.check_permission = MagicMock(return_value=True)
        raising_audit.get_workspace_roles = MagicMock(return_value=(["operator"], {}))
        monkeypatch.setitem(sys.modules, "builtin_tools.audit", raising_audit)

        # Should timeout and swallow the audit exception
        result = await mod.pause_task("timeout-audit-fail", "will timeout")

        assert result["resumed"] is False
        assert "error" in result
        assert "timed out" in result["error"].lower() or "timeout" in result["error"].lower()
