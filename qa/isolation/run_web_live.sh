#!/usr/bin/env bash
# run_web_live.sh <state-dir> <out-dir>  - clients/web live Playwright run INSIDE the gui-test container.
set -eu
R=$(cd "$(dirname "$0")/../.." && pwd); ST=$1; OUT=$2; "$R/tools/isolation/fresh_out.sh" "$OUT"; . "$ST/env"
exec "$R/tools/isolation/run_container.sh" -n hrdiso-web -p "$OUT/container_proof.json" -m 3g -c 2 \
  -r "$R/clients/web:/work/web" -r "$R/qa/isolation/inside:/work/inside" -r "$HOME/.cache/ms-playwright:/ms-playwright" -w "$OUT:/out" \
  -b "$ST/sock" -e SERVER=http://127.0.0.1:38191 -e "OPDS_USER=$OPDS_USER" -e "OPDS_PASS=$OPDS_PASS" -e PLAYWRIGHT_BROWSERS_PATH=/ms-playwright \
  -- sh /work/inside/web_live.sh
