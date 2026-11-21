# Book/story — per-repository research report

- Repo: https://github.com/acclorite/book-story · Revision: `464891ec8cb00196d10b6dbaba3b7409c8fd27a8` (detached checkout at the exact analysis-pinned revision; row `book-s-story`, `current_checkout_matches_analysis=true`)
- Commit date: 2026-10-25 20:17:28 +0200 — "feat: update translations". Full non-shallow clone, fsck clean.
- Ledger: 766 blobs, 36,765,346 bytes — matches analysis `tracked_files=766`. Top extensions (count): kt 601, xml 45, png 29, ttf 28, json 21, txt 15.

**Architecture / data flow.** Kotlin Multiplatform (Compose Multiplatform) ebook reader for Android + desktop: `app/` shared Compose UI, data layer with SQLDelight for library/progress, archive-based format readers (epub/pdf/cbr/fb2) with a custom typesetting engine drawing text directly to the platform canvas (no webview), TTS via platform engines, and a companion web app for browsing one's shelf.

**Notable techniques.** Pure-native typesetting/pagination (font metrics + bidi) instead of HTML rendering — a proven alternative to the foliate-js/webview approach; per-book user styles stored as structured objects; KMP abstraction layer worth studying for Helix's e-ink vs. app targets.

**License.** GPL-3.0. Copyleft on all Kotlin code.

**Porting decision — LEARN ONLY (attributed: Helix research agent, checkout `464891ec8c`).** GPL blocks code reuse in Helix; the pagination-engine design and SQLDelight schema shape are study material. No snippets copied.
