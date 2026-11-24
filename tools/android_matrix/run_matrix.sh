#!/usr/bin/env bash
# run_matrix.sh - HRD-040 Android API-level x screen-size matrix, ONLY in rootless podman containers (no host display, no sudo, no adb on the host).
#   usage: run_matrix.sh images | seed | prepare | warm | unit | apk | emulators [--rows id,id] | all   (EVID=<dir> overrides the evidence dir)
# Design: no host bind mounts at all. Sources go in with `podman cp` (a git-free copy of clients/ made by `git archive`), results come out with `podman cp`;
# the Gradle/SDK cache is the named volume hrdiso-android-cache (outside the repo). Build containers have --network none except `prepare`/`seed`
# (the only stages that need the internet: npm and Maven/SDK downloads). Emulator containers: --network none, --device /dev/kvm only, adb runs INSIDE the container.
# Image pins: JDK/node by digest below; emulator images per qa/android_matrix/matrix.tsv. Containers submodule recipe: pin d71b62058502 (pkg/emulator/Containerfile).
set -eu
HERE=$(cd "$(dirname "$0")" && pwd); REPO=$(cd "$HERE/../.." && pwd)
JDK=docker.io/library/eclipse-temurin@sha256:4e38bcd64a552586921cdd011831cabc0d56c4f7c950aa460c0b9e03237871ed       # temurin 21 jdk jammy
NODE=docker.io/library/node@sha256:12f54cafa58d0e96554ae05871e29663ff7241836377dc9a7916449b8adb062c                 # node 22 alpine (Capacitor CLI needs >= 22)
SDKIMG=localhost/lava-android-builder:latest                                                                         # only used to seed the SDK (sdkmanager); id recorded
VOL=hrdiso-android-cache; CACHE_HOST=${HRD040_CACHE:-$HOME/.cache/hrd040-android}
STAMP=$(date -u +%Y%m%dT%H%M%SZ); EVID=${EVID:-$REPO/evidence/android-matrix/run-$STAMP}; WORK=${HRD040_WORK:-$HOME/.cache/claude-tmp/hrd040-work}
MATRIX=$REPO/qa/android_matrix/matrix.tsv; KVM=/dev/kvm
INSTR_CLASSES=org.helixreader.app.MainActivitySmokeTest,com.getcapacitor.myapp.ExampleInstrumentedTest
log() { echo "[android_matrix] $*" >&2; }
cname() { echo "hrdiso-android-$1"; }

# a build container: image, name, shell command ; $NET is --network none unless set to "" ; the cache volume is mounted
build_ctr() { local img=$1 n=$2 cmd=$3; podman rm -f "$n" >/dev/null 2>&1 || true
  podman create --name "$n" ${NET---network none} --userns=keep-id --cap-drop ALL --security-opt no-new-privileges --memory 6g --cpus 4 --pids-limit 2048 \
    -v "$VOL:/cache:rw" -e GRADLE_USER_HOME=/cache/gradle -e ANDROID_HOME=/cache/sdk -e ANDROID_SDK_ROOT=/cache/sdk -e HOME=/cache/home \
    --entrypoint sh "$img" -c "$cmd" >/dev/null; }

pack_src() { rm -rf "$WORK/src"; mkdir -p "$WORK/src" "$WORK/state"; git -C "$REPO" archive "${HRD040_REV:-HEAD}" clients | tar -x -C "$WORK/src"; }

stage_seed() { # one-time: SDK (builder image's SDK + platform 36) into the cache volume
  NET= build_ctr "$SDKIMG" hrdiso-android-seed 'mkdir -p /cache/home; [ -d /cache/sdk/platforms/android-36 ] || { cp -a /opt/android-sdk /cache/sdk && yes | /cache/sdk/cmdline-tools/latest/bin/sdkmanager --sdk_root=/cache/sdk "platforms;android-36" "build-tools;36.0.0" >/dev/null; }; ls /cache/sdk/platforms'
  podman start -a hrdiso-android-seed; podman rm hrdiso-android-seed >/dev/null; }

stage_prepare() { # npm ci + web build + cap sync (network): produces node_modules, web/dist, android assets and the generated cordova plugin dir
  pack_src
  podman rm -f hrdiso-android-prep >/dev/null 2>&1 || true
  podman create --name hrdiso-android-prep --userns=keep-id --cap-drop ALL --security-opt no-new-privileges --memory 3g --pids-limit 1024 -e HOME=/tmp --entrypoint sh "$NODE" -c \
    'set -e; cd /w/web && npm ci --ignore-scripts --no-audit --no-fund && npm run build && cd ../android && npm ci --ignore-scripts --no-audit --no-fund && npx cap sync android' >/dev/null
  podman cp "$WORK/src/clients/." hrdiso-android-prep:/w
  podman start -a hrdiso-android-prep; rm -rf "$WORK/prepared"; mkdir -p "$WORK/prepared"; podman cp hrdiso-android-prep:/w/. "$WORK/prepared"; podman rm hrdiso-android-prep >/dev/null
  log "prepared tree at $WORK/prepared (git-free, host-only)"; }

# run a gradle task set in the JDK21 container on the prepared tree, offline, no network; copy results out to $2
gradle_run() { local tasks=$1 out=$2; local n=hrdiso-android-gradle; mkdir -p "$out"
  NET=${GR_NET---network none} build_ctr "$JDK" $n "cd /w/android/android && echo sdk.dir=/cache/sdk > local.properties && sh gradlew ${GR_OFF---offline} --no-daemon --console=plain $tasks"
  podman cp "$WORK/prepared/." $n:/w; local rc=0; podman start -a $n > "$out/gradle.log" 2>&1 || rc=$?
  podman cp $n:/w/android/android/app/build/. "$out/build" 2>/dev/null || true; podman rm $n >/dev/null; return $rc; }

stage_unit() { local d=$EVID/unit; log "unit tests (JVM, offline, --network none)"
  gradle_run ':app:testDebugUnitTest --rerun-tasks' "$d" && echo "gradle exit 0" > "$d/exit.txt" || echo "gradle exit nonzero" > "$d/exit.txt"
  mkdir -p "$d/junit"; cp "$d"/build/test-results/testDebugUnitTest/*.xml "$d/junit/" 2>/dev/null || true; rm -rf "$d/build"; }

stage_mutation() { # paired mutant: ApiRequestPolicy accepts POST; the unit tests MUST fail (a green here would mean they prove nothing)
  local f=$WORK/prepared/android/android/app/src/main/java/org/helixreader/app/ApiRequestPolicy.java d=$EVID/unit_mutant; cp "$f" "$f.orig"
  sed -i 's/Arrays.asList("GET", "HEAD", "OPTIONS")/Arrays.asList("GET", "HEAD", "OPTIONS", "POST")/' "$f"; cmp -s "$f" "$f.orig" && { log "mutation did not apply"; return 1; }
  local rc=0; gradle_run ':app:testDebugUnitTest --rerun-tasks' "$d" || rc=$?; mv "$f.orig" "$f"; mkdir -p "$d/junit"; cp "$d"/build/test-results/testDebugUnitTest/*.xml "$d/junit/" 2>/dev/null || true; rm -rf "$d/build"
  echo "gradle exit $rc (nonzero expected)" > "$d/exit.txt"; [ "$rc" -ne 0 ]; }

stage_apk() { local d=$WORK/apk; log "assemble debug + androidTest APKs (offline)"
  gradle_run ':app:assembleDebug :app:assembleDebugAndroidTest' "$d"; cp "$d"/build/outputs/apk/debug/app-debug.apk "$d/app-debug.apk"; cp "$d"/build/outputs/apk/androidTest/debug/app-debug-androidTest.apk "$d/app-debug-androidTest.apk"
  rm -rf "$d/build"; sha256sum "$d"/*.apk > "$d/apk.sha256"; }

stage_images() { # build the extra-API emulator images from the Containers submodule pin (network at build time; recipe is the submodule's own Containerfile)
  local pin; pin=$(git -C "$REPO" ls-tree HEAD submodules/containers | awk '{print $3}'); local src=$CACHE_HOST/containers-src
  rm -rf "$src"; mkdir -p "$src"; git -C "$REPO/submodules/containers" archive "$pin" | tar -x -C "$src"
  for api in ${HRD040_APIS:-30 36}; do podman image exists localhost/hrd040-emu:api$api-x86_64 && continue
    (cd "$src" && podman build -q --build-arg API_LEVEL=$api --build-arg ABI=x86_64 -f pkg/emulator/Containerfile -t localhost/hrd040-emu:api$api-x86_64 .); done; }

wait_boot() { local n=$1 i r; for i in $(seq 1 90); do r=$(podman exec "$n" adb -s emulator-5554 shell getprop sys.boot_completed 2>/dev/null | tr -d '\r' || true); [ "$r" = 1 ] && return 0; sleep 10; done; return 1; }
adb() { local n=$1; shift; podman exec "$n" adb -s emulator-5554 "$@"; }

stage_emulators() { local only=${1:-}; mkdir -p "$EVID"; cp "$MATRIX" "$EVID/matrix.tsv"
  local imgs; imgs=$(grep -v '^#' "$MATRIX" | cut -f3 | awk '!s[$0]++')
  for img in $imgs; do
    local rowsfor; rowsfor=$(grep -v '^#' "$MATRIX" | awk -F'\t' -v i="$img" '$3==i')
    local api; api=$(echo "$rowsfor" | head -1 | cut -f2); local n; n=$(cname api$api)
    while IFS=$'\t' read -r id rapi rimg w h dpi cls; do [ -n "$only" ] && case ",$only," in *",$id,"*) ;; *) continue;; esac
      local rd=$EVID/$id; mkdir -p "$rd"; local base="\"row_id\":\"$id\",\"api\":$rapi,\"image\":\"$rimg\",\"width\":$w,\"height\":$h,\"dpi\":$dpi,\"class\":\"$cls\""
      if ! podman image exists "$img"; then echo "{$base,\"state\":\"SKIP\",\"reason\":\"image_absent: $img not on this host (build with run_matrix.sh images)\"}" > "$rd/row.json"; continue; fi
      if [ ! -r "$KVM" ] || [ ! -w "$KVM" ]; then echo "{$base,\"state\":\"SKIP\",\"reason\":\"kvm_unusable: $KVM not rw for this user; software emulation is not attempted (hours per boot)\"}" > "$rd/row.json"; continue; fi
      if ! podman container exists "$n"; then
        log "boot $n ($img)"; podman run -d --name "$n" --memory 5g --memory-swap 5g --cpus 4 --pids-limit 4096 --oom-score-adj 500 --device /dev/kvm --userns=keep-id \
          --cap-drop ALL --security-opt no-new-privileges --network none "$img" >/dev/null
        podman image inspect "$img" --format '{{.Id}} {{.Digest}}' > "$EVID/image_api$api.txt"
        if ! wait_boot "$n"; then echo "{$base,\"state\":\"BLOCKED\",\"reason\":\"emulator did not report sys.boot_completed within 900s\"}" > "$rd/row.json"; podman logs "$n" 2>&1 | tail -30 > "$rd/boot_tail.log"; podman rm -f "$n" >/dev/null; continue; fi
        adb "$n" shell 'settings put global window_animation_scale 0; settings put global transition_animation_scale 0; settings put global animator_duration_scale 0' >/dev/null
        podman cp "$WORK/apk/app-debug.apk" "$n:/tmp/app.apk"; podman cp "$WORK/apk/app-debug-androidTest.apk" "$n:/tmp/test.apk"; fi
      log "row $id: ${w}x${h}@$dpi"
      adb "$n" shell "wm size ${w}x${h}; wm density $dpi" >/dev/null; sleep 3
      adb "$n" shell pm uninstall org.helixreader.app.test >/dev/null 2>&1 || true; adb "$n" shell pm uninstall org.helixreader.app >/dev/null 2>&1 || true
      { adb "$n" install -r -t /tmp/app.apk; adb "$n" install -r -t /tmp/test.apk; } > "$rd/install.txt" 2>&1 || true
      adb "$n" shell 'logcat -c' || true
      adb "$n" shell "am instrument -w -r -e class $INSTR_CLASSES org.helixreader.app.test/androidx.test.runner.AndroidJUnitRunner" > "$rd/instrument.raw" 2>&1 || true
      tr -d '\r' < "$rd/instrument.raw" > "$rd/instrument.txt"; rm "$rd/instrument.raw"
      adb "$n" shell 'am start -W -n org.helixreader.app/.MainActivity' > "$rd/am_start.txt" 2>&1 || true; sleep 8
      podman exec "$n" sh -c 'adb -s emulator-5554 exec-out screencap -p' > "$rd/screen.png" 2>/dev/null || true
      adb "$n" shell 'logcat -d -s HRDMATRIX:I' 2>&1 | tr -d '\r' > "$rd/logcat_hrdmatrix.txt" || true
      adb "$n" shell 'getprop ro.build.version.sdk; getprop ro.build.version.release; getprop ro.product.cpu.abi; wm size; wm density' 2>&1 | tr -d '\r' > "$rd/device.txt" || true
      adb "$n" shell 'am force-stop org.helixreader.app; wm size reset; wm density reset' >/dev/null 2>&1 || true
      echo "{$base,\"state\":\"RAN\"}" > "$rd/row.json"
    done <<<"$rowsfor"
    podman rm -f "$n" >/dev/null 2>&1 || true
  done; }

case ${1:-} in
  images) stage_images;; seed) stage_seed;;
  warm) pack_src; [ -d "$WORK/prepared" ] || stage_prepare; GR_NET= GR_OFF= gradle_run ":app:assembleDebug :app:assembleDebugAndroidTest :app:testDebugUnitTest" "$WORK/warm";; prepare) stage_prepare;;
  mutation) mkdir -p "$EVID"; stage_mutation;;
  unit) mkdir -p "$EVID"; stage_unit;; apk) stage_apk;;
  emulators) shift; only=; [ "${1:-}" = --rows ] && only=$2; stage_emulators "$only"; python3 -I "$HERE/summarize.py" "$EVID";;
  all) mkdir -p "$EVID"; stage_prepare; stage_unit; stage_apk; stage_emulators ""; python3 -I "$HERE/summarize.py" "$EVID";;
  *) sed -n 2,4p "$0"; exit 2;;
esac
