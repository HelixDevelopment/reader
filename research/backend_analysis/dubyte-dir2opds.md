# dubyte-dir2opds — automated static evidence

- Inventory URL: https://github.com/dubyte/dir2opds
- Origin: https://github.com/dubyte/dir2opds
- Revision: `9f40bc1fd587ec1c1ddbab9888529effa471ea79`
- Tree: `0860254acdd81f71f5437ea532ab10adcadd6428`
- Analysis status: **automated_static_index_complete**; manual review: **pending**.
- Keyword hits are leads for code review, not proof of working support.

## Tree and languages

Tracked files: 51. Text files scanned: 47 (212396 bytes). Files skipped by size, binary classification, or sensitive-name policy remain indexed in the compressed file ledger.

- Go: 17 files
- Shell: 3 files
- HTML: 1 files
- XML: 1 files

## License review

Status: heuristic_requires_legal_review. Candidate notices: AGPL-3.0, GPL-3.0, LGPL.
- `LICENSE`

## Feature evidence

### ai_llm

Role-weighted hits: `{"documentation": 6, "source": 15, "test": 11}`.

- `.goreleaser.yml:21` (source; matched `rag`)
- `AGENTS.md:29` (source; matched `rag`)
- `LICENSE:238` (documentation; matched `rag`)
- `LICENSE:301` (documentation; matched `rag`)
- `LICENSE:413` (documentation; matched `rag`)
- Additional evidence retained in JSON: 8 records.

### annotations

Role-weighted hits: `{"documentation": 2, "source": 4, "test": 7}`.

- `CHANGELOG.md:28` (documentation; matched `annotation`)
- `README.md:304` (documentation; matched `annotation`)
- `internal/service/internal_test.go:103` (test; matched `annotation`)
- `internal/service/internal_test.go:105` (test; matched `annotation`)
- `internal/service/internal_test.go:105` (test; matched `annotation`)
- Additional evidence retained in JSON: 6 records.

### backup_restore

Role-weighted hits: `{"source": 2}`.

- `.goreleaser.yml:46` (source; matched `snapshot`)
- `internal/service/service.go:1240` (source; matched `backup`)

### collections_smart

Role-weighted hits: `{"documentation": 9, "source": 34, "test": 26}`.

- `CHANGELOG.md:35` (documentation; matched `series`)
- `CHANGELOG.md:35` (documentation; matched `series`)
- `CHANGELOG.md:35` (documentation; matched `series`)
- `README.md:7` (documentation; matched `collection`)
- `README.md:38` (documentation; matched `collection`)
- Additional evidence retained in JSON: 13 records.

### comics_manga

Role-weighted hits: `{"documentation": 5, "source": 4}`.

- `CHANGELOG.md:320` (documentation; matched `comic`)
- `README.md:303` (documentation; matched `comic`)
- `docs/index.html:222` (documentation; matched `CBZ`)
- `docs/index.html:222` (documentation; matched `CBR`)
- `docs/index.html:222` (documentation; matched `comic`)
- Additional evidence retained in JSON: 3 records.

### compression_cache

Role-weighted hits: `{"documentation": 64, "source": 92, "test": 139}`.

- `AGENTS.md:156` (source; matched `gzip`)
- `AGENTS.md:156` (source; matched `Gzip`)
- `AGENTS.md:157` (source; matched `gzip`)
- `CHANGELOG.md:14` (documentation; matched `Last-Modified`)
- `CHANGELOG.md:14` (documentation; matched `Last-Modified`)
- Additional evidence retained in JSON: 35 records.

### conversion

Role-weighted hits: `{"documentation": 7, "source": 3}`.

- `CHANGELOG.md:104` (documentation; matched `EPUB and PDF`)
- `README.md:37` (documentation; matched `EPUB, PDF`)
- `README.md:46` (documentation; matched `EPUB/PDF`)
- `README.md:54` (documentation; matched `EPUB, PDF`)
- `docs/index.html:8` (documentation; matched `EPUB, PDF`)
- Additional evidence retained in JSON: 4 records.

### database

Role-weighted hits: `{"documentation": 3, "source": 1}`.

- `LICENSE:627` (documentation; matched `redis`)
- `LICENSE:637` (documentation; matched `redis`)
- `LICENSE:657` (documentation; matched `redis`)
- `main.go:4` (source; matched `redis`)

### deduplication

Role-weighted hits: `{"source": 5}`.

- `.goreleaser.yml:44` (source; matched `checksum`)
- `.goreleaser.yml:45` (source; matched `checksum`)
- `internal/service/AGENTS.md:69` (source; matched `SHA-256`)
- `internal/service/service.go:9` (source; matched `sha256`)
- `internal/service/service.go:203` (source; matched `sha256`)

### deployment

Role-weighted hits: `{"configuration": 3, "documentation": 7, "source": 17}`.

- `.github/workflows/release.yml:27` (configuration; matched `Docker`)
- `.github/workflows/release.yml:28` (configuration; matched `docker`)
- `.github/workflows/release.yml:31` (configuration; matched `docker`)
- `.goreleaser.yml:55` (source; matched `docker`)
- `.goreleaser.yml:59` (source; matched `docker`)
- Additional evidence retained in JSON: 16 records.

### http_protocols

Role-weighted hits: `{"documentation": 5, "test": 6}`.

- `README.md:15` (documentation; matched `Quic`)
- `README.md:15` (documentation; matched `quic`)
- `README.md:58` (documentation; matched `Quic`)
- `docs/index.html:167` (documentation; matched `quic`)
- `docs/index.html:173` (documentation; matched `Quic`)
- Additional evidence retained in JSON: 3 records.

### lcp_drm

Role-weighted hits: `{"documentation": 3}`.

- `LICENSE:6` (documentation; matched `license document`)
- `LICENSE:392` (documentation; matched `license document`)
- `LICENSE:395` (documentation; matched `license document`)

### local_library

Role-weighted hits: `{"documentation": 4, "source": 6}`.

- `AGENTS.md:129` (source; matched `filesystem`)
- `CHANGELOG.md:93` (documentation; matched `filesystem`)
- `CHANGELOG.md:116` (documentation; matched `filesystem`)
- `README.md:45` (documentation; matched `filesystem`)
- `docs/index.html:186` (documentation; matched `filesystem`)
- Additional evidence retained in JSON: 5 records.

### metadata_enrichment

Role-weighted hits: `{"documentation": 22, "source": 58, "test": 10}`.

- `AGENTS.md:76` (source; matched `Metadata`)
- `AGENTS.md:177` (source; matched `metadata`)
- `AGENTS.md:194` (source; matched `Metadata`)
- `CHANGELOG.md:28` (documentation; matched `metadata`)
- `CHANGELOG.md:34` (documentation; matched `metadata`)
- Additional evidence retained in JSON: 21 records.

### metadata_review

Role-weighted hits: `{"documentation": 1}`.

- `CHANGELOG.md:34` (documentation; matched `metadata is now extracted and emitted as `<atom:summary>` in acquisition feeds, enabling richer book`)

### multiuser_auth

Role-weighted hits: `{"documentation": 1, "source": 4}`.

- `INSTALLATION.md:100` (source; matched `multi-user`)
- `LICENSE:311` (documentation; matched `authorization`)
- `files/illumos/manifest/dir2opds.xml:46` (source; matched `multi-user`)
- `files/illumos/manifest/dir2opds.xml:49` (source; matched `multi-user`)
- `files/linux/systemd/dir2opds.service:15` (source; matched `multi-user`)

### opds

Role-weighted hits: `{"documentation": 55, "source": 148, "test": 86}`.

- `.goreleaser.yml:74` (source; matched `opds`)
- `.goreleaser.yml:95` (source; matched `opds`)
- `AGENTS.md:69` (source; matched `opds`)
- `AGENTS.md:74` (source; matched `opds`)
- `AGENTS.md:153` (source; matched `OPDS`)
- Additional evidence retained in JSON: 35 records.

### opensearch

Role-weighted hits: `{"documentation": 3, "source": 18, "test": 3}`.

- `CHANGELOG.md:84` (documentation; matched `OpenSearch`)
- `CHANGELOG.md:102` (documentation; matched `OpenSearch`)
- `README.md:47` (documentation; matched `OpenSearch`)
- `internal/service/AGENTS.md:9` (source; matched `OpenSearch`)
- `internal/service/service.go:886` (source; matched `OpenSearch`)
- Additional evidence retained in JSON: 13 records.

### permissions

Role-weighted hits: `{"configuration": 1, "documentation": 20}`.

- `.github/workflows/release.yml:8` (configuration; matched `permission`)
- `LICENSE:42` (documentation; matched `permission`)
- `LICENSE:85` (documentation; matched `permission`)
- `LICENSE:93` (documentation; matched `permission`)

### reader_typography

Role-weighted hits: `{"documentation": 10, "source": 7}`.

- `docs/index.html:31` (documentation; matched `font-size`)
- `docs/index.html:57` (documentation; matched `line-height`)
- `docs/index.html:63` (documentation; matched `font-size`)
- `internal/service/html.go:34` (source; matched `line-height`)
- `internal/service/html.go:53` (source; matched `font-size`)
- Additional evidence retained in JSON: 1 records.

### security

Role-weighted hits: `{"configuration": 2, "documentation": 31, "source": 25, "test": 5}`.

- `.github/workflows/release.yml:35` (configuration; matched `secret`)
- `.github/workflows/release.yml:47` (configuration; matched `secret`)
- `.goreleaser.yml:4` (source; matched `https`)
- `AGENTS.md:127` (source; matched `path traversal`)
- `AGENTS.md:128` (source; matched `Symlink`)
- Additional evidence retained in JSON: 32 records.

### testing

Role-weighted hits: `{"configuration": 2, "source": 5}`.

- `.github/workflows/go.yml:25` (configuration; matched `go test`)
- `.github/workflows/release.yml:38` (configuration; matched `go test`)
- `AGENTS.md:22` (source; matched `go test`)
- `AGENTS.md:23` (source; matched `go test`)
- `AGENTS.md:26` (source; matched `go test`)


## Format evidence

### cbr

Role-weighted hits: `{"documentation": 1, "source": 2}`.

- `docs/index.html:222` (documentation; matched `CBR`)
- `internal/service/service.go:36` (source; matched `.cbr`)
- `internal/service/service.go:36` (source; matched `cbr`)

### cbz

Role-weighted hits: `{"documentation": 1, "source": 2}`.

- `docs/index.html:222` (documentation; matched `CBZ`)
- `internal/service/service.go:35` (source; matched `.cbz`)
- `internal/service/service.go:35` (source; matched `cbz`)

### epub

Role-weighted hits: `{"documentation": 17, "source": 13, "test": 35}`.

- `CHANGELOG.md:16` (documentation; matched `EPUB `)
- `CHANGELOG.md:34` (documentation; matched `EPUB `)
- `README.md:37` (documentation; matched `EPUB`)
- `README.md:46` (documentation; matched `EPUB`)
- `docs/index.html:8` (documentation; matched `EPUB`)
- Additional evidence retained in JSON: 11 records.

### fb2

Role-weighted hits: `{"documentation": 6, "source": 15, "test": 26}`.

- `CHANGELOG.md:28` (documentation; matched `FB2`)
- `CHANGELOG.md:28` (documentation; matched `.fb2`)
- `README.md:304` (documentation; matched `FB2`)
- `docs/index.html:223` (documentation; matched `FB2`)
- `internal/service/internal_test.go:95` (test; matched `FB2`)
- Additional evidence retained in JSON: 9 records.

### html_xhtml

Role-weighted hits: `{"documentation": 2, "source": 2, "test": 1}`.

- `CHANGELOG.md:6` (documentation; matched `.html`)
- `LICENSE:674` (documentation; matched `.html`)
- `internal/service/html.go:335` (source; matched `.html`)
- `internal/service/service.go:3` (source; matched `.html`)
- `internal/service/service_test.go:67` (test; matched `xhtml`)

### markdown

Role-weighted hits: `{"documentation": 9, "source": 1}`.

- `AGENTS.md:1` (source; matched `.md`)
- `CHANGELOG.md:83` (documentation; matched `.md`)
- `CHANGELOG.md:197` (documentation; matched `.md`)
- `README.md:89` (documentation; matched `.md`)
- `README.md:89` (documentation; matched `.md`)

### mobi_azw

Role-weighted hits: `{"documentation": 8, "source": 6}`.

- `README.md:37` (documentation; matched `MOBI`)
- `README.md:54` (documentation; matched `MOBI`)
- `docs/index.html:8` (documentation; matched `MOBI`)
- `docs/index.html:221` (documentation; matched `MOBI`)
- `internal/service/service.go:33` (source; matched `.mobi`)
- Additional evidence retained in JSON: 5 records.

### opf

Role-weighted hits: `{"documentation": 1, "source": 4, "test": 6}`.

- `AGENTS.md:168` (source; matched `Package document`)
- `CHANGELOG.md:243` (documentation; matched `.opf`)
- `internal/service/internal_test.go:59` (test; matched `.opf`)
- `internal/service/internal_test.go:60` (test; matched `.opf`)
- `internal/service/service.go:494` (source; matched `.opf`)
- Additional evidence retained in JSON: 3 records.

### pdf

Role-weighted hits: `{"configuration": 1, "documentation": 12, "source": 11, "test": 8}`.

- `AGENTS.md:177` (source; matched `pdf`)
- `AGENTS.md:177` (source; matched `PDF`)
- `CHANGELOG.md:37` (documentation; matched `PDF`)
- `CHANGELOG.md:37` (documentation; matched `PDF`)
- `README.md:37` (documentation; matched `PDF`)
- Additional evidence retained in JSON: 14 records.

### txt

Role-weighted hits: `{"source": 1, "test": 16}`.

- `.goreleaser.yml:45` (source; matched `.txt`)
- `internal/service/internal_test.go:28` (test; matched `.txt`)
- `internal/service/internal_test.go:169` (test; matched `plain text`)
- `internal/service/service_test.go:48` (test; matched `.txt`)
- `internal/service/service_test.go:49` (test; matched `.txt`)
