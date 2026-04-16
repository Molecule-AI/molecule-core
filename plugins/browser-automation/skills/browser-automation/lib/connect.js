/**
 * Single source of truth for connecting to the host Chrome via CDP.
 *
 * ALWAYS use this helper — never call puppeteer.connect() directly. It enforces
 * the two settings that broke the social-media cron repeatedly on 2026-04-15:
 *   - defaultViewport: null  (use real Chrome window dims, not the 800x600 default)
 *   - browserWSEndpoint with proxy host rewrite (works inside Docker AND on host)
 *
 * Authentication (#293):
 *   The host-bridge cdp-proxy requires an X-CDP-Proxy-Token header on every
 *   HTTP request + WebSocket upgrade. This helper reads the token from:
 *     1. CDP_PROXY_TOKEN env var (preferred — set by workspace-template
 *        provisioner from a bind-mounted /run/secrets/cdp-proxy-token)
 *     2. /run/secrets/cdp-proxy-token (mount-time secret — default path)
 *     3. ~/.molecule-cdp-proxy-token (fallback when running directly on host)
 *   If no token can be found, connect() throws — there is no unauth mode.
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
const fs = require('fs');
const os = require('os');
const path = require('path');

const HOST_DOCKER = 'host.docker.internal';
const HOST_LOCAL = '127.0.0.1';
const PROXY_PORT = 9223;  // CDP proxy (rewrites Host header + requires token)
const DIRECT_PORT = 9222; // Chrome's native CDP (host-direct fallback, NO auth)

// Token lookup order — first hit wins. See header comment for rationale.
function loadProxyToken() {
  if (process.env.CDP_PROXY_TOKEN && process.env.CDP_PROXY_TOKEN.length >= 16) {
    return process.env.CDP_PROXY_TOKEN;
  }
  const candidates = [
    '/run/secrets/cdp-proxy-token',
    path.join(os.homedir(), '.molecule-cdp-proxy-token'),
  ];
  for (const p of candidates) {
    try {
      const tok = fs.readFileSync(p, 'utf8').trim();
      if (tok.length >= 16) return tok;
    } catch {
      // try next
    }
  }
  return null;
}

function fetchVersion(url, token) {
  return new Promise((resolve, reject) => {
    const headers = {};
    if (token) headers['X-CDP-Proxy-Token'] = token;
    const req = http.get(url, { headers }, r => {
      let d = '';
      r.on('data', c => d += c);
      r.on('end', () => {
        if (r.statusCode === 401) {
          reject(new Error(`CDP proxy unauthorized (401) — token missing or invalid`));
          return;
        }
        try { resolve(JSON.parse(d)); } catch (e) { reject(e); }
      });
    });
    req.on('error', reject);
    req.setTimeout(5000, () => { req.destroy(new Error('timeout')); });
  });
}

async function connect() {
  const token = loadProxyToken();

  // Detect environment: are we inside a Docker container or on the host?
  // host.docker.internal resolves only inside containers.
  let host, port, usingProxy;
  try {
    // Proxy path — token REQUIRED. Throw on missing so the user fixes it
    // at install time rather than silently falling back to an unauth host
    // connection that only works on the host machine itself.
    if (!token) {
      throw new Error('no token — skip proxy path');
    }
    await fetchVersion(`http://${HOST_DOCKER}:${PROXY_PORT}/json/version`, token);
    host = HOST_DOCKER;
    port = PROXY_PORT;
    usingProxy = true;
  } catch {
    // Fallback to direct Chrome CDP (host script running ON the host,
    // no proxy involved). No token needed — Chrome's own port 9222 is
    // loopback-only and doesn't check Host headers from 127.0.0.1.
    host = HOST_LOCAL;
    port = DIRECT_PORT;
    usingProxy = false;
  }

  const data = await fetchVersion(`http://${host}:${port}/json/version`, usingProxy ? token : null);
  // Rewrite localhost in WS URL to whichever host worked above
  const wsUrl = data.webSocketDebuggerUrl
    .replace('localhost:9222', `${host}:${port}`)
    .replace('127.0.0.1:9222', `${host}:${port}`);

  const connectOpts = {
    browserWSEndpoint: wsUrl,
    defaultViewport: null,  // CRITICAL: use Chrome's actual window size
  };
  if (usingProxy) {
    // puppeteer-core v21+ supports connection headers. The proxy's WS
    // upgrade handler validates X-CDP-Proxy-Token before forwarding to
    // Chrome; without this header the upgrade returns 401.
    connectOpts.headers = { 'X-CDP-Proxy-Token': token };
  }

  return puppeteer.connect(connectOpts);
}

module.exports = { connect };
