---
name: GSC analysis — absolute clicks only, not CTR
description: When investigating GSC performance, anchor every diagnosis on absolute click count. CTR is noise the user does not care about.
type: feedback
---

When the user reports that GSC metrics are "dropping" or asks for SEO performance analysis, the metric they care about is **absolute weekly clicks**, not CTR.

**Why:** On 2026-04-08 the user said: "CTR does not matter, I only ask you to improve because abs click count also dropping." I had framed the earlier diagnosis around CTR collapse (1.12% → 0.80%) and tried to soothe by pointing out clicks were +54% WoW in my 7-day window. The user's actual concern was that on a longer rolling window the absolute click count is trending down — and CTR percentages are irrelevant when total volume is small (the difference between 0.5% and 1.5% CTR on 200 impressions is 2 clicks, statistical noise).

**How to apply:**
1. When pulling GSC data for analysis, ALWAYS report the absolute weekly clicks for at least the last 4-6 weeks side-by-side. Don't hide it inside CTR/impression noise.
2. If the user says "dropping," check the trend over multiple windows (7d, 14d, 28d, week-over-week for the last 4 weeks). The chart they're seeing in GSC is the daily clicks line — that's what they're reacting to.
3. Do NOT lead with CTR analysis. CTR can be a follow-up explanation for *why* clicks moved, not the primary metric.
4. If the absolute click count IS rising and the user thinks it's falling, push back with the actual numbers — but lead with the click numbers, not CTR/impression context.
5. The fix proposals should target raising absolute clicks (publish new content that ranks, improve titles on existing pages, internal linking, backlinks, page speed) rather than "CTR optimization" framing.

**Caveat:** This is for the user's monitoring lens. CTR is still a valid technical metric inside diagnosis (e.g., "this page has 400 impressions and 1 click → its title is bad → fix the title"). Just don't report CTR as a top-line summary.
