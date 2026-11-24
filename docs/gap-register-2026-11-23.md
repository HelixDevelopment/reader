# Gap Register 2026-11-23

| Field | Value |
|---|---|
| Document | docs/gap-register-2026-11-23.md |
| Revision | 1.0 (2026-11-23) |
| Type | READ-ONLY audit; no code, registry DB or submodule changed |
| Governing anchors | 11.4.27, 11.4.169, 11.4.278, 11.4.49, 11.4.98, 11.4.266, 11.4.6 |
| Evidence root | evidence/campaign-2026-11-23/gap-audit/ (commands_raw.txt, commands_part2.txt, proven_checks.txt, gates_run.txt, commands.txt) |
| Honesty rule | Counts come from commands pasted in commands.txt. Root causes are UNCONFIRMED unless stated. |

## Contents
1. Authoritative test-type list
2. Surface and platform inventory
3. Coverage matrix
4. PROVEN-claim verification
5. Containers: capability vs use
6. HelixQA: capability vs use
7. Gap summary table and totals
8. Per-gap sections with workable-item drafts
9. Method and limits

## 1. Authoritative test-type list
Source: submodules/constitution/Constitution.md. Section 11.4.169 (line 10189) freezes a CLOSED 13-type set; section 11.4.27 (line 2656) names 3 more (scaling, ui, ux) in the verbatim mandate. Operator says '10+': the binding count is 16 plus cross-cutting rules.

| # | Type | Citation |
|---|---|---|
| 1 | Unit (only layer where mocks allowed) | 11.4.169(1), 11.4.27(A) |
| 2 | Integration (real infra via containers) | 11.4.169(2) |
| 3 | End-to-end | 11.4.169(3) |
| 4 | Full-automation (re-runnable, no manual step, N-iteration deterministic) | 11.4.169(4), 11.4.25, 11.4.98, 11.4.50 |
| 5 | Challenges (challenges submodule) | 11.4.169(5), 11.4.27 |
| 6 | HelixQA (banks + autonomous sessions) | 11.4.169(6), 11.4.27 |
| 7 | DDoS / load-flood | 11.4.169(7) |
| 8 | Security | 11.4.169(8) |
| 9 | Stress + Chaos | 11.4.169(9), 11.4.85 |
| 10 | Concurrency / atomicity | 11.4.169(10) |
| 11 | Race / deadlock | 11.4.169(11) |
| 12 | Memory (leak/soak/peak RSS) | 11.4.169(12) |
| 13 | Benchmark / performance | 11.4.169(13) |
| 14 | Scaling | 11.4.27 verbatim mandate |
| 15 | UI | 11.4.27 verbatim mandate |
| 16 | UX | 11.4.27 verbatim mandate |

Cross-cutting rules (not separate types but binding on every row): 11.4.278 (HelixQA full suites with Vision eyes over every screen, flow, edge case on every platform; Vision model validated first per 11.4.277); 11.4.49 (every feature test in BOTH UI-driven and intent/API variants); 11.4.117/160 (pixel/vision oracle); 11.4.154 (window-scoped capture); 11.4.239 (failure-path scenarios for critical-invariant code); 11.4.243-245 (characterization, contract tests, oracle-first); 11.4.248 (flaky quarantine + protected specs); 11.4.266 (claim-vs-reality ledger); 11.4.143 is scoped to video-streaming apps and is N/A here (UNCONFIRMED that no video playback surface will ship). 11.4.169 states no escape flag exists: a missing type is a release blocker unless honestly SKIP-justified.

## 2. Surface and platform inventory
Surfaces: S1 server HTTP/catalog/ingest/sources; S2 OPDS 1.2/2.0 + PSE; S3 KOSync; S4 LCP/LSD fulfilment; S5 core library (Go, 22 packages); S6 clients/web (React 19/Vite, 17 src files); S7 clients/android (Capacitor shell); S8 clients/desktop (Electron, 6 node test files); S9 demos/interactive_static_demo; S10 tools (firebase validator, helixtrack-sync, interop scripts, isolation guard).
Platforms (clients/supported-platforms.json, ran by script): web VERIFIED; android VERIFIED (emulator only); linux BUILT; windows, macos, ios, android-tv, harmonyos, aurora-os BLOCKED; e-ink NOT_STARTED (10 total).
Screen/device config axes required by directive vs exercised: viewport phone/tablet/desktop/wide x light/dark = exercised on web (16 checks x 4 viewports, evidence/ui/ui-results.json). Not exercised: RTL, large font, high contrast, reduced motion, density set, foldable, TV D-pad, TalkBack, Firefox/WebKit, physical devices.

## 3. Coverage matrix (surface x test type)
Legend: P=PROVEN (evidence path), Pa=PARTIAL, M=MISSING, B=BLOCKED. Server rows: S1 server+catalog, S2 OPDS/PSE, S3 KOSync, S4 LCP.

| Type | S1 server | S2 OPDS/PSE | S3 KOSync | S4 LCP | S5 core | S6 web | S7 android | S8 desktop | S9 demo | S10 tools |
|---|---|---|---|---|---|---|---|---|---|---|
| Unit | P go-test.log | P | P | P (lcp tests, review GREEN logs) | P 87.3% cov | Pa 5 test files, no coverage | M | P 15/15 node tests | M | P python tests |
| Integration | Pa live PG+Redis go-test-live-fixed.log; env-gated | Pa | P redis_live_test | Pa no PG, no real provider | Pa | Pa live server | Pa | Pa live_handler (no window) | M | Pa |
| E2E | P live-server evidence | P pse-spec, interop | P koreader client lua | Pa in-repo provider | n/a | P 67/67 evidence/clients/web | Pa 1 AVD manual adb | Pa gui:false | M | Pa |
| Full-automation | Pa needs env, no single runner | Pa | Pa | Pa | Pa | Pa verify_ui.py | M (hand-driven adb) | M | M | Pa |
| Challenges | M | M | M | M | M | M | M | M | M | M |
| HelixQA | Pa 32 PASS/34 | Pa | Pa | M | n/a | M (2 SKIP) | M | M | M | n/a |
| DDoS | Pa slow-client | Pa | M | M | n/a | M | M | M | M | n/a |
| Security | Pa review round fixes | Pa | Pa | Pa | Pa fuzz | M | M | Pa csp/allowlist/secrets tests | M | Pa |
| Stress/Chaos | Pa daemon-restart, backup-restore | M | M | M | M | M | M | M | M | n/a |
| Concurrency | P multi-instance, lock-scope | M | Pa | M | P hlc/sync | M | n/a | n/a | n/a | n/a |
| Race/deadlock | Pa go-race.log STALE (11-20) | Pa | Pa | M | Pa stale | n/a | n/a | n/a | n/a | n/a |
| Memory | M | M | M | M | M | M | M | M | M | M |
| Benchmark | Pa T2 floor FAIL | M | M | M | Pa 3 Benchmark funcs | M | M | M | M | M |
| Scaling | P 50k-work measurements | Pa | M | M | M | n/a | n/a | n/a | n/a | n/a |
| UI | n/a | n/a | n/a | n/a | n/a | P 4 viewports x2 themes | Pa 1 AVD | Pa | M | n/a |
| UX (flows, a11y, RTL, D-pad) | n/a | n/a | n/a | n/a | n/a | Pa | Pa | M | M | n/a |
| Vision read (11.4.278) | n/a | n/a | n/a | n/a | n/a | M | M | M | M | n/a |

Platform x config coverage: web phone/tablet/desktop/wide x light/dark P; android API34 portrait+landscape Pa; linux Pa (handler only); android-tv/windows/macos/ios/harmonyos/aurora B; e-ink M. Per-cell BLOCKED reasons and reproducers are in clients/supported-platforms.json and evidence/clients/blocked_reproducers.json.

## 4. PROVEN-claim verification (commands in proven_checks.txt)
| Claim | Check run | Result |
|---|---|---|
| Web 67/67 | sha256sum -c evidence/clients/web/MANIFEST.sha256; results.json | OK 26 files; passed 67 failed 0 |
| UI 4 viewports | evidence/ui/ui-results.json | 16 checks each, 16 PASS at phone/tablet/desktop/wide |
| UI mutation | evidence/ui-mutation/mutation-result.json | verifier_exit 1, viewport-meta fails at all 4 |
| Android run | sha256sum -c run-20261123T211340Z/MANIFEST.sha256 | OK 15; APK manifest OK 6; PNG, uidump, mp4, logcat present |
| HelixQA | helixqa/SHA256SUMS; green-1/2.json; red-mutant.json | OK 17; green 34/32/0/2 twice (exit 0); mutant 34/23/9/2 exit 1 |
| Firebase | SHA256SUMS | OK 6 |
| OPDS-LCP | evidence.sha256 | OK 7 |
| Desktop packaged artifacts | artifacts.sha256 | FAIL: .deb and .tar.gz missing (GAP-007) |
Caveat: sha256 proves integrity, not that the run was correct; screenshots were not re-read by Vision (GAP-003).

## 5. Containers: capability vs use
Inspected: submodules/containers (pin 22b4970, 25 commits behind main). Packages: emulator (Containerized boot of AVD in rootless podman with KVM, Install, RunInstrumentation, CaptureScreenshot, Matrix with API level/form factor/NetworkProfile/screenshot-on-failure/concurrency, canary, cleanup), cuttlefish, genymotion, applesim, vm (qemu, macos, ios, matrix), boot, compose, orchestrator, remote (ssh executor, GPU probe, host manager), remoteexec, distribution (failover), network (port allocator, overlay, tunnel), policy, health, lifecycle. CLIs: emulator-matrix, vm-matrix, distributed-test, distributed-build, emulator-canary, emulator-cleanup, applesim, genymotion, boot, deploy-stack.
USED by this repo: podman rootless images for PG/Redis; one Android emulator image via Containers section-6.X recipe (evidence/clients/android). NOT used: emulator-matrix, vm-matrix, distributed-test, applesim, genymotion, cuttlefish, remote/distribution, network chaos, compose orchestrator. Extensions needed: (a) repo script feeding APK+AVD matrix (GAP-004); (b) tv/tablet/foldable/density AVD definitions; (c) Windows/macOS via vm-matrix or remote Mac (GAP-006); (d) headless-display desktop target for Electron (GAP-007); (e) netem profiles for DDoS/chaos (GAP-032); (f) browser matrix container (GAP-026).

## 6. HelixQA: capability vs use
Inspected: submodules/helixqa (pin b16a089, on main). CLI subcommands in cmd/helixqa/main.go: run, autonomous, http, list, report, replay (plus 40+ helper cmds: x11grab, kmsgrab, axtree-windows/darwin, uitars, omniparser, verify-vision/tesseract/whisper, verify-coder-ddos/race/memory/chaos/bench/concurrency, recording-analyzer). Executors in pkg/autonomous/executor_factory.go: android/androidtv (ADB), web (Playwright/CDP), desktop (X11); HTTP executor drives API banks. 90+ bank files in banks/ (ours: 6 YAML in qa/helixqa/banks, 34 cases). `autonomous` mode takes --platforms android,desktop,web, curiosity mode, ticket generation.
USED: `helixqa run` with HTTP banks (auth, catalog, OPDS, KOSync, health). NOT USED: web (Playwright) executor (GAP-002), android ADB executor against our APK, desktop executor, `autonomous` curiosity sessions, vision/recordingqa/frame bridge, verify-coder-* suites for ddos/race/memory/bench/chaos/concurrency (these map to types 7,9,10,11,12,13 and are unexercised), replay, ticket pipeline. Extensions needed: CDP flag on `run` (UNCONFIRMED absent), LCP + PSE banks, a11y/RTL/large-font axes, banks for Android/desktop screens, vision validation per 11.4.277.

## 7. Gap summary

Totals by severity: Critical 4, High 12, Medium 20, Low 4 (total 40).  
Totals by category: Build 1, Docs 2, Governance 2, HelixQA/Vision 2, Performance 1, Platform 9, Process 2, Product 2, Quality 2, Security 3, Server 1, Test 13.

| ID | Sev | Category | Type | Title |
|---|---|---|---|---|
| GAP-001 | Critical | Process | Task | Entire clients/ tree (web, android, desktop) is untracked in git |
| GAP-002 | Critical | HelixQA/Vision | Feature | Web UI has no running HelixQA bank: HLX-WEB-001/002 are SKIP (PLAYWRIGHT-RUNTIME-PENDING) |
| GAP-003 | Critical | HelixQA/Vision | Task | Vision ('eyes') layer never executed; no §11.4.277 llmctl known-answer validation record |
| GAP-004 | Critical | Platform | Feature | No automated multi-device/API-level/screen-size matrix: Containers emulator-matrix, vm-matrix, distributed-test unused by this repo |
| GAP-005 | High | Platform | Feature | Android TV (D-pad/leanback) surface and TalkBack/a11y never exercised |
| GAP-006 | High | Platform | Feature | Windows and macOS desktop and iOS are BLOCKED with no use of Containers vm-matrix / applesim / remote executors |
| GAP-007 | High | Platform | Feature | Electron desktop: no GUI/window run; packaged artifacts listed in artifacts.sha256 are missing from evidence |
| GAP-008 | High | Product | Feature | Web reader renders plain text only; EPUB/PDF/CBZ not rendered (server exposes no text) |
| GAP-009 | High | Server | Bug | Remote (webdav/sftp) sources answer Range with 200 full body, no Accept-Ranges; no PSE link for remote CBZ (HRD-036) |
| GAP-010 | High | Performance | Bug | Write-path throughput floor missed: MergeWork 152-176 ops/s vs 271.94 ops/s floor |
| GAP-011 | High | Security | Task | LCP never exercised against a real provider or third-party reader |
| GAP-012 | High | Security | Task | No Challenges-submodule or Security-submodule use in repo code, gates or scripts |
| GAP-013 | High | Test | Task | Memory / leak soak test type absent |
| GAP-014 | High | Test | Task | Race/deadlock evidence is stale: go-race.log dated 2026-11-20, before LCP/OPDS/PSE/LSD code |
| GAP-015 | High | Test | Task | 57 real t.Skip/Skipf calls (the earlier grep of 83 over-counted report.Skipped); 38 are gated on HELIX_* env vars not set in any standing suite (PG DSN 18, LOAD_RUN 11, REDIS 7, others) |
| GAP-017 | High | Governance | Task | helix-deps.yaml drift against gitlinks (4 DRIFT: llmctl, containers, claude_toolkit, llama.cpp) |
| GAP-016 | Medium | Test | Task | No claim-vs-reality ledger (§11.4.266) and no gate-evidence manifest (§11.4.262) |
| GAP-018 | Medium | Governance | Task | Submodule pins not on main: doc_processor diverged; llm_orchestrator/llm_provider/vision_engine pins ahead of main; containers 25 commits behind |
| GAP-019 | Medium | Build | Task | helixqa was built in a scratch tree outside the repo; several deps are depth-1 scratch clones not submodules |
| GAP-020 | Medium | Quality | Task | gofmt drift in core (hdm/hdm.go, hlc/hlc.go, pkg/crypto/crypto.go, pkg/source/source.go, source_test.go) |
| GAP-021 | Medium | Quality | Task | Header oracles missing (CSP, X-Content-Type-Options, HSTS, Cache-Control) -- only 1 non-test occurrence in server |
| GAP-022 | Medium | Test | Task | Postgres paths unrun for LCP/GC refs |
| GAP-023 | Medium | Product | Task | ID column still named atm_id in SQLite registry and exports |
| GAP-024 | Medium | Docs | Task | Documentation drift: stale 'not implemented' / 'not yet' claims |
| GAP-025 | Medium | Test | Task | Mutation coverage is hand-listed (110 mutants) and spans server/core only; clients and tools have 1 mutation each |
| GAP-026 | Medium | Platform | Feature | No cross-browser run (Chromium only); no Firefox/WebKit viewport runs |
| GAP-027 | Medium | Platform | Feature | Device/screen configurations unexercised: RTL, large font, high contrast, reduced motion, density, foldable, landscape on web |
| GAP-028 | Medium | Platform | Feature | e-ink profile NOT_STARTED; HarmonyOS and Aurora BLOCKED by missing SDKs |
| GAP-029 | Medium | Test | Task | Web unit coverage small: 5 test files for 17 source files; no coverage number |
| GAP-030 | Medium | Test | Task | Server coverage 60.1% without live services; no combined live coverage number |
| GAP-031 | Medium | Test | Task | Interop with real devices unproven: KOReader firmware, Kobo, physical Android phone/tablet |
| GAP-032 | Medium | Test | Task | DDoS/slow-loris/flood and chaos coverage is campaign-run, not standing |
| GAP-033 | Medium | Security | Task | HelixTrack Core defects D1-D5 (default JWT secret, password logging) drafted but not filed; local instance patched with schema SQL |
| GAP-034 | Medium | Platform | Feature | Interactive demo has no platform-matrix automation or HelixQA bank |
| GAP-035 | Medium | Test | Task | Fuzz coverage: 11 Fuzz funcs but none for OPDS XML/JSON generation, LCP license JSON, KOSync body parsers (UNCONFIRMED per file) |
| GAP-036 | Medium | Process | Task | No local CI-less gate runner executes the constitution full suite for this repo; only individually-run gates |
| GAP-037 | Low | Platform | Task | Android build is debug-only; no release/signing/ProGuard or tablet/Wear check |
| GAP-038 | Low | Test | Task | Challenges for mixed Postgres vs in-memory catalog: web/android evidence uses in-memory catalog only |
| GAP-039 | Low | Test | Task | Flaky/quarantine registry absent (§11.4.248); needle tests self-skip when needle does not fire |
| GAP-040 | Low | Docs | Task | Workable-item statuses lag reality: HRD-031/025/026/029 still Queued/In progress while evidence entries claim delivery |

## 8. Per-gap sections

### GAP-001 - Entire clients/ tree (web, android, desktop) is untracked in git
- Severity: Critical; Category: Process; Owner suggestion: operator-gated
- Evidence: git status: '?? clients/'; `git ls-files clients | wc -l` = 0 (commands_part2.txt)
- Root cause (UNCONFIRMED unless stated): UNCONFIRMED: work done in session never committed (operator holds commits per memory note).
- Proposed fix: Commit clients/ (excluding node_modules/dist/out per .gitignore) on main after independent review; add a gate that fails when a supported-platforms.json VERIFIED platform has untracked sources.
- Required test types: unit (gate), full-automation
- Workable-item draft: Type Task; Title: "Entire clients/ tree (web, android, desktop) is untracked in git"; Description: see Evidence and Root cause above; Acceptance: git ls-files clients lists sources; gate GREEN with RED baseline on an untracked tree.

### GAP-002 - Web UI has no running HelixQA bank: HLX-WEB-001/002 are SKIP (PLAYWRIGHT-RUNTIME-PENDING)
- Severity: Critical; Category: HelixQA/Vision; Owner suggestion: operator+helixqa upstream
- Evidence: green-1.json: 34 cases, 32 PASS, 2 SKIP (HLX-WEB-001/002); qa/helixqa/banks/helix-web.yaml lines 19-29 say 'executor is a stub'
- Root cause (UNCONFIRMED unless stated): Stale premise: submodules/helixqa/pkg/autonomous/playwright_executor.go implements a CDP-driven PlaywrightExecutor (wraps userflow.PlaywrightCLIAdapter) and is skipped only when PipelineConfig.PlaywrightCDPURL is empty. UNCONFIRMED whether `helixqa run` (cmd/helixqa/main.go) exposes a CDP flag: grep for 'cdp' in main.go returned nothing, only --browser-url/--desktop-process.
- Proposed fix: Start a Chromium with --remote-debugging-port in a rootless container (Containers), wire CDP URL (extend `helixqa run` if the flag is absent, upstream PR), remove _skip from the bank, add RED mutant for the web UI.
- Required test types: helixqa, ui, ux, e2e, vision
- Workable-item draft: Type Feature; Title: "Web UI has no running HelixQA bank: HLX-WEB-001/002 are SKIP (PLAYWRIGHT-RUNTIME-PENDING)"; Description: see Evidence and Root cause above; Acceptance: helixqa run over helix-web.yaml reports 0 SKIP, 2 PASS with screenshots per step; mutating a UI label turns it RED.

### GAP-003 - Vision ('eyes') layer never executed; no §11.4.277 llmctl known-answer validation record
- Severity: Critical; Category: HelixQA/Vision; Owner suggestion: operator+qa
- Evidence: No vision/OCR artifacts under evidence/ (grep 'vision' in evidence index = none produced by us); EVIDENCE.md lists no llmctl validation; PROVENANCE.txt shows vision_engine/llm_provider only as scratch clones
- Root cause (UNCONFIRMED unless stated): Vision pipeline not wired into any run; §11.4.277 gate unfulfilled. UNCONFIRMED which Vision model/endpoint is available on this host.
- Proposed fix: Run llmctl known-answer validation for the chosen model, then read every captured screenshot (web 4 viewports, android run) through helixqa vision/recordingqa and store verdict JSON beside the PNG.
- Required test types: vision, ui, ux, helixqa
- Workable-item draft: Type Task; Title: "Vision ('eyes') layer never executed; no §11.4.277 llmctl known-answer validation record"; Description: see Evidence and Root cause above; Acceptance: Per-screenshot vision verdict JSON with sha256 manifest; a deliberately wrong-screen mutant is rejected.

### GAP-004 - No automated multi-device/API-level/screen-size matrix: Containers emulator-matrix, vm-matrix, distributed-test unused by this repo
- Severity: Critical; Category: Platform; Owner suggestion: qa+containers
- Evidence: grep for emulator-matrix|vm-matrix|distributed-test|applesim outside submodules/: 0 hits; Android evidence is ONE AVD (API 34, 1080x1920 + one landscape) run by hand-driven adb
- Root cause (UNCONFIRMED unless stated): Containers capability present (pkg/emulator Matrix with AVD{APILevel,FormFactor}, NetworkProfile, CaptureScreenshotOnFailure, Concurrent; cmd/emulator-matrix) but no repo script feeds it an APK + AVD list.
- Proposed fix: Add scripts/run_android_matrix.sh invoking cmd/emulator-matrix with API 24/28/30/34/36 x phone/tablet(7in,10in)/foldable x density mdpi..xxxhdpi x locale(en,ar RTL) x font-scale 1.3/2.0 x dark/light; attest per-row JSON.
- Required test types: e2e, ui, ux, full-automation, helixqa, stress
- Workable-item draft: Type Feature; Title: "No automated multi-device/API-level/screen-size matrix: Containers emulator-matrix, vm-matrix, distributed-test unused by this repo"; Description: see Evidence and Root cause above; Acceptance: Attestation JSON with >=12 AVD rows, each boot_completed + test PASS, sha256 manifest, RED row for a broken APK.

### GAP-005 - Android TV (D-pad/leanback) surface and TalkBack/a11y never exercised
- Severity: High; Category: Platform; Owner suggestion: qa
- Evidence: supported-platforms.json android-tv BLOCKED ('an android-tv system image is installed but no AVD uses it'); no TalkBack or D-pad key events in e2e/live.mjs or verify_ui.py (grep: 0 rtl hits)
- Root cause (UNCONFIRMED unless stated): Reason given is an environment gap, not a hard block: Containers emulator supports FormFactor 'tv'. UNCONFIRMED that web client has focus-order/D-pad handling at all.
- Proposed fix: Create tv AVD via emulator-matrix, add D-pad navigation bank (DPAD_UP/DOWN/CENTER), uiautomator focus assertions; add axe-core + TalkBack content-description checks.
- Required test types: ui, ux, e2e, a11y, helixqa
- Workable-item draft: Type Feature; Title: "Android TV (D-pad/leanback) surface and TalkBack/a11y never exercised"; Description: see Evidence and Root cause above; Acceptance: TV AVD run: every screen reachable by D-pad only, focus never lost; axe-core 0 serious violations on 4 viewports.

### GAP-006 - Windows and macOS desktop and iOS are BLOCKED with no use of Containers vm-matrix / applesim / remote executors
- Severity: High; Category: Platform; Owner suggestion: operator decision (hardware/licence)
- Evidence: supported-platforms.json: windows, macos, ios BLOCKED (host Linux); Containers has pkg/vm (qemu, macos, ios subdirs), pkg/applesim, pkg/remote (ssh executor), pkg/distribution
- Root cause (UNCONFIRMED unless stated): Block reasons assume the operator host only. UNCONFIRMED: qemu Windows guest image availability and licensing; macOS requires Apple hardware (pkg/remote to a Mac host or applesim on one).
- Proposed fix: Evaluate vm-matrix with a Windows guest (licensed image) for Electron NSIS build+smoke; register a remote Mac builder via pkg/remote for macOS/iOS; keep BLOCKED rows with reproducers until proven.
- Required test types: e2e, ui, full-automation, integration
- Workable-item draft: Type Feature; Title: "Windows and macOS desktop and iOS are BLOCKED with no use of Containers vm-matrix / applesim / remote executors"; Description: see Evidence and Root cause above; Acceptance: Either a Windows VM run with install+launch+screenshot evidence, or an operator-signed WON'T-FIX per §11.4.112 with reproducer.

### GAP-007 - Electron desktop: no GUI/window run; packaged artifacts listed in artifacts.sha256 are missing from evidence
- Severity: High; Category: Platform; Owner suggestion: qa
- Evidence: evidence/clients/desktop/artifacts.sha256 -> sha256sum -c: .deb and .tar.gz 'FAILED open or read' (files live in git-ignored clients/desktop/out); live_handler.json states gui:false 'no Electron window'; supported-platforms.json says linux=BUILT (not VERIFIED)
- Root cause (UNCONFIRMED unless stated): Evidence points at build output outside the evidence tree; no display-less GUI path (xvfb/Wayland-headless in a container) was used.
- Proposed fix: Run packaged app under xvfb inside a rootless container, drive with helixqa desktop executor (x11grab), copy artifacts + hashes into evidence; do not use the operator display.
- Required test types: e2e, ui, ux, helixqa, vision
- Workable-item draft: Type Feature; Title: "Electron desktop: no GUI/window run; packaged artifacts listed in artifacts.sha256 are missing from evidence"; Description: see Evidence and Root cause above; Acceptance: Manifest verifies; screenshot of library window with titles from live server; RED when proxy broken.

### GAP-008 - Web reader renders plain text only; EPUB/PDF/CBZ not rendered (server exposes no text)
- Severity: High; Category: Product; Owner suggestion: server+web
- Evidence: supported-platforms.json web notes; evidence screenshots *reader-epub-unsupported.png; docs/formats.md render=NOT IMPLEMENTED for all formats
- Root cause (UNCONFIRMED unless stated): Design gap: server has no content-extraction endpoint; HDM/pagination exists in core but is not exposed over HTTP.
- Proposed fix: Expose /v1/files/:id/text or chapter/page endpoints from core HDM; web reader consumes; CBZ via PSE page images.
- Required test types: unit, integration, e2e, ui, ux, security, benchmark
- Workable-item draft: Type Feature; Title: "Web reader renders plain text only; EPUB/PDF/CBZ not rendered (server exposes no text)"; Description: see Evidence and Root cause above; Acceptance: EPUB and CBZ open in reader on 4 viewports with captured screenshots + DOM text asserted.

### GAP-009 - Remote (webdav/sftp) sources answer Range with 200 full body, no Accept-Ranges; no PSE link for remote CBZ (HRD-036)
- Severity: High; Category: Server; Owner suggestion: server
- Evidence: docs/Issues.md HRD-036; evidence/campaign-2026-11-23/interop/hrd026/providers_verify.txt
- Root cause (UNCONFIRMED unless stated): UNCONFIRMED: remote reader returns io.Reader without Seeker so the httpapi content handler cannot serve ranges; PSE link gated on local path. Not investigated.
- Proposed fix: Root-cause first (§11.4.102): reproduce RED, then implement ranged remote reads (rclone/sftp offset) or bounded cache; PSE eligibility from edition format not locality.
- Required test types: unit, integration, e2e, helixqa, stress, security
- Workable-item draft: Type Bug; Title: "Remote (webdav/sftp) sources answer Range with 200 full body, no Accept-Ranges; no PSE link for remote CBZ (HRD-036)"; Description: see Evidence and Root cause above; Acceptance: Ranged GET on webdav+sftp gives 206 with correct Content-Range, byte-identical; remote CBZ feed carries PSE link; RED baseline kept.

### GAP-010 - Write-path throughput floor missed: MergeWork 152-176 ops/s vs 271.94 ops/s floor
- Severity: High; Category: Performance; Owner suggestion: server
- Evidence: docs/scalability-review.md lines 63-67 (host load ~17 on 16 cores) and line 429 (271.94 baseline)
- Root cause (UNCONFIRMED unless stated): UNCONFIRMED: host contention (load 17) vs genuine regression after set-based upsert; floor was recorded single-worker.
- Proposed fix: Re-measure on quiesced host with control arm in same run (control_needle), fix measurement protocol or code; do not weaken floor silently (§11.4.120).
- Required test types: benchmark, stress, scaling, concurrency
- Workable-item draft: Type Bug; Title: "Write-path throughput floor missed: MergeWork 152-176 ops/s vs 271.94 ops/s floor"; Description: see Evidence and Root cause above; Acceptance: Same-run A/B shows >=floor or a documented, operator-ratified revised SLO.

### GAP-011 - LCP never exercised against a real provider or third-party reader
- Severity: High; Category: Security; Owner suggestion: server
- Evidence: EVIDENCE.md lines 817,839,875 NOT PROVEN/UNCONFIRMED; lsd provider written in-repo from same spec text
- Root cause (UNCONFIRMED unless stated): Self-referential test oracle (§11.4.245): producer and consumer share an author.
- Proposed fix: Stand up Readium lcpserver (or EDRLab test provider) in a container as independent oracle; open produced artifact with Thorium/readium-lcp-client-in-container.
- Required test types: integration, security, e2e, interop
- Workable-item draft: Type Task; Title: "LCP never exercised against a real provider or third-party reader"; Description: see Evidence and Root cause above; Acceptance: Independent reader decrypts artifact; provider LSD register/return/renew round-trip logged; RED with wrong passphrase.

### GAP-012 - No Challenges-submodule or Security-submodule use in repo code, gates or scripts
- Severity: High; Category: Security; Owner suggestion: qa
- Evidence: grep for challenges/security imports in server core scripts qa tools: 0 matches; submodules/challenges and submodules/security are pinned only
- Root cause (UNCONFIRMED unless stated): Submodules vendored but not integrated; §11.4.169 types 5 and 8 depend on them.
- Proposed fix: Create Challenges bank wrapping existing live tests (auth, OPDS, KOSync, LCP); run security submodule scanners (gosec/dep CVE) in container.
- Required test types: challenges, security
- Workable-item draft: Type Task; Title: "No Challenges-submodule or Security-submodule use in repo code, gates or scripts"; Description: see Evidence and Root cause above; Acceptance: challenge run JSON with PASS only on captured evidence; security scan report with zero unaddressed High.

### GAP-013 - Memory / leak soak test type absent
- Severity: High; Category: Test; Owner suggestion: server+qa
- Evidence: grep Benchmark: 3 funcs; no soak/pprof/RSS artifacts under evidence/ (ls evidence)
- Root cause (UNCONFIRMED unless stated): Never authored.
- Proposed fix: Add 24h/N-iteration soak under Containers with pprof heap diff and RSS ceilings; web (heap snapshot) and Android (dumpsys meminfo).
- Required test types: memory, stress
- Workable-item draft: Type Task; Title: "Memory / leak soak test type absent"; Description: see Evidence and Root cause above; Acceptance: Soak JSON: RSS slope ~0, heap diff bounded; RED with injected leak.

### GAP-014 - Race/deadlock evidence is stale: go-race.log dated 2026-11-20, before LCP/OPDS/PSE/LSD code
- Severity: High; Category: Test; Owner suggestion: server
- Evidence: evidence/campaign-2026-11-20/{core,server}/go-race.log; git status shows new core/pkg/lcp/*.go untracked
- Root cause (UNCONFIRMED unless stated): Not re-run after later changes.
- Proposed fix: Re-run `go test -race ./...` core+server with live PG/Redis; store under campaign-2026-11-23.
- Required test types: race, concurrency
- Workable-item draft: Type Task; Title: "Race/deadlock evidence is stale: go-race.log dated 2026-11-20, before LCP/OPDS/PSE/LSD code"; Description: see Evidence and Root cause above; Acceptance: go-race.log exit 0 on current HEAD+worktree, sha256 recorded.

### GAP-015 - 57 real t.Skip/Skipf calls (the earlier grep of 83 over-counted report.Skipped); 38 are gated on HELIX_* env vars not set in any standing suite (PG DSN 18, LOAD_RUN 11, REDIS 7, others)
- Severity: High; Category: Test; Owner suggestion: qa
- Evidence: commands.txt: `grep -rn t.Skipf\?( server core` = 57, 38 with HELIX_ gates
- Root cause (UNCONFIRMED unless stated): Live tests are opt-in; no single runner proves all gated tests execute (0 skips) in one pass.
- Proposed fix: Add scripts/run_all_live.sh that boots PG+Redis via Containers, exports vars, runs with -v, fails if any gated test SKIPs.
- Required test types: full-automation, integration
- Workable-item draft: Type Task; Title: "57 real t.Skip/Skipf calls (the earlier grep of 83 over-counted report.Skipped); 38 are gated on HELIX_* env vars not set in any standing suite (PG DSN 18, LOAD_RUN 11, REDIS 7, others)"; Description: see Evidence and Root cause above; Acceptance: Run log: 0 SKIP for gated tests; skip-count gate with RED baseline.

### GAP-017 - helix-deps.yaml drift against gitlinks (4 DRIFT: llmctl, containers, claude_toolkit, llama.cpp)
- Severity: High; Category: Governance; Owner suggestion: operator
- Evidence: gates_run.txt: CM-HELIX-DEPS-REF-MATCHES-GITLINK FAIL (27 deps, 4 DRIFT); root has gitlink submodules/claude_toolkit without .gitmodules mapping (git submodule status --recursive fatal)
- Root cause (UNCONFIRMED unless stated): UNCONFIRMED whether drift is root manifest or nested constitution manifests; claude_toolkit gitlink unmapped is documented in docs/upstream-issues/claude_toolkit-challenges-gitlink.md.
- Proposed fix: Operator-gated: refresh manifests or pins; fix .gitmodules mapping via fetch+merge on main (no force).
- Required test types: full-automation
- Workable-item draft: Type Task; Title: "helix-deps.yaml drift against gitlinks (4 DRIFT: llmctl, containers, claude_toolkit, llama.cpp)"; Description: see Evidence and Root cause above; Acceptance: Gate exit 0 on real tree.

### GAP-016 - No claim-vs-reality ledger (§11.4.266) and no gate-evidence manifest (§11.4.262)
- Severity: Medium; Category: Test; Owner suggestion: qa
- Evidence: Gates cm_ledger_row_typed... and cm_machine_evidence... PASS only vacuously: docs/claim_reality_ledger.tsv and docs/evidence/gate_evidence_manifest.tsv absent (gates_run.txt)
- Root cause (UNCONFIRMED unless stated): Never created; vacuous PASS masks the gap (§11.4.201).
- Proposed fix: Create ledger rows for every advertised capability (formats matrix, platforms, OPDS, KOSync, LCP) typed from closed vocabulary.
- Required test types: full-automation
- Workable-item draft: Type Task; Title: "No claim-vs-reality ledger (§11.4.266) and no gate-evidence manifest (§11.4.262)"; Description: see Evidence and Root cause above; Acceptance: Ledger present; each row cites a passing challenge; gate non-vacuous.

### GAP-018 - Submodule pins not on main: doc_processor diverged; llm_orchestrator/llm_provider/vision_engine pins ahead of main; containers 25 commits behind
- Severity: Medium; Category: Governance; Owner suggestion: operator
- Evidence: docs/submodule-pins-on-main-2026-11-23.md table
- Root cause (UNCONFIRMED unless stated): Upstream mains lag masters; containers new commits may include matrix improvements.
- Proposed fix: Operator-approved fast-forward of containers; upstream FF of main for three; HelixQA/Containers are PROVENANCE 'scratch clones outside repo' for several deps.
- Required test types: integration
- Workable-item draft: Type Task; Title: "Submodule pins not on main: doc_processor diverged; llm_orchestrator/llm_provider/vision_engine pins ahead of main; containers 25 commits behind"; Description: see Evidence and Root cause above; Acceptance: Pins on main per §11.4.288.

### GAP-019 - helixqa was built in a scratch tree outside the repo; several deps are depth-1 scratch clones not submodules
- Severity: Medium; Category: Build; Owner suggestion: qa
- Evidence: evidence/campaign-2026-11-23/helixqa/PROVENANCE.txt
- Root cause (UNCONFIRMED unless stated): helixqa go.mod replace targets (doc_processor, llm_*, security, vision_engine) are not satisfied by repo submodule layout; qa/helixqa/run_helixqa_banks.sh needs scratch build.
- Proposed fix: Make `helixqa` build reproducible from repo (go.work or Containers build image) and record binary sha256 in-tree.
- Required test types: full-automation, integration
- Workable-item draft: Type Task; Title: "helixqa was built in a scratch tree outside the repo; several deps are depth-1 scratch clones not submodules"; Description: see Evidence and Root cause above; Acceptance: Fresh clone + one script yields helixqa binary identical sha256 twice (§11.4.246).

### GAP-020 - gofmt drift in core (hdm/hdm.go, hlc/hlc.go, pkg/crypto/crypto.go, pkg/source/source.go, source_test.go)
- Severity: Medium; Category: Quality; Owner suggestion: core
- Evidence: `gofmt -l core` output (commands_raw.txt)
- Root cause (UNCONFIRMED unless stated): Files edited without format step; no format gate.
- Proposed fix: gofmt -w those files + add gofmt/vet gate to local runner.
- Required test types: unit (static)
- Workable-item draft: Type Task; Title: "gofmt drift in core (hdm/hdm.go, hlc/hlc.go, pkg/crypto/crypto.go, pkg/source/source.go, source_test.go)"; Description: see Evidence and Root cause above; Acceptance: gofmt -l empty.

### GAP-021 - Header oracles missing (CSP, X-Content-Type-Options, HSTS, Cache-Control) -- only 1 non-test occurrence in server
- Severity: Medium; Category: Quality; Owner suggestion: server
- Evidence: grep count = 1 (commands_part2.txt)
- Root cause (UNCONFIRMED unless stated): Server does not set security headers for HTML/JSON/OPDS responses (UNCONFIRMED per route).
- Proposed fix: Add header middleware + table-driven header oracle in httpapi tests and HelixQA bank asserting headers per route.
- Required test types: security, unit, helixqa
- Workable-item draft: Type Task; Title: "Header oracles missing (CSP, X-Content-Type-Options, HSTS, Cache-Control) -- only 1 non-test occurrence in server"; Description: see Evidence and Root cause above; Acceptance: Every route family has asserted headers; mutant removing middleware is RED.

### GAP-022 - Postgres paths unrun for LCP/GC refs
- Severity: Medium; Category: Test; Owner suggestion: server
- Evidence: EVIDENCE.md line 875 'PostgreSQL-backed catalog for this flow (columns pre-exist and are mapped, not exercised live)'
- Root cause (UNCONFIRMED unless stated): Live PG tests exist for catalog but not the LCP columns / GC reference logic.
- Proposed fix: Add live PG tests for LCP license persistence + GC; run in all_live runner.
- Required test types: integration, concurrency
- Workable-item draft: Type Task; Title: "Postgres paths unrun for LCP/GC refs"; Description: see Evidence and Root cause above; Acceptance: Live PG LCP test PASS with RED mutant.

### GAP-023 - ID column still named atm_id in SQLite registry and exports
- Severity: Medium; Category: Product; Owner suggestion: operator
- Evidence: evidence/ID-KEY-RENAME.md line 21; PRAGMA table_info(items) shows atm_id
- Root cause (UNCONFIRMED unless stated): Constitution lists atm_id as a reserved name; HRD key adopted at presentation only.
- Proposed fix: Operator decision on schema rename vs mapping layer; add test locking the mapping.
- Required test types: unit
- Workable-item draft: Type Task; Title: "ID column still named atm_id in SQLite registry and exports"; Description: see Evidence and Root cause above; Acceptance: Decision recorded; tests pin it.

### GAP-024 - Documentation drift: stale 'not implemented' / 'not yet' claims
- Severity: Medium; Category: Docs; Owner suggestion: docs
- Evidence: docs/architecture.md:25 says LCP fulfillment not shown as connected; docs/formats.md:65 says LSD handling absent; docs/user_guide.md vs HRD-031 'Queued' in registry while EVIDENCE shows LCP/PSE delivered; Issues.md HRD-021 status 'In progress' though bank exists (greps in commands_raw.txt; registry listing in this audit)
- Root cause (UNCONFIRMED unless stated): No automated doc-claim check (§11.4.266).
- Proposed fix: Ledger (GAP-016) + a test that greps docs for 'NOT IMPLEMENTED' per capability and compares to code capability flags.
- Required test types: unit, full-automation
- Workable-item draft: Type Task; Title: "Documentation drift: stale 'not implemented' / 'not yet' claims"; Description: see Evidence and Root cause above; Acceptance: Doc-vs-code test GREEN; RED when a flag flips.

### GAP-025 - Mutation coverage is hand-listed (110 mutants) and spans server/core only; clients and tools have 1 mutation each
- Severity: Medium; Category: Test; Owner suggestion: qa
- Evidence: scripts/verify_go_mutation.py ids=110; evidence/ui-mutation: 1 mutant; firebase mutations.log
- Root cause (UNCONFIRMED unless stated): Mutants authored per defect, no automated mutation scoring (§11.4.248 gate cm_mutation_score_on_diff unused).
- Proposed fix: Adopt go-mutesting or gremlins on changed lines; mutants for web TS (stryker) and desktop JS.
- Required test types: unit, full-automation
- Workable-item draft: Type Task; Title: "Mutation coverage is hand-listed (110 mutants) and spans server/core only; clients and tools have 1 mutation each"; Description: see Evidence and Root cause above; Acceptance: Mutation score on diff >= threshold file; killed-map recorded.

### GAP-026 - No cross-browser run (Chromium only); no Firefox/WebKit viewport runs
- Severity: Medium; Category: Platform; Owner suggestion: web
- Evidence: verify_ui.py:102 and live.mjs:16 launch chromium only
- Root cause (UNCONFIRMED unless stated): Playwright browsers installed only for chromium; helixqa web executor CDP is Chromium-centric.
- Proposed fix: Add playwright firefox+webkit projects in container; run 16 checks x 4 viewports x 3 engines.
- Required test types: ui, ux, e2e
- Workable-item draft: Type Feature; Title: "No cross-browser run (Chromium only); no Firefox/WebKit viewport runs"; Description: see Evidence and Root cause above; Acceptance: 48 check-sets PASS with per-engine screenshots.

### GAP-027 - Device/screen configurations unexercised: RTL, large font, high contrast, reduced motion, density, foldable, landscape on web
- Severity: Medium; Category: Platform; Owner suggestion: web+qa
- Evidence: rtl hits 0 in verify_ui.py and live.mjs; android evidence has portrait 1080x1920 and one landscape only
- Root cause (UNCONFIRMED unless stated): Checks cover 4 viewports x light/dark only.
- Proposed fix: Add config axes to verify_ui.py and android matrix (GAP-004).
- Required test types: ui, ux, a11y
- Workable-item draft: Type Feature; Title: "Device/screen configurations unexercised: RTL, large font, high contrast, reduced motion, density, foldable, landscape on web"; Description: see Evidence and Root cause above; Acceptance: Per-axis result rows with screenshots.

### GAP-028 - e-ink profile NOT_STARTED; HarmonyOS and Aurora BLOCKED by missing SDKs
- Severity: Medium; Category: Platform; Owner suggestion: web
- Evidence: supported-platforms.json
- Root cause (UNCONFIRMED unless stated): SDKs not installed; e-ink needs low-refresh theme + CSS profile.
- Proposed fix: Implement e-ink CSS profile on web (testable headless via screenshot diff, greyscale); SDK installs in container for harmony/aurora per operator.
- Required test types: ui, ux
- Workable-item draft: Type Feature; Title: "e-ink profile NOT_STARTED; HarmonyOS and Aurora BLOCKED by missing SDKs"; Description: see Evidence and Root cause above; Acceptance: e-ink profile screenshots pass greyscale/contrast oracle.

### GAP-029 - Web unit coverage small: 5 test files for 17 source files; no coverage number
- Severity: Medium; Category: Test; Owner suggestion: web
- Evidence: commands_part2.txt (web src 17, tests 5)
- Root cause (UNCONFIRMED unless stated): No vitest coverage run recorded.
- Proposed fix: Run vitest --coverage; add tests for App.tsx, Settings, Reader paths.
- Required test types: unit
- Workable-item draft: Type Task; Title: "Web unit coverage small: 5 test files for 17 source files; no coverage number"; Description: see Evidence and Root cause above; Acceptance: Coverage report archived with threshold.

### GAP-030 - Server coverage 60.1% without live services; no combined live coverage number
- Severity: Medium; Category: Test; Owner suggestion: server
- Evidence: EVIDENCE.md baseline row
- Root cause (UNCONFIRMED unless stated): Live coverage never merged.
- Proposed fix: Run with PG+Redis and -coverprofile merge; archive.
- Required test types: unit, integration
- Workable-item draft: Type Task; Title: "Server coverage 60.1% without live services; no combined live coverage number"; Description: see Evidence and Root cause above; Acceptance: Merged coverage.out archived.

### GAP-031 - Interop with real devices unproven: KOReader firmware, Kobo, physical Android phone/tablet
- Severity: Medium; Category: Test; Owner suggestion: operator+qa
- Evidence: docs/user_guide.md:85; supported-platforms.json android notes (emulator only)
- Root cause (UNCONFIRMED unless stated): No physical device in lab; KOReader tested as Lua parser only.
- Proposed fix: Operator-attended phase per §11.4.185 plus emulator-hosted KOReader (Linux AppImage in container) as automated stand-in.
- Required test types: e2e, interop
- Workable-item draft: Type Task; Title: "Interop with real devices unproven: KOReader firmware, Kobo, physical Android phone/tablet"; Description: see Evidence and Root cause above; Acceptance: Real KOReader client syncs progress with server in container.

### GAP-032 - DDoS/slow-loris/flood and chaos coverage is campaign-run, not standing
- Severity: Medium; Category: Test; Owner suggestion: server+qa
- Evidence: evidence/campaign-2026-11-23/slow-client, evidence/live-server-daemon-restart; no scripts in a standing suite
- Root cause (UNCONFIRMED unless stated): One-off harness runs gated by HELIX_LOAD_RUN.
- Proposed fix: Fold into run_all_live.sh with thresholds; add network chaos via Containers network package (tc/netem).
- Required test types: ddos, chaos, stress
- Workable-item draft: Type Task; Title: "DDoS/slow-loris/flood and chaos coverage is campaign-run, not standing"; Description: see Evidence and Root cause above; Acceptance: Standing run produces results.json with pass/fail thresholds.

### GAP-033 - HelixTrack Core defects D1-D5 (default JWT secret, password logging) drafted but not filed; local instance patched with schema SQL
- Severity: Medium; Category: Security; Owner suggestion: operator
- Evidence: docs/upstream-issues/helixtrack-core-defects.md
- Root cause (UNCONFIRMED unless stated): Upstream issues unfiled.
- Proposed fix: File upstream after operator approval; keep local instance isolated.
- Required test types: security
- Workable-item draft: Type Task; Title: "HelixTrack Core defects D1-D5 (default JWT secret, password logging) drafted but not filed; local instance patched with schema SQL"; Description: see Evidence and Root cause above; Acceptance: Issues filed with links.

### GAP-034 - Interactive demo has no platform-matrix automation or HelixQA bank
- Severity: Medium; Category: Platform; Owner suggestion: web
- Evidence: demos/interactive_static_demo present; HRD-023 Queued; no evidence dir for it
- Root cause (UNCONFIRMED unless stated): Never covered by a bank; docs/architecture.md says web prototype does not call API.
- Proposed fix: Build/serve demo in container, add bank with screenshots at 4 viewports + vision read.
- Required test types: ui, ux, helixqa
- Workable-item draft: Type Feature; Title: "Interactive demo has no platform-matrix automation or HelixQA bank"; Description: see Evidence and Root cause above; Acceptance: Bank run 100% PASS with screenshots.

### GAP-035 - Fuzz coverage: 11 Fuzz funcs but none for OPDS XML/JSON generation, LCP license JSON, KOSync body parsers (UNCONFIRMED per file)
- Severity: Medium; Category: Test; Owner suggestion: core
- Evidence: grep func Fuzz count 11 (commands_raw.txt)
- Root cause (UNCONFIRMED unless stated): Fuzz added for format/range parsers only (HRD-013).
- Proposed fix: Add fuzz targets for LCP/LSD parsers and OPDS inputs; run with -fuzztime in standing suite.
- Required test types: security, unit
- Workable-item draft: Type Task; Title: "Fuzz coverage: 11 Fuzz funcs but none for OPDS XML/JSON generation, LCP license JSON, KOSync body parsers (UNCONFIRMED per file)"; Description: see Evidence and Root cause above; Acceptance: Fuzz corpus archived; no crashers in 5 min.

### GAP-036 - No local CI-less gate runner executes the constitution full suite for this repo; only individually-run gates
- Severity: Medium; Category: Process; Owner suggestion: qa
- Evidence: submodules/constitution/scripts/run_full_suite.sh exists; no evidence of a run in this repo; two gates PASS vacuously
- Root cause (UNCONFIRMED unless stated): §11.4.156 requires enforcement locally.
- Proposed fix: Run run_full_suite.sh (read-only config) and store verdicts; wire as pre-commit local hook.
- Required test types: full-automation
- Workable-item draft: Type Task; Title: "No local CI-less gate runner executes the constitution full suite for this repo; only individually-run gates"; Description: see Evidence and Root cause above; Acceptance: Full suite log archived, 0 FAIL.

### GAP-037 - Android build is debug-only; no release/signing/ProGuard or tablet/Wear check
- Severity: Low; Category: Platform; Owner suggestion: android
- Evidence: supported-platforms.json android notes
- Root cause (UNCONFIRMED unless stated): Not requested yet.
- Proposed fix: Release build in container, apksigner verify, install smoke.
- Required test types: e2e, security
- Workable-item draft: Type Task; Title: "Android build is debug-only; no release/signing/ProGuard or tablet/Wear check"; Description: see Evidence and Root cause above; Acceptance: apksigner verify OK; release APK smoke PASS.

### GAP-038 - Challenges for mixed Postgres vs in-memory catalog: web/android evidence uses in-memory catalog only
- Severity: Low; Category: Test; Owner suggestion: qa
- Evidence: supported-platforms.json web and android notes
- Root cause (UNCONFIRMED unless stated): Convenience fixture.
- Proposed fix: Rerun web+android flows against Postgres-backed server (containers).
- Required test types: integration, e2e
- Workable-item draft: Type Task; Title: "Challenges for mixed Postgres vs in-memory catalog: web/android evidence uses in-memory catalog only"; Description: see Evidence and Root cause above; Acceptance: Same checks PASS with PG catalog.

### GAP-039 - Flaky/quarantine registry absent (§11.4.248); needle tests self-skip when needle does not fire
- Severity: Low; Category: Test; Owner suggestion: server
- Evidence: lock_scope_test.go:389 Skipf 'needle did not fire'; no [PROTECTED-SPEC] tags found
- Root cause (UNCONFIRMED unless stated): Probabilistic needles skip rather than fail.
- Proposed fix: Make needle deterministic or fail-closed after N attempts; add quarantine list.
- Required test types: concurrency, race
- Workable-item draft: Type Task; Title: "Flaky/quarantine registry absent (§11.4.248); needle tests self-skip when needle does not fire"; Description: see Evidence and Root cause above; Acceptance: Needle test either fires deterministically or fails.

### GAP-040 - Workable-item statuses lag reality: HRD-031/025/026/029 still Queued/In progress while evidence entries claim delivery
- Severity: Low; Category: Docs; Owner suggestion: operator
- Evidence: DB listing in this audit vs EVIDENCE.md 2026-11-23 sections
- Root cause (UNCONFIRMED unless stated): Registry not updated pending operator/independent review.
- Proposed fix: Update via lifecycle skill after review (not edit by hand).
- Required test types: full-automation
- Workable-item draft: Type Task; Title: "Workable-item statuses lag reality: HRD-031/025/026/029 still Queued/In progress while evidence entries claim delivery"; Description: see Evidence and Root cause above; Acceptance: Registry statuses match evidence; docs regenerated.

## 9. Method and limits
Read-only. No GUI, no display. Gates run read-only: cm_helix_deps_manifest_present (PASS), cm_helix_deps_ref_matches_gitlink (FAIL, 4 DRIFT), cm_machine_evidence_at_every_gate (vacuous PASS), cm_ledger_row_typed_from_closed_vocabulary (vacuous PASS), cm_provisional_done_anchor_wired (PASS 117/117); output in gates_run.txt. No tests were executed in this audit; all test counts are static greps or existing evidence files. Severity is the auditor's judgement. Items not inspected: Containers and HelixQA tests themselves, server route inventory (grep for HandleFunc returned 0 because routing style differs), Playwright CDP flag on `run`.
