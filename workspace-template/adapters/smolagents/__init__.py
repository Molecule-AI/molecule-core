"""Smolagents adapter for Molecule AI workspace runtime.

Provides env sanitization and safe executor primitives for use with
HuggingFace's smolagents library.

Quick start::

    from adapters.smolagents.env_sanitize import make_safe_env, SafeLocalPythonExecutor
"""

__all__ = ["make_safe_env", "SafeLocalPythonExecutor"]
