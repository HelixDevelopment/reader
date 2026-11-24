#!/usr/bin/env bash
# run_h3_live.sh <state-dir> <out-dir> - LIVE HTTP/3 + Brotli proof for the Helix Reader clients, entirely in rootless containers on an
# INTERNAL podman network (tools/isolation/h3_lab.sh; docs/test-isolation.md section 8). Judged from the SERVER access log, not client claims.
# Scenarios (each has a paired control so a pass cannot be an artefact):
#   desktop-new            Electron shell (this change)  + private-CA QUIC settings            -> expect HTTP/3.0 + br
#   desktop-baseline       Electron shell of PR #18 (Node fetch), same settings                 -> RED: expect HTTP/1.1
#   desktop-no-quic-origin new shell without the QUIC force setting (private CA)               -> expect HTTP/2.0 (documented limit)
#   chromium-web           stock Chromium (what the web client runs in), forced QUIC            -> expect HTTP/3.0 + br
#   chromium-web-no-force  stock Chromium, no force (private CA: Alt-Svc discovery refused)     -> expect HTTP/2.0 + br
#   go-quic-client         quic-go client (helix-http3-check)                                   -> expect HTTP/3.0
# env: HELIX_SERVER_BIN (helix-server build with HELIX_ACCESS_LOG), HELIX_H3CHECK_BIN (helix-http3-check), BASELINE_REF
set -u
R=$(cd "$(dirname "$0")/../.." && pwd); ST=${1:?state dir}; OUT=${2:?out dir}; BIN=${HELIX_SERVER_BIN:?path to a helix-server build with HELIX_ACCESS_LOG support}
H3C=${HELIX_H3CHECK_BIN:?path to helix-http3-check}
export ISO_WORK=$ST; mkdir -p "$OUT" "$ST/out"; LAB="$R/tools/isolation/h3_lab.sh"
"$LAB" down "$ST" 2>/dev/null; rm -rf "${ST:?}/lab"; "$LAB" up "$ST" "$BIN" || exit 1
trap '"$LAB" down "$ST"' EXIT
openssl x509 -in "$ST/lab/srv.pem" -noout -pubkey | openssl pkey -pubin -outform der | openssl dgst -sha256 -binary | base64 > "$ST/spki.txt"; SPKI=$(cat "$ST/spki.txt")
CHROME=/ms-playwright/chromium-1228/chrome-linux64/chrome; EL=/work/desktop/node_modules/electron/dist/electron
cp "$H3C" "$ST/lab/helix-http3-check"; chmod 755 "$ST/lab/helix-http3-check"
rc=0; mkdir -p "$ST/red_app"
if [ ! -d "$ST/red_app/clients/desktop" ]; then git -C "$R" archive "${BASELINE_REF:-github/task/HRD-022-clients-linux-web-android}" clients/desktop/main.js clients/desktop/preload.js clients/desktop/package.json clients/desktop/lib clients/desktop/renderer | tar -x -C "$ST/red_app"; fi
cli() { name=$1; shift; "$LAB" run "$ST" "hrdiso-h3-$name" -r "$R/clients/desktop:/work/desktop" -r "$R/qa/h3:/work/h3" -r "$ST/red_app/clients/desktop:/work/redapp" -r "$ST/lab/helix-http3-check:/work/h3c" -r "$HOME/.cache/ms-playwright:/ms-playwright" -w "$ST/out:/out" "$@"; }
scenario() {  # name path proto enc -- client-run-args...
  local name=$1 path=$2 proto=$3 enc=$4; shift 5
  local before; before=$("$LAB" logs "$ST" | wc -l)
  cli "$name" "$@" > "$OUT/$name.client.txt" 2>&1; local crc=$?
  "$LAB" logs "$ST" | tail -n +"$((before+1))" > "$OUT/$name.server.log"
  python3 -I "$R/qa/h3/check_h3.py" "$OUT/$name.server.log" "$path" "$proto" "$enc" > "$OUT/$name.verdict.json"; local v=$?
  echo "$name client_rc=$crc verdict_rc=$v $(cat "$OUT/$name.verdict.json")"; [ "$v" -eq 0 ] || rc=1
}
DESK_ENV=(-e HELIX_SERVER_URL=https://helix-server:8443)
scenario desktop-new /v1/formats HTTP/3.0 br -- "${DESK_ENV[@]}" -e HELIX_DESKTOP_QUIC_ORIGINS=helix-server:8443 -e "HELIX_DESKTOP_PIN_SPKI_SHA256=$SPKI" -- /usr/local/bin/gui-entrypoint.sh node /work/h3/desktop_h3.mjs $EL /work/desktop /out/desktop-new.json
# RED baseline: the shell of PR #18 has no transport module, so its main process uses Node fetch (trusting the lab CA via NODE_EXTRA_CA_CERTS so TLS is not the variable)
scenario desktop-baseline /v1/formats HTTP/1.1 any -- "${DESK_ENV[@]}" -e HELIX_DESKTOP_QUIC_ORIGINS=helix-server:8443 -e "HELIX_DESKTOP_PIN_SPKI_SHA256=$SPKI" -e NODE_EXTRA_CA_CERTS=/lab/ca.pem -- /usr/local/bin/gui-entrypoint.sh node /work/h3/desktop_h3.mjs $EL /work/redapp /out/desktop-baseline.json
scenario desktop-no-quic-origin /v1/formats HTTP/2.0 br -- "${DESK_ENV[@]}" -e "HELIX_DESKTOP_PIN_SPKI_SHA256=$SPKI" -- /usr/local/bin/gui-entrypoint.sh node /work/h3/desktop_h3.mjs $EL /work/desktop /out/desktop-noq.json
scenario chromium-web /v1/formats HTTP/3.0 br -- -- node /work/h3/probe_chromium.mjs $CHROME https://helix-server:8443/v1/formats /out/chromium-web.json "--ignore-certificate-errors-spki-list=$SPKI" --enable-quic --origin-to-force-quic-on=helix-server:8443
scenario chromium-web-no-force /v1/formats HTTP/2.0 br -- -- node /work/h3/probe_chromium.mjs $CHROME https://helix-server:8443/v1/formats /out/chromium-noforce.json "--ignore-certificate-errors-spki-list=$SPKI"
scenario go-quic-client /v1/formats HTTP/3.0 any -- -- /work/h3c -url https://helix-server:8443/v1/formats -ca /lab/ca.pem
cp "$ST/out/"*.json "$OUT/" 2>/dev/null
echo "overall rc=$rc"; exit $rc
