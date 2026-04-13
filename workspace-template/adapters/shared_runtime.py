"""Shared runtime helpers for A2A-backed workspace executors."""

from __future__ import annotations

from typing import Any

from a2a.server.agent_execution import RequestContext


def _extract_part_text(part) -> str:
    """Extract text from a message part, handling dicts and A2A objects."""
    if isinstance(part, dict):
        text = part.get("text", "")
        if text:
            return text
        root = part.get("root")
        if isinstance(root, dict):
            return root.get("text", "")
        return ""
    if hasattr(part, "text") and part.text:
        return part.text
    if hasattr(part, "root") and hasattr(part.root, "text") and part.root.text:
        return part.root.text
    return ""


def extract_message_text(context_or_parts) -> str:
    """Extract concatenated plain text from A2A message parts."""
    parts = getattr(getattr(context_or_parts, "message", None), "parts", None)
    if parts is None:
        parts = context_or_parts
    return " ".join(
        text for part in (parts or []) if (text := _extract_part_text(part))
    ).strip()


def extract_history(context: RequestContext) -> list[tuple[str, str]]:
    """Extract conversation history from A2A request metadata."""
    messages: list[tuple[str, str]] = []
    request = getattr(context, "request", None)
    metadata = getattr(request, "metadata", None) if request else None
    if not isinstance(metadata, dict):
        metadata = getattr(context, "metadata", None) or {}
    history = metadata.get("history", []) if isinstance(metadata, dict) else []
    if not isinstance(history, list):
        return messages

    for entry in history:
        if not isinstance(entry, dict):
            continue
        role = entry.get("role", "user")
        parts = entry.get("parts", [])
        text = " ".join(
            text for part in (parts or []) if (text := _extract_part_text(part))
        ).strip()
        if text:
            mapped_role = "human" if role == "user" else "ai"
            messages.append((mapped_role, text))
    return messages


def format_conversation_history(history: list[tuple[str, str]]) -> str:
    """Render `(role, text)` history into a stable human-readable transcript."""
    return "\n".join(
        f"{'User' if role == 'human' else 'Agent'}: {text}" for role, text in history
    )


def build_task_text(user_message: str, history: list[tuple[str, str]]) -> str:
    """Build a single task/request string with optional prepended conversation history."""
    if not history:
        return user_message
    transcript = format_conversation_history(history)
    return f"Conversation so far:\n{transcript}\n\nCurrent request: {user_message}"


def append_peer_guidance(
    base_text: str | None,
    peers_info: str,
    *,
    default_text: str,
    tool_name: str,
) -> str:
    """Append peer guidance text when peers are available."""
    text = (base_text or default_text).strip()
    if peers_info:
        text += f"\n\n## Peers\n{peers_info}\nUse {tool_name} to communicate with them."
    return text


def summarize_peer_cards(peers: list[dict[str, Any]]) -> list[dict[str, Any]]:
    """Return compact peer metadata for prompt rendering."""
    summaries: list[dict[str, Any]] = []
    for peer in peers:
        agent_card = peer.get("agent_card")
        if not agent_card:
            continue
        if isinstance(agent_card, str):
            try:
                import json

                agent_card = json.loads(agent_card)
            except Exception:
                continue
        if not isinstance(agent_card, dict):
            continue

        skills = agent_card.get("skills", [])
        summaries.append(
            {
                "id": peer.get("id", "unknown"),
                "name": agent_card.get("name", peer.get("name", "Unknown")),
                "status": peer.get("status", "unknown"),
                "skills": [
                    s.get("name", s.get("id", ""))
                    for s in skills
                    if isinstance(s, dict)
                ],
            }
        )
    return summaries


def build_peer_section(
    peers: list[dict[str, Any]],
    *,
    heading: str = "## Your Peers (workspaces you can delegate to)",
    instruction: str = (
        "Use the `delegate_to_workspace` tool to send tasks to peers. "
        "Only delegate to peers listed above."
    ),
) -> str:
    """Render a stable peer section for system prompts."""
    summaries = summarize_peer_cards(peers)
    if not summaries:
        return ""

    parts = [heading, ""]
    for peer in summaries:
        parts.append(f"- **{peer['name']}** (id: `{peer['id']}`, status: {peer['status']})")
        if peer["skills"]:
            parts.append(f"  Skills: {', '.join(peer['skills'])}")
        parts.append("")
    parts.append(instruction)
    return "\n".join(parts)


def brief_task(text: str, limit: int = 60) -> str:
    """Create a short human-readable task label for the heartbeat banner."""
    return text[:limit] + ("..." if len(text) > limit else "")


async def set_current_task(heartbeat: Any, task: str) -> None:
    """Update current task on heartbeat and push immediately to platform.

    The heartbeat loop only fires every 30s, so quick tasks would finish
    before the canvas ever sees them. Setting a task pushes immediately.
    Clearing a task only updates the heartbeat object — the next heartbeat
    cycle will broadcast the clear, keeping the task visible longer.
    """
    if heartbeat:
        heartbeat.current_task = task
        heartbeat.active_tasks = 1 if task else 0

    # Only push immediately when SETTING a task (not clearing)
    # Clearing is handled by the next heartbeat cycle, which keeps
    # the task visible on the canvas for quick A2A responses
    if not task:
        return

    import os
    workspace_id = os.environ.get("WORKSPACE_ID", "")
    platform_url = os.environ.get("PLATFORM_URL", "")
    if workspace_id and platform_url:
        try:
            import httpx
            async with httpx.AsyncClient(timeout=3.0) as client:
                await client.post(
                    f"{platform_url}/registry/heartbeat",
                    json={
                        "workspace_id": workspace_id,
                        "current_task": task,
                        "active_tasks": 1,
                        "error_rate": 0,
                        "sample_error": "",
                        "uptime_seconds": 0,
                    },
                )
        except Exception:
            pass  # Best-effort
