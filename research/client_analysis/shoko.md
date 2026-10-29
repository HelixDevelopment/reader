# shoko — automated static evidence

- Inventory URL: https://git.sr.ht/~shayan/shoko
- Origin: https://git.sr.ht/~shayan/Shoko
- Revision: `eb319d18d7154ebebb403990f3ac9d99cc1aa893`
- Tree: `d261648b9cdf3c2b741a79f41d3873c3a304f621`
- Analysis status: **automated_static_index_complete**; manual review: **pending**.
- Keyword hits are leads for code review, not proof of working support.

## Tree and languages

Tracked files: 1217. Text files scanned: 1215 (5453435 bytes). Files skipped by size, binary classification, or sensitive-name policy remain indexed in the compressed file ledger.

- Ruby: 1168 files
- C: 6 files
- C/C++ header: 3 files
- Shell: 1 files

## License review

Status: heuristic_requires_legal_review. Candidate notices: MIT.
- `LICENSE`

## Feature evidence

### ai_llm

Role-weighted hits: `{"configuration": 6, "documentation": 8, "source": 940, "test": 646}`.

- `.github/workflows/ruby.yml:86` (configuration; matched `rag`)
- `.github/workflows/ruby.yml:97` (configuration; matched `rag`)
- `.github/workflows/ruby.yml:98` (configuration; matched `rag`)
- `.gitignore:3` (source; matched `rag`)
- `.rubocop.yml:40` (source; matched `rag`)
- Additional evidence retained in JSON: 35 records.

### annotations

Role-weighted hits: `{"documentation": 18, "source": 2860, "test": 828}`.

- `README.md:9` (documentation; matched `Bookmark`)
- `README.md:9` (documentation; matched `annotation`)
- `README.md:9` (documentation; matched `bookmark`)
- `docs/architecture/census-2026-06-01.md:53` (documentation; matched `Annotation`)
- `docs/architecture/port-inventory.yml:19` (documentation; matched `annotation`)
- Additional evidence retained in JSON: 35 records.

### backup_restore

Role-weighted hits: `{"documentation": 13, "source": 897, "test": 820}`.

- `docs/architecture/census-2026-06-01.md:3` (documentation; matched `snapshot`)
- `docs/architecture/port-inventory.yml:53` (documentation; matched `snapshot`)
- `docs/architecture/port-inventory.yml:68` (documentation; matched `snapshot`)
- `docs/architecture/port-inventory.yml:69` (documentation; matched `snapshot`)
- `docs/performance.md:19` (documentation; matched `snapshot`)
- Additional evidence retained in JSON: 35 records.

### collections_smart

Role-weighted hits: `{"source": 3, "test": 2}`.

- `lib/shoko/adapters/storage/json_cache_store.rb:193` (source; matched `collection`)
- `lib/shoko/adapters/storage/json_cache_store.rb:209` (source; matched `collection`)
- `lib/shoko/adapters/translation/model_catalog_service.rb:23` (source; matched `collection`)
- `spec/adapters/book_sources/parser/pdf/pdf_reader_spec.rb:176` (test; matched `collection`)
- `spec/core/architecture/no_reflection_probing_spec.rb:42` (test; matched `collection`)

### comics_manga

Role-weighted hits: `{"source": 1}`.

- `lib/shoko/shared/index_range.rb:7` (source; matched `right-to-left`)

### compression_cache

Role-weighted hits: `{"configuration": 6, "documentation": 21, "source": 2457, "test": 979}`.

- `.github/workflows/fixtures-nightly.yml:25` (configuration; matched `cache`)
- `.github/workflows/ruby.yml:48` (configuration; matched `cache`)
- `.github/workflows/ruby.yml:68` (configuration; matched `cache`)
- `.github/workflows/ruby.yml:82` (configuration; matched `cache`)
- `README.md:14` (documentation; matched `cache`)
- Additional evidence retained in JSON: 35 records.

### conversion

Role-weighted hits: `{"documentation": 4, "source": 34, "test": 20}`.

- `README.md:3` (documentation; matched `epub`, `.fb2`, `.pdf`)
- `README.md:45` (documentation; matched `EPUB`, `PDF`, `FB2`)
- `README.md:51` (documentation; matched `EPUB/FB2`)
- `docs/performance.md:5` (documentation; matched `epub, fb2`)
- `lib/shoko/adapters/book_sources/format_registry.rb:29` (source; matched `epub', '.fb2', '.fb2`)
- Additional evidence retained in JSON: 35 records.

### database

Role-weighted hits: `{"documentation": 3, "source": 63, "test": 136}`.

- `README.md:10` (documentation; matched `sqlite`)
- `README.md:19` (documentation; matched `sqlite`)
- `README.md:19` (documentation; matched `sqlite`)
- `lib/shoko/adapters/input/controllers/dictionary/dataset_installer.rb:46` (source; matched `sqlite`)
- `lib/shoko/adapters/input/controllers/reader/selection_interaction.rb:86` (source; matched `sqlite`)
- Additional evidence retained in JSON: 35 records.

### deduplication

Role-weighted hits: `{"configuration": 6, "documentation": 11, "source": 184, "test": 40}`.

- `.github/workflows/fixtures-nightly.yml:31` (configuration; matched `SHA256`)
- `.github/workflows/fixtures-nightly.yml:31` (configuration; matched `SHA256`)
- `.github/workflows/fixtures-nightly.yml:32` (configuration; matched `checksum`)
- `.rubocop.yml:176` (source; matched `Duplicate`)
- `README.md:29` (documentation; matched `sha256`)
- Additional evidence retained in JSON: 35 records.

### deployment

Role-weighted hits: `{"source": 6, "test": 5}`.

- `lib/shoko/application/workflows/cli/folder_import_readiness_warmup.rb:15` (source; matched `Readiness`)
- `lib/shoko/application/workflows/cli/folder_import_readiness_warmup.rb:127` (source; matched `readiness`)
- `lib/shoko/composition/container_factory.rb:71` (source; matched `readiness`)
- `lib/shoko/composition/container_factory.rb:145` (source; matched `Readiness`)
- `lib/shoko/composition/container_factory.rb:146` (source; matched `Readiness`)
- Additional evidence retained in JSON: 6 records.

### dictionary_language

Role-weighted hits: `{"configuration": 1, "documentation": 25, "source": 1916, "test": 917}`.

- `.github/workflows/ruby.yml:29` (configuration; matched `translation`)
- `.gitignore:10` (source; matched `translation`)
- `README.md:10` (documentation; matched `dictionary`)
- `README.md:10` (documentation; matched `dictionary`)
- `README.md:10` (documentation; matched `dictionary`)
- Additional evidence retained in JSON: 35 records.

### http_protocols

Role-weighted hits: `{"documentation": 1, "source": 8, "test": 10}`.

- `README.md:9` (documentation; matched `quic`)
- `lib/shoko/adapters/book_sources/kindle/parser/kindle_metadata_extractor.rb:18` (source; matched `quic`)
- `script/bench/snappiness_benchmark.rb:41` (source; matched `quic`)
- `script/bench/snappiness_benchmark.rb:71` (source; matched `quic`)
- `script/bench/snappiness_benchmark.rb:84` (source; matched `quic`)
- Additional evidence retained in JSON: 10 records.

### lcp_drm

Role-weighted hits: `{"documentation": 1, "source": 6, "test": 6}`.

- `README.md:49` (documentation; matched `DRM`)
- `lib/shoko/adapters/book_sources/kindle/kindle_importer.rb:128` (source; matched `drm`)
- `lib/shoko/adapters/book_sources/kindle/kindle_importer.rb:247` (source; matched `drm`)
- `lib/shoko/adapters/book_sources/kindle/kindle_importer.rb:248` (source; matched `drm`)
- `lib/shoko/adapters/book_sources/kindle/parser/mobi_header_parser.rb:92` (source; matched `DRM`)
- Additional evidence retained in JSON: 4 records.

### local_library

Role-weighted hits: `{"documentation": 2, "source": 25, "test": 10}`.

- `docs/architecture/constitution.md:66` (documentation; matched `filesystem`)
- `docs/architecture/port-inventory.yml:17` (documentation; matched `filesystem`)
- `ext/shoko_translate/src/spm.c:444` (source; matched `in place`)
- `lib/shoko/adapters/book_sources/book_file_probe.rb:8` (source; matched `filesystem`)
- `lib/shoko/adapters/book_sources/book_finder/scanner_context.rb:7` (source; matched `directory scan`)
- Additional evidence retained in JSON: 32 records.

### metadata_enrichment

Role-weighted hits: `{"documentation": 3, "source": 1214, "test": 483}`.

- `Shoko.gemspec:17` (source; matched `metadata`)
- `Shoko.gemspec:18` (source; matched `metadata`)
- `Shoko.gemspec:33` (source; matched `metadata`)
- `docs/architecture/constitution.md:58` (documentation; matched `metadata`)
- `docs/architecture/port-inventory.yml:36` (documentation; matched `metadata`)
- Additional evidence retained in JSON: 35 records.

### metadata_review

Role-weighted hits: `{"source": 123, "test": 96}`.

- `lib/shoko/adapters/book_sources/epub/parser/xhtml_block_builder.rb:12` (source; matched `locks and metadata`)
- `lib/shoko/adapters/book_sources/epub/parser/xhtml_block_builder.rb:57` (source; matched `metadata = block`)
- `lib/shoko/adapters/book_sources/epub/parser/xhtml_block_builder.rb:60` (source; matched `lock.new(type: paragraph_type(context), segments: segments, metadata: metadata`)
- `lib/shoko/adapters/book_sources/epub/parser/xhtml_content_traversal.rb:187` (source; matched `metadata = @style_resolver.block`)
- `lib/shoko/adapters/book_sources/epub/parser/xhtml_content_traversal.rb:198` (source; matched `locks[-1] = last.with(metadata: last.metadata`)
- Additional evidence retained in JSON: 35 records.

### network_sources

Role-weighted hits: `{"test": 2}`.

- `spec/adapters/rss/article_content_fetcher_spec.rb:76` (test; matched `ftp`)
- `spec/adapters/rss/feed_fetcher_spec.rb:80` (test; matched `ftp`)

### opds

Role-weighted hits: `{"source": 1}`.

- `lib/shoko/adapters/rss/feed_fetcher.rb:102` (source; matched `application/atom+xml`)

### pdf_fixed_layout

Role-weighted hits: `{"source": 2, "test": 1}`.

- `lib/shoko/adapters/book_sources/kindle/kindle_importer.rb:42` (source; matched `fixed-layout`)
- `lib/shoko/adapters/ui/components/screens/rss_reader_screen_component.rb:42` (source; matched `reflow`)
- `spec/adapters/book_sources/parser/kindle/kindle_importer_spec.rb:166` (test; matched `fixed-layout`)

### permissions

Role-weighted hits: `{"configuration": 2, "documentation": 2, "source": 10, "test": 6}`.

- `.github/workflows/fixtures-nightly.yml:12` (configuration; matched `permission`)
- `.github/workflows/ruby.yml:20` (configuration; matched `permission`)
- `LICENSE:5` (documentation; matched `Permission`)
- `LICENSE:12` (documentation; matched `permission`)
- `lib/shoko/adapters/input/controllers/state_controller.rb:29` (source; matched `permission`)
- Additional evidence retained in JSON: 15 records.

### progress_sync

Role-weighted hits: `{"source": 14, "test": 1}`.

- `lib/shoko/adapters/book_sources/library_scanner.rb:153` (source; matched `synchronize { @scan_in_progress`)
- `lib/shoko/adapters/book_sources/library_scanner.rb:157` (source; matched `synchronize { @scan_in_progress`)
- `lib/shoko/adapters/input/controllers/reader/progress_autosave.rb:8` (source; matched `reading progress`)
- `lib/shoko/adapters/storage/repositories/progress_repository.rb:15` (source; matched `reading progress`)
- `lib/shoko/adapters/storage/repositories/progress_repository.rb:34` (source; matched `reading progress`)
- Additional evidence retained in JSON: 9 records.

### reader_typography

Role-weighted hits: `{"documentation": 5, "source": 404, "test": 183}`.

- `README.md:8` (documentation; matched `line spacing`)
- `README.md:8` (documentation; matched `theme`)
- `README.md:14` (documentation; matched `line spacing`)
- `lib/shoko/adapters/book_sources/epub/parser/xhtml_block_builder.rb:143` (source; matched `typography`)
- `lib/shoko/adapters/book_sources/pdf/importer/page_extraction_coordinator.rb:79` (source; matched `font_size`)
- Additional evidence retained in JSON: 35 records.

### search

Role-weighted hits: `{"documentation": 1, "source": 9, "test": 1}`.

- `README.md:10` (documentation; matched `full-text search`)
- `lib/shoko/adapters/book_sources/libgen_client.rb:22` (source; matched `full-text search`)
- `lib/shoko/adapters/book_sources/libgen_client.rb:82` (source; matched `Full-text search`)
- `lib/shoko/adapters/input/controllers/in_book_search_controller.rb:11` (source; matched `full text search`)
- `lib/shoko/adapters/ui/components/dictionary/entry_formatter.rb:176` (source; matched `fuzzy search`)
- Additional evidence retained in JSON: 6 records.

### security

Role-weighted hits: `{"configuration": 2, "documentation": 2, "source": 61, "test": 162}`.

- `.github/workflows/fixtures-nightly.yml:29` (configuration; matched `https`)
- `.github/workflows/ruby.yml:6` (configuration; matched `https`)
- `Gemfile:3` (source; matched `https`)
- `README.md:109` (documentation; matched `https`)
- `Shoko.gemspec:13` (source; matched `https`)
- Additional evidence retained in JSON: 35 records.

### testing

Role-weighted hits: `{"documentation": 12, "source": 32, "test": 3}`.

- `README.md:143` (documentation; matched `Benchmark`)
- `README.md:146` (documentation; matched `benchmark`)
- `README.md:149` (documentation; matched `benchmark`)
- `docs/performance.md:13` (documentation; matched `benchmark`)
- `docs/performance.md:14` (documentation; matched `benchmark`)
- Additional evidence retained in JSON: 21 records.


## Format evidence

### doc_docx

Role-weighted hits: `{"source": 19}`.

- `lib/shoko/adapters/input/controllers/reader/mouse_session.rb:89` (source; matched `.doc`)
- `lib/shoko/adapters/input/controllers/reader/mouse_session.rb:179` (source; matched `.doc`)
- `lib/shoko/adapters/input/controllers/reader/runtime_setup.rb:152` (source; matched `.doc`)
- `lib/shoko/adapters/input/controllers/reader/startup_sequence.rb:26` (source; matched `.doc`)
- `lib/shoko/adapters/input/controllers/reader_controller.rb:338` (source; matched `.doc`)
- Additional evidence retained in JSON: 6 records.

### epub

Role-weighted hits: `{"documentation": 5, "source": 169, "test": 556}`.

- `README.md:3` (documentation; matched `.epub`)
- `README.md:41` (documentation; matched `.epub`)
- `docs/performance.md:5` (documentation; matched `epub`)
- `lib/shoko/adapters/book_sources/book_finder.rb:134` (source; matched `EPUB `)
- `lib/shoko/adapters/book_sources/book_finder/directory_scanner.rb:13` (source; matched `EPUB `)
- Additional evidence retained in JSON: 35 records.

### fb2

Role-weighted hits: `{"documentation": 4, "source": 98, "test": 91}`.

- `README.md:3` (documentation; matched `.fb2`)
- `README.md:45` (documentation; matched `FB2`)
- `docs/performance.md:5` (documentation; matched `fb2`)
- `lib/shoko/adapters/book_sources/book_finder/directory_scanner.rb:144` (source; matched `.fb2.zip`)
- `lib/shoko/adapters/book_sources/fb2/fb2_importer.rb:12` (source; matched `fb2`)
- Additional evidence retained in JSON: 35 records.

### html_xhtml

Role-weighted hits: `{"source": 62, "test": 94}`.

- `lib/shoko/adapters/book_sources/epub/epub_importer.rb:248` (source; matched `.xhtml`)
- `lib/shoko/adapters/book_sources/epub/epub_importer.rb:248` (source; matched `.html`)
- `lib/shoko/adapters/book_sources/epub/epub_resource_loader.rb:74` (source; matched `XHTML`)
- `lib/shoko/adapters/book_sources/epub/parser/markup_visibility.rb:17` (source; matched `HTML5`)
- `lib/shoko/adapters/book_sources/epub/parser/xhtml_block_builder.rb:13` (source; matched `XHTML`)
- Additional evidence retained in JSON: 35 records.

### markdown

Role-weighted hits: `{"documentation": 7, "source": 2}`.

- `README.md:141` (documentation; matched `.md`)
- `README.md:141` (documentation; matched `.md`)
- `docs/architecture/history/constitution-history.md:12` (documentation; matched `.md`)
- `docs/performance.md:31` (documentation; matched `.md`)
- `docs/performance.md:31` (documentation; matched `.md`)
- Additional evidence retained in JSON: 2 records.

### mobi_azw

Role-weighted hits: `{"documentation": 12, "source": 134, "test": 176}`.

- `README.md:3` (documentation; matched `.mobi`)
- `README.md:3` (documentation; matched `.azw`)
- `docs/book-fixtures.md:25` (documentation; matched `.mobi`)
- `docs/book-fixtures.md:26` (documentation; matched `.azw`)
- `docs/performance.md:6` (documentation; matched `mobi`)
- Additional evidence retained in JSON: 35 records.

### odt_rtf

Role-weighted hits: `{"documentation": 5, "source": 84, "test": 142}`.

- `.gitattributes:4` (source; matched `.rtf`)
- `README.md:3` (documentation; matched `.rtf`)
- `README.md:45` (documentation; matched `RTF`)
- `docs/book-fixtures.md:28` (documentation; matched `.rtf`)
- `docs/performance.md:6` (documentation; matched `rtf`)
- Additional evidence retained in JSON: 35 records.

### opf

Role-weighted hits: `{"source": 1, "test": 2}`.

- `lib/shoko/adapters/book_sources/epub/epub_importer.rb:248` (source; matched `.opf`)
- `spec/composition/reader_composition_integration_spec.rb:30` (test; matched `.opf`)
- `spec/composition/reader_composition_integration_spec.rb:69` (test; matched `.opf`)

### pdf

Role-weighted hits: `{"documentation": 7, "source": 121, "test": 141}`.

- `README.md:3` (documentation; matched `.pdf`)
- `README.md:45` (documentation; matched `PDF`)
- `docs/book-fixtures.md:30` (documentation; matched `.pdf`)
- `docs/book-fixtures.md:31` (documentation; matched `.pdf`)
- `docs/performance.md:76` (documentation; matched `PDF`)
- Additional evidence retained in JSON: 35 records.

### txt

Role-weighted hits: `{"source": 9, "test": 26}`.

- `.gitignore:6` (source; matched `.txt`)
- `lib/shoko/adapters/book_sources/fb2/parser/fb2_inline_parser.rb:34` (source; matched `plain text`)
- `lib/shoko/adapters/output/formatting/formatting_service/plain_lines_builder.rb:8` (source; matched `plain text`)
- `lib/shoko/adapters/ui/components/screens/rss_reader_screen_component.rb:122` (source; matched `plain text`)
- `lib/shoko/application/ports/outbound/text_metrics.rb:9` (source; matched `plain text`)
- Additional evidence retained in JSON: 26 records.

### webpub_lpf

Role-weighted hits: `{"test": 2}`.

- `spec/adapters/book_sources/gutendex_client_spec.rb:56` (test; matched `lpf`)
- `spec/adapters/rss/article_content_fetcher_spec.rb:75` (test; matched `lpf`)
