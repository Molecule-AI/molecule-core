"""AutoGen adapter — Microsoft's multi-agent framework with full platform integration.

Uses AutoGen's AssistantAgent with OpenAIChatCompletionClient,
includes all platform tools (delegation, memory, sandbox, approval), skills, and coordinator support.

Requires: pip install autogen-agentchat autogen-ext[openai]
"""

import json
import logging

from adapters.base import BaseAdapter, AdapterConfig
from adapters.shared_runtime import (
    build_task_text,
    brief_task,
    extract_history,
    extract_message_text,
    set_current_task,
)
from a2a.server.agent_execution import AgentExecutor

logger = logging.getLogger(__name__)


def _langchain_to_autogen(lc_tool):
    """Wrap a LangChain BaseTool as an AutoGen FunctionTool.

    AutoGen requires typed function signatures (no **kwargs).
    LangChain tools accept a single string or dict input via ainvoke.
    We bridge them with a single `input: str` parameter.
    """
    from autogen_core.tools import FunctionTool

    async def _invoke(input: str) -> str:  # noqa: A002
        # Try to parse as JSON dict for tools expecting structured input
        try:
            parsed = json.loads(input)
            if isinstance(parsed, dict):
                result = await lc_tool.ainvoke(parsed)
                return str(result)
        except (json.JSONDecodeError, TypeError):
            pass
        result = await lc_tool.ainvoke(input)
        return str(result)

    return FunctionTool(
        _invoke,
        name=lc_tool.name,
        description=lc_tool.description or lc_tool.name,
    )


class AutoGenAdapter(BaseAdapter):

    def __init__(self):
        self.system_prompt = None
        self.autogen_tools = []

    @staticmethod
    def name() -> str:
        return "autogen"

    @staticmethod
    def display_name() -> str:
        return "AutoGen"

    @staticmethod
    def description() -> str:
        return "Microsoft AutoGen — conversable agents with tool use and multi-agent orchestration"

    @staticmethod
    def get_config_schema() -> dict:
        return {
            "model": {"type": "string", "description": "OpenAI model (e.g. openai:gpt-4.1-mini)"},
            "skills": {"type": "array", "items": {"type": "string"}, "description": "Skill folder names to load"},
            "tools": {"type": "array", "items": {"type": "string"}, "description": "Built-in tools"},
        }

    async def setup(self, config: AdapterConfig) -> None:
        try:
            from autogen_agentchat.agents import AssistantAgent  # noqa: F401
            logger.info("AutoGen AgentChat loaded")
        except ImportError:
            raise RuntimeError("autogen-agentchat not installed.")

        result = await self._common_setup(config)
        self.system_prompt = result.system_prompt
        self.autogen_tools = [_langchain_to_autogen(t) for t in result.langchain_tools]
        logger.info(f"AutoGen tools: {[t.name for t in self.autogen_tools]}")

    async def create_executor(self, config: AdapterConfig) -> AgentExecutor:
        return AutoGenA2AExecutor(
            model=config.model,
            system_prompt=self.system_prompt,
            autogen_tools=self.autogen_tools,
            heartbeat=config.heartbeat,
        )


class AutoGenA2AExecutor(AgentExecutor):
    """Wraps AutoGen's AssistantAgent with full platform tools."""

    def __init__(self, model: str, system_prompt: str | None, autogen_tools: list, heartbeat=None):
        self.model = model
        self.system_prompt = system_prompt
        self.autogen_tools = autogen_tools
        self._heartbeat = heartbeat

    async def execute(self, context, event_queue):
        from a2a.utils import new_agent_text_message

        user_message = extract_message_text(context)

        if not user_message:
            await event_queue.enqueue_event(new_agent_text_message("No message provided"))
            return

        await set_current_task(self._heartbeat, brief_task(user_message))

        try:
            from autogen_agentchat.agents import AssistantAgent
            from autogen_ext.models.openai import OpenAIChatCompletionClient

            model_str = self.model
            if ":" in model_str:
                _, model_name = model_str.split(":", 1)
            else:
                model_name = model_str

            task_text = build_task_text(user_message, extract_history(context))

            client = OpenAIChatCompletionClient(model=model_name)
            agent = AssistantAgent(
                name="agent",
                model_client=client,
                system_message=self.system_prompt or "You are a helpful assistant.",
                tools=self.autogen_tools,
            )

            result = await agent.run(task=task_text)

            reply = ""
            if hasattr(result, "messages") and result.messages:
                for msg in reversed(result.messages):
                    if hasattr(msg, "content") and isinstance(msg.content, str):
                        reply = msg.content
                        break
            if not reply:
                reply = str(result)

        except Exception as e:
            reply = f"AutoGen error: {e}"
        finally:
            await set_current_task(self._heartbeat, "")

        await event_queue.enqueue_event(new_agent_text_message(reply))

    async def cancel(self, context, event_queue):  # pragma: no cover
        pass
