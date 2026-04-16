/**
 * YouTube Short upload via studio.youtube.com.
 * Flow: studio.youtube.com → Create button → Upload videos → file picker → wait for processing → set title/description → Next×3 → Publish.
 */
const puppeteer = require('/opt/homebrew/lib/node_modules/puppeteer-core');
const VIDEO = process.argv[2] || '/Users/renostars/dreamina-richmond-whole-house-20260415.mp4';
const TITLE = 'Whole House Refresh — Richmond Townhouse Transformation #shorts';
const DESC = process.argv[3] || `Whole house renovation doesn't always mean tearing the place apart.

This Richmond townhouse client wanted a budget-friendly refresh — not a gut job. We re-tiled the first floor, repainted every room, and replaced the light fixtures throughout. The footprint, the layout, the plumbing — all stayed the same.

What changed was the FEEL. Better lighting alone can make a room read 5 years younger. Tile across an open-concept first floor unifies what used to feel chopped up. Fresh paint hides a decade of small wear-and-tear.

Sometimes the most powerful renovation is the one that doesn't need permits. 🏡

#WholeHouseRenovation #RichmondRenovation #VancouverRenovation #HomeImprovement #BeforeAndAfter #shorts`;
const wait = ms => new Promise(r => setTimeout(r, ms));
const log = m => console.log(`[${new Date().toISOString().substring(11,19)}] ${m}`);

(async () => {
  const browser = await puppeteer.connect({browserURL:'http://127.0.0.1:9222', defaultViewport:null});
  const page = await browser.newPage();
  await page.setViewport({width: 1900, height: 950, deviceScaleFactor: 1});
  await page.goto('https://studio.youtube.com/', {waitUntil:'load', timeout:40000}).catch(()=>{});
  await wait(8000);
  await page.bringToFront();
  page.on('dialog', async d => { log(`dialog: ${d.message().substring(0,80)}`); await d.dismiss(); });
  log(`url ${page.url()}`);

  // STEP 1: click Create then Upload videos
  await page.evaluate(() => {
    const b = document.querySelector('ytcp-button#create-icon, button[aria-label="Create"]');
    b?.click();
  });
  await wait(1500);
  await page.evaluate(() => {
    // Menu items: "Upload videos" / "Go live"
    const items = [...document.querySelectorAll('tp-yt-paper-item, [role="menuitem"]')];
    const upload = items.find(e => /Upload videos/i.test(e.innerText || ''));
    upload?.click();
  });
  await wait(2500);

  // STEP 2: file input
  const inputs = await page.$$('input[type="file"]');
  if (!inputs.length) { log('no file input'); await page.screenshot({path:'/tmp/yt-fail.png'}); await browser.disconnect(); process.exit(3); }
  await inputs[0].uploadFile(VIDEO);
  log('file uploaded');
  await wait(15000); // wait for upload modal to appear

  // STEP 3: Set title — first textbox
  const titleSet = await page.evaluate((title) => {
    // Find Title contenteditable — usually the first ytcp-mention-textbox div[contenteditable]
    const editors = [...document.querySelectorAll('ytcp-mention-textbox div[contenteditable="true"], div#textbox[contenteditable="true"]')];
    if (!editors.length) return null;
    const titleEl = editors[0];
    titleEl.focus();
    // Clear and set
    document.execCommand('selectAll', false, null);
    document.execCommand('insertText', false, title);
    return editors.length;
  }, TITLE);
  log(`title editors: ${titleSet}`);
  await wait(1000);

  // STEP 4: Description = second editor
  const descSet = await page.evaluate((desc) => {
    const editors = [...document.querySelectorAll('ytcp-mention-textbox div[contenteditable="true"], div#textbox[contenteditable="true"]')];
    if (editors.length < 2) return null;
    const el = editors[1];
    el.focus();
    document.execCommand('selectAll', false, null);
    document.execCommand('insertText', false, desc);
    return true;
  }, DESC);
  log(`desc set: ${descSet}`);
  await wait(1500);

  // STEP 5: "Made for kids" → No (radio name="VIDEO_MADE_FOR_KIDS_NOT_MFK")
  await page.evaluate(() => {
    const radios = [...document.querySelectorAll('tp-yt-paper-radio-button')];
    const noKids = radios.find(r => /No, it's not made for kids/i.test(r.innerText || ''));
    noKids?.click();
  });
  await wait(800);

  // STEP 6: Next × 3 (Details → Video elements → Checks → Visibility)
  for (let n=1; n<=3; n++) {
    const ok = await page.evaluate(() => {
      const b = document.querySelector('ytcp-button#next-button');
      if (!b || b.hasAttribute('disabled')) return null;
      b.click();
      return true;
    });
    log(`Next ${n}: ${ok}`);
    if (!ok) break;
    await wait(2500);
  }

  // STEP 7: Visibility = Public
  await page.evaluate(() => {
    const radios = [...document.querySelectorAll('tp-yt-paper-radio-button[name="PUBLIC"]')];
    radios[0]?.click();
  });
  await wait(800);

  // STEP 8: Publish button
  const pub = await page.evaluate(() => {
    const b = document.querySelector('ytcp-button#done-button');
    if (!b || b.hasAttribute('disabled')) return null;
    b.click();
    return true;
  });
  log(`Publish: ${pub}`);
  if (!pub) { log('publish disabled'); await page.screenshot({path:'/tmp/yt-fail.png'}); await browser.disconnect(); process.exit(5); }

  // Verify — modal closes
  for (let i=0; i<60; i++) {
    await wait(2000);
    const state = await page.evaluate(() => {
      const txt = document.body.innerText;
      return {
        success: /Video published|published successfully|Your video has been published/i.test(txt),
        modalGone: !document.querySelector('ytcp-uploads-dialog'),
      };
    });
    log(`t+${i*2}s ${JSON.stringify(state)}`);
    if (state.success || state.modalGone) { log('PUBLISHED'); break; }
  }
  await page.screenshot({path:'/tmp/yt-final.png'});
  await browser.disconnect();
})().catch(e => { console.error(e); process.exit(1); });
