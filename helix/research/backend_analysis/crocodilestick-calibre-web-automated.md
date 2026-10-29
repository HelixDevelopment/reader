# crocodilestick-calibre-web-automated — automated static evidence

- Inventory URL: https://github.com/crocodilestick/calibre-web-automated
- Origin: https://github.com/crocodilestick/calibre-web-automated
- Revision: `4fc2c4459965efb1b8ff43ea7849c2f47a755d9c`
- Tree: `2cbec1a3784cd9822954496040f4c06642e8cfd4`
- Analysis status: **automated_static_index_complete**; manual review: **pending**.
- Keyword hits are leads for code review, not proof of working support.

## Tree and languages

Tracked files: 1198. Text files scanned: 874 (24477756 bytes). Files skipped by size, binary classification, or sensitive-name policy remain indexed in the compressed file ledger.

- Python: 200 files
- JavaScript: 196 files
- HTML: 58 files
- CSS: 36 files
- Shell: 8 files
- XML: 4 files
- Lua: 3 files
- SQL: 1 files

## License review

Status: heuristic_requires_legal_review. Candidate notices: AGPL-3.0, BSD-2-Clause, BSD-3-Clause, GPL-2.0, GPL-3.0, LGPL, MIT, OFL-1.1.
- `LICENSE`
- `cps/static/cmaps/LICENSE`
- `cps/static/js/libs/tinymce/license.txt`
- `cps/static/standard_fonts/LICENSE_FOXIT`
- `cps/static/standard_fonts/LICENSE_LIBERATION`

## Feature evidence

### ai_llm

Role-weighted hits: `{"configuration": 3, "documentation": 25, "source": 1884, "test": 104}`.

- `.github/triage/README.md:19` (documentation; matched `rag`)
- `.github/triage/export.py:15` (configuration; matched `rag`)
- `.github/triage/export.py:269` (configuration; matched `rag`)
- `.github/workflows/tests.yml:87` (test; matched `rag`)
- `.github/workflows/tests.yml:91` (test; matched `rag`)
- Additional evidence retained in JSON: 35 records.

### annotations

Role-weighted hits: `{"configuration": 2, "documentation": 5, "source": 4922}`.

- `.github/triage/README.md:55` (documentation; matched `highlight`)
- `.github/triage/export.py:30` (configuration; matched `annotation`)
- `.github/triage/export.py:44` (configuration; matched `highlight`)
- `changelogs/V4/v4.0.0/V4.0.0_changelog.md:199` (source; matched `Annotation`)
- `changelogs/V4/v4.0.0/V4.0.0_changelog.md:207` (source; matched `annotation`)
- Additional evidence retained in JSON: 35 records.

### backup_restore

Role-weighted hits: `{"configuration": 4, "documentation": 1, "source": 1376, "test": 40}`.

- `.github/copilot-instructions.md:14` (configuration; matched `backup`)
- `.github/copilot-instructions.md:37` (configuration; matched `Backup`)
- `.github/copilot-instructions.md:38` (configuration; matched `backup`)
- `README.md:225` (documentation; matched `backup`)
- `changelogs/V2/V2.1.0_changelog.md:30` (source; matched `backup`)
- Additional evidence retained in JSON: 35 records.

### collections_smart

Role-weighted hits: `{"documentation": 9, "source": 4139, "test": 49}`.

- `README.md:70` (documentation; matched `collection`)
- `README.md:144` (documentation; matched `Shelves`)
- `README.md:144` (documentation; matched `shelves`)
- `changelogs/V3.0/README_V3.md:88` (source; matched `series`)
- `changelogs/V3.0/README_V3.md:261` (source; matched `collection`)
- Additional evidence retained in JSON: 35 records.

### comics_manga

Role-weighted hits: `{"configuration": 7, "documentation": 3, "source": 1441, "test": 4}`.

- `.github/copilot-instructions.md:172` (configuration; matched `cbz`)
- `.github/copilot-instructions.md:172` (configuration; matched `cbr`)
- `.github/copilot-instructions.md:172` (configuration; matched `cb7`)
- `README.md:159` (documentation; matched `cbz`)
- `README.md:159` (documentation; matched `cbr`)
- Additional evidence retained in JSON: 35 records.

### compression_cache

Role-weighted hits: `{"configuration": 55, "documentation": 3, "source": 2767, "test": 157}`.

- `.dockerignore:28` (source; matched `cache`)
- `.github/copilot-instructions.md:37` (configuration; matched `Thumbnail`)
- `.github/copilot-instructions.md:38` (configuration; matched `thumbnail`)
- `.github/triage/README.md:22` (documentation; matched `cache`)
- `.github/triage/README.md:57` (documentation; matched `CACHE`)
- Additional evidence retained in JSON: 35 records.

### conversion

Role-weighted hits: `{"configuration": 35, "documentation": 11, "source": 1414, "test": 83}`.

- `.github/copilot-instructions.md:4` (configuration; matched `conversion`)
- `.github/copilot-instructions.md:12` (configuration; matched `conversion`)
- `.github/copilot-instructions.md:43` (configuration; matched `conversion`)
- `.github/workflows/tests.yml:71` (test; matched `converter`)
- `Dockerfile:24` (configuration; matched `KEPUB`)
- Additional evidence retained in JSON: 35 records.

### daisy_accessibility

Role-weighted hits: `{"source": 1152, "test": 2}`.

- `changelogs/V4/v4.0.0/commits_since_314.md:693` (source; matched `Accessibility`)
- `cps/kobo.py:523` (source; matched `Accessibility`)
- `cps/kobo.py:1658` (source; matched `accessibility`)
- `cps/metadata_provider/lubimyczytac.py:97` (source; matched `aria-`)
- `cps/static/css/caliBlur.css:504` (source; matched `aria-`)
- Additional evidence retained in JSON: 35 records.

### database

Role-weighted hits: `{"configuration": 7, "documentation": 15, "source": 204, "test": 94}`.

- `.github/copilot-instructions.md:17` (configuration; matched `SQLite`)
- `.github/copilot-instructions.md:20` (configuration; matched `SQLite`)
- `.github/copilot-instructions.md:75` (configuration; matched `sqlite`)
- `.gitignore:98` (source; matched `sqlite`)
- `.gitignore:99` (source; matched `sqlite`)
- Additional evidence retained in JSON: 35 records.

### deduplication

Role-weighted hits: `{"configuration": 10, "documentation": 51, "source": 7445, "test": 941}`.

- `.github/triage/README.md:34` (documentation; matched `duplicate`)
- `.github/workflows/docker-image-build-dev.yml:114` (configuration; matched `sha256`)
- `.github/workflows/docker-image-build-dev.yml:217` (configuration; matched `sha256`)
- `.github/workflows/docker-image-build-dev.yml:315` (configuration; matched `sha256`)
- `.github/workflows/docker-image-build-release.yml:76` (configuration; matched `sha256`)
- Additional evidence retained in JSON: 35 records.

### deployment

Role-weighted hits: `{"configuration": 108, "documentation": 28, "source": 367, "test": 386}`.

- `.dockerignore:2` (source; matched `docker`)
- `.dockerignore:12` (source; matched `Docker`)
- `.dockerignore:13` (source; matched `docker`)
- `.gitattributes:6` (source; matched `Docker`)
- `.github/copilot-instructions.md:4` (configuration; matched `Docker`)
- Additional evidence retained in JSON: 35 records.

### dictionary_language

Role-weighted hits: `{"configuration": 27, "documentation": 3, "source": 780, "test": 6}`.

- `.github/copilot-instructions.md:133` (configuration; matched `Translation`)
- `.github/copilot-instructions.md:138` (configuration; matched `translation`)
- `.github/copilot-instructions.md:142` (configuration; matched `translation`)
- `.github/triage/README.md:32` (documentation; matched `translation`)
- `.github/workflows/docker-image-build-dev.yml:15` (configuration; matched `translation`)
- Additional evidence retained in JSON: 35 records.

### http_protocols

Role-weighted hits: `{"documentation": 7, "source": 67, "test": 38}`.

- `README.md:21` (documentation; matched `Quic`)
- `README.md:27` (documentation; matched `Quic`)
- `README.md:27` (documentation; matched `quic`)
- `changelogs/V3.0/README_V3.md:12` (source; matched `Quic`)
- `changelogs/V3.0/README_V3.md:18` (source; matched `Quic`)
- Additional evidence retained in JSON: 35 records.

### kobo_sync

Role-weighted hits: `{"configuration": 3, "documentation": 2, "source": 443, "test": 4}`.

- `.github/copilot-instructions.md:159` (configuration; matched `Kobo Sync`)
- `.github/copilot-instructions.md:160` (configuration; matched `kobo.py` + `cps/kosync`)
- `.github/copilot-instructions.md:161` (configuration; matched `Kobo device sync**: Native Calibre-Web functionality, sync`)
- `README.md:183` (documentation; matched `Kobo sync`)
- `changelogs/V3.0/V3-reddit-release-post.md:56` (source; matched `Kobo and Kindle users using the same library**, now each book can exist both as an EPUB for Send-to-`)
- Additional evidence retained in JSON: 35 records.

### koreader_sync

Role-weighted hits: `{"configuration": 3, "documentation": 31, "source": 674, "test": 166}`.

- `.github/copilot-instructions.md:160` (configuration; matched `kosync`)
- `.github/copilot-instructions.md:162` (configuration; matched `KOReader sync`)
- `.github/copilot-instructions.md:163` (configuration; matched `kosync`)
- `README.md:34` (documentation; matched `KOReader Sync`)
- `README.md:34` (documentation; matched `KOSync`)
- Additional evidence retained in JSON: 35 records.

### lcp_drm

Role-weighted hits: `{"configuration": 1, "documentation": 5, "source": 17}`.

- `.github/copilot-instructions.md:178` (configuration; matched `DRM`)
- `LICENSE:6` (documentation; matched `license document`)
- `LICENSE:392` (documentation; matched `license document`)
- `LICENSE:395` (documentation; matched `license document`)
- `README.md:305` (documentation; matched `DRM`)
- Additional evidence retained in JSON: 14 records.

### local_library

Role-weighted hits: `{"configuration": 2, "documentation": 3, "source": 33, "test": 8}`.

- `.github/copilot-instructions.md:17` (configuration; matched `filesystem`)
- `.github/copilot-instructions.md:50` (configuration; matched `filesystem`)
- `README.md:100` (documentation; matched `filesystem`)
- `README.md:105` (documentation; matched `filesystem`)
- `README.md:110` (documentation; matched `filesystem`)
- Additional evidence retained in JSON: 35 records.

### metadata_enrichment

Role-weighted hits: `{"configuration": 25, "documentation": 36, "source": 4045, "test": 147}`.

- `.github/copilot-instructions.md:13` (configuration; matched `metadata`)
- `.github/copilot-instructions.md:13` (configuration; matched `metadata`)
- `.github/copilot-instructions.md:13` (configuration; matched `metadata`)
- `.github/triage/export.py:17` (configuration; matched `metadata`)
- `README.md:92` (documentation; matched `metadata`)
- Additional evidence retained in JSON: 35 records.

### metadata_review

Role-weighted hits: `{"configuration": 1, "documentation": 1, "source": 87, "test": 6}`.

- `.github/triage/export.py:17` (configuration; matched `metadata, body, comments, review`)
- `README.md:92` (documentation; matched `metadata.db` and `app.db` to prevent lock`)
- `changelogs/V3.1/V3.1.0_changelog.md:36` (source; matched `Metadata interface—making it easier and clearer for you to review`)
- `changelogs/V4/v4.0.0/V4.0.0_changelog.md:312` (source; matched `locking guarantees that many network filesystems can't provide. When `NETWORK_SHARE_MODE=true`, CWA `)
- `changelogs/V4/v4.0.0/V4.0.0_changelog.md:418` (source; matched `Locking** - Fixed database locking and race conditions in cover_enforcer preventing metadata`)
- Additional evidence retained in JSON: 35 records.

### multiuser_auth

Role-weighted hits: `{"configuration": 15, "documentation": 34, "source": 5909, "test": 139}`.

- `.github/copilot-instructions.md:30` (configuration; matched `oauth`)
- `.github/copilot-instructions.md:152` (configuration; matched `OAuth`)
- `.github/copilot-instructions.md:153` (configuration; matched `OAuth`)
- `.github/triage/export.py:66` (configuration; matched `Authorization`)
- `.github/workflows/update-translations.yml:80` (configuration; matched `Authorization`)
- Additional evidence retained in JSON: 35 records.

### network_sources

Role-weighted hits: `{"configuration": 9, "documentation": 23, "source": 94, "test": 6}`.

- `.github/copilot-instructions.md:25` (configuration; matched `NFS`)
- `.github/copilot-instructions.md:25` (configuration; matched `SMB`)
- `.github/copilot-instructions.md:146` (configuration; matched `Network Share`)
- `README.md:89` (documentation; matched `Network Share`)
- `README.md:89` (documentation; matched `NFS`)
- Additional evidence retained in JSON: 35 records.

### notes_export

Role-weighted hits: `{"source": 4}`.

- `cps/static/js/libs/epub.min.js:1` (source; matched `exports&&"object"==typeof module?module.exports=e(require("JSZip")):"function"==typeof define&&defin`)
- `cps/static/js/libs/reader.min.js:8` (source; matched `exports&&"undefined"!=typeof module?b(exports):"function"==typeof define&&define.amd?define(["export`)
- `cps/static/js/libs/tinymce/themes/silver/theme.min.js:9` (source; matched `HighlightSubmenu),i.hasConfigured(dd)&&dd.isDisabled(i))return vt.some(i);var t=c(i);return g.expand`)
- `cps/static/js/libs/tinymce/tinymce.min.js:9` (source; matched `exports:er};function nr(e){setTimeout(function(){throw e},0)}!function(){var e=this,t=function(){var`)

### opds

Role-weighted hits: `{"configuration": 1, "documentation": 4, "source": 544, "test": 9}`.

- `.github/copilot-instructions.md:30` (configuration; matched `opds`)
- `README.md:129` (documentation; matched `OPDS`)
- `README.md:422` (documentation; matched `opds`)
- `README.md:422` (documentation; matched `OPDS`)
- `changelogs/V3.0/README_V3.md:48` (source; matched `OPDS`)
- Additional evidence retained in JSON: 35 records.

### opensearch

Role-weighted hits: `{"source": 5}`.

- `cps/templates/feed.xml:33` (source; matched `application/opensearchdescription+xml`)
- `cps/templates/index.xml:11` (source; matched `application/opensearchdescription+xml`)
- `cps/templates/osd.xml:2` (source; matched `OpenSearch`)
- `cps/templates/osd.xml:2` (source; matched `opensearch`)
- `cps/templates/osd.xml:16` (source; matched `OpenSearch`)

### pdf_fixed_layout

Role-weighted hits: `{"documentation": 1, "source": 18323}`.

- `README.md:159` (documentation; matched `djvu`)
- `changelogs/V3.0/README_V3.md:78` (source; matched `djvu`)
- `changelogs/V3.0/V3-reddit-release-post.md:54` (source; matched `djvu`)
- `changelogs/V3.0/V3.0.0_changelog.md:46` (source; matched `djvu`)
- `cps/__init__.py:56` (source; matched `djvu`)
- Additional evidence retained in JSON: 35 records.

### permissions

Role-weighted hits: `{"configuration": 6, "documentation": 33, "source": 946, "test": 29}`.

- `.github/copilot-instructions.md:10` (configuration; matched `permission`)
- `.github/copilot-instructions.md:22` (configuration; matched `permission`)
- `.github/copilot-instructions.md:129` (configuration; matched `user.role`)
- `.github/workflows/update-translations.yml:15` (configuration; matched `permission`)
- `Dockerfile:251` (configuration; matched `permission`)
- Additional evidence retained in JSON: 35 records.

### progress_sync

Role-weighted hits: `{"documentation": 8, "source": 335, "test": 87}`.

- `README.md:240` (documentation; matched `progress sync`)
- `README.md:303` (documentation; matched `progress sync`)
- `README.md:449` (documentation; matched `syncing functionality, allowing you to sync your reading progress`)
- `changelogs/V3.0/README_V3.md:188` (source; matched `sync read progress`)
- `changelogs/V3.0/V3-reddit-release-post.md:56` (source; matched `sync their read progress`)
- Additional evidence retained in JSON: 35 records.

### reader_typography

Role-weighted hits: `{"configuration": 1, "documentation": 3, "source": 2416, "test": 1}`.

- `.github/copilot-instructions.md:29` (configuration; matched `theme`)
- `.github/triage/README.md:34` (documentation; matched `theme`)
- `README.md:273` (documentation; matched `Dark Mode`)
- `README.md:274` (documentation; matched `theme`)
- `changelogs/V2/V2.1.2_changelog.md:21` (source; matched `Theme`)
- Additional evidence retained in JSON: 35 records.

### search

Role-weighted hits: `{"documentation": 1, "source": 3}`.

- `README.md:304` (documentation; matched `Full Text Search`)
- `changelogs/V4/v4.0.0/V4.0.0_changelog.md:532` (source; matched `Full Text Search`)
- `changelogs/V4/v4.0.0/V4_reddit_release.md:54` (source; matched `Full Text Search`)
- `changelogs/V4/v4.0.0/commits_since_314.md:1266` (source; matched `Full Text Search`)

### security

Role-weighted hits: `{"configuration": 57, "documentation": 80, "source": 2618, "test": 122}`.

- `.github/triage/export.py:43` (configuration; matched `https`)
- `.github/workflows/discord-release-bot.yml:19` (configuration; matched `secret`)
- `.github/workflows/discord-release-bot.yml:21` (configuration; matched `https`)
- `.github/workflows/docker-image-build-dev.yml:74` (configuration; matched `secret`)
- `.github/workflows/docker-image-build-dev.yml:75` (configuration; matched `secret`)
- Additional evidence retained in JSON: 35 records.

### testing

Role-weighted hits: `{"configuration": 8, "documentation": 5, "source": 50, "test": 507}`.

- `.dockerignore:24` (source; matched `pytest`)
- `.dockerignore:28` (source; matched `pytest`)
- `.github/workflows/tests.yml:11` (test; matched `integration test`)
- `.github/workflows/tests.yml:74` (test; matched `unit test`)
- `.github/workflows/tests.yml:78` (test; matched `pytest`)
- Additional evidence retained in JSON: 35 records.

### tts_audio

Role-weighted hits: `{"configuration": 2, "source": 47}`.

- `.github/copilot-instructions.md:179` (configuration; matched `Audiobook`)
- `.github/copilot-instructions.md:179` (configuration; matched `audiobook`)
- `changelogs/V3.1/V3.1.0_changelog.md:20` (source; matched `Audiobook`)
- `changelogs/V4/v4.0.0/commits_since_314.md:709` (source; matched `audiobook`)
- `changelogs/V4/v4.0.7/commits_since_1b80f9a.md:8` (source; matched `audiobook`)
- Additional evidence retained in JSON: 13 records.

### watch_import

Role-weighted hits: `{"configuration": 8, "documentation": 3, "source": 38, "test": 5}`.

- `.github/copilot-instructions.md:12` (configuration; matched `File watcher`)
- `.github/copilot-instructions.md:149` (configuration; matched `file watcher`)
- `.github/copilot-instructions.md:149` (configuration; matched `inotify`)
- `Dockerfile:61` (configuration; matched `inotify`)
- `Dockerfile:208` (configuration; matched `inotify`)
- Additional evidence retained in JSON: 31 records.


## Format evidence

### cb7_cbt_cba

Role-weighted hits: `{"configuration": 1, "documentation": 1, "source": 23, "test": 1}`.

- `.github/copilot-instructions.md:172` (configuration; matched `cb7`)
- `README.md:159` (documentation; matched `.cb7`)
- `changelogs/V3.0/README_V3.md:78` (source; matched `.cb7`)
- `changelogs/V3.0/V3-reddit-release-post.md:54` (source; matched `.cb7`)
- `changelogs/V3.0/V3.0.0_changelog.md:46` (source; matched `.cb7`)
- Additional evidence retained in JSON: 19 records.

### cbr

Role-weighted hits: `{"configuration": 1, "documentation": 1, "source": 31, "test": 1}`.

- `.github/copilot-instructions.md:172` (configuration; matched `cbr`)
- `README.md:159` (documentation; matched `.cbr`)
- `changelogs/V3.0/README_V3.md:78` (source; matched `.cbr`)
- `changelogs/V3.0/V3-reddit-release-post.md:54` (source; matched `.cbr`)
- `changelogs/V3.0/V3.0.0_changelog.md:46` (source; matched `.cbr`)
- Additional evidence retained in JSON: 27 records.

### cbz

Role-weighted hits: `{"configuration": 1, "documentation": 1, "source": 29, "test": 1}`.

- `.github/copilot-instructions.md:172` (configuration; matched `cbz`)
- `README.md:159` (documentation; matched `.cbz`)
- `changelogs/V3.0/README_V3.md:78` (source; matched `.cbz`)
- `changelogs/V3.0/V3-reddit-release-post.md:54` (source; matched `.cbz`)
- `changelogs/V3.0/V3.0.0_changelog.md:46` (source; matched `.cbz`)
- Additional evidence retained in JSON: 26 records.

### daisy

Role-weighted hits: `{"source": 2, "test": 2}`.

- `cps/static/js/libs/compromise.min.js:3` (source; matched `daisy`)
- `cps/static/js/libs/epub.min.js:1` (source; matched `dtbook`)
- `tests/fixtures/sample_books/alice_in_wonderland.txt:66` (test; matched `daisy`)
- `tests/integration/test_ingest_checksums.py:377` (test; matched `.daisy`)

### djvu

Role-weighted hits: `{"documentation": 1, "source": 183}`.

- `README.md:159` (documentation; matched `.djvu`)
- `changelogs/V3.0/README_V3.md:78` (source; matched `.djvu`)
- `changelogs/V3.0/V3-reddit-release-post.md:54` (source; matched `.djvu`)
- `changelogs/V3.0/V3.0.0_changelog.md:46` (source; matched `.djvu`)
- `cps/__init__.py:55` (source; matched `.djv`)
- Additional evidence retained in JSON: 35 records.

### doc_docx

Role-weighted hits: `{"configuration": 1, "documentation": 1, "source": 84, "test": 1}`.

- `.github/copilot-instructions.md:172` (configuration; matched `docx`)
- `README.md:159` (documentation; matched `.docx`)
- `changelogs/V3.0/README_V3.md:78` (source; matched `.docx`)
- `changelogs/V3.0/V3-reddit-release-post.md:54` (source; matched `.docx`)
- `changelogs/V3.0/V3.0.0_changelog.md:46` (source; matched `.docx`)
- Additional evidence retained in JSON: 20 records.

### epub

Role-weighted hits: `{"configuration": 5, "documentation": 28, "source": 1705, "test": 266}`.

- `.github/copilot-instructions.md:45` (configuration; matched `EPUB `)
- `.github/copilot-instructions.md:82` (configuration; matched `.epub`)
- `.github/workflows/tests.yml:70` (test; matched `epub`)
- `.gitignore:34` (source; matched `.epub`)
- `.gitignore:36` (source; matched `.epub`)
- Additional evidence retained in JSON: 35 records.

### fb2

Role-weighted hits: `{"configuration": 1, "documentation": 1, "source": 28, "test": 1}`.

- `.github/copilot-instructions.md:172` (configuration; matched `fb2`)
- `README.md:159` (documentation; matched `.fb2`)
- `changelogs/V3.0/README_V3.md:78` (source; matched `.fb2`)
- `changelogs/V3.0/V3-reddit-release-post.md:54` (source; matched `.fb2`)
- `changelogs/V3.0/V3.0.0_changelog.md:46` (source; matched `.fb2`)
- Additional evidence retained in JSON: 20 records.

### html_xhtml

Role-weighted hits: `{"documentation": 3, "source": 42411, "test": 31}`.

- `LICENSE:674` (documentation; matched `.html`)
- `README.md:129` (documentation; matched `HTML5`)
- `README.md:159` (documentation; matched `.html`)
- `changelogs/V3.0/README_V3.md:48` (source; matched `HTML5`)
- `changelogs/V3.0/README_V3.md:78` (source; matched `.html`)
- Additional evidence retained in JSON: 35 records.

### markdown

Role-weighted hits: `{"configuration": 19, "documentation": 8, "source": 77, "test": 7}`.

- `.dockerignore:10` (source; matched `.md`)
- `.dockerignore:26` (source; matched `.md`)
- `.github/triage/README.md:3` (documentation; matched `markdown`)
- `.github/triage/README.md:17` (documentation; matched `.md`)
- `.github/triage/export.py:7` (configuration; matched `markdown`)
- Additional evidence retained in JSON: 32 records.

### mobi_azw

Role-weighted hits: `{"configuration": 6, "documentation": 5, "source": 212, "test": 81}`.

- `.github/copilot-instructions.md:82` (configuration; matched `.azw`)
- `.github/copilot-instructions.md:172` (configuration; matched `mobi`)
- `.gitignore:37` (source; matched `.mobi`)
- `README.md:156` (documentation; matched `MOBI`)
- `README.md:156` (documentation; matched `AZW3`)
- Additional evidence retained in JSON: 35 records.

### odt_rtf

Role-weighted hits: `{"configuration": 1, "documentation": 2, "source": 32, "test": 1}`.

- `.github/copilot-instructions.md:172` (configuration; matched `odt`)
- `README.md:159` (documentation; matched `.odt`)
- `README.md:159` (documentation; matched `.rtf`)
- `changelogs/V3.0/README_V3.md:78` (source; matched `.odt`)
- `changelogs/V3.0/README_V3.md:78` (source; matched `.rtf`)
- Additional evidence retained in JSON: 21 records.

### opf

Role-weighted hits: `{"source": 13, "test": 15}`.

- `cps/gdriveutils.py:628` (source; matched `.opf`)
- `cps/helper.py:1331` (source; matched `.opf`)
- `cps/schedule.py:29` (source; matched `.opf`)
- `cps/tasks/convert.py:278` (source; matched `.opf`)
- `cps/tasks/metadata_backup.py:87` (source; matched `.opf`)
- Additional evidence retained in JSON: 10 records.

### pdf

Role-weighted hits: `{"configuration": 2, "documentation": 3, "source": 1874, "test": 14}`.

- `.github/copilot-instructions.md:172` (configuration; matched `pdf`)
- `.github/copilot-instructions.md:174` (configuration; matched `PDF`)
- `.gitignore:39` (source; matched `.pdf`)
- `README.md:156` (documentation; matched `PDF`)
- `README.md:159` (documentation; matched `.pdf`)
- Additional evidence retained in JSON: 35 records.

### txt

Role-weighted hits: `{"configuration": 9, "source": 135, "test": 28}`.

- `.dockerignore:25` (source; matched `.txt`)
- `.github/workflows/docker-image-build-dev.yml:13` (configuration; matched `.txt`)
- `.github/workflows/docker-image-build-dev.yml:14` (configuration; matched `.txt`)
- `.github/workflows/tests.yml:63` (test; matched `.txt`)
- `.github/workflows/tests.yml:64` (test; matched `.txt`)
- Additional evidence retained in JSON: 35 records.

### webpub_lpf

Role-weighted hits: `{"configuration": 1, "source": 14}`.

- `.github/triage/export.py:313` (configuration; matched `lpF`)
- `changelogs/V4/v4.0.0/commits_since_314.md:465` (source; matched `lpf`)
- `cps/static/css/caliBlur.css:55` (source; matched `LpF`)
- `cps/static/css/caliBlur.css:55` (source; matched `lpF`)
- `cps/static/css/libs/bootstrap.min.css.map:1` (source; matched `lPF`)
- Additional evidence retained in JSON: 2 records.
