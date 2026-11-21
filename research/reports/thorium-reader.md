# Thorium Reader — per-repository research report

## Pinned revision and file ledger

- Repository: https://github.com/edrlab/thorium-reader
- Revision: `1f3e03448eedf352238c0fe5fe4aa765b9e9a1f2` (matches `research/analysis_summary.json` slug `thorium-reader`, tree `f7d5caef9fe455aa0744f82a9169edb72c0631f6`)
- Branch: `origin/develop` (default), commit date 2026-10-28 18:21:09 +0100 — `chore(dev): Document project stack and targets in AGENTS.md`
- Checkout evidence: full non-shallow clone, `git fsck` clean.
- Ledger (recomputed from `git ls-tree -r HEAD`, matches analysis `tracked_files=1921`): 1,921 files, 55,117,469 bytes. Top extensions: ts 1,175 / 6,279,657; tsx 159 / 1,798,460; svg 227 / 357,126; json 60 / 3,612,364; md 59 / 1,011,172; scss 41 / 271,531; png 30 / 8,997,671. Largest tracked artifacts include `projects/opds-progression/fixtures/accessible_epub_3.epub` (4 MB) and prebuilt `.github_release/LIB/...` native libs.

## Architecture and data flow

Electron desktop reader (Windows/macOS/Linux incl. ARM64) for EPUB 2/3 incl. fixed-layout, PDF, CBZ, audiobooks/DIVINA, with LCP support.

- `src/main/` — Electron main process: `redux/` (state + sagas), `streamer/` (publication loading via Readium streamer), `db/` (local library persistence), `network/` (proxy/requests), `w3c/` (W3C Publication Manifest / timestamps), `customization/` (signed per-institution profile system), `sessions.ts`, `menu.ts`, `services/`, `cbz/`, `pdf/`, `converter/`, `storage/`, `zip/`.
- `src/renderer/` — React UI split into `library/` (catalog, OPDS browsing) and `reader/` (navigator chrome, TTS controls, bookmarks/highlights).
- `src/r2-xxx-js/` — vendored Readium Web toolkits: `r2-shared-js` (models), `r2-streamer-js` (publication server/parsers), `r2-navigator-js` (webview navigator), `r2-opds-js` (OPDS 1/2 models), `r2-lcp-js` (LCP license handling, keys, decryption).
- `src/common/` — types/messages shared across processes; `src/resources/ReadiumCSS/` — paginated CSS engine assets and licensed fonts.
- `projects/opds-progression/` — standalone TypeScript library implementing the OPDS progression/positions contract (with EPUB fixtures); `projects/opds-pkce-test-server` and `opds-downloader-test-server` are protocol test doubles; `projects/pdf-annotations` handles PDF annotation work.
- Flow: open file/OPDS acquisition → r2-parser → Publication model → r2-navigator webview with ReadiumCSS pagination → user interactions and progression links dispatch through redux/sagas → persisted in `db` → interop via library OPDS server endpoints on the local machine.

## Notable implementation techniques

- Progression links (JSON "progression" schema in `projects/opds-progression`) as a first-class serializable position model — directly relevant to Helix's structured progress resolver and `/syncs/progress` shapes.
- Signed customization profiles (`.thorium` files, public-key pair tooling) enable whitelabel institutional builds — a clean multi-tenant distribution pattern.
- Full-process separation: main owns I/O, LCP crypto, persistence; renderer is pure React with typed IPC messages in `src/common`.
- TTS via `@readium/speech` + navigator hooks; media-overlays aware.
- Dedicated test servers for OPDS PKCE/download flows make protocol compliance testable offline — the pattern Helix already mirrors for KOSync tests.

## License review

- `LICENSE`: BSD-3-Clause, Copyright (c) 2017, EDRLab. Permissive; redistribution and reuse allowed with retention of copyright/notice/disclaimer.
- Sub-licenses inside the tree are themselves permissive or clearly scoped: ReadiumCSS (BSD-3), r2-*-js (BSD-3), fonts under `src/resources/ReadiumCSS/fonts/LICENSE-*` (SIL OFL families), test fixture NOTICEs. `test/main/redux/sagas/api/publication/import/licenseNotifications.test.ts` enforces notice preservation in exported packages.

## Porting decision — attributed

**Decision: PORT-WORTHY at design and selective-source level (BSD-3 permits copying with attribution).** (Reviewer: Helix research agent, from pinned checkout `1f3e0344`.)

Rationale: This is the single most license-compatible high-value reference in the research set. Concrete port candidates, each to be recorded in the porting ledger when actually consumed: (1) the `projects/opds-progression` schema + TypeScript validation approach as the model for Helix's structured internal progress resolver wire format; (2) r2-opds-js OPDS 2 feed/entry field mapping for Helix's Gin OPDS handlers; (3) the progression-position semantics used by the navigator for CFI-free resume. Wholesale reuse of Electron/React UI is out of scope (stack mismatch), but no legal barrier exists — BSD-3 only requires preserving the EDRLab notice on copied expression.
