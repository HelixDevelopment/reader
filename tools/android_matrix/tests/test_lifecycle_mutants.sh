#!/usr/bin/env bash
# test_lifecycle_mutants.sh - HRD-040 (review of 2a0c4d8, finding R1, section 11.4.194(6)(d)): each mutant of run_matrix.sh below removes ONE lifetime
# control; test_lifecycle.sh (run against the mutant through HRD040_RUNNER) MUST FAIL for every one. Exit 0 only when all mutants are killed AND the
# unmutated runner passes (golden-FALSE guard). Prints a table: mutant, rc of the suite, the checks that failed.
# Reviewer mutants of rounds 2-3 (N7, M10, N13, N14, N16, N19, N20) and the author's own O-series (round 4, one per runner function / branch) are all in the table.
# EQUIVALENT mutants, measured and NOT in the list (a mutant with no observable behaviour change has nothing to detect):
#   M1b / N8 (no TERM trap) and N9 (no INT trap): bash runs the EXIT trap when it dies of an uncaught fatal SIGTERM/SIGINT, so cleanup still runs (rc 143/130 by the shell); probeSig/ of review cde66642.
#   N12 (mine() overwrites instead of appending): at most ONE container of a run is alive at any time (each stage removes its container itself, each emulator group removes its own - M10 - and fresh() runs before mine()), so the array never holds two live names.
#   N4 / N22 (lease-ts frozen at start, or now-250): reap.sh uses lease-ts only as a start-up grace (REAP_GRACE 120) and judges ownership by the owner-pid; the test accepts any epoch within 300 s, which both satisfy.
set -u
HERE=$(cd "$(dirname "$0")" && pwd); REAL=$(cd "$HERE/.." && pwd); T=$(mktemp -d "${TMPDIR:-/tmp}/hrd040-mut.XXXXXX"); : "${T:?}"; trap 'rm -rf "${T:?}"' EXIT
mutate() { # $1=name $2=python replace expression: old|||new (exact, must match exactly once)
  python3 -I - "$REAL/run_matrix.sh" "$T/$1.sh" "$REAL" "$2" <<'PY'
import sys
src, dst, real, spec = sys.argv[1:5]
old, new = spec.split("|||")
s = open(src).read()
if s.count(old) != 1: sys.exit("mutation does not apply exactly once: %r (%d)" % (old, s.count(old)))
s = s.replace(old, new)
# the copy lives outside the tree: pin HERE/REPO to the real tree (plumbing only, not part of any mutant)
first, rest = s.split("\n", 1)
rest = rest.replace('HERE=$(cd "$(dirname "$0")" && pwd); REPO=$(cd "$HERE/../.." && pwd)', 'HERE=%s; REPO=$(cd "$HERE/../.." && pwd)' % real, 1)
open(dst, "w").write(first + "\n" + rest)
PY
}
declare -A M
M[M1_cleanup_noop]='cleanup() { local c; for c in "${MINE[@]:-}"; do [ -n "$c" ] && podman rm -f "$c" >/dev/null 2>&1 || true; done; }|||cleanup() { :; }'
M[M2_build_ctr_no_lifeargs]='podman create --name "$n" $(LIFEARGS) ${NET---network none}|||podman create --name "$n" ${NET---network none}'
M[M3_prepare_no_lifeargs]='podman create --name "$n" $(LIFEARGS) --userns=keep-id --cap-drop ALL --security-opt no-new-privileges --memory 3g|||podman create --name "$n" --userns=keep-id --cap-drop ALL --security-opt no-new-privileges --memory 3g'
M[M3b_emulator_no_lifeargs]='podman run -d --name "$n" $(LIFEARGS) --memory 5g|||podman run -d --name "$n" --memory 5g'
M[M5_fresh_noop]='if podman container exists "$1" 2>/dev/null; then log "container $1 already exists; refusing to reuse it"; exit 1; fi; }|||:; }'
M[M6_life_default_zero]='LIFE=${HRD040_LIFE:-7200}|||LIFE=${HRD040_LIFE:-0}'
M[M6b_life_unvalidated]='case $LIFE in '"''"'|*[!0-9]*|0|00*)|||case $LIFE in zzz)'
M[M7_owner_pid_1]='org.helix.owner-pid=$$"|||org.helix.owner-pid=1"'
M[M7b_lease_ts_stale]='org.helix.lease-ts=$(date +%s)"|||org.helix.lease-ts=1000"'
M[M7c_timeout_arg_dropped_value]='--timeout "$LIFE"|||--timeout 3'
M[M11_fixed_name]='cname() { echo "hrdiso-android-$1-$RUNID"; }|||cname() { echo "hrdiso-android-$1-fixed"; }'
M[M12_fresh_continues]='refusing to reuse it"; exit 1; fi; }|||refusing to reuse it"; fi; }'
M[M13_runid_ignored]='RUNID=${HRD040_RUNID:-$$-$(date +%s)}|||RUNID=$$-$(date +%s)'
M[M14_no_uninstall]="adb \"\$n\" shell pm uninstall org.helixreader.app.test >/dev/null 2>&1 || true; adb \"\$n\" shell pm uninstall org.helixreader.app >/dev/null 2>&1 || true|||:"
M[M15_no_reap]='case ${1:-} in images|seed|prepare|warm|unit|mutation|apk|emulators|seal|all) reap;; esac|||:'
M[M16_no_fresh_apk_copy]='podman cp "$WORK/apk/app-debug.apk" "$n:/tmp/app.apk"; |||'
M[M17_trap_exit_removed]='trap cleanup EXIT; |||'
M[N7_fresh_prefix_only]='if podman container exists "$1" 2>/dev/null; then log|||if podman container exists "${1%-*}" 2>/dev/null; then log'
M[M10_no_group_rm]=$'    podman rm -f "$n" >/dev/null 2>&1 || true\n  done; }|||  done; }'
M[N13_runid_unvalidated]="case \$RUNID in ''|*[!A-Za-z0-9-]*)|||case \$RUNID in zzz)"
M[N14_emulator_has_network]='--cap-drop ALL --security-opt no-new-privileges --network none "$img"|||--cap-drop ALL --security-opt no-new-privileges "$img"'
M[N16_source_rev_at_apk_time]='cp "$WORK/prepared/.source_rev" "$d/source_rev.txt"|||git -C "$REPO" rev-parse HEAD > "$d/source_rev.txt"'
M[N20_apk_no_source_rev_guard]='[ -s "$WORK/prepared/.source_rev" ] || { log|||: || { log'
M[N19_reap_failure_ignored]='"$REAP" >&2 || { log "reap.sh failed (rc $?): $REAP"; exit 1; }; }|||"$REAP" >&2 || true; }'
# --- round 4: the author's own mutants, across every runner function and branch
M[O01_wait_boot_inverted]='[ "$r" = 1 ] && return 0|||[ "$r" != 1 ] && return 0'
M[O02_kvm_and_not_or]='if [ ! -r "$KVM" ] || [ ! -w "$KVM" ]; then|||if [ ! -r "$KVM" ] && [ ! -w "$KVM" ]; then'
M[O03_dead_group_not_blocked]='dead=1; continue; fi|||continue; fi'
M[O04_image_check_inverted]='if ! podman image exists "$img"; then|||if podman image exists "$img"; then'
M[O05_pkg_hash_empty_path]='[ -z "$path" ] || sha=|||[ -n "$path" ] || sha='
M[O06_wm_size_swapped]='"wm size ${w}x${h}; wm density $dpi"|||"wm size ${h}x${w}; wm density $dpi"'
M[O07_install_flags]='install -r -t /tmp/app.apk|||install -r /tmp/app.apk'
M[O08_test_apk_wrong_source]='podman cp "$WORK/apk/app-debug-androidTest.apk" "$n:/tmp/test.apk"|||podman cp "$WORK/apk/app-debug.apk" "$n:/tmp/test.apk"'
M[O09_instrument_cr_kept]='tr -d '"'"'\r'"'"' < "$rd/instrument.raw" > "$rd/instrument.txt"|||cp "$rd/instrument.raw" "$rd/instrument.txt"'
M[O10_instrument_raw_kept]='rm "$rd/instrument.raw"|||:'
M[O11_oom_score_dropped]='--oom-score-adj 500|||--oom-score-adj 0'
M[O12_swap_unbounded]='--memory-swap 5g|||--memory-swap 50g'
M[O13_build_mem_raised]='--memory 6g|||--memory 60g'
M[O14_unit_exit_hardcoded]='echo "gradle exit $rc" > "$d/exit.txt"; junit_out "$d"; return $rc; }|||echo "gradle exit 0" > "$d/exit.txt"; junit_out "$d"; return $rc; }'
M[O15_mutant_not_restored]='mv "$f.orig" "$f"; junit_out "$d"|||rm "$f.orig"; junit_out "$d"'
M[O16_mutation_verdict_inverted]='[ "$rc" -ne 0 ]; }|||[ "$rc" -eq 0 ]; }'
M[O17_mutation_noapply_unchecked]='cmp -s "$f" "$f.orig" && { log "mutation did not apply"; mv "$f.orig" "$f"; return 1; }|||:'
M[O18_mutant_is_not_post]='"OPTIONS", "POST")/|||"OPTIONS", "PUT")/'
M[O19_rev_pinned_to_head]='${HRD040_REV:-HEAD}|||HEAD'
M[O20_prepare_no_fresh]='n=$(cname prep); fresh "$n"; mine "$n"|||n=$(cname prep); mine "$n"'
M[O21_build_ctr_no_fresh]='local img=$1 n=$2 cmd=$3; fresh "$n"; mine "$n"|||local img=$1 n=$2 cmd=$3; mine "$n"'
M[O22_emulator_not_registered]='fresh "$n"; mine "$n"; log "boot $n|||fresh "$n"; log "boot $n'
M[O23_rows_filter_ignored]='[ "${1:-}" = --rows ] && only=$2|||only='
M[O24_matrix_not_copied]='cp "$MATRIX" "$EVID/matrix.tsv.txt"|||:'
M[O25_geometry_not_reset]='am force-stop org.helixreader.app; wm size reset; wm density reset|||am force-stop org.helixreader.app'
M[O26_instr_runner_wrong]='org.helixreader.app.test/androidx.test.runner.AndroidJUnitRunner|||org.helixreader.app.test/androidx.test.runner.Other'
M[O27_prepare_net_none]='podman create --name "$n" $(LIFEARGS) --userns=keep-id --cap-drop ALL --security-opt no-new-privileges --memory 3g|||podman create --name "$n" $(LIFEARGS) --network none --userns=keep-id --cap-drop ALL --security-opt no-new-privileges --memory 3g'
M[O28_build_net_open]='${NET---network none}|||'
M[O29_host_bind_mount]='-v "$VOL:/cache:rw"|||-v "$VOL:/cache:rw" -v /:/host:rw'
M[O30_unknown_stage_ok]='*) sed -n 2,4p "$0"; exit 2;;|||*) sed -n 2,4p "$0"; exit 0;;'
M[O31_cleanup_order_noop_on_empty_guard]='[ -n "$c" ] && podman rm -f "$c" >/dev/null 2>&1 || true|||[ -n "$c" ] || true'
bad=0; printf '%-32s %-5s %s\n' MUTANT SUITE-RC "FAILED CHECKS"
# golden-FALSE: the real runner passes
HRD040_RUNNER="$REAL/run_matrix.sh" bash "$HERE/test_lifecycle.sh" > "$T/base.out" 2>&1; rcb=$?; printf '%-32s %-5s %s\n' "(unmutated runner)" "$rcb" "$(grep -c '^PASS' "$T/base.out") PASS"; [ "$rcb" = 0 ] || bad=1
for k in $(printf '%s\n' "${!M[@]}" | sort); do
  if ! err=$(mutate "$k" "${M[$k]}" 2>&1); then printf '%-32s %-5s %s\n' "$k" "n/a" "MUTATION NOT APPLIED: $err"; bad=1; continue; fi
  HRD040_RUNNER="$T/$k.sh" bash "${HRD_LC_TEST:-$HERE/test_lifecycle.sh}" > "$T/$k.out" 2>&1; rc=$?
  printf '%-32s %-5s %s\n' "$k" "$rc" "$(grep '^FAIL' "$T/$k.out" | awk '{print $2}' | sort -u | tr '\n' ' ')"
  [ "$rc" != 0 ] || { bad=1; echo "  SURVIVED: $k"; }
done
[ $bad = 0 ] && { echo "ALL MUTANTS KILLED"; exit 0; } || { echo "SOME MUTANTS SURVIVED OR DID NOT APPLY"; exit 1; }
