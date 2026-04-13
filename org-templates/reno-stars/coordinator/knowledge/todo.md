---
name: Active TODO List
description: Current tasks and priorities — check this every session
type: project
---

## Pending

### High Priority
- [x] **Set up Telegram channel** — DMs + group @mentions working. Group: -5219630660 "RENO STARS bot group". All cron reports go to group.
- [x] **Verify cron jobs firing** — All 12 crons confirmed active as of 2026-04-09 (see project_reno_stars_website.md for full list)

### Medium Priority
- [ ] **Meta Pixel ID** — Get NEXT_PUBLIC_META_PIXEL_ID from business.facebook.com → Events Manager → Pixels, add to Vercel
- [ ] **Google Ads call conversion label** — Create "Phone call click" conversion, get NEXT_PUBLIC_AW_CALL_CONVERSION_LABEL, add to Vercel
- [ ] **Add 15 Chinese keywords to Google Ads** — User was doing via Ads Editor
- [ ] **Fix 2 disapproved CN sitelinks** — 厨房翻新, 商业装修 destination not working
- [x] **Audit 小红书** — PAUSED as of 2026-04-09 (platform warning). Do not post until user re-enables.
- [x] **Pinterest business account** — Created 2026-04-09, domain verification meta tag deployed (f4373fe), 5 initial pins published

### Low Priority
- [ ] **Clean up 8 legacy Google Ads campaigns**
- [ ] **Build content calendar for Q2 2026**
- [ ] **Increase Google Ads budgets** — $60/day → $150/day, only after conversion tracking confirmed
- [ ] **TikTok Pixel** — When TikTok ad spend starts

## Completed (2026-04-04)
- [x] Read entire OpenClaw workspace and absorb into Claude Code memory
- [x] Created ~/reno-star-business-intelligent/ automation hub
- [x] Migrated all 6 cron jobs to macOS launchd
- [x] Centralized all config in repo (CLAUDE.md, settings.json, memory files)
- [x] Symlinked ~/.claude/ → repo for portability
- [x] Added hooks: block --no-verify, block rm -rf, protect linter configs
- [x] Added MCP servers: context7, sequential-thinking, playwright, reno-stars-hub (11 tools), reno-stars-invoice (7 tools)
- [x] Added workflow rules: design-before-code, systematic debugging, 3-strike rule, code quality
- [x] Tested memory-compactor and seo-weekly-report crons end-to-end
- [x] Built MCP server with 11 tools (memory, cron, project, telegram, config)
- [x] Connected reno-star-invoice-automation MCP (7 tools)
- [x] Security audit: scrubbed git history, pre-commit secret scanner
- [x] Created GitHub repo: Reno-Stars/reno-star-business-intelligent (private)
- [x] Fixed MCP server config (moved to ~/.claude.json, absolute paths)
- [x] Installed Telegram channel plugin + configured bot token and access.json
- [x] Added heartbeat cron (every 30m, Sonnet) — TODO review, cron health, rotating checks
