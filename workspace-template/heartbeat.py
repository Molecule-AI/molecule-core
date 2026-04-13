"""Heartbeat loop — alive signal + delegation status checker.

Every 30 seconds:
1. Send heartbeat to platform (alive signal with current_task, error_rate)
2. Check pending delegations — any results back?
3. Store completed delegation results for the agent to pick up

Resilient: recreates HTTP client on failure, auto-restarts on crash.
"""

import asyncio
import json
import logging
import os
import time
from pathlib import Path

import httpx

logger = logging.getLogger(__name__)

HEARTBEAT_INTERVAL = 30  # seconds
MAX_CONSECUTIVE_FAILURES = 10
MAX_SEEN_DELEGATION_IDS = 200
SELF_MESSAGE_COOLDOWN = 60  # seconds — minimum between self-messages to prevent loops
# Shared path — also used by cli_executor._read_delegation_results()
DELEGATION_RESULTS_FILE = os.environ.get("DELEGATION_RESULTS_FILE", "/tmp/delegation_results.jsonl")


class HeartbeatLoop:
    def __init__(self, platform_url: str, workspace_id: str):
        self.platform_url = platform_url
        self.workspace_id = workspace_id
        self.start_time = time.time()
        self.error_count = 0
        self.request_count = 0
        self.active_tasks = 0
        self.current_task = ""
        self.sample_error = ""
        self._task = None
        self._consecutive_failures = 0
        self._seen_delegation_ids: set[str] = set()
        self._last_self_message_time = 0.0
        self._parent_name: str | None = None  # Cached after first lookup

    @property
    def error_rate(self) -> float:
        if self.request_count == 0:
            return 0.0
        return self.error_count / self.request_count

    def record_error(self, error: str):
        self.error_count += 1
        self.request_count += 1
        self.sample_error = error

    def record_success(self):
        self.request_count += 1

    def start(self):
        self._task = asyncio.create_task(self._loop())
        self._task.add_done_callback(self._on_done)

    def _on_done(self, task):
        if not task.cancelled() and task.exception():
            logger.error("Heartbeat loop died: %s — restarting", task.exception())
            self._task = asyncio.create_task(self._loop())
            self._task.add_done_callback(self._on_done)

    async def stop(self):
        if self._task:
            self._task.cancel()
            try:
                await self._task
            except asyncio.CancelledError:
                pass

    async def _loop(self):
        while True:
            client = None
            try:
                client = httpx.AsyncClient(timeout=10.0)
                while True:
                    # 1. Send heartbeat (Phase 30.1: include auth header if token known)
                    try:
                        from platform_auth import auth_headers
                        await client.post(
                            f"{self.platform_url}/registry/heartbeat",
                            json={
                                "workspace_id": self.workspace_id,
                                "error_rate": self.error_rate,
                                "sample_error": self.sample_error,
                                "active_tasks": self.active_tasks,
                                "current_task": self.current_task,
                                "uptime_seconds": int(time.time() - self.start_time),
                            },
                            headers=auth_headers(),
                        )
                        self.error_count = 0
                        self.request_count = 0
                        self._consecutive_failures = 0
                    except Exception as e:
                        self._consecutive_failures += 1
                        if self._consecutive_failures <= 3 or self._consecutive_failures % MAX_CONSECUTIVE_FAILURES == 0:
                            logger.warning("Heartbeat failed (%d consecutive): %s", self._consecutive_failures, e)
                        if self._consecutive_failures >= MAX_CONSECUTIVE_FAILURES:
                            logger.info("Heartbeat: recreating HTTP client after %d failures", self._consecutive_failures)
                            try:
                                await client.aclose()
                            except Exception:
                                pass
                            break

                    # 2. Check delegation status
                    try:
                        await self._check_delegations(client)
                    except Exception as e:
                        logger.debug("Delegation check failed: %s", e)

                    await asyncio.sleep(HEARTBEAT_INTERVAL)

            except asyncio.CancelledError:
                raise
            except Exception as e:
                logger.error("Heartbeat loop error: %s — retrying in 30s", e)
                await asyncio.sleep(HEARTBEAT_INTERVAL)
            finally:
                if client:
                    try:
                        await client.aclose()
                    except Exception:
                        pass

    async def _check_delegations(self, client: httpx.AsyncClient):
        """Check for completed delegations and store results for the agent."""
        try:
            resp = await client.get(
                f"{self.platform_url}/workspaces/{self.workspace_id}/delegations"
            )
            if resp.status_code != 200:
                return

            delegations = resp.json()
            if not isinstance(delegations, list):
                return

            new_results = []
            for d in delegations:
                did = d.get("delegation_id", "")
                status = d.get("status", "")

                if not did or did in self._seen_delegation_ids:
                    continue

                if status in ("completed", "failed"):
                    self._seen_delegation_ids.add(did)
                    new_results.append({
                        "delegation_id": did,
                        "target_id": d.get("target_id", ""),
                        "source_id": d.get("source_id", ""),
                        "status": status,
                        "summary": d.get("summary", ""),
                        "response_preview": d.get("response_preview", ""),
                        "error": d.get("error", ""),
                        "timestamp": time.time(),
                    })

            # Evict old seen IDs if over limit
            if len(self._seen_delegation_ids) > MAX_SEEN_DELEGATION_IDS:
                # Keep most recent half
                self._seen_delegation_ids = set(list(self._seen_delegation_ids)[MAX_SEEN_DELEGATION_IDS // 2:])

            if new_results:
                # Append to results file for context injection on next message
                with open(DELEGATION_RESULTS_FILE, "a") as f:
                    for r in new_results:
                        f.write(json.dumps(r) + "\n")
                logger.info("Heartbeat: %d new delegation results — triggering self-message", len(new_results))

                # Build a summary message for the agent
                summary_lines = []
                for r in new_results:
                    line = f"- [{r['status']}] {r['summary'][:80]}"
                    if r.get("response_preview"):
                        line += f"\n  Response: {r['response_preview'][:200]}"
                    if r.get("error"):
                        line += f"\n  Error: {r['error'][:100]}"
                    summary_lines.append(line)

                # Look up parent workspace (cached after first call)
                if self._parent_name is None:
                    try:
                        parent_resp = await client.get(
                            f"{self.platform_url}/workspaces/{self.workspace_id}"
                        )
                        if parent_resp.status_code == 200:
                            parent_id = parent_resp.json().get("parent_id", "")
                            if parent_id:
                                parent_info = await client.get(
                                    f"{self.platform_url}/workspaces/{parent_id}"
                                )
                                if parent_info.status_code == 200:
                                    self._parent_name = parent_info.json().get("name", "")
                        if self._parent_name is None:
                            self._parent_name = ""  # No parent — cache empty
                    except Exception:
                        pass  # Will retry next cycle
                parent_name = self._parent_name or ""

                report_instruction = ""
                if parent_name:
                    report_instruction = (
                        f"\n\nIMPORTANT: Report these results back to your parent '{parent_name}' "
                        f"by delegating a summary to them. Use delegate_task or delegate_task_async "
                        f"with a concise status report. Also use send_message_to_user to notify the user."
                    )
                else:
                    report_instruction = (
                        "\n\nReport results using send_message_to_user to notify the user."
                    )

                trigger_msg = (
                    "Delegation results are ready. Review them and take appropriate action:\n"
                    + "\n".join(summary_lines)
                    + report_instruction
                )

                # Send A2A self-message to wake the agent.
                # Minimum 60s between self-messages to avoid spam, but always send
                # when there are genuinely NEW results to process.
                now = time.time()
                if now - self._last_self_message_time < SELF_MESSAGE_COOLDOWN:
                    logger.debug("Heartbeat: self-message cooldown (60s), will retry next cycle")
                else:
                    self._last_self_message_time = now
                    try:
                        await client.post(
                            f"{self.platform_url}/workspaces/{self.workspace_id}/a2a",
                            json={
                                "method": "message/send",
                                "params": {
                                    "message": {
                                        "role": "user",
                                        "parts": [{"type": "text", "text": trigger_msg}],
                                    },
                                },
                            },
                            timeout=120.0,
                        )
                        logger.info("Heartbeat: self-message sent to process delegation results")
                    except Exception as e:
                        logger.warning("Heartbeat: failed to send self-message: %s", e)

                # Also push notification to user via canvas
                for r in new_results:
                    try:
                        msg = f"Delegation {r['status']}: {r['summary'][:100]}"
                        if r.get("response_preview"):
                            msg += f"\nResult: {r['response_preview'][:200]}"
                        await client.post(
                            f"{self.platform_url}/workspaces/{self.workspace_id}/notify",
                            json={"message": msg, "type": "delegation_result"},
                        )
                    except Exception:
                        pass

        except Exception as e:
            logger.debug("Delegation check error: %s", e)
