#!/usr/bin/env bash
# Needle gate for the content-delivery and KOSync load harnesses (ATM-027).
#
# Each mutation breaks one product property or one harness proof, and the run is
# only accepted if the harness fails *with the intended message*. A mutation that
# passes is a hole in the harness, not a clean test run, so this script exits
# non-zero in that case.
#
# Restores from a cp snapshot, never from git: the working tree carries
# uncommitted campaign work.
set -u

SERVER_DIR="${SERVER_DIR:-$HOME/Projects/helix_reader/server}"
ROUTER="$SERVER_DIR/internal/httpapi/router.go"
KORED="$SERVER_DIR/internal/kosync/redis.go"
SERVICE_GO="$SERVER_DIR/internal/kosync/service.go"
HARNESS="$SERVER_DIR/internal/httpapi/load_delivery_test.go"
EVIDENCE="${EVIDENCE_DIR:-$HOME/Projects/helix_reader/evidence/campaign-2026-11-20/delivery/needles}"
BACKUP="$(mktemp -d /tmp/needle-backup-XXXXXX)"
CONTAINER="${CONTAINER:-helixqual-pg}"
BIN=/tmp/httpapi_load_needle

mkdir -p "$EVIDENCE"
cp "$ROUTER" "$KORED" "$SERVICE_GO" "$HARNESS" "$BACKUP/"
restore() { cp "$BACKUP/router.go" "$ROUTER"; cp "$BACKUP/redis.go" "$KORED"; cp "$BACKUP/service.go" "$SERVICE_GO"; cp "$BACKUP/load_delivery_test.go" "$HARNESS"; }
trap restore EXIT

RESULTS="$EVIDENCE/needle_gate_results.json"
: > "$EVIDENCE/needle_log.txt"
echo '[' > "$RESULTS"
FIRST=1
FAILURES=0

# run_case <name> <file> <expected-message> <test>
run_case() {
  local name="$1" file="$2" expect="$3" test="$4"
  local out="$EVIDENCE/${name}.log" status="NEEDLE_DID_NOT_FIRE" rc=0
  if (cd "$SERVER_DIR" && go test -c -o "$BIN" ./internal/httpapi/) >"$out" 2>&1; then
    podman cp "$BIN" "$CONTAINER:/tmp/httpapi_load_needle" >>"$out" 2>&1
    podman exec "$CONTAINER" bash -c "cd /tmp && GIN_MODE=release HELIX_LOAD_RUN=1 \
      HELIX_MEASUREMENT_PATH=in_container_clean \
      HELIX_TEST_POSTGRES_DSN='postgres://helixqual@127.0.0.1:5432/helix_test?sslmode=disable' \
      HELIX_TEST_REDIS_ADDR='helixqual-redis:6379' \
      HELIX_DELIVERY_SIZES=262144 HELIX_DELIVERY_FILES=1 HELIX_DELIVERY_ROUNDS=2 \
      HELIX_REMOTE_SIZES=262144 HELIX_LOAD_LEVELS=1,4 \
      HELIX_KOSYNC_CONTENDERS=16 HELIX_LOAD_LEVELS=1,4 HELIX_DELIVERY_TIMEOUT_SECONDS=30 \
      timeout 600 /tmp/httpapi_load_needle -test.run $test -test.v" >>"$out" 2>&1
    rc=$?
    if grep -qF -- "$expect" "$out"; then
      if [ "$rc" -ne 0 ]; then status="FIRED"; else status="MUTATION_PASSED_TEST"; fi
    fi
    printf '%s rc=%d status=%s expected=%s\n' "$name" "$rc" "$status" "$expect" >> "$EVIDENCE/needle_log.txt"
    grep -m3 -F -- "$expect" "$out" >> "$EVIDENCE/needle_log.txt"
  else
    printf '%s status=BUILD_FAILED\n' "$name" >> "$EVIDENCE/needle_log.txt"
  fi
  [ "$FIRST" -eq 1 ] && FIRST=0 || sed -i '$ s/$/,/' "$RESULTS"
  printf '  {"case": "%s", "target": "%s", "expected_failure": %s, "status": "%s"}\n' \
    "$name" "$file" "$(python3 -c 'import json,sys; print(json.dumps(sys.argv[1]))' "$expect")" "$status" >> "$RESULTS"
  echo "$name: $status"
  [ "$status" = "FIRED" ] || FAILURES=$((FAILURES+1))
}

# 1. Range requests ignored: every client would pay for the whole object.
restore
python3 - "$ROUTER" <<'MUT'
import sys
p = sys.argv[1]; s = open(p).read()
old = '\tif serveRange {'
assert old in s
open(p, 'w').write(s.replace(old, '\tif false && serveRange {', 1))
MUT
run_case range_ignored "$ROUTER" "not 206" TestLiveLoadContentDelivery

# 2. Range window shifted by one byte: right length, wrong bytes.
restore
python3 - "$ROUTER" <<'MUT'
import sys
p = sys.argv[1]; s = open(p).read()
old = '\t\t\tif _, err := opened.Seek(start, io.SeekStart); err != nil {'
assert old in s
open(p, 'w').write(s.replace(old, '\t\t\tif _, err := opened.Seek(start+1, io.SeekStart); err != nil {', 1))
MUT
run_case wrong_window "$ROUTER" "the right length but the wrong window" TestLiveLoadContentDelivery

# 3. Remote sources start claiming validators they cannot back: docs/api.md says a
#    connector without live revalidation must send a full 200 with no ETag, no
#    Accept-Ranges and no Content-Length taken from the scanned size.
restore
python3 - "$ROUTER" <<'MUT'
import sys
p = sys.argv[1]; s = open(p).read()
old = '\tsize, sizeKnown, current := int64(-1), false, false'
assert old in s
open(p, 'w').write(s.replace(old, '\tsize, sizeKnown, current := f.Size, true, true', 1))
MUT
run_case remote_validator_lie "$ROUTER" "from a catalog size the source was not asked to confirm" TestLiveLoadRemoteSource

# 3. KOSync monotonic rule removed: a delayed update rewinds the reader.
restore
python3 - "$KORED" <<'MUT'
import sys
p = sys.argv[1]; s = open(p).read()
old = 'if existing.Percentage > progress.Percentage {'
assert old in s
open(p, 'w').write(s.replace(old, 'if false && existing.Percentage > progress.Percentage {', 1))
MUT
run_case progress_rewind_serial "$KORED" "rewound stored progress" TestLiveLoadKOSync

# 5. PutProgress answers with the caller's copy instead of the persisted winner:
#    a rejected stale write acknowledges a position the store never kept.
restore
python3 - "$SERVER_DIR/internal/kosync/service.go" <<'MUT'
import sys
p = sys.argv[1]; s = open(p).read()
old = '\treturn s.store.GetProgress(ctx, username, progress.Document)'
assert old in s
open(p, 'w').write(s.replace(old, '\treturn progress, nil', 1))
MUT
run_case progress_ack_lie "$SERVICE_GO" "acknowledged progress is not the persisted state" TestLiveLoadKOSync

# 6. Concurrency proof weakened: staggered starts must not be reported as parallel.
restore
python3 - "$HARNESS" <<'MUT'
import sys
p = sys.argv[1]; s = open(p).read()
old = '\t\t\t\t<-gate.start\n\t\t\t\tgate.enter()\n\t\t\t\trctx, cancel := context.WithTimeout(ctx, timeout)'
assert old in s
open(p, 'w').write(s.replace(old, '\t\t\t\t<-gate.start\n\t\t\t\ttime.Sleep(time.Duration(c) * 250 * time.Millisecond)\n\t\t\t\tgate.enter()\n\t\t\t\trctx, cancel := context.WithTimeout(ctx, timeout)', 1))
MUT
run_case serialised_concurrency "$HARNESS" "requests were open at the same instant" TestLiveLoadContentDelivery

restore
sed -i '$ s/$/,/' "$RESULTS"
if [ "$FAILURES" -eq 0 ]; then
  printf '  {"summary": "PASS", "cases": 6}\n]\n' >> "$RESULTS"
else
  printf '  {"summary": "FAIL", "cases_not_detected": %d}\n]\n' "$FAILURES" >> "$RESULTS"
fi
python3 -c "import json,sys; json.load(open(sys.argv[1]))" "$RESULTS" || echo "WARN: results JSON malformed"
echo "undetected mutations: $FAILURES"
if [ "$FAILURES" -eq 0 ]; then exit 0; fi
exit 1
