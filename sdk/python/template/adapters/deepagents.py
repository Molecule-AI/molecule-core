"""DeepAgents adaptor.

If your plugin defines a sub-agent, swap the import for a custom class
that calls ``ctx.register_subagent(name, spec)`` inside ``install()``.
"""
from molecule_plugin import AgentskillsAdaptor as Adaptor  # noqa: F401
