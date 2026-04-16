#!/bin/sh
# Tenant entrypoint — starts both Go platform (API) and Canvas (UI).
#
# Go platform listens on :8080 (Fly health checks hit this port).
# Canvas Node.js listens on :3000 (internal only).
# The Go platform's fallback handler proxies non-API routes to :3000
# so the browser only ever talks to :8080.
#
# If either process dies, we kill the other and exit non-zero so Fly
# restarts the machine.

set -e

# Start Canvas in background
cd /canvas
PORT=3000 HOSTNAME=0.0.0.0 node server.js &
CANVAS_PID=$!

# Start Go platform in foreground-ish (we trap signals)
cd /
/platform &
PLATFORM_PID=$!

# If either process exits, kill the other
cleanup() {
  kill $CANVAS_PID 2>/dev/null || true
  kill $PLATFORM_PID 2>/dev/null || true
}
trap cleanup EXIT SIGTERM SIGINT

# Wait for either to exit — whichever exits first triggers cleanup
wait -n $CANVAS_PID $PLATFORM_PID
EXIT_CODE=$?
cleanup
exit $EXIT_CODE
