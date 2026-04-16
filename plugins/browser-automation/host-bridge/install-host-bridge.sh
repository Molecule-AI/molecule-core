#!/usr/bin/env bash
# install-host-bridge.sh — run ONCE on the host machine to keep cdp-proxy alive
# across reboots. Workspaces inside Docker then reach Chrome via the proxy.
#
# Supports macOS (launchd) and Linux (systemd --user). No root required.
#
# Usage:
#   bash install-host-bridge.sh            # install + start
#   bash install-host-bridge.sh uninstall  # stop + remove
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROXY_SCRIPT="${SCRIPT_DIR}/cdp-proxy.cjs"
LABEL="com.molecule.browser-automation.cdp-proxy"
NODE_BIN="$(command -v node || echo /usr/local/bin/node)"
TOKEN_FILE="${HOME}/.molecule-cdp-proxy-token"

if [[ ! -f "$PROXY_SCRIPT" ]]; then
  echo "ERROR: $PROXY_SCRIPT not found" >&2
  exit 1
fi
if [[ ! -x "$NODE_BIN" ]]; then
  echo "ERROR: node not on PATH — install Node.js first" >&2
  exit 1
fi

# #293: generate a per-install auth token so the proxy isn't exposed to the
# LAN without authentication. Written to ~/.molecule-cdp-proxy-token with
# 0600 perms. The proxy reads it at startup; workspace containers read it
# via the bundled connect() helper which mounts the token file over a bind.
ensure_token() {
  if [[ -f "$TOKEN_FILE" ]] && [[ "$(wc -c < "$TOKEN_FILE")" -ge 17 ]]; then
    echo "token: reusing existing $TOKEN_FILE"
    return
  fi
  # 32 bytes of random, hex-encoded → 64 chars. openssl is available on
  # every macOS + most Linux installs; fall back to /dev/urandom if not.
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 32 > "$TOKEN_FILE"
  else
    head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n' > "$TOKEN_FILE"
  fi
  chmod 600 "$TOKEN_FILE"
  echo "token: generated new $TOKEN_FILE (0600)"
}

install_macos() {
  local plist="$HOME/Library/LaunchAgents/${LABEL}.plist"
  local token_val
  token_val="$(cat "$TOKEN_FILE")"
  cat > "$plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>Label</key><string>${LABEL}</string>
  <key>ProgramArguments</key>
  <array>
    <string>${NODE_BIN}</string>
    <string>${PROXY_SCRIPT}</string>
  </array>
  <key>EnvironmentVariables</key>
  <dict>
    <key>CDP_PROXY_TOKEN</key><string>${token_val}</string>
  </dict>
  <key>KeepAlive</key><true/>
  <key>RunAtLoad</key><true/>
  <key>StandardOutPath</key><string>${HOME}/.molecule-cdp-proxy.log</string>
  <key>StandardErrorPath</key><string>${HOME}/.molecule-cdp-proxy.log</string>
</dict></plist>
EOF
  launchctl bootout "gui/$(id -u)/${LABEL}" 2>/dev/null || true
  launchctl bootstrap "gui/$(id -u)" "$plist"
  launchctl kickstart -k "gui/$(id -u)/${LABEL}"
  echo "installed macOS launchd agent: $plist"
  echo "logs: ${HOME}/.molecule-cdp-proxy.log"
}

install_linux() {
  local unit_dir="$HOME/.config/systemd/user"
  mkdir -p "$unit_dir"
  local unit="$unit_dir/${LABEL}.service"
  # Read token from the file at service start instead of embedding it in
  # the unit file — unit files are often world-readable, the token file
  # is 0600. systemd EnvironmentFile reads key=value lines so we write a
  # sidecar file containing CDP_PROXY_TOKEN=<value>.
  local env_file="${HOME}/.molecule-cdp-proxy.env"
  printf 'CDP_PROXY_TOKEN=%s\n' "$(cat "$TOKEN_FILE")" > "$env_file"
  chmod 600 "$env_file"
  cat > "$unit" <<EOF
[Unit]
Description=Molecule browser-automation CDP proxy (host → Chrome)
After=network-online.target

[Service]
Type=simple
EnvironmentFile=${env_file}
ExecStart=${NODE_BIN} ${PROXY_SCRIPT}
Restart=always
RestartSec=5

[Install]
WantedBy=default.target
EOF
  systemctl --user daemon-reload
  systemctl --user enable --now "${LABEL}.service"
  echo "installed systemd user unit: $unit"
  echo "logs: journalctl --user -u ${LABEL}.service -f"
}

uninstall() {
  case "$(uname -s)" in
    Darwin)
      launchctl bootout "gui/$(id -u)/${LABEL}" 2>/dev/null || true
      rm -f "$HOME/Library/LaunchAgents/${LABEL}.plist"
      echo "uninstalled macOS launchd agent"
      ;;
    Linux)
      systemctl --user disable --now "${LABEL}.service" 2>/dev/null || true
      rm -f "$HOME/.config/systemd/user/${LABEL}.service"
      systemctl --user daemon-reload
      echo "uninstalled systemd user unit"
      ;;
  esac
}

case "${1:-install}" in
  install)
    ensure_token
    case "$(uname -s)" in
      Darwin) install_macos ;;
      Linux)  install_linux ;;
      *) echo "unsupported OS: $(uname -s)" >&2; exit 1 ;;
    esac
    echo
    echo "next step: launch your Chrome with --remote-debugging-port=9222 (once per reboot)"
    echo "  macOS: open -na 'Google Chrome' --args --remote-debugging-port=9222 --user-data-dir=\"\$HOME/.chrome-molecule\""
    echo "verify:  curl -H \"X-CDP-Proxy-Token: \$(cat $TOKEN_FILE)\" http://127.0.0.1:9223/json/version"
    echo
    echo "container side: mount $TOKEN_FILE into each workspace and the bundled"
    echo "lib/connect.js helper will read it automatically. Bind:"
    echo "  -v $TOKEN_FILE:/run/secrets/cdp-proxy-token:ro"
    ;;
  uninstall)
    uninstall
    rm -f "${HOME}/.molecule-cdp-proxy.env" 2>/dev/null || true
    echo "note: ${TOKEN_FILE} preserved so a future reinstall keeps the same token."
    echo "      delete manually if you want to rotate."
    ;;
  *) echo "usage: $0 [install|uninstall]"; exit 1 ;;
esac
