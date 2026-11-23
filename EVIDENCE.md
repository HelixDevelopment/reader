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
new deployment, or a restore into an empty database. Directly gates HRD-024
(container qualification) and HRD-033 (final archives).

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

## 2026-11-20T18:55Z — HRD-027: connection pools and durable scan records

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

## 2026-11-20T17:46Z — HRD-024: containerized build/deploy/restart/upgrade/rollback qualified (podman)

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

## 2026-11-20T11:40Z — HRD-032: honest per-format fulfillment matrix for the Go core

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

## 2026-11-20T20:05Z — HRD-027 phase 1: pagination cursor, first large-catalog measurements, and DEFECT-003

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
row is *meant* to surface as `HRD allocation contains gaps`; and identifiers are never
reused even once a gap exists.

A third problem came from running the export itself: `docs/*.pdf` were from 2026-10-28
while the Markdown and HTML were regenerated today, and `diff()` cannot see it because
it compares Markdown and HTML only. ReportLab was installed out of tree
(`pip install --target ~/.cache/helix-docs-deps`, no PEP 668 override), the document
set regenerated and confirmed byte-idempotent, and a `pdf-documents-current` gate was
added — size, page count, and the `pdftotext` HRD identifier set must match each PDF's
Markdown sibling. Needle on the retained October artifacts: FAIL, naming
`only in PDF: ['HRD-032']` (HRD-032 was still open when they were generated); PASS on
the regenerated set. The gate requires poppler and records FAIL without it rather than
skipping, because a silently no-opping currency check is the failure mode that produced
this entry.

## 2026-11-20T21:40Z — HRD-027: catalog identity lock scoped per work, edition and source

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

## 2026-11-21T10:57Z — HRD-031: OPDS-PSE page streams qualified against a live server

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

---

## 2026-11-23T18:02Z — HRD-027 continuation: multi-instance contention on one Redis

Continues the gap the queue evidence names —
`evidence/campaign-2026-11-20/queue/SUMMARY.md`: "Still not established:
multi-instance contention on one Redis (two `helix-server` processes sharing a
journal)". Live verification: TWO queue consumers against ONE real Redis —
`helixqual-redis`, DSN `redis://127.0.0.1:34379` (Redis 7.4.11),
`HELIX_TEST_REDIS_ADDR=127.0.0.1:34379`.

**Topology.** Two `jobStore` instances — the queue layer of two application
instances — each with its own `redisScanJournal` client (separate connections).
Concurrent producers (8 per phase; the same total, `HELIX_MI_TOTAL=384`, in both
phases) create jobs through either instance; each instance runs a claim loop that
claims non-terminal jobs from the shared journal (`SETNX helix:scan:claim:<id>`,
owner instance id, 30 s TTL) and executes them through its own store
(`jobStore.run`), capped at `maxConcurrentScans=2` per instance (claim-slot gate +
the store's own worker sem). A single-instance baseline is measured in the same
run.

**Measured** (`multi_instance.json`; p50/p95 per job created→terminal):

| Phase | Submitted | Accepted | Executed | Lost | Double | Peak/instance | Throughput | p50 / p95 ms |
|---|---|---|---|---|---|---|---|---|
| single_instance_baseline | 384 | 64 | 64 | 0 | 0 | 2 | 76.5/s | 413.2 / 800.9 |
| two_instances | 384 | 127 | 127 | 0 | 0 | 2 / 2 | 138.5/s | 401.2 / 868.9 |

Admission rejections: single phase 320/384 queue-full, multi phase 257/384.
Producer rates: 6,138/s and 10,036/s. Claim latency (created→fn-entry):
392.4/780.5 ms single, 380.7/848.3 ms two-instance.

**Assertions — all PASS:**

- **No double execution:** every accepted job executed at most once across both
  instances (`double_executed=0`; execution ledger count per job ID).
- **No lost jobs:** every accepted job either executed or recorded terminal —
  `lost=0`, `failed_terminal_unexecuted=0`.
- **No unhandled errors:** no panic escaping either claim loop, no
  `scan journal write failed` warnings observed (slog capture active).
- **Admission caps hold per consumer:** peak concurrent scans per instance = 2 =
  `maxConcurrentScans`, never above.
- **Multi-instance throughput ≥ single-instance baseline:** 138.5/s ≥ 76.5/s —
  ratio 1.81×. Measured overhead vs the theoretical 2× (two instances, four
  workers): about 10% — per-instance the two-instance run shares
  ~0.91× the single baseline rate per accepted job; latency p50s are comparable
  (401.2 vs 413.2 ms), so contention is minimal on this shape.

**Recorded for HRD-027:** DSN, instance count, producer/consumer rates, total
jobs, executed/failed/lost, p50/p95 latency, assertions —
`evidence/campaign-2026-11-20/multi-instance/multi_instance.json`; run output
`run.log`.

**Server vet/build/test results recorded** (tree has pre-existing uncommitted
drift — see gaps):

| Check | Result | Artifact |
|---|---|---|
| `go vet ./...` (server) | FAIL exit 1 — pre-existing drift: `internal/catalog/slow_client_live_test.go:522,533` type errors (float64 vs int64, in the untracked new file) | `go-vet.log` |
| `go vet ./internal/httpapi/` (server) | PASS exit 0 (includes the new harness) | `go-vet-httpapi.log` |
| `go build ./...` (server) | PASS exit 0 | `go-build.log` |
| `go test ./...` (server, `-count=1`) | exit 1 — `internal/catalog` [build failed] (same pre-existing drift); `internal/httpapi` ok (4.447s); other packages ok | `go-test.log` |

**Gaps recorded (not claimed):**

- Two full `helix-server` HTTP processes were not run; the two-instance
  topology measured is the queue layer (`jobStore` + `redisScanJournal` + claim
  loops) — the layer the gap names.
- The baseline is the single-instance phase of the same run. The tree has
  pre-existing uncommitted drift — untracked files (`load_concurrent_test.go`,
  `load_remote_rclone_test.go`, `load_remote_webdav_test.go`, the slow-client
  file) and modified files — which fails vet/build/test of `internal/catalog`
  today and is not introduced by this continuation; it is recorded, not fixed.
- No service-level objective is claimed; p50/p95 are campaign figures.
- The HRD-027 item itself remains open in `docs/workable_items.db` (status not
  changed by this continuation).

---

## 2026-11-23T19:15Z — HRD-027 (continuation): set-based edition write on the catalog upsert path

Continues the 2026-11-20T21:40Z HRD-027 entry (the lock-scope half). The
delete/reinsert write path that entry noted as half done is now set-based; this
entry records the change and the verification.

**The change** (`server/internal/catalog/postgres.go`, ~312 lines changed):

- `deleteWorkEditions` — one statement, `DELETE FROM editions WHERE work_id=$1
  RETURNING id` — deletes the work's whole old edition set and returns the
  affected ids in bulk.
- `checkEditionOwners` — set-based ownership check: after the delete removed
  every row this work carried, a row still present for one of the incoming
  edition ids belongs to another work; the loser gets the same "already
  belongs to work" error the per-edition check produced.
- `insertEditions` — one multi-row `INSERT INTO editions(…) VALUES … ON
  CONFLICT (id) DO NOTHING` for the new edition set.
- `claimFiles` — one set-based claim: `DELETE FROM files WHERE id = ANY($1) OR
  (source_id, path) IN (VALUES …) RETURNING edition_id` (the re-homing form the
  per-file loop used to express as `id=$1 OR (source_id=$2 AND path=$3)`).
- `insertFiles` — one multi-row `INSERT INTO files(…) VALUES …` (plain: any
  conflict reaching it is an unclaimed duplicate the caller must see, matching
  the loud failure surface the old loop had).
- `pruneRehomed` — set-based: `DELETE FROM editions e WHERE e.id = ANY($1) AND
  NOT EXISTS (SELECT 1 FROM files f WHERE f.edition_id=e.id) RETURNING
  work_id`, then `DELETE FROM works w WHERE w.id = ANY($1) AND NOT EXISTS
  (SELECT 1 FROM editions e WHERE e.work_id=w.id)`.

The per-edition delete/reinsert loop and the per-file claim/reinsert loop are
gone; the identifiers and evidence inserts are unchanged.

**New test** — `TestLivePostgresUpsertReplacesEditionSet`
(`server/internal/catalog/postgres_live_test.go`): seeds a work with two
editions (with files, identifiers and evidence), upserts a different single
edition set, asserts the stored set is exactly the new one (stale editions,
files, identifiers and evidence gone; the new ones present), and asserts a
repeat upsert is idempotent.

**Measured** — DSN `postgres://helixqual@127.0.0.1:34320/helix_test?sslmode=disable`
(helixqual-pg), one run per row:

| Check | Result | Artifact |
|---|---|---|
| `go build ./...` (server) | PASS exit 0 | `evidence/campaign-2026-11-20/upsert-setbased/go-build.log` |
| `go vet ./internal/catalog/` (server) | PASS exit 0 | `evidence/campaign-2026-11-20/upsert-setbased/go-vet-catalog.log` |
| `go test ./internal/catalog/ -count=1` (unit, no DSN) | PASS exit 0 (`ok` 0.006 s) | run in the same pass as the live run below |
| `go test ./internal/catalog/ -count=1 -v` (live, DSN) — final, plain-insert write path | PASS exit 0. `TestLivePostgresUpsertReplacesEditionSet` PASS (0.05 s); lock-scope invariants unchanged: `TestCatalogMergePreservesEveryWriterUnderIdentityLocks` PASS, `TestCatalogMergeNeedleUnlockedLosesWriters` PASS, `TestCatalogFileClaimByDistinctWorksKeepsOneRow` PASS, `TestCatalogFileClaimNeedleUnlockedDuplicatesRow` PASS, `TestCatalogEditionOwnershipIsDecidedOnce` PASS, `TestCatalogReconcileExcludesSameSourceMerge` PASS; `TestLivePostgresCatalogRoundTrip` PASS, `TestLivePostgresBatchLoadMatchesRowByRow` PASS | `evidence/campaign-2026-11-20/upsert-setbased/catalog-live-test-final.log` |
| `go vet ./...` (server) | PASS exit 0, recorded 18:54Z on the same tree state | `evidence/campaign-2026-11-20/upsert-setbased/go-vet.log` |
| `go test ./... -count=1` (server, live, DSN) — earlier, same tree state | PASS 12/12 packages, exit 0 | `evidence/campaign-2026-11-20/upsert-setbased/server-go-test-live.log` |
| `go test ./... -count=1` (server, live, DSN) — final | exit 1: `internal/catalog` ok (58.5 s) but `internal/httpapi` [build failed] from an untracked concurrent WIP file (gaps below) | `evidence/campaign-2026-11-20/upsert-setbased/server-go-test-live-final.log` |
| Mutation gate (`evidence/campaign-2026-11-20/lock-scope/run-mutation-gate.sh`) | PASS exit 0 — drop-work-key detected, drop-source-key detected, drop-edition-key detected, drop-all-locks detected; `gate_status=0 restored_tests=pass sha256_verified=yes`; `postgres.go` sha identical before/after | `evidence/campaign-2026-11-20/upsert-setbased/mutation-gate-run.log`; appended to `evidence/campaign-2026-11-20/lock-scope/mutation-gate.log` |

An intermediate DO-NOTHULL variant of the files insert was also live-tested
(`evidence/campaign-2026-11-20/upsert-setbased/catalog-live-test.log`) before
the plain-insert change; the final run above is the authoritative one for the
delivered code.

**Concurrency note (another session's files).** A concurrent session added
untracked `server/internal/catalog/slow_client_live_test.go` (mtime 18:58:59;
not in this change's diff; restored after the runs, sha256 `2358918b…` verified
identical before and after) and untracked
`server/internal/httpapi/slow_client_live_test.go`, which breaks compilation of
`internal/httpapi` with float64/int64 mismatches. For the final validation
runs the catalog WIP was briefly moved aside (backup in `/tmp`); it was not
edited, only moved and restored.

**Gaps / not done.**

- The mutation gate was not run in the first attempt (the concurrent WIP
  broke compilation of the catalog package); it was run afterwards — results
  above. Phase-4 gate results remain in
  `evidence/campaign-2026-11-20/lock-scope/mutation-gate.log`.
- The set-based write path was not re-measured under load; the gate re-run is
  the measurement-adjacent evidence for it.
- The review doc needs a follow-up line: `docs/scalability-review.md` line 28
  still lists "Upsert still deletes and reinserts the editions of one work" as
  an open finding, and line 42 marks the write path "**Half done** — … the
  delete/reinsert write path is unchanged". The write path is now set-based —
  suggested follow-up: update line 42 to record that the write path landed,
  and adjust line 28's finding row to drop the "upsert still
  delete/reinserts" clause. Not edited here — the review doc is concurrent
  drift on this tree.
- Load-gated harnesses (`TestCatalogIdentityLockScopeThroughput`,
  `TestCatalogMultiProcessMerges`, `TestLiveLoadWriteContention`) SKIP without
  `HELIX_LOAD_RUN=1` / `HELIX_MP_*` — not re-run here.
- No service-level objective was claimed.

**Deliberate behavior notes.**

- `insertEditions` uses `ON CONFLICT (id) DO NOTHING`: after the delete and the
  ownership check, a conflict is a duplicate inside the incoming set, and the
  DO NOTHING makes a replay idempotent (the previous code surfaced a 23505
  there).
- `claimFiles` and `insertFiles` are separate statements by design: the claim
  removes pre-existing rows for the new ids and pairs before the insert, so the
  plain multi-row files insert still surfaces any unclaimed duplicate loudly.
- The `rehomedEditions` prune keeps its per-edition emptiness guard
  (`NOT EXISTS (SELECT 1 FROM files f WHERE f.edition_id=e.id)`), so an edition
  that lost one file but still has others is not pruned — same as before.

## 2026-11-23T19:20Z — HRD-027 (continuation): slow-client and ingest-concurrent-with-reads runs, harness repaired

**Scope.** Completed the two review items whose agents were killed mid-run:
slow clients, and catalog ingest concurrent with reads. Infrastructure: a
throwaway rootless podman Postgres 15 (`127.0.0.1:55499`) and Redis 7
(`127.0.0.1:56499`) dedicated to this run; the shared host containers were not
touched. Go test run with `-p 2`.

**Repairs to the tests (no production code changed).**

- `server/internal/catalog/slow_client_live_test.go`: now gated on
  `HELIX_LOAD_RUN=1` like its siblings; seeds and deletes its own work instead of
  `SELECT id FROM works LIMIT 1` (which failed with `sql: no rows` on an empty
  database); cleanup runs before the repo is closed (the first version leaked
  three works because `defer repo.Close()` ran before `t.Cleanup`).
- `server/internal/httpapi/load_concurrent_test.go`: removed leftover diagnostics
  (a per-merge `GetWork` that added read load to the very throughput being
  measured, a sampler goroutine whose files-per-work query joined the wrong key
  and always reported 0); replaced the "short page is an error" oracle with a
  counted-and-bounded one plus a quiescent full-page control.
- The scratch `zz_scratch_repro_test.go` was investigated, not deleted blind: it
  asserted nothing and only logged the single-threaded ReconcileSource contract.
  It is now `reconcile_live_test.go::TestLiveReconcileSourceKeepsListedFileAndEdition`
  with real assertions (removed==1, listed file and edition survive, an empty
  listing prunes the work). It passes against current `postgres.go`; it does not
  reproduce a defect.
- At HEAD `go build ./...` and `go vet` were clean; the compile errors T4 reported
  in the untracked slow_client files were not present when I started (UNCONFIRMED
  who fixed them; the files are untracked, so there is no history to consult).

**Finding that explains the earlier "works vanish" runs.** `files` is unique on
`(source_id, path)`. Merging work B onto the same stable path as work A moves
that file and prunes A, with no reconcile involved (reproduced with a throwaway
test: A -> `publication not found`, B present; throwaway removed). The harness
reuses `load/book.txt` for every merged work, so only the newest few merge works
exist at any instant. They are also the first rows of the `updated_at DESC`
browse page. `Search` fetches ids and hydrates them in separate statements, so
an id listed and then pruned yields a short page (observed shortfall 2-3 of 20,
645-830 short pages per 5.4-6.3k browse requests). This is a read-consistency
property, not an error; it is counted in the evidence (`browse_short_pages`,
`browse_max_shortfall`) and bounded (empty page, duplicates, or fewer than half a
page still fail).

**Results (all machine-produced; see `evidence/campaign-2026-11-23/`).**

- Slow client, `slow-client/` (raw.log, raw_catalog.log, results.json): catalog
  surface `TestLiveSlowClientCatalogSurface` PASS, 14 asserts, 0 violated; job
  surface `TestLiveSlowClientJobSurface` PASS, 13 asserts, 0 violated. The
  injected stall reached its deadline (catalog slow read 400.3 ms at a 400 ms
  deadline; job slow scan canceled at 503 ms at a 500 ms deadline), so the
  injection is observed, not assumed.
- Ingest concurrent with reads, `concurrent/` (raw_run1-3.log, results.json,
  artifacts): three overlap runs, zero read errors and zero ingest errors in
  each, browse p95 12.7 / 14.8 ms (knee 27.7), text p99 31.9 / 25.2 ms (knee
  12 s); **FAIL on the write floor**: MergeWork 175.7 / 152.1 ops/s (and 159.7 in
  run 1) against 271.94. Two no-reads controls measured 266.1 / 251.1 ops/s (also
  under the floor). Host load average was about 17 on 16 cores throughout, so the
  absolute numbers and the attribution to read overlap are UNCONFIRMED; the
  floor is not met on this host.
- Earlier overlap artifacts in `evidence/campaign-2026-11-20/load/` (17:49-18:25
  and 19:03) come from the killed session and the pre-repair oracle; they are
  kept and are superseded, not deleted.

**Gaps / not done.**

- The write floor of 271.94 ops/s was recorded as a sequential single-worker
  baseline; this harness runs 3 merge + 3 reconcile workers on a loaded host, so
  the comparison is not like-for-like. No claim of PASS is made; the assertion
  stays and fails.
- Slow clients were injected at the database driver and job store, not as TCP
  clients against the HTTP listener; HTTP server socket-level slow-client
  behavior is unmeasured.
- Non-gated suites: `go test -count=1 ./...` passed in `server/` and `core/`.
  Evidence files written by the tests land in `evidence/campaign-2026-11-20/`
  (hardcoded path) and were copied to `evidence/campaign-2026-11-23/`.
- Identifiers in the test files and artifact names were renamed from `atm027` to
  `hrd027` (env vars `HELIX_HRD027_*`, `hrd027_*.json`); see
  `evidence/ID-KEY-RENAME.md`. Artifacts already on disk keep the old names.

## HRD review findings on set-based upsertTx (2026-11-23)

Reproduce-first (§11.4.146): live tests written first and run against the pre-fix code on a throwaway rootless postgres:15. RED output:

```
--- FAIL: TestLivePostgresHRDUpsertDuplicateSourcePathCurrentIDWins
    order [hrd-legacy-file hrd-current-file]: upsert failed: ERROR: duplicate key value violates unique constraint "files_source_path" (SQLSTATE 23505)
--- FAIL: TestLivePostgresHRDUpsertDuplicateEditionIDFailsLoud
    duplicate edition id in the incoming set was accepted silently
--- FAIL: TestLivePostgresHRDUpsertBeyondBindParameterLimit
    large upsert failed: extended protocol limited to 65535 parameters
```

Fixes in server/internal/catalog/postgres.go: dedupeFiles (per (source_id,path) current-ID row wins, else last; per id last wins; caller Work untouched), rejectDuplicateEditionIDs plus plain INSERT (no DO NOTHING), 1000-row chunks for insertEditions/insertFiles/claimFiles in the same transaction. currentFileID mirrors ingest.fileID, pinned by TestHRDCurrentFileIDGoldenVector. TestLivePostgresUpsertReplacesEditionSet now counts file ids against files and asserts no stale rows for the source.

GREEN: the three HRD live tests, golden vector and the corrected replace-set test pass; the full live catalog suite (incl. TestCatalogMergePreservesEveryWriterUnderIdentityLocks, TestLiveScaleCatalog, round trip, reconcile, batch) passes (ok 70s); go build, go vet ./internal/..., gofmt -l clean; non-gated go test ./... all ok.

## HRD-031 — LCP consumer fulfillment (2026-11-23)

Added `core/pkg/lcp.Fulfill` (fulfill.go): given a License Document and the
publication bytes its `publication` link names, it requires the link to state a
length and SHA-256, verifies both, requires an LCP-declaring container with no
license, copies every entry raw and embeds the license byte-for-byte at
`META-INF/license.lcpl`, then re-opens the result. Signature/trust/passphrase
stay in `Container.Unlock`.

- RED: `go test ./pkg/lcp -run Fulfill` failed to build (`undefined: Fulfill`, `ErrPublicationMismatch`, ...) before fulfill.go existed.
- GREEN: `TestFulfillEmbedsLicenseAndUnlocks`, `TestFulfillRefusesWhatItCannotVouchFor` (7 refusals), `TestFulfillRefusesNonPublications`, `TestFulfillNeverRewritesALicensedContainer`, `TestFulfillOverLiveHTTP` pass; `go test -count=1 ./...` in core/ and server/ green.
- Mutation: disabling the SHA-256 comparison makes `TestFulfillOverLiveHTTP` and `TestFulfillRefusesWhatItCannotVouchFor` fail.
- Live qualification over loopback TCP: `evidence/campaign-2026-11-23/opds-lcp/lcp_fulfill_live.json`.
- NOT PROVEN (UNCONFIRMED): interop with a real LCP provider/License Authority or third-party reader; License Status Document (register/return/renew); Readium root trust (Helix embeds none); server route/OPDS wiring for fulfillment (ingest still skips standalone `.lcpl`); hash encoding assumed hex SHA-256, other encodings fail closed.

## HRD-026 kosync reference-conformance (2026-11-23)

Reference (primary sources): `/usr/lib/koreader/plugins/kosync.koplugin/api.json` expected_status register [201,402] (l.16), authorize [200,401] (l.21), update_progress [200,202,401] (l.44), get_progress [200,401] (l.52); a status outside the list makes Spore raise, so `KOSyncClient.lua` returns failure (l.79-83, 169-172). Upstream `koreader/koreader-sync-server` (cloned, `config/errors.lua` l.37-39 + `app/controllers/1/syncs_controller.lua`): duplicate user 2002 => 402 (l.138-139), registration disabled 2005 => 402, unauthorized 2001 => 401 (l.120-126), unknown document => `200 {}` (l.236-241, "do not want an almost empty table"), PUT ok => 200 {document,timestamp} (l.285). `main.lua` l.586-594/630-634 shows `body.message`; l.851 `not body.percentage` => "No progress found".
Changes (`server/internal/httpapi/kosync.go`): duplicate register 409 -> 402; registration-disabled 403 -> 402; unknown-document GET 404 -> 200 `{}`; kosync errors now also carry top-level `message`. Kept stricter (KOReader treats any non-expected code identically as failure): invalid register input 400 (ref 403), invalid document 400, PUT validation 400, percentage in [0,1], username/password length rules.
RED: `registration default is not closed: 403 ...` (TestKOReaderSyncProtocol). GREEN: `go test ./...` all ok, gofmt/vet clean.
Real client after fix: `evidence/campaign-2026-11-23/interop/kosync_koreader_real_client_after_fix.txt` (0 occurrences of "not expected"; unknown doc ok=true body={}; duplicate register status false with message "Username is already registered."). Document id from KOReader's real `util.partialMD5` over sample.epub = af00b1716a8ea20bc7f1108c8358c4a3, independently reproduced in python (LuaJIT bit.lshift masks negative shift, so i=-1 samples offset 0).

### KOSync stale-expectation fix (review follow-up, 2026-11-23)
Files: `server/internal/httpapi/load_delivery_test.go`, `scripts/verify_live_server.py`, `docs/api.md`.
RED (live Redis 7.4 + Postgres 15 podman, `TestLiveLoadKOSync`): `load_delivery_test.go:839: duplicate registration returned 402, want 409`.
Changes: duplicate register expects 402 + message "Username is already registered."; never-written document expects 200 with body exactly `{}`; `getPercentage` fails on empty `document`; verifier accepts {201,402} and on 402 requires the duplicate message; docs/api.md updated (402/402, `200 {}`, top-level `message`).
GREEN: `TestLiveLoadKOSync` PASS (2.79s); `go test -count=1 ./...` all ok; gofmt -l empty; go vet clean. `scripts/verify_live_server.py` run twice against live Redis + pgvector/pg16: both runs status PASS, 77/77 checks, including rerun case (402 + duplicate message) and the new message check. Containers removed.

## LCP License Status Document client (2026-11-23)

`core/pkg/lcp/lsd.go`: parse/validate a Status Document and a client (injected `http.Client`) for fetch, register (POST), return (PUT), renew (PUT, optional `end`), plus `NeedsLicenseRefresh` (updated.license newer than the local License) and profile-based license-link selection. Primary sources, cloned and read: `readium/lcp-specs` commit 812dd91d6b90087e657a5c94d04e6f16abd8873b, `releases/lsd/latest.md` (revision 5; file:line cited in code comments) and `schema/status.schema.json`. Revision 5 followed over rev 4 (after return/renew the client must also re-download the License).
- RED: `go test ./pkg/lcp` failed to build (`undefined: StatusValue`, `LSDClient`, `lsdMediaType`...) before lsd.go existed.
- GREEN: 16 top-level tests / 88 test results (per-status table, 28 rejection cases, RFC 6570 `{?id,name}` / `{?end,id,name}` expansion, problem+json type/title/detail, request verbs and query keys, state-transition checks, live loopback register->renew->(bad renew 403)->return->(second return 403)). `go vet ./...`, gofmt, `go test -count=1 ./...` in core/ green on a scratch copy.
- Mutation (scratch copy): dropping Register's "still ready" check, Return's returned/cancelled check, or flipping the staleness comparison each fails named tests (`evidence/campaign-2026-11-23/opds-lcp/lsd_mutation.json`).
- Live loopback evidence: `evidence/campaign-2026-11-23/opds-lcp/lsd_live.json` (query keys only; device id/name never recorded or put in errors; transport errors stripped of URL).
- NOT PROVEN (UNCONFIRMED): any real LSD provider (the provider in tests is written in-repo from the same spec text); TLS/redirect behaviour of real hosts; `register` verb is POST per rev 5 l.270 while some deployed providers have been reported to use PUT (not verified); Accept-Language localization; strict event-type enum rejects unknown event types (schema-faithful, may be stricter than real providers); no server/OPDS wiring and no persistence of the device id; fulfill.go hash-encoding concern is separate and untouched.

## LCP Fulfill spec-review fixes (2026-11-23)

Independent spec review (lcp-specs 812dd91d, readium-lcp-server) findings addressed in
`core/pkg/lcp/{fulfill,license}.go`, tests in `fulfill_spec_test.go`:
link hash accepts hex or base64 (32 bytes required); `length` accepted as number or string,
negatives/non-numeric rejected at Parse; non-EPUB link types -> `ErrUnsupportedFormat` (root
license.lcpl placement for PDF/audiobook NOT implemented); `AllowUnverified()` opt-in (stated
length/hash still enforced); `FulfillTo(io.Writer, ...)` streaming (verifies before first write,
re-hashes after copy to detect a changing source; no read-back of a bare Writer);
`Link.Templated` + `ErrTemplatedHref`; "<nil>" message fixed; signature-not-verified warning in docs.
Mutation: base64 branch broken -> TestFulfillAcceptsHexAndBase64Hash FAIL; hex branch broken -> 10 FAIL.
Green output: evidence/campaign-2026-11-23/opds-lcp/fulfill_spec_review_green.txt

## LSD client review round 1 fixes (2026-11-23)

Independent review of core/pkg/lcp/lsd.go reproduced 5 defects; fixed test-first in lsd.go / lsd_test.go.
Each new test was RED before the fix and GREEN after; mutation checks (restore verified byte-identical):

- html renew link must be absolute http(s) with host (no javascript:/file:/intent:): mutation "disable check" -> TestLSDInteractiveRenewalURLMustBeAbsoluteHTTP FAILS.
- events missing timestamp/type are kept, not fatal (SCHEMA events items have no `required`); present-but-invalid still rejected. `"rel": []` accepted (link.schema.json has no minItems); license-link requirement unchanged.
- TestLSDOversizedBodyRefused now uses a valid document of 1 MiB+1 KiB (literal): mutation lsdMaxBody 1<<20 -> 4<<20 FAILS, mutation "drop size check" FAILS.
- state-changing requests (POST/PUT) never follow redirects (301/302/303/307/308 -> *LSDProblem with the 3xx status): mutation "follow redirects" FAILS.
- non-templated href containing `{`/`}` is refused with ErrLSDBadTemplate (templated defaults to false, SPEC l.182); mutation FAILS.
- Live lifecycle test now leak-checks every error returned; fixture-only subtest removed.
Gates: gofmt -l clean, go vet ./... clean, go test -count=1 ./... (core) all ok.

## HRD-031 server wiring: LCP license fulfillment over OPDS (2026-11-23)

Scope: EPUB, Basic profile, opt-in (`HELIX_LCP_CACHE_DIR`). Standalone `.lcpl` in a source -> hardened download of the publication link -> `lcp.Fulfill` -> protected EPUB in a server-managed cache (source bytes untouched, HRD-004) -> catalog edition (`drm_scheme=lcp`, `license_href`) -> OPDS 1.2/2.0 direct acquisition of the fulfilled EPUB plus `rel="related"` License Document link. No decryption, passphrase or key on the server.

Code: server/internal/ingest/{lcp.go,service.go}, ingest/lcpfixture (test-only provider fixtures), opds/{detail.go,model.go}, httpapi/{lcp.go,router.go}.
Tests (all green; failure paths assert the explicit error and that nothing is left in the cache): ingest/lcp_test.go (happy path with unlock round trip, idempotent rescan = exactly 1 download, hash mismatch, length mismatch, unreachable host, private-address SSRF block with zero contact, redirect-to-loopback block, oversized body with and without Content-Length, PDF link -> ErrUnsupportedFormat without fetching, templated/file:/ftp:/userinfo hrefs, timeout, 404, unverifiable link fail-closed + opt-in, no URL/token echo in the scan report); httpapi/lcp_test.go (byte-exact content, Range, HEAD, ETag, license route, 401 without OPDS credentials, 404 for non-LCP file, 409 on tampered/missing cache or disabled fulfillment, OPDS 1.2/2.0 links, env opt-in).
Mutation checks: SSRF guard disabled -> `private_address_blocked...` and `redirect_to_a_private_address...` FAIL; `.lcpl` branch disabled -> TestLCPLScanFulfillsStoresAndStaysIdempotent and TestLCPLScanReportsFailureWithoutLeakingURLs FAIL; both restored.
Live: evidence/campaign-2026-11-23/opds-lcp/run.sh builds a fresh helix-server, runs it (in-memory catalog) against a library holding one .lcpl and a loopback provider; results.json (15 checks PASS), feed-1.2.xml, live.log, evidence.sha256, binary.sha256. Downloaded EPUB sha256 equals the independently fulfilled expectation; unlock with the test passphrase returns the original chapter, a wrong passphrase is refused.
Not verified (UNCONFIRMED): any real LCP provider / Readium LCP server; any third-party LCP reader opening the artifact; PostgreSQL-backed catalog for this flow (columns pre-exist and are mapped, not exercised live); server-side signature/trust-chain validation (not implemented by design, scan report warns); PDF/audiobook/Divina LCP; cache eviction of artifacts orphaned by a changed or deleted .lcpl (not implemented).

## HRD-031 independent security review: 7 defects fixed test-first (2026-11-23)

Each new test was RED before the fix (output captured in the session) and GREEN after.
1. templated href (may carry a token) no longer echoed: ingest/lcp.go and core/pkg/lcp/fulfill.go return bare ErrTemplatedHref; TestLCPLTemplatedHrefNeverEchoed, TestFulfillTemplatedErrorDoesNotEchoHref.
2. httpapi serves a cached artifact only after SHA-256 == catalog hash (verdict cached per path/inode/mtime/size): TestLCPContentRejectsSameSizeTamper, TestLCPContentTamperDetectedOnFirstRequest. Mutation "skip hash compare" -> both FAIL.
3. /license is taken from the verified cached artifact's embedded license, not the live .lcpl: TestLCPLicenseIsTheEmbeddedLicenseNotTheLiveSource, TestLCPLicenseDoesNotNeedTheSource.
4. cache GC (ingest/lcp_gc.go) after reconcile / fulfillment: unreferenced artifacts, idx and stale tmp removed after CacheGrace; HELIX_LCP_CACHE_MAX_BYTES evicts oldest unreferenced first; referenced never deleted: TestLCPCacheGC*.
5. unverifiable link refused before any download; body capped at stated length, Content-Length mismatch aborts; per-license exponential backoff (5m..1h): TestLCPFulfillUnverifiableRefusedBeforeDownload, ...CapsBodyAtStatedLength, ...ContentLengthMismatchAborts, ...BacksOffAfterFailure. Mutations (drop pre-check / drop cap / drop backoff) -> each FAILS.
6. injectable Resolver; TestLCPDialsValidatedIPNotHostname, TestLCPResolverReturningPrivateIsBlocked. Passed at baseline (code already dialed the IP); mutation dial(ctx, network, addr) with hostname -> FAILS (dialed [rebind.example:8080]).
7. restrictedIP extended (NAT64, 6to4, Teredo, v4-compat, 0/8, 192.0.0.0/24, 198.18/15, 240/4, fec0::/10, 100::/64): TestRestrictedIPReviewForms with public positive controls. Mutation (empty extra range list) -> FAILS.
Live re-qualification: evidence/campaign-2026-11-23/opds-lcp/run.sh re-run; results.json 15 PASS, sha256 refreshed in evidence.sha256/binary.sha256.
UNCONFIRMED: a same-size flip that also restores mtime and inode of a cache file is not detected between requests (verdict cache; local cache-dir write access is out of scope); backoff state is in memory (a restart permits one extra bounded fetch); GC walks the catalog by keyset twice and relies on a 10-minute grace for works updated mid-walk; PostgreSQL-backed GC not exercised; OPDS rel for the license kept as related (Readium docs show acquisition + license type; no normative rule found, readium.org/lcp-specs/notes/lcp-key-retrieval.html examples only, not file:line citable).

## HRD-029 HelixTrack sync, local Core instance (2026-11-23)

- Target: Helix-Track/Core commit c85221282d8b built from source, loopback only, SQLite, local auth; stopped and removed afterwards.
- Tool: `tools/helixtrack-sync/` ; unit RED then GREEN logs `evidence/campaign-2026-11-23/helixtrack/unit-RED.log` (import failure) and `unit-GREEN.log` (31 tests OK); 3 mutations of the tool each fail the suite.
- Live run `evidence/campaign-2026-11-23/helixtrack/live/` (script `run_live.sh`): dry-run 0 writes; push created 35; compare 35/35 field match; pull 35 in-sync; re-push 35 noop with 0 write requests, ticket count stayed 35; second user edit of HRD-007 detected (remote_version 3 vs base 2); push reported conflict without overwriting; stale-version write rejected (409, current 3); `--overwrite-remote` restored; final pull 35 in-sync, compare 35/35; Core DB history shows hrd_sync and hrd_remote_editor actors. Secret scan of evidence clean. Manifests: `live/SHA256SUMS`, `SHA256SUMS.top`.
- UNCONFIRMED: production HelixTrack instance, external auth/permission services, multi-user permission scoping, attachments, scale.
- Wording and quirks: `docs/helixtrack-sync.md`.
