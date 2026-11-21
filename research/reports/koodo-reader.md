# Koodo Reader — per-repository research report

- Repo: https://github.com/koodo-reader/koodo-reader · Pinned analysis revision `fe1abf4b8960f4e48a2a948659c3bb816a6e2606` is **no longer reachable upstream** (history rewritten on `dev`). Current checkout: full non-shallow clone at `4eadb2ec9fe98abd02e2f31712b95f8c0306e298` (2026-11-20, default branch `dev`), fsck clean. Row recorded `valid` with `current_checkout_matches_analysis=false`; recomputed ledger at HEAD: 878 files, 29,672,440 bytes (analysis row preserved 851 files at the lost revision). Top: tsx 224, bcmap 168 (pdf.js maps), ts 130, css 100, js 84.

**Architecture / data flow.** Electron/React web reader (desktop + browser extension + self-host Docker):.epub/mobi/pdf/cbz rendering via bundled epub.js/pdf.js in iframes, plugin architecture for cloud sync providers, IndexedDB/local storage for library, optional server backend (`docker/`) for multi-device sync.

**Notable techniques.** Popup/note-taking pipeline with highlight→excerpt→export flow; translation-driven i18n with 20+ locales; plugin isolation for third-party sync backends.

**License.** **GNU AGPL-3.0** (LICENSE at `4eadb2ec9f`). Network copyleft on the whole app.

**Porting decision — LEARN ONLY, no code (attributed: Helix research agent, checkout `4eadb2ec9f`).** AGPL is incompatible with Helix's server posture; UX patterns (highlight export, sync-provider plugin shape) are the usable output. Caveat for the record: this checkout cannot corroborate the preserved static analysis at the lost revision — the row is flagged non-matching rather than treated as ledger evidence.
