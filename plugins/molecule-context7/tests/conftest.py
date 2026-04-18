"""conftest for molecule-context7 plugin tests.

Stubs ``langchain_core`` if absent (test environments that don't install the
full workspace-template dependencies), and resets the context7 session call
counter between tests.
"""

import sys
from types import ModuleType

import pytest


def _mock_langchain_if_missing():
    if "langchain_core" not in sys.modules:
        lc_mod = ModuleType("langchain_core")
        lc_tools_mod = ModuleType("langchain_core.tools")
        lc_tools_mod.tool = lambda f: f  # @tool becomes identity decorator
        sys.modules["langchain_core"] = lc_mod
        sys.modules["langchain_core.tools"] = lc_tools_mod


_mock_langchain_if_missing()


@pytest.fixture(autouse=True)
def reset_call_counter():
    """Reset the module-level session call counter before every test."""
    import importlib.util
    import sys
    from pathlib import Path

    # Find and reload the counter so each test starts from 0.
    ctx7_path = Path(__file__).resolve().parents[1] / "skills" / "context7-docs" / "scripts" / "context7.py"
    spec = importlib.util.spec_from_file_location("context7_tools_under_test", ctx7_path)
    mod = importlib.util.module_from_spec(spec)
    sys.modules["context7_tools_under_test"] = mod
    spec.loader.exec_module(mod)
    mod._reset_counter()
    yield
    mod._reset_counter()
