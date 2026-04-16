"""Hermes adapter — Nous Research Hermes models via Nous Portal or OpenRouter.

Uses the OpenAI-compatible client (openai>=1.0.0) to communicate with
either the Nous Portal directly (HERMES_API_KEY) or OpenRouter as a
fallback (OPENROUTER_API_KEY).
"""
import os

from adapters.base import BaseAdapter, AdapterConfig


class HermesAdapter(BaseAdapter):

    @staticmethod
    def name() -> str:
        return "hermes"

    @staticmethod
    def display_name() -> str:
        return "Hermes (Nous Research)"

    @staticmethod
    def description() -> str:
        return "Hermes models via Nous Portal or OpenRouter — openai>=1.0.0 compatible client"

    @staticmethod
    def get_config_schema() -> dict:
        return {
            "model": {
                "type": "string",
                "description": (
                    "Hermes model ID (e.g. nousresearch/hermes-3-llama-3.1-405b for OpenRouter "
                    "or hermes-3-llama-3.1-405b for Nous Portal)"
                ),
            },
        }

    async def setup(self, config: AdapterConfig) -> None:  # pragma: no cover
        try:
            import openai  # noqa: F401
        except ImportError as e:
            raise RuntimeError(
                "Hermes adapter requires openai>=1.0.0 — "
                "install with: pip install 'openai>=1.0.0'"
            ) from e

    async def create_executor(self, config: AdapterConfig):  # pragma: no cover
        """Create and return a HermesA2AExecutor using key resolution from env/config."""
        from .executor import create_executor, HermesA2AExecutor

        # Resolve API key: prefer workspace secrets (runtime_config), then env vars
        hermes_api_key = config.runtime_config.get("hermes_api_key") or None

        # Phase 3 escalation ladder — read from runtime_config.escalation_ladder
        # if present. The platform's org importer copies the ladder from
        # org.yaml (runtime_config.escalation_ladder) into the container's
        # /configs/config.yaml, and the workspace-template loader surfaces it
        # here. Empty / missing = single-shot behaviour (unchanged from pre-
        # Phase-3). See adapters.hermes.escalation for classification rules.
        escalation_ladder = config.runtime_config.get("escalation_ladder") or None

        executor = create_executor(
            hermes_api_key=hermes_api_key,
            config_path=config.config_path,  # Phase 2d-i: system-prompt.md injection
            escalation_ladder=escalation_ladder,
        )

        # Override model from config if provided
        model = config.model
        if ":" in model:
            _, model = model.split(":", 1)
        if model:
            executor.model = model

        executor._heartbeat = config.heartbeat
        return executor
