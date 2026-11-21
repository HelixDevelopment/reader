# Komga — per-repository research report

- Repo: https://github.com/gotson/komga · Revision: `023ec905c10a35bbb8e29095420d266cd3adad60` — detached checkout at the exact analysis-pinned revision; row(s) `gotson-komga`, `komga`, `current_checkout_matches_analysis=true`.
- Commit: 2026-10-21 09:25:49 +0800 — "deps(webui): bump launch-editor from 2.14.1 to 2.14.2 in /". Full non-shallow clone, `git fsck` clean.
- Ledger recomputed at pinned revision: 1908 blobs, 95,653,947 bytes — matches preserved analysis (tracked_files `1908`, `1908`). Top extensions (count): kt 567, ts 483, vue 350, sql 93, json 67, mdx 47.

**Architecture / data flow.** Kotlin/Spring Boot media server (567 kt) + Vue/Nuxt UI (350 vue, 483 ts): book→series→library domain, JDBC/H2 with 93 SQL migrations, OPDS 1.1/2.0, automatic file scanning, read-progress tracking, Docker.
**Notable techniques.** Rock-solid migration versioning; media-analysis sidecars; clean REST/OPDS split — Helix's closest production-grade peer for catalog semantics.
**License.** MIT text at this pinned revision (LICENSE: "MIT License Copyright (c) 2019 Gauthier Roebroeck"). Note: upstream has used MPL-2.0 historically; the pinned tree governs, and any future re-pin must re-check this.
**Porting decision — PORT-WORTHY (attributed: Helix research agent, checkout `023ec905c10a35bbb8e29095420d266cd3adad60`).** MIT at this pin permits selective port; study targets: read-progress model and OPDS pagination semantics. Re-verify license identity before consuming any snippet.