# edrlab-lcp-server — automated static evidence

- Inventory URL: https://github.com/edrlab/lcp-server
- Origin: https://github.com/edrlab/lcp-server
- Revision: `e41ae954097fc42b2d8996fabdfd355a1824108c`
- Tree: `3dc0685e49f2365f3e6c367c5478f520de282798`
- Analysis status: **automated_static_index_complete**; manual review: **pending**.
- Keyword hits are leads for code review, not proof of working support.

## Tree and languages

Tracked files: 99. Text files scanned: 91 (371533 bytes). Files skipped by size, binary classification, or sensitive-name policy remain indexed in the compressed file ledger.

- Go: 54 files
- Shell: 5 files

## License review

Status: heuristic_requires_legal_review. Candidate notices: BSD-2-Clause, BSD-3-Clause, CC0.
- `LICENSE`
- `pkg/api/license_handler.go`
- `pkg/api/licenseinfo_handler.go`
- `pkg/check/data/license.schema.json`
- `pkg/lic/license.go`
- `pkg/lic/license_test.go`
- `pkg/stor/licenseinfo.go`
- `test/license1.lcpl`

## Feature evidence

### ai_llm

Role-weighted hits: `{"configuration": 2, "documentation": 7, "source": 29}`.

- `.gitignore:16` (source; matched `rag`)
- `Dockerfile:10` (configuration; matched `rag`)
- `Dockerfile:11` (configuration; matched `rag`)
- `cmd/lcpencrypt/lcpencrypt.go:30` (source; matched `rag`)
- `cmd/lcpencrypt/lcpencrypt.go:31` (source; matched `rag`)
- Additional evidence retained in JSON: 17 records.

### backup_restore

Role-weighted hits: `{"documentation": 123, "source": 202}`.

- `.gitignore:29` (source; matched `backup`)
- `NGINX-DEPLOYMENT.md:166` (source; matched `backup`)
- `compose.yaml:38` (source; matched `backup`)
- `compose.yaml:177` (source; matched `backup`)
- `compose.yaml:177` (source; matched `backup`)
- Additional evidence retained in JSON: 19 records.

### collections_smart

Role-weighted hits: `{"source": 1}`.

- `pkg/check/data/link.schema.json:24` (source; matched `collection`)

### comics_manga

Role-weighted hits: `{"source": 1, "test": 1}`.

- `pkg/stor/dashboard.go:205` (source; matched `Comic`)
- `test/license1.lcpl:47` (test; matched `Cbr`)

### compression_cache

Role-weighted hits: `{"configuration": 3, "documentation": 3, "source": 12, "test": 3}`.

- `Dockerfile:10` (configuration; matched `cache`)
- `Dockerfile:13` (configuration; matched `cache`)
- `Dockerfile:23` (configuration; matched `cache`)
- `NGINX-DEPLOYMENT.md:30` (source; matched `Gzip`)
- `compose.yaml:195` (source; matched `gzip`)
- Additional evidence retained in JSON: 13 records.

### conversion

Role-weighted hits: `{"documentation": 2, "test": 1}`.

- `README.md:22` (documentation; matched `EPUBs, PDF`)
- `docs/api.md:174` (documentation; matched `epub`, `pdf`, `lcpdf`)
- `pkg/api/api_publication_test.go:208` (test; matched `epub", "lcpdf`)

### database

Role-weighted hits: `{"configuration": 12, "documentation": 87, "source": 263, "test": 21}`.

- `Dockerfile:45` (configuration; matched `MYSQL`)
- `Dockerfile:48` (configuration; matched `MYSQL`)
- `Dockerfile:52` (configuration; matched `MYSQL`)
- `LICENSE:6` (documentation; matched `Redis`)
- `LICENSE:9` (documentation; matched `Redis`)
- Additional evidence retained in JSON: 35 records.

### deduplication

Role-weighted hits: `{"documentation": 10, "source": 18, "test": 11}`.

- `cmd/lcpencrypt/server.go:99` (source; matched `duplicate`)
- `docs/api.md:19` (documentation; matched `duplicate`)
- `docs/api.md:64` (documentation; matched `SHA256`)
- `docs/api.md:64` (documentation; matched `sha256`)
- `pkg/api/api_publication_test.go:26` (test; matched `Checksum`)
- Additional evidence retained in JSON: 23 records.

### deployment

Role-weighted hits: `{"configuration": 3, "documentation": 40, "source": 58}`.

- `.dockerignore:4` (source; matched `docker`)
- `.dockerignore:5` (source; matched `docker`)
- `.dockerignore:5` (source; matched `docker`)
- `Dockerfile:1` (configuration; matched `docker`)
- `Dockerfile:1` (configuration; matched `docker`)
- Additional evidence retained in JSON: 33 records.

### http_protocols

Role-weighted hits: `{"documentation": 2, "source": 2, "test": 1}`.

- `README.md:67` (documentation; matched `quic`)
- `compose.yaml:195` (source; matched `quic`)
- `docs/backup.md:76` (documentation; matched `Quic`)
- `pkg/crypto/aes_gcm_test.go:41` (test; matched `quic`)
- `scripts/hot-backup.sh:44` (source; matched `quic`)

### lcp_drm

Role-weighted hits: `{"configuration": 7, "documentation": 144, "source": 120, "test": 32}`.

- `Dockerfile:39` (configuration; matched `LCP`)
- `Dockerfile:41` (configuration; matched `LCP`)
- `Dockerfile:43` (configuration; matched `LCP`)
- `NGINX-DEPLOYMENT.md:1` (source; matched `LCP`)
- `NGINX-DEPLOYMENT.md:3` (source; matched `LCP`)
- Additional evidence retained in JSON: 35 records.

### local_library

Role-weighted hits: `{"source": 1}`.

- `cmd/lcpencrypt/lcpencrypt.go:139` (source; matched `in place`)

### metadata_enrichment

Role-weighted hits: `{"source": 1}`.

- `cmd/lcpencrypt/lcpencrypt.go:71` (source; matched `metadata`)

### multiuser_auth

Role-weighted hits: `{"configuration": 2, "documentation": 18, "source": 82}`.

- `NGINX-DEPLOYMENT.md:39` (source; matched `authentication`)
- `cmd/lcpserver/authenticator.go:15` (source; matched `jwt`)
- `cmd/lcpserver/authenticator.go:15` (source; matched `jwt`)
- `cmd/lcpserver/authenticator.go:25` (source; matched `jwt`)
- `cmd/lcpserver/router.go:49` (source; matched `Authorization`)
- Additional evidence retained in JSON: 27 records.

### network_sources

Role-weighted hits: `{"documentation": 6, "source": 6}`.

- `compose.yaml:35` (source; matched `FTP`)
- `docs/api.md:27` (documentation; matched `S3`)
- `docs/backup.md:160` (documentation; matched `Cloud Storage`)
- `docs/backup.md:162` (documentation; matched `S3`)
- `docs/backup.md:167` (documentation; matched `s3`)
- Additional evidence retained in JSON: 3 records.

### opds

Role-weighted hits: `{"source": 1}`.

- `config/nginx.conf:30` (source; matched `application/atom+xml`)

### pdf_fixed_layout

Role-weighted hits: `{"configuration": 3}`.

- `Dockerfile:30` (configuration; matched `mupdf`)
- `Dockerfile:64` (configuration; matched `mupdf`)
- `Dockerfile:65` (configuration; matched `mupdf`)

### permissions

Role-weighted hits: `{"documentation": 4, "source": 5, "test": 4}`.

- `LICENSE:18` (documentation; matched `permission`)
- `config/README.md:55` (documentation; matched `permission`)
- `docs/backup.md:277` (documentation; matched `permission`)
- `docs/deployment.md:115` (documentation; matched `permission`)
- `pkg/crypto/aes_cbc.go:13` (source; matched `permission`)
- Additional evidence retained in JSON: 8 records.

### reader_typography

Role-weighted hits: `{"documentation": 3}`.

- `docs/Gemfile:4` (documentation; matched `theme`)
- `docs/_config.yml:3` (documentation; matched `theme`)
- `docs/_config.yml:7` (documentation; matched `theme`)

### readium

Role-weighted hits: `{"configuration": 2, "documentation": 18, "source": 34, "test": 18}`.

- `README.md:3` (documentation; matched `Readium`)
- `README.md:3` (documentation; matched `readium`)
- `README.md:3` (documentation; matched `readium`)
- `cmd/lcpencrypt/process.go:20` (source; matched `readium`)
- `cmd/lcpencrypt/process.go:20` (source; matched `readium`)
- Additional evidence retained in JSON: 35 records.

### security

Role-weighted hits: `{"configuration": 5, "documentation": 75, "source": 200, "test": 23}`.

- `.dockerignore:5` (source; matched `https`)
- `.dockerignore:29` (source; matched `secret`)
- `Dockerfile:65` (configuration; matched `certificate`)
- `Dockerfile:73` (configuration; matched `certificate`)
- `Dockerfile:73` (configuration; matched `certificate`)
- Additional evidence retained in JSON: 35 records.

### testing

Role-weighted hits: `{"documentation": 1, "source": 1}`.

- `.gitignore:13` (source; matched `go test`)
- `docs/choices.md:13` (documentation; matched `unit test`)

### tts_audio

Role-weighted hits: `{"documentation": 1, "source": 4}`.

- `docs/api.md:11` (documentation; matched `audiobook`)
- `pkg/api/publication_handler.go:63` (source; matched `audiobook`)
- `pkg/check/checklicense.go:277` (source; matched `audiobook`)
- `pkg/stor/dashboard.go:202` (source; matched `audiobook`)
- `pkg/stor/dashboard.go:203` (source; matched `Audiobook`)

### watch_import

Role-weighted hits: `{"configuration": 2, "documentation": 1, "source": 11}`.

- `README.md:21` (documentation; matched `watch folder`)
- `cmd/lcpencrypt/lcpencrypt.go:134` (source; matched `Watching director`)
- `cmd/lcpencrypt/server.go:21` (source; matched `fsnotify`)
- `cmd/lcpencrypt/server.go:21` (source; matched `fsnotify`)
- `cmd/lcpencrypt/server.go:79` (source; matched `fsnotify`)
- Additional evidence retained in JSON: 5 records.


## Format evidence

### divina

Role-weighted hits: `{"source": 3}`.

- `pkg/api/publication_handler.go:65` (source; matched `divina`)
- `pkg/check/checklicense.go:278` (source; matched `divina`)
- `pkg/stor/dashboard.go:204` (source; matched `divina`)

### epub

Role-weighted hits: `{"documentation": 3, "source": 5, "test": 10}`.

- `docs/api.md:158` (documentation; matched `.epub`)
- `docs/api.md:159` (documentation; matched `epub`)
- `pkg/api/api_publication_test.go:193` (test; matched `epub `)
- `pkg/api/api_publication_test.go:208` (test; matched `epub`)
- `pkg/api/api_test.go:105` (test; matched `epub`)
- Additional evidence retained in JSON: 10 records.

### html_xhtml

Role-weighted hits: `{"documentation": 2, "source": 2}`.

- `config/nginx.conf:141` (source; matched `.html`)
- `config/nginx.conf:142` (source; matched `.html`)
- `docs/choices.md:31` (documentation; matched `.html`)
- `docs/choices.md:31` (documentation; matched `.html`)

### lcp_protected

Role-weighted hits: `{"configuration": 7, "documentation": 140, "source": 114, "test": 24}`.

- `Dockerfile:39` (configuration; matched `LCP`)
- `Dockerfile:41` (configuration; matched `LCP`)
- `NGINX-DEPLOYMENT.md:1` (source; matched `LCP`)
- `NGINX-DEPLOYMENT.md:3` (source; matched `LCP`)
- `README.md:3` (documentation; matched `LCP`)
- Additional evidence retained in JSON: 35 records.

### markdown

Role-weighted hits: `{"documentation": 1, "source": 1}`.

- `.dockerignore:32` (source; matched `.md`)
- `docs/integration.md:59` (documentation; matched `.md`)

### pdf

Role-weighted hits: `{"documentation": 2, "source": 10, "test": 1}`.

- `README.md:22` (documentation; matched `PDF`)
- `cmd/lcpencrypt/lcpencrypt.go:71` (source; matched `PDF`)
- `docs/api.md:174` (documentation; matched `pdf`)
- `pkg/api/api_publication_test.go:200` (test; matched `application/pdf`)
- `pkg/api/publication_handler.go:57` (source; matched `pdf`)
- Additional evidence retained in JSON: 6 records.

### txt

Role-weighted hits: `{"documentation": 19, "source": 10}`.

- `compose.yaml:152` (source; matched `.txt`)
- `compose.yaml:243` (source; matched `.txt`)
- `config/README.md:7` (documentation; matched `.txt`)
- `config/README.md:20` (documentation; matched `.txt`)
- `docs/deployment.md:24` (documentation; matched `.txt`)
- Additional evidence retained in JSON: 6 records.

### webpub_lpf

Role-weighted hits: `{"source": 2}`.

- `go.sum:58` (source; matched `lPF`)
- `pkg/check/data/lcpdf.schema.json:8` (source; matched `webpub`)
