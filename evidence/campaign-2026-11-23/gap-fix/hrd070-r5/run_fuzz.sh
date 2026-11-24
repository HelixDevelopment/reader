#!/usr/bin/env bash
# HRD-070 fuzz runner: every target for FUZZTIME (default 5m); writes logs, summary and fuzz_results.json
# next to this script. Paths are script-relative (override the repo root with R=).
E=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd); R=${R:-$(cd "$E/../../../.." && pwd)}; T=${FUZZTIME:-300s}; P=${FUZZ_OUT:-fuzz}; rc=0
: >"$E/${P}_summary.txt"; echo '[' >"$E/${P}_results.json"; first=1
run() { # dir pkg target
  ( cd "$R/$1" && go test "$2" -run '^$' -fuzz "^$3\$" -fuzztime "$T" -parallel 4 ) >"$E/${P}_$3.log" 2>&1; r=$?
  n=$(ls "$R/$1/${2#./}/testdata/fuzz/$3" 2>/dev/null | wc -l)
  last=$(grep -E '^fuzz: elapsed' "$E/${P}_$3.log" | tail -1)
  execs=$(echo "$last" | sed -n 's/.*execs: \([0-9]*\).*/\1/p'); inter=$(echo "$last" | sed -n 's/.*new interesting: \([0-9]*\).*/\1/p')
  echo "$3 exit=$r crasher_files=$n execs=${execs:-?} interesting=${inter:-?}" | tee -a "$E/${P}_summary.txt"
  [ $first = 1 ] || echo ',' >>"$E/${P}_results.json"; first=0
  printf '{"target":"%s","package":"%s/%s","fuzztime":"%s","exit":%s,"crasher_files":%s,"execs":%s,"new_interesting":%s}' "$3" "$1" "${2#./}" "$T" "$r" "$n" "${execs:-null}" "${inter:-null}" >>"$E/${P}_results.json"
  [ $r = 0 ] || rc=1; }
for t in FuzzParseStatusDocument FuzzParseLicense FuzzCanonical; do run core ./pkg/lcpfuzz $t; done
run server ./internal/opds FuzzOPDSGeneration
run server ./internal/kosync FuzzPutProgressBody
run server ./internal/kosync FuzzRegisterInput
echo ']' >>"$E/${P}_results.json"; python3 -c "import json,sys;json.load(open(sys.argv[1]))" "$E/${P}_results.json" || rc=1
exit $rc
