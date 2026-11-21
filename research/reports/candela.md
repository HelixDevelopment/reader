# Candela — per-repository research report

- Repo: https://github.com/techempower-org/candela · Revision: `26f079bc6d3dda3e30b8a6fc511990084cc0c838` — detached checkout at the exact analysis-pinned revision; row `candela`, `current_checkout_matches_analysis=true`.
- Commit: 2026-10-23 11:27:36 -0700 — "ci: pin every workflow action to a commit SHA, with Depend". Full non-shallow clone, `git fsck` clean.
- Ledger at pinned revision: 2099 blobs, 47,151,248 bytes — matches analysis tracked_files=2099. Top extensions (count): kt 1594, png 110, xml 102, md 81, kts 67, json 36.

**Architecture / data flow.** Android Kotlin media app (1594 kt): multi-module Gradle (`core-data`, `core-playback`, `core-llm`, `core-plugin-ksp`), KSP-generated plugin APIs, baseline profiles, CI actions pinned to full commit SHAs.
**Notable techniques.** Repo-wide supply-chain pinning of every GitHub action to a commit SHA with Dependabot follow-through — a security practice Helix should adopt verbatim; plugin contract codegen via KSP.
**License.** GPL-3.0.
**Porting decision — LEARN ONLY for code; adopt process practices (attributed: Helix research agent, checkout `26f079bc6d3dda3e30b8a6fc511990084cc0c838`).** GPL bars reuse; the action-SHA-pinning workflow and module decomposition are process/design adoptions, not code.