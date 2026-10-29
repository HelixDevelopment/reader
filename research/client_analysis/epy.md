# epy — automated static evidence

- Inventory URL: https://github.com/wustho/epy
- Origin: https://github.com/wustho/epy
- Revision: `6b0e9fe0773f05fdf844b574f0f28df3961f60ab`
- Tree: `ccd69ae7c260422afe4c12b4482a46dd6bcbcc50`
- Analysis status: **automated_static_index_complete**; manual review: **pending**.
- Keyword hits are leads for code review, not proof of working support.

## Tree and languages

Tracked files: 63. Text files scanned: 62 (542154 bytes). Files skipped by size, binary classification, or sensitive-name policy remain indexed in the compressed file ledger.

- Python: 48 files
- Shell: 1 files

## License review

Status: heuristic_requires_legal_review. Candidate notices: AGPL-3.0, GPL-3.0, LGPL.
- `LICENSE`

## Feature evidence

### ai_llm

Role-weighted hits: `{"configuration": 1, "documentation": 7, "source": 69, "test": 2}`.

- `.github/workflows/tests.yml:26` (test; matched `rag`)
- `.github/workflows/tests.yml:27` (test; matched `rag`)
- `.gitignore:8` (source; matched `rag`)
- `LICENSE:238` (documentation; matched `rag`)
- `LICENSE:301` (documentation; matched `rag`)
- Additional evidence retained in JSON: 23 records.

### annotations

Role-weighted hits: `{"documentation": 3, "source": 47}`.

- `README.md:20` (documentation; matched `Bookmark`)
- `README.md:61` (documentation; matched `bookmark`)
- `README.md:62` (documentation; matched `bookmark`)
- `poetry.lock:579` (source; matched `highlight`)
- `poetry.lock:701` (source; matched `highlight`)
- Additional evidence retained in JSON: 9 records.

### backup_restore

Role-weighted hits: `{"source": 5}`.

- `src/epy_reader/tools/KindleUnpack/mobi_sectioner.py:104` (source; matched `backup`)
- `src/epy_reader/tools/KindleUnpack/mobi_sectioner.py:105` (source; matched `backup`)
- `src/epy_reader/tools/KindleUnpack/mobi_sectioner.py:106` (source; matched `Backup`)

### collections_smart

Role-weighted hits: `{"source": 3, "test": 1}`.

- `poetry.lock:772` (source; matched `Collection`)
- `src/epy_reader/tools/KindleUnpack/mobi_header.py:11` (source; matched `collection`)
- `src/epy_reader/tools/KindleUnpack/mobi_k8resc.py:11` (source; matched `collection`)
- `tests/test_utils.py:1` (test; matched `collection`)

### comics_manga

Role-weighted hits: `{"source": 1}`.

- `src/epy_reader/tools/KindleUnpack/kindleunpack.py:107` (source; matched `right to left`)

### compression_cache

Role-weighted hits: `{"documentation": 1, "source": 33}`.

- `.gitignore:1` (source; matched `cache`)
- `README.md:5` (documentation; matched `cdn`)
- `poetry.lock:805` (source; matched `brotli`)
- `poetry.lock:805` (source; matched `brotli`)
- `poetry.lock:805` (source; matched `brotli`)
- Additional evidence retained in JSON: 11 records.

### conversion

Role-weighted hits: `{"configuration": 1, "source": 5}`.

- `pyproject.toml:9` (configuration; matched `epub", "epub3", "fb2`)
- `src/epy_reader/tools/KindleUnpack/kindleunpack.py:42` (source; matched `conversion`)
- `src/epy_reader/tools/KindleUnpack/kindleunpack.py:134` (source; matched `conversion`)
- `src/epy_reader/tools/KindleUnpack/mobiml2xhtml.py:27` (source; matched `Converter`)
- `src/epy_reader/tools/KindleUnpack/mobiml2xhtml.py:510` (source; matched `Converter`)
- Additional evidence retained in JSON: 1 records.

### daisy_accessibility

Role-weighted hits: `{"source": 9}`.

- `src/epy_reader/ebooks/epub.py:19` (source; matched `DAISY`)
- `src/epy_reader/ebooks/epub.py:19` (source; matched `daisy`)
- `src/epy_reader/ebooks/epub.py:93` (source; matched `DAISY`)
- `src/epy_reader/tools/KindleUnpack/mobi_ncx.py:107` (source; matched `daisy`)
- `src/epy_reader/tools/KindleUnpack/mobi_ncx.py:193` (source; matched `daisy`)

### database

Role-weighted hits: `{"documentation": 3, "source": 22}`.

- `LICENSE:627` (documentation; matched `redis`)
- `LICENSE:637` (documentation; matched `redis`)
- `LICENSE:657` (documentation; matched `redis`)
- `src/epy_reader/reader.py:9` (source; matched `sqlite`)
- `src/epy_reader/reader.py:1344` (source; matched `sqlite`)
- Additional evidence retained in JSON: 10 records.

### deduplication

Role-weighted hits: `{"source": 63}`.

- `build/gen_pkgbuild.sh:26` (source; matched `sha256`)
- `poetry.lock:852` (source; matched `sha256`)
- `poetry.lock:853` (source; matched `sha256`)
- `poetry.lock:858` (source; matched `sha256`)
- `src/epy_reader/config.py:71` (source; matched `duplicate`)
- Additional evidence retained in JSON: 2 records.

### dictionary_language

Role-weighted hits: `{"documentation": 4, "source": 54}`.

- `README.md:21` (documentation; matched `dictionary`)
- `README.md:145` (documentation; matched `Dictionary`)
- `README.md:147` (documentation; matched `dictionary`)
- `src/epy_reader/reader.py:197` (source; matched `Dictionary`)
- `src/epy_reader/reader.py:198` (source; matched `Dictionary`)
- Additional evidence retained in JSON: 20 records.

### http_protocols

Role-weighted hits: `{"source": 4}`.

- `poetry.lock:805` (source; matched `brotli`)
- `poetry.lock:805` (source; matched `brotli`)
- `poetry.lock:805` (source; matched `brotli`)

### lcp_drm

Role-weighted hits: `{"documentation": 3, "source": 19}`.

- `LICENSE:6` (documentation; matched `license document`)
- `LICENSE:392` (documentation; matched `license document`)
- `LICENSE:395` (documentation; matched `license document`)
- `src/epy_reader/tools/KindleUnpack/mobi_header.py:55` (source; matched `Drm`)
- `src/epy_reader/tools/KindleUnpack/mobi_header.py:56` (source; matched `Drm`)
- Additional evidence retained in JSON: 4 records.

### local_library

Role-weighted hits: `{"source": 8}`.

- `src/epy_reader/models.py:216` (source; matched `filesystem`)
- `src/epy_reader/models.py:218` (source; matched `filesystem`)
- `src/epy_reader/tools/KindleUnpack/compatibility_utils.py:106` (source; matched `in place`)
- `src/epy_reader/tools/KindleUnpack/compatibility_utils.py:259` (source; matched `filesystem`)
- `src/epy_reader/tools/KindleUnpack/kindleunpack.py:132` (source; matched `in place`)
- Additional evidence retained in JSON: 3 records.

### metadata_enrichment

Role-weighted hits: `{"documentation": 1, "source": 338}`.

- `README.md:58` (documentation; matched `metadata`)
- `poetry.lock:99` (source; matched `metadata`)
- `poetry.lock:248` (source; matched `metadata`)
- `poetry.lock:250` (source; matched `metadata`)
- `src/epy_reader/ebooks/base.py:4` (source; matched `Metadata`)
- Additional evidence retained in JSON: 35 records.

### metadata_review

Role-weighted hits: `{"source": 5}`.

- `src/epy_reader/tools/KindleUnpack/mobi_opf.py:285` (source; matched `metadata, 'orientation-lock', 'orientation-lock`)
- `src/epy_reader/tools/KindleUnpack/mobi_opf.py:290` (source; matched `metadata, 'Review', 'review`)
- `src/epy_reader/tools/KindleUnpack/mobi_opf.py:583` (source; matched `metadata.get('orientation-lock`)

### multiuser_auth

Role-weighted hits: `{"documentation": 1}`.

- `LICENSE:311` (documentation; matched `authorization`)

### pdf_fixed_layout

Role-weighted hits: `{"source": 9}`.

- `src/epy_reader/tools/KindleUnpack/mobi_header.py:78` (source; matched `fixed-layout`)
- `src/epy_reader/tools/KindleUnpack/mobi_header.py:412` (source; matched `fixed-layout`)
- `src/epy_reader/tools/KindleUnpack/mobi_opf.py:284` (source; matched `fixed-layout`)
- `src/epy_reader/tools/KindleUnpack/mobi_opf.py:284` (source; matched `fixed-layout`)
- `src/epy_reader/tools/KindleUnpack/mobi_opf.py:581` (source; matched `fixed-layout`)

### permissions

Role-weighted hits: `{"documentation": 20}`.

- `LICENSE:42` (documentation; matched `permission`)
- `LICENSE:85` (documentation; matched `permission`)
- `LICENSE:93` (documentation; matched `permission`)

### progress_sync

Role-weighted hits: `{"documentation": 2, "source": 3}`.

- `README.md:19` (documentation; matched `Reading progress`)
- `README.md:106` (documentation; matched `reading progress`)
- `src/epy_reader/reader.py:118` (source; matched `reading progress`)
- `src/epy_reader/reader.py:1516` (source; matched `reading progress`)
- `src/epy_reader/reader.py:1519` (source; matched `reading progress`)

### reader_typography

Role-weighted hits: `{"source": 7}`.

- `poetry.lock:186` (source; matched `theme`)
- `src/epy_reader/tools/KindleUnpack/mobiml2xhtml.py:48` (source; matched `font_size`)
- `src/epy_reader/tools/KindleUnpack/mobiml2xhtml.py:341` (source; matched `font size`)
- `src/epy_reader/tools/KindleUnpack/mobiml2xhtml.py:430` (source; matched `font size`)

### security

Role-weighted hits: `{"configuration": 6, "documentation": 13, "source": 14}`.

- `.github/workflows/release-aur.yml:29` (configuration; matched `secret`)
- `.github/workflows/release-aur.yml:30` (configuration; matched `secret`)
- `.github/workflows/release-aur.yml:31` (configuration; matched `secret`)
- `.github/workflows/release-pypi.yml:22` (configuration; matched `secret`)
- `.github/workflows/release-win.yml:26` (configuration; matched `secret`)
- Additional evidence retained in JSON: 16 records.

### testing

Role-weighted hits: `{"configuration": 1, "source": 75, "test": 1}`.

- `.github/workflows/tests.yml:26` (test; matched `pytest`)
- `Makefile:18` (source; matched `pytest`)
- `Makefile:21` (source; matched `pytest`)
- `poetry.lock:21` (source; matched `pytest`)
- `poetry.lock:32` (source; matched `pytest`)
- Additional evidence retained in JSON: 2 records.

### tts_audio

Role-weighted hits: `{"documentation": 6, "source": 6}`.

- `README.md:23` (documentation; matched `Text-to-Speech`)
- `README.md:23` (documentation; matched `text-to-speech`)
- `README.md:135` (documentation; matched `Text-to-Speech`)
- `src/epy_reader/reader.py:131` (source; matched `TTS`)
- `src/epy_reader/tools/KindleUnpack/mobi_header.py:149` (source; matched `Text-to-Speech`)
- Additional evidence retained in JSON: 4 records.


## Format evidence

### daisy

Role-weighted hits: `{"source": 9}`.

- `src/epy_reader/ebooks/epub.py:19` (source; matched `DAISY`)
- `src/epy_reader/ebooks/epub.py:19` (source; matched `.daisy`)
- `src/epy_reader/tools/KindleUnpack/mobi_ncx.py:107` (source; matched `.daisy`)
- `src/epy_reader/tools/KindleUnpack/mobi_ncx.py:193` (source; matched `.daisy`)

### epub

Role-weighted hits: `{"configuration": 2, "documentation": 4, "source": 95}`.

- `README.md:14` (documentation; matched `Epub`)
- `README.md:14` (documentation; matched `.epub`)
- `pyproject.toml:9` (configuration; matched `epub`)
- `pyproject.toml:9` (configuration; matched `epub3`)
- `src/epy_reader/ebooks/__init__.py:3` (source; matched `Epub`)
- Additional evidence retained in JSON: 23 records.

### fb2

Role-weighted hits: `{"configuration": 1, "documentation": 2, "source": 27}`.

- `README.md:15` (documentation; matched `FictionBook`)
- `README.md:15` (documentation; matched `.fb2`)
- `pyproject.toml:9` (configuration; matched `fb2`)
- `src/epy_reader/ebooks/__init__.py:4` (source; matched `FictionBook`)
- `src/epy_reader/ebooks/__init__.py:13` (source; matched `fictionbook`)
- Additional evidence retained in JSON: 4 records.

### html_xhtml

Role-weighted hits: `{"documentation": 1, "source": 89, "test": 3}`.

- `LICENSE:674` (documentation; matched `.html`)
- `src/epy_reader/__main__.py:21` (source; matched `.html`)
- `src/epy_reader/ebooks/epub.py:22` (source; matched `XHTML`)
- `src/epy_reader/ebooks/epub.py:22` (source; matched `xhtml`)
- `src/epy_reader/ebooks/url.py:16` (source; matched `xhtml`)
- Additional evidence retained in JSON: 21 records.

### markdown

Role-weighted hits: `{"configuration": 1, "source": 2}`.

- `poetry.lock:155` (source; matched `Markdown`)
- `poetry.lock:701` (source; matched `markdown`)
- `pyproject.toml:7` (configuration; matched `.md`)

### mobi_azw

Role-weighted hits: `{"configuration": 2, "documentation": 5, "source": 86}`.

- `README.md:16` (documentation; matched `Mobi`)
- `README.md:16` (documentation; matched `.mobi`)
- `pyproject.toml:9` (configuration; matched `mobi`)
- `pyproject.toml:9` (configuration; matched `azw3`)
- `src/epy_reader/ebooks/__init__.py:5` (source; matched `Mobi`)
- Additional evidence retained in JSON: 20 records.

### opf

Role-weighted hits: `{"source": 9}`.

- `src/epy_reader/ebooks/mobi.py:25` (source; matched `.opf`)
- `src/epy_reader/ebooks/mobi.py:40` (source; matched `.opf`)
- `src/epy_reader/tools/KindleUnpack/kindleunpack.py:109` (source; matched `.opf`)
- `src/epy_reader/tools/KindleUnpack/kindleunpack.py:377` (source; matched `.opf`)
- `src/epy_reader/tools/KindleUnpack/mobi_opf.py:29` (source; matched `.opf`)
- Additional evidence retained in JSON: 4 records.

### pdf

Role-weighted hits: `{"source": 8}`.

- `src/epy_reader/tools/KindleUnpack/kindleunpack.py:62` (source; matched `PDF`)
- `src/epy_reader/tools/KindleUnpack/kindleunpack.py:443` (source; matched `PDF`)
- `src/epy_reader/tools/KindleUnpack/mobi_opf.py:362` (source; matched `.pdf`)
- `src/epy_reader/tools/KindleUnpack/mobi_opf.py:362` (source; matched `application/pdf`)
- `src/epy_reader/tools/KindleUnpack/unpack_structure.py:144` (source; matched `pdf`)

### webpub_lpf

Role-weighted hits: `{"source": 1}`.

- `src/epy_reader/cli.py:86` (source; matched `lpF`)
