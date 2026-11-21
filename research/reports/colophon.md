# Colophon — per-repository research report

- Repo: https://github.com/cgillinger/colophon · Revision: `3676404ab253c7e0ef78fcda973a27424056114d` (detached checkout at the exact analysis-pinned revision; row `cgillinger-colophon`, `current_checkout_matches_analysis=true`)
- Commit date: 2026-10-25 15:47:10 +0100 — "Fix copy buttons on plain HTTP and clarify e-reader setup (v1.64)". Full non-shallow clone, fsck clean.
- Ledger: 475 blobs, 18,388,802 bytes — matches analysis. Top extensions (count): bcmap 168, py 112, js 48, md 27, png 22, html 17.

**Architecture / data flow.** Flask (Python) author-focused book-library web app: `app/` blueprints over a Calibre-authored metadata store, book folders organized by author, built-in PDF.js viewer assets (the 168 bcmap files), OPDS download links, e-reader-friendly setup flows, Docker deployment.

**Notable techniques.** Plain-HTTP affordance handling for e-ink browsers (copy buttons without secure-context APIs), direct-on-filesystem book organization without a heavy DB, THIRD_PARTY_LICENSES.md aggregation practice.

**License.** MIT (Copyright 2025-2026 Christian Gillinger).

**Porting decision — LEARN + idea port (attributed: Helix research agent, checkout `3676404ab2`).** MIT-compatible, but the app overlaps Calibre-Web territory Helix already surveyed; the specifically valuable piece is the e-reader-on-plain-HTTP UX hardening list and its author-folder catalog layout, reimplemented in Go rather than translated.
