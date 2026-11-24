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
t F3_empty_reason_malformed       3 'ALLOWLIST-MALFORMED'             $FX/allow_empty_reason.txt $FX/good.log
t F3_item_suffix_malformed        3 'ALLOWLIST-MALFORMED'             $FX/allow_bad_item.txt $FX/good.log
t F3_item_prefix_malformed        3 'ALLOWLIST-MALFORMED'             $FX/allow_prefix_item.txt $FX/good.log
t F4_fail_line_fails              3 'FAILED-TEST: .*\|TestB'          $A $FX/fail_line.log
t F4_build_failed_fails           3 'BUILD-FAILED: .*/b'              $A $FX/build_failed.log
t F5_pkg_fail_no_results          3 'NO-RESULTS: .*/b'                $A $FX/pkg_fail_no_results.log
t F5_pkg_fail_with_only_passes   3 'PACKAGE-FAILED: .*/a'            $A $FX/pkg_fail_with_pass.log
t F5_timeout_fails                3 'TIMEOUT'                         $A $FX/timeout.log
t F5_panic_fails                  3 'PANIC'                           $A $FX/panic.log
t F8_unused_allow_reported        0 'UNUSED-ALLOW.*TestNever'         $FX/allow_unused.txt $FX/unlisted_skip.log
t I1_truncated_log_not_green      3 'INCOMPLETE'                      $A $FX/truncated.log
t I4_missing_allow_setup_error    2 'SETUP-ERROR'                     $FX/no_such_allow.txt $FX/good.log
t m2_allow_whitespace_comments_ok 0 'GATE GREEN'                      $FX/allow_whitespace.txt $FX/two_skips.log
t m2_pipe_in_reason_malformed     3 'ALLOWLIST-MALFORMED'             $FX/allow_pipe_reason.txt $FX/unlisted_skip.log
t B2_skip_A_pass_B_full_marker    0 'GATE GREEN'                      $FX/allow_empty.txt $FX/a_skip_only.log $FX/phaseb_marker_full.log
t B2_skip_A_pass_B_relpath_marker 3 'UNLISTED-SKIP: .*/a\|TestL'      $FX/allow_empty.txt $FX/a_skip_only.log $FX/phaseb_marker_relpath.log
t B2_skip_in_A_and_B_stays_red    3 'UNLISTED-SKIP: .*/a\|TestL'      $FX/allow_empty.txt $FX/phaseb_marker_skip.log
# phase-B list construction from a phase-A log: exact expected output (B1)
pb=$("$GATE" --out "$W/out" --allow "$A" --print-phase-b "$FX/phase_a.log" 2>&1); prc=$?
want=$'REGEX=TestL|TestLiveLoadRemoteSource\nINRE=TestL\nHRE=TestLiveLoadRemoteSource'
if [ "$prc" = 0 ] && [ "$pb" = "$want" ]; then pass=$((pass+1)); echo "ok   B1_phase_b_lists_exact (rc=0)"
else fail=$((fail+1)); echo "FAIL B1_phase_b_lists_exact: rc=$prc got:"; sed 's/^/     | /' <<<"$pb"; fi
"$GATE" --out "$W/out" --allow "$A" --print-phase-b "$FX/empty.log" >/dev/null 2>&1; prc=$?
if [ "$prc" = 2 ]; then pass=$((pass+1)); echo "ok   B1_phase_b_unusable_log_rc2 (rc=2)"; else fail=$((fail+1)); echo "FAIL B1_phase_b_unusable_log_rc2: rc=$prc"; fi
t B3_real_B2c_B3_B4_order_green   0 'GATE GREEN'                      $FX/allow_empty.txt $FX/b_real_order_phase_a.log $FX/b_real_order.log
t m1_ansi_skip_seen               3 'UNLISTED-SKIP: .*/a\|TestX'      $FX/allow_empty.txt $FX/ansi_skip.log
t m3_contradictory_allow_fails    3 'CONTRADICTORY-ALLOW'             $FX/allow_contradictory.txt $FX/good.log
t m3_duplicate_allow_warns        0 'DUPLICATE-ALLOW'                 $FX/allow_duplicate.txt $FX/good.log
# --- runner self-tests that need no containers ---
chk() { local name=$1; shift; if "$@"; then pass=$((pass+1)); echo "ok   $name"; else fail=$((fail+1)); echo "FAIL $name"; fi; }
SRC=$(dirname "$GATE")/run_gated_tests.sh
# R3: every `# PKG` marker write has a matching `# PKG-END` write, and there are at least 3 (B2c, B3, plus none missing)
nm=$(grep -c 'echo "# PKG \$IMP"' "$SRC"); ne=$(grep -c 'echo "# PKG-END \$IMP"' "$SRC")
chk R3_markers_written_and_closed bash -c "[ $nm -ge 2 ] && [ $nm -eq $ne ]"
# R5: the live path seals through seal_dir (self-contained), after the gate
chk R5_live_path_calls_seal_dir grep -q '^seal_dir "\$OUT" ||' "$SRC"
# R6: B3 workers write per-worker logs concatenated in fixed order
chk R6_b3_per_worker_logs grep -q '> "\$TMPD/worker\$i.log"' "$SRC"
chk R8_port_check_present grep -q 'SETUP-ERROR: could not allocate ports' "$SRC"
# static: every repo path the runner references is tracked in git (no dependence on untracked tools)
untracked=""
for pth in $(grep -v '^[[:space:]]*#' "$SRC" | grep -oE '(tools|tests|scripts|docs|config|server|core)/[A-Za-z0-9_./-]+' | sort -u); do
  case $pth in */) continue;; esac
  [ -n "$(git -C "$ROOT" ls-files -- "$pth" | head -1)" ] || untracked="$untracked $pth"
done
chk R1_runner_references_only_tracked_paths test -z "$untracked"
[ -z "$untracked" ] || echo "     untracked refs:$untracked"
# R2: the ERR trap really aborts a missing command with exit 2
out=$("$GATE" --selftest-missing-cmd 2>&1); rc=$?
chk R2_err_trap_aborts_missing_command bash -c "[ $rc -eq 2 ]"
# preflight: missing python3 => exit 2 (live mode); sandbox PATH with every tool except python3
SB=$W/bin; mkdir -p "$SB"
for tl in bash env git awk sort comm sed sha256sum find xargs go podman base64 tr head grep cat dirname date wc mktemp rm mkdir printf cut paste uniq seq cp tee basename stat; do p=$(command -v $tl) && ln -sf "$p" "$SB/$tl"; done
out=$(PATH=$SB "$GATE" --preflight 2>&1); rc=$?
chk R7_preflight_missing_python3_exit2 bash -c "[ $rc -eq 2 ] && grep -q \"'python3'\" <<<\"\$0\"" "$out"
ln -sf "$(command -v python3)" "$SB/python3"
out=$(PATH=$SB "$GATE" --preflight 2>&1); rc=$?
chk R7b_preflight_all_tools_ok bash -c "[ $rc -eq 0 ]"
# seal_dir: self-contained, excludes itself and hidden files, verifies, deterministic order
SD=$W/seal; mkdir -p "$SD/sub"; echo a > "$SD/b.txt"; echo c > "$SD/sub/a.txt"; echo h > "$SD/.hidden"; echo old > "$SD/SHA256SUMS"
"$GATE" --seal "$SD" >/dev/null 2>&1; src=$?
chk R5b_seal_dir_verifies bash -c "[ $src -eq 0 ] && cd $SD && sha256sum -c SHA256SUMS >/dev/null && [ \$(wc -l < SHA256SUMS) -eq 2 ] && ! grep -q 'SHA256SUMS\\|hidden' SHA256SUMS && [ \"\$(awk '{print \$2}' SHA256SUMS | paste -sd,)\" = 'b.txt,sub/a.txt' ]"
t m5_unclosed_marker_then_ok_incomplete 3 'INCOMPLETE'              $FX/allow_empty.txt $FX/marker_unclosed_then_ok.log
t m6_pkg_end_path_mismatch        3 'INCOMPLETE.*PKG-END-path-mismatch' $FX/allow_empty.txt $FX/marker_end_mismatch.log
t m5_marker_open_at_eof           3 'INCOMPLETE.*marker-open-at-EOF'  $FX/allow_empty.txt $FX/marker_open_at_eof.log
t m5_pkgline_inside_marker_then_end 3 'INCOMPLETE.*package-line-while-marker-open' $FX/allow_empty.txt $FX/marker_ok_then_end.log
t m5_nested_marker                3 'INCOMPLETE.*nested-or-unclosed'  $FX/allow_empty.txt $FX/marker_nested.log
t m6_stray_pkg_end                3 'INCOMPLETE.*PKG-END-without-open' $FX/allow_empty.txt $FX/marker_stray_end.log
# B1 class: logs larger than the pipe buffer must still get a real verdict (grep -q under pipefail SIGPIPEs the writer)
BIG=$W/big.log; BIGS=$W/big_skip.log
awk 'BEGIN{for(i=0;i<30000;i++) printf "=== RUN   TestN%d\n--- PASS: TestN%d (0.00s)\n",i,i; print "PASS"; printf "ok  \texample.com/m/big\t1s\n"}' > "$BIG"
{ cat "$BIG"; printf '=== RUN   TestZ\n--- SKIP: TestZ (0.00s)\nPASS\nok  \texample.com/m/big2\t1s\n'; } > "$BIGS"
chk B1_big_log_over_1MiB bash -c "[ \$(stat -c %s $BIG) -gt 1048576 ]"
t B1_big_log_green                0 'GATE GREEN'                      $FX/allow_empty.txt $BIG
t B1_big_log_unlisted_skip_rc3    3 'UNLISTED-SKIP: .*big2\|TestZ'    $FX/allow_empty.txt $BIGS
out=$("$GATE" --out "$W/out" --allow "$A" --print-phase-b "$BIGS" 2>&1); rc=$?
chk B1_big_log_phase_b_lists bash -c "[ $rc -eq 0 ] && grep -q '^REGEX=TestZ$' <<<\"\$0\"" "$out"
# seal_dir: option-like names, spaces, deterministic sorted order (many files so directory order is not sorted)
SD2=$W/seal2; mkdir -p "$SD2"; for n in -x -c --help 'a b' $(seq 30 -1 1 | sed 's/^/f/'); do echo "$n" > "$SD2/$n"; done
"$GATE" --seal "$SD2" >/dev/null 2>&1; src=$?
chk m1_seal_option_like_names bash -c "[ $src -eq 0 ] && cd $SD2 && sha256sum -c SHA256SUMS >/dev/null 2>&1 && [ \$(wc -l < SHA256SUMS) -eq 34 ] && grep -q ' --help\$' SHA256SUMS"
chk m3_seal_order_is_sorted bash -c "cd $SD2 && awk '{ \$1=\"\"; print substr(\$0,2) }' SHA256SUMS > $W/names.txt && LC_ALL=C sort $W/names.txt | cmp -s - $W/names.txt"
# m4: B3 worker failure propagates into RC
"$GATE" --selftest-wait >/dev/null 2>&1; wrc=$?
chk m4_worker_failure_sets_RC bash -c "[ $wrc -eq 7 ]"
# m8: a missing common tool (seq) is a setup error, exit 2
rm -f "$SB/seq"; out=$(PATH=$SB "$GATE" --preflight 2>&1); rc=$?
chk m8_missing_seq_exit2 bash -c "[ $rc -eq 2 ] && grep -q \"'seq'\" <<<\"\$0\"" "$out"
# round 5: panic/timeout alone (no FAIL line, no INCOMPLETE) must fail; allow-list matching is exact, not substring
t X1_marker_panic_no_fail_line    3 'PANIC: example.com/m/a'          $A $FX/marker_panic_nofail.log
t X2_marker_timeout_no_fail_line  3 'TIMEOUT: example.com/m/a'        $A $FX/marker_timeout_nofail.log
t X4_allow_superstring_not_a_match 3 'UNLISTED-SKIP: example.com/m/a\|TestX' $FX/allow_superstring.txt $FX/skip_x.log
t X5_item_empty_digits_malformed  3 'ALLOWLIST-MALFORMED'             $FX/allow_item_empty_digits.txt $FX/good.log
t X6_nbsp_only_reason_malformed   3 'ALLOWLIST-MALFORMED'             $FX/allow_nbsp_reason.txt $FX/good.log
t X7_space_separated_ok_not_a_package_line 3 'INCOMPLETE'             $A $FX/ok_space_sep.log
t X8_bare_pass_then_eof_incomplete 3 'INCOMPLETE'                     $A $FX/bare_pass_eof.log
t X9_empty_marker_pair_incomplete 3 'INCOMPLETE: example.com/m/a .empty-marker-pair' $A $FX/empty_marker_pair.log
# I4: side-effect cleanup only touches files that were clean before the run
SR=$W/serepo; mkdir -p "$SR/evidence/x"; ( cd "$SR" && git init -q . && git config user.email t@t && git config user.name t \
  && echo base > evidence/x/tracked_pre.txt && echo base > evidence/x/tracked_new.txt && echo base > evidence/x/staged_pre.txt \
  && git add -A && git commit -q -m init ) >/dev/null 2>&1
echo mine > "$SR/evidence/x/tracked_pre.txt"; echo mine > "$SR/evidence/x/untracked_pre.txt"
"$GATE" --selftest-side-effects "$SR" snap "$W/se_before.txt" >/dev/null 2>&1
echo run > "$SR/evidence/x/tracked_new.txt"; echo run > "$SR/evidence/x/untracked_new.txt"; echo more >> "$SR/evidence/x/tracked_pre.txt"
"$GATE" --selftest-side-effects "$SR" restore "$W/se_before.txt" >/dev/null 2>&1; serc=$?
chk I4_cleanup_rc0 bash -c "[ $serc -eq 0 ]"
chk I4_run_created_untracked_removed_and_copied bash -c "[ ! -e $SR/evidence/x/untracked_new.txt ] && [ -f $W/se_before.txt.d/evidence/x/untracked_new.txt ]"
chk I4_run_modified_clean_tracked_reverted bash -c "[ \$(cat $SR/evidence/x/tracked_new.txt) = base ] && [ -f $W/se_before.txt.d/evidence/x/tracked_new.txt ]"
chk I4_preexisting_changes_untouched bash -c "[ \$(wc -l < $SR/evidence/x/tracked_pre.txt) -eq 2 ] && [ -f $SR/evidence/x/untracked_pre.txt ] && [ ! -e $W/se_before.txt.d/evidence/x/untracked_pre.txt ]"
# R1b: every file the tests/harness use is tracked and not ignored (a clean checkout must be able to run them)
ign=""; for pth in $(git -C "$ROOT" ls-files --others --ignored --exclude-standard -- tests/live); do ign="$ign $pth"; done
for pth in $(grep -ohE 'tests/live/[A-Za-z0-9_./-]+' "$ROOT/tests/live/test_run_gated_tests.sh" "$ROOT/tests/live/mutation_gate_check.sh" | sort -u); do
  case $pth in */) continue;; esac
  [ -e "$ROOT/$pth" ] || continue
  [ -n "$(git -C "$ROOT" ls-files -- "$pth" | head -1)" ] || ign="$ign $pth"
done
for f in "$FX"/*; do [ -n "$(git -C "$ROOT" ls-files -- "${f#"$ROOT"/}" | head -1)" ] || ign="$ign ${f#"$ROOT"/}"; done
chk R1b_tests_and_fixtures_tracked_not_ignored test -z "$ign"
[ -z "$ign" ] || echo "     untracked/ignored:$ign"
echo "cases passed=$pass failed=$fail"; [ $fail -eq 0 ]
