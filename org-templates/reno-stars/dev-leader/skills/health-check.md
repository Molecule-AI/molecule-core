You are running a health check on all services for the Reno Stars automation system. Check AND fix issues when possible.

## Config
Read /Users/renostars/reno-star-business-intelligent/config/env.json for paths and credentials.

## CHECKS + AUTO-FIX

### 1. Launchd Cron Jobs
```bash
launchctl list | grep com.renostars
```
Verify all 6 jobs are loaded (seo-builder, seo-weekly-report, facebook-poster, memory-compactor, health-check, heartbeat).

**Auto-fix:** If any job is missing, reload it:
```bash
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/<label>.plist
```
If the plist doesn't exist, regenerate by running:
```bash
cd /Users/renostars/reno-star-business-intelligent && npx tsx src/setup.ts
```

Check last exit code — 0 is healthy, non-zero needs investigation.

### 2. Chrome CDP
```bash
curl -s http://host.docker.internal:9223/json/version
```
Should return JSON with Browser field.

**Auto-fix:** If Chrome CDP is down, relaunch:
```bash
# Chrome runs on host — connect via host.docker.internal:9223 (CDP proxy)
```
Wait 5 seconds, then verify again.

### 3. MCP Servers
Test each MCP server starts and responds to initialize:

```bash
# reno-stars-hub
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"health","version":"1.0"}}}' | perl -e 'alarm 15; exec @ARGV' -- /opt/homebrew/bin/npx tsx /Users/renostars/reno-star-business-intelligent/src/server.ts 2>/dev/null | head -1

# reno-stars-invoice
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"health","version":"1.0"}}}' | perl -e 'alarm 15; exec @ARGV' -- node --import /Users/renostars/.openclaw/workspace/reno-star-invoice-automation/node_modules/tsx/dist/esm/index.mjs /Users/renostars/.openclaw/workspace/reno-star-invoice-automation/src/mcp-server.ts 2>/dev/null | head -1

# playwright wrapper
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"health","version":"1.0"}}}' | perl -e 'alarm 15; exec @ARGV' -- node /Users/renostars/.openclaw/playwright-mcp-wrapper.js --cdp-endpoint http://host.docker.internal:9223 2>/dev/null | head -1
```

For each: check if response contains `"result"` with `"serverInfo"`. Mark PASS/FAIL.

**Auto-fix:** If an MCP server fails:
- Check if node_modules exist in its directory. If not, run `npm install` or `pnpm install`.
- Check if the script file exists. If not, run `git pull` in the repo.
- Report the fix attempt in the log.

### 4. Cron Log Health
Read the last entry from each cron log in /Users/renostars/reno-star-business-intelligent/data/cron-logs/:
- seo-builder.jsonl
- facebook-posts.jsonl
- memory-compactor.jsonl
- seo-weekly-report.jsonl
- health-check.jsonl
- heartbeat.jsonl

Check: last run timestamp (stale if >2x expected interval), last status (error = needs attention).

**Auto-fix:** If a job is stale (hasn't run in >2x its interval), check if it's still loaded in launchd. If loaded but not running, it may be stuck — try unloading and reloading:
```bash
launchctl bootout gui/$(id -u) ~/Library/LaunchAgents/<label>.plist
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/<label>.plist
```

### 5. Git Repos
For each project in config/env.json → projects:
```bash
git -C <path> status --porcelain
git -C <path> rev-list @{u}..HEAD --count 2>/dev/null
```

**Auto-fix:** If unpushed commits found, push them:
```bash
git -C <path> push
```
Do NOT auto-fix dirty working trees — just report them.

### 6. Disk Space
```bash
df -h /
```

**Auto-fix:** If <10% free, clean up known safe targets:
```bash
# Clear old Claude Code sessions (keep last 5)
ls -t ~/.claude/sessions/*.json 2>/dev/null | tail -n +6 | xargs rm -f
# Clear old cron stdout logs (keep last 1000 lines each)
for f in /Users/renostars/reno-star-business-intelligent/data/cron-logs/*.stdout.log; do
  tail -1000 "$f" > "$f.tmp" && mv "$f.tmp" "$f"
done
```
Report what was cleaned.

### 7. OpenClaw Gateway
Check that the old OpenClaw gateway is NOT running (it steals the Telegram bot):
```bash
launchctl list | grep ai.openclaw.gateway
```

**Auto-fix:** If it's running, stop it:
```bash
launchctl bootout gui/$(id -u) ~/Library/LaunchAgents/ai.openclaw.gateway.plist
```

## REPORT FORMAT
Output a structured report:

```
=== HEALTH CHECK ===
Timestamp: <ISO>

Cron Jobs (6):     [ALL LOADED / X MISSING]
Chrome CDP:        [PASS/FAIL]
MCP Servers (5):   [X/5 PASS]
Cron Logs:         [ALL FRESH / X STALE]
Git Repos:         [ALL CLEAN / X DIRTY]
Disk:              [PASS/FAIL] <used>%
OpenClaw Gateway:  [STOPPED / KILLED]

Fixes Applied:
  - <what was fixed>

Issues Remaining:
  - <what still needs manual attention>
```

## LOG
Append one JSON line to /Users/renostars/reno-star-business-intelligent/data/cron-logs/health-check.jsonl:
{"ts": "<ISO>", "job": "health-check", "status": "pass"|"warn"|"fail", "summary": "<one-line>", "checks_passed": <N>, "checks_total": <N>, "issues": [], "fixes": []}

## ON FAILURE
If any critical check fails AND auto-fix didn't resolve it, send a Telegram alert to the group:
```bash
BOT_TOKEN=$(jq -r '.telegram.bot_token' /Users/renostars/reno-star-business-intelligent/config/env.json)
CHAT_ID="-5219630660"  # RENO STARS bot group
curl -s -X POST "https://api.telegram.org/bot${BOT_TOKEN}/sendMessage" \
  -H "Content-Type: application/json" \
  -d "{\"chat_id\": \"${CHAT_ID}\", \"text\": \"⚠️ Health Check Alert\\n\\n<issues and fix attempts>\"}"
```

Only alert if auto-fix FAILED. If the fix worked, just log it — don't bother the group.
