#!/usr/bin/env bash
# §1.1 detection-power gate for the sharded catalog identity lock.
#
# Every "the lock protects X" claim in server/internal/catalog/lock_scope_test.go
# is only worth as much as the failure it would have produced. This script
# temporarily removes one class of lock key from identityLockKeys, runs the test
# that is supposed to depend on that class, and requires it to FAIL. Afterwards it
# restores the file and proves the restore byte-for-byte.
#
# It edits exactly one file, keeps a durable copy beside this script, and restores
# on any exit path; if the shell is killed the copy is still on disk. Nothing is committed.
set -uo pipefail
cd "$(dirname "$0")/../../.." || exit 1
SRC=server/internal/catalog/postgres.go
EVID=evidence/campaign-2026-11-20/lock-scope
LOG="$EVID/mutation-gate.log"
DSN='postgres://helixqual@127.0.0.1:34320/helix_test?sslmode=disable'
mkdir -p "$EVID"
ORIG_SHA=$(sha256sum "$SRC" | awk '{print $1}')
BAK="$EVID/postgres.go.pre-mutation"
cp "$SRC" "$BAK"
restore() { cp "$BAK" "$SRC"; }
trap 'restore' EXIT INT TERM

{
  echo ""
  echo "=== catalog identity lock mutation gate, $(date -Is) ==="
  echo "source=$SRC sha256=$ORIG_SHA"
} >>"$LOG"

# mutate <label> <from> <to> <test>
mutate() {
  local label=$1 from=$2 to=$3 test=$4
  restore
  python3 - "$SRC" "$from" "$to" <<'PY'
import sys
path, old, new = sys.argv[1], sys.argv[2], sys.argv[3]
s = open(path).read()
if s.count(old) != 1:
    sys.exit("expected exactly one occurrence of %r, found %d" % (old, s.count(old)))
open(path, 'w').write(s.replace(old, new))
PY
  if [ $? -ne 0 ]; then
    echo "CASE $label: SKIPPED (mutation anchor not unique)" >>"$LOG"
    echo "CASE $label: SKIPPED"
    return 2
  fi
  local now_sha
  now_sha=$(sha256sum "$SRC" | awk '{print $1}')
  if [ "$now_sha" = "$ORIG_SHA" ]; then
    echo "CASE $label: SKIPPED (mutation did not change the file)" >>"$LOG"
    return 2
  fi
  local out rc
  out=$(cd server && HELIX_TEST_POSTGRES_DSN="$DSN" go test ./internal/catalog -run "$test" -count=1 -timeout 8m 2>&1)
  rc=$?
  {
    echo "## CASE $label"
    echo "mutation: $from  ->  $to"
    echo "test: $test"
    echo "mutated_sha256=$now_sha"
    echo "go_test_exit=$rc"
    echo "$out" | grep -E "^(--- )?(FAIL|ok|PASS)|lock_scope_test.go:" | head -20
  } >>"$LOG"
  if [ $rc -eq 0 ]; then
    echo "CASE $label: NEEDLE BLIND — $test passed with the lock key removed" | tee -a "$LOG"
    return 1
  fi
  echo "CASE $label: detected — $test failed as required (exit $rc)" | tee -a "$LOG"
  return 0
}

STATUS=0
mutate drop-work-key 'out = append(out, "w:"+w.ID)' '_ = w.ID' TestCatalogMergePreservesEveryWriterUnderIdentityLocks || STATUS=1
mutate drop-source-key 'keys["s:"+f.SourceID] = struct{}{}' '_ = f.SourceID' TestCatalogFileClaimByDistinctWorksKeepsOneRow || STATUS=1
mutate drop-edition-key 'keys["e:"+e.ID] = struct{}{}' '_ = e.ID' TestCatalogEditionOwnershipIsDecidedOnce || STATUS=1
# Removing the whole lock acquisition is the control the in-test needles already
# cover; doing it here too proves the gate itself is not keyed to one mechanism.
# Mutating the acquirer itself, not a call site, so every lock in the package
# goes away at once: merge, upsert and reconcile all ask it to hold keys.
mutate drop-all-locks 'func lockIdentities(ctx context.Context, tx *sql.Tx, keys []string) error {' 'func lockIdentities(ctx context.Context, tx *sql.Tx, keys []string) error {
	keys = nil' TestCatalogMergePreservesEveryWriterUnderIdentityLocks || STATUS=1

restore
FINAL_SHA=$(sha256sum "$SRC" | awk '{print $1}')
{
  echo "## restore"
  echo "original_sha256=$ORIG_SHA"
  echo "restored_sha256=$FINAL_SHA"
  echo "restored_identical=$([ "$FINAL_SHA" = "$ORIG_SHA" ] && echo yes || echo NO)"
} >>"$LOG"
if [ "$FINAL_SHA" != "$ORIG_SHA" ]; then
  echo "RESTORE FAILED — $SRC differs from its pre-run hash" | tee -a "$LOG"
  exit 3
fi

# Unmutated confirmation, so the log ends on the state the repository is in.
OUT=$(cd server && HELIX_TEST_POSTGRES_DSN="$DSN" go test ./internal/catalog -run 'TestCatalog(MergePreserves|MergeNeedle|FileClaim|ReconcileExcludes|EditionOwnership)' -count=1 -timeout 15m -v 2>&1)
RC=$?
echo "## restored suite" >>"$LOG"
echo "go_test_exit=$RC" >>"$LOG"
echo "$OUT" | grep -E "^(--- )?(FAIL|ok|PASS)|NEEDLE|MODE |VERDICT" >>"$LOG"
if [ $RC -ne 0 ]; then
  echo "RESTORED RUN FAILED (exit $RC)" | tee -a "$LOG"
  exit 4
fi
echo "gate_status=$STATUS restored_tests=pass sha256_verified=yes" | tee -a "$LOG"
exit $STATUS
