# Quire — per-repository research report

- Repo: https://github.com/bats-lang/quire · Revision: `4cbb7dd401dae870e71468665b9bd3e66b4a583e` — detached checkout at the exact analysis-pinned revision; row `quire`, `current_checkout_matches_analysis=true`.
- Commit: 2026-10-27 18:49:02 +0000 — "Move play-release.yml pin and its pwa-ref with the relock,". Full non-shallow clone, `git fsck` clean.
- Ledger at pinned revision: 3233 blobs, 16,548,080 bytes — matches analysis tracked_files=3233. Top extensions (count): dats 913, (none) 826, expect 781, bats 399, js 166, py 38.

**Architecture / data flow.** Programs written in the Bats language (913 .dats, 781 .expect golden files, 399 .bats test harness): dependent-types proof-carrying components, `bats.toml`/`bats.lock` dependency pinning, pwa homepage.
**Notable techniques.** Lockfile-pinned language toolchain, expect-style golden tests at massive scale — an interesting model for Helix's OPDS conformance fixtures.
**License. NO LICENSE FILE at the pinned revision** — all rights reserved; research-reading only.
**Porting decision — BLOCKED pending license (attributed: Helix research agent, checkout `4cbb7dd401dae870e71468665b9bd3e66b4a583e`).** Testing-methodology ideas only; zero code reuse.