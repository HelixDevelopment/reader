#!/usr/bin/env bash
# test_lifecycle.sh - HRD-040 F1: run_matrix.sh must meet docs/test-isolation.md lifetime rules. Runs the REAL run_matrix.sh `emulators` stage against a
# logging fake `podman` (no containers are started) and asserts: unique per-run names, --timeout + the three org.helix.* labels on every `podman run`,
# reap.sh before the first container, a pre-existing same-name (leftover) container is NEVER reused/touched, APKs are always copied fresh,
# the installed package hash is read back per row and equals the built APK, containers are removed on exit (also when boot fails).
# Mutants (each must be detected): FAKE_STALE_INSTALL=1 makes the device report a hash that differs from the built APK.
set -u
HERE=$(cd "$(dirname "$0")" && pwd); REPO=$(cd "$HERE/../../.." && pwd)
T=$(mktemp -d "${TMPDIR:-/tmp}/hrd040-lc.XXXXXX"); : "${T:?}"; trap 'rm -rf "${T:?}"' EXIT
fail=0; ok() { echo "PASS $1"; }; ko() { echo "FAIL $1"; fail=1; }
mkdir -p "$T/bin" "$T/ctr" "$T/work/apk" "$T/hostcache"
echo apk-app-FRESH > "$T/work/apk/app-debug.apk"; echo apk-test-FRESH > "$T/work/apk/app-debug-androidTest.apk"
( cd "$T/work/apk" && sha256sum app-debug.apk app-debug-androidTest.apk > apk.sha256.txt ); echo 0123456789abcdef0123456789abcdef01234567 > "$T/work/apk/source_rev.txt"
cat > "$T/bin/podman" <<'FAKE'
#!/usr/bin/env bash
echo "$*" >> "$FAKE_LOG"; D=$FAKE_DIR/ctr
sh1() { sha256sum "$1" | cut -d' ' -f1; }
case $1 in
  image) case $2 in exists) exit 0;; inspect) echo "sha256:fakeid sha256:fakedigest"; exit 0;; esac;;
  container) [ -e "$D/$3" ] && exit 0 || exit 1;;
  run) n=; while [ $# -gt 0 ]; do [ "$1" = --name ] && n=$2; shift; done; mkdir -p "$D/$n.d"; : > "$D/$n"; echo "$n"; exit 0;;
  rm) for a; do case $a in -*|rm) ;; *) rm -rf "$D/$a" "$D/$a.d";; esac; done; exit 0;;
  cp) src=$2; dst=$3; case $dst in *:*) n=${dst%%:*}; f=${dst#*:}; [ -e "$D/$n" ] || exit 1; mkdir -p "$D/$n.d"; cp "$src" "$D/$n.d/$(basename "$f")";; esac; exit 0;;
  logs) exit 0;;
  exec) n=$2; shift 2; [ -e "$D/$n" ] || { echo "no such container $n" >&2; exit 125; }
    a="$*"
    case $a in
      *"getprop sys.boot_completed"*) [ "${FAKE_BOOT_FAIL:-}" = 1 ] && echo 0 || echo 1;;
      *"pm path org.helixreader.app.test"*) echo "package:/data/app/test/base.apk";;
      *"pm path org.helixreader.app"*) echo "package:/data/app/app/base.apk";;
      *"sha256sum /data/app/app/base.apk"*) h=$(sh1 "$D/$n.d/app.apk" 2>/dev/null || echo none); [ "${FAKE_STALE_INSTALL:-}" = 1 ] && h=deadbeef$h; echo "$h  /data/app/app/base.apk";;
      *"sha256sum /data/app/test/base.apk"*) h=$(sh1 "$D/$n.d/test.apk" 2>/dev/null || echo none); echo "$h  /data/app/test/base.apk";;
      *"am instrument"*) printf 'INSTRUMENTATION_CODE: -1\r\nOK (2 tests)\r\n';;
      *"logcat -d"*) echo "I/HRDMATRIX( 1): HRDMATRIX: sdk=34 px=1080x2400 dpi=420 widthDp=411 heightDp=900 webview=1080x2200";;
    esac; exit 0;;
esac; exit 0
FAKE
chmod +x "$T/bin/podman"; printf '#!/bin/sh\nexit 0\n' > "$T/bin/sleep"; chmod +x "$T/bin/sleep"
printf '#!/bin/sh\necho reap >> "$FAKE_LOG"\n' > "$T/reap.sh"; chmod +x "$T/reap.sh"
: > "$T/kvm"
run() { # $1=name; extra env via caller
  export FAKE_LOG=$T/$1.log FAKE_DIR=$T; : > "$FAKE_LOG"
  PATH="$T/bin:$PATH" HRD040_WORK="$T/work" HRD040_CACHE="$T/hostcache" HRD040_KVM="$T/kvm" HRD040_REAP="$T/reap.sh" EVID="$T/ev-$1" \
    bash "${HRD040_RUNNER:-$REPO/tools/android_matrix/run_matrix.sh}" emulators --rows api34-phone > "$T/$1.out" 2>&1; echo $? > "$T/$1.rc"; }

# --- scenario A: a leftover container with the OLD fixed name and a stale APK exists
mkdir -p "$T/ctr/hrdiso-android-api34.d"; : > "$T/ctr/hrdiso-android-api34"; echo apk-app-STALE > "$T/ctr/hrdiso-android-api34.d/app.apk"
run A
L=$T/A.log
grep -q '^exec hrdiso-android-api34 ' "$L" && ko "A1 leftover container was REUSED (exec on fixed name)" || ok "A1 leftover container not reused"
grep -q '^rm .*hrdiso-android-api34$' "$L" && ko "A1b leftover container (not ours) was touched" || ok "A1b leftover not touched"
runs=$(grep -c '^run ' "$L"); [ "$runs" -ge 1 ] || { ko "A0 no podman run seen (all later checks are vacuous)"; fail=1; }
bad=$(grep '^run ' "$L" | grep -vc -- '--timeout [0-9]'); [ "$bad" = 0 ] && [ "$runs" -ge 1 ] && ok "A2 every run has --timeout" || ko "A2 $bad run(s) without --timeout"
for lab in org.helix.isolation= org.helix.owner-pid= org.helix.lease-ts=; do
  bad=$(grep '^run ' "$L" | grep -vc -- "--label $lab"); [ "$bad" = 0 ] && [ "$runs" -ge 1 ] && ok "A3 label $lab on every run" || ko "A3 $bad run(s) lack label $lab"; done
first_run=$(grep -n '^run ' "$L" | head -1 | cut -d: -f1); reap=$(grep -n '^reap' "$L" | head -1 | cut -d: -f1)
[ -n "$reap" ] && [ -n "$first_run" ] && [ "$reap" -lt "$first_run" ] && ok "A4 reap before first container" || ko "A4 reap.sh not run before the first container"
names=$(grep '^run ' "$L" | sed -E 's/.*--name ([^ ]+).*/\1/' | sort -u); [ -n "$names" ] && ! echo "$names" | grep -qx 'hrdiso-android-api34' && ok "A5 unique per-run names ($names)"
cps=$(grep -c '^cp .*app-debug.apk' "$L"); [ "$cps" -ge 1 ] && ok "A6 APK copied fresh into the new container" || ko "A6 no fresh APK copy"
left=$(ls "$T/ctr" | grep -v '\.d$' | grep -vx 'hrdiso-android-api34' || true); [ -z "$left" ] && ok "A7 containers of this run removed on exit" || ko "A7 leftover containers: $left"
inst=$T/ev-A/api34-phone/installed.txt
[ -s "$inst" ] && ok "A8 installed.txt recorded" || ko "A8 installed.txt missing"
want=$(cut -d' ' -f1 "$T/work/apk/apk.sha256.txt" | head -1); grep -q "$want" "$inst" 2>/dev/null && ok "A9 installed app hash == built APK" || ko "A9 installed hash does not match the build"
grep -q 0123456789abcdef0123456789abcdef01234567 "$T/ev-A/source_rev.txt" 2>/dev/null && ok "A10 APK source revision recorded" || ko "A10 source_rev.txt missing"
grep -q '^[0-9a-f]\{64\}  [^/]' "$T/ev-A/apk.sha256.txt" 2>/dev/null && ok "A11 apk.sha256.txt uses relative names" || ko "A11 apk.sha256.txt not relative"

# --- scenario B: boot failure -> row BLOCKED, container still removed, nothing of ours left
FAKE_BOOT_FAIL=1 run B
leftB=$(ls "$T/ctr" | grep -v '\.d$' | grep -vx 'hrdiso-android-api34' || true); [ -z "$leftB" ] && grep -q '^run ' "$T/B.log" && ok "B1 failed-boot container removed" || ko "B1 leftover after failed boot: $leftB"
grep -q BLOCKED "$T/ev-B/api34-phone/row.json" 2>/dev/null && ok "B2 row BLOCKED" || ko "B2 row not BLOCKED"
grep -q "trap .*EXIT" "${HRD040_RUNNER:-$REPO/tools/android_matrix/run_matrix.sh}" && ok "B3 EXIT trap present" || ko "B3 no EXIT trap"

# --- scenario C: stale install mutant must be caught by summarize.py
FAKE_STALE_INSTALL=1 run C
inst=$T/ev-C/api34-phone/installed.txt; grep -q deadbeef "$inst" 2>/dev/null && ok "C1 mutant visible in installed.txt" || ko "C1 mutant not recorded"
python3 -I "$REPO/tools/android_matrix/summarize.py" "$T/ev-C" > "$T/C.sum" 2>&1; grep -q 'api34-phone *FAIL' "$T/C.sum" && ok "C2 stale install mutant -> row FAIL" || ko "C2 stale install not detected by summarize"
[ $fail = 0 ] && echo "ALL PASS" || echo "SOME FAILED"; exit $fail
