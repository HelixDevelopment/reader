#!/usr/bin/env bash
# run_uiverify.sh <built-prototype-dir> <out-dir>  - the Go UI checks (tools/verify/cmd/uiverify, replacing scripts/verify_ui.py)
# INSIDE the rootless gui-test container: no network, no host display, no host browser. Needs:
#   ISO_WORK              dedicated scratch root (see docs/test-isolation.md)
#   HELIX_UIVERIFY_BIN    static binary: (cd tools/verify && CGO_ENABLED=0 go build -trimpath -o <bin> ./cmd/uiverify)
#   <built-prototype-dir> output of `npm run build` in demos/interactive_static_demo (the dist/ directory)
set -eu
R=$(cd "$(dirname "$0")/../.." && pwd); DIST=${1:?usage: run_uiverify.sh DIST OUT}; OUT=${2:?usage: run_uiverify.sh DIST OUT}
WORK=${ISO_WORK:?}; BIN=${HELIX_UIVERIFY_BIN:?}
# validate ISO_WORK (dedicated root, canonical, no symlink, not a checkout) BEFORE anything is deleted under it (constitution 11.4.287)
python3 -I - "$R/tools/isolation" "$WORK" "$R" <<'PY' || exit 64
import sys
sys.path.insert(0, sys.argv[1]); import bindcheck
e = bindcheck.iso_error(sys.argv[2], bindcheck.home_of(), sys.argv[3])
if e:
    print("run_uiverify: " + e, file=sys.stderr); sys.exit(64)
PY
"$R/tools/isolation/fresh_out.sh" "$OUT"
W=$WORK/uiv; SIO="python3 -I $R/tools/isolation/safe_io.py"
rm -rf "${W:?}"; mkdir -p "$W/bin" "$W/host-proof"; cp -- "$BIN" "$W/bin/uiverify"; chmod 755 "$W/bin/uiverify"
mkdir -p "$W/dist"; cp -R -- "$DIST/." "$W/dist/"
RAW=$W/host-proof/raw; mkdir -m 700 "$RAW"
rc=0; "$R/tools/isolation/run_container.sh" -n hrdiso-uiv -o "$RAW" -p "$W/host-proof/container_proof.json" -m 2g -c 2 \
  -r "$W/bin:/work/bin" -r "$W/dist:/work/dist" -r "$R/qa/isolation/inside:/work/inside" -r "$HOME/.cache/ms-playwright:/ms-playwright" -w "$OUT:/out" \
  -e PLAYWRIGHT_BROWSERS_PATH=/ms-playwright -- sh /work/inside/uiverify_live.sh || rc=$?
[ -f "$W/host-proof/container_proof.json" ] && $SIO put "$OUT" container_proof.json < "$W/host-proof/container_proof.json"
$SIO scan "$OUT" --allow || { echo "run_uiverify: $OUT holds a link, special file, control artifact or non-allow-listed file written by the container: evidence is tainted" >&2; exit 1; }
exit $rc
