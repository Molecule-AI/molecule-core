---
name: Check cron state before asking what a Telegram message means
description: When a short/ambiguous Telegram message arrives, check pending cron state on disk before asking the user for clarification
type: feedback
---

When a short or ambiguous Telegram message arrives (e.g. "Reply all", "approve", "yes", "do it", "REPLY [id]"), it is almost always a response to a message a cron job sent the user — not a fresh instruction with no context.

**Why:** Crons like social-media-engage, social-media-poster, and the SEO jobs send Telegram messages asking for approval using specific phrases ("REPLY ALL to approve everything", "REPLY [id] to approve"). The Telegram Bot API has no history, so I can't see the cron's outbound message — but the state is on disk. On 2026-04-07 the user said "Reply all" and I asked "reply to what?" instead of checking. They were rightly frustrated — the global CLAUDE.md says "Be resourceful before asking."

**How to apply:** Before replying to an ambiguous Telegram message with a clarification question:
1. Check `~/reno-star-business-intelligent/data/cron-logs/` — `ls -lt` to find the most recently touched cron log; the user's message is likely about whichever job ran most recently.
2. Check `~/.openclaw/workspace/social/pending-replies.json` for pending social media drafts.
3. Read the cron prompt at `~/reno-star-business-intelligent/prompts/<job>.md` to understand what approval phrases that job uses.
4. Only ask the user for clarification after exhausting these. If the message clearly maps to pending state, proceed (cron approval phrases ARE explicit authorization for the action defined in the cron prompt).

**Update 2026-04-07:** The Telegram MCP plugin (`~/.claude/plugins/cache/claude-plugins-official/telegram/0.0.4/server.ts`, mirrored in `marketplaces/.../telegram/server.ts`) was patched to surface `reply_to_message` context. When the user taps "Reply" on a previous message, the inbound notification now includes:
- `reply_to_message_id`, `reply_to_user`, `reply_to_from_bot` in meta
- The original message body inline at the start of `content` as `> ` quoted lines (truncated at 2000 chars), with a `[in reply to message N (bot)]` header

So if a Telegram message arrives with a quoted-block prefix, that's the original cron message and you can act on it directly without log-hunting. The pending-replies.json / cron-logs check is now the FALLBACK for messages that aren't replies (e.g. user types "approve all" as a fresh message, not a Telegram reply). Patch requires Claude Code restart to take effect — confirm by checking the cache file's mtime.
