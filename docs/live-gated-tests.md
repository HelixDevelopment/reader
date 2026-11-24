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
   run exits 3. Tests that write into the tracked `evidence/` tree are only DETECTED and REPORTED (`side_effects_report_<mode>.txt`, plus a `SIDE-EFFECTS-DETECTED` line); the runner never reverts, deletes, moves or copies anything (round 6).

Re-run: `tests/live/run_gated_tests.sh` (add `--no-env` for the RED baseline, `--check LOG` to apply only the skip gate to an existing `-v` log).

## Skip-gate semantics (review fixes F1-F10, B1/B2, I1-I4)
- Identity is `package|Test`, package = full import path: from the `ok|FAIL <pkg>` line, or from a `# PKG <import path>` marker the runner writes (from `go list -f '{{.ImportPath}}'`) before in-container / worker binaries, which print no package line. A test that skips in phase A and passes in phase B under the same import path is green; a relative or different path does not match (stays red).
- Phase B2 derives its test lists from the phase-A log with `phase_b_sets` (REGEX = top-level tests that skipped and never passed, minus the tests B1/B3-B5 run themselves; INRE in-container, HRE host-side). `--print-phase-b LOG` prints them (tested exactly in `tests/live/test_run_gated_tests.sh`). An unusable phase-A log or a missing helper aborts with exit 2; any exit-127 aborts with exit 2 (ERR trap).
- Allow-list `Package|Test|reason|HRD-<digits>`: fields are trimmed, `#` comments may be indented, a last line without newline counts, a reason must contain a non-space character and must not contain `|` (a `|` shifts the fields and is reported ALLOWLIST-MALFORMED). A missing or unreadable `--allow` file exits 2. Unused entries print `UNUSED-ALLOW (warning)`.
- `--check LOG` (repeatable, one log per flag; relative paths resolve against the caller's cwd) exits 2 for a missing, unreadable, empty or non-`go test -v` log (`LOG-UNUSABLE`, `GATE-BLIND`); exits 3 for an unlisted net skip, `--- FAIL`, build failure, failed package, package with no results, panic, timeout, or test records with no closing `PASS`/`FAIL`/package line (`INCOMPLETE`, truncated log); 0 otherwise.
- Portability: no `grep -P`; the runner verifies at start that `grep -x -F`, `grep -E`, awk, sort and comm work and exits 2 otherwise.
- Known limits: a test that prints an unindented `--- PASS: TestX` to stdout can forge a record (UNCONFIRMED in practice); `-race` DATA RACE lines are reported in the summary but are not a gate condition (go test normally fails the test).
- Header of every log records `head`, `dirty_files` and `dirty_diff_sha`.
- Gate tests: `tests/live/test_run_gated_tests.sh` (fixtures in `tests/live/fixtures`, committed: `.gitignore` negates the global `*.log` ignore for them, and test `R1b_tests_and_fixtures_tracked_not_ignored` fails if any file the tests, harness or fixture directory use is untracked or git-ignored); mutants: `tests/live/mutation_gate_check.sh`. They test the gate's own logic, not any live test.
- Sealed logs under `evidence/.../hrd050` predate the `# PKG` markers: their in-container phase-B results are not attributed to a package, and `--check` on them gives a real verdict (UNLISTED-SKIP / FAILED-TEST, exit 3: the in-container tests are unattributed and the throughput test FAILs). A fresh live run is owed (UNCONFIRMED until run).

## Review round 3 additions
- Sealing is self-contained: `seal_dir` (also `--seal DIR`) writes `SHA256SUMS` from `find` + C-locale `sort` + `sha256sum`, excluding itself and hidden files. Earlier heads called an untracked `tools/isolation/seal_evidence.sh`; the test `R1_runner_references_only_tracked_paths` now fails if the runner references any repo path not tracked in git.
- Every `# PKG <import path>` marker is followed by `# PKG-END <import path>` after the run; `parse_gotest.awk` flushes buffered records to the marker's package at PKG-END / next marker / EOF, so B3 records are not credited to B4's package in the logs the runner writes. Marker discipline is checked: a `# PKG` while another is open, a `# PKG-END` with no open marker or a different path, a marker open at EOF, or a package line (`ok|FAIL <pkg>`) while a marker is open all yield `INCOMPLETE` (exit 3). B3 workers write to per-worker logs concatenated in fixed order (0-3), then the verify run.
- Tool availability is checked up front (`need`, `--preflight`: git awk sort comm sed sha256sum find xargs, and for live runs python3 go podman base64 tr head); ports/password are checked non-empty. The ERR trap only covers directly invoked missing commands (not `$(...)` or `|| x` contexts) and is tested with `--selftest-missing-cmd`.
- ANSI escapes and CR are stripped before parsing. Contradictory allow-list entries (same Package|Test, different reason/item) exit 3; identical duplicates only warn.
- Known limit: a column-0 `ok <pkg>` line printed by test code is indistinguishable from the go test package line and can flush records to the wrong package (UNCONFIRMED in practice).

## Round 4 additions and residual limits
- Big-log fix: `grep -q` at the end of a pipe under `pipefail` made the gate blind on logs over the pipe buffer (SIGPIPE, rc 141); the class was swept (no `| grep -q`, no early-exit consumer in a pipe whose status decides a verdict; the only early-exit consumer left is `git ls-files | head -1` inside `$(...)` in the test scripts, where the status is not used). Tests generate a >1 MiB log (30000 PASS) and require a real verdict.
- `seal_dir` passes names after `--` to `sha256sum` (names such as `--help`, `-x`, `-c`, spaces are sealed and verify). File names containing a backslash or newline are UNCONFIRMED/unsupported (newline fails closed). Path format: `seal_dir` writes bare relative paths; the already-sealed `hrd050/SHA256SUMS` was written by the old seal tool with `./`-prefixed paths. Both verify with `sha256sum -c` from the sealed directory.
- `mutation_gate_check.sh` first requires the UNMUTATED copy (with `allowed_skips.txt`) to pass all cases, else exits 2.
- B3 worker failure propagation is tested through `wait_workers` (`--selftest-wait`).
- Residual limits (text mode): a test that prints a column-0 `ok <pkg> ...` line, or an unindented `--- PASS: X`, can mislead the parser (a `go test -json` mode would remove this; follow-up). Whether DATA RACE lines gate is left to go test's own failure.

## Round 5 additions
- Fixtures were previously git-ignored (`*.log`), so a clean checkout failed most gate tests and the mutation harness baseline. They are now tracked; `R1b` sweeps tests, harness and fixtures for untracked or ignored files.
- Parser: the package line must be `ok`/`FAIL` followed by optional spaces and a TAB (space-separated lines are not package lines); records left at EOF with no package line (for example a log cut after a bare `PASS`) and an empty `# PKG`/`# PKG-END` pair are `INCOMPLETE`. A binary that legitimately runs zero tests inside a marker pair is therefore reported INCOMPLETE (conservative).
- Allow-list: a reason made only of NBSP (U+00A0) is blank; `HRD-` with no digits is malformed.
- Panic and timeout alone (no `FAIL` line) fail the gate (fixtures `marker_panic_nofail`, `marker_timeout_nofail`; mutants M37/M38); allow-list matching is exact, not substring (mutant M39).

## Round 6 (operator decision: no cleanup)
- The side-effect snapshot/restore cleanup (round 5) was REMOVED entirely: reviewer finding B1 showed it could destroy uncommitted work. `se_snapshot` records `path<TAB>sha256` of every path `git status -z` lists under `evidence/`; `se_report` writes NEW / MODIFIED / NO-LONGER-LISTED lines to the report file under the run's own output dir (its own output dir is excluded). NEW means newly listed as dirty by this run (a clean tracked file the run modified is NEW). Neither function modifies the working tree; tests X10-X14, X19 checksum a temp git repo's tree before/after (pre-existing dirty and untracked files, names with spaces and non-ASCII); mutants M45/M46/M50. Limits: a path containing a TAB and git rename entries are UNCONFIRMED/unsupported.
- `PASS` followed by a bare `FAIL` inside a `# PKG` marker pair is a failed package (X15, M47). A `#` inside an allow-list entry (`seed#0`) is not a comment (X16/X17, M48). `--check` and the runner keep scratch files in a private mktemp dir, never in the evidence dir (X18, M49).
- Follow-ups (not fixed here): a test name containing `|` cannot be allow-listed (fails closed); commits 79eb80f and 467372f carry `Roadmap-Item` in a non-final trailer paragraph (history is not rewritten); live phase A/B run and tests/live/mutation_check.sh still owed.
