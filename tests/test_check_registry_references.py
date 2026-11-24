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

    def stage(self):
        # Tracking is what the checker measures: stage everything the working tree holds that git does not ignore.
        subprocess.run(["git", "init", "-q"], cwd=self.root, check=True)
        subprocess.run(["git", "add", "-A"], cwd=self.root, check=True)

    def run_script(self, script=SCRIPT):
        self.stage()
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
        self.stage()
        done = subprocess.run([sys.executable, str(SCRIPT), "--root", str(self.root)], capture_output=True, text=True)
        self.assertEqual(done.returncode, 2, done.stdout + done.stderr)

    def test_missing_database_is_blind_not_a_pass(self):
        (self.root / "docs" / "workable_items.db").unlink()
        done = subprocess.run([sys.executable, str(SCRIPT), "--root", str(self.root)], capture_output=True, text=True)
        self.assertEqual(done.returncode, 2, done.stdout + done.stderr)

    def scenario_untracked(self, script=SCRIPT):
        # An ignored file and an empty directory exist in the working tree but not in a clean clone.
        (self.root / ".gitignore").write_text("*.out\n")
        (self.root / "evidence" / "sealed.out").write_text("x\n")
        (self.root / "tests" / "emptydir").mkdir(parents=True, exist_ok=True)
        (self.root / "docs" / "helixtrack-sync.md").write_text("see `evidence/sealed.out` and tests/emptydir/ here\n")
        rc, res = self.run_script(script)
        return {m["path"] for m in res["items"] if m["kind"] == "doc-link"}

    def test_untracked_ignored_and_empty_dirs_do_not_count_as_present(self):
        self.assertIn("evidence/sealed.out", self.scenario_untracked())

    def test_not_a_git_checkout_is_blind(self):
        subprocess.run(["git", "init", "-q"], cwd=self.root, check=True)
        subprocess.run(["git", "add", "-A"], cwd=self.root, check=True)
        shutil.rmtree(self.root / ".git")
        done = subprocess.run([sys.executable, str(SCRIPT), "--root", str(self.root)], capture_output=True, text=True, env={"PATH": "/usr/bin:/bin", "GIT_CEILING_DIRECTORIES": str(self.root.parent)})
        self.assertEqual(done.returncode, 2, done.stdout + done.stderr)

    def scenario_owndir(self, script=SCRIPT):
        # A path cited relative to the document's OWN directory (docs/upstream-issues/x.md citing `sib.json`) resolves there.
        sealed = self.root / "evidence" / "camp" / "ht"
        (sealed / "live").mkdir(parents=True, exist_ok=True)
        (sealed / "live" / "k.json").write_text("{}\n")
        (self.root / "docs" / "upstream-issues").mkdir(exist_ok=True)
        (self.root / "docs" / "upstream-issues" / "sib.json").write_text("{}\n")
        (self.root / "docs" / "upstream-issues" / "x.md").write_text("Evidence `evidence/camp/ht/` and `sib.json` and `live/k.json`.\n")
        rc, res = self.run_script(script)
        return sorted(m["path"] for m in res["items"] if m["kind"] == "doc-relative-link")

    def test_paths_relative_to_the_documents_own_directory_resolve(self):
        self.assertEqual(self.scenario_owndir(), [])

    def scenario_default_docs(self, script=SCRIPT):
        # Every default doc glob must be scanned: one missing link per document class.
        docs = {"docs/helixtrack-sync.md": "docs/gone-sync.md", "docs/gap-register-x.md": "docs/gone-gap.md", "docs/constitution-update-plan-x.md": "docs/gone-plan.md",
                "docs/submodule-pins-x.md": "docs/gone-pins.md", "docs/workable-items.md": "docs/gone-wi.md", "README.md": "docs/gone-readme.md",
                "deploy/README.md": "docs/gone-deploy.md", "docs/Fixed.md": "docs/gone-fixed.md", "docs/upstream-issues/u.md": "docs/gone-up.md"}
        (self.root / "deploy").mkdir(exist_ok=True)
        for name, target in docs.items():
            (self.root / name).write_text(f"see {target}\n")
        rc, res = self.run_script(script)
        return docs, {m["path"] for m in res["items"]}

    def test_default_doc_set_is_pinned(self):
        docs, paths = self.scenario_default_docs()
        for target in docs.values():
            self.assertIn(target, paths)

    def scenario_relative(self, script=SCRIPT):
        sealed = self.root / "evidence" / "camp" / "ht"
        (sealed / "live").mkdir(parents=True, exist_ok=True)
        (sealed / "live" / "here.json").write_text("{}\n")
        (self.root / "docs" / "helixtrack-sync.md").write_text(
            "Evidence `evidence/camp/ht/`: `live/here.json`, `live/gone.json`, `runs/r1/` and `SHA256SUMS.top`.\n")
        own = self.root / "docs" / "tracking" / "pending_ownership.json"
        data = json.loads(own.read_text())
        data["entries"].append({"prefix": "evidence/camp/ht/runs", "task": "HRD-029", "pr": None})
        own.write_text(json.dumps(data))
        rc, res = self.run_script(script)
        return {m["path"]: m["class"] for m in res["items"] if m["kind"] == "doc-relative-link"}

    def test_paths_relative_to_a_cited_directory_are_resolved_and_classified(self):
        rel = self.scenario_relative()
        self.assertEqual(rel, {"evidence/camp/ht/live/gone.json": "NOWHERE", "evidence/camp/ht/runs/r1/": "OWNED-BY-UNDELIVERED-WORK",
                               "evidence/camp/ht/SHA256SUMS.top": "NOWHERE"})

    def scenario_malformed(self, script=SCRIPT):
        own = self.root / "docs" / "tracking" / "pending_ownership.json"
        out = []
        for bad in ("{not json", json.dumps({"entries": "x"}), json.dumps({"entries": [{"prefix": "", "task": "T"}]}), json.dumps({})):
            own.write_text(bad)
            self.stage()
            done = subprocess.run([sys.executable, str(script), "--root", str(self.root)], capture_output=True, text=True)
            out.append((done.returncode, "Traceback" in done.stderr))
        return out

    def test_malformed_ownership_map_is_blind_exit_2(self):
        self.assertEqual(self.scenario_malformed(), [(2, False)] * 4)

    def test_scenario_mutants_are_killed(self):
        source = SCRIPT.read_text()
        mutants = {
            "default-docs-drop-helixtrack-sync": ('"docs/gap-register*.md", "docs/helixtrack-sync.md",', '"docs/gap-register*.md",', self.scenario_default_docs),
            "relative-links-ignored": ('if cited_dirs:', 'if False:', self.scenario_relative),
            "untracked-counts-as-present": ('return q in tracked or q in tracked_dirs', 'return os.path.exists(os.path.join(root, q))', self.scenario_untracked),
            "doc-own-dir-resolution-dropped": (' and not exists(os.path.join(os.path.dirname(rel), tok))', '', self.scenario_owndir),
            "malformed-map-traceback": ('print(f"BLIND: ownership map unreadable or malformed: {exc}", file=sys.stderr); return 2', 'raise', self.scenario_malformed),
        }
        for name, (old, new, scenario) in mutants.items():
            self.assertIn(old, source, name)
            mutated = self.root / f"mutant_{name}.py"
            mutated.write_text(source.replace(old, new, 1))
            self.assertNotEqual(scenario(mutated), scenario(), f"mutant {name} survived")

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
                self.stage()
                done = subprocess.run([sys.executable, str(mutated), "--root", str(self.root)], capture_output=True, text=True)
                self.assertEqual(done.returncode, 0, name)  # the mutant misbehaves ...
                self.assertNotEqual(subprocess.run([sys.executable, str(SCRIPT), "--root", str(self.root)], capture_output=True).returncode, 0)  # ... the real one does not
                continue
            if name == "needle-removed":
                (self.root / "evidence" / "a.txt").unlink()
                self.stage()
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
