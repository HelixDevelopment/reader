# Helix Reader

**A 9-platform reading ecosystem with a shared Go core** — one document model, one sync
engine, one identity pipeline, thin native shells everywhere.

> **Status: active development. No production-ready claim is made.** See
> [`docs/testing.md`](docs/testing.md) for the authoritative qualification matrix and
> [`docs/user_guide.md`](docs/user_guide.md) for what works today versus what is still an
> integration target. This project follows a *positive-evidence-only* rule
> (constitution §7.1 "NO BLUFF"): nothing is claimed without a recorded artifact in
> [`evidence/`](evidence/).

---

## What this repository contains

| Area | Contents |
|---|---|
| [`core/`](core) | Go shared core (module `github.com/helixreader/helix/core`): document model, format registry, identifiers, clocks, cache, sources, parsing, conversion, sync |
| [`server/`](server) | Go HTTP service (module `github.com/helixreader/helix/server`): catalog/ingestion, OPDS 1.2 + 2.0, KOReader sync, covers, identification, AI memory |
| [`demos/interactive_static_demo/`](demos/interactive_static_demo) | React + TypeScript + Vite + Tailwind interactive product prototype (sample in-memory data only, **no backend connected**) |
| [`deploy/`](deploy) | `docker-compose.yml` (PostgreSQL/pgvector + Redis + helix server) |
| [`docs/`](docs) | Architecture, API reference, user/developer guides, FAQ, testing index, workable-items registry, research notes |
| [`spec/`](spec) | SpecKit phase artifacts: constitution → specify → clarify → plan → tasks → analyze → review |
| [`research/`](research) | Static analysis of 111 third-party repositories (backends, clients, file evidence) feeding porting and design decisions |
| [`evidence/`](evidence) | Append-only timestamped verification logs (test runs, live server/transport/UI checks, mutation gates) |
| [`scripts/`](scripts) | Verification and publishing utilities (see below) |
| [`submodules/`](submodules) | `constitution` (process law), `superspec` (Spec-Kit workflow bridge), `containers` (device/emulator matrix) |
| [`upstreams/`](upstreams) | Push recipes for every upstream remote (see [Upstream remotes](#upstream-remotes)) |

## Architecture at a glance

```
Clients        React prototype (sample-only) · third-party OPDS readers · KOReader · admin/API clients
                    │ OPDS 1.2/2.0 · KOReader sync · Bearer admin
                    ▼
Helix Server   Go/Gin HTTP service (HTTP/1.1; HTTP/2/3 with TLS)
                    ├── Catalog repository ── PostgreSQL (pgvector) │ in-memory fallback
                    ├── Progress/KOReader sync ── Redis
                    ├── Covers cache (validated)
                    └── Read-only ingestion: local / mounts / HTTP(S) / WebDAV / OPDS / rclone
```

The **web prototype does not call the API** — it demonstrates the shared visual language
and interaction model with fictional sample data, and is labeled as such on every screen.
Native shells (Android, iOS, desktop, web, HarmonyOS, Aurora) are milestone M6 work in
[`spec/03_plan.md`](spec/03_plan.md).

### Catalog identity model

Works group editions; editions group files. Identity is anchored on a deterministically
selected canonical ISBN-13 with content SHA-256 as the no-valid-ISBN fallback — ISBN-10/13
equivalents, hyphenation, and scheme labels cannot fork identity. Ingestion merges through
one atomic repository operation; scans prune only after an unprefixed complete recursive
listing succeeds. Full details: [`docs/architecture.md`](docs/architecture.md).

## Core packages (`core/`)

| Package | Purpose |
|---|---|
| `hdm` | Helix Document Model — publications, spine, blocks, inline content, word counts |
| `formats` | Format registry + conversion capability matrix (with fuzz tests) |
| `isbn` | ISBN-10/13 validation, check digits, extraction |
| `hlc` | Hybrid logical clocks — mint/receive/merge, skew immunity |
| `cache` | LRU cache with protected entries and oversize rejection |
| `sources` | Source capability table (feature flags per protocol) |
| `pkg/parse` | EPUB, FB2, TXT, CBZ, MOBI parsers |
| `pkg/convert` | Conversion engine hooks (HDM writers) |
| `pkg/sync` | CRDT sync engine (LWW registers, add-wins sets, op log, snapshots) + benchmarks |
| `pkg/identify` | Identification pipeline (metadata evidence, confidence scoring) |
| `pkg/fts` | Full-text search |
| `pkg/store`, `pkg/settings`, `pkg/theme`, `pkg/crypto`, `pkg/source` | Persistence, settings, design tokens, crypto utilities, source abstraction |

The core is designed to be packaged c-shared/c-archive for Android (gomobile), iOS,
HarmonyOS (N-API), Aurora (Qt C++), desktop, and WASM for web (constitution R1).

## Server (`server/`)

Production-shaped Go service. Highlights:

- **Catalog & ingestion** (`internal/catalog`, `internal/ingest`): atomic merge,
  advisory-lock identity serialization, per-source lock stripes, PostgreSQL + memory repos.
- **OPDS** (`internal/opds`): OPDS 1.2 and 2.0 catalogs, pagination, content streaming
  with single-range + strong ETags and live revalidation.
- **KOReader sync** (`internal/kosync`): account + document progress over Redis/memory.
- **Identification** (`internal/identify`): OpenLibrary, Google Books, LLM and JEV
  providers with confidence-scored metadata evidence.
- **Sources** (`internal/sources`): local, HTTP(S), WebDAV, OPDS, rclone remotes.
- **AI memory** (`internal/aimemory`): optional pgvector-backed vector memory.
- HTTP API surface documented in [`docs/api.md`](docs/api.md); health/capabilities,
  works/editions/files, covers, sources, jobs, KOReader endpoints.

## Specification-driven development (`spec/`)

This repo is governed by a SpecKit phase flow (constitution R2):

`00_constitution.md` → `01_specify.md` → `02_clarify.md` → `03_plan.md` → `04_tasks.md` →
`05_analyze.md` → `06_review.md` → execute → LIVE deterministic validation.

- **Milestones M1–M7**: core foundations (done, E1 with 22 deterministic tests), parsers,
  sources, sync (CRDT over HLC), identification/conversion, shells, AI/TTS/stats.
- **Epics E1–E7** in `spec/04_tasks.md` carry stories, subtasks, acceptance tests, and
  evidence artifacts; the analyze phase tracks gaps (OWED items) to closure.

## Submodules

| Submodule | Upstream | Role |
|---|---|---|
| [`submodules/constitution`](submodules/constitution) | `HelixDevelopment/constitution` | Helix Universal Constitution — binding process law (multi-upstream push mechanics, evidence rules, secret handling) |
| [`submodules/superspec`](submodules/superspec) | `WangX0111/superspec` | Superpowers Bridge for Spec-Kit — brainstorming, task decomposition, TDD execution discipline |
| [`submodules/containers`](submodules/containers) | `vasic-digital/containers` | Container orchestration/health/lifecycle Go module; owns device & emulator execution environments |

Initialize all submodules recursively:

```bash
git clone --recursive <this repo>
# or, in an existing checkout:
git submodule update --init --recursive
```

## Upstream remotes

Push recipes live in [`upstreams/`](upstreams) (sourced by the commit-and-push
entrypoint, per constitution §2):

- `upstreams/GitHub.sh` → `git@github.com:HelixDevelopment/reader.git`
- `upstreams/m3w.sh` → `ssh://git@git.m3w.org:2222/Helix/reader.git`

Multi-upstream push is the norm (constitution §2.1): every commit is pushed to **all**
upstreams, every tag is mirrored (§4), and submodule changes land in the submodule
first (§3).

## Development

**Requirements:** Go 1.27.3, Node.js 20+ (prototype only), Python 3 (scripts),
PostgreSQL + Redis (or `deploy/docker-compose.yml`).

```bash
# Run the core/server test suites (deterministic; two identical runs required)
cd core   && go test ./... -count=1
cd server && go test ./... -count=1

# Run the interactive prototype (sample data only)
cd demos/interactive_static_demo && npm install && npm run dev

# Bring up the full local stack
cd deploy && docker compose up --build
```

**Verification utilities** (`scripts/`): `verify_live_server.py`,
`verify_live_transport.py`, `verify_backup_restore.py`, `verify_go_mutation.py`
(mutation gates), `verify_ui.py`, `verify_ui_mutation.py`, `verify_workable_items.py`,
`publish_test_results.py` (regenerates `docs/testing/results/`), `summarize_research.py`,
`analyze_repositories.py`, `workable_items.py` (SQLite registry export/validate).

**Evidence rules:** a check is `PASS` only when its command actually ran and the machine
output satisfies the assertion; missing dependencies are `SKIP`, never a simulated
pass; failed attempts remain preserved in `evidence/`; reports must not contain
credentials or secrets.

## Documentation map

- [`docs/architecture.md`](docs/architecture.md) — runtime topology, identity model, ingestion sequences (mermaid)
- [`docs/api.md`](docs/api.md) — full HTTP API reference
- [`docs/user_guide.md`](docs/user_guide.md) — what works today, prototype behavior, server quickstart
- [`docs/developer_guide.md`](docs/developer_guide.md) — setup for dev machines (incl. CodeGraph)
- [`docs/testing.md`](docs/testing.md) + [`docs/testing/`](docs/testing) — qualification matrix and result index
- [`docs/faq.md`](docs/faq.md) — production readiness, accounts, formats, protocols
- [`docs/scalability-review.md`](docs/scalability-review.md), [`docs/research/`](docs/research) — design studies
- [`docs/workable-items.md`](docs/workable-items.md) — workable-items registry (`docs/workable_items.db` is authoritative; Issues/Fixed docs are generated)

## Research

[`research/`](research) holds the deep-web-research-before-implementation corpus
(constitution §11.4.8): 111 statically analyzed third-party repositories
(`backend_analysis/`, `client_analysis/`, `file_evidence/`), a
[`repository_inventory.json`](research/repository_inventory.json), and a
[`porting_ledger.md`](research/porting_ledger.md). Keyword evidence is not support proof;
each analysis is marked `manual_review_status` accordingly.

## License and governance

Project process law is the [project constitution](spec/00_constitution.md), which
incorporates the Helix Universal Constitution from the `submodules/constitution`
submodule by reference; on conflict the universal constitution wins. Mandatory
constraint for every contributor and automation: **credentials and secrets must never be
committed, printed, logged, or included in evidence or archives.**
