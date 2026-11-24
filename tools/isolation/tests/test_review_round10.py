"""Review round 10 (PR #5) regression tests - one more bounded round beyond the budget, by recorded operator decision.
N2  INVALID-UTF-8 file names (b"\\x9b" is a Linux name byte; Python holds it as the lone surrogate U+DC9B). escape_text did not escape surrogates, git_files() decoded strictly and silently fell
    back to os.walk, and the print in guard/assert_host_untouched.py put the raw byte on stdout under LC_ALL=C / C.UTF-8 and crashed with UnicodeEncodeError under a UTF-8 locale. Now: escape_text
    makes every surrogate visible, git names are decoded with surrogateescape, a failing git listing is REPORTED on stderr before the walk fallback, and every tool's stdout/stderr is switched to
    backslashreplace (safe_streams) so no print can ever raise or emit raw bytes whatever the locale. Bodies below are NUL-FREE on purpose (a NUL body is skipped as binary and hid this in round 9).
    NUL cannot occur in a Linux file name; the NUL character is only tested directly on escape_text.
F1  the message lint's IfExp branch (print(f"{a if c else b}")) is covered.   F2  waivers.history is replayed: the LATEST row of a path must be an add/repin of the pinned hash.
Hermetic: scratch dirs under ~/.cache/claude-tmp, no container, no GUI, no podman."""
import os, pathlib, shutil, subprocess, sys, unittest, importlib.util, textwrap
ROOT = pathlib.Path(__file__).resolve().parents[3]
ISO = ROOT / "tools/isolation"
sys.path.insert(0, str(ISO))

def load(path, name):
    spec = importlib.util.spec_from_file_location(name, str(path)); m = importlib.util.module_from_spec(spec); spec.loader.exec_module(m); return m

R9 = load(ISO / "tests/test_review_round9.py", "rr10_helpers9")
Base, newdir = R9.Base, R9.newdir
import sanitize

BODY = ("#!/bin/sh\n" + "xdo" + "tool type hi\n").encode()
INV = [b"inv\x9b[2J\x9d52;c;cHduZWQ=\x9c.sh", b"ff\xff.sh", b"lone\xc2.sh", b"sur\xed\xa0\x80.sh"]
ESC_OF = {INV[0]: b"inv\\x9b[2J\\x9d52;c;cHduZWQ=\\x9c.sh"}
LOCALES = ("C", "C.UTF-8", "en_US.UTF-8", "en_GB.utf8", "POSIX")

def runenv(cmd, loc, cwd=None):
    env = {"PATH": os.environ["PATH"], "LC_ALL": loc, "HOME": os.environ.get("HOME", "/"), "PYTHONDONTWRITEBYTECODE": "1"}
    return subprocess.run(cmd, capture_output=True, env=env, cwd=cwd)

def ascii_clean(tc, data, what):
    bad = [b for b in data if (b < 0x20 and b not in (9, 10)) or b >= 0x7f]
    tc.assertEqual(bad, [], "non-printable/non-ASCII byte reached %s: %r" % (what, data[:300]))

class EscapeSurrogates(Base):
    def test_surrogateescape_bytes_become_visible_hex(self):
        for b in (0x80, 0x9b, 0x9d, 0x9c, 0xff, 0xc2):
            self.assertEqual(sanitize.escape_text(chr(0xdc00 + b)), "\\x%02x" % b)
        self.assertEqual(sanitize.escape_text(os.fsdecode(b"a\x9bb")), "a\\x9bb")

    def test_other_lone_surrogates_and_nul(self):
        for cp, want in ((0xd800, "\\ud800"), (0xdbff, "\\udbff"), (0xdc7f, "\\udc7f"), (0xdfff, "\\udfff"), (0xdcff, "\\xff")):
            self.assertEqual(sanitize.escape_text(chr(cp)), want)
        self.assertEqual(sanitize.escape_text("a\0b"), "a\\x00b")
        sanitize.escape_text(chr(0xdc9b) + chr(0xd800)).encode("ascii")           # never raises, always ASCII for these inputs

    def test_mutant_surrogates_not_escaped(self):
        t, f = self.mutant("sanitize.py", "\\ud800-\\udfff", "")
        m = load(f, "san_mut"); self.assertEqual(m.escape_text(chr(0xdc9b)), chr(0xdc9b))        # the mutant leaves the lone surrogate raw ...
        self.assertEqual(sanitize.escape_text(chr(0xdc9b)), "\\x9b")                           # ... the real one does not

    def test_safe_streams_prevents_a_crash_for_an_unescaped_surrogate_in_every_locale(self):
        code = "import sys; sys.path.insert(0, %r); import sanitize; sanitize.safe_streams(); print(chr(0xdc9b)); print(chr(0xdc9b), file=sys.stderr)" % str(ISO)
        for loc in LOCALES:
            r = runenv([sys.executable, "-I", "-c", code], loc); self.assertEqual(r.returncode, 0, (loc, r.stderr)); ascii_clean(self, r.stdout + r.stderr, loc)
        t, f = self.mutant("sanitize.py", 'st.reconfigure(errors="backslashreplace")', 'st.reconfigure(errors="strict")')
        code2 = code.replace(str(ISO), str(f.parent))
        self.assertTrue(any(runenv([sys.executable, "-I", "-c", code2], loc).returncode != 0 for loc in LOCALES), "mutant survived")

class InvalidNames(Base):
    def make(self, git=False):
        r = pathlib.Path(newdir()); self.addCleanup(shutil.rmtree, r, True); root = r / "r"
        (root / "tools/isolation/guard").mkdir(parents=True)
        (root / "tools/isolation/guard/waivers.txt").write_text("# kind|path|sha|rule|why|expires\n"); (root / "tools/isolation/guard/waivers.history").write_text("")
        base = os.fsencode(root)
        os.makedirs(base + b"/evidence/p"); os.makedirs(base + b"/evidence/d\x9b\xff")
        for n in INV: open(base + b"/evidence/p/" + n, "wb").write(BODY)
        open(base + b"/evidence/d\x9b\xff/x.sh", "wb").write(BODY)
        os.symlink(b"/etc/t\x9d52;c;x\x9c", base + b"/evidence/p/lnk\x9b.sh")
        if git: subprocess.run(["git", "init", "-q", str(root)], check=True)
        return root

    def guard(self, root, loc, script=None):
        return runenv([sys.executable, "-I", str(script or ISO / "guard/assert_host_untouched.py"), str(root)], loc)

    def test_guard_output_is_ascii_and_names_the_file_in_every_locale_git_and_walk(self):
        for git in (False, True):
            root = self.make(git)
            for loc in LOCALES:
                r = self.guard(root, loc); tag = f"git={git} LC_ALL={loc}"
                self.assertEqual(r.returncode, 1, tag + repr(r.stderr[-300:]))
                self.assertNotIn(b"Traceback", r.stderr, tag); self.assertNotIn(b"UnicodeEncodeError", r.stderr, tag)
                ascii_clean(self, r.stdout, "stdout " + tag); ascii_clean(self, r.stderr, "stderr " + tag)
                self.assertIn(ESC_OF[INV[0]], r.stdout, tag)                       # the reviewer's repro name, visibly escaped
                self.assertIn(b"d\\x9b\\xff/x.sh", r.stdout, tag)                  # directory name
                self.assertIn(b"lnk\\x9b.sh", r.stdout, tag)                       # symlink name
                self.assertIn(b"t\\x9d52;c;x\\x9c", r.stdout, tag)                 # symlink target

    def test_git_listing_is_used_not_the_silent_walk_fallback(self):
        root = self.make(True)
        r = self.guard(root, "C")
        self.assertNotIn(b"walking the tree", r.stderr + r.stdout)

    def test_failed_git_listing_is_reported_not_silent(self):
        root = self.make(False); (root / ".git").write_text("gitdir: /nonexistent\n")      # looks like a repo, git fails
        r = self.guard(root, "C"); self.assertIn(b"walking the tree", r.stderr)

    def test_mutants(self):
        root = self.make(True)
        for sub, old, new, tag in (("guard/assert_host_untouched.py", '.stdout.decode(errors="surrogateescape").split("\\0")', '.stdout.decode().split("\\0")', "strict git decode"),
                                   ("guard/assert_host_untouched.py", "escape_text(rel)", "rel", "raw rel"),
                                   ("sanitize.py", "\\ud800-\\udfff", "", "surrogates unescaped")):
            t, f = self.mutant(sub, old, new)
            gs = f if sub.startswith("guard") else t / "tools/isolation/guard/assert_host_untouched.py"
            res = []
            for git in (True,):
                for loc in LOCALES:
                    r = self.guard(root, loc, gs)
                    ok = (r.returncode == 1 and b"Traceback" not in r.stderr and not [b for b in r.stdout if b >= 0x7f or (b < 0x20 and b not in (9, 10))]
                          and ESC_OF[INV[0]] in r.stdout and b"walking the tree" not in r.stderr)
                    res.append(ok)
            self.assertFalse(all(res), "mutant survived: " + tag)

class OtherToolsStreams(Base):
    def test_safe_io_and_gate_stderr_with_invalid_names(self):
        r = pathlib.Path(newdir()); self.addCleanup(shutil.rmtree, r, True)
        base = os.fsencode(r); os.makedirs(base + b"/out"); open(base + b"/out/in\x9bv.txt", "wb").write(b"x")
        for loc in LOCALES:
            p = runenv([sys.executable, "-I", str(ISO / "safe_io.py"), "scan", str(r / "out")], loc)
            ascii_clean(self, p.stdout + p.stderr, "safe_io " + loc); self.assertNotIn(b"Traceback", p.stderr)
            q = runenv([sys.executable, "-I", str(ISO / "check_raw_consumers.py"), str(r)], loc)
            ascii_clean(self, q.stdout + q.stderr, "check_raw_consumers " + loc); self.assertNotIn(b"Traceback", q.stderr)

class LintIfExp(Base):
    def gate(self, body, mod=None):
        d = pathlib.Path(newdir()); self.addCleanup(shutil.rmtree, d, True)
        (d / "tools/isolation").mkdir(parents=True); (d / "tools/isolation/x.py").write_text(textwrap.dedent(body))
        return subprocess.run([sys.executable, "-I", str(mod or ISO / "check_raw_consumers.py"), str(d)], capture_output=True, text=True)

    def test_ifexp_with_raw_value_is_a_finding_and_safe_one_is_not(self):
        self.assertNotEqual(self.gate('import sys\ndef f(name, x):\n    print(f"{name if x else \'-\'}")\n').returncode, 0)
        self.assertNotEqual(self.gate('import sys\ndef f(name, x):\n    print(f"{\'-\' if x else name}")\n').returncode, 0)
        self.assertEqual(self.gate('from sanitize import escape_text\ndef f(name, x):\n    print(f"{escape_text(name) if x else \'-\'}")\n').returncode, 0)

    def test_mutant_ifexp_safe_M9(self):
        t, f = self.mutant("check_raw_consumers.py", "return _val_unsafe(v.body) or _val_unsafe(v.orelse)", "return False")
        self.assertEqual(self.gate('import sys\ndef f(name, x):\n    print(f"{name if x else \'-\'}")\n', f).returncode, 0)   # the mutant is blind ...
        self.assertNotEqual(self.gate('import sys\ndef f(name, x):\n    print(f"{name if x else \'-\'}")\n').returncode, 0)  # ... the real gate is not

class WaiverHistoryStrict(Base):
    def setUp(self):
        super().setUp()
        T = load(ROOT / "tools/isolation/tests/test_waivers.py", "rr10_tw")
        self.T, self.g = T, T.g
        self.d = T.tree({"x/a.sh": T.EVAL + "\n"}); self.addCleanup(shutil.rmtree, self.d, True)
        self.assertEqual(T.pin(self.d, "add", "waive", "x/a.sh", "obfuscated-exec", "spec tooling", T.FUT).returncode, 0)
        self.hf = pathlib.Path(self.d, "tools/isolation/guard/waivers.history")

    def labels(self): return self.T.labels(self.d)

    def test_baseline_clean(self): self.assertEqual(self.g.scan(self.d), [])

    def test_history_ending_in_remove_voids_the_pin(self):
        rows = self.hf.read_text().splitlines(); sha = rows[-1].split("|")[4]
        self.hf.write_text("\n".join(rows + [f"2026-11-24T00:00:00Z|remove|x/a.sh|{sha}|-|gone"]) + "\n")
        self.assertIn("waiver-unrecorded", self.labels())

    def test_deleting_only_the_latest_add_row_stays_red(self):
        rows = self.hf.read_text().splitlines(); sha = rows[-1].split("|")[4]
        self.hf.write_text("\n".join([f"2026-11-01T00:00:00Z|add|x/a.sh|-|{sha}|older", f"2026-11-02T00:00:00Z|remove|x/a.sh|{sha}|-|gone"]) + "\n")
        self.assertIn("waiver-unrecorded", self.labels())

    def test_remove_then_readd_is_fine(self):
        self.assertEqual(self.T.pin(self.d, "remove", "x/a.sh", "r").returncode, 0)
        self.assertEqual(self.T.pin(self.d, "add", "waive", "x/a.sh", "obfuscated-exec", "spec tooling", self.T.FUT).returncode, 0)
        self.assertEqual(self.g.scan(self.d), [])

    def test_mutant_any_matching_row_accepts(self):
        t, f = self.mutant("guard/assert_host_untouched.py", "                last = f\n", "                last = f if f[1] in ('add', 'repin') else last\n")
        m = load(f, "g_mut")
        rows = self.hf.read_text().splitlines(); sha = rows[-1].split("|")[4]
        self.hf.write_text("\n".join(rows + [f"2026-11-24T00:00:00Z|remove|x/a.sh|{sha}|-|gone"]) + "\n")
        self.assertEqual(m.scan(self.d), [])                                  # the mutant is fooled ...
        self.assertIn("waiver-unrecorded", self.labels())                     # ... the real replay is not

if __name__ == "__main__":
    unittest.main()
