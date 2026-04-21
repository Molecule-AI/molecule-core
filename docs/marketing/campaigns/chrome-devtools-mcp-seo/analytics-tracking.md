# Analytics Tracking Blueprint
## Chrome DevTools MCP SEO Campaign — Blog Post
**Post URL:** /blog/browser-automation-ai-agents-mcp
**Date:** 2026-04-20
**Author:** Content Marketer (executed Actions 3–5)
**Status:** Blueprint — needs to be applied by Marketing Lead or whoever has GA4/PostHog access

---

## GA4 Events to Configure

### Page Views
| Event | Trigger | Parameter |
|---|---|---|
| `page_view` | Automatic | `page_location`, `page_referrer` |
| `blog_view` | Blog post loaded | `post_slug`, `post_title`, `traffic_source` |

### Engagement Events
| Event | Trigger | Parameter |
|---|---|---|
| `scroll` | 75% scroll depth | `post_slug`, `percent_scrolled` |
| `time_on_page` | 30s, 60s, 120s | `post_slug`, `time_bucket` |
| `copy_code` | Code block copied | `post_slug`, `code_type` (CDP example, config, etc.) |

### CTA Clicks (apply to specific links)
| Event | Trigger | Element | GA4 Action |
|---|---|---|---|
| `cta_click` | "Get started on GitHub" link | `text: "Get started on GitHub →"` | `blog_cta_click` |
| `cta_click` | "Quickstart" link | `href: /docs/quickstart` | `blog_cta_click` |
| `cta_click` | "MCP Server Setup Guide" link | `href: /docs/guides/mcp-server-setup` | `blog_cta_click` |
| `cta_click` | GitHub star / repo link | `href: github.com/Molecule-AI/molecule-core` | `github_cta_click` |

**GA4 conversion setup for CTAs:**
- Create a **Blog CTA Click** custom event-based conversion
- Trigger: `event_name = "cta_click"`
- Filter: `post_slug = "browser-automation-ai-agents-mcp"`

---

## PostHog Events to Configure

PostHog has richer user-level tracking. If PostHog is installed on the docs site:

| Event | Trigger | Properties |
|---|---|---|
| `pageview` | Blog loaded | `slug`, `title`, `referrer`, `utm_source`, `utm_medium`, `utm_campaign` |
| `blog_scrolled_75` | 75% scroll | `slug`, `title` |
| `blog_code_copied` | Clipboard write | `slug`, `code_language`, `code_block_type` |
| `blog_cta_clicked` | CTA link clicked | `slug`, `cta_label`, `cta_url`, `destination` |

### PostHog Funnels to Build

**Funnel 1 — Trial conversion**
```
Blog page view → MCP Server Setup Guide click → Quickstart click → GitHub CTA click
```

**Funnel 2 — Engagement depth**
```
Blog page view → 75% scroll → Code copy event → CTA click
```

**Funnel 3 — Resource consumption**
```
Blog page view → Internal link click (deploy-anywhere or fly-machines) → GitHub CTA
```

### PostHog Feature Flags (if relevant)
- If A/B testing CTA copy or placement, use `feature_flag_called("blog_cta_variant")`
- Track per-variant click-through rate

---

## UTM Parameters for Campaign Tracking

Apply these to all outbound links in the blog post and social posts driving traffic to it:

| Source | Medium | Campaign | Content |
|---|---|---|---|
| `linkedin` | `social` | `chrome-devtools-mcp-seo` | `post-1`, `post-2`, `post-3` |
| `twitter` | `social` | `chrome-devtools-mcp-seo` | `thread-p1`, `thread-p2` |
| `direct` | `organic-search` | `chrome-devtools-mcp-seo` | (blank) |
| `newsletter` | `email` | `chrome-devtools-mcp-seo` | (blank) |

---

## SEO Ranking Signals to Monitor

| Signal | Tool | Check frequency |
|---|---|---|
| Keyword ranking: "browser automation AI agents" | Google Search Console | Weekly |
| Keyword ranking: "MCP browser" | GSC | Weekly |
| Impressions + CTR for blog post URL | GSC | Weekly |
| Core Web Vitals (LCP, CLS, INP) for post page | PageSpeed Insights / GSC | At publish + 30 days |
| Backlinks acquired | Ahrefs / Moz | Monthly |

---

## Traffic Baseline

Capture baseline metrics **at time of publish** so 30/60/90-day deltas are meaningful:
- GSC: impressions, clicks, CTR for target keywords
- GA4: blog sessions, scroll depth distribution, CTA click rate
- GitHub: referrer traffic to molecule-core repo

---

## Action Owners

| Task | Owner |
|---|---|
| Apply GA4 events | Marketing Lead or DevRel |
| Apply PostHog events | DevRel |
| Build PostHog funnels | Marketing Lead |
| Monitor GSC rankings weekly | SEO Analyst (your reporting cycle) |
| Backlink outreach | SEO Analyst (Actions 6, pending post review) |

---

*Last updated: 2026-04-20 by Content Marketer*
