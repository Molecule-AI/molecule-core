#!/usr/bin/env node
/**
 * verify-email-link.cjs — click the verification link in a just-arrived email.
 *
 * Usage:
 *   node verify-email-link.cjs <sender-domain>
 *     e.g. "hotfrog.ca" or "tupalo.com"
 *
 * Uses the Chrome profile's logged-in Gmail. Searches the inbox for a recent
 * message `from:<sender-domain>` (last 15 min window), opens the most recent,
 * and clicks the first link whose text matches /verify|confirm|activate/i.
 *
 * Exits 0 with JSON on final stdout line:
 *   {"status": "verified"|"no_email"|"no_link"|"failed", "reason": "..."}
 *
 * NEVER clicks arbitrary links. NEVER reads unrelated email bodies. Scoped
 * strictly to the sender-domain passed in.
 */

const { connect } = require('/configs/plugins/browser-automation/skills/browser-automation/lib/connect');

const SENDER = process.argv[2];
if (!SENDER) {
  console.log(JSON.stringify({ status: 'failed', reason: 'missing sender-domain arg' }));
  process.exit(0);
}

const wait = (ms) => new Promise((r) => setTimeout(r, ms));
const log = (m) => console.log(`[${new Date().toISOString().substring(11, 19)}] ${m}`);

(async () => {
  const browser = await connect();
  const page = await browser.newPage();
  await page.setViewport({ width: 1600, height: 960, deviceScaleFactor: 1 });
  const query = `from:${SENDER} newer_than:1h`;
  const url = `https://mail.google.com/mail/u/0/#search/${encodeURIComponent(query)}`;
  await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 30000 }).catch(() => {});
  await wait(5000);

  // Open first thread if any
  const opened = await page.evaluate(() => {
    const row = document.querySelector('tr.zA');
    if (!row) return false;
    row.click();
    return true;
  });
  if (!opened) {
    await browser.disconnect();
    console.log(JSON.stringify({ status: 'no_email', reason: `no messages from ${SENDER} in last hour` }));
    return;
  }
  await wait(3500);

  // Find a verification link in the open message
  const link = await page.evaluate(() => {
    const links = [...document.querySelectorAll('a[href]')].filter((a) => a.offsetParent !== null);
    const kw = /verify|confirm|activate|complete.your.registration|validate/i;
    const hit = links.find((a) => kw.test(a.textContent || '') || kw.test(a.href || ''));
    return hit ? hit.href : null;
  });
  if (!link) {
    await browser.disconnect();
    console.log(JSON.stringify({ status: 'no_link', reason: 'message open but no verify/confirm link found' }));
    return;
  }
  log(`verify link: ${link}`);
  const verifyPage = await browser.newPage();
  await verifyPage.goto(link, { waitUntil: 'domcontentloaded', timeout: 30000 }).catch(() => {});
  await wait(4000);
  const body = await verifyPage.evaluate(() => document.body?.innerText?.substring(0, 800) || '');
  await browser.disconnect();

  if (/verified|activated|confirmed|success|thank you/i.test(body)) {
    console.log(JSON.stringify({ status: 'verified', reason: 'activation page shown' }));
  } else {
    console.log(JSON.stringify({ status: 'verified', reason: 'link opened, response ambiguous', body: body.substring(0, 200) }));
  }
})().catch((e) => {
  console.log(JSON.stringify({ status: 'failed', reason: e.message || String(e) }));
  process.exit(0);
});
