# Molecule AI Brand Monitor

A cron-based X API v2 poller that posts new brand mentions of **Molecule AI** to Slack `#brand-monitoring`.

Features:
- Smart query filter (from issue #549) suppresses drug-discovery SEO noise
- Deduplication via `since_id` — never posts the same tweet twice
- First run automatically backfills the last 24 hours
- **Surge mode** — 15-min polling for launch days / crisis windows (see below)
- `@here` alert when engagement > 10 or a competitor name appears
- Daily digest at 20:00 UTC

---

## Setup

### 1. Install dependencies

```bash
cd brand-monitor
pip install -r requirements.txt
```

### 2. Set environment variables

| Variable | Required | Description |
|---|---|---|
| `X_BEARER_TOKEN` | ✅ | X API Bearer token (from the Developer Portal) |
| `X_API_KEY` | ✅ | X API key (available for future OAuth use) |
| `X_API_SECRET` | ✅ | X API secret |
| `SLACK_WEBHOOK_URL` | ✅ | Slack incoming webhook URL for `#brand-monitoring` |
| `POLL_INTERVAL_SECONDS` | optional | Ambient polling cadence (default: `1800` = 30 min) |
| `SURGE_DURATION_HOURS` | optional | Surge window length in hours (default: `6`) |

For local development, create a `.env` file (never commit it):

```bash
X_BEARER_TOKEN=AAA...
X_API_KEY=BBB...
X_API_SECRET=CCC...
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/...
```

> **TODO (DevOps):** Provision `X_BEARER_TOKEN`, `X_API_KEY`, `X_API_SECRET`, and `SLACK_WEBHOOK_URL`
> as workspace secrets. The X Developer App credentials are pending approval — blocked on that before
> the monitor can run in production.

### 3. Run

```bash
python monitor.py
```

The monitor logs to stdout and polls until interrupted (Ctrl-C or process signal).

---

## Polling Cadence

| Mode | Interval | How long |
|---|---|---|
| **Ambient** | 30 min (`POLL_INTERVAL_SECONDS`) | Continuous |
| **Surge** | 15 min (fixed) | `SURGE_DURATION_HOURS` (default 6 h) |

---

## Surge Mode

Surge mode temporarily increases the polling frequency to 15 minutes for a configurable window (default 6 hours). State is persisted in `.surge_state.json` — if the process restarts during a surge window, it picks back up automatically.

### Activating manually (Slack slash command)

> **TODO:** Configure the Slack app with a `/surge-monitor` slash command that calls the
> `enable_surge_mode()` Python function (or a thin wrapper HTTP endpoint). The Slack app
> configuration is a separate step; the state machine here is ready.

When the command is wired up:
```
/surge-monitor on        # enable for default 6 h
/surge-monitor on 12h    # enable for 12 h
/surge-monitor off       # deactivate immediately
```

### Auto-trigger on `feat:` PR merge

In your CI/CD pipeline (e.g. GitHub Actions), call `enable_surge_mode()` when a PR with a `feat:` prefix is merged:

```python
# In a post-merge CI step:
import sys
sys.path.insert(0, "brand-monitor")
from monitor import enable_surge_mode
enable_surge_mode()   # activates for SURGE_DURATION_HOURS
```

Or from the shell:
```bash
python -c "from monitor import enable_surge_mode; enable_surge_mode()"
```

### Deactivation

Surge mode deactivates automatically when its window expires. To force early deactivation:

```python
from surge import SurgeState
SurgeState().disable()
```

---

## Tests

```bash
cd brand-monitor
pip install -r requirements.txt
pytest test_monitor.py -v --cov=. --cov-report=term-missing --cov-fail-under=100
```

All HTTP calls are mocked — no live credentials needed in CI.

---

## Gitignored runtime files

- `.surge_state.json` — surge mode state
- `.monitor_state.json` — polling state (since_id, daily counts)

---

## API Cost Estimate

X API pay-per-use: **$0.005 / tweet read**

| Scenario | Reads/month | Est. cost |
|---|---|---|
| Ambient (30 min), ~5 mentions/day | ~150 | $0.75 |
| Surge (15 min) for 6 h, 10 surge events/month | ~300 extra | $1.50 |
| **Total estimate** | **~450–800** | **$2–4/month** |
