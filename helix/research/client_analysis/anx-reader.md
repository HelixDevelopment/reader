# anx-reader — automated static evidence

- Inventory URL: https://github.com/anxcye/anx-reader
- Origin: https://github.com/anxcye/anx-reader
- Revision: `0c7fc1ad4551f7dbbeba6a6468f69d830957fa2a`
- Tree: `c5cd5938f6f9ef32009b1a093dc49de679c9531c`
- Analysis status: **automated_static_index_complete**; manual review: **pending**.
- Keyword hits are leads for code review, not proof of working support.

## Tree and languages

Tracked files: 1304. Text files scanned: 1160 (7654278 bytes). Files skipped by size, binary classification, or sensitive-name policy remain indexed in the compressed file ledger.

- Dart: 725 files
- JavaScript: 94 files
- C/C++ header: 17 files
- C++: 15 files
- XML: 12 files
- CSS: 12 files
- Shell: 11 files
- Swift: 8 files
- TypeScript: 4 files
- HTML: 3 files
- Kotlin: 2 files
- PowerShell: 2 files

## License review

Status: heuristic_requires_legal_review. Candidate notices: Apache-2.0, BSD-2-Clause, BSD-3-Clause, MIT.
- `LICENSE`
- `third_party/googleai_dart/LICENSE`
- `third_party/langchain_google/LICENSE`
- `third_party/webdav_client/LICENSE`
- `third_party/webview_cef/LICENSE`

## Feature evidence

### ai_llm

Role-weighted hits: `{"configuration": 12, "documentation": 104, "source": 2044, "test": 12}`.

- `.github/ISSUE_TEMPLATE/bug-report.yaml:54` (configuration; matched `rag`)
- `.github/workflows/label-trigger.yml:84` (configuration; matched `rag`)
- `.github/workflows/label-trigger.yml:84` (configuration; matched `rag`)
- `Fastfile:60` (source; matched `rag`)
- `README.md:34` (documentation; matched `rag`)
- Additional evidence retained in JSON: 35 records.

### annotations

Role-weighted hits: `{"configuration": 4, "documentation": 31, "source": 3098, "test": 11}`.

- `.github/ISSUE_TEMPLATE/bug-report.yaml:54` (configuration; matched `highlight`)
- `README.md:157` (documentation; matched `highlight`)
- `README.md:162` (documentation; matched `highlight`)
- `README.md:162` (documentation; matched `highlight`)
- `analysis_options.yaml:33` (source; matched `annotation`)
- Additional evidence retained in JSON: 35 records.

### backup_restore

Role-weighted hits: `{"configuration": 1, "documentation": 11, "source": 760, "test": 5}`.

- `.github/workflows/build-linux.yaml:101` (configuration; matched `restore`)
- `android/app/src/main/AndroidManifest.xml:31` (source; matched `Backup`)
- `assets/CHANGELOG.md:6` (documentation; matched `Restore`)
- `assets/CHANGELOG.md:7` (documentation; matched `restore`)
- `assets/CHANGELOG.md:29` (documentation; matched `Restore`)
- Additional evidence retained in JSON: 35 records.

### collections_smart

Role-weighted hits: `{"configuration": 1, "documentation": 29, "source": 856}`.

- `README.md:35` (documentation; matched `shelves`)
- `README.md:94` (documentation; matched `collection`)
- `assets/CHANGELOG.md:22` (documentation; matched `bookshelf`)
- `assets/CHANGELOG.md:33` (documentation; matched `bookshelf`)
- `assets/CHANGELOG.md:64` (documentation; matched `bookshelf`)
- Additional evidence retained in JSON: 35 records.

### comics_manga

Role-weighted hits: `{"documentation": 2, "source": 27}`.

- `assets/CHANGELOG.md:133` (documentation; matched `CBZ`)
- `assets/CHANGELOG.md:133` (documentation; matched `comic`)
- `assets/foliate-js/dist/bundle.js:2` (source; matched `comic`)
- `assets/foliate-js/dist/bundle.js:2` (source; matched `cbz`)
- `assets/foliate-js/dist/bundle.js:2` (source; matched `Comic`)
- Additional evidence retained in JSON: 14 records.

### compression_cache

Role-weighted hits: `{"configuration": 10, "documentation": 32, "source": 1218}`.

- `.github/workflows/build-android.yaml:47` (configuration; matched `cache`)
- `.github/workflows/build-linux.yaml:82` (configuration; matched `cache`)
- `.github/workflows/build-linux.yaml:96` (configuration; matched `Cache`)
- `.github/workflows/build-linux.yaml:97` (configuration; matched `cache`)
- `.github/workflows/build-playstore.yaml:59` (configuration; matched `cache`)
- Additional evidence retained in JSON: 35 records.

### conversion

Role-weighted hits: `{"configuration": 1, "documentation": 12, "source": 109}`.

- `.github/workflows/label-trigger.yml:84` (configuration; matched `epub、pdf、txt等）\n- 或者能够重现问题的最小示例文件\n- 如果涉及特定格式，请说明文件的具体格式和来源\n\n您可以：\n1. 直接拖拽文件到此评论区上传\n2. 提供文件的下载链接\`)
- `README.md:13` (documentation; matched `epub%20%7C%20fb2%20%7C%20mobi%20%7C%20txt%20%7C%20azw3%20%7C%20pdf`)
- `README.md:33` (documentation; matched `EPUB/MOBI/AZW3/FB2/TXT/PDF`)
- `README.md:39` (documentation; matched `conversion`)
- `README_RU.md:13` (source; matched `epub%20%7C%20fb2%20%7C%20mobi%20%7C%20txt%20%7C%20azw3%20%7C%20pdf`)
- Additional evidence retained in JSON: 35 records.

### daisy_accessibility

Role-weighted hits: `{"documentation": 4, "source": 55}`.

- `assets/CHANGELOG.md:147` (documentation; matched `accessibility`)
- `assets/CHANGELOG.md:147` (documentation; matched `screen reader`)
- `assets/CHANGELOG.md:148` (documentation; matched `accessibility`)
- `assets/foliate-js/dist/bundle.js:2` (source; matched `daisy`)
- `assets/foliate-js/dist/pdf-legacy.js:2` (source; matched `ACCESSIBILITY`)
- Additional evidence retained in JSON: 8 records.

### database

Role-weighted hits: `{"configuration": 1, "documentation": 7, "source": 96}`.

- `assets/foliate-js/src/vendor/prism/README.md:48` (documentation; matched `MongoDB`)
- `assets/foliate-js/src/vendor/prism/components/prism-bash.min.js:1` (source; matched `mysql`)
- `assets/foliate-js/src/vendor/prism/components/prism-mongodb.min.js:1` (source; matched `mongodb`)
- `assets/foliate-js/src/vendor/prism/components/prism-mongodb.min.js:1` (source; matched `mongodb`)
- `assets/foliate-js/src/vendor/prism/components/prism-mongodb.min.js:1` (source; matched `mongodb`)
- Additional evidence retained in JSON: 35 records.

### deduplication

Role-weighted hits: `{"documentation": 23, "source": 466, "test": 4}`.

- `assets/CHANGELOG.md:11` (documentation; matched `fingerprint`)
- `assets/foliate-js/dist/bundle.js:2` (source; matched `duplicate`)
- `assets/foliate-js/dist/bundle.js:2` (source; matched `Duplicate`)
- `assets/foliate-js/dist/pdf-legacy.js:2` (source; matched `fingerprint`)
- `assets/foliate-js/dist/pdf-legacy.js:2` (source; matched `fingerprint`)
- Additional evidence retained in JSON: 35 records.

### deployment

Role-weighted hits: `{"configuration": 1, "documentation": 1, "source": 11}`.

- `.github/workflows/release.yaml:105` (configuration; matched `GitHub Actions`)
- `assets/foliate-js/dist/bundle.js:2` (source; matched `SystemD`)
- `assets/foliate-js/src/vendor/prism/README.md:51` (documentation; matched `Docker`)
- `assets/foliate-js/src/vendor/prism/components/prism-bash.min.js:1` (source; matched `docker`)
- `assets/foliate-js/src/vendor/prism/components/prism-bash.min.js:1` (source; matched `docker`)
- Additional evidence retained in JSON: 6 records.

### dictionary_language

Role-weighted hits: `{"documentation": 40, "source": 586}`.

- `CONTRIBUTING.md:48` (documentation; matched `Translation`)
- `CONTRIBUTING.md:49` (documentation; matched `translation`)
- `CONTRIBUTING.md:54` (documentation; matched `translation`)
- `README.md:35` (documentation; matched `dictionary`)
- `README.md:35` (documentation; matched `translation`)
- Additional evidence retained in JSON: 35 records.

### http_protocols

Role-weighted hits: `{"documentation": 13, "source": 468}`.

- `assets/CHANGELOG.md:37` (documentation; matched `Quic`)
- `assets/CHANGELOG.md:94` (documentation; matched `quic`)
- `assets/CHANGELOG.md:105` (documentation; matched `quic`)
- `assets/foliate-js/dist/pdf-legacy.js:2` (source; matched `Accept-Ranges`)
- `assets/foliate-js/dist/pdf-legacy.js:2` (source; matched `Content-Range`)
- Additional evidence retained in JSON: 35 records.

### kobo_sync

Role-weighted hits: `{"source": 1}`.

- `assets/foliate-js/dist/bundle.js:2` (source; matched `kobobooks.display-options.xml");case 19:nt=t.sent;case 20:return(P=Q(nt))&&("true"===P.fixedLayout&&`)

### local_library

Role-weighted hits: `{"documentation": 2, "source": 21}`.

- `assets/CHANGELOG.md:10` (documentation; matched `local library`)
- `assets/foliate-js/dist/bundle.js:2` (source; matched `FileSystem`)
- `assets/foliate-js/dist/bundle.js:2` (source; matched `FileSystem`)
- `assets/foliate-js/dist/bundle.js:2` (source; matched `FileSystem`)
- `assets/foliate-js/src/fixed-layout.js:382` (source; matched `in place`)
- Additional evidence retained in JSON: 15 records.

### metadata_enrichment

Role-weighted hits: `{"documentation": 37, "source": 954}`.

- `.metadata:12` (source; matched `metadata`)
- `android/fastlane/Fastfile:61` (source; matched `metadata`)
- `android/fastlane/Fastfile:92` (source; matched `metadata`)
- `assets/CHANGELOG.md:22` (documentation; matched `metadata`)
- `assets/CHANGELOG.md:127` (documentation; matched `metadata`)
- Additional evidence retained in JSON: 35 records.

### metadata_review

Role-weighted hits: `{"source": 6}`.

- `assets/foliate-js/dist/bundle.js:2` (source; matched `metadata)||void 0===r?void 0:r.language),!e.splitTOCHref||!e.getTOCFragment){t.next=2;break}return s`)
- `assets/foliate-js/dist/pdf-legacy.js:2` (source; matched `LOCKED:128,TOGGLENOVIEW:256,LOCKEDCONTENTS:512},e.AnnotationFieldFlag={READONLY:1,REQUIRED:2,NOEXPOR`)
- `lib/service/ai/tools/book_content_search_tool.dart:17` (source; matched `metadata, snippet preview`)
- `third_party/googleai_dart/lib/googleai_dart.dart:154` (source; matched `metadata/review`)
- `third_party/googleai_dart/openapi.json:7632` (source; matched `metadata for the prompt and response\ntext.\n\nThis indicates which `SafetyCategory`(s) blocked a\nc`)
- Additional evidence retained in JSON: 1 records.

### multiuser_auth

Role-weighted hits: `{"configuration": 7, "documentation": 36, "source": 232}`.

- `.github/workflows/build-appstore.yaml:24` (configuration; matched `AUTHORIZATION`)
- `.github/workflows/build-appstore.yaml:71` (configuration; matched `AUTHORIZATION`)
- `.github/workflows/build-appstore.yaml:71` (configuration; matched `AUTHORIZATION`)
- `.github/workflows/build.yaml:76` (configuration; matched `AUTHORIZATION`)
- `.github/workflows/build.yaml:76` (configuration; matched `AUTHORIZATION`)
- Additional evidence retained in JSON: 35 records.

### network_sources

Role-weighted hits: `{"configuration": 1, "documentation": 4, "source": 48}`.

- `.github/workflows/label-trigger.yml:84` (configuration; matched `cloud storage`)
- `android/Gemfile.lock:21` (source; matched `s3`)
- `android/Gemfile.lock:75` (source; matched `s3`)
- `assets/foliate-js/dist/bundle.js:2` (source; matched `ftp`)
- `assets/foliate-js/dist/pdf-legacy.js:2` (source; matched `ftp`)
- Additional evidence retained in JSON: 32 records.

### notes_export

Role-weighted hits: `{"documentation": 15, "source": 130, "test": 1}`.

- `README.md:37` (documentation; matched `Notes Workspace | Multiple color/style presets<br>Sort by time or chapter with color filters<br>Expo`)
- `assets/CHANGELOG.md:11` (documentation; matched `notes): Export`)
- `assets/foliate-js/dist/bundle.js:2` (source; matched `exports&&"object"==typeof module?module.exports=e():"function"==typeof define&&define.amd?define([],`)
- `assets/foliate-js/dist/pdf-legacy.js:2` (source; matched `exports&&"object"==typeof module?module.exports=e():"function"==typeof define&&define.amd?define([],`)
- `assets/foliate-js/src/vendor/pdfjs/pdf.js:2750` (source; matched `exports.SerializableEmpty = exports.PrintAnnotationStorage = exports.AnnotationS`)
- Additional evidence retained in JSON: 35 records.

### opds

Role-weighted hits: `{"documentation": 6, "source": 22}`.

- `README.md:40` (documentation; matched `OPDS`)
- `README.md:40` (documentation; matched `OPDS`)
- `README_RU.md:39` (source; matched `OPDS`)
- `README_RU.md:39` (source; matched `OPDS`)
- `README_tr.md:40` (source; matched `OPDS`)
- Additional evidence retained in JSON: 9 records.

### opensearch

Role-weighted hits: `{"source": 1}`.

- `assets/foliate-js/src/opds.js:241` (source; matched `OpenSearch`)

### pdf_fixed_layout

Role-weighted hits: `{"configuration": 1, "documentation": 4, "source": 215}`.

- `assets/CHANGELOG.md:7` (documentation; matched `fixed-layout`)
- `assets/foliate-js/debug.html:232` (source; matched `pdfjs`)
- `assets/foliate-js/debug.html:232` (source; matched `pdf.js`)
- `assets/foliate-js/dist/bundle.js:2` (source; matched `pdfjs`)
- `assets/foliate-js/dist/bundle.js:2` (source; matched `pdfjs`)
- Additional evidence retained in JSON: 27 records.

### permissions

Role-weighted hits: `{"configuration": 9, "documentation": 52, "source": 464}`.

- `.github/workflows/build.yaml:10` (configuration; matched `permission`)
- `.github/workflows/label-trigger.yml:10` (configuration; matched `permission`)
- `.github/workflows/pr-check.yml:12` (configuration; matched `permission`)
- `.github/workflows/release.yaml:29` (configuration; matched `permission`)
- `.github/workflows/remove-question-label.yml:17` (configuration; matched `permission`)
- Additional evidence retained in JSON: 35 records.

### progress_sync

Role-weighted hits: `{"documentation": 2, "source": 19}`.

- `README.md:34` (documentation; matched `Sync | Android/iOS/macOS/Windows coverage<br>Sync books, notes, and reading progress`)
- `assets/CHANGELOG.md:131` (documentation; matched `sync): Add WebDAV sync support for reading progress`)
- `assets/foliate-js/dist/bundle.js:2` (source; matched `syncIterator");t.exports=function(t,e){var n=arguments.length<2?u(t,c):e;return n?o(r(n,t)):new i(s(`)
- `assets/foliate-js/dist/pdf-legacy.js:2` (source; matched `sync function _fetchDocument(t,e){if(t.destroyed)throw new Error("Worker was destroyed");const s=awa`)
- `assets/foliate-js/src/vendor/zip.js:1` (source; matched `sync_point=function(){return d==E?1:0}}const R=13,B=[0,0,255,255];function I(){const r=this;function`)
- Additional evidence retained in JSON: 14 records.

### reader_typography

Role-weighted hits: `{"configuration": 1, "documentation": 33, "source": 1548}`.

- `README.md:21` (documentation; matched `theme`)
- `README.md:36` (documentation; matched `font size`)
- `README.md:36` (documentation; matched `theme`)
- `README_RU.md:21` (source; matched `theme`)
- `README_tr.md:21` (source; matched `theme`)
- Additional evidence retained in JSON: 35 records.

### readium

Role-weighted hits: `{"source": 1}`.

- `assets/foliate-js/package-lock.json:2415` (source; matched `Rwpm`)

### search

Role-weighted hits: `{"documentation": 1, "source": 8}`.

- `assets/CHANGELOG.md:137` (documentation; matched `full-text search`)
- `fastlane/metadata/android/en-US/full_description.txt:39` (source; matched `Full-text search`)
- `ios/Podfile.lock:106` (source; matched `fts5`)
- `ios/Podfile.lock:117` (source; matched `fts5`)
- `macos/Podfile.lock:49` (source; matched `fts5`)
- Additional evidence retained in JSON: 4 records.

### security

Role-weighted hits: `{"configuration": 107, "documentation": 514, "source": 1712, "test": 15}`.

- `.github/ISSUE_TEMPLATE/bug-report.yaml:12` (configuration; matched `https`)
- `.github/ISSUE_TEMPLATE/bug-report.yaml:12` (configuration; matched `https`)
- `.github/workflows/build-android.yaml:12` (configuration; matched `secret`)
- `.github/workflows/build-android.yaml:52` (configuration; matched `secret`)
- `.github/workflows/build-android.yaml:53` (configuration; matched `secret`)
- Additional evidence retained in JSON: 35 records.

### testing

Role-weighted hits: `{"documentation": 7, "test": 2}`.

- `CONTRIBUTING.md:103` (documentation; matched `Integration Test`)
- `ohos/entry/src/ohosTest/ets/test/Ability.test.ets:12` (test; matched `unit test`)
- `ohos/entry/src/ohosTest/ets/test/Ability.test.ets:17` (test; matched `unit test`)
- `third_party/googleai_dart/README.md:310` (documentation; matched `integration test`)
- `third_party/googleai_dart/docs/spec.md:462` (documentation; matched `unit test`)
- Additional evidence retained in JSON: 2 records.

### tts_audio

Role-weighted hits: `{"documentation": 48, "source": 365}`.

- `README.md:39` (documentation; matched `TTS`)
- `README_RU.md:38` (source; matched `TTS`)
- `README_tr.md:39` (source; matched `TTS`)
- `assets/CHANGELOG.md:92` (documentation; matched `tts`)
- `assets/CHANGELOG.md:92` (documentation; matched `TTS`)
- Additional evidence retained in JSON: 35 records.

### webdav

Role-weighted hits: `{"configuration": 6, "documentation": 56, "source": 876}`.

- `README.md:34` (documentation; matched `WebDAV`)
- `README.md:99` (documentation; matched `WebDAV`)
- `README_RU.md:33` (source; matched `WebDAV`)
- `README_tr.md:34` (source; matched `WebDAV`)
- `README_tr.md:94` (source; matched `WebDAV`)
- Additional evidence retained in JSON: 35 records.


## Format evidence

### cbr

Role-weighted hits: `{"source": 3}`.

- `assets/foliate-js/src/vendor/pdfjs/pdf.js:8627` (source; matched `cbr`)
- `assets/foliate-js/src/vendor/pdfjs/pdf.js:8658` (source; matched `cbr`)

### cbz

Role-weighted hits: `{"documentation": 1, "source": 2}`.

- `assets/CHANGELOG.md:133` (documentation; matched `CBZ`)
- `assets/foliate-js/dist/bundle.js:2` (source; matched `.cbz`)
- `assets/foliate-js/src/book.js:791` (source; matched `.cbz`)

### daisy

Role-weighted hits: `{"source": 3}`.

- `assets/foliate-js/dist/bundle.js:2` (source; matched `.daisy`)
- `assets/foliate-js/src/epub.js:11` (source; matched `.daisy`)
- `lib/service/convert_to_epub/create_epub.dart:89` (source; matched `.daisy`)

### djvu

Role-weighted hits: `{"documentation": 3}`.

- `docs/issue-triage.md:269` (documentation; matched `DJVU`)
- `docs/issue-triage.md:269` (documentation; matched `DJVU`)

### doc_docx

Role-weighted hits: `{"source": 49}`.

- `assets/foliate-js/dist/bundle.js:2` (source; matched `.doc`)
- `assets/foliate-js/dist/bundle.js:2` (source; matched `.doc`)
- `assets/foliate-js/src/book.js:1318` (source; matched `.doc`)
- `assets/foliate-js/src/book.js:1319` (source; matched `.doc`)
- `assets/foliate-js/src/fb2.js:84` (source; matched `.doc`)
- Additional evidence retained in JSON: 4 records.

### epub

Role-weighted hits: `{"configuration": 2, "documentation": 18, "source": 97, "test": 3}`.

- `.github/workflows/label-trigger.yml:84` (configuration; matched `epub`)
- `.github/workflows/label-trigger.yml:84` (configuration; matched `epub`)
- `README.md:13` (documentation; matched `epub`)
- `README.md:33` (documentation; matched `EPUB`)
- `README_RU.md:13` (source; matched `epub`)
- Additional evidence retained in JSON: 35 records.

### fb2

Role-weighted hits: `{"documentation": 1, "source": 23}`.

- `README.md:33` (documentation; matched `FB2`)
- `README_RU.md:32` (source; matched `FB2`)
- `README_tr.md:33` (source; matched `FB2`)
- `README_zh.md:24` (source; matched `FB2`)
- `assets/foliate-js/dist/bundle.js:2` (source; matched `fictionbook`)
- Additional evidence retained in JSON: 9 records.

### html_xhtml

Role-weighted hits: `{"configuration": 2, "documentation": 23, "source": 95}`.

- `.gitignore:65` (source; matched `.html`)
- `CONTRIBUTING.md:71` (documentation; matched `.html`)
- `CONTRIBUTING.md:74` (documentation; matched `.html`)
- `README.md:144` (documentation; matched `.html`)
- `README.md:145` (documentation; matched `.html`)
- Additional evidence retained in JSON: 35 records.

### markdown

Role-weighted hits: `{"configuration": 8, "documentation": 38, "source": 102}`.

- `.github/ISSUE_TEMPLATE/bug-report.yaml:12` (configuration; matched `.md`)
- `.github/ISSUE_TEMPLATE/bug-report.yaml:12` (configuration; matched `.md`)
- `.github/workflows/release.yaml:106` (configuration; matched `.md`)
- `.github/workflows/release.yaml:124` (configuration; matched `markdown`)
- `.github/workflows/setup.yaml:106` (configuration; matched `.md`)
- Additional evidence retained in JSON: 35 records.

### mobi_azw

Role-weighted hits: `{"documentation": 2, "source": 145}`.

- `README.md:33` (documentation; matched `MOBI`)
- `README.md:33` (documentation; matched `AZW3`)
- `README_RU.md:32` (source; matched `MOBI`)
- `README_RU.md:32` (source; matched `AZW3`)
- `README_tr.md:33` (source; matched `MOBI`)
- Additional evidence retained in JSON: 17 records.

### odt_rtf

Role-weighted hits: `{"source": 1}`.

- `third_party/googleai_dart/lib/src/models/tools/tool.dart:34` (source; matched `RTF`)

### opf

Role-weighted hits: `{"source": 24}`.

- `assets/foliate-js/dist/bundle.js:2` (source; matched `.opf`)
- `assets/foliate-js/dist/bundle.js:2` (source; matched `.opf`)
- `assets/foliate-js/src/epub.js:111` (source; matched `.OPF`)
- `assets/foliate-js/src/epub.js:114` (source; matched `.OPF`)
- `assets/foliate-js/src/epubcfi.js:328` (source; matched `package document`)
- Additional evidence retained in JSON: 3 records.

### pdf

Role-weighted hits: `{"configuration": 2, "documentation": 25, "source": 93}`.

- `.github/workflows/label-trigger.yml:84` (configuration; matched `pdf`)
- `.github/workflows/label-trigger.yml:84` (configuration; matched `pdf`)
- `README.md:33` (documentation; matched `PDF`)
- `README_RU.md:32` (source; matched `PDF`)
- `README_tr.md:33` (source; matched `PDF`)
- Additional evidence retained in JSON: 32 records.

### txt

Role-weighted hits: `{"documentation": 1, "source": 35, "test": 1}`.

- `assets/foliate-js/dist/bundle.js:1` (source; matched `.txt`)
- `assets/foliate-js/dist/bundle.js:2` (source; matched `.txt`)
- `assets/foliate-js/dist/pdf-legacy.js:1` (source; matched `.txt`)
- `assets/foliate-js/src/epub.js:1152` (source; matched `.txt`)
- `ios/Runner.xcodeproj/project.pbxproj:426` (source; matched `.txt`)
- Additional evidence retained in JSON: 26 records.

### webpub_lpf

Role-weighted hits: `{"documentation": 1, "source": 3}`.

- `CONTRIBUTING.md:5` (documentation; matched `lpf`)
- `lib/l10n/app_en.arb:9` (source; matched `lpf`)
- `third_party/googleai_dart/MIGRATION.md:207` (source; matched `lpf`)
- `windows/runner/win32_window.cpp:102` (source; matched `lpf`)
