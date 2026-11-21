# EVIDENCE.md — append-only verification log

Per project constitution §7.1 (NO BLUFF) and §11.4.2 (recorded evidence), every
"done" claim in this repository requires a recorded artifact. Entries are
appended and never rewritten; a superseded result stays visible with a pointer
to the entry that supersedes it. Machine output lives under `evidence/`.

---

## 2026-11-20T07:05Z — Campaign start, baseline re-verification

Host: Ubuntu 26.04, go1.26.8 linux/amd64, podman 5.7.0, PostgreSQL 16.15 +
pgvector 0.8.6 and Redis 7.4.11 in disposable loopback containers
(`helixqual-pg` 127.0.0.1:34320, `helixqual-redis` 127.0.0.1:34379, trust auth,
no credential exists to leak).

| Check | Result | Artifact |
|---|---|---|
| `core` go vet | PASS exit 0 | `evidence/campaign-2026-11-20/core/go-vet.log` |
| `core` go build | PASS exit 0 | `evidence/campaign-2026-11-20/core/go-build.log` |
| `core` go test ./... -count=1 | PASS, 22/22 packages | `evidence/campaign-2026-11-20/core/go-test.log` |
| `core` go test -race | PASS exit 0 | `evidence/campaign-2026-11-20/core/go-race.log` |
| `core` statement coverage | MEASURED 87.3% | `evidence/campaign-2026-11-20/core/coverage.out` |
| `server` go vet / build | PASS exit 0 | `evidence/campaign-2026-11-20/server/go-vet.log`, `go-build.log` |
| `server` go test (live services unset) | PASS 11 packages, live tests skipped | `evidence/campaign-2026-11-20/server/go-test.log` |
| `server` go test -race / coverage | PASS; coverage 60.1% without live services | `evidence/campaign-2026-11-20/server/go-race.log`, `coverage.out` |
| `server` go test -v with PostgreSQL + Redis | **FAIL: 124 pass, 5 fail, 0 skip** | `evidence/campaign-2026-11-20/server/go-test-live.log` |
| `server` full suite after fix, live services | PASS 11/11 packages, exit 0 | `evidence/campaign-2026-11-20/server/go-test-live-fixed.log` |

The 60.1% server coverage figure is recorded because the earlier tree carried no
live-service coverage number; `docs/testing.md` claimed a live PASS that could not
be reproduced here. See the defect entry below for why.

## 2026-11-20T07:05Z — DEFECT-001: catalog migration could not run against an empty schema

**Severity:** Critical. Blocks every first boot: a fresh PostgreSQL volume, a
new deployment, or a restore into an empty database. Directly gates ATM-024
(container qualification) and ATM-033 (final archives).

**Symptom (first observed run, preserved verbatim):**

```
postgres_live_test.go:20: catalog migration record 001_catalog.sql: ERROR: duplicate key
value violates unique constraint "helix_schema_migrations_pkey" (SQLSTATE 23505)
--- FAIL: TestLivePostgresCatalogRoundTrip (0.10s)
FAIL	github.com/helixreader/helix/server/internal/catalog
```

Failing set: `TestLivePostgresCatalogRoundTrip` (catalog) and
`TestLivePostgresFullScanReconcilesDeletedSourceFiles`,
`TestLivePostgresRescanRehomesChangedFileAndRemovesEmptyWork`,
`TestLivePostgresRescanRehomesLegacyCaseFoldedFileID`,
`TestLivePostgresDifferentBytesWithSameISBNShareEdition` (ingest).

**Root cause.** Ownership of one row was duplicated. `Migrate`
(`server/internal/catalog/postgres.go`) creates `helix_schema_migrations`, checks
`SELECT EXISTS(... WHERE version=$1)`, executes the `.sql` body, then inserts the
version row. `server/internal/catalog/migrations/001_catalog.sql:66` also
executed `INSERT INTO helix_schema_migrations(version) VALUES (1) ON CONFLICT DO
NOTHING`. On an empty schema the guard at line 74 sees no row, the migration body
inserts version 1 inside the same transaction, and the runner's insert then
violates the primary key. The advisory lock `731451` serializes migrators and is
held to transaction end, so this is not a race: the failure is deterministic on
every cold start.

**Why previous evidence said PASS.** The `EXISTS` guard short-circuits once the
row is present, so any already-migrated database silently skips the broken path.
Historical `evidence/server/final-go-live-services.log` was therefore produced
against a dirty, previously-migrated database. Per constitution §1.1 the suite
lacked false-positive immunity: no test ever migrated an empty schema.

**Fix.** `001_catalog.sql` no longer writes the bookkeeping table; the runner owns
it. `Migrate`'s insert is now
`INSERT INTO helix_schema_migrations(version) VALUES($1) ON CONFLICT (version) DO NOTHING`,
so a partially-applied transaction or a future baseline insert cannot abort a
migration.

**Regression coverage added** (`server/internal/catalog/postgres_coldstart_test.go`):

- `TestMigrateFromEmptySchema` — creates an isolated empty schema, migrates,
  asserts exactly one bookkeeping row and the full expected table set, and proves
  repeated `Migrate` calls stay a no-op.
- `TestMigrationFilesDoNotRecordOwnVersion` — static invariant over the embedded
  migrations, failing if any `.sql` inserts/updates/deletes the bookkeeping table.

**Needle proof (§1.1).** `TestMigrationFilesDoNotRecordOwnVersion` was run against
the defect reintroduced by re-appending the original line:

```
=== with defect ===
--- FAIL: TestMigrationFilesDoNotRecordOwnVersion (0.00s)
    postgres_coldstart_test.go:122: 001_catalog.sql writes the bookkeeping table
    (insert into helix_schema_migrations); Migrate owns that row
FAIL
=== restored ===
ok  	github.com/helixreader/helix/server/internal/catalog
```

`TestMigrateFromEmptySchema` alone does NOT detect a reintroduced self-insert,
because the runner's `ON CONFLICT` clause now absorbs it; that is precisely why
the invariant test exists. Both guards are required, and this note records which
one carries the detection power.

**Post-fix verification.** Cold start from a newly created empty database passed on
first attempt, warm re-run passed, bookkeeping held exactly one row (`versions=1`),
and the full server suite passed with live PostgreSQL and Redis: 11/11 packages,
exit 0 (`evidence/campaign-2026-11-20/server/go-test-live-fixed.log`).

## 2026-11-20T07:05Z — DEFECT-002: verifier scripts are pinned to a dead machine

Every script under `scripts/` and the reproduction block in `docs/testing.md`
hardcode `/mnt/agents/output/helix`, `/mnt/agents/research/*`, or
`/mnt/agents/output/app`. None exist on this host, so no documented reproduction
command runs and no release evidence can be regenerated. Also: on this host
PostgreSQL is reached over loopback TCP, not `/var/run/postgresql`, and
`redis-server` is not installed at all. Repair is in progress under
`evidence/campaign-2026-11-20/core/script-portability.md`; this entry stands as
the failure record regardless of that outcome.

## 2026-11-20T18:55Z — ATM-027: connection pools and durable scan records

Scope closed in this entry: configurable pool limits, a distinct AI-memory
migration lock, and scan-job durability across restart. Keyset pagination,
advisory-lock granularity, and incremental catalog writes remain open; the item
stays In progress.

**Connection pools.** `catalog` and `aimemory` previously hardcoded 20/5/30m and
10/3/30m, so a deployment could not coordinate its database budget with
`max_connections`. Both now take `WithPool(...)` options, and
`HELIX_CATALOG_POOL_*` / `HELIX_AI_MEMORY_POOL_*` environment variables drive
them. Defaults are the previous hardcoded values, so behavior is unchanged when
unset. Invalid combinations are rejected at startup rather than clamped.

**Migration lock.** `aimemory` migrated under advisory lock `731452`, which is
the catalog write lock, so an unrelated AI-memory migration serialized behind
catalog traffic. It now uses `731453`.

| Check | Result | Artifact |
|---|---|---|
| `go vet` config/catalog/aimemory/httpapi | PASS exit 0 | `evidence/campaign-2026-11-20/server/` |
| `TestPoolConfiguration` (8 subtests) | PASS | `go test ./internal/config -count=1` |
| `TestWithPoolValidation` / `TestWithPoolAppliesLimits` | PASS | `go test ./internal/catalog -count=1` |
| `scripts/verify_live_server.py` (real binary, live PostgreSQL + Redis) | PASS 76/76 | `evidence/live-server/results.json` |
| `TestJournal*` (7 tests, miniredis) | PASS | `evidence/campaign-2026-11-20/jobs/journal-unit.txt` |
| `go test -race ./internal/httpapi` | PASS | rerun of the same suite under the race detector |

**Scan-record durability.** Terminal job records are mirrored to Redis
(`helix:scan:job/<id>` plus a bounded index keyed by creation time) with a key
TTL equal to the in-memory retention window, so the journal cannot resurrect a
record the store had already forgotten. At startup `RestoreScans` repopulates the
store; a record left `queued` or `running` becomes `failed` with
`scan interrupted by server restart` rather than a permanent in-progress entry or
a missing identifier. Restore never re-runs work, and source registrations remain
process-local, which is documented rather than changed here.

Journal writes are deliberately best-effort: an unreachable Redis logs a warning
and the scan still completes. `TestJournalFailureDoesNotFailScans` pins that
boundary, because silently failing ingestion to protect an audit trail would be
the worse trade.

**Needle proof (§1.1).** `api.UseScanJournal(scanClient)` was removed from
`server/cmd/helix-server/main.go`, the tree rebuilt, and the live verifier rerun:

```
{"status": "FAIL", "pass": 26, "fail": 1}
FAIL  pass 26 fail 1
[{'name': 'completed scan job survives server restart', 'status': 'FAIL'}]
```

Recorded in `evidence/campaign-2026-11-20/jobs/live-server-needle-no-journal.json`.
`main.go` was then restored byte-for-byte (`diff` clean) and the verifier returned
39/39. The first version of the second restart assertion only checked that the job
list was non-null, which passes with or without the journal; it was rewritten to
require the restored job's ID in the list, so both live checks carry detection
power.

That needle run also left `evidence/live-server/results.json` in the tree reporting
`FAIL 26/1`, which was committed and read as a red gate. It was not one: the
failure was the injected defect, recorded by the run that proved detection. A
needle artifact must be copied to a named file under `evidence/` before the
mutated tree is restored, or `results.json` at a stable path will outlive the
experiment that produced it.

## 2026-11-20T17:46Z — ATM-024: containerized build/deploy/restart/upgrade/rollback qualified (podman)

First-ever runtime deployment evidence for the compose stack (closes the gap
recorded at `docs/scalability-review.md:27`). Executed on podman 5.7.0 +
podman-compose 1.5.0 rootless; `docker` is not installed on this host, so
docker-compose parity is asserted only at file semantics, not execution.

All seven stages PASS. Artifacts (sanitized, machine-checked for credentials):
`evidence/campaign-2026-11-20/containers/` — `build.log`, `cold-start.log`,
`restart.log`, `upgrade.log`, `rollback.log`, `results.json`, `summary.md`.

Key verified facts:

- Cold start on fresh volumes reached `healthy` in ~8s with zero restarts;
  **DEFECT-001 does not recur inside the container**: `helix_schema_migrations`
  holds exactly one `version=1` row against an empty schema (image carries the
  fixed `001_catalog.sql`).
- Real endpoints answered: `/healthz`, `/readyz`, admin Bearer source create
  (201) → local-protocol scan (`/library` bind mount) → succeeded → works read
  (API + PostgreSQL), OPDS 1.2 Atom and OPDS 2.0 `application/opds+json` 200,
  source delete 204, unauthenticated 401.
- `podman restart` of the server container preserved catalog data (same work
  id, no rescan, no re-migration). Upgrade to rebuilt image v2
  (`revision=v2-upgrade`) and rollback to v1 (`revision=v1-coldstart`) both
  verified via running image id + health + data survival. Teardown left zero
  residual containers/volumes/networks; live services `helixqual-pg` and
  `helixqual-redis` were never touched (isolation prefix `helixqual-deploy_`,
  loopback-only port 127.0.0.1:39080).
- **Build-blocking defect found:** first `podman build` failed on
  `server/internal/httpapi/jobs.go:74: undefined: slog` (tree compile break,
  not a deploy defect); the owning agent's import fix landed immediately and
  the rebuild passed. Timeline preserved in `build.log`.
- **podman-compose divergences recorded (not assumed):** `up -d` recreates all
  services on any change (docker-compose recreates only the changed one);
  top-level `name:` overrides `-p`; `config` prints resolved secrets;
  `down --rmi local` leaves localhost-tagged images.
- Deploy-tree improvements: compose now has an interpolable project name, an
  explicit `${HELIX_IMAGE}` reference for tag-driven upgrade/rollback,
  `${HELIX_PUBLISH_PORT}` loopback publishing (default 8080 unchanged), and a
  `/healthz` healthcheck for the server; `server/Dockerfile` stamps
  `org.opencontainers.image.revision`; `.gitignore` re-includes
  `evidence/**/*.log`.

## 2026-11-20T11:40Z — ATM-032: honest per-format fulfillment matrix for the Go core

Produced `docs/formats.md`: one row per format, five independent stages
(detect / parse / paginate / render / convert), provenance per PASS cell,
explicit NOT IMPLEMENTED cells. Render is NOT IMPLEMENTED for every format
in the core; detection is extension-only (no magic-byte sniffing).

Gaps closed with real implementations, table-driven exact-value tests and
fail-closed limit tests (no new dependencies): `sync.Paginate`/`PageOf`
(HDM pagination in `sync.Position` coordinates, `Furthest`-compatible),
EPUB 3 exporter (`convert.FormatEPUB`, byte-deterministic, refuses lossy
payloads instead of dropping them) making TXT/FB2/MOBI→EPUB matrix rows
executable end-to-end (txt→fb2→epub→txt and mobi→epub chains assert exact
text), CBZ page packaging exporter + directory-only page enumeration
(`cbz.Pages`, proven non-inflating via corrupted-central-directory test),
EPUB spine page extraction (`epub.Spine`, MaxPages/limits enforced), FB2
`<binary>` payloads and image references now reach HDM resources with
per-entry/aggregate limits, MOBI `</p>`/`<br>` paragraph breaks fixed,
EPUB `<hr>` → separator blocks.

Fuzzing (10 targets, 8s live each + seed corpora, see
`evidence/campaign-2026-11-20/formats/fuzz.log`) found one real pre-existing
panic: unclosed nested FB2 `<section>` produced a nil chapter that crashed
`hdm.Validate` (`pkg/parse/fb2/testdata/fuzz/FuzzParse/4a3dc8cee8c1c385`);
fixed in fb2 and hardened in `pkg/hdm` with named regression tests.

Bluffs removed, per constitution §7.1: same-format `Direct` was claimed for
all registry formats incl. PDF/AZW3/DOCX with no parser or writer (now only
EPUB/FB2/TXT/CBZ, guarded by `TestSameFormatRepackIsHonest` and
`TestMatrixNeverOutrunsTheCode`); `ContentAI=true` on all parsed formats
removed (`TestNoDescriptorClaimsUnimplementedAI`).

Verification (`evidence/campaign-2026-11-20/formats/verify.log`, go1.27.3
toolchain): `go vet ./...` clean, `go build ./...` clean,
`go test ./... -count=1` 22/22 ok, `go test -race ./... -count=1` 22/22 ok;
statement coverage 87.29% → 87.32% (340/389 new statements covered).
Matrix summary and full test inventory:
`evidence/campaign-2026-11-20/formats/matrix.md`.

## 2026-11-20T20:05Z — ATM-027 phase 1: pagination cursor, first large-catalog measurements, and DEFECT-003

**What landed.** `catalog.Cursor` (`updated_at`+`id`, opaque base64url) threaded
additively through `Query.After` → `GET /api/works?after=` → OPDS 1.2 `rel="next"`
and OPDS 2.0 continuation. Offset stays supported; pages read `limit+1` rows so
`HasNext` does not depend on the count. A malformed `after` is a 400, found by
testing rather than assumed: `?after=%` had silently returned page 1 with a 200,
because `url.Values` parsing swallows a bare percent. The router now parses
`RawQuery` directly.

**New tests.** `internal/catalog/cursor_test.go`, `internal/httpapi/cursor_test.go`,
OPDS continuation assertions; `TestLivePostgresCursorWalkMatchesOffset` walks the
catalog both ways at three page sizes and requires identical sequences.

**Needle proofs (§1.1).**

- Removing the keyset predicate from `Postgres.Search` → `--- FAIL:
  TestLivePostgresCursorWalkMatchesOffset … walk did not terminate`. Restored → PASS.
- Parsing `after` but not applying it in the router → `--- FAIL:
  TestWorksCursorWalkOverHTTP … did not terminate after 7 pages`. `router.go`
  restored byte-for-byte.
- The first attempt at the first proof printed `ok … 0.005s` and **proved nothing**:
  without `HELIX_TEST_POSTGRES_DSN` the live test skips, and a skip is indistinguishable
  from a pass at that log level. Every needle recorded here was re-run with the test
  named in `-v` output so its `=== RUN` line is visible in the log.

### DEFECT-003: the scale assertions were not immune to their own environment

Two separate false-result mechanisms in `internal/catalog/catalog_scale_test.go`,
both found while turning code-derived claims into measurements.

1. **Transport contamination.** Host-side latency through the rootless podman
   forwarder (`pasta`) reported ~205 ms for statements the server executed in 3–5 ms,
   intermittently and per connection, and made the batched page loader look slower
   than the N+1 it replaced. Any conclusion drawn from that path inverts.
2. **Nondeterministic plan assertion.** The test asserted the browse page does not
   plan a `Seq Scan on works`. The index `works_updated_id` exists and matches the
   `ORDER BY`, but immediately after a bulk load no heap page is all-visible, so
   PostgreSQL prices an index-only scan as expensive and sorts the table instead
   (external merge, 1,984 kB to disk). The assertion passed or failed depending on
   whether autovacuum had visited, which is a test defect, not a schema defect.

**Fix.** All scale figures are now taken inside the database container, and
`TestLiveScaleCatalog` records both plans deliberately: `EXPLAIN` before
`VACUUM (ANALYZE)`, then after, then asserts on the after-plan. The pre-VACUUM plan
is kept in the artifact rather than discarded — it is the plan a client gets right
after a large ingest, and that is an operational fact worth recording
(`docs/scalability-review.md`). Both tests now run the same six-shape
`transportProbe`, comparing client wall time against the server's `EXPLAIN
(ANALYZE)` account of the identical statement; the phase test `Skip`s the ratio
assertions when shapes disagree, and the scale test writes the verdict into its
payload as `measurement_path`.

**Measurement results (50,000 works / 100,000 editions / 200,000 identifiers /
300,000 files, in-container, medians of 7–12 iterations):**
`evidence/campaign-2026-11-20/scalability/` — `in-container-scale.log`,
`in-container-phase.log`, `scale-plans.json`, `keyset-predicate-plan.txt`,
`host-under-load-scale.log`, `host-through-pasta-under-load.log`.

| Result | Value |
|---|---|
| Batched page load vs one call per work ×100 | 21.2 ms vs 76.3 ms (3.6×) |
| 200-value `IN` vs `= ANY($1)` | 3.29 ms vs 3.12 ms (shape is not the cost) |
| Page query, pre-VACUUM vs post-VACUUM | 27.29 ms (seq scan + external sort) vs 6.99 ms (index-only, `Heap Fetches: 0`) |
| `Search` deep page by offset vs by cursor | 11.61 ms vs 11.27 ms |
| Raw id query at depth 40,000, offset vs cursor | 3.59 ms vs 4.12 ms |
| `count(*)` of the fixture, run on every `Search` | 6.73 ms |
| `%text%` search | 773.70 ms |

**The cursor is not a latency win, and that is recorded rather than dressed up.** Its
predicate must be an `OR` because the ordering mixes directions, so PostgreSQL
applies it as a filter and discards 40,001 index entries
(`keyset-predicate-plan.txt`) — the same work `OFFSET` does. It is kept for
traversal stability, and the plan that would make it seek (single-direction
tiebreak) is written down in `docs/scalability-review.md` with its cost and its
non-benefit at this size, and not taken.

| Check | Result |
|---|---|
| `go vet ./internal/catalog ./internal/httpapi ./internal/opds` | PASS exit 0 |
| `go build ./...` | PASS |
| `go test ./... -count=1` with live PostgreSQL + Redis | see `evidence/campaign-2026-11-20/scalability/` run below |
| `TestLiveScaleCatalog` in-container | PASS 34.3 s, `measurement_path=clean` |
| `TestLiveScaleBatchPhases` in-container | PASS, 6/6 shapes `transport=clean` |
| Same two tests on host under CPU load | phase `SKIP` (gate fired), plan assertions unaffected |

## 2026-11-20 (evening) — pool configuration proven, one vacuous test retired, one file lost and recovered

`evidence/campaign-2026-11-20/pool-needles/` (`README.md`, `needle-results.json`,
`live-backend-counts.json`, plus the two scripts that produced them).

The scalability review had recorded pool caps as "done and validated" on the strength
of a test that turned out to have no detection power: it asserted only that
`NewPostgres` failed, against a DSN that cannot dial, so the dial answered for the
validation. Mutating the check away left it green (recorded as `N0`). The checks now
live in `poolSettings.validate()` in both `catalog` and `aimemory` and are called
directly; the constructor path is covered too, but by asserting *which* error appears.
`aimemory.WithPool` previously had no test at all.

Six needles (`N1`–`N6`), each reverting from a copy and re-running green: all killed
their tests. `N6` disables only the idle-above-open cross-check and fails exactly one
named subtest, which is what distinguishes a real dependency from a general error
expectation.

End-to-end, with the shipped binary and the 50,000-work fixture loaded: twelve
concurrent unanchored text searches against `MAX_OPEN_CONNS=3` produced a peak of
**3** PostgreSQL backends, every sample reading 3; with the default they produced
**12**. Invalid pool environment refuses startup at the process boundary with exit 2
before a listener is bound. So `HELIX_CATALOG_POOL_*` demonstrably reaches the live
pool, not merely the parsed `Config`.

Incident, recorded rather than quietly absorbed: during the `N0` needle,
`git checkout -- server/internal/catalog/postgres.go` replaced that file with its
HEAD version. HEAD predates Phases 0–1 — this tree carries the campaign uncommitted —
so the checkout deleted the pool options, the keyset predicate, the limit+1 `HasNext`
read, the batched `getWorks` call, and the `ON CONFLICT` migration bookkeeping. The
file was recovered from `/tmp/pg.bak`, a copy the previous session had taken at 19:29
as part of its own needle run, and verified by marker grep, `gofmt`, `go build`,
`go vet`, and a full live catalog suite pass (57.3 s) that only the Phase 1 version
can pass. The rule taken from it: no `git checkout`/`restore`/`stash`/`clean` in this
repository while the campaign is uncommitted; pre-mutation state is held in `cp`
copies. Full account in that evidence directory.

Also re-ran `TestLiveScaleCatalog` on the host to re-seed the fixture (33.1 s, PASS,
retained via `HELIX_SCALE_KEEP=1`; log `scalability/host-reseed-2026-11-20T20.log`).
It self-labelled `measurement_path=clean` from the host this time. That does not
upgrade the host figures: the stall is intermittent and per-connection, and the
label's own caveat in `docs/scalability-review.md` says so. The in-container logs
remain the authoritative artifacts.

## 2026-11-20 (evening) — DEFECT-002 closed, DEFECT-004 found: the registry test file never existed here

`evidence/campaign-2026-11-20/registry/` (`README.md`, `verifier-2026-11-20.log`;
machine output `evidence/workable-items/results.json`, 11/11 PASS, exit 0).

DEFECT-002's remaining gap is closed: no `py_compile` had ever been run on the
verifier scripts a subagent had rewritten. All 10 modules in `scripts/` and `tests/`
compile under Python 3.14, and the verifier now writes to this checkout's `evidence/`
instead of the dead machine's `/mnt/agents/output/helix/...`.

Compiling them is what exposed **DEFECT-004**: `verify_workable_items.py` copies
`tests/test_workable_items.py` into a scratch root to drive its four mutation gates,
and that file is absent — not from the working tree only, but from every commit
(`git ls-tree -r HEAD` has no `tests/`). It ran on 2026-10-28 (`evidence/workable-items/unit.log`,
12 tests) and was never committed, so registry qualification has been un-runnable on
this machine since the migration, while its PASS artifact pointed at paths that do not
resolve here. The suite was reconstructed from its contract — the 12 names in
`unit.log`, the 4 methods the gates invoke, and the assertion shapes still visible in
the gate details of `final-verifier.log` — and now runs 12 tests green, with all four
mutants rejected by their named tests.

Five of my own first-pass assertions were wrong about the real API and were corrected
against observed behavior instead of defended: pre-export `diff()` reports every file
as drift; `render_item` does not escape pipes because long-form bodies are not table
cells (only `summary_rows` does); `utcnow()` is second-resolution; deleting a middle
row is *meant* to surface as `ATM allocation contains gaps`; and identifiers are never
reused even once a gap exists.

A third problem came from running the export itself: `docs/*.pdf` were from 2026-10-28
while the Markdown and HTML were regenerated today, and `diff()` cannot see it because
it compares Markdown and HTML only. ReportLab was installed out of tree
(`pip install --target ~/.cache/helix-docs-deps`, no PEP 668 override), the document
set regenerated and confirmed byte-idempotent, and a `pdf-documents-current` gate was
added — size, page count, and the `pdftotext` ATM identifier set must match each PDF's
Markdown sibling. Needle on the retained October artifacts: FAIL, naming
`only in PDF: ['ATM-032']` (ATM-032 was still open when they were generated); PASS on
the regenerated set. The gate requires poppler and records FAIL without it rather than
skipping, because a silently no-opping currency check is the failure mode that produced
this entry.

## 2026-11-20T21:40Z — ATM-027: catalog identity lock scoped per work, edition and source

Continues the 18:55Z entry, which recorded advisory-lock granularity as still
open. It is now closed; keyset pagination and incremental catalog writes were
already recorded there or in the scale entries. The item stays In progress: no
service-level objective has been ratified and no multi-process topology has been
measured, so its acceptance text is not satisfiable by this entry.

**The change.** `UpsertWork`, `MergeWork` and `ReconcileSource` each held
`pg_advisory_xact_lock(731452)` — one key for the entire catalog, so any scan of
any source waited behind every other write. Each now takes one
transaction-scoped lock per identity it can touch: `w:<work>` for the edition set
`upsertTx` rewrites, `e:<edition>` for the ownership check, `s:<source>` for file
rows keyed by `(source_id, path)` and for the reconcile prune. Every key comes
from the incoming work before the transaction opens, and advisory locks are taken
before row locks in every path.

**Measured** (`evidence/campaign-2026-11-20/load/`, in-container DSN, one run, the
identical transaction body in all arms): global key 271.94 ops/s, sharded
1592.21 ops/s, unlocked ceiling 1789.00 ops/s — **5.85x**, at **89%** of the
ceiling. The pre-existing contention harness rerun against the new code reports
concurrent `MergeWork` at **5.14x** sequential (previously 0.98x) and
`ReconcileSource` at 2.73x concurrent-over-sequential, with zero errors and zero
advisory waiters sampled. Artifacts: `catalog_lock_scope.jsonl`,
`catalog_contention_sharded.jsonl`; the older serialized `catalog_contention` run
is retained rather than overwritten.

**Evidence that the safety claims are not decorative**
(`evidence/campaign-2026-11-20/lock-scope/`): every invariant test is paired with
a control running the same body under a different locking discipline, and with a
needle. With locks removed, eight concurrent merges of one work lose 7 of 8
editions **and report no error** — the failure mode this lock exists to prevent. A
contested file claim without the source key produced 6 writer errors in 6 of 6
rounds where the sharded path produced none. `run-mutation-gate.sh` then deletes
one key class from the real source at a time and requires the dependent test to
fail: drop-work-key, drop-source-key, drop-edition-key and drop-all-locks were each
detected, the file was restored byte-identically (`sha256_verified=yes`), and the
restored suite passes.

**Deliberate limits.** `23505` unique violations are not retried: they would mean
the key set missed two writers sharing a row, and retrying that would convert a
lock-scope bug into a scan that looks like it succeeded. Cross-source writers of
the *same bytes* still serialize, by design, because work identity is
content-anchored. The sharded locks use the two-argument advisory form and share
no key with the old single-argument `731452`, so two server versions of different
lock discipline must not write one catalog simultaneously; the shipped topology is
one image upgraded by tag. Deadlock between writers whose edition sets interleave
in opposite orders is handled by the retry rather than prevented by ordering; none
was reported in any run here, which is a weaker statement than none occurred.

## 2026-11-21T10:57Z — ATM-031: OPDS-PSE page streams qualified against a live server

`scripts/verify_live_server.py` now ingests a second, purpose-built library of
five files and asserts what the page-stream routes promise about each:

| Fixture | Why it is in the set | Live verdict |
|---|---|---|
| `live-server-pse.cbz` | three PNG pages, one honest type | advertised with `pse:count="3"`, `type="image/png"`, `href` carrying `{pageNumber}`; all three pages fetched byte-identical to the archive members |
| `live-server-mixed.cbz` | one PNG page and one JPEG page | no stream link; page requests refused; the whole file still ships |
| `live-server-forge.cbz` | a `cover.png` whose bytes are text | no stream link — the extension is not proof |
| `live-server-cbr-stub.cbr` | a container the format registry calls streamable but this server cannot enumerate | no stream link; page requests 404 |
| `live-server-plain.txt` | not a comic container at all | no stream link |

Each fixture is matched by sha256 rather than by title, so a catalog record is
credited only if it describes the exact bytes the harness wrote. ETags are
compared against the catalogued digest and the page index, and the entry-level
endpoint `/opds/1.2/editions/{id}` is read instead of a searchable feed, so a
missing link cannot be blamed on pagination or tokenization.

The advertisement is then consumed by `server/cmd/helix-pse-check`, a client built
from the same tree that parses the feed with `core/pkg/pse` — the extension's own
rules — fetches every promised page, and checks each delivery against what the
link said: media type, image signature, distinct ETag, `Accept-Ranges: none`,
`Cache-Control`, `Vary: Authorization`, a 404 stating `0..N-1` one page past the
count, and a 401 when anonymous. `TestVerifyPage*` in that command keeps the
checker itself honest.

**Needle proof (§1.1).** Two defects were injected into
`server/internal/httpapi/pse.go` one at a time, each run reproduced, and the file
restored from a `cp` copy:

| Injection | Predicted rejection | Observed |
|---|---|---|
| `c.Header("Accept-Ranges", "none")` deleted | the client rejects a rendered page that advertises byte ranges | `FAIL 50/1` — `OPDS-PSE client round trip: ['page 0 offers Accept-Ranges ""; a rendered page is not a byte range of the archive', …]`. Fifty assertions pass first, including every producer-side check, so the rejection came from the client and not from a shared assumption. |
| `Count: len(refs)` → `len(refs) + 1` | a feed must not promise a page the archive does not contain | `FAIL 23/1` — `pse:count equals the pages in the archive` |
| `Vary` check in the checker replaced by `false &&` | the checker's own suite must notice a weakened assertion | `go test ./cmd/helix-pse-check` → `--- FAIL: TestVerifyPageRejectsEveryBrokenDelivery/shared_cache` |

Artifacts: `evidence/live-server/results.json`,
`evidence/live-server/pse-client.json` (before restart) and
`evidence/live-server/pse-client-after-restart.json` (after a process restart and
source re-registration, so the advertisement is a property of the bytes and not of
state the first process held open).
