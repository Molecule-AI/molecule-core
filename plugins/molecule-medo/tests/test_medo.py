"""Tests for plugins/molecule-medo/skills/medo-tools/scripts/medo.py.

All tests exercise the mock backend (no MEDO_API_KEY required).

NOTE: @tool is a LangChain decorator that returns a StructuredTool rather than
the raw async function.  conftest.py mocks langchain_core.tools.tool as an
identity decorator so that calling the functions directly (without .ainvoke())
works in tests — matching the original test approach.
"""

import importlib.util
import sys
from pathlib import Path

import pytest

# plugin root: plugins/molecule-medo/
_PLUGIN_ROOT = Path(__file__).resolve().parents[1]
_MEDO_PATH = _PLUGIN_ROOT / "skills" / "medo-tools" / "scripts" / "medo.py"


def _load_medo():
    spec = importlib.util.spec_from_file_location("medo_plugin_tools", _MEDO_PATH)
    mod = importlib.util.module_from_spec(spec)
    sys.modules["medo_plugin_tools"] = mod  # register before exec to handle self-refs
    spec.loader.exec_module(mod)
    return mod


@pytest.fixture()
def medo(monkeypatch):
    monkeypatch.delenv("MEDO_API_KEY", raising=False)
    monkeypatch.delenv("MEDO_BASE_URL", raising=False)
    return _load_medo()


class TestCreateMedoApp:
    @pytest.mark.asyncio
    async def test_requires_name(self, medo):
        result = await medo.create_medo_app(name="")
        assert "error" in result

    @pytest.mark.asyncio
    async def test_rejects_unknown_template(self, medo):
        result = await medo.create_medo_app(name="app", template="unknown")
        assert "error" in result and "template" in result["error"]

    @pytest.mark.asyncio
    async def test_mock_success(self, medo):
        result = await medo.create_medo_app(name="my-app", template="chatbot")
        assert result.get("mock") is True and result.get("status") == "ok"


class TestUpdateMedoApp:
    @pytest.mark.asyncio
    async def test_requires_app_id(self, medo):
        result = await medo.update_medo_app(app_id="", content={"title": "x"})
        assert "error" in result

    @pytest.mark.asyncio
    async def test_requires_non_empty_content(self, medo):
        result = await medo.update_medo_app(app_id="abc", content={})
        assert "error" in result

    @pytest.mark.asyncio
    async def test_mock_success(self, medo):
        result = await medo.update_medo_app(app_id="abc", content={"title": "v2"})
        assert result.get("mock") is True and "abc" in result.get("path", "")


class TestPublishMedoApp:
    @pytest.mark.asyncio
    async def test_requires_app_id(self, medo):
        result = await medo.publish_medo_app(app_id="")
        assert "error" in result

    @pytest.mark.asyncio
    async def test_rejects_invalid_environment(self, medo):
        result = await medo.publish_medo_app(app_id="abc", environment="dev")
        assert "error" in result and "environment" in result["error"]

    @pytest.mark.asyncio
    async def test_mock_success(self, medo):
        result = await medo.publish_medo_app(app_id="abc")
        assert result.get("mock") is True and result.get("status") == "ok"
