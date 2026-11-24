#!/usr/bin/env bash
# run_koreader_gui.sh <out-dir>  - KOReader 2026.07-1 GUI interop (OPDS browse/download, PSE page stream, KOSync push/pull
# between two profiles) INSIDE the gui-test container against a real helix-server on host loopback.
# Host side: server + proxy-access-log + redis (no GUI). All GUI-driving/screenshot code is in qa/isolation/inside/.
set -eu
R=$(cd "$(dirname "$0")/../.." && pwd); OUT=$1; SRV=${HELIX_SERVER_BIN:?set HELIX_SERVER_BIN}; WORK=${ISO_WORK:?set ISO_WORK}
KO_LIB=${KO_INSTALL:-/usr/lib/koreader}
ST=$WORK/st-ko; PR=$WORK/ko-prof; N=hrdiso-ko
"$R/tools/isolation/fresh_out.sh" "$OUT"; mkdir -p "$WORK"; rm -rf "${ST:?}" "${PR:?}"; mkdir -p "$PR"
python3 -I "$R/qa/isolation/make_gui_fixtures.py" koreader "$WORK/fx-ko"
bash "$R/tools/isolation/start_server.sh" start "$SRV" "$ST" "$WORK/fx-ko"
RN=$(cat "$ST/redis.name")
. "$ST/env"
cleanup() { podman rm -f $N >/dev/null 2>&1 || true; bash "$R/tools/isolation/start_server.sh" stop "$ST" || true; }
trap cleanup EXIT
KOP=$(head -c9 /dev/urandom | base64 | tr -dc A-Za-z0-9); KOK=$(printf %s "$KOP" | md5sum | cut -d' ' -f1); KOU=ko$RANDOM
curl -fs --unix-socket "$ST/sock/h.sock" -XPOST http://localhost/users/create -H 'Content-Type: application/json' -H 'Accept: application/vnd.koreader.v1+json' -d "{\"username\":\"$KOU\",\"password\":\"$KOK\"}" >/dev/null
for P in A B; do D=$PR/$P/.config/koreader/settings; mkdir -p "$D" "$PR/$P/Books"
  printf 'return { ["settings"] = { ["username"]="%s", ["userkey"]="%s", ["custom_server"]="http://127.0.0.1:38191", ["auto_sync"]=true, ["sync_forward"]=2, ["sync_backward"]=2, ["checksum_method"]=0 } }\n' "$KOU" "$KOK" > "$D/kosync.lua"
  printf 'return { ["servers"] = { { ["title"]="Helix", ["url"]="http://127.0.0.1:38191/opds/1.2", ["username"]="%s", ["password"]="%s" } } }\n' "$OPDS_USER" "$OPDS_PASS" > "$D/opds.lua"
  chmod 600 "$D"/*.lua; done
"$R/tools/isolation/run_container.sh" -d -n $N -m 2g -b "$ST/sock" -p "$OUT/container_proof.json" -e GUI_WM=none -r "$KO_LIB:/opt/koreader" -r "$R/qa/isolation/inside:/work/inside" -w "$PR:/prof" -w "$OUT:/out" -- sleep 1800 >/dev/null
sleep 3
D() { podman exec $N sh /work/inside/ko_drive.sh "$@"; }
LOG=$OUT/steps.log; : > "$LOG"; sink() { grep -E "syncs/progress" "$ST/access.log" | cut -c12-110 | tee -a "$LOG" || true; }; say() { echo "$(date -u +%FT%TZ) $*" | tee -a "$LOG"; }
say "warm-up: first launch of each fresh profile shows one-time notices that swallow the first tap; dismiss and exit"
for P in A B; do D start $P; D tap 300 450 1; D tap 300 450 1; D shot warm_$P; D close >/dev/null; sleep 2; done
say "profile A: start, OPDS browse, download, read now"
D start A; D shot A01_filemanager; D opds_open; D shot A02_opds_all_publications; D opds_download_read; D shot A03_book_opened_page1
D pages 3; D shot A04_page4; say "A close: $(D close)"; sleep 3; sink
say "profile B: start, OPDS browse, download"
D start B; D opds_open; D shot B01_opds; D opds_download_read; D shot B02_book_opened_pulled
sleep 3; D shot B03_after_pull; say "B docsettings: $(D docpercent B)"; say "B turns 2 pages"; D pages 2; D shot B04_page6; say "B close: $(D close)"; sleep 3; sink
say "A re-open book: pull B's progress"
BOOK=$(ls "$PR/A/Books"/*.epub | head -1); D start A "/prof/A/Books/$(basename "$BOOK")"; D shot A05_reopened_pulled; say "A docsettings: $(D docpercent A)"; say "A close: $(D close)"; sleep 3; sink
say "PSE: profile B streams the CBZ"
D start B; D opds_open; D opds_comic_stream; D shot B05_pse_page1
say "pse page1 centre pixel: $(D pixel 300 400)"
D key Next 4; D shot B06_pse_page2; say "pse page2 centre pixel: $(D pixel 300 400)"
say "B close: $(D close)"
cp "$ST/access.log" "$OUT/server_access.log"; cp "$ST/access.log.bodies" "$OUT/kosync_bodies.jsonl" 2>/dev/null || true
for k in $(podman exec "$RN" redis-cli --scan --pattern 'helix:v1:kosync:*'); do t=$(podman exec "$RN" redis-cli type "$k"); echo "KEY $k ($t)"; [ "$t" = string ] && podman exec "$RN" redis-cli get "$k"; done > "$OUT/redis_sink.txt"
sed -i "s/$KOK/<redacted>/g" "$OUT/redis_sink.txt" "$OUT/server_access.log" 2>/dev/null || true
python3 -I "$R/qa/isolation/kosync_proof.py" "$OUT/kosync_bodies.jsonl" "$OUT/kosync_proof.json" | tee -a "$LOG" || say "KOSYNC PROOF FAILED"
for P in A B; do cp "$PR/$P/ko.log" "$OUT/ko_$P.log" 2>/dev/null || true; done
say done
