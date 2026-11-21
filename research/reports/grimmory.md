# Grimmory — per-repository research report

- Repo: https://github.com/grimmory-tools/grimmory · Revision: `021cb138dc7f19583142c55fd5070201c40b403b` — detached checkout at the exact analysis-pinned revision; row(s) `grimmory-tools-grimmory`, `grimmory`, `current_checkout_matches_analysis=true`.
- Commit: 2026-10-27 18:52:41 +0000 — "perf(api): resolve named filters to IDs (#2852)". Full non-shallow clone, `git fsck` clean.
- Ledger recomputed at pinned revision: 2887 blobs, 22,836,681 bytes — matches preserved analysis (tracked_files `2887`, `2887`). Top extensions (count): java 1166, ts 974, scss 190, html 185, sql 148, fixture 34.

**Architecture / data flow.** Java/Spring Boot comic-library server (`server` ~1166 java files) with Vue/TS admin UI, SQLite/H2 persistence and Flyway SQL migrations (148 sql): multi-user RBAC, smart filters, OPDS, embedded reader, Docker-first.
**Notable techniques.** Named-filter-to-ID query resolution at the API edge; strong AI-policy/justfile governance; schema-versioned migration discipline.
**License.** AGPL-3.0.
**Porting decision — LEARN ONLY (attributed: Helix research agent, checkout `021cb138dc7f19583142c55fd5070201c40b403b`).** AGPL network copyleft excludes code flow; harvest ideas for Helix's catalog filters and per-user permission model.