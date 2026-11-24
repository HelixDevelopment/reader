#!/usr/bin/env bash
# mutation_gate_check.sh - HRD-050: paired mutants of the skip gate. Each mutant weakens ONE check in a temp
# copy of tests/live/{run_gated_tests.sh,parse_gotest.awk}; test_run_gated_tests.sh (GATE=<copy>) must then
# FAIL (mutant KILLED). A mutant that survives means that check is not load-bearing. Exit 0 iff all killed.
set -uo pipefail
ROOT=$(cd "$(dirname "$0")/../.." && pwd) || exit 2
W=$(mktemp -d); trap 'rm -rf "$W"' EXIT
# name@file@sed expression (separator @; sed uses | internally)
M=(
"M1-F1-no-unusable-check@run_gated_tests.sh@s/\[ -n \"\$LOGBAD\" \] && return 2$/:/"
"M2-F2-bare-name-key@parse_gotest.awk@s/print bk\\[i\\] \"\\\\t\" p \"\\\\t\" bn\\[i\\]/print bk[i] \"\\\\tX\\\\t\" bn[i]/"
"M3-F3-space-only-reason@run_gated_tests.sh@s/! \\[\\[ \\\$r =~ \\[\\^\\[:space:\\]\\] \\]\\]/[ -z \"\${r\\/\\/ }\" ]/"
"M4-F3-unanchored-item@run_gated_tests.sh@s/\[0-9\]+\[\[:space:\]\]\*\$ \]\]/[0-9]+ ]]/"
"M5-F4-ignore-FAIL-tests@run_gated_tests.sh@s/FAIL) echo \"FAILED-TEST: \\\$pk|\\\$n\"; bad=1;;/FAIL) :;;/"
"M6-F4-ignore-build-failed@parse_gotest.awk@s/print \"BUILDFAIL/print \"IGNORED/"
"M7-F5-ignore-timeout@parse_gotest.awk@s/print \"TIMEOUT/print \"IGNORED/"
"M8-F5-ignore-panic@parse_gotest.awk@s/print \"PANIC/print \"IGNORED/"
"M9-F5-ignore-no-results@run_gated_tests.sh@s/NORESULTS) echo \"NO-RESULTS: \\\$pk (package FAIL with no test results)\"; bad=1;;/NORESULTS) :;;/"
"M10-F5-ignore-pkgfail@run_gated_tests.sh@s/PKGFAIL) echo \"PACKAGE-FAILED: \\\$pk\"; bad=1;;/PKGFAIL) :;;/"
"M11-F8-no-unused-report@run_gated_tests.sh@s/echo \"UNUSED-ALLOW/echo \"IGNORED/"
)
rc=0
for m in "${M[@]}"; do
  IFS='@' read -r id file sedx <<<"$m"
  rm -rf "$W/c"; mkdir -p "$W/c"; cp "$ROOT"/tests/live/run_gated_tests.sh "$ROOT"/tests/live/parse_gotest.awk "$W/c/"
  # the gate resolves ROOT as ../.. of itself: mirror the layout
  mkdir -p "$W/c/tests/live"; mv "$W/c/run_gated_tests.sh" "$W/c/parse_gotest.awk" "$W/c/tests/live/"
  sed -i "$sedx" "$W/c/tests/live/$file"
  if cmp -s "$W/c/tests/live/$file" "$ROOT/tests/live/$file"; then echo "$id: MUTATION-NOT-APPLIED (sed matched nothing)"; rc=1; continue; fi
  if GATE="$W/c/tests/live/run_gated_tests.sh" bash "$ROOT/tests/live/test_run_gated_tests.sh" >"$W/o.txt" 2>&1; then
    echo "$id: SURVIVED (check not load-bearing)"; rc=1
  else echo "$id: KILLED ($(grep -c '^FAIL ' "$W/o.txt") case(s) failed)"; fi
done
exit $rc
