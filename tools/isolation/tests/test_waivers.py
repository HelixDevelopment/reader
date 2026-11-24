"""N3: exemptions are pinned to (path, content sha256) in tools/isolation/guard/waivers.txt. A trailing deny-list marker comment
exempts nothing unless that exact file content is pinned; changing a pinned file, deleting its waiver, an expired/orphan/unrecorded/
malformed entry all FAIL the lint. Re-pinning goes through pin_waivers.py and leaves a record in waivers.history."""
import datetime, os, pathlib, subprocess, sys, tempfile, unittest
ROOT = pathlib.Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / "tools/isolation/guard"))
import assert_host_untouched as g
PIN = str(ROOT / "tools/isolation/guard/pin_waivers.py")
FUT = (datetime.date.today() + datetime.timedelta(days=90)).isoformat()
LIT = "".join(["xdo", "tool"])
LAB = g.RULES[0][0]                                   # the rule label of that literal (xdo...-family)
EVAL = "".join(["ev", "al \"$CMD\""])
MARK = "".join(["# isolation: ", "deny", "list"])

def tree(files):
    d = tempfile.mkdtemp(prefix="wv-")
    for n, b in files.items():
        p = pathlib.Path(d, n); p.parent.mkdir(parents=True, exist_ok=True); p.write_text(b)
    pathlib.Path(d, "tools/isolation/guard").mkdir(parents=True, exist_ok=True)
    return d

def pin(d, *a):
    return subprocess.run([sys.executable, "-I", PIN, "--root", d, *a], capture_output=True, text=True)

def labels(d):
    return sorted({h[2] for h in g.scan(d)})

class Waivers(unittest.TestCase):
    def test_denylist_marker_alone_exempts_nothing(self):
        d = tree({"tools/isolation/x.sh": f"case $1 in {LIT}) exit 1;; esac  {MARK}\n"})
        self.assertIn("denylist-marker-unpinned", labels(d)); self.assertIn(LAB, labels(d))

    def test_marker_outside_tools_isolation_never_exempts_even_if_pinned(self):
        d = tree({"scripts/x.sh": f"echo {LIT} {MARK}\n"})
        self.assertEqual(pin(d, "add", "denylist", "scripts/x.sh", "*", "reason", FUT).returncode, 0)
        self.assertIn(LAB, labels(d))

    def test_pinned_denylist_file_is_exempt_then_modification_fails(self):
        d = tree({"tools/isolation/x.sh": f"case $1 in {LIT}) exit 1;; esac  {MARK}\n"})
        r = pin(d, "add", "denylist", "tools/isolation/x.sh", "*", "deny-list literal, runtime guard", FUT); self.assertEqual(r.returncode, 0, r.stderr)
        self.assertEqual(g.scan(d), [])
        p = pathlib.Path(d, "tools/isolation/x.sh"); p.write_text(p.read_text() + f"{LIT} key a  {MARK}\n")
        self.assertIn("waiver-pin-mismatch", labels(d))                     # an extra marker line in a pinned file is NOT exempt
        self.assertIn("denylist-marker-unpinned", labels(d))
        r = pin(d, "repin", "tools/isolation/x.sh", "reviewed the added line"); self.assertEqual(r.returncode, 0)
        self.assertIn("RE-PINNED", r.stdout)                                # the update is flagged loudly
        self.assertEqual(g.scan(d), [])
        self.assertIn("repin", pathlib.Path(d, "tools/isolation/guard/waivers.history").read_text())

    def test_deleting_a_waiver_fails_the_gate(self):
        d = tree({"x/a.sh": EVAL + "\n"})
        self.assertEqual(pin(d, "add", "waive", "x/a.sh", "obfuscated-exec", "spec tooling", FUT).returncode, 0)
        self.assertEqual(g.scan(d), [])
        os.remove(pathlib.Path(d, "tools/isolation/guard/waivers.txt"))
        self.assertIn("obfuscated-exec", labels(d))

    def test_self_pin_skips_scan_until_content_changes(self):
        d = tree({"tools/isolation/lint.py": f'R = "{LIT}"\n'})
        self.assertIn(LAB, labels(d))
        self.assertEqual(pin(d, "add", "self", "tools/isolation/lint.py", "*", "the lint's own rule literals", FUT).returncode, 0)
        self.assertEqual(g.scan(d), [])
        pathlib.Path(d, "tools/isolation/lint.py").write_text(f'R = "{LIT}"\nimport os; os.system("{LIT} key a")\n')
        self.assertIn("waiver-pin-mismatch", labels(d))

    def test_waive_covers_only_its_rule_and_its_file(self):
        d = tree({"x/a.sh": f"{EVAL}\n{LIT} key a\n", "x/b.sh": EVAL + "\n"})
        pin(d, "add", "waive", "x/a.sh", "obfuscated-exec", "spec tooling", FUT)
        hits = {(h[0], h[2]) for h in g.scan(d)}
        self.assertIn(("x/a.sh", LAB), hits); self.assertIn(("x/b.sh", "obfuscated-exec"), hits)
        self.assertNotIn(("x/a.sh", "obfuscated-exec"), hits)

    def test_expired_orphan_unrecorded_malformed_and_glob_entries_fail(self):
        d = tree({"x/a.sh": "true\n"})
        h = "0" * 64
        wf = pathlib.Path(d, "tools/isolation/guard/waivers.txt")
        past = (datetime.date.today() - datetime.timedelta(days=1)).isoformat()
        far = (datetime.date.today() + datetime.timedelta(days=900)).isoformat()
        wf.write_text("\n".join([f"waive|x/a.sh|{h}|obfuscated-exec|why|{past}", f"waive|x/gone.sh|{h}|obfuscated-exec|why|{FUT}",
                                 f"waive|x/*.sh|{h}|obfuscated-exec|why|{FUT}", f"waive|x/a.sh|{h}|obfuscated-exec||{FUT}",
                                 f"waive|x/a.sh|{h}|obfuscated-exec|why|{far}", "waive|x/a.sh|not-a-hash|r|why|" + FUT]) + "\n")
        got = labels(d)
        for want in ("waiver-expired", "waiver-orphan", "waiver-malformed"):
            self.assertIn(want, got)
        import hashlib
        real = hashlib.sha256(pathlib.Path(d, "x/a.sh").read_bytes()).hexdigest()
        wf.write_text(f"waive|x/a.sh|{real}|obfuscated-exec|hand edited|{FUT}\n")         # correct hash, but never recorded by pin_waivers
        self.assertIn("waiver-unrecorded", labels(d))

    def test_pin_tool_rejects_far_expiry_empty_reason_and_globs(self):
        d = tree({"x/a.sh": "true\n"})
        far = (datetime.date.today() + datetime.timedelta(days=900)).isoformat()
        self.assertNotEqual(pin(d, "add", "waive", "x/a.sh", "r", "why", far).returncode, 0)
        self.assertNotEqual(pin(d, "add", "waive", "x/a.sh", "r", " ", FUT).returncode, 0)
        self.assertNotEqual(pin(d, "add", "waive", "x/*.sh", "r", "why", FUT).returncode, 0)

    def test_repository_waivers_are_all_pinned_and_recorded(self):
        ents, probs = g.load_waivers(str(ROOT))
        active = g.pinned(str(ROOT), ents, probs)
        self.assertEqual(probs, [], probs)
        self_paths = sorted(p for p in active if any(e["kind"] == "self" for e in active[p]))
        self.assertEqual(self_paths, sorted(["tools/isolation/tests/test_guard_matrix.py", "tools/isolation/guard/assert_host_untouched.py",
                                             "tools/isolation/guard/test_assert_host_untouched.py", "tools/isolation/tests/test_runtime_checks.py",
                                             "tools/isolation/tests/test_bindcheck.py", "tools/isolation/tests/test_review_round1.py", "tools/isolation/tests/test_review_round2.py", "tools/isolation/tests/test_review_round3.py", "tools/isolation/tests/test_review_round4.py", "tools/isolation/tests/test_review_round5.py", "tools/isolation/tests/test_review_round6.py"]))     # exactly eleven lint-owned files, each pinned by hash

if __name__ == "__main__":
    unittest.main()
