# Chytanka web reader — per-repository research report

- Repo: https://github.com/chytanka/chytanka.github.io · Revision: `39b23eeb2abc0b9205695fd7b773ae616e2c3db5` — detached checkout at the exact analysis-pinned revision; row `chytanka`, `current_checkout_matches_analysis=true`.
- Commit: 2026-06-24 17:21:18 +0300 — "Merge pull request #60 from chytanka/develop". Full non-shallow clone, `git fsck` clean.
- Ledger at pinned revision: 390 blobs, 5,296,640 bytes — matches analysis tracked_files=390. Top extensions (count): ts 211, scss 59, html 46, svg 24, json 13, png 12.

**Architecture / data flow.** Angular (211 ts) PWA reader + Service Worker (`ngsw-config`) + small Node server (`server.ts`): EPUB rendering with epub.js-family engine, offline shelf, Ukrainian-language UX focus.
**Notable techniques.** Fully offline-capable book delivery via Workbox precaching; tidy i18n for a non-English-first reader market.
**License. NO LICENSE FILE at the pinned revision** — default copyright, all rights reserved; the repo is readable for study but legally unusable for reuse, and even its public "open-source" status is unconfirmed.
**Porting decision — BLOCKED pending license (attributed: Helix research agent, checkout `39b23eeb2abc0b9205695fd7b773ae616e2c3db5`).** Record as research-visible but port-prohibited; only behavioral ideas may be independently reimplemented.