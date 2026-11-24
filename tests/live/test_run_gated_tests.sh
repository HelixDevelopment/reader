#!/usr/bin/env bash
# test_run_gated_tests.sh - HRD-050: executable regression test of the skip gate (run_gated_tests.sh --check).
# Drives the REAL script on committed golden-good / golden-bad `go test -v` log fixtures (tests/live/fixtures);
# this tests the gate's own logic only - it is not evidence that any live test works (11.4.289(C)(2)).
# Env GATE=<path> runs the same cases against another copy of the gate (used by mutation_gate_check.sh).
# Exit 0 = all cases as expected, 1 = a case differed.
set -uo pipefail
ROOT=$(cd "$(dirname "$0")/../.." && pwd) || exit 2
GATE=${GATE:-$ROOT/tests/live/run_gated_tests.sh}; FX=${FX:-$ROOT/tests/live/fixtures}
W=$(mktemp -d); trap 'rm -rf "$W"' EXIT
pass=0; fail=0
# t NAME EXPECTED_RC EXPECTED_OUTPUT_REGEX ALLOWFILE LOG...
t() { local name=$1 erc=$2 ere=$3 allow=$4; shift 4; local args=() l out rc
  for l in "$@"; do args+=(--check "$l"); done
  out=$("$GATE" --out "$W/out" --allow "$allow" "${args[@]}" 2>&1); rc=$?
  if [ "$rc" = "$erc" ] && grep -qE -- "$ere" <<<"$out"; then pass=$((pass+1)); echo "ok   $name (rc=$rc)"
  else fail=$((fail+1)); echo "FAIL $name: want rc=$erc /$ere/ got rc=$rc"; sed 's/^/     | /' <<<"$out"; fi; }
A=$FX/allow_good.txt
t golden_good_green               0 'GATE GREEN'                      $FX/allow_empty.txt $FX/good.log
t skip_listed_green               0 'GATE GREEN'                      $A $FX/unlisted_skip.log
t unlisted_skip_fails             3 'UNLISTED-SKIP: .*a\|TestX'       $FX/allow_empty.txt $FX/unlisted_skip.log
t skip_then_pass_same_pkg_green   0 'GATE GREEN'                      $FX/allow_empty.txt $FX/skip_then_pass_same_pkg.log
t F2_same_name_other_pkg_not_masked 3 'UNLISTED-SKIP: .*/b\|TestOpen' $FX/allow_empty.txt $FX/samename_masked.log
t F2_allow_wrong_package_fails    3 'UNLISTED-SKIP'                   $FX/allow_wrong_pkg.txt $FX/unlisted_skip.log
t F2_old_3field_format_malformed   3 'ALLOWLIST-MALFORMED'            $FX/allow_old_format.txt $FX/good.log
t F1_missing_log_blind            2 'LOG-UNUSABLE.*missing'           $A $FX/does_not_exist.log
t F1_empty_log_blind              2 'LOG-UNUSABLE.*empty'             $A $FX/empty.log
t F1_no_test_lines_blind          2 'LOG-UNUSABLE.*no .--- PASS'      $A $FX/no_test_lines.log
t F1_one_bad_log_among_good       2 'LOG-UNUSABLE'                    $A $FX/good.log $FX/empty.log
t F3_tab_only_reason_malformed    3 'ALLOWLIST-MALFORMED'             $FX/allow_tab_reason.txt $FX/good.log
t F3_item_suffix_malformed        3 'ALLOWLIST-MALFORMED'             $FX/allow_bad_item.txt $FX/good.log
t F3_item_prefix_malformed        3 'ALLOWLIST-MALFORMED'             $FX/allow_prefix_item.txt $FX/good.log
t F4_fail_line_fails              3 'FAILED-TEST: .*\|TestB'          $A $FX/fail_line.log
t F4_build_failed_fails           3 'BUILD-FAILED: .*/b'              $A $FX/build_failed.log
t F5_pkg_fail_no_results          3 'NO-RESULTS: .*/b'                $A $FX/pkg_fail_no_results.log
t F5_pkg_fail_with_only_passes   3 'PACKAGE-FAILED: .*/a'            $A $FX/pkg_fail_with_pass.log
t F5_timeout_fails                3 'TIMEOUT'                         $A $FX/timeout.log
t F5_panic_fails                  3 'PANIC'                           $A $FX/panic.log
t F8_unused_allow_reported        0 'UNUSED-ALLOW.*TestNever'         $FX/allow_unused.txt $FX/unlisted_skip.log
echo "cases passed=$pass failed=$fail"; [ $fail -eq 0 ]
