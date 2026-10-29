# Helix Open Workable Items

Generated from `docs/workable_items.db`; do not edit directly.

In progress: 4, In testing: 0, Operator-blocked: 0, Queued: 9, Ready for testing: 0, Reopened: 0

## [ATM-019] Complete revision-pinned research for every deduplicated reader repository

**Status:** In progress
**Type:** Task
**Severity:** High
**Reopens:** 0

**Description:** Complete revision-pinned research for every deduplicated reader repository. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** All repositories listed in Opensourced_Reader_Apps_And_Repos.md.

**Reproduction / current observation:** Schema-2 summary currently records 106 analyses and five unavailable checkouts; partial clones are not research evidence.

**Acceptance:** Every target has a complete checkout, exact revision, file ledger, manual architecture review, license review, and porting decision.

**Closure criteria:** Every target has a complete checkout, exact revision, file ledger, manual architecture review, license review, and porting decision.

**Evidence:** research/analysis_summary.json; research/reports

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: research/analysis_summary.json; research/reports

## [ATM-020] Perform manual architecture, code, license, and innovation review of indexed repositories

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

## [ATM-021] Build and execute Helix-specific HelixQA challenge banks

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

## [ATM-022] Deliver catalog and reader clients for native desktop, mobile, and workstation platforms

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

## [ATM-023] Provide interactive demonstrations across the platform and device matrix

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

## [ATM-024] Qualify containerized build, deployment, restart, and distribution workflows

**Status:** Queued
**Type:** Feature
**Severity:** Critical
**Reopens:** 0

**Description:** Qualify containerized build, deployment, restart, and distribution workflows. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** Containers submodule, Docker or Podman, compose, and distribution providers.

**Reproduction / current observation:** Dockerfiles and compose configuration exist, but container tooling is unavailable and no container run has occurred.

**Acceptance:** Cold start, health, restart, upgrade, rollback, and distribution tests pass in actual container runtimes.

**Closure criteria:** Cold start, health, restart, upgrade, rollback, and distribution tests pass in actual container runtimes.

**Evidence:** deploy/docker-compose.yml; server/Dockerfile; submodules/containers

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: deploy/docker-compose.yml; server/Dockerfile; submodules/containers

## [ATM-025] Configure and validate Firebase distribution without treating configuration as deployment

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

## [ATM-026] Validate interoperability on real readers, OPDS clients, and remote providers

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

## [ATM-027] Remove catalog and API scalability bottlenecks under measured enterprise loads

**Status:** In progress
**Type:** Task
**Severity:** Critical
**Reopens:** 0

**Description:** Remove catalog and API scalability bottlenecks under measured enterprise loads. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** PostgreSQL writes and search, API pagination, job retention, and connection pools.

**Reproduction / current observation:** Global catalog write serialization, offset pagination, per-work child-query fan-out, and in-memory job history remain measured design risks.

**Acceptance:** Representative large-catalog benchmarks and load tests establish limits, durable queue behavior, and bounded query plans.

**Closure criteria:** Representative large-catalog benchmarks and load tests establish limits, durable queue behavior, and bounded query plans.

**Evidence:** docs/scalability-review.md; server/internal/catalog/postgres.go; server/internal/httpapi/jobs.go

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: docs/scalability-review.md; server/internal/catalog/postgres.go; server/internal/httpapi/jobs.go

## [ATM-029] Implement verified bidirectional HelixTrack synchronization

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

## [ATM-031] Implement OPDS-PSE, LCP fulfillment, and remaining external reader integrations

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

## [ATM-032] Complete rendering, conversion, and content-AI fulfillment for every required format

**Status:** Queued
**Type:** Feature
**Severity:** Critical
**Reopens:** 0

**Description:** Complete rendering, conversion, and content-AI fulfillment for every required format. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** All formats in Additional_Formats_We_MUST_Support.jpg and their fulfillment stages.

**Reproduction / current observation:** Current detection and parsing coverage must not be confused with complete rendering, conversion, AI, or DRM support.

**Acceptance:** The per-format matrix has live tests for every claimed stage and explicit non-pass entries for the rest.

**Closure criteria:** The per-format matrix has live tests for every claimed stage and explicit non-pass entries for the rest.

**Evidence:** docs/formats.md; Additional_Formats_We_MUST_Support.jpg

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: docs/formats.md; Additional_Formats_We_MUST_Support.jpg

## [ATM-033] Run final clean qualification and produce fresh ZIP and tar.gz archives

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
