---
name: Reno Stars Website — Project Context
description: Production website project details, tech stack, SEO status, tracking setup
type: project
---

## Tech Stack
- Next.js 16, bilingual (en/zh), deployed on Vercel
- Neon PostgreSQL (free tier — watch data transfer, use ISR revalidation)
- All public pages have `revalidate = 3600` to avoid Neon transfer limits

## SEO Status (as of 2026-04-07)
- Sitemap: 430+ URLs, resubmitted 2026-03-26 (was broken/404 since April 2025)
- GSC: 453 indexed pages, 486 discovered (as of 2026-04-02)
- Traffic: Still very new — 79 impressions over 28 days, 0 clicks initially
- Priority keywords: "reno stars" (pos 2), "bathroom renovation richmond" (pos 10.7)
- SEO builder run 2 (2026-04-07): built basement-renovation-delta-bc (pos 19, 16 imp), commit b855524
- W3C errors: 43 → 27 (fixed 16 in run 2: role=listitem on Link, aria-controls Navbar, aria-required ContactForm)
- PageSpeed: mobile 64/100 (LCP 5.6s — needs fix), desktop 97/100
- SSL: A+ (cert expires 2026-06-18)
- Next targets: fix mobile LCP, build "average bathroom renovation cost" (pos 18.2), "basement renovation Richmond" (pos 14.7)

## Tracking Status
| Platform | Status |
|---|---|
| GA4 (G-3EZTQFQ7XH) | Live |
| Google Ads conversion (form fill) | Live |
| Microsoft Clarity (w5mxyzdnlh) | Live |
| Meta Pixel | Component committed (0b7cc0a), needs NEXT_PUBLIC_META_PIXEL_ID env var in Vercel |
| Google Ads call conversion | Component committed (0b7cc0a), needs NEXT_PUBLIC_AW_CALL_CONVERSION_LABEL env var in Vercel |

## Recent SEO Work (2026-04-07 to 2026-04-10)
- Meta titles/descriptions optimized for homepage and area pages, nearby areas cross-links added (5a9c0d5, 2026-04-10)
- Pinterest domain verification meta tag added (f4373fe)
- Service and Article structured data schemas enhanced (91d3691)
- Security: removed hardcoded admin password and DB credentials from scripts (86355ab, 2bc2af9)
- Bathroom cost page, Maple Ridge page optimized for top GSC keywords
- Hand-tuned metadata for high-priority city+service combos

## Cron Jobs (migrated to launchd — active)
12 cron jobs active as of 2026-04-09. Prompts and config live in `~/reno-star-business-intelligent/`.

| Job | launchd Label | Schedule |
|---|---|---|
| SEO Builder | com.renostars.seo-builder | Daily 6:17 AM |
| SEO Weekly Report | com.renostars.seo-weekly-report | Monday 8:03 AM |
| Social Media Poster | com.renostars.social-media-poster | Every 6h |
| Social Media Monitor | com.renostars.social-media-monitor | Every 6h |
| Social Media Engage | com.renostars.social-media-engage | Every 6h |
| Reddit Reminder | com.renostars.reddit-reminder | (see prompt) |
| Memory Compactor | com.renostars.memory-compactor | Every 6h |
| Health Check | com.renostars.health-check | Every 1h |
| Heartbeat | com.renostars.heartbeat | Every 30m (Sonnet) |
| Daily Summary | com.renostars.daily-summary | Daily |
| Email Classification Review | com.renostars.email-classification-review | (see prompt) |
| Facebook Poster | com.renostars.facebook-poster | (legacy, see prompt) |

**How to apply:** Edit prompts in `~/reno-star-business-intelligent/prompts/`, then run `pnpm run setup` to reinstall.
