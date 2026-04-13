You are generating the weekly SEO report for reno-stars.com.

## Config
Read /Users/renostars/reno-star-business-intelligent/config/env.json for paths and credentials.

## STEPS
1. Try running the existing report script:
   ```
   node /Users/renostars/.openclaw/workspace/scripts/seo-weekly-report.mjs
   ```
2. If the script works, summarize the output
3. If it errors with 'not yet authorized', note that ${GSC_SERVICE_ACCOUNT} needs to be added to Search Console
4. If the script doesn't exist or fails for other reasons, generate the report manually:

### Manual Report Generation
```bash
TOKEN=$(PATH=$PATH:/opt/homebrew/share/google-cloud-sdk/bin gcloud auth application-default print-access-token)

# This week
curl -X POST "https://searchconsole.googleapis.com/webmasters/v3/sites/https%3A%2F%2Fwww.reno-stars.com%2F/searchAnalytics/query" \
  -H "Authorization: Bearer $TOKEN" -H "x-goog-user-project: ${GCP_PROJECT_ID}" \
  -H "Content-Type: application/json" \
  -d '{"startDate":"7daysAgo","endDate":"today","dimensions":["query"],"rowLimit":25}'

# Last week (for comparison)
curl -X POST "https://searchconsole.googleapis.com/webmasters/v3/sites/https%3A%2F%2Fwww.reno-stars.com%2F/searchAnalytics/query" \
  -H "Authorization: Bearer $TOKEN" -H "x-goog-user-project: ${GCP_PROJECT_ID}" \
  -H "Content-Type: application/json" \
  -d '{"startDate":"14daysAgo","endDate":"7daysAgo","dimensions":["query"],"rowLimit":25}'
```

5. Compare week-over-week: clicks, impressions, CTR, position changes
6. Highlight notable movers (keywords gaining/losing position)

---

## BUSINESS PROFILE ANALYTICS

After GSC data, collect weekly metrics from all business listing platforms. Use Chrome CDP (port 9222) with puppeteer-core at /opt/homebrew/lib/node_modules/puppeteer-core. Launch Chrome if needed:
```bash
open -na "Google Chrome" --args --user-data-dir="/Users/renostars/.openclaw/chrome-profile" --remote-debugging-port=9222
sleep 3
```

### Google Business Profile (GBP) Insights
Use the Business Profile Performance API with gcloud token:
```bash
TOKEN=$(PATH=$PATH:/opt/homebrew/share/google-cloud-sdk/bin gcloud auth application-default print-access-token)
LOCATION_ID="1497199709887249563"

# Daily metrics for past 7 days
curl -s "https://businessprofileperformance.googleapis.com/v1/locations/${LOCATION_ID}:getDailyMetricsTimeSeries?dailyMetric=BUSINESS_IMPRESSIONS_DESKTOP_MAPS&dailyMetric=BUSINESS_IMPRESSIONS_DESKTOP_SEARCH&dailyMetric=BUSINESS_IMPRESSIONS_MOBILE_MAPS&dailyMetric=BUSINESS_IMPRESSIONS_MOBILE_SEARCH&dailyMetric=CALL_CLICKS&dailyMetric=WEBSITE_CLICKS&dailyMetric=BUSINESS_DIRECTION_REQUESTS&dailyRange.startDate.year=$(date +%Y)&dailyRange.startDate.month=$(date -v-7d +%m)&dailyRange.startDate.day=$(date -v-7d +%d)&dailyRange.endDate.year=$(date +%Y)&dailyRange.endDate.month=$(date +%m)&dailyRange.endDate.day=$(date +%d)" \
  -H "Authorization: Bearer $TOKEN" \
  -H "x-goog-user-project: ${GCP_PROJECT_ID}"
```
Report: total impressions (maps + search), website clicks, calls, direction requests. Compare to prior week if data available.

Also navigate Chrome to the GBP panel via:
`https://www.google.com/search?q=Reno+Stars+Local+Renovation+Company&authuser=0#mpd=~1497199709887249563/promote/photos/mediatool`
And read the review count + rating displayed.

### Yelp Analytics
Connect puppeteer to Chrome CDP (port 9222). Navigate to:
`https://biz.yelp.com/home/S_kdh-5GuSvSiY_P43jLsw`
Extract from the page:
- People finding on Yelp (weekly count shown in Performance Summary)
- Total review count + current star rating
- Any new reviews since last week (check Reviews page: `https://biz.yelp.com/r2r/S_kdh-5GuSvSiY_P43jLsw`)
- Photo count: `https://biz.yelp.com/biz_photos/S_kdh-5GuSvSiY_P43jLsw`

### Bing Places Analytics
Navigate Chrome to:
`https://www.bing.com/forbusiness/analytics?bizid=65003580-d585-43d0-90df-cff52c957356`
Extract: impressions, clicks, calls, direction requests for the week.
Also check `https://www.bing.com/forbusiness/singleEntity?bizid=65003580-d585-43d0-90df-cff52c957356` for review count.

### Apple Business Connect
Navigate Chrome to `https://businessconnect.apple.com/` — if logged in, extract weekly impressions and actions from the dashboard.
If not logged in, skip and note "Apple Business Connect: login required".

---

## LOCAL MARKETING REPORTS — Rank Tracker + GBP Audit

This is a third-party rank tracker (PagePros / Local Marketing Reports) that tracks 21 local keywords for Reno Stars in the Richmond V6W area, plus a GBP audit and citation builder. The dashboards are inside an authenticated session in the user's existing Chrome — drive via puppeteer-core CDP, NOT the playwright MCP (the wrapper is broken — see `feedback_playwright_timeouts.md`).

**Account ID** (in URL): `b53751d832fda91f52ede41e3e213e13bd1c13d6`

### Rank Tracker
URL: `https://www.local-marketing-reports.com/location-dashboard/b53751d832fda91f52ede41e3e213e13bd1c13d6/ranking-reports`

```js
// Connect via direct puppeteer-core (NOT playwright MCP — it hangs)
const puppeteer = require('/opt/homebrew/lib/node_modules/puppeteer-core');
const browser = await puppeteer.connect({ browserURL: 'http://127.0.0.1:9222', defaultViewport: null });
const pages = await browser.pages();
let lmr = pages.find(p => p.url().includes('local-marketing-reports.com'));
if (!lmr) lmr = await browser.newPage();
await lmr.bringToFront();
await lmr.goto('https://www.local-marketing-reports.com/location-dashboard/b53751d832fda91f52ede41e3e213e13bd1c13d6/ranking-reports', { waitUntil: 'networkidle2', timeout: 30000 });
await new Promise(r => setTimeout(r, 4000));
const body = await lmr.evaluate(() => document.body.innerText);
// Extract: average position, keyword movement, position distribution, full keyword table
```

**Extract from the page body:**
- **Average Google Position** (first numeric after "Average Google Position") + the trend delta
- **Keyword and Positional Movement** numbers (e.g. "10 Keyword Change" + "20 Positional Change")
- **Google Local Pack Coverage** percentage — if 0%, flag as 🔴 critical
- **Position distribution**: how many keywords at #1 / #2-5 / #6-10 / #11-20 / #21-50 / #51+
- **Rankings Table**: keyword name + current Local Finder rank + change since last comparison
  - Format each as: `<keyword> <rank> <change>` where rank "-" means NOT RANKING
  - Sort by rank (best first), then by change magnitude
- Highlight any keyword that moved into or out of page 1 (rank 1-10)
- Highlight any keyword that's NOT RANKING but should be (these need new content or page optimization)

### GBP Audit
URL: `https://www.local-marketing-reports.com/location-dashboard/b53751d832fda91f52ede41e3e213e13bd1c13d6/gbpa-reports`

Extract from the page body:
- **NAP Data**: Name / Address / Website / Phone / Categories — flag if Website is `http://` not `https://`
- **Photo count** (e.g. "Images271")
- **30-day insights**: Total Views, breakdown (Search Desktop / Search Mobile / Maps Desktop / Maps Mobile)
- **30-day actions**: Total Actions, Website clicks, Direction requests, Phone calls
- **Phone calls** total + day-of-week heatmap if visible
- Flag if call count is < 5% of website clicks (signals weak phone CTA)

### Citation Builder
URL: `https://www.local-marketing-reports.com/location-dashboard/b53751d832fda91f52ede41e3e213e13bd1c13d6/citation-builder`

Extract:
- Active campaign ID + date
- Citations Submission Status: Ordered / To Do / Submitted / Pending / Live / Updated / Existing / Replaced counts
- List of live citation sites (table rows)
- The "SET UP" badge on the sidebar is misleading — there IS a campaign, the badge is an upsell prompt, NOT a setup-required signal

### Local Marketing Reports Summary Format
Add to the weekly report alongside the GSC + business profile blocks:
```
📊 LOCAL RANK TRACKER — Week of <date>

Avg Local Finder Position: <X.X> (Δ <±N.N> vs prior week)
Local Pack Coverage: <X%>  ⚠️ if 0%
Keyword movement: <N> keywords moved, <N> total positional improvements

Position distribution:
  #1: <N>  |  #2-5: <N>  |  #6-10: <N>  |  #11-20: <N>  |  #21-50: <N>  |  #51+/unranked: <N>

Top movers (gained):
  • <keyword> rank <N> (+<delta>)
  ...

Top decliners (lost):
  • <keyword> rank <N> (-<delta>)
  ...

Not ranking (build/improve content):
  • <keyword>
  ...

GBP audit flags:
  • <flag 1>
  • <flag 2>

Citations: <N> live / <N> pending / <campaign date>
```

### Decision logic
- **Local Pack Coverage 0%** → top action item: review GBP categories, service area, and local citations
- **Keyword dropped from page 1** → audit the page for content thinness or recent changes
- **Keyword not ranking but page exists** → page is too thin / templated / no inbound links — flag for the seo-builder cron's IMPROVE_EXISTING mode
- **Average position trending up by >0.5 weekly** → notable positive momentum, mention in headline
- **Average position trending down by >0.5 weekly** → critical, investigate immediately

### Business Profile Summary Format
Report all platforms in one block:
```
📍 BUSINESS PROFILES — Week of <date>

GBP:    <impressions> views | <clicks> site clicks | <calls> calls | <directions> directions | ⭐ <rating> (<count> reviews)
Yelp:   <N> people found | ⭐ <rating> (<count> reviews) | <photo_count> photos
Bing:   <impressions> views | <clicks> clicks | <calls> calls
Apple:  <impressions> impressions | <actions> actions  (or: login required)

📝 New reviews this week: <list any new reviews with platform + snippet>
⚠️  Action items: <e.g. "respond to 2 Yelp reviews", "GBP photo pending approval", etc.>
```

---

## Log
Append one JSON line to /Users/renostars/reno-star-business-intelligent/data/cron-logs/seo-weekly-report.jsonl:
{"ts": "<ISO>", "job": "seo-weekly-report", "status": "success"|"error", "summary": "<brief WoW summary>", "error": null}
