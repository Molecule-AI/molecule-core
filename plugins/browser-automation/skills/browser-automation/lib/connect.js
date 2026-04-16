/**
 * Single source of truth for connecting to the host Chrome via CDP.
 *
 * ALWAYS use this helper — never call puppeteer.connect() directly. It enforces
 * the two settings that broke the social-media cron repeatedly on 2026-04-15:
 *   - defaultViewport: null  (use real Chrome window dims, not the 800x600 default)
 *   - browserWSEndpoint with proxy host rewrite (works inside Docker AND on host)
 *
 * Usage:
 *   const { connect } = require('./lib/connect');  // adjust path
 *   const browser = await connect();
 *   const page = (await browser.pages())[0];
 *   // ... do work ...
 *   await browser.disconnect();  // NEVER browser.close() (kills shared Chrome)
 */
const puppeteer = require('puppeteer-core');
const http = require('http');

const HOST_DOCKER = 'host.docker.internal';
const HOST_LOCAL = '127.0.0.1';
const PROXY_PORT = 9223;  // CDP proxy (rewrites Host header)
const DIRECT_PORT = 9222; // Chrome's native CDP

function fetchVersion(url) {
  return new Promise((resolve, reject) => {
    const req = http.get(url, r => {
      let d = '';
      r.on('data', c => d += c);
      r.on('end', () => { try { resolve(JSON.parse(d)); } catch (e) { reject(e); } });
    });
    req.on('error', reject);
    req.setTimeout(5000, () => { req.destroy(new Error('timeout')); });
  });
}

async function connect() {
  // Detect environment: are we inside a Docker container or on the host?
  // host.docker.internal resolves only inside containers.
  let host, port;
  try {
    await fetchVersion(`http://${HOST_DOCKER}:${PROXY_PORT}/json/version`);
    host = HOST_DOCKER;
    port = PROXY_PORT;
  } catch {
    // Fallback to direct connection (host script)
    host = HOST_LOCAL;
    port = DIRECT_PORT;
  }

  const data = await fetchVersion(`http://${host}:${port}/json/version`);
  // Rewrite localhost in WS URL to whichever host worked above
  const wsUrl = data.webSocketDebuggerUrl
    .replace('localhost:9222', `${host}:${port}`)
    .replace('127.0.0.1:9222', `${host}:${port}`);

  return puppeteer.connect({
    browserWSEndpoint: wsUrl,
    defaultViewport: null,  // CRITICAL: use Chrome's actual window size
  });
}

module.exports = { connect };
