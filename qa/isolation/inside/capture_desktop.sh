#!/bin/sh
# isolation: runs-inside-container
# Runs INSIDE the gui-test container (HRD-076): the packaged Linux desktop client on the private Xvfb, then capture_desktop.mjs.
set -eu
cd /work/desk-e2e
node /work/inside/capture_desktop.mjs /work/desk-app/helix-reader "$SERVER" "$OPDS_USER" /pw/pw /out
