const puppeteer = require('/opt/homebrew/lib/node_modules/puppeteer-core');
const wait = ms => new Promise(r => setTimeout(r, ms));
const log = m => console.log(`[${new Date().toISOString().substring(11,19)}] ${m}`);
const VIDEO = '/Users/renostars/dreamina-richmond-whole-house-20260415.mp4';
const CAPTION = `Same room. Same angle. New everything that mattered.

Richmond townhouse — re-tiled the first floor, full repaint, new lighting. Budget-friendly whole house refresh proving you don't always need to gut to transform.

#BeforeAndAfter #WholeHouseRenovation #RichmondHomes #VancouverRenovation #HomeRenovation #TownhouseReno #RenovationDesign #HomeTransform`;

(async () => {
  const browser = await puppeteer.connect({browserURL:'http://127.0.0.1:9222', defaultViewport:null});
  const pages = await browser.pages();
  let page = pages.find(p => p.url().includes('instagram.com')) || pages[0];
  await page.bringToFront();
  await wait(400);
  page.on('dialog', async d => { log(`dialog: ${d.message().substring(0,80)}`); await d.dismiss(); });

  // Close any existing modal first
  await page.evaluate(() => {
    const close = document.querySelector('svg[aria-label="Close"]');
    close?.closest('[role="button"], div[tabindex]')?.click();
  });
  await wait(1000);
  // Confirm discard if asked
  await page.evaluate(() => {
    const discard = [...document.querySelectorAll('button, div[role="button"]')].find(b => /Discard/i.test(b.textContent || '') && b.offsetParent !== null);
    discard?.click();
  });
  await wait(1500);

  // Reload to clean state
  await page.goto('https://www.instagram.com/', {waitUntil:'domcontentloaded',timeout:25000});
  await wait(4000);
  await page.setViewport({width: 1900, height: 950, deviceScaleFactor: 1});
  await wait(500);
  const vp = await page.evaluate(() => ({w: innerWidth, h: innerHeight}));
  log(`viewport ${vp.w}x${vp.h}`);

  // STEP 1: Open New post → Post
  await page.evaluate(() => {
    const np = document.querySelector('svg[aria-label="New post"]');
    np?.closest('a, [role="button"], div[tabindex]')?.click();
  });
  await wait(2000);
  await page.evaluate(() => {
    const opt = [...document.querySelectorAll('span, a, div')].find(e => e.textContent?.trim() === 'Post' && e.offsetParent !== null);
    opt?.click();
  });
  await wait(2500);
  log('opened composer');

  // STEP 2: upload
  const inputs = await page.$$('input[type="file"]');
  if (!inputs.length) { log('no input'); await browser.disconnect(); process.exit(3); }
  await inputs[0].uploadFile(VIDEO);
  log('uploaded');
  await wait(10000);

  // Dismiss reels modal
  await page.evaluate(() => {
    const ok = [...document.querySelectorAll('button')].find(b => ['OK','Ok','Got it'].includes(b.textContent?.trim()) && b.offsetParent !== null);
    ok?.click();
  });
  await wait(2500);

  // STEP 3: Next x2 — modal-top-right filter
  for (let n=1; n<=2; n++) {
    const r = await page.evaluate(() => {
      const c = [...document.querySelectorAll('div[role="button"]')]
        .filter(b => b.textContent?.trim() === 'Next' && b.offsetParent !== null)
        .map(b => ({el: b, r: b.getBoundingClientRect()}))
        .filter(o => o.r.y < 200 && o.r.x > 800);
      if (!c.length) return null;
      c.sort((a,b) => a.r.y - b.r.y);
      c[0].el.click();
      return {x: Math.round(c[0].r.x), y: Math.round(c[0].r.y)};
    });
    log(`Next ${n} ${JSON.stringify(r)}`);
    await wait(4500);
  }

  // STEP 4: Caption — find ON-SCREEN visible+sized lexical box
  const target = await page.evaluate(() => {
    const candidates = [...document.querySelectorAll('div[aria-label="Write a caption..."][data-lexical-editor]')]
      .filter(b => {
        const r = b.getBoundingClientRect();
        return r.x >= 0 && r.x < innerWidth && r.y >= 0 && r.y < innerHeight && r.width > 100 && r.height > 30;
      })
      .sort((a,b) => b.getBoundingClientRect().width - a.getBoundingClientRect().width);
    if (!candidates.length) return null;
    const r = candidates[0].getBoundingClientRect();
    return {x: Math.round(r.x + r.width/2), y: Math.round(r.y + r.height/2), count: candidates.length};
  });
  if (!target) { log('no caption box'); await page.screenshot({path:'/tmp/ig-fail.png'}); await browser.disconnect(); process.exit(3); }
  log(`caption ${JSON.stringify(target)}`);

  await page.mouse.click(target.x, target.y);
  await wait(800);
  const focused = await page.evaluate(() => {
    const a = document.activeElement;
    return {tag: a?.tagName, aria: a?.getAttribute('aria-label'), ce: a?.getAttribute('contenteditable')};
  });
  log(`focused: ${JSON.stringify(focused)}`);

  await page.keyboard.type(CAPTION, {delay: 5});
  await wait(2000);

  const len = await page.evaluate(() => {
    const els = [...document.querySelectorAll('div[aria-label="Write a caption..."]')];
    return els.map(e => e.textContent?.length || 0);
  });
  log(`caption lens: ${JSON.stringify(len)}`);
  if (Math.max(...len, 0) < 100) {
    log('caption typing missed');
    await page.screenshot({path:'/tmp/ig-typed-fail.png'});
    await browser.disconnect();
    process.exit(4);
  }

  // STEP 5: Share
  const share = await page.evaluate(() => {
    const c = [...document.querySelectorAll('div[role="button"]')]
      .filter(b => b.textContent?.trim() === 'Share' && b.offsetParent !== null)
      .map(b => ({el: b, r: b.getBoundingClientRect()}))
      .filter(o => o.r.y < 200 && o.r.x > 800);
    if (!c.length) return null;
    c.sort((a,b) => a.r.y - b.r.y);
    c[0].el.click();
    return {x: Math.round(c[0].r.x), y: Math.round(c[0].r.y)};
  });
  if (!share) { log('no Share'); await browser.disconnect(); process.exit(5); }
  log(`Share ${JSON.stringify(share)}`);

  for (let i=0; i<40; i++) {
    await wait(3000);
    const state = await page.evaluate(() => {
      const txt = document.body.innerText;
      return {
        sharing: /Sharing/i.test(txt),
        shared: /Your reel has been shared|Your post has been shared|Reel shared|reel has been shared/i.test(txt),
        captionStill: !!document.querySelector('div[aria-label="Write a caption..."]'),
      };
    });
    log(`t+${i*3}s ${JSON.stringify(state)}`);
    if (state.shared) { log('SHARED'); break; }
    if (!state.captionStill && !state.sharing && i > 4) { log('composer gone'); break; }
  }

  await page.screenshot({path:'/tmp/ig-final.png'});
  await browser.disconnect();
})().catch(e => { console.error(e); process.exit(1); });
