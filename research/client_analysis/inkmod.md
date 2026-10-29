# inkmod — automated static evidence

- Inventory URL: https://github.com/inkmod/inkmod
- Origin: https://github.com/alpzoloto-sudo/inkmod
- Revision: `51a381bd24076263c02782d7511e1bd54e74715b`
- Tree: `4330f95bb85877e910bd4373b0ce0d148bd2d1a1`
- Analysis status: **automated_static_index_complete**; manual review: **pending**.
- Keyword hits are leads for code review, not proof of working support.

## Tree and languages

Tracked files: 1175. Text files scanned: 208 (3891332 bytes). Files skipped by size, binary classification, or sensitive-name policy remain indexed in the compressed file ledger.

- C/C++ header: 524 files
- C++: 289 files
- Python: 44 files
- C: 36 files
- CSS: 13 files
- JavaScript: 11 files
- Shell: 9 files
- HTML: 8 files
- XML: 2 files
- PowerShell: 1 files

## License review

Status: heuristic_requires_legal_review. Candidate notices: MIT.
- `LICENSE`
- `freeink-sdk/LICENSE`
- `freeink-sdk/NOTICE`
- `freeink-sdk/libs/book/ContentProtection/third_party/miniz/LICENSE`
- `freeink-sdk/libs/book/FreeInkBook/third_party/expat/COPYING`
- `freeink-sdk/libs/book/FreeInkBook/third_party/hyphen-patterns/LICENSE`
- `freeink-sdk/libs/book/FreeInkBook/third_party/libunibreak/LICENCE`
- `freeink-sdk/libs/book/FreeInkBook/third_party/miniz/LICENSE`
- `freeink-sdk/libs/book/FreeInkBook/third_party/pngle/LICENSE`

## Feature evidence

### ai_llm

Role-weighted hits: `{"documentation": 59, "source": 350, "test": 79}`.

- `.clang-format:134` (source; matched `rag`)
- `.clang-format:134` (source; matched `rag`)
- `CHANGELOG.md:52` (documentation; matched `rag`)
- `CHANGELOG.md:57` (documentation; matched `rag`)
- `CHANGELOG.md:100` (documentation; matched `rag`)
- Additional evidence retained in JSON: 35 records.

### annotations

Role-weighted hits: `{"documentation": 14, "source": 46}`.

- `.clang-format:118` (source; matched `Annotation`)
- `CHANGELOG.md:49` (documentation; matched `bookmark`)
- `CHANGELOG.md:90` (documentation; matched `annotation`)
- `CHANGELOG.md:153` (documentation; matched `highlight`)
- `README.md:42` (documentation; matched `annotation`)
- Additional evidence retained in JSON: 20 records.

### backup_restore

Role-weighted hits: `{"documentation": 28, "source": 12, "test": 1}`.

- `CHANGELOG.md:13` (documentation; matched `backup`)
- `CHANGELOG.md:42` (documentation; matched `Restore`)
- `CHANGELOG.md:51` (documentation; matched `restore`)
- `RELEASE_NOTES_1.1.9.md:5` (source; matched `restore`)
- `USER_GUIDE.md:515` (source; matched `restore`)
- Additional evidence retained in JSON: 14 records.

### collections_smart

Role-weighted hits: `{"documentation": 1, "source": 3, "test": 1}`.

- `CHANGELOG.md:270` (documentation; matched `collection`)
- `RELEASE_NOTES_1.1.8.md:25` (source; matched `series`)
- `RELEASE_NOTES_1.1.9.md:13` (source; matched `collection`)
- `freeink-sdk/libs/book/FreeInkBook/test/fixtures/fonts/DEJAVU-LICENSE:178` (test; matched `collection`)
- `freeink-sdk/libs/book/FreeInkBook/third_party/expat/expat.h:786` (source; matched `series`)

### comics_manga

Role-weighted hits: `{"source": 3}`.

- `.clang-format:208` (source; matched `CBr`)
- `freeink-sdk/libs/book/FreeInkBook/third_party/expat/xmlparse.c:39` (source; matched `cBr`)
- `freeink-sdk/libs/book/FreeInkBook/third_party/expat/xmltok.c:25` (source; matched `cBr`)

### compression_cache

Role-weighted hits: `{"documentation": 70, "source": 186, "test": 165}`.

- `.gitattributes:3` (source; matched `gzip`)
- `.gitattributes:5` (source; matched `gzip`)
- `.gitignore:7` (source; matched `cache`)
- `CHANGELOG.md:70` (documentation; matched `cache`)
- `CHANGELOG.md:142` (documentation; matched `cache`)
- Additional evidence retained in JSON: 35 records.

### conversion

Role-weighted hits: `{"configuration": 2, "documentation": 18, "source": 24, "test": 2}`.

- `.github/release-notes/v1.1.10.md:16` (configuration; matched `EPUB/FB2, TXT`)
- `.github/release-notes/v1.1.10.md:23` (configuration; matched `EPUB/FB2 или TXT. Для TXT`)
- `CHANGELOG.md:16` (documentation; matched `EPUB/FB2 reader orientation; TXT`)
- `CHANGELOG.md:70` (documentation; matched `EPUB, FB2`)
- `CHANGELOG.md:74` (documentation; matched `EPUB/FB2`)
- Additional evidence retained in JSON: 19 records.

### daisy_accessibility

Role-weighted hits: `{"documentation": 1, "test": 2}`.

- `CHANGELOG.md:276` (documentation; matched `accessibility`)
- `freeink-sdk/libs/book/FreeInkBook/test/fixtures/minimal/OEBPS/toc.ncx:2` (test; matched `daisy`)
- `freeink-sdk/libs/book/FreeInkBook/test/fixtures/ncxonly/OEBPS/toc.ncx:2` (test; matched `daisy`)

### database

Role-weighted hits: `{"documentation": 1, "source": 3}`.

- `CHANGELOG.md:227` (documentation; matched `redis`)
- `THIRD_PARTY.md:9` (source; matched `redis`)
- `THIRD_PARTY.md:15` (source; matched `redis`)
- `freeink-sdk/libs/book/FreeInkBook/third_party/libunibreak/LICENCE:11` (source; matched `redis`)

### deduplication

Role-weighted hits: `{"documentation": 11, "source": 48, "test": 2}`.

- `CHANGELOG.md:110` (documentation; matched `duplicate`)
- `USER_GUIDE.md:5` (source; matched `duplicate`)
- `freeink-sdk/README.md:134` (documentation; matched `fingerprint`)
- `freeink-sdk/README.md:176` (documentation; matched `fingerprint`)
- `freeink-sdk/README.md:264` (documentation; matched `fingerprint`)
- Additional evidence retained in JSON: 35 records.

### deployment

Role-weighted hits: `{"source": 1}`.

- `.gitattributes:6` (source; matched `GitHub Actions`)

### dictionary_language

Role-weighted hits: `{"documentation": 20, "source": 25, "test": 1}`.

- `CHANGELOG.md:101` (documentation; matched `Dictionary`)
- `CHANGELOG.md:188` (documentation; matched `dictionary`)
- `CHANGELOG.md:188` (documentation; matched `StarDict`)
- `README.md:53` (documentation; matched `dictionary`)
- `README.md:57` (documentation; matched `dictionary`)
- Additional evidence retained in JSON: 15 records.

### http_protocols

Role-weighted hits: `{"documentation": 8, "source": 7, "test": 1}`.

- `CHANGELOG.md:16` (documentation; matched `quic`)
- `CHANGELOG.md:68` (documentation; matched `Quic`)
- `CHANGELOG.md:138` (documentation; matched `Quic`)
- `README.md:93` (documentation; matched `Quic`)
- `README.md:96` (documentation; matched `Quic`)
- Additional evidence retained in JSON: 8 records.

### koreader_sync

Role-weighted hits: `{"documentation": 5, "source": 2}`.

- `CHANGELOG.md:68` (documentation; matched `Quick Sync`)
- `README.md:96` (documentation; matched `Quick Sync`)
- `README.md:119` (documentation; matched `KOReader sync`)
- `RELEASE_NOTES_1.1.8.md:40` (source; matched `Quick Sync`)
- `freeink-sdk/README.md:317` (documentation; matched `KOSync`)
- Additional evidence retained in JSON: 2 records.

### lcp_drm

Role-weighted hits: `{"documentation": 2, "source": 16}`.

- `freeink-sdk/docs/freeink-book.md:97` (documentation; matched `DRM`)
- `freeink-sdk/docs/freeink-book.md:251` (documentation; matched `DRM`)
- `freeink-sdk/libs/book/ContentProtection/include/ProtectedBook.h:8` (source; matched `rights.xml`)
- `freeink-sdk/libs/book/ContentProtection/include/ProtectedBook.h:35` (source; matched `rights.xml`)
- `freeink-sdk/libs/book/ContentProtection/include/ProtectedBook.h:38` (source; matched `rights.xml`)
- Additional evidence retained in JSON: 10 records.

### local_library

Role-weighted hits: `{"documentation": 11, "source": 39, "test": 1}`.

- `CHANGELOG.md:49` (documentation; matched `file browser`)
- `CHANGELOG.md:58` (documentation; matched `File Browser`)
- `CHANGELOG.md:76` (documentation; matched `file browser`)
- `README.md:51` (documentation; matched `file browser`)
- `README.md:64` (documentation; matched `File browser`)
- Additional evidence retained in JSON: 31 records.

### metadata_enrichment

Role-weighted hits: `{"configuration": 1, "documentation": 10, "source": 63, "test": 20}`.

- `.github/workflows/c-cpp.yml:29` (configuration; matched `metadata`)
- `CHANGELOG.md:184` (documentation; matched `metadata`)
- `CHANGELOG.md:185` (documentation; matched `metadata`)
- `CHANGELOG.md:194` (documentation; matched `metadata`)
- `README.md:180` (documentation; matched `metadata`)
- Additional evidence retained in JSON: 35 records.

### multiuser_auth

Role-weighted hits: `{"source": 1, "test": 3}`.

- `freeink-sdk/libs/book/ContentProtection/src/WolfsslCrypto.cpp:11` (source; matched `authentication`)
- `freeink-sdk/libs/book/FreeInkBook/test/fixtures/fonts/DEJAVU-LICENSE:48` (test; matched `authorization`)
- `freeink-sdk/libs/book/FreeInkBook/test/fixtures/fonts/DEJAVU-LICENSE:96` (test; matched `authorization`)
- `freeink-sdk/libs/book/FreeInkBook/test/fixtures/fonts/DEJAVU-LICENSE:158` (test; matched `authorization`)

### network_sources

Role-weighted hits: `{"documentation": 46, "source": 7}`.

- `CHANGELOG.md:47` (documentation; matched `S3`)
- `README.md:29` (documentation; matched `S3`)
- `README.md:104` (documentation; matched `S3`)
- `RELEASE_NOTES_1.1.8.md:7` (source; matched `S3`)
- `RELEASE_NOTES_1.1.8.md:36` (source; matched `S3`)
- Additional evidence retained in JSON: 20 records.

### opds

Role-weighted hits: `{"documentation": 6, "source": 3}`.

- `CHANGELOG.md:49` (documentation; matched `OPDS`)
- `CHANGELOG.md:69` (documentation; matched `OPDS`)
- `README.md:17` (documentation; matched `OPDS`)
- `README.md:51` (documentation; matched `OPDS`)
- `README.md:66` (documentation; matched `OPDS`)
- Additional evidence retained in JSON: 3 records.

### pdf_fixed_layout

Role-weighted hits: `{"source": 4, "test": 1}`.

- `.clang-format:259` (source; matched `Reflow`)
- `THIRD_PARTY.md:25` (source; matched `MuPDF`)
- `freeink-sdk/libs/book/FreeInkBook/include/css/Css.h:5` (source; matched `reflow`)
- `freeink-sdk/libs/book/FreeInkBook/src/layout/ChapterLayout.cpp:37` (source; matched `fixed layout`)
- `freeink-sdk/libs/book/FreeInkBook/test/host/test_layout.cpp:274` (test; matched `fixed layout`)

### permissions

Role-weighted hits: `{"configuration": 1, "documentation": 11, "source": 63, "test": 6}`.

- `.github/workflows/release.yml:18` (configuration; matched `permission`)
- `LICENSE:5` (documentation; matched `Permission`)
- `LICENSE:12` (documentation; matched `permission`)
- `README.md:65` (documentation; matched `library again. This behaviour is share`)
- `freeink-sdk/LICENSE:5` (documentation; matched `Permission`)
- Additional evidence retained in JSON: 35 records.

### progress_sync

Role-weighted hits: `{"documentation": 2, "source": 3}`.

- `CHANGELOG.md:68` (documentation; matched `Sync for KOReader progress`)
- `README.md:96` (documentation; matched `progress sync`)
- `USER_GUIDE.md:94` (source; matched `reading progress`)
- `USER_GUIDE.md:365` (source; matched `reading progress`)
- `freeink-sdk/libs/book/FreeInkBook/include/cache/PageCache.h:57` (source; matched `sync-style progress`)

### reader_typography

Role-weighted hits: `{"documentation": 61, "source": 48, "test": 34}`.

- `CHANGELOG.md:41` (documentation; matched `custom font`)
- `CHANGELOG.md:75` (documentation; matched `theme`)
- `CHANGELOG.md:81` (documentation; matched `theme`)
- `README.md:17` (documentation; matched `custom font`)
- `README.md:42` (documentation; matched `typography`)
- Additional evidence retained in JSON: 35 records.

### search

Role-weighted hits: `{"documentation": 1}`.

- `CHANGELOG.md:188` (documentation; matched `search index`)

### security

Role-weighted hits: `{"configuration": 4, "documentation": 53, "source": 84}`.

- `.clang-format:209` (source; matched `CSp`)
- `.clang-format:210` (source; matched `CSp`)
- `.github/FUNDING.yml:3` (configuration; matched `https`)
- `.github/release-notes/v1.1.10.md:43` (configuration; matched `https`)
- `.github/release-notes/v1.1.10.md:43` (configuration; matched `https`)
- Additional evidence retained in JSON: 35 records.

### testing

Role-weighted hits: `{"documentation": 1, "source": 2, "test": 2}`.

- `freeink-sdk/docs/freeink-ui.md:20` (documentation; matched `unit test`)
- `freeink-sdk/libs/book/FreeInkBook/include/BookStorage.h:7` (source; matched `unit test`)
- `freeink-sdk/libs/book/FreeInkBook/include/FreeInkBook.h:19` (source; matched `unit test`)
- `freeink-sdk/libs/book/FreeInkBook/test/host/test_catalog.cpp:1` (test; matched `unit test`)
- `freeink-sdk/libs/book/FreeInkBook/test/host/test_freeinkbook.cpp:1` (test; matched `unit test`)

### tts_audio

Role-weighted hits: `{"source": 1}`.

- `freeink-sdk/libs/book/FreeInkBook/src/layout/ChapterLayout.cpp:606` (source; matched `chapter mark`)


## Format evidence

### daisy

Role-weighted hits: `{"test": 2}`.

- `freeink-sdk/libs/book/FreeInkBook/test/fixtures/minimal/OEBPS/toc.ncx:2` (test; matched `.daisy`)
- `freeink-sdk/libs/book/FreeInkBook/test/fixtures/ncxonly/OEBPS/toc.ncx:2` (test; matched `.daisy`)

### epub

Role-weighted hits: `{"configuration": 2, "documentation": 42, "source": 92, "test": 64}`.

- `.github/release-notes/v1.1.10.md:16` (configuration; matched `EPUB`)
- `.github/release-notes/v1.1.10.md:23` (configuration; matched `EPUB`)
- `CHANGELOG.md:16` (documentation; matched `EPUB`)
- `CHANGELOG.md:70` (documentation; matched `EPUB`)
- `README.md:17` (documentation; matched `EPUB`)
- Additional evidence retained in JSON: 35 records.

### fb2

Role-weighted hits: `{"configuration": 6, "documentation": 45, "source": 19}`.

- `.github/release-notes/v1.1.10.md:16` (configuration; matched `FB2`)
- `.github/release-notes/v1.1.10.md:23` (configuration; matched `FB2`)
- `CHANGELOG.md:16` (documentation; matched `FB2`)
- `CHANGELOG.md:23` (documentation; matched `FB2`)
- `README.md:17` (documentation; matched `FB2`)
- Additional evidence retained in JSON: 5 records.

### html_xhtml

Role-weighted hits: `{"documentation": 3, "source": 4, "test": 95}`.

- `CHANGELOG.md:185` (documentation; matched `XHTML`)
- `freeink-sdk/README.md:382` (documentation; matched `.html`)
- `freeink-sdk/README.md:384` (documentation; matched `.html`)
- `freeink-sdk/libs/book/FreeInkBook/include/layout/ChapterLayout.h:5` (source; matched `XHTML`)
- `freeink-sdk/libs/book/FreeInkBook/include/layout/ChapterLayout.h:119` (source; matched `XHTML`)
- Additional evidence retained in JSON: 29 records.

### lcp_protected

Role-weighted hits: `{"source": 1}`.

- `freeink-sdk/libs/book/ContentProtection/library.json:5` (source; matched `lcp`)

### markdown

Role-weighted hits: `{"configuration": 1, "documentation": 20, "source": 2}`.

- `.github/workflows/release.yml:113` (configuration; matched `.md`)
- `CHANGELOG.md:119` (documentation; matched `.md`)
- `CHANGELOG.md:133` (documentation; matched `.md`)
- `README.md:106` (documentation; matched `.md`)
- `README.md:106` (documentation; matched `.md`)
- Additional evidence retained in JSON: 6 records.

### opf

Role-weighted hits: `{"source": 3, "test": 6}`.

- `freeink-sdk/libs/book/FreeInkBook/include/BookCatalog.h:65` (source; matched `package document`)
- `freeink-sdk/libs/book/FreeInkBook/include/epub/XmlSax.h:48` (source; matched `package document`)
- `freeink-sdk/libs/book/FreeInkBook/include/text/EntityFilter.h:14` (source; matched `Package document`)
- `freeink-sdk/libs/book/FreeInkBook/test/fixtures/gen_omnibus.py:17` (test; matched `.opf`)
- `freeink-sdk/libs/book/FreeInkBook/test/fixtures/gen_omnibus.py:81` (test; matched `.opf`)
- Additional evidence retained in JSON: 4 records.

### pdf

Role-weighted hits: `{"source": 1}`.

- `freeink-sdk/libs/book/FreeInkBook/src/epub/XmlUtil.h:111` (source; matched `pdf`)

### txt

Role-weighted hits: `{"configuration": 4, "documentation": 7, "source": 8, "test": 11}`.

- `.github/workflows/c-cpp.yml:27` (configuration; matched `.txt`)
- `.github/workflows/release.yml:10` (configuration; matched `.txt`)
- `.github/workflows/release.yml:39` (configuration; matched `.txt`)
- `CHANGELOG.md:141` (documentation; matched `.txt`)
- `README.md:101` (documentation; matched `.txt`)
- Additional evidence retained in JSON: 17 records.

### xtc_xtch

Role-weighted hits: `{"configuration": 1, "documentation": 8, "source": 2}`.

- `.github/release-notes/v1.1.10.md:16` (configuration; matched `XTC`)
- `CHANGELOG.md:231` (documentation; matched `XTC`)
- `CHANGELOG.md:231` (documentation; matched `XTCH`)
- `README.md:39` (documentation; matched `XTC`)
- `README.md:39` (documentation; matched `XTCH`)
- Additional evidence retained in JSON: 2 records.
