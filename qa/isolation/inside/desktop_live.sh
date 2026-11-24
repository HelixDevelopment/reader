#!/bin/sh
# isolation: runs-inside-container
# Runs INSIDE the gui-test container: Electron desktop client e2e (Playwright _electron) on the private Xvfb.
set -eu
# SANDBOX_OFF=0 tries the Chromium sandbox (needs unprivileged user namespaces inside the container); 1 passes no-sandbox
export HELIX_DESKTOP_NO_CHROMIUM_SANDBOX=${SANDBOX_OFF:-1}
[ "${SANDBOX_OFF:-1}" = 1 ] || export HELIX_EXPECT_SANDBOX=1
cd /work/desk-e2e
node live.mjs /work/desk-app/helix-reader "$SERVER" "$OPDS_USER" /pw/pw /out
