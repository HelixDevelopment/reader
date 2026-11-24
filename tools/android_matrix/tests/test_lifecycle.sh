#!/usr/bin/env bash
# test_lifecycle.sh - HRD-040 F1: run_matrix.sh must meet docs/test-isolation.md lifetime rules. Runs the REAL run_matrix.sh `emulators` stage against a
# logging fake `podman` (no containers are started) and asserts: unique per-run names, --timeout + the three org.helix.* labels on every `podman run`,
# reap.sh before the first container, a pre-existing same-name (leftover) container is NEVER reused/touched, APKs are always copied fresh,
# the installed package hash is read back per row and equals the built APK, containers are removed on exit (also when boot fails).
# Also driven: the build stages (prepare/unit/apk) through the same fake (every `create` carries --timeout N>0 and the three labels, VALUES checked: owner-pid
# is the runner's live pid != 1, lease-ts a recent epoch), an aborted run (a failing step mid-row) and a TERM'd run (no container of ours may remain),
# a pinned colliding name (HRD040_RUNID) which must be REFUSED, and HRD040_LIFE validation (0 / non-numeric / negative rejected, a valid value reaches --timeout).
# Mutants (each must be detected): FAKE_STALE_INSTALL=1 makes the device report a hash that differs from the built APK; tests/test_lifecycle_mutants.sh runs
# 14+ mutants of run_matrix.sh against this file and requires each to FAIL it.
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
reg() { # create / run: record the lifetime args with their VALUES, then make the container exist
  local verb=$1 n= to= iso= own= lease= prev=; shift
  for a in "$@"; do
    case $prev in --name) n=$a;; --timeout) to=$a;; --label) case $a in org.helix.isolation=*) iso=${a#*=};; org.helix.owner-pid=*) own=${a#*=};; org.helix.lease-ts=*) lease=${a#*=};; esac;; esac
    prev=$a; done
  alive=0; case $own in ''|*[!0-9]*) ;; *) kill -0 "$own" 2>/dev/null && alive=1;; esac
  echo "LBL verb=$verb name=$n timeout=$to iso=$iso owner=$own ppid=$PPID alive=$alive lease=$lease now=$(date +%s)" >> "$FAKE_LOG"
  mkdir -p "$D/$n.d"; : > "$D/$n"; echo "$n"; }
case $1 in
  image) case $2 in exists) exit 0;; inspect) echo "sha256:fakeid sha256:fakedigest"; exit 0;; esac;;
  container) [ -e "$D/$3" ] && exit 0 || exit 1;;
  create) reg create "$@"; exit 0;;
  run) reg run "$@"; exit 0;;
  start) [ "${FAKE_START_RC:-0}" = 0 ] || exit "$FAKE_START_RC"; exit 0;;
  rm) for a; do case $a in -*|rm) ;; *) rm -rf "$D/$a" "$D/$a.d";; esac; done; exit 0;;
  cp) src=$2; dst=$3
    case $src in *:*) # container -> host: materialise a fake build tree where the runner expects one
      mkdir -p "$dst" "$dst/outputs/apk/debug" "$dst/outputs/apk/androidTest/debug"; echo apk-app-FRESH > "$dst/outputs/apk/debug/app-debug.apk"; echo apk-test-FRESH > "$dst/outputs/apk/androidTest/debug/app-debug-androidTest.apk"; exit 0;; esac
    case $dst in *:*) n=${dst%%:*}; f=${dst#*:}; [ -e "$D/$n" ] || exit 1; mkdir -p "$D/$n.d"; [ -f "$src" ] && cp "$src" "$D/$n.d/$(basename "$f")";; esac; exit 0;;
  logs) exit 0;;
  exec) n=$2; shift 2; [ -e "$D/$n" ] || { echo "no such container $n" >&2; exit 125; }
    a="$*"
    case $a in
      *"getprop sys.boot_completed"*) [ "${FAKE_HANG:-}" = 1 ] && { echo $$ > "$FAKE_DIR/hang.pid"; exec /bin/sleep 20; }; [ "${FAKE_BOOT_FAIL:-}" = 1 ] && echo 0 || echo 1;;
      *"wm size"*) [ "${FAKE_ABORT:-}" = 1 ] && exit 7;;
      *"pm uninstall org.helixreader.app.test"*) echo "uninstall test" >> "$FAKE_LOG"; rm -f "$D/$n.d/test.inst";;
      *"pm uninstall org.helixreader.app"*) echo "uninstall app" >> "$FAKE_LOG"; rm -f "$D/$n.d/app.inst";;
      *"install -r -t /tmp/app.apk"*) [ "${FAKE_INSTALL_FAIL:-}" = 1 ] && exit 1; : > "$D/$n.d/app.inst";;
      *"install -r -t /tmp/test.apk"*) [ "${FAKE_INSTALL_FAIL:-}" = 1 ] && exit 1; : > "$D/$n.d/test.inst";;
      *"pm path org.helixreader.app.test"*) [ -e "$D/$n.d/test.inst" ] && echo "package:/data/app/test/base.apk";;
      *"pm path org.helixreader.app"*) [ -e "$D/$n.d/app.inst" ] && echo "package:/data/app/app/base.apk";;
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
RUNNER=${HRD040_RUNNER:-$REPO/tools/android_matrix/run_matrix.sh}
runx() { # $1=name, rest = runner args; extra env via the caller; HRD040_WORK defaults to the emulator-stage work dir
  local nm=$1; shift; export FAKE_LOG=$T/$nm.log FAKE_DIR=$T; : > "$FAKE_LOG"
  PATH="$T/bin:$PATH" HRD040_WORK="${WORKDIR:-$T/work}" HRD040_CACHE="$T/hostcache" HRD040_KVM="$T/kvm" HRD040_REAP="$T/reap.sh" EVID="$T/ev-$nm" \
    bash "$RUNNER" "$@" > "$T/$nm.out" 2>&1; echo $? > "$T/$nm.rc"; }
run() { runx "$1" emulators --rows api34-phone; }
nocreate_left() { ls "$T/ctr" | grep -v '\.d$' | grep -Ev '^hrdiso-android-api34(-pin1)?$' || true; }
# lifetime checks on every create/run line of a log: --timeout N>0, labels present, and the label VALUES
lifetime_checks() { # $1=log $2=tag
  local L=$1 g=$2 n; n=$(grep -cE '^(create|run) ' "$L")
  [ "$n" -ge 1 ] || { ko "$g-0 no podman create/run seen (all later checks are vacuous)"; return; }
  [ "$(grep -E '^(create|run) ' "$L" | grep -vcE -- '--timeout [1-9][0-9]* ')" = 0 ] && ok "$g-1 every create/run has --timeout N>0 ($n seen)" || ko "$g-1 a create/run lacks --timeout N>0"
  for lab in org.helix.isolation= org.helix.owner-pid= org.helix.lease-ts=; do
    [ "$(grep -E '^(create|run) ' "$L" | grep -vc -- "--label $lab")" = 0 ] && ok "$g-2 label $lab on every create/run" || ko "$g-2 a create/run lacks label $lab"; done
  [ "$(grep -c '^LBL ' "$L")" = "$n" ] && ok "$g-3 every create/run was recorded with its values" || ko "$g-3 value records missing"
  local bad=0 line
  while read -r line; do
    t=${line#*timeout=}; t=${t%% *}; o=${line#*owner=}; o=${o%% *}; pp=${line#*ppid=}; pp=${pp%% *}; al=${line#*alive=}; al=${al%% *}
    ls_=${line#*lease=}; ls_=${ls_%% *}; nw=${line#*now=}; is=${line#*iso=}; is=${is%% *}
    case $t$o$ls_$nw in *[!0-9]*|'') bad=1; echo "  non-numeric value in: $line";; esac
    [ "$bad" = 0 ] && { [ "$t" -gt 0 ] && [ "$o" -gt 1 ] && [ "$al" = 1 ] && [ "$o" = "$pp" ] && [ "$is" = gui-test ] && [ $((nw - ls_)) -ge 0 ] && [ $((nw - ls_)) -le 300 ] || { bad=1; echo "  bad label value in: $line"; }; }
  done < <(grep '^LBL ' "$L")
  [ "$bad" = 0 ] && ok "$g-4 timeout>0, owner-pid = the runner's LIVE pid != 1, isolation=gui-test, lease-ts recent epoch (by value)" || ko "$g-4 label/timeout VALUES wrong"; }

# --- scenario A: a leftover container with the OLD fixed name and a stale APK exists
mkdir -p "$T/ctr/hrdiso-android-api34.d"; : > "$T/ctr/hrdiso-android-api34"; echo apk-app-STALE > "$T/ctr/hrdiso-android-api34.d/app.apk"
run A
L=$T/A.log
grep -q '^exec hrdiso-android-api34 ' "$L" && ko "A1 leftover container was REUSED (exec on fixed name)" || ok "A1 leftover container not reused"
grep -q '^rm .*hrdiso-android-api34$' "$L" && ko "A1b leftover container (not ours) was touched" || ok "A1b leftover not touched"
lifetime_checks "$L" A2
first_run=$(grep -n '^run ' "$L" | head -1 | cut -d: -f1); reap=$(grep -n '^reap' "$L" | head -1 | cut -d: -f1)
[ -n "$reap" ] && [ -n "$first_run" ] && [ "$reap" -lt "$first_run" ] && ok "A4 reap before first container" || ko "A4 reap.sh not run before the first container"
namesA=$(grep '^run ' "$L" | sed -E 's/.*--name ([^ ]+).*/\1/' | sort -u)
echo "$namesA" | grep -qE '^hrdiso-android-api34-[0-9]+-[0-9]+$' && ok "A5 per-run name has the pid-epoch suffix ($namesA)" || ko "A5 name is not unique-per-run: '$namesA'"
cps=$(grep -c '^cp .*app-debug.apk' "$L"); [ "$cps" -ge 1 ] && ok "A6 APK copied fresh into the new container" || ko "A6 no fresh APK copy"
left=$(nocreate_left); [ -z "$left" ] && ok "A7 containers of this run removed on exit" || ko "A7 leftover containers: $left"
inst=$T/ev-A/api34-phone/installed.txt
[ -s "$inst" ] && ok "A8 installed.txt recorded" || ko "A8 installed.txt missing"
want=$(cut -d' ' -f1 "$T/work/apk/apk.sha256.txt" | head -1); grep -q "$want" "$inst" 2>/dev/null && ok "A9 installed app hash == built APK" || ko "A9 installed hash does not match the build"
grep -q 0123456789abcdef0123456789abcdef01234567 "$T/ev-A/source_rev.txt" 2>/dev/null && ok "A10 APK source revision recorded" || ko "A10 source_rev.txt missing"
grep -q '^[0-9a-f]\{64\}  [^/]' "$T/ev-A/apk.sha256.txt" 2>/dev/null && ok "A11 apk.sha256.txt uses relative names" || ko "A11 apk.sha256.txt not relative"
u=$(grep -n '^uninstall app' "$L" | head -1 | cut -d: -f1); i=$(grep -n 'install -r -t /tmp/app.apk' "$L" | head -1 | cut -d: -f1)
[ -n "$u" ] && [ -n "$i" ] && [ "$u" -lt "$i" ] && ok "A12 uninstall precedes install" || ko "A12 no uninstall before install"

# --- scenario B: boot failure -> row BLOCKED, container still removed, nothing of ours left
FAKE_BOOT_FAIL=1 run B
leftB=$(nocreate_left); [ -z "$leftB" ] && grep -q '^run ' "$T/B.log" && ok "B1 failed-boot container removed" || ko "B1 leftover after failed boot: $leftB"
grep -q BLOCKED "$T/ev-B/api34-phone/row.json" 2>/dev/null && ok "B2 row BLOCKED" || ko "B2 row not BLOCKED"
grep -q "trap .*EXIT" "$RUNNER" && ok "B3 EXIT trap present (text only; B4/B5 prove it works)" || ko "B3 no EXIT trap"

# --- scenario B4: an ABORTED run (a step fails mid-row, set -e) must leave no container of ours (the EXIT trap is what removes it)
FAKE_ABORT=1 run B4
rcB4=$(cat "$T/B4.rc"); leftB4=$(nocreate_left)
[ "$rcB4" = 7 ] && grep -q '^run ' "$T/B4.log" && ok "B4a aborted run exits with the failing step's rc 7 after starting a container" || ko "B4a abort scenario did not abort (rc $rcB4) or started nothing"
[ -z "$leftB4" ] && ok "B4b aborted run left no container" || ko "B4b aborted run LEFT containers: $leftB4"
# --- scenario B5: TERM during boot wait -> rc 143 and no container left
export FAKE_LOG=$T/B5.log FAKE_DIR=$T; : > "$FAKE_LOG"; rm -f "$T/hang.pid"
FAKE_HANG=1 PATH="$T/bin:$PATH" HRD040_WORK="$T/work" HRD040_CACHE="$T/hostcache" HRD040_KVM="$T/kvm" HRD040_REAP="$T/reap.sh" EVID="$T/ev-B5" bash "$RUNNER" emulators --rows api34-phone > "$T/B5.out" 2>&1 &
rpid=$!; for _ in $(seq 1 100); do [ -s "$T/hang.pid" ] && break; /bin/sleep 0.1; done
hp=$(cat "$T/hang.pid" 2>/dev/null || true)
case $rpid in ''|*[!0-9]*|0|1) echo "refusing to signal pid '$rpid'"; fail=1;; *)
  case $hp in ''|*[!0-9]*|0|1) ;; *) kill -TERM "$hp" 2>/dev/null || true;; esac
  kill -TERM "$rpid" 2>/dev/null || true;; esac
wait "$rpid"; rcB5=$?; leftB5=$(nocreate_left)
[ -n "$hp" ] && ok "B5a runner was inside the boot wait when TERM arrived" || ko "B5a never reached the boot wait (vacuous)"
[ "$rcB5" = 143 ] && ok "B5b TERM -> rc 143" || ko "B5b TERM gave rc $rcB5"
[ -z "$leftB5" ] && ok "B5c TERM'd run left no container" || ko "B5c TERM'd run LEFT containers: $leftB5"

# --- scenario C: stale install mutant must be caught by summarize.py
FAKE_STALE_INSTALL=1 run C
inst=$T/ev-C/api34-phone/installed.txt; grep -q deadbeef "$inst" 2>/dev/null && ok "C1 mutant visible in installed.txt" || ko "C1 mutant not recorded"
python3 -I "$REPO/tools/android_matrix/summarize.py" "$T/ev-C" > "$T/C.sum" 2>&1; grep -q 'api34-phone *FAIL' "$T/C.sum" && ok "C2 stale install mutant -> row FAIL" || ko "C2 stale install not detected by summarize"
namesC=$(grep '^run ' "$T/C.log" | sed -E 's/.*--name ([^ ]+).*/\1/' | sort -u); [ -n "$namesC" ] && [ "$namesC" != "$namesA" ] && ok "C3 a second run gets a different container name" || ko "C3 two runs share the name '$namesC'"
FAKE_INSTALL_FAIL=1 run C4
python3 -I "$REPO/tools/android_matrix/summarize.py" "$T/ev-C4" > "$T/C4.sum" 2>&1; grep -q 'api34-phone *FAIL' "$T/C4.sum" && ok "C4 install failure (nothing installed) -> row FAIL" || ko "C4 failed install not detected"

# --- scenario D: the build stages (prepare, unit, apk) are driven through the same fake: lifetime args on EVERY container, not only the emulator ones
WORKDIR=$T/work2; export WORKDIR
runx D1 prepare; runx D2 unit; runx D3 apk
for s in D1 D2 D3; do [ "$(cat "$T/$s.rc")" = 0 ] && ok "$s stage ran (rc 0)" || { ko "$s stage rc $(cat "$T/$s.rc"): $(tail -3 "$T/$s.out")"; }; done
cat "$T/D1.log" "$T/D2.log" "$T/D3.log" > "$T/D.log"
[ "$(grep -c '^create ' "$T/D.log")" -ge 3 ] && ok "D4 prepare + unit + apk each created a container" || ko "D4 fewer than 3 create lines (vacuous)"
for s in D1 D2 D3; do [ "$(grep -c '^create ' "$T/$s.log")" -ge 1 ] || ko "D5 stage $s created no container"; lifetime_checks "$T/$s.log" "$s-"; done
unset WORKDIR

# --- scenario D0: `seed` refuses without EVID (no stray evidence dir in the repo) and, with EVID, records SDK provenance and carries the lifetime args
export FAKE_LOG=$T/D0a.log FAKE_DIR=$T; : > "$FAKE_LOG"
( unset EVID; PATH="$T/bin:$PATH" HRD040_WORK="$T/work3" HRD040_CACHE="$T/hostcache" HRD040_KVM="$T/kvm" HRD040_REAP="$T/reap.sh" bash "$RUNNER" seed > "$T/D0a.out" 2>&1 ); rcD0=$?
[ "$rcD0" = 2 ] && ! grep -qE '^(run|create) ' "$T/D0a.log" && ok "D0a seed without EVID refused (rc 2, nothing started)" || ko "D0a seed without EVID rc $rcD0"
runx D0b seed; [ "$(cat "$T/D0b.rc")" = 0 ] && [ -f "$T/ev-D0b/sdk_builder_image.txt" ] && [ -f "$T/ev-D0b/sdk_packages.txt" ] && ok "D0b seed with EVID records sdk_builder_image.txt + sdk_packages.txt" || ko "D0b seed provenance files missing (rc $(cat "$T/D0b.rc"))"
lifetime_checks "$T/D0b.log" D0b-

# --- scenario E: HRD040_LIFE validation (rejected before any container) and propagation of a valid value
for bad in 0 abc -5 1.5 007x; do
  HRD040_LIFE=$bad runx "E$bad" emulators --rows api34-phone; r=$(cat "$T/E$bad.rc")
  [ "$r" != 0 ] && ! grep -qE '^(run|create) ' "$T/E$bad.log" && ok "E rejected HRD040_LIFE=$bad (rc $r, no container)" || ko "E HRD040_LIFE=$bad was accepted (rc $r)"
done
HRD040_LIFE=900 runx Eok emulators --rows api34-phone
grep -qE '^run .*--timeout 900 ' "$T/Eok.log" && ok "E valid HRD040_LIFE=900 reaches --timeout" || ko "E valid HRD040_LIFE not propagated"

# --- scenario F: a same-name container (pinned run id) is REFUSED: not reused, not touched, nothing started
mkdir -p "$T/ctr/hrdiso-android-api34-pin1.d"; : > "$T/ctr/hrdiso-android-api34-pin1"
HRD040_RUNID=pin1 runx F emulators --rows api34-phone; rcF=$(cat "$T/F.rc")
[ "$rcF" != 0 ] && ok "F1 collision -> non-zero rc ($rcF)" || ko "F1 collision did not abort"
! grep -qE '^(run|create) ' "$T/F.log" && ok "F2 nothing started under a colliding name" || ko "F2 a container was started despite the collision"
! grep -qE '^(exec|rm|cp) .*hrdiso-android-api34-pin1' "$T/F.log" && ok "F3 colliding container not touched" || ko "F3 colliding container was used/removed"
[ -e "$T/ctr/hrdiso-android-api34-pin1" ] && ok "F4 colliding container still exists" || ko "F4 colliding container was removed"
[ $fail = 0 ] && echo "ALL PASS" || echo "SOME FAILED"; exit $fail
