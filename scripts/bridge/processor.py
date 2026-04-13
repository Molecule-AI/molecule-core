"""Message processor interface and built-in implementations.

To add a new backend:
1. Subclass MessageProcessor
2. Implement process(message, sender, context) -> str
3. Register in PROCESSORS dict
"""

import json
import logging
import os
import subprocess
from abc import ABC, abstractmethod
from pathlib import Path

logger = logging.getLogger("bridge.processor")


class MessageProcessor(ABC):
    """Interface for processing incoming A2A messages."""

    @abstractmethod
    def process(self, message: str, sender: str, context: dict) -> str:
        """Process an incoming message and return the response text.

        Args:
            message: The incoming message text
            sender: Name of the sending workspace
            context: Additional context (sender_id, workspace_id, etc.)

        Returns:
            Response text to send back via A2A
        """
        ...

    @property
    @abstractmethod
    def name(self) -> str:
        """Human-readable name for this processor."""
        ...


class ClaudeCodeProcessor(MessageProcessor):
    """Spawns `claude --print` CLI with full codebase access."""

    name = "claude-code"

    def __init__(self, cwd: str | None = None, model: str = ""):
        self.cwd = cwd or str(Path(__file__).parent.parent.parent)
        self.model = model

    def process(self, message: str, sender: str, context: dict) -> str:
        system_prompt = (
            f"You are an AI technical advisor for the Molecule AI Molecule AI platform. "
            f"Agent '{sender}' is asking you a question via A2A protocol. "
            f"You have access to the full codebase at the current directory. "
            f"Respond concisely and helpfully. Keep responses under 500 words "
            f"unless a detailed analysis is needed."
        )

        cmd = ["claude", "--print", "--dangerously-skip-permissions",
               "--system-prompt", system_prompt, "-p", message]
        if self.model:
            cmd.extend(["--model", self.model])

        try:
            result = subprocess.run(
                cmd, capture_output=True, text=True,
                timeout=300, cwd=self.cwd,
            )
            if result.returncode == 0 and result.stdout.strip():
                try:
                    out = json.loads(result.stdout)
                    if isinstance(out, dict) and "result" in out:
                        return out["result"]
                except json.JSONDecodeError:
                    pass
                return result.stdout.strip()
            return f"Processing error: {result.stderr.strip()[:200]}"
        except subprocess.TimeoutExpired:
            return "Request timed out (5 min limit)."
        except FileNotFoundError:
            return "Claude CLI not found. Install: npm install -g @anthropic-ai/claude-code"
        except Exception as e:
            return f"Error: {e}"


class OpenAIProcessor(MessageProcessor):
    """Calls OpenAI-compatible API (GPT, local LLM, OpenRouter, etc.)."""

    name = "openai"

    def __init__(self, model: str = "gpt-4.1-mini", base_url: str = "", api_key: str = ""):
        self.model = model
        self.base_url = base_url or os.environ.get("OPENAI_BASE_URL", "https://api.openai.com/v1")
        self.api_key = api_key or os.environ.get("OPENAI_API_KEY", "")
        if not self.api_key:
            logger.warning("OpenAI processor: no API key set (OPENAI_API_KEY env var or --api-key)")

    def process(self, message: str, sender: str, context: dict) -> str:
        if not self.api_key:
            return "OpenAI API key not configured. Set OPENAI_API_KEY environment variable."
        try:
            import httpx
            resp = httpx.post(
                f"{self.base_url}/chat/completions",
                headers={"Authorization": f"Bearer {self.api_key}"},
                json={
                    "model": self.model,
                    "messages": [
                        {"role": "system", "content": f"You are a technical advisor. Agent '{sender}' is asking you a question."},
                        {"role": "user", "content": message},
                    ],
                    "max_tokens": 1000,
                },
                timeout=60,
            )
            data = resp.json()
            return data["choices"][0]["message"]["content"]
        except Exception as e:
            return f"OpenAI API error: {e}"


class AnthropicProcessor(MessageProcessor):
    """Calls Anthropic API directly."""

    name = "anthropic"

    def __init__(self, model: str = "claude-sonnet-4-6", api_key: str = ""):
        self.model = model
        self.api_key = api_key or os.environ.get("ANTHROPIC_API_KEY", "")
        if not self.api_key:
            logger.warning("Anthropic processor: no API key set (ANTHROPIC_API_KEY env var)")

    def process(self, message: str, sender: str, context: dict) -> str:
        if not self.api_key:
            return "Anthropic API key not configured. Set ANTHROPIC_API_KEY environment variable."
        try:
            import httpx
            resp = httpx.post(
                "https://api.anthropic.com/v1/messages",
                headers={
                    "x-api-key": self.api_key,
                    "anthropic-version": "2023-06-01",
                    "content-type": "application/json",
                },
                json={
                    "model": self.model,
                    "max_tokens": 1000,
                    "system": f"You are a technical advisor. Agent '{sender}' is asking you a question.",
                    "messages": [{"role": "user", "content": message}],
                },
                timeout=60,
            )
            data = resp.json()
            return data["content"][0]["text"]
        except Exception as e:
            return f"Anthropic API error: {e}"


class HTTPForwardProcessor(MessageProcessor):
    """Forwards the message to an arbitrary HTTP endpoint."""

    name = "http"

    def __init__(self, url: str = "", headers: dict | None = None):
        self.url = url or os.environ.get("BRIDGE_FORWARD_URL", "")
        self.headers = headers or {}

    def process(self, message: str, sender: str, context: dict) -> str:
        if not self.url:
            return "HTTP forward URL not configured"
        try:
            import httpx
            resp = httpx.post(
                self.url,
                json={"message": message, "sender": sender, **context},
                headers=self.headers,
                timeout=60,
            )
            return resp.text
        except Exception as e:
            return f"HTTP forward error: {e}"


class EchoProcessor(MessageProcessor):
    """Simple echo for testing — returns the message back."""

    name = "echo"

    def __init__(self, **kwargs):
        pass  # No config needed

    def process(self, message: str, sender: str, context: dict) -> str:
        return f"Echo from bridge: {message}"


# Registry of available processors
PROCESSORS: dict[str, type[MessageProcessor]] = {
    "claude-code": ClaudeCodeProcessor,
    "openai": OpenAIProcessor,
    "anthropic": AnthropicProcessor,
    "http": HTTPForwardProcessor,
    "echo": EchoProcessor,
}


def create_processor(name: str, **kwargs) -> MessageProcessor:
    """Create a processor by name with optional config."""
    cls = PROCESSORS.get(name)
    if not cls:
        raise ValueError(f"Unknown processor: {name}. Available: {list(PROCESSORS.keys())}")
    return cls(**kwargs)
