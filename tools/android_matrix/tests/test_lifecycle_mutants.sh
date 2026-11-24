#!/usr/bin/env bash
# test_lifecycle_mutants.sh - HRD-040 (review of 2a0c4d8, finding R1, section 11.4.194(6)(d)): each mutant of run_matrix.sh below removes ONE lifetime
# control; test_lifecycle.sh (run against the mutant through HRD040_RUNNER) MUST FAIL for every one. Exit 0 only when all mutants are killed AND the
# unmutated runner passes (golden-FALSE guard). Prints a table: mutant, rc of the suite, the checks that failed.
# EQUIVALENT mutants, measured and NOT in the list: M10 (no `podman rm -f` at the end of an emulator group; the EXIT trap removes the container anyway) and
# M1b (no TERM trap: bash runs the EXIT trap on SIGTERM death too, rc stays 143; it SURVIVED this suite, and an equivalent mutant has no behaviour to detect).
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
