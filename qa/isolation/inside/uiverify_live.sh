#!/bin/sh
# isolation: runs-inside-container
# Runs INSIDE the gui-test container: the Go uiverify binary first validates its engine on the fixture page, then serves the built prototype on the container loopback and drives
# the Playwright-bundled Chromium against it (headless, no display). Writes screenshots and ui-results.json to /out.
set -eu
CHROME=$(ls /ms-playwright/chromium-*/chrome-linux*/chrome | head -1)
# 1) the locator/visibility engine against known content (golden-good and golden-bad cases): a broken engine must not be able to PASS the product checks
frc=0; /work/bin/uiverify --fixture --timeout 3s --chrome "$CHROME" /out/fixture || frc=$?
# 2) the product checks
prc=0; /work/bin/uiverify --serve /work/dist --chrome "$CHROME" http://127.0.0.1:4173/ /out || prc=$?
[ "$frc" -eq 0 ] || exit "$frc"
exit "$prc"
