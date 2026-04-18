"""Create the Deep Agent with model + skills + tools."""

import os
import logging

from langgraph.prebuilt import create_react_agent

logger = logging.getLogger(__name__)


def create_agent(model_str: str, tools: list, system_prompt: str):
    """Create a LangGraph ReAct agent.

    Args:
        model_str: LangChain-compatible model string (e.g., 'anthropic:claude-sonnet-4-6')
        tools: List of tool functions
        system_prompt: The system prompt for the agent
    """
    # Parse provider:model format
    if ":" in model_str:
        provider, model_name = model_str.split(":", 1)
    else:
        provider = "anthropic"
        model_name = model_str

    # Import the provider package
    try:
        if provider in ("anthropic",):
            from langchain_anthropic import ChatAnthropic as LLMClass
        elif provider in ("openai", "openrouter", "groq", "cerebras", "qianfan"):
            from langchain_openai import ChatOpenAI as LLMClass
        elif provider == "google_genai":
            from langchain_google_genai import ChatGoogleGenerativeAI as LLMClass
        elif provider == "ollama":
            from langchain_ollama import ChatOllama as LLMClass
        else:
            raise ValueError(f"Unsupported model provider: {provider}")
    except ImportError as e:
        pkg = "langchain-openai" if provider == "openrouter" else f"langchain-{provider}"
        raise ImportError(f"Provider '{provider}' requires package '{pkg}'. Install: pip install {pkg}") from e

    # Instantiate the LLM
    if provider == "anthropic":
        llm_kwargs = {"model": model_name}
        anthropic_base_url = os.environ.get("ANTHROPIC_BASE_URL", "")
        if anthropic_base_url:
            llm_kwargs["anthropic_api_url"] = anthropic_base_url
        llm = LLMClass(**llm_kwargs)
    elif provider == "openrouter":
        api_key = os.environ.get("OPENROUTER_API_KEY", os.environ.get("OPENAI_API_KEY", ""))
        max_tokens = int(os.environ.get("MAX_TOKENS", "2048"))
        llm = LLMClass(
            model=model_name,
            openai_api_key=api_key,
            openai_api_base="https://openrouter.ai/api/v1",
            max_tokens=max_tokens,
        )
    elif provider == "groq":
        api_key = os.environ.get("GROQ_API_KEY", "")
        llm = LLMClass(
            model=model_name,
            openai_api_key=api_key,
            openai_api_base="https://api.groq.com/openai/v1",
        )
    elif provider == "cerebras":
        api_key = os.environ.get("CEREBRAS_API_KEY", "")
        llm = LLMClass(
            model=model_name,
            openai_api_key=api_key,
            openai_api_base="https://api.cerebras.ai/v1",
        )
    elif provider == "qianfan":
        api_key = os.environ.get("QIANFAN_API_KEY", os.environ.get("AISTUDIO_API_KEY", ""))
        llm = LLMClass(
            model=model_name,
            openai_api_key=api_key,
            openai_api_base="https://qianfan.baidubce.com/v2",
        )
    elif provider == "openai":
        llm_kwargs = {"model": model_name}
        openai_base_url = os.environ.get("OPENAI_BASE_URL", "")
        if openai_base_url:
            llm_kwargs["openai_api_base"] = openai_base_url
        llm = LLMClass(**llm_kwargs)
    else:
        llm = LLMClass(model=model_name)

    # Auto-inject Langfuse tracing if env vars are present
    callbacks = _setup_langfuse()
    if callbacks:
        llm.callbacks = callbacks

    agent = create_react_agent(
        model=llm,
        tools=tools,
        prompt=system_prompt,
    )

    return agent


def _setup_langfuse():
    """Set up Langfuse tracing if LANGFUSE_* env vars are present.

    Returns list of callbacks to pass to agent invocations, or empty list.
    """
    langfuse_host = os.environ.get("LANGFUSE_HOST")
    langfuse_public = os.environ.get("LANGFUSE_PUBLIC_KEY")
    langfuse_secret = os.environ.get("LANGFUSE_SECRET_KEY")

    if not (langfuse_host and langfuse_public and langfuse_secret):
        return []

    try:
        from langfuse.callback import CallbackHandler

        handler = CallbackHandler(
            host=langfuse_host,
            public_key=langfuse_public,
            secret_key=langfuse_secret,
        )
        logger.info("Langfuse tracing enabled: %s", langfuse_host)

        # Also set LANGSMITH_TRACING for LangGraph native integration
        os.environ.setdefault("LANGSMITH_TRACING", "true")

        return [handler]
    except ImportError:
        logger.warning("Langfuse env vars set but langfuse package not installed")
        return []
    except Exception as e:
        logger.warning("Langfuse setup failed: %s", e)
        return []
