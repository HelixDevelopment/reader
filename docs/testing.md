# Helix Reader — verification and qualification

## Authoritative result index

The current unified result index is [`docs/testing/results/index.md`](testing/results/index.md), with machine-readable status and evidence hashes in [`docs/testing/results/summary.json`](testing/results/summary.json). Regenerate it with:

```bash
python3 scripts/publish_test_results.py
```

**Release qualification is NOT READY.** Suite-specific passes do not close native-client, full-format, external-interoperability, deployment, enterprise-load, research, HelixQA, or HelixTrack work. Fresh final archives remain withheld.

## Evidence rules

A check is `PASS` only when its command actually ran and its machine output satisfies the stated assertion. A missing dependency, unavailable service, or unexecuted platform is `SKIP`, `NOT RUN`, or `NOT DELIVERED`, never a simulated pass. Compilation alone is not behavioral evidence. Failed attempts remain preserved even when later runs pass. Reports must not contain credentials, populated environment files, DSNs, private keys, or secret-bearing URLs.

## Current qualification matrix

Paths below are relative to the project root. The generated result index is authoritative when it contains a newer artifact.

| Layer | Current outcome | Evidence and boundary |
|---|---|---|
| Core unit/integration | PASS | `evidence/core/final-go-test.log` |
| Core race and vet | PASS | `evidence/core/final-go-race.log`; `evidence/core/final-go-vet.log` |
| Server unit/integration and three repeats | PASS | `evidence/server/final-go-test.log`; `evidence/server/final-repeat-*.log` |
| Server race and vet | PASS | `evidence/server/final-go-race.log`; `evidence/server/final-go-vet.log` |
| Statement coverage | MEASURED | `evidence/core/final-go-cover.log`; `evidence/server/final-go-cover.log`; coverage is not a completeness guarantee |
| Fuzz | PASS, seven targets | Six core format logs plus `evidence/server/final-go-fuzz-single-range.log` |
| Real PostgreSQL, pgvector, and Redis | PASS in the local qualification topology | `evidence/server/final-go-live-services.log`; not a managed-service or multi-region claim |
| Live API after keepalive fix | PASS: three 37/37 runs | `evidence/live-server/post-fix-repeat-01/` through `post-fix-repeat-03/` |
| Daemon/PostgreSQL/Redis restart | PASS: 38/38 after fix | `evidence/live-server/post-fix-daemon-restart/results.json` |
| Earlier daemon-restart defect | FAIL preserved | `evidence/live-server-daemon-restart/results.json`; root-caused to unused-compressor finalization on identity fallback |
| Live HTTP transport | PASS: three consecutive 5/5 reruns | `evidence/live-transport/post-compression-repeat-02/` through `repeat-04/`; HTTP/1.1, HTTP/2, HTTP/3, TLS, Alt-Svc |
| Transport readiness attempt | FAIL preserved | `evidence/live-transport/post-compression-failure-01/`; initial run lacked diagnostics, direct reproduction passed, verifier was instrumented before reruns |
| Backup/restore | PASS: 13/13 after compression fix | `evidence/backup-restore/post-compression/results.json` |
| Go mutation | PASS: 21/21 mutants rejected | `evidence/go-mutation/results.json`; original-tree hashes unchanged |
| Workable-items registry | PASS: ten qualification groups | `evidence/workable-items/results.json`; includes 12 real-SQLite tests, stress, crash/corruption recovery, and four rejecting mutations |
| Frontend lint/build | PASS | `evidence/ui/final-npm-lint.log`; `evidence/ui/final-npm-build.log` |
| Browser UI | PASS: 64/64 across four viewports | `evidence/ui/final-run.log`; sample-only prototype, not a backend-connected or native-client claim |
| Microbenchmarks | Recorded for three focused targets | `evidence/core/benchmark-*.log`; `evidence/server/benchmark-range.log`; not enterprise-load evidence |
| HelixQA | FAIL | `evidence/helixqa-build-test.log`; declared own-org dependencies are incomplete |
| Repository research | PARTIAL | 111/111 preserved static analyses with complete file ledgers; current checkouts at exact pinned revisions cover 97/111 rows (91/105 unique URLs), 93 rows matching the analysis revision; 14 large targets remain UNAVAILABLE (unpinned, not unreachable); 40 per-repository architecture/license/porting reports under `research/reports/`; manual review remains open |
| Enterprise scalability | NOT RUN | `docs/scalability-review.md`; functional tests do not prove capacity or horizontal scale |
| Native clients | NOT DELIVERED | No native OS application or device matrix exists |
| Container deployment | NOT RUN | Recipes exist; no Docker/Podman cold start, restart, upgrade, or rollback result |
| Firebase distribution | NOT RUN | No project-bound distribution run |
| External interoperability | NOT RUN | No physical KOReader, external OPDS client, or deployment-specific rclone provider run |
| HelixTrack remote synchronization | NOT RUN | Offline outbox only; no verified remote API contract or push/pull receipt |
| Final archives | NOT RUN | Withheld pending genuine scope completion and clean qualification |

## Reproduction commands

Use the installed Go toolchain and offline module cache where applicable:

```bash
cd core
go test ./... -count=1
go test -race ./... -count=1
go vet ./...

cd ../server
go test ./... -count=1
go test -race ./... -count=1
go vet ./...
```

Frontend (the React/Vite prototype lives at `demos/interactive_static_demo`):

```bash
cd demos/interactive_static_demo
npm ci --dry-run
npm run lint
npm run build
```

Browser checks never run on the host (HARD RULE, [`docs/test-isolation.md`](test-isolation.md)). Run the Go UI verifier inside the
rootless container (it serves `dist/` itself; no host display, no network):

```bash
(cd tools/verify && CGO_ENABLED=0 go build -trimpath -o "$HELIX_UIVERIFY_BIN" ./cmd/uiverify)
ISO_WORK=$HOME/.cache/claude-tmp/<dir> HELIX_UIVERIFY_BIN=... \
  qa/isolation/run_uiverify.sh demos/interactive_static_demo/dist evidence/<fresh-dir>
```

See [`tools/verify/README.md`](../tools/verify/README.md) and `PARITY.md` there. `scripts/verify_ui.py` is kept but is a host-browser
script and must not be run on the host.

Live current-tree verification (run from the repository root; all verification
scripts derive the repository root from their own location and build binaries
and scratch data under the gitignored `.helix-verify/` directory):

```bash
cd "$(git rev-parse --show-toplevel)"
python3 scripts/verify_live_server.py
python3 scripts/verify_live_server.py --restart-daemons
python3 scripts/verify_live_transport.py
python3 scripts/verify_backup_restore.py
python3 scripts/verify_go_mutation.py
python3 scripts/verify_workable_items.py
```

Service endpoints are supplied through the environment, falling back to the
historical loopback defaults when unset: `HELIX_TEST_POSTGRES_DSN` selects the
disposable test database for the verifiers (passed to the server as
`HELIX_POSTGRES_DSN`), `HELIX_TEST_REDIS_ADDR` (`host:port`) selects the Redis
endpoint (passed to the server as `HELIX_REDIS_URL`), and `HELIX_GO` overrides
the `go` binary used to build the exact tree under test (otherwise `go` from
`PATH`). When `redis-server` is not installed locally and no
`HELIX_TEST_REDIS_ADDR` is provided, the daemon-restart path records an honest
SKIP instead of starting a daemon.

The live scripts use disposable loopback-only credentials and local test services. The daemon-restart option requires permission to restart qualification PostgreSQL and Redis services; it must never target production. TLS uses a disposable IP-SAN certificate and a verified HTTP/3 client, not insecure certificate skipping.

## Deterministic UI assertions

Each viewport runs in a new browser context with reduced motion. State-changing checks compare before/after values rather than relying only on screenshots:

- chapter title changes after Next;
- computed reader font size changes after the control;
- accepting a proposal increases the accepted count;
- source submission changes from validation error to a new row and queued state;
- conflict resolution exposes the resolved state;
- theme selection changes the root `data-theme` value.

The suite checks visible sample-data disclosure, horizontal overflow, console errors, and missing assets. Screenshots are reviewed separately; DOM assertions alone do not prove visual quality.

## Mutation validation

`verify_go_mutation.py` applies 21 single-contract defects to isolated module copies. Every targeted test must exit non-zero, every mutation anchor must be unique, and original-file hashes must remain unchanged.

`verify_workable_items.py` independently proves that closure-vocabulary, drift-detection, remote-push anti-bluff, and attribution-roundtrip defects are rejected. It also exercises real SQLite concurrency, 1,000-row insertion, SIGKILL rollback, and corrupt-database recovery.

## Remaining release gates

Before a production-ready claim, the project still needs the open items in `docs/Issues.md` to reach evidence-backed closure, including complete research and manual license decisions, HelixQA execution, full format fulfillment, native applications and device runs, external OPDS/KOReader/provider interoperability, actual container and Firebase deployment, enterprise-load qualification, verified HelixTrack synchronization, and final secret scanning plus clean archive extraction/build/test.
