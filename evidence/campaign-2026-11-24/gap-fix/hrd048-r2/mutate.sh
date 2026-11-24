#!/usr/bin/env bash
# Paired mutants for the HRD-048 soak tests (round 2). Restores soak_test.go on
# any exit via trap; verifies restoration with git diff against a saved copy.
set -u
cd "$(dirname "$0")/../../../../server/internal/httpapi" || exit 2
F=soak_test.go
B=$(mktemp) ; cp "$F" "$B"
restore() { cp "$B" "$F"; }
trap 'restore; rm -f "$B"' EXIT
run() { # name pattern replacement testregex
  restore
  python3 - "$F" "$2" "$3" <<'P' || { echo "$1: MUTATION NOT APPLIED"; return; }
import sys
f,a,b=sys.argv[1:4]
s=open(f).read()
assert a in s, "pattern missing"
open(f,"w").write(s.replace(a,b,1))
P
  if go test -count=1 -run "$4" . >/tmp/mut.$$ 2>&1; then echo "$1: SURVIVED"; else echo "$1: KILLED"; fi
  rm -f /tmp/mut.$$
}
run M01 'get("/v1/works", ""' 'get("/v1/nonexistent-a", ""' 'TestSoakServerRequestMixDoesNotLeak'
run M02 'server.URL+path' 'server.URL+"/zz-no-such"+path' 'TestSoakServerRequestMixDoesNotLeak'
run M07 '(r2 > soakMinR2 || (r2 > soakStepR2 && res.SteadyGrowthB > soakStepGrowth))' 'true' 'TestSoakGate'
run M08 'skip := res.Samples * 3 / 10' 'skip := 0' 'TestSoakGate'
run M09 'res.SteadyGrowthB > soakGrowthLimit &&' '' 'TestSoakGate'
run M10 'res.SlopeBytesIter > soakSlopeLimit &&' '' 'TestSoakGate'
run M11 'res.GoroutineDelta > soakGoroutineSlop' 'res.GoroutineDelta > soakGoroutineSlop+1' 'TestSoakGate'
restore
cmp -s "$B" "$F" && echo "restored: yes" || echo "restored: NO"
