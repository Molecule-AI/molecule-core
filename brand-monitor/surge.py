"""Surge mode state machine.

Surge mode increases polling frequency from 30 min to 15 min for a
configurable window (default 6 h).  State is persisted in a JSON file so
restarts during an active surge window continue in surge mode.

Activation paths:
  1. Manual: call enable_surge_mode() (or the Slack slash command /surge-monitor on)
  2. Auto: any PR merged with a 'feat:' prefix calls enable_surge_mode()
"""

import json
import logging
import os
from datetime import datetime, timedelta, timezone

logger = logging.getLogger(__name__)

DEFAULT_SURGE_FILE = ".surge_state.json"
DEFAULT_SURGE_DURATION_HOURS = 6


class SurgeState:
    """Persist and query surge mode activation.

    Args:
        state_file: Path to the JSON state file.  Defaults to
            ``.surge_state.json`` in the current directory.
    """

    def __init__(self, state_file=DEFAULT_SURGE_FILE):
        self.state_file = state_file

    # ------------------------------------------------------------------
    # State I/O
    # ------------------------------------------------------------------

    def _load(self):
        """Return parsed state dict, or None if the file doesn't exist."""
        if not os.path.exists(self.state_file):
            return None
        with open(self.state_file) as fh:
            return json.load(fh)

    def _write(self, state):
        with open(self.state_file, "w") as fh:
            json.dump(state, fh, indent=2)

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def enable(self, duration_hours=DEFAULT_SURGE_DURATION_HOURS):
        """Activate surge mode for *duration_hours* hours.

        Writes ``.surge_state.json`` so that restarts re-enter surge mode.

        Args:
            duration_hours: How long surge mode stays active (default 6 h).
        """
        expires_at = (
            datetime.now(timezone.utc) + timedelta(hours=duration_hours)
        ).isoformat()
        state = {
            "active": True,
            "enabled_at": datetime.now(timezone.utc).isoformat(),
            "expires_at": expires_at,
            "duration_hours": duration_hours,
        }
        self._write(state)
        logger.info("Surge mode enabled for %dh — expires at %s", duration_hours, expires_at)

    def disable(self):
        """Deactivate surge mode and remove the state file."""
        if os.path.exists(self.state_file):
            os.remove(self.state_file)
        logger.info("Surge mode disabled")

    def is_active(self):
        """Return True if surge mode is currently active (and not expired).

        Side effect: auto-disables if the expiry timestamp has passed.
        """
        state = self._load()
        if not state:
            return False
        expires_at = datetime.fromisoformat(state["expires_at"])
        if datetime.now(timezone.utc) >= expires_at:
            logger.info("Surge mode expired — auto-disabling")
            self.disable()
            return False
        return True

    def check_expiry(self):
        """Auto-disable surge if its window has elapsed.

        Returns:
            bool: whether surge mode is still active after the check.
        """
        return self.is_active()

    def get_interval(self, normal_interval, surge_interval):
        """Return the appropriate polling interval in seconds.

        Args:
            normal_interval: Seconds to sleep in ambient mode.
            surge_interval:  Seconds to sleep while surge is active.

        Returns:
            int: surge_interval if surge is active, else normal_interval.
        """
        if self.is_active():
            return surge_interval
        return normal_interval
