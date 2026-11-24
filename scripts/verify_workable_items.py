#!/usr/bin/env python3
"""Anti-bluff qualification for the real SQLite workable-items registry."""
from __future__ import annotations

import argparse
import importlib.util
import json
import os
from pathlib import Path
import re
import resource
import shutil
import signal
import sqlite3
import subprocess
import sys
import tempfile
import time

ROOT = Path(__file__).resolve().parents[1]
EVIDENCE = ROOT / "evidence" / "workable-items"
SPEC = importlib.util.spec_from_file_location("wi", ROOT / "scripts" / "workable_items.py")
wi = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(wi)
RESULTS = []


def record(name, status, detail, **extra):
    row = {"name": name, "status": status, "detail": detail, **extra}
    RESULTS.append(row)
    print(json.dumps(row, sort_keys=True), flush=True)


def run(name, command, cwd=ROOT):
    completed = subprocess.run(command, cwd=cwd, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    record(name, "PASS" if completed.returncode == 0 else "FAIL", completed.stdout[-6000:], returncode=completed.returncode, command=command)
    return completed.returncode == 0


def live_snapshot(root: Path) -> Path:
    """Copy the authoritative registry through a READ-ONLY connection (SQLite backup API).

    Every gate below that needs the real registry's content works on this copy, so the
    verifier never opens docs/workable_items.db read-write and cannot leave the tracked
    database (or its -wal/-shm files) modified."""
    target = root / "live-snapshot.db"
    source = sqlite3.connect(f"file:{ROOT / 'docs' / 'workable_items.db'}?mode=ro", uri=True)
    copy = sqlite3.connect(target)
    source.backup(copy)
    copy.close()
    source.close()
    return target


def stress(root: Path):
    path = root / "stress.db"
    db = wi.connect(path)
    db.execute("BEGIN IMMEDIATE")
    now = "2026-10-29T00:00:00Z"
    for number in range(1, 1001):
        item = {
            "atm_id": wi.item_id(number), "type": "Task", "status": "Queued", "severity": "Low",
            "title": f"Stress registry row {number}", "description": "A complete stress fixture description with enough meaningful words and characters.",
            "affected_scope": "One thousand real SQLite registry rows.", "reproduction": "Insert one thousand rows in a real transaction.",
            "acceptance": "All rows persist with unique gapless identifiers and valid lifecycle fields.", "created_at": now, "last_modified": now,
            "current_location": "Issues", "forensic_anchor": "Stress fixture", "closure_criteria": "One thousand rows validate.", "composes_with": "[]",
            "evidence_path": "", "created_by": "AI", "assigned_to": "Operator", "reopens_count": 0,
        }
        db.execute(f"INSERT INTO items({','.join(wi.ITEM_COLUMNS)}) VALUES({','.join('?' for _ in wi.ITEM_COLUMNS)})", [item[c] for c in wi.ITEM_COLUMNS])
        db.execute("INSERT INTO item_history(atm_id,event_type,actor,on_date,reason,evidence_path) VALUES(?,?,?,?,?,?)", (item["atm_id"], "Opened", "AI", "2026-10-29", "stress-fixture", ""))
    wi.update_meta(db, "stress")
    db.execute("COMMIT")
    count = db.execute("SELECT count(*) FROM items").fetchone()[0]
    problems = wi.validate(db)
    check = db.execute("PRAGMA integrity_check").fetchone()[0]
    db.close()
    record("stress-1000-real-sqlite-rows", "PASS" if count == 1000 and not problems and check == "ok" else "FAIL", f"rows={count} integrity={check} validation_errors={problems}")


def kill_and_recovery(root: Path):
    path = root / "kill.db"
    db = wi.connect(path)
    db.close()
    child_code = '''
import importlib.util,os,sys,time
spec=importlib.util.spec_from_file_location("wi",sys.argv[1]);wi=importlib.util.module_from_spec(spec);spec.loader.exec_module(wi)
db=wi.connect(wi.Path(sys.argv[2]))
db.execute("CREATE TABLE committed_marker(id INTEGER PRIMARY KEY,value TEXT)")
db.execute("INSERT INTO committed_marker VALUES(1,'committed')")
db.execute("BEGIN IMMEDIATE")
for n in range(2,10000): db.execute("INSERT INTO committed_marker VALUES(?,?)",(n,"uncommitted"))
print("READY",flush=True)
time.sleep(60)
'''
    child = subprocess.Popen([sys.executable, "-c", child_code, str(ROOT / "scripts" / "workable_items.py"), str(path)], stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    ready = child.stdout.readline().strip()
    if ready == "READY":
        child.kill()
    child.wait(timeout=10)
    db = wi.connect(path)
    count = db.execute("SELECT count(*) FROM committed_marker").fetchone()[0]
    integrity = db.execute("PRAGMA integrity_check").fetchone()[0]
    db.close()
    record("sigkill-uncommitted-transaction-recovery", "PASS" if ready == "READY" and count == 1 and integrity == "ok" else "FAIL", f"ready={ready} committed_rows={count} integrity={integrity} child_returncode={child.returncode}")


def corruption_and_md_recovery(root: Path, source: Path):
    corrupt = root / "corrupt.db"
    shutil.copyfile(source, corrupt)
    with corrupt.open("r+b") as handle:
        handle.seek(100)
        handle.write(b"CORRUPT-HELIX-TEST")
    detected = False
    message = ""
    try:
        db = sqlite3.connect(corrupt)
        db.execute("PRAGMA integrity_check").fetchall()
        db.close()
    except sqlite3.DatabaseError as exc:
        detected = True
        message = str(exc)
    recovered = wi.connect(root / "recovered-after-corruption.db")
    count = wi.import_markdown(recovered, ROOT / "docs")
    valid = wi.validate(recovered)
    recovered.close()
    record("corrupt-db-detection-and-md-recovery", "PASS" if detected and count == len(wi.items(wi.connect(source))) and not valid else "FAIL", f"corruption_detected={detected} error={message!r} recovered_rows={count} validation_errors={valid}")


def mutations(root: Path):
    cases = [
        ("closure-vocabulary-gate", 'if item["status"] in TERMINAL and item["status"] != "Obsolete (→ Fixed.md)" and item["status"] != CLOSED.get(item["type"]):', 'if False and item["status"] in TERMINAL:', "test_type_aware_closure_vocabulary"),
        ("markdown-drift-gate", 'if not path.exists() or path.read_bytes() != content.encode("utf-8"):', 'if False:', "test_export_and_drift_detection"),
        ("remote-push-anti-bluff", '"payload": payload, "remote_status": "NOT RUN",', '"payload": payload, "remote_status": "pushed",', "test_outbox_never_claims_remote_push"),
        ("attribution-roundtrip", 'created_by = "" if created_by == "UNKNOWN" else created_by', 'created_by = ""', "test_markdown_database_markdown_round_trip"),
        # HTML history rendering is the fix for the pdf-documents-current defect: if the
        # generator drops history from one format again, this mutant must be rejected.
        # A progress event that loses its artifact pointer would make the history
        # uncheckable, so dropping it on update must break a test.
        ("update-evidence-gate", 'history(db, args.atm_id, "Updated", args.actor, args.reason, evidence)', 'history(db, args.atm_id, "Updated", args.actor, args.reason, "")', "test_close_reopen_and_update_lifecycle"),
        # Status-transition gates (start/pause): a Fixed item moving to In progress, the
        # legality check vanishing, evidence dropped from history, or the
        # transaction being left unrolled must each be rejected by a test.
        ("transition-closed-to-in-progress",
         '    "In progress": {"Queued"},\n',
         '    "In progress": {"Queued"},\n    "Fixed (→ Fixed.md)": {"In progress"},\n',
         "test_illegal_transitions_are_refused_without_side_effects"),
        ("transition-legality-check",
         'if target not in TRANSITIONS.get(current, set()):',
         'if False:',
         "test_illegal_transitions_are_refused_without_side_effects"),
        ("transition-evidence-in-history",
         'history(db, args.atm_id, TRANSITION_EVENTS[target], args.actor, args.reason, "; ".join(evidence))',
         'history(db, args.atm_id, TRANSITION_EVENTS[target], args.actor, args.reason, "")',
         "test_start_pause_and_restart_record_history"),
        ("transition-evidence-existence",
         'if bad:\n        raise ValueError(f"evidence must be',
         'if False:\n        raise ValueError(f"evidence must be',
         "test_transition_requires_reason_and_existing_evidence"),
        ("transition-atomicity", '        db.execute("ROLLBACK")\n        raise\n\n\ndef unblock_item(', '        raise\n\n\ndef unblock_item(', "test_transition_is_atomic_under_injected_failure"),
        # Review-finding mutants: each must be killed by the named test.
        ("transition-actor-hardcoded",
         'history(db, args.atm_id, TRANSITION_EVENTS[target], args.actor, args.reason,',
         'history(db, args.atm_id, TRANSITION_EVENTS[target], "AI", args.reason,',
         "test_transition_records_the_supplied_actor"),
        ("transition-in-progress-extra-exit",
         '    "In progress": {"Queued"},\n',
         '    "In progress": {"Queued", "Reopened", "Completed (→ Fixed.md)"},\n',
         "test_in_progress_has_exactly_one_exit"),
        ("transition-begin-deferred", '    db.execute("BEGIN IMMEDIATE")\n    try:\n        base = write_baseline(db)\n        item = require_item(db, args.atm_id)\n        current', '    db.execute("BEGIN")\n    try:\n        base = write_baseline(db)\n        item = require_item(db, args.atm_id)\n        current', "test_transition_takes_the_write_lock_at_begin"),
        ("cli-pause-acts-as-start",
         'args.target = "In progress" if args.command == "start" else "Queued"',
         'args.target = "In progress"',
         "test_pause_cli_end_to_end_and_locked_database_exit"),
        ("cli-locked-traceback", 'if "locked" in text:', 'if False:', "test_pause_cli_end_to_end_and_locked_database_exit"),
        ("evidence-escapes-repo",
         'if target == allowed or not target.is_relative_to(allowed):',
         'if False:',
         "test_transition_evidence_is_confined_to_the_evidence_tree"),
        ("evidence-absolute-allowed",
         'if not part or Path(part).is_absolute():',
         'if not part:',
         "test_transition_evidence_is_confined_to_the_evidence_tree"),
        ("history-control-chars-allowed",
         'if value and CONTROL_RE.search(value):',
         'if False:',
         "test_history_fields_refuse_control_characters"),
        ("export-history-unescaped", 'flat = (lambda value: md_inline(CONTROL_RE.sub(" ", value or ""))) if md else (lambda value: CONTROL_RE.sub(" ", value or ""))', 'flat = lambda value: value or ""', "test_forged_history_cannot_survive_export_and_import"),
        ("validate-ignores-integrity-hash",
         'if stored[0] != integrity(db):',
         'if False:',
         "test_validate_detects_tampering_via_integrity_hash"),
        # Second review round.
        ("evidence-not-normalised", 'evidence = normalised', 'pass', "test_transition_evidence_is_confined_to_the_evidence_tree"),
        ("evidence-special-files-allowed", 'if not (stat.S_ISREG(mode) or stat.S_ISDIR(mode)):', 'if False:', "test_transition_evidence_is_confined_to_the_evidence_tree"),
        ("charclass-drops-vt-ff-nel", r'\x0a-\x1f\x7f-\x9f', r'\x0a\x0d', "test_history_fields_refuse_bidi_zero_width_and_exotic_controls"),
        ("charclass-drops-bidi-zero-width", '\\u200b-\\u200f\\u2028-\\u202e', '\\u2028', "test_history_fields_refuse_bidi_zero_width_and_exotic_controls"),
        ("add-free-text-unchecked", ['refuse_control_chars(title=args.title, description=args.description, scope=args.scope,', 'if CONTROL_RE.search(item[column] or ""):'], ['refuse_control_chars(scope=args.scope,', 'if False:'], "test_free_text_fields_refuse_line_breaks_and_invisible_controls"),
        ("block-free-text-unchecked", 'refuse_control_chars(what=args.what, why=args.why, unblock=args.unblock, who=args.who)', 'pass', "test_free_text_fields_refuse_line_breaks_and_invisible_controls"),
        ("update-free-text-unchecked", ['refuse_control_chars(**{key: value for key, value in changes.items() if isinstance(value, str)})', 'if CONTROL_RE.search(item[column] or ""):'], ['pass', 'if False:'], "test_free_text_fields_refuse_line_breaks_and_invisible_controls"),
        ("render-not-flattened", 'flat = lambda value: md_inline(CONTROL_RE.sub(" ", value))', 'flat = lambda value: value', "test_description_line_cannot_forge_history_through_export_and_import"),
        ("history-parsed-from-whole-body", 'HISTORY_LINE_RE.findall(history_section)', 'HISTORY_LINE_RE.findall(body)', "test_history_is_parsed_only_from_the_history_section"),
        ("add-skips-tamper-check", 'try:\n        base = write_baseline(db)\n        maximum', 'try:\n        base = set()\n        maximum', "test_every_writer_refuses_a_tampered_registry_before_writing"),
        ("close-skips-tamper-check", 'base = write_baseline(db)\n        item = require_item(db, args.atm_id)\n        if item["status"] in TERMINAL:\n            raise ValueError(f\'{args.atm_id}: already closed', 'base = set()\n        item = require_item(db, args.atm_id)\n        if item["status"] in TERMINAL:\n            raise ValueError(f\'{args.atm_id}: already closed', "test_every_writer_refuses_a_tampered_registry_before_writing"),
        ("reopen-skips-tamper-check", 'base = write_baseline(db)\n        item = require_item(db, args.atm_id)\n        if item["status"] not in TERMINAL:', 'base = set()\n        item = require_item(db, args.atm_id)\n        if item["status"] not in TERMINAL:', "test_every_writer_refuses_a_tampered_registry_before_writing"),
        ("block-skips-tamper-check", 'base = write_baseline(db)\n        item = require_item(db, args.atm_id)\n        if item["status"] in TERMINAL:\n            raise ValueError(f\'{args.atm_id}: a closed', 'base = set()\n        item = require_item(db, args.atm_id)\n        if item["status"] in TERMINAL:\n            raise ValueError(f\'{args.atm_id}: a closed', "test_every_writer_refuses_a_tampered_registry_before_writing"),
        ("transition-skips-tamper-check", 'base = write_baseline(db)\n        item = require_item(db, args.atm_id)\n        current', 'base = set()\n        item = require_item(db, args.atm_id)\n        current', "test_every_writer_refuses_a_tampered_registry_before_writing"),
        ("update-skips-tamper-check", 'base = write_baseline(db)\n        require_item(db, args.atm_id)\n        db.execute("UPDATE items SET " ', 'base = set()\n        require_item(db, args.atm_id)\n        db.execute("UPDATE items SET " ', "test_every_writer_refuses_a_tampered_registry_before_writing"),
        ("import-skips-tamper-check", 'base = write_baseline(db)\n        changes = plan_import', 'base = set()\n        changes = plan_import', "test_every_writer_refuses_a_tampered_registry_before_writing"),
        ("export-skips-tamper-check", '    require_untampered(db)\n    output_dir.mkdir', '    output_dir.mkdir', "test_every_writer_refuses_a_tampered_registry_before_writing"),
        ("integrity-ignores-firebase-and-outbox", 'payload.update({key: rows for key, rows in extra.items() if rows})', 'pass', "test_integrity_covers_every_detail_and_sync_table"),
        ("validate-not-one-snapshot", '    if own:\n        db.execute("BEGIN")\n', '    if False:\n        db.execute("BEGIN")\n', "test_validate_reads_hash_and_content_in_one_snapshot"),
        ("update-skips-meta-refresh", 'update_meta(db, "update")', 'pass', "test_update_is_validated_immediately_after_each_writer"),
        ("transition-keeps-old-last-modified", '(target, utcnow(), args.atm_id))', '(target, item["last_modified"], args.atm_id))', "test_transition_sets_last_modified_and_stores_all_evidence"),
        ("transition-stores-first-evidence-only", 'args.actor, args.reason, "; ".join(evidence))', 'args.actor, args.reason, evidence[0])', "test_transition_sets_last_modified_and_stores_all_evidence"),
        ("busy-env-ignored", 'int(os.environ.get("HELIX_WORKABLE_ITEMS_BUSY_MS", "30000"))', 'int("30000")', "test_busy_timeout_environment_variable_is_honoured"),
        ("every-operational-error-called-locked", 'if "locked" in text:', 'if True:', "test_only_lock_errors_are_reported_as_locked"),
        ("rehash-blesses-invalid-registry", 'if problems:\n            raise ValueError("rehash refused', 'if False:\n            raise ValueError("rehash refused', "test_rehash_is_the_only_way_past_tampering_and_is_recorded"),
        ("rehash-unrecorded", 'update_meta(db, "rehash")', 'update_meta(db, "db-to-docs")', "test_rehash_is_the_only_way_past_tampering_and_is_recorded"),
        ("writer-errors-traceback", '    except ValueError as error:\n        print', '    except KeyError as error:\n        print', "test_writer_refusals_end_in_error_exit_2"),
        # Third review round.
        ("rehash-log-overwritten", '\'rehash_log\',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", ((log[0] + "\\n" if log else "") + entry,))', '\'rehash_log\',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", (entry,))', "test_rehash_refuses_empty_reason_and_logs_chained_appends"),
        ("rehash-no-reason-check", 'if not (args.reason or "").strip():\n        raise ValueError("rehash requires a reason")', 'pass', "test_rehash_refuses_empty_reason_and_logs_chained_appends"),
        ("rehash-confirm-skipped", 'if not getattr(args, "confirm", None) or args.confirm != candidate:', 'if False:', "test_rehash_requires_matching_confirm_and_refuses_noop"),
        ("rehash-noop-allowed", 'if previous == candidate:', 'if False:', "test_rehash_requires_matching_confirm_and_refuses_noop"),
        ("rehash-chain-unchecked", 'if entry.get("prev_entry") != previous:', 'if False:', "test_rehash_chain_break_is_reported"),
        ("rehash-log-not-in-hash", 'payload[key] = row[0]', 'pass', "test_rehash_refuses_empty_reason_and_logs_chained_appends"),
        ("rehash-changed-items-not-listed", '        changed = changed_items(db)\n', '        changed = None\n', "test_rehash_requires_matching_confirm_and_refuses_noop"),
        ("commit-gate-disabled", 'introduced = [e for e in _validate(db, False) if e not in baseline]', 'introduced = []', "test_every_writer_rolls_back_when_the_result_would_be_invalid"),
        ("import-overwrite-unchecked", 'if changes and not allow_overwrite:', 'if False:', "test_import_refuses_silent_overwrite_of_existing_items"),
        ("import-evidence-unconfined", 'repo_evidence(item["evidence_path"], must_exist=False)', 'pass', "test_import_refuses_unconfined_evidence_even_with_overwrite"),
        ("close-evidence-unresolved", 'raise ValueError("closure requires captured evidence")\n    evidence = repo_evidence(args.evidence)', 'raise ValueError("closure requires captured evidence")\n    evidence = args.evidence', "test_closure_evidence_tier_is_repo_confined_and_must_exist"),
        ("reopen-evidence-unresolved", 'raise ValueError("reopen requires positive evidence")\n    evidence = repo_evidence(args.evidence)', 'raise ValueError("reopen requires positive evidence")\n    evidence = args.evidence', "test_closure_evidence_tier_is_repo_confined_and_must_exist"),
        ("update-evidence-unresolved", 'evidence = repo_evidence(args.evidence) if (args.evidence or "").strip() else ""', 'evidence = args.evidence', "test_closure_evidence_tier_is_repo_confined_and_must_exist"),
        ("repo-evidence-escape-allowed", 'ok = target.is_relative_to(root) and target != root and ".git" not in', 'ok = True or ".git" not in', "test_closure_evidence_tier_is_repo_confined_and_must_exist"),
        ("close-already-closed-allowed", 'if item["status"] in TERMINAL:\n            raise ValueError(f\'{args.atm_id}: already closed', 'if False:\n            raise ValueError(f\'{args.atm_id}: already closed', "test_lifecycle_state_rules"),
        ("reopen-non-closed-allowed", 'if item["status"] not in TERMINAL:', 'if False:', "test_lifecycle_state_rules"),
        ("block-closed-allowed", 'if item["status"] in TERMINAL:\n            raise ValueError(f\'{args.atm_id}: a closed item cannot', 'if False:\n            raise ValueError(f\'{args.atm_id}: a closed item cannot', "test_lifecycle_state_rules"),
        ("close-keeps-block-details", 'db.execute("DELETE FROM operator_block_details WHERE atm_id=?", (args.atm_id,))', 'pass', "test_lifecycle_state_rules"),
        ("boundary-token-allowed", ['if value and token and token.search(value):', 'if any(BOUNDARY_RE.search(row[column] or "") for column in ("reason", "evidence_path")):'], ['if False:', 'if False:'], "test_history_field_boundary_tokens_are_refused"),
        ("validate-ignores-boundary-token", 'if any(BOUNDARY_RE.search(row[column] or "") for column in ("reason", "evidence_path")):', 'if False:', "test_validate_flags_boundary_tokens_in_stored_history"),
        ("md-escape-backslash-skipped", r'md_inline(text.replace("\\", "\\\\"))', 'md_inline(text)', "test_markdown_escaping_of_pipes_links_images_and_tags"),
        ("md-inline-image-unescaped", r'text.replace("![", "!\\[")', 'text', "test_markdown_escaping_of_pipes_links_images_and_tags"),
        ("md-unescape-skipped", r'return text.replace("\\<", "<").replace("]\\(", "](").replace("!\\[", "![")', 'return text', "test_markdown_escaping_of_pipes_links_images_and_tags"),
        ("md-history-not-escaped", 'flat = (lambda value: md_inline(CONTROL_RE.sub(" ", value or ""))) if md else', 'flat = (lambda value: CONTROL_RE.sub(" ", value or "")) if md else', "test_markdown_escaping_of_pipes_links_images_and_tags"),
        ("schema-objects-allowed", 'if objects:', 'if False:', "test_triggers_and_views_make_the_registry_invalid"),
        ("tamper-check-ignores-schema", 'if schema:\n        return schema', 'if False:\n        return schema', "test_triggers_and_views_make_the_registry_invalid"),
        ("title-may-be-blank", 'if not item["title"].strip():', 'if False:', "test_every_writer_rolls_back_when_the_result_would_be_invalid"),
        # Fifth review round (PR #8 independent review minors).
        ("charclass-drops-variation-selectors", '\\ufe00-\\ufe0f\\u180b-\\u180d', '', "test_variation_selectors_and_size_cap_and_git_evidence_and_bad_key"),
        ("charclass-drops-vs-supplement", '\\U000e0100-\\U000e01ef', '', "test_variation_selectors_and_size_cap_and_git_evidence_and_bad_key"),
        ("free-text-size-cap-removed", 'if value and len(value) > MAX_FIELD_CHARS:', 'if False:', "test_variation_selectors_and_size_cap_and_git_evidence_and_bad_key"),
        ("evidence-allows-git-dir", ' and ".git" not in target.relative_to(root).parts', '', "test_variation_selectors_and_size_cap_and_git_evidence_and_bad_key"),
        ("bad-key-traceback", 'raise SystemExit(2)', 'raise', "test_variation_selectors_and_size_cap_and_git_evidence_and_bad_key"),
        ("charclass-drops-more-invisibles", '\\u034f\\u061c\\u115f\\u180e', '', "test_more_invisible_characters_are_refused"),
        ("charclass-drops-tag-chars", r'\U000e0000-\U000e007f', r'\U000e0000', "test_more_invisible_characters_are_refused"),
        ("item-hashes-baseline-dropped", 'db.execute("INSERT INTO meta(key,value) VALUES(\'item_hashes\',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", (json.dumps(item_snapshot_hashes(db), sort_keys=True),))', 'pass', "test_schema_item_hashes_baseline_is_maintained"),
        ("history-html-unescaped", '<li>{html.escape(event)}</li>', '<li>{event}</li>', "test_html_export_escapes_history_markup"),
        ("pdf-values-unescaped", 'story.append(Paragraph(f"<b>{label}:</b> {html.escape(value)}", body))', 'story.append(Paragraph(f"<b>{label}:</b> {value}", body))', "test_pdf_export_escapes_markup_in_values_and_history"),
        ("history-order-by-date", 'FROM item_history ORDER BY sequence")],\n        "operator_blocks"', 'FROM item_history ORDER BY on_date")],\n        "operator_blocks"', "test_integrity_detects_history_reordering"),
        ("validate-ignores-history-control-chars", 'if CONTROL_RE.search(row[column] or ""):', 'if False:', "test_validate_reports_control_characters_in_history_rows"),
        ("import-plan-skips-detail-tables", '            if not raw:\n                continue\n', '            continue\n', "test_import_plan_covers_reopen_and_block_detail_tables"),
        ("import-reopen-evidence-unconfined", 'repo_evidence(REOPEN_DETAIL_RE.match(item["_reopened_details"]).group(4), must_exist=False)', 'pass', "test_import_confines_reopen_and_history_evidence_and_rejects_bad_markdown"),
        ("import-history-evidence-unconfined", 'if event["evidence_path"]:\n                        repo_evidence(', 'if False:\n                        repo_evidence(', "test_import_confines_reopen_and_history_evidence_and_rejects_bad_markdown"),
        ("import-duplicate-ids-allowed", 'if item["atm_id"] in seen:', 'if False:', "test_import_confines_reopen_and_history_evidence_and_rejects_bad_markdown"),
        ("history-unknown-event-dropped", 'if not HISTORY_LINE_RE.match(candidate_line):', 'if False:', "test_import_confines_reopen_and_history_evidence_and_rejects_bad_markdown"),
        ("import-overwrite-unrecorded", 'if actor not in ("AI", "User") or not (reason or "").strip():', 'if False:', "test_import_overwrite_requires_actor_reason_and_is_logged_in_the_hash"),
        ("import-log-not-in-hash", '("rehash_log", "item_hashes", "import_log")', '("rehash_log", "item_hashes")', "test_import_overwrite_requires_actor_reason_and_is_logged_in_the_hash"),
        ("integrity-omits-item-hashes", '("rehash_log", "item_hashes", "import_log")', '("rehash_log", "import_log")', "test_integrity_covers_item_hashes_baseline"),
        ("snapshot-omits-history", '(("item_history", "sequence"), ("reopen_details", "atm_id")', '(("reopen_details", "atm_id")', "test_item_snapshot_covers_history"),
        ("export-no-second-tamper-check", '    try:\n        require_untampered(db)\n        update_meta(db, "db-to-docs")', '    try:\n        update_meta(db, "db-to-docs")', "test_export_rechecks_tampering_after_building_documents"),
        ("attribution-label-allowed", ['"created_by": ASSIGN_RE, "assigned_to": ASSIGN_RE,', 'if ASSIGN_RE.search(item[column] or ""):'], ['', 'if False:'], "test_attribution_and_block_labels_cannot_shift_field_boundaries"),
        ("block-label-allowed", ['"what": BLOCK_LABEL_RE, "why": BLOCK_LABEL_RE, "unblock": BLOCK_LABEL_RE, "who": BLOCK_LABEL_RE}', 'if BLOCK_LABEL_RE.search(row[column] or ""):'], ['}', 'if False:'], "test_attribution_and_block_labels_cannot_shift_field_boundaries"),
        ("outbox-skips-tamper-check", '-> None:\n    require_untampered(db)\n    records = []', '-> None:\n    records = []', "test_outbox_refuses_a_tampered_registry"),
        ("validate-opens-read-write", 'db = connect_readonly(args.db) if args.command in ("validate", "diff") else connect(args.db)', 'db = connect(args.db)', "test_read_only_database_validates_without_traceback"),
        ("unblock-any-status", 'if item["status"] != "Operator-blocked":', 'if False:', "test_unblock_is_a_recorded_transition"),
        ("unblock-keeps-block-details", 'db.execute("DELETE FROM operator_block_details WHERE atm_id=?", (args.atm_id,))\n        history(db, args.atm_id, "Unblocked"', 'pass\n        history(db, args.atm_id, "Unblocked"', "test_unblock_is_a_recorded_transition"),
        ("unblock-event-unknown-to-import", '|Paused|Unblocked|', '|Paused|', "test_unblock_is_a_recorded_transition"),
        ("gap-error-unspecific", '", ".join(item_id(n) for n in missing[:20])', '""', "test_validate_errors_name_the_affected_ids"),
        ("foreign-key-error-unspecific", '", ".join(sorted(affected)[:20])', '""', "test_validate_errors_name_the_affected_ids"),
        ("html-history-gate", '                events = history_lines(db, i["atm_id"])\n                if events:', '                events = []\n                if events:', "test_derived_documents_carry_history_and_lifecycle_details"),
    ]
    controls = {}
    for name, old, new, test in cases:
        # A kill only counts when the SAME test passes on the unmutated source in an
        # identical sandbox; otherwise the sandbox, not the mutant, made the test fail.
        if test not in controls:
            control_case = make_case(root, "control-" + test)
            controls[test] = run_case_test(control_case, test)
            record("mutation-control-" + test, "PASS" if controls[test].returncode == 0 else "FAIL",
                   controls[test].stdout[-1500:], control_returncode=controls[test].returncode)
        if controls[test].returncode != 0:
            record("mutation-" + name, "FAIL", f"INVALID: the unmutated control for {test} fails in the sandbox, so a kill would prove nothing", invalid=True)
            continue
        case = make_case(root, name)
        path = case / "scripts" / "workable_items.py"
        text = path.read_text()
        pairs = list(zip(old, new)) if isinstance(old, list) else [(old, new)]
        if any(text.count(o) < 1 for o, _ in pairs):
            record("mutation-" + name, "FAIL", "mutation anchor missing")
            continue
        for o, n in pairs:
            text = text.replace(o, n, 1)
        path.write_text(text)
        completed = run_case_test(case, test)
        record("mutation-" + name, "PASS" if completed.returncode != 0 else "FAIL", completed.stdout[-3000:], mutant_rejected=completed.returncode != 0)


# Repository evidence the tests reference, seeded into every sandbox so that a test cannot
# fail merely because the sandbox lacks the repo's evidence/ tree.
SEED_DIRS = ["evidence/workable-items", "evidence/campaign-2026-11-20/core", "evidence/campaign-2026-11-20/formats",
             "evidence/campaign-2026-11-20/containers"]
SEED_FILES = ["evidence/campaign-2026-11-20/pool-needles/README.md", "evidence/campaign-2026-11-20/lock-scope/summary.md"]


def make_case(root: Path, name: str) -> Path:
    case = root / name
    (case / "scripts").mkdir(parents=True)
    (case / "tests").mkdir()
    shutil.copyfile(ROOT / "scripts" / "workable_items.py", case / "scripts" / "workable_items.py")
    shutil.copyfile(ROOT / "tests" / "test_workable_items.py", case / "tests" / "test_workable_items.py")
    (case / "tests" / "__init__.py").write_text("")
    for directory in SEED_DIRS:
        (case / directory).mkdir(parents=True)
        (case / directory / "seed.txt").write_text("seeded evidence\n")
    for file in SEED_FILES:
        (case / file).parent.mkdir(parents=True, exist_ok=True)
        (case / file).write_text("seeded evidence\n")
    return case


def run_case_test(case: Path, test: str):
    return subprocess.run([sys.executable, "-m", "unittest", "-v", f"tests.test_workable_items.WorkableItemsTest.{test}"],
                          cwd=case, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)


def tamper_detection(root: Path, source_path: Path):
    """Flip one byte of a text field (and separately edit item_history) in a scratch copy
    of the authoritative registry; validate() must FAIL both, and PASS the untouched copy."""
    source = sqlite3.connect(f"file:{source_path}?mode=ro", uri=True)
    results = {}
    for name, sql in (("untouched", None),
                      ("text-byte-flip", "UPDATE items SET title=substr(title,1,length(title)-1)||'~' WHERE atm_id=(SELECT min(atm_id) FROM items)"),
                      ("history-edit", "UPDATE item_history SET reason=reason||'x' WHERE sequence=(SELECT min(sequence) FROM item_history)")):
        path = root / f"tamper-{name}.db"
        copy = sqlite3.connect(path)
        source.backup(copy)
        copy.close()
        db = wi.connect(path)
        if sql:
            db.execute(sql)
        results[name] = wi.validate(db)
        db.close()
    source.close()
    ok = not results["untouched"] and bool(results["text-byte-flip"]) and bool(results["history-edit"])
    record("tamper-detection-integrity-hash", "PASS" if ok else "FAIL", json.dumps(results, sort_keys=True))


def pdf_currency(snapshot: Path):
    """`diff()` compares generated Markdown and HTML only, so a PDF left behind by an
    export that died partway is invisible to the drift gate. This checks each PDF
    against the registry that produced it: it must exist, be non-trivial, and carry
    exactly the ticket identifiers its Markdown sibling carries."""
    if shutil.which("pdftotext") is None or shutil.which("pdfinfo") is None:
        record("pdf-documents-current", "FAIL", "poppler pdftotext/pdfinfo absent; PDF currency cannot be verified rather than assumed")
        return
    db = wi.connect(snapshot)
    documents = wi.markdown_documents(db)
    db.close()
    problems = []
    detail = {}
    for name, markdown in documents.items():
        pdf = ROOT / "docs" / (name[:-3] + ".pdf")
        if not pdf.exists():
            problems.append(f"{name}: PDF missing")
            continue
        size = pdf.stat().st_size
        pages = subprocess.run(["pdfinfo", str(pdf)], text=True, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL).stdout
        page_count = next((int(line.split()[-1]) for line in pages.splitlines() if line.startswith("Pages:")), 0)
        text = subprocess.run(["pdftotext", str(pdf), "-"], text=True, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL).stdout
        in_pdf = set(re.findall(rf"{wi.PROJECT_KEY}-\d{{3,}}", text))
        in_markdown = set(re.findall(rf"{wi.PROJECT_KEY}-\d{{3,}}", markdown))
        detail[name] = {"bytes": size, "pages": page_count, "identifiers": len(in_pdf)}
        if size < 1024 or page_count < 1:
            problems.append(f"{name}: PDF is {size} bytes / {page_count} pages")
        if in_pdf != in_markdown:
            problems.append(f"{name}: identifiers differ from Markdown (only in PDF: {sorted(in_pdf - in_markdown)}, only in Markdown: {sorted(in_markdown - in_pdf)})")
    record("pdf-documents-current", "PASS" if not problems else "FAIL", json.dumps({"problems": problems, **detail}, sort_keys=True))


def main():
    EVIDENCE.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="helix-wi-qualification-") as temp:
        root = Path(temp)
        snapshot = live_snapshot(root)
        run("registry-unit-integration", [sys.executable, "-m", "unittest", "-v", "tests.test_workable_items"])
        run("authoritative-registry-validation", [sys.executable, "scripts/workable_items.py", "--db", str(snapshot), "validate"])
        run("authoritative-export-drift", [sys.executable, "scripts/workable_items.py", "--db", str(snapshot), "diff"])
        pdf_currency(snapshot)
        stress(root)
        kill_and_recovery(root)
        corruption_and_md_recovery(root, snapshot)
        tamper_detection(root, snapshot)
        mutations(root)
    status = "PASS" if all(row["status"] == "PASS" for row in RESULTS) else "FAIL"
    envelope = {"schema": "helix.workable-items.qualification.v1", "status": status, "checks": RESULTS, "summary": {"pass": sum(r["status"] == "PASS" for r in RESULTS), "fail": sum(r["status"] != "PASS" for r in RESULTS)}}
    output = EVIDENCE / "results.json"
    output.write_text(json.dumps(envelope, indent=2, sort_keys=True) + "\n")
    print(json.dumps({"status": status, "output": str(output), **envelope["summary"]}, sort_keys=True))
    return status != "PASS"


if __name__ == "__main__":
    raise SystemExit(main())
