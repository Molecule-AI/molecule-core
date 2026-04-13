"""LangGraph adapter — Python-based ReAct agent with skills, tools, and plugins."""

import os
import logging

from adapters.base import BaseAdapter, AdapterConfig
from a2a.server.agent_execution import AgentExecutor

logger = logging.getLogger(__name__)


class LangGraphAdapter(BaseAdapter):

    @staticmethod
    def name() -> str:
        return "langgraph"

    @staticmethod
    def display_name() -> str:
        return "LangGraph"

    @staticmethod
    def description() -> str:
        return "LangGraph ReAct agent — Python-based with skills, tools, plugins, and peer coordination"

    @staticmethod
    def get_config_schema() -> dict:
        return {
            "model": {"type": "string", "description": "LangChain model string (e.g. openrouter:google/gemini-2.5-flash)"},
            "skills": {"type": "array", "items": {"type": "string"}, "description": "Skill folder names to load"},
            "tools": {"type": "array", "items": {"type": "string"}, "description": "Built-in tools (web_search, filesystem, etc.)"},
        }

    def __init__(self):
        self.loaded_skills = []
        self.all_tools = []
        self.system_prompt = None

    async def setup(self, config: AdapterConfig) -> None:
        result = await self._common_setup(config)
        self.loaded_skills = result.loaded_skills
        self.all_tools = result.langchain_tools
        self.system_prompt = result.system_prompt

    async def create_executor(self, config: AdapterConfig) -> AgentExecutor:
        from agent import create_agent
        from a2a_executor import LangGraphA2AExecutor

        agent = create_agent(config.model, self.all_tools, self.system_prompt)
        return LangGraphA2AExecutor(agent, heartbeat=config.heartbeat, model=config.model)
