# Helix Open Workable Items

Generated from `docs/workable_items.db`; do not edit directly.

In progress: 4, In testing: 0, Operator-blocked: 1, Queued: 7, Ready for testing: 0, Reopened: 0

## [HRD-019] Complete revision-pinned research for every deduplicated reader repository

**Status:** In progress

**Type:** Task

**Severity:** High

**Reopens:** 0

**Description:** 5 primary and 35 supplementary repositories now cloned full, non-shallow, fsck clean, detached at the exact pinned revision, with recomputed file ledgers and per-repository reports under research/reports/. Scope corrected from prose: 111 analyses, of which 14 unique URLs remain unpinned by clone-time budget (not unreachable; ls-remote recorded). BookOrbit is AGPL-3.0-only with section 7 additional terms, so its decision is BLOCKED-BY-LICENSE; koodo-reader's pinned revision was rewritten out of upstream history. Manual architecture and license review still pending, which is what this item gates on.

**Affected scope:** All repositories listed in Opensourced_Reader_Apps_And_Repos.md.

**Reproduction / current observation:** Schema-2 summary currently records 106 analyses and five unavailable checkouts; partial clones are not research evidence.

**Acceptance:** Every target has a complete checkout, exact revision, file ledger, manual architecture review, license review, and porting decision.

**Closure criteria:** Every target has a complete checkout, exact revision, file ledger, manual architecture review, license review, and porting decision.

**Evidence:** research/analysis_summary.json; research/reports

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-11-20T18:52:25Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: research/analysis_summary.json; research/reports
- 2026-11-20 — Updated by AI: registry-detail-update; evidence: none recorded

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

**Description:** Build and execute Helix-specific HelixQA challenge banks. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** HelixQA CLI, own-org dependency modules, and Helix end-to-end challenges.

**Reproduction / current observation:** The actual CLI build failed because seven declared dependency checkouts are absent; only VisionEngine cloned successfully.

**Acceptance:** All declared dependencies resolve, Helix-specific banks run live, and machine results preserve every failure and skip.

**Closure criteria:** All declared dependencies resolve, Helix-specific banks run live, and machine results preserve every failure and skip.

**Evidence:** evidence/helixqa-build-test.log; submodules/qa/helix-deps.yaml

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: evidence/helixqa-build-test.log; submodules/qa/helix-deps.yaml

## [HRD-022] Deliver catalog and reader clients for native desktop, mobile, and workstation platforms

**Status:** Queued

**Type:** Feature

**Severity:** Critical

**Reopens:** 0

**Description:** Deliver catalog and reader clients for native desktop, mobile, and workstation platforms. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** Windows, macOS, Linux, Android, iOS, and additional device classes.

**Reproduction / current observation:** Only the disclosed web prototype exists; no native application matrix is delivered or validated.

**Acceptance:** Each promised client has installable artifacts, real reader functionality, accessibility support, and device-specific live evidence.

**Closure criteria:** Each promised client has installable artifacts, real reader functionality, accessibility support, and device-specific live evidence.

**Evidence:** docs/architecture.md

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: docs/architecture.md

## [HRD-023] Provide interactive demonstrations across the platform and device matrix

**Status:** Queued

**Type:** Feature

**Severity:** High

**Reopens:** 0

**Description:** Provide interactive demonstrations across the platform and device matrix. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** Phone, tablet, desktop, workstation, low-end device, and varied screen demonstrations.

**Reproduction / current observation:** Four browser viewport checks exist but are not native-device demonstrations.

**Acceptance:** Interactive demos exercise real client behavior with explicit hardware and runtime evidence.

**Closure criteria:** Interactive demos exercise real client behavior with explicit hardware and runtime evidence.

**Evidence:** evidence/ui/final-run.log

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: evidence/ui/final-run.log

## [HRD-025] Configure and validate Firebase distribution without treating configuration as deployment

**Status:** Queued

**Type:** Feature

**Severity:** High

**Reopens:** 0

**Description:** Configure and validate Firebase distribution without treating configuration as deployment. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** Firebase application distribution and associated release metadata.

**Reproduction / current observation:** No Firebase configuration or CLI qualification has been delivered in this tree.

**Acceptance:** A real project-bound preflight and distribution run produces non-secret machine evidence or an explicit skip.

**Closure criteria:** A real project-bound preflight and distribution run produces non-secret machine evidence or an explicit skip.

**Evidence:** docs/deployment.md

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: docs/deployment.md

## [HRD-026] Validate interoperability on real readers, OPDS clients, and remote providers

**Status:** Queued

**Type:** Feature

**Severity:** Critical

**Reopens:** 0

**Description:** Validate interoperability on real readers, OPDS clients, and remote providers. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** KOReader devices, third-party OPDS clients, and deployment-specific rclone providers.

**Reproduction / current observation:** Protocol-level tests exist, but no physical reader, external OPDS client, or provider-specific deployment run is recorded.

**Acceptance:** External clients complete real acquisition and sync workflows with captured version and environment evidence.

**Closure criteria:** External clients complete real acquisition and sync workflows with captured version and environment evidence.

**Evidence:** docs/testing.md

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: docs/testing.md

## [HRD-027] Remove catalog and API scalability bottlenecks under measured enterprise loads

**Status:** In progress

**Type:** Task

**Severity:** Critical

**Reopens:** 0

**Description:** Phase 1 landed: configurable catalog/AI-memory pools, distinct aimemory migration lock, durable scan records across restart, batched page child loads (measured 3.6x), and an additive keyset cursor through HTTP and OPDS with needle-proofed tests. First 50k-work measurements recorded in docs/scalability-review.md and evidence/campaign-2026-11-20/scalability/. DEFECT-003 (transport-contaminated and nondeterministic scale assertions) found and fixed. Identity locks are now scoped per work, edition and source instead of one global key: 5.85x write throughput in one process (271.94 -> 1592.21 ops/s) and 1.53x to 4.07x across separate OS processes, while the no-lock needle silently loses 8 of 480 concurrently merged editions (evidence/campaign-2026-11-20/lock-scope/ and .../multiprocess/). DEFECT-004: the multi-process needle was blind until a database barrier aligned process starts, because the unsafe window is only the first write to an identity that does not exist yet; the runner now refuses a run where the unlocked variant verifies clean. Concurrent read envelope measured at 1/8/32/64 clients with zero errors: indexed browse and OPDS plateau near 1000 req/s (repo) and 900 (HTTP), limited by database CPU rather than the pool (pool=64 probe 942 vs 992 req/s, zero pool waits), while unanchored %text% search plateaus near 9 req/s at p99 12.0 s with pool starvation stacked on that wall. Release gate still unmet: no ratified SLO, no ingest-concurrent-with-reads test, and content delivery, KOSync/Redis and remote-source connectors remain unmeasured under load. Vector search measured and fixed: Search carried the id tiebreak inside the ORDER BY distance clause, an ordering no pgvector index can satisfy, so every search scanned and sorted every stored embedding while the HNSW index was still maintained on insert (DEFECT-005). Moved outside the subquery: 28.3 ms -> 1.86 ms p50 and 172.8 -> 2748 req/s at 32 clients, 50k rows x 384 dims, and the index path is now chosen without being forced. The harness refuses to be trusted on its own word: catalog-level schema comparison, per-connection session attestation, and a cardinality-independent plan control, with a nine-mutation gate at 9/9 (evidence/campaign-2026-11-20/vector/). New open question, not a defect: hnsw.ef_search (compile-time default 40) is not configurable anywhere in the product, and recall@10 against an exhaustive scan of the same fixture is 0.1025 at that width on 384-dim uniform vectors, 0.5475 at width 400.

**Affected scope:** PostgreSQL writes and search, API pagination, job retention, and connection pools.

**Reproduction / current observation:** Measured and addressed: offset pagination (keyset cursor), per-work child-query fan-out (batched loads), catalog pool caps, and global write serialization, which is now one advisory lock per touched work/edition/source instead of one key for every writer (5.85x on disjoint concurrent merges in a single in-container run, at 89% of the unlocked ceiling; invariants carried by server/internal/catalog/lock_scope_test.go with paired needles and a key-class mutation gate). Vector search: fixed (DEFECT-005, 15.2x p50 at 50k rows, needle-gated harness). Still open: no ratified service-level objective, no recall or search-width objective for AI memory, no multi-process or multi-node topology result, unindexed text search and per-page count(*) deliberately unbudgeted, and same-content merges from distinct sources serialize by design.

**Acceptance:** Representative large-catalog benchmarks and load tests establish limits, durable queue behavior, and bounded query plans.

**Closure criteria:** Representative large-catalog benchmarks and load tests establish limits, durable queue behavior, and bounded query plans.

**Evidence:** docs/scalability-review.md; server/internal/catalog/postgres.go; server/internal/httpapi/jobs.go; server/internal/aimemory/vector_load_test.go; evidence/campaign-2026-11-20/vector

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-11-20T22:17:24Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: docs/scalability-review.md; server/internal/catalog/postgres.go; server/internal/httpapi/jobs.go
- 2026-11-20 — Updated by AI: registry-detail-update; evidence: none recorded
- 2026-11-20 — Updated by AI: catalog write hotspot fixed by scoping identity locks; measured 5.85x with needles and a mutation gate; evidence: evidence/campaign-2026-11-20/lock-scope/summary.md
- 2026-11-20 — Updated by AI: Multi-process lock-scope qualification and concurrent read envelope recorded; DEFECT-004 documented.; evidence: docs/scalability-review.md; server/internal/catalog/postgres.go; server/internal/httpapi/jobs.go; evidence/campaign-2026-11-20/scalability/; evidence/campaign-2026-11-20/load/; evidence/campaign-2026-11-20/lock-scope/; evidence/campaign-2026-11-20/multiprocess/
- 2026-11-20 — Updated by AI: vector-search load harness landed; DEFECT-005 measured and fixed; ef_search recall question opened; evidence: docs/scalability-review.md; server/internal/catalog/postgres.go; server/internal/httpapi/jobs.go; server/internal/aimemory/vector_load_test.go; evidence/campaign-2026-11-20/vector

## [HRD-029] Implement verified bidirectional HelixTrack synchronization

**Status:** In progress

**Type:** Feature

**Severity:** Critical

**Reopens:** 0

**Description:** Implement verified bidirectional HelixTrack synchronization. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** HelixTrack mapping, idempotency, conflict handling, and remote receipts.

**Reproduction / current observation:** Only an offline outbox contract is being established; exact remote API payloads and credentials are unverified.

**Acceptance:** Documented API-compatible push and pull operations pass against a real HelixTrack instance without inventing endpoints.

**Closure criteria:** Documented API-compatible push and pull operations pass against a real HelixTrack instance without inventing endpoints.

**Evidence:** docs/tracking/helixtrack-outbox.json

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: docs/tracking/helixtrack-outbox.json

## [HRD-031] Implement OPDS-PSE, LCP fulfillment, and remaining external reader integrations

**Status:** Queued

**Type:** Feature

**Severity:** High

**Reopens:** 0

**Description:** Implement OPDS-PSE, LCP fulfillment, and remaining external reader integrations. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** Page streaming, DRM fulfillment, and vendor-specific synchronization.

**Reproduction / current observation:** Documentation explicitly identifies these surfaces as unimplemented; detection never implies fulfillment.

**Acceptance:** Each integration has a working implementation, legal DRM handling, interoperability tests, and honest capability reporting.

**Closure criteria:** Each integration has a working implementation, legal DRM handling, interoperability tests, and honest capability reporting.

**Evidence:** docs/api.md; docs/faq.md; docs/formats.md

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: docs/api.md; docs/faq.md; docs/formats.md

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

## [HRD-034] Verify container distribution against a hosted registry and docker-compose CLI runtime parity

**Status:** Operator-blocked

**Type:** Task

**Severity:** Medium

**Reopens:** 0

**Operator-Block-Details:** WHAT: Cannot execute docker-compose CLI parity or an authenticated/signing hosted-registry distribution pass on this host. WHY: Attempted: command -v docker docker-compose cosign crane skopeo oras all report ABSENT and /var/run/docker.sock does not exist, so no Docker engine is reachable; podman/podman-compose are the only container tooling installed. Attempted: emulated Docker semantics with podman-compose, which reproduces build, compose up, health, restart, upgrade and rollback but not docker-compose resource naming or single-service recreation, so it cannot stand in for the CLI claim. Attempted: distribution against a local registry:2 container with --tls-verify=false, which proves push/pull content equality but cannot prove authentication or signature verification. Attempted: installing cosign/skopeo or a Docker engine, which on this PEP 668 externally managed Ubuntu host requires system package installation outside the scope of what may be changed for a qualification run. Attempted: registry credentials, of which none exist in the environment (0 matching variables). UNBLOCK CONDITION: Operator provides either a host with a Docker engine and docker-compose CLI, or explicit authorization to install Docker plus a signing tool, and registry credentials for a hosted registry; alternatively the operator accepts a documented scope limit stating that distribution qualification is podman-and-local-registry only. WHO: Operator

**Description:** HRD-024 closed its acceptance under podman, but two distribution- and tooling-level claims remain unexecuted and are kept separate so neither is implied by that closure. First, the push/pull round trip used a local registry:2 container with TLS verification disabled and no authentication; signing, attestation, and a real registry with credentials, retention and pull-through behavior are unverified. Second, deploy/docker-compose.yml has only ever been executed by podman-compose on this host; docker-compose naming conventions (project-service-N), single-service recreation on change, and secret-handling behavior are asserted from file semantics only. Acceptance requires either a Docker-engine host for a real docker-compose run or an explicit documentation statement that docker-compose is unsupported, plus a push/pull against a registry that requires authentication, with signature verification demonstrated end to end.

**Affected scope:** deploy/docker-compose.yml, server/Dockerfile, evidence/campaign-2026-11-20/containers/, submodules/containers distribution notes.

**Reproduction / current observation:** evidence/campaign-2026-11-20/containers/distribution.log records digest_match=yes and config_id_matches_build=yes against 127.0.0.1:15000 with --tls-verify=false, and results.json records docker_compose_cli_parity as NOT RUN because the docker binary is absent from this host.

**Acceptance:** A push and cache-free pull against an authenticated hosted registry completes with signature verification enabled and the verified digest recorded, OR the registry and README state plainly that distribution qualification covers local registries only; and a docker-compose CLI run of deploy/docker-compose.yml produces health, restart, upgrade and rollback evidence equivalent to the podman-compose pass, OR deploy/README records docker-compose as untested with the reason.

**Closure criteria:** A push and cache-free pull against an authenticated hosted registry completes with signature verification enabled and the verified digest recorded, OR the registry and README state plainly that distribution qualification covers local registries only; and a docker-compose CLI run of deploy/docker-compose.yml produces health, restart, upgrade and rollback evidence equivalent to the podman-compose pass, OR deploy/README records docker-compose as untested with the reason.

**Evidence:** Not yet available; no pass is claimed.

**Forensic anchor:** Second-pass HRD-024 distribution run on 2026-11-20

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-11-20T19:34:10Z / 2026-11-20T19:34:29Z

**History:**

- 2026-11-20 — Opened by AI: new-work-item; evidence: none recorded
- 2026-11-20 — Updated by AI: operator-blocked-after-exhaustion; evidence: none recorded
