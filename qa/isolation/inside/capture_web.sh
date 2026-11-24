#!/bin/sh
# isolation: runs-inside-container
# Runs INSIDE the gui-test container (HRD-076): vite preview of the built web client, then capture_web.mjs against the host server
# reached at $SERVER (unix-socket bridge on the container loopback). Output goes to /out.
set -eu
cd /work/web
HELIX_API_PROXY="$SERVER" node node_modules/vite/bin/vite.js preview --configLoader runner --host 127.0.0.1 --port 4173 >/dev/null 2>&1 &
for _ in $(seq 1 50); do curl -fs http://127.0.0.1:4173/ >/dev/null 2>&1 && break; sleep 0.2; done
CHROME=$(ls /ms-playwright/chromium-*/chrome-linux*/chrome | head -1)
node /work/inside/capture_web.mjs http://127.0.0.1:4173/ "$SERVER" "$OPDS_USER" "$OPDS_PASS" /out "$CHROME"
