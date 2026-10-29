# Helix Closed Workable Items

Generated from `docs/workable_items.db`; do not edit directly.

## [ATM-001] Establish core document, cache, settings, and synchronization primitives

**Status:** Completed (→ Fixed.md)
**Type:** Task
**Severity:** High
**Reopens:** 0

**Description:** Establish core document, cache, settings, and synchronization primitives. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** Core Go module and shared domain model.

**Reproduction / current observation:** Core tests, race detector, vet, coverage, and benchmarks execute against the current tree.

**Acceptance:** Shared primitives remain deterministic, race-free, and covered by repeatable tests.

**Closure criteria:** Shared primitives remain deterministic, race-free, and covered by repeatable tests.

**Evidence:** evidence/core/final-go-test.log; evidence/core/final-go-race.log; evidence/core/final-go-vet.log

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: evidence/core/final-go-test.log; evidence/core/final-go-race.log; evidence/core/final-go-vet.log
- 2026-10-28 — Completed by AI: evidence-backed-baseline-closure; evidence: evidence/core/final-go-test.log; evidence/core/final-go-race.log; evidence/core/final-go-vet.log

## [ATM-002] Implement multi-format detection and bounded document parsing

**Status:** Implemented (→ Fixed.md)
**Type:** Feature
**Severity:** High
**Reopens:** 0

**Description:** Implement multi-format detection and bounded document parsing. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** EPUB, CBZ, FB2, MOBI, TXT, and related registry paths.

**Reproduction / current observation:** Format fixtures and seven fuzz targets exercise malformed and valid inputs without claiming full rendering or conversion.

**Acceptance:** Detection and parsing stay bounded, deterministic, and explicit about unsupported fulfillment stages.

**Closure criteria:** Detection and parsing stay bounded, deterministic, and explicit about unsupported fulfillment stages.

**Evidence:** evidence/core/final-go-test.log; evidence/core/final-go-fuzz-formats.log; docs/formats.md

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: evidence/core/final-go-test.log; evidence/core/final-go-fuzz-formats.log; docs/formats.md
- 2026-10-28 — Implemented by AI: evidence-backed-baseline-closure; evidence: evidence/core/final-go-test.log; evidence/core/final-go-fuzz-formats.log; docs/formats.md

## [ATM-003] Canonicalize ISBN identity across equivalent representations

**Status:** Fixed (→ Fixed.md)
**Type:** Bug
**Severity:** Critical
**Reopens:** 0

**Description:** Canonicalize ISBN identity across equivalent representations. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** Work, edition, file identification and ingest reconciliation.

**Reproduction / current observation:** ISBN-10/13, separators, schemes, and identifier ordering previously risked divergent identity.

**Acceptance:** Equivalent valid ISBNs converge while invalid identifiers fall back to content identity.

**Closure criteria:** Equivalent valid ISBNs converge while invalid identifiers fall back to content identity.

**Evidence:** core/isbn/isbn.go; server/internal/ingest/isbn_identity_test.go; evidence/server/final-go-test.log

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: core/isbn/isbn.go; server/internal/ingest/isbn_identity_test.go; evidence/server/final-go-test.log
- 2026-10-28 — Fixed by AI: evidence-backed-baseline-closure; evidence: core/isbn/isbn.go; server/internal/ingest/isbn_identity_test.go; evidence/server/final-go-test.log

## [ATM-004] Index local and remote libraries in place without reorganizing source bytes

**Status:** Implemented (→ Fixed.md)
**Type:** Feature
**Severity:** Critical
**Reopens:** 0

**Description:** Index local and remote libraries in place without reorganizing source bytes. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** Local, HTTP, WebDAV, OPDS, FTP, SFTP, SMB, S3, and rclone connector surfaces.

**Reproduction / current observation:** Connectors preserve source paths and capabilities; deployment-specific provider validation remains separate open work.

**Acceptance:** Scans index metadata and references without Calibre-style copying or destructive relocation.

**Closure criteria:** Scans index metadata and references without Calibre-style copying or destructive relocation.

**Evidence:** server/internal/sources; docs/sources.md; evidence/server/final-go-test.log

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: server/internal/sources; docs/sources.md; evidence/server/final-go-test.log
- 2026-10-28 — Implemented by AI: evidence-backed-baseline-closure; evidence: server/internal/sources; docs/sources.md; evidence/server/final-go-test.log

## [ATM-005] Make OPDS acquisition identity collision-safe and atomically replaceable

**Status:** Fixed (→ Fixed.md)
**Type:** Bug
**Severity:** Critical
**Reopens:** 0

**Description:** Make OPDS acquisition identity collision-safe and atomically replaceable. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** OPDS feed pagination and acquisition-map generations.

**Reproduction / current observation:** Credential-free public keys, deterministic collision fragments, previous-generation reuse, and stale pruning are exercised.

**Acceptance:** Concurrent feed refreshes cannot expose partially replaced acquisition maps or unstable colliding keys.

**Closure criteria:** Concurrent feed refreshes cannot expose partially replaced acquisition maps or unstable colliding keys.

**Evidence:** server/internal/sources/opds.go; server/internal/sources/opds_test.go; evidence/server/final-go-test.log

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: server/internal/sources/opds.go; server/internal/sources/opds_test.go; evidence/server/final-go-test.log
- 2026-10-28 — Fixed by AI: evidence-backed-baseline-closure; evidence: server/internal/sources/opds.go; server/internal/sources/opds_test.go; evidence/server/final-go-test.log

## [ATM-006] Revalidate content before strong ETag and byte-range responses

**Status:** Fixed (→ Fixed.md)
**Type:** Bug
**Severity:** Critical
**Reopens:** 0

**Description:** Revalidate content before strong ETag and byte-range responses. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** Local and remote content GET and HEAD fulfillment.

**Reproduction / current observation:** Live size and modification-time checks gate scan-time hashes; local GET uses one revalidated descriptor.

**Acceptance:** Changed or deleted bytes never retain a stale strong validator or splice old and new ranges.

**Closure criteria:** Changed or deleted bytes never retain a stale strong validator or splice old and new ranges.

**Evidence:** server/internal/sources/local.go; server/internal/httpapi/content_test.go; evidence/server/final-go-test.log

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: server/internal/sources/local.go; server/internal/httpapi/content_test.go; evidence/server/final-go-test.log
- 2026-10-28 — Fixed by AI: evidence-backed-baseline-closure; evidence: server/internal/sources/local.go; server/internal/httpapi/content_test.go; evidence/server/final-go-test.log

## [ATM-007] Preserve persistent HTTP connections during identity compression fallback

**Status:** Fixed (→ Fixed.md)
**Type:** Bug
**Severity:** Critical
**Reopens:** 0

**Description:** Preserve persistent HTTP connections during identity compression fallback. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** Brotli and gzip middleware on HTTP/1.1 persistent connections.

**Reproduction / current observation:** An unused negotiated compressor was finalized after an identity attachment, appending an empty stream and resetting the next response.

**Acceptance:** Identity fallback disables compressor finalization; raw-socket Brotli and gzip keepalive regressions and mutation reject the defect.

**Closure criteria:** Identity fallback disables compressor finalization; raw-socket Brotli and gzip keepalive regressions and mutation reject the defect.

**Evidence:** evidence/live-server/post-fix-repeat-01/results.json; evidence/live-server/post-fix-daemon-restart/results.json; evidence/go-mutation/results.json

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: evidence/live-server/post-fix-repeat-01/results.json; evidence/live-server/post-fix-daemon-restart/results.json; evidence/go-mutation/results.json
- 2026-10-28 — Fixed by AI: evidence-backed-baseline-closure; evidence: evidence/live-server/post-fix-repeat-01/results.json; evidence/live-server/post-fix-daemon-restart/results.json; evidence/go-mutation/results.json

## [ATM-008] Provide HTTP/1.1, HTTP/2, HTTP/3, TLS, and content negotiation transports

**Status:** Implemented (→ Fixed.md)
**Type:** Feature
**Severity:** High
**Reopens:** 0

**Description:** Provide HTTP/1.1, HTTP/2, HTTP/3, TLS, and content negotiation transports. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** Gin server transport and live protocol verification.

**Reproduction / current observation:** Current-tree binaries are probed over supported protocols with safe fallback and explicit TLS floors.

**Acceptance:** Transport evidence is regenerated from the current tree and distinguishes unavailable protocol support.

**Closure criteria:** Transport evidence is regenerated from the current tree and distinguishes unavailable protocol support.

**Evidence:** evidence/live-transport/final-current-tree/results.json; docs/deployment.md

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: evidence/live-transport/final-current-tree/results.json; docs/deployment.md
- 2026-10-28 — Implemented by AI: evidence-backed-baseline-closure; evidence: evidence/live-transport/final-current-tree/results.json; docs/deployment.md

## [ATM-009] Reconcile durable PostgreSQL catalogs atomically after source scans

**Status:** Fixed (→ Fixed.md)
**Type:** Bug
**Severity:** Critical
**Reopens:** 0

**Description:** Reconcile durable PostgreSQL catalogs atomically after source scans. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** PostgreSQL work, edition, file, and provenance persistence.

**Reproduction / current observation:** Shared validation and identity locking prevent partial reconciliation and cross-work file ownership conflicts.

**Acceptance:** Deleted source files and rehomed identities commit atomically with matching memory-store semantics.

**Closure criteria:** Deleted source files and rehomed identities commit atomically with matching memory-store semantics.

**Evidence:** server/internal/catalog/postgres.go; evidence/server/final-go-live-services.log

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: server/internal/catalog/postgres.go; evidence/server/final-go-live-services.log
- 2026-10-28 — Fixed by AI: evidence-backed-baseline-closure; evidence: server/internal/catalog/postgres.go; evidence/server/final-go-live-services.log

## [ATM-010] Implement monotonic KOReader progress synchronization with Redis

**Status:** Implemented (→ Fixed.md)
**Type:** Feature
**Severity:** High
**Reopens:** 0

**Description:** Implement monotonic KOReader progress synchronization with Redis. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** KOSync account, document progress, and Redis optimistic transaction paths.

**Reproduction / current observation:** WATCH/MULTI/EXEC and monotonic progress behavior are exercised against a real Redis service.

**Acceptance:** Progress cannot regress under stale updates and live-service evidence remains separate from unit doubles.

**Closure criteria:** Progress cannot regress under stale updates and live-service evidence remains separate from unit doubles.

**Evidence:** server/internal/kosync; evidence/server/final-go-live-services.log

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: server/internal/kosync; evidence/server/final-go-live-services.log
- 2026-10-28 — Implemented by AI: evidence-backed-baseline-closure; evidence: server/internal/kosync; evidence/server/final-go-live-services.log

## [ATM-011] Deliver an honest responsive web catalog prototype

**Status:** Implemented (→ Fixed.md)
**Type:** Feature
**Severity:** Medium
**Reopens:** 0

**Description:** Deliver an honest responsive web catalog prototype. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** React catalog-management demonstration across four viewport classes.

**Reproduction / current observation:** The interface discloses sample-only data and the absence of a backend connection.

**Acceptance:** Lint, production build, and four-viewport browser checks pass without presenting sample data as live catalog state.

**Closure criteria:** Lint, production build, and four-viewport browser checks pass without presenting sample data as live catalog state.

**Evidence:** evidence/ui/final-npm-build.log; evidence/ui/final-run.log

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: evidence/ui/final-npm-build.log; evidence/ui/final-run.log
- 2026-10-28 — Implemented by AI: evidence-backed-baseline-closure; evidence: evidence/ui/final-npm-build.log; evidence/ui/final-run.log

## [ATM-012] Establish repeatable deterministic core and server qualification

**Status:** Completed (→ Fixed.md)
**Type:** Task
**Severity:** Critical
**Reopens:** 0

**Description:** Establish repeatable deterministic core and server qualification. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** Unit, integration, race, vet, repeat, and coverage campaigns.

**Reproduction / current observation:** Current-tree logs include genuine failures alongside passing reruns; no result is inferred from command invocation alone.

**Acceptance:** Every published result links machine output and distinguishes pass, fail, skip, and not-run states.

**Closure criteria:** Every published result links machine output and distinguishes pass, fail, skip, and not-run states.

**Evidence:** evidence/core; evidence/server; docs/testing/results

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: evidence/core; evidence/server; docs/testing/results
- 2026-10-28 — Completed by AI: evidence-backed-baseline-closure; evidence: evidence/core; evidence/server; docs/testing/results

## [ATM-013] Fuzz all exposed format and byte-range parsers

**Status:** Completed (→ Fixed.md)
**Type:** Task
**Severity:** High
**Reopens:** 0

**Description:** Fuzz all exposed format and byte-range parsers. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** Formats, CBZ, EPUB, FB2, MOBI, TXT, and single-range parsing.

**Reproduction / current observation:** Seven bounded fuzz campaigns ran against the current tree with retained logs.

**Acceptance:** Fuzz targets terminate without crashes or unbounded parser behavior in the recorded campaigns.

**Closure criteria:** Fuzz targets terminate without crashes or unbounded parser behavior in the recorded campaigns.

**Evidence:** evidence/core/final-go-fuzz-formats.log; evidence/core/final-go-fuzz-cbz.log; evidence/core/final-go-fuzz-epub.log; evidence/core/final-go-fuzz-fb2.log; evidence/core/final-go-fuzz-mobi.log; evidence/core/final-go-fuzz-txt.log; evidence/server/final-go-fuzz-single-range.log

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: evidence/core/final-go-fuzz-formats.log; evidence/core/final-go-fuzz-cbz.log; evidence/core/final-go-fuzz-epub.log; evidence/core/final-go-fuzz-fb2.log; evidence/core/final-go-fuzz-mobi.log; evidence/core/final-go-fuzz-txt.log; evidence/server/final-go-fuzz-single-range.log
- 2026-10-28 — Completed by AI: evidence-backed-baseline-closure; evidence: evidence/core/final-go-fuzz-formats.log; evidence/core/final-go-fuzz-cbz.log; evidence/core/final-go-fuzz-epub.log; evidence/core/final-go-fuzz-fb2.log; evidence/core/final-go-fuzz-mobi.log; evidence/core/final-go-fuzz-txt.log; evidence/server/final-go-fuzz-single-range.log

## [ATM-014] Prove regression tests reject targeted restoration of fixed defects

**Status:** Completed (→ Fixed.md)
**Type:** Task
**Severity:** High
**Reopens:** 0

**Description:** Prove regression tests reject targeted restoration of fixed defects. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** Go mutation campaign across core identity, ingest, sources, and HTTP behavior.

**Reproduction / current observation:** Twenty-one targeted mutants were rejected and original-file hashes remained unchanged.

**Acceptance:** Every mutant fails its focused test and the original tree remains byte-identical after restoration.

**Closure criteria:** Every mutant fails its focused test and the original tree remains byte-identical after restoration.

**Evidence:** evidence/go-mutation/results.json; evidence/go-mutation/final-verifier.log

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: evidence/go-mutation/results.json; evidence/go-mutation/final-verifier.log
- 2026-10-28 — Completed by AI: evidence-backed-baseline-closure; evidence: evidence/go-mutation/results.json; evidence/go-mutation/final-verifier.log

## [ATM-015] Exercise real PostgreSQL, pgvector, and Redis integrations

**Status:** Completed (→ Fixed.md)
**Type:** Task
**Severity:** Critical
**Reopens:** 0

**Description:** Exercise real PostgreSQL, pgvector, and Redis integrations. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** Durable catalog, vector memory, KOSync, and internal progress services.

**Reproduction / current observation:** Live-service tests ran separately from in-memory and miniredis tests.

**Acceptance:** Integration claims require real service output and preserve unavailable credentials or services as non-pass.

**Closure criteria:** Integration claims require real service output and preserve unavailable credentials or services as non-pass.

**Evidence:** evidence/server/final-go-live-services.log

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: evidence/server/final-go-live-services.log
- 2026-10-28 — Completed by AI: evidence-backed-baseline-closure; evidence: evidence/server/final-go-live-services.log

## [ATM-016] Verify backup and restore recovery against the current tree

**Status:** Completed (→ Fixed.md)
**Type:** Task
**Severity:** High
**Reopens:** 0

**Description:** Verify backup and restore recovery against the current tree. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** Catalog and synchronization backup/restore workflow.

**Reproduction / current observation:** The current-tree recovery verifier completed thirteen checks before the latest compression edit and requires final refresh.

**Acceptance:** A clean recovery reproduces catalog and sync state without secret disclosure.

**Closure criteria:** A clean recovery reproduces catalog and sync state without secret disclosure.

**Evidence:** evidence/backup-restore/final-current-tree/results.json

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: evidence/backup-restore/final-current-tree/results.json
- 2026-10-28 — Completed by AI: evidence-backed-baseline-closure; evidence: evidence/backup-restore/final-current-tree/results.json

## [ATM-017] Prevent credentials and sensitive values from entering source and evidence

**Status:** Completed (→ Fixed.md)
**Type:** Task
**Severity:** Critical
**Reopens:** 0

**Description:** Prevent credentials and sensitive values from entering source and evidence. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** Constitution, project instructions, connectors, diagnostics, and release scans.

**Reproduction / current observation:** Secret-handling constraints are mandatory and endpoint credentials are rejected; final archive scanning remains a separate release gate.

**Acceptance:** No credential is committed, printed, logged, or included in evidence or archives.

**Closure criteria:** No credential is committed, printed, logged, or included in evidence or archives.

**Evidence:** AGENTS.md; CLAUDE.md; spec/00_constitution.md; server/internal/sources/http.go

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: AGENTS.md; CLAUDE.md; spec/00_constitution.md; server/internal/sources/http.go
- 2026-10-28 — Completed by AI: evidence-backed-baseline-closure; evidence: AGENTS.md; CLAUDE.md; spec/00_constitution.md; server/internal/sources/http.go

## [ATM-018] Keep deterministic metadata evidence authoritative over AI proposals

**Status:** Implemented (→ Fixed.md)
**Type:** Feature
**Severity:** High
**Reopens:** 0

**Description:** Keep deterministic metadata evidence authoritative over AI proposals. This registry entry records the implemented or remaining work without expanding evidence beyond its actual scope.

**Affected scope:** Identification pipeline, provenance, confidence, and AI memory.

**Reproduction / current observation:** Embedded deterministic identifiers take precedence and provider proposals retain confidence and provenance.

**Acceptance:** Low-confidence AI output is never silently promoted to verified metadata fact.

**Closure criteria:** Low-confidence AI output is never silently promoted to verified metadata fact.

**Evidence:** server/internal/identify; server/internal/ingest; docs/architecture.md

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:35:51Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: server/internal/identify; server/internal/ingest; docs/architecture.md
- 2026-10-28 — Implemented by AI: evidence-backed-baseline-closure; evidence: server/internal/identify; server/internal/ingest; docs/architecture.md

## [ATM-028] Operate the SQLite workable-items single source of truth and synchronized exports

**Status:** Completed (→ Fixed.md)
**Type:** Task
**Severity:** Critical
**Reopens:** 0

**Description:** Operate the authoritative SQLite registry with complete lifecycle commands, evidence-linked history, deterministic document exports, drift detection, disaster recovery, and an explicitly offline HelixTrack outbox.

**Affected scope:** Workable-item database, lifecycle history, exports, validation, and drift gates.

**Reproduction / current observation:** Twelve real-SQLite tests, ten concurrent writers, one thousand stress rows, SIGKILL recovery, corrupt-database recovery, and four paired mutation gates passed; remote HelixTrack delivery remains a separate open item.

**Acceptance:** The database validates, all derived documents regenerate without drift, round trips preserve content, lifecycle and stress gates pass, and no remote push is claimed without a receipt.

**Closure criteria:** The database validates, all derived documents regenerate without drift, round trips preserve content, lifecycle and stress gates pass, and no remote push is claimed without a receipt.

**Evidence:** evidence/workable-items/results.json; evidence/workable-items/final-verifier.log; evidence/workable-items/pdf-qa

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:43:09Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: docs/workable_items.db; scripts/workable_items.py; evidence/workable-items
- 2026-10-28 — Updated by AI: tracker-qualification-completed; evidence: none recorded
- 2026-10-28 — Completed by AI: real-sqlite-and-mutation-qualification-passed; evidence: evidence/workable-items/results.json; evidence/workable-items/final-verifier.log; evidence/workable-items/pdf-qa

## [ATM-030] Publish all current test results and limitations under docs/testing/results

**Status:** Completed (→ Fixed.md)
**Type:** Task
**Severity:** High
**Reopens:** 0

**Description:** Publish a unified machine-readable and Markdown result index covering current passes, preserved failures, partial research, unavailable runtimes, unexecuted external checks, and withheld final archives.

**Affected scope:** Machine-result index for unit, live, UI, integration, deployment, and non-run checks.

**Reproduction / current observation:** The generated index contains 27 scoped outcomes with evidence existence and SHA-256 metadata; historical failures and non-run targets remain visible and release qualification is NOT READY.

**Acceptance:** Every registered suite and unavailable target links its current artifact or explicit non-run statement, and publication never converts missing scope into a pass.

**Closure criteria:** Every registered suite and unavailable target links its current artifact or explicit non-run statement, and publication never converts missing scope into a pass.

**Evidence:** docs/testing/results/summary.json; docs/testing/results/index.md; scripts/publish_test_results.py

**Forensic anchor:** Retrospective evidence-backed baseline imported on 2026-10-29; registry timestamps are import times, not claimed original implementation dates.

**Composes with:** None recorded

**Attribution:** created by AI; assigned to Operator

**Created / modified:** 2026-10-28T17:35:51Z / 2026-10-28T17:50:23Z

**History:**

- 2026-10-28 — Opened by AI: retrospective-evidence-baseline; evidence: docs/testing/results
- 2026-10-28 — Updated by AI: unified-results-publication-completed; evidence: none recorded
- 2026-10-28 — Completed by AI: all-current-result-surfaces-published; evidence: docs/testing/results/summary.json; docs/testing/results/index.md; scripts/publish_test_results.py
