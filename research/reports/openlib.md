# OpenLib — per-repository research report

- Repo: https://github.com/dstark5/openlib · Revision: `fd779d7dd8dee1ab07aaf189f406dc00275c6321` — detached checkout at the exact analysis-pinned revision; row(s) `openlib`, `current_checkout_matches_analysis=true`.
- Commit: 2026-01-18 18:16:14 +0530 — "Add automated release workflow with permissions". Full non-shallow clone, `git fsck` clean.
- Ledger recomputed at pinned revision: 217 blobs, 4,229,327 bytes — matches preserved analysis (tracked_files `217`). Top extensions (count): png 65, dart 33, svg 9, xml 8, ttf 8, xcconfig 8.

**Architecture / data flow.** Flutter mobile app (33 dart files — thin UI) for Calibre libraries: REST/OPDS client, offline shelf, release automation.
**Notable techniques.** Minimal-dependency Calibre REST client shape; fastlane-based mobile release pipeline.
**License.** GPL-3.0.
**Porting decision — LEARN ONLY (attributed: Helix research agent, checkout `fd779d7dd8dee1ab07aaf189f406dc00275c6321`).** Small GPL surface; documents what a mobile client actually needs from Helix's API (a useful requirements checklist, not code).