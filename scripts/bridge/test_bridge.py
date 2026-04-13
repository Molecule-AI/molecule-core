"""Unit tests for the external workspace bridge."""

import json
import io
import os
import sys
from pathlib import Path
from unittest.mock import MagicMock, patch
import pytest

# Add scripts/ to path so bridge package is importable
sys.path.insert(0, str(Path(__file__).parent.parent))

from bridge.processor import (
    MessageProcessor,
    ClaudeCodeProcessor,
    OpenAIProcessor,
    AnthropicProcessor,
    HTTPForwardProcessor,
    EchoProcessor,
    PROCESSORS,
    create_processor,
)


# ─── Processor registry ───

class TestProcessorRegistry:
    def test_all_processors_registered(self):
        expected = {"claude-code", "openai", "anthropic", "http", "echo"}
        assert set(PROCESSORS.keys()) == expected

    def test_create_processor_valid(self):
        p = create_processor("echo")
        assert isinstance(p, EchoProcessor)

    def test_create_processor_invalid(self):
        with pytest.raises(ValueError, match="Unknown processor"):
            create_processor("nonexistent")

    def test_create_processor_with_kwargs(self):
        p = create_processor("openai", model="gpt-4o", api_key="test-key")
        assert p.model == "gpt-4o"
        assert p.api_key == "test-key"


# ─── EchoProcessor ───

class TestEchoProcessor:
    def test_echoes_message(self):
        p = EchoProcessor()
        result = p.process("hello", "PM", {})
        assert result == "Echo from bridge: hello"

    def test_name(self):
        assert EchoProcessor.name == "echo"

    def test_empty_message(self):
        p = EchoProcessor()
        result = p.process("", "PM", {})
        assert result == "Echo from bridge: "


# ─── OpenAIProcessor ───

class TestOpenAIProcessor:
    def test_missing_api_key_returns_error(self):
        p = OpenAIProcessor(api_key="")
        result = p.process("hello", "PM", {})
        assert "not configured" in result

    def test_init_reads_env(self, monkeypatch):
        monkeypatch.setenv("OPENAI_API_KEY", "sk-test-123")
        p = OpenAIProcessor()
        assert p.api_key == "sk-test-123"

    def test_init_explicit_key_overrides_env(self, monkeypatch):
        monkeypatch.setenv("OPENAI_API_KEY", "from-env")
        p = OpenAIProcessor(api_key="explicit")
        assert p.api_key == "explicit"

    @patch("httpx.post")
    def test_successful_response(self, mock_post):
        mock_resp = MagicMock()
        mock_resp.json.return_value = {
            "choices": [{"message": {"content": "Hello from GPT"}}]
        }
        mock_post.return_value = mock_resp

        p = OpenAIProcessor(api_key="test-key")
        result = p.process("hi", "PM", {})
        assert result == "Hello from GPT"

    @patch("httpx.post", side_effect=Exception("connection refused"))
    def test_api_error(self, mock_post):
        p = OpenAIProcessor(api_key="test-key")
        result = p.process("hi", "PM", {})
        assert "OpenAI API error" in result


# ─── AnthropicProcessor ───

class TestAnthropicProcessor:
    def test_missing_api_key_returns_error(self):
        p = AnthropicProcessor(api_key="")
        result = p.process("hello", "PM", {})
        assert "not configured" in result

    def test_init_reads_env(self, monkeypatch):
        monkeypatch.setenv("ANTHROPIC_API_KEY", "sk-ant-test")
        p = AnthropicProcessor()
        assert p.api_key == "sk-ant-test"

    @patch("httpx.post")
    def test_successful_response(self, mock_post):
        mock_resp = MagicMock()
        mock_resp.json.return_value = {
            "content": [{"text": "Hello from Claude"}]
        }
        mock_post.return_value = mock_resp

        p = AnthropicProcessor(api_key="test-key")
        result = p.process("hi", "PM", {})
        assert result == "Hello from Claude"


# ─── HTTPForwardProcessor ───

class TestHTTPForwardProcessor:
    def test_no_url_returns_error(self):
        p = HTTPForwardProcessor(url="")
        result = p.process("hi", "PM", {})
        assert "not configured" in result

    @patch("httpx.post")
    def test_forwards_message(self, mock_post):
        mock_resp = MagicMock()
        mock_resp.text = "forwarded response"
        mock_post.return_value = mock_resp

        p = HTTPForwardProcessor(url="http://my-agent:8000")
        result = p.process("hello", "PM", {"sender_id": "ws-1"})
        assert result == "forwarded response"
        mock_post.assert_called_once()
        call_kwargs = mock_post.call_args
        assert call_kwargs[1]["json"]["message"] == "hello"
        assert call_kwargs[1]["json"]["sender"] == "PM"


# ─── ClaudeCodeProcessor ───

class TestClaudeCodeProcessor:
    def test_name(self):
        assert ClaudeCodeProcessor.name == "claude-code"

    @patch("subprocess.run")
    def test_successful_response(self, mock_run):
        mock_run.return_value = MagicMock(
            returncode=0, stdout="The answer is 42", stderr=""
        )
        p = ClaudeCodeProcessor(cwd="/tmp")
        result = p.process("what is the answer?", "PM", {})
        assert result == "The answer is 42"

    @patch("subprocess.run")
    def test_json_output_parsed(self, mock_run):
        mock_run.return_value = MagicMock(
            returncode=0,
            stdout=json.dumps({"result": "parsed answer", "session_id": "abc"}),
            stderr="",
        )
        p = ClaudeCodeProcessor(cwd="/tmp")
        result = p.process("test", "PM", {})
        assert result == "parsed answer"

    @patch("subprocess.run")
    def test_error_exit_code(self, mock_run):
        mock_run.return_value = MagicMock(
            returncode=1, stdout="", stderr="auth error"
        )
        p = ClaudeCodeProcessor(cwd="/tmp")
        result = p.process("test", "PM", {})
        assert "auth error" in result

    @patch("subprocess.run", side_effect=FileNotFoundError)
    def test_claude_not_installed(self, mock_run):
        p = ClaudeCodeProcessor(cwd="/tmp")
        result = p.process("test", "PM", {})
        assert "not found" in result.lower()

    def test_model_flag(self):
        p = ClaudeCodeProcessor(model="opus")
        assert p.model == "opus"


# ─── MessageProcessor interface ───

class TestMessageProcessorInterface:
    def test_cannot_instantiate_abstract(self):
        with pytest.raises(TypeError):
            MessageProcessor()

    def test_subclass_must_implement_process(self):
        class Incomplete(MessageProcessor):
            name = "incomplete"
        with pytest.raises(TypeError):
            Incomplete()

    def test_valid_subclass(self):
        class Valid(MessageProcessor):
            name = "valid"
            def process(self, message, sender, context):
                return "ok"
        v = Valid()
        assert v.process("x", "y", {}) == "ok"
