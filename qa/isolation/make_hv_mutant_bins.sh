#!/usr/bin/env bash
# make_hv_mutant_bins.sh OUT_DIR   - build the good uiverify binary and two binaries whose locator engine (tools/verify/internal/uiverify/hv.js)
# has isVisible replaced by `return true;` / `return false;` (negative controls for the engine). Host-safe: compiles only, starts no browser.
# Run each with qa/isolation/run_uiverify.sh inside the container: the fixture (uiverify --fixture) MUST FAIL for both mutants.
set -eu
R=$(cd "$(dirname "$0")/../.." && pwd); OUT=${1:?usage: make_hv_mutant_bins.sh OUT_DIR}
case $OUT in "$HOME"/.cache/claude-tmp/?*|"$HOME"/.cache/helix-iso/?*) ;; *) echo "make_hv_mutant_bins: OUT_DIR must be below ~/.cache/claude-tmp or ~/.cache/helix-iso" >&2; exit 64;; esac
OUT=$(realpath -m -- "$OUT"); mkdir -p "$OUT/src"; rm -rf "${OUT:?}/src/verify"; cp -R "$R/tools/verify" "$OUT/src/verify"
( cd "$R/tools/verify" && CGO_ENABLED=0 go build -trimpath -o "$OUT/uiverify-good" ./cmd/uiverify )
cd "$OUT/src/verify"
V='return r.width > 0 && r.height > 0 && st.visibility !== "hidden";'
cp internal/uiverify/hv.js "$OUT/hv.js.orig"
[ "$(grep -c -F -- "$V" "$OUT/hv.js.orig")" = 1 ] || { echo "make_hv_mutant_bins: isVisible anchor not found exactly once" >&2; exit 1; }
for pair in always-visible:true never-visible:false; do n=${pair%%:*}; r=${pair##*:}
  python3 -I - "$OUT/hv.js.orig" internal/uiverify/hv.js "$V" "return $r;" <<'PY'
import sys; s = open(sys.argv[1]).read(); open(sys.argv[2], "w").write(s.replace(sys.argv[3], sys.argv[4]))
PY
  CGO_ENABLED=0 go build -trimpath -o "$OUT/uiverify-hv-$n" ./cmd/uiverify
done
sha256sum "$OUT"/uiverify-*
