# Myne — per-repository research report

- Repo: https://github.com/pool-of-tears/myne · Revision: `7ec1fdb19ebf19de13d7b61bcb52b3ce4878a579` — detached checkout at the exact analysis-pinned revision; row `myne`, `current_checkout_matches_analysis=true`.
- Commit: 2026-09-21 19:59:29 +0530 — "fix: null-safe text selection menu itemId (#315)". Full non-shallow clone, `git fsck` clean.
- Ledger at pinned revision: 299 blobs, 11,091,405 bytes — matches analysis tracked_files=299. Top extensions (count): kt 88, txt 64, xml 63, png 29, ttf 11, json 10.

**Architecture / data flow.** Android Kotlin reader (88 kt): offline catalog scraping, download manager, EPUB rendering via bundled web assets; 64 .txt legal/notices files show disciplined third-party notice shipping.
**Notable techniques.** Catalog-provider abstraction for remote sources — closest OSS analogue to Helix's OPDS acquisition client on mobile; explicit legal notice pipeline.
**License.** Apache-2.0.
**Porting decision — PORT-WITH-ATTRIBUTION at design level (attributed: Helix research agent, checkout `7ec1fdb19ebf19de13d7b61bcb52b3ce4878a579`).** Apache-2.0 permits reuse with notice + license retention; primary target is the provider-abstraction shape for Helix's mobile acquisition, not the Android UI.