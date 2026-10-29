# readium-readium-lcp-server — automated static evidence

- Inventory URL: https://github.com/readium/readium-lcp-server
- Origin: https://github.com/readium/readium-lcp-server
- Revision: `32570b69d02948801f161e7923648dcc0bbfb178`
- Tree: `cbf50f4a5ecd8986176a8d57d4ee5e12a472f6ae`
- Analysis status: **automated_static_index_complete**; manual review: **pending**.
- Keyword hits are leads for code review, not proof of working support.

## Tree and languages

Tracked files: 241. Text files scanned: 232 (1252602 bytes). Files skipped by size, binary classification, or sensitive-name policy remain indexed in the compressed file ledger.

- Go: 90 files
- TypeScript: 72 files
- HTML: 26 files
- SQL: 12 files
- JavaScript: 7 files
- CSS: 4 files
- SCSS: 2 files

## License review

Status: heuristic_requires_legal_review. Candidate notices: BSD-2-Clause, BSD-3-Clause, MIT.
- `LICENSE.md`
- `frontend/api/license.go`
- `frontend/manage/LICENSE-angular-quickstart`
- `frontend/manage/app/license/license-info.component.html`
- `frontend/manage/app/license/license-info.component.ts`
- `frontend/manage/app/license/license-routing.module.ts`
- `frontend/manage/app/license/license.component.html`
- `frontend/manage/app/license/license.component.ts`
- `frontend/manage/app/license/license.module.ts`
- `frontend/manage/app/license/license.service.ts`
- `frontend/manage/app/license/license.ts`
- `frontend/manage/app/lsd/license-status.ts`
- `lcpserver/api/license.go`
- `license/license.go`
- `license/license_test.go`
- `license_statuses/license_status.go`
- `license_statuses/license_statuses.go`
- `license_statuses/license_statuses_test.go`
- `lsdserver/api/license_status.go`

## Feature evidence

### ai_llm

Role-weighted hits: `{"documentation": 23, "source": 255, "test": 5}`.

- `README.md:36` (documentation; matched `rag`)
- `README.md:160` (documentation; matched `rag`)
- `README.md:221` (documentation; matched `rag`)
- `config/config.go:18` (source; matched `rag`)
- `config/config.go:18` (source; matched `rag`)
- Additional evidence retained in JSON: 35 records.

### annotations

Role-weighted hits: `{"source": 4}`.

- `frontend/manage/karma-test-shim.js:20` (source; matched `cFi`)
- `frontend/manage/karma-test-shim.js:32` (source; matched `cFi`)
- `frontend/manage/karma-test-shim.js:33` (source; matched `cFi`)

### collections_smart

Role-weighted hits: `{"source": 38}`.

- `frontend/package-lock.json:437` (source; matched `series`)
- `frontend/package-lock.json:439` (source; matched `series`)
- `frontend/package-lock.json:439` (source; matched `series`)
- `pack/w3cpackage.go:111` (source; matched `collection`)
- `rwpm/metadata.go:46` (source; matched `collection`)
- Additional evidence retained in JSON: 5 records.

### comics_manga

Role-weighted hits: `{"documentation": 1, "source": 6}`.

- `README.md:6` (documentation; matched `comic`)
- `frontend/package-lock.json:3070` (source; matched `CbZ`)
- `frontend/package-lock.json:4508` (source; matched `cB7`)
- `frontend/package-lock.json:5971` (source; matched `CBr`)

### compression_cache

Role-weighted hits: `{"documentation": 3, "source": 66, "test": 2}`.

- `README.md:247` (documentation; matched `cache`)
- `README.md:321` (documentation; matched `cache`)
- `README.md:383` (documentation; matched `cache`)
- `frontend/package-lock.json:555` (source; matched `cache`)
- `frontend/package-lock.json:798` (source; matched `etag`)
- Additional evidence retained in JSON: 4 records.

### conversion

Role-weighted hits: `{"documentation": 1, "source": 6}`.

- `README.md:6` (documentation; matched `EPUB or PDF`)
- `frontend/manage/app/publication/publication-form.component.html:20` (source; matched `EPUB, PDF`)
- `frontend/manage/app/publication/publication-form.component.html:29` (source; matched `epub, .pdf`)
- `frontend/manage/app/publication/publication-form.component.ts:66` (source; matched `epub" || extension === "pdf`)
- `frontend/manage/app/publication/publication-form.component.ts:154` (source; matched `EPUB, PDF`)
- Additional evidence retained in JSON: 2 records.

### daisy_accessibility

Role-weighted hits: `{"source": 10}`.

- `frontend/manage/app/license/license-info.component.html:50` (source; matched `aria-`)
- `frontend/manage/app/publication/publication-list.component.html:41` (source; matched `aria-`)
- `frontend/manage/app/publication/publication-list.component.html:45` (source; matched `aria-`)
- `frontend/manage/app/purchase/purchase-list.component.html:71` (source; matched `aria-`)
- `frontend/manage/app/purchase/purchase-status.component.html:14` (source; matched `aria-`)
- Additional evidence retained in JSON: 4 records.

### database

Role-weighted hits: `{"configuration": 2, "documentation": 21, "source": 93, "test": 52}`.

- `.gitignore:5` (source; matched `sqlite`)
- `LICENSE.md:6` (documentation; matched `Redis`)
- `LICENSE.md:9` (documentation; matched `Redis`)
- `LICENSE.md:12` (documentation; matched `Redis`)
- `README.md:40` (documentation; matched `SQLite`)
- Additional evidence retained in JSON: 35 records.

### deduplication

Role-weighted hits: `{"configuration": 1, "documentation": 6, "source": 84, "test": 13}`.

- `crypto/aes_cbc_test.go:31` (test; matched `sha256`)
- `crypto/aes_cbc_test.go:100` (test; matched `sha256`)
- `dbmodel/mysql_db_setup_lcpserver.sql:6` (source; matched `sha256`)
- `dbmodel/postgres_db_setup_lcpserver.sql:7` (source; matched `sha256`)
- `dbmodel/sqlite_db_setup_lcpserver.sql:6` (source; matched `sha256`)
- Additional evidence retained in JSON: 35 records.

### http_protocols

Role-weighted hits: `{"documentation": 6, "source": 3, "test": 2}`.

- `README.md:212` (documentation; matched `Quic`)
- `README.md:214` (documentation; matched `quic`)
- `crypto/aes_gcm_test.go:41` (test; matched `quic`)
- `frontend/manage/README.md:1` (documentation; matched `Quic`)
- `frontend/manage/README.md:89` (documentation; matched `quic`)
- Additional evidence retained in JSON: 5 records.

### lcp_drm

Role-weighted hits: `{"configuration": 2, "documentation": 94, "source": 264, "test": 20}`.

- `.gitignore:2` (source; matched `lcp`)
- `CONTRIBUTIONS.md:1` (source; matched `LCP`)
- `CONTRIBUTIONS.md:3` (source; matched `LCP`)
- `README.md:1` (documentation; matched `LCP`)
- `README.md:6` (documentation; matched `LCP`)
- Additional evidence retained in JSON: 35 records.

### local_library

Role-weighted hits: `{"documentation": 4, "source": 11, "test": 4}`.

- `README.md:28` (documentation; matched `in place`)
- `README.md:275` (documentation; matched `filesystem`)
- `README.md:296` (documentation; matched `filesystem`)
- `config/config.go:74` (source; matched `FileSystem`)
- `config/config.go:80` (source; matched `FileSystem`)
- Additional evidence retained in JSON: 11 records.

### metadata_enrichment

Role-weighted hits: `{"configuration": 1, "source": 109, "test": 35}`.

- `encrypt/process_encrypt.go:337` (source; matched `metadata`)
- `encrypt/process_encrypt.go:338` (source; matched `Metadata`)
- `encrypt/process_encrypt.go:339` (source; matched `Metadata`)
- `epub/epub.go:45` (source; matched `Metadata`)
- `epub/opf/opf.go:17` (source; matched `Metadata`)
- Additional evidence retained in JSON: 31 records.

### multiuser_auth

Role-weighted hits: `{"documentation": 9, "source": 24, "test": 2}`.

- `README.md:67` (documentation; matched `authentication`)
- `README.md:84` (documentation; matched `authentication`)
- `README.md:196` (documentation; matched `authentication`)
- `api/common_server.go:91` (source; matched `Authorization`)
- `api/common_server.go:135` (source; matched `Authorization`)
- Additional evidence retained in JSON: 14 records.

### network_sources

Role-weighted hits: `{"documentation": 11, "source": 49}`.

- `README.md:9` (documentation; matched `S3`)
- `README.md:36` (documentation; matched `S3`)
- `README.md:59` (documentation; matched `S3`)
- `encrypt/process_encrypt.go:104` (source; matched `S3`)
- `encrypt/process_encrypt.go:104` (source; matched `s3`)
- Additional evidence retained in JSON: 12 records.

### permissions

Role-weighted hits: `{"documentation": 1, "source": 22, "test": 12}`.

- `LICENSE.md:18` (documentation; matched `permission`)
- `authentication/md5password/makeMd5Password.go:13` (source; matched `permission`)
- `crypto/aes_cbc.go:13` (source; matched `permission`)
- `crypto/aes_cbc_test.go:13` (test; matched `permission`)
- `crypto/aes_gcm.go:13` (source; matched `permission`)
- Additional evidence retained in JSON: 30 records.

### progress_sync

Role-weighted hits: `{"test": 1}`.

- `pack/rwppackage_test.go:172` (test; matched `reading progress`)

### reader_typography

Role-weighted hits: `{"source": 19}`.

- `frontend/manage/app/components/purchases.css:16` (source; matched `font-size`)
- `frontend/manage/app/components/resource.css:25` (source; matched `font-size`)
- `frontend/manage/app/components/resource.css:30` (source; matched `font-size`)
- `frontend/manage/app/components/resource.css:35` (source; matched `font-size`)
- `frontend/manage/app/components/user.css:2` (source; matched `font-size`)
- Additional evidence retained in JSON: 7 records.

### readium

Role-weighted hits: `{"configuration": 2, "documentation": 88, "source": 548, "test": 83}`.

- `.gitignore:2` (source; matched `readium`)
- `CONTRIBUTIONS.md:1` (source; matched `Readium`)
- `CONTRIBUTIONS.md:3` (source; matched `Readium`)
- `LICENSE.md:3` (documentation; matched `Readium`)
- `README.md:1` (documentation; matched `Readium`)
- Additional evidence retained in JSON: 35 records.

### security

Role-weighted hits: `{"configuration": 1, "documentation": 54, "source": 1671, "test": 11}`.

- `README.md:17` (documentation; matched `https`)
- `README.md:24` (documentation; matched `https`)
- `README.md:32` (documentation; matched `https`)
- `api/common_server.go:15` (source; matched `cors`)
- `api/common_server.go:62` (source; matched `https`)
- Additional evidence retained in JSON: 35 records.

### testing

Role-weighted hits: `{"documentation": 4}`.

- `frontend/manage/README.md:61` (documentation; matched `unit test`)
- `frontend/manage/README.md:66` (documentation; matched `unit test`)
- `frontend/manage/README.md:70` (documentation; matched `unit test`)

### tts_audio

Role-weighted hits: `{"documentation": 2, "source": 13, "test": 1}`.

- `README.md:6` (documentation; matched `audiobook`)
- `README.md:6` (documentation; matched `audiobook`)
- `encrypt/process_encrypt.go:148` (source; matched `audiobook`)
- `encrypt/process_encrypt.go:279` (source; matched `audiobook`)
- `encrypt/process_encrypt.go:281` (source; matched `audiobook`)
- Additional evidence retained in JSON: 9 records.


## Format evidence

### audiobook

Role-weighted hits: `{"documentation": 1, "source": 5}`.

- `README.md:6` (documentation; matched `.audiobook`)
- `encrypt/process_encrypt.go:148` (source; matched `.audiobook`)
- `encrypt/process_encrypt.go:279` (source; matched `.audiobook`)
- `frontend/manage/app/publication/publication-form.component.html:29` (source; matched `.audiobook`)

### divina

Role-weighted hits: `{"documentation": 1, "source": 7}`.

- `README.md:6` (documentation; matched `.divina`)
- `encrypt/process_encrypt.go:148` (source; matched `.divina`)
- `encrypt/process_encrypt.go:282` (source; matched `.divina`)
- `frontend/manage/app/publication/publication-form.component.html:29` (source; matched `.divina`)
- `frontend/manage/app/publication/publication-form.component.ts:67` (source; matched `divina`)

### epub

Role-weighted hits: `{"documentation": 5, "source": 91, "test": 23}`.

- `README.md:6` (documentation; matched `EPUB `)
- `dbmodel/mysql_db_setup_lcpserver.sql:7` (source; matched `epub`)
- `dbmodel/postgres_db_setup_lcpserver.sql:8` (source; matched `epub`)
- `dbmodel/sqlite_db_setup_lcpserver.sql:7` (source; matched `epub`)
- `encrypt/process_encrypt.go:21` (source; matched `epub`)
- Additional evidence retained in JSON: 35 records.

### html_xhtml

Role-weighted hits: `{"configuration": 4, "documentation": 3, "source": 33, "test": 16}`.

- `README.md:329` (documentation; matched `.html`)
- `epub/epub.go:23` (source; matched `XHTML`)
- `epub/epub.go:23` (source; matched `xhtml`)
- `epub/reader_test.go:56` (test; matched `.xhtml`)
- `epub/reader_test.go:66` (test; matched `XHTML`)
- Additional evidence retained in JSON: 35 records.

### lcp_protected

Role-weighted hits: `{"configuration": 2, "documentation": 92, "source": 255, "test": 20}`.

- `.gitignore:2` (source; matched `lcp`)
- `CONTRIBUTIONS.md:1` (source; matched `LCP`)
- `CONTRIBUTIONS.md:3` (source; matched `LCP`)
- `README.md:1` (documentation; matched `LCP`)
- `README.md:6` (documentation; matched `LCP`)
- Additional evidence retained in JSON: 35 records.

### markdown

Role-weighted hits: `{"source": 1}`.

- `frontend/manage/.editorconfig:21` (source; matched `.md`)

### opf

Role-weighted hits: `{"test": 4}`.

- `epub/reader_test.go:56` (test; matched `.opf`)
- `epub/writer_test.go:39` (test; matched `.opf`)
- `epub/writer_test.go:69` (test; matched `.opf`)

### pdf

Role-weighted hits: `{"documentation": 1, "source": 45, "test": 2}`.

- `README.md:6` (documentation; matched `PDF`)
- `encrypt/process_encrypt.go:144` (source; matched `.pdf`)
- `encrypt/process_encrypt.go:276` (source; matched `.pdf`)
- `frontend/manage/app/publication/publication-form.component.html:20` (source; matched `PDF`)
- `frontend/manage/app/publication/publication-form.component.html:29` (source; matched `.pdf`)
- Additional evidence retained in JSON: 12 records.

### txt

Role-weighted hits: `{"documentation": 1, "source": 1, "test": 2}`.

- `frontend/manage/README.md:106` (documentation; matched `.txt`)
- `frontend/manage/protractor.config.js:69` (source; matched `.txt`)
- `pack/rwppackage_test.go:87` (test; matched `.txt`)
- `pack/rwppackage_test.go:99` (test; matched `.txt`)

### webpub_lpf

Role-weighted hits: `{"documentation": 6, "source": 86, "test": 22}`.

- `README.md:6` (documentation; matched `LPF`)
- `encrypt/process_encrypt.go:146` (source; matched `.lpf`)
- `encrypt/process_encrypt.go:147` (source; matched `LPF`)
- `frontend/api/common.go:35` (source; matched `webpub`)
- `frontend/api/common.go:44` (source; matched `webpub`)
- Additional evidence retained in JSON: 34 records.
