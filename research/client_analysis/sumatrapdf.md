# sumatrapdf — automated static evidence

- Inventory URL: https://github.com/sumatrapdfreader/sumatrapdf
- Origin: https://github.com/sumatrapdfreader/sumatrapdf
- Revision: `a30e4d6429419361132c23c797877f242092ea0e`
- Tree: `83e38d974e4084f0f9cc0f2f0c85d2fb4e44dd88`
- Analysis status: **automated_static_index_complete**; manual review: **pending**.
- Keyword hits are leads for code review, not proof of working support.

## Tree and languages

Tracked files: 5690. Text files scanned: 5353 (94834412 bytes). Files skipped by size, binary classification, or sensitive-name policy remain indexed in the compressed file ledger.

- C/C++ header: 1557 files
- C: 1061 files
- C++: 771 files
- TypeScript: 619 files
- Java: 74 files
- JavaScript: 50 files
- Python: 45 files
- Shell: 25 files
- HTML: 6 files
- XML: 5 files
- C++ header: 4 files
- CSS: 3 files
- Objective-C++: 3 files
- C#: 2 files
- Lua: 2 files
- Go: 2 files
- Objective-C: 1 files

## License review

Status: heuristic_requires_legal_review. Candidate notices: AGPL-3.0, Apache-2.0, BSD-2-Clause, BSD-3-Clause, CC0, GPL-2.0, GPL-3.0, LGPL, MIT, MPL-2.0, OFL-1.1.
- `COPYING`
- `COPYING.BSD`
- `ext/a-brotli/LICENSE`
- `ext/a-bzip2/LICENSE`
- `ext/a-freetype/LICENSE.TXT`
- `ext/a-harfbuzz/COPYING`
- `ext/a-jbig2dec/COPYING`
- `ext/a-jbig2dec/LICENSE`
- `ext/a-lcms2/LICENSE`
- `ext/a-libarchive/COPYING`
- `ext/a-libwebp/COPYING`
- `ext/a-mujs/COPYING`
- `ext/a-openjpeg/LICENSE`
- `ext/a-unrar/license.txt`
- `ext/a-zlib/LICENSE`
- `ext/a-zopfli/COPYING`
- `ext/chmdec/LICENSE.md`
- `ext/cmark-gfm/COPYING`
- `ext/darkmodelib/LICENSE-MIT.md`
- `ext/darkmodelib/LICENSE.md`
- `ext/darkmodelib/docs/LICENSE-PolyHook_2_0.md`
- `ext/darkmodelib/docs/LICENSE-UAHMenuBar.md`
- `ext/darkmodelib/docs/LICENSE-win32-darkmode.md`
- `ext/dav1d/COPYING`
- `ext/libjpeg-turbo/LICENSE.md`
- `ext/mupdf/COPYING`
- `ext/mupdf/docs/license.md`
- `ext/mupdf/resources/fonts/droid/NOTICE`
- `ext/mupdf/resources/fonts/han/LICENSE.txt`
- `ext/mupdf/resources/fonts/noto/COPYING`
- `ext/synctex/LICENSE`
- `ext/tesseract/LICENSE`
- `ext/webview2/LICENSE.txt`
- `ext/zint/LICENSE`
- `ext/zlib-ng/LICENSE.md`
- `ext/zxing-cpp/LICENSE`

## Feature evidence

### ai_llm

Role-weighted hits: `{"documentation": 132, "source": 7531, "test": 364}`.

- `.claude/skills/update-brotli/SKILL.md:50` (source; matched `rag`)
- `.claude/skills/update-brotli/SKILL.md:51` (source; matched `rag`)
- `.claude/skills/update-brotli/SKILL.md:51` (source; matched `rag`)
- `.claude/skills/update-gumbo/SKILL.md:48` (source; matched `rag`)
- `.claude/skills/update-harfbuzz/SKILL.md:28` (source; matched `rag`)
- Additional evidence retained in JSON: 35 records.

### annotations

Role-weighted hits: `{"documentation": 727, "source": 8856, "test": 1344}`.

- `.clangd:220` (source; matched `Annotation`)
- `.clangd:267` (source; matched `Annotation`)
- `.clangd:285` (source; matched `Highlight`)
- `.claude/skills/verify-mac/SKILL.md:41` (source; matched `highlight`)
- `.claude/skills/verify-mac/SKILL.md:47` (source; matched `highlight`)
- Additional evidence retained in JSON: 35 records.

### backup_restore

Role-weighted hits: `{"documentation": 48, "source": 1358, "test": 552}`.

- `.claude/skills/verify-mac/SKILL.md:12` (source; matched `restore`)
- `.claude/skills/verify/SKILL.md:12` (source; matched `restore`)
- `agents.md:37` (source; matched `restore`)
- `agents.md:120` (source; matched `restore`)
- `agents.md:183` (source; matched `snapshot`)
- Additional evidence retained in JSON: 35 records.

### collections_smart

Role-weighted hits: `{"configuration": 2, "documentation": 17, "source": 337, "test": 2}`.

- `cmd/upload-fonts.ts:174` (source; matched `collection`)
- `docs/gen_docs.render.js:148` (documentation; matched `collection`)
- `docs/gen_docs.search.html:2` (documentation; matched `collection`)
- `docs/gen_docs.search.js:2` (documentation; matched `collection`)
- `docs/gen_docs.search.js:78` (documentation; matched `collection`)
- Additional evidence retained in JSON: 35 records.

### comics_manga

Role-weighted hits: `{"configuration": 1, "documentation": 37, "source": 1378, "test": 278}`.

- `.github/workflows/linux-daily.yml:29` (configuration; matched `cbz`)
- `cmd/deps-build-defs.ts:541` (source; matched `cbz`)
- `cmd/deps-build-defs.ts:541` (source; matched `cbz`)
- `cmd/deps-build-defs.ts:634` (source; matched `cbz`)
- `cmd/gen-commands.ts:34` (source; matched `Manga`)
- Additional evidence retained in JSON: 35 records.

### compression_cache

Role-weighted hits: `{"configuration": 4, "documentation": 141, "source": 13545, "test": 571}`.

- `.clangd:283` (source; matched `Thumbnail`)
- `.clangd:371` (source; matched `Cache`)
- `.claude/skills/update-brotli/SKILL.md:2` (source; matched `brotli`)
- `.claude/skills/update-brotli/SKILL.md:3` (source; matched `Brotli`)
- `.claude/skills/update-brotli/SKILL.md:3` (source; matched `brotli`)
- Additional evidence retained in JSON: 35 records.

### conversion

Role-weighted hits: `{"documentation": 86, "source": 1082, "test": 16}`.

- `.clang-tidy:27` (source; matched `conversion`)
- `.clangd:31` (source; matched `conversion`)
- `agents.md:198` (source; matched `conversion`)
- `agents.md:321` (source; matched `converter`)
- `agents.md:347` (source; matched `epubPropsMap`, `pdfCreatorPropsMap`, `mupdfPropsMap`, `pdf`)
- Additional evidence retained in JSON: 35 records.

### daisy_accessibility

Role-weighted hits: `{"documentation": 72, "source": 46, "test": 7}`.

- `docs/gen_docs.fulltext_search.js:169` (documentation; matched `aria-`)
- `docs/gen_docs.fulltext_search.js:174` (documentation; matched `aria-`)
- `docs/gen_docs.fulltext_search.js:175` (documentation; matched `aria-`)
- `docs/gen_docs.render.js:10` (documentation; matched `aria-`)
- `docs/gen_docs.render.js:11` (documentation; matched `aria-`)
- Additional evidence retained in JSON: 35 records.

### database

Role-weighted hits: `{"documentation": 109, "source": 2502, "test": 74}`.

- `COPYING:627` (documentation; matched `redis`)
- `COPYING:637` (documentation; matched `redis`)
- `COPYING:657` (documentation; matched `redis`)
- `COPYING.BSD:3` (documentation; matched `Redis`)
- `COPYING.BSD:6` (documentation; matched `Redis`)
- Additional evidence retained in JSON: 35 records.

### deduplication

Role-weighted hits: `{"documentation": 34, "source": 1235, "test": 54}`.

- `.vscode/tasks.json:18` (source; matched `duplicate`)
- `.vscode/tasks.json:42` (source; matched `duplicate`)
- `.vscode/tasks.json:66` (source; matched `duplicate`)
- `cmd/amalgam.ts:146` (source; matched `dedup`)
- `cmd/amalgam.ts:150` (source; matched `dedup`)
- Additional evidence retained in JSON: 35 records.

### deployment

Role-weighted hits: `{"configuration": 2, "documentation": 9, "source": 63, "test": 5}`.

- `agents.md:166` (source; matched `docker`)
- `agents.md:457` (source; matched `GitHub Actions`)
- `cmd/helper/ci-build.ts:2` (source; matched `GitHub Actions`)
- `cmd/helper/ng-compile.ts:357` (source; matched `systemD`)
- `cmd/helper/ng-compile.ts:914` (source; matched `systemD`)
- Additional evidence retained in JSON: 35 records.

### dictionary_language

Role-weighted hits: `{"documentation": 92, "source": 2251, "test": 96}`.

- `.clangd:429` (source; matched `Translation`)
- `.claude/skills/update-harfbuzz/SKILL.md:51` (source; matched `translation`)
- `.claude/skills/update-lcms2/SKILL.md:51` (source; matched `translation`)
- `AUTHORS:35` (documentation; matched `Translation`)
- `agents.md:183` (source; matched `translation`)
- Additional evidence retained in JSON: 35 records.

### http_protocols

Role-weighted hits: `{"configuration": 2, "documentation": 33, "source": 5032, "test": 60}`.

- `.claude/skills/update-brotli/SKILL.md:2` (source; matched `brotli`)
- `.claude/skills/update-brotli/SKILL.md:3` (source; matched `Brotli`)
- `.claude/skills/update-brotli/SKILL.md:3` (source; matched `brotli`)
- `.vscode/c_cpp_properties.json:12` (source; matched `brotli`)
- `cmd/amalgam.ts:472` (source; matched `brotli`)
- Additional evidence retained in JSON: 35 records.

### lcp_drm

Role-weighted hits: `{"documentation": 12, "source": 69}`.

- `COPYING:6` (documentation; matched `license document`)
- `COPYING:392` (documentation; matched `license document`)
- `COPYING:395` (documentation; matched `license document`)
- `ext/a-freetype/docs/GPLv2.TXT:7` (documentation; matched `license document`)
- `ext/a-jbig2dec/COPYING:6` (documentation; matched `license document`)
- Additional evidence retained in JSON: 34 records.

### local_library

Role-weighted hits: `{"documentation": 18, "source": 626, "test": 65}`.

- `.claude/skills/update-gumbo/SKILL.md:48` (source; matched `in place`)
- `.claude/skills/update-unrar/SKILL.md:51` (source; matched `filesystem`)
- `.vscode/settings.json:80` (source; matched `filesystem`)
- `agents.md:11` (source; matched `in place`)
- `agents.md:39` (source; matched `in place`)
- Additional evidence retained in JSON: 35 records.

### metadata_enrichment

Role-weighted hits: `{"documentation": 54, "source": 902, "test": 54}`.

- `.claude/skills/verify/SKILL.md:38` (source; matched `metadata`)
- `cmd/gen-litdoc.ts:236` (source; matched `metadata`)
- `cmd/gen-litdoc.ts:236` (source; matched `metadata`)
- `cmd/gen-litdoc.ts:238` (source; matched `metadata`)
- `cmd/gen-settings.ts:1073` (source; matched `metadata`)
- Additional evidence retained in JSON: 35 records.

### metadata_review

Role-weighted hits: `{"source": 20}`.

- `cmd/move-header-fn-comments.ts:1035` (source; matched `DRY RUN`)
- `ext/a-brotli/brotli.h:856` (source; matched `metadata block`)
- `ext/a-brotli/brotli.h:859` (source; matched `metadata_chunk_func as more metadata block`)
- `ext/a-brotli/brotli.h:863` (source; matched `metadata block`)
- `ext/a-brotli/brotli_internal.h:1518` (source; matched `LOCK_HEADER_METADATA`)
- Additional evidence retained in JSON: 4 records.

### multiuser_auth

Role-weighted hits: `{"documentation": 8, "source": 59, "test": 8}`.

- `COPYING:311` (documentation; matched `authorization`)
- `agents.md:203` (source; matched `authorization`)
- `cmd/crashes.ts:470` (source; matched `Authorization`)
- `cmd/crashes.ts:471` (source; matched `Authorization`)
- `cmd/gen-settings.ts:878` (source; matched `Authorization`)
- Additional evidence retained in JSON: 35 records.

### network_sources

Role-weighted hits: `{"source": 478, "test": 12}`.

- `cmd/helper/ci-build.ts:101` (source; matched `S3`)
- `cmd/helper/ci-build.ts:142` (source; matched `s3`)
- `cmd/helper/ci-build.ts:147` (source; matched `s3`)
- `cmd/markdown-it.min.js:2` (source; matched `ftp`)
- `cmd/markdown-it.min.js:2` (source; matched `ftp`)
- Additional evidence retained in JSON: 35 records.

### notes_export

Role-weighted hits: `{"source": 1}`.

- `ext/mupdf/platform/java/mupdf_native.h:5641` (source; matched `EXPORT jobjectArray JNICALL Java_com_artifex_mupdf_fitz_PDFPage_getAnnotations`)

### opensearch

Role-weighted hits: `{"documentation": 1}`.

- `docs/gen_docs.fulltext_search.js:597` (documentation; matched `openSearch`)

### pdf_fixed_layout

Role-weighted hits: `{"configuration": 44, "documentation": 1242, "source": 16432, "test": 447}`.

- `.clang-tidy:46` (source; matched `mupdf`)
- `.clang-tidy:49` (source; matched `mupdf`)
- `.clangd:17` (source; matched `mupdf`)
- `.clangd:273` (source; matched `Mupdf`)
- `.claude/skills/fix-crash/SKILL.md:58` (source; matched `mupdf`)
- Additional evidence retained in JSON: 35 records.

### permissions

Role-weighted hits: `{"configuration": 6, "documentation": 191, "source": 1159, "test": 8}`.

- `.claude/skills/verify-mac/SKILL.md:70` (source; matched `permission`)
- `.github/workflows/build.yml:7` (configuration; matched `permission`)
- `.github/workflows/codeql.yml:24` (configuration; matched `permission`)
- `.github/workflows/linux-daily.yml:7` (configuration; matched `permission`)
- `.github/workflows/mac-daily.yml:7` (configuration; matched `permission`)
- Additional evidence retained in JSON: 35 records.

### progress_sync

Role-weighted hits: `{"source": 22, "test": 3}`.

- `cmd/gen-settings.ts:527` (source; matched `reading progress`)
- `cmd/gen-settings.ts:1192` (source; matched `reading progress`)
- `cmd/gen-settings.ts:1518` (source; matched `reading progress`)
- `cmd/ng-gen-settings.ts:527` (source; matched `reading progress`)
- `cmd/ng-gen-settings.ts:1192` (source; matched `reading progress`)
- Additional evidence retained in JSON: 19 records.

### reader_typography

Role-weighted hits: `{"documentation": 385, "source": 4955, "test": 220}`.

- `.clangd:234` (source; matched `Theme`)
- `.clangd:423` (source; matched `Theme`)
- `.claude/skills/verify/SKILL.md:55` (source; matched `dark mode`)
- `.claude/skills/verify/SKILL.md:57` (source; matched `theme`)
- `.claude/skills/verify/SKILL.md:58` (source; matched `theme`)
- Additional evidence retained in JSON: 35 records.

### search

Role-weighted hits: `{"documentation": 1, "source": 14}`.

- `cmd/gen-docs.ts:115` (source; matched `full-text search`)
- `docs/wishlist.txt:96` (documentation; matched `full-text search`)
- `ext/mupdf/resources/hyphen/core/de:10559` (source; matched `fts5`)
- `ext/mupdf/resources/hyphen/extra/is:918` (source; matched `fts5`)
- `ext/mupdf/resources/hyphen/extra/is:1185` (source; matched `fts5`)
- Additional evidence retained in JSON: 8 records.

### security

Role-weighted hits: `{"configuration": 21, "documentation": 431, "source": 3323, "test": 273}`.

- `.clang-format:1` (source; matched `https`)
- `.clang-format:2` (source; matched `https`)
- `.clangd:1` (source; matched `https`)
- `.github/workflows/build.yml:20` (configuration; matched `credential`)
- `.github/workflows/build.yml:27` (configuration; matched `SECRET`)
- Additional evidence retained in JSON: 35 records.

### testing

Role-weighted hits: `{"configuration": 3, "documentation": 17, "source": 63, "test": 62}`.

- `.claude/skills/fix-crash/SKILL.md:56` (source; matched `unit test`)
- `.claude/skills/fix-crashes/SKILL.md:93` (source; matched `unit test`)
- `.claude/skills/verify-mac/SKILL.md:78` (source; matched `Unit test`)
- `.claude/skills/verify/SKILL.md:78` (source; matched `Unit test`)
- `.github/workflows/windows-daily.yml:66` (configuration; matched `unit test`)
- Additional evidence retained in JSON: 35 records.

### tts_audio

Role-weighted hits: `{"documentation": 21, "source": 76, "test": 12}`.

- `cmd/gen-settings.ts:1800` (source; matched `text-to-speech`)
- `cmd/gen-settings.ts:1806` (source; matched `text-to-speech`)
- `cmd/ng-gen-settings.ts:1767` (source; matched `text-to-speech`)
- `cmd/ng-gen-settings.ts:1773` (source; matched `text-to-speech`)
- `docs/plus-merge.md:9` (documentation; matched `tts`)
- Additional evidence retained in JSON: 35 records.

### watch_import

Role-weighted hits: `{"source": 36, "test": 3}`.

- `ext/liblzma/api/lzma.h:185` (source; matched `auto-import`)
- `ext/mupdf/Makefile:550` (source; matched `inotify`)
- `ext/mupdf/Makefile:553` (source; matched `inotify`)
- `src/MergePdf.cpp:1922` (source; matched `file watcher`)
- `src/MergePdf.cpp:1948` (source; matched `file watcher`)
- Additional evidence retained in JSON: 14 records.


## Format evidence

### cb7_cbt_cba

Role-weighted hits: `{"documentation": 1, "source": 36}`.

- `cmd/helper/ng-mac-bundle.ts:23` (source; matched `cb7`)
- `cmd/helper/ng-mac-bundle.ts:23` (source; matched `cbt`)
- `cmd/helper/ng-targets.ts:988` (source; matched `cbt`)
- `cmd/show-preview-reg.ts:21` (source; matched `.cb7`)
- `cmd/show-preview-reg.ts:21` (source; matched `.cbt`)
- Additional evidence retained in JSON: 25 records.

### cbr

Role-weighted hits: `{"documentation": 2, "source": 28, "test": 22}`.

- `cmd/helper/ng-mac-bundle.ts:23` (source; matched `cbr`)
- `cmd/ng-deb.ts:148` (source; matched `CBR`)
- `cmd/show-preview-reg.ts:21` (source; matched `.cbr`)
- `docs/manual.shell.html:9` (documentation; matched `cbr`)
- `ext/mupdf/CHANGES:351` (source; matched `CBR`)
- Additional evidence retained in JSON: 33 records.

### cbz

Role-weighted hits: `{"configuration": 1, "documentation": 22, "source": 149, "test": 95}`.

- `.github/workflows/linux-daily.yml:29` (configuration; matched `.cbz`)
- `cmd/deps-build-defs.ts:541` (source; matched `cbz`)
- `cmd/deps-build-defs.ts:634` (source; matched `cbz`)
- `cmd/gen-settings.ts:1872` (source; matched `CBZ`)
- `cmd/helper/ng-mac-bundle.ts:23` (source; matched `cbz`)
- Additional evidence retained in JSON: 35 records.

### daisy

Role-weighted hits: `{"test": 2}`.

- `tests/issue-5943.ts:59` (test; matched `.daisy`)
- `tests/toc-title-fallback.ts:93` (test; matched `.daisy`)

### djvu

Role-weighted hits: `{"configuration": 1, "documentation": 31, "source": 102, "test": 12}`.

- `.github/workflows/linux-daily.yml:29` (configuration; matched `.djvu`)
- `AUTHORS:47` (documentation; matched `djvu`)
- `cmd/gen-commands.ts:122` (source; matched `Djvu`)
- `cmd/gen-settings.ts:1368` (source; matched `DjVu`)
- `cmd/gen-settings.ts:1872` (source; matched `DjVu`)
- Additional evidence retained in JSON: 35 records.

### doc_docx

Role-weighted hits: `{"documentation": 36, "source": 132, "test": 19}`.

- `cmd/amalgam.ts:639` (source; matched `docx`)
- `cmd/deps-build-defs.ts:636` (source; matched `docx`)
- `cmd/gen-settings.ts:1139` (source; matched `.doc`)
- `cmd/gen-settings.ts:1146` (source; matched `.doc`)
- `cmd/ng-gen-settings.ts:1139` (source; matched `.doc`)
- Additional evidence retained in JSON: 35 records.

### epub

Role-weighted hits: `{"configuration": 1, "documentation": 97, "source": 295, "test": 358}`.

- `.github/workflows/linux-daily.yml:29` (configuration; matched `.epub`)
- `cmd/deps-build-defs.ts:700` (source; matched `epub`)
- `cmd/gen-code.ts:310` (source; matched `epub`)
- `cmd/gen-settings.ts:245` (source; matched `epub`)
- `cmd/gen-settings.ts:553` (source; matched `EPUB `)
- Additional evidence retained in JSON: 35 records.

### fb2

Role-weighted hits: `{"documentation": 15, "source": 137, "test": 36}`.

- `cmd/gen-settings.ts:1872` (source; matched `FB2`)
- `cmd/gen-settings.ts:2026` (source; matched `FB2`)
- `cmd/helper/ng-mac-bundle.ts:26` (source; matched `fb2`)
- `cmd/ng-deb.ts:150` (source; matched `FictionBook`)
- `cmd/ng-gen-settings.ts:1839` (source; matched `FB2`)
- Additional evidence retained in JSON: 35 records.

### html_xhtml

Role-weighted hits: `{"documentation": 105, "source": 1040, "test": 153}`.

- `.clang-format:1` (source; matched `.html`)
- `AUTHORS:45` (documentation; matched `.html`)
- `AUTHORS:49` (documentation; matched `.html`)
- `COPYING:674` (documentation; matched `.html`)
- `cmd/crashes.ts:1290` (source; matched `xHtml`)
- Additional evidence retained in JSON: 35 records.

### lcp_protected

Role-weighted hits: `{"source": 2}`.

- `ext/tesseract/src/ccutil/universalambigs.h:5533` (source; matched `Lcp`)
- `ext/tesseract/src/ccutil/universalambigs.h:16652` (source; matched `lCp`)

### markdown

Role-weighted hits: `{"documentation": 114, "source": 896, "test": 210}`.

- `.clangd:315` (source; matched `Markdown`)
- `.clangd:317` (source; matched `Markdown`)
- `.claude/skills/fix-crashes/SKILL.md:9` (source; matched `.md`)
- `.claude/skills/fix-crashes/SKILL.md:20` (source; matched `.md`)
- `.gitattributes:3` (source; matched `.md`)
- Additional evidence retained in JSON: 35 records.

### mobi_azw

Role-weighted hits: `{"documentation": 24, "source": 249, "test": 166}`.

- `cmd/deps-build-defs.ts:707` (source; matched `mobi`)
- `cmd/gen-settings.ts:245` (source; matched `mobi`)
- `cmd/gen-settings.ts:1524` (source; matched `MOBI`)
- `cmd/helper/ng-mac-bundle.ts:26` (source; matched `mobi`)
- `cmd/helper/ng-mac-bundle.ts:26` (source; matched `azw`)
- Additional evidence retained in JSON: 35 records.

### odt_rtf

Role-weighted hits: `{"source": 22}`.

- `cmd/amalgam.ts:648` (source; matched `odt`)
- `ext/a-extract/extract.c:7288` (source; matched `.odt`)
- `ext/a-extract/extract.c:14532` (source; matched `odt`)
- `ext/mupdf/CHANGES:567` (source; matched `ODT`)
- `ext/mupdf/Makelists:1052` (source; matched `odt`)
- Additional evidence retained in JSON: 12 records.

### opf

Role-weighted hits: `{"source": 5, "test": 34}`.

- `src/shared/EbookDoc.cpp:1355` (source; matched `.opf`)
- `src/shared/EbookDoc.cpp:1357` (source; matched `.opf`)
- `src/shared/LitDoc.cpp:1356` (source; matched `.opf`)
- `src/shared/LitDoc.cpp:1384` (source; matched `.opf`)
- `tests/ad-hoc-issue-4600.ts:91` (test; matched `.opf`)
- Additional evidence retained in JSON: 33 records.

### pdf

Role-weighted hits: `{"configuration": 14, "documentation": 620, "source": 4383, "test": 2987}`.

- `.claude/skills/update-zlib/SKILL.md:48` (source; matched `.pdf`)
- `.claude/skills/verify-mac/SKILL.md:16` (source; matched `PDF`)
- `.claude/skills/verify-mac/SKILL.md:16` (source; matched `.pdf`)
- `.claude/skills/verify/SKILL.md:37` (source; matched `PDF`)
- `.claude/skills/verify/SKILL.md:37` (source; matched `.pdf`)
- Additional evidence retained in JSON: 35 records.

### txt

Role-weighted hits: `{"configuration": 18, "documentation": 89, "source": 669, "test": 324}`.

- `.clangd:2` (source; matched `.txt`)
- `.claude/skills/fix-crash/SKILL.md:20` (source; matched `.txt`)
- `.claude/skills/fix-crash/SKILL.md:20` (source; matched `.txt`)
- `.claude/skills/fix-crashes/SKILL.md:9` (source; matched `.txt`)
- `.claude/skills/fix-crashes/SKILL.md:9` (source; matched `.txt`)
- Additional evidence retained in JSON: 35 records.

### webpub_lpf

Role-weighted hits: `{"documentation": 3, "source": 1481, "test": 14}`.

- `ext/a-freetype/include/freetype/ftglyph.h:682` (source; matched `lpf`)
- `ext/a-libwebp/webp/demux.h:343` (source; matched `lpf`)
- `ext/a-unrar/unrar.cpp:15787` (source; matched `lpF`)
- `ext/darkmodelib/dmlib_demo/demo.cpp:202` (source; matched `lpf`)
- `ext/darkmodelib/include/Darkmodelib.h:735` (source; matched `lpf`)
- Additional evidence retained in JSON: 35 records.
