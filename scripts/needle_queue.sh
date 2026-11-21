#!/usr/bin/env bash
# Needle gate for the durable-queue load harness (ATM-027, Phase 1).
#
# ATM-027's acceptance names "durable queue behavior", and the functional
# durability tests already exist; what this harness adds is the claim that the
# bounds hold *under a simultaneous burst*. A burst claim is worthless unless the
# bound it measures can be felt, so each bound is deleted in turn and the run
# must fail on the matching assertion.
#
# Case 6 mutates the harness rather than the product: it staggers submissions and
# must be rejected by the harness for proving nothing. Without it, a run where
# nothing was ever rejected because everyone arrived slowly would look like a
# passing bound check.
#
# Restores from a cp snapshot, never from git — the working tree carries
# uncommitted campaign work.
set -u

SERVER_DIR="${SERVER_DIR:-$HOME/Projects/helix_reader/server}"
JOBS="$SERVER_DIR/internal/httpapi/jobs.go"
JOURNAL="$SERVER_DIR/internal/httpapi/jobs_journal.go"
HARNESS="$SERVER_DIR/internal/httpapi/load_jobs_test.go"
EVIDENCE="${EVIDENCE_DIR:-$HOME/Projects/helix_reader/evidence/campaign-2026-11-20/queue/needles}"
BACKUP="$(mktemp -d /tmp/needle-queue-backup-XXXXXX)"
CONTAINER="${CONTAINER:-helixqual-pg}"
REDIS="${REDIS:-helixqual-redis:6379}"
BIN=/tmp/httpapi_queue_needle

mkdir -p "$EVIDENCE"
cp "$JOBS" "$BACKUP/jobs.go"
cp "$JOURNAL" "$BACKUP/jobs_journal.go"
cp "$HARNESS" "$BACKUP/load_jobs_test.go"
restore() { cp "$BACKUP/jobs.go" "$JOBS"; cp "$BACKUP/jobs_journal.go" "$JOURNAL"; cp "$BACKUP/load_jobs_test.go" "$HARNESS"; }
trap restore EXIT

RESULTS="$EVIDENCE/needle_gate_results.json"
: > "$EVIDENCE/needle_log.txt"
echo '[' > "$RESULTS"
FIRST=1
FAILURES=0

# run_case <name> <file> <expected-message> [bursts]
run_case() {
  local name="$1" file="$2" expect="$3" burstn="${4:-3}"
  local out="$EVIDENCE/${name}.log" status="NEEDLE_DID_NOT_FIRE" rc=0
  if (cd "$SERVER_DIR" && go test -c -o "$BIN" ./internal/httpapi/) >"$out" 2>&1; then
    podman cp "$BIN" "$CONTAINER:/tmp/httpapi_queue_needle" >>"$out" 2>&1
    podman exec "$CONTAINER" bash -c "cd /tmp && GIN_MODE=release HELIX_LOAD_RUN=1 \
      HELIX_MEASUREMENT_PATH=in_container_clean HELIX_TEST_REDIS_ADDR='$REDIS' \
      HELIX_JOBS_BURSTS=$burstn HELIX_JOBS_PER_BURST=96 HELIX_JOBS_SCAN_MS=20 \
      timeout 900 /tmp/httpapi_queue_needle -test.run TestLiveLoadJobQueueDurability -test.v" >>"$out" 2>&1
    rc=$?
    if grep -qF -- "$expect" "$out"; then
      if [ "$rc" -ne 0 ]; then status="FIRED"; else status="MUTATION_PASSED_TEST"; fi
    fi
    printf '%s rc=%d status=%s expected=%s\n' "$name" "$rc" "$status" "$expect" >> "$EVIDENCE/needle_log.txt"
    grep -m2 -F -- "$expect" "$out" >> "$EVIDENCE/needle_log.txt"
  else
    printf '%s status=BUILD_FAILED\n' "$name" >> "$EVIDENCE/needle_log.txt"
  fi
  [ "$FIRST" -eq 1 ] && FIRST=0 || sed -i '$ s/$/,/' "$RESULTS"
  printf '  {"case": "%s", "target": "%s", "expected_failure": %s, "status": "%s"}\n' \
    "$name" "$file" "$(python3 -c 'import json,sys; print(json.dumps(sys.argv[1]))' "$expect")" "$status" >> "$RESULTS"
  echo "$name: $status"
  [ "$status" = "FIRED" ] || FAILURES=$((FAILURES+1))
}

# 1. Admission bound removed: the queue accepts without counting.
restore
python3 - "$JOBS" <<'MUT'
import sys
p = sys.argv[1]; s = open(p).read()
old = '\tif pending >= maxPendingScans || len(s.jobs) >= maxRetainedScans {'
assert old in s, "admission anchor"
open(p, 'w').write(s.replace(old, '\tif false && (pending >= maxPendingScans || len(s.jobs) >= maxRetainedScans) {', 1))
MUT
run_case admission_unbounded "$JOBS" "admission is not bounded"

# 2. Worker semaphore bypassed: every accepted scan runs at once.
restore
python3 - "$JOBS" <<'MUT'
import sys
p = sys.argv[1]; s = open(p).read()
old = ('\tselect {\n\tcase s.sem <- struct{}{}:\n\t\tdefer func() { <-s.sem }()\n'
       '\tcase <-ctx.Done():\n\t\ts.finish(id, ingest.Report{}, ctx.Err(), true)\n\t\treturn\n\t}')
assert old in s, "semaphore anchor"
new = ('\tselect {\n\tcase <-ctx.Done():\n\t\ts.finish(id, ingest.Report{}, ctx.Err(), true)\n\t\treturn\n'
       '\tdefault:\n\t}')
open(p, 'w').write(s.replace(old, new, 1))
MUT
run_case workers_unbounded "$JOBS" "the worker semaphore is not holding"

# 3. Successful scans stop writing their terminal record: the mirror keeps
#    showing "running" for work that finished.
restore
python3 - "$JOBS" <<'MUT'
import sys
p = sys.argv[1]; s = open(p).read()
old = ('\tj.Report = cloneReport(&report)\n\tdone := cloneJob(j)\n\ts.mu.Unlock()\n\ts.journalSave(done)\n}')
assert old in s, "terminal write anchor"
new = '\tj.Report = cloneReport(&report)\n\ts.mu.Unlock()\n}'
open(p, 'w').write(s.replace(old, new, 1))
MUT
run_case terminal_write_lost "$JOBS" "journaled fewer than 3 times"

# 4. Restore stops reconciling in-flight records: a job whose worker is gone is
#    handed back as still running, forever.
restore
python3 - "$JOURNAL" <<'MUT'
import sys
p = sys.argv[1]; s = open(p).read()
old = '\t\tif !terminal(record.State) {'
assert old in s, "restore anchor"
open(p, 'w').write(s.replace(old, '\t\tif false && !terminal(record.State) {', 1))
MUT
run_case interrupted_not_reconciled "$JOURNAL" "restore reported"

# 5. Index trimming removed. Needs enough accepted jobs to overflow the cap, so
#    this case runs more bursts than the others. The harm is not the raw member
#    count: recent() reads at most maxRetainedScans entries, so an untrimmed set
#    leaves older records listed in the index but unreachable to any restore. That
#    is the assertion this mutation must trip, and it fires before the ZCard bound
#    check, so expecting the latter would be expecting the wrong message.
restore
python3 - "$JOURNAL" <<'MUT'
import sys
p = sys.argv[1]; s = open(p).read()
old = '\tif err := j.client.ZRemRangeByRank(ctx, scanKeyIndex, 0, -(maxRetainedScans + 1)).Err(); err != nil {'
assert old in s, "index bound anchor"
new = '\tif false {\n\t\tif err := j.client.ZRemRangeByRank(ctx, scanKeyIndex, 0, -(maxRetainedScans + 1)).Err(); err != nil {'
s = s.replace(old, new, 1)
# close the injected block after the original error branch
old2 = new + '\n\t\treturn err\n\t}\n\treturn nil\n}'
assert old2 in s, "index bound close"
s = s.replace(old2, new + '\n\t\treturn err\n\t\t}\n\t}\n\treturn nil\n}', 1)
open(p, 'w').write(s)
MUT
run_case index_unbounded "$JOURNAL" "indexed but not restored" 6

# 6. The harness's own power: stagger the submissions and the burst stops
#    contesting the bound. Must be refused, not reported as a clean run.
restore
python3 - "$HARNESS" <<'MUT'
import sys
p = sys.argv[1]; s = open(p).read()
old = '\t\t\t\t<-gate.start\n\t\t\t\tsourceID := "load-src-"'
assert old in s, "barrier anchor"
new = '\t\t\t\t<-gate.start\n\t\t\t\ttime.Sleep(time.Duration(i) * 25 * time.Millisecond)\n\t\t\t\tsourceID := "load-src-"'
open(p, 'w').write(s.replace(old, new, 1))
MUT
run_case serial_submissions_refused "$HARNESS" "nothing was rejected"

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
