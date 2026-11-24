# Helix Open Workable Items Summary

Generated from the authoritative SQLite registry.

In progress: 17, In testing: 0, Operator-blocked: 0, Queued: 44, Ready for testing: 0, Reopened: 0

| ID | Type | Severity | Status | Reopens | Description |
|---|---|---|---|---:|---|
| HRD-019 | Task | High | In progress | 0 | Complete revision-pinned research for every deduplicated reader repository |
| HRD-020 | Task | High | Queued | 0 | Perform manual architecture, code, license, and innovation review of indexed repositories |
| HRD-021 | Feature | Critical | In progress | 0 | Build and execute Helix-specific HelixQA challenge banks |
| HRD-022 | Feature | Critical | In progress | 0 | Deliver catalog and reader clients for native desktop, mobile, and workstation platforms |
| HRD-023 | Feature | High | Queued | 0 | Provide interactive demonstrations across the platform and device matrix |
| HRD-025 | Feature | High | Queued | 0 | Configure and validate Firebase distribution without treating configuration as deployment |
| HRD-026 | Feature | Critical | In progress | 0 | Validate interoperability on real readers, OPDS clients, and remote providers |
| HRD-027 | Task | Critical | In progress | 0 | Remove catalog and API scalability bottlenecks under measured enterprise loads |
| HRD-029 | Feature | Critical | In progress | 0 | Implement verified bidirectional HelixTrack synchronization |
| HRD-031 | Feature | High | In progress | 0 | Implement OPDS-PSE, LCP fulfillment, and remaining external reader integrations |
| HRD-033 | Task | Critical | Queued | 0 | Run final clean qualification and produce fresh ZIP and tar.gz archives |
| HRD-036 | Bug | Medium | In progress | 0 | Remote (webdav/sftp) sources ignore Range requests and expose no PSE link for remote CBZ |
| HRD-037 | Task | Critical | Queued | 0 | Entire clients/ tree (web, android, desktop) is untracked in git |
| HRD-038 | Feature | Critical | In progress | 0 | Web UI has no running HelixQA bank: HLX-WEB-001/002 are SKIP (PLAYWRIGHT-RUNTIME-PENDING) |
| HRD-039 | Task | Critical | In progress | 0 | Vision ('eyes') layer never executed; no §11.4.277 llmctl known-answer validation record |
| HRD-040 | Feature | Critical | In progress | 0 | No automated multi-device/API-level/screen-size matrix: Containers emulator-matrix, vm-matrix, distributed-test unused by this repo |
| HRD-041 | Feature | High | Queued | 0 | Android TV (D-pad/leanback) surface and TalkBack/a11y never exercised |
| HRD-042 | Feature | High | Queued | 0 | Windows and macOS desktop and iOS are BLOCKED with no use of Containers vm-matrix / applesim / remote executors |
| HRD-043 | Feature | High | Queued | 0 | Electron desktop: no GUI/window run; packaged artifacts listed in artifacts.sha256 are missing from evidence |
| HRD-044 | Feature | High | Queued | 0 | Web reader renders plain text only; EPUB/PDF/CBZ not rendered (server exposes no text) |
| HRD-045 | Bug | High | Queued | 0 | Write-path throughput floor missed: MergeWork 152-176 ops/s vs 271.94 ops/s floor |
| HRD-046 | Task | High | Queued | 0 | LCP never exercised against a real provider or third-party reader |
| HRD-047 | Task | High | Queued | 0 | No Challenges-submodule or Security-submodule use in repo code, gates or scripts |
| HRD-048 | Task | High | In progress | 0 | Memory / leak soak test type absent |
| HRD-049 | Task | High | In progress | 0 | Race/deadlock evidence is stale: go-race.log dated 2026-11-20, before LCP/OPDS/PSE/LSD code |
| HRD-050 | Task | High | Queued | 0 | 57 real t.Skip/Skipf calls (the earlier grep of 83 over-counted report.Skipped); 38 are gated on HELIX_* env vars not set in any standing suite (PG DSN 18, LOAD_RUN 11, REDIS 7, others) |
| HRD-051 | Task | Medium | In progress | 0 | No claim-vs-reality ledger (§11.4.266) and no gate-evidence manifest (§11.4.262) |
| HRD-052 | Task | High | In progress | 0 | helix-deps.yaml drift against gitlinks (4 DRIFT: llmctl, containers, claude_toolkit, llama.cpp) |
| HRD-053 | Task | Medium | Queued | 0 | Submodule pins not on main: doc_processor diverged; llm_orchestrator/llm_provider/vision_engine pins ahead of main; containers 25 commits behind |
| HRD-054 | Task | Medium | Queued | 0 | helixqa was built in a scratch tree outside the repo; several deps are depth-1 scratch clones not submodules |
| HRD-055 | Task | Medium | In progress | 0 | gofmt drift in core (hdm/hdm.go, hlc/hlc.go, pkg/crypto/crypto.go, pkg/source/source.go, source_test.go) |
| HRD-056 | Task | Medium | Queued | 0 | Header oracles missing (CSP, X-Content-Type-Options, HSTS, Cache-Control) -- only 1 non-test occurrence in server |
| HRD-057 | Task | Medium | Queued | 0 | Postgres paths unrun for LCP/GC refs |
| HRD-058 | Task | Medium | Queued | 0 | ID column still named atm_id in SQLite registry and exports |
| HRD-059 | Task | Medium | Queued | 0 | Documentation drift: stale 'not implemented' / 'not yet' claims |
| HRD-060 | Task | Medium | Queued | 0 | Mutation coverage is hand-listed (110 mutants) and spans server/core only; clients and tools have 1 mutation each |
| HRD-061 | Feature | Medium | Queued | 0 | No cross-browser run (Chromium only); no Firefox/WebKit viewport runs |
| HRD-062 | Feature | Medium | Queued | 0 | Device/screen configurations unexercised: RTL, large font, high contrast, reduced motion, density, foldable, landscape on web |
| HRD-063 | Feature | Medium | Queued | 0 | e-ink profile NOT_STARTED; HarmonyOS and Aurora BLOCKED by missing SDKs |
| HRD-064 | Task | Medium | Queued | 0 | Web unit coverage small: 5 test files for 17 source files; no coverage number |
| HRD-065 | Task | Medium | Queued | 0 | Server coverage 60.1% without live services; no combined live coverage number |
| HRD-066 | Task | Medium | Queued | 0 | Interop with real devices unproven: KOReader firmware, Kobo, physical Android phone/tablet |
| HRD-067 | Task | Medium | Queued | 0 | DDoS/slow-loris/flood and chaos coverage is campaign-run, not standing |
| HRD-068 | Task | Medium | Queued | 0 | HelixTrack Core defects D1-D5 (default JWT secret, password logging) drafted but not filed; local instance patched with schema SQL |
| HRD-069 | Feature | Medium | Queued | 0 | Interactive demo has no platform-matrix automation or HelixQA bank |
| HRD-070 | Task | Medium | Queued | 0 | Fuzz coverage: 11 Fuzz funcs but none for OPDS XML/JSON generation, LCP license JSON, KOSync body parsers (UNCONFIRMED per file) |
| HRD-071 | Task | Medium | Queued | 0 | No local CI-less gate runner executes the constitution full suite for this repo; only individually-run gates |
| HRD-072 | Task | Low | Queued | 0 | Android build is debug-only; no release/signing/ProGuard or tablet/Wear check |
| HRD-073 | Task | Low | Queued | 0 | Challenges for mixed Postgres vs in-memory catalog: web/android evidence uses in-memory catalog only |
| HRD-074 | Task | Low | Queued | 0 | Flaky/quarantine registry absent (§11.4.248); needle tests self-skip when needle does not fire |
| HRD-075 | Task | Low | Queued | 0 | Workable-item statuses lag reality: HRD-031/025/026/029 still Queued/In progress while evidence entries claim delivery |
| HRD-076 | Feature | High | Queued | 0 | Real screenshots and video of everyday use of ALL clients captured only in the isolated container runner and embedded in every doc category |
| HRD-077 | Bug | Medium | Queued | 0 | WebDAV.List listed directories as zero-byte files |
| HRD-078 | Task | High | Queued | 0 | Advertised-but-unwired libraries: pkg/fts, hlc, LRU cache, settings/theme/crypto are claimed tested but wired into neither server nor clients |
| HRD-079 | Bug | Medium | Queued | 0 | llmctl install ignores its arguments so llmctl install --help performs the real install |
| HRD-080 | Task | Low | Queued | 0 | HRD-039b follow-up: use ExecStartPre=+ for the llmctl engine gate to keep ProtectSystem=full |
| HRD-081 | Task | High | Queued | 0 | Submodule host-side entry points are scanned report-only by the isolation guard: 1457 findings need triage, upstream fix or gate |
| HRD-082 | Bug | Medium | Queued | 0 | helixqa upstream defects worked around in tests/helixqa_web: PlaywrightCLIAdapter never closes the CDP connection, splitSelectorValue splits fill at first space, helixqa http skips playwright steps |
| HRD-083 | Task | High | Queued | 0 | Adopt the PR-driven workflow: branch, PR, independent review agents, merge to main, never force-push, multi-upstream mirror |
| HRD-084 | Task | High | Queued | 0 | Update the constitution submodule to the latest pin with recursive incorporation and post-update hooks |
| HRD-085 | Task | Medium | In progress | 0 | Registry hardening and gap-register import landing (PR #8) |
