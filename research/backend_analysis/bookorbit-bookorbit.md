# bookorbit-bookorbit — automated static evidence

- Inventory URL: https://github.com/bookorbit/bookorbit
- Origin: https://github.com/bookorbit/bookorbit
- Revision: `f2a7a111881899e7c6cb233a0ec2d8a9c25235c6`
- Tree: `f1074f1ca0257d226f40762c614264f053879798`
- Analysis status: **automated_static_index_complete**; manual review: **pending**.
- Keyword hits are leads for code review, not proof of working support.

## Tree and languages

Tracked files: 5902. Text files scanned: 5870 (123305430 bytes). Files skipped by size, binary classification, or sensitive-name policy remain indexed in the compressed file ledger.

- TypeScript: 4257 files
- Vue: 901 files
- Lua: 212 files
- SQL: 135 files
- JavaScript: 20 files
- Shell: 11 files
- CSS: 8 files
- HTML: 1 files
- Python: 1 files

## License review

Status: heuristic_requires_legal_review. Candidate notices: AGPL-3.0.
- `LICENSE`
- `NOTICE`

## Feature evidence

### ai_llm

Role-weighted hits: `{"configuration": 27, "documentation": 86, "source": 5124, "test": 2207}`.

- `.dockerignore:13` (source; matched `rag`)
- `.github/ISSUE_TEMPLATE/bug_report.yml:56` (configuration; matched `rag`)
- `.github/ISSUE_TEMPLATE/question.yml:20` (configuration; matched `rag`)
- `.github/ISSUE_TEMPLATE/question.yml:20` (configuration; matched `rag`)
- `.github/ISSUE_TEMPLATE/question.yml:21` (configuration; matched `rag`)
- Additional evidence retained in JSON: 35 records.

### annotations

Role-weighted hits: `{"configuration": 4, "documentation": 92, "source": 32937, "test": 8307}`.

- `.github/workflows/ci.yml:526` (configuration; matched `annotation`)
- `.github/workflows/e2e-runner.yml:88` (configuration; matched `annotation`)
- `README.md:43` (documentation; matched `highlight`)
- `README.md:91` (documentation; matched `annotation`)
- `README.md:91` (documentation; matched `highlight`)
- Additional evidence retained in JSON: 35 records.

### backup_restore

Role-weighted hits: `{"configuration": 6, "documentation": 22, "source": 11020, "test": 1952}`.

- `.github/actions/setup-node-pnpm/action.yml:58` (configuration; matched `restore`)
- `.github/workflows/repo-charts.yml:44` (configuration; matched `Restore`)
- `.github/workflows/repo-charts.yml:48` (configuration; matched `restore`)
- `.github/workflows/repo-charts.yml:53` (configuration; matched `Restore`)
- `client/public/assets/foliate/iframe-loader.js:35` (source; matched `Restore`)
- Additional evidence retained in JSON: 35 records.

### collections_smart

Role-weighted hits: `{"configuration": 12, "documentation": 51, "source": 30588, "test": 8229}`.

- `.github/workflows/repo-charts.yml:46` (configuration; matched `SERIES`)
- `.github/workflows/repo-charts.yml:48` (configuration; matched `series`)
- `.github/workflows/repo-charts.yml:52` (configuration; matched `SERIES`)
- `AGENTS.md:19` (source; matched `collection`)
- `CLAUDE.md:19` (source; matched `collection`)
- Additional evidence retained in JSON: 35 records.

### comics_manga

Role-weighted hits: `{"configuration": 1, "documentation": 40, "source": 5378, "test": 2868}`.

- `README.md:8` (documentation; matched `comic`)
- `README.md:59` (documentation; matched `comic`)
- `README.md:89` (documentation; matched `comic`)
- `client/README.md:30` (documentation; matched `cbz`)
- `client/public/assets/foliate/comic-book.js:1` (source; matched `Comic`)
- Additional evidence retained in JSON: 35 records.

### compression_cache

Role-weighted hits: `{"configuration": 32, "documentation": 103, "source": 6526, "test": 3017}`.

- `.github/actions/setup-node-pnpm/action.yml:17` (configuration; matched `cache`)
- `.github/actions/setup-node-pnpm/action.yml:21` (configuration; matched `cache`)
- `.github/actions/setup-node-pnpm/action.yml:22` (configuration; matched `cache`)
- `.github/workflows/ci.yml:685` (configuration; matched `cache`)
- `.github/workflows/ci.yml:686` (configuration; matched `cache`)
- Additional evidence retained in JSON: 35 records.

### conversion

Role-weighted hits: `{"configuration": 3, "documentation": 13, "source": 3535, "test": 1309}`.

- `.gitattributes:1` (source; matched `kepub`)
- `Dockerfile:76` (configuration; matched `kepub`)
- `Dockerfile:76` (configuration; matched `kepub`)
- `Dockerfile:79` (configuration; matched `kepub`)
- `README.md:89` (documentation; matched `EPUB, KEPUB, MOBI, AZW3, AZW, FB2), PDF`)
- Additional evidence retained in JSON: 35 records.

### daisy_accessibility

Role-weighted hits: `{"configuration": 1, "documentation": 2, "source": 3591, "test": 919}`.

- `.github/pull_request_template.md:33` (configuration; matched `accessibility`)
- `client/public/assets/foliate/epub.js:11` (source; matched `daisy`)
- `client/src/assets/main.css:486` (source; matched `aria-`)
- `client/src/components/AccentPicker.vue:42` (source; matched `aria-`)
- `client/src/components/AccentPicker.vue:43` (source; matched `aria-`)
- Additional evidence retained in JSON: 35 records.

### database

Role-weighted hits: `{"configuration": 6, "documentation": 46, "source": 910, "test": 477}`.

- `ADDITIONAL_TERMS.md:23` (source; matched `redis`)
- `AGENTS.md:9` (source; matched `PostgreSQL`)
- `AGENTS.md:26` (source; matched `Drizzle`)
- `AGENTS.md:27` (source; matched `PostgreSQL`)
- `CLAUDE.md:9` (source; matched `PostgreSQL`)
- Additional evidence retained in JSON: 35 records.

### deduplication

Role-weighted hits: `{"configuration": 1, "documentation": 11, "source": 16918, "test": 1535}`.

- `AGENTS.md:98` (source; matched `duplicate`)
- `CLAUDE.md:98` (source; matched `duplicate`)
- `Dockerfile:1` (configuration; matched `sha256`)
- `client/scripts/locale-configuration.mjs:57` (source; matched `duplicate`)
- `client/scripts/sync-crowdin-translations.mjs:279` (source; matched `Duplicate`)
- Additional evidence retained in JSON: 35 records.

### deployment

Role-weighted hits: `{"configuration": 63, "documentation": 83, "source": 126, "test": 79}`.

- `.dockerignore:8` (source; matched `docker`)
- `.github/ISSUE_TEMPLATE/bug_report.yml:39` (configuration; matched `Docker`)
- `.github/ISSUE_TEMPLATE/bug_report.yml:49` (configuration; matched `docker`)
- `.github/ISSUE_TEMPLATE/question.yml:44` (configuration; matched `Docker`)
- `.github/dependabot.yml:34` (configuration; matched `Docker`)
- Additional evidence retained in JSON: 35 records.

### dictionary_language

Role-weighted hits: `{"configuration": 29, "documentation": 73, "source": 419, "test": 172}`.

- `.github/workflows/ci.yml:282` (configuration; matched `translation`)
- `.github/workflows/crowdin-branch-cleanup.yml:12` (configuration; matched `translation`)
- `.github/workflows/crowdin-branch-cleanup.yml:27` (configuration; matched `translation`)
- `.github/workflows/crowdin-translations.yml:1` (configuration; matched `Translation`)
- `.github/workflows/crowdin-translations.yml:6` (configuration; matched `translation`)
- Additional evidence retained in JSON: 35 records.

### http_protocols

Role-weighted hits: `{"documentation": 9, "source": 526, "test": 269}`.

- `README.md:31` (documentation; matched `quic`)
- `README.md:31` (documentation; matched `Quic`)
- `README.md:110` (documentation; matched `Quic`)
- `client/public/assets/foliate/streaming-loader.js:35` (source; matched `quic`)
- `client/src/features/achievements/utils/achievementToast.test.ts:86` (test; matched `quic`)
- Additional evidence retained in JSON: 35 records.

### kobo_sync

Role-weighted hits: `{"documentation": 11, "source": 5450, "test": 1055}`.

- `AGENTS.md:18` (source; matched `Kobo sync`)
- `CLAUDE.md:18` (source; matched `Kobo sync`)
- `README.md:43` (documentation; matched `Kobo, or KOReader. Your reading progress, highlights, and status stay sync`)
- `README.md:170` (documentation; matched `Kobo sync`)
- `README.md:221` (documentation; matched `Kobo sync`)
- Additional evidence retained in JSON: 35 records.

### koreader_sync

Role-weighted hits: `{"documentation": 2, "source": 78, "test": 53}`.

- `client/src/features/koreader/composables/__tests__/useKoreaderSync.spec.ts:359` (test; matched `KOReader sync`)
- `client/src/features/koreader/composables/__tests__/useKoreaderSync.spec.ts:364` (test; matched `KOReader sync`)
- `client/src/features/settings/KoreaderSettings.vue:77` (source; matched `koreader-sync`)
- `client/src/features/settings/KoreaderSettings.vue:827` (source; matched `koreader-sync`)
- `client/src/features/settings/KoreaderSettings.vue:1039` (source; matched `koreader-sync`)
- Additional evidence retained in JSON: 35 records.

### lcp_drm

Role-weighted hits: `{"documentation": 3, "source": 28, "test": 8}`.

- `LICENSE:6` (documentation; matched `license document`)
- `LICENSE:380` (documentation; matched `license document`)
- `LICENSE:383` (documentation; matched `license document`)
- `client/src/locales/de.json:2015` (source; matched `DRM`)
- `client/src/locales/el.json:2015` (source; matched `DRM`)
- Additional evidence retained in JSON: 30 records.

### local_library

Role-weighted hits: `{"configuration": 4, "documentation": 11, "source": 200, "test": 191}`.

- `.github/ISSUE_TEMPLATE/question.yml:21` (configuration; matched `local storage`)
- `.github/pull_request_template.md:33` (configuration; matched `filesystem`)
- `.github/pull_request_template.md:41` (configuration; matched `filesystem`)
- `AGENTS.md:18` (source; matched `filesystem`)
- `CLAUDE.md:18` (source; matched `filesystem`)
- Additional evidence retained in JSON: 35 records.

### metadata_enrichment

Role-weighted hits: `{"configuration": 7, "documentation": 336, "source": 44416, "test": 15419}`.

- `.github/workflows/release.yml:388` (configuration; matched `metadata`)
- `.github/workflows/release.yml:389` (configuration; matched `metadata`)
- `.github/workflows/release.yml:404` (configuration; matched `metadata`)
- `.gitignore:32` (source; matched `metadata`)
- `AGENTS.md:18` (source; matched `metadata`)
- Additional evidence retained in JSON: 35 records.

### metadata_review

Role-weighted hits: `{"configuration": 6, "documentation": 5, "source": 1181, "test": 667}`.

- `.github/workflows/release.yml:171` (configuration; matched `dry-run`)
- `.github/workflows/release.yml:172` (configuration; matched `Dry Run`)
- `.github/workflows/release.yml:192` (configuration; matched `dry run`)
- `.github/workflows/stale-prs.yml:12` (configuration; matched `dry-run`)
- `.github/workflows/stale-prs.yml:14` (configuration; matched `dry-run`)
- Additional evidence retained in JSON: 35 records.

### multiuser_auth

Role-weighted hits: `{"configuration": 10, "documentation": 51, "source": 20008, "test": 1627}`.

- `.github/workflows/ci.yml:116` (configuration; matched `Authorization`)
- `.github/workflows/container-image.yml:78` (configuration; matched `Authorization`)
- `.github/workflows/container-image.yml:82` (configuration; matched `Authorization`)
- `.github/workflows/e2e-runner.yml:35` (configuration; matched `JWT`)
- `.github/workflows/e2e-runner.yml:36` (configuration; matched `JWT`)
- Additional evidence retained in JSON: 35 records.

### network_sources

Role-weighted hits: `{"documentation": 2, "source": 13, "test": 47}`.

- `README.md:186` (documentation; matched `SMB`)
- `README.md:186` (documentation; matched `NFS`)
- `client/src/features/author/lib/__tests__/author-facts.spec.ts:54` (test; matched `ftp`)
- `client/src/features/reader/media-overlay/lib/media-overlay-range.test.ts:14` (test; matched `s3`)
- `client/src/features/reader/media-overlay/lib/media-overlay-range.test.ts:17` (test; matched `s3`)
- Additional evidence retained in JSON: 35 records.

### notes_export

Role-weighted hits: `{"documentation": 1, "source": 89, "test": 12}`.

- `README.md:93` (documentation; matched `Annotations & Highlights**: Highlights from the web reader, KOReader, and Kobo merge into one search`)
- `client/scripts/locale-protected-terms.mjs:6` (source; matched `annotations.hub.export`)
- `client/scripts/sync-crowdin-translations.spec.mjs:612` (test; matched `annotations.hub.export`)
- `client/scripts/sync-crowdin-translations.spec.mjs:630` (test; matched `annotations.hub.export`)
- `client/scripts/sync-crowdin-translations.spec.mjs:657` (test; matched `annotations: { hub: { export`)
- Additional evidence retained in JSON: 35 records.

### opds

Role-weighted hits: `{"documentation": 18, "source": 727, "test": 319}`.

- `README.md:45` (documentation; matched `OPDS`)
- `README.md:107` (documentation; matched `OPDS`)
- `README.md:170` (documentation; matched `OPDS`)
- `client/src/features/settings/OpdsSettings.vue:40` (source; matched `opds`)
- `client/src/features/settings/OpdsSettings.vue:43` (source; matched `opds`)
- Additional evidence retained in JSON: 35 records.

### opensearch

Role-weighted hits: `{"source": 21, "test": 26}`.

- `client/src/components/view-header/ViewHeaderDesktopSearch.vue:26` (source; matched `openSearch`)
- `client/src/components/view-header/ViewHeaderDesktopSearch.vue:74` (source; matched `openSearch`)
- `client/src/features/book-dock/components/BookDockFileSheet.vue:287` (source; matched `openSearch`)
- `client/src/features/book-dock/components/BookDockFileSheet.vue:661` (source; matched `openSearch`)
- `client/src/features/book-dock/components/__tests__/BookDockFileSheet.spec.ts:122` (test; matched `openSearch`)
- Additional evidence retained in JSON: 23 records.

### pdf_fixed_layout

Role-weighted hits: `{"configuration": 4, "documentation": 38, "source": 503, "test": 360}`.

- `.github/workflows/e2e-runner.yml:75` (configuration; matched `DjVu`)
- `.github/workflows/e2e-runner.yml:76` (configuration; matched `djvu`)
- `.github/workflows/e2e-runner.yml:77` (configuration; matched `djvu`)
- `Dockerfile:61` (configuration; matched `djvu`)
- `client/public/assets/foliate/epub.js:1074` (source; matched `fixed-layout`)
- Additional evidence retained in JSON: 35 records.

### permissions

Role-weighted hits: `{"configuration": 32, "documentation": 96, "source": 7104, "test": 4171}`.

- `.github/pull_request_template.md:32` (configuration; matched `permission`)
- `.github/workflows/ci.yml:18` (configuration; matched `permission`)
- `.github/workflows/ci.yml:48` (configuration; matched `permission`)
- `.github/workflows/ci.yml:308` (configuration; matched `permission`)
- `.github/workflows/container-image.yml:22` (configuration; matched `permission`)
- Additional evidence retained in JSON: 35 records.

### progress_sync

Role-weighted hits: `{"documentation": 16, "source": 1431, "test": 403}`.

- `README.md:43` (documentation; matched `reading progress`)
- `README.md:59` (documentation; matched `synchronized playback progress`)
- `README.md:90` (documentation; matched `synchronize with your BookOrbit server from a native, offline-capable iPhone app. Send audiobooks to`)
- `client/src/features/book/components/BookCoverCard.test.ts:689` (test; matched `reading progress`)
- `client/src/features/book/components/BookCoverCard.test.ts:691` (test; matched `reading progress`)
- Additional evidence retained in JSON: 35 records.

### reader_typography

Role-weighted hits: `{"documentation": 5, "source": 1749, "test": 753}`.

- `AGENTS.md:90` (source; matched `Theme`)
- `CLAUDE.md:90` (source; matched `Theme`)
- `client/README.md:39` (documentation; matched `theme`)
- `client/README.md:55` (documentation; matched `Theme`)
- `client/index.html:13` (source; matched `theme`)
- Additional evidence retained in JSON: 35 records.

### readium

Role-weighted hits: `{"source": 11}`.

- `client/public/assets/foliate/epub.js:248` (source; matched `webpub`)
- `client/public/assets/foliate/epub.js:273` (source; matched `webpub`)
- `client/public/assets/foliate/epub.js:287` (source; matched `webpub`)
- `pnpm-lock.yaml:3443` (source; matched `readium`)
- `pnpm-lock.yaml:3447` (source; matched `readium`)
- Additional evidence retained in JSON: 1 records.

### search

Role-weighted hits: `{"source": 3, "test": 2}`.

- `client/src/features/book-requests/__tests__/RequestSearchPanel.spec.ts:532` (test; matched `Search index`)
- `client/src/features/book-requests/__tests__/RequestSearchPanel.spec.ts:673` (test; matched `Search index`)
- `client/src/locales/en.json:8995` (source; matched `Search index`)
- `client/src/locales/en.json:9000` (source; matched `Search index`)
- `server/src/modules/position-converter/chapter-text-index.ts:216` (source; matched `search index`)

### security

Role-weighted hits: `{"configuration": 57, "documentation": 152, "source": 4849, "test": 3387}`.

- `.github/ISSUE_TEMPLATE/config.yml:4` (configuration; matched `https`)
- `.github/ISSUE_TEMPLATE/discussion.yml:12` (configuration; matched `https`)
- `.github/ISSUE_TEMPLATE/discussion.yml:14` (configuration; matched `https`)
- `.github/ISSUE_TEMPLATE/discussion.yml:16` (configuration; matched `https`)
- `.github/ISSUE_TEMPLATE/question.yml:12` (configuration; matched `https`)
- Additional evidence retained in JSON: 35 records.

### testing

Role-weighted hits: `{"configuration": 26, "documentation": 48, "source": 157, "test": 1166}`.

- `.github/dependabot.yml:23` (configuration; matched `vitest`)
- `.github/dependabot.yml:23` (configuration; matched `vitest`)
- `.github/workflows/ci.yml:488` (configuration; matched `vitest`)
- `.github/workflows/ci.yml:490` (configuration; matched `vitest`)
- `.github/workflows/ci.yml:534` (configuration; matched `Unit Test`)
- Additional evidence retained in JSON: 35 records.

### tts_audio

Role-weighted hits: `{"configuration": 7, "documentation": 27, "source": 34866, "test": 6471}`.

- `README.md:8` (documentation; matched `audiobook`)
- `README.md:59` (documentation; matched `audiobook`)
- `README.md:59` (documentation; matched `text to speech`)
- `client/eslint.config.ts:31` (source; matched `podcast`)
- `client/eslint.config.ts:35` (source; matched `podcast`)
- Additional evidence retained in JSON: 35 records.

### watch_import

Role-weighted hits: `{"documentation": 5, "source": 382, "test": 85}`.

- `README.md:108` (documentation; matched `Book Dock`)
- `client/src/components/sidebar/__tests__/SidebarCustomizer.spec.ts:104` (test; matched `Book Dock`)
- `client/src/composables/__tests__/useSidebarNav.spec.ts:79` (test; matched `Book Dock`)
- `client/src/composables/__tests__/useSidebarNav.spec.ts:101` (test; matched `Book Dock`)
- `client/src/composables/__tests__/useSidebarNav.spec.ts:171` (test; matched `Book Dock`)
- Additional evidence retained in JSON: 35 records.


## Format evidence

### audiobook

Role-weighted hits: `{"source": 47, "test": 26}`.

- `client/src/features/book-requests/__tests__/requestPair.spec.ts:106` (test; matched `.audiobook`)
- `client/src/features/book-requests/__tests__/requestPair.spec.ts:110` (test; matched `.audiobook`)
- `client/src/features/book-requests/__tests__/useBookRequests.spec.ts:257` (test; matched `.audiobook`)
- `client/src/features/book-requests/__tests__/useCandidateGroups.spec.ts:290` (test; matched `.audiobook`)
- `client/src/features/book-requests/__tests__/useRequestSubmission.spec.ts:250` (test; matched `.audiobook`)
- Additional evidence retained in JSON: 35 records.

### cb7_cbt_cba

Role-weighted hits: `{"documentation": 4, "source": 294, "test": 98}`.

- `README.md:89` (documentation; matched `CB7`)
- `client/src/assets/theme/tokens.css:277` (source; matched `cb7`)
- `client/src/assets/theme/tokens.css:528` (source; matched `cb7`)
- `client/src/features/book-requests/__tests__/BookMissingFormatRequest.spec.ts:73` (test; matched `cb7`)
- `client/src/features/book-requests/__tests__/RequestSearchPanel.spec.ts:385` (test; matched `CB7`)
- Additional evidence retained in JSON: 35 records.

### cbr

Role-weighted hits: `{"documentation": 1, "source": 415, "test": 111}`.

- `README.md:89` (documentation; matched `CBR`)
- `client/src/assets/theme/tokens.css:276` (source; matched `cbr`)
- `client/src/assets/theme/tokens.css:527` (source; matched `cbr`)
- `client/src/features/book-requests/__tests__/BookMissingFormatRequest.spec.ts:73` (test; matched `cbr`)
- `client/src/features/book-requests/__tests__/ReleasePickerPanel.spec.ts:111` (test; matched `CBR`)
- Additional evidence retained in JSON: 35 records.

### cbz

Role-weighted hits: `{"documentation": 4, "source": 609, "test": 568}`.

- `README.md:89` (documentation; matched `CBZ`)
- `client/README.md:30` (documentation; matched `cbz`)
- `client/public/assets/foliate/view.js:72` (source; matched `.cbz`)
- `client/src/assets/theme/tokens.css:275` (source; matched `cbz`)
- `client/src/assets/theme/tokens.css:526` (source; matched `cbz`)
- Additional evidence retained in JSON: 35 records.

### daisy

Role-weighted hits: `{"source": 1, "test": 1}`.

- `client/public/assets/foliate/epub.js:11` (source; matched `.daisy`)
- `client/src/features/reader/epub/__tests__/opf-namespace.spec.ts:15` (test; matched `.daisy`)

### djvu

Role-weighted hits: `{"configuration": 1, "documentation": 26, "source": 215, "test": 188}`.

- `.github/workflows/e2e-runner.yml:76` (configuration; matched `djvu`)
- `client/src/assets/theme/tokens.css:272` (source; matched `djvu`)
- `client/src/assets/theme/tokens.css:523` (source; matched `djvu`)
- `client/src/features/book-dock/composables/__tests__/useBookDockUpload.spec.ts:118` (test; matched `DjVu`)
- `client/src/features/book-dock/composables/__tests__/useBookDockUpload.spec.ts:120` (test; matched `.DJVU`)
- Additional evidence retained in JSON: 35 records.

### doc_docx

Role-weighted hits: `{"source": 23, "test": 33}`.

- `client/public/assets/foliate/fb2.js:89` (source; matched `.doc`)
- `client/public/assets/foliate/fb2.js:103` (source; matched `.doc`)
- `client/public/assets/foliate/tts.js:235` (source; matched `.doc`)
- `client/public/assets/foliate/view.js:404` (source; matched `.doc`)
- `client/public/assets/foliate/view.js:794` (source; matched `.doc`)
- Additional evidence retained in JSON: 19 records.

### epub

Role-weighted hits: `{"documentation": 32, "source": 2192, "test": 6881}`.

- `README.md:89` (documentation; matched `EPUB`)
- `README.md:150` (documentation; matched `EPUB3`)
- `client/README.md:30` (documentation; matched `epub`)
- `client/public/assets/foliate/epub.js:7` (source; matched `EPUB`)
- `client/public/assets/foliate/epub.js:27` (source; matched `epub`)
- Additional evidence retained in JSON: 35 records.

### fb2

Role-weighted hits: `{"documentation": 1, "source": 430, "test": 339}`.

- `README.md:89` (documentation; matched `FB2`)
- `client/public/assets/foliate/fb2.js:87` (source; matched `fb2`)
- `client/public/assets/foliate/fb2.js:88` (source; matched `.fb2`)
- `client/public/assets/foliate/view.js:77` (source; matched `fictionbook`)
- `client/public/assets/foliate/view.js:77` (source; matched `.fb2`)
- Additional evidence retained in JSON: 35 records.

### html_xhtml

Role-weighted hits: `{"documentation": 2, "source": 97, "test": 851}`.

- `client/public/assets/foliate/epub.js:5` (source; matched `XHTML`)
- `client/public/assets/foliate/epub.js:5` (source; matched `xhtml`)
- `client/public/assets/foliate/fb2.js:17` (source; matched `XHTML`)
- `client/public/assets/foliate/fb2.js:17` (source; matched `xhtml`)
- `client/public/assets/foliate/iframe-loader.js:2` (source; matched `xhtml`)
- Additional evidence retained in JSON: 35 records.

### markdown

Role-weighted hits: `{"configuration": 17, "documentation": 84, "source": 87, "test": 39}`.

- `.github/CODEOWNERS:2` (configuration; matched `.md`)
- `.github/CODEOWNERS:3` (configuration; matched `.md`)
- `.github/ISSUE_TEMPLATE/bug_report.yml:58` (configuration; matched `markdown`)
- `.github/ISSUE_TEMPLATE/discussion.yml:6` (configuration; matched `markdown`)
- `.github/ISSUE_TEMPLATE/discussion.yml:20` (configuration; matched `.md`)
- Additional evidence retained in JSON: 35 records.

### mobi_azw

Role-weighted hits: `{"documentation": 3, "source": 1265, "test": 464}`.

- `README.md:89` (documentation; matched `MOBI`)
- `README.md:89` (documentation; matched `AZW3`)
- `client/public/assets/foliate/mobi.js:23` (source; matched `PALMDOC`)
- `client/public/assets/foliate/mobi.js:329` (source; matched `PalmDOC`)
- `client/public/assets/foliate/view.js:193` (source; matched `MOBI`)
- Additional evidence retained in JSON: 35 records.

### odt_rtf

Role-weighted hits: `{"source": 1}`.

- `server/src/modules/metadata/lib/opf-parser.ts:47` (source; matched `rtf`)

### opf

Role-weighted hits: `{"source": 72, "test": 221}`.

- `client/public/assets/foliate/epub.js:210` (source; matched `.OPF`)
- `client/public/assets/foliate/epub.js:213` (source; matched `.OPF`)
- `client/public/assets/foliate/epubcfi.js:338` (source; matched `package document`)
- `client/src/features/book/lib/__tests__/filename-stem.spec.ts:40` (test; matched `.opf`)
- `client/src/features/library/components/__tests__/LibraryCreatorMetadata.spec.ts:85` (test; matched `.opf`)
- Additional evidence retained in JSON: 35 records.

### pdf

Role-weighted hits: `{"configuration": 1, "documentation": 6, "source": 1876, "test": 957}`.

- `client/README.md:30` (documentation; matched `pdf`)
- `client/public/assets/foliate/epub.js:644` (source; matched `pdf`)
- `client/public/assets/foliate/view.js:189` (source; matched `pdf`)
- `client/src/assets/theme/tokens.css:274` (source; matched `pdf`)
- `client/src/assets/theme/tokens.css:525` (source; matched `pdf`)
- Additional evidence retained in JSON: 35 records.

### txt

Role-weighted hits: `{"configuration": 5, "documentation": 1, "source": 21, "test": 108}`.

- `Dockerfile:56` (configuration; matched `.txt`)
- `Dockerfile:56` (configuration; matched `.txt`)
- `client/public/assets/foliate/epub.js:1226` (source; matched `.txt`)
- `client/public/assets/foliate/streaming-loader.js:28` (source; matched `.txt`)
- `client/src/features/author/lib/author-facts.ts:76` (source; matched `plain text`)
- Additional evidence retained in JSON: 35 records.

### webpub_lpf

Role-weighted hits: `{"documentation": 1, "source": 17}`.

- `client/public/assets/foliate/epub.js:248` (source; matched `webpub`)
- `client/public/assets/foliate/epub.js:273` (source; matched `webpub`)
- `client/src/features/reader/epub/composables/useFoliate.ts:197` (source; matched `lpf`)
- `client/src/features/settings/components/RequestDestinationRow.vue:25` (source; matched `lpf`)
- `client/src/locales/de.json:1059` (source; matched `lpf`)
- Additional evidence retained in JSON: 4 records.
