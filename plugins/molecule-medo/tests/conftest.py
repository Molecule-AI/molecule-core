"""Minimal conftest for molecule-medo plugin tests.

langchain_core is a declared dependency of workspace-template (>=0.3.0) and
is expected to be present in the test environment.  If it is absent, mock it
so the @tool decorator in medo.py is a no-op and the tests can still run.
"""

import sys
from types import ModuleType


def _mock_langchain_if_missing():
    if "langchain_core" not in sys.modules:
        lc_mod = ModuleType("langchain_core")
        lc_tools_mod = ModuleType("langchain_core.tools")
        lc_tools_mod.tool = lambda f: f  # @tool becomes identity decorator
        sys.modules["langchain_core"] = lc_mod
        sys.modules["langchain_core.tools"] = lc_tools_mod


_mock_langchain_if_missing()
