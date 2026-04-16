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

if [[ ! -f "$PROXY_SCRIPT" ]]; then
  echo "ERROR: $PROXY_SCRIPT not found" >&2
  exit 1
fi
if [[ ! -x "$NODE_BIN" ]]; then
  echo "ERROR: node not on PATH — install Node.js first" >&2
  exit 1
fi

install_macos() {
  local plist="$HOME/Library/LaunchAgents/${LABEL}.plist"
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
  cat > "$unit" <<EOF
[Unit]
Description=Molecule browser-automation CDP proxy (host → Chrome)
After=network-online.target

[Service]
Type=simple
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
    case "$(uname -s)" in
      Darwin) install_macos ;;
      Linux)  install_linux ;;
      *) echo "unsupported OS: $(uname -s)" >&2; exit 1 ;;
    esac
    echo
    echo "next step: launch your Chrome with --remote-debugging-port=9222 (once per reboot)"
    echo "  macOS: open -na 'Google Chrome' --args --remote-debugging-port=9222 --user-data-dir=\"\$HOME/.chrome-molecule\""
    echo "verify:  curl http://127.0.0.1:9223/json/version"
    ;;
  uninstall) uninstall ;;
  *) echo "usage: $0 [install|uninstall]"; exit 1 ;;
esac
