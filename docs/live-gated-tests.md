# Live-gated tests: zero-skip gate (HRD-050 / GAP-015)

| Field | Value |
|---|---|
| Item | HRD-050 (GAP-015) |
| Runner | `tests/live/run_gated_tests.sh` |
| Allow-list | `tests/live/allowed_skips.txt` (`Package|Test|reason|HRD-NNN`) |
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

## Skip-gate semantics (review fixes F1-F10, B1/B2, I1-I4)
- Identity is `package|Test`, package = full import path: from the `ok|FAIL <pkg>` line, or from a `# PKG <import path>` marker the runner writes (from `go list -f '{{.ImportPath}}'`) before in-container / worker binaries, which print no package line. A test that skips in phase A and passes in phase B under the same import path is green; a relative or different path does not match (stays red).
- Phase B2 derives its test lists from the phase-A log with `phase_b_sets` (REGEX = top-level tests that skipped and never passed, minus the tests B1/B3-B5 run themselves; INRE in-container, HRE host-side). `--print-phase-b LOG` prints them (tested exactly in `tests/live/test_run_gated_tests.sh`). An unusable phase-A log or a missing helper aborts with exit 2; any exit-127 aborts with exit 2 (ERR trap).
- Allow-list `Package|Test|reason|HRD-<digits>`: fields are trimmed, `#` comments may be indented, a last line without newline counts, a reason must contain a non-space character and must not contain `|` (a `|` shifts the fields and is reported ALLOWLIST-MALFORMED). A missing or unreadable `--allow` file exits 2. Unused entries print `UNUSED-ALLOW (warning)`.
- `--check LOG` (repeatable, one log per flag; relative paths resolve against the caller's cwd) exits 2 for a missing, unreadable, empty or non-`go test -v` log (`LOG-UNUSABLE`, `GATE-BLIND`); exits 3 for an unlisted net skip, `--- FAIL`, build failure, failed package, package with no results, panic, timeout, or test records with no closing `PASS`/`FAIL`/package line (`INCOMPLETE`, truncated log); 0 otherwise.
- Portability: no `grep -P`; the runner verifies at start that `grep -x -F`, `grep -E`, awk, sort and comm work and exits 2 otherwise.
- Known limits: a test that prints an unindented `--- PASS: TestX` to stdout can forge a record (UNCONFIRMED in practice); `-race` DATA RACE lines are reported in the summary but are not a gate condition (go test normally fails the test).
- Header of every log records `head`, `dirty_files` and `dirty_diff_sha`.
- Gate tests: `tests/live/test_run_gated_tests.sh` (fixtures in `tests/live/fixtures`); mutants: `tests/live/mutation_gate_check.sh`. They test the gate's own logic, not any live test.
- Sealed logs under `evidence/.../hrd050` predate the `# PKG` markers: their in-container phase-B results are not attributed to a package, so `--check` on them is stricter than when sealed. A fresh live run is owed (UNCONFIRMED until run).
