# kiikoread / read-pico-reader — per-repository research report

- Repo: https://github.com/wegooo-cell/read-pico-reader · Revision: `2f1ef29d446b66ec1d0bf1afa39f1caa8622bbc8` — detached checkout at the exact analysis-pinned revision; row `read-pico-reader`, `current_checkout_matches_analysis=true`.
- Commit: 2026-10-27 11:07:45 +0800 — "release(pico): bind final rc101 font-path guard build". Full non-shallow clone, `git fsck` clean.
- Ledger at pinned revision: 1320 blobs, 64,749,484 bytes — matches analysis tracked_files=1320. Top extensions (count): h 378, c 258, py 133, txt 64, png 57, js 56.

**Architecture / data flow.** Raspberry Pi Pico-based e-ink reader firmware (378 h/258 c incl. vendored pico SDK headers), Python tooling (133 py), C/C++ font pipeline, staged RC releases with build guards.
**Notable techniques.** Font-path guard at release time; ultra-constrained rendering pipeline; THIRD_PARTY_NOTICES.md maintained.
**License. CUSTOM LICENSE ONLY** — root LICENSE is a bilingual "kiikoread license scope" policy (dated 2026-10-15), not any recognized open-source license; scope must be read before any use.
**Porting decision — PROHIBITED until license scope cleared (attributed: Helix research agent, checkout `2f1ef29d446b66ec1d0bf1afa39f1caa8622bbc8`).** Non-standard terms; no code or protocol reuse without explicit rights analysis. Kept pinned for completeness; treat as proprietary-adjacent.