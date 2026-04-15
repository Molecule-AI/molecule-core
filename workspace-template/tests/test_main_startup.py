"""Regression tests for workspace startup — issue #204.

Guards against re-introducing the abstract PushNotificationSender instantiation
in main.py that causes a TypeError crash on every workspace agent startup.

Background: commit 1c07046 (PR #198) originally wired
  push_sender=PushNotificationSender()
PushNotificationSender is an ABC with abstract method send_notification — Python
raises TypeError at runtime before any agent code runs.

Fix: replaced with BasePushNotificationSender(httpx.AsyncClient(), push_config_store).
These 12 tests prevent regression via source-code analysis and live SDK checks.
"""

import importlib.util
import inspect
import os
import re

import pytest


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

_MAIN_PY = os.path.join(os.path.dirname(os.path.dirname(__file__)), "main.py")


def _read_main() -> str:
    with open(_MAIN_PY) as f:
        return f.read()


def _load_real_module(dotted_name: str):
    """Load the real (installed) module, bypassing conftest sys.modules mocks.

    conftest replaces a2a.* entries in sys.modules with stubs whose __spec__ is
    None, causing importlib.util.find_spec to raise ValueError. Work around by
    temporarily removing all a2a.* mocks, importing the real package, then
    restoring the stubs so subsequent test code still sees the mocks.
    """
    import sys

    # Save and remove all a2a.* mock entries so importlib sees the real package
    saved = {k: v for k, v in sys.modules.items() if k == "a2a" or k.startswith("a2a.")}
    for k in saved:
        del sys.modules[k]

    try:
        return importlib.import_module(dotted_name)
    except Exception:
        return None
    finally:
        # Always restore the conftest mocks so other tests are unaffected
        for k, v in saved.items():
            if k not in sys.modules:
                sys.modules[k] = v


# ---------------------------------------------------------------------------
# 1–4: Source-code regression guards — no SDK needed
# ---------------------------------------------------------------------------

def test_main_does_not_import_bare_push_notification_sender():
    """PushNotificationSender (abstract) must never appear on an import line in main.py.

    BasePushNotificationSender is acceptable; bare PushNotificationSender is not.
    """
    src = _read_main()
    bad_import_lines = [
        line for line in src.splitlines()
        if "PushNotificationSender" in line
        and "BasePushNotificationSender" not in line
        and re.match(r"\s*(import|from)\s", line)
    ]
    assert not bad_import_lines, (
        f"main.py imports bare (abstract) PushNotificationSender: {bad_import_lines}"
    )


def test_main_does_not_instantiate_bare_push_notification_sender():
    """PushNotificationSender() call must not appear in main.py (it is abstract)."""
    src = _read_main()
    # Match "PushNotificationSender(" not preceded by "Base"
    matches = re.findall(r"(?<!Base)PushNotificationSender\s*\(", src)
    assert not matches, (
        "main.py instantiates abstract PushNotificationSender() — "
        "use BasePushNotificationSender instead"
    )


def test_main_push_sender_uses_base_concrete_class():
    """push_sender= in DefaultRequestHandler must use BasePushNotificationSender."""
    src = _read_main()
    match = re.search(r"push_sender\s*=\s*(\w+)", src)
    if match is None:
        pytest.skip("push_sender not explicitly set in main.py (None default is also acceptable)")
    used_class = match.group(1)
    assert used_class == "BasePushNotificationSender", (
        f"push_sender= uses '{used_class}' — must be BasePushNotificationSender (concrete)"
    )


def test_main_is_syntactically_valid():
    """main.py must compile without SyntaxError."""
    src = _read_main()
    try:
        compile(src, _MAIN_PY, "exec")
    except SyntaxError as e:
        raise AssertionError(f"main.py has a SyntaxError: {e}") from e


# ---------------------------------------------------------------------------
# 5–8: Real SDK checks — load installed a2a package bypassing conftest mocks
# ---------------------------------------------------------------------------

def test_push_notification_sender_is_abstract_in_sdk():
    """Regression guard: PushNotificationSender must be an ABC in the installed SDK.

    If the SDK ever makes it concrete, we should revisit whether BasePushNotificationSender
    is still the right choice.
    """
    real = _load_real_module("a2a.server.tasks")
    if real is None:
        pytest.skip("Could not load real a2a.server.tasks")
    assert inspect.isabstract(real.PushNotificationSender), (
        "PushNotificationSender is no longer abstract in the installed SDK — "
        "update this test and reconsider main.py wiring"
    )


def test_base_push_notification_sender_is_concrete_in_sdk():
    """BasePushNotificationSender must be concrete (safe to instantiate)."""
    real = _load_real_module("a2a.server.tasks")
    if real is None:
        pytest.skip("Could not load real a2a.server.tasks")
    assert not inspect.isabstract(real.BasePushNotificationSender), (
        "BasePushNotificationSender has become abstract — update main.py"
    )


def test_base_push_notification_sender_constructor_signature():
    """BasePushNotificationSender.__init__ must accept (httpx_client, config_store)."""
    real = _load_real_module("a2a.server.tasks")
    if real is None:
        pytest.skip("Could not load real a2a.server.tasks")
    sig = inspect.signature(real.BasePushNotificationSender.__init__)
    params = list(sig.parameters.keys())
    assert "httpx_client" in params, (
        f"BasePushNotificationSender.__init__ no longer has 'httpx_client' param: {params}"
    )
    assert "config_store" in params, (
        f"BasePushNotificationSender.__init__ no longer has 'config_store' param: {params}"
    )


def test_default_request_handler_push_sender_defaults_to_none():
    """DefaultRequestHandler must accept push_sender=None as its default.

    This means omitting push_sender entirely is safe — no crash on startup.
    """
    real = _load_real_module("a2a.server.request_handlers")
    if real is None:
        pytest.skip("Could not load real a2a.server.request_handlers")
    sig = inspect.signature(real.DefaultRequestHandler.__init__)
    assert "push_sender" in sig.parameters, (
        "DefaultRequestHandler no longer has a push_sender parameter"
    )
    assert sig.parameters["push_sender"].default is None, (
        "DefaultRequestHandler.push_sender no longer defaults to None"
    )


# ---------------------------------------------------------------------------
# 9–12: Capability and behaviour guards (source-code + a2a_executor)
# ---------------------------------------------------------------------------

def test_main_advertises_state_transition_history():
    """AgentCapabilities in main.py must set stateTransitionHistory=True (issue #174)."""
    src = _read_main()
    assert "stateTransitionHistory=True" in src, (
        "main.py must set stateTransitionHistory=True in AgentCapabilities"
    )


def test_main_wires_push_notifications_from_config():
    """AgentCapabilities.pushNotifications must be wired from config (not hardcoded False)."""
    src = _read_main()
    assert "pushNotifications=" in src, (
        "AgentCapabilities must wire pushNotifications"
    )
    # Must not be hardcoded off
    assert "pushNotifications=False" not in src, (
        "pushNotifications must not be hardcoded False — read from config"
    )


def test_main_push_config_store_shared_between_handler_and_sender():
    """push_config_store must be the same object passed to both DefaultRequestHandler
    and BasePushNotificationSender, ensuring registration and delivery share state.
    """
    src = _read_main()
    # Both usages must reference the same variable (not two separate InMemory... calls)
    inline_stores = re.findall(r"InMemoryPushNotificationConfigStore\(\)", src)
    assert len(inline_stores) <= 1, (
        "main.py creates multiple InMemoryPushNotificationConfigStore instances — "
        "the same store must be shared between DefaultRequestHandler and BasePushNotificationSender"
    )


def test_cancel_method_is_not_a_pass_stub():
    """a2a_executor.cancel() must not be a no-op pass stub (issue #173).

    Regression: cancel() was previously 'pass' with # pragma: no cover.
    It must emit at least one event to the queue.
    """
    from a2a_executor import LangGraphA2AExecutor
    src = inspect.getsource(LangGraphA2AExecutor.cancel)
    non_trivial_lines = [
        line.strip() for line in src.splitlines()
        if line.strip()
        and not line.strip().startswith(('"""', "'''", "#", "async def", "def"))
        and line.strip() != "pass"
    ]
    assert non_trivial_lines, (
        "cancel() must not be a no-op stub — it must emit a TaskStatusUpdateEvent "
        "with state=canceled so clients see the state transition"
    )
