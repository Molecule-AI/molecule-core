"""CrewAI adapter — role-based multi-agent framework with full platform integration.

Creates a CrewAI Agent + Task + Crew with all platform tools (delegation, memory,
sandbox, approval), skills, plugins, and coordinator support.

Requires: pip install crewai
"""

import asyncio
import logging

from adapters.base import BaseAdapter, AdapterConfig
from a2a.server.agent_execution import AgentExecutor

logger = logging.getLogger(__name__)


def _langchain_to_crewai(lc_tool):
    """Wrap a LangChain BaseTool as a sync CrewAI @tool.

    CrewAI's @tool decorator requires the function to have a docstring
    at decoration time, so we set __doc__ before applying the decorator.
    """
    from crewai.tools import tool as crewai_tool

    def wrapper(**kwargs) -> str:
        """Placeholder."""
        result = asyncio.get_event_loop().run_until_complete(lc_tool.ainvoke(kwargs))
        return str(result)

    wrapper.__name__ = lc_tool.name
    wrapper.__doc__ = lc_tool.description or f"Tool: {lc_tool.name}"
    return crewai_tool(lc_tool.name)(wrapper)


class CrewAIAdapter(BaseAdapter):

    def __init__(self):
        self.system_prompt = None
        self.crewai_tools = []

    @staticmethod
    def name() -> str:
        return "crewai"

    @staticmethod
    def display_name() -> str:
        return "CrewAI"

    @staticmethod
    def description() -> str:
        return "CrewAI — role-based agent with task delegation and crew orchestration"

    @staticmethod
    def get_config_schema() -> dict:
        return {
            "model": {"type": "string", "description": "LLM model (e.g. openai:gpt-4.1-mini)"},
            "skills": {"type": "array", "items": {"type": "string"}, "description": "Skill folder names to load"},
            "tools": {"type": "array", "items": {"type": "string"}, "description": "Built-in tools"},
        }

    async def setup(self, config: AdapterConfig) -> None:
        try:
            import crewai  # noqa: F401
            logger.info(f"CrewAI version: {crewai.__version__}")
        except ImportError:
            raise RuntimeError("crewai not installed.")

        result = await self._common_setup(config)
        self.system_prompt = result.system_prompt
        self.crewai_tools = [_langchain_to_crewai(t) for t in result.langchain_tools]
        logger.info(f"CrewAI tools: {[t.name for t in result.langchain_tools]}")

    async def create_executor(self, config: AdapterConfig) -> AgentExecutor:
        return CrewAIA2AExecutor(
            model=config.model,
            system_prompt=self.system_prompt,
            crewai_tools=self.crewai_tools,
            heartbeat=config.heartbeat,
        )


class CrewAIA2AExecutor(AgentExecutor):
    """Wraps CrewAI's Agent + Crew.kickoff() with full platform tools."""

    def __init__(self, model: str, system_prompt: str | None, crewai_tools: list, heartbeat=None):
        self.model = model
        self.system_prompt = system_prompt
        self.crewai_tools = crewai_tools
        self._heartbeat = heartbeat

    async def execute(self, context, event_queue):
        from a2a.utils import new_agent_text_message
        from adapters.shared_runtime import extract_history, build_task_text, brief_task, set_current_task

        from adapters.shared_runtime import extract_message_text
        user_message = extract_message_text(context)

        if not user_message:
            await event_queue.enqueue_event(new_agent_text_message("No message provided"))
            return

        await set_current_task(self._heartbeat, brief_task(user_message))

        try:
            from crewai import Agent, Task, Crew

            model_str = self.model
            if model_str.startswith("openai:"):
                model_str = model_str.replace("openai:", "openai/")

            backstory = self.system_prompt or "You are a helpful AI agent."

            history = extract_history(context)
            task_desc = build_task_text(user_message, history)

            agent = Agent(
                role=backstory.split("\n")[0][:100],
                goal="Help the user and coordinate with peer agents when needed",
                backstory=backstory,
                llm=model_str,
                tools=self.crewai_tools,
                verbose=False,
            )

            task = Task(
                description=task_desc,
                expected_output="A helpful response",
                agent=agent,
            )

            crew = Crew(agents=[agent], tasks=[task], verbose=False)
            result = await asyncio.to_thread(crew.kickoff)
            reply = str(result)

        except Exception as e:
            reply = f"CrewAI error: {e}"
        finally:
            await set_current_task(self._heartbeat, "")

        await event_queue.enqueue_event(new_agent_text_message(reply))

    async def cancel(self, context, event_queue):  # pragma: no cover
        pass
