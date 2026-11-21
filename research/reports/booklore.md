# Booklore — per-repository research report

- Repo: https://github.com/booklore-app/booklore · Revision: `01b3407528a6ce00d064dbb9aa3f69b72d82641b` — detached checkout at the exact analysis-pinned revision; row(s) `booklore`, `current_checkout_matches_analysis=true`.
- Commit: 2026-09-19 11:01:12 -0600 — "chore(deps): bump docker/setup-buildx-action from 3.12.0 t". Full non-shallow clone, `git fsck` clean.
- Ledger recomputed at pinned revision: 2786 blobs, 30,420,513 bytes — matches preserved analysis (tracked_files `2786`). Top extensions (count): java 1028, json 681, ts 425, scss 194, html 176, sql 131.

**Architecture / data flow.** Spring Boot (Java 1028 files) + Angular/TS UI (425 ts): multi-source book manager (Calibre libraries, physical scans), metadata fetch orchestrators, scheduled library maintenance, Docker deployment.
**Notable techniques.** Pluggable metadata-provider fan-out with user-priority rules; entity/DTO separation with MapStruct; strong migration churn discipline (681 json fixtures/configs).
**License.** AGPL-3.0.
**Porting decision — LEARN ONLY (attributed: Helix research agent, checkout `01b3407528a6ce00d064dbb9aa3f69b72d82641b`).** AGPL bars code; its provider-orchestration and dedup UX are design references for Helix's future metadata pipeline.