# Helix Open Workable Items

Generated from `docs/workable_items.db`; do not edit directly.

In progress: 17, In testing: 0, Operator-blocked: 0, Queued: 44, Ready for testing: 0, Reopened: 0

## [HRD-019] Complete revision-pinned research for every deduplicated reader repository

**Status:** In progress

**Type:** Task

**Severity:** High

**Reopens:** 0

**Description:** 5 primary and 35 supplementary repositories now cloned full, non-shallow, fsck clean, detached at the exact pinned revision, with recomputed file ledgers and per-repository reports under research/reports/. Scope corrected from prose: 111 analyses, of which 14 unique URLs remain unpinned by clone-time budget (not unreachable; ls-remote recorded). BookOrbit is AGPL-3.0-only with section 7 additional terms, so its decision is BLOCKED-BY-LICENSE; koodo-reader's pinned revision was rewritten out of upstream history. Manual architecture and license review still pending, which is what this item gates on. UPDATE 2026-11-23: All 18 re-pin targets proven fetchable with tree-hash match; 107/111 checkouts present, 106 match (research/analysis_summary.json regenerated 2026-11-23); koodo-reader pin bundled (evidence/campaign-2026-11-23/research/bundles, sha256 c33946cf0d6cb8e7b500a5e8ea3775b86fa698d074336d26cb5d3d9b874ebfdb). Remaining: bookorbit client/backend pin alias conflict (needs decision), dyslibria/epuppy/inkmod/shoko upstreams gone (no checkout), manual review (HRD-020) pending.

**Affected scope:** All repositories listed in Opensourced_Reader_Apps_And_Repos.md.

**Reproduction / current observation:** 107/111 checkouts present, 106 match their pin; bookorbit client/backend pin alias conflict unresolved; dyslibria/epuppy/inkmod/shoko upstreams gone (no checkout); manual architecture/license review (HRD-020) pending.

**Acceptance:** Every target has a complete checkout, exact revision, file ledger, manual architecture review, license review, and porting decision.

**Closure criteria:** Every target has a complete checkout, exact revision, file ledger, manual architecture review, license review, and porting decision.

**Evidence:** research/analysis_summary.json; research/reports

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-11-23T21:15:58Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: research/analysis_summary.json; research/reports
- 2026-11-20 — Updated by AI: registry-detail-update; evidence: none recorded
- 2026-11-23 — Updated by AI: repin-targets-proven-fetchable-partial; evidence: evidence/campaign-2026-11-23/research/pin_fetch_probe.txt

## [HRD-020] Perform manual architecture, code, license, and innovation review of indexed repositories

**Status:** Queued

**Type:** Task

**Severity:** High

**Reopens:** 0

**Description:** Perform manual architecture, code, license, and innovation review of indexed repositories. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** The 106 statically indexed research checkouts and any later recovered targets.

**Reproduction / current observation:** Automated keyword indexing exists but does not satisfy manual architecture or license review.

**Acceptance:** Each repository receives an auditable manual comparison and attributed implementation decision linked to tests.

**Closure criteria:** Each repository receives an auditable manual comparison and attributed implementation decision linked to tests.

**Evidence:** research/analysis_summary.json

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: research/analysis_summary.json

## [HRD-021] Build and execute Helix-specific HelixQA challenge banks

**Status:** In progress

**Type:** Feature

**Severity:** Critical

**Reopens:** 0

**Description:** Build and execute Helix-specific HelixQA challenge banks. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope. UPDATE 2026-11-23: Build/dependency half closed: Challenges+HelixQA+6 replace-target modules are submodules (grouped layout), root helix-deps.yaml verifier PASS pass=21; helixqa v0.2.0 builds from the repo; 34 banks (32 PASS/0 FAIL/2 SKIP web) pass twice live; mutant server turned 9 cases RED. NOT done: web bank execution, Vision/UI QA (11.4.277/278 unverified), header oracles, LCP happy-path; GUI evidence moves to the isolated container runner. UPDATE 2026-11-23 (gap register, docs/gap-register-2026-11-23.md): linked gap items: HRD-038 (GAP-002); HRD-039 (GAP-003); HRD-047 (GAP-012); HRD-052 (GAP-017); HRD-053 (GAP-018); HRD-054 (GAP-019); HRD-056 (GAP-021); HRD-059 (GAP-024). See docs/gap-register-item-map-2026-11-23.md.

**Affected scope:** HelixQA CLI, own-org dependency modules, and Helix end-to-end challenges.

**Reproduction / current observation:** 34 banks 32 PASS/0 FAIL/2 SKIP (web) twice live (evidence/campaign-2026-11-23/helixqa/green-1.txt, green-2.txt); mutant server 9 cases RED (red-mutant.txt). Web bank execution, Vision/UI QA, header oracles and LCP happy-path NOT done.

**Acceptance:** All declared dependencies resolve, Helix-specific banks run live, and machine results preserve every failure and skip.

**Closure criteria:** All declared dependencies resolve, Helix-specific banks run live, and machine results preserve every failure and skip.

**Evidence:** evidence/helixqa-build-test.log; submodules/qa/helix-deps.yaml

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-11-23T21:22:45Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: evidence/helixqa-build-test.log; submodules/qa/helix-deps.yaml
- 2026-11-23 — Updated by AI: helixqa-banks-green-twice-mutant-red-partial; evidence: evidence/campaign-2026-11-23/helixqa/PROVENANCE.txt
- 2026-11-23 — Updated by AI: gap-register cross-link (covered by / linked to gap items); evidence: docs/gap-register-item-map-2026-11-23.md

## [HRD-022] Deliver catalog and reader clients for native desktop, mobile, and workstation platforms

**Status:** In progress

**Type:** Feature

**Severity:** Critical

**Reopens:** 0

**Description:** Deliver catalog and reader clients for native desktop, mobile, and workstation platforms. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope. UPDATE 2026-11-23: Operator re-scoped this item on 2026-11-23 to web + Linux + Android. Windows, macOS, iOS, Android TV, HarmonyOS and Aurora remain BLOCKED with reproducers in clients/supported-platforms.json (evidence/clients/blocked_reproducers.json). Web: VERIFIED (67/67 live checks, Chromium; evidence/clients/web). Android: VERIFIED on emulator only (debug build, no physical device; evidence/clients/android/run-20261123T211340Z). Linux desktop: NOT YET (Electron shell in progress). The reader renders plain text only; there is no in-app EPUB/PDF rendering because the server exposes no extracted text. Lifecycle status requested by operator: In progress (the registry tool offers no transition to that status; see report). UPDATE 2026-11-23 (gap register, docs/gap-register-2026-11-23.md): linked gap items: HRD-037 (GAP-001); HRD-040 (GAP-004); HRD-041 (GAP-005); HRD-042 (GAP-006); HRD-043 (GAP-007); HRD-044 (GAP-008); HRD-061 (GAP-026); HRD-062 (GAP-027); HRD-063 (GAP-028); HRD-064 (GAP-029); HRD-072 (GAP-037); HRD-073 (GAP-038). See docs/gap-register-item-map-2026-11-23.md.

**Affected scope:** Windows, macOS, Linux, Android, iOS, and additional device classes.

**Reproduction / current observation:** Web VERIFIED (67/67 live Chromium checks, evidence/clients/web); Android VERIFIED on emulator only, debug build, no physical device (evidence/clients/android/run-20261123T211340Z); Linux desktop NOT YET (Electron shell in progress); reader renders only plain text, no EPUB/PDF in-app rendering because the server exposes no extracted text; Windows/macOS/iOS/Android TV/HarmonyOS/Aurora BLOCKED (clients/supported-platforms.json).

**Acceptance:** Each promised client has installable artifacts, real reader functionality, accessibility support, and device-specific live evidence.

**Closure criteria:** Each promised client has installable artifacts, real reader functionality, accessibility support, and device-specific live evidence.

**Evidence:** docs/architecture.md

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-11-23T21:22:45Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: docs/architecture.md
- 2026-11-23 — Updated by AI: operator-rescope-web-linux-android-partial-evidence; evidence: evidence/clients/web/MANIFEST.sha256
- 2026-11-23 — Started by AI: work under way 2026-11-23; evidence: clients/supported-platforms.json; evidence/clients/android/run-20261123T211340Z/MANIFEST.sha256
- 2026-11-23 — Updated by AI: gap-register cross-link (covered by / linked to gap items); evidence: docs/gap-register-item-map-2026-11-23.md

## [HRD-023] Provide interactive demonstrations across the platform and device matrix

**Status:** Queued

**Type:** Feature

**Severity:** High

**Reopens:** 0

**Description:** Provide interactive demonstrations across the platform and device matrix. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope. UPDATE 2026-11-23 (gap register, docs/gap-register-2026-11-23.md): linked gap items: HRD-069 (GAP-034). See docs/gap-register-item-map-2026-11-23.md.

**Affected scope:** Phone, tablet, desktop, workstation, low-end device, and varied screen demonstrations.

**Reproduction / current observation:** Four browser viewport checks exist but are not native-device demonstrations.

**Acceptance:** Interactive demos exercise real client behavior with explicit hardware and runtime evidence.

**Closure criteria:** Interactive demos exercise real client behavior with explicit hardware and runtime evidence.

**Evidence:** evidence/ui/final-run.log

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-11-23T21:22:45Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: evidence/ui/final-run.log
- 2026-11-23 — Updated by AI: gap-register cross-link (covered by / linked to gap items); evidence: docs/gap-register-item-map-2026-11-23.md

## [HRD-025] Configure and validate Firebase distribution without treating configuration as deployment

**Status:** Queued

**Type:** Feature

**Severity:** High

**Reopens:** 0

**Description:** Configure and validate Firebase distribution without treating configuration as deployment. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope. UPDATE 2026-11-23: Configuration + offline validator + hosting emulator qualified (deploy/firebase, tools/firebase, evidence/campaign-2026-11-23/firebase). DEPLOYMENT: NOT RUN; needs operator-provided Firebase project id, Android app id, GOOGLE_APPLICATION_CREDENTIALS, helix-internal group, and a signed APK. UPDATE 2026-11-23 (gap register, docs/gap-register-2026-11-23.md): linked gap items: HRD-072 (GAP-037); HRD-075 (GAP-040). See docs/gap-register-item-map-2026-11-23.md.

**Affected scope:** Firebase application distribution and associated release metadata.

**Reproduction / current observation:** Offline validator and hosting emulator qualified (evidence/campaign-2026-11-23/firebase/validate-real-tree.txt, hosting-emulator-curl.txt, unit-GREEN.log, unit-RED.log, mutations.log). Real deployment NOT RUN.

**Acceptance:** A real project-bound preflight and distribution run produces non-secret machine evidence or an explicit skip.

**Closure criteria:** A real project-bound preflight and distribution run produces non-secret machine evidence or an explicit skip.

**Evidence:** docs/deployment.md

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-11-23T21:22:45Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: docs/deployment.md
- 2026-11-23 — Updated by AI: firebase-config-and-emulator-qualified-deploy-not-run; evidence: evidence/campaign-2026-11-23/firebase/SHA256SUMS
- 2026-11-23 — Updated by AI: gap-register cross-link (covered by / linked to gap items); evidence: docs/gap-register-item-map-2026-11-23.md

## [HRD-026] Validate interoperability on real readers, OPDS clients, and remote providers

**Status:** In progress

**Type:** Feature

**Severity:** Critical

**Reopens:** 0

**Description:** Validate interoperability on real readers, OPDS clients, and remote providers. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope. UPDATE 2026-11-23: Scripted interop PASS: KOReader real kosync client + partialMD5 push/pull with redis sink; real opdsparser.lua; EPUB+PSE byte-identical; rclone WebDAV+SFTP loopback providers byte-identical (evidence/campaign-2026-11-23/interop/hrd026). Finding (tracked as its own Bug): remote webdav/sftp sources serve Range as full 200 (no 206) and no PSE link on remote CBZ. NOT done: KOReader GUI (pending isolated container runner), real cloud providers, S3 (rclone 1.60.1 has no S3 server). UPDATE 2026-11-23: The Range/PSE defect is tracked as HRD-036. UPDATE 2026-11-23 (gap register, docs/gap-register-2026-11-23.md): linked gap items: HRD-036 (GAP-009); HRD-046 (GAP-011); HRD-066 (GAP-031); HRD-075 (GAP-040). See docs/gap-register-item-map-2026-11-23.md.

**Affected scope:** KOReader devices, third-party OPDS clients, and deployment-specific rclone providers.

**Reproduction / current observation:** Scripted interop PASS (kosync_koreader_real_client.txt, opds_koreader_parser_output.txt, hrd026/providers_verify.txt). KOReader GUI, real cloud providers and S3 NOT run.

**Acceptance:** External clients complete real acquisition and sync workflows with captured version and environment evidence.

**Closure criteria:** External clients complete real acquisition and sync workflows with captured version and environment evidence.

**Evidence:** docs/testing.md

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-11-23T21:22:46Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: docs/testing.md
- 2026-11-23 — Updated by AI: scripted-interop-pass-partial-gui-cloud-s3-not-run; evidence: evidence/campaign-2026-11-23/interop/hrd026/providers_verify.txt
- 2026-11-23 — Updated by AI: link-range-pse-finding-to-hrd-036; evidence: evidence/campaign-2026-11-23/interop/hrd026/providers_verify.txt
- 2026-11-23 — Started by AI: work under way 2026-11-23; evidence: evidence/campaign-2026-11-23/interop/hrd026/
- 2026-11-23 — Updated by AI: gap-register cross-link (covered by / linked to gap items); evidence: docs/gap-register-item-map-2026-11-23.md

## [HRD-027] Remove catalog and API scalability bottlenecks under measured enterprise loads

**Status:** In progress

**Type:** Task

**Severity:** Critical

**Reopens:** 0

**Description:** Phase 1 landed: configurable catalog/AI-memory pools, distinct aimemory migration lock, durable scan records across restart, batched page child loads (measured 3.6x), and an additive keyset cursor through HTTP and OPDS with needle-proofed tests. First 50k-work measurements recorded in docs/scalability-review.md and evidence/campaign-2026-11-20/scalability/. DEFECT-003 (transport-contaminated and nondeterministic scale assertions) found and fixed. Identity locks are now scoped per work, edition and source instead of one global key: 5.85x write throughput in one process (271.94 -> 1592.21 ops/s) and 1.53x to 4.07x across separate OS processes, while the no-lock needle silently loses 8 of 480 concurrently merged editions (evidence/campaign-2026-11-20/lock-scope/ and .../multiprocess/). DEFECT-004: the multi-process needle was blind until a database barrier aligned process starts, because the unsafe window is only the first write to an identity that does not exist yet; the runner now refuses a run where the unlocked variant verifies clean. Concurrent read envelope measured at 1/8/32/64 clients with zero errors: indexed browse and OPDS plateau near 1000 req/s (repo) and 900 (HTTP), limited by database CPU rather than the pool (pool=64 probe 942 vs 992 req/s, zero pool waits), while unanchored %text% search plateaus near 9 req/s at p99 12.0 s with pool starvation stacked on that wall. Release gate still unmet: no ratified SLO, no ingest-concurrent-with-reads test, and content delivery, KOSync/Redis and remote-source connectors remain unmeasured under load. Vector search measured and fixed: Search carried the id tiebreak inside the ORDER BY distance clause, an ordering no pgvector index can satisfy, so every search scanned and sorted every stored embedding while the HNSW index was still maintained on insert (DEFECT-005). Moved outside the subquery: 28.3 ms -> 1.86 ms p50 and 172.8 -> 2748 req/s at 32 clients, 50k rows x 384 dims, and the index path is now chosen without being forced. The harness refuses to be trusted on its own word: catalog-level schema comparison, per-connection session attestation, and a cardinality-independent plan control, with a nine-mutation gate at 9/9 (evidence/campaign-2026-11-20/vector/). New open question, not a defect: hnsw.ef_search (compile-time default 40) is not configurable anywhere in the product, and recall@10 against an exhaustive scan of the same fixture is 0.1025 at that width on 384-dim uniform vectors, 0.5475 at width 400. UPDATE 2026-11-23 (gap register, docs/gap-register-2026-11-23.md): linked gap items: HRD-045 (GAP-010); HRD-048 (GAP-013); HRD-067 (GAP-032); HRD-074 (GAP-039). See docs/gap-register-item-map-2026-11-23.md.

**Affected scope:** PostgreSQL writes and search, API pagination, job retention, and connection pools.

**Reproduction / current observation:** Measured and addressed: offset pagination (keyset cursor), per-work child-query fan-out (batched loads), catalog pool caps, and global write serialization, which is now one advisory lock per touched work/edition/source instead of one key for every writer (5.85x on disjoint concurrent merges in a single in-container run, at 89% of the unlocked ceiling; invariants carried by server/internal/catalog/lock_scope_test.go with paired needles and a key-class mutation gate). Vector search: fixed (DEFECT-005, 15.2x p50 at 50k rows, needle-gated harness). Still open: no ratified service-level objective, no recall or search-width objective for AI memory, no multi-process or multi-node topology result, unindexed text search and per-page count(*) deliberately unbudgeted, and same-content merges from distinct sources serialize by design.

**Acceptance:** Representative large-catalog benchmarks and load tests establish limits, durable queue behavior, and bounded query plans.

**Closure criteria:** Representative large-catalog benchmarks and load tests establish limits, durable queue behavior, and bounded query plans.

**Evidence:** docs/scalability-review.md; server/internal/catalog/postgres.go; server/internal/httpapi/jobs.go; server/internal/aimemory/vector_load_test.go; evidence/campaign-2026-11-20/vector

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-11-23T21:22:46Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: docs/scalability-review.md; server/internal/catalog/postgres.go; server/internal/httpapi/jobs.go
- 2026-11-20 — Updated by AI: registry-detail-update; evidence: none recorded
- 2026-11-20 — Updated by AI: catalog write hotspot fixed by scoping identity locks; measured 5.85x with needles and a mutation gate; evidence: evidence/campaign-2026-11-20/lock-scope/summary.md
- 2026-11-20 — Updated by AI: Multi-process lock-scope qualification and concurrent read envelope recorded; DEFECT-004 documented.; evidence: docs/scalability-review.md; server/internal/catalog/postgres.go; server/internal/httpapi/jobs.go; evidence/campaign-2026-11-20/scalability/; evidence/campaign-2026-11-20/load/; evidence/campaign-2026-11-20/lock-scope/; evidence/campaign-2026-11-20/multiprocess/
- 2026-11-20 — Updated by AI: vector-search load harness landed; DEFECT-005 measured and fixed; ef_search recall question opened; evidence: docs/scalability-review.md; server/internal/catalog/postgres.go; server/internal/httpapi/jobs.go; server/internal/aimemory/vector_load_test.go; evidence/campaign-2026-11-20/vector
- 2026-11-23 — Updated by AI: gap-register cross-link (covered by / linked to gap items); evidence: docs/gap-register-item-map-2026-11-23.md

## [HRD-029] Implement verified bidirectional HelixTrack synchronization

**Status:** In progress

**Type:** Feature

**Severity:** Critical

**Reopens:** 0

**Description:** Implement verified bidirectional HelixTrack synchronization. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope. UPDATE 2026-11-23 (gap register, docs/gap-register-2026-11-23.md): linked gap items: HRD-068 (GAP-033); HRD-075 (GAP-040). See docs/gap-register-item-map-2026-11-23.md.

**Affected scope:** HelixTrack mapping, idempotency, conflict handling, and remote receipts.

**Reproduction / current observation:** Only an offline outbox contract is being established; exact remote API payloads and credentials are unverified.

**Acceptance:** Documented API-compatible push and pull operations pass against a real HelixTrack instance without inventing endpoints.

**Closure criteria:** Documented API-compatible push and pull operations pass against a real HelixTrack instance without inventing endpoints.

**Evidence:** docs/tracking/helixtrack-outbox.json

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-11-23T21:22:46Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: docs/tracking/helixtrack-outbox.json
- 2026-11-23 — Updated by AI: gap-register cross-link (covered by / linked to gap items); evidence: docs/gap-register-item-map-2026-11-23.md

## [HRD-031] Implement OPDS-PSE, LCP fulfillment, and remaining external reader integrations

**Status:** In progress

**Type:** Feature

**Severity:** High

**Reopens:** 0

**Description:** Implement OPDS-PSE, LCP fulfillment, and remaining external reader integrations. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope. UPDATE 2026-11-23: OPDS-PSE served and qualified; LCP producer+reader, consumer fulfillment (lcp.Fulfill/FulfillTo), License Status Document client, server .lcpl ingest with SSRF-guarded fetch + cache GC + verified serving + OPDS license acquisition link are implemented (evidence/campaign-2026-11-23/opds-lcp). NOT done/unconfirmed: real LCP provider/reader interop, server-side signature validation, PDF/audiobook/Divina LCP, LSD wiring in the server, Postgres live run of GC refs query. UPDATE 2026-11-23 (gap register, docs/gap-register-2026-11-23.md): linked gap items: HRD-036 (GAP-009); HRD-046 (GAP-011); HRD-057 (GAP-022); HRD-059 (GAP-024); HRD-075 (GAP-040). See docs/gap-register-item-map-2026-11-23.md.

**Affected scope:** Page streaming, DRM fulfillment, and vendor-specific synchronization.

**Reproduction / current observation:** Implemented and exercised against a live server: evidence/campaign-2026-11-23/opds-lcp/live.log, lcp_fulfill_live.json, lsd_live.json, lsd_mutation.json. Real LCP provider/reader interop, server-side signature validation, PDF/audiobook/Divina LCP, server LSD wiring and Postgres GC-refs run NOT done/UNCONFIRMED.

**Acceptance:** Each integration has a working implementation, legal DRM handling, interoperability tests, and honest capability reporting.

**Closure criteria:** Each integration has a working implementation, legal DRM handling, interoperability tests, and honest capability reporting.

**Evidence:** docs/api.md; docs/faq.md; docs/formats.md

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-11-23T21:22:46Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: docs/api.md; docs/faq.md; docs/formats.md
- 2026-11-23 — Updated by AI: opds-pse-and-lcp-fulfillment-implemented-partial; evidence: evidence/campaign-2026-11-23/opds-lcp/results.json
- 2026-11-23 — Started by AI: work under way 2026-11-23; evidence: evidence/campaign-2026-11-23/opds-lcp/results.json
- 2026-11-23 — Updated by AI: gap-register cross-link (covered by / linked to gap items); evidence: docs/gap-register-item-map-2026-11-23.md

## [HRD-033] Run final clean qualification and produce fresh ZIP and tar.gz archives

**Status:** Queued

**Type:** Task

**Severity:** Critical

**Reopens:** 0

**Description:** Run final clean qualification and produce fresh ZIP and tar.gz archives. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** Clean extraction, full tests, secret scan, deterministic archive-content validation, and release notes.

**Reproduction / current observation:** No new-round archive is permitted while requested features, research, external validation, and qualification remain open.

**Acceptance:** All blocking work is genuinely closed, final evidence is refreshed, and both archives pass clean-room verification.

**Closure criteria:** All blocking work is genuinely closed, final evidence is refreshed, and both archives pass clean-room verification.

**Evidence:** docs/testing/results; CHANGELOG.md

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: docs/testing/results; CHANGELOG.md

## [HRD-036] Remote (webdav/sftp) sources ignore Range requests and expose no PSE link for remote CBZ

**Status:** In progress

**Type:** Bug

**Severity:** Medium

**Reopens:** 0

**Description:** Found 2026-11-23 during HRD-026 interop. For library sources backed by remote providers (rclone WebDAV and SFTP loopback), a ranged GET of a book file is answered with the full body and HTTP 200 instead of 206 Partial Content, with no Accept-Ranges or Content-Range headers; and a remote CBZ publication carries no OPDS-PSE page-stream link, while a local-library CBZ does. Local library sources behave correctly (206, Content-Range bytes 1000-2047/N). Impact: clients that depend on Range (seeking, resume, large-file reading) download whole files from remote sources, and streaming comic readers cannot page-stream remote CBZ. Root cause NOT yet investigated (UNCONFIRMED). UPDATE 2026-11-23 (gap register, docs/gap-register-2026-11-23.md): linked gap items: GAP-009 is covered by HRD-036 (no new item minted). See docs/gap-register-item-map-2026-11-23.md.

**Affected scope:** server remote-source serving path (webdav/sftp providers), OPDS feed generation for remote CBZ, HRD-026 interop evidence.

**Reproduction / current observation:** See evidence/campaign-2026-11-23/interop/hrd026/providers_verify.txt: lines for p-sftp and p-webdav show range_status=200, accept-ranges=None, content-range=None for comic.cbz and three EPUBs, while the lib source rows show range_status=206 with content-range=bytes 1000-2047/N. Reproducer script and loopback provider setup are under evidence/campaign-2026-11-23/interop/hrd026/ (providers_scan.txt, providers_verify.txt).

**Acceptance:** A ranged GET against webdav and sftp backed sources returns 206 with correct Content-Range and byte-identical partial content, and a remote CBZ publication exposes an OPDS-PSE link whose pages stream byte-identically, proven by a live run recorded under evidence/ with a RED baseline on the current build.

**Closure criteria:** A ranged GET against webdav and sftp backed sources returns 206 with correct Content-Range and byte-identical partial content, and a remote CBZ publication exposes an OPDS-PSE link whose pages stream byte-identically, proven by a live run recorded under evidence/ with a RED baseline on the current build.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** HRD-026

**Composes with:** None recorded

**Attribution:** created by User; assigned to Operator

**Created / modified:** 2026-11-23T21:16:05Z / 2026-11-23T22:50:13Z

**History:**

- 2026-11-23 — Opened by User: new-work-item; evidence: none recorded
- 2026-11-23 — Updated by AI: gap-register cross-link (covered by / linked to gap items); evidence: docs/gap-register-item-map-2026-11-23.md
- 2026-11-23 — Started by AI: Work in progress with independent review; uncommitted, not merged; evidence: evidence/campaign-2026-11-23/gap-fix/hrd036

## [HRD-037] Entire clients/ tree (web, android, desktop) is untracked in git

**Status:** Queued

**Type:** Task

**Severity:** Critical

**Reopens:** 0

**Description:** Entire clients/ tree (web, android, desktop) is untracked in git. Filed 2026-11-23 from the read-only gap audit (GAP-001, Process, severity Critical); no work started. EVIDENCE: git status: '?? clients/'; `git ls-files clients | wc -l` = 0 (commands_part2.txt) ROOT CAUSE (UNCONFIRMED unless stated): UNCONFIRMED: work done in session never committed (operator holds commits per memory note). PROPOSED FIX: Commit clients/ (excluding node_modules/dist/out per .gitignore) on main after independent review; add a gate that fails when a supported-platforms.json VERIFIED platform has untracked sources. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Unit (11.4.169(1)) as static gate; Full-automation (11.4.169(4)). Cross-links (11.4.214, both ways): HRD-022. Source: docs/gap-register-2026-11-23.md#gap-001---entire-clients-tree-web-android-desktop-is-untracked-in-git. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: operator-gated.

**Affected scope:** Process gap GAP-001: Entire clients/ tree (web, android, desktop) is untracked in git

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-001---entire-clients-tree-web-android-desktop-is-untracked-in-git; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: git status: '?? clients/'; `git ls-files clients | wc -l` = 0 (commands_part2.txt)

**Acceptance:** git ls-files clients lists sources; gate GREEN with RED baseline on an untracked tree. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** git ls-files clients lists sources; gate GREEN with RED baseline on an untracked tree. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-001 docs/gap-register-2026-11-23.md#gap-001---entire-clients-tree-web-android-desktop-is-untracked-in-git

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:37Z / 2026-11-23T21:22:37Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-038] Web UI has no running HelixQA bank: HLX-WEB-001/002 are SKIP (PLAYWRIGHT-RUNTIME-PENDING)

**Status:** In progress

**Type:** Feature

**Severity:** Critical

**Reopens:** 0

**Description:** Web UI has no running HelixQA bank: HLX-WEB-001/002 are SKIP (PLAYWRIGHT-RUNTIME-PENDING). Filed 2026-11-23 from the read-only gap audit (GAP-002, HelixQA/Vision, severity Critical); no work started. EVIDENCE: green-1.json: 34 cases, 32 PASS, 2 SKIP (HLX-WEB-001/002); qa/helixqa/banks/helix-web.yaml lines 19-29 say 'executor is a stub' ROOT CAUSE (UNCONFIRMED unless stated): Stale premise: submodules/helixqa/pkg/autonomous/playwright_executor.go implements a CDP-driven PlaywrightExecutor (wraps userflow.PlaywrightCLIAdapter) and is skipped only when PipelineConfig.PlaywrightCDPURL is empty. UNCONFIRMED whether `helixqa run` (cmd/helixqa/main.go) exposes a CDP flag: grep for 'cdp' in main.go returned nothing, only --browser-url/--desktop-process. PROPOSED FIX: Start a Chromium with --remote-debugging-port in a rootless container (Containers), wire CDP URL (extend `helixqa run` if the flag is absent, upstream PR), remove _skip from the bank, add RED mutant for the web UI. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): HelixQA (11.4.169(6)); UI (11.4.27 verbatim mandate); UX (11.4.27 verbatim mandate); End-to-end (11.4.169(3)); Vision read (cross-cutting 11.4.278/11.4.117/11.4.160, not a 11.4.169 member). Cross-links (11.4.214, both ways): HRD-021. Source: docs/gap-register-2026-11-23.md#gap-002---web-ui-has-no-running-helixqa-bank-hlx-web-001002-are-skip-playwright-runtime-pending. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: operator+helixqa upstream.

**Affected scope:** HelixQA/Vision gap GAP-002: Web UI has no running HelixQA bank: HLX-WEB-001/002 are SKIP (PLAYWRIGHT-RUNTIME-PENDING)

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-002---web-ui-has-no-running-helixqa-bank-hlx-web-001002-are-skip-playwright-runtime-pending; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: green-1.json: 34 cases, 32 PASS, 2 SKIP (HLX-WEB-001/002); qa/helixqa/banks/helix-web.yaml lines 19-29 say 'executor is a stub'

**Acceptance:** helixqa run over helix-web.yaml reports 0 SKIP, 2 PASS with screenshots per step; mutating a UI label turns it RED. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** helixqa run over helix-web.yaml reports 0 SKIP, 2 PASS with screenshots per step; mutating a UI label turns it RED. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-002 docs/gap-register-2026-11-23.md#gap-002---web-ui-has-no-running-helixqa-bank-hlx-web-001002-are-skip-playwright-runtime-pending

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:38Z / 2026-11-23T22:50:14Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded
- 2026-11-23 — Started by AI: Work in progress with independent review; uncommitted, not merged; evidence: evidence/campaign-2026-11-23/gap-fix/hrd038

## [HRD-039] Vision ('eyes') layer never executed; no §11.4.277 llmctl known-answer validation record

**Status:** In progress

**Type:** Task

**Severity:** Critical

**Reopens:** 0

**Description:** Vision ('eyes') layer never executed; no §11.4.277 llmctl known-answer validation record. Filed 2026-11-23 from the read-only gap audit (GAP-003, HelixQA/Vision, severity Critical); no work started. EVIDENCE: No vision/OCR artifacts under evidence/ (grep 'vision' in evidence index = none produced by us); EVIDENCE.md lists no llmctl validation; PROVENANCE.txt shows vision_engine/llm_provider only as scratch clones ROOT CAUSE (UNCONFIRMED unless stated): Vision pipeline not wired into any run; §11.4.277 gate unfulfilled. UNCONFIRMED which Vision model/endpoint is available on this host. PROPOSED FIX: Run llmctl known-answer validation for the chosen model, then read every captured screenshot (web 4 viewports, android run) through helixqa vision/recordingqa and store verdict JSON beside the PNG. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Vision read (cross-cutting 11.4.278/11.4.117/11.4.160, not a 11.4.169 member); UI (11.4.27 verbatim mandate); UX (11.4.27 verbatim mandate); HelixQA (11.4.169(6)). Cross-links (11.4.214, both ways): HRD-021. Source: docs/gap-register-2026-11-23.md#gap-003---vision-eyes-layer-never-executed-no-114277-llmctl-known-answer-validation-record. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: operator+qa. UPDATE 2026-11-23: root cause confirmed and fixed externally in llmctl (uncommitted, see HRD-079 and HRD-080); vision validated (evidence/campaign-2026-11-23/gap-fix/hrd039/green_vision_validate.json); verifier tooling reviewed GO. Evidence: evidence/campaign-2026-11-23/gap-fix/hrd039/.

**Affected scope:** HelixQA/Vision gap GAP-003: Vision ('eyes') layer never executed; no §11.4.277 llmctl known-answer validation record

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-003---vision-eyes-layer-never-executed-no-114277-llmctl-known-answer-validation-record; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: No vision/OCR artifacts under evidence/ (grep 'vision' in evidence index = none produced by us); EVIDENCE.md lists no llmctl validation; PROVENANCE.txt shows vision_engine/llm_provider only as scratch clones

**Acceptance:** Per-screenshot vision verdict JSON with sha256 manifest; a deliberately wrong-screen mutant is rejected. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Per-screenshot vision verdict JSON with sha256 manifest; a deliberately wrong-screen mutant is rejected. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-003 docs/gap-register-2026-11-23.md#gap-003---vision-eyes-layer-never-executed-no-114277-llmctl-known-answer-validation-record

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:38Z / 2026-11-23T22:50:14Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded
- 2026-11-23 — Updated by AI: Record 2026-11-23 operator decision and status update; evidence: evidence/campaign-2026-11-23/gap-fix/hrd039
- 2026-11-23 — Started by AI: Work in progress with independent review; uncommitted, not merged; evidence: evidence/campaign-2026-11-23/gap-fix/hrd039

## [HRD-040] No automated multi-device/API-level/screen-size matrix: Containers emulator-matrix, vm-matrix, distributed-test unused by this repo

**Status:** In progress

**Type:** Feature

**Severity:** Critical

**Reopens:** 0

**Description:** No automated multi-device/API-level/screen-size matrix: Containers emulator-matrix, vm-matrix, distributed-test unused by this repo. Filed 2026-11-23 from the read-only gap audit (GAP-004, Platform, severity Critical); no work started. EVIDENCE: grep for emulator-matrix|vm-matrix|distributed-test|applesim outside submodules/: 0 hits; Android evidence is ONE AVD (API 34, 1080x1920 + one landscape) run by hand-driven adb ROOT CAUSE (UNCONFIRMED unless stated): Containers capability present (pkg/emulator Matrix with AVD{APILevel,FormFactor}, NetworkProfile, CaptureScreenshotOnFailure, Concurrent; cmd/emulator-matrix) but no repo script feeds it an APK + AVD list. PROPOSED FIX: Add scripts/run_android_matrix.sh invoking cmd/emulator-matrix with API 24/28/30/34/36 x phone/tablet(7in,10in)/foldable x density mdpi..xxxhdpi x locale(en,ar RTL) x font-scale 1.3/2.0 x dark/light; attest per-row JSON. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): End-to-end (11.4.169(3)); UI (11.4.27 verbatim mandate); UX (11.4.27 verbatim mandate); Full-automation (11.4.169(4)); HelixQA (11.4.169(6)); Stress + Chaos (11.4.169(9)). Cross-links (11.4.214, both ways): HRD-022. Source: docs/gap-register-2026-11-23.md#gap-004---no-automated-multi-deviceapi-levelscreen-size-matrix-containers-emulator-matrix-vm-matrix-distributed-test-unused-by-this-repo. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: qa+containers. UPDATE 2026-11-23 (operator decision: approve after disk cleanup): operator approved disk cleanup first (dangling podman images, go modcache, gradle/npm/Cypress caches, with the system logs trim done by the operator). PREREQUISITES: that cleanup, about 4 Android system images, and a real instrumented TEST_CLASS. Work in progress (uncommitted) with independent review rounds: evidence evidence/campaign-2026-11-23/gap-fix/hrd040/ (rev2 and rev3 files). Cross-links (text): HRD-081 (isolation scope).

**Affected scope:** Platform gap GAP-004: No automated multi-device/API-level/screen-size matrix: Containers emulator-matrix, vm-matrix, distributed-test unused by this repo

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-004---no-automated-multi-deviceapi-levelscreen-size-matrix-containers-emulator-matrix-vm-matrix-distributed-test-unused-by-this-repo; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: grep for emulator-matrix|vm-matrix|distributed-test|applesim outside submodules/: 0 hits; Android evidence is ONE AVD (API 34, 1080x1920 + one landscape) run by hand-driven adb

**Acceptance:** Attestation JSON with >=12 AVD rows, each boot_completed + test PASS, sha256 manifest, RED row for a broken APK. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Attestation JSON with >=12 AVD rows, each boot_completed + test PASS, sha256 manifest, RED row for a broken APK. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-004 docs/gap-register-2026-11-23.md#gap-004---no-automated-multi-deviceapi-levelscreen-size-matrix-containers-emulator-matrix-vm-matrix-distributed-test-unused-by-this-repo

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:38Z / 2026-11-23T22:50:14Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded
- 2026-11-23 — Updated by AI: Record 2026-11-23 operator decision and status update; evidence: evidence/campaign-2026-11-23/gap-fix/hrd040
- 2026-11-23 — Started by AI: Work in progress with independent review; uncommitted, not merged; evidence: evidence/campaign-2026-11-23/gap-fix/hrd040

## [HRD-041] Android TV (D-pad/leanback) surface and TalkBack/a11y never exercised

**Status:** Queued

**Type:** Feature

**Severity:** High

**Reopens:** 0

**Description:** Android TV (D-pad/leanback) surface and TalkBack/a11y never exercised. Filed 2026-11-23 from the read-only gap audit (GAP-005, Platform, severity High); no work started. EVIDENCE: supported-platforms.json android-tv BLOCKED ('an android-tv system image is installed but no AVD uses it'); no TalkBack or D-pad key events in e2e/live.mjs or verify_ui.py (grep: 0 rtl hits) ROOT CAUSE (UNCONFIRMED unless stated): Reason given is an environment gap, not a hard block: Containers emulator supports FormFactor 'tv'. UNCONFIRMED that web client has focus-order/D-pad handling at all. PROPOSED FIX: Create tv AVD via emulator-matrix, add D-pad navigation bank (DPAD_UP/DOWN/CENTER), uiautomator focus assertions; add axe-core + TalkBack content-description checks. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): UI (11.4.27 verbatim mandate); UX (11.4.27 verbatim mandate); End-to-end (11.4.169(3)); Accessibility (UX axis, cross-cutting); HelixQA (11.4.169(6)). Cross-links (11.4.214, both ways): HRD-022, HRD-040 (GAP-004). Source: docs/gap-register-2026-11-23.md#gap-005---android-tv-d-padleanback-surface-and-talkbacka11y-never-exercised. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: qa.

**Affected scope:** Platform gap GAP-005: Android TV (D-pad/leanback) surface and TalkBack/a11y never exercised

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-005---android-tv-d-padleanback-surface-and-talkbacka11y-never-exercised; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: supported-platforms.json android-tv BLOCKED ('an android-tv system image is installed but no AVD uses it'); no TalkBack or D-pad key events in e2e/live.mjs or verify_ui.py (grep: 0 rtl hits)

**Acceptance:** TV AVD run: every screen reachable by D-pad only, focus never lost; axe-core 0 serious violations on 4 viewports. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** TV AVD run: every screen reachable by D-pad only, focus never lost; axe-core 0 serious violations on 4 viewports. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-005 docs/gap-register-2026-11-23.md#gap-005---android-tv-d-padleanback-surface-and-talkbacka11y-never-exercised

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:38Z / 2026-11-23T21:22:38Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-042] Windows and macOS desktop and iOS are BLOCKED with no use of Containers vm-matrix / applesim / remote executors

**Status:** Queued

**Type:** Feature

**Severity:** High

**Reopens:** 0

**Description:** Windows and macOS desktop and iOS are BLOCKED with no use of Containers vm-matrix / applesim / remote executors. Filed 2026-11-23 from the read-only gap audit (GAP-006, Platform, severity High); no work started. EVIDENCE: supported-platforms.json: windows, macos, ios BLOCKED (host Linux); Containers has pkg/vm (qemu, macos, ios subdirs), pkg/applesim, pkg/remote (ssh executor), pkg/distribution ROOT CAUSE (UNCONFIRMED unless stated): Block reasons assume the operator host only. UNCONFIRMED: qemu Windows guest image availability and licensing; macOS requires Apple hardware (pkg/remote to a Mac host or applesim on one). PROPOSED FIX: Evaluate vm-matrix with a Windows guest (licensed image) for Electron NSIS build+smoke; register a remote Mac builder via pkg/remote for macOS/iOS; keep BLOCKED rows with reproducers until proven. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): End-to-end (11.4.169(3)); UI (11.4.27 verbatim mandate); Full-automation (11.4.169(4)); Integration (11.4.169(2)). Cross-links (11.4.214, both ways): HRD-022, HRD-040 (GAP-004). Source: docs/gap-register-2026-11-23.md#gap-006---windows-and-macos-desktop-and-ios-are-blocked-with-no-use-of-containers-vm-matrix--applesim--remote-executors. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: operator decision (hardware/licence).

**Affected scope:** Platform gap GAP-006: Windows and macOS desktop and iOS are BLOCKED with no use of Containers vm-matrix / applesim / remote executors

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-006---windows-and-macos-desktop-and-ios-are-blocked-with-no-use-of-containers-vm-matrix--applesim--remote-executors; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: supported-platforms.json: windows, macos, ios BLOCKED (host Linux); Containers has pkg/vm (qemu, macos, ios subdirs), pkg/applesim, pkg/remote (ssh executor), pkg/distribution

**Acceptance:** Either a Windows VM run with install+launch+screenshot evidence, or an operator-signed WON'T-FIX per §11.4.112 with reproducer. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Either a Windows VM run with install+launch+screenshot evidence, or an operator-signed WON'T-FIX per §11.4.112 with reproducer. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-006 docs/gap-register-2026-11-23.md#gap-006---windows-and-macos-desktop-and-ios-are-blocked-with-no-use-of-containers-vm-matrix--applesim--remote-executors

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:38Z / 2026-11-23T21:22:38Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-043] Electron desktop: no GUI/window run; packaged artifacts listed in artifacts.sha256 are missing from evidence

**Status:** Queued

**Type:** Feature

**Severity:** High

**Reopens:** 0

**Description:** Electron desktop: no GUI/window run; packaged artifacts listed in artifacts.sha256 are missing from evidence. Filed 2026-11-23 from the read-only gap audit (GAP-007, Platform, severity High); no work started. EVIDENCE: evidence/clients/desktop/artifacts.sha256 -> sha256sum -c: .deb and .tar.gz 'FAILED open or read' (files live in git-ignored clients/desktop/out); live_handler.json states gui:false 'no Electron window'; supported-platforms.json says linux=BUILT (not VERIFIED) ROOT CAUSE (UNCONFIRMED unless stated): Evidence points at build output outside the evidence tree; no display-less GUI path (xvfb/Wayland-headless in a container) was used. PROPOSED FIX: Run packaged app under xvfb inside a rootless container, drive with helixqa desktop executor (x11grab), copy artifacts + hashes into evidence; do not use the operator display. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): End-to-end (11.4.169(3)); UI (11.4.27 verbatim mandate); UX (11.4.27 verbatim mandate); HelixQA (11.4.169(6)); Vision read (cross-cutting 11.4.278/11.4.117/11.4.160, not a 11.4.169 member). Cross-links (11.4.214, both ways): HRD-022, HRD-040 (GAP-004). Source: docs/gap-register-2026-11-23.md#gap-007---electron-desktop-no-guiwindow-run-packaged-artifacts-listed-in-artifactssha256-are-missing-from-evidence. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: qa.

**Affected scope:** Platform gap GAP-007: Electron desktop: no GUI/window run; packaged artifacts listed in artifacts.sha256 are missing from evidence

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-007---electron-desktop-no-guiwindow-run-packaged-artifacts-listed-in-artifactssha256-are-missing-from-evidence; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: evidence/clients/desktop/artifacts.sha256 -> sha256sum -c: .deb and .tar.gz 'FAILED open or read' (files live in git-ignored clients/desktop/out); live_handler.json states gui:false 'no Electron window'; supported-platforms.json says linux=BUILT (not VERIFIED)

**Acceptance:** Manifest verifies; screenshot of library window with titles from live server; RED when proxy broken. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Manifest verifies; screenshot of library window with titles from live server; RED when proxy broken. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-007 docs/gap-register-2026-11-23.md#gap-007---electron-desktop-no-guiwindow-run-packaged-artifacts-listed-in-artifactssha256-are-missing-from-evidence

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:39Z / 2026-11-23T21:22:39Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-044] Web reader renders plain text only; EPUB/PDF/CBZ not rendered (server exposes no text)

**Status:** Queued

**Type:** Feature

**Severity:** High

**Reopens:** 0

**Description:** Web reader renders plain text only; EPUB/PDF/CBZ not rendered (server exposes no text). Filed 2026-11-23 from the read-only gap audit (GAP-008, Product, severity High); no work started. EVIDENCE: supported-platforms.json web notes; evidence screenshots *reader-epub-unsupported.png; docs/formats.md render=NOT IMPLEMENTED for all formats ROOT CAUSE (UNCONFIRMED unless stated): Design gap: server has no content-extraction endpoint; HDM/pagination exists in core but is not exposed over HTTP. PROPOSED FIX: Expose /v1/files/:id/text or chapter/page endpoints from core HDM; web reader consumes; CBZ via PSE page images. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Unit (11.4.169(1)); Integration (11.4.169(2)); End-to-end (11.4.169(3)); UI (11.4.27 verbatim mandate); UX (11.4.27 verbatim mandate); Security (11.4.169(8)); Benchmark / performance (11.4.169(13)). Cross-links (11.4.214, both ways): HRD-022, HRD-032. Source: docs/gap-register-2026-11-23.md#gap-008---web-reader-renders-plain-text-only-epubpdfcbz-not-rendered-server-exposes-no-text. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: server+web.

**Affected scope:** Product gap GAP-008: Web reader renders plain text only; EPUB/PDF/CBZ not rendered (server exposes no text)

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-008---web-reader-renders-plain-text-only-epubpdfcbz-not-rendered-server-exposes-no-text; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: supported-platforms.json web notes; evidence screenshots *reader-epub-unsupported.png; docs/formats.md render=NOT IMPLEMENTED for all formats

**Acceptance:** EPUB and CBZ open in reader on 4 viewports with captured screenshots + DOM text asserted. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** EPUB and CBZ open in reader on 4 viewports with captured screenshots + DOM text asserted. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-008 docs/gap-register-2026-11-23.md#gap-008---web-reader-renders-plain-text-only-epubpdfcbz-not-rendered-server-exposes-no-text

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:39Z / 2026-11-23T21:22:39Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-045] Write-path throughput floor missed: MergeWork 152-176 ops/s vs 271.94 ops/s floor

**Status:** Queued

**Type:** Bug

**Severity:** High

**Reopens:** 0

**Description:** Write-path throughput floor missed: MergeWork 152-176 ops/s vs 271.94 ops/s floor. Filed 2026-11-23 from the read-only gap audit (GAP-010, Performance, severity High); no work started. EVIDENCE: docs/scalability-review.md lines 63-67 (host load ~17 on 16 cores) and line 429 (271.94 baseline) ROOT CAUSE (UNCONFIRMED unless stated): UNCONFIRMED: host contention (load 17) vs genuine regression after set-based upsert; floor was recorded single-worker. PROPOSED FIX: Re-measure on quiesced host with control arm in same run (control_needle), fix measurement protocol or code; do not weaken floor silently (§11.4.120). REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Benchmark / performance (11.4.169(13)); Stress + Chaos (11.4.169(9)); Scaling (11.4.27 verbatim mandate); Concurrency / atomicity (11.4.169(10)). Cross-links (11.4.214, both ways): HRD-027. Source: docs/gap-register-2026-11-23.md#gap-010---write-path-throughput-floor-missed-mergework-152-176-opss-vs-27194-opss-floor. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: server.

**Affected scope:** Performance gap GAP-010: Write-path throughput floor missed: MergeWork 152-176 ops/s vs 271.94 ops/s floor

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-010---write-path-throughput-floor-missed-mergework-152-176-opss-vs-27194-opss-floor; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: docs/scalability-review.md lines 63-67 (host load ~17 on 16 cores) and line 429 (271.94 baseline)

**Acceptance:** Same-run A/B shows >=floor or a documented, operator-ratified revised SLO. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Same-run A/B shows >=floor or a documented, operator-ratified revised SLO. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-010 docs/gap-register-2026-11-23.md#gap-010---write-path-throughput-floor-missed-mergework-152-176-opss-vs-27194-opss-floor

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:39Z / 2026-11-23T21:22:39Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-046] LCP never exercised against a real provider or third-party reader

**Status:** Queued

**Type:** Task

**Severity:** High

**Reopens:** 0

**Description:** LCP never exercised against a real provider or third-party reader. Filed 2026-11-23 from the read-only gap audit (GAP-011, Security, severity High); no work started. EVIDENCE: EVIDENCE.md lines 817,839,875 NOT PROVEN/UNCONFIRMED; lsd provider written in-repo from same spec text ROOT CAUSE (UNCONFIRMED unless stated): Self-referential test oracle (§11.4.245): producer and consumer share an author. PROPOSED FIX: Stand up Readium lcpserver (or EDRLab test provider) in a container as independent oracle; open produced artifact with Thorium/readium-lcp-client-in-container. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Integration (11.4.169(2)); Security (11.4.169(8)); End-to-end (11.4.169(3)); Interop (End-to-end 11.4.169(3) against third-party peer). Cross-links (11.4.214, both ways): HRD-031, HRD-026. Source: docs/gap-register-2026-11-23.md#gap-011---lcp-never-exercised-against-a-real-provider-or-third-party-reader. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: server.

**Affected scope:** Security gap GAP-011: LCP never exercised against a real provider or third-party reader

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-011---lcp-never-exercised-against-a-real-provider-or-third-party-reader; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: EVIDENCE.md lines 817,839,875 NOT PROVEN/UNCONFIRMED; lsd provider written in-repo from same spec text

**Acceptance:** Independent reader decrypts artifact; provider LSD register/return/renew round-trip logged; RED with wrong passphrase. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Independent reader decrypts artifact; provider LSD register/return/renew round-trip logged; RED with wrong passphrase. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-011 docs/gap-register-2026-11-23.md#gap-011---lcp-never-exercised-against-a-real-provider-or-third-party-reader

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:39Z / 2026-11-23T21:22:39Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-047] No Challenges-submodule or Security-submodule use in repo code, gates or scripts

**Status:** Queued

**Type:** Task

**Severity:** High

**Reopens:** 0

**Description:** No Challenges-submodule or Security-submodule use in repo code, gates or scripts. Filed 2026-11-23 from the read-only gap audit (GAP-012, Security, severity High); no work started. EVIDENCE: grep for challenges/security imports in server core scripts qa tools: 0 matches; submodules/challenges and submodules/security are pinned only ROOT CAUSE (UNCONFIRMED unless stated): Submodules vendored but not integrated; §11.4.169 types 5 and 8 depend on them. PROPOSED FIX: Create Challenges bank wrapping existing live tests (auth, OPDS, KOSync, LCP); run security submodule scanners (gosec/dep CVE) in container. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Challenges (11.4.169(5)); Security (11.4.169(8)). Cross-links (11.4.214, both ways): HRD-021. Source: docs/gap-register-2026-11-23.md#gap-012---no-challenges-submodule-or-security-submodule-use-in-repo-code-gates-or-scripts. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: qa.

**Affected scope:** Security gap GAP-012: No Challenges-submodule or Security-submodule use in repo code, gates or scripts

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-012---no-challenges-submodule-or-security-submodule-use-in-repo-code-gates-or-scripts; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: grep for challenges/security imports in server core scripts qa tools: 0 matches; submodules/challenges and submodules/security are pinned only

**Acceptance:** challenge run JSON with PASS only on captured evidence; security scan report with zero unaddressed High. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** challenge run JSON with PASS only on captured evidence; security scan report with zero unaddressed High. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-012 docs/gap-register-2026-11-23.md#gap-012---no-challenges-submodule-or-security-submodule-use-in-repo-code-gates-or-scripts

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:39Z / 2026-11-23T21:22:39Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-048] Memory / leak soak test type absent

**Status:** In progress

**Type:** Task

**Severity:** High

**Reopens:** 0

**Description:** Memory / leak soak test type absent. Filed 2026-11-23 from the read-only gap audit (GAP-013, Test, severity High); no work started. EVIDENCE: grep Benchmark: 3 funcs; no soak/pprof/RSS artifacts under evidence/ (ls evidence) ROOT CAUSE (UNCONFIRMED unless stated): Never authored. PROPOSED FIX: Add 24h/N-iteration soak under Containers with pprof heap diff and RSS ceilings; web (heap snapshot) and Android (dumpsys meminfo). REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Memory (11.4.169(12)); Stress + Chaos (11.4.169(9)). Cross-links (11.4.214, both ways): HRD-027. Source: docs/gap-register-2026-11-23.md#gap-013---memory--leak-soak-test-type-absent. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: server+qa.

**Affected scope:** Test gap GAP-013: Memory / leak soak test type absent

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-013---memory--leak-soak-test-type-absent; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: grep Benchmark: 3 funcs; no soak/pprof/RSS artifacts under evidence/ (ls evidence)

**Acceptance:** Soak JSON: RSS slope ~0, heap diff bounded; RED with injected leak. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Soak JSON: RSS slope ~0, heap diff bounded; RED with injected leak. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-013 docs/gap-register-2026-11-23.md#gap-013---memory--leak-soak-test-type-absent

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:39Z / 2026-11-23T22:50:14Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded
- 2026-11-23 — Started by AI: Work in progress with independent review; uncommitted, not merged; evidence: evidence/campaign-2026-11-23/gap-fix/hrd048

## [HRD-049] Race/deadlock evidence is stale: go-race.log dated 2026-11-20, before LCP/OPDS/PSE/LSD code

**Status:** In progress

**Type:** Task

**Severity:** High

**Reopens:** 0

**Description:** Race/deadlock evidence is stale: go-race.log dated 2026-11-20, before LCP/OPDS/PSE/LSD code. Filed 2026-11-23 from the read-only gap audit (GAP-014, Test, severity High); no work started. EVIDENCE: evidence/campaign-2026-11-20/{core,server}/go-race.log; git status shows new core/pkg/lcp/*.go untracked ROOT CAUSE (UNCONFIRMED unless stated): Not re-run after later changes. PROPOSED FIX: Re-run `go test -race ./...` core+server with live PG/Redis; store under campaign-2026-11-23. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Race / deadlock (11.4.169(11)); Concurrency / atomicity (11.4.169(10)). Cross-links (11.4.214, both ways): HRD-012. Source: docs/gap-register-2026-11-23.md#gap-014---racedeadlock-evidence-is-stale-go-racelog-dated-2026-11-20-before-lcpopdspselsd-code. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: server.

**Affected scope:** Test gap GAP-014: Race/deadlock evidence is stale: go-race.log dated 2026-11-20, before LCP/OPDS/PSE/LSD code

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-014---racedeadlock-evidence-is-stale-go-racelog-dated-2026-11-20-before-lcpopdspselsd-code; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: evidence/campaign-2026-11-20/{core,server}/go-race.log; git status shows new core/pkg/lcp/*.go untracked

**Acceptance:** go-race.log exit 0 on current HEAD+worktree, sha256 recorded. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** go-race.log exit 0 on current HEAD+worktree, sha256 recorded. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-014 docs/gap-register-2026-11-23.md#gap-014---racedeadlock-evidence-is-stale-go-racelog-dated-2026-11-20-before-lcpopdspselsd-code

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:39Z / 2026-11-23T22:50:15Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded
- 2026-11-23 — Started by AI: Work in progress with independent review; uncommitted, not merged; evidence: evidence/campaign-2026-11-23/gap-fix/hrd049

## [HRD-050] 57 real t.Skip/Skipf calls (the earlier grep of 83 over-counted report.Skipped); 38 are gated on HELIX_* env vars not set in any standing suite (PG DSN 18, LOAD_RUN 11, REDIS 7, others)

**Status:** Queued

**Type:** Task

**Severity:** High

**Reopens:** 0

**Description:** 57 real t.Skip/Skipf calls (the earlier grep of 83 over-counted report.Skipped); 38 are gated on HELIX_* env vars not set in any standing suite (PG DSN 18, LOAD_RUN 11, REDIS 7, others). Filed 2026-11-23 from the read-only gap audit (GAP-015, Test, severity High); no work started. EVIDENCE: commands.txt: `grep -rn t.Skipf\?( server core` = 57, 38 with HELIX_ gates ROOT CAUSE (UNCONFIRMED unless stated): Live tests are opt-in; no single runner proves all gated tests execute (0 skips) in one pass. PROPOSED FIX: Add scripts/run_all_live.sh that boots PG+Redis via Containers, exports vars, runs with -v, fails if any gated test SKIPs. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Full-automation (11.4.169(4)); Integration (11.4.169(2)). Cross-links (11.4.214, both ways): HRD-015. Source: docs/gap-register-2026-11-23.md#gap-015---57-real-tskipskipf-calls-the-earlier-grep-of-83-over-counted-reportskipped-38-are-gated-on-helix_-env-vars-not-set-in-any-standing-suite-pg-dsn-18-load_run-11-redis-7-others. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: qa.

**Affected scope:** Test gap GAP-015: 57 real t.Skip/Skipf calls (the earlier grep of 83 over-counted report.Skipped); 38 are gated on HELIX_* env vars not set in any standing suite (PG DSN 18, LOAD_RUN 11, REDIS 7, others)

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-015---57-real-tskipskipf-calls-the-earlier-grep-of-83-over-counted-reportskipped-38-are-gated-on-helix_-env-vars-not-set-in-any-standing-suite-pg-dsn-18-load_run-11-redis-7-others; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: commands.txt: `grep -rn t.Skipf\?( server core` = 57, 38 with HELIX_ gates

**Acceptance:** Run log: 0 SKIP for gated tests; skip-count gate with RED baseline. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Run log: 0 SKIP for gated tests; skip-count gate with RED baseline. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-015 docs/gap-register-2026-11-23.md#gap-015---57-real-tskipskipf-calls-the-earlier-grep-of-83-over-counted-reportskipped-38-are-gated-on-helix_-env-vars-not-set-in-any-standing-suite-pg-dsn-18-load_run-11-redis-7-others

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:39Z / 2026-11-23T21:22:39Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-051] No claim-vs-reality ledger (§11.4.266) and no gate-evidence manifest (§11.4.262)

**Status:** In progress

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** No claim-vs-reality ledger (§11.4.266) and no gate-evidence manifest (§11.4.262). Filed 2026-11-23 from the read-only gap audit (GAP-016, Test, severity Medium); no work started. EVIDENCE: Gates cm_ledger_row_typed... and cm_machine_evidence... PASS only vacuously: docs/claim_reality_ledger.tsv and docs/evidence/gate_evidence_manifest.tsv absent (gates_run.txt) ROOT CAUSE (UNCONFIRMED unless stated): Never created; vacuous PASS masks the gap (§11.4.201). PROPOSED FIX: Create ledger rows for every advertised capability (formats matrix, platforms, OPDS, KOSync, LCP) typed from closed vocabulary. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Full-automation (11.4.169(4)). Cross-links (11.4.214, both ways): HRD-059 (GAP-024). Source: docs/gap-register-2026-11-23.md#gap-016---no-claim-vs-reality-ledger-114266-and-no-gate-evidence-manifest-114262. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: qa.

**Affected scope:** Test gap GAP-016: No claim-vs-reality ledger (§11.4.266) and no gate-evidence manifest (§11.4.262)

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-016---no-claim-vs-reality-ledger-114266-and-no-gate-evidence-manifest-114262; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: Gates cm_ledger_row_typed... and cm_machine_evidence... PASS only vacuously: docs/claim_reality_ledger.tsv and docs/evidence/gate_evidence_manifest.tsv absent (gates_run.txt)

**Acceptance:** Ledger present; each row cites a passing challenge; gate non-vacuous. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Ledger present; each row cites a passing challenge; gate non-vacuous. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-016 docs/gap-register-2026-11-23.md#gap-016---no-claim-vs-reality-ledger-114266-and-no-gate-evidence-manifest-114262

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:40Z / 2026-11-23T22:50:15Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded
- 2026-11-23 — Started by AI: Work in progress with independent review; uncommitted, not merged; evidence: evidence/campaign-2026-11-23/gap-fix/hrd051_red.txt; evidence/campaign-2026-11-23/gap-fix/hrd051_green.txt; evidence/campaign-2026-11-23/gap-fix/hrd051_mutations.txt

## [HRD-052] helix-deps.yaml drift against gitlinks (4 DRIFT: llmctl, containers, claude_toolkit, llama.cpp)

**Status:** In progress

**Type:** Task

**Severity:** High

**Reopens:** 0

**Description:** helix-deps.yaml drift against gitlinks (4 DRIFT: llmctl, containers, claude_toolkit, llama.cpp). Filed 2026-11-23 from the read-only gap audit (GAP-017, Governance, severity High); no work started. EVIDENCE: gates_run.txt: CM-HELIX-DEPS-REF-MATCHES-GITLINK FAIL (27 deps, 4 DRIFT); root has gitlink submodules/claude_toolkit without .gitmodules mapping (git submodule status --recursive fatal) ROOT CAUSE (UNCONFIRMED unless stated): UNCONFIRMED whether drift is root manifest or nested constitution manifests; claude_toolkit gitlink unmapped is documented in docs/upstream-issues/claude_toolkit-challenges-gitlink.md. PROPOSED FIX: Operator-gated: refresh manifests or pins; fix .gitmodules mapping via fetch+merge on main (no force). REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Full-automation (11.4.169(4)). Cross-links (11.4.214, both ways): HRD-021. Source: docs/gap-register-2026-11-23.md#gap-017---helix-depsyaml-drift-against-gitlinks-4-drift-llmctl-containers-claude_toolkit-llamacpp. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: operator.

**Affected scope:** Governance gap GAP-017: helix-deps.yaml drift against gitlinks (4 DRIFT: llmctl, containers, claude_toolkit, llama.cpp)

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-017---helix-depsyaml-drift-against-gitlinks-4-drift-llmctl-containers-claude_toolkit-llamacpp; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: gates_run.txt: CM-HELIX-DEPS-REF-MATCHES-GITLINK FAIL (27 deps, 4 DRIFT); root has gitlink submodules/claude_toolkit without .gitmodules mapping (git submodule status --recursive fatal)

**Acceptance:** Gate exit 0 on real tree. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Gate exit 0 on real tree. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-017 docs/gap-register-2026-11-23.md#gap-017---helix-depsyaml-drift-against-gitlinks-4-drift-llmctl-containers-claude_toolkit-llamacpp

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:40Z / 2026-11-23T22:50:15Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded
- 2026-11-23 — Started by AI: Work in progress with independent review; uncommitted, not merged; evidence: evidence/campaign-2026-11-23/gap-fix/hrd052_red.txt; evidence/campaign-2026-11-23/gap-fix/hrd052_green.txt

## [HRD-053] Submodule pins not on main: doc_processor diverged; llm_orchestrator/llm_provider/vision_engine pins ahead of main; containers 25 commits behind

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** Submodule pins not on main: doc_processor diverged; llm_orchestrator/llm_provider/vision_engine pins ahead of main; containers 25 commits behind. Filed 2026-11-23 from the read-only gap audit (GAP-018, Governance, severity Medium); no work started. EVIDENCE: docs/submodule-pins-on-main-2026-11-23.md table ROOT CAUSE (UNCONFIRMED unless stated): Upstream mains lag masters; containers new commits may include matrix improvements. PROPOSED FIX: Operator-approved fast-forward of containers; upstream FF of main for three; HelixQA/Containers are PROVENANCE 'scratch clones outside repo' for several deps. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Integration (11.4.169(2)). Cross-links (11.4.214, both ways): HRD-021. Source: docs/gap-register-2026-11-23.md#gap-018---submodule-pins-not-on-main-doc_processor-diverged-llm_orchestratorllm_providervision_engine-pins-ahead-of-main-containers-25-commits-behind. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: operator.

**Affected scope:** Governance gap GAP-018: Submodule pins not on main: doc_processor diverged; llm_orchestrator/llm_provider/vision_engine pins ahead of main; containers 25 commits behind

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-018---submodule-pins-not-on-main-doc_processor-diverged-llm_orchestratorllm_providervision_engine-pins-ahead-of-main-containers-25-commits-behind; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: docs/submodule-pins-on-main-2026-11-23.md table

**Acceptance:** Pins on main per §11.4.288. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Pins on main per §11.4.288. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-018 docs/gap-register-2026-11-23.md#gap-018---submodule-pins-not-on-main-doc_processor-diverged-llm_orchestratorllm_providervision_engine-pins-ahead-of-main-containers-25-commits-behind

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:40Z / 2026-11-23T21:22:40Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-054] helixqa was built in a scratch tree outside the repo; several deps are depth-1 scratch clones not submodules

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** helixqa was built in a scratch tree outside the repo; several deps are depth-1 scratch clones not submodules. Filed 2026-11-23 from the read-only gap audit (GAP-019, Build, severity Medium); no work started. EVIDENCE: evidence/campaign-2026-11-23/helixqa/PROVENANCE.txt ROOT CAUSE (UNCONFIRMED unless stated): helixqa go.mod replace targets (doc_processor, llm_*, security, vision_engine) are not satisfied by repo submodule layout; qa/helixqa/run_helixqa_banks.sh needs scratch build. PROPOSED FIX: Make `helixqa` build reproducible from repo (go.work or Containers build image) and record binary sha256 in-tree. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Full-automation (11.4.169(4)); Integration (11.4.169(2)). Cross-links (11.4.214, both ways): HRD-021. Source: docs/gap-register-2026-11-23.md#gap-019---helixqa-was-built-in-a-scratch-tree-outside-the-repo-several-deps-are-depth-1-scratch-clones-not-submodules. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: qa.

**Affected scope:** Build gap GAP-019: helixqa was built in a scratch tree outside the repo; several deps are depth-1 scratch clones not submodules

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-019---helixqa-was-built-in-a-scratch-tree-outside-the-repo-several-deps-are-depth-1-scratch-clones-not-submodules; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: evidence/campaign-2026-11-23/helixqa/PROVENANCE.txt

**Acceptance:** Fresh clone + one script yields helixqa binary identical sha256 twice (§11.4.246). Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Fresh clone + one script yields helixqa binary identical sha256 twice (§11.4.246). Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-019 docs/gap-register-2026-11-23.md#gap-019---helixqa-was-built-in-a-scratch-tree-outside-the-repo-several-deps-are-depth-1-scratch-clones-not-submodules

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:40Z / 2026-11-23T21:22:40Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-055] gofmt drift in core (hdm/hdm.go, hlc/hlc.go, pkg/crypto/crypto.go, pkg/source/source.go, source_test.go)

**Status:** In progress

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** gofmt drift in core (hdm/hdm.go, hlc/hlc.go, pkg/crypto/crypto.go, pkg/source/source.go, source_test.go). Filed 2026-11-23 from the read-only gap audit (GAP-020, Quality, severity Medium); no work started. EVIDENCE: `gofmt -l core` output (commands_raw.txt) ROOT CAUSE (UNCONFIRMED unless stated): Files edited without format step; no format gate. PROPOSED FIX: gofmt -w those files + add gofmt/vet gate to local runner. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Unit (11.4.169(1)) as static gate. Cross-links (11.4.214, both ways): none. Source: docs/gap-register-2026-11-23.md#gap-020---gofmt-drift-in-core-hdmhdmgo-hlchlcgo-pkgcryptocryptogo-pkgsourcesourcego-source_testgo. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: core.

**Affected scope:** Quality gap GAP-020: gofmt drift in core (hdm/hdm.go, hlc/hlc.go, pkg/crypto/crypto.go, pkg/source/source.go, source_test.go)

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-020---gofmt-drift-in-core-hdmhdmgo-hlchlcgo-pkgcryptocryptogo-pkgsourcesourcego-source_testgo; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: `gofmt -l core` output (commands_raw.txt)

**Acceptance:** gofmt -l empty. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** gofmt -l empty. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-020 docs/gap-register-2026-11-23.md#gap-020---gofmt-drift-in-core-hdmhdmgo-hlchlcgo-pkgcryptocryptogo-pkgsourcesourcego-source_testgo

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:40Z / 2026-11-23T22:50:15Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded
- 2026-11-23 — Started by AI: Work in progress with independent review; uncommitted, not merged; evidence: evidence/campaign-2026-11-23/gap-fix/hrd055_red.txt; evidence/campaign-2026-11-23/gap-fix/hrd055_green.txt

## [HRD-056] Header oracles missing (CSP, X-Content-Type-Options, HSTS, Cache-Control) -- only 1 non-test occurrence in server

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** Header oracles missing (CSP, X-Content-Type-Options, HSTS, Cache-Control) -- only 1 non-test occurrence in server. Filed 2026-11-23 from the read-only gap audit (GAP-021, Quality, severity Medium); no work started. EVIDENCE: grep count = 1 (commands_part2.txt) ROOT CAUSE (UNCONFIRMED unless stated): Server does not set security headers for HTML/JSON/OPDS responses (UNCONFIRMED per route). PROPOSED FIX: Add header middleware + table-driven header oracle in httpapi tests and HelixQA bank asserting headers per route. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Security (11.4.169(8)); Unit (11.4.169(1)); HelixQA (11.4.169(6)). Cross-links (11.4.214, both ways): HRD-021. Source: docs/gap-register-2026-11-23.md#gap-021---header-oracles-missing-csp-x-content-type-options-hsts-cache-control----only-1-non-test-occurrence-in-server. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: server.

**Affected scope:** Quality gap GAP-021: Header oracles missing (CSP, X-Content-Type-Options, HSTS, Cache-Control) -- only 1 non-test occurrence in server

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-021---header-oracles-missing-csp-x-content-type-options-hsts-cache-control----only-1-non-test-occurrence-in-server; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: grep count = 1 (commands_part2.txt)

**Acceptance:** Every route family has asserted headers; mutant removing middleware is RED. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Every route family has asserted headers; mutant removing middleware is RED. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-021 docs/gap-register-2026-11-23.md#gap-021---header-oracles-missing-csp-x-content-type-options-hsts-cache-control----only-1-non-test-occurrence-in-server

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:40Z / 2026-11-23T21:22:40Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-057] Postgres paths unrun for LCP/GC refs

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** Postgres paths unrun for LCP/GC refs. Filed 2026-11-23 from the read-only gap audit (GAP-022, Test, severity Medium); no work started. EVIDENCE: EVIDENCE.md line 875 'PostgreSQL-backed catalog for this flow (columns pre-exist and are mapped, not exercised live)' ROOT CAUSE (UNCONFIRMED unless stated): Live PG tests exist for catalog but not the LCP columns / GC reference logic. PROPOSED FIX: Add live PG tests for LCP license persistence + GC; run in all_live runner. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Integration (11.4.169(2)); Concurrency / atomicity (11.4.169(10)). Cross-links (11.4.214, both ways): HRD-031, HRD-015, HRD-050 (GAP-015). Source: docs/gap-register-2026-11-23.md#gap-022---postgres-paths-unrun-for-lcpgc-refs. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: server.

**Affected scope:** Test gap GAP-022: Postgres paths unrun for LCP/GC refs

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-022---postgres-paths-unrun-for-lcpgc-refs; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: EVIDENCE.md line 875 'PostgreSQL-backed catalog for this flow (columns pre-exist and are mapped, not exercised live)'

**Acceptance:** Live PG LCP test PASS with RED mutant. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Live PG LCP test PASS with RED mutant. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-022 docs/gap-register-2026-11-23.md#gap-022---postgres-paths-unrun-for-lcpgc-refs

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:40Z / 2026-11-23T21:22:40Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-058] ID column still named atm_id in SQLite registry and exports

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** ID column still named atm_id in SQLite registry and exports. Filed 2026-11-23 from the read-only gap audit (GAP-023, Product, severity Medium); no work started. EVIDENCE: evidence/ID-KEY-RENAME.md line 21; PRAGMA table_info(items) shows atm_id ROOT CAUSE (UNCONFIRMED unless stated): Constitution lists atm_id as a reserved name; HRD key adopted at presentation only. PROPOSED FIX: Operator decision on schema rename vs mapping layer; add test locking the mapping. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Unit (11.4.169(1)). Cross-links (11.4.214, both ways): HRD-028. Source: docs/gap-register-2026-11-23.md#gap-023---id-column-still-named-atm_id-in-sqlite-registry-and-exports. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: operator.

**Affected scope:** Product gap GAP-023: ID column still named atm_id in SQLite registry and exports

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-023---id-column-still-named-atm_id-in-sqlite-registry-and-exports; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: evidence/ID-KEY-RENAME.md line 21; PRAGMA table_info(items) shows atm_id

**Acceptance:** Decision recorded; tests pin it. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Decision recorded; tests pin it. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-023 docs/gap-register-2026-11-23.md#gap-023---id-column-still-named-atm_id-in-sqlite-registry-and-exports

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:40Z / 2026-11-23T21:22:40Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-059] Documentation drift: stale 'not implemented' / 'not yet' claims

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** Documentation drift: stale 'not implemented' / 'not yet' claims. Filed 2026-11-23 from the read-only gap audit (GAP-024, Docs, severity Medium); no work started. EVIDENCE: docs/architecture.md:25 says LCP fulfillment not shown as connected; docs/formats.md:65 says LSD handling absent; docs/user_guide.md vs HRD-031 'Queued' in registry while EVIDENCE shows LCP/PSE delivered; Issues.md HRD-021 status 'In progress' though bank exists (greps in commands_raw.txt; registry listing in this audit) ROOT CAUSE (UNCONFIRMED unless stated): No automated doc-claim check (§11.4.266). PROPOSED FIX: Ledger (GAP-016) + a test that greps docs for 'NOT IMPLEMENTED' per capability and compares to code capability flags. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Unit (11.4.169(1)); Full-automation (11.4.169(4)). Cross-links (11.4.214, both ways): HRD-031, HRD-021, HRD-051 (GAP-016). Source: docs/gap-register-2026-11-23.md#gap-024---documentation-drift-stale-not-implemented--not-yet-claims. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: docs.

**Affected scope:** Docs gap GAP-024: Documentation drift: stale 'not implemented' / 'not yet' claims

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-024---documentation-drift-stale-not-implemented--not-yet-claims; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: docs/architecture.md:25 says LCP fulfillment not shown as connected; docs/formats.md:65 says LSD handling absent; docs/user_guide.md vs HRD-031 'Queued' in registry while EVIDENCE shows LCP/PSE delivered; Issues.md HRD-021 status 'In progress' though bank exists (greps in commands_raw.txt; registry listing in this audit)

**Acceptance:** Doc-vs-code test GREEN; RED when a flag flips. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Doc-vs-code test GREEN; RED when a flag flips. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-024 docs/gap-register-2026-11-23.md#gap-024---documentation-drift-stale-not-implemented--not-yet-claims

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:41Z / 2026-11-23T21:22:41Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-060] Mutation coverage is hand-listed (110 mutants) and spans server/core only; clients and tools have 1 mutation each

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** Mutation coverage is hand-listed (110 mutants) and spans server/core only; clients and tools have 1 mutation each. Filed 2026-11-23 from the read-only gap audit (GAP-025, Test, severity Medium); no work started. EVIDENCE: scripts/verify_go_mutation.py ids=110; evidence/ui-mutation: 1 mutant; firebase mutations.log ROOT CAUSE (UNCONFIRMED unless stated): Mutants authored per defect, no automated mutation scoring (§11.4.248 gate cm_mutation_score_on_diff unused). PROPOSED FIX: Adopt go-mutesting or gremlins on changed lines; mutants for web TS (stryker) and desktop JS. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Unit (11.4.169(1)); Full-automation (11.4.169(4)). Cross-links (11.4.214, both ways): HRD-014. Source: docs/gap-register-2026-11-23.md#gap-025---mutation-coverage-is-hand-listed-110-mutants-and-spans-servercore-only-clients-and-tools-have-1-mutation-each. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: qa.

**Affected scope:** Test gap GAP-025: Mutation coverage is hand-listed (110 mutants) and spans server/core only; clients and tools have 1 mutation each

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-025---mutation-coverage-is-hand-listed-110-mutants-and-spans-servercore-only-clients-and-tools-have-1-mutation-each; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: scripts/verify_go_mutation.py ids=110; evidence/ui-mutation: 1 mutant; firebase mutations.log

**Acceptance:** Mutation score on diff >= threshold file; killed-map recorded. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Mutation score on diff >= threshold file; killed-map recorded. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-025 docs/gap-register-2026-11-23.md#gap-025---mutation-coverage-is-hand-listed-110-mutants-and-spans-servercore-only-clients-and-tools-have-1-mutation-each

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:41Z / 2026-11-23T21:22:41Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-061] No cross-browser run (Chromium only); no Firefox/WebKit viewport runs

**Status:** Queued

**Type:** Feature

**Severity:** Medium

**Reopens:** 0

**Description:** No cross-browser run (Chromium only); no Firefox/WebKit viewport runs. Filed 2026-11-23 from the read-only gap audit (GAP-026, Platform, severity Medium); no work started. EVIDENCE: verify_ui.py:102 and live.mjs:16 launch chromium only ROOT CAUSE (UNCONFIRMED unless stated): Playwright browsers installed only for chromium; helixqa web executor CDP is Chromium-centric. PROPOSED FIX: Add playwright firefox+webkit projects in container; run 16 checks x 4 viewports x 3 engines. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): UI (11.4.27 verbatim mandate); UX (11.4.27 verbatim mandate); End-to-end (11.4.169(3)). Cross-links (11.4.214, both ways): HRD-022, HRD-038 (GAP-002). Source: docs/gap-register-2026-11-23.md#gap-026---no-cross-browser-run-chromium-only-no-firefoxwebkit-viewport-runs. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: web.

**Affected scope:** Platform gap GAP-026: No cross-browser run (Chromium only); no Firefox/WebKit viewport runs

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-026---no-cross-browser-run-chromium-only-no-firefoxwebkit-viewport-runs; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: verify_ui.py:102 and live.mjs:16 launch chromium only

**Acceptance:** 48 check-sets PASS with per-engine screenshots. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** 48 check-sets PASS with per-engine screenshots. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-026 docs/gap-register-2026-11-23.md#gap-026---no-cross-browser-run-chromium-only-no-firefoxwebkit-viewport-runs

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:41Z / 2026-11-23T21:22:41Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-062] Device/screen configurations unexercised: RTL, large font, high contrast, reduced motion, density, foldable, landscape on web

**Status:** Queued

**Type:** Feature

**Severity:** Medium

**Reopens:** 0

**Description:** Device/screen configurations unexercised: RTL, large font, high contrast, reduced motion, density, foldable, landscape on web. Filed 2026-11-23 from the read-only gap audit (GAP-027, Platform, severity Medium); no work started. EVIDENCE: rtl hits 0 in verify_ui.py and live.mjs; android evidence has portrait 1080x1920 and one landscape only ROOT CAUSE (UNCONFIRMED unless stated): Checks cover 4 viewports x light/dark only. PROPOSED FIX: Add config axes to verify_ui.py and android matrix (GAP-004). REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): UI (11.4.27 verbatim mandate); UX (11.4.27 verbatim mandate); Accessibility (UX axis, cross-cutting). Cross-links (11.4.214, both ways): HRD-022, HRD-040 (GAP-004). Source: docs/gap-register-2026-11-23.md#gap-027---devicescreen-configurations-unexercised-rtl-large-font-high-contrast-reduced-motion-density-foldable-landscape-on-web. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: web+qa.

**Affected scope:** Platform gap GAP-027: Device/screen configurations unexercised: RTL, large font, high contrast, reduced motion, density, foldable, landscape on web

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-027---devicescreen-configurations-unexercised-rtl-large-font-high-contrast-reduced-motion-density-foldable-landscape-on-web; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: rtl hits 0 in verify_ui.py and live.mjs; android evidence has portrait 1080x1920 and one landscape only

**Acceptance:** Per-axis result rows with screenshots. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Per-axis result rows with screenshots. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-027 docs/gap-register-2026-11-23.md#gap-027---devicescreen-configurations-unexercised-rtl-large-font-high-contrast-reduced-motion-density-foldable-landscape-on-web

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:41Z / 2026-11-23T21:22:41Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-063] e-ink profile NOT_STARTED; HarmonyOS and Aurora BLOCKED by missing SDKs

**Status:** Queued

**Type:** Feature

**Severity:** Medium

**Reopens:** 0

**Description:** e-ink profile NOT_STARTED; HarmonyOS and Aurora BLOCKED by missing SDKs. Filed 2026-11-23 from the read-only gap audit (GAP-028, Platform, severity Medium); no work started. EVIDENCE: supported-platforms.json ROOT CAUSE (UNCONFIRMED unless stated): SDKs not installed; e-ink needs low-refresh theme + CSS profile. PROPOSED FIX: Implement e-ink CSS profile on web (testable headless via screenshot diff, greyscale); SDK installs in container for harmony/aurora per operator. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): UI (11.4.27 verbatim mandate); UX (11.4.27 verbatim mandate). Cross-links (11.4.214, both ways): HRD-022. Source: docs/gap-register-2026-11-23.md#gap-028---e-ink-profile-not_started-harmonyos-and-aurora-blocked-by-missing-sdks. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: web.

**Affected scope:** Platform gap GAP-028: e-ink profile NOT_STARTED; HarmonyOS and Aurora BLOCKED by missing SDKs

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-028---e-ink-profile-not_started-harmonyos-and-aurora-blocked-by-missing-sdks; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: supported-platforms.json

**Acceptance:** e-ink profile screenshots pass greyscale/contrast oracle. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** e-ink profile screenshots pass greyscale/contrast oracle. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-028 docs/gap-register-2026-11-23.md#gap-028---e-ink-profile-not_started-harmonyos-and-aurora-blocked-by-missing-sdks

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:41Z / 2026-11-23T21:22:41Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-064] Web unit coverage small: 5 test files for 17 source files; no coverage number

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** Web unit coverage small: 5 test files for 17 source files; no coverage number. Filed 2026-11-23 from the read-only gap audit (GAP-029, Test, severity Medium); no work started. EVIDENCE: commands_part2.txt (web src 17, tests 5) ROOT CAUSE (UNCONFIRMED unless stated): No vitest coverage run recorded. PROPOSED FIX: Run vitest --coverage; add tests for App.tsx, Settings, Reader paths. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Unit (11.4.169(1)). Cross-links (11.4.214, both ways): HRD-022. Source: docs/gap-register-2026-11-23.md#gap-029---web-unit-coverage-small-5-test-files-for-17-source-files-no-coverage-number. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: web.

**Affected scope:** Test gap GAP-029: Web unit coverage small: 5 test files for 17 source files; no coverage number

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-029---web-unit-coverage-small-5-test-files-for-17-source-files-no-coverage-number; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: commands_part2.txt (web src 17, tests 5)

**Acceptance:** Coverage report archived with threshold. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Coverage report archived with threshold. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-029 docs/gap-register-2026-11-23.md#gap-029---web-unit-coverage-small-5-test-files-for-17-source-files-no-coverage-number

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:41Z / 2026-11-23T21:22:41Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-065] Server coverage 60.1% without live services; no combined live coverage number

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** Server coverage 60.1% without live services; no combined live coverage number. Filed 2026-11-23 from the read-only gap audit (GAP-030, Test, severity Medium); no work started. EVIDENCE: EVIDENCE.md baseline row ROOT CAUSE (UNCONFIRMED unless stated): Live coverage never merged. PROPOSED FIX: Run with PG+Redis and -coverprofile merge; archive. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Unit (11.4.169(1)); Integration (11.4.169(2)). Cross-links (11.4.214, both ways): HRD-015, HRD-012. Source: docs/gap-register-2026-11-23.md#gap-030---server-coverage-601-without-live-services-no-combined-live-coverage-number. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: server.

**Affected scope:** Test gap GAP-030: Server coverage 60.1% without live services; no combined live coverage number

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-030---server-coverage-601-without-live-services-no-combined-live-coverage-number; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: EVIDENCE.md baseline row

**Acceptance:** Merged coverage.out archived. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Merged coverage.out archived. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-030 docs/gap-register-2026-11-23.md#gap-030---server-coverage-601-without-live-services-no-combined-live-coverage-number

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:42Z / 2026-11-23T21:22:42Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-066] Interop with real devices unproven: KOReader firmware, Kobo, physical Android phone/tablet

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** Interop with real devices unproven: KOReader firmware, Kobo, physical Android phone/tablet. Filed 2026-11-23 from the read-only gap audit (GAP-031, Test, severity Medium); no work started. EVIDENCE: docs/user_guide.md:85; supported-platforms.json android notes (emulator only) ROOT CAUSE (UNCONFIRMED unless stated): No physical device in lab; KOReader tested as Lua parser only. PROPOSED FIX: Operator-attended phase per §11.4.185 plus emulator-hosted KOReader (Linux AppImage in container) as automated stand-in. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): End-to-end (11.4.169(3)); Interop (End-to-end 11.4.169(3) against third-party peer). Cross-links (11.4.214, both ways): HRD-026. Source: docs/gap-register-2026-11-23.md#gap-031---interop-with-real-devices-unproven-koreader-firmware-kobo-physical-android-phonetablet. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: operator+qa.

**Affected scope:** Test gap GAP-031: Interop with real devices unproven: KOReader firmware, Kobo, physical Android phone/tablet

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-031---interop-with-real-devices-unproven-koreader-firmware-kobo-physical-android-phonetablet; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: docs/user_guide.md:85; supported-platforms.json android notes (emulator only)

**Acceptance:** Real KOReader client syncs progress with server in container. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Real KOReader client syncs progress with server in container. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-031 docs/gap-register-2026-11-23.md#gap-031---interop-with-real-devices-unproven-koreader-firmware-kobo-physical-android-phonetablet

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:42Z / 2026-11-23T21:22:42Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-067] DDoS/slow-loris/flood and chaos coverage is campaign-run, not standing

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** DDoS/slow-loris/flood and chaos coverage is campaign-run, not standing. Filed 2026-11-23 from the read-only gap audit (GAP-032, Test, severity Medium); no work started. EVIDENCE: evidence/campaign-2026-11-23/slow-client, evidence/live-server-daemon-restart; no scripts in a standing suite ROOT CAUSE (UNCONFIRMED unless stated): One-off harness runs gated by HELIX_LOAD_RUN. PROPOSED FIX: Fold into run_all_live.sh with thresholds; add network chaos via Containers network package (tc/netem). REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): DDoS / load-flood (11.4.169(7)); Stress + Chaos (11.4.169(9)). Cross-links (11.4.214, both ways): HRD-027, HRD-050 (GAP-015). Source: docs/gap-register-2026-11-23.md#gap-032---ddosslow-lorisflood-and-chaos-coverage-is-campaign-run-not-standing. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: server+qa.

**Affected scope:** Test gap GAP-032: DDoS/slow-loris/flood and chaos coverage is campaign-run, not standing

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-032---ddosslow-lorisflood-and-chaos-coverage-is-campaign-run-not-standing; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: evidence/campaign-2026-11-23/slow-client, evidence/live-server-daemon-restart; no scripts in a standing suite

**Acceptance:** Standing run produces results.json with pass/fail thresholds. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Standing run produces results.json with pass/fail thresholds. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-032 docs/gap-register-2026-11-23.md#gap-032---ddosslow-lorisflood-and-chaos-coverage-is-campaign-run-not-standing

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:42Z / 2026-11-23T21:22:42Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-068] HelixTrack Core defects D1-D5 (default JWT secret, password logging) drafted but not filed; local instance patched with schema SQL

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** HelixTrack Core defects D1-D5 (default JWT secret, password logging) drafted but not filed; local instance patched with schema SQL. Filed 2026-11-23 from the read-only gap audit (GAP-033, Security, severity Medium); no work started. EVIDENCE: docs/upstream-issues/helixtrack-core-defects.md ROOT CAUSE (UNCONFIRMED unless stated): Upstream issues unfiled. PROPOSED FIX: File upstream after operator approval; keep local instance isolated. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Security (11.4.169(8)). Cross-links (11.4.214, both ways): HRD-029. Source: docs/gap-register-2026-11-23.md#gap-033---helixtrack-core-defects-d1-d5-default-jwt-secret-password-logging-drafted-but-not-filed-local-instance-patched-with-schema-sql. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: operator.

**Affected scope:** Security gap GAP-033: HelixTrack Core defects D1-D5 (default JWT secret, password logging) drafted but not filed; local instance patched with schema SQL

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-033---helixtrack-core-defects-d1-d5-default-jwt-secret-password-logging-drafted-but-not-filed-local-instance-patched-with-schema-sql; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: docs/upstream-issues/helixtrack-core-defects.md

**Acceptance:** Issues filed with links. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Issues filed with links. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-033 docs/gap-register-2026-11-23.md#gap-033---helixtrack-core-defects-d1-d5-default-jwt-secret-password-logging-drafted-but-not-filed-local-instance-patched-with-schema-sql

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:42Z / 2026-11-23T21:22:42Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-069] Interactive demo has no platform-matrix automation or HelixQA bank

**Status:** Queued

**Type:** Feature

**Severity:** Medium

**Reopens:** 0

**Description:** Interactive demo has no platform-matrix automation or HelixQA bank. Filed 2026-11-23 from the read-only gap audit (GAP-034, Platform, severity Medium); no work started. EVIDENCE: demos/interactive_static_demo present; HRD-023 Queued; no evidence dir for it ROOT CAUSE (UNCONFIRMED unless stated): Never covered by a bank; docs/architecture.md says web prototype does not call API. PROPOSED FIX: Build/serve demo in container, add bank with screenshots at 4 viewports + vision read. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): UI (11.4.27 verbatim mandate); UX (11.4.27 verbatim mandate); HelixQA (11.4.169(6)). Cross-links (11.4.214, both ways): HRD-023, HRD-038 (GAP-002). Source: docs/gap-register-2026-11-23.md#gap-034---interactive-demo-has-no-platform-matrix-automation-or-helixqa-bank. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: web.

**Affected scope:** Platform gap GAP-034: Interactive demo has no platform-matrix automation or HelixQA bank

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-034---interactive-demo-has-no-platform-matrix-automation-or-helixqa-bank; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: demos/interactive_static_demo present; HRD-023 Queued; no evidence dir for it

**Acceptance:** Bank run 100% PASS with screenshots. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Bank run 100% PASS with screenshots. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-034 docs/gap-register-2026-11-23.md#gap-034---interactive-demo-has-no-platform-matrix-automation-or-helixqa-bank

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:43Z / 2026-11-23T21:22:43Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-070] Fuzz coverage: 11 Fuzz funcs but none for OPDS XML/JSON generation, LCP license JSON, KOSync body parsers (UNCONFIRMED per file)

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** Fuzz coverage: 11 Fuzz funcs but none for OPDS XML/JSON generation, LCP license JSON, KOSync body parsers (UNCONFIRMED per file). Filed 2026-11-23 from the read-only gap audit (GAP-035, Test, severity Medium); no work started. EVIDENCE: grep func Fuzz count 11 (commands_raw.txt) ROOT CAUSE (UNCONFIRMED unless stated): Fuzz added for format/range parsers only (HRD-013). PROPOSED FIX: Add fuzz targets for LCP/LSD parsers and OPDS inputs; run with -fuzztime in standing suite. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Security (11.4.169(8)); Unit (11.4.169(1)). Cross-links (11.4.214, both ways): HRD-013. Source: docs/gap-register-2026-11-23.md#gap-035---fuzz-coverage-11-fuzz-funcs-but-none-for-opds-xmljson-generation-lcp-license-json-kosync-body-parsers-unconfirmed-per-file. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: core.

**Affected scope:** Test gap GAP-035: Fuzz coverage: 11 Fuzz funcs but none for OPDS XML/JSON generation, LCP license JSON, KOSync body parsers (UNCONFIRMED per file)

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-035---fuzz-coverage-11-fuzz-funcs-but-none-for-opds-xmljson-generation-lcp-license-json-kosync-body-parsers-unconfirmed-per-file; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: grep func Fuzz count 11 (commands_raw.txt)

**Acceptance:** Fuzz corpus archived; no crashers in 5 min. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Fuzz corpus archived; no crashers in 5 min. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-035 docs/gap-register-2026-11-23.md#gap-035---fuzz-coverage-11-fuzz-funcs-but-none-for-opds-xmljson-generation-lcp-license-json-kosync-body-parsers-unconfirmed-per-file

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:43Z / 2026-11-23T21:22:43Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-071] No local CI-less gate runner executes the constitution full suite for this repo; only individually-run gates

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** No local CI-less gate runner executes the constitution full suite for this repo; only individually-run gates. Filed 2026-11-23 from the read-only gap audit (GAP-036, Process, severity Medium); no work started. EVIDENCE: submodules/constitution/scripts/run_full_suite.sh exists; no evidence of a run in this repo; two gates PASS vacuously ROOT CAUSE (UNCONFIRMED unless stated): §11.4.156 requires enforcement locally. PROPOSED FIX: Run run_full_suite.sh (read-only config) and store verdicts; wire as pre-commit local hook. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Full-automation (11.4.169(4)). Cross-links (11.4.214, both ways): HRD-012, HRD-051 (GAP-016). Source: docs/gap-register-2026-11-23.md#gap-036---no-local-ci-less-gate-runner-executes-the-constitution-full-suite-for-this-repo-only-individually-run-gates. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: qa.

**Affected scope:** Process gap GAP-036: No local CI-less gate runner executes the constitution full suite for this repo; only individually-run gates

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-036---no-local-ci-less-gate-runner-executes-the-constitution-full-suite-for-this-repo-only-individually-run-gates; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: submodules/constitution/scripts/run_full_suite.sh exists; no evidence of a run in this repo; two gates PASS vacuously

**Acceptance:** Full suite log archived, 0 FAIL. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Full suite log archived, 0 FAIL. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-036 docs/gap-register-2026-11-23.md#gap-036---no-local-ci-less-gate-runner-executes-the-constitution-full-suite-for-this-repo-only-individually-run-gates

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:43Z / 2026-11-23T21:22:43Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-072] Android build is debug-only; no release/signing/ProGuard or tablet/Wear check

**Status:** Queued

**Type:** Task

**Severity:** Low

**Reopens:** 0

**Description:** Android build is debug-only; no release/signing/ProGuard or tablet/Wear check. Filed 2026-11-23 from the read-only gap audit (GAP-037, Platform, severity Low); no work started. EVIDENCE: supported-platforms.json android notes ROOT CAUSE (UNCONFIRMED unless stated): Not requested yet. PROPOSED FIX: Release build in container, apksigner verify, install smoke. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): End-to-end (11.4.169(3)); Security (11.4.169(8)). Cross-links (11.4.214, both ways): HRD-022, HRD-025. Source: docs/gap-register-2026-11-23.md#gap-037---android-build-is-debug-only-no-releasesigningproguard-or-tabletwear-check. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: android.

**Affected scope:** Platform gap GAP-037: Android build is debug-only; no release/signing/ProGuard or tablet/Wear check

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-037---android-build-is-debug-only-no-releasesigningproguard-or-tabletwear-check; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: supported-platforms.json android notes

**Acceptance:** apksigner verify OK; release APK smoke PASS. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** apksigner verify OK; release APK smoke PASS. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-037 docs/gap-register-2026-11-23.md#gap-037---android-build-is-debug-only-no-releasesigningproguard-or-tabletwear-check

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:43Z / 2026-11-23T21:22:43Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-073] Challenges for mixed Postgres vs in-memory catalog: web/android evidence uses in-memory catalog only

**Status:** Queued

**Type:** Task

**Severity:** Low

**Reopens:** 0

**Description:** Challenges for mixed Postgres vs in-memory catalog: web/android evidence uses in-memory catalog only. Filed 2026-11-23 from the read-only gap audit (GAP-038, Test, severity Low); no work started. EVIDENCE: supported-platforms.json web and android notes ROOT CAUSE (UNCONFIRMED unless stated): Convenience fixture. PROPOSED FIX: Rerun web+android flows against Postgres-backed server (containers). REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Integration (11.4.169(2)); End-to-end (11.4.169(3)). Cross-links (11.4.214, both ways): HRD-022, HRD-040 (GAP-004). Source: docs/gap-register-2026-11-23.md#gap-038---challenges-for-mixed-postgres-vs-in-memory-catalog-webandroid-evidence-uses-in-memory-catalog-only. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: qa.

**Affected scope:** Test gap GAP-038: Challenges for mixed Postgres vs in-memory catalog: web/android evidence uses in-memory catalog only

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-038---challenges-for-mixed-postgres-vs-in-memory-catalog-webandroid-evidence-uses-in-memory-catalog-only; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: supported-platforms.json web and android notes

**Acceptance:** Same checks PASS with PG catalog. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Same checks PASS with PG catalog. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-038 docs/gap-register-2026-11-23.md#gap-038---challenges-for-mixed-postgres-vs-in-memory-catalog-webandroid-evidence-uses-in-memory-catalog-only

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:43Z / 2026-11-23T21:22:43Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-074] Flaky/quarantine registry absent (§11.4.248); needle tests self-skip when needle does not fire

**Status:** Queued

**Type:** Task

**Severity:** Low

**Reopens:** 0

**Description:** Flaky/quarantine registry absent (§11.4.248); needle tests self-skip when needle does not fire. Filed 2026-11-23 from the read-only gap audit (GAP-039, Test, severity Low); no work started. EVIDENCE: lock_scope_test.go:389 Skipf 'needle did not fire'; no [PROTECTED-SPEC] tags found ROOT CAUSE (UNCONFIRMED unless stated): Probabilistic needles skip rather than fail. PROPOSED FIX: Make needle deterministic or fail-closed after N attempts; add quarantine list. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Concurrency / atomicity (11.4.169(10)); Race / deadlock (11.4.169(11)). Cross-links (11.4.214, both ways): HRD-027, HRD-014. Source: docs/gap-register-2026-11-23.md#gap-039---flakyquarantine-registry-absent-114248-needle-tests-self-skip-when-needle-does-not-fire. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: server.

**Affected scope:** Test gap GAP-039: Flaky/quarantine registry absent (§11.4.248); needle tests self-skip when needle does not fire

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-039---flakyquarantine-registry-absent-114248-needle-tests-self-skip-when-needle-does-not-fire; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: lock_scope_test.go:389 Skipf 'needle did not fire'; no [PROTECTED-SPEC] tags found

**Acceptance:** Needle test either fires deterministically or fails. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Needle test either fires deterministically or fails. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-039 docs/gap-register-2026-11-23.md#gap-039---flakyquarantine-registry-absent-114248-needle-tests-self-skip-when-needle-does-not-fire

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:44Z / 2026-11-23T21:22:44Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-075] Workable-item statuses lag reality: HRD-031/025/026/029 still Queued/In progress while evidence entries claim delivery

**Status:** Queued

**Type:** Task

**Severity:** Low

**Reopens:** 0

**Description:** Workable-item statuses lag reality: HRD-031/025/026/029 still Queued/In progress while evidence entries claim delivery. Filed 2026-11-23 from the read-only gap audit (GAP-040, Docs, severity Low); no work started. EVIDENCE: DB listing in this audit vs EVIDENCE.md 2026-11-23 sections ROOT CAUSE (UNCONFIRMED unless stated): Registry not updated pending operator/independent review. PROPOSED FIX: Update via lifecycle skill after review (not edit by hand). REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Full-automation (11.4.169(4)). Cross-links (11.4.214, both ways): HRD-031, HRD-025, HRD-026, HRD-029, HRD-028. Source: docs/gap-register-2026-11-23.md#gap-040---workable-item-statuses-lag-reality-hrd-031025026029-still-queuedin-progress-while-evidence-entries-claim-delivery. Audit evidence: evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Owner suggestion: operator.

**Affected scope:** Docs gap GAP-040: Workable-item statuses lag reality: HRD-031/025/026/029 still Queued/In progress while evidence entries claim delivery

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-040---workable-item-statuses-lag-reality-hrd-031025026029-still-queuedin-progress-while-evidence-entries-claim-delivery; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: DB listing in this audit vs EVIDENCE.md 2026-11-23 sections

**Acceptance:** Registry statuses match evidence; docs regenerated. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Registry statuses match evidence; docs regenerated. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-040 docs/gap-register-2026-11-23.md#gap-040---workable-item-statuses-lag-reality-hrd-031025026029-still-queuedin-progress-while-evidence-entries-claim-delivery

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:44Z / 2026-11-23T21:22:44Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-076] Real screenshots and video of everyday use of ALL clients captured only in the isolated container runner and embedded in every doc category

**Status:** Queued

**Type:** Feature

**Severity:** High

**Reopens:** 0

**Description:** BACKGROUND IMPORTANT operator directive 2026-11-23: obtain real, stunning screenshots and video of everyday use of ALL clients (web at several screen sizes, Electron desktop, KOReader sync/OPDS/PSE flows, Android emulator) showing power-feature use cases, captured ONLY in the isolated container runner (tools/isolation, qa/isolation; HARD RULE: never touch the operator desktop or apps), each with a capture manifest (client, version, scenario, resolution, sha256, command), and embed them in ALL docs: user guides, manuals, tutorials, quick starts, FAQs and the README. Create every missing doc category exhaustively (11.4.257 user-manual/guide/FAQ coverage, 11.4.258 architecture/data-flow/state-machine/sequence diagrams, 11.4.212 README as canonical entry point). ROOT CAUSE (UNCONFIRMED unless stated): docs carry little or no real captured media of the clients in everyday use; capture runs so far were test-evidence oriented, not documentation oriented. PROPOSED FIX: scenario catalogue per client and power feature, container-only capture runner producing PNG/MP4 plus manifest, vision verification of each asset (HRD-039), doc generator embedding assets by manifest id, and a gate failing on embedded media without a manifest row. Cross-links (11.4.214, text): HRD-038, HRD-039, HRD-040, HRD-043, HRD-044, HRD-061, HRD-062, HRD-069 (capture runners and vision), HRD-059 (documentation drift). Owner suggestion: docs+qa.

**Affected scope:** web UI (several viewports), Electron desktop, KOReader sync/OPDS/PSE flows, Android emulator, tools/isolation, qa/isolation, README.md, docs/user_guide.md, docs/faq.md, quick starts, tutorials, manuals, diagrams.

**Reproduction / current observation:** Observable current behavior: docs contain no real captured screenshots or video of the clients in everyday use and several doc categories are missing.

**Acceptance:** Every client has screenshots and video of its power-feature scenarios produced inside the isolated container runner with a manifest row (client, version, scenario, resolution, sha256, command) per asset; every user guide, manual, tutorial, quick start, FAQ and the README embeds them; all 11.4.257 doc categories and 11.4.258 diagrams exist; a gate rejects embedded media lacking a manifest row and a deliberately wrong asset (mutant) is rejected; guard evidence shows no host-desktop capture. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): End-to-end (11.4.169(3)); UI (11.4.27 verbatim mandate); UX (11.4.27 verbatim mandate); Full-automation (11.4.169(4)); HelixQA (11.4.169(6)); Vision read (cross-cutting 11.4.278/11.4.117/11.4.160, not a 11.4.169 member). Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Every client has screenshots and video of its power-feature scenarios produced inside the isolated container runner with a manifest row (client, version, scenario, resolution, sha256, command) per asset; every user guide, manual, tutorial, quick start, FAQ and the README embeds them; all 11.4.257 doc categories and 11.4.258 diagrams exist; a gate rejects embedded media lacking a manifest row and a deliberately wrong asset (mutant) is rejected; guard evidence shows no host-desktop capture. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): End-to-end (11.4.169(3)); UI (11.4.27 verbatim mandate); UX (11.4.27 verbatim mandate); Full-automation (11.4.169(4)); HelixQA (11.4.169(6)); Vision read (cross-cutting 11.4.278/11.4.117/11.4.160, not a 11.4.169 member). Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T22:49:59Z / 2026-11-23T22:49:59Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-077] WebDAV.List listed directories as zero-byte files

**Status:** Queued

**Type:** Bug

**Severity:** Medium

**Reopens:** 0

**Description:** WebDAV.List in server/internal/sources/webdav.go reported collection (directory) entries as zero-byte regular files instead of directories. Found while fixing HRD-036. STATUS NOTE: fix present uncommitted in the working tree (merged() in server/internal/sources/webdav.go), independent review owed. EVIDENCE: evidence/campaign-2026-11-23/gap-fix/hrd036/RED_list_collection_as_file.log (RED) and evidence/campaign-2026-11-23/gap-fix/hrd036/GREEN_list_collection_as_file.log (GREEN). ROOT CAUSE (UNCONFIRMED beyond the RED/GREEN pair): collection entries were not distinguished from files when merging PROPFIND results. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Unit (11.4.169(1)); Integration (11.4.169(2)); Mutation (11.4.169 member). Cross-links (11.4.214, text): HRD-036 (found while fixing), HRD-031. Owner suggestion: server.

**Affected scope:** server/internal/sources/webdav.go (WebDAV.List, merged()), remote-source browsing and ingest.

**Reproduction / current observation:** Listing a WebDAV collection containing sub-collections returned them as zero-byte files (see the RED log under evidence/campaign-2026-11-23/gap-fix/hrd036/).

**Acceptance:** WebDAV.List reports collections as directories; the RED test fails on the old code and passes on the fix; a mutant reverting merged() is killed; independent review GO. Fix is uncommitted and review is owed. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** WebDAV.List reports collections as directories; the RED test fails on the old code and passes on the fix; a mutant reverting merged() is killed; independent review GO. Fix is uncommitted and review is owed. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T22:49:59Z / 2026-11-23T22:49:59Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-078] Advertised-but-unwired libraries: pkg/fts, hlc, LRU cache, settings/theme/crypto are claimed tested but wired into neither server nor clients

**Status:** Queued

**Type:** Task

**Severity:** High

**Reopens:** 0

**Description:** README and claim-vs-reality ledger rows LED-037, LED-038, LED-039 and LED-044 (docs/claim_reality_ledger.tsv, state config-present-but-unwired) advertise core/pkg/fts full-text search, core/hlc hybrid logical clocks, the LRU cache with protected entries, and settings/theme/crypto utilities as tested capabilities, but code grep 2026-11-23 shows no non-test importer in the server or clients (library plus unit tests only). Note LED-040..043 are a different class (green-but-broken) and are not covered here. DECISION NEEDED: wire each library into the server/clients or de-advertise it in README and docs (11.4.266 claim-vs-reality). Scope of the decision is the operator's; status is Queued with this note (the tool's operator-block rules are not triggered because investigation and de-advertise work can proceed). ROOT CAUSE (UNCONFIRMED unless stated): libraries were built ahead of integration and the README described them as shipped features. Cross-links (11.4.214, text): HRD-051 (ledger), HRD-059 (documentation drift), HRD-075 (status lag). Owner suggestion: operator (scope) then core+server.

**Affected scope:** core/pkg/fts, core/hlc, core cache, core/pkg/settings, core/pkg/theme, core/pkg/crypto, README.md lines 64-72, docs/claim_reality_ledger.tsv LED-037/038/039/044, server and client importers.

**Reproduction / current observation:** grep for non-test imports of these packages in server/ and clients/ returns none while README advertises them.

**Acceptance:** For each of the four capability groups either (a) a real wired code path with an end-to-end test proving use by the server or a client, or (b) README, docs and ledger rows de-advertised, with the ledger state updated and the operator decision recorded. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Unit (11.4.169(1)); Integration (11.4.169(2)); End-to-end (11.4.169(3)); Documentation claim gate (11.4.266). Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** For each of the four capability groups either (a) a real wired code path with an end-to-end test proving use by the server or a client, or (b) README, docs and ledger rows de-advertised, with the ledger state updated and the operator decision recorded. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Unit (11.4.169(1)); Integration (11.4.169(2)); End-to-end (11.4.169(3)); Documentation claim gate (11.4.266). Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T22:49:59Z / 2026-11-23T22:49:59Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-079] llmctl install ignores its arguments so llmctl install --help performs the real install

**Status:** Queued

**Type:** Bug

**Severity:** Medium

**Reopens:** 0

**Description:** EXTERNAL REPO (~/Projects/llmctl), tracked here for visibility only. `llmctl install` ignores its arguments, so `llmctl install --help` performs the real install: it regenerated unit files and enabled the revalidate timer. Found during HRD-039. A patch plus test exist at evidence/campaign-2026-11-23/gap-fix/hrd039/llmctl_install_args.patch, uncommitted in the llmctl repo. ROOT CAUSE (CONFIRMED by the patch and its test): the install subcommand never parsed argv. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Unit (11.4.169(1)); Integration (11.4.169(2)); Regression-guard (11.4.135). Cross-links (11.4.214, text): HRD-039. Owner suggestion: llmctl maintainer.

**Affected scope:** llmctl repository (outside this repo): install subcommand argument parsing.

**Reproduction / current observation:** Run `llmctl install --help` on an isolated target and observe units regenerated and the timer enabled.

**Acceptance:** `llmctl install --help` prints usage and changes nothing (RED on old code, GREEN on patched code), unknown flags are rejected, and the fix lands in the llmctl repo with independent review GO. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** `llmctl install --help` prints usage and changes nothing (RED on old code, GREEN on patched code), unknown flags are rejected, and the fix lands in the llmctl repo with independent review GO. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T22:49:59Z / 2026-11-23T22:49:59Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-080] HRD-039b follow-up: use ExecStartPre=+ for the llmctl engine gate to keep ProtectSystem=full

**Status:** Queued

**Type:** Task

**Severity:** Low

**Reopens:** 0

**Description:** Follow-up to HRD-039: run the llmctl engine gate as ExecStartPre=+ so ProtectSystem=full can stay enabled on the unit. Design note: ~/Projects/llmctl/docs/hrd-039b-execstartpre-plus-design-note.md. Probe evidence: evidence/campaign-2026-11-23/gap-fix/hrd039/probe_execstartpre_plus.txt. Only needed if the operator wants ProtectSystem restored; otherwise leave Queued. External repo (llmctl), tracked here for visibility. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Integration (11.4.169(2)); Security (11.4.169 member); Regression-guard (11.4.135). Cross-links (11.4.214, text): HRD-039. Owner suggestion: llmctl maintainer.

**Affected scope:** llmctl systemd unit generation and engine gate (outside this repo).

**Reproduction / current observation:** Current unit omits ProtectSystem=full because the gate needs privileged writes in ExecStartPre.

**Acceptance:** Unit keeps ProtectSystem=full and the engine gate still passes on an isolated target, with RED on a unit lacking the plus prefix. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Unit keeps ProtectSystem=full and the engine gate still passes on an isolated target, with RED on a unit lacking the plus prefix. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T22:50:00Z / 2026-11-23T22:50:00Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-081] Submodule host-side entry points are scanned report-only by the isolation guard: 1457 findings need triage, upstream fix or gate

**Status:** Queued

**Type:** Task

**Severity:** High

**Reopens:** 0

**Description:** The isolation guard scans submodule host-side entry points report-only and found 1457 findings, mostly submodules/helixqa host-side capture, uinput and portal executors, plus the constitution zap script using --network host. Report: evidence/campaign-2026-11-23/isolated/hardening/submodule_host_entrypoints_report.txt. HARD RULE: never touch the operator desktop or apps; report-only scanning means a submodule host-side path could still reach the desktop. ROOT CAUSE (UNCONFIRMED unless stated): submodules ship host-side executors that are safe only if never invoked outside the container runner; nothing enforces that. PROPOSED FIX: triage findings into reachable/unreachable from our entry points, upstream-fix or wrap the reachable ones, and turn the scan into a failing gate with an allow-list. Cross-links (11.4.214, text): HRD-038, HRD-043, HRD-047. Owner suggestion: qa+containers.

**Affected scope:** submodules/helixqa, submodules/constitution scripts, tools/isolation, qa/isolation guard.

**Reproduction / current observation:** Run the isolation guard scan and read the report: 1457 report-only findings.

**Acceptance:** Every finding is classified, reachable host-side executors are fixed upstream or blocked by the guard, the scan fails the gate on any un-allow-listed finding, and a RED fixture proves a host-side executor is rejected. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Security (11.4.169 member); Full-automation (11.4.169(4)); Integration (11.4.169(2)); Mutation (11.4.169 member). Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Every finding is classified, reachable host-side executors are fixed upstream or blocked by the guard, the scan fails the gate on any un-allow-listed finding, and a RED fixture proves a host-side executor is rejected. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Security (11.4.169 member); Full-automation (11.4.169(4)); Integration (11.4.169(2)); Mutation (11.4.169 member). Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T22:50:00Z / 2026-11-23T22:50:00Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-082] helixqa upstream defects worked around in tests/helixqa_web: PlaywrightCLIAdapter never closes the CDP connection, splitSelectorValue splits fill at first space, helixqa http skips playwright steps

**Status:** Queued

**Type:** Bug

**Severity:** Medium

**Reopens:** 0

**Description:** Three upstream helixqa defects worked around in tests/helixqa_web: (1) PlaywrightCLIAdapter never closes the CDP connection so steps hang, (2) splitSelectorValue splits a fill step at the first space, mangling values containing spaces, (3) `helixqa http` never executes playwright: steps and silently SKIPs them with exit code 0. Related evidence: evidence/campaign-2026-11-23/gap-fix/hrd038/red1_http_runner_skips.txt, evidence/campaign-2026-11-23/gap-fix/hrd038/red2_noskip_http_runner_still_skips.txt, evidence/campaign-2026-11-23/gap-fix/hrd038/red3_upstream_adapter_hangs.txt. ROOT CAUSE (UNCONFIRMED beyond the RED logs): upstream adapter and runner code. Fix belongs in the helixqa submodule upstream. Cross-links (11.4.214, text): HRD-038 (GAP-002), HRD-054. Owner suggestion: qa+helixqa upstream.

**Affected scope:** submodules/helixqa PlaywrightCLIAdapter, splitSelectorValue, `helixqa http` runner, tests/helixqa_web workarounds.

**Reproduction / current observation:** See the RED logs listed in the description.

**Acceptance:** Upstream fixes merged for all three defects, workarounds removed from tests/helixqa_web, `helixqa http` fails (non-zero exit) rather than SKIPs on playwright: steps, with RED then GREEN logs. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Unit (11.4.169(1)); Integration (11.4.169(2)); HelixQA (11.4.169(6)); Mutation (11.4.169 member). Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Upstream fixes merged for all three defects, workarounds removed from tests/helixqa_web, `helixqa http` fails (non-zero exit) rather than SKIPs on playwright: steps, with RED then GREEN logs. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Unit (11.4.169(1)); Integration (11.4.169(2)); HelixQA (11.4.169(6)); Mutation (11.4.169 member). Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T22:50:00Z / 2026-11-23T22:50:00Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-083] Adopt the PR-driven workflow: branch, PR, independent review agents, merge to main, never force-push, multi-upstream mirror

**Status:** Queued

**Type:** Task

**Severity:** High

**Reopens:** 0

**Description:** Operator directive 2026-11-23: adopt PR-driven work: each change goes branch -> PR -> independent review agents (11.4.142) -> merge to main (11.4.288), never force-push (11.4.113), with fast-forward mirror to every upstream (2.1). Composes with the constitution update another agent is preparing (see the constitution-update item and docs/constitution-update-plan-2026-11-23.md). ROOT CAUSE (UNCONFIRMED unless stated): work so far lands directly on main from a dirty working tree, which has accumulated many uncommitted areas. PROPOSED FIX: document the workflow, add scripts for branch/PR/review/merge/mirror stages (no CI/CD per 11.4.156, enforcement local), pilot on one small change. Owner suggestion: operator+governance.

**Affected scope:** git workflow, scripts, constitution hooks, all own-org submodules, docs.

**Reproduction / current observation:** Changes currently accumulate uncommitted on main with no PR or review record.

**Acceptance:** A documented and scripted PR-driven flow exists, a pilot change passes branch, PR, independent review record, merge to main and mirror to every upstream without force, and a gate rejects a force-push attempt. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Integration (11.4.169(2)); Full-automation (11.4.169(4)); Security (11.4.169 member). Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** A documented and scripted PR-driven flow exists, a pilot change passes branch, PR, independent review record, merge to main and mirror to every upstream without force, and a gate rejects a force-push attempt. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Integration (11.4.169(2)); Full-automation (11.4.169(4)); Security (11.4.169 member). Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T22:50:00Z / 2026-11-23T22:50:00Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-084] Update the constitution submodule to the latest pin with recursive incorporation and post-update hooks

**Status:** Queued

**Type:** Task

**Severity:** High

**Reopens:** 0

**Description:** Update the constitution submodule to the latest pin, incorporate recursively (every own-org submodule and hub engine), and run the post-update hooks per docs/constitution-update-plan-2026-11-23.md (current pin 6d93ccd19e1c2fe0e1a97ece22b49e962ef3005e). Registry grep found no existing item covering this (HRD-052 covers helix-deps drift, HRD-053 covers other submodule pins). Plan status: in-repo skill links applied, everything outside the repo awaiting operator. Governing anchors 11.4.164, 11.4.32, 11.4.228(D), 11.4.234, 11.4.272, 11.4.281(C). Cross-links (11.4.214, text): HRD-052, HRD-053, and the PR-driven workflow item (composes). Owner suggestion: operator+governance.

**Affected scope:** submodules/constitution pin, nested submodules, post_update_hook.sh steps, .claude/skills links, helix-deps.yaml.

**Reproduction / current observation:** Constitution pin lags canon; post-update hook steps outside the repo not yet applied.

**Acceptance:** Constitution at the latest pin on main in every upstream, post-pull validation (11.4.32) passes with captured output, the post-update hook steps are applied or explicitly declined by the operator and recorded, and all gates run green or are recorded as findings. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Integration (11.4.169(2)); Full-automation (11.4.169(4)); Regression-guard (11.4.135). Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Constitution at the latest pin on main in every upstream, post-pull validation (11.4.32) passes with captured output, the post-update hook steps are applied or explicitly declined by the operator and recorded, and all gates run green or are recorded as findings. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Integration (11.4.169(2)); Full-automation (11.4.169(4)); Regression-guard (11.4.135). Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T22:50:00Z / 2026-11-23T22:50:00Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-085] Registry hardening and gap-register import landing (PR #8)

**Status:** In progress

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** Land the hardened workable-items registry tooling (scripts/workable_items.py, verify_workable_items.py, 85+ tests, docs/workable-items.md) together with the registry content the dirty shared worktree accumulated on 2026-11-23: 49 new items (HRD-036 onward) filed from the gap register, status or description updates on 16 pre-existing items, and the regenerated Issues/Fixed documents and outbox. Makes the registry self-consistent on a clean clone for documents and sealed evidence: adds the gap-register documents and cited evidence files, a reference checker (scripts/check_registry_references.py) with an ownership map for paths delivered by other work, and an outbox-versus-database consistency check in diff. Cited executable code is deliberately not landed here; the checker lists it as owned by its own item.

**Affected scope:** scripts/workable_items.py, scripts/verify_workable_items.py, scripts/check_registry_references.py, tests/, docs/workable_items.db, docs/tracking/*, the generated docs/Issues*, docs/Fixed*, README.md and deploy/README.md (HRD-034 scope note), and the documents and sealed evidence files the registry cites (gap register, upstream-issue drafts, campaign evidence). Executable product code cited by the registry (tools/firebase, tests/helixqa_web, clients, run_android_matrix.sh) is NOT part of this item.

**Reproduction / current observation:** On github/main before this PR the registry lacks the gap-register items; after merge without the cited files, 120 stored evidence paths or docs links dangle (evidence/workable-items/registry-hardening-pr8/RED_dangling_references_before_fix.txt).

**Acceptance:** validate and diff PASS (diff also compares the HelixTrack outbox with the registry and fails when it is missing); verify_workable_items.py reports zero failures with reportlab; unit tests pass with and without reportlab; check_registry_references.py classifies every missing reference as OWNED-BY-PENDING-PR, OWNED-BY-UNDELIVERED-WORK or NOWHERE and its output is recorded; zero NOWHERE is NOT required for this item because the NOWHERE citations come from earlier work and are tracked separately; independent review GO.

**Closure criteria:** validate and diff PASS (diff also compares the HelixTrack outbox with the registry and fails when it is missing); verify_workable_items.py reports zero failures with reportlab; unit tests pass with and without reportlab; check_registry_references.py classifies every missing reference as OWNED-BY-PENDING-PR, OWNED-BY-UNDELIVERED-WORK or NOWHERE and its output is recorded; zero NOWHERE is NOT required for this item because the NOWHERE citations come from earlier work and are tracked separately; independent review GO.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** docs/pr-delivery.md

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T10:35:56Z / 2026-11-24T11:32:28Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded
- 2026-11-24 — Started by AI: Review round 1 of PR #8 returned NO-GO; fixing dangling references, id choice, HRD-034 scope note and minors; evidence: evidence/workable-items/registry-hardening-pr8
- 2026-11-24 — Updated by AI: Narrow HRD-085 scope after PR #10 review: executable product code removed, ownership map added; evidence: docs/tracking/pending_ownership.json
- 2026-11-24 — Updated by AI: Acceptance corrected to what is achievable: zero NOWHERE references cannot be reached by registry-tooling work alone; the earlier text named stale counts and a pass criterion this item cannot meet.; evidence: evidence/workable-items/registry-hardening-pr8/final_check_references.txt
