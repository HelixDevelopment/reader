#!/usr/bin/env bash
# run_helixqa_container.sh <out-dir> - the HelixQA `http` banks executed INSIDE the container against a host helix-server
# (MODE=hq: the fixed throwaway credentials the banks hard-code). HELIXQA_BIN must be a static build
# (CGO_ENABLED=0 go build ./cmd/helixqa in submodules/helixqa).
set -u
R=$(cd "$(dirname "$0")/../.." && pwd); OUT=$1; SRV=${HELIX_SERVER_BIN:?}; HQA=${HELIXQA_BIN:?}; WORK=${ISO_WORK:?}; ST=$WORK/st-hq
"$R/tools/isolation/fresh_out.sh" "$OUT"; mkdir -p "$WORK/bin"; rm -rf "${ST:?}"; cp "$HQA" "$WORK/bin/helixqa"; python3 -I "$R/qa/helixqa/fixtures/make_fixtures.py" "$WORK/fx-hq" || exit 70
MODE=hq PROXY_POLICY=helixqa PROXY_REWRITE=hq-admin-token SOURCE_ID=hq-lib EXTERNAL_URL=http://127.0.0.1:28090 bash "$R/tools/isolation/start_server.sh" start "$SRV" "$ST" "$WORK/fx-hq" || exit 71
trap 'bash "$R/tools/isolation/start_server.sh" stop "$ST" || true' EXIT
"$R/tools/isolation/run_container.sh" -n hrdiso-hqa -p "$OUT/container_proof.json" -m 512m -c 1 -b "$ST/sock" -r "$WORK/bin/helixqa:/work/helixqa" -r "$R/qa/helixqa/banks:/work/banks" \
  -- /work/helixqa http --banks /work/banks --base-url http://127.0.0.1:38191 --verbose --json > "$OUT/helixqa_http.raw.txt" 2> "$OUT/helixqa_http.stderr.txt"
RC=$?
python3 -I - "$OUT/helixqa_http.raw.txt" "$OUT/helixqa_http" <<'PY'
import sys
raw = open(sys.argv[1], "rb").read().decode("utf-8", "replace")
i = raw.index("\n{\n") + 1
open(sys.argv[2] + ".txt", "w", encoding="utf-8").write(raw[:i])
open(sys.argv[2] + ".json", "w", encoding="utf-8").write(raw[i:])
PY
rm -f "$OUT/helixqa_http.raw.txt"; echo "helixqa exit=$RC" | tee "$OUT/helixqa_http.exit"
cp "$ST/access.log" "$OUT/server_access.log"; exit $RC
