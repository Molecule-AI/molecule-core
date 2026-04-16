"""Adapter registry shim.

Adapters extracted to standalone repos (molecule-ai-workspace-template-*).
ADAPTER_MODULE env var is the primary discovery mechanism in production.
This shim provides backward-compatible imports for local dev + tests.
"""
import importlib
import os
import logging
from adapter_base import BaseAdapter, AdapterConfig

logger = logging.getLogger(__name__)

def get_adapter(runtime: str) -> type[BaseAdapter]:
    adapter_module = os.environ.get("ADAPTER_MODULE")
    if adapter_module:
        mod = importlib.import_module(adapter_module)
        return getattr(mod, "Adapter")
    raise KeyError(
        f"No ADAPTER_MODULE set for runtime '{runtime}'. "
        "Adapters now live in standalone template repos."
    )
