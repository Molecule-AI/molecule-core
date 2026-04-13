"""Claude Code adaptor.

For most plugins the generic filesystem installer is enough — it copies
skill dirs to /configs/skills/ and appends rules to CLAUDE.md. Replace
with a custom class if you need to register runtime tools or sub-agents.
"""
from molecule_plugin import AgentskillsAdaptor as Adaptor  # noqa: F401
