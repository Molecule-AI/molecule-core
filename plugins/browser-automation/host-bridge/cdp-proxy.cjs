#!/usr/bin/env node
/**
 * CDP proxy — bridges a Docker container to the user's Chrome running on the host.
 *
 * Why: Chrome on macOS rejects DevTools Protocol connections whose Host header
 * is anything other than `localhost`. A container hitting `host.docker.internal:9222`
 * fails the check. This proxy listens on 0.0.0.0:9223, rewrites the Host header,
 * and forwards both HTTP (tab listing, screenshots) and WebSocket upgrades.
 *
 * Usage:
 *   # Launch your Chrome with the debug port once (once per reboot):
 *   open -na "Google Chrome" --args \
 *     --user-data-dir="$HOME/.chrome-molecule" \
 *     --profile-directory=Default \
 *     --remote-debugging-port=9222
 *
 *   # Then start the proxy (stays in foreground; run in a launchd/systemd unit):
 *   node cdp-proxy.cjs
 *
 * Env overrides:
 *   CHROME_PORT  (default 9222)
 *   PROXY_PORT   (default 9223)
 *   BIND_ADDR    (default 0.0.0.0)
 *
 * Container side: connect via `host.docker.internal:9223` (Docker Desktop) or
 * `172.17.0.1:9223` (Linux). The bundled `lib/connect.js` helper auto-detects.
 */
const http = require('http');
const net = require('net');

const CHROME_PORT = parseInt(process.env.CHROME_PORT || '9222', 10);
const PROXY_PORT = parseInt(process.env.PROXY_PORT || '9223', 10);
const BIND_ADDR = process.env.BIND_ADDR || '0.0.0.0';

const proxy = http.createServer((req, res) => {
  const options = {
    hostname: '127.0.0.1',
    port: CHROME_PORT,
    path: req.url,
    method: req.method,
    headers: { ...req.headers, host: `localhost:${CHROME_PORT}` },
  };
  const proxyReq = http.request(options, (proxyRes) => {
    res.writeHead(proxyRes.statusCode, proxyRes.headers);
    proxyRes.pipe(res);
  });
  req.pipe(proxyReq);
  proxyReq.on('error', (e) => {
    res.writeHead(502);
    res.end(`proxy error: ${e.code || e.message}`);
  });
});

proxy.on('upgrade', (req, socket, head) => {
  const conn = net.connect(CHROME_PORT, '127.0.0.1', () => {
    const upgradeReq =
      `${req.method} ${req.url} HTTP/1.1\r\n` +
      `Host: localhost:${CHROME_PORT}\r\n` +
      Object.entries(req.headers)
        .filter(([k]) => k.toLowerCase() !== 'host')
        .map(([k, v]) => `${k}: ${v}`)
        .join('\r\n') +
      '\r\n\r\n';
    conn.write(upgradeReq);
    if (head.length) conn.write(head);
    socket.pipe(conn);
    conn.pipe(socket);
  });
  conn.on('error', () => socket.destroy());
  socket.on('error', () => conn.destroy());
});

proxy.listen(PROXY_PORT, BIND_ADDR, () => {
  console.log(`cdp-proxy listening on ${BIND_ADDR}:${PROXY_PORT} → 127.0.0.1:${CHROME_PORT}`);
});

process.on('SIGTERM', () => proxy.close(() => process.exit(0)));
process.on('SIGINT', () => proxy.close(() => process.exit(0)));
