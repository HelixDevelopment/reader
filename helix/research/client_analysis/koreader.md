# koreader — automated static evidence

- Inventory URL: https://github.com/koreader/koreader
- Origin: https://github.com/koreader/koreader
- Revision: `c8eb592a8b6cac9e65be7288bd27fb0be9405ad1`
- Tree: `781ca7f415bfc77933dde65aaa1e960c5c47ed67`
- Analysis status: **automated_static_index_complete**; manual review: **pending**.
- Keyword hits are leads for code review, not proof of working support.

## Tree and languages

Tracked files: 1004. Text files scanned: 976 (11565652 bytes). Files skipped by size, binary classification, or sensitive-name policy remain indexed in the compressed file ledger.

- Lua: 601 files
- Shell: 58 files
- XML: 3 files
- Python: 2 files

## License review

Status: heuristic_requires_legal_review. Candidate notices: AGPL-3.0, BSD-2-Clause, BSD-3-Clause, MIT, OFL-1.1.
- `COPYING`
- `plugins/newsdownloader.koplugin/lib/LICENCE_lua-feedparser`
- `plugins/newsdownloader.koplugin/lib/LICENSE_LuaXML`
- `resources/icons/mdlight/LICENSE-SIL-OFL-1.1.md`
- `resources/icons/mdlight/license.txt`
- `resources/icons/mdlight/notice-info.svg`
- `resources/icons/mdlight/notice-question.svg`
- `resources/icons/mdlight/notice-warning.svg`
- `resources/icons/src/license.txt`

## Feature evidence

### ai_llm

Role-weighted hits: `{"configuration": 1, "documentation": 17, "source": 874, "test": 89}`.

- `.ci/after_success.sh:10` (source; matched `rag`)
- `.ci/check.sh:26` (source; matched `rag`)
- `.ci/check.sh:33` (source; matched `rag`)
- `.ci/check.sh:53` (source; matched `rag`)
- `.codecov.yml:1` (source; matched `rag`)
- Additional evidence retained in JSON: 35 records.

### annotations

Role-weighted hits: `{"documentation": 3, "source": 3380, "test": 445}`.

- `Makefile:97` (source; matched `bookmark`)
- `Makefile:98` (source; matched `highlight`)
- `defaults.lua:50` (source; matched `highlight`)
- `defaults.lua:52` (source; matched `HIGHLIGHT`)
- `doc/Building.md:260` (documentation; matched `bookmark`)
- Additional evidence retained in JSON: 35 records.

### backup_restore

Role-weighted hits: `{"configuration": 28, "source": 462, "test": 35}`.

- `.github/workflows/build_matrix.yml:131` (configuration; matched `Restore`)
- `.github/workflows/build_matrix.yml:132` (configuration; matched `restore`)
- `.github/workflows/build_matrix.yml:133` (configuration; matched `restore`)
- `frontend/apps/filemanager/filemanager.lua:794` (source; matched `backup`)
- `frontend/apps/filemanager/filemanager.lua:796` (source; matched `backup`)
- Additional evidence retained in JSON: 35 records.

### collections_smart

Role-weighted hits: `{"source": 889, "test": 123}`.

- `frontend/apps/filemanager/filemanager.lua:14` (source; matched `Collection`)
- `frontend/apps/filemanager/filemanager.lua:14` (source; matched `collection`)
- `frontend/apps/filemanager/filemanager.lua:28` (source; matched `Collection`)
- `frontend/apps/filemanager/filemanagerbookinfo.lua:37` (source; matched `series`)
- `frontend/apps/filemanager/filemanagerbookinfo.lua:38` (source; matched `series`)
- Additional evidence retained in JSON: 35 records.

### comics_manga

Role-weighted hits: `{"documentation": 2, "source": 216, "test": 8}`.

- `README.md:22` (documentation; matched `CBT`)
- `README.md:22` (documentation; matched `CBZ`)
- `frontend/apps/filemanager/filemanagermenu.lua:353` (source; matched `Manga`)
- `frontend/apps/filemanager/filemanagermenu.lua:355` (source; matched `Manga`)
- `frontend/apps/reader/modules/readerhighlight.lua:758` (source; matched `manga`)
- Additional evidence retained in JSON: 35 records.

### compression_cache

Role-weighted hits: `{"configuration": 75, "documentation": 31, "source": 2001, "test": 93}`.

- `.ci/build_environment_setup.sh:16` (source; matched `cache`)
- `.ci/build_environment_setup.sh:17` (source; matched `CACHE`)
- `.ci/build_environment_setup.sh:17` (source; matched `cache`)
- `.ci/build_jobs.yml:47` (source; matched `cache`)
- `.ci/build_jobs.yml:56` (source; matched `cache`)
- Additional evidence retained in JSON: 35 records.

### conversion

Role-weighted hits: `{"documentation": 2, "source": 161, "test": 19}`.

- `README.md:22` (documentation; matched `EPUB, FB2, Mobi, DOC, RTF, HTML, CHM, TXT). Scanned PDF/DjVu documents can also be reflowed with the`)
- `doc/config.ld:3` (documentation; matched `EPUB, FB2`)
- `frontend/apps/filemanager/filemanager.lua:15` (source; matched `Converter`)
- `frontend/apps/filemanager/filemanager.lua:15` (source; matched `converter`)
- `frontend/apps/filemanager/filemanager.lua:282` (source; matched `Converter`)
- Additional evidence retained in JSON: 35 records.

### daisy_accessibility

Role-weighted hits: `{"source": 17}`.

- `frontend/ui/widget/container/inputcontainer.lua:407` (source; matched `daisy`)
- `frontend/ui/widget/container/inputcontainer.lua:421` (source; matched `daisy`)
- `frontend/ui/widget/container/inputcontainer.lua:427` (source; matched `Daisy`)
- `frontend/ui/wikipedia.lua:1439` (source; matched `daisy`)
- `frontend/ui/wikipedia.lua:1440` (source; matched `daisy`)
- Additional evidence retained in JSON: 12 records.

### database

Role-weighted hits: `{"documentation": 11, "source": 77, "test": 11}`.

- `COPYING:625` (documentation; matched `redis`)
- `COPYING:635` (documentation; matched `redis`)
- `doc/DataStore.md:14` (documentation; matched `SQLite`)
- `doc/DataStore.md:16` (documentation; matched `SQLite`)
- `doc/DataStore.md:19` (documentation; matched `sqlite`)
- Additional evidence retained in JSON: 35 records.

### deduplication

Role-weighted hits: `{"documentation": 3, "source": 116, "test": 11}`.

- `.ci/build_matrix_generate.sh:18` (source; matched `sha256`)
- `doc/Collaborating_with_Git.md:129` (documentation; matched `duplicate`)
- `frontend/apps/filemanager/filemanager.lua:1148` (source; matched `checksum`)
- `frontend/apps/reader/modules/readerbookmark.lua:1663` (source; matched `duplicate`)
- `frontend/apps/reader/modules/readerbookmark.lua:1663` (source; matched `duplicate`)
- Additional evidence retained in JSON: 35 records.

### deployment

Role-weighted hits: `{"configuration": 1, "documentation": 7, "source": 63, "test": 1}`.

- `.ci/build_environment_setup.sh:39` (source; matched `Docker`)
- `.ci/build_environment_setup.sh:41` (source; matched `docker`)
- `.ci/build_environment_setup.sh:48` (source; matched `docker`)
- `.ci/common.sh:64` (source; matched `Docker`)
- `.ci/common.sh:69` (source; matched `docker`)
- Additional evidence retained in JSON: 18 records.

### dictionary_language

Role-weighted hits: `{"configuration": 6, "documentation": 10, "source": 2703, "test": 30}`.

- `.ci/translations_update.sh:14` (source; matched `translation`)
- `.ci/translations_update.sh:17` (source; matched `translation`)
- `.ci/translations_update.sh:19` (source; matched `Translation`)
- `.github/workflows/doc_and_l10n_update.yml:56` (configuration; matched `translation`)
- `.github/workflows/doc_and_l10n_update.yml:57` (configuration; matched `translation`)
- Additional evidence retained in JSON: 35 records.

### http_protocols

Role-weighted hits: `{"documentation": 4, "source": 415, "test": 41}`.

- `doc/Collaborating_with_Git.md:59` (documentation; matched `Quic`)
- `doc/DataStore.md:20` (documentation; matched `quic`)
- `doc/Development_guide.md:99` (documentation; matched `quic`)
- `doc/Hacking.md:36` (documentation; matched `quic`)
- `frontend/apps/filemanager/filemanagershortcuts.lua:4` (source; matched `Quic`)
- Additional evidence retained in JSON: 35 records.

### kobo_sync

Role-weighted hits: `{"source": 5}`.

- `defaults.lua:188` (source; matched `KOBO_SYNC`)
- `frontend/device/kobo/device.lua:1534` (source; matched `Kobo suspend: sync`)
- `frontend/device/kobo/powerd.lua:24` (source; matched `KoboPowerD:_sync`)
- `frontend/device/kobo/powerd.lua:218` (source; matched `KOBO_SYNC`)
- `frontend/ui/data/onetime_migration.lua:856` (source; matched `kobolight", "send2ebook", "storagestat", "zsync`)

### koreader_sync

Role-weighted hits: `{"source": 186, "test": 124}`.

- `frontend/device/input.lua:1584` (source; matched `KOSync`)
- `frontend/pluginloader.lua:48` (source; matched `kosync`)
- `frontend/ui/data/onetime_migration.lua:563` (source; matched `KOSync`)
- `frontend/ui/data/onetime_migration.lua:569` (source; matched `kosync`)
- `frontend/ui/data/onetime_migration.lua:570` (source; matched `KOSync`)
- Additional evidence retained in JSON: 25 records.

### lcp_drm

Role-weighted hits: `{"documentation": 3, "source": 4}`.

- `COPYING:6` (documentation; matched `license document`)
- `COPYING:380` (documentation; matched `license document`)
- `COPYING:383` (documentation; matched `license document`)
- `frontend/ui/quickstart.lua:94` (source; matched `DRM`)
- `kodev:421` (source; matched `drm`)
- Additional evidence retained in JSON: 2 records.

### local_library

Role-weighted hits: `{"documentation": 1, "source": 122, "test": 4}`.

- `doc/Building.md:132` (documentation; matched `filesystem`)
- `frontend/apps/filemanager/filemanagerbookinfo.lua:280` (source; matched `file browser`)
- `frontend/apps/filemanager/filemanagerbookinfo.lua:1021` (source; matched `file browser`)
- `frontend/apps/filemanager/filemanagerbookinfo.lua:1021` (source; matched `file browser`)
- `frontend/apps/filemanager/filemanagercollection.lua:459` (source; matched `file browser`)
- Additional evidence retained in JSON: 35 records.

### metadata_enrichment

Role-weighted hits: `{"configuration": 1, "documentation": 3, "source": 971, "test": 60}`.

- `.ci/assets_parse.jq:100` (source; matched `metadata`)
- `.github/workflows/stable_to_ota.yml:36` (configuration; matched `metadata`)
- `README.md:26` (documentation; matched `metadata`)
- `doc/Building.md:39` (documentation; matched `metadata`)
- `frontend/apps/filemanager/filemanager.lua:564` (source; matched `Metadata`)
- Additional evidence retained in JSON: 35 records.

### metadata_review

Role-weighted hits: `{"source": 9}`.

- `.ci/common.sh:15` (source; matched `DRY_RUN`)
- `.ci/common.sh:15` (source; matched `DRY_RUN`)
- `.ci/common.sh:41` (source; matched `DRY_RUN`)
- `kodev:23` (source; matched `dry_run`)
- `kodev:67` (source; matched `dry_run`)
- Additional evidence retained in JSON: 2 records.

### multiuser_auth

Role-weighted hits: `{"documentation": 1, "source": 40, "test": 1}`.

- `COPYING:299` (documentation; matched `authorization`)
- `defaults.lua:192` (source; matched `authentication`)
- `frontend/socketutil.lua:185` (source; matched `authorization`)
- `frontend/socketutil.lua:187` (source; matched `authorization`)
- `frontend/ui/network/manager.lua:1179` (source; matched `authentication`)
- Additional evidence retained in JSON: 23 records.

### network_sources

Role-weighted hits: `{"documentation": 1, "source": 80}`.

- `README.md:34` (documentation; matched `FTP`)
- `frontend/ui/data/dictionaries.lua:171` (source; matched `ftp`)
- `frontend/ui/data/dictionaries.lua:171` (source; matched `ftp`)
- `frontend/ui/data/dictionaries.lua:395` (source; matched `ftp`)
- `frontend/ui/quickstart.lua:256` (source; matched `Cloud storage`)
- Additional evidence retained in JSON: 17 records.

### notes_export

Role-weighted hits: `{"source": 60, "test": 21}`.

- `frontend/apps/filemanager/filemanager.lua:291` (source; matched `ExportHighlights`)
- `frontend/apps/filemanager/filemanager.lua:596` (source; matched `ExportHighlights`)
- `frontend/apps/filemanager/filemanagercollection.lua:298` (source; matched `ExportHighlights`)
- `frontend/apps/filemanager/filemanagercollection.lua:1658` (source; matched `ExportHighlights`)
- `frontend/apps/filemanager/filemanagercollection.lua:1660` (source; matched `Export highlights`)
- Additional evidence retained in JSON: 27 records.

### opds

Role-weighted hits: `{"documentation": 2, "source": 120, "test": 180}`.

- `README.md:26` (documentation; matched `OPDS`)
- `README.md:34` (documentation; matched `OPDS`)
- `frontend/apps/filemanager/filemanagermenu.lua:171` (source; matched `OPDS`)
- `frontend/document/credocument.lua:1608` (source; matched `OPDS`)
- `frontend/document/credocument.lua:1614` (source; matched `OPDS`)
- Additional evidence retained in JSON: 26 records.

### opds_pse

Role-weighted hits: `{"source": 18}`.

- `plugins/opds.koplugin/opdsbrowser.lua:17` (source; matched `OPDSPSE`)
- `plugins/opds.koplugin/opdsbrowser.lua:17` (source; matched `opdspse`)
- `plugins/opds.koplugin/opdsbrowser.lua:40` (source; matched `opds-pse`)
- `plugins/opds.koplugin/opdspse.lua:15` (source; matched `OPDSPSE`)
- `plugins/opds.koplugin/opdspse.lua:18` (source; matched `OPDSPSE`)
- Additional evidence retained in JSON: 1 records.

### opensearch

Role-weighted hits: `{"source": 4, "test": 33}`.

- `plugins/opds.koplugin/opdsbrowser.lua:36` (source; matched `opensearch`)
- `plugins/opds.koplugin/opdsbrowser.lua:603` (source; matched `OpenSearch`)
- `plugins/opds.koplugin/opdsbrowser.lua:603` (source; matched `OpenSearch`)
- `spec/unit/opds_spec.lua:13` (test; matched `opensearch`)
- `spec/unit/opds_spec.lua:13` (test; matched `opensearch`)
- Additional evidence retained in JSON: 1 records.

### pdf_fixed_layout

Role-weighted hits: `{"documentation": 8, "source": 584, "test": 44}`.

- `README.md:22` (documentation; matched `DjVu`)
- `README.md:22` (documentation; matched `reflow`)
- `README.md:22` (documentation; matched `DjVu`)
- `defaults.lua:10` (source; matched `DjVu`)
- `defaults.lua:10` (source; matched `djvu`)
- Additional evidence retained in JSON: 35 records.

### permissions

Role-weighted hits: `{"configuration": 9, "documentation": 20, "source": 41, "test": 2}`.

- `.github/workflows/build.yml:5` (configuration; matched `permission`)
- `.github/workflows/build_matrix.yml:5` (configuration; matched `permission`)
- `.github/workflows/cache_cleanup.yml:5` (configuration; matched `permission`)
- `.github/workflows/doc_and_l10n_update.yml:5` (configuration; matched `permission`)
- `.github/workflows/lint.yml:5` (configuration; matched `permission`)
- Additional evidence retained in JSON: 35 records.

### progress_sync

Role-weighted hits: `{"source": 61, "test": 28}`.

- `frontend/ui/elements/common_settings_menu_table.lua:586` (source; matched `reading progress`)
- `frontend/ui/elements/screensaver_menu.lua:54` (source; matched `reading progress`)
- `frontend/ui/quickstart.lua:107` (source; matched `reading progress`)
- `frontend/ui/quickstart.lua:117` (source; matched `reading progress`)
- `metadata/en-US/full_description.txt:9` (source; matched `synchronize your reading progress`)
- Additional evidence retained in JSON: 33 records.

### reader_typography

Role-weighted hits: `{"documentation": 3, "source": 1310}`.

- `README.md:24` (documentation; matched `line spacing`)
- `README.md:24` (documentation; matched `hyphenation`)
- `defaults.lua:78` (source; matched `FONT_SIZE`)
- `defaults.lua:100` (source; matched `font size`)
- `defaults.lua:102` (source; matched `FONT_SIZE`)
- Additional evidence retained in JSON: 35 records.

### search

Role-weighted hits: `{"source": 24}`.

- `frontend/apps/reader/modules/readerdictionary.lua:330` (source; matched `fuzzy search`)
- `frontend/apps/reader/modules/readerdictionary.lua:1242` (source; matched `fuzzy search`)
- `frontend/apps/reader/modules/readerdictionary.lua:1283` (source; matched `fuzzy search`)
- `frontend/apps/reader/modules/readerkeyselection.lua:988` (source; matched `fuzzy search`)
- `frontend/apps/reader/modules/readerkeyselection.lua:1173` (source; matched `fuzzy search`)
- Additional evidence retained in JSON: 4 records.

### security

Role-weighted hits: `{"configuration": 10, "documentation": 91, "source": 2348, "test": 125}`.

- `.ci/after_success.sh:12` (source; matched `https`)
- `.ci/after_success.sh:13` (source; matched `https`)
- `.emmyrc.json:2` (source; matched `https`)
- `.github/CODEOWNERS:1` (configuration; matched `https`)
- `.github/workflows/build_matrix.yml:83` (configuration; matched `https`)
- Additional evidence retained in JSON: 35 records.

### testing

Role-weighted hits: `{"documentation": 2, "source": 20, "test": 13}`.

- `doc/Building.md:250` (documentation; matched `unit test`)
- `doc/Building.md:257` (documentation; matched `unit test`)
- `doc/Unit_tests.md:1` (test; matched `Unit Test`)
- `doc/Unit_tests.md:3` (test; matched `Unit test`)
- `frontend/apps/reader/modules/readerfooter.lua:467` (source; matched `unit test`)
- Additional evidence retained in JSON: 25 records.

### tts_audio

Role-weighted hits: `{"source": 3, "test": 1}`.

- `frontend/apps/reader/modules/readercoptlistener.lua:460` (source; matched `Chapter mark`)
- `frontend/apps/reader/modules/readerfooter.lua:1394` (source; matched `chapter mark`)
- `frontend/apps/reader/modules/readerfooter.lua:1410` (source; matched `Chapter mark`)
- `spec/unit/readerfooter_spec.lua:331` (test; matched `chapter mark`)

### watch_import

Role-weighted hits: `{"source": 1}`.

- `platform/kindle/BR-2020.02-config:2634` (source; matched `INOTIFY`)

### webdav

Role-weighted hits: `{"source": 70}`.

- `frontend/ui/quickstart.lua:256` (source; matched `Webdav`)
- `plugins/cloudstorage.koplugin/providers/webdav.lua:17` (source; matched `WebDav`)
- `plugins/cloudstorage.koplugin/providers/webdav.lua:18` (source; matched `WebDAV`)
- `plugins/cloudstorage.koplugin/providers/webdav.lua:19` (source; matched `webdav`)


## Format evidence

### cb7_cbt_cba

Role-weighted hits: `{"documentation": 1, "source": 57}`.

- `README.md:22` (documentation; matched `CBT`)
- `frontend/apps/reader/modules/readerhighlight.lua:3017` (source; matched `cbt`)
- `frontend/document/pdfdocument.lua:374` (source; matched `cbt`)
- `metadata/bg_BG/full_description.txt:4` (source; matched `CBT`)
- `metadata/cs/full_description.txt:4` (source; matched `CBT`)
- Additional evidence retained in JSON: 30 records.

### cbr

Role-weighted hits: `{"source": 7}`.

- `frontend/document/pdfdocument.lua:372` (source; matched `cbr`)
- `frontend/document/pdfdocument.lua:373` (source; matched `cbr`)
- `metadata/ca/full_description.txt:4` (source; matched `CBR`)
- `metadata/es/full_description.txt:4` (source; matched `CBR`)
- `platform/linux/koreader.metainfo.xml:181` (source; matched `CBR`)
- Additional evidence retained in JSON: 2 records.

### cbz

Role-weighted hits: `{"documentation": 1, "source": 71}`.

- `README.md:22` (documentation; matched `CBZ`)
- `frontend/apps/reader/modules/readerhighlight.lua:3017` (source; matched `cbz`)
- `frontend/document/pdfdocument.lua:375` (source; matched `cbz`)
- `frontend/document/pdfdocument.lua:376` (source; matched `cbz`)
- `metadata/bg_BG/full_description.txt:4` (source; matched `CBZ`)
- Additional evidence retained in JSON: 35 records.

### daisy

Role-weighted hits: `{"source": 7}`.

- `frontend/ui/widget/container/inputcontainer.lua:407` (source; matched `daisy`)
- `frontend/ui/widget/container/inputcontainer.lua:421` (source; matched `daisy`)
- `frontend/ui/wikipedia.lua:1439` (source; matched `.daisy`)
- `frontend/ui/wikipedia.lua:1440` (source; matched `.daisy`)
- `plugins/newsdownloader.koplugin/epubdownloadbackend.lua:643` (source; matched `.daisy`)
- Additional evidence retained in JSON: 1 records.

### djvu

Role-weighted hits: `{"documentation": 3, "source": 178, "test": 2}`.

- `README.md:22` (documentation; matched `DjVu`)
- `README.md:22` (documentation; matched `DjVu`)
- `defaults.lua:10` (source; matched `DjVu`)
- `defaults.lua:10` (source; matched `djvu`)
- `doc/config.ld:3` (documentation; matched `DJVU`)
- Additional evidence retained in JSON: 35 records.

### doc_docx

Role-weighted hits: `{"source": 17}`.

- `frontend/document/credocument.lua:377` (source; matched `.doc`)
- `frontend/document/credocument.lua:1359` (source; matched `.doc`)
- `frontend/document/documentregistry.lua:254` (source; matched `.doc`)
- `frontend/document/koptinterface.lua:767` (source; matched `.doc`)
- `frontend/document/koptinterface.lua:879` (source; matched `.doc`)
- Additional evidence retained in JSON: 9 records.

### epub

Role-weighted hits: `{"documentation": 2, "source": 218, "test": 112}`.

- `README.md:22` (documentation; matched `EPUB`)
- `doc/config.ld:3` (documentation; matched `EPUB`)
- `frontend/apps/filemanager/filemanagerbookinfo.lua:155` (source; matched `EPUB`)
- `frontend/apps/filemanager/filemanagercollection.lua:1094` (source; matched `epub`)
- `frontend/apps/filemanager/filemanagerconverter.lua:23` (source; matched `epub`)
- Additional evidence retained in JSON: 35 records.

### fb2

Role-weighted hits: `{"documentation": 2, "source": 129, "test": 10}`.

- `README.md:22` (documentation; matched `FB2`)
- `doc/config.ld:3` (documentation; matched `FB2`)
- `frontend/apps/filemanager/filemanagerutil.lua:52` (source; matched `fb2`)
- `frontend/apps/reader/modules/readerlink.lua:1455` (source; matched `FB2`)
- `frontend/apps/reader/modules/readertypeset.lua:98` (source; matched `fb2`)
- Additional evidence retained in JSON: 35 records.

### fb3

Role-weighted hits: `{"source": 9}`.

- `frontend/apps/reader/modules/readertypeset.lua:278` (source; matched `FB3`)
- `frontend/document/credocument.lua:1621` (source; matched `fb3`)
- `frontend/document/credocument.lua:1621` (source; matched `fb3`)
- `platform/linux/koreader.desktop:9` (source; matched `fb3`)
- `platform/linux/koreader.desktop:9` (source; matched `fb3`)
- Additional evidence retained in JSON: 4 records.

### html_xhtml

Role-weighted hits: `{"documentation": 3, "source": 151, "test": 5}`.

- `.luacheckrc:37` (source; matched `.html`)
- `doc/DataStore.md:69` (documentation; matched `.html`)
- `doc/DataStore.md:70` (documentation; matched `.html`)
- `doc/Development_guide.md:3` (documentation; matched `.html`)
- `frontend/apps/filemanager/filemanagerconverter.lua:91` (source; matched `.html`)
- Additional evidence retained in JSON: 35 records.

### markdown

Role-weighted hits: `{"configuration": 2, "documentation": 22, "source": 55, "test": 6}`.

- `.ci/build_cache_key_paths.txt:12` (source; matched `.md`)
- `.editorconfig:12` (source; matched `.md`)
- `.github/ISSUE_TEMPLATE/bug_report.yml:4` (configuration; matched `markdown`)
- `.github/ISSUE_TEMPLATE/bug_report.yml:122` (configuration; matched `markdown`)
- `Makefile:56` (source; matched `.md`)
- Additional evidence retained in JSON: 35 records.

### mobi_azw

Role-weighted hits: `{"documentation": 1, "source": 74}`.

- `README.md:22` (documentation; matched `Mobi`)
- `frontend/device/android/device.lua:60` (source; matched `mobi`)
- `frontend/device/android/device.lua:61` (source; matched `mobi`)
- `frontend/document/credocument.lua:1607` (source; matched `azw`)
- `frontend/document/credocument.lua:1608` (source; matched `azw`)
- Additional evidence retained in JSON: 35 records.

### odt_rtf

Role-weighted hits: `{"documentation": 1, "source": 17}`.

- `README.md:22` (documentation; matched `RTF`)
- `frontend/apps/filemanager/filemanagerutil.lua:52` (source; matched `rtf`)
- `frontend/apps/reader/modules/readertypeset.lua:189` (source; matched `rtf`)
- `frontend/document/credocument.lua:163` (source; matched `rtf`)
- `frontend/document/credocument.lua:1633` (source; matched `odt`)
- Additional evidence retained in JSON: 9 records.

### opf

Role-weighted hits: `{"source": 7}`.

- `frontend/ui/wikipedia.lua:961` (source; matched `.opf`)
- `frontend/ui/wikipedia.lua:966` (source; matched `.opf`)
- `plugins/newsdownloader.koplugin/epubdownloadbackend.lua:556` (source; matched `.opf`)
- `plugins/newsdownloader.koplugin/epubdownloadbackend.lua:562` (source; matched `.opf`)

### pdf

Role-weighted hits: `{"documentation": 3, "source": 269, "test": 120}`.

- `README.md:22` (documentation; matched `PDF`)
- `README.md:22` (documentation; matched `PDF`)
- `doc/config.ld:3` (documentation; matched `PDF`)
- `frontend/apps/filemanager/filemanagerbookinfo.lua:155` (source; matched `PDF`)
- `frontend/apps/filemanager/filemanagercollection.lua:1094` (source; matched `pdf`)
- Additional evidence retained in JSON: 35 records.

### txt

Role-weighted hits: `{"configuration": 1, "documentation": 3, "source": 42, "test": 17}`.

- `.ci/build_cache_key_paths.txt:4` (source; matched `.txt`)
- `.ci/build_matrix_generate.sh:16` (source; matched `.txt`)
- `.github/workflows/build_matrix.yml:179` (configuration; matched `.txt`)
- `.gitignore:35` (source; matched `.txt`)
- `Makefile:38` (source; matched `.txt`)
- Additional evidence retained in JSON: 35 records.

### webpub_lpf

Role-weighted hits: `{"source": 12, "test": 1}`.

- `frontend/device/devicelistener.lua:478` (source; matched `lpf`)
- `frontend/device/input.lua:1348` (source; matched `lpf`)
- `frontend/document/doccache.lua:75` (source; matched `lpf`)
- `frontend/ui/widget/buttontable.lua:234` (source; matched `lpf`)
- `frontend/ui/widget/focusmanager.lua:72` (source; matched `lpf`)
- Additional evidence retained in JSON: 8 records.
