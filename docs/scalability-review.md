# Backend and API scalability review

## Verdict

**Enterprise-scale qualification is not complete.** Functional, race, live-service,
transport, and recovery tests establish correctness in the recorded environments, and
the three measured sections below establish query cost, concurrent read envelope,
write-path lock cost, and separate-process write safety on a 50,000-work synthetic
catalog in one container. They do not establish horizontal scalability, a durable
scan queue, content-delivery or vector-search behavior under load, or any production
service-level objective. ATM-027 remains open.

This review covers the current source tree. The service table records code-derived
observations plus the artifact each claim came from. The measured sections report
figures with the fixture size and measurement path each one came from; superseded
numbers are named as such rather than removed. No capacity number outside those
sections is claimed.

## Service-by-service findings

| Surface | Existing controls | Scalability risks and open work | Current evidence boundary |
|---|---|---|---|
| Gin HTTP API | Request-size limits on JSON mutation endpoints; bounded page size; security headers; content negotiation; request cancellation | Saturation envelope measured only for read workloads on a synthetic fixture; no per-tenant fairness layer and no distributed rate limiter; write and content-delivery paths not load-tested at the HTTP layer | Correctness: unit, race, live API, transport. Load: 1/8/32/64-client ladder through the real router, in-container (`evidence/campaign-2026-11-20/load/http_ladder.jsonl`) |
| HTTP content delivery | Live revalidation; ranges; single-descriptor local reads; streaming; unknown-size fallback; connection/header timeouts in sources | Strong scan-time hashes still require ingest work; remote content can consume upstream bandwidth; no production CDN/cache qualification | Live content, restart, transport, and raw keepalive tests |
| Scan jobs | Two active scans, 64 pending jobs, 256 retained jobs, cancellation, panic containment, graceful wait group | Terminal job records survive a SIGKILL and restart (measured: 12/12 readable, 3 succeeded reports intact, 9 in-flight marked `failed` rather than re-dispatched); the *pending* set is process-local, so a restart drops queued work and an operator or client must re-trigger it; no distributed scheduling or durable retry policy | Deterministic job tests, live scans, and a kill/restart scenario in `evidence/campaign-2026-11-20/load/` |
| Ingest pipeline | Bounded parser inputs, content identity, source reconciliation, deterministic metadata precedence | Whole-library scans can repeatedly parse/hash large files; no measured incremental-scan throughput or backpressure across remote sources | Unit, integration, identity regression, and live scans |
| Memory catalog | Mutex-protected repository with shared validation | Single-process and volatile; unsuitable as a durable or horizontally shared production catalog | Used explicitly for evaluation/tests, not represented as durable |
| PostgreSQL catalog | Pool caps now deployment-configurable (`HELIX_CATALOG_POOL_*`), transactional migrations with advisory lock, indexed foreign keys and updated-at ordering, batched page-child loads, keyset cursor alongside offset, identity locks scoped per work/edition/source instead of one global key | Upsert still deletes and reinserts the editions of one work; all writers of one *source* still serialize on that source key (measured: it is where the sharded lock stops beating the global one); `%text%` search and the per-page `count(*)` remain unindexed costs | 50k-work fixture measurements and query plans (above); contention and lock-scope throughput in `evidence/campaign-2026-11-20/load/` and `evidence/campaign-2026-11-20/lock-scope/`; separate-process write throughput and lost-update needles in `evidence/campaign-2026-11-20/multiprocess/` — one container, write traffic only, no target SLO |
| PostgreSQL AI memory | Separate pool caps, also deployment-configurable (`HELIX_AI_MEMORY_POOL`, default 10 open/3 idle), dimension checks, bounded search limit, HNSW cosine index, expiry index | No vector-cardinality/recall/latency benchmark; model and embedding dimensions constrain migration | Real pgvector tests with small fixtures |
| Redis KOSync | Optimistic WATCH/MULTI/EXEC, monotonic progress, bounded payloads | No Redis failover, cluster, persistence, or contention-throughput qualification; availability inherits Redis deployment | Real Redis integration tests |
| Internal sync store | PostgreSQL-backed progress logic and tests | No multi-region conflict, retention, or high-contention benchmark; no claimed exactly-once cross-system semantics | Correctness tests only |
| OPDS and OpenSearch | Pagination bounds, acquisition maps, atomic replacement, deterministic public keys | Feed generation inherits catalog query fan-out; large feeds and third-party client behavior are unmeasured; OPDS-PSE remains unimplemented | API tests, not large-feed or external-client certification |
| Remote source connectors | Context cancellation, connection/header timeouts, redirect limits, SSRF dial validation, bounded XML/JSON reads | Long streams deliberately have no total client timeout; deployments need cancellation and bandwidth policy; provider-specific rate limits and retries are not globally normalized | Connector tests; many real providers remain unrun |
| Identification and covers | Provider response-size limits and client timeouts; deterministic evidence precedence | External API quotas, cache hit ratio, thundering-herd behavior, and provider outage budgets lack production measurements | Unit/provider-contract behavior; no production quota result |
| HTTP/1.1, HTTP/2, HTTP/3 | TLS floors, Alt-Svc, Brotli/gzip negotiation, safe identity fallback | Protocol correctness does not measure slow-client exhaustion, QUIC connection capacity, or load-balancer compatibility | Five-check live transport verifier, repeated after compression fix |
| Web prototype | Production build, responsive viewport checks, explicit sample-data disclosure | No backend-connected client workload, virtualized large-catalog measurement, offline storage pressure, or low-end-device profile | 64 browser checks; no native or low-end hardware claim |
| Deployment | Multi-stage non-root image recipe, compose health checks, loopback-published dependencies, secret-injected environment | No orchestration, horizontal replica, resource requests/limits, backup scheduling, or autoscaling qualification | Cold start, restart, upgrade and rollback on podman 5.7.0 recorded in `evidence/campaign-2026-11-20/containers/`; docker-compose parity asserted at file semantics only |

## Highest-priority engineering actions

1. Add repeatable large-catalog fixtures and record PostgreSQL query plans, lock waits, latency percentiles, memory, and throughput before changing identity-locking semantics. **Done for the read path and the write path** — plans in `evidence/campaign-2026-11-20/scalability/`, percentiles, advisory-lock waiters, pool waits, cgroup/RSS and throughput in `evidence/campaign-2026-11-20/load/`, lock-scope throughput in `…/lock-scope/`, separate-process write throughput in `…/multiprocess/`. Not done: any of it against a ratified objective, or with ingest concurrent to reads.
2. Replace whole-work delete/reinsert and per-work child loading with measured set-based writes and batched reads while preserving provenance and atomic identity behavior. **Half done:** batched reads landed and measured 3.6×; the delete/reinsert write path is unchanged.
3. Add keyset pagination for stable large-result traversal; keep offset compatibility only where required. **Done, and measured to be neutral on latency** — see the finding above. It is kept for traversal stability.
4. Introduce durable scan scheduling before promising replica-safe or restart-safe background work. **Partly done:** terminal job records survive restart; the queue itself is still process-local.
5. Make pool and queue limits deployment-configurable with validated bounds and safe defaults; coordinate them with PostgreSQL/Redis capacity rather than increasing them blindly. **Pools done and validated; queue limits still constants.**
6. Run multi-client content, OPDS, KOSync, vector-search, and remote-source load tests, including slow clients, cancellation, provider throttling, and service restart. **Partly done:** OPDS feed build and catalog read paths load-tested at 1–64 clients, and a scan-job SIGKILL/restart scenario run in `evidence/campaign-2026-11-20/load/`. Content delivery, KOSync/Redis, vector search and remote-source connectors remain unmeasured under any client count, and no slow-client or provider-throttling test exists.
7. Test actual containers and orchestration, then define resource requests, limits, disruption behavior, backup objectives, and rollback procedures from observed data. **Containers done** (cold start, restart, upgrade, rollback); requests/limits, backup objectives and autoscaling are not.

## Measured on a 50,000-work fixture (2026-11-20)

First large-cardinality query evidence for this repository. Fixture: 50,000 works,
100,000 editions, 200,000 identifiers, 300,000 files, 100,000 evidence rows, seeded
in 22.7 s. Every figure below was taken **inside the PostgreSQL container's own
network namespace**, and each shape was cross-checked against the server's own
`EXPLAIN (ANALYZE)` time. Raw logs and plans:
`evidence/campaign-2026-11-20/scalability/`.

| Measurement | Median | What it shows |
|---|---|---|
| `Search` browse, offset 0 / limit 20 | 9.30 ms | dominated by the per-request `count(*)` (6.73 ms alone) |
| `Search` browse, offset 49,000 / limit 20 | 12.44 ms | OFFSET adds ~3.6 ms at 49k depth; grows with depth |
| `Search` by cursor, depth 40,000 | 11.27 ms | **no measurable gain over OFFSET** — see the finding below |
| Page child load, one `GetWork` per work ×100 | 76.31 ms | the N+1 shape this replaced |
| Page child load, batched `getWorks` ×100 | 21.34 ms | **3.6× faster**; fewer round trips, same rows |
| 200-value parameterized `IN` list | 3.29 ms | equivalent to `= ANY($1)` (3.12 ms): list shape is not the cost |
| 200 generated rows, no table touched | 0.83 ms | row transfer itself is cheap |
| 200-value `IN` matching nothing | 1.39 ms | binding 200 parameters is not the cost either |
| `Search` with `State` filter, offset 40,000 | 95.31 ms | `EXISTS` over editions, unindexed for this shape |
| `Search` with `%text%` predicate | 773.70 ms | unanchored `ILIKE` across four columns plus two `EXISTS` |

### Finding: the keyset cursor does not make deep pages faster on PostgreSQL

`TestLiveScaleCatalog` measures the parts of a deep request separately, and the
cursor does not win any of them. The raw id query at depth 40,000 costs 3.59 ms by
`OFFSET` and 4.12 ms by cursor; the whole `Search` costs 11.61 ms and 11.27 ms. The
plan explains why (`evidence/campaign-2026-11-20/scalability/keyset-predicate-plan.txt`):

```
->  Index Only Scan using works_updated_id on works w (actual time=4.714..4.717 rows=20 loops=1)
      Filter: ((updated_at < …) OR ((updated_at = …) AND (id > …)))
      Rows Removed by Filter: 40001
```

The cursor predicate has to be written as an `OR` because the ordering mixes
directions — `updated_at DESC, id ASC` — and PostgreSQL has no row-comparison
syntax for mixed directions. An `OR` cannot become a single btree start point, so
the scan walks 40,001 index entries and discards them, doing the same work `OFFSET`
does. Neither figure is stable enough to rank one above the other: a direct `psql`
run of the same pair gave 4.4 ms for the cursor and 7.7 ms for the offset, and the
repeated medians gave the reverse. What both agree on is `Rows Removed by Filter:
40001` versus 40,020 rows scanned — the work is the same order as depth either way.
The keyset is not broken; it is just not the thing that was slow.

What the cursor does provide is traversal stability: an offset page taken mid-scan
can repeat or skip a work whose `updated_at` moved, and a cursor cannot. It is also
the shape OPDS continuation and a syncing client need. It is kept for those reasons.
Its latency benefit is recorded as **zero at this scale**, and the only change that
would make it non-zero is redefining the browse tiebreak so the ordering is
single-direction (`updated_at DESC, id DESC`), which turns the predicate into a
seekable row comparison. That is a change to the documented result order of an
existing endpoint, and it buys about 3.6 ms per request at 50,000 works, so it is
not taken. The cost it would remove grows linearly with depth, which is the bound a
future decision should be made against.

### Finding: where a browse request actually goes

At 50,000 works and limit 20, roughly 6.7 ms of an ~11 ms request is the
`count(*)` `Search` runs on every call, and about 4 ms per 20 works is loading the
child editions, identifiers and files. Neither is a pagination concern. The batched
child load already cut that second term by 3.6×; the count is deliberate and is
discussed under **Deliberately not done**.

### Finding: the plan for the browse page flips on visibility-map freshness

`ORDER BY updated_at DESC, id` has a matching index (`works_updated_id`) and the
projection is covered by it, so the ideal plan is an index-only scan. Immediately
after the bulk load the planner instead chose a sequential scan and an **external
merge sort spilling 1,984 kB to disk** — 27.29 ms — because no heap page was marked
all-visible yet, so every index page was priced with a heap fetch. After
`VACUUM (ANALYZE)`: 6.99 ms, `Heap Fetches: 0`, same query, same schema, same data.

This is not a schema defect; it is a deployment property with an operational
consequence. The moment a library is largest is right after a full scan, which is
exactly when the index is least usable. A large ingest should be followed by
`VACUUM (ANALYZE) works`, or the first browse requests after it pay a full sort.
Both plans are retained in `scale-plans.json`
(`page_offset_40000_pre_vacuum`, `page_offset_40000_post_vacuum`) rather than only
the flattering one, and `TestLiveScaleCatalog` asserts against the post-VACUUM plan
because asserting on the pre-VACUUM one would test planner mood, not the schema.

### Finding: latency measured through a container forwarder is not query latency

The first attempt at these numbers used the loopback-published port
(127.0.0.1:34320) from the host. Some 200-row shapes reported ~208 ms there while
PostgreSQL reported 3–5 ms for the identical statement, and the same shapes
measured 0.5–4 ms inside the container. `ss -tin` showed the bytes held in the send
queue of the socket served by `pasta`, rootless podman's userspace forwarder.

It is intermittent and it is per-connection. Reproduced under CPU load on a
16-core host (artifacts: `host-through-pasta-under-load.log`,
`host-under-load-scale.log`), the same shapes measured 3–6 ms on the same port
minutes earlier with the host quieter — and in one run the two tests disagreeing at
the same moment, one pool stalling at 205 ms while another on the identical path
reported clean. Single-packet responses (a 20-row page, 200 generated rows) were
never affected; the stall attaches to multi-packet responses. A ~200 ms quantum for
a sub-5 ms statement is not a query property under any interpretation.

What that does to a conclusion matters more than the number itself: through pasta
the batched loader measured 232 ms against the N+1's 104 ms, i.e. **the artifact
inverts the finding it is attached to**. `TestLiveScaleBatchPhases` therefore
refuses to report ratios when its own transport is unfaithful: it compares each
client wall-clock median with the server's account of the same statement across six
shapes and `Skip`s the comparison out loud when they disagree by more than 5× + 25 ms.
`TestLiveScaleCatalog` runs the same probe and records its verdict in the payload as
`measurement_path`. That label is advisory, not a guarantee: because the stall is
per-connection, a clean label does not prove every statement in the payload crossed
a clean path. The in-container log is the authoritative artifact for these numbers,
and the host logs are kept as the record of what the wrong path reports.

### Deliberately not done

- **No `pg_trgm`/GIN index for the 731 ms text search.** It would need
  `CREATE EXTENSION pg_trgm` inside a catalog migration. No catalog migration
  requires an extension today, and adding one would make migration fail outright on
  a managed or restricted PostgreSQL where the role is not superuser. Trading the
  ability to start at all for a search speedup on a single-user server is not a
  call to make implicitly; the number is recorded here so the trade can be made
  explicitly.
- **`count(*)` still runs on every page, including cursor pages.** Skipping it would
  make `Page.Total` ambiguous between "empty result" and "not computed", and
  `Total` is the field OPDS `totalResults`/`numberOfItems` is built from. The cursor
  exists precisely so a client can page without the total; the cost stays for
  consumers that still ask for it.

## Measured under concurrent load (2026-11-20)

Same fixture, same container, 1 → 8 → 32 → 64 simultaneous clients, all figures
`in_container_clean`, zero errors at every level of every workload. Full method,
resource envelope and caveats: `evidence/campaign-2026-11-20/load/README.md`.

| Workload | req/s across the ladder | p99 ms |
|---|---|---|
| repo offset browse page | 117 → 638 → 994 → **992** | 9.1 → 27.7 → 49.4 → 126.8 |
| repo deep keyset page | 93 → 483 → 801 → 822 | 12.0 → 34.0 → 59.3 → 126.6 |
| repo `%text%` search | 0.9 → 6.5 → 8.9 → 8.7 | 1101 → 1449 → 5548 → **12043** |
| HTTP `/v1/works` browse | 102 → 540 → 787 → 897 | 12.3 → 25.3 → 76.1 → 114.4 |
| HTTP `/v1/works` text search | 1.0 → 6.0 → 9.4 → 9.5 | ~1.0 s → ~11.0 s |
| HTTP OPDS 2.0 feed (20 entries) | 82 → 543 → 776 → 825 | 17.7 → 27.8 → 71.2 → 127.6 |

Three things follow, and one does not:

- **Indexed reads knee between 32 and 64 clients** at ~1000 req/s (repo) / ~900
  (HTTP) while p95 doubles. The 20-connection pool is **not** the limiter: a
  pool=64 probe measured 942 req/s against pool=20's 992 with zero pool waits at
  every browse level, and the sampled wait event during those phases was plain CPU.
  Database execution is the saturation resource.
- **The unanchored text search is a different object.** It plateaus near 9 req/s at
  32 and 64 clients, costs ~1.0–1.5 CPU-seconds per query, and at 64 clients drives
  p99 to 12.0 s. Pool starvation stacks on top of that wall (728 waits, ~574 s of
  total wait time) because each query outlives a 20-slot lease by an order of
  magnitude. This is the same predicate the single-connection 731 ms figure came
  from, now with its concurrency behaviour attached.
- **OPDS rendering is not a second bottleneck** at this page size: feed build costs
  the same as the underlying page query (825 vs 900 req/s at 64 clients).
- **None of this is a service-level objective.** The knees are candidate inputs for
  defining one; no target has been ratified, so no number here passes or fails.

Process envelope at these workloads: harness RSS 23–29 MB (catalog layer) and 58–91
MB (HTTP layer, server plus co-resident clients), peak PostgreSQL backends 26 at the
default pool and 75 in the pool=64 probe against `max_connections=100`, temp-file
spill observed only during the write-contention phases (44 files, ~87 MB), zero
deadlocks.

## Finding (2026-11-20): the identity lock was the serializer, and its scope was too wide

The contention harness measured concurrent `MergeWork` at 0.98x of the sequential
baseline while the identical transaction body with the lock omitted reached 5.1x.
That attributed the ceiling to `pg_advisory_xact_lock(731452)`: one key taken by
every catalog write, so a scan of one source waited behind an unrelated scan of
another. The fix is not to remove the lock — the unlocked arm silently lost 7 of
8 concurrently merged editions, with no error reported — but to take one lock per
identity a write can actually touch: the work whose editions are rewritten, each
edition whose ownership is checked, and each source namespace whose file rows are
claimed, moved, or pruned.

| arm (same body, same run, in-container) | ops/s |
| --- | --- |
| one global key (previous behavior) | 271.94 |
| sharded identity keys (shipped) | **1592.21** |
| no advisory lock (unsafe; ceiling) | 1789.00 |

5.85x from lock scope alone, at 89% of the unlocked ceiling; the independent
contention phases went from 0.98x to 5.14x of sequential and reconcile from parity
to 2.73x. Content identity is shared across sources by design, so two scans of the
same bytes from different locations still collide on the work and edition keys and
serialize exactly as before. `evidence/campaign-2026-11-20/lock-scope/summary.md`
carries the invariants, the paired needles, and a mutation gate that removes one
key class at a time and requires the dependent test to fail.

The bounded retry added alongside it absorbs only `40001`, `40P01` and `23503`.
`23505` is left loud on purpose: a unique violation would mean the key set missed
two writers sharing a row, and retrying that would turn a lock-scope bug into a
scan that quietly succeeds.

## Finding (2026-11-20): the lock-scope result holds across processes, and so does the serialization the design keeps

Single-process concurrency proves an argument, not a lock: within one process the
pool can serialize work on its own, and `SELECT … FOR UPDATE` already protects a
read-modify-write on rows that exist. `TestCatalogMultiProcessMerges` therefore runs
as separate OS processes against one PostgreSQL, each with its own pool, released as
simultaneously as a database barrier can manage, in three arms — works nothing else
touches, works every process merges, and one file row per path under a single source
id. Five post-barrier runs (two at 4 processes, three at 8) are in
`evidence/campaign-2026-11-20/multiprocess/`.

- **Safety is equal to the old global key and unequal to no lock.** `sharded` and
  `global` both ended every run with every concurrently merged edition present
  (480/480 at 8 processes), no duplicate `(source_id, path)`, no orphans. The
  `unlocked` variant lost 8 of 480 editions in all three 8-process runs while
  reporting **zero errors**, which is the failure mode the lock exists to prevent.
- **Throughput is better than the global key on every run**, by 1.53× to 4.07×
  (median 2.48×). The spread is real and is recorded rather than averaged into a
  single claim; the direction is what replicated.
- **The source-namespace key still serializes, as designed and now as measured.** On
  the third arm all writers share one `source_id`, so they contend on one `s:` key
  and land within run noise of the global key — in one of three 8-process runs
  slightly behind it (0.46×). The gain is on the arms where writers do not share a
  source. Two scans of identical bytes from different locations still collide on the
  work and edition keys for the same reason.

The first two runs of this harness were refused by its own gate: `unlocked` verified
clean at 240/240, which meant the multi-process assertions had no detection power at
all (recorded as DEFECT-004). Staggered process startup let the first writer create
each contended row before the others began, leaving only row locks to cover a window
that no longer existed. The barrier exists because the unsafe window is specifically
the *first* write to an identity that does not exist yet.

This does not close the gate below. It is one container, one server binary per
process, synthetic uniform works, and write traffic only: no ingest running while
reads and OPDS feeds are served, no HTTP surface, no real title distribution, no
latency percentile target, and no memory or CPU envelope.

## Finding (2026-11-20): vector `Search` could never use its HNSW index — DEFECT-005

`ai_memories` has an HNSW index on `embedding vector_cosine_ops`, and `Search` ordered
by `embedding <=> $2, id`. pgvector can satisfy an ordering of exactly
`embedding <=> const`; adding a secondary key inside that clause makes the ordering one
no index can supply, so the planner had no option but to read every stored embedding and
top-N sort it — while still maintaining the index on every insert. The tiebreak is what
makes results deterministic for clients that paginate or diff them, so it could not just
be deleted; it moved outside the subquery, with `OFFSET 0` to stop the planner merging
the subquery away and pushing it back in.

One run, 50,000 rows × 384 dimensions, k=10, 100 sequential queries then a 1/8/32-client
ladder, executed inside `helixqual-pg` against the server's own `searchSQL` constant on a
`search_path`-shadowed table whose columns and indexes are compared against
`public.ai_memories` in the catalogs at run time (run tag `20261120T220611Z`,
`evidence/campaign-2026-11-20/vector/`):

| statement | plan class | p50 | p99 | req/s |
| --- | --- | --- | --- | --- |
| released, no override | `seq_scan_topn` | 28.3 ms | 31.8 ms | 34.9 |
| released, scans denied | `other_index_then_sort` | 42.8 ms | 50.7 ms | 23.1 |
| fixed, no override | `ann_index` | **1.86 ms** | 7.0 ms | 435 |
| fixed, index forced | `ann_index` | 1.65 ms | 3.3 ms | 563 |
| fixed, exhaustive (ground truth) | `seq_scan_topn` | 29.8 ms | 39.2 ms | 32.6 |

15.2× on p50 against the statement as released, and the index path is now what the
planner picks **without** being forced. Under the 32-client ladder the fixed statement
held 2,748 req/s at p99 69.7 ms where the released one managed 172.8 req/s at p99
399.9 ms. Denying scans made the released statement *slower* (42.8 ms), which is the
signature of an index that can only be used for the filter, not the ordering.

**What changed semantically.** The returned *set* is now the index's approximate top-k
instead of the exact top-k; the returned *order* is unchanged and was verified stable
across five identical runs. The cost of that trade is recall, and it is not small on this
fixture: **recall@10 was 0.114** against the exhaustive scan of the same rows.

**That number is a property of the data and of a setting nothing in this product
exposes.** pgvector widens HNSW search with `hnsw.ef_search` (compile-time default 40),
and `Search` never sets it. Sweeping the width over the same fixture, with the value read
back off the session before measuring:

| `hnsw.ef_search` | recall@10, 50k × 384 | recall@10, 5k × 64 |
| --- | --- | --- |
| 40 (what ships) | 0.1025 | 0.8725 |
| 100 | 0.215 | 0.99 |
| 400 | 0.5475 | 1.0 |

Recall rises monotonically with width at both shapes, so the harness is comparing real
neighbourhoods and the limiter is the width, not the fix. Uniform random vectors on a
high-dimensional sphere are the adversarial case for a proximity graph — real embeddings
cluster — so **0.114 must not be read as "Helix's memory search returns 11% of the right
answers"**. What it does establish is that the product has no way to trade search width
for latency at all, and that at 384 dimensions the shipped width is far from the
quality/latency knee (at `ef_search=400`, recall is still only 0.55 here). No client-visible
recall target exists to judge that against; that absence is a gate item, not a finding.

**Cold statistics.** While the table was being seeded, the fixed statement classified
`seq_scan_topn` at 1,000, 5,000, 20,000 and 50,000 rows, and only became `ann_index` after
`VACUUM ANALYZE`. The planner's choice here depends on row estimates, so a freshly loaded
memory table can be scanned exhaustively until it is analyzed. Recorded as an observation
from a loading sequence, not a measured production steady state: autovacuum would analyze
well before 50k rows accumulate, and that was not tested.

**Why these figures are trustworthy when the harness itself is the thing being trusted.**
Three independent layers, added because the first draft of this harness reported a
sequential scan as an ANN result:

1. *Schema*: the bench table's column list, types, index set and index names are compared
   against production's in the catalogs, and the comparison is what fails the run — not a
   comment asserting they match.
2. *Session*: every profile's planner settings are read back off every pool connection
   (all held simultaneously, so the pool is forced to actually open them) and compared
   against what the profile's label requires.
3. *Plan*: a plain range query that must sequential-scan under the default profile and
   index-scan under the forced one, at any cardinality, plus a classifier positive control
   requiring the exact and index profiles to disagree on the vector statement.

Nine mutations of the harness were then run against it
(`evidence/campaign-2026-11-20/vector/needle_gate_results.json`): index never created,
forced profile neutralized, ladder secretly serial, queries asking for k−1, fixture
collapsed to one repeated vector, `search_path` never applied, column list drifted, index
opclass switched to L2, and serial ladder *combined with* a sampler that counts idle
backends. All nine were detected (`all_detected: true`), each by a needle meant to catch
it, and the tree was restored byte-for-byte afterwards. Two things this exposed about the
harness rather than the database: the forced-profile mutation survived the first version
because at 5,000 rows the planner takes the index on its own, so the plan class could not
tell "forced" from "default" — layers 2 and 3 above exist because of it; and the
`pg_stat_activity` backend count is corroborating evidence only, since it counts every
active backend in the database including the harness's own other pools, which is why a
serialized ladder was caught by the throughput needle and not by the backend needle in the
final gate run.

**Scope.** One container, co-resident harness, synthetic uniform vectors, one scope
matching every row, no deletes, no concurrent writer during the ladder, no real embedding
model, and pgvector's compiled-in index parameters. The latency comparison and the
plan-class results are the load-bearing parts; the absolute req/s figures describe this
process placement.

## What this still does not establish

The release gate below stands unchanged. Every measurement here comes from one
disposable container on one 16-core host, against a synthetic uniform 50,000-work
fixture, with the load harness co-resident with the database. Observed RSS, goroutine
and backend counts describe those processes; they are not a container resource
request, and no memory or CPU limit was enforced anywhere. Nothing says anything about
real title/author distributions, feed generation over large result sets (the OPDS
figures are 20-entry pages), replica behavior, or any service-level objective.
Nothing here may be cited as enterprise-scale proof.

Concurrency and lock contention are no longer in that list of unknowns, and neither
is the multi-process case in the narrow sense the finding above defines it: several
processes of the same binary, one catalog, one container, synthetic disjoint and
contested fixtures, write traffic only, and relative rankings between locking
disciplines measured in the same run. What is still unmeasured is the shape a gate
would need — ingest running while reads and OPDS feeds are served, real title
distributions, content delivery and KOSync/Redis under any client
count, TLS/HTTP-2/HTTP-3 termination under load, a second database or application
node, and any target latency or throughput objective at all.

## Required release gate

ATM-027 can close only after representative workloads and target service-level objectives are defined, the above hotspots are addressed or explicitly bounded, and repeated machine-readable load results pass in the intended deployment topology. Current functional PASS artifacts must not be cited as enterprise-scale proof.
