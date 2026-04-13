"""WebSocket subscriber for platform events.

Subscribes to the platform WebSocket with X-Workspace-ID header
so the workspace only receives events about reachable peers.
Triggers system prompt rebuild on relevant peer changes.
"""

import asyncio
import json
import logging

import httpx

logger = logging.getLogger(__name__)

# Events that should trigger a system prompt rebuild
REBUILD_EVENTS = {
    "WORKSPACE_ONLINE",
    "WORKSPACE_OFFLINE",
    "WORKSPACE_EXPANDED",
    "WORKSPACE_COLLAPSED",
    "WORKSPACE_REMOVED",
    "AGENT_CARD_UPDATED",
}


class PlatformEventSubscriber:
    """Subscribes to platform WebSocket for peer events."""

    def __init__(
        self,
        platform_url: str,
        workspace_id: str,
        on_peer_change=None,
    ):
        self.ws_url = platform_url.replace("http://", "ws://").replace("https://", "wss://") + "/ws"
        self.workspace_id = workspace_id
        self.on_peer_change = on_peer_change
        self._running = False
        self._reconnect_delay = 1.0

    async def start(self):
        """Connect to platform WebSocket with exponential backoff reconnect."""
        self._running = True

        while self._running:
            try:
                await self._connect()
            except Exception as e:
                if not self._running:
                    break
                logger.warning("WebSocket disconnected: %s. Reconnecting in %.0fs...", e, self._reconnect_delay)
                await asyncio.sleep(self._reconnect_delay)
                self._reconnect_delay = min(self._reconnect_delay * 2, 30.0)

    async def _connect(self):
        """Establish WebSocket connection and process events."""
        try:
            import websockets
        except ImportError:
            logger.warning("websockets package not installed, skipping event subscription")
            self._running = False
            return

        headers = {"X-Workspace-ID": self.workspace_id}
        logger.info("Connecting to platform WebSocket: %s", self.ws_url)

        async with websockets.connect(self.ws_url, additional_headers=headers) as ws:
            self._reconnect_delay = 1.0  # Reset on successful connect
            logger.info("Platform WebSocket connected")

            async for message in ws:
                try:
                    event = json.loads(message)
                    event_type = event.get("event", "")

                    if event_type in REBUILD_EVENTS:
                        logger.info("Peer event: %s for workspace %s",
                                    event_type, event.get("workspace_id", ""))
                        if self.on_peer_change:
                            await self.on_peer_change(event)
                except json.JSONDecodeError:
                    continue
                except Exception as e:
                    logger.warning("Error processing event: %s", e)

    def stop(self):
        self._running = False
