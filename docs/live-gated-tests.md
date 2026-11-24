# Live-gated tests: zero-skip gate (HRD-050 / GAP-015)

| Field | Value |
|---|---|
| Item | HRD-050 (GAP-015) |
| Runner | `tests/live/run_gated_tests.sh` |
| Allow-list | `tests/live/allowed_skips.txt` (`Name|reason|HRD-NNN`) |
| Mutation proof | `tests/live/mutation_check.sh` |
| Evidence | `evidence/campaign-2026-11-23/gap-fix/hrd050/` |

## Why

Without environment, `go test ./... -v` in `server` + `core` reports 47 SKIP lines: PostgreSQL (`HELIX_TEST_POSTGRES_DSN`),
Redis (`HELIX_TEST_REDIS_ADDR`), measurement harnesses (`HELIX_LOAD_RUN=1`), the multi-process merge roles
(`HELIX_MP_ROLE`), the LCP live qualification (`HELIX_LCP_LIVE_BIN`/`_EVIDENCE`) and the OPDS dump (`HELIX_OPDS_DUMP`).
Race and regression evidence therefore did not cover those paths.

## What the runner does

1. Starts throwaway rootless-podman containers named `hrdlive-<pid>-*` on random `127.0.0.1` ports: pgvector PG16 (512m), Redis 7 (128m, shares PG's
   network namespace), and an alpine test-runner (2g, same netns). A trap removes them in reverse order.
2. Phase A: whole server + core suite with `-race -count=1 -v -p 1`, PG and Redis live.
3. Phase B (no `-race`; throughput assertions are meaningless under the race detector): seeds the 50k-work scale fixture, then runs every test that skipped in
   phase A. PG-latency-sensitive tests run **inside the netns of the database** (static test binaries copied into the runner container,
   `HELIX_MEASUREMENT_PATH=in_container_clean`); tests that need host binaries (rclone) or do not measure PG latency run on the host and are labelled
   `host_forwarded`. The 4-process merge test, LCP live qualification (fresh `helix-server` binary) and OPDS dump run as their own steps.
4. Skip gate: a test counts as skipped when it SKIPs somewhere and PASSes nowhere. Each must be in the allow-list with a reason and an `HRD-NNN` item, or the
   run exits 3. Tests that write into the tracked `evidence/` tree are reverted/moved to `side_effects/`.

Re-run: `tests/live/run_gated_tests.sh` (add `--no-env` for the RED baseline, `--check LOG` to apply only the skip gate to an existing `-v` log).
