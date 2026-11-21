# Stump — per-repository research report

- Repo: https://github.com/stumpapp/stump · Revision: `d0f51c8e6da74d584a2a4cda8d9db02337de73ca` (detached checkout at the exact analysis-pinned revision; rows `stumpapp-stump` (backend) and `stump` (client) both match)
- Commit date: 2026-10-20 07:27:31 -0700 — "🔖 Release v0.1.11 (#1474)". Full non-shallow clone, fsck clean.
- Ledger: 2,808 blobs, 60,840,923 bytes — matches analysis. Top extensions (count): tsx 1095, ts 619, rs 537, png 88, json 88, woff2 70.

**Architecture / data flow.** Rust (Axum) comic-book server with tracing, config, and a resolver for filesystem-mapped packages; `core/` crate for domain logic, `apps/` for CLI + server, React/TypeScript web UI. Serves OPDS 1.2 +2 feeds and its own REST API, automatic scans with configurable patterns — the closest single analogue to Helix's Go+Gin catalog server.

**Notable techniques.** File-watching + resumable scanning jobs, pattern-based library resolution, config-first multi-user profiles, generated OpenAPI client binding the TS UI to Rust.

**License.** MIT (Copyright 2022 Aaron Leopold). Permissive; attribution required on copies.

**Porting decision — PORT-WORTHY at design level, selective source OK under MIT (attributed: Helix research agent, checkout `d0f51c8e6d`).** Its OPDS feed assembly and scan-then-resolve pipeline are the most directly comparable prior art for Helix's catalog service; Go reimplementation guided by its module boundaries is the intended path. Any literal Rust→Go transliteration must record this revision and the MIT notice.
