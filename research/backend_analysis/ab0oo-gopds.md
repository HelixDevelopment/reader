# ab0oo-gopds — automated static evidence

- Inventory URL: https://github.com/ab0oo/gopds
- Origin: https://github.com/ab0oo/gopds
- Revision: `b418cabdb67479f1b5b07f1e78c2b0c5cd86b5fb`
- Tree: `07f89781eae49daca96acc355944ea7700e68cef`
- Analysis status: **automated_static_index_complete**; manual review: **pending**.
- Keyword hits are leads for code review, not proof of working support.

## Tree and languages

Tracked files: 50. Text files scanned: 49 (510325 bytes). Files skipped by size, binary classification, or sensitive-name policy remain indexed in the compressed file ledger.

- Go: 38 files
- JavaScript: 1 files
- HTML: 1 files
- CSS: 1 files

## License review

Status: heuristic_requires_legal_review. Candidate notices: MIT.
- `LICENSE`

## Feature evidence

### ai_llm

Role-weighted hits: `{"source": 40, "test": 4}`.

- `cmd/gopds/web/ui/app.js:919` (source; matched `rag`)
- `cmd/gopds/web/ui/app.js:958` (source; matched `rag`)
- `cmd/gopds/web/ui/app.js:1787` (source; matched `rag`)
- `cmd/gopds/web/ui/style.css:577` (source; matched `rag`)
- `internal/database/sqlite.go:108` (source; matched `rag`)
- Additional evidence retained in JSON: 10 records.

### annotations

Role-weighted hits: `{"documentation": 1, "source": 4, "test": 2}`.

- `README.md:215` (documentation; matched `annotation`)
- `internal/normalize/normalize.go:114` (source; matched `annotation`)
- `internal/normalize/normalize.go:221` (source; matched `annotation`)
- `internal/normalize/normalize_test.go:28` (test; matched `annotation`)
- `internal/normalize/normalize_test.go:156` (test; matched `annotation`)
- Additional evidence retained in JSON: 2 records.

### backup_restore

Role-weighted hits: `{"source": 9}`.

- `cmd/gopds/web/ui/app.js:647` (source; matched `Restore`)
- `cmd/gopds/web/ui/app.js:650` (source; matched `Restore`)
- `cmd/gopds/web/ui/app.js:675` (source; matched `Restore`)

### collections_smart

Role-weighted hits: `{"documentation": 29, "source": 839, "test": 128}`.

- `README.md:15` (documentation; matched `Series`)
- `README.md:15` (documentation; matched `series`)
- `README.md:26` (documentation; matched `Series`)
- `cmd/gopds/web/ui/app.js:13` (source; matched `series`)
- `cmd/gopds/web/ui/app.js:13` (source; matched `Series`)
- Additional evidence retained in JSON: 35 records.

### compression_cache

Role-weighted hits: `{"configuration": 6, "documentation": 6, "source": 66, "test": 4}`.

- `.github/workflows/docker-build.yml:88` (configuration; matched `cache`)
- `.github/workflows/docker-build.yml:89` (configuration; matched `cache`)
- `.github/workflows/docker-build.yml:101` (configuration; matched `cache`)
- `Dockerfile:12` (configuration; matched `cache`)
- `README.md:37` (documentation; matched `Cache`)
- Additional evidence retained in JSON: 29 records.

### conversion

Role-weighted hits: `{"source": 3}`.

- `internal/kindle/epub.go:129` (source; matched `conversion`)
- `internal/kindle/epub.go:217` (source; matched `conversion`)
- `internal/web/server.go:1824` (source; matched `conversion`)

### daisy_accessibility

Role-weighted hits: `{"source": 30}`.

- `cmd/gopds/web/ui/app.js:117` (source; matched `aria-`)
- `cmd/gopds/web/ui/app.js:117` (source; matched `aria-`)
- `cmd/gopds/web/ui/app.js:120` (source; matched `aria-`)
- `cmd/gopds/web/ui/index.html:16` (source; matched `aria-`)
- `cmd/gopds/web/ui/index.html:17` (source; matched `aria-`)
- Additional evidence retained in JSON: 1 records.

### database

Role-weighted hits: `{"configuration": 1, "documentation": 2, "source": 6, "test": 1}`.

- `.gitignore:9` (source; matched `SQLite`)
- `README.md:8` (documentation; matched `SQLite`)
- `README.md:59` (documentation; matched `SQLite`)
- `go.mod:7` (configuration; matched `sqlite`)
- `go.sum:50` (source; matched `sqlite`)
- Additional evidence retained in JSON: 5 records.

### deduplication

Role-weighted hits: `{"source": 5}`.

- `internal/scanner/scanner.go:867` (source; matched `duplicate`)
- `internal/web/server.go:1037` (source; matched `dedup`)
- `internal/web/server.go:1280` (source; matched `dedup`)
- `internal/web/server.go:1288` (source; matched `dedup`)

### deployment

Role-weighted hits: `{"configuration": 13, "documentation": 5, "source": 1}`.

- `.github/workflows/docker-build.yml:1` (configuration; matched `Docker`)
- `.github/workflows/docker-build.yml:47` (configuration; matched `Docker`)
- `.github/workflows/docker-build.yml:48` (configuration; matched `docker`)
- `README.md:94` (documentation; matched `docker`)
- `README.md:334` (documentation; matched `CI/CD`)
- Additional evidence retained in JSON: 2 records.

### local_library

Role-weighted hits: `{"source": 9}`.

- `cmd/gopds/web/ui/app.js:2211` (source; matched `in place`)
- `internal/database/sqlite.go:55` (source; matched `in place`)
- `internal/scanner/cleanup.go:23` (source; matched `in place`)
- `internal/scanner/scanner.go:200` (source; matched `in place`)
- `internal/web/download.go:16` (source; matched `filesystem`)
- Additional evidence retained in JSON: 4 records.

### metadata_enrichment

Role-weighted hits: `{"configuration": 2, "documentation": 22, "source": 478, "test": 43}`.

- `.github/workflows/docker-build.yml:50` (configuration; matched `metadata`)
- `.github/workflows/docker-build.yml:52` (configuration; matched `metadata`)
- `.gitignore:14` (source; matched `metadata`)
- `README.md:6` (documentation; matched `metadata`)
- `README.md:21` (documentation; matched `metadata`)
- Additional evidence retained in JSON: 35 records.

### metadata_review

Role-weighted hits: `{"documentation": 4, "source": 35, "test": 1}`.

- `README.md:225` (documentation; matched `Dry run`)
- `README.md:249` (documentation; matched `dry-run`)
- `README.md:253` (documentation; matched `Metadata Review`)
- `cmd/gopds/web/ui/app.js:1939` (source; matched `review-dialog" role="dialog" aria-modal="true" aria-label="Metadata`)
- `cmd/gopds/web/ui/app.js:1941` (source; matched `Metadata Review`)
- Additional evidence retained in JSON: 15 records.

### multiuser_auth

Role-weighted hits: `{"source": 1, "test": 2}`.

- `internal/scanner/cleanup_test.go:115` (test; matched `oAuth`)
- `internal/web/cleanup_test.go:138` (test; matched `oAuth`)
- `internal/web/server.go:835` (source; matched `Authentication`)

### opds

Role-weighted hits: `{"documentation": 32, "source": 112}`.

- `README.md:6` (documentation; matched `OPDS`)
- `README.md:12` (documentation; matched `OPDS`)
- `README.md:13` (documentation; matched `OPDS`)
- `internal/web/server.go:263` (source; matched `opds`)
- `internal/web/server.go:264` (source; matched `opds`)
- Additional evidence retained in JSON: 1 records.

### opensearch

Role-weighted hits: `{"source": 6}`.

- `internal/web/server.go:1966` (source; matched `OpenSearch`)
- `internal/web/server.go:1989` (source; matched `opensearch`)
- `internal/web/server.go:2007` (source; matched `opensearch`)

### permissions

Role-weighted hits: `{"configuration": 1, "documentation": 2, "source": 8}`.

- `.github/workflows/docker-build.yml:15` (configuration; matched `permission`)
- `LICENSE:5` (documentation; matched `Permission`)
- `LICENSE:12` (documentation; matched `permission`)
- `internal/web/cleanup.go:517` (source; matched `Permission`)
- `internal/web/cleanup.go:518` (source; matched `permission`)
- Additional evidence retained in JSON: 5 records.

### reader_typography

Role-weighted hits: `{"source": 41}`.

- `cmd/gopds/web/ui/app.js:29` (source; matched `font-size`)
- `cmd/gopds/web/ui/style.css:129` (source; matched `font-size`)
- `cmd/gopds/web/ui/style.css:136` (source; matched `font-size`)
- `cmd/gopds/web/ui/style.css:170` (source; matched `font-size`)

### search

Role-weighted hits: `{"source": 1}`.

- `internal/web/server.go:1991` (source; matched `Full-text search`)

### security

Role-weighted hits: `{"configuration": 3, "documentation": 7, "source": 43, "test": 4}`.

- `.github/workflows/docker-build.yml:67` (configuration; matched `secret`)
- `Dockerfile:12` (configuration; matched `certificate`)
- `Dockerfile:14` (configuration; matched `https`)
- `README.md:4` (documentation; matched `https`)
- `README.md:68` (documentation; matched `TLS`)
- Additional evidence retained in JSON: 22 records.

### testing

Role-weighted hits: `{"configuration": 1, "documentation": 2}`.

- `.github/workflows/docker-build.yml:35` (configuration; matched `go test`)
- `README.md:314` (documentation; matched `go test`)
- `README.md:340` (documentation; matched `go test`)


## Format evidence

### epub

Role-weighted hits: `{"documentation": 25, "source": 99, "test": 59}`.

- `README.md:6` (documentation; matched `EPUB `)
- `README.md:20` (documentation; matched `.epub`)
- `cmd/gopds/web/ui/app.js:117` (source; matched `EPUB `)
- `cmd/gopds/web/ui/app.js:119` (source; matched `EPUB `)
- `internal/database/sqlite.go:61` (source; matched `epub`)
- Additional evidence retained in JSON: 35 records.

### html_xhtml

Role-weighted hits: `{"source": 3, "test": 16}`.

- `internal/kindle/epub.go:85` (source; matched `.html`)
- `internal/kindle/epub.go:85` (source; matched `.xhtml`)
- `internal/kindle/epub_test.go:70` (test; matched `.xhtml`)
- `internal/kindle/epub_test.go:71` (test; matched `.xhtml`)
- `internal/scanner/cleanup_test.go:18` (test; matched `.xhtml`)
- Additional evidence retained in JSON: 5 records.

### opf

Role-weighted hits: `{"source": 9, "test": 11}`.

- `internal/kindle/epub_test.go:61` (test; matched `.opf`)
- `internal/kindle/epub_test.go:69` (test; matched `.opf`)
- `internal/scanner/library_test.go:28` (test; matched `.opf`)
- `internal/scanner/library_test.go:29` (test; matched `.opf`)
- `internal/scanner/scanner.go:226` (source; matched `package document`)
- Additional evidence retained in JSON: 3 records.
