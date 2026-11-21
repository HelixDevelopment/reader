# Bookokrat — per-repository research report

- Repo: https://github.com/bugzmanov/bookokrat · Revision: `9c02ee3b3a0f86791b3917cb1a93436d92a8603e` (matches analysis pin, `tracked_files=371` ✓; checked out detached from `main`) · Commit date: 2026-10-21 08:39:14 -0400 — "issue/206 - flatten restored images onto white". Full non-shallow clone, fsck clean.
- Ledger: 47,154,944 bytes. Top: rs 161, svg 109, tape 21 (vhs test fixtures), pdf 14, sh 11.

**Architecture / data flow.** Tauri (Rust core, React shell) "all-in-one knowledge center": `src-tauri/` commands over rusqlite metadata DB; import pipeline converts epub/mobi/pdf/cbz into normalized internal representations with image post-processing (e.g. flattening transparency); library browsing, annotation store, and a plugin/webview reading surface.

**Notable techniques.** VHS golden-file tests for the conversion pipeline; aggressive image preprocessing at ingest (white-flatten for e-ink) — directly relevant to Helix's future e-ink cover pipeline; SQLite-first desktop design without a server.

**License.** GNU AGPL-3.0-only (LICENSE). Network copyleft applies to any distribution.

**Porting decision — LEARN ONLY (attributed: Helix research agent, checkout `9c02ee3b3a`).** AGPL blocks code flow into Helix; harvestable ideas: ingest-time image normalization for grayscale readers, Tauri command/DB layering as a local-library reference, golden-fixture testing of converters.
