---
id: browser-automation
name: browser-automation
description: Connect to Chrome via CDP proxy to automate web interactions — posting, scraping, form filling. Uses puppeteer-core (no bundled Chromium).
tags: [browser, puppeteer, cdp]
---

# Browser Automation via Chrome CDP

Connect to the host Chrome browser via the CDP proxy to automate web interactions.

## Connection

```javascript
const puppeteer = require('puppeteer-core');
const http = require('http');

// Get WebSocket URL from CDP proxy and rewrite for Docker networking
const data = await new Promise((res, rej) => {
  http.get('http://host.docker.internal:9223/json/version', r => {
    let d = ''; r.on('data', c => d += c); r.on('end', () => res(JSON.parse(d)));
  }).on('error', rej);
});
const wsUrl = data.webSocketDebuggerUrl.replace('localhost:9222', 'host.docker.internal:9223');
const browser = await puppeteer.connect({browserWSEndpoint: wsUrl, defaultViewport: null});
```

**Important:** Always use `browserWSEndpoint` with URL rewrite, NOT `browserURL`. The CDP proxy runs on port 9223 and rewrites the Host header for Chrome compatibility.

## Key Patterns

- **Tab listing:** `http://host.docker.internal:9223/json`
- **Navigate:** `await page.goto(url, {waitUntil: 'networkidle2'})`
- **Disconnect (don't close):** `browser.disconnect()` — never `browser.close()` (that kills the shared Chrome)

## Available Accounts

The Chrome profile has active sessions for:
- YouTube, Instagram, Facebook, X/Twitter, LinkedIn, TikTok
- Gmail, InvoiceSimple, Google Search Console
- Manta, TrustedPros, Foursquare, Pinterest, Medium
