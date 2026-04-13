# Social Media Monitor — Reno Stars

Check for DMs, replies, and approval responses across all platforms. Notify via Telegram.

## HONESTY RULE (CRITICAL)
When responding to DMs or comments on behalf of Reno Stars, only state facts you can verify from the website or database. Never guess prices, availability, timelines, or make promises. For pricing questions, say "it depends on scope — happy to set up a walkthrough" not a specific number. For availability, say "let me check with the team" not "we're available next week".

> **READ FIRST**: `~/.claude/skills/social-media-post/SKILL.md` and memory `feedback_social_media_platforms.md` for all platform quirks. **Reddit is PAUSED until 2026-04-21** (see Reddit section below) — skip it entirely.
>
> **Telegram approval flow note**: when you receive an ambiguous short message ("reply all", "approve", "yes"), ALWAYS check `~/.openclaw/workspace/social/pending-replies.json` and the most recent log in `~/reno-star-business-intelligent/data/cron-logs/` BEFORE asking the user "what?". Telegram Bot API has no message history; the cron's outbound message lives only on disk. (Confirmed user frustration with this on 2026-04-07.)

## Config
Read `/Users/renostars/reno-star-business-intelligent/config/env.json` for credentials.

```bash
BOT_TOKEN=$(jq -r '.telegram.bot_token' /Users/renostars/reno-star-business-intelligent/config/env.json)
CHAT_ID="-5219630660"
DB=$(jq -r '.services.neon_db' /Users/renostars/reno-star-business-intelligent/config/env.json)
```

## Pending Posts File
`/Users/renostars/.openclaw/workspace/social/pending-posts.json`

---

## PHASE 1: Process Telegram Approvals

Poll Telegram for new messages since the last processed update:

```bash
LAST_ID=$(jq -r '.last_telegram_update_id' /Users/renostars/.openclaw/workspace/social/pending-posts.json)
curl -s "https://api.telegram.org/bot${BOT_TOKEN}/getUpdates?offset=$((LAST_ID + 1))&limit=100&timeout=0"
```

Parse the response. For each message in the chat (`-5219630660`):
- Check if the text matches: `APPROVE <post_id>` or `APPROVE <post_id> <platform1,platform2>`
- If match found:
  1. Load pending-posts.json
  2. Find the entry with matching `id`
  3. If `status == "pending_approval"`:
     - If platforms specified, set `approved_platforms` to that list; otherwise default to all platforms in the entry
     - Set `status: "approved"`
     - Save pending-posts.json
     - Send Telegram confirmation: `✅ Post [post_id] approved for [platforms]. Will publish on next run.`
  4. Update `last_telegram_update_id` to the latest processed update_id in pending-posts.json

Also handle `REJECT <post_id>`:
  - Set status to "rejected"
  - Send confirmation: `❌ Post [post_id] rejected and removed from queue.`

---

## PHASE 2: Check Platform DMs and Notifications

Connect to Chrome CDP at `http://127.0.0.1:9222` using puppeteer-core at `/opt/homebrew/lib/node_modules/puppeteer-core`.
Launch Chrome if needed: `open -na "Google Chrome" --args --user-data-dir="/Users/renostars/.openclaw/chrome-profile" --remote-debugging-port=9222`

Keep track of what was already notified using `/Users/renostars/.openclaw/workspace/social/monitor-state.json`:
```json
{
  "last_checked": "<ISO>",
  "notified_message_ids": {
    "facebook": [],
    "instagram": [],
    "linkedin": [],
    "x": [],
    "xiaohongshu": [],
    "tiktok": [],
    "youtube": [],
    "reddit": []
  }
}
```
Only notify about messages NOT already in `notified_message_ids`.

### Facebook Messages
Navigate to `https://www.facebook.com/messages/` or `https://business.facebook.com/latest/inbox/all/?asset_id=100374582261988`
Check for unread message threads. For each unread thread NOT in notified list:
- Get sender name and first ~100 chars of message
- Add to notifications

Also check `https://www.facebook.com/profile.php?id=100068876523966` for new comments on recent posts.

### Instagram DMs
Navigate to `https://www.instagram.com/direct/inbox/`
Check for unread threads. For each unread thread:
- Get sender and message preview

Also check `https://www.instagram.com/renostarsvancouver/` for new comments on recent posts.

### LinkedIn Messages
Navigate to `https://www.linkedin.com/messaging/`
Check for unread messages. For each new thread:
- Get sender name and preview

Also check notifications at `https://www.linkedin.com/notifications/`

### X (Twitter)
Navigate to `https://x.com/messages` (logged in as @Renostars_ca)
Check for unread DMs.

Also check `https://x.com/notifications` for replies and mentions.

### Xiaohongshu
Navigate to `https://www.xiaohongshu.com/` and check message/notification icon.

### TikTok
Navigate to `https://www.tiktok.com/` and check the inbox/notification icon (bell icon, top nav).
Look for: new comments on posts, new followers, DMs.

### YouTube
Navigate to `https://studio.youtube.com/` and check notifications.
Also check `https://www.youtube.com/` bell icon for comments on Community Posts or channel activity.

### Reddit — PAUSED until 2026-04-21
Account was deleted on 2026-04-07 after fresh-account shadow ban. Skip this section entirely.

---

## PHASE 2.5: Lead Detection

For every new DM, comment, or mention found in Phase 2, classify it:

**🔴 HOT LEAD** — respond ASAP (flag in Telegram with 🔴):
- DMs asking about services, pricing, availability, or scheduling
- Comments saying "do you serve [city]?" / "how much would this cost?" / "can you do my [room]?"
- Messages with project details (room type, address, timeline, budget)
- Anyone who says "I need a contractor" / "looking for recommendations" / "can you help?"

**🟡 WARM** — engage within 24h:
- Comments with genuine questions about our work ("how long did this take?", "what tile is that?")
- DMs saying "hi" or "interested" without specifics
- Tagged mentions or shares of our content

**🟢 GENERAL** — engage when convenient:
- Generic compliments ("nice work!", "looks great")
- Bot/spam DMs (ignore)

For HOT LEADs: respond within 1 hour if possible. The reply should:
1. Thank them warmly
2. Ask one qualifying question ("What room are you looking to renovate?" or "What area are you in?")
3. Let them know someone will follow up ("I'll have our project manager reach out")
4. NEVER send a price estimate in a social media comment — move to DM or phone

## PHASE 3: Send Telegram Summary

If any new DMs or replies were found, send a consolidated Telegram message:

```
📬 SOCIAL MEDIA NOTIFICATIONS — [timestamp]

🔴 HOT LEADS (respond ASAP):
• [Platform] [Sender]: "[message preview]" — [why it's a lead]

[For each platform with activity:]

📘 FACEBOOK — [N] new message(s):
• [Sender]: "[message preview]"

📸 INSTAGRAM — [N] new DM(s):
• [Sender]: "[message preview]"

💼 LINKEDIN — [N] new message(s):
• [Sender]: "[message preview]"

🐦 X — [N] new mention(s)/DM(s):
• @[handle]: "[preview]"

🎵 TIKTOK — [N] new comment(s)/DM(s):
• [user]: "[preview]"

▶️ YOUTUBE — [N] new comment(s):
• [user]: "[preview]"

⚠️ Action needed: [N] hot leads need response. Reply directly on each platform.
```

If nothing new: no Telegram message needed (silent run).

Update `monitor-state.json` with the new `last_checked` timestamp and add all notified message IDs to the respective arrays.

---

## PHASE 4: Reminder for Stale Pending Posts

Check `pending-posts.json` for any items with `status: "pending_approval"` that are older than 6 hours. If found, resend the draft summary to Telegram:

```
⏰ REMINDER: Post draft waiting for your approval (6h+)

[post_id] — [content_type]: [title]

Reply APPROVE [post_id] to publish or REJECT [post_id] to discard.
```

---

## Log

Append to `/Users/renostars/reno-star-business-intelligent/data/cron-logs/social-media-monitor.jsonl`:
```json
{"timestamp":"<ISO>","job":"social-media-monitor","approvalsProcessed":<n>,"newMessages":<n>,"platforms":["facebook","instagram"],"error":null}
```
