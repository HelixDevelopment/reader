# Calibre-Web Automated (CWA) — per-repository research report

## Pinned revision and file ledger

- Repository: https://github.com/crocodilestick/calibre-web-automated
- Revision: `4fc2c4459965efb1b8ff43ea7849c2f47a755d9c` (matches `research/analysis_summary.json` slug `crocodilestick-calibre-web-automated`, tree `2cbec1a3784cd9822954496040f4c06642e8cfd4`)
- Branch: `origin/main` (default), commit date 2026-10-01 13:53:50 +0000 — `Update translations [skip ci]`
- Checkout evidence: full non-shallow clone, `git fsck` clean.
- Ledger (recomputed from `git ls-tree -r HEAD`, matches analysis `tracked_files=1198`): 1,198 files, 154,095,066 bytes. Top extensions: py 200 / 3,120,905; js 196 / 7,835,213; bcmap 168 / 1,165,667 (PDF.js binary maps); ftl 111 / 1,486,513 (Fluent translations); svg 95 / 423,342; png 84 / 38,350,272; html 58 / 831,792; css 36 / 1,283,160; po 28 / 6,017,925; epub fixtures 13 / 26,188,877.

## Architecture and data flow

Docker-first appliance that packages an in-repo **fork of Calibre-Web** (`cps/`, the Flask application: `calibre_init.py`, `converter.py`, `kobo.py`/`kobo_auth.py`/`kobo_sync_status.py`, `audio.py`, `comic.py`, `epub.py`, `fb2.py`, `gdriveutils.py`, `duplicates.py`, `cover.py`, `cw_advocate` for OIDC) together with an automation layer in `scripts/`:

- `scripts/` — `auto_library.py` (watch/scheduled scan → Calibre import), `ingest_processor.py` (queue worker: dedupe, convert, sidecar metadata), `auto_zip.py`, `convert_library.py`, `cover_enforcer.py`, `kindle_epub_fixer.py`, `audiobook.py`, `cwa_db.py` (own SQLite state + `cwa_schema.sql`), `cwa_internal_auth.py` (internal service auth), `generate_book_checksums.py`, SPDX header tooling.
- `Dockerfile`, `docker-compose*.yml`, `kubernetes/` — single-container deployment (Calibre + Calibre-Web + cron-style scanners) and K8s manifests.
- `koreader/plugins/cwasync.koplugin/` — KOReader sync companion plugin.
- `cps.py`, `dirs.json`, `empty_library/` — entrypoint and bootstrapped library layout; `tests/` + `run_tests.sh` for the automation scripts.
- Flow: files dropped into watched folders → `ingest_processor` normalizes/converts and registers in the Calibre DB → Calibre-Web (`cps/`) serves the web UI, OPDS, and Kobo-native sync endpoints → send-to-device, email, audiobook assembly, and cover enforcement run as script jobs → translations compiled via Babel (`*.po`, `messages.pot`).

## Notable implementation techniques

- Practical "glue architecture": instead of forking features into the app, most behavior is added by scripts mutating the Calibre library + a small internal auth boundary (`cwa_internal_auth.py`) between the appliance layer and the web app.
- `check_spdx_headers.py`/`update_spdx_headers.py`: mechanically enforced per-file SPDX license headers — cheap license hygiene Helix could adopt.
- Kobo sync + KOReader plugin shipped side by side, a working precedent for Helix's KOSync compatibility scope.
- Translation pipeline (Weblate/Crowdin, `fix_po_duplicates.py`, `generate_translation_status.py`) keeps i18n cost low for a small team.

## License review

- Root `LICENSE`: GNU GPL v3; `scripts/` carry `SPDX-License-Identifier: GPL-3.0-or-later` headers. The vendored `cps/` fork inherits Calibre-Web's GPL (unresolved-UPSTREAM attribution files present), and bundled JS assets carry their own permissive notices.
- Net effect: all application code in this repo is GPL-3.0-family copyleft. Docker/K8s configuration and the automation scripts are GPL too (no separation by directory).

## Porting decision — attributed

**Decision: LEARN ONLY — operational/design reference; NO code port.** (Reviewer: Helix research agent, from pinned checkout `4fc2c445`.)

Rationale: GPL-3.0 copyleft over the whole tree; Helix's Go server and web client do not take GPL derivative code. The transferable knowledge is process-shaped: watched-folder ingest as an idempotent queue worker over a canonical library store, the separate appliance SQLite (`cwa_db.py`) kept out of the app database, internal service auth for sidecar scripts, and mechanically-enforced SPDX headers. The Kobo sync surface reinforces requirements Helix already tracks as gaps (Kobo-specific sync, watch folders, OIDC). Recorded as `researched_not_ported_license` in the porting ledger.
