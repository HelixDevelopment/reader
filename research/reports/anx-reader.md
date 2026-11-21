# anx-reader — per-repository research report

## Pinned revision and file ledger

- Repository: https://github.com/anxcye/anx-reader
- Revision: `0c7fc1ad4551f7dbbeba6a6468f69d830957fa2a` (matches `research/analysis_summary.json` slug `anx-reader`, tree `c5cd5938f6f9ef32009b1a093dc49de679c9531c`)
- Branch: `origin/develop` (default branch; commit is contained in it)
- Commit date: 2026-10-17 20:54:44 +0800 — `release: v1.15.0-alpha.35`
- Checkout evidence: full non-shallow clone, `git fsck` clean, `is-shallow-repository=false`.
- Ledger (recomputed from `git ls-tree -r HEAD`, matches analysis `tracked_files=1304`): 1,304 files, 58,057,708 bytes. Top extensions (count, bytes): dart 725 / 2,921,406; png 104 / 36,556,057; js 94 / 5,124,039 (vendored foliate-js assets); strings 43 / 88,315; md 31 / 273,801; yaml 22 / 48,629; json 20 / 629,012; arb 16 / 867,610.

## Architecture and data flow

Flutter/Dart cross-platform reader (Android, iOS, Linux, macOS, Windows, Web, HarmonyOS `ohos/`).

- `lib/dao/` — SQLite data access (`database.dart`, `book.dart`, `book_note.dart`, `reading_time.dart`, `tag.dart`, `theme.dart`): books, notes/highlights, reading sessions, tags.
- `lib/models/` — plain data models (book, bookmark, toc_item, reading_info, sync_state_model, remote_file).
- `lib/providers/` — Riverpod state layer (~40 providers): `current_reading`, `book_toc`, `bookmark`, `sync`/`sync_status`, `tts_providers`, statistics/heatmap/streak providers, `ai_*` providers.
- `lib/service/` — book lifecycle services: `book.dart` (open/metadata), `book_player/` (`book_player_server.dart` local HTTP server that hosts the webview reader and pairs devices), `bookshelf/`, `config/`, `convert_to_epub/`, `database_sync_manager.dart`, `font.dart`, `iap/`, and `ai/`.
- Rendering: EPUB/FB2/TXT rendering is delegated to vendored **foliate-js** (`assets/foliate-js/`, bundled JS renderer using pdf.js for fixed layout) loaded in a webview; `third_party/` pins forked `webview_cef`, `webdav_client`, `googleai_dart`, `langchain_google`.
- Sync: WebDAV-based file + database sync (`webdav_client`, `database_sync_manager`) with a companion web client reached through `book_player_server`.
- Flow: file import → metadata extract → DAO persist → Riverpod providers → book_player spawns local server → webview renders foliate-js → progress/notes flow back through JS bridge (`chapter_content_bridge.dart`) → DAO → sync layer uploads.

## Notable implementation techniques

- Webview-hosted foliate-js instead of native pagination: one renderer across all six targets; the Dart side only exchanges location/progress JSON over the bridge.
- Local HTTP book-player server doubles as device-pairing surface (read the same book on e-ink companion from the phone acting as server).
- FSRS-style spaced repetition over highlights (review cards derived from notes), reading-statistics heatmaps computed from DAO session rows.
- `chapter_split_rule`/`chapter_split_presets`: user-defined txt-to-epub chapter splitting on import.
- Per-platform conditional imports and a vendored CEF webview for desktop parity.

## License review

- Root `LICENSE`: MIT, Copyright (c) 2025 Anxcye.
- Bundled foliate-js assets are MIT; `third_party/` items (webdav_client MIT, googleai_dart BSD-3, langchain_google BSD/Apache-family) are permissively licensed; pubspec pins are normal package usage.
- No copyleft components detected on the Dart/JS code paths. MIT is compatible with Helix's licensing model; attribution must be preserved on any copied snippet.

## Porting decision — attributed

**Decision: LEARN + selective port, AT SOURCE LEVEL, with MIT attribution.** (Reviewer: Helix research agent, from pinned checkout `0c7fc1ad`.)

Rationale: MIT permits copying with attribution. The most transferable surface is not the Flutter UI but (a) the foliate-js-in-webview integration contract — how progress/location objects are exchanged and how CFI-like positions are persisted; (b) the chapter-splitting rules model for TXT import; (c) the WebDAV database-sync conflict shape. Helix's core is Go + web; porting Dart service code wholesale is not profitable — port designs and the JS bridge message shapes, keep Flutter layers as reference only. No code is copied yet; any future copy must carry the MIT notice and record the revision here.
