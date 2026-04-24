/**
 * Lighthouse audit runner for the Chrome DevTools MCP SEO blog post.
 * Uses puppeteer (which bundles a compatible Chrome) to avoid Chrome-not-found issues.
 */
const playwright = require('playwright');
const lighthouse = require('lighthouse');
const path = require('path');
const fs = require('fs');

const URL = process.argv[2] || 'http://localhost:3001/blog/chrome-devtools-mcp';
const OUTPUT = process.argv[3] || '/tmp/lh-blog.json';

async function main() {
  console.error(`Launching browser for: ${URL}`);

  // Use playwright's bundled Chromium
  const browser = await playwright.chromium.launch({
    args: ['--no-sandbox', '--disable-dev-shm-usage', '--disable-setuid-sandbox'],
    headless: true,
  });

  const page = await browser.newPage();

  // Block third-party requests to avoid analytics/telemetry noise
  await page.setRequestInterception(true);
  page.on('request', req => {
    const url = req.url();
    if (url.includes('google-analytics') || url.includes('segment') || url.includes('hotjar')) {
      req.abort();
    } else {
      req.continue();
    }
  });

  console.error('Browser launched. Running Lighthouse...');

  const result = await lighthouse(URL, {
    port: new URL(browser.wsEndpoint()).port,
    output: 'json',
    logLevel: 'info',
    onlyCategories: ['performance', 'accessibility', 'best-practices', 'seo'],
    formFactor: 'desktop',
    screenEmulation: { mobileEnabled: false },
    throttling: { rtt: 0, throughput: 10 * 1024 * 1024, cpuSlowdownMultiplier: 1 },
  });

  const report = result.report;
  fs.writeFileSync(OUTPUT, report);
  console.error(`Report saved to ${OUTPUT}`);

  const scores = {};
  for (const [key, cat] of Object.entries(result.lhr.categories)) {
    scores[key] = Math.round(cat.score * 100);
  }

  console.log(JSON.stringify({ url: URL, scores, reportPath: OUTPUT }, null, 2));

  // Print failed/warning audits
  const lhr = result.lhr;
  const issues = [];
  for (const [id, audit] of Object.entries(lhr.audits)) {
    if (audit.score !== null && audit.score < 1) {
      const level = audit.score === 0 ? 'FAIL' : 'WARN';
      issues.push({ level, id, score: audit.score, title: audit.title, description: audit.description?.slice(0, 200) });
    }
  }

  console.error('\n=== AUDIT RESULTS ===');
  for (const [cat, score] of Object.entries(scores)) {
    const icon = score >= 90 ? '✅' : score >= 50 ? '⚠️' : '❌';
    console.error(`${icon} ${cat}: ${score}/100`);
  }

  console.error('\n=== FAILED / WARNING AUDITS ===');
  for (const issue of issues) {
    console.error(`${issue.level}: [${issue.id}] ${issue.title}`);
    if (issue.description) console.error(`   → ${issue.description}`);
  }

  await browser.close();
  process.exit(0);
}

main().catch(err => {
  console.error('Fatal:', err.message);
  process.exit(1);
});
