# Helix Reader — Developer Guide

## Repository layout

- `core/` — Go module `github.com/helixreader/helix/core`: format detection and conversion contracts, HDM, ISBN, hybrid logical clocks, cache, source primitives, and synchronization logic.
- `server/` — Go/Gin service: catalog, ingestion, OPDS, source connectors, covers, Redis progress, KOReader sync, optional pgvector memory, and deployment entry point.
- `deploy/` — PostgreSQL/pgvector, Redis, and Helix Compose template. Loopback-only defaults are not a public production deployment.
- `research/` — repository inventories, revision-pinned static evidence, compressed per-file ledgers, backend comparisons, and porting records. Keyword evidence is a review lead, not a support claim.
- `scripts/` — deterministic repository analysis, real-browser checks, actual-server transport/API qualification, daemon-restart persistence and isolated backup/restore rehearsals, and isolated Go mutation verification.
- `evidence/` — machine output from checks that actually ran. A missing result is not a pass.
- `spec/` — constitution and product specifications. Specifications describe targets and must not be used as executable-capability evidence.
- `submodules/` — constitution, containers, QA, and CodeGraph development resources.
- `demos/interactive_static_demo/` — React/Vite responsive prototype. It is intentionally disconnected from the server and uses sample data.

## Toolchains

- Go 1.27.3, matching both `go.mod` files.
- Node.js and npm versions compatible with the Vite lockfile in the prototype.
- Python 3 with Playwright and Chromium for browser verification.
- PostgreSQL, Redis, and optionally pgvector for live integration checks.
- Docker Compose for the supplied deployment template.

Never place populated `.env` files, DSNs, access tokens, password digests, private keys, or credential-bearing URLs in source, test fixtures, logs, screenshots, evidence, or release archives.

## Configuration

The server reads configuration from the environment only (`server/internal/config`).
There is no config file and no flag parsing. Startup exits with status `2` on any
rejected value, so an invalid deployment fails instead of running with a silently
different setting.

| Variable | Default | Behavior when set |
| --- | --- | --- |
| `HELIX_ADDRESS` | `127.0.0.1:8080` | TCP (and HTTP/3) listen address. |
| `HELIX_TLS_CERT`, `HELIX_TLS_KEY` | unset | Serves TLS and enables HTTP/3. Must be configured together; one without the other is rejected. Minimum TLS 1.2 over TCP, 1.3 over QUIC. |
| `HELIX_ADMIN_TOKEN` | empty | Bearer credential for management routes. Required whenever `HELIX_ADDRESS` is not a loopback IP, unless the insecure override below is set. |
| `HELIX_ALLOW_INSECURE_ADMIN` | `false` | Permitting an empty admin token on a non-loopback listener. Unset this for any real deployment. |
| `HELIX_OPDS_USERNAME`, `HELIX_OPDS_PASSWORD` | empty | Basic credential for catalog reads, content, covers, and OPDS. Must be configured together. An empty username logs a startup warning; those routes are then unauthenticated. |
| `HELIX_EXTERNAL_URL` | empty | Public origin used to build absolute feed links. Trailing slashes are trimmed. |
| `HELIX_POSTGRES_DSN` | empty | Durable catalog. Empty falls back to a volatile in-memory catalog with a startup warning. |
| `HELIX_COVER_CACHE` | `./data/covers` | Directory for validated rendered covers. |
| `HELIX_REDIS_URL` | empty | Enables internal progress sync and KOReader sync. Without it those routes return `503`. An unparseable URL or an unreachable server aborts startup. |
| `HELIX_KOSYNC_REGISTRATION` | `false` | Allows `POST /users/create`. Keep disabled outside provisioning. |
| `HELIX_AI_MEMORY` | `false` | Enables the pgvector memory store. Rejected unless `HELIX_POSTGRES_DSN` is also set. |
| `HELIX_EMBEDDING_DIMENSIONS` | `1536` | Vector column width; accepted range `1`–`2000`. A store migrated at a different width is rejected at startup rather than silently reused. |
| `HELIX_HTTP3` | `true` | Disables the QUIC listener when false. A non-boolean value is rejected. HTTP/3 still requires a TLS certificate. |
| `HELIX_BROTLI_LEVEL` | `4` | Brotli level `0`–`11` for compressible responses. Out-of-range or non-integer is rejected. |
| `HELIX_CATALOG_POOL_MAX_OPEN_CONNS` | `20` | Catalog pool ceiling. Must be ≥ 1. |
| `HELIX_CATALOG_POOL_MAX_IDLE_CONNS` | `5` | Must be ≥ 0 and ≤ the open limit. |
| `HELIX_CATALOG_POOL_LIFETIME_SECONDS` | `1800` | Connection reuse deadline; must be ≥ 1. |
| `HELIX_AI_MEMORY_POOL_MAX_OPEN_CONNS` | `10` | AI-memory pool ceiling; same rules as the catalog pool. |
| `HELIX_AI_MEMORY_POOL_MAX_IDLE_CONNS` | `3` | |
| `HELIX_AI_MEMORY_POOL_LIFETIME_SECONDS` | `1800` | |

Two sharp edges worth knowing: `HELIX_KOSYNC_REGISTRATION`, `HELIX_AI_MEMORY`, and
`HELIX_ALLOW_INSECURE_ADMIN` parse booleans leniently, so an unrecognized value
such as `yes` becomes `false` without an error, while `HELIX_HTTP3` rejects the same
input. The shutdown drain budget is fixed at 15 seconds in `config.Config` and is not
configurable from the environment.

Pool limits are shared-process budgets: the catalog and AI-memory stores each open
their own pool against the same PostgreSQL instance, so their combined maxima must
stay inside the server's `max_connections` alongside every other client. Raise them
only with evidence from `pg_stat_activity`, not by default.

## Build and test

Run commands from the corresponding module directory. `-count=1` disables Go’s test-result cache. Repeat deterministic suites and compare machine reports; do not infer success from compilation alone.

```bash
cd core
go test ./... -count=1
go test -race ./... -count=1
go vet ./...

cd ../server
go test ./... -count=1
go test -race ./... -count=1
go vet ./...

cd ../demos/interactive_static_demo
npm ci
npm run lint
npm run build
npm ci --dry-run
```

The pinned sandbox toolchain and module cache are environment details, not project dependencies. On a normal development machine, use the declared Go version and the module checksums in `go.sum`.

## Browser verification

Build the prototype, then run the Go verifier INSIDE the rootless isolation container (HARD RULE: browser and GUI runs never touch the
host; see [`docs/test-isolation.md`](test-isolation.md)). The binary serves `dist/` itself on the container loopback.

```bash
(cd demos/interactive_static_demo && npm ci && npm run build)
(cd tools/verify && CGO_ENABLED=0 go build -trimpath -o "$HELIX_UIVERIFY_BIN" ./cmd/uiverify)
ISO_WORK=$HOME/.cache/claude-tmp/<dir> HELIX_UIVERIFY_BIN=... \
  qa/isolation/run_uiverify.sh demos/interactive_static_demo/dist evidence/<fresh-dir>
```

Details: [`tools/verify/README.md`](../tools/verify/README.md). `scripts/verify_ui.py` stays in the tree but drives a host browser and is not to be run on the host.

The suite uses fresh browser contexts at 375×812, 768×1024, 1024×900, and 1440×1000. It checks viewport metadata, horizontal overflow, disclosure text, search, reader navigation and typography deltas, passage state, metadata decisions, source validation/addition/scan state, conflict resolution, theme changes, console errors, and missing assets. It writes `ui-results.json` even when a viewport aborts unexpectedly.

The mutation gate removes the viewport declaration from a temporary production build and requires the verifier to reject all four viewports. See `evidence/ui-mutation/mutation-result.json` for the recorded run.

## Live, fuzz, and mutation verification

With disposable local PostgreSQL/pgvector and Redis services available, rebuild the actual server and HTTP/3 client, then run `scripts/verify_live_server.py`, `scripts/verify_live_transport.py`, and optionally `scripts/verify_live_server.py --restart-daemons`. The last command restarts the local qualification daemons and requires the corresponding service permissions; never run it against production. `scripts/verify_backup_restore.py` separately restores PostgreSQL and Redis into isolated test services and verifies the recovered records through a fresh server. Evidence contains assertion names and statuses, not credentials or DSNs.

Run every parser/format fuzz target and the HTTP single-range target with a fixed `-fuzztime`. `scripts/verify_go_mutation.py` must reject all ten deliberate backend mutants in isolated copies and verify original-file hashes. Benchmark logs are microbenchmark observations, not service-level throughput or latency guarantees. See `docs/testing.md` for the current evidence matrix and explicit non-PASS tracks.

## KOReader sync development

The implementation is original protocol code under `server/internal/kosync/`; it is not a copy of an upstream server. The porting record identifies the upstream revisions and protocol files used for comparison.

- Redis is required by the server wiring so accounts and progress survive restarts.
- Registration defaults to disabled. Enable `HELIX_KOSYNC_REGISTRATION` only while provisioning accounts.
- `x-auth-key` is treated as a password and stored only as a bcrypt verifier.
- Unknown-user authentication performs a dummy bcrypt comparison to reduce account-discovery timing differences.
- Request bodies, field lengths, percentages, and document identifiers are bounded.
- Timestamps come from the server. Progress is isolated by account and document.
- TLS is required in transit. Automated tests do not prove interoperability with every KOReader release or device.

Run focused tests with:

```bash
cd server
go test ./internal/kosync ./internal/httpapi -count=1
go test -race ./internal/kosync ./internal/httpapi -count=1
```

## Research evidence

`scripts/analyze_repositories.py` indexes valid Git checkouts into revision- and tree-pinned JSON, Markdown, and compressed per-file ledgers. It records blob IDs, sizes, file roles, content hashes for scanned text, language counts, license-notice candidates, and feature/format keyword locations. Sensitive-looking files and large/binary files are not content-scanned; they remain represented in the ledger with a skip reason.

```bash
python3 scripts/analyze_repositories.py --workers 3
python3 scripts/summarize_research.py
```

A static index is not an exhaustive behavioral review. Manual review status remains `pending` until architecture, implementation paths, tests, licenses, strengths, limitations, and Helix porting decisions have been checked against actual source files. Failed or unavailable clones are listed explicitly and never marked researched.

## Contribution rules

1. Read `submodules/constitution/Constitution.md`, especially the no-bluff, credential, testing, and evidence requirements.
2. Describe implemented behavior separately from planned architecture.
3. Add deterministic tests with each bug fix and prove the test fails for the original defect where practical.
4. Treat rendering, conversion, DRM, external providers, and device compatibility as unimplemented until the actual integration and runtime evidence exist.
5. Record upstream revision, file evidence, license review, and Helix implementation links for every ported idea. Do not copy code across incompatible licenses.
6. Keep user files read-only during indexing. Preserve work → edition → file identity and edition-specific identifiers.
7. Do not publish, deploy, claim production readiness, or produce final archives without the corresponding live validation and archive-content checks.
