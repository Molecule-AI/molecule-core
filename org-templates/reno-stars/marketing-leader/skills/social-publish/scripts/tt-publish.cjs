/**
 * TikTok video upload via web. Uses fresh tab.
 * Flow: studio.tiktok.com/upload?lang=en → file picker → caption → Post.
 */
const puppeteer = require('/opt/homebrew/lib/node_modules/puppeteer-core');
const VIDEO = process.argv[2] || '/Users/renostars/dreamina-richmond-whole-house-20260415.mp4';
const CAPTION = process.argv[3] || `Same room. Same angle. Different vibe. 🏡

Richmond townhouse whole house refresh — new floor tile, full repaint, fresh lighting. No demolition drama. Just smart updates that read across every room.

#BeforeAndAfter #WholeHouseRenovation #RichmondRenovation #VancouverRenovation #HomeReno`;
const wait = ms => new Promise(r => setTimeout(r, ms));
const log = m => console.log(`[${new Date().toISOString().substring(11,19)}] ${m}`);

(async () => {
  const browser = await puppeteer.connect({browserURL:'http://127.0.0.1:9222', defaultViewport:null});
  const page = await browser.newPage();
  await page.setViewport({width: 1900, height: 950, deviceScaleFactor: 1});
  await page.goto('https://www.tiktok.com/tiktokstudio/upload?from=upload&lang=en', {waitUntil:'load', timeout:40000}).catch(()=>{});
  await wait(8000);
  await page.bringToFront();
  page.on('dialog', async d => { log(`dialog: ${d.message().substring(0,80)}`); await d.dismiss(); });
  log(`url ${page.url()}`);

  // Wait for and find file input
  let videoIn;
  for (let i=0; i<10; i++) {
    const inputs = await page.$$('input[type="file"]');
    for (const fi of inputs) {
      const acc = await fi.evaluate(el => el.accept || '');
      if (acc.includes('video') || acc === '') { videoIn = fi; break; }
    }
    if (videoIn) break;
    await wait(1500);
  }
  if (!videoIn) { log('no video input'); await page.screenshot({path:'/tmp/tt-fail.png'}); await browser.disconnect(); process.exit(3); }
  await videoIn.uploadFile(VIDEO);
  log('uploaded — waiting for processing');
  await wait(35000); // TikTok takes time

  // Caption — DraftEditor or contenteditable
  const cap = await page.evaluate(() => {
    const sels = ['div[data-e2e="post-editor-textarea"] div[contenteditable="true"]', '.public-DraftEditor-content', 'div[contenteditable="true"][role="textbox"]', 'div[contenteditable="true"]'];
    for (const s of sels) {
      const els = [...document.querySelectorAll(s)].filter(e => {
        const r = e.getBoundingClientRect();
        return r.width > 200 && r.height > 20 && e.offsetParent !== null;
      });
      if (els.length) {
        const r = els[0].getBoundingClientRect();
        return {sel: s, x: Math.round(r.x + r.width/2), y: Math.round(r.y + 20)};
      }
    }
    return null;
  });
  if (!cap) { log('no caption editor'); await page.screenshot({path:'/tmp/tt-fail.png'}); await browser.disconnect(); process.exit(4); }
  log(`caption editor: ${JSON.stringify(cap)}`);
  await page.mouse.click(cap.x, cap.y);
  await wait(500);
  // Select all + delete pre-fill (TikTok pre-fills filename)
  await page.keyboard.down('Meta'); await page.keyboard.press('a'); await page.keyboard.up('Meta');
  await page.keyboard.press('Backspace');
  await wait(300);
  await page.keyboard.type(CAPTION, {delay: 8});
  await wait(2500);

  const len = await page.evaluate(() => {
    const el = document.querySelector('div[data-e2e="post-editor-textarea"], .public-DraftEditor-content, div[contenteditable="true"][role="textbox"]');
    return el?.innerText?.length || 0;
  });
  log(`caption length: ${len}`);
  if (len < 50) { log('typing missed'); await page.screenshot({path:'/tmp/tt-fail.png'}); await browser.disconnect(); process.exit(5); }

  // Wait for upload finish — Post button enabled
  let postedTry = null;
  for (let i=0; i<30; i++) {
    postedTry = await page.evaluate(() => {
      const candidates = [...document.querySelectorAll('button')].filter(b => /^Post$/i.test(b.innerText?.trim() || '') && b.offsetParent !== null);
      const enabled = candidates.find(b => !b.disabled && b.getAttribute('aria-disabled') !== 'true');
      return {found: candidates.length, enabled: !!enabled};
    });
    log(`post-btn poll: ${JSON.stringify(postedTry)}`);
    if (postedTry.enabled) break;
    await wait(3000);
  }
  if (!postedTry?.enabled) { log('post button never enabled'); await page.screenshot({path:'/tmp/tt-fail.png'}); await browser.disconnect(); process.exit(6); }

  await page.evaluate(() => {
    const b = [...document.querySelectorAll('button')].find(b => /^Post$/i.test(b.innerText?.trim() || '') && !b.disabled && b.offsetParent !== null);
    b?.click();
  });
  log('Post clicked');

  for (let i=0; i<60; i++) {
    await wait(2000);
    const state = await page.evaluate(() => {
      const txt = document.body.innerText;
      return {url: location.pathname, success: /Your video is being uploaded|Your post is being processed|posted successfully|Manage your posts/i.test(txt)};
    });
    log(`t+${i*2}s ${JSON.stringify(state)}`);
    if (state.success || state.url.includes('/manage')) { log('SUCCESS'); break; }
  }
  await page.screenshot({path:'/tmp/tt-final.png'});
  await browser.disconnect();
})().catch(e => { console.error(e); process.exit(1); });
