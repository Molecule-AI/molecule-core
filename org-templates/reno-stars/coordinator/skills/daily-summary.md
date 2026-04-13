# Daily Summary — Reno Stars

Generate a concise daily summary of everything that was accomplished today and post it to the Telegram group.

## Config
Read `/Users/renostars/reno-star-business-intelligent/config/env.json` for Telegram credentials.

## Steps

1. **Gather today's activity** from ALL sources:

   a. **Git commits** (website repo):
   ```bash
   cd /Users/renostars/.openclaw/workspace/reno-stars-nextjs-prod
   git log --since="today 00:00" --oneline --no-merges 2>/dev/null
   ```

   b. **Cron job logs** (all jobs):
   ```bash
   for f in /Users/renostars/reno-star-business-intelligent/data/cron-logs/*.jsonl; do
     echo "=== $(basename $f) ==="
     jq -r 'select(.ts >= "'$(date -u +%Y-%m-%d)'") | "\(.ts) \(.status): \(.summary // .error // "no summary")"' "$f" 2>/dev/null | tail -5
   done
   ```

   c. **Social media posts** (check log):
   ```bash
   jq -r 'select(.ts >= "'$(date -u +%Y-%m-%d)'")' /Users/renostars/reno-star-business-intelligent/data/cron-logs/social-media-posts.jsonl 2>/dev/null | tail -5
   ```

   d. **Dreamina video history** (new videos):
   ```bash
   jq -r 'select(.used_at >= "'$(date -u +%Y-%m-%d)'")' /Users/renostars/reno-star-business-intelligent/data/dreamina-video-history.jsonl 2>/dev/null
   ```

   e. **Vercel deployments**:
   ```bash
   vercel ls 2>/dev/null | head -5
   ```

   f. **Invoice activity**:
   ```bash
   ls -lt /Users/renostars/.openclaw/workspace/reno-star-invoice-automation/invoices/*.md 2>/dev/null | head -3
   ```

2. **Format the summary** as a Telegram message:

```
📋 Daily Summary — <date>

🔧 Code & SEO
• <commit summary 1>
• <commit summary 2>
• ...

📱 Social Media
• <platforms posted to, content type>

🎬 Video
• <dreamina videos generated, if any>

📊 Cron Jobs
• SEO Builder: <status>
• Social Media Poster: <status>
• Social Media Monitor: <status>
• Social Media Engage: <status>
• SEO Weekly Report: <status> (Monday only)

🧾 Invoices
• <new estimates created, if any>

🚀 Deployments
• <count> deployments today

📝 Notes
• <any notable events, errors, or items needing attention>
```

3. **Send to Telegram**:
```bash
BOT_TOKEN=$(jq -r '.telegram.bot_token' /Users/renostars/reno-star-business-intelligent/config/env.json)
CHAT_ID=$(jq -r '.telegram.group_chat_id' /Users/renostars/reno-star-business-intelligent/config/env.json)
curl -s -X POST "https://api.telegram.org/bot${BOT_TOKEN}/sendMessage" \
  -H "Content-Type: application/json" \
  -d "{\"chat_id\": \"${CHAT_ID}\", \"text\": \"${MESSAGE}\", \"parse_mode\": \"HTML\"}"
```

## Pending Verifications Check

Read `/Users/renostars/reno-star-business-intelligent/data/pending-verifications.json`. For each item with `status: "pending"`:
1. Navigate to the `check_url` (or `alt_search`) and verify if it's now live/working
2. If live: update `status` to `"verified"`, set `last_checked` to today
3. If still pending: update `last_checked` to today, keep `status: "pending"`
4. If failed/broken: update `status` to `"failed"` with a note

Include in the daily summary:
```
⏳ Pending Verifications
• [item description] — [status: still pending / NOW LIVE ✅ / FAILED ❌]
```

For items that become live, take a screenshot as proof and note the verified URL.

## Rules
- Keep it concise — max 30 lines
- If nothing happened today (no commits, no cron runs, no posts), send: "📋 Daily Summary — <date>\n\n🟢 Quiet day. No activity."
- Group related items (don't list every commit individually if there are 10+ — summarize)
- Highlight errors or failures prominently with ⚠️
- Include counts, not just statuses (e.g. "3 pages improved" not just "SEO builder ran")

## Log
Append one JSON line to /Users/renostars/reno-star-business-intelligent/data/cron-logs/daily-summary.jsonl:
{"ts": "<ISO>", "job": "daily-summary", "status": "success"|"error", "summary": "<one-line summary>", "error": null}
