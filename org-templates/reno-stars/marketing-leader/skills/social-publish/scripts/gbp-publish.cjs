const puppeteer = require('/opt/homebrew/lib/node_modules/puppeteer-core');
const VIDEO = process.argv[2] || '/Users/renostars/dreamina-richmond-whole-house-20260415.mp4';
const TEXT = `Richmond Whole House Renovation ✨

Budget-friendly transformation of a Richmond townhouse — re-tiled first floor, full repaint, and updated light fixtures throughout. A whole-house refresh without the demolition.

📞 Free consultation: 778-960-7999
🌐 reno-stars.com`;
const wait = ms => new Promise(r => setTimeout(r, ms));
const log = m => console.log(`[${new Date().toISOString().substring(11,19)}] ${m}`);

(async () => {
  const browser = await puppeteer.connect({browserURL:'http://127.0.0.1:9222', defaultViewport:null});
  const page = await browser.newPage();
  await page.setViewport({width: 1900, height: 950, deviceScaleFactor: 1});
  await page.goto('https://www.google.com/search?q=Reno+Stars+-+Local+Renovation+Company&authuser=0', {waitUntil:'load', timeout:40000}).catch(()=>{});
  await wait(5000);
  await page.bringToFront();
  page.on('dialog', async d => { log(`dialog: ${d.message().substring(0,80)}`); await d.dismiss(); });

  await page.evaluate(() => {
    const b = [...document.querySelectorAll('div[role="button"], button, a')].find(e => e.innerText?.trim() === 'Add update' && e.offsetParent !== null);
    b?.click();
  });
  await wait(5000);

  // Wait for the iframe to load
  let gbpFrame;
  for (let i=0; i<10; i++) {
    gbpFrame = page.frames().find(f => f.url().includes('/local/business/') && f.url().includes('/promote/updates'));
    if (gbpFrame) break;
    await wait(1500);
  }
  if (!gbpFrame) { log('no gbp frame'); await browser.disconnect(); process.exit(3); }
  log(`frame: ${gbpFrame.url().substring(0,120)}`);

  // Wait for compose ready
  await wait(3000);

  // Inspect what's in the frame
  const inv = await gbpFrame.evaluate(() => {
    const txt = [...document.querySelectorAll('textarea, div[contenteditable="true"]')].map(e => ({tag: e.tagName, ph: e.getAttribute('placeholder'), aria: e.getAttribute('aria-label'), w: Math.round(e.getBoundingClientRect().width), visible: e.offsetParent !== null}));
    const inputs = [...document.querySelectorAll('input[type="file"]')].map(e => ({accept: e.accept}));
    const buttons = [...document.querySelectorAll('button')].map(b => b.innerText?.trim()).filter(t => t && t.length < 40);
    return {txt, inputs, buttons: buttons.slice(0, 30)};
  });
  log(`frame inv: ${JSON.stringify(inv)}`);

  // Type text
  const typed = await gbpFrame.evaluate((text) => {
    const el = [...document.querySelectorAll('textarea')].find(e => e.offsetParent !== null);
    if (!el) return null;
    el.focus();
    el.value = text;
    el.dispatchEvent(new Event('input', {bubbles: true}));
    el.dispatchEvent(new Event('change', {bubbles: true}));
    return el.value.length;
  }, TEXT);
  log(`typed: ${typed}`);

  // Try uploading photo (videos may not be supported on GBP posts — fallback to image_url's hero)
  // Try video first
  const fileInputs = await gbpFrame.$$('input[type="file"]').catch(() => []);
  log(`file inputs in frame: ${fileInputs.length}`);
  if (fileInputs.length) {
    try {
      // Use hero image if video isn't acceptable - GBP often rejects video
      // Download hero image first
      const heroPath = '/tmp/gbp-hero.jpg';
      await page.evaluate(async (url) => { /* no-op - download via shell */ }, '');
      // Actually try the hero PNG already on disk. The pending-posts has image_url R2.
      // For now try the video; if fails, fallback to existing image
      await fileInputs[0].uploadFile(VIDEO);
      log('video uploaded to GBP');
      await wait(10000);
    } catch (e) {
      log(`video upload err: ${e.message}`);
    }
  }

  await page.screenshot({path:'/tmp/gbp-composed.png'});

  // Look for Post button
  const posted = await gbpFrame.evaluate(() => {
    const b = [...document.querySelectorAll('button')].find(b => /^Post$/i.test(b.innerText?.trim() || '') && b.offsetParent !== null && !b.disabled);
    if (!b) return null;
    b.click();
    return true;
  });
  log(`Post: ${posted}`);
  if (!posted) { 
    // Maybe the button label is different
    const all = await gbpFrame.evaluate(() => [...document.querySelectorAll('button')].filter(b => b.offsetParent !== null && !b.disabled).map(b => b.innerText?.trim()).filter(Boolean));
    log(`buttons: ${JSON.stringify(all)}`);
  }
  await wait(6000);
  await page.screenshot({path:'/tmp/gbp-final.png'});
  await browser.disconnect();
})().catch(e => { console.error(e); process.exit(1); });
