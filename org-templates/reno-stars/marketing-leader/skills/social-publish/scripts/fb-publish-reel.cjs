#!/usr/bin/env node
/**
 * FB Reel publisher — battle-tested 2026-04-15 after ~3h debugging.
 *
 * USAGE:
 *   node fb-publish-reel.cjs <video-path> "<caption>"
 *
 * RETURNS:
 *   exit 0 — composer closed (post committed). Caller should still feed-verify.
 *   exit 1 — fatal puppeteer error
 *   exit 2 — viewport too small (Chrome window <1200px wide)
 *   exit 3 — no on-screen Lexical caption box found
 *   exit 4 — caption typing produced <50 chars (focus missed)
 *   exit 5 — Post button not visible
 *
 * LESSONS BAKED IN:
 *   1. CONNECT, never LAUNCH — `puppeteer.connect({browserURL, defaultViewport: null})`
 *      uses real Chrome window dims. `puppeteer.launch()` spawns fresh Chromium with
 *      no cookies — that's the "all sessions expired" false positive.
 *   2. FB has 4-6 Lexical mirror instances, most off-screen at negative x or y > 1000.
 *      Pick by: visible viewport rect + width > 200 + non-comment-box.
 *   3. Lexical doesn't accept execCommand/clipboard. Use mouse.click(target) to focus,
 *      then page.keyboard.type() — REAL keystrokes the Lexical input handlers fire on.
 *   4. Reel composer flow: Upload → Next (advance to Edit) → Next (advance to Settings)
 *      → Post button is at (~291, 802) in a 1920-wide window (left side).
 *   5. After Post, Meta shows post-publish UPSELLS ("Add WhatsApp button",
 *      "Speak With People Directly", "Boost"). Always click "Not now"/"Skip"/etc.
 *      Failure to dismiss → Chrome beforeunload triggers on next navigation
 *      → "Leave site? Changes may not be saved" dialog blocks the script.
 *   6. Register a `page.on('dialog', d => d.dismiss())` BEFORE clicking Post so
 *      any beforeunload that does fire is auto-cancelled.
 *   7. Verify success by composer-disappearance (selectors on `[aria-label="Edit reel"]`
 *      / `[aria-label="Reel settings"]`), NOT by upsell modal appearance.
 *      Then feed-verify separately (post text + recent timestamp).
 */
const puppeteer = require('puppeteer-core');
const path = require('path');

const VIDEO = process.argv[2];
const CAPTION = process.argv[3];
const PROFILE_URL = process.env.FB_PROFILE_URL || 'https://www.facebook.com/profile.php?id=100068876523966';
const CDP_URL = process.env.CDP_URL || 'http://127.0.0.1:9222';

if (!VIDEO || !CAPTION) {
  console.error('Usage: fb-publish-reel.cjs <video-path> "<caption>"');
  process.exit(1);
}

const log = (m) => console.log(`[${new Date().toISOString().substring(11,19)}] ${m}`);
const wait = (ms) => new Promise(r => setTimeout(r, ms));

(async () => {
  const browser = await puppeteer.connect({ browserURL: CDP_URL, defaultViewport: null });
  const pages = await browser.pages();
  let page = pages.find(p => p.url().includes('facebook.com/profile.php'));
  if (!page) {
    page = pages[0];
    await page.goto(PROFILE_URL, { waitUntil: 'domcontentloaded', timeout: 25000 });
    await wait(3500);
  }
  await page.bringToFront();
  await wait(800);

  // LESSON 6: register beforeunload dismisser BEFORE any state changes
  page.on('dialog', async d => {
    log(`native dialog auto-dismissed: ${d.type()} "${d.message().substring(0, 60)}"`);
    await d.dismiss();
  });

  // LESSON 1: verify viewport
  const vp = await page.evaluate(() => ({ w: window.innerWidth, h: window.innerHeight }));
  log(`viewport ${vp.w}x${vp.h}`);
  if (vp.w < 1200) { log('ABORT: viewport <1200px wide'); await browser.disconnect(); process.exit(2); }

  // STEP 1: open Reel composer
  let dialog = await page.evaluate(() => !!document.querySelector('[role="dialog"]'));
  if (!dialog) {
    log('clicking Reel button');
    await page.evaluate(() => {
      const sp = [...document.querySelectorAll('span')].find(s => s.textContent.trim() === 'Reel');
      sp?.closest('[role="button"]')?.click();
    });
    await wait(3000);
  }

  // STEP 2: upload video into the video-accepting hidden file input
  const inputs = await page.$$('input[type="file"]');
  let videoIn = null;
  for (const i of inputs) {
    const accept = await i.evaluate(el => el.accept || '');
    if (accept.includes('video/')) { videoIn = i; break; }
  }
  if (videoIn) {
    log(`uploading ${path.basename(VIDEO)}`);
    await videoIn.uploadFile(VIDEO);
    await wait(8000);
  } else {
    log('no video input — assuming already uploaded');
  }

  // STEP 3: Next to Edit reel + Next to Reel settings (2 clicks)
  for (let n = 1; n <= 2; n++) {
    const target = await page.evaluate(() => {
      const btns = [...document.querySelectorAll('[role="button"]')].filter(b => {
        const r = b.getBoundingClientRect();
        return b.textContent.trim() === 'Next' && r.x > 0 && r.y > 0 && r.height > 20 && r.x < window.innerWidth && r.y < window.innerHeight;
      });
      if (!btns.length) return null;
      const r = btns[0].getBoundingClientRect();
      return { x: Math.round(r.x + r.width / 2), y: Math.round(r.y + r.height / 2) };
    });
    if (!target) { log(`Next ${n}: not found, aborting`); await browser.disconnect(); process.exit(5); }
    await page.mouse.click(target.x, target.y);
    log(`Next ${n} clicked at (${target.x}, ${target.y})`);
    await wait(5000);
  }

  // STEP 4: LESSON 2 + 3 — fill caption via mouse.click + keyboard.type on the
  // widest on-screen Lexical textbox
  const target = await page.evaluate(() => {
    const candidates = [...document.querySelectorAll('div[role="textbox"][data-lexical-editor]')]
      .filter(b => {
        const r = b.getBoundingClientRect();
        return r.x >= 0 && r.x < window.innerWidth && r.y >= 0 && r.y < window.innerHeight && r.width > 200;
      })
      .sort((a, b) => b.getBoundingClientRect().width - a.getBoundingClientRect().width);
    if (!candidates.length) return null;
    const r = candidates[0].getBoundingClientRect();
    return { x: Math.round(r.x + r.width / 2), y: Math.round(r.y + r.height / 2) };
  });
  if (!target) { log('ABORT: no caption box'); await browser.disconnect(); process.exit(3); }
  await page.mouse.click(target.x, target.y);
  await wait(800);
  await page.keyboard.type(CAPTION, { delay: 5 });
  await wait(2000);

  const len = await page.evaluate(() => {
    const b = [...document.querySelectorAll('div[role="textbox"][data-lexical-editor]')]
      .find(b => b.getBoundingClientRect().width > 200 && b.innerText.trim().length > 0);
    return b ? b.innerText.length : 0;
  });
  log(`caption inserted: ${len} chars`);
  if (len < 50) { log('ABORT: typing missed'); await browser.disconnect(); process.exit(4); }

  // STEP 5: click Post (at ~291, 802 in 1920-wide window — bottom-left of Reel settings)
  const post = await page.evaluate(() => {
    const btns = [...document.querySelectorAll('[role="button"]')].filter(b => {
      const r = b.getBoundingClientRect();
      return ['Post', 'Publish', 'Share now'].includes(b.textContent.trim()) && r.x > 0 && r.y > 0 && r.height > 20 && r.x < window.innerWidth && r.y < window.innerHeight;
    });
    if (!btns.length) return null;
    const r = btns[0].getBoundingClientRect();
    return { x: Math.round(r.x + r.width / 2), y: Math.round(r.y + r.height / 2), text: btns[0].textContent.trim() };
  });
  if (!post) { log('ABORT: Post button not visible'); await browser.disconnect(); process.exit(5); }
  await page.mouse.click(post.x, post.y);
  log(`Post clicked: ${post.text}`);

  // STEP 6: wait for Reel composer to close (real success signal)
  let closed = false;
  for (let i = 0; i < 45; i++) {
    await wait(2000);
    closed = await page.evaluate(() =>
      !document.querySelector('[aria-label="Edit reel"]') &&
      !document.querySelector('[aria-label="Reel settings"]')
    );
    if (closed) { log(`composer closed after ${(i+1)*2}s`); break; }
  }
  if (!closed) { log('FAIL: composer never closed'); await browser.disconnect(); process.exit(5); }

  // STEP 7: LESSON 5 — dismiss any post-publish upsell ("Add WhatsApp button", etc.)
  for (let i = 0; i < 4; i++) {
    const dismissed = await page.evaluate(() => {
      const btns = [...document.querySelectorAll('[role="button"], button')].filter(b => {
        const r = b.getBoundingClientRect();
        const t = (b.textContent || '').trim();
        return ['Not now', 'Skip', 'Maybe later', 'No thanks'].includes(t) && r.x >= 0 && r.y >= 0 && r.height > 20 && r.x < window.innerWidth && r.y < window.innerHeight;
      });
      if (!btns.length) return null;
      btns[0].click();
      return btns[0].textContent.trim();
    });
    if (!dismissed) break;
    log(`dismissed upsell #${i+1}: ${dismissed}`);
    await wait(1500);
  }

  log('DONE');
  await browser.disconnect();
  process.exit(0);
})().catch(e => { console.error('FATAL:', e.message); process.exit(1); });
