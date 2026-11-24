"""Review round 9 (PR #5) regression tests - one more bounded round beyond the budget, by recorded operator decision.
N1  the B1 class ("tree-derived strings printed raw") in the repo GATE itself: guard/assert_host_untouched.py printed scanned file NAMES, file CONTENT lines and symlink targets raw, and
    check_raw_consumers.py printed repo paths raw. Both now pass every such string through sanitize.escape_text. The message-site lint covers tools/isolation/** (guard/ included, its own
    file included) and every composed-string form (f-string, +, %, .format) on print / sys.stderr.write / sys.stdout.write / logging.
I1  escape_text C1 (0x80-0x9f) and bidi (U+202A-E) are killed by direct tests and by deny-table names carrying them.   I3  bindcheck refuses ANY control character in a source.
Hermetic: scratch dirs under ~/.cache/claude-tmp, no container, no GUI, no podman."""
import os, pathlib, re, shutil, subprocess, sys, tempfile, textwrap, unittest, importlib.util
ROOT = pathlib.Path(__file__).resolve().parents[3]
ISO = ROOT / "tools/isolation"
CT = os.path.expanduser("~/.cache/claude-tmp")

def load(path, name):
    spec = importlib.util.spec_from_file_location(name, str(path)); m = importlib.util.module_from_spec(spec); spec.loader.exec_module(m); return m

R1 = load(ISO / "tests/test_review_round1.py", "rr9_helpers1")
R8 = load(ISO / "tests/test_review_round8.py", "rr9_helpers8")
mutated_copy, CTRL, run = R1.mutated_copy, R8.CTRL, R8.run

def newdir(): return os.path.realpath(tempfile.mkdtemp(prefix="rr9-", dir=CT))

class Base(unittest.TestCase):
    def setUp(self): self.d = newdir(); self.addCleanup(shutil.rmtree, self.d, True)

    def mutant(self, sub, old, new):
        """one text mutation in a copy of tools/isolation INCLUDING guard/ (tests/ and caches left out) -> (repo-shaped tmp, mutated file)"""
        t = pathlib.Path(newdir()); self.addCleanup(shutil.rmtree, t, True)
        shutil.copytree(ISO, t / "tools/isolation", ignore=shutil.ignore_patterns("__pycache__", "tests")); (t / "qa/isolation").mkdir(parents=True)
        f = t / "tools/isolation" / sub; src = f.read_text(); assert old in src, f"mutation anchor not found in {sub}: {old!r}"
        f.write_text(src.replace(old, new, 1)); return t, f

def clean(tc, data, what=""):
    txt = data.decode("utf-8", "replace")
    tc.assertIsNone(CTRL.search(txt.replace("\n", "")), "control character reached the terminal stream %s: %r" % (what, txt[:300]))

LINE_OK = re.compile(r"^(NOTICE |assert_host_untouched: |check_raw_consumers: |[^\x00-\x1f]+:\d+: FORBIDDEN\[)")

# ---------------------------------------------------------------------------------------------------------------------------------------- N1
class GuardOutput(Base):
    NAMES = ["a\x1b]52;c;cHduZWQ=\x07.sh", "b\x1b[2J\x1b]0;owned\x07.sh", "c\x9b31m.sh", "d‮.sh", "e\nFORBIDDEN[fake] forged.sh", "f؜​.sh"]
    BODY = "#!/bin/sh\n" + "xdo" + "tool type \x1b]0;owned\x07 \x1b[2J \x9b31m \u202e end\n"

    def plant(self):
        w = newdir(); self.addCleanup(shutil.rmtree, w, True); r = pathlib.Path(w, "r"); (r / "tools/isolation/guard").mkdir(parents=True); (r / "evidence/p").mkdir(parents=True)
        for n in ("waivers.txt", "waivers.history"): (r / "tools/isolation/guard" / n).write_text("# kind|path|rule|justification\n" if n == "waivers.txt" else "")
        for n in self.NAMES: pathlib.Path(r, "evidence/p", n).write_bytes(self.BODY.encode("utf-8"))
        os.symlink("/etc/\x1b]52;c;x\x07tgt", r / "evidence/p/lnk\x1b[2Jx.sh")
        return r

    def guard(self, script=None, root=None):
        return run([sys.executable, "-I", str(script or ISO / "guard/assert_host_untouched.py"), str(root or self.plant())])

    def test_planted_names_content_and_symlink_targets_are_escaped(self):
        r = self.guard(); self.assertEqual(r.returncode, 1)
        clean(self, r.stdout, "stdout"); clean(self, r.stderr, "stderr")
        self.assertIn(b"\\x1b]52;c;x\\x07tgt", r.stdout); self.assertIn(b"a\\x1b]52;c;cHduZWQ=\\x07.sh", r.stdout); self.assertIn(b"\\x1b]0;owned\\x07", r.stdout)
        for ln in r.stdout.decode().splitlines():
            self.assertRegex(ln, LINE_OK, "forged / split line")

    def test_mutants_raw_rel_raw_text(self):
        for old, new in (("escape_text(rel)", "rel"), ("escape_text(text)", "text")):
            t, f = self.mutant("guard/assert_host_untouched.py", old, new)
            r = self.guard(f); self.assertTrue(CTRL.search(r.stdout.decode("utf-8", "replace").replace("\n", "")), "mutant survived: " + old)

class RawConsumersOutput(Base):
    def test_gate_paths_are_escaped(self):
        d = pathlib.Path(self.d, "r"); sub = d / "qa" / "x\x1b]52;c;cHduZWQ=\x07"; sub.mkdir(parents=True)
        (sub / "bad.sh").write_text("#!/bin/bash\nOUT=$(tools/isolation/run_container.sh -n a -- true)\n")
        r = run([sys.executable, "-I", str(ISO / "check_raw_consumers.py"), str(d)])
        self.assertEqual(r.returncode, 1); clean(self, r.stderr); self.assertIn(b"\\x1b]52;c;cHduZWQ=\\x07", r.stderr)

    def test_mutant_gate_prints_raw_path(self):
        t, f = self.mutant("check_raw_consumers.py", "{escape_text(p)}:{int(n)}", "{p}:{int(n)}")
        d = pathlib.Path(self.d, "r"); sub = d / "qa" / "x\x1b]52;c;cHduZWQ=\x07"; sub.mkdir(parents=True)
        (sub / "bad.sh").write_text("#!/bin/bash\nOUT=$(tools/isolation/run_container.sh -n a -- true)\n")
        r = run([sys.executable, "-I", str(f), str(d)]); self.assertTrue(CTRL.search(r.stderr.decode("utf-8", "replace").replace("\n", "")))

class MessageLintForms(Base):
    def gate(self, body, rel="tools/isolation/guard/g.py", mod=None):
        p = pathlib.Path(self.d, rel); p.parent.mkdir(parents=True, exist_ok=True); p.write_text(textwrap.dedent(body))
        return run([sys.executable, "-I", str(mod or ISO / "check_raw_consumers.py"), self.d])

    BADS = ['print("x " + name)', 'print("%s" % name)', 'print("%s %s" % (a, name))', 'print("{}".format(name))', 'import sys\nsys.stderr.write("x" + name)', 'import sys\nsys.stdout.write(f"{name}")',
            'import logging\nlogging.error("x " + name)', 'import logging\nlog = logging.getLogger("a")\nlog.warning(f"{name}")', 'print(f"{str(name)}")', 'print(f"{name.upper()}")', 'print("a", "b" + name)']
    GOODS = ['from sanitize import escape_text\nprint("x " + escape_text(name))', 'from sanitize import escape_text\nprint("%s" % escape_text(name))', 'print("n=%d" % len(a))',
             'import sys\nprint("a", file=sys.stderr)', 'print(f"{len(a)} of {int(b)}")', 'from sanitize import escape_text\nprint("{}".format(escape_text(name)))', 'print(name)']

    def test_every_composed_form_is_flagged_in_guard_dir(self):
        for b in self.BADS:
            r = self.gate(b + "\n"); self.assertEqual(r.returncode, 1, b)

    def test_escaped_and_plain_forms_pass(self):
        for b in self.GOODS:
            r = self.gate(b + "\n"); self.assertEqual(r.returncode, 0, (b, r.stderr))

    def test_tests_dirs_are_out_of_scope(self):
        self.assertEqual(self.gate('print("x " + name)\n', "tools/isolation/tests/test_x.py").returncode, 0)

    def test_tree_including_guard_and_self_is_clean(self):
        r = run([sys.executable, "-I", str(ISO / "check_raw_consumers.py")]); self.assertEqual(r.returncode, 0, r.stderr)

    def test_mutant_any_call_accepted(self):                                   # C10
        t, f = self.mutant("check_raw_consumers.py", 'ast.unparse(v.func) in ("len", "int")', "True")
        self.assertEqual(self.gate('print(f"{str(name)}")\n', mod=f).returncode, 0)

    def test_mutant_lint_scope_excludes_guard(self):
        t, f = self.mutant("check_raw_consumers.py", "tools|qa", "tools/isolation/[a-z_]+\\.py|qa")  # a scope without guard/ must be seen
        self.assertEqual(self.gate('print("x " + name)\n', mod=f).returncode, 0)

    def test_mutant_plus_form_ignored(self):
        t, f = self.mutant("check_raw_consumers.py", "ast.BinOp", "ast.Pass")
        self.assertEqual(self.gate('print("x " + name)\n', mod=f).returncode, 0)

# ---------------------------------------------------------------------------------------------------------------------------------------- I1
class EscapeText(Base):
    def esc(self, tree, ch):
        return subprocess.run([sys.executable, "-I", "-c", "import sys; sys.path.insert(0, sys.argv[1]); from sanitize import escape_text as e; sys.stdout.write(e(bytes.fromhex(sys.argv[2]).decode()))", str(tree), ch.encode().hex()], capture_output=True).stdout.decode()

    ALL = [chr(c) for c in range(0x80, 0xa0)] + [chr(c) for c in range(0x202a, 0x202f)] + ["⁦", "⁩", "‎", "‏", " ", " ", "؜", "​", "﻿", "\x1b", "\x07", "\n", "\x00", "\x7f"]

    def test_every_control_is_escaped_and_printables_untouched(self):
        for ch in self.ALL:
            out = self.esc(ISO, "a" + ch + "b"); self.assertTrue(out.startswith("a\\") and out.endswith("b") and out.isascii(), repr(ch) + " -> " + repr(out))
        self.assertEqual(self.esc(ISO, "ok é 日本.txt"), "ok é 日本.txt")

    def test_mutant_c1_not_escaped(self):                                        # C5
        t, f = self.mutant("sanitize.py", '_ESCAPE = re.compile("[\\x00-\\x1f\\x7f-\\x9f', '_ESCAPE = re.compile("[\\x00-\\x1f\\x7f')
        self.assertEqual(self.esc(t / "tools/isolation", "\x9b"), "\x9b")

    def test_mutant_bidi_not_escaped(self):                                      # C6
        t, f = self.mutant("sanitize.py", '_ESCAPE = re.compile("[\\x00-\\x1f\\x7f-\\x9f\\u200b-\\u200f\\u2028-\\u202e', '_ESCAPE = re.compile("[\\x00-\\x1f\\x7f-\\x9f\\u200b-\\u200f\\u2028-\\u2029')
        self.assertEqual(self.esc(t / "tools/isolation", "‮"), "‮")

    DENY = [".claude\x9b2J", ".claude‮", ".cursor⁦\x85", ".codex\x1b]52;c;x\x07"]

    def test_deny_table_names_with_c1_and_bidi_reach_escape_text(self):
        for n in self.DENY:
            d = newdir(); self.addCleanup(shutil.rmtree, d, True); pathlib.Path(d, n).write_bytes(b"x")
            os.mkdir(pathlib.Path(d, "sub")); os.symlink("tgt\x1b]52;c;x\x07", pathlib.Path(d, "sub", "l.png")); os.mkfifo(pathlib.Path(d, "sub", "f‮.png"))
            r = run([sys.executable, "-I", str(ISO / "safe_io.py"), "scan", d, "--allow"])
            self.assertEqual(r.returncode, 1); clean(self, r.stderr, repr(n)); self.assertIn(b"deny table", r.stderr)

    def test_mutants_reach_through_deny_table(self):
        for old, new, name in (('\\x7f-\\x9f\\u200b', '\\x7f\\u200b', ".claude\x9b2J"), ('\\u2028-\\u202e\\u2060', '\\u2028-\\u2029\\u2060', ".claude\u202e")):
            m = pathlib.Path(self.d, "m%d" % len(name)); shutil.copytree(ISO, m / "tools/isolation", ignore=shutil.ignore_patterns("__pycache__", "tests", "guard"))
            f = m / "tools/isolation/sanitize.py"; s2 = f.read_text(); j = s2.index("_ESCAPE = re.compile"); self.assertIn(old, s2[j:]); f.write_text(s2[:j] + s2[j:].replace(old, new, 1))
            d = newdir(); self.addCleanup(shutil.rmtree, d, True); pathlib.Path(d, name).write_bytes(b"x")
            r = run([sys.executable, "-I", str(m / "tools/isolation/safe_io.py"), "scan", d, "--allow"])
            self.assertTrue(CTRL.search(r.stderr.decode("utf-8", "replace").replace("\n", "")), "mutant survived via deny table: " + repr(name))

# ---------------------------------------------------------------------------------------------------------------------------------------- I3
class BindcheckControlChars(Base):
    def test_any_control_character_in_a_source_is_refused(self):
        w = newdir(); self.addCleanup(shutil.rmtree, w, True)
        for ch in ("\x1b[31m", "\x9b", "\t", "‮"):
            sub = pathlib.Path(w, "s" + ch); sub.mkdir()
            r = run([sys.executable, "-I", str(ISO / "bindcheck.py"), "rw", str(sub), "/out"], iso=w)
            self.assertEqual(r.returncode, 64, repr(ch)); clean(self, r.stderr); self.assertEqual(r.stdout, b"")

    def test_plain_source_still_allowed(self):
        w = newdir(); self.addCleanup(shutil.rmtree, w, True); sub = pathlib.Path(w, "out"); sub.mkdir()
        r = run([sys.executable, "-I", str(ISO / "bindcheck.py"), "rw", str(sub), "/out"], iso=w); self.assertEqual(r.returncode, 0, r.stderr)

if __name__ == "__main__":
    unittest.main()
