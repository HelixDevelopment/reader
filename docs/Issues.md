# Helix Open Workable Items

Generated from `docs/workable_items.db`; do not edit directly.

In progress: 18, In testing: 0, Operator-blocked: 0, Queued: 81, Ready for testing: 0, Reopened: 0

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

**Description:** LCP never exercised against a real provider or third-party reader. Filed 2026-11-23 from the read-only gap audit (GAP-011, Security, severity High). PARTIAL PROGRESS 2026-11-24 (PR #33): core/pkg/lcp was exercised against the Readium reference License Server, Status Server and lcpencrypt (readium-lcp-server@32570b69, third-party provider, real processes in a rootless network-less container): 17 PASS assertions + 2 INFO records in 3 identical sealed runs (evidence/lcp-interop-run1..3), a wrong-oracle mutant and two reviewer mutants each FAIL as expected (sealed under evidence/lcp-interop-*mutant*). Library consumer only. STILL OPEN: no third-party READER opened a Helix artifact; the real EDRLab/Readium root is not verified (HRD-122); renew not exercised (provider config renew false); the server/OPDS path is not covered; HRD-123 found a same-second NeedsLicenseRefresh defect. ROOT CAUSE (UNCONFIRMED unless stated): Self-referential test oracle (§11.4.245): producer and consumer share an author. PROPOSED FIX: open produced artifact with Thorium/readium-lcp-client-in-container. REQUIRED TEST TYPES (11.4.169 / 11.4.27 members): Integration (11.4.169(2)); Security (11.4.169(8)); Interop against a third-party peer (a library consumer here; not full end-to-end). Cross-links (11.4.214, both ways): HRD-031, HRD-026, HRD-122, HRD-123. Source: docs/gap-register-2026-11-23.md#gap-011---lcp-never-exercised-

**Affected scope:** Security gap GAP-011: LCP never exercised against a real provider or third-party reader

**Reproduction / current observation:** See docs/gap-register-2026-11-23.md#gap-011---lcp-never-exercised-against-a-real-provider-or-third-party-reader; audit evidence evidence/campaign-2026-11-23/gap-audit/commands.txt, evidence/campaign-2026-11-23/gap-audit/commands_part2.txt, evidence/campaign-2026-11-23/gap-audit/commands_raw.txt, evidence/campaign-2026-11-23/gap-audit/proven_checks.txt, evidence/campaign-2026-11-23/gap-audit/gates_run.txt. Register evidence: EVIDENCE.md lines 817,839,875 NOT PROVEN/UNCONFIRMED; lsd provider written in-repo from same spec text

**Acceptance:** Independent reader decrypts artifact; provider LSD register/return/renew round-trip logged; RED with wrong passphrase. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Closure criteria:** Independent reader decrypts artifact; provider LSD register/return/renew round-trip logged; RED with wrong passphrase. Evidence must be deterministic machine output (result JSON plus sha256 manifest under evidence/), with a RED baseline on the broken artifact where a defect or gate is involved (11.4.115). COMPLETION CONDITION: independent review GO (11.4.142) AND evidence class recorded at closure (11.4.226).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** GAP-011 docs/gap-register-2026-11-23.md#gap-011---lcp-never-exercised-against-a-real-provider-or-third-party-reader

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-23T21:22:39Z / 2026-11-24T21:11:29Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded
- 2026-11-24 — Updated by AI: PR review fix round: registry state follows the work; evidence: evidence/lcp-interop-run1

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

**Status:** In progress

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

**Created / modified:** 2026-11-23T22:49:59Z / 2026-11-24T20:14:42Z

**History:**

- 2026-11-23 — Opened by AI: new-work-item; evidence: none recorded
- 2026-11-24 — Started by AI: HRD-076: web, desktop and KOReader captured in the isolated runner; Android not re-captured (UNCONFIRMED); PR open for independent review; evidence: evidence/screenshots

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

## [HRD-086] Allocate registry identifiers HRD-086 to HRD-103 for work streams landed or reviewed on 2026-11-24

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** Registry change that allocates tracked identifiers for the Cronet and HTTP/3 clients, the Go live verifiers, the Go mutation and UI verifiers, the packaging ignore rules, the Android HTTP/3 lab follow-up and the review follow-ups, so every branch and pull request can carry a valid Roadmap-Item trailer. This item is the change's own item and closes when the registry pull request is merged and read back.

**Affected scope:** docs/workable_items.db and the generated docs/Issues*, docs/Fixed*, docs/tracking/helixtrack-outbox.json only.

**Reproduction / current observation:** Work streams exist (Cronet, Go verifiers, packaging, review follow-ups) with no HRD ids, so their commits cannot carry a Roadmap-Item trailer that resolves in the registry.

**Acceptance:** HRD-087 to HRD-103 exist once each, validate and diff PASS, python3 -m unittest -v tests.test_workable_items passes, and the PR is merged with a merge commit and read back. No duplicate of an existing item (HRD-084 constitution pin update is not duplicated).

**Closure criteria:** HRD-087 to HRD-103 exist once each, validate and diff PASS, python3 -m unittest -v tests.test_workable_items passes, and the PR is merged with a merge commit and read back. No duplicate of an existing item (HRD-084 constitution pin update is not duplicated).

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T16:39:05Z / 2026-11-24T16:39:05Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-087] Cronet and HTTP/3 clients: desktop Chromium-session h3, Android Cronet forced QUIC, server access log middleware and qa/h3 verifier

**Status:** Queued

**Type:** Feature

**Severity:** High

**Reopens:** 0

**Description:** Stream A. Adds HTTP/3 capable client transports: the desktop client talks h3 through a Chromium session, the Android client uses Cronet with QUIC forced, the server gains a HELIX_ACCESS_LOG middleware so negotiated protocol is observable server side, and qa/h3 holds the verification harness. Stacked on pull request #18. Evidence: /home/milosvasic/.cache/claude-tmp/wt-cronet-state/final with a matrix of 28 files and an android bundle of 13 files. Evidence is held outside the repository under /home/milosvasic/.cache/claude-tmp and is not yet sealed into evidence/ (UNCONFIRMED: a later change must copy and seal it with tools/isolation/seal_evidence.sh).

**Affected scope:** clients/desktop, clients/android, server/internal/httpapi (access log middleware), qa/h3, docs on transports.

**Reproduction / current observation:** Before: no client negotiates HTTP/3 and the server cannot show which protocol served a request. Reproduce by running the qa/h3 matrix and reading the access log protocol field.

**Acceptance:** Desktop h3 session and Android Cronet request are shown by the server access log as h3 (matrix 28 files, android 13 files re-run on a clean checkout), PR #18 stacking resolved, and every gap below is closed or tracked. UNCONFIRMED: Android h3 end to end (private CA QUIC certificate verification fails with error -356), tracked as its own item.

**Closure criteria:** Desktop h3 session and Android Cronet request are shown by the server access log as h3 (matrix 28 files, android 13 files re-run on a clean checkout), PR #18 stacking resolved, and every gap below is closed or tracked. UNCONFIRMED: Android h3 end to end (private CA QUIC certificate verification fails with error -356), tracked as its own item.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T16:39:05Z / 2026-11-24T16:39:05Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-088] Android HTTP/3 lab: prove QUIC certificate verification (error -356) with a writable-system or known-root lab

**Status:** Queued

**Type:** Task

**Severity:** High

**Reopens:** 0

**Description:** Follow-up of the Cronet stream. Android Cronet with forced QUIC fails certificate verification against the private CA (net error -356), so Android h3 is UNCONFIRMED. A lab is needed where the private CA is trusted at system level (writable system image or a known publicly rooted certificate) and h3 is shown by the server access log from the real Android client in a rootless container (test isolation rule).

**Affected scope:** clients/android Cronet configuration, qa/h3 android leg, emulator image in the Containers submodule, trust store provisioning.

**Reproduction / current observation:** Run the Android h3 leg: the request fails with -356 and the access log shows no h3 request. Evidence bundle: /home/milosvasic/.cache/claude-tmp/wt-cronet-state/final (android 13 files), unsealed.

**Acceptance:** In the lab the Android client completes an h3 request, the server access log records protocol h3 for it, the evidence is sealed under evidence/ with sha256, and a negative control (untrusted CA) still fails closed.

**Closure criteria:** In the lab the Android client completes an h3 request, the server access log records protocol h3 for it, the evidence is sealed under evidence/ with sha256, and a negative control (untrusted CA) still fails closed.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T16:39:06Z / 2026-11-24T16:39:06Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-089] Go live verifiers: server, transport and backup-restore checks as Go commands in server/internal/liveverify

**Status:** Queued

**Type:** Feature

**Severity:** High

**Reopens:** 0

**Description:** Stream B. Port of the Python live verification to Go: package server/internal/liveverify plus the commands helix-verify-live-server, helix-verify-live-transport and helix-verify-backup-restore. Recorded results: live-server-go 77 passed and 0 failed, live-transport-go 15 passed and 0 failed, backup-restore-go 13 passed and 0 failed. Evidence is held outside the repository under /home/milosvasic/.cache/claude-tmp and is not yet sealed into evidence/ (UNCONFIRMED: a later change must copy and seal it with tools/isolation/seal_evidence.sh).

**Affected scope:** server/internal/liveverify, cmd/helix-verify-live-server, cmd/helix-verify-live-transport, cmd/helix-verify-backup-restore, evidence live-server-go, live-transport-go, backup-restore-go.

**Reproduction / current observation:** Run the three commands against a live server container and read the pass and fail counts; compare with the Python verifier results.

**Acceptance:** Commands build and pass on a clean checkout with real services (no stubs), mutation of the verifier kills at least one mutant per check family, evidence sealed in the repo, and gates reference the Go verifiers.

**Closure criteria:** Commands build and pass on a clean checkout with real services (no stubs), mutation of the verifier kills at least one mutant per check family, evidence sealed in the repo, and gates reference the Go verifiers.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T16:39:06Z / 2026-11-24T16:39:06Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-090] Go mutation and UI verifiers: tools/verify gomutate and uiverify ported from Python

**Status:** Queued

**Type:** Feature

**Severity:** High

**Reopens:** 0

**Description:** Stream C. Adds tools/verify with gomutate (Go mutation runner) and uiverify (UI verification) and evidence go-verify-port-2026-11-24. The Python-versus-Go live UI comparison is UNCONFIRMED: no run has shown both produce the same verdicts on the same UI. Evidence is held outside the repository under /home/milosvasic/.cache/claude-tmp and is not yet sealed into evidence/ (UNCONFIRMED: a later change must copy and seal it with tools/isolation/seal_evidence.sh).

**Affected scope:** tools/verify/gomutate, tools/verify/uiverify, evidence/go-verify-port-2026-11-24.

**Reproduction / current observation:** Run gomutate on core and server and uiverify on the web client in the isolation container; compare verdicts with the Python tools.

**Acceptance:** Equivalent verdicts to the Python tools on the same inputs (documented diff, zero unexplained differences), control and mutant pairs for every check, evidence sealed, and the UNCONFIRMED live comparison either completed or recorded as a limit.

**Closure criteria:** Equivalent verdicts to the Python tools on the same inputs (documented diff, zero unexplained differences), control and mutant pairs for every check, evidence sealed, and the UNCONFIRMED live comparison either completed or recorded as a limit.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T16:39:06Z / 2026-11-24T16:39:06Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-091] Packaging: ignore local-only paths and reference the Koodo-reader bundle by hash manifest

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** Add .gitignore entries for .claude symlinks, *.bak backups, background_queue.md and .env.sh, and replace the vendored Koodo-reader bundle by a hash-reference manifest. Existing branch task/HRD-PKG-gitignore at /home/milosvasic/.cache/claude-tmp/wt-gitignore (commit 7cdd283) must be renamed to the allocated id and its Roadmap-Item trailer fixed once this item exists.

**Affected scope:** .gitignore, the Koodo-reader bundle manifest, branch and trailer hygiene for task/HRD-PKG-gitignore.

**Reproduction / current observation:** git status of a working checkout lists .claude symlinks, .bak files, background_queue.md and .env.sh as untracked noise, and the bundle bytes are not hash-referenced.

**Acceptance:** Entries ignore exactly those paths (git check-ignore proves each), the manifest lists the bundle sha256 and the commit trailer names this item id; the branch is renamed task/HRD-nnn-\<slug> and merged by pull request.

**Closure criteria:** Entries ignore exactly those paths (git check-ignore proves each), the manifest lists the bundle sha256 and the commit trailer names this item id; the branch is renamed task/HRD-nnn-\<slug> and merged by pull request.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T16:39:08Z / 2026-11-24T16:39:08Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-092] PR #5 review follow-ups FU1 to FU5 and F5

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** Open follow-ups of the independent reviews of pull request 5 (isolation guard; MERGED), transcribed from pr5-review-7f6c7be.txt lines 71 to 76 and pr5-review-e15f0e8.txt lines 39 to 42 (all test-instrumentation or process-doc, none blocking). FU1 (test-instrumentation, important): no test kills mutant K13, so the waivers.history sha comparison (the hand-edited-hash tripwire) is unguarded; add the hand-edit case as a test with a paired mutant. FU2 (minor): malformed or padded history rows are silently ignored (should report waiver-malformed) and mutant K15 (prefix-path) survives; add a prefix-sharing path case. FU3 (minor): safe_streams call sites (K9, K18, K19, K20) and rev-parse decode (K7) have no killing test; assert each entry point stream errors equal backslashreplace. FU4 (process-doc): three commits carry Roadmap-Item outside the git trailer block (cannot be amended, 11.4.113). FU5 (process-doc): operator home path in frozen captured evidence files; consider host-path redaction in future captures. F5 (carried): tests/live/run_gated_tests.sh lines 152 and 156 echo container-influenced Go test names unescaped (FAILED-TEST and UNLISTED-SKIP); that is main file. Previous text of this item said UNCONFIRMED; now transcribed.

**Affected scope:** tools/isolation (assert_host_untouched.py and its guard tests), tests/live/run_gated_tests.sh, evidence of PR 5.

**Reproduction / current observation:** Open the PR #5 review comments and list FU1 to FU5 and F5 with their cited lines.

**Acceptance:** Each of FU1 to FU3 and F5 fixed with a test that fails without the fix (K13 and K15 killed) or closed with a recorded reason; FU4 and FU5 closed with a recorded reason; validate passes.

**Closure criteria:** Each of FU1 to FU3 and F5 fixed with a test that fails without the fix (K13 and K15 killed) or closed with a recorded reason; FU4 and FU5 closed with a recorded reason; validate passes.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T16:39:08Z / 2026-11-24T18:57:42Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded
- 2026-11-24 — Updated by AI: Transcribe PR 5 follow-ups from the review files (HRD-112); history kept; evidence: none recorded

## [HRD-093] PR #10 review follow-ups N1 to N7 and F-a, F-b, F-c (registry hardening)

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** Open follow-ups of the independent reviews of pull request 10 (registry hardening, MERGED), transcribed from pr10-review-3355491.txt lines 60 to 80, pr10-review-97d1a9a.txt lines 39 to 42 and pr10-review-226110e.txt lines 30 to 36. N1 (test-instrumentation, important): reviewer mutants C6 and C9 survive the 95-test suite though each changes real checker output (C6: 56 to 106 missing; C9: 10 items PENDING-PR to NOWHERE): the cited-dir depth floor and the underscore-prefix ownership rule have no killing test. N2 (important): mutants W2 and W3 survive: diff returning PASS on a corrupt or non-object outbox is not caught by any test. N3 (nit): stale ownership entry tools/helixtrack-sync (HRD-029) remains although the path is on main. N4 (nit): the words (upstream Core) added in docs/helixtrack-sync.md are not mentioned in the commit message or body. N5 (nit): body said main 316e950 was merged; the merge parent is 03226cc. N6 (minor): mutants C3, C8, C11, C12 survive (equivalent on today data, untested). N7 (latent, minor): presence uses git ls-files without --recurse-submodules, so a path inside a submodule would be reported missing; document or recurse. F-a: export is not byte-reproducible (PDF CreationDate and ID, meta.last_sync_timestamp in the DB), so git diff --exit-code after export is never clean; diff is the oracle; consider SOURCE_DATE_EPOCH and not bumping last_sync_timestamp on an unchanged export. F-b: export without reportlab raises a traceback instead of a clean refusal. F-c (process-doc): PR body refresh section could record the evidence-check pass and the 187/0 reportlab run. P1 (process-doc): body planned merge order omitted PR 16. P2: round-5 refresh section sits after the attribution footer and the 56/17 breakdown is stale (54 = 22/17/15). P3: evidence/workable-items/registry-hardening-pr8/final_check_references.txt is a dated snapshot showing 56. Related: HRD-111, HRD-085. N7 text of the review was read; exact wording of N4 to N6 beyond the above is UNKNOWN.

**Affected scope:** scripts/workable_items.py, scripts/verify_workable_items.py, scripts/check_registry_references.py, tests/test_workable_items.py, tests/test_check_registry_references.py, docs/workable-items.md, docs/helixtrack-sync.md.

**Reproduction / current observation:** Read the PR #10 review comments and list N1 to N7, F-a, F-b, F-c.

**Acceptance:** N1, N2, N6 each fixed with a test and a paired mutant that is killed; N3, N7, F-a, F-b fixed or closed with a recorded reason; the process-doc items closed with a recorded reason; validate and diff PASS.

**Closure criteria:** N1, N2, N6 each fixed with a test and a paired mutant that is killed; N3, N7, F-a, F-b fixed or closed with a recorded reason; the process-doc items closed with a recorded reason; validate and diff PASS.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T16:39:08Z / 2026-11-24T18:57:42Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded
- 2026-11-24 — Updated by AI: Transcribe PR 10 follow-ups from the review files (HRD-112); history kept; evidence: none recorded

## [HRD-094] Follow-ups of the reviews of pull requests #15 and #17

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** Open follow-ups of the independent reviews of pull requests 15 (HelixTrack sync tool, MERGED) and 17 (slow-client live tests, HRD-067, MERGED). PR 15 (pr15-review-e9aab9c.txt lines 89 to 130; the source-defect M1 is its own item HRD-119): M2 (minor): --record JSONL is written with json.dumps(ensure_ascii=False), so C1, DEL, bidi, zero-width and line-separator characters land raw in evidence files (98 of 493 hostile runs); use ensure_ascii=True. M3 (minor): two shape classes escape ShapeError (409/3006 with a 5000-digit version raises ValueError; two tickets claiming one key with ids 5 and s raise TypeError in sorted) and an int ticket id passes _id() but load_state() rejects it next run. M4 (minor): with --record a lone surrogate in a response aborts every run (UnicodeEncodeError, rc=3). N1 (nit): no overall request deadline (urllib timeout is per socket operation; a drip server held a run for more than 12 s with timeout 1). N2 (nit): KEY_RE and MARKER_RE use backslash-d which matches non-ASCII digits; use 0-9. N3 (test gaps): mutants id-bool-accepted, lastresort-no-report, safe_text-zs-allowed, syncerror-report-no-safe survive. N4 to N6 carried from the previous review (pr15-review-662a1f2.txt lines 49 to 60): N4 host paths in evidence logs (139 occurrences), N5 response bodies read without a size cap (a cap of 8 MiB is mentioned in the later review, verify), N6 the tool reads the generated outbox not the DB and nothing checks the outbox is current. Earlier N1 and N2 of that file: unmapped registry statuses (Ready for testing, In testing, Reopened, Obsolete) raise; docs/helixtrack-sync.md says 68 tests and 31 mutants but the suite has 110 tests. N3: --record file written with default umask (measured 664). N7 (nit, later review): PR body says 12 files not in the top seal, tree shows 13. PR 17 (pr17-review-47c7aeb.txt lines 72 to 78 and pr17-review-4f3f1ca.txt; N8 and the stale Mirror parity section are item HRD-118): N1 (source-defect minor, commit-message overclaim): pending bound is not checked at exactly 64, the lower tolerance lets 63 pass (mutant R01 survives); count non-terminal jobs immediately after rejection. N2: shutdown_bounded runs only with no job in flight, so cancel-on-shutdown is never exercised (R12 survives). N3 (usability): one HELIX_SLOW_CLIENT_OUT shared by both live tests makes the second fail; use per-test subdirectories. N4 (robustness): the post-arm drain loop has no watchdog (semaphore leak R08 hangs until the go test timeout). N5: Redis-unreachable skip has no closed reason code. N6 (body): Evidence section lists 2026-11-23 json files that PROVENANCE.md says are not evidence for this head. N7 (scope): the job test drives the in-memory store, not HTTP slow sockets; run_all_live.sh folding and tc/netem chaos are not done; HRD-067 must not be closed on this alone. From the earlier review (pr17-review-47b7f95.txt lines 73 to 77): catalog header claims pool backpressure holds but no assert covers pool state and stallHits is never asserted; PR body names the gates but not HELIX_TEST_REDIS_ADDR; catalog read-phase timeout is mostly enforced by the test driver wrapper. Previous text of this item said UNCONFIRMED; now transcribed.

**Affected scope:** tools/helixtrack-sync/helixtrack_sync.py and its tests, docs/helixtrack-sync.md, server/internal/httpapi and server/internal/catalog slow-client tests (HRD-067), evidence/campaign-2026-11-2x/slow-client.

**Reproduction / current observation:** Read the review comments of #15 and #17 and list every open follow-up.

**Acceptance:** Every listed follow-up fixed with a test that fails without the fix (the named surviving mutants killed) or closed with a recorded reason in its own commit carrying this id; validate passes.

**Closure criteria:** Every listed follow-up fixed with a test that fails without the fix (the named surviving mutants killed) or closed with a recorded reason in its own commit carrying this id; validate passes.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T16:39:08Z / 2026-11-24T18:57:42Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded
- 2026-11-24 — Updated by AI: Transcribe PR 15 and PR 17 follow-ups from the review files (HRD-112); history kept; evidence: none recorded

## [HRD-095] PR #16 review follow-ups F1 to F6

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** Open follow-ups of the independent reviews of PR #16 (HRD-027 and HRD-048, server load and soak tests), verdict files /home/milosvasic/.cache/claude-tmp/pr16-HRD-027-review-8e81a0f6.txt, pr16-HRD-027-review-c3d9d6cb.txt, pr16-HRD-027-review-5c73cf24.txt (all GO-with-follow-ups, no Important findings) and the earlier pr16-review-54ea2d7.txt and pr16-review-d9c49c6.txt. F1 (test-instrumentation, minor): soak detector parameters not pinned: mutants G4, G5, G8 survive, plus G9 (soakGoroutineSlop 3 to 2 survives; the slop test is relative to the constant); G10 (> to >= on the goroutine slop) is killed. F2 (test-instrumentation, minor): freshEvidenceDir, the HELIX_SOAK_ITERS cap and the rclone-probe gate are untested. F3 (usability, minor): a shared HELIX_LOAD_EVIDENCE_DIR across harnesses in one run makes the second harness t.Fatal. F4: hrd048 evidence not marked superseded (addressed in the PR body per the d9c49c6 review). F5 (observation): the server-mix soak is close to the growth and R2 gates; a longer run is owed. F6 (minor): tools mutate.sh writes /tmp/mut.$$ and has a vacuous HEAD body check. N2 (nit): the PR body Test types line lost its hermetic-versus-live qualifier (now only in the Scope section); restore it. O1 (observation): the author-only reviewer report with read-back (11.4.297(D)(6)) must be read back before merge (PR 16 is MERGED (gh pr view 16 on 2026-11-24: merge commit 49137a14f31607467e8cc0e1b543d4fe88e6b723, head 5c73cf24); the author-only reviewer report with read-back was owed before that merge and whether it was posted is UNKNOWN). F7 and F8 and N1 were body-text items addressed at 5c73cf2. Live legs (PostgreSQL, Redis, rclone, HELIX_LOAD_RUN, long soak) remain UNCONFIRMED. Related: HRD-027, HRD-045, HRD-048. Registered by HRD-104.

**Affected scope:** server/internal/httpapi (soak_test.go, soak_detector_gates_test.go, load_concurrent_test.go), server/internal/sources, mutate.sh (path UNKNOWN: read the PR 16 diff), evidence/campaign-2026-11-23/gap-fix/hrd048*

**Reproduction / current observation:** F1: apply mutants G4, G5, G8 or G9 (soakGoroutineSlop 3 to 2) to the soak detector and run go test -p 2 -count=1 -run TestSoakGate ./internal/httpapi: green. F3: run two harnesses in one process with HELIX_LOAD_EVIDENCE_DIR set: the second t.Fatal. F6: read mutate.sh.

**Acceptance:** Each of F1, F2, F3, F5, F6, N2 and O1 fixed with a test that fails without the fix (mutants G4, G5, G8, G9 killed) or closed with a recorded reason; F5 closed only by a longer soak run with captured evidence; python3 scripts/workable_items.py validate passes.

**Closure criteria:** Each of F1, F2, F3, F5, F6, N2 and O1 fixed with a test that fails without the fix (mutants G4, G5, G8, G9 killed) or closed with a recorded reason; F5 closed only by a longer soak run with captured evidence; python3 scripts/workable_items.py validate passes.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T16:39:09Z / 2026-11-24T18:57:41Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded
- 2026-11-24 — Updated by AI: Transcribe PR 16 review findings; registered by HRD-104; evidence: none recorded
- 2026-11-24 — Updated by AI: Correct PR 29 review finding 3 (HRD-112): PR 16 is MERGED; evidence: none recorded

## [HRD-096] PR #18 review follow-ups N1 to N6

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** Open follow-ups of the independent reviews of PR #18 (HRD-022 clients), verdict files /home/milosvasic/.cache/claude-tmp/pr18-HRD-022-review-5dd0a7bc.txt (round 4, GO-with-follow-ups at 5dd0a7bc), pr18-review-fdf73b1.txt (round 3), pr18-review-83b94d6.txt (round 2), pr18-review-81a6a31.txt (round 1). Round 4: (1) process-doc, minor: stale pending_evidence in clients/supported-platforms.json: 5 of the 8 paths (isolated/desktop 01_library.png, 02_detail.png, server_access.log, SHA256SUMS, hardening/desktop_sandbox_attempt.log) now exist on main sealed after PR #5 merged, yet the matrix still labels them as shipping in PR #5 and the linux note still names PR #5; remove them from pending_evidence, keep the 3 PR #9 ledger-snapshot paths only if still absent (PR #9 has since merged, so re-check), rewrite the linux note; keep linux BUILT while any cited path is pending. (2) process-doc, minor: the android row named Android phone and tablet is VERIFIED on an emulated device only, no tablet-size run; under 11.4.289 an emulated device is a test double, so VERIFIED claims more than the evidence shows; qualify the status (for example VERIFIED on emulator, or BUILT) or narrow the name until a physical phone and a tablet size run; operator decision. (3) process-doc, minor: PR body stale (head at open, 2 failures alone, GO and self-review PENDING). (4) test-instrumentation, nit: a VERIFIED row can drop its MANIFEST.sha256 entry without any test failing (reviewer mutant R3 survived); require a sealed manifest for VERIFIED rows (this is the item the conductor phrased as VERIFIED row need not cite MANIFEST). (5) carried items, below. Carried from round 3: N1 (test-instrumentation, minor) live-probe wiring unguarded (mutants O1, O4, O5 survive): extract the android-tv sub-check into a pure function taking adb devices output and getprop by serial, or test with a shim adb, with unauthorized and offline fixture lines, and add O1, O4, O5 to tests/mutate_supported_platforms.py; N2 (nit) find_adb has no test for an absolute ANDROID_HOME whose adb does not exist (O2); N3 (minor) stale pending_evidence is only a warnings.warn (O3 survives); make it fail once the cited PR has merged and check the PR number not only its prefix (round-2 F4(a)); N4 resolved; N5 the android-tv reproducer.command uses ~/Android/Sdk/platform-tools/adb and counts any attached Android device (grep -cw device); N6 reviewer assignment read-back owed. Carried from round 2: F7 main.js sandbox false, contextIsolation, nodeIntegration and the IPC sender check have no unit test (3 surviving mutants), gradle wrapper checksum not verified, Basic credentials over plain http; F2 Roadmap-Item trailer form on pushed commits 81a6a31, 49473ef, 83b94d6 (not fixable without a force-push, 11.4.113: accept and record). No HTTP/3 or Cronet claim exists in clients/android (h3 is out of this PR). Related: HRD-022, HRD-087. Registered by HRD-104.

**Affected scope:** clients/supported-platforms.json, tests/test_supported_platforms.py, tests/mutate_supported_platforms.py, clients/desktop/main.js and its tests, clients/android gradle wrapper, evidence/clients/*

**Reproduction / current observation:** (1) python3 -m unittest -v tests.test_supported_platforms: UserWarning about pending evidence that exists on main. (4) remove the MANIFEST.sha256 entries from the android evidence list: tests still pass (mutant R3). N1, N2, N3: reviewer mutants O1 to O5 in the round 3 review.

**Acceptance:** Each of round-4 findings 1 to 4 and carried N1, N2, N3, N5, F7 fixed with a killed mutant, or closed with a recorded reason (F2 closed as unfixable, recorded); android row status qualified or evidence extended by a physical-device run; python3 scripts/workable_items.py validate passes.

**Closure criteria:** Each of round-4 findings 1 to 4 and carried N1, N2, N3, N5, F7 fixed with a killed mutant, or closed with a recorded reason (F2 closed as unfixable, recorded); android row status qualified or evidence extended by a physical-device run; python3 scripts/workable_items.py validate passes.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T16:39:09Z / 2026-11-24T18:29:58Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded
- 2026-11-24 — Updated by AI: Transcribe PR 18 review findings; registered by HRD-104; evidence: none recorded

## [HRD-097] PR #19 review follow-ups m1 to m7

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** Minor and nit findings of the independent review of PR #19 (HRD-026 interop scripts), verdict files /home/milosvasic/.cache/claude-tmp/pr19-review-2122e89.txt (m1 to m7), pr19-HRD-026-review-07a317a9.txt (n1), pr19-review-5c8729e.txt (earlier NO-GO). m1 (test-instrumentation, minor): tools/interop/check_hygiene.sh misses equivalent shapes (rm -fr, rm -r -f, ST="$1" quoted) and does not check kill and pid identity, argv credentials or run-root and scratch checks (M8 to M17 survive); the scripts themselves are clean under a stronger instrument; widen the patterns and add the missed needles. m2 (process-doc, minor): credential-in-argv disclosure is incomplete: besides rclone --pass, $PP is in argv of rclone obscure (line 30) and sed -i s/$PP/redacted/ (line 43) and kosync_pull_probe.lua takes the password as argv[3]; throwaway random credentials on loopback; disclose or move to file or environment. m3 (nit): stop identity is exe-path equality not instance identity (recycled pid running the same binary would be signalled; add /proc/pid/stat start time or environ match) and stop deletes server.pid when it refused to signal. m4 (nit): RUNROOT (HRD026_RUN_ROOT or TMPDIR) is not validated absolute or non-root; providers.sh does not require ST under the run root; both scripts source $ST/env as code. m5 (nit): scan() uses python3 -c without -I; rclone may write its cache under the real HOME (UNCONFIRMED, not run). m6 (nit): PR body lists the three evidence paths twice; N6 unused spec and quoting residue left. m7 (minor, carried): non-parsing Roadmap-Item trailer on 5c8729e and 09dfaf1 (accepted, not fixable without a force-push) and the home path in frozen after_fix line 4 (disclosed) and the Redis image digest UNCONFIRMED. n1 (process-doc, minor, from the 07a317a9 review): PROVENANCE.md still calls hrd026/ open PR #10 although those 6 files are on main (commit 39e00c5), and main hrd026 files sit inside the sealed interop directory but are not in SHA256SUMS: two HRD-026 evidence sets, one sealed, one not; reconcile or seal in a follow-up with a successor note because PROVENANCE is frozen. Not exercised (UNCONFIRMED): any run with valid inputs, rclone, redis, server behaviour, trap cleanup, redaction effectiveness. Related: HRD-026. Registered by HRD-104.

**Affected scope:** tools/interop/check_hygiene.sh, tools/interop/hrd026_providers.sh, tools/interop/hrd026_server.sh, tools/interop/kosync_pull_probe.lua, evidence/campaign-2026-11-23/interop/ (PROVENANCE.md, SHA256SUMS, hrd026/)

**Reproduction / current observation:** m1: run tools/interop/check_hygiene.sh against a copy with the shapes rm -fr or ST="$1" appended: it passes. See the reviewer mutant list in pr19-review-2122e89-mutants.py and pr19-review-5c8729e-instruments.sh under /home/milosvasic/.cache/claude-tmp/.

**Acceptance:** Each of m1 to m7 and n1 fixed with a needle or test that fails without the fix, or closed with a recorded reason (m7 trailer and path are closed as unfixable and disclosed); sealed evidence reconciled for n1; python3 scripts/workable_items.py validate passes.

**Closure criteria:** Each of m1 to m7 and n1 fixed with a needle or test that fails without the fix, or closed with a recorded reason (m7 trailer and path are closed as unfixable and disclosed); sealed evidence reconciled for n1; python3 scripts/workable_items.py validate passes.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T16:39:10Z / 2026-11-24T18:29:58Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded
- 2026-11-24 — Updated by AI: Transcribe PR 19 review findings; registered by HRD-104; evidence: none recorded

## [HRD-098] PR #20 review follow-ups N1 to N3

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** Open follow-ups of the independent review of pull request 20 (scripts/pr/deliver.py evidence rules, HRD-083, MERGED), transcribed from pr20-review-9d36c11.txt lines 27 to 30. N1 (minor, test-instrumentation): the F4 fix (sorted(set(cited)) in pr_body) has no test; reviewer mutant M3 (drop the dedupe) survives; add a pr_body test with a path cited twice. N2 (minor, test-instrumentation): git log --topo-order in commits() has no mutant or test (disclosed); a test with two parallel branches and skewed GIT_COMMITTER_DATE would pin it. N3 (minor, process-doc, pre-existing): EVIDENCE_RE silently ignores malformed citation lines (short hash, Evidence: without a space); consider reporting malformed red, green or Evidence lines as INCONCLUSIVE rather than skipping them. Related: HRD-110, HRD-083.

**Affected scope:** scripts/pr/deliver.py, tests/test_pr_deliver.py, scripts/pr/mutate_deliver.py.

**Reproduction / current observation:** Read the PR #20 review and list N1 to N3.

**Acceptance:** N1 and N2 fixed with tests (mutant M3 and a topo-order mutant killed); N3 fixed or closed with a recorded reason; unittest tests.test_pr_deliver passes.

**Closure criteria:** N1 and N2 fixed with tests (mutant M3 and a topo-order mutant killed); N3 fixed or closed with a recorded reason; unittest tests.test_pr_deliver passes.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T16:39:10Z / 2026-11-24T18:57:43Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded
- 2026-11-24 — Updated by AI: Transcribe PR 20 follow-ups from the review file (HRD-112); history kept; evidence: none recorded

## [HRD-099] PR #21 finding F3: make the retry bound consumer data

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** Finding F3 of the review of PR #21 (HRD-084, constitution pin landing parity), file /home/milosvasic/.cache/claude-tmp/pr21-review-e722e6a.txt line 17 (minor, carried over from the constitution review): state that the retry bound is consumer DATA (11.4.35), already listed in the PR body. CORRECTION of the earlier text of this item: the review text names a statement to make (the retry bound for landing parity is consumer data), not a literal in code; UNKNOWN: whether a retry loop with a code literal exists anywhere; the loop and its location are in the constitution amendment 7bd8f37 (OWED-GATE-186 CM-LANDING-ALL-UPSTREAMS-READBACK) and must be read before work starts. Other notes of the PR 21 reviews (no action needed): F2 brief accuracy (7bd8f37 parent is 1fe6383); N1 commit e722e6a message claims 7bd8f37 on all six URLs (superseded, not rewritten); N2 the pin 74c415d is behind constitution main 7ac2926 by 3 commits, a later bump is a separate item; the 37abaeb delta review noted a stat-only check cannot detect a one-digit-off gitlink, exact sha plus ancestry checks are needed. Related: HRD-084. Registered by HRD-104. Resolved note (HRD-112): finding F1 of pr21-review-e722e6a.txt (HRD-084 absent from the main DB at that time) is resolved: HRD-084 now exists in the registry.

**Affected scope:** Constitution amendment 7bd8f37 text on landing parity and the PR 21 body; code location UNKNOWN until read

**Reproduction / current observation:** Read the constitution amendment 7bd8f37 (OWED-GATE-186 CM-LANDING-ALL-UPSTREAMS-READBACK) and the PR 21 body and decide whether any retry loop with a code literal exists; the review text names a statement to make, not a literal in code (UNKNOWN whether a literal exists).

**Acceptance:** The retry bound is stated as consumer DATA in the cited text, or the retry code location is found and its bound is read from configuration with a documented default and a test; HRD-099 closed with a recorded reason if no code literal exists; validate passes.

**Closure criteria:** The retry bound is stated as consumer DATA in the cited text, or the retry code location is found and its bound is read from configuration with a documented default and a test; HRD-099 closed with a recorded reason if no code literal exists; validate passes.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T16:39:10Z / 2026-11-24T18:57:41Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded
- 2026-11-24 — Updated by AI: Correct and transcribe PR 21 review finding F3; registered by HRD-104; evidence: none recorded
- 2026-11-24 — Updated by AI: Align scope and acceptance with the corrected description; registered by HRD-104; evidence: none recorded
- 2026-11-24 — Updated by AI: Correct PR 29 review findings 2 and 4 (HRD-112): align reproduction field, record resolved F1 of pr21-review-e722e6a.txt; evidence: none recorded

## [HRD-100] Constitution wave: decide doc_processor main versus master divergence (submodule pin)

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** The doc_processor submodule has diverged between main and master, so its pin cannot be placed on main. An operator decision is needed which branch is canonical, then the pin is moved. Related to HRD-053 (pins not on main), which stays the umbrella item; this item holds the single decision. No commit is made in a third-party or foreign repository without the decision. Evidence added by HRD-112 from pr27-HRD-102-review-70c3afd8.txt item 10 (2026-11-24): doc_processor gitlink 12a5fb7 equals the master tip; upstream main is 86f1a98; master is 126 commits ahead of main and main is 1 commit ahead of master (diverged); the 11.4.288(D) violation remains and is left to an operator decision.

**Affected scope:** submodules/doc_processor, helix-deps.yaml, docs/submodule-pins-on-main-2026-11-23.md.

**Reproduction / current observation:** git -C submodules/doc_processor branch -a and compare main with master and with the recorded gitlink.

**Acceptance:** Recorded operator decision (section 11.4.66), pin on the canonical main, helix-deps.yaml consistent, mirror parity table PRESENT on every upstream.

**Closure criteria:** Recorded operator decision (section 11.4.66), pin on the canonical main, helix-deps.yaml consistent, mirror parity table PRESENT on every upstream.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T16:39:11Z / 2026-11-24T18:57:45Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded
- 2026-11-24 — Updated by AI: Add PR 27 review evidence (HRD-112); evidence: none recorded

## [HRD-101] Constitution wave: fix llm_orchestrator gitlab-helix credentials so its pin can be fetched

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** The llm_orchestrator submodule cannot be fetched from the gitlab-helix remote because credentials are missing or rejected, so its pin cannot be verified on main (relates to HRD-053). Credentials must be provided through the environment by variable name only and are never written to the repository. UNCONFIRMED: whether the cause is a missing deploy key, an expired token or an access grant. Evidence added by HRD-112 from pr27-HRD-102-review-70c3afd8.txt item 9 (2026-11-24, re-measured with ls-remote): llm_orchestrator HEAD, main and master are all 8ee5795 (main parent c84c410), so that gitlink is on main now; the PR 27 body statement that it is a master tip one ahead of main is stale or wrong (UNCONFIRMED which); re-read the credential state before acting.

**Affected scope:** submodules/llm_orchestrator remotes, local git configuration of the operator host.

**Reproduction / current observation:** git ls-remote against the gitlab-helix remote of llm_orchestrator fails authentication.

**Acceptance:** git ls-remote succeeds with variable-name referenced credentials, no secret appears in tracked files or evidence, and the pin is verified reachable from the upstream main.

**Closure criteria:** git ls-remote succeeds with variable-name referenced credentials, no secret appears in tracked files or evidence, and the pin is verified reachable from the upstream main.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T16:39:11Z / 2026-11-24T18:57:45Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded
- 2026-11-24 — Updated by AI: Add PR 27 review evidence (HRD-112); evidence: none recorded

## [HRD-102] Constitution wave: bump challenges, containers and helixqa submodule pins on main

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** Move the pins of the challenges, containers and helixqa submodules to the latest commit on each upstream main by pull request in each own-org repository first, then the parent pin bump with a mirror parity table. Related to HRD-052, HRD-053 and HRD-084 (constitution pin update, not duplicated here). UNCONFIRMED: which of the three are already on main.

**Affected scope:** submodules/challenges, submodules/containers, submodules/helixqa, helix-deps.yaml.

**Reproduction / current observation:** Compare each recorded gitlink with the upstream main tip (git ls-remote).

**Acceptance:** Each gitlink equals or is an ancestor of the upstream main on every upstream, helix-deps.yaml refs agree, parity table PRESENT, and gates pass after the bump.

**Closure criteria:** Each gitlink equals or is an ancestor of the upstream main on every upstream, helix-deps.yaml refs agree, parity table PRESENT, and gates pass after the bump.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T16:39:11Z / 2026-11-24T16:39:11Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-103] Constitution wave: declare superspec as a third_party dependency in helix-deps.yaml

**Status:** Queued

**Type:** Task

**Severity:** Low

**Reopens:** 0

**Description:** Add superspec to helix-deps.yaml with third_party true and a licence field, as required by the dependency manifest rule that every non-own-org dependency is declared third-party. UNCONFIRMED: the licence text and ref have to be read from the superspec checkout.

**Affected scope:** helix-deps.yaml and the manifest verifier output.

**Reproduction / current observation:** scripts/deps_manifest verification reports superspec as an undeclared or mis-declared dependency (verify with the constitution manifest verifier).

**Acceptance:** helix-deps.yaml lists superspec with third_party true, a licence and a ref, and the manifest verifier passes.

**Closure criteria:** helix-deps.yaml lists superspec with third_party true, a licence and a ref, and the manifest verifier passes.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T16:39:11Z / 2026-11-24T16:39:11Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-104] Register review follow-ups of PRs 9, 16, 18, 19, 21, 22, 23, 24 and 25 as tracked items

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** Umbrella and change item for the registry pull request that records the minor follow-ups raised by the independent reviews of pull requests 9, 16, 18, 19, 21, 22, 23, 24 and 25 so none is lost (Constitution 11.4.197, 11.4.202, 11.4.214). It creates the cluster items HRD-105 to HRD-111 and transcribes the findings into the existing placeholder items HRD-095, HRD-096, HRD-097 and HRD-099 instead of minting duplicates. Sources are the review verdict files under /home/milosvasic/.cache/claude-tmp/. Closes when the registry pull request is merged and read back.

**Affected scope:** docs/workable_items.db and its generated exports (docs/Issues*, docs/Fixed*, docs/tracking/helixtrack-outbox.json), evidence/workable-items/results.json only if the qualification run rewrites it

**Reproduction / current observation:** Not a defect. Read the verdict files and compare with python3 scripts/workable_items.py diff.

**Acceptance:** validate and diff PASS; every follow-up named in the verdict files is present in exactly one item; HRD-095, HRD-096, HRD-097 and HRD-099 no longer say UNCONFIRMED about their texts; unit tests tests.test_workable_items and tests.test_pr_deliver pass.

**Closure criteria:** validate and diff PASS; every follow-up named in the verdict files is present in exactly one item; HRD-095, HRD-096, HRD-097 and HRD-099 no longer say UNCONFIRMED about their texts; unit tests tests.test_workable_items and tests.test_pr_deliver pass.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by User; assigned to Operator

**Created / modified:** 2026-11-24T18:28:30Z / 2026-11-24T18:28:30Z

**History:**

- 2026-11-24 — Opened by User: new-work-item; evidence: none recorded

## [HRD-105] PR #9 (HRD-051) review follow-ups F1 to F5: untested hidden-flag and future-timestamp branches, untested helpers, stale body, host paths

**Status:** Queued

**Type:** Task

**Severity:** Low

**Reopens:** 0

**Description:** Follow-ups of the independent review of PR #9 (gate-evidence ledger), verdict GO-with-follow-ups at head 0e52c228, file /home/milosvasic/.cache/claude-tmp/pr9-HRD-051-review-0e52c228.txt. F1 (test-instrumentation, minor): reviewer mutant X01 survived: dropping the hidden_flag check (assume-unchanged / skip-worktree on the snapshot file or PROVENANCE.tsv) from the SNAPSHOT_MODIFIED branch leaves both suites green; behaviour is correct today (rc=1 SNAPSHOT_MODIFIED), only the pin is missing. F2 (test-instrumentation, minor): mutant X02 survived: removing the snapshotted_at-not-in-the-future check leaves suites green because existing future tests also set captured_at in the future; needs a case with captured_at in the past and snapshotted_at in 2099. F3 (process-doc, minor): PR body was stale in the false-red direction (says gate red alone, 6/7, until PRs 5, 10, 18, 19 land; all four merged, head is 7/7 alone); PR 9 has since merged, so this is only a record note. F4 (test-instrumentation, minor, 11.4.224): scripts/gates/snapshot_evidence.sh has no test of its own; scripts/gates/gofmt_clean.sh reports clean when neither core/ nor server/ exists (vacuous clean, not reachable on this repo). F5 (nit): evidence logs and one snapshot embed absolute host paths of the operator home (privacy, not a secret). Also requested by the conductor and not found in the verdict files, UNKNOWN: HRD-052 transcript evidence and a weak LED-019 pointer; the details must be read from the PR 9 body (gh pr view 9) and docs/claim_reality_ledger.tsv before work starts. Related: HRD-051, HRD-052, HRD-111 (reportlab harness defect found by the same review).

**Affected scope:** scripts/gates/ (snapshot_evidence.sh, gofmt_clean.sh, build_evidence_manifest.sh, run_local_gates.sh and their tests), docs/claim_reality_ledger.tsv, evidence/campaign-2026-11-23/gap-fix/**

**Reproduction / current observation:** X01: in scripts/gates/build_evidence_manifest.sh remove the || [ -n "$(hidden_flag "$sfile")" ] clause from the SNAPSHOT_MODIFIED branch; run scripts/gates/test_build_evidence_manifest.sh: still all passed. X02: remove [ "$ts" -le "$now" ]; suites still pass. gofmt_clean.sh in a tree without core/ and server/ prints clean.

**Acceptance:** Each listed follow-up is either fixed with a test that fails without the fix (paired mutant killed) or closed with a recorded reason; evidence path cited; python3 scripts/workable_items.py validate passes. Specifically: a test pins X01 and X02 (mutants then KILLED); snapshot_evidence.sh has a test; gofmt_clean.sh refuses or reports SKIP when no Go tree exists; the LED-019 and HRD-052 transcript items are either transcribed with evidence or closed as not applicable.

**Closure criteria:** Each listed follow-up is either fixed with a test that fails without the fix (paired mutant killed) or closed with a recorded reason; evidence path cited; python3 scripts/workable_items.py validate passes. Specifically: a test pins X01 and X02 (mutants then KILLED); snapshot_evidence.sh has a test; gofmt_clean.sh refuses or reports SKIP when no Go tree exists; the LED-019 and HRD-052 transcript items are either transcribed with evidence or closed as not applicable.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by User; assigned to Operator

**Created / modified:** 2026-11-24T18:29:19Z / 2026-11-24T18:29:19Z

**History:**

- 2026-11-24 — Opened by User: new-work-item; evidence: none recorded

## [HRD-106] PR #25 (HRD-091) review follow-ups: reviewer assignment read-back, fixed symlink ignore list, broad *.bak rule, SHA256SUMS shallow wording

**Status:** Queued

**Type:** Task

**Severity:** Low

**Reopens:** 0

**Description:** Follow-ups of the independent review of PR #25 (packaging ignore rules and Koodo-reader bundle manifest), verdict GO-with-follow-ups at head 788b60ed, file /home/milosvasic/.cache/claude-tmp/pr25-HRD-091-review-788b60ed.txt. (1) PR reviewRequests was empty at review time; the 11.4.297(D) reviewer assignment or the author-only report with read-back was owed before merge (process; closes when read back on the platform). (2) The skill-symlink ignores in .gitignore are a fixed enumeration; a new constitution skill symlink needs a manual addition; consider a generated list. (3) The global *.bak ignore rule is repo-wide; acceptable now because no .bak is tracked, but broad; narrow it or add a gate that no tracked file is hidden by it. (4) Pre-existing wording discrepancy outside the PR: the bundle SHA256SUMS says shallow bundle while git bundle verify reports that it records a complete history. Related: HRD-091.

**Affected scope:** .gitignore, tests/test_packaging_ignore.py, docs/bundles/koodo-reader.manifest.json and its SHA256SUMS

**Reproduction / current observation:** (2) add a new symlink under .claude/skills and run git status: it shows as untracked. (3) git ls-files -ci --exclude-standard lists the files hidden by tracked-but-ignored rules (6 pre-existing at review). (4) git bundle verify on the Koodo-reader bundle versus the wording in SHA256SUMS.

**Acceptance:** Each listed follow-up is either fixed with a test that fails without the fix (paired mutant killed) or closed with a recorded reason; evidence path cited; python3 scripts/workable_items.py validate passes. Specifically: ignore list generated or a test fails on a new skill symlink; *.bak scope decided; SHA256SUMS wording matches git bundle verify.

**Closure criteria:** Each listed follow-up is either fixed with a test that fails without the fix (paired mutant killed) or closed with a recorded reason; evidence path cited; python3 scripts/workable_items.py validate passes. Specifically: ignore list generated or a test fails on a new skill symlink; *.bak scope decided; SHA256SUMS wording matches git bundle verify.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by User; assigned to Operator

**Created / modified:** 2026-11-24T18:29:20Z / 2026-11-24T18:29:20Z

**History:**

- 2026-11-24 — Opened by User: new-work-item; evidence: none recorded

## [HRD-107] PR #24 (HRD-090) review follow-ups F1 to F4: scratchPath isolating tests, test touching real HOME, make_hv_mutant_bins.sh path check, reviewer request

**Status:** Queued

**Type:** Task

**Severity:** Low

**Reopens:** 0

**Description:** Follow-ups of the independent review of PR #24 (Go mutation and UI verifiers), verdict GO-with-follow-ups at head a6d16895, file /home/milosvasic/.cache/claude-tmp/pr24-HRD-090-review-a6d16895.txt (a second verdict file for head ce78b51a holds only the verdict header). F1 (test-instrumentation, minor): reviewer mutants that survive go test in scratchPath: (B) the descendant loop starting at the resolved path (accepts .helix-verify itself) and (C) the final-symlink Lstat refusal dropped; behaviour stays safe through the marker and descendant checks of the resolved target, but those two layers are untested although test comments claim them; add isolating tests (the marker directory itself; a symlink to a marked scratch). F2 (hygiene, minor, 11.4.287(C)(3)): tools/verify scratch_test.go line 65 calls removeScratch on the real HOME and its parent (refusal expected); use t.Setenv("HOME", tempdir) and a sandbox victim. F3 (minor): qa/isolation/make_hv_mutant_bins.sh checks OUT with a case pattern before realpath, so $HOME/.cache/claude-tmp/../../x passes and a later rm -rf targets $HOME/x/src/verify; validate after canonicalising or reject .. components. F4 (merge-time process): reviewRequests empty; the 11.4.297(D) request and read-back were owed. Related: HRD-090. The PR was not re-read by the registering agent; the line numbers are the reviewer's.

**Affected scope:** tools/verify (scratch.go, scratch_test.go), qa/isolation/make_hv_mutant_bins.sh

**Reproduction / current observation:** F1: apply reviewer mutants (B) and (C) to scratchPath and run go test ./...: both survive. F3: OUT="$HOME/.cache/claude-tmp/../../x" qa/isolation/make_hv_mutant_bins.sh in a sandbox HOME with a fake rm: the later deletion targets $HOME/x/src/verify.

**Acceptance:** Each listed follow-up is either fixed with a test that fails without the fix (paired mutant killed) or closed with a recorded reason; evidence path cited; python3 scripts/workable_items.py validate passes. Specifically: mutants (B) and (C) killed by new tests; no test passes the real HOME to a deleting helper; make_hv_mutant_bins.sh refuses a non-canonical OUT before any delete.

**Closure criteria:** Each listed follow-up is either fixed with a test that fails without the fix (paired mutant killed) or closed with a recorded reason; evidence path cited; python3 scripts/workable_items.py validate passes. Specifically: mutants (B) and (C) killed by new tests; no test passes the real HOME to a deleting helper; make_hv_mutant_bins.sh refuses a non-canonical OUT before any delete.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by User; assigned to Operator

**Created / modified:** 2026-11-24T18:29:20Z / 2026-11-24T18:29:20Z

**History:**

- 2026-11-24 — Opened by User: new-work-item; evidence: none recorded

## [HRD-108] PR #23 (HRD-089) review follow-ups F-1 to F-3: Redis INFO without run_id, restored Redis identity, PR body note

**Status:** Queued

**Type:** Task

**Severity:** Low

**Reopens:** 0

**Description:** Follow-ups of the independent review of PR #23 (Go live verifiers), verdict GO-with-follow-ups at head 25304a78 (delta GO at merge 72cffa97 carries the same follow-ups), files /home/milosvasic/.cache/claude-tmp/pr23-HRD-089-review-25304a78.txt and pr23-HRD-089-review-72cffa97.txt (bb1b53c9 file holds only the verdict header). F-1 (test-instrumentation, minor): reviewer mutant survived: accepting a Redis INFO reply that lacks run_id; add a test that such a reply leaves the redis identity unresolved (the run then fails). F-2: the restored Redis identity in the backup-restore verifier is not recorded; the PR states this as UNCONFIRMED; record it. F-3 (process-doc): the PR body could cite the dirty-flag reproduction (binaries bit-identical to the source of the PR head) so readers need not repeat it; the PR has since merged, so this is a record note unless the evidence README is extended. Not re-run by the reviewer: live-server and backup-restore legs (no PostgreSQL test DSN or Redis on the review host), UNCONFIRMED. Related: HRD-089.

**Affected scope:** server/internal/liveverify (identity.go, review_fixes_test.go), the backup-restore verifier command, evidence/ for backup-restore-go

**Reproduction / current observation:** F-1: in server/internal/liveverify/identity.go make the Redis INFO parser accept a reply without run_id; go test -p 2 -count=1 ./internal/liveverify/... stays green.

**Acceptance:** Each listed follow-up is either fixed with a test that fails without the fix (paired mutant killed) or closed with a recorded reason; evidence path cited; python3 scripts/workable_items.py validate passes. Specifically: a test kills the run_id mutant; the restored Redis identity is recorded in the backup-restore evidence or the gap is stated as UNKNOWN with a reason.

**Closure criteria:** Each listed follow-up is either fixed with a test that fails without the fix (paired mutant killed) or closed with a recorded reason; evidence path cited; python3 scripts/workable_items.py validate passes. Specifically: a test kills the run_id mutant; the restored Redis identity is recorded in the backup-restore evidence or the gap is stated as UNKNOWN with a reason.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by User; assigned to Operator

**Created / modified:** 2026-11-24T18:29:20Z / 2026-11-24T18:29:20Z

**History:**

- 2026-11-24 — Opened by User: new-work-item; evidence: none recorded

## [HRD-109] PR #22 (HRD-086) review follow-ups F1 to F4: reciprocal cross-links, exact evidence paths, scope wording

**Status:** Queued

**Type:** Task

**Severity:** Low

**Reopens:** 0

**Description:** Follow-ups of the independent review of PR #22 (registry allocation of HRD-086 to HRD-103), verdict GO-with-follow-ups at head 2b3b5aef, file /home/milosvasic/.cache/claude-tmp/pr22-review-2b3b5ae.txt. F1 (minor): HRD-100, HRD-101 and HRD-102 carve single decisions out of the umbrella HRD-053 (HRD-102 also overlaps HRD-052) and link to it in prose only; composes_with is empty and HRD-053 and HRD-052 do not link back; 11.4.214 asks for links both ways; add reciprocal cross-links. F2 (minor): HRD-089 and HRD-090 cite their evidence only as under /home/milosvasic/.cache/claude-tmp with several copies and a volatile scratch area; record the exact path plus sha256 or seal the evidence under evidence/ (PRs 23 and 24 may already have sealed it; verify). F3 (nit): HRD-086 affected_scope lists docs/Fixed* although the change did not modify them (harmless upper bound). F4 (nit): HRD-087, HRD-089, HRD-090 affected_scope name paths that were not on main at the time (clients/desktop, clients/android, qa/h3, server/internal/liveverify, tools/verify, cmd/helix-verify-*); state the branch or PR for each; HRD-103 cites scripts/deps_manifest, which is in the constitution submodule, not the root. Note: the update subcommand has no flag for composes_with, so reciprocal links are written in the description text unless the tool gains such a flag (UNKNOWN: whether it should).

**Affected scope:** Registry items HRD-052, HRD-053, HRD-086, HRD-087, HRD-089, HRD-090, HRD-100, HRD-101, HRD-102, HRD-103 (text only)

**Reproduction / current observation:** python3 scripts/workable_items.py export, then read docs/Issues.md for HRD-100 to HRD-102: no reference back from HRD-053 or HRD-052.

**Acceptance:** Each listed follow-up is either fixed with a test that fails without the fix (paired mutant killed) or closed with a recorded reason; evidence path cited; python3 scripts/workable_items.py validate passes. Specifically: HRD-053 and HRD-052 mention HRD-100, HRD-101, HRD-102 and the reverse; HRD-089 and HRD-090 cite an exact evidence path with sha256 or a sealed evidence directory.

**Closure criteria:** Each listed follow-up is either fixed with a test that fails without the fix (paired mutant killed) or closed with a recorded reason; evidence path cited; python3 scripts/workable_items.py validate passes. Specifically: HRD-053 and HRD-052 mention HRD-100, HRD-101, HRD-102 and the reverse; HRD-089 and HRD-090 cite an exact evidence path with sha256 or a sealed evidence directory.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by User; assigned to Operator

**Created / modified:** 2026-11-24T18:29:20Z / 2026-11-24T18:29:20Z

**History:**

- 2026-11-24 — Opened by User: new-work-item; evidence: none recorded

## [HRD-110] scripts/pr/deliver.py open regenerates the whole PR body and overwrites the Test types line (and any hand-edited body text) on a re-run

**Status:** Queued

**Type:** Bug

**Severity:** Low

**Reopens:** 0

**Description:** Reported by the conductor, verified by reading scripts/pr/deliver.py on github/main f1d63855: cmd_open, when the pull request already exists, calls gh pr edit --body with the body rebuilt by pr_body(), whose Test types run section is a.test_types or the literal NOT STATED. Re-running open without repeating --test-types therefore replaces a previously written Test types line with NOT STATED and discards every manual edit of the body (self-review notes, UNCONFIRMED lists, merge order). Observed on pull requests whose bodies were edited by hand after open (see /home/milosvasic/.cache/claude-tmp/pr16-body-old.txt and pr16-body-new.txt, pr21-body-orig.txt and pr21-body-new.txt, pr5-body-*.txt). Two further delivery follow-ups reported by the conductor and NOT yet verified by the registering agent, UNKNOWN: (a) deliver.py retire prints nothing when run in the background (reading the code, cmd_retire ends with say(RETIRED ...), so the cause may be stdout buffering or the harness; reproduce first); (b) a task branch lingering on a remote after retire (cmd_retire deletes only on ctx.remote and does not delete on the mirror remotes in cfg mirror_remotes; the lingering ref named by the conductor, origin/task/..., is UNKNOWN until reproduced).

**Affected scope:** scripts/pr/deliver.py (cmd_open, pr_body, cmd_retire), tests/test_pr_deliver.py, docs/pr-delivery.md

**Reproduction / current observation:** (1) With a hermetic fake gh as in tests/test_pr_deliver.py: run open --test-types X, then open again without it: the body now says NOT STATED. (2) and (3) UNKNOWN: reproduce retire in the background and with a mirror remote that still holds the branch.

**Acceptance:** open on an existing pull request preserves a previously recorded Test types line or refuses without --test-types; hand-edited sections are not silently discarded (or the tool says it will replace them and requires an explicit flag); retire output reaches the log when backgrounded and the lingering-branch claim is either reproduced and fixed or closed as not reproducible with evidence (11.4.292); paired mutants in tests.test_pr_deliver; python3 -m unittest tests.test_pr_deliver passes.

**Closure criteria:** open on an existing pull request preserves a previously recorded Test types line or refuses without --test-types; hand-edited sections are not silently discarded (or the tool says it will replace them and requires an explicit flag); retire output reaches the log when backgrounded and the lingering-branch claim is either reproduced and fixed or closed as not reproducible with evidence (11.4.292); paired mutants in tests.test_pr_deliver; python3 -m unittest tests.test_pr_deliver passes.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by User; assigned to Operator

**Created / modified:** 2026-11-24T18:29:21Z / 2026-11-24T18:29:21Z

**History:**

- 2026-11-24 — Opened by User: new-work-item; evidence: none recorded

## [HRD-111] verify_workable_items.py counts a skipped reportlab mutation test as FAIL (mutation-pdf-values-unescaped): environmental SKIP reported as FAIL

**Status:** Queued

**Type:** Bug

**Severity:** Low

**Reopens:** 0

**Description:** Found by the independent review of PR #9 (file /home/milosvasic/.cache/claude-tmp/pr9-HRD-051-review-0e52c228.txt): python3 scripts/verify_workable_items.py reports 186 pass and 1 fail on both the PR head and github/main; the single failing check is mutation-pdf-values-unescaped, whose detail is test skipped 'reportlab unavailable' because reportlab is not installed on the host. The mutation harness counts a skipped inner test as FAIL, so an environmental absence turns the qualification red. This is a pre-existing FAIL on main: reportlab is not installed on the host (docs/workable-items.md lines 64 to 67 only say that PDF export needs ReportLab; CLAUDE.md carries no note about an expected FAIL, per the review of PR 29, finding 1). With PYTHONPATH=/home/milosvasic/.cache/helix-docs-deps the same run gives 187 pass and 0 fail. Under 11.4.3 and 11.4.69 an absent dependency is an honest SKIP with a closed reason code, never a FAIL and never a PASS. UNKNOWN: whether the inner skip is reported by unittest as skipped in a way the harness can parse; read scripts/verify_workable_items.py and tests/test_workable_items.py before fixing. Related: HRD-105.

**Affected scope:** scripts/verify_workable_items.py, tests/test_workable_items.py, evidence/workable-items/results.json

**Reproduction / current observation:** python3 scripts/verify_workable_items.py on a host without reportlab: 1 fail (mutation-pdf-values-unescaped); restore evidence/workable-items/results.json afterwards with git checkout.

**Acceptance:** The harness records SKIP with a reason code from the closed 11.4.69 set when reportlab is absent (UNKNOWN: which member fits a missing Python library; hardware_not_present is questionable because a library is not hardware; pick from the closed set later, and if no member fits that is itself a finding for the constitution), FAIL only when the mutation test runs and does not detect the mutant; results.json shows 0 fail on a host without reportlab; with reportlab installed the mutation is still killed; unit tests pass.

**Closure criteria:** The harness records SKIP with a reason code from the closed 11.4.69 set when reportlab is absent (UNKNOWN: which member fits a missing Python library; hardware_not_present is questionable because a library is not hardware; pick from the closed set later, and if no member fits that is itself a finding for the constitution), FAIL only when the mutation test runs and does not detect the mutant; results.json shows 0 fail on a host without reportlab; with reportlab installed the mutation is still killed; unit tests pass.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by User; assigned to Operator

**Created / modified:** 2026-11-24T18:29:21Z / 2026-11-24T18:57:41Z

**History:**

- 2026-11-24 — Opened by User: new-work-item; evidence: none recorded
- 2026-11-24 — Updated by AI: Correct PR 29 review finding 1 (HRD-112): drop nonexistent CLAUDE.md attribution and the questionable reason code; evidence: none recorded

## [HRD-112] Register review follow-ups of PRs 5, 10 to 15, 17, 20, 26 to 28 and correct the minor errors of the PR 29 review (second registry follow-up)

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** Second registry follow-up after PR 29 (HRD-104). (A) Corrects the minor errors the review of PR 29 found (file pr29-HRD-104-review-4ec8117d.txt): HRD-111 no longer cites a nonexistent CLAUDE.md note, HRD-099 reproduction field aligned, HRD-095 states PR 16 is MERGED, HRD-099 records the resolved F1 of pr21-review-e722e6a.txt. (B) Registers follow-ups of the independent reviews of PRs 5, 10, 11, 12, 13, 14, 15, 17 and 20 that had no registry text, per 11.4.214 (checked against existing items first; HRD-092, HRD-093, HRD-094 and HRD-098 were placeholders marked UNCONFIRMED and are updated with the transcribed text, their history is kept) and new items HRD-113 to HRD-119. (C) Registers follow-ups of the reviews of PRs 26, 27 and 28 (HRD-120, HRD-121). HRD-100 and HRD-101 already exist and are not duplicated. Registered by HRD-112. This item changes only the registry and its generated documents; no code.

**Affected scope:** docs/workable_items.db and the generated docs/Issues*, docs/Fixed*, docs/tracking/helixtrack-outbox.json

**Reproduction / current observation:** python3 scripts/workable_items.py validate and diff after the change; the review files named in each item are in /home/milosvasic/.cache/claude-tmp/.

**Acceptance:** Every transcribed follow-up is traceable to its review file and line or marked UNKNOWN; validate and diff PASS; tests.test_workable_items and tests.test_pr_deliver pass.

**Closure criteria:** Every transcribed follow-up is traceable to its review file and line or marked UNKNOWN; validate and diff PASS; tests.test_workable_items and tests.test_pr_deliver pass.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T18:57:41Z / 2026-11-24T18:57:41Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-113] PR #11 (HRD-050 gate) review follow-ups: ls -A failure, TMPDIR in repo, test gaps t1 to t3, newline paths, fixtures, live run

**Status:** Queued

**Type:** Task

**Severity:** Low

**Reopens:** 0

**Description:** Open non-blocking follow-ups of PR 11 (HRD-050 gate logic, tree-identical to PR 4 reviewed head 2da41b5) from the author self-review pr11-selfreview.txt (the independent GO file evidence/reviews/pr11-HRD-050-review-e312849.txt was in the reviewer worktree and is UNKNOWN here; wording below is from the self-review and pr11body.md lines 43 to 72): m1 an ls -A failure reads as an empty out directory; m3 TMPDIR inside the repo; t1, t2, t3 test gaps (texts UNKNOWN); F3 newline in paths mangles the report (fix not done); F4 extra fixtures and mutants OM4 and OM9 not done; n1 PR body cites a review file not in the tree; F8 a fresh live end-to-end run is owed (HRD-050 stays PROVISIONAL; the sealed live run was red: TestLiveLoadIngestConcurrentWithReads). Disclosure: review rounds exceeded the default budget and authorisation beyond one bounded round at round 6 is UNCONFIRMED as a separately recorded decision. Related: HRD-050, HRD-051.

**Affected scope:** HRD-050 gate scripts and tests named in PR 11 (paths UNKNOWN: read the PR 11 diff), evidence/reviews.

**Reproduction / current observation:** Read pr11body.md and the PR 11 diff; run the gate with an ls -A failure and a newline path.

**Acceptance:** Each of m1, m3, t1 to t3, F3, F4, n1 fixed with a test that fails without the fix or closed with a recorded reason; F8 closed only by a fresh live run with captured evidence; validate passes.

**Closure criteria:** Each of m1, m3, t1 to t3, F3, F4, n1 fixed with a test that fails without the fix or closed with a recorded reason; F8 closed only by a fresh live run with captured evidence; validate passes.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T18:57:43Z / 2026-11-24T18:57:43Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-114] PR #12 (HRD-070 fuzz targets) review follow-ups F1 to F5, I1, M1 to M5, N-A

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** Open follow-ups of the independent reviews of PR 12 (core/pkg/lcpfuzz test-only fuzz targets v2), from pr12-review-5aedf44.txt lines 23 to 47 and pr12-review-5aedf44-final.txt line 21 to 32. F1 (test-instrumentation, important): reviewer mutant R1 survives: in canonical.go withoutTopLevelMember the first strings.TrimLeft(..., space tab CR LF) changed to space tab only makes valid JSON with a newline before the colon fail CanonicalForSigning (probe in={"a"\n:1,"signature":{}} out="" err not valid JSON). F2: mutant R3 survives: strconv.Quote(key)+":" replaced by a hand-quoted form; no signing-path test uses keys that need escaping (same line as disclosed follow-up I1, canonical.go:152); close the class (whitespace around colon, escaped keys, control-character keys) rather than per instance. F3 (process-doc): rulemap.tsv X18 row is hand-edited after enumerate_rules.py map regenerated it (NOTES says Regenerated); disclose or teach map to take the why from equiv.tsv/oracle.tsv. F4 (process-doc): PR body reference R07 survives is wrong (R07_lsd_link_type_dropped is KILLED in hrd070-r7/mutation.log line 473); rename to F2. F5: see HRD-115 (production bug N1, filed as its own Bug). I1 (disclosed): canonical.go:152 strconv.Quote. M1 to M5: listed in the PR 12 body as not fixed; their text is UNKNOWN (not in the review files read). N-A (process-doc, minor): a PR comment by the author account still carries the earlier body text (298 killed, R07 survives); add a reply saying it is superseded. Related: HRD-070, HRD-115.

**Affected scope:** core/pkg/lcpfuzz (canonical.go, tests), evidence hrd070-r5..r7, rulemap.tsv, PR 12 body and comments.

**Reproduction / current observation:** Apply mutant R1 or R3 to withoutTopLevelMember and run the PR suite (go test ./pkg/lcpfuzz -run Fuzz|TestHRD070|TestSeed): green.

**Acceptance:** F1 and F2 killed by new tests as a class; F3, F4, N-A closed with a recorded reason; M1 to M5 transcribed from the PR 12 body then fixed or closed; validate passes.

**Closure criteria:** F1 and F2 killed by new tests as a class; F3, F4, N-A closed with a recorded reason; M1 to M5 transcribed from the PR 12 body then fixed or closed; validate passes.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T18:57:43Z / 2026-11-24T18:57:43Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-115] LCP license Parse accepts case-variant member names (ID accepted as id, listed as unknown member too)

**Status:** Queued

**Type:** Bug

**Severity:** Medium

**Reopens:** 0

**Description:** Production bug found by the review of PR 12 (pr12-review-b1ecbb9.txt line 35, N1; carried as F5 in pr12-review-5aedf44.txt line 47): Parse accepts case-variant member names through Go case-insensitive struct matching. Probe: a document with "ID":"x" instead of "id" is ACCEPTED with id="x" AND UnknownMembers=[ID], which is self-contradictory (the same member is used and listed unknown). LCP and JSON member names are case-sensitive; required-member checks can be satisfied by wrongly cased members. It is also the root of finding B1 of that review. Not fixed in PR 12 (test-only). UNCONFIRMED whether other required members behave the same. Related: HRD-031, HRD-070, HRD-114.

**Affected scope:** core lcp license Parse (path UNKNOWN: read the PR 12 and PR 13 diffs), core/pkg/lcpfuzz tests.

**Reproduction / current observation:** Parse a license JSON with the member written ID instead of id and read id and UnknownMembers (the probe is in the PR 12 review b1ecbb9, zz_probe2_test.go).

**Acceptance:** Parse rejects or ignores case-variant member names consistently (required member not satisfied by a wrongly cased one, never both used and unknown); a RED test fails before the fix and passes after, with the check on other required members; validate passes.

**Closure criteria:** Parse rejects or ignores case-variant member names consistently (required member not satisfied by a wrongly cased one, never both used and unknown); a RED test fails before the fix and passes after, with the check on other required members; validate passes.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T18:57:43Z / 2026-11-24T18:57:43Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-116] PR #13 (HRD-031 LCP server wiring) review follow-ups I1 to I4, A6/A7, proxy doc note, N3, trailer layout, sealed evidence re-run

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** Follow-ups of the independent reviews of PR 13 (pr13-review-8ccba96.txt lines 102 to 115 and pr13-review-ee8e478.txt lines 83 to 94; pr13body.md lines 34 to 39). The review required them to be filed as tracked items before merge; the PR body says they were not filed in the PR. I1: redirect user:password refusal untested (mutant M2). I2: second GC lease check untested in isolation (M6, M7 equivalent alone). I3: redirect scheme check caught only by a reviewer test (M3). I4: cache-hit hash compare (mutant M11) is caught by no PR test: a corrupted cached artifact is never refetched by the PR own tests. A6 and A7: adopt the reviewer tests (reviewer tests R1, R2, R16, R17). m3 (performance): GC Collect calls repo.ReferencedLCPArtifacts once per reclaim candidate, O(N) full catalog reads per run. m4 (documentation): Proxy is nil so HTTP(S)_PROXY is ignored on purpose (SSRF hardening); the LCPFromEnv variable list does not say so. N3: 2.0 plain-link href assertion gap (mutant n12). Trailer layout defect (process-doc): Roadmap-Item and Out-of-scope lines are in a paragraph before Co-Authored-By so git interpret-trailers sees only Co-Authored-By, in commits 8ccba96 and ee8e478 (cannot be amended, 11.4.113). Sealed HRD-070 evidence (hrd070-r5..r7 mutation logs) references two test names that were not edited: re-run owed. Related: HRD-031, HRD-070, HRD-075.

**Affected scope:** server/internal LCP fulfilment, GC and OPDS code and tests named in PR 13 (paths UNKNOWN: read the PR 13 diff), docs for LCPFromEnv, evidence hrd070-r5..r7.

**Reproduction / current observation:** Apply mutants M2, M3, M6+M7 and M11 from the PR 13 review and run the PR tests: the PR suite is green for I1 to I4.

**Acceptance:** I1 to I4 each covered by a test that kills its mutant (reviewer tests adopted); m3, m4, N3 fixed or closed with a recorded reason; sealed evidence re-run; validate passes.

**Closure criteria:** I1 to I4 each covered by a test that kills its mutant (reviewer tests adopted); m3, m4, N3 fixed or closed with a recorded reason; sealed evidence re-run; validate passes.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T18:57:44Z / 2026-11-24T18:57:44Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-117] PR #14 (HRD-036 remote HTTP Range and revalidation) review follow-ups N2 to N6, N10 to N20

**Status:** Queued

**Type:** Task

**Severity:** Low

**Reopens:** 0

**Description:** Open non-blocking follow-ups of PR 14 from pr14-review-4f21293.txt lines 30 to 46, pr14-review-706414e.txt lines 41 to 47, pr14-review-27a637d.txt lines 36 to 64 and pr14-review-782478d.txt lines 71 to 142 (final verdict GO-with-follow-ups). N2: Revalidate to OpenRange is not atomic (no If-Match or If-Unmodified-Since on the remote GET) so a file replaced between PROPFIND and GET can be served under the scan-time ETag; the router comment overstates the guarantee. N3 and N14 and N19 (test-instrumentation): surviving mutants O7, O10 (end clamp), O11, O12, O9b, Z3, Z5, Z6, Z15c (total != size to total > size: a shrunk origin would be accepted); no rclone Revalidate unit test. N4: rclone Revalidate maps any stderr containing not found to ErrNotFound (a missing --config file answers 404 instead of 502, confirmed live) and the traversal check HasPrefix(name, "..") rejects a legal ..x.epub. N5 (pre-existing): unknown range unit and Bytes= are answered 416 while RFC 9110 14.2 says an unknown unit must be ignored. N6 (pre-existing): no rclone concurrency cap (two rclone processes per remote request). N7 (process): Roadmap-Item trailer not parsed by git for commit 4f21293 (cannot be amended). N10: origin answers 206 with a short body gives a truncated response (unexpected EOF) once headers are sent. N11 (body accuracy). N12 and N13: full GET trusts the PROPFIND size for Content-Length; same size and mtime but changed bytes served under the old strong ETag. N15: no If-None-Match or 304 handling. N16: duplicate Content-Range header read from the first value only (wrong bytes possible); refuse unless exactly one. N17: unsolicited Content-Encoding on a 206 or 200 passed through raw under the identity ETag. N18: range unit is case-insensitive in RFC 9110 (BYTES refused with 502; conservative). N20 (body accuracy): Evidence section omits hrd036-r5 and has a missing newline. Related: HRD-036.

**Affected scope:** server/internal/sources (webdav.go, rclone.go), server/internal/httpapi/router.go, their tests, PR 14 body.

**Reproduction / current observation:** Apply the reviewer mutants named above (Z15c, Z5, Z6, O10) and run go test ./internal/sources ./internal/httpapi: green; for N4 run an rclone source with a missing --config file.

**Acceptance:** Each of N2, N4, N5, N10, N12, N13, N16, N17, N18 fixed with a test that fails without the fix or closed with a recorded reason; the surviving mutants of N3 and N19 killed; N6, N15 decided; validate passes.

**Closure criteria:** Each of N2, N4, N5, N10, N12, N13, N16, N17, N18 fixed with a test that fails without the fix or closed with a recorded reason; the surviving mutants of N3 and N19 killed; N6, N15 decided; validate passes.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T18:57:44Z / 2026-11-24T18:57:44Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-118] Skips in httpapi and catalog tests carry no closed 11.4.69 reason code (26 skips)

**Status:** Queued

**Type:** Task

**Severity:** Low

**Reopens:** 0

**Description:** Follow-up N8 of the review of PR 17 (pr17-review-4f3f1ca.txt line 37): in go test ./internal/httpapi ./internal/catalog (127 PASS, 28 SKIP, 0 FAIL at head 4f3f1ca) 26 skips carry no closed reason code: 2 with HELIX_LOAD_RUN is not 1; measurement harness opted out, 23 with HELIX_TEST_POSTGRES_DSN is not configured, 1 with set HELIX_LCP_LIVE_BIN and HELIX_LCP_LIVE_EVIDENCE. Only 2 use feature_disabled_by_config. They belong to tests outside PR 17 (pre-existing). Also from the same review: the PR 17 body Mirror parity section was stale for the new head (process-doc, minor; PR is merged, nothing to fix there). The set of codes for an absent dependency is the closed 11.4.69 set (UNKNOWN which member fits a missing DSN, address or binary: read the set before choosing). Related: HRD-067, HRD-111.

**Affected scope:** server/internal/httpapi and server/internal/catalog tests (skip calls).

**Reproduction / current observation:** go test -p 2 -count=1 -v ./internal/httpapi ./internal/catalog and list SKIP reasons without a closed code.

**Acceptance:** Every skip message starts with a code from the closed 11.4.69 set, enforced by a test that fails when a skip has no code (paired mutant); skips still never count as PASS; validate passes.

**Closure criteria:** Every skip message starts with a code from the closed 11.4.69 set, enforced by a test that fails when a skip has no code (paired mutant); skips still never count as PASS; validate passes.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T18:57:44Z / 2026-11-24T18:57:44Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-119] HelixTrack sync: final --report is written after the lock is released, a later run marker can be overwritten

**Status:** Queued

**Type:** Bug

**Severity:** Medium

**Reopens:** 0

**Description:** Source defect M1 of the review of PR 15 (pr15-review-e9aab9c.txt lines 89 to 105): the final --report is written AFTER the lock is released (main(): finally lock.release() then save_json_atomic(args.report, out), lines 1015 to 1020 of tools/helixtrack-sync/helixtrack_sync.py at that head). A run B that starts in that window acquires the lock and writes its InProgress marker, then run A overwrites the marker with its completed result. Reproduced by the reviewer (race.py): while B runs, and permanently if B is killed, the report shows the previous run as the latest. This contradicts the code comment at line 963 and the PR body claim that a stale report cannot survive, also after a kill. Narrow window; state and remote stay correct and the next run reconciles, so not data loss. Fix per the review: write the final report (and the compare report) before releasing the lock, optionally add a run id and timestamp to the report, add a test, correct the body sentence. Related: HRD-094, HRD-029.

**Affected scope:** tools/helixtrack-sync/helixtrack_sync.py (main, lock handling, save_json_atomic), its tests.

**Reproduction / current observation:** Reviewer race.py: in-process run A with a release hook that starts a real-CLI run B against a slow loopback server; read the report while B runs.

**Acceptance:** A RED test reproduces the overwritten marker; after the fix the report is written before the lock is released and the test passes; the PR 15 body sentence corrected or noted; validate passes.

**Closure criteria:** A RED test reproduces the overwritten marker; after the fix the report is written before the lock is released and the test passes; the PR 15 body sentence corrected or noted; validate passes.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T18:57:44Z / 2026-11-24T18:57:44Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-120] PR #26 (HRD-087 Cronet and HTTP/3 clients) review follow-ups: verdict identity, access-log flag, ApiRequestPolicy, JVM test not executed

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** Open follow-ups of the reviews of PR 26 (pr26-HRD-087-review-c186a013.txt lines 5 to 16 and pr26-HRD-087-review-3c0f920b.txt; also PR 27 and 28 reviews have no finding here). (1) Verdicts lack server and APK identity (11.4.289(D)): the verdicts do not record the sha256 of the helix-server binary or the APK; evidence/h3-cronet/README.md asserts the server in all runs is the binary built from this PR head without recorded identity: mark UNCONFIRMED or reword (review 3c0f920b new finding 2). (2) HELIX_ACCESS_LOG: config does c.AccessLog, _ = strconv.ParseBool(...), so an invalid value silently turns the evidence instrument off (HELIX_HTTP3 follows the same pattern). (3) ApiRequestPolicy.shouldHandle intercepts /v1/ on ANY https or http host, not only the configured server, and answers CORS for the app origin. (4) HEAD responses do not cancel the Cronet request. (5) Whether Cronet honours the app networkSecurityConfig cleartext rule is UNCONFIRMED. (6) ApiRequestPolicyTest.java (clients/android/android/app/src/test/java/org/helixreader/app/) is committed but not executed anywhere: the host has java but no javac and no junit jar, and Gradle and emulator are forbidden on the host by the test isolation rule; it needs a containerised run. (7) Mutation M2 on tools/isolation/h3_lab.sh survives: case $N in changed to case $n in (the upper-casing of the env denylist) because all denied test cases are upper-case; add a lower-case case or drop the upper-casing. (8) .gitignore /test anchoring no longer ignores a non-root path named test (informational; no junk newly un-ignored by any measured path). (9) PR 26 body parity section stale after the base moved (process-doc; PR merged). Related: HRD-087, HRD-088.

**Affected scope:** server/internal/config, clients/android (ApiRequestPolicy.java and its JVM test), tools/isolation/h3_lab.sh, evidence/h3-cronet/README.md and verdict writer scripts (qa/h3/check_h3.py, run_android_h3.sh).

**Reproduction / current observation:** Set HELIX_ACCESS_LOG to an invalid value and start the server: no error; read the verdict JSON for a server or APK sha256: absent.

**Acceptance:** Verdicts carry the sha256 of the server binary and APK (or are labelled UNCONFIRMED); an invalid HELIX_ACCESS_LOG is reported at startup; shouldHandle scope and HEAD cancellation fixed or closed with a reason; ApiRequestPolicyTest executed in a rootless container with captured output; the lower-case denylist case added and its mutant killed; the cleartext rule checked on a device or kept UNCONFIRMED; validate passes.

**Closure criteria:** Verdicts carry the sha256 of the server binary and APK (or are labelled UNCONFIRMED); an invalid HELIX_ACCESS_LOG is reported at startup; shouldHandle scope and HEAD cancellation fixed or closed with a reason; ApiRequestPolicyTest executed in a rootless container with captured output; the lower-case denylist case added and its mutant killed; the cleartext rule checked on a device or kept UNCONFIRMED; validate passes.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T18:57:45Z / 2026-11-24T18:57:45Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-121] Gate tool gaps found reviewing PR 27: helix-deps ref compared by 7 characters, gitlink_on_remote BLIND instead of FAIL, no pointer-on-main check (OWED-GATE-151)

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** Gate-tool gaps found by the review of PR 27 (pr27-HRD-102-review-70c3afd8.txt item 8, 9, 10, 11), with own mutations in a scratch clone. (A) scripts/gates/cm_helix_deps_ref_matches_gitlink compares only the 7-character ref prefix: a fabricated sha with the same 7-character prefix passes (compare full shas). (B) cm_gitlink_on_remote returns BLIND (rc=2, commit object not present) instead of FAIL when the object exists on no upstream; only a fresh-clone submodule update would fail. (C) An owned CM-SUBMODULE-POINTER-ON-MAIN check is missing (registered as deferral OWED-GATE-151 in the constitution): setting the gitlink AND the helix-deps ref both to a master-only commit (the review used containers 505a4d1) passes both gates (PASS and PUBLISHED); only merge-base --is-ancestor against upstream main (rc=1) catches the 11.4.288(D) off-main pointer; a gitlink-only change is caught by the ref match check as DRIFT. (D) PR 27 body said llm_orchestrator (8ee5795) and vision_engine (1b2b991) gitlinks are master tips one ahead of main; re-measured 2026-11-24 they are main (parents c84c410 and d52fa44), so the body sentence was stale or wrong (UNCONFIRMED which). (E) doc_processor gitlink 12a5fb7 is the master tip, main 86f1a98 is diverged (master 126 commits ahead, main 1 ahead): the violation remains and is HRD-100; llm_orchestrator credentials are HRD-101 (not duplicated here; evidence above applies to them). Related: HRD-100, HRD-101, HRD-102, HRD-103.

**Affected scope:** scripts/gates/cm_helix_deps_ref_matches_gitlink.sh, scripts/gates/cm_gitlink_on_remote.sh, a new pointer-on-main gate, tests and mutants for them.

**Reproduction / current observation:** In a scratch clone set a gitlink and helix-deps ref both to a master-only commit of a submodule and run the two gates: both PASS; set ref to a 7-character-prefix match of a fabricated sha: the ref check PASSES.

**Acceptance:** Full-sha comparison with a RED test and a killed mutant; a missing object on every upstream is FAIL (with BLIND kept only for no checkout); a pointer-on-main gate fails the off-main mutations (gitlink and ref both on master-only) and passes a detached checkout on main; paired mutants killed; validate passes.

**Closure criteria:** Full-sha comparison with a RED test and a killed mutant; a missing object on every upstream is FAIL (with BLIND kept only for no checkout); a pointer-on-main gate fails the off-main mutations (gitlink and ref both on master-only) and passes a detached checkout on main; paired mutants killed; validate passes.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T18:57:45Z / 2026-11-24T18:57:45Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-122] Readium LCP test CA root is not distributed with the reference server: shipped EDRLab test provider cert cannot be chain-verified by Helix

**Status:** Queued

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Description:** Found by HRD-046. test/cert/cert-edrlab-test.pem in readium-lcp-server@32570b6 is issued by 'EDRLab Readium LCP test CA', but no root/CA certificate exists in that repo or in readium/lcp-specs@812dd91 (evidence/lcp-interop-provenance/edrlab_test_cert_chain_gap.txt). Helix embeds no root (fail-closed ErrUntrustedRoot, correct), so interop was run with an operator-generated test root; verification against the real EDRLab/Readium root remains UNCONFIRMED. Needs the operator to obtain the published test/production root from EDRLab (or the Readium License Authority) and re-run qa/lcp-interop with it. Cross-links (11.4.214): HRD-046 (found by), HRD-123.

**Affected scope:** core/pkg/lcp trust-root policy; qa/lcp-interop

**Reproduction / current observation:** evidence/lcp-interop-provenance/edrlab_test_cert_chain_gap.txt

**Acceptance:** Real EDRLab test root obtained from an authoritative source with recorded provenance; qa/lcp-interop run with it unlocks a license signed with the EDRLab test provider cert; evidence sealed.

**Closure criteria:** Real EDRLab test root obtained from an authoritative source with recorded provenance; qa/lcp-interop run with it unlocks a license signed with the EDRLab test provider cert; evidence sealed.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T21:11:29Z / 2026-11-24T21:11:29Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded

## [HRD-123] NeedsLicenseRefresh compares whole-second Status timestamps strictly: an update inside the issuing second is not detected

**Status:** Queued

**Type:** Bug

**Severity:** Medium

**Reopens:** 0

**Description:** Found by HRD-046. The Readium reference Status Server stamps updated.license with whole-second resolution. lcp.NeedsLicenseRefresh uses After; on the first interop run (issue, register and return inside one second) it returned false after a return (status.updated.license == license.issued). Runs 1-3 of the committed test wait 1.1 s so the check is deterministic. Decide whether a tie should be treated as stale after a return/renew (spec: client must download the updated license after return/renew, lsd latest.md l.326/l.383, so the interaction itself, not the timestamp, is the trigger). The original failing output was archived outside the repo by fresh_out (UNCONFIRMED as repo evidence). Reproduced independently by the PR #33 reviewer (mutant M3: removing the 1.1 s wait gives needs_license_refresh_after_return FAIL, lcpserver log shows generate and update inside one second). Spec MUST not met: lcp-specs lsd latest.md l.326 (return) and l.383 (renew) say the client must download the updated License Document. Cross-links (11.4.214): HRD-046 (found by), HRD-122.

**Affected scope:** core/pkg/lcp/lsd.go NeedsLicenseRefresh

**Reproduction / current observation:** Issue license and Return within one second against the reference lsdserver; NeedsLicenseRefresh(status, license) == false.

**Acceptance:** Decision recorded; if changed, unit test plus interop re-run; evidence sealed.

**Closure criteria:** Decision recorded; if changed, unit test plus interop re-run; evidence sealed.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** User work-item request

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-24T21:11:29Z / 2026-11-24T21:11:29Z

**History:**

- 2026-11-24 — Opened by AI: new-work-item; evidence: none recorded
