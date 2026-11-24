# Workable-item ID key rename: ATM -> HRD

Date: 2026-11-23. Decision source: operator instruction that every project's
ticket-id key derives from the project name (constitution section 11.4.54,
per-project key extension). Helix Reader's key is `HRD`; `ATM` belongs to a
different project and was wrongly hardcoded here.

## Mapping

Every identifier `ATM-NNN` became `HRD-NNN` with the same number (HRD-001 ..
HRD-035). Numbers were not renumbered, reused, or skipped.

## Where the key now lives

`scripts/workable_items.py` -> `DEFAULT_PROJECT_KEY = "HRD"` (override with env
`HELIX_WORKABLE_ITEMS_KEY`). `ID_RE`, minting, markdown parsing, validation
messages, and verifier checks all derive from it via `set_project_key()`.
Table headers use the generic word `ID`. Test `tests/test_workable_items.py::ProjectKeyTest`
proves a different key (`ZQX`) works end to end.

The SQLite column is still named `atm_id`: the constitution lists `atm_id` as a
legacy NAME that is not renamed without a schema migration (and the shared
engine reads it). Its VALUES are all `HRD-NNN`.

## Deliberately preserved historical artifacts

Raw logs, JSON results, scripts and narrative summaries under
`evidence/campaign-2026-11-20/**` and `evidence/campaign-2026-11-23/**`, and
`evidence/workable-items/results-2026-11-20T2036-before-pdf-history-fix.json`,
were captured while the old key was in use and still say `ATM-NNN`. They are
evidence of what ran and were not rewritten. Read `ATM-NNN` there as
`HRD-NNN`.

## Go test identifiers and artifact names (ATM-027 -> HRD-027)

Renamed in `server/internal/httpapi/load_{concurrent,jobs_multiinstance}_test.go`
and `load_remote_{webdav,throttle,rclone}_test.go`: identifiers `atm027*` ->
`hrd027*`, search term `ATM027` -> `HRD027`, env vars `HELIX_ATM027_*` ->
`HELIX_HRD027_*` (BROWSE_CLIENTS, MERGE_WORKERS, OPDS_CLIENTS, OVERLAP_MS,
RECONCILE_WORKERS, SEED_WORKS, TEXT_CLIENTS; no fallback, no script sets them),
artifact prefix `atm027_ingest_concurrent_with_reads_*` ->
`hrd027_ingest_concurrent_with_reads_*`, temp dir/tag `atm027src-`, and the
`host_atm027_overlap` label. Existing files under
`evidence/campaign-2026-11-20/load/` named `atm027_*` were left as captured.
