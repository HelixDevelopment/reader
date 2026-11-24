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
the named volume `hrdiso-android-cache`, never in the repo. The JDK 21 and node 22 images are pinned by registry digest inside the script; the SDK seed image `lava-android-builder` is a LOCAL-ONLY image (no registry
copy exists), pinned by its local manifest digest, and its id is recorded in `sdk_builder_image.txt` when `seed` runs.

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
10 percent below the configured one (system bars); the WebView fills the display (width equal, height >= 85 percent); the screenshot is exactly the row's
size and has at least 8 distinct sampled colours in the content region (top and bottom 8 percent, the system bars, are excluded); the package path and
sha256 read back from the device equal the APKs that were built (a stale install fails the row). A `unit` / `mutation` result is part of the verdict:
the JVM run must exit 0 and the mutant run must exit nonzero, otherwise `summarize.py` and the `all` stage exit 1.
`CronetLiveTest` is NOT run here: it needs the live h3 lab (`-e server https://...`, `qa/h3`).

## Containers submodule tooling: what was used and what was not

The matrix is driven by `tools/android_matrix/run_matrix.sh`, not by `submodules/containers/cmd/emulator-matrix`. Reason: that tool boots
AVDs through the host Android SDK and host adb (`AndroidSdkRoot`, `${AndroidSdkRoot}/platform-tools/adb`), which the isolation policy forbids; the
runner reuses the submodule's container recipe (Containerfile, entrypoint) instead. The `emulator-matrix` binary itself, `vm-matrix` and
`distributed-test` were NOT executed: UNCONFIRMED whether `emulator-matrix` can run its containerized mode end to end under this policy (follow-up).

## Reproduce

```bash
export EVID=evidence/android-matrix/run-$(date -u +%Y%m%dT%H%M%SZ); mkdir -p "$EVID/selftests"   # EVID is required by `seed` (it refuses without it)
tools/android_matrix/run_matrix.sh seed && tools/android_matrix/run_matrix.sh images   # once (network); seed writes sdk_builder_image.txt + sdk_packages.txt into $EVID
tools/android_matrix/run_matrix.sh warm                                                   # once (network)
tools/android_matrix/run_matrix.sh all 2> "$EVID/runner.log"   # prepare (network), unit, mutation (must fail), apk, emulators, summarize; exit 1 on any FAIL or when no row ran
bash tools/android_matrix/tests/test_lifecycle.sh > "$EVID/selftests/lifecycle_GREEN.txt"
bash tools/android_matrix/tests/test_lifecycle_mutants.sh > "$EVID/selftests/lifecycle_mutants_killed.txt"   # every mutant must FAIL test_lifecycle.sh
python3 -I tools/android_matrix/tests/test_summarize.py > "$EVID/selftests/summarize_GREEN.txt" 2>&1
tools/android_matrix/run_matrix.sh seal         # tools/isolation/seal_evidence.sh "$EVID" (writes SHA256SUMS)
```

Single stages (`unit`, `mutation`, `apk`, `emulators --rows a,b`) take the same `EVID`. The runner writes only publishable names into `$EVID`
(junit xml and the matrix are stored as `*.xml.txt` / `matrix.tsv.txt`, so `seal` accepts the directory without hand renaming). `apk.sha256.txt` uses
relative names; `source_rev.txt` is the git revision the sources were packed from at `prepare` time (`HRD040_REV`, default `HEAD`; resolved once and carried in `prepared/.source_rev`, so
separately run stages cannot record a different revision than the one built). `apk` refuses without a prepared tree.

## Results (one run, 2026-11-24, host with usable `/dev/kvm`)

Evidence (sealed, `SHA256SUMS` in each directory): `evidence/android-matrix/run-v2/` (all rows, unit, mutant, APK hashes, installed-package read-back,
runner log, runner/summarizer self-tests with their RED runs) and `evidence/android-matrix/run-red/` (the RED run before the fix below, older runner).
`runner.log` and `selftests/*` in the sealed runs contain absolute host paths (`/home/<user>/...`) as recorded; sealed evidence is frozen and is not edited, so that is a
recorded note, not a fix. `run-v2` records the SDK builder image only by the pinned digest in `run_matrix.sh`; it has no `sdk_builder_image.txt` / `sdk_packages.txt` (the
`seed` stage writes them from now on), so the seeded SDK in the cache volume is UNCONFIRMED for `run-v2`.
`evidence/android-matrix/run-final/` is the SUPERSEDED first run (earlier runner, no installed-hash read-back, absolute paths in `apk.sha256.txt`); it stays only because earlier commits cite it and evidence is frozen once recorded. Sealed manifest of `run-v2`: sha256 of its `SHA256SUMS` = `77958a73aa4750cbe16de4a725957fd0a3af7d27ebebd85b0d48396baa913903`. APKs built from source revision `ef369a58bf6fb29e7c883b21889dcc6036ddfce2`
(`source_rev.txt`); every row's installed `base.apk` sha256 equals the build (`installed.txt`).

| Check | Result |
|---|---|
| JVM unit tests, `--network none --offline` | gradle exit 0 (`unit/exit.txt`), junit in `unit/junit/*.xml.txt` (`ApiRequestPolicyTest`, `ExampleUnitTest`) |
| Paired mutant (POST accepted by `ApiRequestPolicy`) | killed: gradle exit 1 (`unit_mutant/exit.txt`) |
| api34-phone 1080x2400@420 | PASS: `OK (2 tests)`, display sdk=34 px=1080x2337, webview 1080x2274, 318 content colours |
| api34-small 720x1280@320 | PASS: px=720x1232, webview 720x1184, 309 colours |
| api34-tablet 1600x2560@320 | PASS: px=1600x2440, webview 1600x2392, 240 colours |
| api30-phone 1080x2400@420 | PASS: sdk=30, px=1080x2274, 318 colours |
| api36-phone 1080x2400@420 | PASS: sdk=36, px=1080x2400, 318 colours |

Self-tests of the runner and the summarizer (`tools/android_matrix/tests/`): `test_lifecycle.sh` (fake `podman`, no containers) and `test_summarize.py`.
Their RED runs against the previous runner/summarizer (b5b7112) and GREEN runs are in `run-v2/selftests/`; seven mutants of the new runner (no
`--timeout`, no lease-ts label, no reap call, no EXIT trap, fixed container name, no fresh APK copy, no installed-hash read-back) are each killed by a
named check (`lifecycle_mutants_7_of_7_killed.txt`). After the second review (finding R1) `tests/test_lifecycle_mutants.sh` runs 55 mutants (17 of the second round, then the reviewer mutants N7 / M10 / N13 / N14 / N16 / N19 / N20 and 31 further author mutants O01..O31; no-op cleanup, missing
lifetime args on build/prepare/emulator containers, no-op or non-refusing same-name check, timeout 0 or unvalidated, owner-pid 1, stale lease-ts, fixed name, no uninstall,
no reap, no fresh APK copy, no EXIT trap, ...) and requires each to FAIL `test_lifecycle.sh`; the equivalent mutants (M1b / N8 no TERM trap, N9 no INT trap, N12 mine() overwrite, N4 / N22 lease-ts) are listed there with the reasoning.

The emulator ran with KVM (rootless, device ACL) in containers with `--network none`; no host display, no host adb, no sudo. The screenshots show the
WebView UI with "Unexpected error" in the library card because no server is configured; that statement is the author's reading of one screenshot, while the pass/fail verdict of every row rests on the recorded machine checks (colour counts, sizes, hashes) and not on any
Vision-model judgement. This matrix checks that the shell starts and lays out on each API/size, NOT library behaviour (covered by the web/Android
live runs, HRD-022).

Finding (fixed here, RED in `run-red/api34-phone/instrument.txt`): `ExampleInstrumentedTest` (Capacitor scaffold) asserted package
`com.getcapacitor.app` while the app id is `org.helixreader.app`, so it failed on every device; it had never been run. The assertion now names the
real id. Registry: NOT yet registered. HRD-122 and HRD-123 have since landed (with main bd804e5); the registry requires gap-free ids, so this finding is still to be
registered as the next free id (HRD-124 at the time of writing, Bug, Low) through `scripts/workable_items.py add`; it is a follow-up, not done by this PR. The fix itself is in this PR.

Not run / UNCONFIRMED: orientation changes, foldables, API 24..29/31..33/35, ARM images, Android TV, `CronetLiveTest` (needs the h3 lab), the
Containers `emulator-matrix` / `vm-matrix` / `distributed-test` binaries, behaviour on physical devices, flakiness (one run only), and the
Vision model (no screenshot judgement beyond the colour-count probe and my own reading of one screenshot).

## Review round 4: convergence assessment (§11.4.276(E)) and scenario-isolation audit

Convergence assessment (STRUCTURAL round, fresh fixer instance, model Sonnet; effort not readable through the dispatch path). Recurring class, named: *a proving
test in `tools/android_matrix/tests/test_lifecycle.sh` lets a reviewer mutant survive because scenarios share state or the assertions do not discriminate* (round 2:
R1, the shared-state-free mutants; round 3: N7, scenario F passing on scenario A's leftover container). Round 2 and 3 fixed the reported instance; round 4 fixes the class: (a) every scenario now
has its OWN fake container store (`$T/s/<scenario>/ctr`) and its own log, evidence directory and hang pid file; (b) every runner function and every branch got a named assertion;
(c) the table of 55 mutants (17 earlier, 7 adopted reviewer mutants, 31 new O-series ones, one per function / branch / flag / argument) was run and all are killed.

Shared-state audit of the self-test (what each scenario shares with another, after the change):

| Scenario | Container store | Shared with another scenario | Verdict |
|---|---|---|---|
| A, B, B4, B5, C, C4, E*, F, F5, G, H*, I1, I2, K1..K3, K2b/c, M (reads A's log), Z | own `$T/s/<name>/ctr` | read-only inputs `$T/work/apk`, `$T/hostcache`, `$T/kvm`, `$T/reap.sh`; M reads A's log and evidence (read-only, after A finished) | isolated |
| D1 prepare -> D2 unit -> D3 apk (and U1..U4) | own store each | `$T/work2` deliberately: prepare produces the tree that unit and apk consume (a pipeline dependency, asserted by D2/D3 rc 0) | intended chain, no container shared |
| J1 prepare -> J2 apk, J3 | own store each | `$T/work5` (J1 -> J2, the pack-time revision handed on, the point of the test); J3 has an empty `$T/work6` | intended chain |
| L1..L3 | own store each | `$T/work7` prepared tree, restored by the stage itself and asserted | isolated |
| D0a, D0b | own store each | none | isolated |

Collision checks (N7): F seeds ONLY the exact colliding name (no prefix container in its store), so a prefix-only check cannot pass for the wrong reason; F5 is the inverse (only the
prefix container exists, the pinned name is free): it must NOT be refused. U2..U4 do the same for the prepare, build and seed containers.
Registry: the Android finding named above is still NOT registered (HRD-124 is the next free id); a follow-up, not part of this PR.
