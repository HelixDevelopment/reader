# Foliate (GTK4 rewrite) — per-repository research report

- Repo: https://github.com/johnfactotum/foliate · Revision: `67b6676d3f936c5edea91d4d903385ef39dd25c0` (matches analysis pin, `tracked_files=140` = 139 blobs + 1 submodule entry ✓; default branch `gtk4`, checked out detached) · Commit date: 2026-04-08 15:11:30 +0000 — "Add CONTRIBUTING.md guidelines". Full non-shallow clone, fsck clean.
- Ledger: 8,960,089 bytes total. Top: po 34, js 24, ui 18, (none) 11, svg 11.

**Architecture / data flow.** GTK4/libadwaita GNOME reader written in JavaScript (GJS) with a WebKitGTK content view: `lib/` document backends (epub.js-derived, Mobi, PDF via Poppler-GObject, comic, FB2), a view layer paginating in WebKit, and GTK shells (`src/`, `.ui` files). Book data flows file → backend → metadata/view model → WebKit page with injected CSS.

**Notable techniques.** The pure-JS EPUB engine here is the ancestor of **foliate-js**, which anx-reader embeds (see anx-reader report) — the engine/UI separation is the cleanest in the ebook world; smart-render hints, hyphenation via Liang patterns, footnote popup architecture.

**License.** GNU GPL-3.0 (COPYING). The extracted `foliate-js` package separately carries MIT — verify per-tree which artifact you touch; at this revision the root tree is GPL-3.

**Porting decision — LEARN ONLY from this tree; the MIT foliate-js distribution may be used with attribution (attributed: Helix research agent, checkout `67b6676d3f`).** Helix should consume foliate-js as an MIT-licensed component if needed, never GPL code from this repo. Design input: document-backend interface shape and CFI-free position model.
