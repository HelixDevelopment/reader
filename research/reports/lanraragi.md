# LANraragi — per-repository research report

- Repo: https://github.com/difegue/lanraragi · Revision: `adfe9e75cbb8dd4db0fef2c2e35698d557c5a59b` — detached checkout at the exact analysis-pinned revision; row(s) `lanraragi`, `current_checkout_matches_analysis=true`.
- Commit: 2026-10-03 22:37:54 +0200 — "Bump brace-expansion from 5.0.9 to 5.0.12 (#1734)". Full non-shallow clone, `git fsck` clean.
- Ledger recomputed at pinned revision: 461 blobs + 1 submodule entries, 17,760,901 bytes — matches preserved analysis (tracked_files `462`). Top extensions (count): pm 100, md 53, png 46, t 40, json 28, (none) 25.

**Architecture / data flow.** Perl/Mojolicious archive server (100 .pm): SQLite (via Mojo) + Redis-backed job queue for metadata plugins, OPDS, reader UI, Docker native.
**Notable techniques.** Plugin-based metadata enrichment with runtime install; event-driven scan pipeline in a language Helix doesn't use — process ideas only.
**License.** MIT at this pinned revision (COPYING contains MIT text; historically GPL — pin governs, recheck on re-pin).
**Porting decision — LEARN + selective port (attributed: Helix research agent, checkout `adfe9e75cbb8dd4db0fef2c2e35698d557c5a59b`).** MIT at the pin keeps the plugin/queue decomposition portable in spirit; no Perl reuse planned.