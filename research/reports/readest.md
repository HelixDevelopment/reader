# Readest — per-repository research report

- Repo: https://github.com/readest/readest · Revision: `2f260ee10ff3500f77a43cfbb24254b7ac422aa7` — detached checkout at the exact analysis-pinned revision; row(s) `readest`, `current_checkout_matches_analysis=true`.
- Commit: 2026-10-28 01:16:05 +0800 — "fix(reader): never open the media viewer for note-referenc". Full non-shallow clone, `git fsck` clean.
- Ledger recomputed at pinned revision: 4484 blobs + 8 submodule entries, 130,932,490 bytes — matches preserved analysis (tracked_files `4492`). Top extensions (count): ts 2062, md 794, tsx 695, json 164, png 119, rs 96.

**Architecture / data flow.** Tauri v2 + Next.js cross-platform reader (2062 ts, 96 rs, 8 submodules incl. CEF fork): `apps/readest-app` client, sidecar PDF engine (pdfium via CEF/wasm), BookDir metadata store, Readest Sync server component, DeepL/AI translation seams.
**Notable techniques.** Dual-engine rendering (wasm epub + pdfium) behind one TS interface; library sync with per-book CRDT-like merge of positions; Docker self-host path.
**License.** AGPL-3.0.
**Porting decision — LEARN ONLY (attributed: Helix research agent, checkout `2f260ee10ff3500f77a43cfbb24254b7ac422aa7`).** AGPL; the position-merge strategy and pdfium-wasm packaging are the valuable references, reimplemented only.