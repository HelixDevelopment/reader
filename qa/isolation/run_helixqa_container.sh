#!/usr/bin/env bash
# run_helixqa_container.sh <out-dir> - the HelixQA `http` banks executed INSIDE the container against a host helix-server
# (MODE=hq: the fixed throwaway credentials the banks hard-code). HELIXQA_BIN must be a static build
# (CGO_ENABLED=0 go build ./cmd/helixqa in submodules/helixqa).
set -eu -o pipefail
R=$(cd "$(dirname "$0")/../.." && pwd); OUT=$1; SRV=${HELIX_SERVER_BIN:?}; HQA=${HELIXQA_BIN:?}; WORK=${ISO_WORK:?}; ST=$WORK/st-hq
"$R/tools/isolation/fresh_out.sh" "$OUT"; mkdir -p "$WORK/bin"; rm -rf "${ST:?}"; cp "$HQA" "$WORK/bin/helixqa"; python3 -I "$R/qa/helixqa/fixtures/make_fixtures.py" "$WORK/fx-hq" || exit 70
MODE=hq PROXY_POLICY=helixqa PROXY_REWRITE=hq-admin-token SOURCE_ID=hq-lib EXTERNAL_URL=http://127.0.0.1:28090 bash "$R/tools/isolation/start_server.sh" start "$SRV" "$ST" "$WORK/fx-hq" || exit 71
trap 'bash "$R/tools/isolation/start_server.sh" stop "$ST" || true' EXIT
# R5-B1: this runner PARSES the container's output, so it uses the wrapper's RAW-capture mode (-o): stdout and stderr unmerged and uncapped in a host-only 0700 dir; the
# terminal only ever gets a sanitized copy. EXIT STATUS (round 6, m5): when the container never starts (wrapper exit 64 usage, 65 isolation violation, 66 image missing, 67 no terminal and no -o)
# there are no raw files, the publisher fails closed, nothing is published and THIS runner exits 1 with the wrapper's status in its message - never the wrapper's own code and never 0. The raw files are KEPT (post-mortem); evidence is published only by helixqa_publish.py, which fails closed.
RAW=$WORK/hq-raw; rm -rf "${RAW:?}"; mkdir -m 700 "$RAW"; RC=0
"$R/tools/isolation/run_container.sh" -n hrdiso-hqa -p "$OUT/container_proof.json" -o "$RAW" -m 512m -c 1 -b "$ST/sock" -r "$WORK/bin/helixqa:/work/helixqa" -r "$R/qa/helixqa/banks:/work/banks" \
  -- /work/helixqa http --banks /work/banks --base-url http://127.0.0.1:38191 --verbose --json || RC=$?
python3 -I "$R/qa/isolation/helixqa_publish.py" "$RAW/stdout.raw" "$RAW/stderr.raw" "$OUT" || { echo "run_helixqa_container: could not turn the HelixQA output into evidence (container/wrapper exit $RC); nothing published, raw output kept in $RAW" >&2; exit 1; }
echo "helixqa exit=$RC" | tee "$OUT/helixqa_http.exit.txt"
cp "$ST/access.log" "$OUT/server_access.log"; exit $RC
