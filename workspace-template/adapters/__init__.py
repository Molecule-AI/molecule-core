"""Adapter registry — discovers and loads agent infrastructure adapters."""

import importlib
import logging
from pathlib import Path
from .base import BaseAdapter, AdapterConfig

logger = logging.getLogger(__name__)

_ADAPTER_CACHE: dict[str, type[BaseAdapter]] = {}


def discover_adapters() -> dict[str, type[BaseAdapter]]:
    """Scan subdirectories for adapter modules. Each must export an Adapter class."""
    if _ADAPTER_CACHE:
        return _ADAPTER_CACHE

    adapters_dir = Path(__file__).parent
    for entry in sorted(adapters_dir.iterdir()):
        if not entry.is_dir() or entry.name.startswith("_"):
            continue
        try:
            mod = importlib.import_module(f"adapters.{entry.name}")
            adapter_cls = getattr(mod, "Adapter", None)
            if adapter_cls and issubclass(adapter_cls, BaseAdapter):
                _ADAPTER_CACHE[adapter_cls.name()] = adapter_cls
                logger.debug(f"Loaded adapter: {adapter_cls.name()} ({adapter_cls.display_name()})")
        except Exception as e:
            # Log but don't crash — adapter may have uninstalled deps
            logger.debug(f"Skipped adapter {entry.name}: {e}")

    return _ADAPTER_CACHE


def get_adapter(runtime: str) -> type[BaseAdapter]:
    """Get adapter class by runtime name. Raises KeyError if not found."""
    adapters = discover_adapters()
    if runtime not in adapters:
        available = ", ".join(sorted(adapters.keys()))
        raise KeyError(f"Unknown runtime '{runtime}'. Available: {available}")
    return adapters[runtime]


def list_adapters() -> list[dict]:
    """Return metadata for all discovered adapters (for API/UI)."""
    adapters = discover_adapters()
    return [
        {
            "name": cls.name(),
            "display_name": cls.display_name(),
            "description": cls.description(),
            "config_schema": cls.get_config_schema(),
        }
        for cls in adapters.values()
    ]


__all__ = ["BaseAdapter", "AdapterConfig", "get_adapter", "list_adapters", "discover_adapters"]
