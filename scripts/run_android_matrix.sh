#!/usr/bin/env bash
# run_android_matrix.sh - HRD-040 / GAP-004: feed the Helix Reader debug APK + the AVD matrix (tests/android_matrix/matrix_spec.json)
# to the Containers submodule's emulator-matrix, ONE invocation per API level (each API has its own emulator image), then merge the
# per-API attestations and verify them.
# usage: run_android_matrix.sh <apk> <evidence-dir>
# env:   TEST_CLASS (REQUIRED, fully-qualified instrumented test class), ANDROID_SDK_ROOT (must contain platform-tools/adb + platforms|system-images)
# The APK must have a build-step record <apk>.sha256 (written by the build, a different producer) that matches it.
# emulator-matrix flags used: --runner containerized --container-image <exact allow-listed image> --container-runtime podman
#   (verified against the real binary: `--runner container` is REJECTED by it; containerized requires --container-image).
# PRODUCTION vs TEST: MATRIX_TEST_MODE=1 is the ONLY way to use the test overrides MATRIX_IMAGES_FILE, MIN_FREE_GB, APK_BUILD_SHA_FILE,
#   MATRIX_PRINT_ARGV, MATRIX_EMULATOR_BIN (a fake emulator-matrix for pipeline tests); without it those variables are REFUSED (fail closed), and evidence made with it is stamped mode=TEST in
#   apk_binding.json, which scripts/verify_android_matrix.py never accepts as closure evidence.
# PREFLIGHT REFUSES (exit 78, full list, no evidence dir created). A real run is an emulator launch: §11.4.109/§11.4.76 operator approval
# plus the disk cleanup are prerequisites.
set -eu
APK=${1:?apk}; EV=${2:?evidence dir}; R=$(cd "$(dirname "$0")/.." && pwd); SPEC=$R/tests/android_matrix/matrix_spec.json
miss=(); MODE=PRODUCTION
if [ "${MATRIX_TEST_MODE:-}" = 1 ]; then MODE=TEST
else for v in MATRIX_IMAGES_FILE MIN_FREE_GB APK_BUILD_SHA_FILE MATRIX_PRINT_ARGV MATRIX_EMULATOR_BIN; do
  [ -z "${!v:-}" ] || miss+=("$v is a test-only override: refused without MATRIX_TEST_MODE=1"); done
fi
[ -f "$APK" ] || miss+=("APK not found: $APK")
BSF=$APK.sha256; [ "$MODE" = TEST ] && BSF=${APK_BUILD_SHA_FILE:-$APK.sha256}
if [ -f "$APK" ]; then
  [ -f "$BSF" ] || miss+=("build-step sha256 file missing: $BSF")
  if [ -f "$BSF" ] && [ "$(awk '{print tolower($1)}' "$BSF")" != "$(sha256sum "$APK" | awk '{print $1}')" ]; then miss+=("APK sha256 differs from the build step's record $BSF (tampered or stale APK)"); fi
  python3 -I -c "import zipfile,sys; z=zipfile.ZipFile(sys.argv[1]); sys.exit(1 if z.testzip() else 0)" "$APK" 2>/dev/null || miss+=("APK is not a valid zip")
fi
[[ "${TEST_CLASS:-}" =~ ^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)+$ ]] || miss+=("TEST_CLASS unset or not a fully-qualified class name")
SDK=${ANDROID_SDK_ROOT:-/nonexistent}
{ [ -x "$SDK/platform-tools/adb" ] && { [ -d "$SDK/platforms" ] || [ -d "$SDK/system-images" ]; }; } || miss+=("ANDROID_SDK_ROOT is not an Android SDK (needs executable platform-tools/adb and platforms/ or system-images/)")
{ [ -r /dev/kvm ] && [ -w /dev/kvm ]; } || miss+=("/dev/kvm absent or not read-write for this user")
MINGB=20; [ "$MODE" = TEST ] && MINGB=${MIN_FREE_GB:-20}
free_gb=$(df -Pk "$(dirname "$EV")" 2>/dev/null | awk 'NR==2{print int($4/1048576)}'); free_gb=${free_gb:-0}
[ "$free_gb" -ge "$MINGB" ] || miss+=("only ${free_gb} GiB free (need >= $MINGB; §12.9 disk headroom)")
[ -z "$(podman ps -q --filter name=hrdiso 2>/dev/null)" ] || miss+=("an hrdiso-* container is running (one container at a time)")
if [ "$MODE" = TEST ] && [ -n "${MATRIX_IMAGES_FILE:-}" ]; then imgs=$(cat "$MATRIX_IMAGES_FILE"); else imgs=$(podman images --format '{{.Repository}}:{{.Tag}}' 2>/dev/null || true); fi
PLAN=$(python3 -I -c "
import json,sys
s=json.load(open(sys.argv[1])); by={}
for a in s['avds']:
    n,api,ff=a.split(':'); by.setdefault(api,[]).append(a)
for api in sorted(by,key=int): print(api, s['images'][api], ','.join(by[api]))" "$SPEC")
while read -r api img avds; do
  printf '%s\n' "$imgs" | grep -Fxq -- "$img" || miss+=("allow-listed emulator image for API $api not present (exact name: $img)")
done <<< "$PLAN"
if [ ${#miss[@]} -gt 0 ]; then
  printf 'run_android_matrix: REFUSED, %d precondition(s) missing:\n' "${#miss[@]}" >&2; printf '  - %s\n' "${miss[@]}" >&2; exit 78
fi
PRE=$(sha256sum "$APK" | awk '{print $1}'); RUNAPK=$EV/apk-$PRE/app-under-test.apk
argv_for() { # api image avds evdir  -> ARGV
  ARGV=("$EMBIN" --android-sdk-root "$SDK" --apk "$RUNAPK" --test-class "$TEST_CLASS" --evidence-dir "$4" --avds "$3"
        --runner containerized --container-image "$2" --container-runtime podman --cold-boot --capture-screenshot-on-failure --concurrent 1)
}
EMBIN=${EV}/emulator-matrix
if [ "$MODE" = TEST ] && [ -n "${MATRIX_PRINT_ARGV:-}" ]; then   # tests: show the argv for the first API, run nothing
  read -r api img avds <<< "$(echo "$PLAN" | head -1)"; argv_for "$api" "$img" "$avds" "$EV/api$api"; printf '%s\n' "${ARGV[@]}"; exit 0
fi
mkdir -p "$EV"; mkdir -p "$(dirname "$RUNAPK")"; cp "$APK" "$RUNAPK"; chmod 0444 "$RUNAPK"
if [ "$MODE" = TEST ] && [ -n "${MATRIX_EMULATOR_BIN:-}" ]; then EMBIN=$MATRIX_EMULATOR_BIN
else ( cd "$R/submodules/containers" && go build -o "$EMBIN" ./cmd/emulator-matrix ); fi
while read -r api img avds; do
  argv_for "$api" "$img" "$avds" "$EV/api$api"; "${ARGV[@]}" || true
done <<< "$PLAN"
POST=$(sha256sum "$RUNAPK" | awk '{print $1}')
printf '{"mode":"%s","build_sha256":"%s","pre_run_sha256":"%s","post_run_sha256":"%s","run_apk":"%s"}\n' "$MODE" "$(awk '{print tolower($1)}' "$BSF")" "$PRE" "$POST" "${RUNAPK#"$EV"/}" > "$EV/apk_binding.json"
python3 -I "$R/scripts/merge_android_attestations.py" "$EV" || true
"$R/tools/isolation/seal_evidence.sh" "$EV" >/dev/null
python3 -I "$R/scripts/verify_android_matrix.py" --attestation "$EV/real-device-verification.json" --apk "$APK" --spec "$SPEC" \
  --test-class "$TEST_CLASS" --binding "$EV/apk_binding.json" --manifest "$EV/SHA256SUMS"
