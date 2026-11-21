# Quill (reMarkable browser) — per-repository research report

- Repo: https://github.com/quill-os/quill · Revision: `790ffd6fb73318a76888d66cb32c88b2477f09dd` — detached checkout at the exact analysis-pinned revision; row `quill`, `current_checkout_matches_analysis=true`.
- Commit: 2026-08-14 17:52:11 -0400 — "N367, N428: Show pop-up when USB is connected". Full non-shallow clone, `git fsck` clean.
- Ledger at pinned revision: 278 blobs + 1 submodule entries, 9,282,447 bytes — matches analysis tracked_files=279. Top extensions (count): png 103, cpp 49, h 49, ui 43, ttf 12, jpg 7.

**Architecture / data flow.** C++ Qt app for reMarkable e-ink tablets (49 cpp/49 h/43 .ui): `src/` browser+reader surfaces, `inkbox.pro` build, libs pinned as submodule; USB-connect UX per ticket series.
**Notable techniques.** E-ink-specific scroll/refresh management, deterministic repaint regions; device-integration popups for transfer flows.
**License.** GPL-3.0 plus `LEGAL.md`/`AUTHORS` documenting bundled parts.
**Porting decision — LEARN ONLY (attributed: Helix research agent, checkout `790ffd6fb73318a76888d66cb32c88b2477f09dd`).** GPL bars reuse; the repaint/refresh UX catalog for e-ink informs Helix UI specs as behavioral reference.