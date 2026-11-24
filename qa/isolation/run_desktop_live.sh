#!/usr/bin/env bash
# run_desktop_live.sh <state-dir> <out-dir> - clients/desktop Electron e2e INSIDE the gui-test container.
set -eu
R=$(cd "$(dirname "$0")/../.." && pwd); ST=$1; OUT=$2; "$R/tools/isolation/fresh_out.sh" "$OUT"; . "$ST/env"
umask 077; printf %s "$OPDS_PASS" > "$ST/pw"
exec "$R/tools/isolation/run_container.sh" -n hrdiso-desktop -p "$OUT/container_proof.json" -m 3g -c 2 \
  -r "$R/clients/desktop/out/helix-reader-linux-x64:/work/desk-app" -r "$R/clients/desktop/e2e:/work/desk-e2e" \
  -r "$R/clients/web/node_modules:/work/node_modules" -r "$R/qa/isolation/inside:/work/inside" -r "$ST/pw:/pw/pw" -w "$OUT:/out" \
  -b "$ST/sock" -e "SANDBOX_OFF=${SANDBOX_OFF:-1}" -e SERVER=http://127.0.0.1:38191 -e "OPDS_USER=$OPDS_USER" -- sh /work/inside/desktop_live.sh
