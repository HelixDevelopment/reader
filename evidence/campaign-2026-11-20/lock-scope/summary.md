# Catalog identity lock scoping — ATM-027 hotspot 3

Date: 2026-11-20. Code: `server/internal/catalog/postgres.go`.
Tests: `server/internal/catalog/lock_scope_test.go`.
Measurement: `../load/catalog_lock_scope.jsonl`, `../load/catalog_contention_sharded.jsonl`.
Detection gate: `mutation-gate.log` produced by `run-mutation-gate.sh`.

## What was changed

`UpsertWork`, `MergeWork` and `ReconcileSource` each took
`pg_advisory_xact_lock(731452)` — one key for the whole catalog. Every writer
waited behind every other writer regardless of which rows it touched, which the
phase-1 measurement had already attributed 0.98x of the sequential baseline to.

They now take one transaction-scoped lock per identity the write can touch:

| key | row set it protects |
| --- | --- |
| `w:<work id>` | `upsertTx` deletes and rewrites every edition of one work |
| `e:<edition id>` | the edition ownership check and its insert |
| `s:<source id>` | file rows keyed by `(source_id, path)`, including re-homing and reconcile prune |

All keys are derived from the incoming work before the transaction starts, so no
writer has to guess what it will touch. Advisory locks are acquired before any
row lock in every path, which removes the advisory-versus-row wait cycle. The
`e:` key is the one that cannot be dropped: content identity is shared across
sources by design (`ingest.workIdentityID` anchors a work to its bytes), so two
scans of the same book from different sources collide on `w:` and `e:` and
serialize exactly as before, while two scans of different books do not.

`inTx` now retries up to four attempts, and only for `40001`, `40P01` and `23503`
— the codes that mean a row moved underneath an idempotent, content-keyed write.
`23505` is deliberately **not** retried: a unique violation would mean the key
set failed to exclude two writers that share a row, and swallowing that would
hide a real defect behind a successful-looking scan.

Rolling upgrade note: the sharded locks use the two-argument advisory form
(class `731452`, object `hashtext(key)`), which shares no key with the old
single-argument `731452`. Two binaries of different versions must not run
against one catalog at the same time; the shipped deployment is one image behind
one compose stack, upgraded and rolled back by tag.

## Measured effect (in-container, DSN 127.0.0.1:5432, `measurement_path=in_container_clean`)

Same transaction body in all three arms, 8 clients x 40 merges of disjoint works,
one process, one connection pool, one machine, one run:

| arm | ops/s | errors |
| --- | --- | --- |
| `legacy-global` (one lock for all writers) | 271.94 | 0 |
| `production` (sharded identity locks) | **1592.21** | 0 |
| `unlocked` (no advisory lock — unsafe, ceiling only) | 1789.00 | 0 |

**5.85x** from lock scope alone, reaching **89%** of the unlocked ceiling. The
ratio is reported against the ceiling as well as the floor so the number cannot
be read as if the lock were free.

Independently, the pre-existing contention harness rerun against the new code
(`catalog_contention_sharded.jsonl`, verdict recorded as
`parallel: sharded identity locks let concurrent disjoint MergeWork scale past
the sequential baseline…`):

- concurrent `MergeWork` 1513.56 ops/s = **5.14x** the sequential baseline of 294.51 (was 0.98x)
- unlocked control 1953.42 ops/s = 6.63x, the headroom still on the table
- concurrent `ReconcileSource` 1943.63 ops/s vs 713.25 sequential = **2.73x**
- 0 advisory waiters sampled, 0 ungranted lock peaks, 0 errors in every phase

## Why the invariants are believed, not asserted

Each safety claim is paired with a control that runs the identical body under a
different locking discipline, and a needle that must break:

| claim | test | needle result |
| --- | --- | --- |
| eight concurrent merges of one work keep all eight editions | `TestCatalogMergePreservesEveryWriterUnderIdentityLocks` | with locks removed: **6 of 6 rounds lost writers, worst 7 of 8 editions**, and **zero errors** — the corruption is silent |
| one file row per `(source, path)` under contested claims | `TestCatalogFileClaimByDistinctWorksKeepsOneRow` | with locks removed: **6 of 6 rounds** produced 6 writer errors instead of serialized claims |
| a contested edition claim is decided once, and the loser gets the catalog's ownership error rather than a constraint violation | `TestCatalogEditionOwnershipIsDecidedOnce` | 8 rounds per mode, both lock disciplines |
| reconcile cannot prune a file a same-source merge is moving | `TestCatalogReconcileExcludesSameSourceMerge` | 4 rows kept, 0 orphans, concurrent rewrite present |

`run-mutation-gate.sh` then removes one key class at a time from
`identityLockKeys` in the real source and requires the dependent test to fail:

```
CASE drop-work-key:    detected (exit 1)
CASE drop-source-key:  detected (exit 1)
CASE drop-edition-key: detected (exit 1)
CASE drop-all-locks:   detected (exit 1)
gate_status=0 restored_tests=pass sha256_verified=yes
```

The gate restores `postgres.go` on every exit path and verifies the restored file
against its pre-run SHA-256; `postgres.go.pre-mutation` is kept beside it.

## Limits of this evidence

- Throughput numbers are from the qualification PostgreSQL container on this host
  (4 cores available to it). They rank lock disciplines against each other in one
  run; they are not a capacity plan for a different machine.
- 89% of the unlocked ceiling is not 100%. Each remaining gap is the cost of
  taking the locks, which is the price of the invariant, not slack to be removed.
- The `s:` key serializes all writers of one source, so a single source scanned
  by several app instances still merges serially. That is intended: those writers
  genuinely share file rows.
- Deadlock between two writers whose *edition* sets interleave in opposite orders
  is theoretically reachable and is handled by retry, not prevented by ordering.
  No deadlock was observed in any run here (`errors=0` in all phases, 40P01
  absorbed silently would still leave `errors=0`, so the claim is that none was
  *reported*, not that none occurred).
- The retry loop cannot be distinguished from a lock miss by looking at the
  success count alone; that is why `23505` stays loud and why the mutation gate
  exists.

## Follow-up: the same claims tested across processes

Everything above was measured with concurrent goroutines in **one** process, which
cannot validate a lock whose purpose is coordinating separate writers: within one
process the pool can serialize work, and `SELECT … FOR UPDATE` already protects a
read-modify-write on rows that exist. `TestCatalogMultiProcessMerges` re-ran the
comparison as separate OS processes with a database barrier on the start, and the
sharded key stayed ahead of the global key in all five post-barrier runs (1.53× to
4.07×, median 2.48×) while the lock-free variant lost 8 of 480 concurrently merged
editions in every 8-process run. See `../multiprocess/summary.md`. The single-process
figures in this directory are retained and are not restated there.

The `errors=0` caveat above also carries, and it was open when this file was first
written: the multi-process workers reported no errors, which does not prove no
deadlock occurred, because `40P01` is absorbed by the retry. `TestCatalogMultiProcessMerges`
was therefore extended to sample `pg_stat_database.deadlocks` before and after every arm
in every process. Run `209138` shows 24/24 zero per-process deltas and a zero run-wide
span across all three arms of all three variants — a lower bound on a shared, cumulative
counter, but the trace `errors=0` could not provide. No deadlock has been observed in any
run in this directory or `../multiprocess/`.
