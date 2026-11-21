#!/usr/bin/env bash
# Authoritative vector-search measurement for ATM-027 (campaign 2026-11-20).
#
# Everything except the build happens INSIDE the helixqual-pg container against
# 127.0.0.1:5432, because rootless podman's pasta forwarder stalls multi-packet
# responses on the host path (docs/scalability-review.md). Payloads are labelled
# with the path they came from and a host-path run is never used for a claim.
#
#   ./run_vector.sh              one 50k-row x 384-dim run
#   ./run_vector.sh smoke        1k-row plumbing check, labelled non-authoritative
#   ./run_vector.sh high         200k rows, for the plan-crossover curve
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$HERE/../../.." && pwd)"
SERVER="$ROOT/server"
PG="${HELIX_QUAL_PG_CONTAINER:-helixqual-pg}"
HOST_DSN="${HELIX_TEST_POSTGRES_DSN:-postgres://helixqual@127.0.0.1:34320/helix_test?sslmode=disable}"
# The test runs inside the container, where PostgreSQL listens on its own default
# port; the 34320 mapping only exists on the host. Passing the host DSN in here made
# every connection refuse and the run still reported success, so the two are now
# separate variables instead of one that is wrong on one side.
IN_DSN="${HELIX_QUAL_IN_CONTAINER_DSN:-postgres://helixqual@127.0.0.1:5432/helix_test?sslmode=disable}"
LOG="$HERE/run.log"
RUN_TAG="$(date -u +%Y%m%dT%H%M%SZ)"

mode="${1:-main}"
case "$mode" in
  smoke) ROWS=1000;   DIMS=64;  QUERIES=20;  LEVELS=1,8;     LABEL="smoke_host_build_in_container" ;;
  high)  ROWS=200000; DIMS=384; QUERIES=100; LEVELS=1,8,32;  LABEL="in_container_clean" ;;
  main)  ROWS=50000;  DIMS=384; QUERIES=100; LEVELS=1,8,32;  LABEL="in_container_clean" ;;
  *) echo "usage: $0 [main|smoke|high]" >&2; exit 2 ;;
esac

log() { printf '[%s] %s\n' "$(date -u +%H:%M:%S)" "$*" | tee -a "$LOG"; }

podman inspect "$PG" >/dev/null 2>&1 || { echo "container $PG not running (see ../README.md topology)" >&2; exit 1; }

log "=== run_vector mode=$mode tag=$RUN_TAG rows=$ROWS dims=$DIMS levels=$LEVELS ==="
for f in internal/aimemory/postgres.go internal/aimemory/vector_load_test.go; do
  log "fingerprint $f sha256=$(sha256sum "$SERVER/$f" | cut -d' ' -f1)"
done

log "building test binary on the host"
( cd "$SERVER" && CGO_ENABLED=0 go test -c -o /tmp/helix-aimemory-load.test ./internal/aimemory )
podman cp /tmp/helix-aimemory-load.test "$PG":/tmp/aimem.test

# The bench schema is dropped by the test itself; drop it again defensively in
# case a previous run died between the fatal path and its cleanup.
podman exec "$PG" psql -U helixqual -d helix_test -q -c 'DROP SCHEMA IF EXISTS helix_bench CASCADE' >>"$LOG" 2>&1 || true

log "running TestLiveLoadVectorSearch inside $PG (this seeds $ROWS rows, expect minutes)"
set +e
podman exec -i \
  -e HELIX_TEST_POSTGRES_DSN="$IN_DSN" \
  -e HELIX_LOAD_RUN=1 \
  -e HELIX_MEASUREMENT_PATH="$LABEL" \
  -e HELIX_LOAD_OUTPUT=/tmp/vector-$RUN_TAG.jsonl \
  -e HELIX_VECTOR_ROWS="$ROWS" \
  -e HELIX_VECTOR_DIMS="$DIMS" \
  -e HELIX_VECTOR_QUERIES="$QUERIES" \
  -e HELIX_VECTOR_K="${HELIX_VECTOR_K:-10}" \
  -e HELIX_LOAD_LEVELS="$LEVELS" \
  -e HELIX_LOAD_TIMEOUT_SECONDS="${HELIX_LOAD_TIMEOUT_SECONDS:-1500}" \
  "$PG" bash -c "/tmp/aimem.test -test.run TestLiveLoadVectorSearch -test.v -test.count=1 -test.timeout 40m > /tmp/vector-$RUN_TAG.log 2>&1; rc=\$?; echo exit=\$rc; exit \$rc" | tee -a "$LOG"
rc="${PIPESTATUS[0]}"
set -e

podman cp "$PG:/tmp/vector-$RUN_TAG.jsonl" "$HERE/raw-$RUN_TAG.jsonl" >>"$LOG" 2>&1 || log "copy-back failed"
podman cp "$PG:/tmp/vector-$RUN_TAG.log" "$HERE/raw-$RUN_TAG.log" >>"$LOG" 2>&1 || log "log copy-back failed"

python3 - "$HERE" "$RUN_TAG" "$mode" "$rc" <<'PY' || rc=1
import json, sys
here, tag, mode, rc = sys.argv[1], sys.argv[2], sys.argv[3], int(sys.argv[4])
rows = []
try:
    with open(f"{here}/raw-{tag}.jsonl") as handle:
        for line in handle:
            if line.strip():
                row = json.loads(line)
                row["run_tag"] = tag
                row["run_mode"] = mode
                rows.append(row)
except FileNotFoundError:
    print(f"no raw jsonl for {tag}", file=sys.stderr)
with open(f"{here}/vector.jsonl", "a") as handle:
    for row in rows:
        handle.write(json.dumps(row) + "\n")
if not rows:
    # A run that emitted no payload is not a run that passed. Without this the
    # summary step happily reported "0 rows across 0 runs" with exit 0.
    print(f"FAIL: run {tag} emitted no LOADJSON payload (test exit={rc})", file=sys.stderr)
    sys.exit(1)
print(f"appended {len(rows)} rows to vector.jsonl (exit={rc})")
if rc != 0:
    print(f"run {tag} failed with exit {rc}; payloads kept for diagnosis", file=sys.stderr)
    sys.exit(rc)
PY

python3 "$HERE/summarize.py" || { log "summarize.py failed"; rc=1; }
log "run_vector finished rc=$rc tag=$RUN_TAG"
exit "$rc"
