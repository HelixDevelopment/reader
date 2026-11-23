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
        self.assertEqual(wi.validate(self.db), [f"{KEY} allocation contains gaps"])
        self.assertEqual(wi.add_item(self.db, new_item_args(title="Reallocated")), f"{KEY}-004")
        self.db.execute(f"INSERT INTO items(atm_id,type,status,severity,title,description,affected_scope,reproduction,acceptance,created_at,last_modified,current_location,forensic_anchor,closure_criteria,composes_with,evidence_path,created_by,assigned_to,reopens_count) VALUES('{KEY}-002','Task','Queued','High','Restored',?,?,?,?,?,?,'Issues','tests','Acceptance','[]','','AI','Operator',0)",
                        (LONG, "Scope of the restored row.", "Reinsert the row.", "Acceptance condition.", "2026-10-29T00:00:00Z", "2026-10-29T00:00:00Z"))
        self.db.execute(f"INSERT INTO item_history(atm_id,event_type,actor,on_date,reason,evidence_path) VALUES('{KEY}-002','Opened','AI','2026-10-29','new-work-item','')")
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
            wi.close_item(self.db, ns(atm_id=atm, status={"Bug": "fixed", "Feature": "implemented", "Task": "completed"}[kind],
                                      evidence=f"evidence/campaign-2026-11-20/{kind.lower()}", actor="AI", reason="verified"))
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
