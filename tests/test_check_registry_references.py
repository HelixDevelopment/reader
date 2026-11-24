#!/usr/bin/env python3
"""Tests for scripts/check_registry_references.py, with paired mutants (constitution section 1.1)."""
import json, shutil, sqlite3, subprocess, sys, tempfile, unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / "scripts" / "check_registry_references.py"


class CheckerTest(unittest.TestCase):
    def setUp(self):
        t = tempfile.TemporaryDirectory(prefix="helix-refcheck-")
        self.addCleanup(t.cleanup)
        self.root = Path(t.name)
        (self.root / "docs").mkdir()
        (self.root / "evidence").mkdir()
        db = sqlite3.connect(self.root / "docs" / "workable_items.db")
        db.execute("CREATE TABLE items(atm_id TEXT, evidence_path TEXT)")
        db.execute("CREATE TABLE item_history(atm_id TEXT, evidence_path TEXT)")
        db.execute("CREATE TABLE reopen_details(atm_id TEXT, evidence_path TEXT)")
        db.execute("CREATE TABLE obsolete_details(atm_id TEXT, triple_check_evidence TEXT)")
        db.execute("CREATE TABLE operator_block_details(atm_id TEXT, what TEXT)")
        db.execute("INSERT INTO reopen_details VALUES('HRD-001','evidence/reopen-gone.txt')")
        db.execute("INSERT INTO obsolete_details VALUES('HRD-001','evidence/obsolete-gone.txt')")
        db.execute("INSERT INTO items VALUES('HRD-001','evidence/a.txt; evidence/gone.txt (note)')")
        db.execute("INSERT INTO item_history VALUES('HRD-001','none')")
        db.commit(); db.close()
        (self.root / "evidence" / "a.txt").write_text("a\n")
        (self.root / "docs" / "Issues.md").write_text("see docs/present.md and docs/absent.md#frag and docs/Glob*.md\n")
        (self.root / "docs" / "present.md").write_text("p\n")
        (self.root / "docs" / "tracking").mkdir()
        (self.root / "docs" / "tracking" / "pending_ownership.json").write_text(json.dumps({"entries": [
            {"prefix": "evidence/reopen-gone.txt", "task": "HRD-900", "pr": 77},
            {"prefix": "evidence/obsolete-gone.txt", "task": "HRD-901", "pr": None}]}))
        (self.root / "docs" / "upstream-issues").mkdir()
        (self.root / "docs" / "upstream-issues" / "n.md").write_text("see docs/in-upstream-doc-gone.md\n")

    def run_script(self, script=SCRIPT):
        done = subprocess.run([sys.executable, str(script), "--root", str(self.root), "--json"], capture_output=True, text=True)
        return done.returncode, (json.loads(done.stdout) if done.stdout.strip() else {})

    def test_reports_missing_evidence_and_doc_links_but_not_present_ones(self):
        rc, res = self.run_script()
        self.assertEqual(rc, 1, res)
        paths = {m["path"] for m in res["items"]}
        self.assertEqual(paths, {"evidence/gone.txt", "docs/absent.md", "evidence/reopen-gone.txt", "evidence/obsolete-gone.txt", "docs/in-upstream-doc-gone.md"})
        classes = {m["path"]: m["class"] for m in res["items"]}
        self.assertEqual(classes["evidence/reopen-gone.txt"], "OWNED-BY-PENDING-PR")
        self.assertEqual(classes["evidence/obsolete-gone.txt"], "OWNED-BY-UNDELIVERED-WORK")
        self.assertEqual(classes["evidence/gone.txt"], "NOWHERE")

    def test_passes_when_everything_exists(self):
        (self.root / "evidence" / "gone.txt").write_text("g\n")
        (self.root / "docs" / "absent.md").write_text("x\n")
        for n in ("reopen-gone.txt", "obsolete-gone.txt"):
            (self.root / "evidence" / n).write_text("x\n")
        (self.root / "docs" / "in-upstream-doc-gone.md").write_text("x\n")
        rc, res = self.run_script()
        self.assertEqual((rc, res["status"], res["missing"]), (0, "PASS", 0), res)

    def test_control_needle_blind_when_no_stored_path_resolves(self):
        (self.root / "evidence" / "a.txt").unlink()
        done = subprocess.run([sys.executable, str(SCRIPT), "--root", str(self.root)], capture_output=True, text=True)
        self.assertEqual(done.returncode, 2, done.stdout + done.stderr)

    def test_missing_database_is_blind_not_a_pass(self):
        (self.root / "docs" / "workable_items.db").unlink()
        done = subprocess.run([sys.executable, str(SCRIPT), "--root", str(self.root)], capture_output=True, text=True)
        self.assertEqual(done.returncode, 2, done.stdout + done.stderr)

    def test_paired_mutants_are_killed(self):
        source = SCRIPT.read_text()
        mutants = {
            "never-missing": ('        if not exists(p):\n            missing.append({"kind": "registry-evidence"', '        if False:\n            missing.append({"kind": "registry-evidence"'),
            "doc-links-ignored": ('for m in sorted(found):', 'for m in []:'),
            "needle-removed": ('if not any(exists(p) for _, _, p in paths):', 'if False:'),
            "doc-scan-drops-upstream-issues": ('"docs/upstream-issues/*.md", ', ''),
            "detail-tables-ignored": ('TABLES = ("items", "item_history", "reopen_details", "obsolete_details", "operator_block_details")', 'TABLES = ("items", "item_history")'),
            "classify-everything-nowhere": ('return ("OWNED-BY-PENDING-PR" if e.get("pr") else "OWNED-BY-UNDELIVERED-WORK"), e', 'return "NOWHERE", None'),
            "blind-is-pass": ('print("BLIND: registry database not found", file=sys.stderr); return 2', 'return 0'),
        }
        for name, (old, new) in mutants.items():
            self.assertIn(old, source, name)
            mutated = self.root / f"mutant_{name}.py"
            mutated.write_text(source.replace(old, new, 1))
            if name == "blind-is-pass":
                (self.root / "docs" / "workable_items.db").unlink()
                done = subprocess.run([sys.executable, str(mutated), "--root", str(self.root)], capture_output=True, text=True)
                self.assertEqual(done.returncode, 0, name)  # the mutant misbehaves ...
                self.assertNotEqual(subprocess.run([sys.executable, str(SCRIPT), "--root", str(self.root)], capture_output=True).returncode, 0)  # ... the real one does not
                continue
            if name == "needle-removed":
                (self.root / "evidence" / "a.txt").unlink()
                done = subprocess.run([sys.executable, str(mutated), "--root", str(self.root)], capture_output=True, text=True)
                self.assertNotEqual(done.returncode, 2, name)  # the mutant stops being blind ...
                self.assertEqual(subprocess.run([sys.executable, str(SCRIPT), "--root", str(self.root)], capture_output=True).returncode, 2)
                (self.root / "evidence" / "a.txt").write_text("a\n")
                continue
            rc, res = self.run_script(mutated)
            real_rc, real_res = self.run_script()
            self.assertEqual(real_rc, 1)
            self.assertNotEqual(res.get("items"), real_res.get("items"), f"mutant {name} survived")


if __name__ == "__main__":
    unittest.main()
