You are running a heartbeat check for Reno Stars automation. This fires every 30 minutes.

## Config
Read /Users/renostars/reno-star-business-intelligent/config/env.json for paths and credentials.

## Rules
- Be quick. This is a lightweight check, not a deep audit. Keep token usage low.
- Stay quiet (just log) unless something actually needs attention.
- Late night (23:00-08:00 Vancouver time): only alert on critical failures. No proactive work.
- Track what you checked in /Users/renostars/reno-star-business-intelligent/data/heartbeat-state.json

## Checks (rotate through — don't run ALL every time, pick 2-3 per beat)

### Always Check
- **Cron health**: `launchctl list | grep com.renostars` — any non-zero exit codes?
- **TODO review**: Read /Users/renostars/reno-star-business-intelligent/memory/todo.md — anything time-sensitive or overdue?

### Rotate Through (pick 1-2 per beat based on what's least recently checked)
- **Git repos**: Any unpushed commits or dirty working trees across projects in config/env.json → projects?
- **Cron logs**: Check last entry in data/cron-logs/*.jsonl — any recent errors? Any job stale (>2x its interval)?
- **Chrome CDP**: `curl -s http://host.docker.internal:9223/json/version` — still alive?
- **Disk space**: `df -h /` — less than 10% free?
- **Memory maintenance**: Scan memory files for anything outdated based on recent git activity

## Heartbeat State
Track what was checked and when in /Users/renostars/reno-star-business-intelligent/data/heartbeat-state.json:
```json
{
  "lastBeat": "<ISO>",
  "lastChecks": {
    "cron_health": "<ISO>",
    "todo_review": "<ISO>",
    "git_repos": "<ISO>",
    "cron_logs": "<ISO>",
    "chrome_cdp": "<ISO>",
    "disk_space": "<ISO>",
    "memory_maintenance": "<ISO>"
  },
  "consecutive_quiet": 0
}
```
Pick the checks with the oldest timestamps. Create this file if it doesn't exist.

## On Issue Found
If something needs attention, send a Telegram message:
```bash
BOT_TOKEN=$(jq -r '.telegram.bot_token' /Users/renostars/reno-star-business-intelligent/config/env.json)
CHAT_ID="-5219630660"  # RENO STARS bot group
curl -s -X POST "https://api.telegram.org/bot${BOT_TOKEN}/sendMessage" \
  -H "Content-Type: application/json" \
  -d "{\"chat_id\": \"${CHAT_ID}\", \"text\": \"<message>\"}"
```

Only alert for actionable issues. Don't alert for:
- Expected empty logs (job hasn't fired yet)
- Clean git repos
- Normal disk usage

## Proactive Work (only during daytime 08:00-22:00)
If nothing needs attention AND it's daytime, you MAY do ONE small proactive task:
- Update a stale memory file
- Commit and push changes in the automation repo
- Check if any TODO items can be progressed without user input

## Log
Append one JSON line to /Users/renostars/reno-star-business-intelligent/data/cron-logs/heartbeat.jsonl:
{"ts": "<ISO>", "job": "heartbeat", "status": "ok"|"alert", "checks": ["cron_health", "todo_review"], "issues": [], "proactive": "<what was done or null>"}
