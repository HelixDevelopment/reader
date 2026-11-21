# DIY ESP32-S3 EPUB Reader (fork) — per-repository research report

- Repo: https://github.com/juicecultus/diy-esp32s3-epub-reader · Revision: `3b35576127dd2e3b27a46336e301c7b4793f95b7` — detached checkout at the exact analysis-pinned revision; row `diy-esp32s3-epub-reader`, `current_checkout_matches_analysis=true`.
- Commit: 2025-12-14 18:05:39 +0000 — "Port Atomic14 app to work on M5 Paper S3". Full non-shallow clone, `git fsck` clean.
- Ledger at pinned revision: 1454 blobs + 4 submodule entries, 75,918,464 bytes — matches analysis tracked_files=1458. Top extensions (count): h 451, c 285, mk 102, html 77, (none) 48, js 43.

**Architecture / data flow.** Fork of atomic14's reader ported to M5 Stack Paper S3 (ESP32-S3): bundled ESP-IDF components (451 h/285 c), FreeType included, browser-side layout preserved; +4 submodules.
**Notable techniques.** Board-port delta from atomic14's design — a checklist for Helix e-ink device support across MCU variants.
**License.** MIT (inherits atomic14 copyright + fork commits).
**Porting decision — PORT-WORTHY at protocol level (attributed: Helix research agent, checkout `3b35576127dd2e3b27a46336e301c7b4793f95b7`).** Same rationale as atomic14 report; MIT keeps device-port contracts free to reuse with notice.