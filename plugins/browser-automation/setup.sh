#!/bin/bash
# Install puppeteer-core (no bundled Chromium — connects to existing Chrome via CDP)
set -e
npm install -g puppeteer-core 2>/dev/null || true
echo "browser-automation: puppeteer-core installed"
