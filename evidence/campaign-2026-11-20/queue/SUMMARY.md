# Durable-queue load harness — Phase 1 (ATM-027)

Run 2026-11-21, `TestLiveLoadJobQueueDurability` in `server/internal/httpapi/load_jobs_test.go`,
compiled binary executed inside `helixqual-pg` against `helixqual-redis:6379`.
Payloads: `job_queue_durability.jsonl` (3 and 6 bursts), needles: `needles/needle_gate_results.json`.

## Why this exists

ATM-027's acceptance names three things: established limits, **durable queue behavior**, and
bounded query plans. The durability *semantics* were already covered functionally by
`jobs_journal_test.go` (7 tests, one job at a time). What was missing is the load dimension:
the bounds only mean something if a **simultaneous** burst is stopped by the admission check
rather than by arrivals happening to be slow. This harness is also the fourth harness this
campaign to need a start barrier for exactly that reason.

Redis, not miniredis: a journal latency figure taken against an in-process library would
describe the library, not the deployment.

## Measured — `in_container_clean`

96 submissions released at once per burst, `maxConcurrentScans=2`, `maxPendingScans=64`,
`maxRetainedScans=256`, 20 ms simulated scan.

| bursts | accepted | rejected `queue_full` | peak pending | peak running | journal saves | p50 / p95 ms |
|---|---|---|---|---|---|---|
| 3 | 192 | 32 per burst, every burst | 64 (= cap, never above) | 2 (= cap) | 576 = 3 per job | 0.34 / 26.0 |
| 6 | 384 | 32 per burst, every burst | 64 | 2 | 1152 = 3 per job | 0.33 / 6.7 |

- **Admission is bounded and it bites.** Exactly 64 admitted and 32 refused in every burst,
  deterministically — the rejection count equals the cap, which is the signature of an
  actual bound rather than of timing luck.
- **Drain is worker-bound, not Redis-bound.** ~670–690 ms per burst against a theoretical
  64 jobs / 2 workers × 20 ms = 640 ms. The journal round trips hide inside that.
- **Reconstruction holds.** A fresh `jobStore` on the same Redis restored every accepted scan
  the index still listed as `succeeded`, and rewrote none of them as interrupted.
- **Interrupted work is reported, at scale.** 64 records left non-terminal in Redis came back
  as `failed` with `scan interrupted by server restart`; none came back as `running`, none
  vanished.
- **Retention is a real cap, measured.** At 384 accepted the index pins at 256 and displaces
  128 older records; writing the 64 interrupted records displaces more. The harness reports
  `journal_index_before` / `journal_index_size` / `evicted_by_retention` /
  `displaced_by_retention` and treats displacement as documented behavior — a record the
  index no longer lists is not a lost record, and the journal write history is what proves it
  finished.

## Two harness defects the process caught, both mine

1. **The queue "drained" before it had.** The first version polled the store for a settled
   state and then asserted the journal held three writes per job. A job flips to `succeeded`
   under the store lock *before* its terminal journal write completes, so ~1 in 5 runs failed
   with `journal=[queued running] store="succeeded"` and no error logged anywhere. The
   product was fine; the harness was reading a moment that cannot be observed atomically.
   Fixed by waiting on the admission-to-run-return bracket (`pending.live == 0`), then
   re-verified 15/15 plus a `-race` run with the harness actually enabled — the earlier
   `-race` pass had skipped it.
2. **A bounded store is not a leak.** At 6 bursts, `pruneLocked` correctly evicts terminal
   records past 256, and my "every accepted job must still be in the store" check called that
   loss. Resolved against the journal's write history instead of presence.

## Needle gate — 6/6 fired (`needles/needle_log.txt`)

| case | mutation | caught by |
|---|---|---|
| `admission_unbounded` | `create()` stops counting pending | `admission is not bounded` |
| `workers_unbounded` | `run()` bypasses the semaphore | `the worker semaphore is not holding` |
| `terminal_write_lost` | successful scans skip their terminal write | `journaled fewer than 3 times` |
| `interrupted_not_reconciled` | `restore()` stops reconciling in-flight records | `restore reported 0 interrupted` |
| `index_unbounded` | index trimming removed | `indexed but not restored` |
| `serial_submissions_refused` | **harness**: submissions staggered instead of simultaneous | `nothing was rejected` |

Two notes on the gate itself. `index_unbounded` first reported NEEDLE_DID_NOT_FIRE while the
run was in fact going red: without trimming, the sorted set outgrows what `recent()` reads
back, so older records stay listed and become unreachable, and the harness caught that on the
reconstruction assertion before reaching the size assertion. The needle was wrong about *which*
message names the harm, not about whether the harm is detected — so the expectation moved to
`indexed but not restored`, which is the truer description. And the last case mutates the
harness rather than the product: if submissions stop being simultaneous, a passing bound check
would mean nothing, so that condition has to fail loudly.

Tree restored from `/tmp/needle-queue-backup-*` and byte-verified after every run; never from
git, because HEAD predates this campaign.

## What this does and does not license

Established here: the queue's admission and worker bounds hold under simultaneous bursts, the
journal keeps three writes per accepted job under that pressure, and a restarted process can
reconstruct what finished and what was interrupted.

**A real process kill is already covered elsewhere — do not read this file as the only
durability evidence, and do not read it as replacing it.**
`../load/restart_queue.sh` enqueues 12 concurrent scans against the live catalog, `SIGKILL`s
the server mid-queue and restarts it, asserting on persisted state: 7 queued + 2 running + 3
succeeded before the kill, 9 `failed` + 3 `succeeded` after, `records_lost: []`, terminal
records preserved. It labels itself `host_through_pasta_state_assertions_only`, which is
honest — it claims state correctness, not latency, so the forwarded host→DB path cannot
invalidate it. That harness proves the crash path across a genuine process boundary; this one
proves the bounds hold when arrival is simultaneous and the write volume is high. Neither
substitutes for the other.

Still not established: multi-instance contention on one Redis (two `helix-server` processes
sharing a journal), and any figure for p95 journal latency as a stable number — it swung
6.7–26 ms between two runs of the same build and is reported here as variance, not as a
service level.
