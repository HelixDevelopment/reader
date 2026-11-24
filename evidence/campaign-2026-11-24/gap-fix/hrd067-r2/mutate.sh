#!/usr/bin/env bash
# HRD-067 round 2: production mutants of server/internal/httpapi/jobs.go that the
# hermetic slow-client test must kill. Run from the repo root of a clean worktree.
# The original jobs.go is restored by a trap, even when interrupted.
set -u
F=server/internal/httpapi/jobs.go
BAK=$(mktemp "${TMPDIR:-/tmp}/jobs.go.XXXXXX") || exit 2
cp "$F" "${BAK:?}" || exit 2
trap 'cp "${BAK:?}" "$F"; rm -f "${BAK:?}"' EXIT INT TERM
run() { # name sed-expr
  cp "${BAK:?}" "$F"
  sed -i -E "$2" "$F"
  if cmp -s "$F" "${BAK:?}"; then echo "$1: MUTATION-DID-NOT-APPLY"; return; fi
  out=$(cd server && go test -count=1 -run 'TestSlowClientJobSurfaceHermetic' ./internal/httpapi 2>&1); rc=$?
  if [ $rc -ne 0 ]; then echo "$1: KILLED"; echo "$out" | grep -E 'violated|FAIL' | head -3; else echo "$1: SURVIVED"; fi
}
run P1_drop_pending_bound 's/pending >= maxPendingScans \|\| len\(s.jobs\) >= maxRetainedScans/len(s.jobs) >= maxRetainedScans/'
run P2_drop_scan_semaphore 's/case s.sem <- struct\{\}\{\}:/default:/; s/defer func\(\) \{ <-s.sem \}\(\)/_ = 0/'
run P3_scan_ignores_deadline 's/return fn\(ctx\)/return fn(context.Background())/'
run P6_pending_bound_200 's/maxPendingScans    = 64/maxPendingScans    = 200/'
trap - EXIT; cp "${BAK:?}" "$F"; rm -f "${BAK:?}"; echo "restored: $(git diff --quiet -- $F && echo yes || echo NO)"
