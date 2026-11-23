# Workable-items registry

`docs/workable_items.db` is the authoritative SQLite registry. The initial 33-item baseline was reconstructed from preserved repository evidence on 2026-10-29. Registry timestamps identify that import and later lifecycle operations; they do not claim to be original implementation dates.

## Derived documents

The following files are generated, not independently edited:

- `Issues.md`, `Issues.html`, `Issues.pdf`
- `Issues_Summary.md`, `Issues_Summary.html`, `Issues_Summary.pdf`
- `Fixed.md`, `Fixed.html`, `Fixed.pdf`
- `Fixed_Summary.md`, `Fixed_Summary.html`, `Fixed_Summary.pdf`
- `tracking/helixtrack-outbox.json`

The generator enforces stable gapless HRD identifiers, the closed Type and Status vocabularies, type-aware closure wording, location consistency, comprehensive descriptions, closure evidence, reopen attribution, operator-block exhaustion details, reopen counts, and SQLite foreign keys.

## Commands

Run from the repository root:

```bash
python3 scripts/workable_items.py validate
python3 scripts/workable_items.py diff
python3 scripts/workable_items.py export
python3 scripts/workable_items.py helixtrack-outbox
```

Add work with comprehensive fields:

```bash
python3 scripts/workable_items.py add Task High \
  --title 'Short self-contained title' \
  --description 'A complete description of the work and its context.' \
  --scope 'Affected services, clients, and documents.' \
  --reproduction 'Observable current behavior or missing capability.' \
  --acceptance 'Observable conditions required for closure.'
```

Lifecycle commands are `update`, `close`, `reopen`, and `operator-block`. Closure derives `Fixed`, `Implemented`, or `Completed` from the item type. Reopening requires an actor, reason, and positive evidence. Operator blocking requires `Attempted:` exhaustion details, a concrete unblock condition, and an owner.

For disaster recovery, `import-md` reconstructs item rows, attribution, composition references, history, and conditional lifecycle details from the generated long-form Markdown:

```bash
python3 scripts/workable_items.py --db /path/to/recovered.db import-md --input docs
python3 scripts/workable_items.py --db /path/to/recovered.db export --output /path/to/recovered-docs
```

PDF export requires ReportLab. A missing PDF dependency fails export rather than silently omitting a mandatory document. On an externally managed (PEP 668) host, install it out of tree instead of overriding the marker:

```bash
python3 -m pip install --target ~/.cache/helix-docs-deps reportlab
PYTHONPATH=~/.cache/helix-docs-deps python3 scripts/workable_items.py export --output docs
```

`diff()` compares generated Markdown and HTML; it cannot see a PDF left behind by an export that died partway. `verify_workable_items.py` therefore adds `pdf-documents-current`, which extracts the HRD identifiers from each PDF with `pdftotext` and requires the same set its Markdown sibling carries, plus a non-trivial byte size and page count. A PDF set that predates the current registry fails that check rather than passing the drift gate — verified against the 2026-10-28 set, which still showed HRD-032 as open.

## HelixTrack boundary

`helixtrack-outbox.json` is an offline, idempotent upsert plan keyed by HRD identifier. Every record says `remote_status: NOT RUN`. It is not an API-compatible remote receipt and does not prove synchronization. Live bidirectional HelixTrack delivery remains HRD-029 until the exact API contract, authentication, conflict semantics, and real-instance push/pull results are available.

## Qualification

Run:

```bash
python3 scripts/verify_workable_items.py
python3 -m unittest -v tests.test_workable_items
```

Machine results are in `evidence/workable-items/results.json`. The campaign covers real SQLite CRUD and lifecycle behavior, concurrent writers, a 1,000-row transaction, SIGKILL rollback, corrupt-database detection and recovery, byte-identical round trips, drift detection, and rejecting mutations. PDF visual-inspection artifacts are under `evidence/workable-items/pdf-qa`.
