# Registry qualification restored — DEFECT-002 tail, and a test file that did not exist

Generated 2026-11-20. Verdict in this directory: `verifier-2026-11-20.log` (last line
`{"fail": 0, "pass": 11, "status": "PASS"}`), authoritative machine output in
`../../workable-items/results.json`.

## Two defects, both found by trying to run the documented command

**DEFECT-002 tail — path portability is closed.** Every `scripts/verify_*.py` was
rewritten off the dead machine's hardcoded `/mnt/agents/output/helix/...` paths, but
no `py_compile` had ever been run on them (recorded as unverified in the campaign
plan). All 10 Python modules in `scripts/` and `tests/` now compile under Python
3.14, and `verify_workable_items.py` writes its results to
`/home/milosvasic/Projects/helix_reader/evidence/workable-items/results.json` — the
previous artifact named `/mnt/agents/output/helix/...`, which is what made the old
PASS unreadable here.

**DEFECT-004 — `tests/test_workable_items.py` did not exist, and the verifier
required it.** `docs/workable-items.md` documented
`python3 -m unittest -v tests.test_workable_items`, and
`scripts/verify_workable_items.py:125` copies that file into a scratch root to run its
four mutation gates. The file was absent from the working tree, absent from `HEAD`,
and absent from every commit (`git ls-tree -r HEAD` → 0 entries under `tests/`), yet
`evidence/workable-items/unit.log` shows 12 tests of exactly that module name running
on 2026-10-28. So it lived only on the dead machine and was never committed: the
registry qualification had been un-runnable since the migration, and its last PASS
pointed at paths that no longer resolve.

Rather than delete the gates, the test module was reconstructed from its contract:
the 12 test names in `unit.log`, the four method names the mutation gates invoke, and
the assertion shapes still visible in the gate details inside
`evidence/workable-items/final-verifier.log` (e.g. the attribution mutant produced a
1,207-character `Created by UNKNOWN` vs `by AI` diff at line 170). `tests/__init__.py`
was added so `tests.test_workable_items` is importable as documented.

## What the reconstructed suite pins

`validate()` fail-closed behavior (status/type/severity vocabulary, status↔location
agreement, description sufficiency, terminal-needs-evidence, reopen attribution,
operator-block `Attempted:` audit, reopen-count-vs-history, gapless allocation,
`PRAGMA foreign_key_check`); `add_item` monotonic identifier allocation; 8-thread
concurrent `add_item` producing 16 unique gapless identifiers under `BEGIN IMMEDIATE`;
type-aware closure vocabulary plus alias rejection in `close_item`; the full
close → reopen → update lifecycle with history ordering; operator-block command
round-trip through generated Markdown; pipe/HTML escaping in the summary table and the
HTML documents; `export` → `diff` drift detection including "nothing generated yet is
drift"; Markdown → database → Markdown byte equality with `integrity()` hash
equality; and the outbox never claiming `remote_status` other than `NOT RUN`.

Five assertions of mine were wrong about the real API and were corrected against
observed behavior rather than assumed: `diff()` reports *every* file as drift before
the first export (missing output is drift, not "no opinion"); `render_item` does not
escape pipes because long-form bodies are not table cells — only `summary_rows` does;
`utcnow()` is second-resolution so two regenerations in one second share a
`generated_at`; deleting a middle row is surfaced as `ATM allocation contains gaps`
rather than being silently legal; and the identifier allocator never reuses a number
even after a gap exists.

## Gate results

| Check | Result |
|---|---|
| `python3 -m unittest tests.test_workable_items` | 12 tests, OK |
| `verify_workable_items.py` | 11/11 PASS, exit 0 |
| `mutation-closure-vocabulary-gate` | mutant rejected — `test_type_aware_closure_vocabulary` FAILs |
| `mutation-markdown-drift-gate` | mutant rejected — `test_export_and_drift_detection` FAILs |
| `mutation-remote-push-anti-bluff` | mutant rejected — `test_outbox_never_claims_remote_push` FAILs |
| `mutation-attribution-roundtrip` | mutant rejected — `test_markdown_database_markdown_round_trip` FAILs |

Each named mutant fails its named test, which is the only reason the gates mean
anything; a gate that passes because the test file was missing would have been the
same defect that hid here in the first place.

## New gate: PDF currency

Running the export surfaced a third problem. `docs/*.pdf` were dated 2026-10-28 while
`docs/*.md` and `docs/*.html` had been regenerated on 2026-11-20, and **`diff()`
cannot see that** — it compares Markdown and HTML only. The export had in fact
crashed partway (after writing Markdown, before PDFs) because ReportLab was missing;
the failure was loud, but the resulting inconsistency was silent.

ReportLab was installed out of tree (`pip install --target ~/.cache/helix-docs-deps`)
rather than overriding the host's PEP 668 marker, the full document set was
regenerated, and export was confirmed byte-idempotent (second export leaves identical
Markdown/HTML hashes). `pdf-documents-current` was then added to the verifier: size,
page count, and the ATM identifier set extracted by `pdftotext` must match the
Markdown sibling.

Needle, run against the retained October artifacts: with the 2026-10-28 PDFs in place
the check **FAILs**, reporting `only in PDF: ['ATM-032']` for `Issues.md` and the
converse for `Fixed.md` — ATM-032 was still open when those PDFs were made and is
closed now. With the regenerated set it PASSes (21 closed / 12 open identifiers on
both sides). That is the detection power, demonstrated on the exact stale state the
check exists to catch.

## Boundary

`verify_workable_items.py` now needs `pdftotext`/`pdfinfo` (poppler) and records FAIL
without them rather than skipping — a currency check that silently no-ops is how this
directory ended up claiming a PASS for a machine nobody can reach. The PDFs are
derived artifacts: they are checked against the registry, never edited, and the
registry remains the only authoritative store.
