#!/usr/bin/env bash
# ATM-027 load harness runner. Re-runnable: each step appends its payloads to
# a per-stem .jsonl (inside the measurement container, copied back after the
# step) and rebuilds results.json immediately, so no result is held only in a
# terminal buffer. Safe against concurrent production edits: source
# fingerprints are logged before and after each step.
set -uo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$HERE/../../.." && pwd)"
SERVER="$ROOT/server"
PG=helixqual-pg
RUNLOG="$HERE/run.log"
CONTAINER_DSN='postgres://helixqual@127.0.0.1:5432/helix_test?sslmode=disable'
HOST_DSN='postgres://helixqual@127.0.0.1:34320/helix_test?sslmode=disable'
log(){ printf '%s %s\n' "$(date -Is)" "$*" | tee -a "$RUNLOG"; }
fingerprint(){
  ( cd "$ROOT" && git rev-parse HEAD \
    && sha256sum server/internal/catalog/postgres.go server/internal/catalog/postgres_batch.go \
       server/internal/httpapi/jobs.go server/internal/httpapi/router.go \
       server/internal/ingest/service.go server/internal/catalog/lock_scope_test.go ) 2>&1
}

build(){
  log "build: go test -c (catalog + httpapi load binaries)"
  ( cd "$SERVER" && CGO_ENABLED=0 go test -c -o /tmp/helix-catalog-load.test ./internal/catalog \
      && CGO_ENABLED=0 go test -c -o /tmp/helix-httpapi-load.test ./internal/httpapi ) 2>&1 | tee -a "$RUNLOG"
  [ -x /tmp/helix-catalog-load.test ] && [ -x /tmp/helix-httpapi-load.test ]
}

# run_step <bin-in-container> <test-regex> <stem> <measurement-path> <cgroup:1|0> [ENV=VAL...]
run_step(){
  local bin=$1 test=$2 stem=$3 path=$4 cgroup=$5; shift 5
  local out="$HERE/$stem.jsonl" logf="$HERE/$stem.log"
  log "step $stem: copying $bin"
  podman cp "$bin" "$PG:/tmp/load.test" || { log "step $stem: cp FAILED"; return 1; }
  log "step $stem: running $test measurement_path=$path"
  podman exec -i \
    -e HELIX_TEST_POSTGRES_DSN="$CONTAINER_DSN" \
    -e HELIX_LOAD_RUN=1 \
    -e HELIX_MEASUREMENT_PATH="$path" \
    -e HELIX_LOAD_OUTPUT="/tmp/$stem.jsonl" \
    -e HELIX_LOAD_CGROUP="$cgroup" \
    "$@" \
    "$PG" bash -c "/tmp/load.test -test.run '$test' -test.v -test.count=1 -test.timeout 60m > /tmp/$stem.log 2>&1; ec=\$?; echo exit=\$ec; exit \$ec" 2>&1 | tee -a "$RUNLOG"
  local rc=${PIPESTATUS[0]}
  podman cp "$PG:/tmp/$stem.jsonl" "$out" 2>/dev/null || log "step $stem: no jsonl produced (rc=$rc)"
  podman cp "$PG:/tmp/$stem.log" "$logf" 2>/dev/null || log "step $stem: no log produced (rc=$rc)"
  podman exec "$PG" rm -f "/tmp/$stem.jsonl" "/tmp/$stem.log" /tmp/load.test 2>/dev/null
  python3 "$HERE/summarize.py" | tee -a "$RUNLOG"
  log "step $stem: rc=$rc artifacts=$out $logf"
  return $rc
}

# run_host_step <bin> <test-regex> <stem> [extra test flags] — degraded-path comparison
run_host_step(){
  local bin=$1 test=$2 stem=$3; shift 3
  local out="$HERE/$stem.jsonl" logf="$HERE/$stem.log"
  log "host step $stem: running $test through pasta (non-authoritative)"
  ( cd "$SERVER" && env HELIX_TEST_POSTGRES_DSN="$HOST_DSN" HELIX_LOAD_RUN=1 \
      HELIX_MEASUREMENT_PATH=host_through_pasta HELIX_LOAD_OUTPUT="$out" \
      "$bin" -test.run "$test" -test.v -test.count=1 -test.timeout 60m "$@" ) \
      > "$logf" 2>&1
  local rc=$?
  log "host step $stem: rc=$rc $(grep -c '^.*LOADJSON' "$logf" 2>/dev/null || echo 0) payload lines; transport labels: $(grep -o 'MEASUREMENT PATH DEGRADED[^"]*' "$logf" | head -2 | tr '\n' ';')"
  python3 "$HERE/summarize.py" | tee -a "$RUNLOG"
  return $rc
}

mode=${1:-all}
shift || true
log "=== run_load.sh mode=$mode ==="
FP_BEFORE=$(fingerprint); log "fingerprint before: $(echo "$FP_BEFORE" | tr '\n' ' ')"
build || { log "build failed"; exit 1; }
case "$mode" in
  contention) run_step /tmp/helix-catalog-load.test TestLiveLoadWriteContention catalog_contention in_container_clean 1 "$@" ;;
  contention-sharded) run_step /tmp/helix-catalog-load.test TestLiveLoadWriteContention catalog_contention_sharded in_container_clean 1 "$@" ;;
  lock-scope) run_step /tmp/helix-catalog-load.test TestCatalogIdentityLockScopeThroughput catalog_lock_scope in_container_clean 1 "$@" ;;
  ladder)     run_step /tmp/helix-catalog-load.test TestLiveLoadConcurrencyLadder catalog_ladder in_container_clean 1 "$@" ;;
  http)       run_step /tmp/helix-httpapi-load.test TestLiveLoadHTTPLadder http_ladder in_container_clean 1 "$@" ;;
  host-ladder) run_host_step /tmp/helix-catalog-load.test TestLiveLoadConcurrencyLadder host_catalog_ladder ;;
  all)
    run_step /tmp/helix-catalog-load.test TestLiveLoadWriteContention catalog_contention in_container_clean 1
    run_step /tmp/helix-catalog-load.test TestLiveLoadConcurrencyLadder catalog_ladder in_container_clean 1
    run_step /tmp/helix-httpapi-load.test TestLiveLoadHTTPLadder http_ladder in_container_clean 1
    ;;
  *) log "unknown mode $mode"; exit 2 ;;
esac
rc=$?
FP_AFTER=$(fingerprint)
if [ "$FP_BEFORE" != "$FP_AFTER" ]; then
  log "WARNING production fingerprint changed during the run (concurrent edit); before: $FP_BEFORE after: $FP_AFTER"
else
  log "fingerprint stable across run"
fi
log "=== run_load.sh mode=$mode done rc=$rc ==="
exit $rc
