# DIY ESP32 EPUB Reader (atomic14) — per-repository research report

- Repo: https://github.com/atomic14/diy-esp32-epub-reader · Revision: `2c2f57fdd7e2a788d14a0bcb26b9e845a47aac42` — detached checkout at the exact analysis-pinned revision; row `diy-esp32-epub-reader`, `current_checkout_matches_analysis=true`.
- Commit: 2022-11-29 11:56:48 +0000 — "Merge pull request #85 from hpsaturn/av/improved_ini_confi". Full non-shallow clone, `git fsck` clean.
- Ledger at pinned revision: 139 blobs + 3 submodule entries, 6,287,307 bytes — matches analysis tracked_files=142. Top extensions (count): h 46, cpp 29, html 8, c 8, (none) 6, png 6.

**Architecture / data flow.** ESP32 + Waveshare e-Paper firmware (C++/PlatformIO): Wi-Fi web uploader serves a browser-based EPUB front-end (8 html assets) that parses the book client-side and streams page bitmaps to the device over a tiny HTTP API; `lib/epub`-style glue + FreeType glyph rasterization on-device (3 submodules: epd driver, http server, filesystem).
**Notable techniques.** Split-rendering: expensive layout in the browser, cheap bitmap blit on MCU — the cheapest viable path for e-ink companion devices; partial-refresh discipline; OTA-safe partition table.
**License.** MIT (Chris Greening 2021).
**Porting decision — PORT-WORTHY at protocol level (attributed: Helix research agent, checkout `2c2f57fdd7e2a788d14a0bcb26b9e845a47aac42`).** MIT; its browser-to-e-ink paging protocol is the template for a Helix e-ink push viewer. No firmware reuse needed; the API contract is the asset.