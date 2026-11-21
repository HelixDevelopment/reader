# Kavita — per-repository research report

- Repo: https://github.com/Kareadita/Kavita · Revision: `d549d5b1609086c8190b9bee6eb48ebfed71e1df` — detached checkout at the exact analysis-pinned revision; row(s) `kareadita-kavita`, `kavita`, `current_checkout_matches_analysis=true`.
- Commit: 2026-10-22 18:05:58 +0000 — "Release changes for 0.9.1.15". Full non-shallow clone, `git fsck` clean.
- Ledger recomputed at pinned revision: 3917 blobs, 115,775,962 bytes — matches preserved analysis (tracked_files `3917`, `3917`). Top extensions (count): cs 1787, ts 905, scss 351, html 330, json 134, png 88.

**Architecture / data flow.** .NET 8 API (1787 cs) + Angular-ish TS client (905 ts): MediatR-style service layers, EF Core migrations, task scheduler for scans, OPDS 1.2/2.0, multi-user age-restriction policy engine.
**Notable techniques.** Collection+smart-filter expression language; per-user-scoped scan tasks; extensive test projects (integration tests in-repo).
**License.** GPL-3.0.
**Porting decision — LEARN ONLY (attributed: Helix research agent, checkout `d549d5b1609086c8190b9bee6eb48ebfed71e1df`).** GPL bars reuse; smart-filter expression design and age-restriction policy modeling are ideas to reimplement originally in Go.