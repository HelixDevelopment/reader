"""Review round 8 (PR #5) regression tests - one more bounded round beyond the budget, by operator decision.
B1  the terminal-bytes class through FILE NAMES: a container plants a file name carrying OSC 52 / CSI / OSC 0 / C1 / bidi / newline; safe_io scan (deny-table check came before the
    charset check) and bindcheck (hostile_tree_error / tree_error) printed it raw. Fix as a class: sanitize.escape_text on every container-derived value in every message
    (safe_io, bindcheck, container_proof) + a message-site lint in check_raw_consumers.py (print of an unescaped f-string value in tools/isolation/*.py, qa/isolation/*.py).
B2  qa/isolation/run_koreader_gui.sh: a mid-line comment dropped five steps of the "A re-open" line; a hermetic step-list test runs the real line with stubs.
I1/I2/m3/m4  honest gate coverage (docs 8.5), mutants M12 (RT_VERBS without logs) and M18 (.bash skipped), sanitize of U+061C / zero-width / BOM, json_after last line, has_token stdout only.
Hermetic: scratch dirs under ~/.cache/claude-tmp, no container, no GUI, no podman."""
import os, pathlib, re, shutil, subprocess, sys, tempfile, textwrap, unittest, importlib.util
ROOT = pathlib.Path(__file__).resolve().parents[3]
ISO = ROOT / "tools/isolation"
QA = ROOT / "qa/isolation"
CT = os.path.expanduser("~/.cache/claude-tmp")

def load(path, name):
    spec = importlib.util.spec_from_file_location(name, str(path)); m = importlib.util.module_from_spec(spec); spec.loader.exec_module(m); return m

R1 = load(ISO / "tests/test_review_round1.py", "rr8_helpers1")
mutated_copy = R1.mutated_copy
CTRL = re.compile("[\x00-\x08\x0b-\x1f\x7f-\x9f​-‏ -‮⁠⁦-⁩؜﻿]")

HOSTILE_NAMES = [".claude\x1b]52;c;cHduZWQ=\x07", ".cursor\x1b[2J\x1b]0;owned\x07", ".git\x9b31m", ".husky‮", ".claude\nFAKE: all ok", ".claude؜​﻿", ".codex\x1b]0;t\x1b\\"]

def newdir(prefix="rr8-"):
    return os.path.realpath(tempfile.mkdtemp(prefix=prefix, dir=CT))

class Base(unittest.TestCase):
    def setUp(self): self.d = newdir(); self.addCleanup(shutil.rmtree, self.d, True)

def run(argv, iso=None, **kw):
    env = {"PATH": os.environ["PATH"], "HOME": os.environ["HOME"]}
    if iso: env["ISO_WORK"] = iso
    return subprocess.run(argv, capture_output=True, env=env, timeout=60, **kw)

def plant(d):
    for n in HOSTILE_NAMES: pathlib.Path(d, n).write_bytes(b"x")

def assert_clean_stderr2(tc, err):
    tc.assertIsNone(CTRL.search(err.decode("utf-8", "replace").replace("\n", "")), "control character on the terminal stream: %r" % err[:300])

def assert_clean_stderr(tc, err):
    txt = err.decode("utf-8", "replace")
    tc.assertIsNone(CTRL.search(txt.replace("\n", "")), "control character on the terminal stream: %r" % txt[:300])
    for ln in txt.splitlines():
        tc.assertTrue(ln.startswith(("safe_io: refused", "run_container: bind refused")), "forged / split line: %r" % ln[:120])

# ---------------------------------------------------------------------------------------------------------------------------------------- B1
class HostileNames(Base):
    def scan(self, root=ISO):
        plant(self.d); return run([sys.executable, "-I", str(root / "safe_io.py"), "scan", self.d, "--allow"])

    def test_scan_deny_table_names_are_escaped(self):
        r = self.scan(); self.assertEqual(r.returncode, 1); assert_clean_stderr(self, r.stderr)
        self.assertIn(b"\\x1b]52;c;cHduZWQ=\\x07", r.stderr)          # visible, not removed: two names stay distinguishable

    def test_scan_each_name_alone(self):
        for n in HOSTILE_NAMES:
            d = newdir(); self.addCleanup(shutil.rmtree, d, True); pathlib.Path(d, n).write_bytes(b"x")
            r = run([sys.executable, "-I", str(ISO / "safe_io.py"), "scan", d, "--allow"])
            self.assertEqual(r.returncode, 1, repr(n)); assert_clean_stderr(self, r.stderr)

    def test_bindcheck_bridge_dir_names_are_escaped(self):
        sock = pathlib.Path(self.d, "sock"); sock.mkdir(); plant(sock)
        r = run([sys.executable, "-I", str(ISO / "bindcheck.py"), "rw", str(sock), "/run/helixsock", "--bridge"], iso=self.d)
        self.assertEqual(r.returncode, 64); assert_clean_stderr(self, r.stderr); self.assertIn(b"git/hook/agent-config entry", r.stderr)    # the hostile-name branch really ran

    def test_bindcheck_each_name_alone(self):
        for n in HOSTILE_NAMES:
            w = newdir(); self.addCleanup(shutil.rmtree, w, True); d = pathlib.Path(w, "s"); d.mkdir(); pathlib.Path(d, n).write_bytes(b"x")
            r = run([sys.executable, "-I", str(ISO / "bindcheck.py"), "rw", str(d), "/run/helixsock", "--bridge"], iso=w)
            if r.returncode == 0: continue                  # a name the deny table does not know is not refused by the pre-run check (the post-run scan refuses it by charset: test_scan_each_name_alone)
            self.assertEqual(r.returncode, 64, repr(n)); self.assertIn(b"agent-config entry", r.stderr, repr(n)); assert_clean_stderr(self, r.stderr)

    def test_refused_copy_name_is_escaped(self):
        pathlib.Path(self.d, "sub").mkdir()
        r = run([sys.executable, "-I", str(ISO / "safe_io.py"), "copy", self.d, "sub/ev\x1b]52;c;x\x07il", self.d, "out.txt"])
        self.assertEqual(r.returncode, 1); assert_clean_stderr2(self, r.stderr); self.assertTrue(r.stderr.startswith(b"safe_io: refused: "))

    def test_mutants_scan_and_bindcheck(self):
        for sub, old, new in (("safe_io.py", 'refused {escape_text(b)}', 'refused {b}'),
                              ("safe_io.py", 'refused: {escape_text(r)}', 'refused: {r}'),
                              ("bindcheck.py", 'print(escape_text(f"run_container: bind refused ({mode} {src}:{dst}): {err}"), file=sys.stderr)', 'print(f"run_container: bind refused ({mode} {src}:{dst}): {err}", file=sys.stderr)')):
            t, f = mutated_copy(sub, old, new); self.addCleanup(shutil.rmtree, t, True)
            d = newdir(); self.addCleanup(shutil.rmtree, d, True); plant(d)
            if sub == "safe_io.py" and "{r}" in new:
                pathlib.Path(d, "sub").mkdir(); r = run([sys.executable, "-I", str(f), "copy", d, "sub/ev\x1b]52;c;x\x07il", d, "out.txt"]); leaked = CTRL.search(r.stderr.decode().replace("\n", ""))
            elif sub == "safe_io.py":
                r = run([sys.executable, "-I", str(f), "scan", d, "--allow"]); leaked = CTRL.search(r.stderr.decode().replace("\n", ""))
            else:
                sk = pathlib.Path(d, "sock"); sk.mkdir(); plant(sk); r = run([sys.executable, "-I", str(f), "rw", str(sk), "/run/helixsock", "--bridge"], iso=d); leaked = CTRL.search(r.stderr.decode().replace("\n", ""))
            self.assertIsNotNone(leaked, "mutant survived: " + new[:40])

class MessageSiteLint(Base):
    def gate(self, body, rel="tools/isolation/x.py", mod=None):
        p = pathlib.Path(self.d, rel); p.parent.mkdir(parents=True, exist_ok=True); p.write_text(textwrap.dedent(body))
        return run([sys.executable, "-I", str(mod or ISO / "check_raw_consumers.py"), self.d])

    BAD = 'import sys\nprint(f"refused {name}", file=sys.stderr)\n'
    GOOD = 'import sys\nfrom sanitize import escape_text\nprint(f"refused {escape_text(name)}", file=sys.stderr)\nprint(escape_text(f"x {name}"), file=sys.stderr)\nprint(f"n={len(a)}")\n'

    def test_flags_unescaped_message(self):
        r = self.gate(self.BAD); self.assertEqual(r.returncode, 1); self.assertIn(b"escape_text", r.stderr)

    def test_flags_in_qa_dir_too(self):
        self.assertEqual(self.gate(self.BAD, "qa/isolation/y.py").returncode, 1)

    def test_escaped_messages_pass(self):
        r = self.gate(self.GOOD); self.assertEqual(r.returncode, 0, r.stderr)

    def test_out_of_scope_dir_not_flagged(self):
        self.assertEqual(self.gate(self.BAD, "scripts/z.py").returncode, 0)

    def test_mutant_check_disabled(self):
        t, f = mutated_copy("check_raw_consumers.py", "bad += check_messages(rel, text)", "pass"); self.addCleanup(shutil.rmtree, t, True)
        self.assertEqual(self.gate(self.BAD, mod=f).returncode, 0)

    def test_tree_has_no_unescaped_message(self):
        r = run([sys.executable, "-I", str(ISO / "check_raw_consumers.py")]); self.assertEqual(r.returncode, 0, r.stderr)

class GateVerbs(Base):
    def gate(self, rel, body, mod=None):
        p = pathlib.Path(self.d, rel); p.parent.mkdir(parents=True, exist_ok=True); p.write_text(textwrap.dedent(body))
        return run([sys.executable, "-I", str(mod or ISO / "check_raw_consumers.py"), self.d])

    def test_python_each_verb(self):
        for verb in ("logs", "attach", "cp", "exec", "run"):
            r = self.gate("tools/v.py", 'import subprocess\nsubprocess.run(["podman", "%s", "x"], capture_output=True)\n' % verb)
            self.assertEqual(r.returncode, 1, verb)

    def test_python_logs_inherited_stdout_and_global_options(self):
        self.assertEqual(self.gate("tools/v.py", 'import subprocess\nsubprocess.run(["podman", "logs", "x"])\n').returncode, 1)
        self.assertEqual(self.gate("tools/v.py", 'import subprocess\nsubprocess.run(["/usr/bin/podman", "--log-level", "error", "container", "logs", "x"])\n').returncode, 1)

    def test_python_run_detached_allowed(self):
        self.assertEqual(self.gate("tools/v.py", 'import subprocess\nsubprocess.run(["podman", "run", "-d", "img"], capture_output=True)\n').returncode, 0)

    def test_shell_logs_attach(self):
        for line in ("podman logs x", "out=$(podman logs x); echo $out", "docker container logs x", "podman attach x", "foo | podman --remote logs x"):
            self.assertEqual(self.gate("tools/s.sh", "#!/bin/bash\n%s\n" % line).returncode, 1, line)
        self.assertEqual(self.gate("tools/s.sh", "#!/bin/bash\necho 'see the podman logs docs'  # raw-consumer-exempt: text\npodman logs x | python3 sanitize.py  # raw-consumer-exempt: sanitized\n").returncode, 0)

    def test_bash_extension_scanned(self):
        self.assertEqual(self.gate("tools/s.bash", '#!/bin/bash\nx=$(tools/isolation/run_container.sh -n a -- true)\n').returncode, 1)

    def test_mutant_logs_removed_from_verbs(self):      # M12
        t, f = mutated_copy("check_raw_consumers.py", '"logs", "cp", "attach", "start"}', '"cp", "attach", "start"}'); self.addCleanup(shutil.rmtree, t, True)
        r = self.gate("tools/v.py", 'import subprocess\nsubprocess.run(["podman", "logs", "x"])\n', mod=f)
        self.assertEqual(r.returncode, 0)

    def test_mutant_bash_skipped(self):                  # M18
        t, f = mutated_copy("check_raw_consumers.py", 'EXTS = (".sh", ".bash", ".py")', 'EXTS = (".sh", ".py")'); self.addCleanup(shutil.rmtree, t, True)
        r = self.gate("tools/s.bash", '#!/bin/bash\nx=$(tools/isolation/run_container.sh -n a -- true)\n', mod=f)
        self.assertEqual(r.returncode, 0)

# ---------------------------------------------------------------------------------------------------------------------------------------- B2
class KoReaderStepList(Base):
    def lines(self, path=None):
        src = pathlib.Path(path or QA / "run_koreader_gui.sh").read_text().splitlines()
        i = next(k for k, l in enumerate(src) if l.startswith('say "A re-open book'))
        j = next(k for k, l in enumerate(src) if l.startswith('say "PSE'))
        return "\n".join(src[i:j])

    def exec_steps(self, text):
        sio = pathlib.Path(self.d, "sio"); sio.write_text("#!/bin/bash\necho b.epub\n"); sio.chmod(0o755)
        script = 'say() { :; }\nsink() { echo "SINK" >> "$LOG"; }\nD() { echo "D $*" >> "$LOG"; }\nSIO=%s\nPR=/x\nPRC=0\n%s\n' % (sio, text)
        log = pathlib.Path(self.d, "log"); log.write_text("")
        subprocess.run(["bash", "-c", script], env={"PATH": os.environ["PATH"], "LOG": str(log)}, capture_output=True, timeout=30)
        return log.read_text().splitlines()

    def test_a_reopen_runs_all_steps(self):
        got = self.exec_steps(self.lines())
        self.assertEqual(got, ["D start A /prof/A/Books/b.epub", "D shot A05_reopened_pulled", "D docpercent A", "D close", "SINK"])

    def test_no_code_line_hides_steps_behind_a_midline_comment(self):
        for f in sorted(QA.glob("*.sh")):
            for n, l in enumerate(f.read_text().splitlines(), 1):
                m = re.search(r"[;&|]\s*#[^\"']*;\s*[A-Za-z_]", l)        # `cmd; # note; more` : everything after # is a comment
                self.assertIsNone(m, f"{f.name}:{n}: a mid-line comment swallows code after it")

    def test_mutant_midline_comment_restored(self):
        src = (QA / "run_koreader_gui.sh").read_text()
        old = 'D start A "/prof/A/Books/$BN"; D shot A05_reopened_pulled;'
        self.assertIn(old, src)
        mut = pathlib.Path(self.d, "m.sh"); mut.write_text(src.replace(old, 'D start A "/prof/A/Books/$BN"   # name read; D shot A05_reopened_pulled;', 1))
        self.assertEqual(self.exec_steps(self.lines(mut)), ["D start A /prof/A/Books/b.epub"])

# ---------------------------------------------------------------------------------------------------------------------------------------- m3 m4
SNIP = "import sys; sys.path.insert(0, sys.argv[1]); %s"

class SanitizeAndParsers(Base):
    def py(self, tree, code):
        return subprocess.run([sys.executable, "-I", "-c", SNIP % code, str(tree)], capture_output=True, timeout=30).stdout.decode()

    def test_sanitize_strips_arabic_mark_zero_width_bom(self):
        self.assertEqual(self.py(ISO, "from sanitize import sanitize_line as s; print(s('a\\u061cb\\u200bc\\u200dd\\ufeffe\\u2060f'))").strip(), "abcdef")

    def test_mutant_sanitize_keeps_them(self):
        t, f = mutated_copy("sanitize.py", "\\u061c\\ufeff]\")", "]\")"); self.addCleanup(shutil.rmtree, t, True)
        self.assertNotEqual(self.py(pathlib.Path(t, "tools/isolation"), "from sanitize import sanitize_line as s; print(s('a\\u061cb\\ufeffc'))").strip(), "abc")

    def test_json_after_takes_last_line(self):
        code = "from rawrun import Raw; r = Raw(0, b'ADMIN:{\"a\":1}\\nADMIN:{\"a\":2}\\n', b'', b'', None); print(r.json_after('ADMIN:'))"
        self.assertEqual(self.py(ISO, code).strip(), "{'a': 2}")

    def test_has_token_stdout_only(self):
        code = "from rawrun import Raw; r = Raw(0, b'x', b'TOKEN', b'', None); print(r.has_token('TOKEN'), Raw(0, b'TOKEN', b'', b'', None).has_token('TOKEN'))"
        self.assertEqual(self.py(ISO, code).strip(), "False True")

    def test_mutants_parsers(self):
        for old, new, code in (("lines[-1][0 if keep_prefix", "lines[0][0 if keep_prefix", "from rawrun import Raw; r = Raw(0, b'ADMIN:{\"a\":1}\\nADMIN:{\"a\":2}\\n', b'', b'', None); print(r.json_after('ADMIN:'))"),
                               ("in self.raw_stdout\n", "in self.raw_stdout + self.raw_stderr\n", "from rawrun import Raw; print(Raw(0, b'x', b'TOKEN', b'', None).has_token('TOKEN'))")):
            t, f = mutated_copy("rawrun.py", old, new); self.addCleanup(shutil.rmtree, t, True)
            good = self.py(ISO, code); bad = self.py(pathlib.Path(t, "tools/isolation"), code)
            self.assertNotEqual(good, bad, "mutant survived: " + new[:30])

if __name__ == "__main__":
    unittest.main()
