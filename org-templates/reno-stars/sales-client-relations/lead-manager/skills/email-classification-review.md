# Email Classification Review — Daily 9 AM

Review the last 24 hours of email classifications from the email AI service and catch any misclassified leads.

## Config
Read `/Users/renostars/reno-star-business-intelligent/config/env.json` for credentials.

## Steps

### 1. Get recent classifications from Railway DB
```bash
cd /Users/renostars/.openclaw/workspace/reno-star-email-ai-handle-service
railway run -- node -e "
const { drizzle } = require('drizzle-orm/postgres-js');
const postgres = require('postgres');
const sql = postgres(process.env.DATABASE_URL);
const db = drizzle(sql);
sql\`
  SELECT e.id, e.subject, e.from_address, e.body_text, e.received_at,
         c.category, c.sub_type, c.confidence, c.reasoning, c.reply_sent
  FROM emails e
  LEFT JOIN classifications c ON c.email_id = e.id
  WHERE e.received_at > NOW() - INTERVAL '24 hours'
  ORDER BY e.received_at DESC
\`.then(rows => { console.log(JSON.stringify(rows)); sql.end(); });
"
```

If `railway run` doesn't work, use the admin API:
```bash
curl -s 'https://reno-star-email-ai-handle-service-production.up.railway.app/admin/status' \
  -H 'Authorization: Bearer <token from env.json>'
```

### 2. Review each classification

For each email in the last 24h, check:

**Misclassified as info-only or ignore (should be needs-reply):**
- Contact form submissions (subject contains "Contact Form") → ALWAYS needs-reply
- Emails asking about renovation services, pricing, scheduling → needs-reply
- Emails with phone numbers + names + project descriptions → needs-reply
- Bilingual emails with renovation keywords (装修, 翻新, kitchen, bathroom, basement) → needs-reply

**Misclassified as needs-reply (should be spam/info-only):**
- SEO/marketing pitches → spam
- Software sales → spam
- Newsletter/notification emails → info-only
- Auto-replies/bounce-backs → ignore

### 3. Fix misclassifications

**Check backfill history first** — read `/Users/renostars/reno-star-business-intelligent/data/email-backfill-history.json`. Skip any email whose gmailMessageId is already in the `backfilled` array. Do NOT re-report or re-backfill emails that were already handled.

For each NEW misclassified email (not in backfill history):
1. Note the email ID, actual category it should be, and why
2. If a lead was MISSED (classified as info-only/ignore but should be needs-reply):
   - Flag it immediately via Telegram: "⚠️ MISSED LEAD: [name] [phone] [email] — [message]. Was classified as [category], should be needs-reply."
   - Use the backfill-lead admin endpoint (runs the FULL pipeline — AI reply, forward, Sheets, follow-ups):
     ```bash
     curl -X POST 'https://reno-star-email-ai-handle-service-production.up.railway.app/admin/backfill-lead' \
       -H 'Authorization: Bearer <token>' \
       -H 'Content-Type: application/json' \
       -d '{"gmailMessageId": "<gmail_message_id_hex>"}'
     ```
   - After successful backfill, append the gmailMessageId to the `backfilled` array in `email-backfill-history.json`
3. If spam was classified as needs-reply — less urgent, just note it

### 4. Report to Telegram

Send a summary to the Telegram group:
```
📧 Email Classification Review — <date>

Reviewed: <N> emails in last 24h
Classifications: <N> needs-reply | <N> info-only | <N> spam | <N> ignore

✅ All correct
OR
⚠️ Found <N> misclassifications:
• [email subject] — was [category], should be [correct category]
  Action: [what was done — backfilled to sheets / flagged / no action needed]

Missed leads recovered: <N>
```

### 5. Pattern detection

If you notice a PATTERN in misclassifications (e.g. all bilingual emails get wrong category, all short messages get wrong category), note it and suggest a prompt improvement. Save the suggestion to:
`/Users/renostars/reno-star-business-intelligent/data/cron-logs/email-review-suggestions.jsonl`

## Log
Append one JSON line to `/Users/renostars/reno-star-business-intelligent/data/cron-logs/email-classification-review.jsonl`:
```json
{"ts": "<ISO>", "job": "email-classification-review", "status": "success"|"error", "reviewed": <N>, "misclassified": <N>, "missed_leads": <N>, "summary": "<brief>"}
```
