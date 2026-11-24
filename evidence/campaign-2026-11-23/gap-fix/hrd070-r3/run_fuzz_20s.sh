#!/usr/bin/env bash
# 20s fuzz of every HRD-070 target; paths are script-relative (override root with R=).
E=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd); R=${R:-$(cd "$E/../../../.." && pwd)}; rc=0; : >"$E/fuzz20_summary.txt"
run() { ( cd "$R/$1" && go test "$2" -run '^$' -fuzz "^$3\$" -fuzztime 20s -parallel 4 ) >"$E/fuzz20_$3.log" 2>&1; r=$?
  n=$(ls "$R/$1/${2#./}/testdata/fuzz/$3" 2>/dev/null | wc -l); echo "$3 exit=$r crasher_files=$n" | tee -a "$E/fuzz20_summary.txt"; [ $r = 0 ] || rc=1; }
for t in FuzzParseStatusDocument FuzzParseLicense FuzzCanonical; do run core ./pkg/lcpfuzz $t; done
run server ./internal/opds FuzzOPDSGeneration
run server ./internal/kosync FuzzPutProgressBody
run server ./internal/kosync FuzzRegisterInput
exit $rc
