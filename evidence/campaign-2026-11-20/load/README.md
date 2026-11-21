# ATM-027 concurrency, contention, memory and queue-survival measurements

Campaign 2026-11-20, Phase 1. First measurements of concurrent load, lock
contention, resource envelope under load, and scan-queue behavior across a
server restart for the Helix catalog/API. Source fingerprint `f0a5f18` +
uncommitted tree; per-step SHA-256 fingerprints of `postgres.go`,
`postgres_batch.go`, `jobs.go`, `router.go`, `ingest/service.go` are in
`run.log` (stable across every run here).

## What was measured and how

| Artifact | What it is |
|---|---|
| `catalog_ladder.jsonl` / `.log` | repo-level concurrency ladder: offset browse, deep keyset, `%text%` search at 1/8/32/64 clients + pool-attribution probe |
| `http_ladder.jsonl` / `.log` | HTTP-level ladder through the real Gin router: `GET /v1/works` (offset, deep cursor, search) and `GET /opds/2.0/publications` at 1/8/32/64 clients |
| `http_ladder_repeat_{1,2,3}.jsonl` | three repeats of that ladder, same binary and fixture — the run-to-run spread behind *Repeatability and host contention* |
| `http_ladder_contention_needle_{quiet,loaded}.jsonl` | `clients=8` ladder run with and without 16 deliberate host CPU hogs, proving the `host_contention` field detects co-tenant load |
| `http_ladder_host_gate_needle_{a,b}.jsonl` | runs with CPU hogs started *mid-run*, proving the `host_quiet` gate fires on load that begins after the early cells |
| `catalog_contention.jsonl` / `.log` | write-path serialization test (4 phases with a no-advisory-lock control) |
| `restart_queue.jsonl`, `pre_kill.json`, `post_restart.json` | durable-queue-across-SIGKILL evidence, real `helix-server` binary |
| `host_catalog_ladder.jsonl` / `.log` | same ladder driven from the host through podman's pasta forwarder (path comparison, non-authoritative) |
| `results.json` | every payload above, machine-readable, rebuilt by `summarize.py` from the `*.jsonl` files |
| `run.log` | timestamped step log with fingerprints and step results |

Environment: rootless podman 5.7.0, 16-core host, PostgreSQL 17 in the
pre-existing disposable container `helixqual-pg` (unlimited CPU quota,
`max_connections=100`, `shared_buffers=128MB`), fixture = the existing 50,000-work
scale fixture (titles `Scale Volume <g>`, 100k editions, 200k identifiers,
300k files) in `helix_test`, already VACUUMed by `TestLiveScaleCatalog`.

## Measurement paths

- **Authoritative (clean path):** all `catalog_ladder`, `http_ladder`,
  `catalog_contention` payloads — the test binaries were compiled on the host
  (`go test -c`), `podman cp`'d into the database container and run there
  against `127.0.0.1:5432`, labelled `measurement_path=in_container_clean`.
  The catalog ladder additionally re-ran the shared six-shape transport probe
  and recorded `transport_probe=clean`.
- **Non-authoritative:** `host_catalog_ladder` ran on the host against the
  published port `127.0.0.1:34320` through pasta. This time the probe came out
  clean and numbers landed within ~10–25% of the in-container run (keyset@64:
  595 vs 822 req/s). The ~205 ms pasta stall is intermittent (documented in
  `docs/scalability-review.md`), so host-path numbers are kept for comparison
  only and are never used for a claim.
- `restart_queue` asserts persisted state (rows, job records), not latency;
  it ran on the host (`host_through_pasta_state_assertions_only`), which
  cannot invalidate a state assertion.

## Headline numbers (all `in_container_clean` unless noted)

Throughput / latency at 1 → 8 → 32 → 64 simultaneous clients:

| Workload | req/s | p95 ms | p99 ms |
|---|---|---|---|
| repo offset browse page | 117 → 638 → 994 → **992** | 9.1 → 23.2 → 44.9 → 107.6 | 9.1 → 27.7 → 49.4 → 126.8 |
| repo deep keyset page | 93 → 483 → 801 → 822 | 12.0 → 25.2 → 52.8 → 108.2 | 12.0 → 34.0 → 59.3 → 126.6 |
| repo `%text%` search | 0.9 → 6.5 → 8.9 → 8.7 | 1101 → 1449 → 5205 → 11352 | 1101 → 1449 → 5548 → 12043 |
| HTTP `/v1/works` browse | 102 → 540 → 787 → 897 | 12.3 → 23.6 → 68.1 → 99.3 | 12.3 → 25.3 → 76.1 → 114.4 |
| HTTP `/v1/works` deep keyset | 78 → 463 → 669 → 671 | 14.4 → 22.6 → 66.7 → 136.6 | 14.4 → 32.9 → 69.5 → 158.1 |
| HTTP `/v1/works` text search | 1.0 → 6.0 → 9.4 → 9.5 | 1031 → 1466 → 5120 → 11049 | (rounds/client=2; see *Repeatability and host contention* before quoting these rps figures) |
| HTTP OPDS 2.0 feed (20 entries) | 82 → 543 → 776 → 825 | 17.7 → 22.4 → 61.0 → 109.1 | 17.7 → 27.8 → 71.2 → 127.6 |

Errors: **zero** in every level of every workload (no timeouts, no 5xx, no
pool exhaustion errors, no DB connection rejections).

Candidate knee points (observed, **not** ratified service-level objectives —
none is approved by the project owner):

- **Browse/keyset/OPDS (indexed reads): knee between 32 and 64 clients.**
  Throughput plateaus at ~1000 req/s (repo) / ~900 req/s (HTTP) while p95
  doubles from 32→64. The 20-connection pool is **not** the limiter: a probe
  with pool=64 measured 942 req/s — statistically the same as pool=20's 992 —
  and pool waits were zero at every browse level. Database execution CPU is
  the saturation resource (sampled wait events during these phases are
  `cpu:(none)`; ungranted locks: 0).
- **`%text%` contains-search: knee between 8 and 32 clients.** Ceiling is
  ~9 req/s at every level ≥32 on a quiet host, and 5.5–9.5 req/s across five
  runs on this machine: the plateau's *shape* (flat past 32 clients) is the
  reproducible claim, a single rps value is host-load dependent — see
  *Repeatability and host contention*. Unanchored ILIKE across four columns
  plus two EXISTS sublinks; at 64 clients the database burned ~195 CPU-seconds in
  ~15 wall seconds (≈13 cores busy) and p99 reached 12.0 s. Pool starvation
  appears **on top of** the CPU wall here: 728 pool waits, ~574 s of total
  wait time (queries run ~10× longer than the 20-slot pool's lease comfort).
- **OPDS 2.0 acquisition-feed build costs the same as the underlying page
  query** (~825 vs ~900 req/s at 64 clients): rendering 20 entries is not a
  separate bottleneck at this page size.

### Repeatability and host contention (added 2026-11-21)

A re-run of the HTTP ladder on 2026-11-21 measured the search plateau at
7.5–7.8 req/s against the 9.4/9.5 recorded above, which looked like a
regression. It is not one, and the investigation produced two findings about
the measurement rather than the catalog. Both matter to every latency figure
in this directory.

**1. The search cell is a thin sample.** `HELIX_LOAD_ROUNDS` defaults to 8, but
the search route runs `rounds/4` = 2 rounds per client, so the 32-client cell is
64 requests and the 64-client cell 128 — a handful of ~3 s scans. Three
back-to-back repeats of the same binary against the same fixture:

| run | clients=1 | clients=8 | clients=32 | clients=64 |
|---|---|---|---|---|
| 2026-11-20 20:03Z (`http_ladder.jsonl`) | 1.0 rps / 947 ms | 6.0 / 1211 | 9.4 / 2812 | 9.5 / 5773 |
| 2026-11-21 00:57Z (`http_ladder-2026-11-21T0155Z.jsonl`) | 1.3 / 772 | 5.7 / 1186 | 7.8 / 3274 | 7.5 / 7356 |
| `http_ladder_repeat_1.jsonl` | 0.8 / 1171 | 4.1 / 1776 | 5.5 / 4765 | 5.9 / 9319 |
| `http_ladder_repeat_2.jsonl` | 0.8 / 1154 | 4.6 / 1511 | 6.2 / 4297 | 6.2 / 9222 |
| `http_ladder_repeat_3.jsonl` | 1.1 / 713 | 5.9 / 1165 | 7.6 / 3502 | 7.4 / 7797 |

Spread at the saturated levels is ±20–30% on throughput and up to ±60% on p50.
**A single ladder run cannot detect a change smaller than roughly a quarter of
search throughput**; only the ordering across levels (1 → 8 rises, 32 → 64 flat)
is stable enough to claim.

**2. This host is shared, and the harness could not see it.** The load suite
runs at `in_container_clean` because pasta's network path was the known
confounder, but CPU is not namespaced either: during these runs an unrelated
project's Gradle release build (`lava-android-builder`) held 2–16 of the 16
cores. Nothing in the payload recorded that, so two runs were being compared as
if they described the same machine.

The ladder now records `host_contention` per cell: `entry_busy_cores` and
`exit_busy_cores` are host-wide cores busy over a 150 ms window *before* the
cell releases any client and *after* it finishes (so they are co-tenant load,
not this harness's own parallel scans), and `cpu_busy_cores` covers the cell
itself. Validated by measurement, not by
inspection — same binary, `HELIX_LOAD_LEVELS=8`, once with the machine as-is and
once with 16 `yes` loops added:

| workload | rps (entry 2.3–10.5 cores) | rps (entry ≈16 cores) | p50 ms quiet → loaded |
|---|---|---|---|
| HTTP browse | 620.3 | 214.1 | 10.9 → 30.8 |
| HTTP deep keyset | 415.8 | 107.3 | 16.9 → 65.1 |
| HTTP text search | 5.2 | 2.8 | 1291.6 → 2183.8 |
| HTTP OPDS feed | 298.1 | 143.5 | 24.0 → 42.6 |

Files: `http_ladder_contention_needle_quiet.jsonl`,
`http_ladder_contention_needle_loaded.jsonl`. Co-tenant CPU moves these
workloads by 2.0–3.9×, which is larger than the difference between the
2026-11-20 and 2026-11-21 runs — so that difference is not evidence about the
catalog either way.

The first version of the probe read `busy_cores=15.9` on a machine the kernel
reported as 1.7 cores busy. The needle caught it: `/proc/stat` was parsed whole,
so the per-core `cpuN` lines and the `intr` counters were summed into the
aggregate totals. Fixed to the first line only (excluding `guest`, which the
kernel already counts inside `user`) and re-needled, which is the table above.

**Consequence for re-running — now enforced by the harness, not by this note.**
Every `http_ladder` payload carries `host_quiet`, `host_contended_cells` and
`host_external_busy_max`: a cell counts as contended when host cores busy
measured *both* just before and just after it exceed half the machine, and the
test logs `HOST NOT QUIET` while still reporting `measurement_path` unchanged —
the network path and the machine state are separate conditions. Verified by
launching 16 CPU hogs one second into a run, so the load begins after the entry
probes of the early cells: two cells saw 15.7 and 16.0 cores and the run was
flagged (`http_ladder_host_gate_needle_{a,b}.jsonl`). The intended quiet control
tripped too — the Gradle build was bursting between cells — which is the
behaviour the field exists to reveal rather than a fault in it. Per-cell values
inside those runs still spanned 2.1 to 16.0 cores.

No `http_ladder` run after `2026-11-20T20:03Z` on this machine qualifies as
authoritative under this gate; the 2026-11-20 figures stand, and a clean-host
re-run is outstanding. Check before quoting:

```bash
podman exec helixqual-pg bash -c 'cd /tmp && HELIX_LOAD_RUN=1 HELIX_MEASUREMENT_PATH=in_container_clean \
  HELIX_TEST_POSTGRES_DSN="postgres://helixqual@127.0.0.1:5432/helix_test?sslmode=disable" \
  /tmp/httpapi_current -test.run TestLiveLoadHTTPLadder -test.v' | grep -E 'HOST NOT QUIET|host_cores at_entry'
```

### Contention verdict (measured, not read off the source)

8 concurrent writers over row-disjoint data (`load-contend-*` works; no row
lock can contend by construction), through the production `MergeWork` path:

| Phase | ops/s | p50 ms | advisory-lock waiters observed |
|---|---|---|---|
| A: sequential `MergeWork` | 291 | 3.2 | — |
| B: concurrent `MergeWork` (8) | **251** | 29.7 | up to 7 ungranted; 8/9 samples with `Lock:advisory` waiters |
| C: same tx body, `pg_advisory_xact_lock(731452)` omitted (control) | **1499** | 4.2 | 0 |
| D: concurrent `ReconcileSource` (8) | 695 vs 723 sequential | 9.6 | same lock, same signature |

B scales 0.86× versus A (no gain, slight loss) while C scales 5.14×: the
**global advisory transaction lock on the catalog identity path serializes
all writers** — merges, upserts and reconciliations — regardless of data
disjointness. Deadlocks recorded: 0. This confirms the `docs/scalability-review.md`
risk item as a measured property: ingest/merge write throughput is capped at
≈290 ops/s per process (a single serialized critical section), independent of
concurrency.

### Resource envelope (observed at these workloads)

- Catalog-layer harness process: peak RSS 23–29 MB, peak goroutines 8→79;
  Go heap stayed under 5 MB at browse levels.
- HTTP harness process (server + co-resident load clients): peak RSS
  58–91 MB at 32–64 clients, peak goroutines 344.
- PostgreSQL: peak backends 26 at pool default (max_connections=100), 75 in
  the pool=64 probe; temp-file spill 44 files / ~87 MB observed during the
  write-contention phases; 0 deadlocks throughout.
- CPU: browse plateau ≈1000 req/s costs the database far less CPU per
  request (~10–20 ms each) than search (~1.0–1.5 CPU-s each).

### Queue-across-restart verdict (real `helix-server`, SIGKILL)

12 scan jobs enqueued against 3 local sources (2 active, 10 queued cap are
the in-process constants); SIGKILL at ~1.5 s; restart with the same
PostgreSQL + Redis journal:

- **No record lost**: 12/12 job IDs readable after restart; 3 already
  `succeeded` kept their state **and** their report bytes.
- **Nothing requeued or re-run**: all 9 `queued`/`running` records became
  `failed` with `scan interrupted by server restart`. This is the implemented
  policy (`RestoreScans`); it is durable *records*, not a durable *queue* —
  re-dispatch is an explicit operator/client action.
- **Committed progress survived**: 171→231 catalog files were durably
  committed by the time of the kill (per-file merge transactions, no
  rollback). The interrupted scan of `load-restart-src-2` stopped at
  111/120 files: those 9 are **lost from the job** and only return on an
  explicit re-scan.
- **Re-scan is idempotent**: a completed re-scan of
  `load-restart-src-1` (already fully catalogued) added **0 duplicate rows**
  (120 files before, 120 after; the `(source_id,path)` unique index and
  content-keyed ids make duplicates structurally impossible, and the
  re-scan succeeded).

## Reproduction

```bash
# 1. containers must exist (pre-existing, disposable):
#    helixqual-pg (PG17+pgvector, 127.0.0.1:34320) and helixqual-redis
# 2. seed the 50k fixture if absent (≈25 s):
cd server && HELIX_TEST_POSTGRES_DSN='postgres://helixqual@127.0.0.1:34320/helix_test?sslmode=disable' \
  go test ./internal/catalog -run TestLiveScaleCatalog -v -timeout 30m
# 3. authoritative run (compiles, copies into the DB container, runs, copies
#    back, appends results.json per step):
./evidence/campaign-2026-11-20/load/run_load.sh all          # contention, ladder, http
./evidence/campaign-2026-11-20/load/run_load.sh host-ladder  # pasta-path comparison
# 4. durable-queue restart scenario (real server binary, kill -9, restart):
./evidence/campaign-2026-11-20/load/restart_queue.sh
# knobs: HELIX_LOAD_LEVELS / HELIX_LOAD_ROUNDS / HELIX_LOAD_OPS /
#        HELIX_LOAD_TIMEOUT_SECONDS (see load_test.go)
```

Both new test files (`server/internal/catalog/load_test.go`,
`server/internal/httpapi/load_test.go`) skip cleanly unless
`HELIX_TEST_POSTGRES_DSN` is set **and** `HELIX_LOAD_RUN=1`; `go test ./...`
in `server/` is unaffected. `results.json` is rebuilt from all `*.jsonl` by
`summarize.py` after every step. In-container steps write their jsonl fresh
inside the container and copy it back over the host file of the same stem, so
each such stem holds the latest run; `restart_queue.jsonl` appends, so it
holds both the first and the (re-)instrumented second run of the scenario —
same verdicts, the second adds per-source duplicate-row counts. An earlier
4-op smoke of the contention test was superseded by the 24-op authoritative
line recorded here.

## What these numbers do NOT establish

- **No SLO.** No latency or throughput target has been ratified by the project
  owner; nothing here "passes" or "fails" a goal. The knee points are
  candidate inputs for defining one (constitution §7.1).
- Single host, single disposable PostgreSQL container, one client process;
  no TLS/HTTP-2/HTTP-3 termination (the ladder used plain HTTP/1.1
  keep-alive), no Brotli on these routes' hot path, no CDN, no replicas.
- Synthetic uniform fixture: every browse page is 20 works with 2 editions /
  2 files / 1 evidence row, one author, identical publisher; the text-search
  pattern matches a large uniform share. Real distributions (long tails,
  huge single-work child counts, sparse matches vs the 731 ms worst case
  already recorded) will differ.
- Content delivery (`/v1/files/:id/content`), KOSync/Redis sync, AI/vector
  search and remote-source connectors were **not** load-tested here at all.
- The write-contention test measures the catalog seam inside one process;
  cross-process multi-writer behavior (e.g. two server replicas) shares the same
  advisory lock keys but its WAL/IO interplay at scale is not measured. See
  `lock-scope-rerun` below for what changed in the lock itself.
- The pool=64 probe ran once; queue-level claims at exactly 64 clients are
  one-sample observations, not medians of repeated runs.
- Restart evidence covers one SIGKILL window (~1.5 s into 12 queued jobs).
  Crash *during* reconciliation (the `ReconcileSource` phase of a scan) is not
  exercised, and no automatic re-dispatch exists to measure.

## Lock-scope rerun (2026-11-20, supersedes the write-contention verdict above)

`server/internal/catalog/postgres.go` no longer takes one global advisory key for
every catalog write; it takes one per work, per edition and per source. Two stems
in `results.json` belong to that change and were produced by
`run_load.sh contention-sharded` and `run_load.sh lock-scope`:

| stem | what it ranks | result |
| --- | --- | --- |
| `catalog_contention_sharded` | production write path (now sharded) vs sequential and vs unlocked | concurrent `MergeWork` **5.14x** sequential (was 0.98x), unlocked control 6.63x, reconcile 2.73x concurrent-vs-sequential, 0 errors, 0 advisory waiters sampled |
| `catalog_lock_scope` | three locking disciplines over the *identical* transaction body in one run | global-key 271.94 ops/s, **sharded 1592.21 ops/s (5.85x)**, unlocked ceiling 1789.00 ops/s (sharded reaches 89%) |

The earlier `catalog_contention` stem is kept as the record of the serialized
behavior that motivated the change; its verdict string describes the code as it
was when it ran. Invariants, paired needles and the mutation gate live in
`../lock-scope/`.
