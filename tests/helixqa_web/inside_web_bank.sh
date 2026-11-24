#!/bin/sh
# isolation: runs-inside-container
# Runs INSIDE the gui-test container (tools/isolation/run_container.sh): vite preview of the built web
# client, a headless Chromium exposing CDP on 127.0.0.1:9222 (container-private), and the HRD-038
# webrunner driving the HelixQA bank through upstream autonomous.PlaywrightExecutor.
# env: SERVER (host helix-server via pasta map), BANK (bank path), OUT (/out), WEBRUNNER (binary)
set -eu
cd /work/web
HELIX_API_PROXY="$SERVER" node node_modules/vite/bin/vite.js preview --configLoader runner --host 127.0.0.1 --port 4173 >/out/vite.log 2>&1 &
for _ in $(seq 1 50); do curl -fs http://127.0.0.1:4173/ >/dev/null 2>&1 && break; sleep 0.2; done
CHROME=$(ls /ms-playwright/chromium-*/chrome-linux*/chrome | head -1)
# upstream userflow adapter does require('playwright'); only playwright-core is installed -> shim on NODE_PATH.
mkdir -p /tmp/shim/node_modules/playwright
echo "module.exports = require('/work/web/node_modules/playwright-core')" > /tmp/shim/node_modules/playwright/index.js
# upstream scripts never close the CDP connection, so `node -e` hangs forever after printing its single
# result line (RED-3 in evidence). Preload: exit shortly after that line; a rejection still exits non-zero.
cat > /tmp/shim/autoexit.js <<'JS'
const l = console.log.bind(console)
console.log = (...a) => { l(...a); setTimeout(() => process.exit(0), 20) }
JS
"$CHROME" --headless=new --no-sandbox --disable-gpu --remote-debugging-port=9222 --remote-debugging-address=127.0.0.1 \
  --user-data-dir=/tmp/chrome-profile --window-size=1920,1080 about:blank >/out/chrome.log 2>&1 &
for _ in $(seq 1 50); do curl -fs http://127.0.0.1:9222/json/version >/dev/null 2>&1 && break; sleep 0.2; done
curl -fs http://127.0.0.1:9222/json/version | head -c 200 > /out/cdp-version.json || { echo "chrome CDP not up" >&2; exit 70; }
cd /out
[ -n "${NO_AUTOEXIT_SHIM:-}" ] || export NODE_OPTIONS="--require /tmp/shim/autoexit.js" # NO_AUTOEXIT_SHIM=1 only to reproduce the hang (RED)
NODE_PATH=/tmp/shim/node_modules /work/bin/webrunner -bank "$BANK" -cdp http://127.0.0.1:9222 -out /out/shots
