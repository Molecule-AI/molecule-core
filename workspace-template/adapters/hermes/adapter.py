"""Hermes adapter stub — full executor implementation ships in PR 2.

PR 1 shell: registers 'hermes' with discover_adapters() so the adapter
catalogue is complete. setup() validates the openai dep is present.
create_executor() raises NotImplementedError until PR 2 lands.
"""
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

    async def setup(self, config: AdapterConfig) -> None:  # pragma: no cover
        try:
            import openai  # noqa: F401
        except ImportError as e:
            raise RuntimeError(
                "Hermes adapter requires openai>=1.0.0 — "
                "install with: pip install 'openai>=1.0.0'"
            ) from e

    async def create_executor(self, config: AdapterConfig):  # pragma: no cover
        raise NotImplementedError(
            "HermesAdapter.create_executor not yet implemented — ships in PR 2"
        )
