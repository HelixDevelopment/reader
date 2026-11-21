# SumatraPDF — per-repository research report

## Pinned revision and file ledger

- Repository: https://github.com/sumatrapdfreader/sumatrapdf
- Revision: `a30e4d6429419361132c23c797877f242092ea0e` (matches `research/analysis_summary.json` slug `sumatrapdf`, tree `83e38d974e4084f0f9cc0f2f0c85d2fb4e44dd88`)
- Branch: `origin/master` (default), commit date 2026-10-28 18:05:59 +0100 — `Deflake issue-6388 slider drag`
- Checkout evidence: full non-shallow clone, `git fsck` clean, `is-shallow-repository=false`.
- Ledger (recomputed from `git ls-tree -r HEAD`, matches analysis `tracked_files=5690`): 5,690 files, 198,724,072 bytes. Top extensions: h 1,557 / 22,315,824; c 1,061 / 42,476,643; cpp 770 / 26,332,745; ts 619 / 4,002,388 (bundled tooling/typescript under src/tools); otf 163 / 14,559,280; asm 145 / 6,999,553; vcxproj 78 / 4,754,101; txt 64 / 9,578,308.

## Architecture and data flow

Native Windows C++ reader (PDF, EPUB, MOBI, CBZ/CBR, FB2, CHM, XPS, DjVu), built with premake5/MSVC.

- `ext/` — vendored third-party engines: MuPDF (PDF/EPUB/MOBI/XPS/CBZ rendering core), chromium `base` subset, lcms, unrar/7z, zlib etc.; patches for each applied via `.patch` files.
- `src/mupdf/` — the glue layer: document loaders, `EngineXPS`-style backends, font management, EPUB/MOBI document implementations bridging MuPDF into Sumatra's `Document` interface.
- `src/` core UI: `SumatraPDF.cpp` (process entry, window/tab management), `MainWindow.cpp`, `TextViewWnd.cpp` + `Canvas`/`gfx` (custom Direct2D-rendered widget and text-view layer — no native controls), `Settings.cpp` (SettingsGUI/forward-sync prefs), `Tabs.cpp`, `Toolbar`, `FindWindow`, `PageThumbnails`, `TableOfContents`, `SidebarPanel`.
- Feature dialogs/modules live flat in `src/` (annotations: `AnnotPlacement`, `AnnotEditToolbar`, `AnnotFilterToolbar`, `SignDocumentDialog`; ebook: `EbookSettingsDialog`; reading aids: `ReadingAutoScroll`, `ReadAloud`, `RefHover`/`RefHoverCanvas` for footnote popups; `Favorites`, `NavFilesInFolder`, `CommandPalette`).
- `src/uia/` — UI Automation provider for accessibility; `src/ifilter/` — Windows Search iFilter; `src/previewer/` — Explorer thumbnail previewer; `cmd/` — auxiliary tools (`sumatrapdf-tool.cpp`).
- `src/ng/` — "next-gen" restructured components being incrementally migrated out of the legacy files.
- Flow: open file → format dispatch to MuPDF engine wrapper → paged/continuous `Document` model → Direct2D `Canvas` render loop with damage tracking → UI event loop; sync search and forward search map between DVI/PDF-like positions; settings persisted per file + global.

## Notable implementation techniques

- Own immediate-mode-ish D2D UI stack (custom scrollbars, overlay scrollbars, dark-mode handling via `DarkMode.cpp`) instead of Win32 common controls — relevant to any low-overhead renderer.
- Fast reload/delta rendering: change-monitor + partial invalidation; "keep window open" restart via saved session state.
- Small-binary discipline: premake-based build, dead-code stripping, DLL-size budgets documented in `docs/`.
- `RefHover` footnote hover rendering composites popup canvases; `MuPDF_Exports.cpp` controls exactly which engine symbols cross the boundary — a clean vendored-engine seam.
- `AdvancedSettingsDialog` + `RegistrySearchFilter`: power-user configuration discovered from a schema, not hard-coded dialogs.

## License review — (A)GPLv3 with BSD islands

- `readme.md`: "SumatraPDF is a multi-format … reader for Windows under (A)GPLv3 license, with some code under BSD license (see AUTHORS)". `COPYING` is GPLv3; `COPYING.BSD` covers BSD-licensed parts and `AUTHORS` enumerates per-component notices.
- The "(A)GPL" wording is the crux: Sumatra links MuPDF, which is AGPL-3.0. Distributing Sumatra with the MuPDF build obligates AGPL-3.0 for the combined work (Sumatra historically also offered a non-AGPL commercial-engine build; the open tree here is the AGPL-engine one). Any derivative must treat the whole as AGPL-3.0 unless MuPDF is swapped out.
- `ext/` contains many additional notices (unrar is non-free and shipped behind a separate flag upstream) — a wholesale copy drags in per-dependency review.

## Porting decision — attributed

**Decision: LEARN ONLY — no code port.** (Reviewer: Helix research agent, from pinned checkout `a30e4d64`.)

Rationale: Helix core is Go (Apache/MIT-style posture) serving web and e-ink clients; importing (A)GPLv3 C++ would either force AGPL relicensing of the consuming component or require a clean-process separation not worth the value — the techniques worth having (damage-tracked canvas rendering, delta reload, vendored-engine export seam, advanced-settings schema discovery) are design-level and reimplementable; the code is Windows-specific C++ with no direct binding to Helix's stack. Recorded as `researched_not_ported_license` in the porting ledger, with the MuPDF AGPL caveat noted for any future PDF-engine selection work.
