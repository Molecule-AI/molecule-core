"""File watcher for hot-reloading skills and config changes.

Monitors the config directory for file changes and triggers
agent rebuild + Agent Card update broadcast.
"""

import asyncio
import hashlib
import logging
import os
from pathlib import Path

import httpx

logger = logging.getLogger(__name__)

DEBOUNCE_SECONDS = 2.0
POLL_INTERVAL = 3.0  # seconds between filesystem checks


class ConfigWatcher:
    """Watches the config directory for changes and triggers reload callbacks."""

    def __init__(
        self,
        config_path: str,
        platform_url: str,
        workspace_id: str,
        on_reload=None,
    ):
        self.config_path = config_path
        self.platform_url = platform_url
        self.workspace_id = workspace_id
        self.on_reload = on_reload
        self._file_hashes: dict[str, str] = {}
        self._running = False

    def _hash_file(self, path: str) -> str:
        try:
            return hashlib.md5(Path(path).read_bytes()).hexdigest()
        except (OSError, IOError):
            return ""

    def _scan_hashes(self) -> dict[str, str]:
        """Scan all files in config directory and return hash map."""
        hashes = {}
        for root, _, files in os.walk(self.config_path):
            for fname in files:
                if fname.startswith("."):
                    continue
                fpath = os.path.join(root, fname)
                rel = os.path.relpath(fpath, self.config_path)
                hashes[rel] = self._hash_file(fpath)
        return hashes

    def _detect_changes(self) -> list[str]:
        """Compare current state with cached hashes, return changed files."""
        current = self._scan_hashes()
        changed = []

        for path, h in current.items():
            if path not in self._file_hashes or self._file_hashes[path] != h:
                changed.append(path)

        for path in self._file_hashes:
            if path not in current:
                changed.append(path)

        self._file_hashes = current
        return changed

    async def _notify_platform(self, agent_card: dict):
        """Push updated Agent Card to the platform."""
        try:
            async with httpx.AsyncClient(timeout=10.0) as client:
                await client.post(
                    f"{self.platform_url}/registry/update-card",
                    json={
                        "workspace_id": self.workspace_id,
                        "agent_card": agent_card,
                    },
                )
                logger.info("Agent Card updated via platform")
        except Exception as e:
            logger.warning("Failed to update Agent Card: %s", e)

    async def start(self):
        """Start watching for changes in a background loop."""
        self._running = True
        self._file_hashes = self._scan_hashes()
        logger.info("Config watcher started for %s", self.config_path)

        while self._running:
            await asyncio.sleep(POLL_INTERVAL)

            changed = self._detect_changes()
            if not changed:
                continue

            logger.info("Config changes detected: %s", changed)

            # Debounce — wait for writes to settle
            await asyncio.sleep(DEBOUNCE_SECONDS)

            # Re-scan after debounce (more changes may have occurred)
            self._detect_changes()

            # Trigger reload callback
            if self.on_reload:
                try:
                    await self.on_reload()
                except Exception as e:
                    logger.error("Reload callback failed: %s", e)

    def stop(self):
        self._running = False
