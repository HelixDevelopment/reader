#!/usr/bin/env bash
# test_lifecycle.sh - HRD-040 F1: run_matrix.sh must meet docs/test-isolation.md lifetime rules. Runs the REAL run_matrix.sh `emulators` stage against a
# logging fake `podman` (no containers are started) and asserts: unique per-run names, --timeout + the three org.helix.* labels on every `podman run`,
# reap.sh before the first container, a pre-existing same-name (leftover) container is NEVER reused/touched, APKs are always copied fresh,
# the installed package hash is read back per row and equals the built APK, containers are removed on exit (also when boot fails).
# Also driven: the build stages (prepare/unit/apk) through the same fake (every `create` carries --timeout N>0 and the three labels, VALUES checked: owner-pid
# is the runner's live pid != 1, lease-ts a recent epoch), an aborted run (a failing step mid-row) and a TERM'd run (no container of ours may remain),
# a pinned colliding name (HRD040_RUNID) which must be REFUSED, and HRD040_LIFE validation (0 / non-numeric / negative rejected, a valid value reaches --timeout).
# Mutants (each must be detected): FAKE_STALE_INSTALL=1 makes the device report a hash that differs from the built APK; tests/test_lifecycle_mutants.sh runs
# 55 mutants of run_matrix.sh against this file and requires each to FAIL it.
set -u
HERE=$(cd "$(dirname "$0")" && pwd); REPO=$(cd "$HERE/../../.." && pwd)
T=$(mktemp -d "${TMPDIR:-/tmp}/hrd040-lc.XXXXXX"); : "${T:?}"; trap 'rm -rf "${T:?}"' EXIT
fail=0; ok() { echo "PASS $1"; }; ko() { echo "FAIL $1"; fail=1; }
mkdir -p "$T/bin" "$T/s" "$T/work/apk" "$T/hostcache"   # NO shared container store: every scenario gets $T/s/<name>/ctr (see the isolation audit below)
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
  image) case $2 in exists) [ "${FAKE_NOIMAGE:-}" = 1 ] && exit 1; exit 0;; inspect) echo "sha256:fakeid sha256:fakedigest"; exit 0;; esac;;
  container) [ -e "$D/$3" ] && exit 0 || exit 1;;
  create) reg create "$@"; exit 0;;
  run) reg run "$@"; exit 0;;
  start) [ -n "${FAKE_WATCH:-}" ] && echo "WATCH posts=$(grep -c '\"POST\"' "$FAKE_WATCH")" >> "$FAKE_LOG"; [ "${FAKE_START_RC:-0}" = 0 ] || exit "$FAKE_START_RC"; exit 0;;
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
      *"sha256sum"*) echo "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  -";;   # what a real device prints for an EMPTY path (hash of stdin): must never be recorded as a package hash
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
  local nm=$1; shift; export FAKE_LOG=$T/$nm.log FAKE_DIR=$T/s/$nm; mkdir -p "$FAKE_DIR/ctr"; : > "$FAKE_LOG"
  PATH="$T/bin:$PATH" HRD040_WORK="${WORKDIR:-$T/work}" HRD040_CACHE="$T/hostcache" HRD040_KVM="${KVMX:-$T/kvm}" HRD040_REAP="${REAPX:-$T/reap.sh}" EVID="$T/ev-$nm" \
    bash "$RUNNER" "$@" > "$T/$nm.out" 2>&1; echo $? > "$T/$nm.rc"; }
run() { runx "$1" emulators --rows api34-phone; }
# containers left in ONE scenario's own store ($1 = scenario name); a deliberately seeded leftover (api34 / api34-pin1) is not "ours"
nocreate_left() { ls "$T/s/$1/ctr" | grep -v '\.d$' | grep -Ev '^hrdiso-android-api34(-pin1)?$' || true; }
seedctr() { mkdir -p "$T/s/$1/ctr/$2.d"; : > "$T/s/$1/ctr/$2"; }   # $1=scenario $2=container name: a pre-existing container in THAT scenario's store only
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
seedctr A hrdiso-android-api34; echo apk-app-STALE > "$T/s/A/ctr/hrdiso-android-api34.d/app.apk"
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
left=$(nocreate_left A); [ -z "$left" ] && ok "A7 containers of this run removed on exit" || ko "A7 leftover containers: $left"
inst=$T/ev-A/api34-phone/installed.txt
[ -s "$inst" ] && ok "A8 installed.txt recorded" || ko "A8 installed.txt missing"
want=$(cut -d' ' -f1 "$T/work/apk/apk.sha256.txt" | head -1); grep -q "$want" "$inst" 2>/dev/null && ok "A9 installed app hash == built APK" || ko "A9 installed hash does not match the build"
grep -q 0123456789abcdef0123456789abcdef01234567 "$T/ev-A/source_rev.txt" 2>/dev/null && ok "A10 APK source revision recorded" || ko "A10 source_rev.txt missing"
grep -q '^[0-9a-f]\{64\}  [^/]' "$T/ev-A/apk.sha256.txt" 2>/dev/null && ok "A11 apk.sha256.txt uses relative names" || ko "A11 apk.sha256.txt not relative"
want_t=$(cut -d' ' -f1 "$T/work/apk/apk.sha256.txt" | sed -n 2p); grep -q "$want_t" "$inst" 2>/dev/null && ok "A9b installed test-APK hash == built androidTest APK" || ko "A9b installed test hash does not match the build"
u=$(grep -n '^uninstall app' "$L" | head -1 | cut -d: -f1); i=$(grep -n 'install -r -t /tmp/app.apk' "$L" | head -1 | cut -d: -f1)
[ -n "$u" ] && [ -n "$i" ] && [ "$u" -lt "$i" ] && ok "A12 uninstall precedes install" || ko "A12 no uninstall before install"

# --- scenario B: boot failure -> row BLOCKED, container still removed, nothing of ours left
FAKE_BOOT_FAIL=1 run B
leftB=$(nocreate_left B); [ -z "$leftB" ] && grep -q '^run ' "$T/B.log" && ok "B1 failed-boot container removed" || ko "B1 leftover after failed boot: $leftB"
grep -q BLOCKED "$T/ev-B/api34-phone/row.json" 2>/dev/null && ok "B2 row BLOCKED" || ko "B2 row not BLOCKED"
grep -q "trap .*EXIT" "$RUNNER" && ok "B3 EXIT trap present (text only; B4/B5 prove it works)" || ko "B3 no EXIT trap"

# --- scenario B4: an ABORTED run (a step fails mid-row, set -e) must leave no container of ours (the EXIT trap is what removes it)
FAKE_ABORT=1 run B4
rcB4=$(cat "$T/B4.rc"); leftB4=$(nocreate_left B4)
[ "$rcB4" = 7 ] && grep -q '^run ' "$T/B4.log" && ok "B4a aborted run exits with the failing step's rc 7 after starting a container" || ko "B4a abort scenario did not abort (rc $rcB4) or started nothing"
[ -z "$leftB4" ] && ok "B4b aborted run left no container" || ko "B4b aborted run LEFT containers: $leftB4"
# --- scenario B5: TERM during boot wait -> rc 143 and no container left
export FAKE_LOG=$T/B5.log FAKE_DIR=$T/s/B5; mkdir -p "$FAKE_DIR/ctr"; : > "$FAKE_LOG"
FAKE_HANG=1 PATH="$T/bin:$PATH" HRD040_WORK="$T/work" HRD040_CACHE="$T/hostcache" HRD040_KVM="$T/kvm" HRD040_REAP="$T/reap.sh" EVID="$T/ev-B5" bash "$RUNNER" emulators --rows api34-phone > "$T/B5.out" 2>&1 &
rpid=$!; for _ in $(seq 1 100); do [ -s "$T/s/B5/hang.pid" ] && break; /bin/sleep 0.1; done
hp=$(cat "$T/s/B5/hang.pid" 2>/dev/null || true)
case $rpid in ''|*[!0-9]*|0|1) echo "refusing to signal pid '$rpid'"; fail=1;; *)
  case $hp in ''|*[!0-9]*|0|1) ;; *) kill -TERM "$hp" 2>/dev/null || true;; esac
  kill -TERM "$rpid" 2>/dev/null || true;; esac
wait "$rpid"; rcB5=$?; leftB5=$(nocreate_left B5)
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
grep -q '^app none none$' "$T/ev-C4/api34-phone/installed.txt" 2>/dev/null && grep -q '^test none none$' "$T/ev-C4/api34-phone/installed.txt" && ok "C4b nothing installed -> 'none none' (the empty-path hash is never recorded)" || ko "C4b failed install recorded a hash"

# --- scenario D: the build stages (prepare, unit, apk) are driven through the same fake: lifetime args on EVERY container, not only the emulator ones
WORKDIR=$T/work2; export WORKDIR
runx D1 prepare; runx D2 unit; runx D3 apk
for s in D1 D2 D3; do [ "$(cat "$T/$s.rc")" = 0 ] && ok "$s stage ran (rc 0)" || { ko "$s stage rc $(cat "$T/$s.rc"): $(tail -3 "$T/$s.out")"; }; done
cat "$T/D1.log" "$T/D2.log" "$T/D3.log" > "$T/D.log"
[ "$(grep -c '^create ' "$T/D.log")" -ge 3 ] && ok "D4 prepare + unit + apk each created a container" || ko "D4 fewer than 3 create lines (vacuous)"
for s in D1 D2 D3; do [ "$(grep -c '^create ' "$T/$s.log")" -ge 1 ] || ko "D5 stage $s created no container"; lifetime_checks "$T/$s.log" "$s-"; done
unset WORKDIR

# --- scenario D0: `seed` refuses without EVID (no stray evidence dir in the repo) and, with EVID, records SDK provenance and carries the lifetime args
export FAKE_LOG=$T/D0a.log FAKE_DIR=$T/s/D0a; mkdir -p "$FAKE_DIR/ctr"; : > "$FAKE_LOG"
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

# --- scenario F: a same-name container (pinned run id) is REFUSED: not reused, not touched, nothing started.
# F's store holds ONLY the exact colliding container (no api34 leftover), so a collision check that compares the wrong name (N7: prefix only) cannot pass for the wrong reason.
seedctr F hrdiso-android-api34-pin1
HRD040_RUNID=pin1 runx F emulators --rows api34-phone; rcF=$(cat "$T/F.rc")
[ "$rcF" != 0 ] && ok "F1 collision -> non-zero rc ($rcF)" || ko "F1 collision did not abort"
! grep -qE '^(run|create) ' "$T/F.log" && ok "F2 nothing started under a colliding name" || ko "F2 a container was started despite the collision"
! grep -qE '^(exec|rm|cp) .*hrdiso-android-api34-pin1' "$T/F.log" && ok "F3 colliding container not touched" || ko "F3 colliding container was used/removed"
[ -e "$T/s/F/ctr/hrdiso-android-api34-pin1" ] && ok "F4 colliding container still exists" || ko "F4 colliding container was removed"
grep -q 'already exists' "$T/F.out" && ok "F4b refusal says why" || ko "F4b no refusal message"
# F5 (inverse of F): ONLY a prefix-named container exists (what N7's check looks at); the pinned exact name is free -> must NOT be refused
seedctr F5 hrdiso-android-api34
HRD040_RUNID=pin1 runx F5 emulators --rows api34-phone; rcF5=$(cat "$T/F5.rc")
! grep -q 'already exists' "$T/F5.out" && grep -q '^run .*--name hrdiso-android-api34-pin1 ' "$T/F5.log" && ok "F5 a prefix-only container does NOT cause refusal (the exact name is free; run started)" || ko "F5 prefix-only container wrongly refused (rc $rcF5)"
[ -e "$T/s/F5/ctr/hrdiso-android-api34" ] && ok "F5b the unrelated prefix container was left alone" || ko "F5b prefix container was removed"

# --- scenario G (M10): two image groups -> the first group's container is REMOVED before the second group starts (peak memory = one emulator, not one per image)
runx G emulators --rows api34-phone,api30-phone
nrun=$(grep -c '^run ' "$T/G.log"); firstrm=$(grep -n '^rm -f hrdiso-android-api34-' "$T/G.log" | head -1 | cut -d: -f1); secondrun=$(grep -n '^run .*--name hrdiso-android-api30-' "$T/G.log" | head -1 | cut -d: -f1)
[ "$nrun" = 2 ] && ok "G1 two image groups -> two emulator containers" || ko "G1 expected 2 runs, saw $nrun"
[ -n "$firstrm" ] && [ -n "$secondrun" ] && [ "$firstrm" -lt "$secondrun" ] && ok "G2 group 1's container removed (log line $firstrm) before group 2 starts (line $secondrun)" || ko "G2 group 1's container not removed before group 2 (rm=$firstrm run2=$secondrun)"
# --- N14: the emulator container has no network
grep -E '^run ' "$T/A.log" | grep -q -- '--network none' && ok "G3 emulator container runs with --network none" || ko "G3 emulator container lacks --network none"
grep -E '^run ' "$T/A.log" | grep -q -- '--device /dev/kvm ' && ok "G3b only /dev/kvm is passed as a device" || ko "G3b /dev/kvm device arg missing"

# --- scenario H (N13): HRD040_RUNID validation, rejected before any container (rc 2)
for bad in 'a/b' 'a b' 'x;y' 'a_b'; do
  tag=H$(printf %s "$bad" | tr -c 'A-Za-z0-9' _); HRD040_RUNID=$bad runx "$tag" emulators --rows api34-phone; r=$(cat "$T/$tag.rc")
  [ "$r" = 2 ] && ! grep -qE '^(run|create) ' "$T/$tag.log" && ok "H HRD040_RUNID='$bad' rejected (rc 2, no container)" || ko "H HRD040_RUNID='$bad' accepted (rc $r)"
done

# --- scenario I (N19): a failing / missing reap.sh aborts the run BEFORE any container and is reported
printf '#!/bin/sh\nexit 3\n' > "$T/reap_fail.sh"; chmod +x "$T/reap_fail.sh"
REAPX="$T/reap_fail.sh" runx I1 emulators --rows api34-phone
[ "$(cat "$T/I1.rc")" = 1 ] && ! grep -qE '^(run|create) ' "$T/I1.log" && grep -q 'reap.sh failed (rc 3)' "$T/I1.out" && ok "I1 failing reap.sh -> rc 1, reported with its rc, nothing started" || ko "I1 failing reap.sh not handled (rc $(cat "$T/I1.rc"))"
REAPX="$T/does-not-exist.sh" runx I2 emulators --rows api34-phone
[ "$(cat "$T/I2.rc")" = 1 ] && ! grep -qE '^(run|create) ' "$T/I2.log" && grep -q 'missing or not executable' "$T/I2.out" && ok "I2 missing reap.sh -> rc 1, reported as missing, nothing started" || ko "I2 missing reap.sh not handled"

# --- scenario J (N16, N20): the source revision is resolved at PACK time and carried to the apk stage
OLDREV=$(git -C "$REPO" rev-list HEAD -- clients | tail -1); NOWREV=$(git -C "$REPO" rev-parse HEAD)
if [ -n "$OLDREV" ] && [ "$OLDREV" != "$NOWREV" ]; then
  WORKDIR=$T/work5; export WORKDIR
  HRD040_REV=$OLDREV runx J1 prepare; runx J2 apk     # apk runs with HRD040_REV unset: HEAD is NOT the revision that was packed
  [ "$(cat "$T/work5/prepared/.source_rev" 2>/dev/null)" = "$OLDREV" ] && ok "J1 prepare records the packed revision ($OLDREV)" || ko "J1 prepared/.source_rev wrong"
  [ "$(cat "$T/J2.rc")" = 0 ] && [ "$(cat "$T/work5/apk/source_rev.txt" 2>/dev/null)" = "$OLDREV" ] && ok "J2 apk/source_rev.txt = the PACKED revision, not the current HEAD ($NOWREV)" || ko "J2 apk source_rev is not the packed revision"
  unset WORKDIR
else ko "J0 cannot pick a revision different from HEAD that has clients/ (vacuous)"; fi
WORKDIR=$T/work6; export WORKDIR; mkdir -p "$T/work6"
runx J3 apk
[ "$(cat "$T/J3.rc")" != 0 ] && ! grep -qE '^(create|run) ' "$T/J3.log" && grep -q 'no prepared/.source_rev' "$T/J3.out" && ok "J3 apk without a prepared tree is refused before any container" || ko "J3 apk ran without prepared/.source_rev (rc $(cat "$T/J3.rc"))"
unset WORKDIR

# --- scenario K: honest non-run rows (image absent, KVM unusable, an emulator that never booted blocks the rest of its image group); no container for SKIP rows
FAKE_NOIMAGE=1 runx K1 emulators --rows api34-phone
grep -q '"state":"SKIP","reason":"image_absent' "$T/ev-K1/api34-phone/row.json" 2>/dev/null && ! grep -qE '^(run|create) ' "$T/K1.log" && ok "K1 absent image -> SKIP image_absent, no container" || ko "K1 absent image not an honest SKIP"
KVMX="$T/no-such-kvm" runx K2 emulators --rows api34-phone
grep -q '"state":"SKIP","reason":"kvm_unusable' "$T/ev-K2/api34-phone/row.json" 2>/dev/null && ! grep -qE '^(run|create) ' "$T/K2.log" && ok "K2 unusable KVM -> SKIP kvm_unusable, no container" || ko "K2 unusable KVM not an honest SKIP"
FAKE_BOOT_FAIL=1 runx K3 emulators --rows api34-phone,api34-small
[ "$(grep -c '^run ' "$T/K3.log")" = 1 ] && grep -q '"state":"BLOCKED","reason":"emulator did not report' "$T/ev-K3/api34-phone/row.json" && grep -q '"state":"BLOCKED","reason":"emulator of this image never booted' "$T/ev-K3/api34-small/row.json" && [ -e "$T/ev-K3/api34-phone/boot_tail.log" ] && grep -q '^rm -f hrdiso-android-api34-' "$T/K3.log" && ok "K3 failed boot: one container, row 1 BLOCKED (+boot_tail.log), row 2 BLOCKED without a second boot, container removed" || ko "K3 boot-failure handling wrong"

# --- scenario M: resource/safety flags of every container kind + per-row artefacts, from the recorded argv (the section-6 'by value' claims)
EL=$(grep -E '^run ' "$T/A.log")
for f in '--memory 5g' '--memory-swap 5g' '--cpus 4' '--pids-limit 4096' '--oom-score-adj 500' '--userns=keep-id' '--cap-drop ALL' '--security-opt no-new-privileges'; do
  echo "$EL" | grep -q -- "$f" && ok "M1 emulator container carries $f" || ko "M1 emulator container lacks $f"; done
for s_ in D1 D2 D3; do C=$(grep -E '^create ' "$T/$s_.log")
  for f in '--userns=keep-id' '--cap-drop ALL' '--security-opt no-new-privileges' '--pids-limit'; do echo "$C" | grep -q -- "$f" && ok "M2 $s_ container carries $f" || ko "M2 $s_ container lacks $f"; done; done
grep -q -- '--memory 3g' "$T/D1.log" && grep -q -- '--pids-limit 1024' "$T/D1.log" && grep -E '^create .*--memory 6g' "$T/D2.log" | grep -q -- '--cpus 4 --pids-limit 2048' && ok "M2b memory/cpu/pids limits on prepare and build containers" || ko "M2b build container limits wrong"
grep -E '^create ' "$T/D1.log" | grep -q -- '--network none' && ko "M3 prepare (needs npm network) has --network none" || ok "M3 prepare keeps network (documented exception)"
for s_ in D2 D3; do grep -E '^create ' "$T/$s_.log" | grep -q -- '--network none' && ok "M3 $s_ build container has --network none" || ko "M3 $s_ build container lacks --network none"
  grep -E '^create ' "$T/$s_.log" | grep -q -- '-v hrdiso-android-cache:/cache:rw' && ok "M3b $s_ mounts only the named cache volume" || ko "M3b $s_ cache volume mount wrong"; done
! grep -E '^(create|run) ' "$T/A.log" "$T/D2.log" "$T/D3.log" | grep -qE -- ' -v /| --volume|--privileged|--network host' && ok "M4 no host bind mount / privileged / host network" || ko "M4 host bind mount or privileged/host network present"
grep -q 'wm size 1080x2400; wm density 420' "$T/A.log" && ok "M5 row geometry applied (wm size/density)" || ko "M5 wm size/density not applied"
grep -q 'am force-stop org.helixreader.app; wm size reset; wm density reset' "$T/A.log" && ok "M5b geometry reset after the row" || ko "M5b no geometry reset"
for f in row.json install.txt installed.txt instrument.txt am_start.txt screen.png logcat_hrdmatrix.txt device.txt; do [ -e "$T/ev-A/api34-phone/$f" ] && ok "M6 artefact $f" || ko "M6 artefact $f missing"; done
grep -q '"state":"RAN"' "$T/ev-A/api34-phone/row.json" && ok "M6b row.json RAN" || ko "M6b row.json not RAN"
! grep -q $'\r' "$T/ev-A/api34-phone/instrument.txt" && grep -q 'OK (2 tests)' "$T/ev-A/api34-phone/instrument.txt" && ok "M6c instrument.txt CR-stripped and kept" || ko "M6c instrument.txt not normalised"
[ ! -e "$T/ev-A/api34-phone/instrument.raw" ] && ok "M6e instrument.raw removed" || ko "M6e instrument.raw left behind"
[ -s "$T/ev-A/matrix.tsv.txt" ] && ok "M6g matrix copied into the evidence" || ko "M6g matrix.tsv.txt missing"
[ "$(ls "$T/ev-A" | grep -c '^api')" = 1 ] && ok "M6f --rows selects exactly the requested row" || ko "M6f --rows ignored"
[ -s "$T/ev-A/image_api34.txt" ] && ok "M6d image identity recorded" || ko "M6d image_api34.txt missing"
grep -q 'am instrument -w -r -e class org.helixreader.app.MainActivitySmokeTest,com.getcapacitor.myapp.ExampleInstrumentedTest org.helixreader.app.test/androidx.test.runner.AndroidJUnitRunner' "$T/A.log" && ok "M7 instrumentation command (classes + runner)" || ko "M7 instrumentation command wrong"
[ "$(cat "$T/ev-D2/unit/exit.txt" 2>/dev/null)" = "gradle exit 0" ] && ok "M8 unit stage records its gradle exit" || ko "M8 unit exit.txt wrong"
runx Z bogus; [ "$(cat "$T/Z.rc")" = 2 ] && ! grep -q '^reap' "$T/Z.log" && ok "Z unknown stage -> usage, rc 2, no reap/containers" || ko "Z unknown stage handling wrong"

# --- scenario L: the paired unit-test mutation stage (the mutant must be APPLIED while gradle runs, RESTORED afterwards, and gradle must FAIL on it)
mkpol() { local d=$1/prepared/android/android/app/src/main/java/org/helixreader/app; mkdir -p "$d"; printf '%s\n' "$2" > "$d/ApiRequestPolicy.java"; echo "$d/ApiRequestPolicy.java"; }
GOODSRC='List<String> ok = Arrays.asList("GET", "HEAD", "OPTIONS");'
PF=$(mkpol "$T/work7" "$GOODSRC"); WORKDIR=$T/work7; export WORKDIR
FAKE_WATCH=$PF FAKE_START_RC=1 runx L1 mutation
[ "$(cat "$T/L1.rc")" = 0 ] && grep -q '^WATCH posts=1' "$T/L1.log" && [ "$(cat "$PF")" = "$GOODSRC" ] && grep -q 'gradle exit 1 (nonzero expected)' "$T/ev-L1/unit_mutant/exit.txt" && ok "L1 mutant applied during the run (POST seen), restored after, failing gradle = rc 0" || ko "L1 mutation stage wrong (rc $(cat "$T/L1.rc"))"
FAKE_WATCH=$PF FAKE_START_RC=0 runx L2 mutation
[ "$(cat "$T/L2.rc")" != 0 ] && [ "$(cat "$PF")" = "$GOODSRC" ] && ok "L2 mutant that does NOT fail gradle -> rc != 0 (tests prove nothing), source restored" || ko "L2 surviving mutant not reported (rc $(cat "$T/L2.rc"))"
PF2=$(mkpol "$T/work7" 'no policy list here'); runx L3 mutation
[ "$(cat "$T/L3.rc")" != 0 ] && grep -q 'mutation did not apply' "$T/L3.out" && [ "$(cat "$PF2")" = 'no policy list here' ] && ! grep -qE '^start ' "$T/L3.log" && ok "L3 mutation that cannot apply -> rc != 0, file untouched, gradle never run" || ko "L3 non-applying mutation not refused"
unset WORKDIR

# --- scenario K2b/K2c: KVM readable-but-not-writable and writable-but-not-readable are both unusable (the check is "not readable OR not writable")
if [ "$(id -u)" = 0 ]; then echo "SKIP K2b/K2c (root ignores file modes)"; else
  : > "$T/kvm_ro"; chmod 444 "$T/kvm_ro"; : > "$T/kvm_wo"; chmod 222 "$T/kvm_wo"
  KVMX="$T/kvm_ro" runx K2b emulators --rows api34-phone; KVMX="$T/kvm_wo" runx K2c emulators --rows api34-phone
  for k in K2b K2c; do grep -q '"reason":"kvm_unusable' "$T/ev-$k/api34-phone/row.json" 2>/dev/null && ! grep -qE '^(run|create) ' "$T/$k.log" && ok "$k half-usable KVM -> SKIP kvm_unusable" || ko "$k half-usable KVM not refused"; done
  chmod 644 "$T/kvm_ro" "$T/kvm_wo"
fi
# --- scenario U: a failing gradle run is recorded and returned (exit.txt carries the REAL rc), and a pinned colliding name is refused by EVERY stage that creates a container
WORKDIR=$T/work2; export WORKDIR
FAKE_START_RC=1 runx U1 unit
[ "$(cat "$T/U1.rc")" != 0 ] && [ "$(cat "$T/ev-U1/unit/exit.txt")" = "gradle exit 1" ] && ok "U1 failing gradle -> rc != 0 and exit.txt 'gradle exit 1'" || ko "U1 failing gradle not recorded/returned (rc $(cat "$T/U1.rc"))"
seedctr U2 hrdiso-android-prep-pin1; HRD040_RUNID=pin1 runx U2 prepare
[ "$(cat "$T/U2.rc")" != 0 ] && ! grep -qE '^(create|run) ' "$T/U2.log" && grep -q 'already exists' "$T/U2.out" && ok "U2 prepare refuses a colliding name" || ko "U2 prepare reused/overwrote a colliding container"
seedctr U3 hrdiso-android-gradle-pin1; HRD040_RUNID=pin1 runx U3 unit
[ "$(cat "$T/U3.rc")" != 0 ] && ! grep -qE '^(create|run) ' "$T/U3.log" && grep -q 'already exists' "$T/U3.out" && ok "U3 build container refuses a colliding name" || ko "U3 build container reused a colliding container"
seedctr U4 hrdiso-android-seed-pin1; HRD040_RUNID=pin1 runx U4 seed
[ "$(cat "$T/U4.rc")" != 0 ] && ! grep -qE '^(create|run) ' "$T/U4.log" && ok "U4 seed refuses a colliding name" || ko "U4 seed reused a colliding container"
unset WORKDIR
[ $fail = 0 ] && echo "ALL PASS" || echo "SOME FAILED"; exit $fail
