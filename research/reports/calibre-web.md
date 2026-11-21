# Calibre-Web — per-repository research report

- Repo: https://github.com/janeczku/calibre-web · Revision: `6e840acbda8d62fdd159bf3855c01872d629a914` — detached checkout at the exact analysis-pinned revision; row(s) `calibre-web`, `current_checkout_matches_analysis=true`.
- Commit: 2026-09-20 09:31:43 +0200 — "Fix for #3702 (Handle NULL author_sort during view lists a". Full non-shallow clone, `git fsck` clean.
- Ledger recomputed at pinned revision: 972 blobs, 24,911,139 bytes — matches preserved analysis (tracked_files `972`). Top extensions (count): js 194, bcmap 168, ftl 111, py 101, svg 92, html 65.

**Architecture / data flow.** Flask app (`cps/`, 101 py): SQLAlchemy over the Calibre library DB, full-text search via Whoosh, OPDS, email-to-device, Kobo sync stubs, user roles; Jinja UI with bundled pdf.js (168 bcmap).
**Notable techniques.** Read-mostly access to a foreign canonical DB (Calibre) without owning it — the same posture Helix takes toward existing libraries; graceful NULL-metadata repair paths.
**License.** GPL-3.0 text at this pinned revision (LICENSE). (Upstream previously AGPL; the pin governs — recheck on re-pin.)
**Porting decision — LEARN ONLY (attributed: Helix research agent, checkout `6e840acbda8d62fdd159bf3855c01872d629a914`).** Copyleft; reference for Calibre-DB compatibility edge cases Helix's catalog importer must survive.