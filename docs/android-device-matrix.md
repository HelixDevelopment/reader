# Android device matrix (HRD-040)

Status words (constitution 11.4.297(G)): the runner `tools/android_matrix/run_matrix.sh` was executed on one host (evidence below). Anything not
listed under "Results" is UNCONFIRMED. Isolation policy: `docs/test-isolation.md` (rootless podman only; no host display, no host adb, no sudo).

## What the matrix is

`qa/android_matrix/matrix.tsv`: one row = API level x image x screen (width, height, density) x class. Rows that share an image share one
emulator boot; the screen is switched with `wm size` / `wm density` (real framework display change, the activity is relaunched on it).
Rows: API 34 phone 1080x2400@420, API 34 small phone 720x1280@320, API 34 tablet 1600x2560@320, API 30 phone, API 36 phone.
`minSdk` is 24, `targetSdk` 36 (`clients/android/android/variables.gradle`): API 24..29 and 31..33, 35 are NOT in the matrix (UNCONFIRMED there).

Image pinning: API 34 = `ghcr.io/vasic-digital/lava-android-emulator@sha256:afbc7a1c...` (digest). API 30 and 36 are built locally by
`run_matrix.sh images` from the Containers submodule pin `d71b62058502` (`pkg/emulator/Containerfile`, `--build-arg API_LEVEL=<n>`); a local image
cannot carry a registry digest, so the evidence records its image ID (`image_api<N>.txt`). Images and the Gradle/SDK cache live in podman storage and
the named volume `hrdiso-android-cache`, never in the repo. The build tools/JDK images are pinned by digest inside the script.

## What runs where

| Stage | Command | Container | Network |
|---|---|---|---|
| web build + `cap sync` (generates `node_modules`, assets, cordova plugin dir) | `run_matrix.sh prepare` | node 22 (digest) | default (npm) |
| SDK seed / Gradle warm-up | `seed`, `warm` | builder / JDK 21 (digest) | default (SDK, Maven) |
| JVM unit tests (`:app:testDebugUnitTest`) incl. `ApiRequestPolicyTest` | `unit` | JDK 21 | none, `--offline` |
| unit-test mutant (POST added to the policy; tests MUST fail) | `mutation` | JDK 21 | none |
| APKs (debug + androidTest) | `apk` | JDK 21 | none |
| emulator rows: install, `am instrument`, screenshot, display facts | `emulators [--rows a,b]` | emulator image | none, `--device /dev/kvm` |

Emulator rows need `/dev/kvm` readable and writable by the user without root (ACL, verified on this host); otherwise the row is an honest SKIP
`kvm_unusable` (software emulation is not attempted). A missing image is a SKIP `image_absent`. No source is bind-mounted: sources go in with
`podman cp` from a `git archive` copy, so the container cannot write into the work tree.

Per row PASS needs all of: `am instrument` prints `OK (N tests)` with N >= 1 (classes `MainActivitySmokeTest`, `ExampleInstrumentedTest`); the
`HRDMATRIX` logcat line (written by the smoke test from inside the activity) shows the row's SDK, width and density exactly and a height within
10 percent below the configured one (system bars); the screenshot of the launched `MainActivity` has at least 8 distinct sampled colours.
`CronetLiveTest` is NOT run here: it needs the live h3 lab (`-e server https://...`, `qa/h3`).

## Containers submodule tooling: what was used and what was not

The matrix is driven by `tools/android_matrix/run_matrix.sh`, not by `submodules/containers/cmd/emulator-matrix`. Reason: that tool boots
AVDs through the host Android SDK and host adb (`AndroidSdkRoot`, `${AndroidSdkRoot}/platform-tools/adb`), which the isolation policy forbids; the
runner reuses the submodule's container recipe (Containerfile, entrypoint) instead. The `emulator-matrix` binary itself, `vm-matrix` and
`distributed-test` were NOT executed: UNCONFIRMED whether `emulator-matrix` can run its containerized mode end to end under this policy (follow-up).

## Reproduce

```bash
tools/android_matrix/run_matrix.sh seed && tools/android_matrix/run_matrix.sh images   # once (network)
tools/android_matrix/run_matrix.sh prepare && tools/android_matrix/run_matrix.sh warm   # once (network)
EVID=evidence/android-matrix/run-$(date -u +%Y%m%dT%H%M%SZ) tools/android_matrix/run_matrix.sh unit
tools/android_matrix/run_matrix.sh apk && tools/android_matrix/run_matrix.sh emulators
tools/isolation/seal_evidence.sh "$EVID"
```

## Results (one run, 2026-11-24, host with usable `/dev/kvm`)

Evidence (sealed, `SHA256SUMS` in each directory): `evidence/android-matrix/run-final/` (all rows, unit, mutant, APK hashes, runner log) and
`evidence/android-matrix/run-red/` (the RED run before the fix below).

| Check | Result |
|---|---|
| JVM unit tests, `--network none --offline` | 8 tests, 0 failures: `ApiRequestPolicyTest` 7/7 (first recorded execution of this class), `ExampleUnitTest` 1/1 |
| Paired mutant (POST accepted by `ApiRequestPolicy`) | killed: `ApiRequestPolicyTest` 1 of 7 fails, gradle exit 1 |
| api34-phone 1080x2400@420 | PASS: `OK (2 tests)`, display sdk=34 px=1080x2337 dpi=420, screenshot 464 colours |
| api34-small 720x1280@320 | PASS: `OK (2 tests)`, px=720x1232, dpi=320 |
| api34-tablet 1600x2560@320 | PASS: `OK (2 tests)`, px=1600x2440, dpi=320 |
| api30-phone 1080x2400@420 | PASS: `OK (2 tests)`, sdk=30 |
| api36-phone 1080x2400@420 | PASS: `OK (2 tests)`, sdk=36 |

The emulator ran with KVM (rootless, device ACL) in containers with `--network none`; no host display, no host adb, no sudo. The screenshot of
every row is the real WebView UI; it shows "Unexpected error" in the library card because no server is configured: this matrix checks that the
shell starts and lays out on each API/size, NOT library behaviour (covered by the web/Android live runs, HRD-022).

Finding (fixed here, RED in `run-red/api34-phone/instrument.txt`): `ExampleInstrumentedTest` (Capacitor scaffold) asserted package
`com.getcapacitor.app` while the app id is `org.helixreader.app`, so it failed on every device; it had never been run. The assertion now names the
real id. Registry: NOT yet registered. PR 33 allocates HRD-122 and HRD-123 and the registry requires gap-free ids, so this finding is to be
registered as HRD-124 (Bug, Low) after PR 33 lands (UNCONFIRMED until then); the fix itself is in this PR.

Not run / UNCONFIRMED: orientation changes, foldables, API 24..29/31..33/35, ARM images, Android TV, `CronetLiveTest` (needs the h3 lab), the
Containers `emulator-matrix` / `vm-matrix` / `distributed-test` binaries, behaviour on physical devices, flakiness (one run only), and the
Vision model (no screenshot judgement beyond the colour-count probe and my own reading of one screenshot).
