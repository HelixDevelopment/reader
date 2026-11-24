#!/usr/bin/env python3
"""Unit and integration tests for the authoritative workable-items registry.

Reconstructed on 2026-11-20 after the original file was found to be absent from the
repository: it existed only on the machine that produced
`evidence/workable-items/unit.log` (2026-10-28) and was never committed, so
`scripts/verify_workable_items.py` — which copies this file into a scratch root to
run its mutation gates — crashed with FileNotFoundError instead of qualifying the
registry. The names and the four mutation-gate targets are preserved from that log
so the gates in `verify_workable_items.py` bind to tests that exist here.

Every test builds its own scratch registry. None reads `docs/workable_items.db`,
because a test that asserts against the authoritative file passes or fails depending
on what the operator last wrote to it, and the mutation gates run this file from a
directory that has no `docs/` at all.
"""
from __future__ import annotations

import argparse
import concurrent.futures
import html as html_module
import importlib.util
import json
from pathlib import Path
import sqlite3
import re
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
SPEC = importlib.util.spec_from_file_location("wi", ROOT / "scripts" / "workable_items.py")
wi = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(wi)

KEY = wi.PROJECT_KEY
LONG = "A complete registry description with enough words and characters to satisfy validation."


def ns(**kwargs) -> argparse.Namespace:
    return argparse.Namespace(**kwargs)


def new_item_args(**overrides) -> argparse.Namespace:
    base = dict(type="Task", severity="High", title="Registry behaviour under test",
                description=LONG, scope="The registry tooling and its generated documents.",
                reproduction="Add an item and inspect the lifecycle tables.",
                acceptance="Lifecycle transitions validate and are recorded in history.",
                anchor="tests/test_workable_items.py", created_by="AI", assigned_to="Operator")
    base.update(overrides)
    return ns(**base)


class WorkableItemsTest(unittest.TestCase):
    def setUp(self) -> None:
        temp = tempfile.TemporaryDirectory(prefix="helix-wi-test-")
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name)
        self.docs = self.root / "docs"
        self.docs.mkdir()
        self.db = wi.connect(self.root / "workable_items.db")
        self.addCleanup(self.db.close)

    def add(self, **overrides) -> str:
        return wi.add_item(self.db, new_item_args(**overrides))

    def errors_containing(self, needle: str) -> list[str]:
        return [error for error in wi.validate(self.db) if needle in error]

    # --- schema and happy path ------------------------------------------------

    def test_schema_and_valid_item(self):
        tables = {row[0] for row in self.db.execute("SELECT name FROM sqlite_master WHERE type='table'")}
        for name in ("items", "item_history", "operator_block_details", "reopen_details", "obsolete_details", "meta"):
            self.assertIn(name, tables)
        self.assertEqual(self.db.execute("PRAGMA foreign_keys").fetchone()[0], 1)
        self.assertEqual(self.db.execute("PRAGMA journal_mode").fetchone()[0].lower(), "wal")
        self.assertEqual(wi.validate(self.db), [])
        atm = self.add()
        self.assertEqual(wi.validate(self.db), [], wi.validate(self.db))
        item = wi.require_item(self.db, atm)
        self.assertEqual((item["status"], item["current_location"]), ("Queued", "Issues"))
        self.assertEqual(self.db.execute("SELECT count(*) FROM item_history WHERE atm_id=?", (atm,)).fetchone()[0], 1)

    def test_add_allocates_monotonic_identifier(self):
        allocated = [self.add() for _ in range(3)]
        self.assertEqual(allocated, [f"{KEY}-001", f"{KEY}-002", f"{KEY}-003"])
        self.assertEqual(wi.validate(self.db), [])
        # Identifiers are allocated from the maximum ever issued, so a later item
        # cannot reuse one; deleting a row is instead surfaced as a gap, because a
        # silently removed identifier would look like a renumbered registry.
        self.db.execute(f"DELETE FROM item_history WHERE atm_id='{KEY}-002'")
        self.db.execute(f"DELETE FROM items WHERE atm_id='{KEY}-002'")
        self.refresh_hash()  # fixture edits are raw SQL; the hash gate is covered by its own test
        gaps = wi.validate(self.db)
        self.assertEqual(len(gaps), 1)
        self.assertIn(f"{KEY} allocation contains gaps", gaps[0])
        self.assertIn(f"{KEY}-002", gaps[0], "the error must name the missing identifier")
        self.assertEqual(wi.add_item(self.db, new_item_args(title="Reallocated")), f"{KEY}-004")
        self.db.execute(f"INSERT INTO items(atm_id,type,status,severity,title,description,affected_scope,reproduction,acceptance,created_at,last_modified,current_location,forensic_anchor,closure_criteria,composes_with,evidence_path,created_by,assigned_to,reopens_count) VALUES('{KEY}-002','Task','Queued','High','Restored',?,?,?,?,?,?,'Issues','tests','Acceptance','[]','','AI','Operator',0)",
                        (LONG, "Scope of the restored row.", "Reinsert the row.", "Acceptance condition.", "2026-10-29T00:00:00Z", "2026-10-29T00:00:00Z"))
        self.db.execute(f"INSERT INTO item_history(atm_id,event_type,actor,on_date,reason,evidence_path) VALUES('{KEY}-002','Opened','AI','2026-10-29','new-work-item','')")
        self.refresh_hash()
        self.assertEqual(wi.validate(self.db), [], wi.validate(self.db))

    def test_concurrent_writers_preserve_unique_ids(self):
        def allocate(_):
            connection = wi.connect(self.root / "workable_items.db")
            try:
                return wi.add_item(connection, new_item_args(title="Concurrent allocation"))
            finally:
                connection.close()

        with concurrent.futures.ThreadPoolExecutor(max_workers=8) as pool:
            allocated = list(pool.map(allocate, range(16)))
        self.assertEqual(len(set(allocated)), 16, f"duplicate identifiers issued: {allocated}")
        self.assertEqual(sorted(int(a.split("-")[1]) for a in allocated), list(range(1, 17)))
        self.assertEqual(wi.validate(self.db), [], wi.validate(self.db))

    # --- fail-closed validation ----------------------------------------------

    def test_invalid_status_type_and_location_fail_closed(self):
        atm = self.add()
        for column, value, needle in (
            ("status", "Pending", "invalid status"),
            ("type", "Story", "invalid type"),
            ("severity", "Blocker", "invalid severity"),
            ("current_location", "Fixed", "status/location mismatch"),
            ("description", "too short", "description is not comprehensive"),
            ("affected_scope", "   ", "missing affected_scope"),
        ):
            self.db.execute(f"UPDATE items SET {column}=? WHERE atm_id=?", (value, atm))
            self.assertTrue(self.errors_containing(needle), f"{column}={value!r} accepted by validate()")
            self.db.execute(f"UPDATE items SET {column}=? WHERE atm_id=?",
                            ({"status": "Queued", "type": "Task", "severity": "High",
                              "current_location": "Issues", "description": LONG, "affected_scope": "Scope."}[column], atm))
        self.refresh_hash()
        self.assertEqual(wi.validate(self.db), [])
        # The CHECK constraint, not only validate(), rejects an impossible location.
        with self.assertRaises(sqlite3.IntegrityError):
            self.db.execute("UPDATE items SET current_location='Maybe' WHERE atm_id=?", (atm,))
        self.db.execute("UPDATE items SET current_location='Issues' WHERE atm_id=?", (atm,))

    def test_type_aware_closure_vocabulary(self):
        # Closure wording is derived from type; a Bug closed as "Implemented" is a
        # vocabulary error even though "Implemented" is itself a terminal status.
        bug, feature, task = self.add(type="Bug"), self.add(type="Feature"), self.add(type="Task")
        for atm, alias, expected in ((bug, "fixed", "Fixed (→ Fixed.md)"),
                                     (feature, "implemented", "Implemented (→ Fixed.md)"),
                                     (task, "completed", "Completed (→ Fixed.md)")):
            with self.assertRaises(ValueError):
                wi.close_item(self.db, ns(atm_id=atm, status="completed" if alias != "completed" else "fixed",
                                          evidence="evidence/campaign-2026-11-20/core", actor="AI", reason="verified"))
            wi.close_item(self.db, ns(atm_id=atm, status=alias, evidence="evidence/campaign-2026-11-20/core",
                                      actor="AI", reason="verified"))
            self.assertEqual(wi.require_item(self.db, atm)["status"], expected)
        self.assertEqual(wi.validate(self.db), [])
        self.db.execute("UPDATE items SET status='Fixed (→ Fixed.md)' WHERE atm_id=?", (feature,))
        self.assertTrue(any("closure vocabulary" in error for error in wi.validate(self.db)),
                        "a Feature carrying the Bug closure word must be reported")
        self.db.execute("UPDATE items SET status='Implemented (→ Fixed.md)' WHERE atm_id=?", (feature,))
        self.assertEqual(wi.validate(self.db), [])
        # A terminal item with no evidence is an unsupported claim.
        self.db.execute("UPDATE items SET evidence_path='' WHERE atm_id=?", (task,))
        self.assertTrue(self.errors_containing("terminal item lacks evidence"))

    # --- lifecycle -------------------------------------------------------------

    def test_close_reopen_and_update_lifecycle(self):
        atm = self.add(type="Bug")
        with self.assertRaises(ValueError):
            wi.close_item(self.db, ns(atm_id=atm, status="fixed", evidence="   ", actor="AI", reason="verified"))
        with self.assertRaises(ValueError):
            wi.reopen_item(self.db, ns(atm_id=atm, evidence="", actor="AI", reason="needle reproduced"))
        wi.close_item(self.db, ns(atm_id=atm, status="fixed", evidence="evidence/campaign-2026-11-20/core",
                                  actor="AI", reason="verified by mutation gate"))
        item = wi.require_item(self.db, atm)
        self.assertEqual((item["status"], item["current_location"]), ("Fixed (→ Fixed.md)", "Fixed"))
        wi.reopen_item(self.db, ns(atm_id=atm, evidence="evidence/campaign-2026-11-20/pool-needles/README.md",
                                   actor="AI", reason="test-with-no-detection-power shipped the wrong claim"))
        item = wi.require_item(self.db, atm)
        self.assertEqual((item["status"], item["current_location"], item["reopens_count"]), ("Reopened", "Issues", 1))
        reopen = self.db.execute("SELECT * FROM reopen_details WHERE atm_id=?", (atm,)).fetchone()
        self.assertEqual(reopen["actor"], "AI")
        self.assertEqual(wi.validate(self.db), [], wi.validate(self.db))
        # Reopening without attribution is the state validate() must refuse.
        self.db.execute("DELETE FROM reopen_details WHERE atm_id=?", (atm,))
        self.assertTrue(self.errors_containing("incomplete reopen attribution"))
        self.db.execute("INSERT INTO reopen_details(atm_id,actor,on_date,reason,evidence_path) VALUES(?,?,?,?,?)",
                        (atm, "AI", "2026-11-20", "restored for the next assertion", "evidence/x"))
        self.refresh_hash()  # raw-SQL fixture; writers refuse unrefreshed (tampered) content
        wi.update_item(self.db, ns(atm_id=atm, title="Revised title", description=None, scope=None,
                                   reproduction=None, acceptance="Revised acceptance condition.", severity=None,
                                   actor="AI", reason="wording",
                                   evidence="evidence/campaign-2026-11-20/lock-scope/summary.md"))
        item = wi.require_item(self.db, atm)
        self.assertEqual((item["title"], item["acceptance"], item["closure_criteria"]),
                         ("Revised title", "Revised acceptance condition.", "Revised acceptance condition."))
        with self.assertRaises(ValueError):
            wi.update_item(self.db, ns(atm_id=atm, title=None, description=None, scope=None, reproduction=None,
                                       acceptance=None, severity=None, actor="AI", reason="nothing to do", evidence=""))
        events = [row[0] for row in self.db.execute("SELECT event_type FROM item_history WHERE atm_id=? ORDER BY sequence", (atm,))]
        self.assertEqual(events, ["Opened", "Fixed", "Reopened", "Updated"])
        # A progress event has to name the artifact that justifies it, or the
        # history is a diary entry instead of a pointer.
        updated = self.db.execute(
            "SELECT reason, evidence_path FROM item_history WHERE atm_id=? AND event_type='Updated'", (atm,)
        ).fetchone()
        self.assertEqual(updated["reason"], "wording")
        self.assertEqual(updated["evidence_path"], "evidence/campaign-2026-11-20/lock-scope/summary.md")
        self.assertEqual(self.db.execute("SELECT count(*) FROM item_history WHERE atm_id=? AND event_type='Reopened'", (atm,)).fetchone()[0],
                         item["reopens_count"])

    # --- status transitions (start / pause) -------------------------------------

    def transition(self, atm, target, **overrides):
        base = dict(atm_id=atm, target=target, actor="AI", reason="work under way 2026-11-23", evidence="evidence/evidence.txt")
        base.update(overrides)
        return wi.transition_item(self.db, ns(**base))

    def use_evidence_root(self):
        (self.root / "evidence").mkdir(exist_ok=True)
        (self.root / "evidence" / "evidence.txt").write_text("proof\n")
        original = wi.EVIDENCE_ROOT
        wi.EVIDENCE_ROOT = self.root
        self.addCleanup(setattr, wi, "EVIDENCE_ROOT", original)

    def snapshot(self):
        return (wi.integrity(self.db), self.db.execute("SELECT count(*) FROM item_history").fetchone()[0])

    def test_start_pause_and_restart_record_history(self):
        self.use_evidence_root()
        atm = self.add()
        self.assertEqual(self.transition(atm, "In progress"), ("Queued", "In progress"))
        item = wi.require_item(self.db, atm)
        self.assertEqual((item["status"], item["current_location"]), ("In progress", "Issues"))
        self.assertEqual(wi.validate(self.db), [], wi.validate(self.db))
        self.assertEqual(self.transition(atm, "Queued", reason="paused for a dependency"), ("In progress", "Queued"))
        self.assertEqual(self.transition(atm, "In progress"), ("Queued", "In progress"))
        rows = self.db.execute("SELECT event_type,actor,on_date,reason,evidence_path FROM item_history WHERE atm_id=? ORDER BY sequence", (atm,)).fetchall()
        self.assertEqual([r["event_type"] for r in rows], ["Opened", "Started", "Paused", "Started"])
        self.assertEqual((rows[1]["actor"], rows[1]["reason"], rows[1]["evidence_path"]), ("AI", "work under way 2026-11-23", "evidence/evidence.txt"))
        self.assertRegex(rows[1]["on_date"], r"^\d{4}-\d{2}-\d{2}$")
        self.assertEqual(rows[2]["reason"], "paused for a dependency")

    def test_reopened_item_can_resume_work(self):
        self.use_evidence_root()
        atm = self.add(type="Bug")
        wi.close_item(self.db, ns(atm_id=atm, status="fixed", evidence="evidence/evidence.txt", actor="AI", reason="verified"))
        wi.reopen_item(self.db, ns(atm_id=atm, evidence="evidence/evidence.txt", actor="AI", reason="test-failed"))
        self.assertEqual(self.transition(atm, "In progress"), ("Reopened", "In progress"))
        item = wi.require_item(self.db, atm)
        self.assertEqual((item["status"], item["reopens_count"]), ("In progress", 1))
        self.assertEqual(wi.validate(self.db), [], wi.validate(self.db))

    def test_illegal_transitions_are_refused_without_side_effects(self):
        self.use_evidence_root()
        queued = self.add()
        closed = self.add(type="Bug")
        wi.close_item(self.db, ns(atm_id=closed, status="fixed", evidence="evidence/evidence.txt", actor="AI", reason="verified"))
        blocked = self.add()
        wi.block_item(self.db, ns(atm_id=blocked, what="Blocked", why="Attempted: all paths, recorded.", unblock="Device", who="Operator"))
        cases = [(closed, "In progress"), (closed, "Queued"), (queued, "Queued"), (queued, "Fixed (→ Fixed.md)"),
                 (queued, "Ready for testing"), (blocked, "In progress"), (blocked, "Queued"), ("HRD-999", "In progress")]
        for atm, target in cases:
            before = self.snapshot()
            with self.assertRaises(ValueError, msg=f"{atm}->{target} accepted"):
                self.transition(atm, target)
            self.assertEqual(self.snapshot(), before, f"{atm}->{target} left a trace")
        self.assertEqual(wi.require_item(self.db, closed)["status"], "Fixed (→ Fixed.md)")
        self.assertEqual(wi.validate(self.db), [], wi.validate(self.db))

    def test_transition_requires_reason_and_existing_evidence(self):
        self.use_evidence_root()
        atm = self.add()
        before = self.snapshot()
        for overrides in ({"reason": "  "}, {"evidence": ""}, {"evidence": " ; "}, {"evidence": "no/such/artifact.txt"},
                          {"evidence": "evidence/evidence.txt; no/such/artifact.txt"}):
            with self.assertRaises(ValueError, msg=str(overrides)):
                self.transition(atm, "In progress", **overrides)
        self.assertEqual(self.snapshot(), before)

    def test_transition_is_atomic_under_injected_failure(self):
        self.use_evidence_root()
        atm = self.add()
        before = self.snapshot()
        original = wi.update_meta
        def boom(*_a, **_k):
            raise RuntimeError("injected failure after the history write")
        wi.update_meta = boom
        try:
            with self.assertRaises(RuntimeError):
                self.transition(atm, "In progress")
        finally:
            wi.update_meta = original
        self.assertEqual(wi.require_item(self.db, atm)["status"], "Queued")
        self.assertEqual(self.snapshot(), before)
        self.assertFalse(self.db.in_transaction)
        self.transition(atm, "In progress")

    def test_transition_integrity_hash_and_export_parity(self):
        self.use_evidence_root()
        atm = self.add()
        self.transition(atm, "In progress")
        stored = {r["key"]: r["value"] for r in self.db.execute("SELECT key,value FROM meta")}
        self.assertEqual(stored["integrity_hash"], wi.integrity(self.db))
        self.assertEqual(stored["last_sync_direction"], "start")
        wi.export(self.db, self.docs, include_pdf=False)
        for name in ("Issues.md", "Issues.html", "Issues_Summary.md", "Issues_Summary.html"):
            self.assertIn("In progress", (self.docs / name).read_text(encoding="utf-8"), name)
        self.assertIn("Started by AI: work under way 2026-11-23; evidence: evidence/evidence.txt", (self.docs / "Issues.md").read_text(encoding="utf-8"))
        self.assertEqual(wi.diff(self.db, self.docs), [])
        recovered = wi.connect(self.root / "recovered.db")
        self.addCleanup(recovered.close)
        self.assertEqual(wi.import_markdown(recovered, self.docs), 1)
        self.assertEqual(wi.require_item(recovered, atm)["status"], "In progress")
        self.assertEqual(wi.integrity(recovered), wi.integrity(self.db))

    def test_start_pause_cli_exit_codes(self):
        import subprocess, sys
        self.use_evidence_root()
        atm = self.add()
        self.db.close()
        script = str(ROOT / "scripts" / "workable_items.py")
        base = [sys.executable, script, "--db", str(self.root / "workable_items.db")]
        common = ["--actor", "AI", "--reason", "work under way", "--evidence", "evidence/workable-items"]
        bad = subprocess.run(base + ["start", "HRD-999", *common], text=True, capture_output=True)
        self.assertEqual(bad.returncode, 2)
        self.assertIn("error:", bad.stderr)
        self.assertNotIn("Traceback", bad.stderr)
        ok = subprocess.run(base + ["start", atm, *common], text=True, capture_output=True, cwd=ROOT)
        self.assertEqual(ok.returncode, 0, ok.stderr)
        self.assertEqual(json.loads(ok.stdout)["lifecycle"], "In progress")
        again = subprocess.run(base + ["start", atm, *common], text=True, capture_output=True, cwd=ROOT)
        self.assertEqual(again.returncode, 2)
        self.assertIn("illegal transition", again.stderr)
        self.db = wi.connect(self.root / "workable_items.db")

    # --- review findings: evidence confinement, history forgery, hash gate, mutants ----

    def use_confined_root(self):
        """EVIDENCE_ROOT is a `repo` subdirectory so files beside it are demonstrably outside."""
        repo = self.root / "repo"
        (repo / "evidence" / "dir").mkdir(parents=True)
        (repo / "evidence" / "a.txt").write_text("proof\n")
        (repo / "evidence" / "dir" / "b.log").write_text("x\n")
        (repo / "scripts").mkdir()
        (repo / "docs").mkdir()
        (self.root / "outside.txt").write_text("secret\n")
        original = wi.EVIDENCE_ROOT
        wi.EVIDENCE_ROOT = repo
        self.addCleanup(setattr, wi, "EVIDENCE_ROOT", original)
        return repo

    def test_transition_evidence_is_confined_to_the_evidence_tree(self):
        repo = self.use_confined_root()
        os_mkfifo = __import__("os").mkfifo
        os_mkfifo(repo / "evidence" / "fifo")
        (repo / "evidence" / "link_out").symlink_to(self.root / "outside.txt")
        (repo / "evidence" / "link_etc").symlink_to("/etc/passwd")
        (repo / "evidence" / "link_code").symlink_to(repo / "scripts")
        (repo / "evidence" / "link_in").symlink_to(repo / "evidence" / "a.txt")
        atm = self.add()
        before = self.snapshot()
        for bad in ("/etc/passwd", str(self.root / "outside.txt"), str(repo / "evidence" / "a.txt"),
                    "../outside.txt", "../../etc/passwd", "evidence/../../outside.txt", "link_out",
                    "evidence/link_out", "evidence/link_etc", "evidence/link_code",
                    ".", "./", "", "docs/..", "docs", "scripts", "scripts/", "evidence", "evidence/", "evidence/..",
                    "evidence/fifo", "evidence/a.txt; ../outside.txt"):
            with self.assertRaises(ValueError, msg=f"{bad!r} accepted"):
                self.transition(atm, "In progress", evidence=bad)
            self.assertEqual(self.snapshot(), before, f"{bad!r} left a trace")
        # Files and directories under evidence/ are fine, and are stored normalised.
        self.transition(atm, "In progress", evidence="./evidence//a.txt; evidence/dir/; evidence/link_in; evidence/dir/../a.txt")
        stored = self.db.execute("SELECT evidence_path FROM item_history WHERE event_type='Started'").fetchone()[0]
        self.assertEqual(stored, "evidence/a.txt; evidence/dir; evidence/link_in; evidence/a.txt")

    def test_history_fields_refuse_control_characters(self):
        self.use_evidence_root()
        atm = self.add(type="Bug")
        before = self.snapshot()
        forged = "ok\n- 2026-01-01 — Reopened by User: forged; evidence: x"
        for field in ("reason", "actor", "evidence"):
            for bad in ("a\nb", "a\rb", "a\x00b", "a\x1bb", "a\u2028b"):
                kw = {field: bad}
                if field == "evidence":
                    kw = {"evidence": "evidence.txt"}   # evidence also checked for existence; control chars must fail first
                    kw["evidence"] = "evidence/evidence.txt;" + bad
                with self.assertRaises(ValueError, msg=f"transition {field}={bad!r}"):
                    self.transition(atm, "In progress", **kw)
        update = dict(atm_id=atm, title=None, description=None, scope=None, reproduction=None, acceptance=None,
                      severity=None, actor="AI", reason="wording", evidence="")
        for field in ("reason", "actor", "evidence"):
            with self.assertRaises(ValueError, msg=f"update {field}"):
                wi.update_item(self.db, ns(**{**update, "title": "New title", field: forged}))
        with self.assertRaises(ValueError):
            wi.close_item(self.db, ns(atm_id=atm, status="fixed", evidence="evidence.txt\n## [HRD-999]", actor="AI", reason="r"))
        with self.assertRaises(ValueError):
            wi.close_item(self.db, ns(atm_id=atm, status="fixed", evidence="evidence/evidence.txt", actor="AI", reason=forged))
        with self.assertRaises(ValueError):
            wi.reopen_item(self.db, ns(atm_id=atm, evidence="evidence/evidence.txt", actor="AI", reason=forged))
        with self.assertRaises(ValueError):
            wi.reopen_item(self.db, ns(atm_id=atm, evidence=forged, actor="AI", reason="test-failed"))
        self.assertEqual(self.snapshot(), before)

    def test_forged_history_cannot_survive_export_and_import(self):
        # Even if a hostile row reaches item_history (raw SQL), export must neutralise
        # line breaks so import_markdown cannot mint events or phantom items from it.
        atm = self.add()
        forged = f"x\n- 2026-01-01 — Reopened by User: forged; evidence: e\n\n## [{KEY}-999] Phantom"
        self.db.execute("INSERT INTO item_history(atm_id,event_type,actor,on_date,reason,evidence_path) VALUES(?,?,?,?,?,?)",
                        (atm, "Updated", "AI", "2026-11-23", forged, "e\r\n## [HRD-998] Phantom"))
        self.refresh_hash()  # hostile row that was also rehashed
        wi.export(self.db, self.docs, include_pdf=False)
        recovered = wi.connect(self.root / "recovered.db")
        self.addCleanup(recovered.close)
        # The planted event carries the field-boundary token, so the import gate refuses it outright.
        with self.assertRaises(ValueError):
            wi.import_markdown(recovered, self.docs)
        self.assertEqual(wi.items(recovered), [])
        self.assertEqual(recovered.execute("SELECT count(*) FROM item_history WHERE event_type='Reopened'").fetchone()[0], 0)
        for line in wi.history_lines(self.db, atm):
            self.assertNotRegex(line, r"[\r\n\x00-\x08\x0b-\x1f\u2028\u2029]")

    def refresh_hash(self):
        wi.update_meta(self.db, "test-fixture")

    def test_validate_detects_tampering_via_integrity_hash(self):
        atm = self.add()
        self.assertEqual(wi.validate(self.db), [])
        self.db.execute("UPDATE items SET title=? WHERE atm_id=?", ("Registry behaviour under tesT", atm))  # one-byte flip
        self.assertTrue(self.errors_containing("integrity"), "text-field flip passed validate()")
        self.db.execute("UPDATE items SET title=? WHERE atm_id=?", ("Registry behaviour under test", atm))
        self.assertEqual(wi.validate(self.db), [])
        self.db.execute("UPDATE item_history SET reason='forged' WHERE atm_id=?", (atm,))
        self.assertTrue(self.errors_containing("integrity"), "item_history edit passed validate()")
        self.refresh_hash()
        self.assertEqual(wi.validate(self.db), [])
        self.db.execute("DELETE FROM meta WHERE key='integrity_hash'")
        self.assertTrue(self.errors_containing("integrity"), "missing hash passed validate()")

    def test_every_legitimate_writer_keeps_the_hash_valid(self):
        self.use_evidence_root()
        a = self.add(type="Bug")
        b = self.add()
        self.assertEqual(wi.validate(self.db), [])
        self.transition(a, "In progress")
        self.assertEqual(wi.validate(self.db), [])
        wi.update_item(self.db, ns(atm_id=a, title="Another title", description=None, scope=None, reproduction=None,
                                   acceptance=None, severity=None, actor="AI", reason="r", evidence=""))
        wi.close_item(self.db, ns(atm_id=a, status="fixed", evidence="evidence/evidence.txt", actor="AI", reason="v"))
        wi.reopen_item(self.db, ns(atm_id=a, evidence="evidence/evidence.txt", actor="User", reason="test-failed"))
        wi.block_item(self.db, ns(atm_id=b, what="W", why="Attempted: everything.", unblock="U", who="Operator"))
        self.assertEqual(wi.validate(self.db), [], wi.validate(self.db))
        wi.export(self.db, self.docs, include_pdf=False)
        recovered = wi.connect(self.root / "recovered.db")
        self.addCleanup(recovered.close)
        wi.import_markdown(recovered, self.docs)
        self.assertEqual(wi.validate(recovered), [], wi.validate(recovered))

    def test_transition_records_the_supplied_actor(self):
        self.use_evidence_root()
        atm = self.add()
        self.transition(atm, "In progress", actor="User")
        self.transition(atm, "Queued", actor="AI")
        self.transition(atm, "In progress", actor="User")
        actors = [r[0] for r in self.db.execute("SELECT actor FROM item_history WHERE atm_id=? AND event_type IN ('Started','Paused') ORDER BY sequence", (atm,))]
        self.assertEqual(actors, ["User", "AI", "User"])

    def test_in_progress_has_exactly_one_exit(self):
        self.use_evidence_root()
        self.assertEqual(wi.TRANSITIONS, {"Queued": {"In progress"}, "Reopened": {"In progress"}, "In progress": {"Queued"}})
        atm = self.add()
        self.transition(atm, "In progress")
        before = self.snapshot()
        for target in ("Reopened", "Completed (→ Fixed.md)", "Fixed (→ Fixed.md)", "Implemented (→ Fixed.md)",
                       "Obsolete (→ Fixed.md)", "Operator-blocked", "In progress"):
            with self.assertRaises(ValueError, msg=f"In progress -> {target} accepted"):
                self.transition(atm, target)
        self.assertEqual(self.snapshot(), before)

    def test_transition_takes_the_write_lock_at_begin(self):
        self.use_evidence_root()
        atm = self.add()
        probe = sqlite3.connect(self.root / "workable_items.db", timeout=0, isolation_level=None)
        self.addCleanup(probe.close)
        outcomes = []

        class Spy:
            def __init__(self, inner):
                self.inner = inner
            def __getattr__(self, name):
                return getattr(self.inner, name)
            def execute(self, sql, *rest):
                result = self.inner.execute(sql, *rest)
                if sql.strip().upper().startswith("BEGIN"):
                    try:
                        probe.execute("BEGIN IMMEDIATE")
                        outcomes.append("lock-free")
                        probe.execute("ROLLBACK")
                    except sqlite3.OperationalError as exc:
                        outcomes.append(str(exc))
                return result

        wi.transition_item(Spy(self.db), ns(atm_id=atm, target="In progress", actor="AI", reason="r", evidence="evidence/evidence.txt"))
        self.assertEqual(len(outcomes), 1)
        self.assertIn("locked", outcomes[0], "BEGIN did not take the write lock")

    def test_pause_cli_end_to_end_and_locked_database_exit(self):
        import os, subprocess, sys
        self.use_evidence_root()
        atm = self.add()
        self.db.close()
        script = str(ROOT / "scripts" / "workable_items.py")
        dbpath = str(self.root / "workable_items.db")
        base = [sys.executable, script, "--db", dbpath]
        common = ["--actor", "User", "--reason", "cli", "--evidence", "evidence/workable-items"]
        paused_early = subprocess.run(base + ["pause", atm, *common], text=True, capture_output=True, cwd=ROOT)
        self.assertEqual(paused_early.returncode, 2, "pause of a Queued item must be refused")
        self.assertIn("illegal transition", paused_early.stderr)
        self.assertEqual(subprocess.run(base + ["start", atm, *common], text=True, capture_output=True, cwd=ROOT).returncode, 0)
        paused = subprocess.run(base + ["pause", atm, *common], text=True, capture_output=True, cwd=ROOT)
        self.assertEqual(paused.returncode, 0, paused.stderr)
        self.assertEqual(json.loads(paused.stdout), {"status": "PASS", "atm_id": atm, "from": "In progress", "lifecycle": "Queued"})
        db = wi.connect(Path(dbpath))
        rows = db.execute("SELECT event_type,actor FROM item_history WHERE atm_id=? ORDER BY sequence", (atm,)).fetchall()
        self.assertEqual([(r[0], r[1]) for r in rows], [("Opened", "AI"), ("Started", "User"), ("Paused", "User")])
        db.close()
        # A held write lock must end in a clean exit 2, not a traceback.
        holder = sqlite3.connect(dbpath, timeout=0, isolation_level=None)
        holder.execute("BEGIN IMMEDIATE")
        try:
            env = {**os.environ, "HELIX_WORKABLE_ITEMS_BUSY_MS": "200"}
            locked = subprocess.run(base + ["start", atm, *common], text=True, capture_output=True, cwd=ROOT, env=env, timeout=60)
        finally:
            holder.execute("ROLLBACK")
            holder.close()
        self.assertEqual(locked.returncode, 2, locked.stdout + locked.stderr)
        self.assertIn("locked", locked.stderr)
        self.assertNotIn("Traceback", locked.stderr)
        self.db = wi.connect(Path(dbpath))

    # --- second review round: free-text forgery, tamper-before-write, read snapshot ----

    def free_text_args(self, **over):
        return new_item_args(**over)

    def test_free_text_fields_refuse_line_breaks_and_invisible_controls(self):
        for field in ("title", "description", "scope", "reproduction", "acceptance", "anchor", "created_by", "assigned_to"):
            for bad in ("a\nb", "a\rb", "a\x0bb", "a\x0cb", "a\x85b", "a\x7fb", "a\u202eb", "a\u2066b",
                        "a\u200bb", "a\ufeffb", "a\u2060b", "a\u00adb", "a\u2028b"):
                with self.assertRaises(ValueError, msg=f"add {field}={bad!r}"):
                    wi.add_item(self.db, new_item_args(**{field: LONG + bad if field == "description" else bad}))
        self.assertEqual(wi.items(self.db), [])
        atm = self.add()
        before = self.snapshot()
        for field in ("title", "description", "scope", "reproduction", "acceptance"):
            with self.assertRaises(ValueError, msg=f"update {field}"):
                wi.update_item(self.db, ns(**{**dict(atm_id=atm, title=None, description=None, scope=None, reproduction=None,
                                           acceptance=None, severity=None, actor="AI", reason="r", evidence=""),
                                           field: "x\n\n## [HRD-900] Phantom"}))
        for field in ("what", "why", "unblock", "who"):
            args = dict(atm_id=atm, what="W", why="Attempted: all.", unblock="U", who="Op")
            args[field] = args[field] + "\n## [HRD-901] Phantom"
            with self.assertRaises(ValueError, msg=f"block {field}"):
                wi.block_item(self.db, ns(**args))
        self.assertEqual(self.snapshot(), before)

    def test_history_fields_refuse_bidi_zero_width_and_exotic_controls(self):
        self.use_evidence_root()
        atm = self.add()
        for bad in ("a\x0bb", "a\x0cb", "a\x85b", "a\x7fb", "a\u202ab", "a\u202eb", "a\u2066b", "a\u2069b",
                    "a\u200bb", "a\u200fb", "a\ufeffb", "a\u2060b", "a\u00adb"):
            with self.assertRaises(ValueError, msg=repr(bad)):
                self.transition(atm, "In progress", reason=bad)
            with self.assertRaises(ValueError, msg=repr(bad)):
                wi.reopen_item(self.db, ns(atm_id=atm, evidence="evidence/evidence.txt", actor="AI", reason=bad))
        self.assertEqual(wi.CONTROL_RE.sub("", "tab\there"), "tab\there")

    def test_description_line_cannot_forge_history_through_export_and_import(self):
        # Attack 1: heading injection in a title; attack 2: a history-grammar line in a description.
        # Both are refused on entry; and even planted by raw SQL, export->import must not create them.
        atm = self.add()
        forged_line = "- 2026-01-01 — Fixed by User: forged-closure-approved; evidence: evidence/forged-proof"
        self.db.execute("UPDATE items SET description=? WHERE atm_id=?", (LONG + "\n\n" + forged_line, atm))
        self.db.execute("UPDATE items SET title=? WHERE atm_id=?", ("T\n\n## [HRD-900] Phantom item", atm))
        self.refresh_hash()
        self.assertTrue(self.errors_containing("control"), "validate accepted control characters in stored text")
        wi.export(self.db, self.docs, include_pdf=False)
        recovered = wi.connect(self.root / "recovered.db")
        self.addCleanup(recovered.close)
        wi.import_markdown(recovered, self.docs)
        self.assertEqual([i["atm_id"] for i in wi.items(recovered)], [atm])
        self.assertEqual([r[0] for r in recovered.execute("SELECT event_type FROM item_history")], ["Opened"])
        text = (self.docs / "Issues.md").read_text(encoding="utf-8")
        self.assertNotRegex(text, r"(?m)^## \[HRD-900\]")
        self.assertNotRegex(text, r"(?m)^- 2026-01-01 — Fixed by User")

    def test_history_is_parsed_only_from_the_history_section(self):
        atm = self.add()
        wi.export(self.db, self.docs, include_pdf=False)
        path = self.docs / "Issues.md"
        text = path.read_text(encoding="utf-8")
        planted = "\n**Description:** " + LONG + "\n\n- 2026-01-01 — Fixed by User: forged-closure-approved; evidence: evidence/forged-proof\n"
        text = text.replace("**History:**", planted + "**History:**", 1)
        path.write_text(text, encoding="utf-8")
        parsed = wi.parse_markdown(path, "Issues")
        self.assertEqual([e["event_type"] for e in parsed[0]["_history"]], ["Opened"])

    def test_writer_refusals_end_in_error_exit_2(self):
        import subprocess, sys
        atm = self.add()
        self.db.close()
        base = [sys.executable, str(ROOT / "scripts" / "workable_items.py"), "--db", str(self.root / "workable_items.db")]
        for args in (["update", atm, "--title", "x\ny"], ["update", "HRD-999", "--title", "x"],
                     ["close", atm, "--evidence", "a\nb"], ["reopen", atm, "--actor", "AI", "--reason", "r", "--evidence", "a\nb"],
                     ["operator-block", atm, "--what", "w", "--why", "not attempted", "--unblock", "u", "--who", "w"],
                     ["close", "HRD-999", "--evidence", "evidence/x"]):
            done = subprocess.run(base + args, text=True, capture_output=True, cwd=ROOT)
            self.assertEqual(done.returncode, 2, (args, done.stderr))
            self.assertIn("error:", done.stderr)
            self.assertNotIn("Traceback", done.stderr)
        self.db = wi.connect(self.root / "workable_items.db")

    def tamper_all_writers(self):
        self.use_evidence_root()
        a = self.add(type="Bug")
        b = self.add()
        self.db.execute("UPDATE items SET title='Tampered title' WHERE atm_id=?", (a,))  # no hash refresh
        return a, b

    def test_every_writer_refuses_a_tampered_registry_before_writing(self):
        a, b = self.tamper_all_writers()
        calls = {
            "add": lambda: wi.add_item(self.db, new_item_args()),
            "update": lambda: wi.update_item(self.db, ns(atm_id=b, title="New", description=None, scope=None, reproduction=None, acceptance=None, severity=None, actor="AI", reason="r", evidence="")),
            "transition": lambda: self.transition(b, "In progress"),
            "close": lambda: wi.close_item(self.db, ns(atm_id=b, status="task" if False else None, evidence="evidence/evidence.txt", actor="AI", reason="r")),
            "reopen": lambda: wi.reopen_item(self.db, ns(atm_id=b, evidence="evidence/evidence.txt", actor="AI", reason="test-failed")),
            "block": lambda: wi.block_item(self.db, ns(atm_id=b, what="W", why="Attempted: x.", unblock="U", who="O")),
            "export": lambda: wi.export(self.db, self.docs, include_pdf=False),
            "import": lambda: (wi.export if False else wi.import_markdown)(self.db, self.docs),
        }
        wi_hash = {r[0]: r[1] for r in self.db.execute("SELECT key,value FROM meta")}["integrity_hash"]
        for name, call in calls.items():
            if name == "import":
                (self.docs / "Issues.md").write_text("## [HRD-001] x\n\n**Status:** Queued\n", encoding="utf-8")
                (self.docs / "Fixed.md").write_text("", encoding="utf-8")
            with self.assertRaises(ValueError, msg=name) as ctx:
                call()
            self.assertIn("tampered", str(ctx.exception), name)
            if name == "export":
                self.assertEqual(list(self.docs.iterdir()), [], "export wrote documents from a tampered registry")
            self.assertFalse(self.db.in_transaction, name)
            self.assertEqual({r[0]: r[1] for r in self.db.execute("SELECT key,value FROM meta")}["integrity_hash"], wi_hash, name)
        self.assertEqual(self.db.execute("SELECT count(*) FROM items").fetchone()[0], 2)

    def test_rehash_is_the_only_way_past_tampering_and_is_recorded(self):
        a, b = self.tamper_all_writers()
        with self.assertRaises(ValueError):
            wi.rehash(self.db, ns(actor="AI\n", reason="r", confirm=wi.integrity(self.db), dry_run=False))
        before = self.db.execute("SELECT value FROM meta WHERE key='integrity_hash'").fetchone()[0]
        out = self.rehash(reason="reviewed manual SQL fix of HRD-001 title")
        self.assertEqual(out["previous"], before)
        self.assertEqual(out["new"], wi.integrity(self.db))
        self.assertEqual(wi.validate(self.db), [])
        meta = {r[0]: r[1] for r in self.db.execute("SELECT key,value FROM meta")}
        self.assertEqual(meta["last_sync_direction"], "rehash")
        self.assertIn("reviewed manual SQL fix", meta["rehash_log"])
        self.assertIn("User", meta["rehash_log"])
        self.add()  # writers work again
        # rehash must not bless structurally invalid content.
        self.db.execute("UPDATE items SET status='Pending' WHERE atm_id=?", (a,))
        with self.assertRaises(ValueError):
            self.rehash(reason="try to bless an invalid status")

    def test_rehash_cli(self):
        import subprocess, sys
        a, b = self.tamper_all_writers()
        self.db.close()
        base = [sys.executable, str(ROOT / "scripts" / "workable_items.py"), "--db", str(self.root / "workable_items.db")]
        bad = subprocess.run(base + ["validate"], text=True, capture_output=True, cwd=ROOT)
        self.assertEqual(bad.returncode, 1)
        blocked = subprocess.run(base + ["start", a, "--actor", "AI", "--reason", "r", "--evidence", "evidence/workable-items"], text=True, capture_output=True, cwd=ROOT)
        self.assertEqual(blocked.returncode, 2)
        self.assertIn("tampered", blocked.stderr)
        dry = subprocess.run(base + ["rehash", "--actor", "User", "--reason", "reviewed", "--dry-run"], text=True, capture_output=True, cwd=ROOT)
        self.assertEqual(dry.returncode, 0, dry.stderr)
        candidate = json.loads(dry.stdout)["candidate"]
        refused = subprocess.run(base + ["rehash", "--actor", "User", "--reason", "reviewed"], text=True, capture_output=True, cwd=ROOT)
        self.assertEqual(refused.returncode, 2)
        ok = subprocess.run(base + ["rehash", "--actor", "User", "--reason", "reviewed", "--confirm", candidate], text=True, capture_output=True, cwd=ROOT)
        self.assertEqual(ok.returncode, 0, ok.stderr)
        self.assertEqual(json.loads(ok.stdout)["status"], "PASS")
        self.assertEqual(subprocess.run(base + ["validate"], text=True, capture_output=True, cwd=ROOT).returncode, 0)
        self.db = wi.connect(self.root / "workable_items.db")

    def test_integrity_covers_every_detail_and_sync_table(self):
        self.use_evidence_root()
        a = self.add(type="Bug")
        b = self.add()
        wi.close_item(self.db, ns(atm_id=a, status="fixed", evidence="evidence/evidence.txt", actor="AI", reason="v"))
        wi.reopen_item(self.db, ns(atm_id=a, evidence="evidence/evidence.txt", actor="AI", reason="test-failed"))
        wi.block_item(self.db, ns(atm_id=b, what="W", why="Attempted: x.", unblock="U", who="O"))
        self.db.execute("INSERT INTO obsolete_details(atm_id,since,reason,superseding_item,triple_check_evidence) VALUES(?,?,?,?,?)", (a, "2026-01-01", "r", "", "e"))
        self.db.execute("INSERT INTO firebase_metadata(atm_id,payload) VALUES(?,?)", (a, "{}"))
        self.db.execute("INSERT INTO sync_outbox(atm_id,revision,payload_hash,state,updated_at) VALUES(?,?,?,?,?)", (a, 1, "h", "pending", "2026-01-01T00:00:00Z"))
        self.refresh_hash()
        self.assertEqual(wi.validate(self.db), [], wi.validate(self.db))
        for sql in ("UPDATE reopen_details SET reason='forged' ", "UPDATE operator_block_details SET who='forged' ",
                    "UPDATE obsolete_details SET reason='forged' ", "UPDATE firebase_metadata SET payload='{\"x\":1}' ",
                    "UPDATE sync_outbox SET state='pushed' "):
            snap = self.db.execute("SELECT value FROM meta WHERE key='integrity_hash'").fetchone()[0]
            self.db.execute("SAVEPOINT t")
            self.db.execute(sql)
            self.assertTrue(self.errors_containing("integrity"), sql)
            self.db.execute("ROLLBACK TO t")
            self.db.execute("RELEASE t")
            self.assertEqual(wi.validate(self.db), [], sql)

    def test_validate_reads_hash_and_content_in_one_snapshot(self):
        self.add()
        other = wi.connect(self.root / "workable_items.db")
        self.addCleanup(other.close)
        fired = []

        class Spy:
            def __init__(self, inner):
                self.inner = inner
            def __getattr__(self, name):
                return getattr(self.inner, name)
            def execute(self, sql, *rest):
                if sql.startswith("SELECT * FROM items") and not fired:
                    fired.append(1)
                    wi.add_item(other, new_item_args(title="Concurrent legitimate write"))
                return self.inner.execute(sql, *rest)

        self.assertEqual(wi.validate(Spy(self.db)), [])
        self.assertEqual(fired, [1])
        self.assertFalse(self.db.in_transaction, "validate left its read transaction open")

    def test_update_is_validated_immediately_after_each_writer(self):
        self.use_evidence_root()
        a = self.add(type="Bug")
        steps = [
            lambda: wi.update_item(self.db, ns(atm_id=a, title="Another title", description=None, scope=None, reproduction=None, acceptance=None, severity=None, actor="AI", reason="r", evidence="")),
            lambda: self.transition(a, "In progress"),
            lambda: self.transition(a, "Queued"),
            lambda: wi.close_item(self.db, ns(atm_id=a, status="fixed", evidence="evidence/evidence.txt", actor="AI", reason="v")),
            lambda: wi.reopen_item(self.db, ns(atm_id=a, evidence="evidence/evidence.txt", actor="User", reason="test-failed")),
            lambda: wi.block_item(self.db, ns(atm_id=a, what="W", why="Attempted: x.", unblock="U", who="O")),
            lambda: wi.export(self.db, self.docs, include_pdf=False),
        ]
        for n, step in enumerate(steps):
            step()
            self.assertEqual(wi.validate(self.db), [], f"step {n}")

    def test_transition_sets_last_modified_and_stores_all_evidence(self):
        self.use_evidence_root()
        (self.root / "evidence" / "second.txt").write_text("more\n")
        atm = self.add()
        old = wi.utcnow
        wi.utcnow = lambda: "2031-05-06T07:08:09Z"
        self.addCleanup(setattr, wi, "utcnow", old)
        self.transition(atm, "In progress", evidence="evidence/evidence.txt; evidence/second.txt")
        self.assertEqual(wi.require_item(self.db, atm)["last_modified"], "2031-05-06T07:08:09Z")
        stored = self.db.execute("SELECT evidence_path FROM item_history WHERE event_type='Started'").fetchone()[0]
        self.assertEqual(stored, "evidence/evidence.txt; evidence/second.txt")

    def test_busy_timeout_environment_variable_is_honoured(self):
        import os
        old = os.environ.get("HELIX_WORKABLE_ITEMS_BUSY_MS")
        self.addCleanup(lambda: os.environ.__setitem__("HELIX_WORKABLE_ITEMS_BUSY_MS", old) if old is not None else os.environ.pop("HELIX_WORKABLE_ITEMS_BUSY_MS", None))
        os.environ["HELIX_WORKABLE_ITEMS_BUSY_MS"] = "123"
        self.assertEqual(wi.busy_ms(), 123)
        conn = wi.connect(self.root / "b.db")
        self.addCleanup(conn.close)
        self.assertEqual(conn.execute("PRAGMA busy_timeout").fetchone()[0], 123)
        os.environ["HELIX_WORKABLE_ITEMS_BUSY_MS"] = "junk"
        self.assertEqual(wi.busy_ms(), 30000)
        os.environ.pop("HELIX_WORKABLE_ITEMS_BUSY_MS")
        self.assertEqual(wi.busy_ms(), 30000)

    def test_only_lock_errors_are_reported_as_locked(self):
        import contextlib, io, sys
        original_connect, original_ro, original_argv = wi.connect, wi.connect_readonly, sys.argv
        def boom(_path):
            raise sqlite3.OperationalError("no such table: items")
        wi.connect = wi.connect_readonly = boom
        sys.argv = ["workable_items.py", "--db", str(self.root / "x.db"), "validate"]
        try:
            with self.assertRaises(sqlite3.OperationalError):
                with contextlib.redirect_stderr(io.StringIO()):
                    wi.main()
            wi.connect = wi.connect_readonly = lambda _p: (_ for _ in ()).throw(sqlite3.OperationalError("database is locked"))
            err = io.StringIO()
            with contextlib.redirect_stderr(err):
                self.assertEqual(wi.main(), 2)
            self.assertIn("locked", err.getvalue())
        finally:
            wi.connect, wi.connect_readonly, sys.argv = original_connect, original_ro, original_argv

    def test_outbox_timestamp_is_the_only_nondeterministic_field(self):
        self.add()
        one, two = self.root / "o1.json", self.root / "o2.json"
        wi.helixtrack_outbox(self.db, one)
        wi.helixtrack_outbox(self.db, two)
        a, b = json.loads(one.read_text()), json.loads(two.read_text())
        a.pop("generated_at"), b.pop("generated_at")
        self.assertEqual(a, b)

    # --- third review round: import laundering, rehash controls, commit gate, tiers ----

    def closed_item_docs(self):
        self.use_evidence_root()
        atm = self.add(type="Bug")
        wi.close_item(self.db, ns(atm_id=atm, status="fixed", evidence="evidence/evidence.txt", actor="AI", reason="verified"))
        wi.export(self.db, self.docs, include_pdf=False)
        return atm

    def edit_doc(self, name, old, new):
        path = self.docs / name
        text = path.read_text(encoding="utf-8")
        self.assertIn(old, text)
        path.write_text(text.replace(old, new, 1), encoding="utf-8")

    def test_import_refuses_silent_overwrite_of_existing_items(self):
        atm = self.closed_item_docs()
        self.edit_doc("Fixed.md", "**Severity:** High", "**Severity:** Low")
        before = self.snapshot()
        with self.assertRaises(ValueError) as ctx:
            wi.import_markdown(self.db, self.docs)
        self.assertIn(atm, str(ctx.exception))
        self.assertIn("severity", str(ctx.exception))
        self.assertEqual(self.snapshot(), before)
        plan = wi.plan_import(self.db, self.docs)
        self.assertTrue(any(atm in line and "severity" in line and "High" in line and "Low" in line for line in plan), plan)
        wi.import_markdown(self.db, self.docs, allow_overwrite=True, actor="User", reason="reviewed severity correction")
        self.assertEqual(wi.require_item(self.db, atm)["severity"], "Low")
        self.assertEqual(wi.validate(self.db), [], wi.validate(self.db))

    def test_import_refuses_unconfined_evidence_even_with_overwrite(self):
        atm = self.closed_item_docs()
        for bad in ("/etc/shadow", "../../etc/passwd", "evidence/../../x"):
            self.edit_doc("Fixed.md", "**Evidence:** evidence/evidence.txt", f"**Evidence:** {bad}")
            with self.assertRaises(ValueError, msg=bad):
                wi.import_markdown(self.db, self.docs, allow_overwrite=True)
            self.edit_doc("Fixed.md", f"**Evidence:** {bad}", "**Evidence:** evidence/evidence.txt")
        fresh = wi.connect(self.root / "fresh.db")
        self.addCleanup(fresh.close)
        self.edit_doc("Fixed.md", "**Evidence:** evidence/evidence.txt", "**Evidence:** /etc/shadow")
        with self.assertRaises(ValueError):
            wi.import_markdown(fresh, self.docs)
        self.assertEqual(wi.items(fresh), [])

    def test_import_gate_rejects_control_characters_and_invalid_rows(self):
        self.add()
        wi.export(self.db, self.docs, include_pdf=False)
        fresh = wi.connect(self.root / "fresh.db")
        self.addCleanup(fresh.close)
        for old, new in (("] Registry behaviour under test", "] Registry behaviour"), ("**Severity:** High", "**Severity:** Mighty"),
                         (LONG, "short")):
            self.edit_doc("Issues.md", old, new)
            with self.assertRaises(ValueError, msg=new):
                wi.import_markdown(fresh, self.docs)
            self.assertEqual(wi.items(fresh), [], new)
            self.edit_doc("Issues.md", new, old)
        self.assertEqual(wi.import_markdown(fresh, self.docs), 1)

    def candidate(self):
        return wi.integrity(self.db)

    def rehash(self, **over):
        base = dict(actor="User", reason="reviewed manual SQL fix", confirm=self.candidate(), dry_run=False)
        base.update(over)
        return wi.rehash(self.db, ns(**base))

    def test_rehash_requires_matching_confirm_and_refuses_noop(self):
        a, b = self.tamper_all_writers()
        before = self.snapshot()
        for confirm in (None, "", "0" * 64):
            with self.assertRaises(ValueError, msg=repr(confirm)):
                self.rehash(confirm=confirm)
        dry = self.rehash(dry_run=True, confirm=None)
        self.assertEqual(dry["candidate"], self.candidate())
        self.assertEqual(dry["changed"], [a])
        self.assertEqual(self.db.execute("SELECT count(*) FROM meta WHERE key='rehash_log'").fetchone()[0], 0)
        out = self.rehash()
        self.assertEqual(out["changed"], [a])
        self.assertEqual(wi.validate(self.db), [])
        with self.assertRaises(ValueError, msg="no-op rehash accepted"):
            self.rehash()
        self.assertEqual(before[1], self.snapshot()[1])

    def test_rehash_refuses_empty_reason_and_logs_chained_appends(self):
        a, b = self.tamper_all_writers()
        for reason in ("", "   ", None):
            with self.assertRaises(ValueError, msg=repr(reason)):
                self.rehash(reason=reason)
        self.rehash(reason="first review")
        self.db.execute("UPDATE items SET title='Second tamper' WHERE atm_id=?", (b,))
        self.rehash(reason="second review")
        lines = self.db.execute("SELECT value FROM meta WHERE key='rehash_log'").fetchone()[0].split("\n")
        self.assertEqual(len(lines), 2, "log must append, not overwrite")
        entries = [json.loads(line) for line in lines]
        self.assertEqual([e["reason"] for e in entries], ["first review", "second review"])
        import hashlib
        self.assertEqual(entries[0]["prev_entry"], "")
        self.assertEqual(entries[1]["prev_entry"], hashlib.sha256(lines[0].encode()).hexdigest())
        self.assertEqual(entries[1]["changed"], [b])
        self.assertEqual(wi.validate(self.db), [])
        # The log is covered by the integrity hash and its chain is verified.
        meta = self.db.execute("SELECT value FROM meta WHERE key='rehash_log'").fetchone()[0]
        self.db.execute("UPDATE meta SET value=? WHERE key='rehash_log'", (meta.replace("first review", "forged review"),))
        self.assertTrue(self.errors_containing("integrity"), "rehash_log edit not detected")
        self.db.execute("UPDATE meta SET value=? WHERE key='rehash_log'", (meta,))
        self.assertEqual(wi.validate(self.db), [])

    def test_rehash_chain_break_is_reported(self):
        a, b = self.tamper_all_writers()
        self.rehash(reason="first")
        self.db.execute("UPDATE items SET title='Second tamper' WHERE atm_id=?", (b,))
        self.rehash(reason="second")
        lines = self.db.execute("SELECT value FROM meta WHERE key='rehash_log'").fetchone()[0].split("\n")
        broken = json.loads(lines[1])
        broken["prev_entry"] = "0" * 64
        self.db.execute("UPDATE meta SET value=? WHERE key='rehash_log'", (lines[0] + "\n" + json.dumps(broken, sort_keys=True),))
        self.refresh_hash()
        self.assertTrue(self.errors_containing("chain"), wi.validate(self.db))

    def test_every_writer_rolls_back_when_the_result_would_be_invalid(self):
        self.use_evidence_root()
        a = self.add(type="Bug")
        b = self.add()
        base = dict(title=None, description=None, scope=None, reproduction=None, acceptance=None, severity=None, actor="AI", reason="r", evidence="")
        for field, bad in (("description", "too short"), ("title", ""), ("scope", " "), ("title", "   ")):
            before = self.snapshot()
            with self.assertRaises(ValueError, msg=f"{field}={bad!r}"):
                wi.update_item(self.db, ns(atm_id=a, **{**base, field: bad}))
            self.assertEqual(self.snapshot(), before)
            self.assertFalse(self.db.in_transaction)
        writers = {
            "add": lambda: wi.add_item(self.db, new_item_args()),
            "update": lambda: wi.update_item(self.db, ns(atm_id=a, **{**base, "title": "Fine title"})),
            "transition": lambda: self.transition(b, "In progress"),
            "close": lambda: wi.close_item(self.db, ns(atm_id=a, status="fixed", evidence="evidence/evidence.txt", actor="AI", reason="v")),
            "block": lambda: wi.block_item(self.db, ns(atm_id=b, what="W", why="Attempted: x.", unblock="U", who="O")),
        }
        original = wi._validate
        for name, call in writers.items():
            calls = []
            def fake(db, check_hash, _calls=calls):
                _calls.append(1)
                return [] if len(_calls) == 1 else ["HRD-001: injected post-write defect"]
            wi._validate = fake
            before = self.snapshot()
            try:
                with self.assertRaises(ValueError, msg=name):
                    call()
            finally:
                wi._validate = original
            self.assertEqual(self.snapshot(), before, f"{name} committed an invalid registry")
            self.assertFalse(self.db.in_transaction, name)

    def test_import_gate_runs_inside_the_import_transaction(self):
        self.add()
        wi.export(self.db, self.docs, include_pdf=False)
        fresh = wi.connect(self.root / "fresh.db")
        self.addCleanup(fresh.close)
        original = wi._validate
        wi._validate = lambda db, check_hash, _n=[0]: (_n.append(1) or ([] if len(_n) == 2 else ["HRD-001: injected"]))
        try:
            with self.assertRaises(ValueError):
                wi.import_markdown(fresh, self.docs)
        finally:
            wi._validate = original
        self.assertEqual(wi.items(fresh), [])

    def test_closure_evidence_tier_is_repo_confined_and_must_exist(self):
        self.use_evidence_root()
        a = self.add(type="Bug")
        base = dict(atm_id=a, status="fixed", actor="AI", reason="v")
        for bad in ("../../../etc/passwd", "/etc/passwd", "no/such/artifact", ".", "a/../../x", "evidence/evidence.txt; ../../../etc/passwd"):
            with self.assertRaises(ValueError, msg=f"close accepted {bad!r}"):
                wi.close_item(self.db, ns(evidence=bad, **base))
        (self.root / "escape").symlink_to("/etc/passwd")
        (self.root / "escape_dir").symlink_to("/etc")
        for bad in ("escape", "escape_dir", "escape_dir/passwd"):
            with self.assertRaises(ValueError, msg=f"close accepted symlink escape {bad!r}"):
                wi.close_item(self.db, ns(evidence=bad, **base))
        self.assertEqual(wi.require_item(self.db, a)["status"], "Queued")
        wi.close_item(self.db, ns(evidence="./evidence//evidence.txt", **base))
        self.assertEqual(wi.require_item(self.db, a)["evidence_path"], "evidence/evidence.txt")
        for bad in ("../../../etc/passwd", "/etc/passwd", "no/such/artifact"):
            with self.assertRaises(ValueError, msg=f"reopen accepted {bad!r}"):
                wi.reopen_item(self.db, ns(atm_id=a, actor="AI", reason="test-failed", evidence=bad))
            with self.assertRaises(ValueError, msg=f"update accepted {bad!r}"):
                wi.update_item(self.db, ns(atm_id=a, title="New title", description=None, scope=None, reproduction=None, acceptance=None, severity=None, actor="AI", reason="r", evidence=bad))
        # Tier 2 allows any existing repo path; tier 1 (start/pause) stays inside evidence/.
        (self.root / "docs").mkdir(exist_ok=True)
        (self.root / "docs" / "note.md").write_text("x\n")
        wi.reopen_item(self.db, ns(atm_id=a, actor="AI", reason="test-failed", evidence="docs/note.md"))
        with self.assertRaises(ValueError):
            self.transition(a, "In progress", evidence="docs/note.md")

    def test_history_field_boundary_tokens_are_refused(self):
        self.use_evidence_root()
        atm = self.add(type="Bug")
        before = self.snapshot()
        for bad in ("x; evidence: y", "x;evidence:y", "x ; EVIDENCE : y", "x; Evidence: y"):
            with self.assertRaises(ValueError, msg=bad):
                self.transition(atm, "In progress", reason=bad)
            with self.assertRaises(ValueError, msg=bad):
                wi.update_item(self.db, ns(atm_id=atm, title="T2 title", description=None, scope=None, reproduction=None, acceptance=None, severity=None, actor="AI", reason=bad, evidence=""))
            with self.assertRaises(ValueError, msg=bad):
                wi.close_item(self.db, ns(atm_id=atm, status="fixed", evidence="evidence/evidence.txt", actor="AI", reason=bad))
        self.assertEqual(self.snapshot(), before)
        wi.close_item(self.db, ns(atm_id=atm, status="fixed", evidence="evidence/evidence.txt", actor="AI", reason="mentions evidence: only inline"))
        with self.assertRaises(ValueError):
            wi.reopen_item(self.db, ns(atm_id=atm, evidence="evidence/evidence.txt", actor="AI", reason="a; Evidence: b"))
        wi.reopen_item(self.db, ns(atm_id=atm, evidence="evidence/evidence.txt", actor="AI", reason="test-failed"))
        wi.export(self.db, self.docs, include_pdf=False)
        recovered = wi.connect(self.root / "rec.db")
        self.addCleanup(recovered.close)
        wi.import_markdown(recovered, self.docs)
        rows = lambda d: [tuple(r) for r in d.execute("SELECT event_type,actor,on_date,reason,evidence_path FROM item_history ORDER BY sequence")]
        self.assertEqual(rows(recovered), rows(self.db))

    def test_validate_flags_boundary_tokens_in_stored_history(self):
        atm = self.add()
        self.db.execute("INSERT INTO item_history(atm_id,event_type,actor,on_date,reason,evidence_path) VALUES(?,?,?,?,?,?)", (atm, "Updated", "AI", "2026-11-23", "a; evidence: b", "e"))
        self.refresh_hash()
        self.assertTrue(self.errors_containing("boundary"), wi.validate(self.db))

    def test_markdown_escaping_of_pipes_links_images_and_tags(self):
        self.assertEqual(wi.md_escape("a\\|y"), "a\\\\\\|y")
        self.assertEqual(wi.md_escape("a|b"), "a\\|b")
        self.assertNotIn("![", wi.md_escape("![x](http://e/i.png)"))
        text = LONG + " ![x](http://evil.example/i.png) <b>bold</b> <!-- c --> </div>"
        self.use_evidence_root()
        atm = self.add(description=text, title="T ![i](http://e/x) <u>t</u>")
        self.transition(atm, "In progress", reason="see ![h](http://evil/h.png) <i>why</i>")
        wi.export(self.db, self.docs, include_pdf=False)
        md = (self.docs / "Issues.md").read_text(encoding="utf-8")
        self.assertNotIn("![x](", md)
        for tag in ("<b>", "<!--", "<u>", "</div>", "<i>", "![h]("):
            self.assertNotRegex(md, r"(?<!\\)" + re.escape(tag))
        self.assertIn("\\<b>", md)
        recovered = wi.connect(self.root / "rec.db")
        self.addCleanup(recovered.close)
        wi.import_markdown(recovered, self.docs)
        item = wi.require_item(recovered, atm)
        self.assertEqual((item["description"], item["title"]), (text, "T ![i](http://e/x) <u>t</u>"))
        self.assertIn("see ![h](http://evil/h.png) <i>why</i>", [r[0] for r in recovered.execute("SELECT reason FROM item_history")])
        self.assertEqual(wi.diff(self.db, self.docs), [])

    def test_html_export_escapes_history_markup(self):  # m27
        self.use_evidence_root()
        atm = self.add()
        self.transition(atm, "In progress", reason="<img src=x onerror=alert(1)> & more")
        wi.export(self.db, self.docs, include_pdf=False)
        page = (self.docs / "Issues.html").read_text(encoding="utf-8")
        self.assertIn("&lt;img src=x onerror=alert(1)&gt; &amp; more", page)
        self.assertNotIn("<img src=x", page)

    def test_pdf_export_escapes_markup_in_values_and_history(self):  # m28
        try:
            import reportlab  # noqa: F401
        except ImportError:
            self.skipTest("reportlab unavailable")
        self.use_evidence_root()
        atm = self.add(description=LONG + " <notatag> & <b>unclosed more </i>")
        self.transition(atm, "In progress", reason="<para>x & y")
        paths = wi.pdf_documents(self.db, self.docs)
        self.assertEqual(len(paths), 4)
        self.assertTrue(all(p.stat().st_size > 500 for p in paths))

    def test_integrity_detects_history_reordering(self):  # m32
        self.use_evidence_root()
        atm = self.add()
        self.transition(atm, "In progress")
        self.transition(atm, "Queued", reason="paused")
        seqs = [r[0] for r in self.db.execute("SELECT sequence FROM item_history ORDER BY sequence")]
        self.assertEqual(len(seqs), 3)
        # Distinct dates that agree with the original order, so only the sequence column can tell a reorder.
        for seq, day in zip(seqs, ("2026-11-21", "2026-11-22", "2026-11-23")):
            self.db.execute("UPDATE item_history SET on_date=? WHERE sequence=?", (day, seq))
        self.refresh_hash()
        self.assertEqual(wi.validate(self.db), [])
        self.db.execute("UPDATE item_history SET sequence=-1 WHERE sequence=?", (seqs[1],))
        self.db.execute("UPDATE item_history SET sequence=? WHERE sequence=?", (seqs[1], seqs[2]))
        self.db.execute("UPDATE item_history SET sequence=? WHERE sequence=-1", (seqs[2],))
        self.assertTrue(self.errors_containing("integrity"), "history reorder (same dates) not detected")

    def test_validate_reports_control_characters_in_history_rows(self):  # m34
        atm = self.add()
        self.db.execute("INSERT INTO item_history(atm_id,event_type,actor,on_date,reason,evidence_path) VALUES(?,?,?,?,?,?)", (atm, "Updated", "AI", "2026-11-23", "bad‮reason", "e"))
        self.refresh_hash()
        errors = self.errors_containing("history")
        self.assertTrue(any("control characters" in e for e in errors), wi.validate(self.db))

    def test_more_invisible_characters_are_refused(self):
        self.use_evidence_root()
        atm = self.add()
        for ch in ("᠎", "؜", "ㅤ", "⠀", "￹", "￻", "͏", "ᅟ", "⁡", "⁤", "\U000e0001", "\U000e007f"):
            with self.assertRaises(ValueError, msg=hex(ord(ch))):
                self.transition(atm, "In progress", reason="a" + ch + "b")
            with self.assertRaises(ValueError, msg=hex(ord(ch))):
                wi.add_item(self.db, new_item_args(title="a" + ch + "b"))

    def test_variation_selectors_and_size_cap_and_git_evidence_and_bad_key(self):
        self.use_evidence_root()
        atm = self.add()
        for ch in ("\ufe0f", "\ufe00", "\u180b", "\U000e0100", "\U000e01ef"):
            with self.assertRaises(ValueError, msg=hex(ord(ch))):
                wi.add_item(self.db, new_item_args(title="a" + ch + "b"))
        with self.assertRaisesRegex(ValueError, "too long"):
            wi.add_item(self.db, new_item_args(description="x" * (wi.MAX_FIELD_CHARS + 1)))
        wi.add_item(self.db, new_item_args(description=("word " * wi.MAX_FIELD_CHARS)[:wi.MAX_FIELD_CHARS]))
        (self.root / ".git").mkdir()
        (self.root / ".git" / "config").write_text("c\n")
        for bad in (".git", ".git/config", "evidence/../.git/config"):
            with self.assertRaises(ValueError, msg=bad):
                wi.repo_evidence(bad)
        (self.root / "evidence" / "gitlink").symlink_to(self.root / ".git")
        with self.assertRaises(ValueError, msg="symlink into .git"):
            wi.repo_evidence("evidence/gitlink/config")
        self.assertEqual(wi.repo_evidence("evidence/evidence.txt"), "evidence/evidence.txt")
        before = self.snapshot()
        with self.assertRaises(ValueError):
            wi.update_item(self.db, ns(atm_id=f"{KEY}-999", title="t", description=None, scope=None, reproduction=None,
                                       acceptance=None, severity=None, actor="AI", reason="r", evidence=""))
        self.assertEqual(self.snapshot(), before)
        import subprocess, sys, os
        done = subprocess.run([sys.executable, str(ROOT / "scripts" / "workable_items.py"), "--db", str(self.root / "x.db"), "validate"],
                              env={**os.environ, "HELIX_WORKABLE_ITEMS_KEY": "hrd"}, capture_output=True, text=True)
        self.assertEqual(done.returncode, 2, done.stderr)
        self.assertTrue(done.stderr.startswith("error:"), done.stderr)
        self.assertNotIn("Traceback", done.stderr)

    def test_triggers_and_views_make_the_registry_invalid(self):
        self.use_evidence_root()
        atm = self.add()
        self.db.execute("CREATE VIEW evil AS SELECT atm_id FROM items")
        self.assertTrue(self.errors_containing("view"), wi.validate(self.db))
        with self.assertRaises(ValueError):
            self.transition(atm, "In progress")
        self.db.execute("DROP VIEW evil")
        self.assertEqual(wi.validate(self.db), [])
        self.db.execute("CREATE TRIGGER evil_t AFTER UPDATE ON items BEGIN UPDATE items SET severity='Low'; END")
        self.assertTrue(self.errors_containing("trigger"), wi.validate(self.db))
        with self.assertRaises(ValueError):
            self.transition(atm, "In progress")
        with self.assertRaises(ValueError):
            self.rehash()

    def test_lifecycle_state_rules(self):
        self.use_evidence_root()
        a = self.add(type="Bug")
        with self.assertRaises(ValueError, msg="reopen of a Queued item"):
            wi.reopen_item(self.db, ns(atm_id=a, actor="AI", reason="test-failed", evidence="evidence/evidence.txt"))
        blocked = self.add()
        wi.block_item(self.db, ns(atm_id=blocked, what="W", why="Attempted: x.", unblock="U", who="O"))
        wi.close_item(self.db, ns(atm_id=blocked, status="completed", evidence="evidence/evidence.txt", actor="AI", reason="unblocked and done"))
        self.assertEqual(self.db.execute("SELECT count(*) FROM operator_block_details WHERE atm_id=?", (blocked,)).fetchone()[0], 0)
        self.assertEqual(wi.validate(self.db), [], wi.validate(self.db))
        with self.assertRaises(ValueError, msg="close of a closed item"):
            wi.close_item(self.db, ns(atm_id=blocked, status="completed", evidence="evidence/evidence.txt", actor="AI", reason="again"))
        with self.assertRaises(ValueError, msg="block of a closed item"):
            wi.block_item(self.db, ns(atm_id=blocked, what="W", why="Attempted: x.", unblock="U", who="O"))
        wi.reopen_item(self.db, ns(atm_id=blocked, actor="AI", reason="test-failed", evidence="evidence/evidence.txt"))
        with self.assertRaises(ValueError, msg="reopen of a Reopened item"):
            wi.reopen_item(self.db, ns(atm_id=blocked, actor="AI", reason="test-failed", evidence="evidence/evidence.txt"))
        self.assertEqual(wi.validate(self.db), [], wi.validate(self.db))

    def test_schema_item_hashes_baseline_is_maintained(self):
        atm = self.add()
        stored = json.loads(self.db.execute("SELECT value FROM meta WHERE key='item_hashes'").fetchone()[0])
        self.assertEqual(sorted(stored), [atm])

    # --- fourth review round ----------------------------------------------------

    def blocked_and_reopened_docs(self):
        self.use_evidence_root()
        r = self.add(type="Bug")
        wi.close_item(self.db, ns(atm_id=r, status="fixed", evidence="evidence/evidence.txt", actor="AI", reason="v"))
        wi.reopen_item(self.db, ns(atm_id=r, evidence="evidence/evidence.txt", actor="AI", reason="test-failed"))
        b = self.add()
        wi.block_item(self.db, ns(atm_id=b, what="Blocked surface", why="Attempted: all paths.", unblock="A device", who="Operator"))
        wi.export(self.db, self.docs, include_pdf=False)
        return r, b

    def test_import_plan_covers_reopen_and_block_detail_tables(self):
        r, b = self.blocked_and_reopened_docs()
        self.edit_doc("Issues.md", "Reason: test-failed;", "Reason: forged-reason;")
        self.edit_doc("Issues.md", "WHO: Operator", "WHO: Mallory")
        before = self.snapshot()
        with self.assertRaises(ValueError) as ctx:
            wi.import_markdown(self.db, self.docs)
        message = str(ctx.exception)
        self.assertIn("reopen_details", message)
        self.assertIn("operator_block_details", message)
        self.assertEqual(self.snapshot(), before)
        plan = wi.plan_import(self.db, self.docs)
        self.assertTrue(any(r in line and "reopen_details" in line and "forged-reason" in line for line in plan), plan)
        self.assertTrue(any(b in line and "operator_block_details" in line and "Mallory" in line for line in plan), plan)

    def test_import_overwrite_requires_actor_reason_and_is_logged_in_the_hash(self):
        atm = self.closed_item_docs()
        self.edit_doc("Fixed.md", "**Severity:** High", "**Severity:** Low")
        for kw in ({}, {"actor": "User"}, {"reason": "r"}, {"actor": "User", "reason": "  "}, {"actor": "AI\n", "reason": "r"}):
            with self.assertRaises(ValueError, msg=str(kw)):
                wi.import_markdown(self.db, self.docs, allow_overwrite=True, **kw)
        self.assertEqual(wi.require_item(self.db, atm)["severity"], "High")
        wi.import_markdown(self.db, self.docs, allow_overwrite=True, actor="User", reason="severity corrected after review")
        log = self.db.execute("SELECT value FROM meta WHERE key='import_log'").fetchone()[0]
        entry = json.loads(log.split("\n")[-1])
        self.assertEqual((entry["actor"], entry["reason"]), ("User", "severity corrected after review"))
        self.assertTrue(any(atm in c and "severity" in c for c in entry["changes"]))
        self.assertEqual(wi.validate(self.db), [])
        self.db.execute("UPDATE meta SET value=? WHERE key='import_log'", (log.replace("corrected", "forged"),))
        self.assertTrue(self.errors_containing("integrity"), "import_log is not covered by the hash")

    def test_import_confines_reopen_and_history_evidence_and_rejects_bad_markdown(self):
        r, b = self.blocked_and_reopened_docs()
        fresh = wi.connect(self.root / "fresh.db")
        self.addCleanup(fresh.close)
        def attempt(name, old, new, count=1):
            self.edit_doc(name, old, new)
            try:
                with self.assertRaises(ValueError, msg=new):
                    wi.import_markdown(fresh, self.docs)
                self.assertEqual(wi.items(fresh), [], new)
            finally:
                self.edit_doc(name, new, old)

        text = (self.docs / "Issues.md").read_text(encoding="utf-8")
        reopened_line = next(l for l in text.split("\n") if l.startswith("**Reopened-Details:**"))
        attempt("Issues.md", reopened_line, reopened_line.rsplit("Evidence: ", 1)[0] + "Evidence: /etc/shadow")
        history_line = next(l for l in text.split("\n") if l.startswith("- ") and "Reopened by" in l)
        attempt("Issues.md", history_line, history_line.rsplit("evidence: ", 1)[0] + "evidence: ../../etc/passwd")
        attempt("Issues.md", history_line, "- 2026-01-01 — Hacked by AI: nope; evidence: none recorded\n" + history_line)
        # Duplicate identifiers in the Markdown.
        section = text[text.index("## [" + r + "]"):]
        section = section[:section.index("\n## [")] if "\n## [" in section[3:] else section
        path = self.docs / "Issues.md"
        path.write_text(text + "\n" + section, encoding="utf-8")
        with self.assertRaises(ValueError, msg="duplicate id accepted"):
            wi.import_markdown(fresh, self.docs)
        path.write_text(text, encoding="utf-8")
        self.assertEqual(wi.import_markdown(fresh, self.docs), 2)

    def test_attribution_and_block_labels_cannot_shift_field_boundaries(self):
        atm = self.add()
        for field in ("created_by", "assigned_to"):
            for bad in ("x; assigned to y", "x;assigned  to y", "X; ASSIGNED TO y"):
                with self.assertRaises(ValueError, msg=f"{field}={bad!r}"):
                    wi.add_item(self.db, new_item_args(**{field: bad}))
        for field in ("what", "why", "unblock", "who"):
            for label in (" WHY: ", " UNBLOCK CONDITION: ", " WHO: "):
                args = dict(atm_id=atm, what="W", why="Attempted: x.", unblock="U", who="O")
                args[field] = args[field] + "a" + label + "b"
                with self.assertRaises(ValueError, msg=f"{field} {label!r}"):
                    wi.block_item(self.db, ns(**args))
        self.assertEqual(wi.require_item(self.db, atm)["status"], "Queued")
        b = self.add(created_by="User", assigned_to="Reviewer team")
        wi.block_item(self.db, ns(atm_id=b, what="Why not who: plain", why="Attempted: all, recorded.", unblock="When ready", who="Operator"))
        wi.export(self.db, self.docs, include_pdf=False)
        rec = wi.connect(self.root / "rec.db")
        self.addCleanup(rec.close)
        wi.import_markdown(rec, self.docs)
        for table in ("operator_block_details",):
            self.assertEqual([tuple(r) for r in rec.execute(f"SELECT * FROM {table}")], [tuple(r) for r in self.db.execute(f"SELECT * FROM {table}")])
        self.assertEqual([(i["created_by"], i["assigned_to"]) for i in wi.items(rec)], [(i["created_by"], i["assigned_to"]) for i in wi.items(self.db)])

    def test_stored_attribution_and_block_labels_are_flagged_by_validate(self):
        atm = self.add()
        self.db.execute("UPDATE items SET created_by=? WHERE atm_id=?", ("x; assigned to y", atm))
        self.refresh_hash()
        self.assertTrue(self.errors_containing("boundary"), wi.validate(self.db))

    def test_integrity_covers_item_hashes_baseline(self):
        self.add()
        raw = self.db.execute("SELECT value FROM meta WHERE key='item_hashes'").fetchone()[0]
        self.db.execute("UPDATE meta SET value=? WHERE key='item_hashes'", (raw.replace("0", "1", 1) if "0" in raw else raw + " ",))
        self.assertTrue(self.errors_containing("integrity"), "item_hashes edit not detected")

    def test_item_snapshot_covers_history(self):
        self.use_evidence_root()
        a = self.add()
        b = self.add()
        self.db.execute("INSERT INTO item_history(atm_id,event_type,actor,on_date,reason,evidence_path) VALUES(?,?,?,?,?,?)", (b, "Updated", "AI", "2026-11-23", "sneaky", "e"))
        dry = wi.rehash(self.db, ns(actor="User", reason="r", confirm=None, dry_run=True))
        self.assertEqual(dry["changed"], [b])

    def test_export_rechecks_tampering_after_building_documents(self):
        self.use_evidence_root()
        atm = self.add()
        wi.export(self.db, self.docs, include_pdf=False)
        before = self.db.execute("SELECT value FROM meta WHERE key='integrity_hash'").fetchone()[0]
        original = wi.html_documents
        def tamper_then_build(db):
            db.execute("UPDATE items SET title='Concurrent tamper' WHERE atm_id=?", (atm,))
            return original(db)
        wi.html_documents = tamper_then_build
        try:
            with self.assertRaises(ValueError) as ctx:
                wi.export(self.db, self.docs, include_pdf=False)
        finally:
            wi.html_documents = original
        self.assertIn("tampered", str(ctx.exception))
        self.assertEqual(self.db.execute("SELECT value FROM meta WHERE key='integrity_hash'").fetchone()[0], before, "export blessed concurrent tampering")

    def test_validate_errors_name_the_affected_ids(self):
        for _ in range(4):
            self.add()
        self.db.execute("PRAGMA foreign_keys=OFF")
        self.db.execute("DELETE FROM items WHERE atm_id IN (?,?)", (f"{KEY}-002", f"{KEY}-003"))
        self.refresh_hash()
        gap = [e for e in wi.validate(self.db) if "allocation" in e]
        self.assertEqual(len(gap), 1)
        self.assertIn(f"{KEY}-002", gap[0])
        self.assertIn(f"{KEY}-003", gap[0])
        self.assertTrue(any("foreign key" in e and f"{KEY}-002" in e for e in wi.validate(self.db)), wi.validate(self.db))

    def test_outbox_refuses_a_tampered_registry(self):
        atm = self.add()
        self.db.execute("UPDATE items SET title='Tampered title' WHERE atm_id=?", (atm,))
        out = self.root / "outbox.json"
        with self.assertRaises(ValueError) as ctx:
            wi.helixtrack_outbox(self.db, out)
        self.assertIn("tampered", str(ctx.exception))
        self.assertFalse(out.exists())

    def test_read_only_database_validates_without_traceback(self):
        import os, stat, subprocess, sys
        self.add()
        self.db.execute("PRAGMA wal_checkpoint(TRUNCATE)")
        self.db.close()
        path = self.root / "workable_items.db"
        os.chmod(path, 0o444)
        self.addCleanup(os.chmod, path, 0o644)
        base = [sys.executable, str(ROOT / "scripts" / "workable_items.py"), "--db", str(path)]
        done = subprocess.run(base + ["validate"], text=True, capture_output=True, cwd=ROOT)
        self.assertNotIn("Traceback", done.stderr)
        self.assertEqual(done.returncode, 0, done.stderr)
        self.assertEqual(json.loads(done.stdout)["status"], "PASS")
        done = subprocess.run(base + ["start", "HRD-001", "--actor", "AI", "--reason", "r", "--evidence", "evidence/workable-items"], text=True, capture_output=True, cwd=ROOT)
        self.assertEqual(done.returncode, 2)
        self.assertNotIn("Traceback", done.stderr)
        self.assertIn("error:", done.stderr)
        missing = subprocess.run([sys.executable, str(ROOT / "scripts" / "workable_items.py"), "--db", str(self.root / "nope.db"), "validate"], text=True, capture_output=True, cwd=ROOT)
        self.assertNotIn("Traceback", missing.stderr)
        self.assertEqual(missing.returncode, 2)
        self.db = wi.connect(self.root / "other.db")

    def test_unblock_is_a_recorded_transition(self):
        self.use_evidence_root()
        atm = self.add()
        args = dict(atm_id=atm, actor="User", reason="device attached", evidence="evidence/evidence.txt")
        with self.assertRaises(ValueError, msg="unblock of a Queued item"):
            wi.unblock_item(self.db, ns(**args))
        wi.block_item(self.db, ns(atm_id=atm, what="W", why="Attempted: x.", unblock="U", who="O"))
        for bad in ({"evidence": "../outside"}, {"reason": " "}, {"reason": "a; evidence: b"}, {"actor": "AI\n"}):
            with self.assertRaises(ValueError, msg=str(bad)):
                wi.unblock_item(self.db, ns(**{**args, **bad}))
        self.assertEqual(wi.require_item(self.db, atm)["status"], "Operator-blocked")
        wi.unblock_item(self.db, ns(**args))
        self.assertEqual(wi.require_item(self.db, atm)["status"], "Queued")
        self.assertEqual(self.db.execute("SELECT count(*) FROM operator_block_details").fetchone()[0], 0)
        last = self.db.execute("SELECT event_type,actor,reason,evidence_path FROM item_history WHERE atm_id=? ORDER BY sequence DESC", (atm,)).fetchone()
        self.assertEqual(tuple(last), ("Unblocked", "User", "device attached", "evidence/evidence.txt"))
        self.assertEqual(wi.validate(self.db), [], wi.validate(self.db))
        wi.export(self.db, self.docs, include_pdf=False)
        rec = wi.connect(self.root / "rec.db")
        self.addCleanup(rec.close)
        wi.import_markdown(rec, self.docs)
        self.assertEqual(rec.execute("SELECT count(*) FROM item_history WHERE event_type='Unblocked'").fetchone()[0], 1)
        self.transition(atm, "In progress")

    def test_operator_block_requires_exhaustion_and_details(self):
        atm = self.add()
        with self.assertRaises(ValueError):
            wi.block_item(self.db, ns(atm_id=atm, what="Blocked surface", why="Could not do it",
                                      unblock="Credentials exist", who="Operator"))
        wi.block_item(self.db, ns(atm_id=atm, what="Blocked surface",
                                  why="Attempted: three documented approaches, each recorded as failing.",
                                  unblock="A target instance exists.", who="Operator"))
        item = wi.require_item(self.db, atm)
        self.assertEqual((item["status"], item["current_location"]), ("Operator-blocked", "Issues"))
        self.assertEqual(wi.validate(self.db), [], wi.validate(self.db))
        self.db.execute("DELETE FROM operator_block_details WHERE atm_id=?", (atm,))
        self.assertTrue(self.errors_containing("incomplete operator-block details"))
        self.refresh_hash()
        wi.block_item(self.db, ns(atm_id=atm, what="Blocked surface", why="Attempted: recorded again.",
                                  unblock="A target instance exists.", who="Operator"))
        self.db.execute("UPDATE operator_block_details SET why_exhausted_alternatives='Tried some things' WHERE atm_id=?", (atm,))
        self.assertTrue(self.errors_containing("missing self-resolution exhaustion audit"))

    def test_operator_block_command_round_trip(self):
        blocked = self.add()
        wi.block_item(self.db, ns(atm_id=blocked, what="Real reader hardware is absent",
                                  why="Attempted: emulator, vendor cloud, loopback harness; all recorded as insufficient.",
                                  unblock="A device is attached to the qualification host.", who="Operator"))
        reopened = self.add(type="Feature")
        wi.close_item(self.db, ns(atm_id=reopened, status="implemented", evidence="evidence/campaign-2026-11-20/core",
                                  actor="AI", reason="verified"))
        wi.reopen_item(self.db, ns(atm_id=reopened, evidence="evidence/campaign-2026-11-20/formats",
                                   actor="User", reason="behavior regressed on the device"))
        wi.export(self.db, self.docs, include_pdf=False)
        recovered = wi.connect(self.root / "recovered.db")
        self.addCleanup(recovered.close)
        self.assertEqual(wi.import_markdown(recovered, self.docs), 2)
        self.assertEqual(wi.validate(recovered), [], wi.validate(recovered))
        detail = recovered.execute("SELECT * FROM operator_block_details WHERE atm_id=?", (blocked,)).fetchone()
        self.assertEqual(dict(detail)["why_exhausted_alternatives"],
                         "Attempted: emulator, vendor cloud, loopback harness; all recorded as insufficient.")
        self.assertEqual(dict(detail)["unblock_condition"], "A device is attached to the qualification host.")
        reopen = recovered.execute("SELECT * FROM reopen_details WHERE atm_id=?", (reopened,)).fetchone()
        self.assertEqual((reopen["actor"], reopen["reason"]), ("User", "behavior regressed on the device"))
        self.assertEqual(recovered.execute("SELECT status,current_location,reopens_count FROM items WHERE atm_id=?", (reopened,)).fetchone()[0:3],
                         ("Reopened", "Issues", 1))
        self.assertEqual([row[0] for row in recovered.execute("SELECT event_type FROM item_history WHERE atm_id=? ORDER BY sequence", (reopened,))],
                         ["Opened", "Implemented", "Reopened"])

    # --- generated documents ---------------------------------------------------

    def test_html_and_markdown_escape_content(self):
        payload = 'Pipe | break and <img src=x onerror="alert(1)">'
        self.add(title=payload, description=LONG, reproduction=payload)
        wi.export(self.db, self.docs, include_pdf=False)
        markdown = (self.docs / "Issues.md").read_text(encoding="utf-8")
        summary = (self.docs / "Issues_Summary.md").read_text(encoding="utf-8")
        page = (self.docs / "Issues.html").read_text(encoding="utf-8")
        # Only the summary is a table, so only the summary escapes pipes; the long-form
        # body keeps content verbatim, which is what the importer round-trips.
        self.assertIn("Pipe | break", markdown)
        self.assertIn("Pipe \\| break", summary, "table cells must not be splittable by content pipes")
        self.assertNotIn("<img src=x", page, "raw markup from an item field reaches the HTML")
        self.assertIn(html_module.escape(payload), page)
        self.assertIn("&lt;img src=x", page)

    def test_derived_documents_carry_history_and_lifecycle_details(self):
        # History and lifecycle details used to be rendered into Markdown only. A close
        # reason naming another identifier then reached Fixed.md but not Fixed.html or
        # Fixed.pdf, so the format-currency gate compared two documents that the
        # generator had never agreed to make equal.
        tracked = self.add()
        wi.block_item(self.db, ns(atm_id=tracked, what="Real reader hardware is absent",
                                  why="Attempted: emulator, vendor cloud, loopback harness; all recorded as insufficient.",
                                  unblock="A device is attached to the qualification host.", who="Operator"))
        closed = self.add(type="Feature")
        wi.close_item(self.db, ns(atm_id=closed, status="implemented", evidence="evidence/campaign-2026-11-20/containers",
                                  actor="AI", reason=f"residual tracked separately as {tracked}"))
        wi.export(self.db, self.docs, include_pdf=False)
        documents = wi.markdown_documents(self.db)
        self.assertIn(tracked, documents["Fixed.md"], "closure history is missing from Markdown")
        fixed_html = (self.docs / "Fixed.html").read_text(encoding="utf-8")
        self.assertIn(tracked, fixed_html, "closure history is missing from HTML")
        self.assertIn("residual tracked separately as", fixed_html, "closure reason text is missing from HTML history")
        issues_html = (self.docs / "Issues.html").read_text(encoding="utf-8")
        self.assertIn("Attempted: emulator, vendor cloud", issues_html, "operator-block exhaustion detail is missing from HTML")
        # Every field the registry holds for an item must appear in every long-form
        # format, not only the ones a previous renderer happened to enumerate.
        for label, _ in wi.long_form_fields(wi.require_item(self.db, tracked), self.db):
            self.assertIn(f"<h3>{label}</h3>", issues_html, f"Issues.html omits the {label} section")
            self.assertIn(f"**{label}:**", documents["Issues.md"], f"Issues.md omits the {label} section")

    def test_export_and_drift_detection(self):
        self.add()
        # Nothing generated yet is drift, not "no opinion": diff names every file a
        # reader would otherwise find missing.
        self.assertEqual(sorted(wi.diff(self.db, self.docs)), sorted(
            ["Issues.md", "Fixed.md", "Issues_Summary.md", "Fixed_Summary.md",
             "Issues.html", "Fixed.html", "Issues_Summary.html", "Fixed_Summary.html"]))
        written = wi.export(self.db, self.docs, include_pdf=False)
        names = sorted(path.name for path in written)
        self.assertEqual(names, sorted(["Issues.md", "Fixed.md", "Issues_Summary.md", "Fixed_Summary.md",
                                        "Issues.html", "Fixed.html", "Issues_Summary.html", "Fixed_Summary.html"]))
        self.assertEqual(wi.diff(self.db, self.docs), [])
        path = self.docs / "Issues.md"
        path.write_text(path.read_text(encoding="utf-8") + "\nHand-edited after generation.\n", encoding="utf-8")
        self.assertEqual(["Issues.md"], wi.diff(self.db, self.root / "docs"))
        # Re-exporting is the repair, and it must be byte-stable.
        first = (self.docs / "Issues.md").read_bytes()
        wi.export(self.db, self.docs, include_pdf=False)
        self.assertEqual(wi.diff(self.db, self.docs), [])
        self.assertNotEqual(first, (self.docs / "Issues.md").read_bytes())
        self.assertEqual({row["key"]: row["value"] for row in self.db.execute("SELECT key,value FROM meta")}["last_sync_direction"],
                         "db-to-docs")

    def test_markdown_database_markdown_round_trip(self):
        for kind in ("Bug", "Feature", "Task"):
            atm = self.add(type=kind, created_by="User", assigned_to="Reviewer")
            if kind == "Feature":
                self.db.execute("UPDATE items SET composes_with=? WHERE atm_id=?", (json.dumps(["core/formats"]), atm))
                self.refresh_hash()
            wi.close_item(self.db, ns(atm_id=atm, status={"Bug": "fixed", "Feature": "implemented", "Task": "completed"}[kind],
                                      evidence="evidence/workable-items", actor="AI", reason="verified"))
        wi.export(self.db, self.docs, include_pdf=False)
        recovered = wi.connect(self.root / "recovered.db")
        self.addCleanup(recovered.close)
        self.assertEqual(wi.import_markdown(recovered, self.docs), 3)
        self.assertEqual(wi.validate(recovered), [], wi.validate(recovered))
        self.assertEqual(wi.integrity(recovered), wi.integrity(self.db), "recovered registry diverges from the source")
        documents = wi.markdown_documents(recovered)
        for name, content in wi.markdown_documents(self.db).items():
            self.assertEqual(content, documents[name], name)
        self.assertEqual(sorted(documents), ["Fixed.md", "Fixed_Summary.md", "Issues.md", "Issues_Summary.md"])
        self.assertIn("created by User", documents["Fixed.md"], "attribution is lost in the round trip")
        self.assertIn("core/formats", documents["Fixed.md"])

    def test_outbox_never_claims_remote_push(self):
        self.add()
        path = self.docs / "tracking" / "helixtrack-outbox.json"
        wi.helixtrack_outbox(self.db, path)
        envelope = json.loads(path.read_text(encoding="utf-8"))
        self.assertEqual(envelope["schema"], "helix.workable-items.outbox.v1")
        self.assertEqual(len(envelope["records"]), 1)
        self.assertEqual(envelope["records"][0]["remote_status"], "NOT RUN")
        self.assertEqual(envelope["records"][0]["operation"], "upsert")
        self.assertEqual(envelope["records"][0]["client_key"], f"{KEY}-001")
        self.assertIn("not", envelope["records"][0]["reason"].lower())
        for record in envelope["records"]:
            self.assertNotIn(record.get("remote_status"), {"pushed", "synced", "ok", "complete"})
        # Idempotent and keyed: regenerating with the same registry changes nothing but
        # the timestamp, so a retry can never be mistaken for a second delivery.
        again = json.loads((lambda: (wi.helixtrack_outbox(self.db, path), path.read_text(encoding="utf-8"))[1])())
        for left, right in zip(envelope["records"], again["records"]):
            self.assertEqual(left, right)
        self.assertEqual(envelope["schema"], again["schema"])


class ProjectKeyTest(unittest.TestCase):
    """No ticket-id key may be hardcoded: a different project changes ONE value."""

    def setUp(self) -> None:
        self.original = wi.PROJECT_KEY
        self.addCleanup(wi.set_project_key, self.original)
        temp = tempfile.TemporaryDirectory(prefix="helix-wi-key-")
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name)

    def test_default_key_is_hrd_and_never_atm(self):
        self.assertEqual(wi.DEFAULT_PROJECT_KEY, "HRD")
        self.assertIsNone(wi.ID_RE.match("ATM-001"))
        self.assertTrue(wi.ID_RE.match("HRD-001"))

    def test_foreign_project_key_end_to_end(self):
        wi.set_project_key("ZQX")
        db = wi.connect(self.root / "other.db")
        self.addCleanup(db.close)
        ids = [wi.add_item(db, new_item_args(title=f"Other project item {n}")) for n in range(3)]
        self.assertEqual(ids, ["ZQX-001", "ZQX-002", "ZQX-003"])
        self.assertEqual(wi.validate(db), [])
        docs = self.root / "docs"
        wi.export(db, docs, False)
        text = (docs / "Issues.md").read_text(encoding="utf-8") + (docs / "Issues_Summary.md").read_text(encoding="utf-8")
        self.assertIn("## [ZQX-001] ", text)
        self.assertIn("| ID |", text)
        for forbidden in ("HRD-", "ATM-", "ATM ID"):
            self.assertNotIn(forbidden, text)
        self.assertEqual(wi.import_markdown(db, docs), 3)
        self.assertEqual(wi.diff(db, docs), [])

    def test_foreign_key_rows_rejected_under_other_key(self):
        wi.set_project_key("ZQX")
        db = wi.connect(self.root / "mixed.db")
        self.addCleanup(db.close)
        wi.add_item(db, new_item_args())
        wi.set_project_key("HRD")
        self.assertTrue(any("invalid or duplicate HRD identifier" in e for e in wi.validate(db)))

    def test_invalid_key_rejected(self):
        for bad in ("", "a1", "X", "TOOLONGKEY", "ab"):
            with self.assertRaises(ValueError):
                wi.set_project_key(bad)


if __name__ == "__main__":
    unittest.main()
