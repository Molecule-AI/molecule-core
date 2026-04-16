/**
 * X (Twitter) post publisher with video.
 * Flow: x.com/home → "Post" composer (Drafts modal) → upload video → type caption → "Post" button.
 */
const puppeteer = require('/opt/homebrew/lib/node_modules/puppeteer-core');
const path = require('path');
const VIDEO = process.argv[2] || '/Users/renostars/dreamina-richmond-whole-house-20260415.mp4';
const CAPTION = process.argv[3] || 'Whole house refresh in Richmond — re-tiled first floor, full repaint, new fixtures. Same townhouse, modern living. Before → after 🏡';
const wait = ms => new Promise(r => setTimeout(r, ms));
const log = m => console.log(`[${new Date().toISOString().substring(11,19)}] ${m}`);

(async () => {
  const browser = await puppeteer.connect({browserURL:'http://127.0.0.1:9222', defaultViewport:null});
  const pages = await browser.pages();
  let page = pages.find(p => p.url().includes('x.com') || p.url().includes('twitter.com'));
  if (!page) {
    page = pages[0];
    await page.goto('https://x.com/home', {waitUntil:'domcontentloaded', timeout:25000});
    await wait(5000);
  } else {
    await page.goto('https://x.com/home', {waitUntil:'domcontentloaded', timeout:25000});
    await wait(4000);
  }
  await page.setViewport({width: 1900, height: 950, deviceScaleFactor: 1});
  await page.bringToFront();
  await wait(800);
  page.on('dialog', async d => { log(`dialog: ${d.message().substring(0,80)}`); await d.dismiss(); });

  const vp = await page.evaluate(() => ({w: innerWidth, h: innerHeight}));
  log(`viewport ${vp.w}x${vp.h}`);

  // STEP 1: Find the inline composer textbox on /home
  const composerInfo = await page.evaluate(() => {
    const els = [...document.querySelectorAll('div[data-testid="tweetTextarea_0"], div[role="textbox"][contenteditable="true"]')]
      .map(el => ({el, r: el.getBoundingClientRect()}))
      .filter(o => o.r.width > 100 && o.r.x >= 0 && o.r.y >= 0);
    if (!els.length) return null;
    const t = els[0];
    return {x: Math.round(t.r.x + t.r.width/2), y: Math.round(t.r.y + t.r.height/2)};
  });
  if (!composerInfo) {
    log('no inline composer — clicking post-button to open modal');
    await page.evaluate(() => {
      const b = document.querySelector('a[data-testid="SideNav_NewTweet_Button"]');
      b?.click();
    });
    await wait(2500);
  } else {
    await page.mouse.click(composerInfo.x, composerInfo.y);
    await wait(500);
  }

  // STEP 2: upload video via the file input near composer
  const fileInputs = await page.$$('input[type="file"]');
  let videoIn = null;
  for (const fi of fileInputs) {
    const acc = await fi.evaluate(el => el.accept || '');
    if (acc.includes('video') || acc.includes('image') || acc === '') videoIn = fi;
  }
  if (!videoIn && fileInputs.length) videoIn = fileInputs[0];
  if (!videoIn) { log('no file input'); await browser.disconnect(); process.exit(3); }
  await videoIn.uploadFile(VIDEO);
  log('video uploaded');
  await wait(15000); // wait for video processing/preview

  // STEP 3: Click composer + type caption
  const target = await page.evaluate(() => {
    const el = document.querySelector('div[data-testid="tweetTextarea_0"]');
    if (!el) return null;
    const r = el.getBoundingClientRect();
    return {x: Math.round(r.x + r.width/2), y: Math.round(r.y + 20)};
  });
  if (!target) { log('no composer'); await browser.disconnect(); process.exit(3); }
  await page.mouse.click(target.x, target.y);
  await wait(500);
  await page.keyboard.type(CAPTION, {delay: 8});
  await wait(2000);

  const len = await page.evaluate(() => document.querySelector('div[data-testid="tweetTextarea_0"]')?.textContent?.length || 0);
  log(`caption length: ${len}`);
  if (len < 30) { log('caption typing missed'); await page.screenshot({path:'/tmp/x-fail.png'}); await browser.disconnect(); process.exit(4); }

  // STEP 4: Click "Post" — testid="tweetButtonInline" or "tweetButton"
  const posted = await page.evaluate(() => {
    const btn = document.querySelector('button[data-testid="tweetButton"], button[data-testid="tweetButtonInline"]');
    if (!btn) return null;
    if (btn.disabled || btn.getAttribute('aria-disabled') === 'true') return 'disabled';
    btn.click();
    return 'clicked';
  });
  log(`Post button: ${posted}`);
  if (posted !== 'clicked') { log('post failed'); await page.screenshot({path:'/tmp/x-fail.png'}); await browser.disconnect(); process.exit(5); }

  // Verify by composer disappearing or success message
  for (let i=0; i<30; i++) {
    await wait(2000);
    const state = await page.evaluate(() => {
      const txt = document.body.innerText;
      const composer = !!document.querySelector('div[data-testid="tweetTextarea_0"]');
      const url = location.pathname;
      return {composer, posted: /Your post was sent|Your tweet was sent/i.test(txt), url};
    });
    log(`t+${i*2}s ${JSON.stringify(state)}`);
    if (state.posted) { log('POSTED'); break; }
    if (!state.composer && i > 3) { log('composer gone — likely posted'); break; }
  }
  await page.screenshot({path:'/tmp/x-final.png'});
  await browser.disconnect();
})().catch(e => { console.error(e); process.exit(1); });
