# Pool configuration — needle proofs and an incident record

Scope: the last unproven part of ATM-027's "make pool limits deployment-configurable"
item (`server/internal/config/config.go`, `server/internal/catalog/postgres.go`,
`server/internal/aimemory/postgres.go`, `server/cmd/helix-server/main.go`).
Generated 2026-11-20. Commands are reproducible from `../../../../../scripts`-relative
paths in the mutation script's own text below.

## What was already claimed, and what was actually true

`docs/scalability-review.md` recorded "Pool caps now deployment-configurable
(`HELIX_CATALOG_POOL_*`)". The code did read and validate the environment, and
`main.go` did pass the values to both constructors. Two things were false:

1. `TestWithPoolValidation` (catalog) had **no detection power**. It constructed a
   repository against `postgres://invalid.invalid:5432/none` and asserted only that
   `NewPostgres` returned an error. `sql.Open` succeeds against a bogus host and
   `PingContext` then fails, so the assertion was satisfied by the dial. Deleting
   the `maxOpenConns < 1` check left the test green — mutation `N0` below.
2. `aimemory.WithPool` had **no test of any kind**, although the server wires it
   from `HELIX_AI_MEMORY_POOL_*` exactly like the catalog one.

The fix is structural, not a stronger assertion on the same shape: the checks moved
from inline in `NewPostgres` to a `poolSettings.validate()` method in both packages,
so they are exercised without a network in the way. `TestNewPostgresRejectsPoolBeforeDialing`
still covers the wiring through the constructor, but asserts **which** error arrives
(`strings.Contains(err, "MaxIdleConns")`), which a dial failure cannot satisfy.

## N0 — the mutation that exposed the vacuous test

```
old: if settings.maxOpenConns < 1 {
new: if false && settings.maxOpenConns < 1 {
```

Result: `go test ./internal/catalog -run TestWithPoolValidation` → `ok`. The test
passed with the validation deleted. That is the reason this section exists; it was
found by attempting a needle, not by reading the test.

## Needle battery

`needle-results.json` (same directory) holds the machine-readable run. Each needle
deletes the behavior under test, runs the tests that are supposed to depend on it,
restores the file from a copy, and re-runs to confirm the suite is green again.

| ID | Mutation | Expected | Result | Restored green |
|---|---|---|---|---|
| N1-catalog-validate-noop | `poolSettings.validate()` returns nil immediately | catalog pool tests FAIL | killed | yes |
| N2-catalog-drop-setmaxopen | remove `db.SetMaxOpenConns` (catalog) | `TestWithPoolAppliesLimits` FAIL | killed | yes |
| N3-aimemory-validate-noop | `validate()` returns nil immediately (aimemory) | aimemory pool tests FAIL | killed | yes |
| N4-aimemory-drop-setmaxopen | remove `db.SetMaxOpenConns` (aimemory) | `TestWithPoolAppliesLimits` FAIL | killed | yes |
| N5-config-applyPoolEnv-noop | `applyPoolEnv` returns nil immediately | `TestPoolConfiguration` override + all six rejection cases FAIL | killed | yes |
| N6-config-drop-idle-crosscheck | disable only the idle-above-open cross-check | exactly `idle_above_open_rejected` FAIL | killed | yes |

N6 is the discriminating one: it fails a single named subtest, so the cross-check is
shown to be the thing that test depends on, not the surrounding parse errors.

## End-to-end wiring, from the process that ships

`live-backend-counts.json`. Library tests prove `WithPool` sets `sql.DB` limits; they
do not prove the server passes the environment into it. The real `helix-server`
binary was started against the qualification PostgreSQL with the 50,000-work fixture
loaded and hit with 12 concurrent unanchored text searches (~770 ms each), while
`pg_stat_activity` was sampled from inside `helixqual-pg` counting TCP backends only
(the sampling `psql` arrives over a unix socket and is excluded).

| Case | `HELIX_CATALOG_POOL_MAX_OPEN_CONNS` | Peak PostgreSQL backends |
|---|---|---|
| capped | 3 | **3** (every sample read 3; the other 9 requests queued) |
| default | unset (20) | **12** (all twelve concurrent) |

Startup rejection was also observed at the process boundary:
`MAX_OPEN_CONNS=3 MAX_IDLE_CONNS=99` exits 2 with
`HELIX_CATALOG_POOL_MAX_IDLE_CONNS 99 exceeds HELIX_CATALOG_POOL_MAX_OPEN_CONNS 3`
(`/tmp/pool-capped.log` pattern; the server never binds a listener).

## Incident: `server/internal/catalog/postgres.go` was destroyed and recovered

Recorded because §11.4.2 covers mistakes, not only passes, and because the recovery
path is not obvious from the tree.

While running the N0 needle above, the file was reverted with
`git checkout -- server/internal/catalog/postgres.go`. That was wrong: this working
tree carries the entire campaign **uncommitted** (HEAD is `f0a5f18`, which predates
Phases 0–1), so the checkout replaced the Phase 1 file with the pre-campaign version,
deleting the pool options, the keyset predicate, the `HasNext`/limit+1 read, the
batched `getWorks` call, and the `ON CONFLICT` migration bookkeeping.

Recovery, in order:

1. `/tmp/pg.bak` — a copy the previous session had made at 19:29 as part of its own
   needle run (transcript records 2459, 2471: `cp internal/catalog/postgres.go
   /tmp/pg.bak && … ; cp /tmp/pg.bak internal/catalog/postgres.go`). It is the
   post-Phase-1 state: the two later records touching the file were the N0 mutation
   (wanted reverted) and the `validate()` extraction (redone here).
2. Verified, not assumed: `grep` confirmed all five markers present
   (`WithPool`, `poolSettings`, `q.After`, `HasNext`, `ON CONFLICT (version)`);
   `gofmt -l` clean; `go build ./...` OK; `go vet` clean.
3. Proved equivalent by behavior: full `go test ./internal/catalog -count=1` against
   live PostgreSQL passed in 57.3 s, including the cursor-equivalence and batch tests
   that only exist in the Phase 1 version.
4. The HEAD version that was briefly in place is kept at `/tmp/pg.head-version.broken`
   so the diff `git diff HEAD -- server/internal/catalog/postgres.go` (+61/−13) can be
   inspected against either side.

Standing rule taken from this: **no `git checkout --`, `git restore`, `git stash`, or
`git clean` in this repository while the campaign is uncommitted.** Reversible local
copies (`cp` to `/tmp`) are the only sanctioned way to hold a pre-mutation state, and
the mutation script above restores from that copy rather than from git.

## Boundaries of this section

- The fixture must be re-seeded (`HELIX_SCALE_KEEP=1`) for the backend-count case to
  mean anything; with an empty `works` table twelve text searches finish too fast to
  overlap, and the peak would read low for the wrong reason.
- Peak-backend counts were taken through the host→container path. That path is the
  one `docs/scalability-review.md` shows can stall multi-packet responses; it does
  not affect a *count of concurrent backends*, but it is why the numbers are reported
  as 3-versus-12 and not as a latency claim.
- What is still unproven about pools: behavior under a PostgreSQL-side connection
  limit (`max_connections` exhaustion), and `ConnMaxLifetime` churn under load. The
  lifetime value is validated and passed through; no test observes a connection being
  recycled because of it.
- `applyPoolEnv` is unit-proven at the config layer and end-to-end proven for the
  catalog pool. The AI-memory equivalent is proven down to `aimemory.NewPostgres`
  only; `HELIX_AI_MEMORY=1` was not started in this run.
