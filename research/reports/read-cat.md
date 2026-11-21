# Read Cat — per-repository research report

- Repo: https://github.com/read-cat/read-cat · Revision: `da008dced50c56a3fd58f3a7240d811f56e88ce4` — detached checkout at the exact analysis-pinned revision; row `read-cat`, `current_checkout_matches_analysis=true`.
- Commit: 2024-11-09 18:59:00 +0800 — "Merge branch 'dev'". Full non-shallow clone, `git fsck` clean.
- Ledger at pinned revision: 347 blobs, 6,162,483 bytes — matches analysis tracked_files=347. Top extensions (count): ts 143, woff2 78, svg 59, vue 34, png 9, json 4.

**Architecture / data flow.** Electron + Vue (34 vue/143 ts) EPUB reader with custom font bundling (78 woff2): library, dictionary popups, TTS hooks; builder/make scripts for packaged desktop apps.
**Notable techniques.** Typography-first font packaging strategy; dictionary integration over webview selections.
**License.** GPL-3.0.
**Porting decision — LEARN ONLY (attributed: Helix research agent, checkout `da008dced50c56a3fd58f3a7240d811f56e88ce4`).** GPL; font-bundling layout ideas reimplemented natively if used.