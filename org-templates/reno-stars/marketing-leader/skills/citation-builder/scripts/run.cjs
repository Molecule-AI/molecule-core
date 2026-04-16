#!/usr/bin/env node
/**
 * run.cjs — pick the next pending directory from queue.json, attempt submission,
 * try to verify-via-Gmail if needed, update queue + log, exit.
 *
 * This is the entrypoint the Marketing Leader agent invokes via the
 * citation-builder skill. One run = one directory attempt.
 *
 * Usage:
 *   node run.cjs [--dry-run]
 *
 * Writes:
 *   /configs/skills/citation-builder/queue.json     (status updates)
 *   /configs/skills/citation-builder/log.jsonl      (append-only run log)
 */

const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');

const SKILL_DIR = path.dirname(__dirname); // resolves to skills/citation-builder/
const QUEUE_PATH = path.join(SKILL_DIR, 'queue.json');
const LOG_PATH = path.join(SKILL_DIR, 'log.jsonl');
const PROFILE_PATH = '/configs/business-profile.json';
const SCRIPTS = __dirname; // this file's dir
const DRY = process.argv.includes('--dry-run');

const log = (m) => console.log(`[${new Date().toISOString().substring(11, 19)}] ${m}`);

function loadQueue() {
  return JSON.parse(fs.readFileSync(QUEUE_PATH, 'utf8'));
}
function saveQueue(q) {
  fs.writeFileSync(QUEUE_PATH, JSON.stringify(q, null, 2));
}
function appendLog(obj) {
  fs.appendFileSync(LOG_PATH, JSON.stringify(obj) + '\n');
}

function adapterFor(entry) {
  const slug = entry.name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
  const specific = path.join(SCRIPTS, `${slug}.cjs`);
  return fs.existsSync(specific) ? specific : path.join(SCRIPTS, '_generic.cjs');
}

function runAdapter(adapter, url) {
  log(`adapter ${path.basename(adapter)} ← ${url}`);
  const r = spawnSync('node', [adapter, url, PROFILE_PATH], {
    encoding: 'utf8',
    timeout: 5 * 60 * 1000,
  });
  if (r.error) return { status: 'failed', reason: r.error.message };
  const tail = (r.stdout || '').trim().split('\n').pop();
  try { return JSON.parse(tail); } catch { return { status: 'failed', reason: 'adapter did not emit JSON', raw: tail?.substring(0, 200) }; }
}

function runEmailVerify(senderDomain) {
  const r = spawnSync('node', [path.join(SCRIPTS, 'verify-email-link.cjs'), senderDomain], {
    encoding: 'utf8', timeout: 2 * 60 * 1000,
  });
  const tail = (r.stdout || '').trim().split('\n').pop();
  try { return JSON.parse(tail); } catch { return { status: 'failed', reason: 'verify script did not emit JSON' }; }
}

(function main() {
  const q = loadQueue();
  // Skip status=live, skip, failed-already. Take first pending.
  const next = q.entries.find((e) => e.status === 'pending');
  if (!next) {
    log('queue exhausted — nothing pending');
    appendLog({ ts: new Date().toISOString(), kind: 'queue_exhausted' });
    return;
  }
  log(`→ attempting ${next.name}  (${next.url})`);
  if (DRY) { log('DRY RUN — no submit'); return; }

  const t0 = Date.now();
  const result = runAdapter(adapterFor(next), next.url);
  next.last_attempt = new Date().toISOString();
  next.status = result.status;
  if (result.reason) next.last_reason = result.reason;
  if (result.screenshot) next.last_screenshot = result.screenshot;

  // Auto-attempt Gmail verify if adapter said pending_email_verify
  if (result.status === 'pending_email_verify') {
    const sender = new URL(next.url).hostname.replace(/^(www\.|admin\.)/, '');
    log(`trying gmail verify for sender ${sender}`);
    const v = runEmailVerify(sender);
    log(`verify result: ${JSON.stringify(v)}`);
    if (v.status === 'verified') {
      next.status = 'verified_pending_listing';
      next.last_reason = 'email link clicked; listing may need more steps';
    }
  }

  appendLog({
    ts: new Date().toISOString(),
    kind: 'attempt',
    directory: next.name,
    url: next.url,
    duration_ms: Date.now() - t0,
    result,
  });
  saveQueue(q);
  log(`done — ${next.name}: ${next.status}`);
})();
