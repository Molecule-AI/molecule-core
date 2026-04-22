"""Memory consolidation loop.

When an agent is idle (no active tasks for a configurable period),
the consolidation loop wakes up and summarizes noisy local memory
entries into dense, high-value knowledge facts.

Similar to human sleep consolidation — raw scratchpad entries get
compressed into reusable knowledge.
"""

import asyncio
import logging
import os

import httpx

from platform_auth import auth_headers

logger = logging.getLogger(__name__)

if os.path.exists("/.dockerenv") or os.environ.get("DOCKER_VERSION"):
    PLATFORM_URL = os.environ.get("PLATFORM_URL", "http://host.docker.internal:8080")
else:
    PLATFORM_URL = os.environ.get("PLATFORM_URL", "http://localhost:8080")
# Lazy WORKSPACE_ID — raises only when actually used in a workspace context.
# Allows test collection (pytest import) to succeed without WORKSPACE_ID set,
# while preserving the guard for production usage.
_WORKSPACE_ID_raw = os.environ.get("WORKSPACE_ID")


def __getattr__(name: str):
    if name == "WORKSPACE_ID":
        if not _WORKSPACE_ID_raw:
            raise RuntimeError("WORKSPACE_ID environment variable is required but not set")
        return _WORKSPACE_ID_raw
    raise AttributeError(f"module {__name__!r} has no attribute {name!r}")
CONSOLIDATION_INTERVAL = float(os.environ.get("CONSOLIDATION_INTERVAL", "300"))  # 5 min
CONSOLIDATION_THRESHOLD = int(os.environ.get("CONSOLIDATION_THRESHOLD", "10"))  # min memories before consolidating


class ConsolidationLoop:
    """Background loop that consolidates local memories when idle."""

    def __init__(self, agent=None):
        self.agent = agent
        self._running = False

    async def start(self):
        """Start the consolidation loop."""
        self._running = True
        logger.info("Memory consolidation loop started (interval=%ss, threshold=%d)",
                     CONSOLIDATION_INTERVAL, CONSOLIDATION_THRESHOLD)

        while self._running:
            await asyncio.sleep(CONSOLIDATION_INTERVAL)

            if not self._running:
                break

            try:
                await self._consolidate()
            except Exception as e:
                logger.warning("Consolidation error: %s", e)

    async def _consolidate(self):
        """Check if consolidation is needed and run it."""
        async with httpx.AsyncClient(timeout=10.0) as client:
            # Fetch local memories
            resp = await client.get(
                f"{PLATFORM_URL}/workspaces/{WORKSPACE_ID}/memories",
                params={"scope": "LOCAL"},
                headers=auth_headers(),
            )
            if resp.status_code != 200:
                return

            memories = resp.json()
            if len(memories) < CONSOLIDATION_THRESHOLD:
                return

            logger.info("Consolidating %d local memories", len(memories))

            # Build a summary of all local memories
            contents = [m["content"] for m in memories]
            summary_prompt = (
                "Summarize the following workspace memories into 3-5 key facts. "
                "Each fact should be a single, clear sentence capturing the most "
                "important and reusable knowledge:\n\n"
                + "\n".join(f"- {c}" for c in contents)
            )

            # Use the agent to generate the summary if available
            summary = ""
            if self.agent:
                try:
                    result = await self.agent.ainvoke(
                        {"messages": [("user", summary_prompt)]},
                        config={"configurable": {"thread_id": "consolidation"}},
                    )
                    messages = result.get("messages", [])
                    summary = ""
                    for msg in reversed(messages):
                        content = getattr(msg, "content", "")
                        if isinstance(content, str) and content.strip():
                            msg_type = getattr(msg, "type", "")
                            if msg_type != "human":
                                summary = content
                                break

                    if summary:
                        # Store consolidated summary as a TEAM memory — only delete originals if POST succeeds
                        resp = await client.post(
                            f"{PLATFORM_URL}/workspaces/{WORKSPACE_ID}/memories",
                            json={"content": f"[Consolidated] {summary}", "scope": "TEAM"},
                            headers=auth_headers(),
                        )
                        if resp.status_code in (200, 201):
                            # Safe to delete originals — consolidated version is saved
                            for m in memories:
                                await client.delete(
                                    f"{PLATFORM_URL}/workspaces/{WORKSPACE_ID}/memories/{m['id']}",
                                    headers=auth_headers(),
                                )
                            logger.info("Consolidated %d memories into team knowledge", len(memories))
                        else:
                            logger.warning("Consolidation POST failed (status %d) — keeping originals", resp.status_code)
                except Exception as e:
                    logger.error(
                        "CONSOLIDATION: Agent summarization failed (rate limit? model error?): %s. "
                        "Falling back to simple concatenation.", e
                    )
                    # Fall through to concatenation below

            # Fallback: concatenate without agent summarization
            if not (self.agent and summary):
                combined = " | ".join(contents[:20])
                await client.post(
                    f"{PLATFORM_URL}/workspaces/{WORKSPACE_ID}/memories",
                    json={"content": f"[Consolidated] {combined}", "scope": "TEAM"},
                    headers=auth_headers(),
                )
                logger.info("Consolidated %d memories via concatenation fallback", len(memories))

    def stop(self):
        self._running = False
