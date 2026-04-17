"""Brand monitor — main poller entry point.

Entry point:
    python monitor.py

Environment variables (all required at startup):
    X_BEARER_TOKEN   — X API Bearer token
    X_API_KEY        — X API key (available for future OAuth use)
    X_API_SECRET     — X API secret
    SLACK_WEBHOOK_URL — Slack incoming webhook URL

Optional tuning:
    POLL_INTERVAL_SECONDS — ambient polling cadence in seconds (default: 1800 = 30 min)
    SURGE_DURATION_HOURS  — surge window length in hours (default: 6)
"""

import json
import logging
import os
import time
from datetime import datetime, timedelta, timezone

from slack_client import SlackClient
from surge import SurgeState
from x_client import XClient

logger = logging.getLogger(__name__)

# ------------------------------------------------------------------
# Constants
# ------------------------------------------------------------------

REQUIRED_ENV_VARS = ["X_BEARER_TOKEN", "X_API_KEY", "X_API_SECRET", "SLACK_WEBHOOK_URL"]

DEFAULT_STATE_FILE = ".monitor_state.json"

# Ambient cadence: 30 min per issue spec (configurable via env)
POLL_INTERVAL_SECONDS = int(os.environ.get("POLL_INTERVAL_SECONDS", "1800"))

# Surge cadence: fixed at 15 min
SURGE_INTERVAL_SECONDS = 900

# Surge window length (configurable via env)
SURGE_DURATION_HOURS = int(os.environ.get("SURGE_DURATION_HOURS", "6"))

# UTC hour at which the daily digest is sent
DIGEST_HOUR_UTC = 20


# ------------------------------------------------------------------
# Startup validation
# ------------------------------------------------------------------

def validate_env():
    """Raise EnvironmentError if any required env var is absent."""
    missing = [v for v in REQUIRED_ENV_VARS if not os.environ.get(v)]
    if missing:
        raise EnvironmentError(
            f"Missing required environment variable(s): {', '.join(missing)}"
        )


# ------------------------------------------------------------------
# Surge mode public entry point (callable from CI/CD on feat: PR merge)
# ------------------------------------------------------------------

def enable_surge_mode(duration_hours=None, state_file=None):
    """Enable surge mode.  Call this from CI/CD hooks on feat: PR merges.

    Args:
        duration_hours: Override for surge window length.  Defaults to the
            SURGE_DURATION_HOURS env var (or 6 h).
        state_file: Override path for .surge_state.json (mainly for tests).
    """
    hours = duration_hours if duration_hours is not None else SURGE_DURATION_HOURS
    kwargs = {}
    if state_file is not None:
        kwargs["state_file"] = state_file
    surge = SurgeState(**kwargs)
    surge.enable(hours)
    logger.info("enable_surge_mode: activated for %d hour(s)", hours)


# ------------------------------------------------------------------
# Monitor class
# ------------------------------------------------------------------

class Monitor:
    """Cron-style poller: fetches new X mentions and posts them to Slack.

    Args:
        state_file: Path to the JSON file that persists polling state
            (since_id, daily_count, etc.).  Defaults to
            ``.monitor_state.json`` in the current directory.
        surge_state_file: Path to the surge state JSON file.
    """

    def __init__(self, state_file=DEFAULT_STATE_FILE, surge_state_file=None):
        validate_env()
        self.x_client = XClient()
        self.slack_client = SlackClient()
        surge_kwargs = {}
        if surge_state_file is not None:
            surge_kwargs["state_file"] = surge_state_file
        self.surge = SurgeState(**surge_kwargs)
        self.state_file = state_file
        self.state = self._load_state()

    # ------------------------------------------------------------------
    # State persistence
    # ------------------------------------------------------------------

    def _load_state(self):
        if os.path.exists(self.state_file):
            with open(self.state_file) as fh:
                return json.load(fh)
        return {}

    def _save_state(self):
        with open(self.state_file, "w") as fh:
            json.dump(self.state, fh, indent=2)

    # ------------------------------------------------------------------
    # Core poll
    # ------------------------------------------------------------------

    def run_poll(self):
        """Fetch new tweets and post them to Slack.

        On first run (no saved since_id) backfills the last 24 h.
        Tracks the newest tweet ID so subsequent runs avoid duplicates.

        Returns:
            list: tweets posted this cycle (may be empty).
        """
        since_id = self.state.get("since_id")
        start_time = None

        if not since_id:
            # First run: backfill last 24 h
            start_time = (
                datetime.now(timezone.utc) - timedelta(hours=24)
            ).strftime("%Y-%m-%dT%H:%M:%SZ")
            logger.info("First run — backfilling last 24 h (start_time=%s)", start_time)

        tweets = self.x_client.search_recent(since_id=since_id, start_time=start_time)

        if tweets:
            self.slack_client.post_mentions(tweets)
            # X API returns tweets newest-first; store the top ID as next since_id
            self.state["since_id"] = tweets[0]["id"]

        return tweets

    # ------------------------------------------------------------------
    # Daily digest
    # ------------------------------------------------------------------

    def _should_send_digest(self):
        """True if it's 20:00 UTC and today's digest hasn't been sent yet."""
        now = datetime.now(timezone.utc)
        if now.hour != DIGEST_HOUR_UTC:
            return False
        today = now.strftime("%Y-%m-%d")
        return self.state.get("last_digest_date") != today

    def run_daily_digest(self):
        """Compile and post the daily summary to Slack, then reset the counter."""
        mention_count = self.state.get("daily_count", 0)
        self.slack_client.post_digest({"count": mention_count})
        self.state["daily_count"] = 0
        self.state["last_digest_date"] = datetime.now(timezone.utc).strftime("%Y-%m-%d")
        self._save_state()
        logger.info("Daily digest sent (count=%d)", mention_count)

    # ------------------------------------------------------------------
    # Main loop
    # ------------------------------------------------------------------

    def _run_once(self):
        """Execute one full polling cycle.

        Returns:
            int: seconds to sleep before the next cycle.
        """
        self.surge.check_expiry()
        tweets = self.run_poll()

        # Accumulate daily mention count
        self.state["daily_count"] = self.state.get("daily_count", 0) + len(tweets)
        self._save_state()

        if self._should_send_digest():
            self.run_daily_digest()

        return self.surge.get_interval(POLL_INTERVAL_SECONDS, SURGE_INTERVAL_SECONDS)

    def run(self):
        """Blocking main loop.  Runs until interrupted."""
        logger.info(
            "Brand monitor starting — ambient interval %ds, surge interval %ds",
            POLL_INTERVAL_SECONDS,
            SURGE_INTERVAL_SECONDS,
        )
        while True:
            try:
                interval = self._run_once()
            except Exception as exc:  # noqa: BLE001
                logger.error("Poll cycle failed: %s", exc)
                interval = POLL_INTERVAL_SECONDS
            logger.debug("Sleeping %ds until next poll", interval)
            time.sleep(interval)


# ------------------------------------------------------------------
# Entry point
# ------------------------------------------------------------------

if __name__ == "__main__":  # pragma: no cover
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(name)s — %(message)s",
    )
    monitor = Monitor()
    monitor.run()
