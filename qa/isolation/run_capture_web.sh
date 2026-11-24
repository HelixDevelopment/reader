#!/usr/bin/env bash
# run_capture_web.sh <state-dir> <out-dir> - HRD-076: documentation screenshots + walkthrough video of clients/web, captured INSIDE the
# gui-test container only (private headless browser, --network none, only the unix-socket bridge to the host test server).
# <out-dir> must lie below <repo>/evidence/. Same host-proof / safe_io discipline as run_web_live.sh.
set -eu
R=$(cd "$(dirname "$0")/../.." && pwd); ST=$1; OUT=$2; "$R/tools/isolation/fresh_out.sh" "$OUT"; . "$ST/env"
HW=$ST/host-proof; SIO="python3 -I $R/tools/isolation/safe_io.py"; rm -rf "${HW:?}"; mkdir -p "$HW"
RAW=$HW/raw; mkdir -m 700 "$RAW"
rc=0; "$R/tools/isolation/run_container.sh" -n hrdiso-capweb -o "$RAW" -p "$HW/container_proof.json" -m 3g -c 2 \
  -r "$R/clients/web:/work/web" -r "$R/qa/isolation/inside:/work/inside" -r "$HOME/.cache/ms-playwright:/ms-playwright" -w "$OUT:/out" \
  -b "$ST/sock" -e SERVER=http://127.0.0.1:38191 -e "OPDS_USER=$OPDS_USER" -e "OPDS_PASS=$OPDS_PASS" -e PLAYWRIGHT_BROWSERS_PATH=/ms-playwright \
  -- sh /work/inside/capture_web.sh || rc=$?
[ -f "$HW/container_proof.json" ] && $SIO put "$OUT" container_proof.json < "$HW/container_proof.json"
$SIO scan "$OUT" --allow || { echo "run_capture_web: $OUT holds a link, special file or non-allow-listed file written by the container: tainted" >&2; exit 1; }
exit $rc
