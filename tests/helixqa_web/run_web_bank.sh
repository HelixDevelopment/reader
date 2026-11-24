#!/usr/bin/env bash
# run_web_bank.sh - HRD-038: execute qa/helixqa/banks/helix-web.yaml in a rootless, network-less container.
# usage: run_web_bank.sh <helix-server-bin> <out-dir under evidence/> [bank.yaml]      env: WEBRUNNER=<built webrunner binary>
# Host side is headless: server + redis (128 MiB) via tools/isolation/start_server.sh (MODE=hq: the fixed throwaway
# credentials the bank hard-codes). The browser runs ONLY inside the container made by tools/isolation/run_container.sh
# (--network none; the server is reached through the unix-socket bridge on the container's 127.0.0.1:38191).
# Memory: GUI container 1.5 GiB + redis 128 MiB < 2 GiB; one container at a time (refuses if any hrdiso-* is running).
set -eu
SERVER_BIN=$1; OUT=$2; HERE=$(cd "$(dirname "$0")" && pwd); R=$(cd "$HERE/../.." && pwd)
BANK=${3:-$R/qa/helixqa/banks/helix-web.yaml}; : "${WEBRUNNER:?set WEBRUNNER to the built webrunner binary}"
[ -z "$(podman ps -q --filter name=hrdiso)" ] || { echo "another hrdiso-* container is running; one at a time" >&2; exit 75; }
mkdir -p "$OUT"; OUT=$(cd "$OUT" && pwd)
ISO_WORK=$(mktemp -d "${TMPDIR:-/tmp}/hrd038.XXXXXX"); export ISO_WORK; ST=$ISO_WORK/st
cleanup() { "$R/tools/isolation/start_server.sh" stop "$ST" >/dev/null 2>&1 || true; [ -n "${ISO_WORK:-}" ] && rm -rf "${ISO_WORK:?}"; }
trap cleanup EXIT
mkdir -p "$ISO_WORK/lib" "$ISO_WORK/bin" "$ISO_WORK/banks" "$ISO_WORK/inside"
python3 -I "$R/qa/helixqa/fixtures/make_fixtures.py" "$ISO_WORK/lib"
cp "$WEBRUNNER" "$ISO_WORK/bin/webrunner"; chmod 755 "$ISO_WORK/bin/webrunner"
WEB=$R/clients/web; [ -z "${WEBDIR:-}" ] || { cp -a "$WEBDIR" "$ISO_WORK/web"; WEB=$ISO_WORK/web; }   # WEBDIR: a (mutant) copy of the client
cp "$BANK" "$ISO_WORK/banks/bank.yaml"; cp "$HERE/inside_web_bank.sh" "$ISO_WORK/inside/"
MODE=hq "$R/tools/isolation/start_server.sh" start "$SERVER_BIN" "$ST" "$ISO_WORK/lib"
set +e
"$R/tools/isolation/run_container.sh" -n hrdiso-hrd038 -p "$OUT/container_proof.json" -m 1536m -c 2 -t 600 \
  -r "$WEB:/work/web" -r "$ISO_WORK/inside:/work/inside" -r "$ISO_WORK/bin:/work/bin" -r "$ISO_WORK/banks:/work/banks" \
  -r "$HOME/.cache/ms-playwright:/ms-playwright" -w "$OUT:/out" -b "$ST/sock" \
  -e SERVER=http://127.0.0.1:38191 -e BANK=/work/banks/bank.yaml -e PLAYWRIGHT_BROWSERS_PATH=/ms-playwright -e "NO_AUTOEXIT_SHIM=${NO_AUTOEXIT_SHIM:-}" \
  -- sh /work/inside/inside_web_bank.sh > "$OUT/run.txt" 2>&1
RC=$?
# post-run cross-check: report.json re-derived against the files it claims (hashes, counts, pins, attempts, viewport)
if [ $RC -eq 0 ]; then python3 -I "$HERE/verify_report.py" "$OUT/shots" >> "$OUT/run.txt" 2>&1 || RC=$?; else echo "verify_report skipped: run failed" >> "$OUT/run.txt"; fi
echo "exit=$RC" >> "$OUT/run.txt"; cat "$OUT/run.txt"
exit $RC
