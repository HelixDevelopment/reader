# Multi-process lock-scope qualification (ATM-027)

## What this closes

`lock-scope/` measured the sharded identity lock with several goroutines inside
**one** process. A single-process measurement cannot prove anything about a lock
whose purpose is coordinating separate writers: within one process the Go side of
the pool can serialise work, and a `SELECT … FOR UPDATE` that matches an existing
row protects a read-modify-write across processes on its own. The open question the
scalability review named was *several server processes against one catalog*. This
directory answers it for the write path.

## What was measured

`TestCatalogMultiProcessMerges` runs as separate OS processes, each launched by its
own `podman exec` of the same test binary inside `helixqual-pg`, each with its own
`sql.DB`, all against one PostgreSQL. Three arms per process:

| Arm | Shape | What it stresses |
|---|---|---|
| `disjoint` | every process writes its own works | work/edition keys that never collide — should scale with processes |
| `contested_work` | N processes merge the same N works | the `w:`/`e:` keys, i.e. delete-reinsert of one work's editions |
| `shared_file_row` | N processes claim one file row per path under **one** source id | the `s:` key — the deliberate serialization point |

Each process runs 60 ops per arm. A `verify` role runs afterwards in a separate
process and re-checks the database, not the in-memory view: no disjoint work
missing, every contended work holding `processes × ops` editions, no `(source_id,
path)` with more than one file row, no orphaned file rows, exactly `pool` file rows
in the shared source.

Three variants of `identityLockKeys`/`lockIdentities` are built from real source,
one at a time, and restored with a SHA-256 check:

- `sharded` — the shipped code.
- `global` — one `pg_advisory_xact_lock(731452)` for every catalog write (the previous behaviour).
- `unlocked` — no advisory lock at all. Must lose writes or the assertions are worthless.

## Finding (DEFECT-004): the multi-process needle was blind until the start was simultaneous

The first two runs of the `unlocked` variant **passed** verification: 240/240
editions present with the lock removed, in both. That is not safety, it is a test with
no detection power. The runner's own gate refused them —
`variant unlocked: NEEDLE BLIND …`, `multiprocess status=1` for run tags 207361 and
207444 — which is the only reason this was caught instead of reported as a pass.

The cause: worker processes were launched by a shell loop, so the first starter
created each contended work before the last one began; from then on a plain row lock
under `SELECT … FOR UPDATE` covered the read-modify-write, which is exactly the case
the identity lock is not needed for. The window the identity lock exists to cover is
the **first write to an identity that does not exist yet**, and reaching it requires a
simultaneous start.

`mpBarrier` now makes every process wait in the database — each inserts a row and
polls until all `processes` rows are visible — immediately before arms 2 and 3. It is
a DB barrier rather than a file or signal one, because the processes are separate
`podman exec` invocations sharing nothing but PostgreSQL. Arm 1 runs before the
barrier so its scaling figure is not padded by barrier wait.

From run tag 207537 on, the same `unlocked` arm loses writes in every run: 3 of 240
editions at 4 processes, 8 of 480 at 8, always concentrated on one contended work,
with **zero reported errors** in all of them. That last clause is the point of the
needle: the destructive path fails silently, so only a post-hoc row count can catch
it.

## Results

Throughput is the wall-clock of the whole worker phase (all processes running), from
`results.json`; per-arm sums are the runner-side totals of each process's own arm
rate, from `per_arm_summary.json`. Every number here is `measurement_path:
in_container_clean` — taken inside the PostgreSQL container, never through `pasta`.

| Run | Procs | Variant | Merges | Wall (s) | Aggregate ops/s | Verify |
|---|---|---|---|---|---|---|
| 207537 | 4 | sharded | 720 | 4.866 | 147.97 | pass, 240/240 |
| 207537 | 4 | global | 720 | 12.317 | 58.46 | pass, 240/240 |
| 207537 | 4 | unlocked | 720 | 2.766 | 260.30 | **fail, 237/240** |
| 207640 | 4 | sharded | 720 | 5.729 | 125.68 | pass, 240/240 |
| 207640 | 4 | global | 720 | 10.970 | 65.63 | pass, 240/240 |
| 207640 | 4 | unlocked | 720 | 3.639 | 197.86 | **fail, 237/240** |
| 207704 | 8 | sharded | 1440 | 9.348 | 154.04 | pass, 480/480 |
| 207704 | 8 | global | 1440 | 38.088 | 37.81 | pass, 480/480 |
| 207704 | 8 | unlocked | 1440 | 10.564 | 136.31 | **fail, 472/480** |
| 207824 | 8 | sharded | 1440 | 19.561 | 73.62 | pass, 480/480 |
| 207824 | 8 | global | 1440 | 48.563 | 29.65 | pass, 480/480 |
| 207824 | 8 | unlocked | 1440 | 11.617 | 123.96 | **fail, 472/480** |
| 207982 | 8 | sharded | 1440 | 20.543 | 70.10 | pass, 480/480 |
| 207982 | 8 | global | 1440 | 31.515 | 45.69 | pass, 480/480 |
| 207982 | 8 | unlocked | 1440 | 5.746 | 250.61 | **fail, 472/480** |
| 209138 | 8 | sharded | 1440 | 14.101 | 102.12 | pass, 480/480 |
| 209138 | 8 | global | 1440 | 25.185 | 57.18 | pass, 480/480 |
| 209138 | 8 | unlocked | 1440 | 5.723 | 251.62 | **fail, 473/480** |

Sharded exceeded the single global key in all six post-barrier runs; the margin is
**1.53× to 4.07×** (median 2.20×), and it is not stable enough to quote as one
number. What is stable across every run is the direction and the safety outcome:
sharded and global both preserved all writes, unlocked did not. Run 209138 is the
authoritative complete set for the deadlock observation below (worker and verify
`.jsonl` payloads plus logs); 207824 and 207982 also carry full payloads, and the
earlier tags retain logs only.

Per-arm sums for the four 8-process runs, sharded vs global:

| Arm | sharded Σ ops/s | global Σ ops/s | Ratio |
|---|---|---|---|
| `disjoint` | 1199.8 / 583.8 / 1146.9 / 1832.4 | 342.7 / 220.4 / 337.4 / 459.0 | 3.5× / 2.6× / 3.4× / 4.0× |
| `contested_work` | 132.3 / 41.9 / 47.2 / 79.2 | 15.3 / 12.6 / 19.1 / 23.8 | 8.6× / 3.3× / 2.5× / 3.3× |
| `shared_file_row` | 126.3 / 105.2 / 66.0 / 80.0 | 128.2 / 79.5 / 143.3 / 187.1 | 1.0× / 1.3× / **0.46×** / 0.43× |

The `shared_file_row` row is not noise. All eight processes write through **one
source id**, so the shipped code takes one `s:<sourceID>` advisory lock and
serialises — the same behavior the single-process lock-scope finding predicted for
shared identity, now observed across processes. Sharded is within run-to-run noise of
global on that arm; the third run has global ahead, and that is recorded rather than
dropped. The gain is on the arms where writers do not share a source.

**Deadlocks: zero, measured rather than assumed.** Run 209138 sampled
`pg_stat_database.deadlocks` before and after every arm in every process. All 24
per-process deltas are 0 and the run-wide span is 0 for all three arms of all three
variants. Because a retried `40P01` never surfaces as a worker error, `errors=0` on
its own proves nothing about the retry path; this counter is the trace that does. The
reading is bounded: the counter is database-wide and cumulative and the container is
shared, so it is a lower bound, and it says that in this shape the ordering discipline
(advisory locks first, deterministic key order) avoided deadlocks instead of relying on
retry to absorb them. It does not show the retry path works — `TestCatalogIdentityLockScopeThroughput`
and the classification test cover that logic, and a deadlock has never been observed in
any run here.

For reference the unlocked arm's `shared_file_row` sums are 404.9 / 474.7 / 1006.3:
several times the locked code, while losing edition writes elsewhere. Removing the
lock always "wins" throughput; that is why the ceiling is only reported next to the
loss it causes.

## Limits of this evidence

- One container, one PostgreSQL, one host. Aggregate throughput on the contended arms
  does not grow from 4 to 8 processes under either locking discipline — this measures
  relative cost of lock scope, not a scaling curve.
- Synthetic works with uniform, seeded content. No ingest runs concurrently with
  reads, no HTTP surface, no real title distribution.
- The barrier is a best-effort simultaneous start, not a guarantee: it aligns the
  processes at a DB-visible instant after connection setup, so a process still in
  connect or in pool warm-up can join late and shrink the window for that run.
- Arm 3 measures the source-key serialization only; it does not exercise
  `ReconcileSource` concurrently with merges (that interleaving is covered by
  `TestCatalogReconcileExcludesSameSourceMerge`, single process).
- Nothing here is a service-level objective, and none is claimed. It does not close
  ATM-027's release gate.

## Harness defects found and fixed while producing this

Recorded because the artifacts are append-only and readers need to know which runs
are complete:

1. **DEFECT-004** — the blind multi-process needle, described above. Fixed by
   `mpBarrier`; runs 207361 and 207444 predate the third arm and/or the barrier and
   are superseded (they remain in this directory).
2. `run_multiprocess.sh` copied worker payloads back with a shell glob handed to
   `podman cp`, which takes literal paths. Every per-worker and per-verify `.jsonl`
   was therefore silently dropped (the `|| true` swallowed it). Fixed to name each
   file, and to copy the verify log into this directory. Runs 207537, 207640 and
   207704 have worker/verify **logs** but no `.jsonl`; their verify outcome is the
   `VERIFY …` line in `run.log`. Runs 207824 and 207982 are complete.
3. `summarize.py` first read `contended_works_lost` instead of `contested_works_lost`
   and reported the loss as `null`; fixed, and it now records which payload key each
   arm figure came from (`payload_key`) so a renamed field cannot silently read as 0.

## Reproducing

```sh
PROCS=8 OPS=60 bash evidence/campaign-2026-11-20/multiprocess/run_multiprocess.sh
python3 evidence/campaign-2026-11-20/multiprocess/summarize.py
```

The runner requires `helixqual-pg` running, builds the catalog test binary with
`go test -c`, mutates and restores `postgres.go` itself (and fails if the restore is
not byte-identical), and refuses a run where the `unlocked` variant verifies clean:

- `NEEDLE BLIND — verify passed with the lock removed, so the multi-process
  assertions have no detection power`

## Files

- `run_multiprocess.sh` — orchestration, mutation/restore, hashing, gate.
- `run_multiprocess.sh.pre-copyfix` — the runner as first written, kept for the diff.
- `results.json` — one row per variant per run, append-only, with `run_tag` and `sync`.
- `per_arm_summary.json` — derived from the payloads by `summarize.py`.
- `<variant>-<tag>-worker-<i>.{log,jsonl}`, `<variant>-<tag>-verify.{log,jsonl}` — raw.
- `run.log` — the runner's own log for every run in this directory.
