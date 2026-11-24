#!/usr/bin/env bash
# HRD-070 acceptance: 5 minutes per fuzz target, 4 workers, no crashers. Logs next to this script.
set -u; R=/home/milosvasic/Projects/helix_reader; E=$R/evidence/campaign-2026-11-23/gap-fix/hrd070; T=${FUZZTIME:-300s}; rc=0
run() { # dir pkg target
  local pk=${2#./} n
  ( cd "$R/$1" && go test "$2" -run '^$' -fuzz "^$3\$" -fuzztime "$T" -parallel 4 ) >"$E/fuzz_$3.log" 2>&1; r=$?
  n=$(ls "$R/$1/$pk/testdata/fuzz/$3" 2>/dev/null | wc -l)
  echo "$3 exit=$r crasher_files=$n" | tee -a "$E/fuzz_summary.txt"; [ $r = 0 ] || rc=1
}
: >"$E/fuzz_summary.txt"
run core ./pkg/lcpfuzz FuzzParseLicense
run core ./pkg/lcpfuzz FuzzCanonical
run server ./internal/opds FuzzOPDSGeneration
run server ./internal/kosync FuzzPutProgressBody
run server ./internal/kosync FuzzRegisterInput
exit $rc
