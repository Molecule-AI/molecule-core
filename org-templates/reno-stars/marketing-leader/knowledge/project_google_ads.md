---
name: Google Ads — Campaign Status and Context
description: Google Ads account structure, campaigns, and optimization status for Reno Stars
type: project
---

## Account
- MCC: ${GADS_MCC_ID} | CID: ${GADS_CUSTOMER_ID}
- Dev token: in config/env.json → google.ads_dev_token (Test Access, Basic pending)
- 14 campaigns total (4 active targets, 8 legacy/dead, 2 other) — all currently PAUSED

## Active Campaigns
- AI Bathroom Renovation - EN: 282 impr, 15 clicks, $87 spent
- AI Kitchen Renovation - EN: 413 impr, 17 clicks, $89 spent
- AI Full Home Renovation - EN: 592 impr, 20 clicks, 2 conv, $86 (best performer)
- Chinese ads 2026: 386 impr, 22 clicks, $91 spent
- Best keyword: "remodeling company near me" — 11.63% CTR, 20% conv rate

## Completed Work
- 72 negative keywords added across all 4 campaigns
- ~58 keywords added across 3 EN campaigns
- Sitelink URLs fixed (was pointing to old routes)
- RSA ad final URLs fixed (was causing 308 redirects)
- Multiple callout extensions added/renamed

## Still TODO
- Add 15 Chinese keywords (user was doing via Ads Editor)
- Fix 2 disapproved CN sitelinks (厨房翻新, 商业装修)
- Increase budgets $60/day → $150/day (only after conversion tracking confirmed)
- Clean up 8 legacy campaigns

## Technical Notes
- Google Ads Angular UI is extremely difficult to automate via Playwright
- CDP keyboard input into Angular forms is unreliable
- Chrome CDP triggers false "ad blocker detected" — need stealth injector bypass

**How to apply:** Detailed plans and change logs are in ~/.openclaw/workspace/docs/google-ads-*.md
