#!/usr/bin/env bash
# run_web_live.sh <state-dir> <out-dir>  - clients/web live Playwright run INSIDE the gui-test container.
set -eu
R=$(cd "$(dirname "$0")/../.." && pwd); ST=$1; OUT=$2; "$R/tools/isolation/fresh_out.sh" "$OUT"; . "$ST/env"
HW=$ST/host-proof; SIO="python3 -I $R/tools/isolation/safe_io.py"; rm -rf "${HW:?}"; mkdir -p "$HW"   # HOST-ONLY: the proof is written here, never into the container-writable $OUT
RAW=$HW/raw; mkdir -m 700 "$RAW"   # R6-I1: attached mode needs a terminal on stdout; this runner is also run from pipes / CI-less automation, so it uses the raw-capture mode (host-only dir; the terminal gets a sanitized copy on stderr)
rc=0; "$R/tools/isolation/run_container.sh" -n hrdiso-web -o "$RAW" -p "$HW/container_proof.json" -m 3g -c 2 \
  -r "$R/clients/web:/work/web" -r "$R/qa/isolation/inside:/work/inside" -r "$HOME/.cache/ms-playwright:/ms-playwright" -w "$OUT:/out" \
  -b "$ST/sock" -e SERVER=http://127.0.0.1:38191 -e "OPDS_USER=$OPDS_USER" -e "OPDS_PASS=$OPDS_PASS" -e PLAYWRIGHT_BROWSERS_PATH=/ms-playwright \
  -- sh /work/inside/web_live.sh || rc=$?
# the container is gone: publish the host-written proof through safe_io, then refuse tainted / non-allow-listed / oversized output (review round 3)
[ -f "$HW/container_proof.json" ] && $SIO put "$OUT" container_proof.json < "$HW/container_proof.json"
$SIO scan "$OUT" --allow || { echo "run_web_live: $OUT holds a link, special file, control artifact or non-allow-listed file written by the container: evidence is tainted" >&2; exit 1; }
exit $rc
