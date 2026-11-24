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

Lifecycle commands are `update`, `start`, `pause`, `close`, `reopen`, `operator-block`, `unblock`, and `rehash`. `start` moves `Queued` or `Reopened` to `In progress`; `pause` moves `In progress` back to `Queued`; both require `--actor`, `--reason` and `--evidence` (existing file(s) or directory under `evidence/`, `;`-separated), record a `Started`/`Paused` history event, run in one transaction with the integrity hash refreshed, and refuse every other transition (closed statuses, `Operator-blocked`, unknown ids) with exit code 2. Closure derives `Fixed`, `Implemented`, or `Completed` from the item type. Reopening requires an actor, reason, and positive evidence. Operator blocking requires `Attempted:` exhaustion details, a concrete unblock condition, and an owner.

Hardening rules (independent reviews of the start/pause change):

- Evidence references on `start`/`pause` must name an existing regular file or directory strictly inside `evidence/` (a directory is legitimate: an evidence bundle such as a campaign folder). Absolute paths, `.`, `..` escapes, bare directories such as `scripts` or `docs/..`, the `evidence` root itself, symlinks resolving outside `evidence/`, and FIFOs/sockets/devices are refused. Stored references are normalised POSIX (`./a//b/` becomes `a/b`) and every `;`-separated reference is kept in the history event.
- Every free-text value the tool records is refused if it contains line breaks, C0/C1/DEL controls (tab excepted), the Unicode line/paragraph separators, or invisible/bidi characters (U+00AD, U+200B-200F, U+202A-202E, U+2060, U+2066-2069, U+FEFF). That covers `add` (title, description, scope, reproduction, acceptance, anchor, created-by, assigned-to), `update`, `operator-block` (what, why, unblock, who) and the actor/reason/evidence of `update`, `start`, `pause`, `close`, `reopen`, `rehash`. No stored value in the live registry is multi-line, so exports stay byte-identical. As defence in depth the Markdown renderer flattens any such character, `validate` reports stored values that contain them, and `import-md` reads history events only from each item's `**History:**` section.
- `validate` reads the stored integrity hash and recomputes the content inside one read transaction, and fails on a mismatch (or a missing hash in a non-empty registry). The hash covers items, history, reopen/operator-block/obsolete details, and (when they hold rows) `firebase_metadata` and `sync_outbox`; they are keyed in only when non-empty, so a registry that does not use them keeps its existing hash and needs no migration (the live registry's hash was unchanged by this change).
- Every writer (`add`, `update`, `start`, `pause`, `close`, `reopen`, `operator-block`, `import-md`, `export`) verifies the stored hash against the recomputed content BEFORE writing and refuses with `registry tampered; run validate` (exit 2), so a later tool write cannot launder an out-of-band edit. The one deliberate way past it is `rehash --actor {AI|User} --reason TEXT`: it refuses if anything other than the hash is wrong, then records actor, reason, time, previous and new hash in `meta.rehash_log` and sets `last_sync_direction=rehash`. Raw-SQL edits to the registry therefore need a reviewed `rehash`.
- Any refusal (`ValueError`) from any command ends as `error: ...` with exit code 2, and a held write lock ends as exit 2 too (other `OperationalError`s are not reported as lock errors). `HELIX_WORKABLE_ITEMS_BUSY_MS` shortens the 30 s lock wait.
- `docs/tracking/helixtrack-outbox.json` carries a wall-clock `generated_at`; it is the only non-deterministic field, the outbox is not part of `diff`, and regenerating it without registry changes changes only that timestamp.
- `scripts/verify_workable_items.py` works on a read-only backup copy of `docs/workable_items.db` (it never opens the tracked database read-write); only `evidence/workable-items/results.json` is rewritten.
- Round 4 additions. Evidence has two tiers: `start`/`pause` (tier 1) require an existing file or directory strictly under `evidence/`; `close`, `reopen` and `update --evidence` (tier 2) require a relative path that exists, is a regular file or directory, and stays inside the repository root once `..` and symlinks are resolved (`;`-separated, stored normalised). Existing registry rows are not re-checked, so legacy annotated references stay valid; only new writes and `import-md` of terminal items (confinement only, no existence check) are held to the rules.
- Every writer runs inside one `BEGIN IMMEDIATE` transaction: it verifies the stored hash and records the pre-existing validation errors, writes, and before `COMMIT` re-validates; any newly introduced validation error (short description, blank title or scope, control characters, closure/reopen/block rules, a rehash-log break) rolls the whole write back. `import-md` is covered the same way and additionally refuses to change an existing item unless `--allow-overwrite` is given (the row-level plan is printed as `overwritten`; `plan_import` produces it). `close` of a closed item, `reopen` of a non-closed item and `operator-block` of a closed item are refused; `close` of a blocked item removes its `operator_block_details` row (the block stays in history).
- `rehash` needs `--confirm <candidate hash>` (obtain it and the list of items changed since the last tool write with `rehash --dry-run`), refuses a no-op, an empty reason, and any registry that has other problems or contains triggers/views. Entries in `meta.rehash_log` are append-only, hash-chained (`prev_entry`), name the changed items (from the per-item baseline `meta.item_hashes`, refreshed by every tool write; absent on a registry that predates it, in which case `changed` is null), and the log and baseline are covered by the integrity hash once present. The integrity hash is NOT a security boundary: anyone who can write the database file can also run `rehash` or recompute the hash. It is a tripwire plus audit trail; the git history of the tracked `docs/workable_items.db` is the real control, so review database diffs in git.
- `;` followed by `evidence:` (any case) is refused in actor/reason/evidence because the Markdown history and reopen lines are split on it when re-imported. Markdown output neutralises `![`, `](` and `<tag` openers (reversed by `import-md`) and the summary tables escape backslash before pipe; text without those sequences is unchanged, so existing exports are byte-identical. The invisible-character set also covers U+034F, U+061C, U+115F, U+180E, U+2061-2064, U+2800, U+3164, U+FFF9-FFFB and the tag characters U+E0000-E007F. Triggers and views make `validate` fail and every writer refuse.
- Round 5 additions. `import-md` plans and gates changes to `reopen_details` and `operator_block_details` as well as item columns; `--allow-overwrite` now also requires `--actor` and `--reason`, which are appended with the row-level changes to `meta.import_log` (covered by the integrity hash). Import also refuses duplicate ids, unrecognised history lines (instead of dropping them), and any unconfined evidence in terminal items, reopen details and the history of new items. `;assigned to` in created-by/assigned-to and the ` WHY: `, ` UNBLOCK CONDITION: `, ` WHO: ` labels in the operator-block fields are refused (they are split on when re-imported). `unblock ATM --actor --reason --evidence` moves an Operator-blocked item back to Queued with a recorded `Unblocked` event and removes its block details. Global validation errors name the affected ids so a worse state with the same wording is detected as new. `helixtrack-outbox` refuses a tampered registry. `validate` and `diff` open the database read-only (no DDL, no WAL pragma) and fail with a clean `error:` and exit code 2 when the file cannot be opened.
- The verifier's mutant sandboxes are seeded with the evidence files the tests reference, and each target test must pass unmutated (a recorded `mutation-control-*` check) before its mutants can count as killed; a failing control reports the mutants as INVALID.

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

## Known limits (accepted residual risk)

- `md_unescape` is lossy for text that literally contains a backslash followed by `<`, `](` or `![` (it is read back as the unescaped form).
- The integrity hash is a tripwire, not a security boundary. Anyone able to write the database file can recompute the hash, poison the per-item baseline, or run `rehash`; the git history of the tracked `docs/workable_items.db` is the real control.
- Editing the last `meta.rehash_log` entry is covered by the integrity hash but, with no later entry to chain to, the chain check alone cannot show it; only the hash (and git) can.
- `rehash` cannot repair a broken `rehash_log` chain (validation fails first); recovery is `git restore` of the database or a reviewed manual repair.
- Closure, reopen and `update --evidence` references must exist, so an annotated reference such as `file.go(TestX)` is refused on new writes (legacy rows are untouched).
- An Operator-blocked item leaves that state only through `unblock` (to Queued) or `close`.
- Edits to the history lines of EXISTING items in imported Markdown are silently ignored: `import-md` takes history only for new items and never rewrites the stored history of an item already in the registry.
- An open (non-terminal) item's Evidence field is not confined on import: confinement and existence rules apply to terminal items, reopen details and new-item history only (known limit).
- The pre-existing import-md round-trip hash issue (a rebuilt registry's recomputed hash differs from the original) remains out of scope.
