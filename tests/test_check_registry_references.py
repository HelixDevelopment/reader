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
        db.execute("INSERT INTO items VALUES('HRD-001','evidence/a.txt; evidence/gone.txt (note)')")
        db.execute("INSERT INTO item_history VALUES('HRD-001','none')")
        db.commit(); db.close()
        (self.root / "evidence" / "a.txt").write_text("a\n")
        (self.root / "docs" / "Issues.md").write_text("see docs/present.md and docs/absent.md#frag and docs/Glob*.md\n")
        (self.root / "docs" / "present.md").write_text("p\n")

    def run_script(self, script=SCRIPT):
        done = subprocess.run([sys.executable, str(script), "--root", str(self.root), "--json"], capture_output=True, text=True)
        return done.returncode, (json.loads(done.stdout) if done.stdout.strip() else {})

    def test_reports_missing_evidence_and_doc_links_but_not_present_ones(self):
        rc, res = self.run_script()
        self.assertEqual(rc, 1, res)
        paths = {m["path"] for m in res["items"]}
        self.assertEqual(paths, {"evidence/gone.txt", "docs/absent.md"})

    def test_passes_when_everything_exists(self):
        (self.root / "evidence" / "gone.txt").write_text("g\n")
        (self.root / "docs" / "absent.md").write_text("x\n")
        rc, res = self.run_script()
        self.assertEqual((rc, res["status"], res["missing"]), (0, "PASS", 0), res)

    def test_missing_database_is_blind_not_a_pass(self):
        (self.root / "docs" / "workable_items.db").unlink()
        done = subprocess.run([sys.executable, str(SCRIPT), "--root", str(self.root)], capture_output=True, text=True)
        self.assertEqual(done.returncode, 2, done.stdout + done.stderr)

    def test_paired_mutants_are_killed(self):
        source = SCRIPT.read_text()
        mutants = {
            "never-missing": ('if not os.path.exists(os.path.join(root, p.split("#")[0])):', 'if False:'),
            "doc-links-ignored": ('for m in sorted(found):', 'for m in []:'),
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
            rc, res = self.run_script(mutated)
            real_rc, real_res = self.run_script()
            self.assertEqual(real_rc, 1)
            self.assertNotEqual(res.get("items"), real_res.get("items"), f"mutant {name} survived")


if __name__ == "__main__":
    unittest.main()
