#!/usr/bin/env node
/**
 * _generic.cjs — one-size-fits-some directory submitter.
 *
 * Usage:
 *   node _generic.cjs <url> <profile-json-path>
 *
 * Exits 0 with a JSON object on the final stdout line:
 *   {"status": "live"|"pending_email_verify"|"pending_human"|"failed",
 *    "reason": "<short>", "public_url": "<url|null>"}
 *
 * Strategy: fetch the add-business URL, pattern-match form fields by name/
 * placeholder/aria-label, fill what we can from business-profile.json, click
 * the most-prominent submit button, then screenshot + classify the response.
 *
 * Never retries. Never spams. One attempt per invocation. If the generic
 * shape doesn't match, we escalate to pending_human so a per-site adapter
 * can be written later from the screenshot.
 */

const { connect } = require('/configs/plugins/browser-automation/skills/browser-automation/lib/connect');
const fs = require('fs');
const path = require('path');

const URL = process.argv[2];
const PROFILE_PATH = process.argv[3] || '/configs/business-profile.json';

if (!URL) {
  console.error(JSON.stringify({ status: 'failed', reason: 'missing url arg' }));
  process.exit(1);
}

const log = (m) => console.log(`[${new Date().toISOString().substring(11, 19)}] ${m}`);
const wait = (ms) => new Promise((r) => setTimeout(r, ms));

// Field synonyms — lowercase substrings we try against (name, id, placeholder,
// aria-label, neighboring label text). Order matters for deduplication.
const FIELDS = [
  { key: 'business_name', patterns: ['business name', 'company name', 'company', 'business', 'name of business'] },
  { key: 'first_name', patterns: ['first name', 'firstname', 'given name'] },
  { key: 'last_name', patterns: ['last name', 'lastname', 'surname', 'family name'] },
  { key: 'full_name', patterns: ['full name', 'your name', 'contact name'] },
  { key: 'email', patterns: ['email', 'e-mail'] },
  { key: 'phone', patterns: ['phone', 'telephone', 'mobile', 'tel'] },
  { key: 'website', patterns: ['website', 'url', 'web address', 'web'] },
  { key: 'address', patterns: ['street address', 'street', 'address line', 'address'] },
  { key: 'city', patterns: ['city', 'town'] },
  { key: 'province', patterns: ['province', 'state', 'region'] },
  { key: 'postal', patterns: ['postal', 'zip', 'post code'] },
  { key: 'country', patterns: ['country'] },
  { key: 'category', patterns: ['category', 'industry', 'business type', 'services'] },
  { key: 'description', patterns: ['description', 'about', 'bio', 'details'] },
  { key: 'password', patterns: ['password', 'choose password'] },
  { key: 'password_confirm', patterns: ['confirm password', 're-enter password', 'password again'] },
];

async function classifyInputs(page) {
  // Snapshot every input/textarea/select with its discoverable label.
  return page.evaluate(() => {
    const descLabel = (el) => {
      const byAria = el.getAttribute('aria-label');
      if (byAria) return byAria;
      const byId = el.id ? document.querySelector(`label[for="${el.id}"]`) : null;
      if (byId) return byId.innerText;
      // walk back up for an enclosing <label>
      let cur = el;
      for (let i = 0; i < 4 && cur; i++) {
        if (cur.tagName === 'LABEL') return cur.innerText;
        cur = cur.parentElement;
      }
      return '';
    };
    return [...document.querySelectorAll('input, textarea, select')]
      .filter((el) => el.offsetParent !== null)
      .filter((el) => !['hidden', 'submit', 'button'].includes(el.type))
      .map((el) => {
        const r = el.getBoundingClientRect();
        return {
          tag: el.tagName,
          type: el.type,
          name: el.name || '',
          id: el.id || '',
          placeholder: el.placeholder || '',
          label: descLabel(el),
          required: !!el.required,
          x: Math.round(r.x),
          y: Math.round(r.y),
        };
      });
  });
}

function matchField(input) {
  const haystack = [input.name, input.id, input.placeholder, input.label]
    .join(' | ')
    .toLowerCase();
  for (const f of FIELDS) {
    if (f.patterns.some((p) => haystack.includes(p))) return f.key;
  }
  return null;
}

async function run() {
  if (!fs.existsSync(PROFILE_PATH)) {
    console.log(JSON.stringify({ status: 'failed', reason: `profile not found at ${PROFILE_PATH}` }));
    process.exit(0);
  }
  const p = JSON.parse(fs.readFileSync(PROFILE_PATH, 'utf8'));
  const creds = { email: p.email, password: 'RenoStars2026!Directory' };

  const values = {
    business_name: p.name,
    first_name: 'Reno Stars',
    last_name: 'Vancouver',
    full_name: 'Reno Stars Vancouver',
    email: creds.email,
    phone: p.phone,
    website: p.website,
    address: p.address?.street || '',
    city: p.address?.city || '',
    province: p.address?.province || '',
    postal: p.address?.postal || '',
    country: p.address?.country || 'Canada',
    category: (p.categories_primary || 'General Contractor'),
    description: p.description_short || p.description_long || '',
    password: creds.password,
    password_confirm: creds.password,
  };

  const browser = await connect();
  const page = await browser.newPage();
  await page.setViewport({ width: 1600, height: 960, deviceScaleFactor: 1 });
  page.on('dialog', async (d) => { await d.dismiss(); });

  try {
    await page.goto(URL, { waitUntil: 'networkidle2', timeout: 40000 });
  } catch (e) {
    // networkidle2 often times out on sites with long-poll or ads; proceed anyway
    log(`goto finished with ${e.code || e.message}`);
  }
  await wait(3500);

  const inputs = await classifyInputs(page);
  log(`saw ${inputs.length} visible fields`);

  if (inputs.length === 0) {
    const shot = `/tmp/citation-${Date.now()}.png`;
    await page.screenshot({ path: shot });
    await browser.disconnect();
    console.log(JSON.stringify({ status: 'pending_human', reason: 'no visible form inputs', screenshot: shot }));
    return;
  }

  // Fill first input matching each canonical key (don't double-fill)
  const filled = {};
  for (const inp of inputs) {
    const key = matchField(inp);
    if (!key || filled[key]) continue;
    const v = values[key];
    if (!v) continue;
    try {
      // Click first to ensure focus, then type
      await page.mouse.click(inp.x + 10, inp.y + 10);
      await wait(80);
      // Clear any prefill
      await page.keyboard.down('Meta');
      await page.keyboard.press('a');
      await page.keyboard.up('Meta');
      await page.keyboard.press('Backspace');
      await wait(60);
      await page.keyboard.type(String(v), { delay: 15 });
      filled[key] = true;
    } catch (e) {
      log(`fill ${key}: ${e.message}`);
    }
  }
  log(`filled: ${Object.keys(filled).join(', ')}`);
  await wait(600);

  // Submit — pick the most prominent enabled button matching our keyword set
  const submitted = await page.evaluate(() => {
    const kw = /(submit|continue|register|sign.?up|get.?started|add.my.business|add.your.business|list.my.business|create.account|save)/i;
    const btns = [...document.querySelectorAll('button, input[type="submit"]')]
      .filter((b) => b.offsetParent !== null && !b.disabled);
    const matches = btns.filter((b) => kw.test((b.textContent || b.value || '').trim()));
    const pick = (matches.length ? matches : btns)[0];
    if (!pick) return null;
    const label = (pick.textContent || pick.value || '').trim();
    pick.click();
    return label;
  });
  log(`submit: ${submitted}`);

  if (!submitted) {
    const shot = `/tmp/citation-${Date.now()}.png`;
    await page.screenshot({ path: shot });
    await browser.disconnect();
    console.log(JSON.stringify({ status: 'pending_human', reason: 'no submit button found', screenshot: shot, filled: Object.keys(filled) }));
    return;
  }

  await wait(7000);
  const shot = `/tmp/citation-${Date.now()}.png`;
  await page.screenshot({ path: shot });
  const body = await page.evaluate(() => document.body?.innerText?.substring(0, 1500) || '');
  await browser.disconnect();

  if (/captcha|robot|are you human|cloudflare/i.test(body)) {
    console.log(JSON.stringify({ status: 'pending_human', reason: 'captcha or bot challenge', screenshot: shot }));
    return;
  }
  if (/verify your email|check your email|confirmation email|activation email|verification link/i.test(body)) {
    console.log(JSON.stringify({ status: 'pending_email_verify', reason: 'awaiting email verification', screenshot: shot }));
    return;
  }
  if (/thank you|successfully|listing is (now )?(live|active|published)|business (has been )?submitted|profile (created|saved)/i.test(body)) {
    console.log(JSON.stringify({ status: 'live', reason: 'success page shown', screenshot: shot }));
    return;
  }

  console.log(JSON.stringify({ status: 'pending_human', reason: 'unknown post-submit state', screenshot: shot }));
}

run().catch((e) => {
  console.error(e);
  console.log(JSON.stringify({ status: 'failed', reason: e.message || String(e) }));
  process.exit(0);
});
