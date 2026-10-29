# Helix Reader — Recovered Project and Conversation History

**Coverage:** earliest recoverable project record through 2026-10-29  
**Current-codebase root used for all code notes:** [`helix/`](helix/)  
**Document generated:** 2026-10-29  
**History status:** **PARTIAL VERBATIM RECOVERY / DETAILED SESSION RECONSTRUCTION**

> **Important completeness statement.** The accessible workspace does not contain verbatim transcripts of every earlier chat session. This document therefore does **not** pretend to be a complete word-for-word export. It preserves every user prompt available verbatim in the inherited conversation record, identifies summarized rather than verbatim material, reconstructs the technical chronology from the inherited session record and current evidence, and links notes exclusively to the latest accessible codebase. Missing original wording, exact session boundaries, and unavailable assistant messages are marked as unrecovered rather than invented.

## 1. How to read this document

Three kinds of record are kept visibly separate:

- **Verbatim prompt** — wording preserved from the accessible conversation record. Spelling, capitalization, and emphasis are retained.
- **Recovered response record** — a detailed summary of assistant work or conclusions preserved by the continuation record. It is not represented as the original word-for-word assistant reply.
- **Current-codebase note** — a cross-reference checked against the present `helix/` tree. These notes describe the code as it exists now, not necessarily as it existed on the historical date being discussed.

The original number of chat sessions and the exact timestamp of every turn are unavailable. Consequently, the “session” headings below are **recoverable work segments**, not a claim that each heading corresponds exactly to one user-interface session.

## 2. Executive chronology

| Date or position | Recoverable segment | Principal user direction | Principal recorded outcome |
|---|---|---|---|
| Undated, before the later hardening rounds | Initial and continuing project-finalization request | Clone and investigate all listed reader repositories; complete formats, backend, clients, documentation, tests, and final packages without bluff. | Multi-round implementation and research program established. Exact original wording and earliest assistant replies are not fully recoverable. |
| 2026-10-14 | M1 foundation | Establish the project and its constitution-driven implementation. | Core Go primitives, source/format registries, specifications, submodules, early tests, and historical packages recorded by the current changelog. |
| Undated, after M1 | “Nano details” investigation round | Find anything missed, exhaustively. | Exact assistant reply unrecovered; later work shows continued parser, identity, source, protocol, and documentation audits. |
| Undated, subsequent round | Deep gaps/danger-zones round | File-by-file and component-by-component audit; machine-only deterministic evidence; live fixes; deep research; production-grade scope. | Large hardening campaign across parsers, catalog identity, remote sources, OPDS, Redis/PostgreSQL, HTTP delivery, UI evidence, and documentation. |
| 2026-10-27 to 2026-10-28 | Backend, UI, interoperability, and evidence expansion | Continue exhaustive development and verification. | Real service tests, responsive prototype checks, source hardening, format parsing, synchronization, OPDS, and transport work preserved in current evidence. |
| 2026-10-28 | Expanded “one last round” request | Add scalability, all-platform clients/demos, deployment and Firebase, full HelixQA, current research, SQLite work tracking and exports, HelixTrack, independent reviews, and final archives only after genuine completion. | Scalability review, unified results, real SQLite tracker and exports, transport/compression fix, live reruns, and HelixQA dependency recovery followed; major scope remained open. |
| 2026-10-28 to 2026-10-29 | Continuation after context exhaustion | Resume the active HelixQA build from exact-version cached dependencies. | Moby submodule handling and numerous pinned dependencies were resolved; the runner had not built and no HelixQA challenge pass was claimed at the history-request checkpoint. |
| 2026-10-29 | Repository-index completion report | Complete remaining repository clones and revision-pinned static analyses. | Automated index reached 111/111 analyses and 111/111 file ledgers; manual architecture/code/license review remained pending. |
| 2026-10-29 | This history-export request | Produce one large chronological `history.md` containing all sessions, prompts, responses, and current-code notes. | This evidence-labelled recovered history was created; unavailable verbatim dialogue is explicitly identified. |

## 3. Initial project-finalization request — inherited main direction

**Date:** not recovered; preceding the later exhaustive rounds  
**Record type:** summarized user direction, not verbatim quotation

The inherited main request directed the assistant to continue “Project Final Packaging” and to treat packaging as the final step rather than the starting point. The work was to include:

1. Clone every deduplicated repository in `Opensourced_Reader_Apps_And_Repos.md`.
2. Investigate each codebase and compare architecture, UI/UX, format handling, optimizations, innovations, strengths, and weaknesses with Helix.
3. Port meaningful improvements with attribution, exact revisions, license compliance, implementation links, and tests.
4. Support every format shown in `Additional_Formats_We_MUST_Support.jpg`, while distinguishing detection, metadata, streaming, rendering, conversion, content AI, and DRM fulfillment.
5. Complete the Go/Gin backend, including HTTP/1.1, HTTP/2, HTTP/3/QUIC, Brotli, safe fallbacks, OPDS 1.2/2.0, OpenSearch, acquisition links, and third-party interoperability.
6. Index content in place across local storage, SMB/NFS mounts, HTTP(S), WebDAV, OPDS, FTP/FTPS, SFTP, SMB, S3, and cloud/rclone sources, without Calibre-style copy-and-reorganize behavior.
7. Preserve work → edition → file identity, edition-specific ISBNs, multiple files per edition, metadata provenance, and review states.
8. Keep deterministic evidence authoritative over AI metadata proposals.
9. Investigate stubs, mocks, unfinished code, inconsistencies, missing documentation, weak spots, and danger zones file by file and component by component.
10. Test every supported category deterministically and live, preserving machine-only evidence.
11. Produce fresh ZIP and tar.gz archives only after incorporation, clean qualification, secret scanning, and validation were complete.
12. Avoid “AI slop” and every form of bluff.

### Recovered assistant response record

The exact first assistant reply is unavailable. The inherited record shows that the request became an continuing implementation and qualification program rather than a one-pass packaging operation. The resulting current tree separates executable behavior from targets, retains failure artifacts, and withholds new final archives while major requirements remain open.

### Current-codebase notes

- The present capability boundary is stated in [`helix/README.md`](helix/README.md): Helix is an actively developed read-only indexing and OPDS server, not a production-ready complete reader suite.
- The original uploaded research inputs remain outside the code tree; their current research output is indexed under [`helix/research/`](helix/research/).
- Current source connectors are under [`helix/server/internal/sources/`](helix/server/internal/sources/), while shared source primitives are under [`helix/core/sources/`](helix/core/sources/) and [`helix/core/pkg/source/`](helix/core/pkg/source/).
- Current format code is split between [`helix/core/formats/`](helix/core/formats/) and the bounded parsers in [`helix/core/pkg/parse/`](helix/core/pkg/parse/).
- Final packaging remains an open tracked item, not an accomplished result; see ATM-033 in [`helix/docs/Issues.md`](helix/docs/Issues.md).

## 4. M1 foundation — 2026-10-14

**Record type:** current changelog and artifact-based reconstruction; original dialogue unrecovered

The earliest dated project record in the current tree is `0.1.0-M1 (2026-10-14)`. It records adoption of the Helix Universal Constitution, cloning of development/QA/container/CodeGraph-related submodules, a Go core M1, and completion of the SpecKit flow through review for EPIC E1 with E2–E7 planned.

The M1 core included:

- HDM primitives;
- format and conversion registries;
- validated ISBN extraction;
- hybrid logical clock behavior intended to tolerate clock skew;
- a protected LRU cache;
- a source registry covering 18 protocol surfaces;
- 20 tests, with two defects found and fixed through live runs;
- two final green runs recorded in the early evidence directory.

Historical package files dated 2026-10-14 remain in the workspace, but they are **not** treated as this round’s final deliverables and are not used as the source for the code notes in this document.

### Current-codebase notes

- The historical entry is preserved in [`helix/CHANGELOG.md`](helix/CHANGELOG.md).
- Current HDM code is in [`helix/core/hdm/hdm.go`](helix/core/hdm/hdm.go) and [`helix/core/pkg/hdm/hdm.go`](helix/core/pkg/hdm/hdm.go).
- Current clock code is in [`helix/core/hlc/hlc.go`](helix/core/hlc/hlc.go).
- Current ISBN canonicalization is in [`helix/core/isbn/isbn.go`](helix/core/isbn/isbn.go).
- Current cache behavior is in [`helix/core/cache/cache.go`](helix/core/cache/cache.go).
- The early test log remains at [`helix/evidence/go_test_run3.log`](helix/evidence/go_test_run3.log); it is historical evidence, not a substitute for the latest qualification matrix.

## 5. “Nano details” investigation round

**Date:** not recovered; after the initial development request  
**Record type:** verbatim user prompt; assistant reply unrecovered

### Verbatim prompt

> Continue with next round - investigate up to the nano details exhaustivelly if we may have missed anything?

### Recovered assistant response record

The original assistant response text is not available. The continuation record demonstrates that subsequent work repeatedly returned to identity, parser bounds, stale content, persistent connections, source reconciliation, and evidence accuracy. Those later results are documented below rather than backfilled into an invented reply to this prompt.

## 6. Deep gaps, weak spots, danger zones, and machine-evidence round

**Date:** not recovered; before the expanded final-round request  
**Record type:** verbatim user prompt followed by reconstructed technical response record

### Verbatim prompt

> Lets dive with our next round! Of top of everything do exhaustive analisys and investiogation for any gaps, shortcomings, inconsistencies, weak spots and danger zones, unfinished and unimplemented parts, any stubs or mocks left, parts or pieces of solution with no documentation, guides or tutorials, diagrams, wschemes, and graphs, FAQs, full tests coverage of all 7+ supported test types and all of them producing MACHINE ONLY rock-solid evidence which is fully deterministically validated and verified with zero bluff of any kind and form and no AI slop! There MUST EXIST NO false or faulty results, no fake and no issues! Do this deep investigation, file by file, class by class, component by component, functionality from specs by functionality from specs, every nano detail, no avoidng, skipping or ignoring - ALL OF THIS MUST BE fully covered and tackled! Root causes deeply checked, fixed, polished and tested LIVE deterministically! Everything MUST BE in enterprise-grade production ready state - all backend services and components, everything client side - all apps and services and tools! Do additional deep web research rounds for finding more proper solutions for most demanding challenges and issues! Seek for all innovative ideas, fame changer opsncourced codebases and make sure we do incorporate them or port them into our solution and our components! Once all is done, fully LIVE tested, confirmed, absolutely consistent, validated and verified provide us final downloadable zip and tar.gz like so far containing all of our work!

The inherited record also states that subsequent messages repeated the demand for further exhaustive rounds and new downloadable archives after completion. The exact wording and count of those repeated turns were not preserved.

### Recovered assistant response record — format and parser hardening

The deep-round work replaced placeholder MOBI compression-type-2 behavior with bounded PalmDOC LZ77 decoding. EPUB, FB2, TXT, and CBZ parser paths received bounds and content-extraction hardening. FB2 ISBN extraction was normalized and validated, and ISBN handling was unified rather than allowing equivalent representations to create separate identities.

The work deliberately separated:

- format registration;
- signature or extension detection;
- metadata extraction;
- bounded content parsing;
- byte streaming;
- rendering;
- conversion;
- content-AI processing;
- DRM fulfillment.

This separation prevented a parser or registry entry from being presented as a complete reader or converter.

**Current-codebase notes:**

- PalmDOC implementation: [`helix/core/pkg/parse/mobi/palmdoc.go`](helix/core/pkg/parse/mobi/palmdoc.go)
- MOBI parser: [`helix/core/pkg/parse/mobi/mobi.go`](helix/core/pkg/parse/mobi/mobi.go)
- Shared parser limits: [`helix/core/pkg/parse/parseutil/limits.go`](helix/core/pkg/parse/parseutil/limits.go)
- EPUB, FB2, TXT, and CBZ implementations: [`helix/core/pkg/parse/`](helix/core/pkg/parse/)
- Format fuzz targets: [`helix/core/formats/fuzz_test.go`](helix/core/formats/fuzz_test.go) and parser-specific test files
- Current qualification logs: [`helix/evidence/core/`](helix/evidence/core/)

### Recovered assistant response record — catalog and identity

The catalog model retained work, edition, and file identities rather than flattening every input into a single book record. Equivalent ISBN-10/ISBN-13 forms, separators, URI schemes, hyphenation, and identifier ordering were canonicalized. Invalid or absent identifiers could fall back to deterministic content identity.

Source/path identity was made stable across metadata improvements. Reconciliation gained current-byte replacement, atomic cross-work file re-homing, and cleanup of empty works after changed content. Memory-repository edition/file lookups were deep-cloned, and memory validation was aligned with PostgreSQL behavior.

PostgreSQL migrations became transactional and versioned. Metadata evidence merges became idempotent. pgvector content, hash, and vector updates were made atomic. Reconciliation was made fail-safe so partial scans could not delete still-valid catalog files.

**Current-codebase notes:**

- ISBN implementation and tests: [`helix/core/isbn/`](helix/core/isbn/) and [`helix/server/internal/ingest/isbn_identity_test.go`](helix/server/internal/ingest/isbn_identity_test.go)
- Ingest implementation: [`helix/server/internal/ingest/`](helix/server/internal/ingest/)
- Catalog implementations: [`helix/server/internal/catalog/`](helix/server/internal/catalog/)
- Live PostgreSQL evidence: [`helix/evidence/server/final-go-live-services.log`](helix/evidence/server/final-go-live-services.log)

### Recovered assistant response record — in-place and remote sources

The source work retained read-only indexing: source books were not copied, moved, or reorganized. Scan execution was bounded to two active workers and 64 unfinished jobs. Retention limits, panic containment, queued/running cancellation, and shutdown draining were added. Scan admission and source deletion were serialized, and deletion of an active source produced a conflict rather than racing a scan.

Atomic stale-file reconciliation was limited to complete recursive full-source scans. Prefix scans, non-recursive feeds, cancellations, and ingest errors did not authorize pruning. Stable validated source IDs allowed daemon restart reattachment without duplicating files. Local sources stopped advertising an unimplemented watch service.

Remote-source hardening included:

- dial-time OPDS acquisition SSRF checks for direct and redirected URLs;
- an explicit trusted-private-network opt-in;
- permanent rejection of link-local and restricted ranges;
- removal of configured credentials on cross-origin redirects or acquisitions;
- replacement of whole-body transfer deadlines with connection/header bounds and request cancellation;
- pre-request failure for unsafe plain HTTP/WebDAV cross-origin redirects;
- provider timeout and public-error hardening;
- credential redaction in scan reports, report locations, and job errors, including malformed and presigned URLs;
- credential-free public OPDS acquisition paths, with signed URLs held only in process-local connector memory.

Signed remote sources consequently require registration and rescanning after restart rather than leaking signed URLs into public catalog data.

**Current-codebase notes:**

- Connector directory: [`helix/server/internal/sources/`](helix/server/internal/sources/)
- Local single-descriptor validation: [`helix/server/internal/sources/local.go`](helix/server/internal/sources/local.go)
- OPDS acquisition identity and redirect behavior: [`helix/server/internal/sources/opds.go`](helix/server/internal/sources/opds.go)
- Scan jobs: [`helix/server/internal/httpapi/jobs.go`](helix/server/internal/httpapi/jobs.go)
- Current source capability summary: [`helix/README.md`](helix/README.md) and [`helix/docs/api.md`](helix/docs/api.md). The tracker’s inherited reference to `docs/sources.md` does not resolve in the latest tree and is recorded as documentation drift, not silently repaired by this history.

### Recovered assistant response record — OPDS, content delivery, and transport

OPDS catalog metadata, covers, and content were protected with OPDS Basic authentication when OPDS credentials were configured. Distinct query-addressed acquisition URLs were prevented from collapsing into one public key. Deterministic credential-free synthetic fragments, previous-generation key reuse, and atomic map replacement/pruning addressed collisions, reorder churn, and partial updates.

Content delivery corrections included:

- real-network Brotli/gzip `Content-Length` framing;
- local-size revalidation before downloads;
- HEAD support;
- safe If-Range, multi-range, and unknown-size behavior;
- identity encoding for attachments;
- strong scan-time SHA-256 ETags gated by current size and modification time;
- open-then-fstat local GET streaming through one validated descriptor;
- 404 responses for deleted local paths;
- context propagation into local range calls.

Transport support and qualification covered HTTP/1.1 persistent connections, HTTP/2 over TLS, HTTP/3/QUIC, Alt-Svc, a TLS 1.2 TCP floor, a TLS 1.3 QUIC floor, and Brotli/gzip/identity negotiation.

**Current-codebase notes:**

- HTTP compression middleware: [`helix/server/internal/httpapi/compression.go`](helix/server/internal/httpapi/compression.go)
- HTTP router and regression tests: [`helix/server/internal/httpapi/router_test.go`](helix/server/internal/httpapi/router_test.go)
- Content tests: [`helix/server/internal/httpapi/content_test.go`](helix/server/internal/httpapi/content_test.go)
- Transport verifier: [`helix/scripts/verify_live_transport.py`](helix/scripts/verify_live_transport.py)
- Transport evidence: [`helix/evidence/live-transport/`](helix/evidence/live-transport/)

### Recovered assistant response record — synchronization and real services

An original KOReader-compatible account/progress protocol implementation was added with Redis persistence, bcrypt verifiers, closed registration by default, bounded requests, server timestamps, and per-user isolation. Redis writes became monotonic through retried WATCH/MULTI/EXEC optimistic transactions so stale updates could not regress progress.

Real PostgreSQL, pgvector, and Redis tests were kept distinct from in-memory repositories and miniredis-based unit tests. This was an important anti-bluff boundary: protocol doubles can establish local behavior, but they do not prove interoperability with a physical KOReader device or an external OPDS reader.

**Current-codebase notes:**

- KOSync implementation: [`helix/server/internal/kosync/`](helix/server/internal/kosync/)
- AI/vector memory: [`helix/server/internal/aimemory/`](helix/server/internal/aimemory/) if present in the current module layout; the authoritative live evidence is [`helix/evidence/server/final-go-live-services.log`](helix/evidence/server/final-go-live-services.log)
- External-device non-run status: [`helix/docs/testing/results/index.md`](helix/docs/testing/results/index.md)

### Recovered assistant response record — web prototype and documentation

A responsive React product prototype received deterministic phone, tablet, desktop, and wide-screen checks. The recorded campaign passed 64/64 browser checks and included a deliberate UI mutation gate. The interface explicitly disclosed sample-only data and the absence of a live backend connection; the checks were not represented as native-client or real-device evidence.

Aspirational documentation was replaced with current-capability guides. Architecture diagrams, API documentation, testing matrices, font notices, and upstream porting records were added or corrected.

**Current-codebase notes:**

- Prototype workspace: [`app/`](app/)
- UI evidence: [`helix/evidence/ui/`](helix/evidence/ui/)
- UI mutation evidence: [`helix/evidence/ui-mutation/`](helix/evidence/ui-mutation/)
- Current architecture: [`helix/docs/architecture.md`](helix/docs/architecture.md)
- Current API reference: [`helix/docs/api.md`](helix/docs/api.md)
- User and developer guides: [`helix/docs/user_guide.md`](helix/docs/user_guide.md), [`helix/docs/developer_guide.md`](helix/docs/developer_guide.md)
- FAQ: [`helix/docs/faq.md`](helix/docs/faq.md)

### Recovered assistant response record — security policy propagation

The verbatim mandatory constraint was propagated to the main project policy and every submodule control file:

> Credentials or any secret and sensitive data MUST NOT leak! This MUST BE added as the mandatory constraint into all Submodules and main project's Constitution, CLAUDE.MD and AGENTS.MD.

The standing interpretation prohibited secrets in source, commits, logs, evidence, screenshots, and archives; credentials in endpoint URLs; forwarding configured secret headers cross-origin; and shell interpolation for rclone. Indexing was required to remain non-destructive. Failed clones could not be marked researched, unexecuted UI checks could not be called passed, and missing credentials or dependencies had to produce explicit non-pass results.

**Current-codebase notes:**

- Main policy files: [`helix/AGENTS.md`](helix/AGENTS.md), [`helix/CLAUDE.md`](helix/CLAUDE.md)
- Changelog record of propagation: [`helix/CHANGELOG.md`](helix/CHANGELOG.md)
- Security boundary summary: [`helix/README.md`](helix/README.md)

## 7. Expanded “one last round” request

**Date:** inherited record places this before the 2026-10-28 continuation work  
**Record type:** verbatim excerpts plus preserved requirement decomposition

### Verbatim opening

> Now one last round on top of all work done so far - exhaustive investigation and research with systematic LIVE fully deterministic testing and debugging!

### Preserved full requirement set

The expanded request required all of the following in addition to the earlier scope:

- a final exhaustive investigation and research round with systematic live deterministic testing and debugging;
- enterprise-grade scalability review of every backend service and API;
- catalog-management and reader clients for all operating systems, platforms, device classes, screen sizes, low-end devices, and workstations;
- interactive demos for that platform/device matrix;
- containerized bash build, deployment, and distribution paths using the containers submodule, Firebase, and other viable major providers;
- full incorporation of HelixQA;
- all testing results documented under `docs/testing/results`;
- deep current web research on reader, backend, accessibility, synchronization, and deployment innovations;
- full compliance with the main constitution and workable-items philosophy;
- a complete current SQLite work-item database, mandatory Markdown/HTML/PDF exports, and a working HelixTrack synchronization mechanism;
- independent specialized code and other review phases;
- no false results, no bluff, and no AI slop;
- final downloadable ZIP and tar.gz archives only when everything was actually complete.

The request reiterated that no issue could be missed, avoided, ignored, or skipped and that no false results were acceptable.

### Recovered assistant response record — overall disposition

The response did not claim that this expanded scope had been completed. Work proceeded in separable qualification and implementation tracks, with `PASS`, `FAIL`, `PARTIAL`, `SKIP`, `NOT RUN`, and `NOT DELIVERED` kept distinct. No production-ready, native-client-complete, external-interoperability-complete, deployment-complete, research-complete, HelixQA-complete, or final-archive claim was made.

## 8. Persistent-connection compression defect and live recovery

**Date:** 2026-10-28  
**Record type:** detailed inherited assistant/work record with current-code verification

### Root cause

A persistent HTTP/1.1 connection could be corrupted after an identity attachment response. Brotli or gzip had initially been negotiated, but the attachment path selected identity fallback. Finalizing the now-unused compressor appended an empty compressed stream after the identity body. The next response on that persistent connection could then be reset or misframed.

### Fix

Identity fallback was changed to write directly through the underlying response writer and disable compressor finalization:

```go
w.encoding = "identity"
w.writer = w.ResponseWriter
w.closer = nil
w.ResponseWriter.WriteHeader(code)
```

Finalization now closes only an active compressor:

```go
if cw.closer != nil {
    if err := cw.closer.Close(); err != nil {
        _ = c.Error(err)
    }
}
```

A real-server/raw-TCP regression verified an identity attachment followed by compressed JSON on the same connection for both Brotli and gzip. The mutation campaign added `identity-response-closes-unused-compressor`; the recorded result was 21/21 targeted mutants rejected with original files unchanged.

### Live qualification after the fix

- Three ordinary live-server runs passed 37/37.
- The daemon/PostgreSQL/Redis restart run passed 38/38.
- Backup/restore was rerun after the compression edit and passed 13/13.
- The earlier daemon-restart connection-reset failure remained preserved rather than overwritten.

### Separate readiness failure retained

A subsequent transport attempt failed at verified TLS readiness with `PASS=0`, `FAIL=1`. The verifier initially discarded the request exception, leaving inadequate diagnostics. Direct reproduction against the current binary succeeded over HTTP/2 and HTTPS, with Alt-Svc advertising HTTP/3. The failure was not declared explained.

Readiness handling was changed to retain the last error and allow up to 300 attempts:

```python
response = None
last_error = "no request completed"
for _ in range(300):
    if process.poll() is not None:
        raise RuntimeError("TLS server exited before readiness")
    try:
        response = session.get(
            BASE + "/healthz",
            verify=str(WORK / "cert.pem"),
            timeout=1,
        )
        if response.status_code == 200:
            break
        last_error = f"HTTP {response.status_code}"
    except requests.RequestException as exc:
        last_error = f"{type(exc).__name__}: {exc}"
        time.sleep(0.1)
```

Three post-instrumentation transport runs passed 5/5. The original failure artifact remains a `FAIL` in the results index.

### Current-codebase notes

- Fix: [`helix/server/internal/httpapi/compression.go`](helix/server/internal/httpapi/compression.go)
- Raw-socket regression: [`helix/server/internal/httpapi/router_test.go`](helix/server/internal/httpapi/router_test.go)
- Mutation verifier: [`helix/scripts/verify_go_mutation.py`](helix/scripts/verify_go_mutation.py)
- Readiness instrumentation: [`helix/scripts/verify_live_transport.py`](helix/scripts/verify_live_transport.py)
- Preserved transport failure: [`helix/evidence/live-transport/post-compression-failure-01/results.json`](helix/evidence/live-transport/post-compression-failure-01/results.json)
- Passing transport repeats: [`helix/evidence/live-transport/post-compression-repeat-02/results.json`](helix/evidence/live-transport/post-compression-repeat-02/results.json), [`helix/evidence/live-transport/post-compression-repeat-03/results.json`](helix/evidence/live-transport/post-compression-repeat-03/results.json), [`helix/evidence/live-transport/post-compression-repeat-04/results.json`](helix/evidence/live-transport/post-compression-repeat-04/results.json)
- Backup/restore verifier and result: [`helix/scripts/verify_backup_restore.py`](helix/scripts/verify_backup_restore.py), [`helix/evidence/backup-restore/post-compression/results.json`](helix/evidence/backup-restore/post-compression/results.json)

## 9. SQLite workable-items registry, exports, and HelixTrack boundary

**Date:** 2026-10-28  
**Record type:** detailed inherited assistant/work record

### Authoritative registry

A real SQLite registry became the authoritative source for work items. The default database is `docs/workable_items.db`; the inherited baseline contained 33 evidence-backed items. The baseline was explicitly retrospective: import timestamps were not presented as original implementation dates.

Mandatory tables included:

- `items`;
- `item_history`;
- `obsolete_details`;
- `operator_block_details`;
- `reopen_details`;
- `firebase_metadata`;
- `meta`;
- `sync_outbox`.

Lifecycle vocabularies were closed and type-aware:

```python
ACTIVE = {
    "Queued", "In progress", "Ready for testing",
    "In testing", "Reopened", "Operator-blocked",
}
CLOSED = {
    "Bug": "Fixed (→ Fixed.md)",
    "Feature": "Implemented (→ Fixed.md)",
    "Task": "Completed (→ Fixed.md)",
}
TERMINAL = set(CLOSED.values()) | {"Obsolete (→ Fixed.md)"}
TYPES = {"Bug", "Feature", "Task"}
```

Connections enabled foreign keys, WAL, a busy timeout, and full synchronization:

```python
db.execute("PRAGMA foreign_keys=ON")
db.execute("PRAGMA busy_timeout=30000")
db.execute("PRAGMA journal_mode=WAL")
db.execute("PRAGMA synchronous=FULL")
```

Validation covered ATM identifiers, type/status/severity vocabularies, location, type-aware closure wording, comprehensive descriptions, closure evidence, reopen attribution, operator-block exhaustion details, reopen counts, gapless IDs, and foreign keys.

Lifecycle operations included add, update, close, reopen, and operator block. Closure status was derived from item type. `reopen_details.atm_id` was corrected from `INTEGER` to `TEXT` before creation of the real registry.

### Exports and deterministic recovery

Markdown, HTML, and ReportLab-generated PDF surfaces were produced for open items, closed items, and their summaries:

- `Issues.md`, `Issues.html`, `Issues.pdf`;
- `Issues_Summary.md`, `Issues_Summary.html`, `Issues_Summary.pdf`;
- `Fixed.md`, `Fixed.html`, `Fixed.pdf`;
- `Fixed_Summary.md`, `Fixed_Summary.html`, `Fixed_Summary.pdf`.

Markdown import restored item fields, attribution, composition references, history, reopen details, and operator-block details. An initial round-trip test failed because creator/assignee became `UNKNOWN`; the initial failure was preserved. The parser was corrected, after which the main registry’s DB → Markdown/HTML → DB → Markdown/HTML regeneration produced byte-identical exports.

Twelve real-SQLite tests covered schema and valid rows, invalid vocabularies, closure wording, export drift, escaping, operator blocks, monotonic allocation, outbox anti-bluff behavior, concurrent writers, lifecycle transitions, operator-block recovery, and byte-identical Markdown recovery.

The dedicated verifier recorded 10 passing groups and no failures, including:

- authoritative validation and export diff;
- 1,000-row stress insertion;
- SIGKILL recovery;
- corrupt-database detection/recovery;
- four mutations proving rejection of disabled closure validation, disabled drift detection, fake remote `pushed` status, and dropped attribution.

Initial PDF visual inspection covered all four PDF surfaces. Because later item closures regenerated the PDFs, final visual QA still required refresh; initial inspection was not treated as final qualification.

### HelixTrack anti-bluff boundary

The outbox always recorded:

```json
"remote_status": "NOT RUN"
```

No exact remote API mapping or credential verification had occurred. ATM-028 (tracker implementation) and ATM-030 (unified result publication) were closed as completed tasks; ATM-029 (real bidirectional HelixTrack synchronization) remained open.

### Current-codebase notes

- Registry/generator: [`helix/scripts/workable_items.py`](helix/scripts/workable_items.py)
- Tests: [`helix/tests/test_workable_items.py`](helix/tests/test_workable_items.py)
- Qualification: [`helix/scripts/verify_workable_items.py`](helix/scripts/verify_workable_items.py) and [`helix/evidence/workable-items/results.json`](helix/evidence/workable-items/results.json)
- SQLite database: [`helix/docs/workable_items.db`](helix/docs/workable_items.db)
- Usage and recovery guide: [`helix/docs/workable-items.md`](helix/docs/workable-items.md)
- Offline outbox: [`helix/docs/tracking/helixtrack-outbox.json`](helix/docs/tracking/helixtrack-outbox.json)
- Open and closed exports: [`helix/docs/Issues.md`](helix/docs/Issues.md), [`helix/docs/Fixed.md`](helix/docs/Fixed.md)

## 10. Scalability review and unified test-result publication

**Date:** 2026-10-28  
**Record type:** detailed inherited assistant/work record

### Scalability verdict

The explicit verdict was that enterprise-scale qualification was **not complete**. The review covered the Gin API, content delivery, scan jobs, ingest, memory/PostgreSQL catalogs, AI memory, Redis KOSync, internal sync, OPDS/OpenSearch, remote sources, identification/covers, transports, the web prototype, and deployment.

Unresolved risks included:

- a global advisory lock serializing catalog writes;
- whole-work delete/reinsert behavior;
- offset pagination and `%text%` predicates;
- per-work child-query fan-out;
- process-local jobs and history;
- fixed pool limits;
- no enterprise load, saturation, query-plan, or multi-node results.

The existence of connection-pool bounds and atomic reconciliation was not treated as evidence of unlimited scalability.

### Results publication

A publisher generated `docs/testing/results/index.md` and `summary.json` from preserved artifacts, including evidence existence and SHA-256 metadata. At the inherited checkpoint there were 27 outcomes: 15 PASS, 3 FAIL, 2 PARTIAL, 6 NOT RUN, and 1 NOT DELIVERED. Overall release qualification was `NOT READY`.

The testing guide was rewritten to link the authoritative index, remove stale ten-mutant and old-artifact references, record the 21-mutant campaign, and retain major non-run areas.

### Current-codebase notes

- Review: [`helix/docs/scalability-review.md`](helix/docs/scalability-review.md)
- Publisher: [`helix/scripts/publish_test_results.py`](helix/scripts/publish_test_results.py)
- Authoritative results: [`helix/docs/testing/results/index.md`](helix/docs/testing/results/index.md), [`helix/docs/testing/results/summary.json`](helix/docs/testing/results/summary.json)
- Testing guide: [`helix/docs/testing.md`](helix/docs/testing.md)

> **Snapshot caution:** the published index is a generated point-in-time record. The later 111/111 automated repository-index milestone described below is newer than the index’s 106/111 research row. That newer automated milestone does not complete manual review or authorize changing unrelated qualification states without regenerating the index.

## 11. HelixQA incorporation attempt and exact-version dependency recovery

**Date:** 2026-10-28 through the 2026-10-29 history-request checkpoint  
**Record type:** inherited work record plus current continuation record  
**Status at checkpoint:** **NOT BUILT / NO HELIXQA CHALLENGE PASS**

### Initial failure

The real HelixQA CLI/bank-runner build failed because declared own-organization dependency checkouts were incomplete. A narrow build of `pkg/testbank` first established a real requirement on `digital.vasic.challenges`. GitHub API access returned 403 and Git clones timed out; GitHub codeload archives provided an alternative retrieval path. Challenges was retrieved with SHA-256 evidence.

The build then required DocProcessor, LLMOrchestrator, LLMProvider, LLMsVerifier, and security. These were also retrieved through codeload. Archive-root capitalization differences for Challenges and LLMProvider required extraction corrections. Eventually all eight own-org source trees were available for qualification:

- Challenges;
- Containers, using the existing current submodule;
- DocProcessor;
- LLMOrchestrator;
- LLMProvider;
- LLMsVerifier;
- security;
- VisionEngine.

VisionEngine had an exact recorded revision, `d52fa447d168b2023613cdd6e0bcf7e9c2148bbb`. Other own-org dependencies initially had branch-archive SHA-256 evidence but not established exact commit IDs. That provenance gap remained explicit.

### Isolated qualification build

The active build used a temporary copy of the QA module with absolute local replacements into the qualification dependency workspace. The current project’s original `submodules/qa/go.mod` was not overwritten by this experiment. This was a qualification-only arrangement, not a claim of clean incorporation.

Both the Go proxy and ordinary Git operations were unreliable. Exact-version source archives were therefore resolved iteratively from codeload. The resolver ran the bank-runner build offline, parsed actual missing-module errors, retrieved the requested version, recorded hashes, and appended a local replacement.

Important resolver corrections included:

1. **Pseudo-versions:** accepting the optional `0.` before the 14-digit timestamp and using the trailing commit for retrieval.
2. **Legacy modules:** synthesizing only minimal `go.mod` metadata for pre-module source archives such as go-spew; source implementation files were not replaced with stubs.
3. **Moby submodules:** retrieving `api/v1.54.1` and `client/v0.4.0` tags and selecting the corresponding module subdirectory.
4. **Selective monorepo extraction:** after full Moby extraction proved too slow, extracting only the required `api` or `client` subtree.
5. **Additional real module layouts:** adding mappings for Moby `sys` submodules, OpenTelemetry auto/contrib/core modules, genproto subdirectories, gRPC, protobuf, YAML, and other actual module hosts as the compiler reached them.

The resolver did not silently substitute a different Docker API version or manufacture local API implementations.

### Preserved errors and boundary

Errors preserved in the build log included:

- missing own-org modules;
- `module lookup disabled by GOPROXY=off` for exact transitive dependencies;
- a 404 from treating Moby `api` as a root `v1.54.1` tag;
- a 404 from treating Moby `sys/sequential` as a root tag;
- unsupported module-host errors before real OpenTelemetry mappings were added;
- repeated bounded-run timeouts during dependency extraction/resolution.

At the latest pre-history checkpoint, the build had progressed through many container, WebRTC, OpenTelemetry, Go `x/*`, genproto, and other dependencies, reaching `google.golang.org/grpc@v1.80.0`. The bank-runner binary still did not exist. No `BUILD PASS`, Helix-specific bank result, device result, or ADB result was claimed.

### Current-codebase notes

- QA dependency declaration: [`helix/submodules/qa/helix-deps.yaml`](helix/submodules/qa/helix-deps.yaml)
- Real bank-runner entry point: [`helix/submodules/qa/cmd/helixqa-bank-session/main.go`](helix/submodules/qa/cmd/helixqa-bank-session/main.go)
- Existing build-failure evidence: [`helix/evidence/helixqa-build-test.log`](helix/evidence/helixqa-build-test.log)
- Exact-archive build campaign log: [`helix/evidence/helixqa-archive-build.log`](helix/evidence/helixqa-archive-build.log)
- Open tracker item: ATM-021 in [`helix/docs/Issues.md`](helix/docs/Issues.md)

## 12. Context-preservation requests and inherited continuation record

**Date:** between the expanded work and the current continuation  
**Record type:** partially recovered user prompts; generated summary available

Two summary requests preserved the common instruction:

> CRITICAL: Respond with TEXT ONLY. Do NOT call any tools.

The earlier request required plain-text `<analysis>` followed by `<summary>`. The later request required a detailed summary with nine specified sections and no tool calls. The complete wording of those summary specifications was not retained in the accessible user-turn record, so it is not reconstructed here.

The resulting continuation summary is the principal source for this document’s detailed reconstruction. It preserved:

- the primary request and expanded intent;
- key technical concepts;
- files and code sections;
- errors and fixes;
- solved and unresolved problem areas;
- known user messages;
- security and quality constraints;
- pending tasks;
- active HelixQA work;
- the next bounded build step.

That summary is an assistant-generated continuity record, not a verbatim transcript of the preceding sessions.

## 13. Continuation: HelixQA resolver resumed

**Date:** 2026-10-29  
**Record type:** current session, with verbatim assistant progress messages where available

The active instruction was to resume from the cached exact-version dependency archives, inspect the next actual build error, and continue without claiming incorporation or producing final archives.

### Verbatim assistant progress message

> The cached Moby source extracted successfully, but the earlier timeout occurred before its local replacement was recorded. I’m resuming from that cache so the next build exposes the actual remaining dependency or compiler error.

### Work record

The Moby API archive had downloaded, but full extraction had materialized only a small portion before the execution window expired. Selective extraction of the `api` module completed, its `go.mod` was verified, and its replacement was recorded. The next genuine build error was `github.com/moby/moby/client@v0.4.0`. Client resolution then progressed to `patternmatcher` and `sys/sequential`, exposing the separately tagged Moby `sys` repository layout.

A later run resolved numerous Pion/WebRTC and Prometheus-related dependencies before reaching OpenTelemetry. The resolver initially stopped at `go.opentelemetry.io/auto/sdk@v1.2.1` because that module host had no mapping.

### Verbatim assistant progress message

> The build has progressed through the container and WebRTC dependency sets. It is now at OpenTelemetry’s separately tagged modules; I’m adding their real repository/submodule mappings rather than substituting versions or creating API stubs.

Real OpenTelemetry, genproto, gRPC, protobuf, and related mappings were then added. Subsequent resolution reached the gRPC archive extraction before another execution-window timeout. At the moment the history request took priority, no successful bank-runner build or HelixQA challenge execution had followed.

## 14. Automated repository-index completion — 2026-10-29

**Record type:** current background completion report; automated coverage only

A background research worker reported completion of the outstanding clone/index work:

- **111/111 analyses present**;
- **111/111 file ledgers present**;
- **0 missing automated analyses**;
- **0 unavailable automated targets**;
- status: `automated_index_complete_manual_review_pending`;
- `manual_review_status=pending` throughout;
- `support_claims_allowed=false` on the 15 newly completed targets.

The report did **not** claim manual architecture/code/license review or feature-support proof.

### Fifteen completed targets

| Area | Repository slug | Pinned revision prefix | Indexed files / ledger entries |
|---|---|---:|---:|
| Backend | ab0oo-gopds | `b418cabdb674` | 50 |
| Backend | bookorbit-bookorbit | `f2a7a1118818` | 5,902 |
| Backend | crocodilestick-calibre-web-automated | `4fc2c4459965` | 1,198 |
| Backend | dubyte-dir2opds | `9f40bc1fd587` | 51 |
| Backend | edrlab-lcp-server | `e41ae954097f` | 99 |
| Backend | mikespub-org-seblucas-cops | `856e45308371` | 719 |
| Backend | readium-readium-lcp-server | `32570b69d029` | 241 |
| Client | anx-reader | `0c7fc1ad4551` | 1,304 |
| Client | dyslibria | `7ad405d3d612` | 233 |
| Client | epuppy | `a415f588a6ff` | 4 |
| Client | inkmod | `51a381bd2407` | 1,175 |
| Client | koreader | `c8eb592a8b6c` | 1,004 |
| Client | shoko | `eb319d18d715` | 1,217 |
| Client | sumatrapdf | `a30e4d642941` | 5,690 |
| Client | thorium-reader | `1f3e03448eed` | 1,921 |

Additional provenance notes from the completion report:

- BookOrbit used the client/bookorbit alias checkout with origin `github.com/bookorbit/bookorbit`.
- Dyslibria used corrected URL `wilburforce83/dyslibria-library`.
- Epuppy used corrected URL `tlinden/epuppy`.
- Inkmod used corrected URL `alpzoloto-sudo/inkmod`.
- Shoko used `git.sr.ht/~shayan/Shoko`.
- All 15 were real shallow single-branch Git clones; no partial checkout was analyzed.
- Thorium Reader and SumatraPDF branch heads advanced during the long clone window. The recorded revisions are the actual cloned HEADs, not earlier remote readings.
- Network failures included slow transfer phases, connection refusals, early EOF, and connect timeouts. Bounded retries eventually completed; staging directories were removed and retry workers exited.

### Current-codebase notes

- Machine summary: [`helix/research/analysis_summary.json`](helix/research/analysis_summary.json)
- Backend analyses: [`helix/research/backend_analysis/`](helix/research/backend_analysis/)
- Client analyses: [`helix/research/client_analysis/`](helix/research/client_analysis/)
- File ledgers: [`helix/research/file_evidence/`](helix/research/file_evidence/)
- The manual-review task remains ATM-020 in [`helix/docs/Issues.md`](helix/docs/Issues.md); its wording still reflects the earlier 106-repository checkpoint and needs reconciliation with the newer automated summary.

## 15. Current history-export request — 2026-10-29

### Verbatim prompt

> Create us now for mall sessions we did for this project (Helix Reader) from the oldest date to the last one (today) one huge downloadable history.md file which will contain the whole history / conversation / prompts with all details in the thread - every prompt we have given, every response / answer we have received, and all this added into one nicely and properly structured and organized markdown document. Add if possible additional notes through the document referencing the codebase - only tha latest codebase version we have now!

### Verbatim assistant progress messages

> I’ll assemble the recoverable session record in chronological order and cross-reference it to the current Helix tree. Where an older verbatim turn is not present in the accessible record, I’ll mark that explicitly instead of reconstructing it as if it were original dialogue.

> No verbatim archive of the earlier chat sessions is present in the accessible workspace. The document will therefore distinguish three evidence classes clearly: verbatim recovered prompts, inherited session-summary records, and current-codebase/evidence references—so it remains useful without pretending that missing dialogue has been recovered.

> The history will also include a requirement-to-code index, the preserved failure/pass chronology, and the current 111-repository automated-index milestone. Those notes will point only to the present Helix tree; older archives will be mentioned as historical artifacts, not used as code references.

### Response embodied by this document

The assistant searched the accessible workspace for transcript/history/session records and found no earlier verbatim dialogue archive. It used the inherited continuation summary, current documentation, machine evidence, the current research summary, and the live continuation turns. The resulting file is intentionally labelled as partial verbatim recovery rather than falsely titled a complete transcript.

# Part II — Current-codebase traceability

## 16. Requirement-to-implementation map

This map is deliberately about the **latest accessible codebase**, not about reconstructing old file versions. “Implemented” is scoped to the current code and evidence; it does not imply completion of external, native, production, or release qualification.

| Requirement area | Current implementation/reference | Current evidence or boundary |
|---|---|---|
| Work → edition → file identity | [`helix/server/internal/catalog/`](helix/server/internal/catalog/), [`helix/server/internal/ingest/`](helix/server/internal/ingest/) | Core model present; enterprise-scale reconciliation remains open. |
| Edition-specific ISBN identity | [`helix/core/isbn/isbn.go`](helix/core/isbn/isbn.go) | Canonicalization and focused tests exist. |
| Multiple files per edition | [`helix/server/internal/catalog/`](helix/server/internal/catalog/) | Catalog model present; no complete native reader claim. |
| Metadata provenance and review states | [`helix/server/internal/catalog/`](helix/server/internal/catalog/), [`helix/server/internal/ingest/`](helix/server/internal/ingest/) | Deterministic evidence remains authoritative over AI proposals. |
| Read-only local indexing | [`helix/server/internal/sources/local.go`](helix/server/internal/sources/local.go) | Current GET path revalidates one open descriptor. |
| HTTP(S), WebDAV, OPDS sources | [`helix/server/internal/sources/`](helix/server/internal/sources/) | Connector behavior exists; deployment-specific external interoperability is not fully qualified. |
| FTP/FTPS, SFTP, SMB, S3, cloud/rclone | [`helix/server/internal/sources/`](helix/server/internal/sources/), [`helix/core/sources/sources.go`](helix/core/sources/sources.go) | Protocol surfaces must not be confused with provider-specific live validation. |
| Bounded EPUB parsing | [`helix/core/pkg/parse/epub/`](helix/core/pkg/parse/epub/) | Parser/fuzz evidence; complete rendering is separate. |
| Bounded FB2 parsing | [`helix/core/pkg/parse/fb2/`](helix/core/pkg/parse/fb2/) | Parser and ISBN metadata behavior; complete conversion is separate. |
| MOBI/PalmDOC | [`helix/core/pkg/parse/mobi/`](helix/core/pkg/parse/mobi/) | Bounded LZ77 decoding and parser tests; no full DRM/render claim. |
| TXT and CBZ | [`helix/core/pkg/parse/txt/`](helix/core/pkg/parse/txt/), [`helix/core/pkg/parse/cbz/`](helix/core/pkg/parse/cbz/) | Detection/parsing evidence only at their implemented stages. |
| Format registry and conversion matrix | [`helix/core/formats/`](helix/core/formats/), [`helix/core/pkg/convert/`](helix/core/pkg/convert/) | Registry/matrix entries are not proof that every conversion executes. |
| OPDS 1.2/2.0 and OpenSearch | [`helix/server/internal/httpapi/`](helix/server/internal/httpapi/), [`helix/docs/api.md`](helix/docs/api.md) | Live API evidence exists; third-party client matrix remains non-run. |
| Acquisition identity | [`helix/server/internal/sources/opds.go`](helix/server/internal/sources/opds.go) | Collision-safe public keys and atomic replacement implemented and tested. |
| HTTP/1.1, HTTP/2, HTTP/3/QUIC | [`helix/server/`](helix/server/), [`helix/scripts/verify_live_transport.py`](helix/scripts/verify_live_transport.py) | Three post-instrumentation 5/5 repeats; one earlier readiness failure retained. |
| Brotli/gzip and identity fallback | [`helix/server/internal/httpapi/compression.go`](helix/server/internal/httpapi/compression.go) | Keepalive defect fixed with raw-socket regression and mutation. |
| Strong ETags and ranges | [`helix/server/internal/httpapi/`](helix/server/internal/httpapi/), [`helix/server/internal/sources/local.go`](helix/server/internal/sources/local.go) | Live revalidation and range tests; stale-validator defect fixed. |
| PostgreSQL catalog | [`helix/server/internal/catalog/postgres.go`](helix/server/internal/catalog/postgres.go) | Real-service tests exist; load/saturation/multi-node qualification absent. |
| Redis KOSync | [`helix/server/internal/kosync/`](helix/server/internal/kosync/) | Real Redis tests; physical KOReader interoperability not run. |
| AI/pgvector memory | [`helix/server/`](helix/server/) and live-service evidence | Deterministic metadata precedence retained; external production LLM wiring incomplete. |
| Responsive web catalog prototype | [`app/`](app/) | Four-viewport sample-only demo; disconnected from server, not native-client evidence. |
| Native Windows/macOS/Linux clients | [`helix/docs/testing/results/index.md`](helix/docs/testing/results/index.md) | **NOT DELIVERED.** |
| Native Android/iOS readers | [`helix/docs/testing/results/index.md`](helix/docs/testing/results/index.md) | **NOT DELIVERED.** |
| Low-end and workstation device demos | [`helix/docs/testing/results/index.md`](helix/docs/testing/results/index.md) | Browser viewports only; hardware matrix incomplete. |
| Container build/deploy/restart | [`helix/deploy/docker-compose.yml`](helix/deploy/docker-compose.yml), [`helix/server/Dockerfile`](helix/server/Dockerfile) | Recipes exist; runtime cold start/restart/upgrade/rollback **NOT RUN**. |
| Firebase distribution | ATM-025 in [`helix/docs/Issues.md`](helix/docs/Issues.md) | Project-bound execution **NOT RUN**; the tracker’s inherited `docs/deployment.md` reference is absent from the latest tree. |
| SQLite workable-items registry | [`helix/docs/workable_items.db`](helix/docs/workable_items.db), [`helix/scripts/workable_items.py`](helix/scripts/workable_items.py) | Real registry, lifecycle, recovery, stress, and mutation qualification passed. |
| Markdown/HTML/PDF tracker exports | [`helix/docs/`](helix/docs/) | Exports generated; final PDF visual QA must be refreshed after changes. |
| HelixTrack remote synchronization | [`helix/docs/tracking/helixtrack-outbox.json`](helix/docs/tracking/helixtrack-outbox.json) | Offline outbox only; remote push/pull **NOT RUN**. |
| Automated repository indexing | [`helix/research/analysis_summary.json`](helix/research/analysis_summary.json) | 111/111 analyses and ledgers; manual review pending. |
| Manual architecture/code/license review | [`helix/research/`](helix/research/) | **PENDING**; automated keyword evidence is not a porting decision. |
| HelixQA incorporation | [`helix/submodules/qa/`](helix/submodules/qa/) | Exact-source build campaign ongoing at checkpoint; no runner or challenge pass yet. |
| Enterprise scalability | [`helix/docs/scalability-review.md`](helix/docs/scalability-review.md) | Risks documented; representative load and multi-node qualification **NOT RUN**. |
| OPDS-PSE and LCP | [`helix/docs/api.md`](helix/docs/api.md), [`helix/docs/faq.md`](helix/docs/faq.md) | Open ATM-031; no fulfillment claim. |
| Full rendering/conversion/AI/DRM matrix | ATM-032 in [`helix/docs/Issues.md`](helix/docs/Issues.md) | Open; detection must not be inflated into fulfillment. The inherited `docs/formats.md` reference is absent from the latest tree. |
| Independent specialized reviews | [`helix/docs/testing/results/index.md`](helix/docs/testing/results/index.md) | Not completed as a comprehensive independent release gate. |
| Secret scanning and final clean-room package | [`helix/docs/testing/results/index.md`](helix/docs/testing/results/index.md) | Final qualification **NOT RUN**; fresh final archives withheld. |

## 17. Qualification chronology and preserved outcomes

The current result publication is significant because later passes do not erase earlier defects.

| Campaign | Recorded result | Interpretation |
|---|---:|---|
| Core unit/integration | PASS | Scoped to current core test log. |
| Core race detector | PASS | Scoped to recorded race campaign. |
| Core vet | PASS | Scoped static analysis result. |
| Core format/parser fuzzing | PASS | Six core format targets plus server range target in the published matrix. |
| Server unit/integration and repeats | PASS | Test plus three repeat logs. |
| Server race and vet | PASS | Recorded campaigns only. |
| Real PostgreSQL/pgvector/Redis | PASS | Real services, distinct from unit doubles. |
| Web lint/build/dependency dry run | PASS | Prototype engineering checks. |
| Four-viewport web checks | 64/64 PASS | Sample-only prototype, not native-client evidence. |
| Go mutation | 21/21 rejected | Original-tree hash verification included. |
| Workable-items qualification | 10 groups PASS | SQLite stress/recovery/corruption/mutations. |
| Ordinary live-server post-fix runs | 3 × 37/37 PASS | Current-tree post-compression behavior. |
| Daemon/PostgreSQL/Redis restart | 38/38 PASS | Stable reattachment and non-duplicating reconciliation. |
| Backup/restore post-fix | 13/13 PASS | Recovery after compression change. |
| Transport post-instrumentation | 3 × 5/5 PASS | HTTP/1.1, HTTP/2, HTTP/3, TLS, Alt-Svc. |
| Earlier live daemon-restart attempt | FAIL | Connection reset after identity attachment; retained. |
| Post-fix transport readiness attempt | FAIL | Unexplained initial readiness failure; retained. |
| Initial HelixQA build | FAIL | Missing own-org dependencies; no test pass. |
| Repository research at publication | PARTIAL, 106/111 | Superseded only for automated count by later 111/111 report. |
| Enterprise load | NOT RUN | No representative saturation/multi-node result. |
| Native clients | NOT DELIVERED | No installable native application matrix. |
| Device demos | PARTIAL | Browser viewports only. |
| Container runtime | NOT RUN | Configuration is not execution. |
| Firebase distribution | NOT RUN | No project-bound evidence. |
| External interoperability | NOT RUN | No physical/external-client/provider matrix. |
| HelixTrack remote sync | NOT RUN | Offline outbox only. |
| Final archives | NOT RUN | Withheld pending actual scope completion. |

**Current-codebase authority:** [`helix/docs/testing/results/index.md`](helix/docs/testing/results/index.md) and the linked artifacts under [`helix/evidence/`](helix/evidence/).

## 18. Current open-scope summary

The following were still open in the recovered record or current tracker and must not be lost in a historical narrative:

1. Manual architecture, code, license, innovation, and porting review for all indexed repositories.
2. Exact commit provenance for own-org HelixQA dependencies fetched as branch archives.
3. Successful real HelixQA bank-runner build and Helix-specific challenge execution.
4. Native clients and real-device/low-end/workstation interactive demos.
5. Actual Docker/Podman cold start, restart, upgrade, rollback, and distribution.
6. Firebase configuration and project-bound distribution execution.
7. Physical KOReader, external OPDS client, and provider-specific rclone validation.
8. Enterprise load, saturation, query-plan, and multi-node qualification and hotspot remediation.
9. Verified bidirectional HelixTrack synchronization against the exact remote API.
10. Full required-format rendering, conversion, content-AI, DRM, OPDS-PSE, and LCP scope.
11. Current reader/backend/accessibility/synchronization/deployment research and traced implementation decisions.
12. Independent specialized code, security, architecture, and deployment review phases.
13. Refreshed final tracker-PDF visual QA after the latest exports.
14. Final secret scans, clean extraction/build/test, deterministic archive-content checks, and only then fresh ZIP/tar.gz creation.

This list explains why historical packages must not be confused with a new final release.

# Part III — Current work-item snapshot

## 19. Registry items preserved in the latest tracker

These titles and statuses are extracted from the current generated Markdown exports. They are a snapshot for orientation; the authoritative database is [`helix/docs/workable_items.db`](helix/docs/workable_items.db).

### Open items

| ID | Title | Current recorded status |
|---|---|---|
| ATM-019 | Complete revision-pinned research for every deduplicated reader repository | In progress |
| ATM-020 | Perform manual architecture, code, license, and innovation review of indexed repositories | Queued |
| ATM-021 | Build and execute Helix-specific HelixQA challenge banks | In progress |
| ATM-022 | Deliver catalog and reader clients for native desktop, mobile, and workstation platforms | Queued |
| ATM-023 | Provide interactive demonstrations across the platform and device matrix | Queued |
| ATM-024 | Qualify containerized build, deployment, restart, and distribution workflows | Queued |
| ATM-025 | Configure and validate Firebase distribution without treating configuration as deployment | Queued |
| ATM-026 | Validate interoperability on real readers, OPDS clients, and remote providers | Queued |
| ATM-027 | Remove catalog and API scalability bottlenecks under measured enterprise loads | In progress |
| ATM-029 | Implement verified bidirectional HelixTrack synchronization | In progress |
| ATM-031 | Implement OPDS-PSE, LCP fulfillment, and remaining external reader integrations | Queued |
| ATM-032 | Complete rendering, conversion, and content-AI fulfillment for every required format | Queued |
| ATM-033 | Run final clean qualification and produce fresh ZIP and tar.gz archives | Queued |

### Closed items

| ID | Title | Current recorded status |
|---|---|---|
| ATM-001 | Establish core document, cache, settings, and synchronization primitives | Completed (→ Fixed.md) |
| ATM-002 | Implement multi-format detection and bounded document parsing | Implemented (→ Fixed.md) |
| ATM-003 | Canonicalize ISBN identity across equivalent representations | Fixed (→ Fixed.md) |
| ATM-004 | Index local and remote libraries in place without reorganizing source bytes | Implemented (→ Fixed.md) |
| ATM-005 | Make OPDS acquisition identity collision-safe and atomically replaceable | Fixed (→ Fixed.md) |
| ATM-006 | Revalidate content before strong ETag and byte-range responses | Fixed (→ Fixed.md) |
| ATM-007 | Preserve persistent HTTP connections during identity compression fallback | Fixed (→ Fixed.md) |
| ATM-008 | Provide HTTP/1.1, HTTP/2, HTTP/3, TLS, and content negotiation transports | Implemented (→ Fixed.md) |
| ATM-009 | Reconcile durable PostgreSQL catalogs atomically after source scans | Fixed (→ Fixed.md) |
| ATM-010 | Implement monotonic KOReader progress synchronization with Redis | Implemented (→ Fixed.md) |
| ATM-011 | Deliver an honest responsive web catalog prototype | Implemented (→ Fixed.md) |
| ATM-012 | Establish repeatable deterministic core and server qualification | Completed (→ Fixed.md) |
| ATM-013 | Fuzz all exposed format and byte-range parsers | Completed (→ Fixed.md) |
| ATM-014 | Prove regression tests reject targeted restoration of fixed defects | Completed (→ Fixed.md) |
| ATM-015 | Exercise real PostgreSQL, pgvector, and Redis integrations | Completed (→ Fixed.md) |
| ATM-016 | Verify backup and restore recovery against the current tree | Completed (→ Fixed.md) |
| ATM-017 | Prevent credentials and sensitive values from entering source and evidence | Completed (→ Fixed.md) |
| ATM-018 | Keep deterministic metadata evidence authoritative over AI proposals | Implemented (→ Fixed.md) |
| ATM-028 | Operate the SQLite workable-items single source of truth and synchronized exports | Completed (→ Fixed.md) |
| ATM-030 | Publish all current test results and limitations under docs/testing/results | Completed (→ Fixed.md) |

# Part IV — Automated research inventory

## 20. All 111 currently indexed repositories

This inventory is generated from the latest [`helix/research/analysis_summary.json`](helix/research/analysis_summary.json). Every row has automated static analysis and a file ledger. **Every row still requires manual architecture/code/license review; no feature-support claim is implied.**

| # | Kind | Slug | Exact revision | Files / ledger records | Manual review |
|---:|---|---|---|---:|---|
| 1 | backend | ab0oo-gopds | `b418cabdb67479f1b5b07f1e78c2b0c5cd86b5fb` | 50 | pending |
| 2 | backend | advplyr-audiobookshelf | `edb1f7efc92ecacdef51aafb4d9d5054ce10604f` | 954 | pending |
| 3 | backend | anansi-project-opds-pse | `9a082e4e254048a584df55255114dca765c246ba` | 8 | pending |
| 4 | backend | beyondstorage-go-storage | `1d5ce66ccc0132bcb6d8a224e93a0c2d3963a460` | 691 | pending |
| 5 | backend | bookorbit-bookorbit | `f2a7a111881899e7c6cb233a0ec2d8a9c25235c6` | 5902 | pending |
| 6 | backend | bornholm-go-webdav | `08cd422bee325e0ff85b76b3afcff86830711aca` | 76 | pending |
| 7 | backend | cgillinger-colophon | `3676404ab253c7e0ef78fcda973a27424056114d` | 475 | pending |
| 8 | backend | crocodilestick-calibre-web-automated | `4fc2c4459965efb1b8ff43ea7849c2f47a755d9c` | 1198 | pending |
| 9 | backend | dubyte-dir2opds | `9f40bc1fd587ec1c1ddbab9888529effa471ea79` | 51 | pending |
| 10 | backend | edrlab-lcp-server | `e41ae954097fc42b2d8996fabdfd355a1824108c` | 99 | pending |
| 11 | backend | fellipec-kosync-go | `f74cc7409ff6035212f09be877963c1949145fdf` | 14 | pending |
| 12 | backend | gotson-komga | `023ec905c10a35bbb8e29095420d266cd3adad60` | 1908 | pending |
| 13 | backend | grimmory-tools-grimmory | `021cb138dc7f19583142c55fd5070201c40b403b` | 2887 | pending |
| 14 | backend | kareadita-kavita | `d549d5b1609086c8190b9bee6eb48ebfed71e1df` | 3917 | pending |
| 15 | backend | koreader-koreader-sync-server | `46ef6b84a3930b3f0265efac74899f2256c158f1` | 31 | pending |
| 16 | backend | mikespub-org-seblucas-cops | `856e45308371ef578d7492929b321dff66beb2a2` | 719 | pending |
| 17 | backend | petr-prikryl-opds-abs | `add44a30d4cdd150194e86115a1f6b1dcee62d40` | 55 | pending |
| 18 | backend | readium-readium-lcp-server | `32570b69d02948801f161e7923648dcc0bbfb178` | 241 | pending |
| 19 | backend | seblucas-cops | `7828cc600e904941583227071b8f4efd3b9385fb` | 388 | pending |
| 20 | backend | smoores-dev-storyteller | `281597bcdda93c438f263720ce805b3ae679cba9` | 3402 | pending |
| 21 | backend | stumpapp-stump | `d0f51c8e6da74d584a2a4cda8d9db02337de73ca` | 2808 | pending |
| 22 | backend | vaggeliskls-cloud-webdav-server | `31a43793da61e6554ec836a63fae66b25d0bc831` | 60 | pending |
| 23 | client | acbr | `1dec0803273fcbf050540a076e79ddcc2c61a651` | 405 | pending |
| 24 | client | aniyomi | `97414446b8a95994c72dd33c41c971a89d4d25b8` | 2039 | pending |
| 25 | client | anx-reader | `0c7fc1ad4551f7dbbeba6a6468f69d830957fa2a` | 1304 | pending |
| 26 | client | audiobookshelf | `edb1f7efc92ecacdef51aafb4d9d5054ce10604f` | 954 | pending |
| 27 | client | baca | `13ee794b08a4852f63002e7b0ca6a5c82b4239d4` | 60 | pending |
| 28 | client | bk | `8d527788821faee1a2f961d576997351cd68767d` | 10 | pending |
| 29 | client | book-s-story | `464891ec8cb00196d10b6dbaba3b7409c8fd27a8` | 766 | pending |
| 30 | client | booklore | `01b3407528a6ce00d064dbb9aa3f69b72d82641b` | 2786 | pending |
| 31 | client | bookokrat | `9c02ee3b3a0f86791b3917cb1a93436d92a8603e` | 371 | pending |
| 32 | client | bookorbit | `99aa7dce705986eac29a8f73aa65331de5f50680` | 5549 | pending |
| 33 | client | bookr | `ec5da5a5b57b9d229caf85acc7e22caa31ff5f16` | 7 | pending |
| 34 | client | bookworm | `035d836d814bf32bab56e58e951665879a1948d7` | 344 | pending |
| 35 | client | calibre | `6be72572e5b8b5963ba1887c492442633e41b537` | 5003 | pending |
| 36 | client | calibre-web | `6e840acbda8d62fdd159bf3855c01872d629a914` | 972 | pending |
| 37 | client | candela | `26f079bc6d3dda3e30b8a6fc511990084cc0c838` | 2099 | pending |
| 38 | client | chytanka | `39b23eeb2abc0b9205695fd7b773ae616e2c3db5` | 390 | pending |
| 39 | client | codexa | `083dffe43e2fcd31a1222ea09196909f78e7fc2c` | 358 | pending |
| 40 | client | cops | `7828cc600e904941583227071b8f4efd3b9385fb` | 388 | pending |
| 41 | client | crossmux | `6b5566c844667857acebfe565d373423cdd16e06` | 1758 | pending |
| 42 | client | crosspoint-reader | `9ac185c8c50d0397ab4308af245f498ea4e8b817` | 1035 | pending |
| 43 | client | diy-esp32-epub-reader | `2c2f57fdd7e2a788d14a0bcb26b9e845a47aac42` | 142 | pending |
| 44 | client | diy-esp32s3-epub-reader | `3b35576127dd2e3b27a46336e301c7b4793f95b7` | 1458 | pending |
| 45 | client | dyslibria | `7ad405d3d6121f978b623009f965dee550f9ab6f` | 233 | pending |
| 46 | client | ebook-viewer | `975a7bcecabcf4739363cbd87a88c1c1b4551f1c` | 32 | pending |
| 47 | client | eboreader | `e8421fafaac8418e9e3c86ed6ea5ed023c74733c` | 51 | pending |
| 48 | client | epub-inkplate | `7360a3fa966d2048202a470d9d1a93362b1337d6` | 1859 | pending |
| 49 | client | epub-js | `eee359d0790002115a1156a9833c54f4bcd44c1d` | 181 | pending |
| 50 | client | epubwebreader | `65b298c95f7f7d4205d268563e61bcb811979f7c` | 133 | pending |
| 51 | client | epuppy | `a415f588a6ff2b5c8a354f56cf7e116a9a96afbe` | 4 | pending |
| 52 | client | epy | `6b0e9fe0773f05fdf844b574f0f28df3961f60ab` | 63 | pending |
| 53 | client | ferrous | `066cb49a2d42fc17064cf50a41ebc108b65f0d48` | 218 | pending |
| 54 | client | flow | `36ce6fe4d577305f0c1fb9fb7d9b64a71b37d330` | 345 | pending |
| 55 | client | foliate | `67b6676d3f936c5edea91d4d903385ef39dd25c0` | 140 | pending |
| 56 | client | foliate-js | `78914aef4466eb960965702401634c2cb348e9b1` | 231 | pending |
| 57 | client | folio | `3da2b3d9157cc1eccdf39063533910ac8d558c41` | 455 | pending |
| 58 | client | gopherbook | `e515c353c4272d46e5e815df899e4c9d8ce7d4ab` | 43 | pending |
| 59 | client | goreader | `e461f046ea6fef8d5e496ffd4008dc995b96227e` | 33 | pending |
| 60 | client | grimmory | `021cb138dc7f19583142c55fd5070201c40b403b` | 2887 | pending |
| 61 | client | inkmod | `51a381bd24076263c02782d7511e1bd54e74715b` | 1175 | pending |
| 62 | client | jellybook | `505684a96fd2529cf17697a97824064b74671297` | 419 | pending |
| 63 | client | kavita | `d549d5b1609086c8190b9bee6eb48ebfed71e1df` | 3917 | pending |
| 64 | client | komga | `023ec905c10a35bbb8e29095420d266cd3adad60` | 1908 | pending |
| 65 | client | koodo-reader | `fe1abf4b8960f4e48a2a948659c3bb816a6e2606` | 851 | pending |
| 66 | client | koreader | `c8eb592a8b6cac9e65be7288bd27fb0be9405ad1` | 1004 | pending |
| 67 | client | lanraragi | `adfe9e75cbb8dd4db0fef2c2e35698d557c5a59b` | 462 | pending |
| 68 | client | lector | `1b1d87739a8c14e0e22009435350b155fc3e3b77` | 213 | pending |
| 69 | client | legado | `9bb0569269ed557753fc3866250970de6d454fcf` | 2 | pending |
| 70 | client | librera-reader | `35b032bc12ae65d37c263cb29506589b8c7c6836` | 3650 | pending |
| 71 | client | libreread | `0f6ec029b29bba92f7c6983291b291727eb303fe` | 524 | pending |
| 72 | client | librum | `5fdac3454c3c3fbc4191aa6a4f277368895c9c70` | 844 | pending |
| 73 | client | lilybook | `37a48bf3b53140c9de9ccda87e7cc9c34583eaf5` | 150 | pending |
| 74 | client | mangayomi | `0c09db48d057f217c9c75f1edd2fff4c39324783` | 1478 | pending |
| 75 | client | mihon | `e588e04359fe1058206a2cbac1b01018ad787359` | 1473 | pending |
| 76 | client | minimal-ereader | `bb47d6fcbb7d4648549fabdaddeb8c0cc6c20d96` | 31 | pending |
| 77 | client | mitsu | `492d0cc71570936e195b8244efcc70102b5e5209` | 115 | pending |
| 78 | client | myne | `7ec1fdb19ebf19de13d7b61bcb52b3ce4878a579` | 299 | pending |
| 79 | client | novelist | `c8ce3a8534d7eb713c3e5e0261c1da52606acf13` | 155 | pending |
| 80 | client | olib-mobile | `0deb406d6da8d7b802e8d131755eba3eb1866b99` | 265 | pending |
| 81 | client | omnibus | `6055d6ee34865852c801dced9540c270ab248d25` | 787 | pending |
| 82 | client | open-reading | `b379b9693cbc13cbf3a27f0d2ac702ebff9a5789` | 639 | pending |
| 83 | client | opencomic | `67e8564deb02aeb861acd8e5b8c542407e8da135` | 684 | pending |
| 84 | client | openlib | `fd779d7dd8dee1ab07aaf189f406dc00275c6321` | 217 | pending |
| 85 | client | paperpaw | `388e5ac030b15d71dfcad6b3de2b54103307448f` | 2279 | pending |
| 86 | client | polka | `01c2712a771fc72b7eced5887eea68e6bb1876a0` | 209 | pending |
| 87 | client | quill | `790ffd6fb73318a76888d66cb32c88b2477f09dd` | 279 | pending |
| 88 | client | quire | `4cbb7dd401dae870e71468665b9bd3e66b4a583e` | 3233 | pending |
| 89 | client | r2d2bc | `bb1235a82e05074b2ce1e9d131b78ad044e067b3` | 269 | pending |
| 90 | client | rayniyomi | `2cef1d9c878092c64b7252e86c30f0628d01f16c` | 2883 | pending |
| 91 | client | read-cat | `da008dced50c56a3fd58f3a7240d811f56e88ce4` | 347 | pending |
| 92 | client | read-pico-reader | `2f1ef29d446b66ec1d0bf1afa39f1caa8622bbc8` | 1320 | pending |
| 93 | client | readany | `fce44689e1377df1ae99a35e8e71d5e8b3a639d1` | 1414 | pending |
| 94 | client | readest | `2f260ee10ff3500f77a43cfbb24254b7ac422aa7` | 4492 | pending |
| 95 | client | repy | `0a3c3a2063af87fd5d0824cb4588ccfd043cdcf4` | 116 | pending |
| 96 | client | sdlreader-brick | `cdbf9a3d3898da4c004a348d668658727fb14de7` | 144 | pending |
| 97 | client | shoko | `eb319d18d7154ebebb403990f3ac9d99cc1aa893` | 1217 | pending |
| 98 | client | sigmaos-xreader | `fa76a9e2a24bab4db1d46cf03e7f9b2b93d06740` | 4 | pending |
| 99 | client | speakub | `e8e4c6f1fb39a2c24259778677b6b95ac8231aa4` | 75 | pending |
| 100 | client | stanza-redux | `8465c132f9cacd39d0d2931236bc94251ceb8965` | 169 | pending |
| 101 | client | stump | `d0f51c8e6da74d584a2a4cda8d9db02337de73ca` | 2808 | pending |
| 102 | client | sumatrapdf | `a30e4d6429419361132c23c797877f242092ea0e` | 5690 | pending |
| 103 | client | tbr | `26504507d04f5408373ecc7fa5a68a85da3d8450` | 74 | pending |
| 104 | client | termepub | `70a1886f691bcfd87d05829a2839638b4ee24969` | 8 | pending |
| 105 | client | thorium-reader | `1f3e03448eedf352238c0fe5fe4aa765b9e9a1f2` | 1921 | pending |
| 106 | client | thorium-web | `8d49c2b116e93ca50d4ccfdcb3226efde7bc49c8` | 678 | pending |
| 107 | client | tidepage | `7e9caeb0ce4c7eb9922c5abf59cade2d86617004` | 233 | pending |
| 108 | client | verso | `65b4b48cc002661a2061f183640e957509647037` | 104 | pending |
| 109 | client | villain | `8af69e32c3ba617e87dcd945126c2bef517302d5` | 159 | pending |
| 110 | client | xreader | `a4e5ed2c916e6241efb7d9e86d01b709e45a18d8` | 597 | pending |
| 111 | client | yomikiru | `3087618628734ef7e986fcd26b331a80dd4f3b57` | 508 | pending |

## 21. Research interpretation

- Automated coverage at this checkpoint: 111 analyses, 111 file ledgers, zero missing automated targets.
- A current checkout is not available for every historically indexed target; the summary correctly separates checkout availability from analysis coverage.
- Revisions above identify the analyzed source snapshots.
- Keyword matches and file inventories are not manual architecture review, license clearance, or proof that a feature works.
- No support claim is authorized merely by inclusion in this inventory.

# Part V — Conversation completeness register

## 22. Verbatim user-prompt catalogue

This catalogue consolidates the user wording that was actually recoverable. It duplicates some material intentionally so a reader can audit prompt coverage without relying only on the narrative.

### 22.1 Next-round nano-detail investigation

> Continue with next round - investigate up to the nano details exhaustivelly if we may have missed anything?

### 22.2 Deep analysis and deterministic-evidence demand

> Lets dive with our next round! Of top of everything do exhaustive analisys and investiogation for any gaps, shortcomings, inconsistencies, weak spots and danger zones, unfinished and unimplemented parts, any stubs or mocks left, parts or pieces of solution with no documentation, guides or tutorials, diagrams, wschemes, and graphs, FAQs, full tests coverage of all 7+ supported test types and all of them producing MACHINE ONLY rock-solid evidence which is fully deterministically validated and verified with zero bluff of any kind and form and no AI slop! There MUST EXIST NO false or faulty results, no fake and no issues! Do this deep investigation, file by file, class by class, component by component, functionality from specs by functionality from specs, every nano detail, no avoidng, skipping or ignoring - ALL OF THIS MUST BE fully covered and tackled! Root causes deeply checked, fixed, polished and tested LIVE deterministically! Everything MUST BE in enterprise-grade production ready state - all backend services and components, everything client side - all apps and services and tools! Do additional deep web research rounds for finding more proper solutions for most demanding challenges and issues! Seek for all innovative ideas, fame changer opsncourced codebases and make sure we do incorporate them or port them into our solution and our components! Once all is done, fully LIVE tested, confirmed, absolutely consistent, validated and verified provide us final downloadable zip and tar.gz like so far containing all of our work!

### 22.3 Expanded final-round opening

> Now one last round on top of all work done so far - exhaustive investigation and research with systematic LIVE fully deterministic testing and debugging!

The expanded requirement decomposition accompanying this opening is preserved in Section 7. Its complete original paragraph-by-paragraph wording was not available.

### 22.4 Summary-request constraint

> CRITICAL: Respond with TEXT ONLY. Do NOT call any tools.

This constraint appeared in two recovered summary requests. One requested plain-text `<analysis>` followed by `<summary>`; the other requested a detailed nine-section summary. The remaining exact wording was not preserved.

### 22.5 Current history request

> Create us now for mall sessions we did for this project (Helix Reader) from the oldest date to the last one (today) one huge downloadable history.md file which will contain the whole history / conversation / prompts with all details in the thread - every prompt we have given, every response / answer we have received, and all this added into one nicely and properly structured and organized markdown document. Add if possible additional notes through the document referencing the codebase - only tha latest codebase version we have now!

### 22.6 Security constraint preserved in the project record

> Credentials or any secret and sensitive data MUST NOT leak! This MUST BE added as the mandatory constraint into all Submodules and main project's Constitution, CLAUDE.MD and AGENTS.MD.

### 22.7 Additional test constraint preserved in the project record

> Additional tests with fully incorprated helix qa is mandatory!

### 22.8 Prompts known only by summary

The initial “Project Final Packaging” instruction and several repeated exhaustive-round requests are known through summaries, not full original wording. Section 3 preserves their detailed requirements without presenting that paraphrase as quotation. The exact number and wording of those turns remain unrecovered.

## 23. Assistant-response coverage register

| Conversation area | Available assistant record | Treatment in this file |
|---|---|---|
| Initial project setup and packaging | No complete verbatim reply | M1/current-changelog reconstruction only. |
| Nano-details round | No verbatim reply | Not invented; later technical work recorded separately. |
| Deep gaps/danger-zones round | Detailed continuation summary | Reconstructed response/work record with current-code references. |
| Expanded final round | Detailed continuation summary | Compression, tracker, scalability, results, HelixQA, and open-scope sections. |
| Summary requests | The resulting inherited continuation summary | Used as a secondary record, not quoted as original dialogue. |
| HelixQA continuation | Current actions plus verbatim progress messages | Chronological build and error record. |
| Repository clone/index completion | Detailed worker completion report | 111/111 automated milestone and 15-target table. |
| History-export request | Current progress messages and this document | Verbatim messages preserved in Section 15. |

**Unrecovered material:** original word-for-word assistant responses from earlier sessions, exact turn timestamps, exact session boundaries, and any dialogue not represented in the continuation record. No technical paraphrase in this file should be mistaken for recovered verbatim assistant speech.

## 24. Latest-code documentation-drift note

While checking links for this history, three inherited tracker references were found to be absent from the latest tree:

- `docs/sources.md`;
- `docs/deployment.md`;
- `docs/formats.md`.

They remain visible in historical/tracker discussion where necessary, but the clickable current-code notes use existing entry points such as [`helix/README.md`](helix/README.md), [`helix/docs/api.md`](helix/docs/api.md), and the relevant ATM item in [`helix/docs/Issues.md`](helix/docs/Issues.md). Their absence is not silently treated as completed documentation.

This is a current-tree observation, not a claim that those files never existed in an older version.

## 25. Current source map for readers of this history

| Current path | Role |
|---|---|
| [`helix/core/`](helix/core/) | Format, parser, identity, cache, settings, source, storage, theme, and synchronization primitives. |
| [`helix/server/`](helix/server/) | Go/Gin service, ingest, catalogs, connectors, OPDS, KOSync, AI memory, and transport behavior. |
| [`app/`](app/) | Separate responsive React/Vite sample-data prototype. |
| [`helix/deploy/`](helix/deploy/) | Deployment configuration; not evidence of a completed runtime campaign. |
| [`helix/docs/`](helix/docs/) | Current guides, API/architecture notes, tracker database/exports, scalability review, and test-result index. |
| [`helix/research/`](helix/research/) | Revision-pinned static repository analyses and file-evidence ledgers. |
| [`helix/scripts/`](helix/scripts/) | Research, publication, registry, and deterministic/live verification programs. |
| [`helix/evidence/`](helix/evidence/) | Preserved machine artifacts, including failures as well as passes. |
| [`helix/spec/`](helix/spec/) | Product targets and specifications; not implementation proof. |
| [`helix/submodules/qa/`](helix/submodules/qa/) | HelixQA source and dependency declarations; incorporation not complete at the checkpoint. |
| [`helix/submodules/containers/`](helix/submodules/containers/) | Container resources; runtime qualification remains separate. |
| [`helix/CHANGELOG.md`](helix/CHANGELOG.md) | Dated M1 and hardening-round summary. |
| [`helix/CREDITS.md`](helix/CREDITS.md) | Current upstream attribution entry point. |

## 26. Final integrity statement

This history is as comprehensive as the accessible record permits, but it is **not a complete verbatim archive of every session**. Its central compromise is explicit: detailed reconstruction is supplied where dialogue is missing, while verbatim status is reserved for wording that actually survived. Current-codebase links were checked against the latest accessible tree, and known broken inherited documentation references are disclosed.

A genuinely complete verbatim edition would require exporting the original chat transcripts from the conversation platform and merging them with this document. Until then, no missing prompt or response has been fabricated to fill the gaps.
