/**
 * LinkedIn company-page video post.
 * Flow: /company/103326696/admin/dashboard → "Create" → "Start a post" → "Add media" (legacy picker) → upload → Next → caption → Post.
 */
const puppeteer = require('/opt/homebrew/lib/node_modules/puppeteer-core');
const VIDEO = process.argv[2] || '/Users/renostars/dreamina-richmond-whole-house-20260415.mp4';
const CAPTION = process.argv[3] || `A common ask we get: "how much can I really change without gutting?"

This Richmond townhouse is the answer. We re-tiled the first floor, repainted the entire home, and swapped out the lighting. No structural work, no plumbing relocation — but the finished result reads as a completely different home.

For owners weighing renovate-vs-sell-vs-do-nothing, scope discipline like this is usually the highest-ROI play.

Free consultation → 778-960-7999 | reno-stars.com

#Renovation #Vancouver #HomeImprovement #WholeHouseRenovation`;
const wait = ms => new Promise(r => setTimeout(r, ms));
const log = m => console.log(`[${new Date().toISOString().substring(11,19)}] ${m}`);

(async () => {
  const browser = await puppeteer.connect({browserURL:'http://127.0.0.1:9222', defaultViewport:null});
  const page = await browser.newPage();
  await page.setViewport({width: 1900, height: 950, deviceScaleFactor: 1});
  await page.goto('https://www.linkedin.com/company/103326696/admin/dashboard/', {waitUntil:'load', timeout:40000}).catch(()=>{});
  await wait(6000);
  await page.bringToFront();
  page.on('dialog', async d => { log(`dialog: ${d.message().substring(0,80)}`); await d.dismiss(); });

  log(`url ${page.url()}`);

  // STEP 1: click "Create" button
  const c1 = await page.evaluate(() => {
    const b = [...document.querySelectorAll('button')].find(b => b.innerText?.trim() === 'Create' && b.offsetParent !== null);
    if (!b) return null;
    const r = b.getBoundingClientRect();
    b.click();
    return {x: Math.round(r.x), y: Math.round(r.y)};
  });
  log(`Create: ${JSON.stringify(c1)}`);
  await wait(1500);

  // STEP 2: click "Start a post"
  const c2 = await page.evaluate(() => {
    const candidates = [...document.querySelectorAll('*')].filter(e => {
      for (const node of e.childNodes) if (node.nodeType === 3 && /^Start a post$/i.test(node.textContent.trim())) return true;
      return false;
    });
    for (const cand of candidates) {
      let cur = cand;
      for (let i=0; i<6 && cur; i++) {
        if (cur.tagName === 'A' || cur.tagName === 'BUTTON' || cur.getAttribute?.('role') === 'button') {
          if (cur.offsetParent !== null) {
            cur.click();
            return {tag: cur.tagName};
          }
        }
        cur = cur.parentElement;
      }
    }
    return null;
  });
  log(`Start a post: ${JSON.stringify(c2)}`);
  await wait(3000);

  // STEP 3: click "Add media" (registers file chooser first)
  const fcPromise = page.waitForFileChooser({timeout: 8000});
  const c3 = await page.evaluate(() => {
    const b = [...document.querySelectorAll('button[aria-label="Add media"], button')]
      .filter(b => b.getAttribute('aria-label') === 'Add media' && b.offsetParent !== null);
    if (!b.length) return null;
    const r = b[0].getBoundingClientRect();
    b[0].click();
    return {x: Math.round(r.x), y: Math.round(r.y)};
  });
  log(`Add media: ${JSON.stringify(c3)}`);
  let chooser;
  try { chooser = await fcPromise; } catch (e) { log('file chooser timeout'); await page.screenshot({path:'/tmp/li-fail.png'}); await browser.disconnect(); process.exit(3); }
  await chooser.accept([VIDEO]);
  log('file accepted');
  await wait(20000); // allow upload

  // STEP 4: click Next
  for (let n=1; n<=2; n++) {
    const nxt = await page.evaluate(() => {
      const b = [...document.querySelectorAll('button')].find(b => /^Next$/i.test(b.innerText?.trim() || '') && b.offsetParent !== null && !b.disabled);
      if (!b) return null;
      b.click();
      return true;
    });
    log(`Next ${n}: ${nxt}`);
    if (!nxt) break;
    await wait(3500);
  }

  // STEP 5: type caption — .ql-editor
  const cap = await page.evaluate(() => {
    const el = document.querySelector('.ql-editor');
    if (!el) return null;
    const r = el.getBoundingClientRect();
    return {x: Math.round(r.x + r.width/2), y: Math.round(r.y + 20)};
  });
  if (!cap) { log('no .ql-editor'); await page.screenshot({path:'/tmp/li-fail.png'}); await browser.disconnect(); process.exit(4); }
  await page.mouse.click(cap.x, cap.y);
  await wait(500);
  await page.keyboard.type(CAPTION, {delay: 6});
  await wait(2000);

  const len = await page.evaluate(() => document.querySelector('.ql-editor')?.innerText?.length || 0);
  log(`caption length ${len}`);
  if (len < 100) { log('caption typing missed'); await page.screenshot({path:'/tmp/li-fail.png'}); await browser.disconnect(); process.exit(5); }

  // STEP 6: Post button
  const posted = await page.evaluate(() => {
    const b = [...document.querySelectorAll('button')].find(b => b.innerText?.trim() === 'Post' && b.offsetParent !== null && !b.disabled);
    if (!b) return null;
    b.click();
    return true;
  });
  log(`Post: ${posted}`);
  if (!posted) { log('no Post button'); await page.screenshot({path:'/tmp/li-fail.png'}); await browser.disconnect(); process.exit(6); }

  // Verify
  for (let i=0; i<30; i++) {
    await wait(2000);
    const state = await page.evaluate(() => {
      const txt = document.body.innerText;
      return {
        url: location.pathname,
        success: /Post successful|Your post has been shared/i.test(txt),
        editorGone: !document.querySelector('.ql-editor'),
      };
    });
    log(`t+${i*2}s ${JSON.stringify(state)}`);
    if (state.success || (state.editorGone && i > 3)) { log('SUCCESS'); break; }
  }
  await page.screenshot({path:'/tmp/li-final.png'});
  await browser.disconnect();
})().catch(e => { console.error(e); process.exit(1); });
