# Audiobookshelf — per-repository research report

- Repo: https://github.com/advplyr/audiobookshelf · Revision: `edb1f7efc92ecacdef51aafb4d9d5054ce10604f` — detached checkout at the exact analysis-pinned revision; row(s) `advplyr-audiobookshelf`, `audiobookshelf`, `current_checkout_matches_analysis=true`.
- Commit: 2026-10-23 14:36:19 -0500 — "Version bump v2.37.2". Full non-shallow clone, `git fsck` clean.
- Ledger recomputed at pinned revision: 954 blobs, 14,215,208 bytes — matches preserved analysis (tracked_files `954`, `954`). Top extensions (count): js 507, vue 229, (none) 66, json 54, yaml 30, yml 13.

**Architecture / data flow.** Node.js (Express-like, plain JS 507 files) self-hosted audiobook/podcast server: `server/` services for scan, metadata providers, playback sync, users, OPDL feeds; Vue 2 client (229 .vue) with HLS streaming; JSON-file-first storage per library.
**Notable techniques.** Watcher-driven incremental scan with fingerprinting; playback-session sync across devices; custom-metadata-provider-specification.yaml documents a third-party provider protocol — a clean extensibility seam Helix could mirror.
**License.** GPL-3.0.
**Porting decision — LEARN ONLY (attributed: Helix research agent, checkout `edb1f7efc92ecacdef51aafb4d9d5054ce10604f`).** GPL; the transferable design is its provider-spec pattern and scan fingerprinting, reimplemented originally in Go if needed.