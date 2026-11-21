# COPS (Calibre OPDS PHP) — per-repository research report

- Repo: https://github.com/seblucas/cops · Revision: `7828cc600e904941583227071b8f4efd3b9385fb` — detached checkout at the exact analysis-pinned revision; row(s) `seblucas-cops`, `cops`, `current_checkout_matches_analysis=true`.
- Commit: 2024-02-17 22:13:02 +0100 — "Update README.md". Full non-shallow clone, `git fsck` clean.
- Ledger recomputed at pinned revision: 388 blobs, 6,686,080 bytes — matches preserved analysis (tracked_files `388`, `388`). Top extensions (count): php 266, json 32, html 16, png 14, css 12, js 5.

**Architecture / data flow.** PHP OPDS 1.2 catalog ("Calibre OPDS PHP"): `base.php` bootstrap, `src/` classes building ATOM feeds directly from the Calibre SQLite library, thumbnail/acquisition endpoints, per-server `config.php`.
**Notable techniques.** Zero-build, zero-runtime-dependency feed generation straight from the Calibre DB; feed caching headers tuned for e-ink browsers; multi-library configuration with one install.
**License.** GPL-2.0 (COPYING).
**Porting decision — LEARN ONLY (attributed: Helix research agent, checkout `7828cc600e904941583227071b8f4efd3b9385fb`).** Tiny GPL surface but Helix's Go OPDS layer already implements the same feed shapes independently; value limited to confirming field-level OPDS 1.2 quirks that real e-reader clients depend on. No code reuse.