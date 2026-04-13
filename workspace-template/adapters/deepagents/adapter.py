"""DeepAgents adapter — fully utilizing the DeepAgents SDK.

Uses create_deep_agent() with:
- FilesystemBackend(/workspace) — persistent file access across messages
- MemorySaver checkpointer — session continuity
- Memory files — CLAUDE.md loaded natively
- Filesystem permissions — restrict writes to /workspace and /configs
- InMemoryCache — avoid repeat API calls
- All built-in tools: write_todos, read_file, write_file, edit_file,
  ls, glob, grep, execute, task

Supports: anthropic, openai, openrouter, groq, cerebras, google_genai, ollama.
"""

import os
import glob as globmod
import logging

from adapters.base import BaseAdapter, AdapterConfig
from a2a.server.agent_execution import AgentExecutor

logger = logging.getLogger(__name__)


class DeepAgentsAdapter(BaseAdapter):

    def __init__(self):
        self.agent = None
        self._checkpointer = None

    @staticmethod
    def name() -> str:
        return "deepagents"

    @staticmethod
    def display_name() -> str:
        return "DeepAgents"

    @staticmethod
    def description() -> str:
        return "LangChain DeepAgents — planning, filesystem, sub-agents, shell execution, session persistence"

    @staticmethod
    def get_config_schema() -> dict:
        return {
            "model": {
                "type": "string",
                "description": "provider:model (e.g. google_genai:gemini-2.5-flash, groq:llama-3.3-70b-versatile)",
                "default": "google_genai:gemini-2.5-flash",
            },
            "skills": {"type": "array", "items": {"type": "string"}},
            "tools": {"type": "array", "items": {"type": "string"}},
        }

    def _create_llm(self, model_str: str):
        """Create a LangChain LLM from a provider:model string."""
        if ":" in model_str:
            provider, model_name = model_str.split(":", 1)
        else:
            provider, model_name = "anthropic", model_str

        if provider == "openai":
            from langchain_openai import ChatOpenAI
            kwargs = {"model": model_name}
            base_url = os.environ.get("OPENAI_BASE_URL", "")
            if base_url:
                kwargs["openai_api_base"] = base_url
            return ChatOpenAI(**kwargs)
        elif provider == "openrouter":
            from langchain_openai import ChatOpenAI
            return ChatOpenAI(
                model=model_name,
                openai_api_key=os.environ.get("OPENROUTER_API_KEY", os.environ.get("OPENAI_API_KEY", "")),
                openai_api_base="https://openrouter.ai/api/v1",
                max_tokens=int(os.environ.get("MAX_TOKENS", "2048")),
            )
        elif provider == "groq":
            from langchain_openai import ChatOpenAI
            return ChatOpenAI(
                model=model_name,
                openai_api_key=os.environ.get("GROQ_API_KEY", ""),
                openai_api_base="https://api.groq.com/openai/v1",
            )
        elif provider == "cerebras":
            from langchain_openai import ChatOpenAI
            return ChatOpenAI(
                model=model_name,
                openai_api_key=os.environ.get("CEREBRAS_API_KEY", ""),
                openai_api_base="https://api.cerebras.ai/v1",
            )
        elif provider == "qianfan":
            from langchain_openai import ChatOpenAI
            return ChatOpenAI(
                model=model_name,
                openai_api_key=os.environ.get("QIANFAN_API_KEY", os.environ.get("AISTUDIO_API_KEY", "")),
                openai_api_base="https://qianfan.baidubce.com/v2",
            )
        elif provider == "anthropic":
            from langchain_anthropic import ChatAnthropic
            kwargs = {"model": model_name}
            base_url = os.environ.get("ANTHROPIC_BASE_URL", "")
            if base_url:
                kwargs["anthropic_api_url"] = base_url
            return ChatAnthropic(**kwargs)
        elif provider == "google_genai":
            from langchain_google_genai import ChatGoogleGenerativeAI
            return ChatGoogleGenerativeAI(model=model_name)
        elif provider == "ollama":
            from langchain_ollama import ChatOllama
            return ChatOllama(model=model_name)
        else:
            raise ValueError(f"Unsupported model provider: {provider}")

    async def setup(self, config: AdapterConfig) -> None:
        try:
            from deepagents import create_deep_agent, FilesystemPermission
            from deepagents.backends import FilesystemBackend
            from langgraph.checkpoint.memory import MemorySaver
            from langchain_core.caches import InMemoryCache
        except ImportError as e:
            raise RuntimeError(f"deepagents not installed: {e}")

        result = await self._common_setup(config)
        logger.info("DeepAgents platform tools: %s", [t.name for t in result.langchain_tools])

        llm = self._create_llm(config.model)

        # FilesystemBackend — persistent file access
        workspace_dir = "/workspace" if os.path.isdir("/workspace") else "/configs"
        # virtual_mode=False: read/write the real bind-mounted filesystem so
        # read_file/ls/write_file/edit_file match what `bash` sees. With
        # virtual_mode=True agents operate on an in-memory snapshot and
        # report real files as "missing" (and writes don't persist across
        # restarts). Permissions below still scope access to /workspace + /configs.
        backend = FilesystemBackend(root_dir=workspace_dir, virtual_mode=False)

        # MemorySaver — session continuity
        self._checkpointer = MemorySaver()

        # Memory — load CLAUDE.md natively
        memory_files = []
        claude_md = os.path.join(config.config_path, "CLAUDE.md")
        if os.path.exists(claude_md):
            memory_files.append(claude_md)

        # Filesystem permissions
        permissions = [
            FilesystemPermission(operations=["read", "write"], paths=["/workspace/**"], mode="allow"),
            FilesystemPermission(operations=["read", "write"], paths=["/configs/**"], mode="allow"),
        ]

        # Native skills from /configs/skills/*.py
        deepagent_skills = []
        skills_dir = os.path.join(config.config_path, "skills")
        if os.path.isdir(skills_dir):
            deepagent_skills = globmod.glob(os.path.join(skills_dir, "**", "*.py"), recursive=True)

        # LLM cache
        cache = InMemoryCache()

        self.agent = create_deep_agent(
            model=llm,
            tools=result.langchain_tools,
            system_prompt=result.system_prompt,
            backend=backend,
            checkpointer=self._checkpointer,
            memory=memory_files if memory_files else None,
            permissions=permissions,
            skills=deepagent_skills if deepagent_skills else None,
            cache=cache,
        )

        logger.info(
            "DeepAgents: %d tools, backend=%s, checkpointer=MemorySaver, "
            "cache=InMemoryCache, memory=%d, permissions=%d, skills=%d",
            len(result.langchain_tools), type(backend).__name__,
            len(memory_files), len(permissions), len(deepagent_skills),
        )

    async def create_executor(self, config: AdapterConfig) -> AgentExecutor:
        if self.agent is None:
            raise RuntimeError("setup() must be called before create_executor()")
        from a2a_executor import LangGraphA2AExecutor
        return LangGraphA2AExecutor(self.agent, heartbeat=config.heartbeat, model=config.model)
