# Polka — per-repository research report

- Repo: https://github.com/vestigiumincaligne/polka · Revision: `01c2712a771fc72b7eced5887eea68e6bb1876a0` — detached checkout at the exact analysis-pinned revision; row `polka`, `current_checkout_matches_analysis=true`.
- Commit: 2026-09-12 10:08:51 +0300 — "CI: report vitest failures as annotations and give tests C". Full non-shallow clone, `git fsck` clean.
- Ledger at pinned revision: 209 blobs, 5,605,233 bytes — matches analysis tracked_files=209. Top extensions (count): go 109, jsx 24, css 20, js 16, png 8, json 7.

**Architecture / data flow.** Go server (109 go) with embedded JSX/web UI: personal book/media shelf with collections subsystem (`collections/`), Docker/compose deployment, Make-driven release, vitest UI tests annotated in CI.
**Notable techniques.** Small single-binary Go shelf with per-collection routing — a near-minimal precedent for Helix's catalog shape; CI annotating test failures directly on diffs.
**License.** MIT (2026 mrtooman).
**Porting decision — LEARN ONLY (attributed: Helix research agent, checkout `01c2712a771fc72b7eced5887eea68e6bb1876a0`).** MIT-compatible, but its surface is a strict subset of what Helix already specifies; retained as a Go structure cross-check, nothing to port.