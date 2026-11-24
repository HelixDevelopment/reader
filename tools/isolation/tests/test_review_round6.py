"""Review round 6 (PR #5) regression tests - one bounded round beyond the budget, by operator decision.
R6-B1  the terminal-bytes class, regressed in round 5: rawrun.Raw.stderr held RAW container stderr and test_network_isolation passed it as a failure message.
       Fix as a class: Raw keeps the byte-exact streams ONLY in raw_stdout / raw_stderr (bytes, UNSAFE TO PRINT); .stdout / .stderr / .wrapper_stderr / str / repr / format are
       sanitized-only; parsers return DATA (json_after, has_token); a bare `podman run|exec` capture goes through rawrun.run_direct. Sweep of every consumer (table in
       docs/test-isolation.md 8.5): canary_probe, test_network_isolation (failure messages + the access.log assertIn), test_runtime_checks (direct podman), test_reaper (not a reader),
       qa/isolation/*.sh. Gate check_raw_consumers.py also flags raw_stdout / raw_stderr outside rawrun.py and hand-captured podman output. Each has a paired mutant here.
R6-I1  run_container.sh in ATTACHED mode (no -o, no -d) refuses with exit 67 when stdout is not a terminal: the structural fix for the evasions of the grep gate. The gate stays as
       defence in depth; tests below run the reviewer's evasion shapes for real against the wrapper (refused) and through the gate (flagged), with mutants.
Minors real-wrapper -o path into the publisher (parent-symlink RAWDIR, re-serialised JSON), NaN/Infinity and deep nesting in the publisher, the runner's exit status when the container never starts.
Hermetic: scratch dirs under ~/.cache/claude-tmp, PATH stubs (NEVER the real podman), no container, no GUI."""
import importlib.util, json, os, pathlib, re, shutil, stat, subprocess, sys, tempfile, textwrap, unittest
ROOT = pathlib.Path(__file__).resolve().parents[3]
ISO = ROOT / "tools/isolation"
QA = ROOT / "qa/isolation"
CT = os.path.expanduser("~/.cache/claude-tmp")

def load(path, name):
    spec = importlib.util.spec_from_file_location(name, str(path)); m = importlib.util.module_from_spec(spec); spec.loader.exec_module(m); return m

R1 = load(ISO / "tests/test_review_round1.py", "rr6_helpers1")
R4 = load(ISO / "tests/test_review_round4.py", "rr6_helpers4")
R5 = load(ISO / "tests/test_review_round5.py", "rr6_helpers5")
mutated_copy, Stub, run_rc = R1.mutated_copy, R1.Stub, R1.run_rc
HOSTILE, clean = R4.HOSTILE, R4.clean
sys.path.insert(0, str(ISO)); import rawrun       # noqa: E402

HOSTILE_ALL = HOSTILE + "bidi \u202e\u2066 ls \u2028 c1 \u009b[2J nul-free end\n".encode()

def newdir(prefix="rr6-"):
    return os.path.realpath(tempfile.mkdtemp(prefix=prefix, dir=CT))

class Base(unittest.TestCase):
    def setUp(self): self.d = newdir(); self.addCleanup(shutil.rmtree, self.d, True)

# --------------------------------------------------------------------------------------------------------------------------------------- R6-B1
HOSTILE_WRAPPER = textwrap.dedent('''\
    #!/bin/bash
    # stub wrapper following the -o contract: writes hostile bytes to the raw files and hostile text to ITS OWN stderr too
    R=; while [ $# -gt 0 ]; do case $1 in -o) R=$2; shift;; esac; shift; done
    printf '%b' "$HQ_OUT" > "$R/stdout.raw"; printf '%b' "$HQ_ERR" > "$R/stderr.raw"; printf '%b' "$HQ_WERR" >&2; exit "${HQ_RC:-0}"
''')
ESCAPED = lambda b: b.decode("latin-1").replace("\\", "\\\\").replace("\n", "\\n").replace("\x1b", "\\033").replace("\x07", "\\007").replace("\x00", "\\0000").replace("%", "%%")

def hostile_env():
    return dict(os.environ, HQ_OUT=ESCAPED(HOSTILE_ALL) + 'ADMIN:{"k":1}\\n', HQ_ERR=ESCAPED(HOSTILE_ALL) + "ERRMARK\\n", HQ_WERR=ESCAPED(HOSTILE_ALL))

def surfaces(r):
    """every str-ish surface of a Raw a failure message / print / log could use"""
    out = {"stdout": r.stdout, "stderr": r.stderr, "wrapper_stderr": r.wrapper_stderr, "str": str(r), "repr": repr(r), "format": f"{r}", "msg": "%s %r" % (r, r), "rawdir": str(r.rawdir)}
    for n in dir(r):
        if n.startswith("__") or n in ("raw_stdout", "raw_stderr"): continue
        try: v = getattr(r, n)
        except Exception: continue
        if isinstance(v, str): out["attr:" + n] = v
        if isinstance(v, bytes) and n.startswith("_"): out["private:" + n] = None      # private byte buffers exist, they are not a surface; nothing to check
    return {k: v for k, v in out.items() if v is not None}

class RawSurfaces(Base):
    def mkwrapper(self):
        w = os.path.join(self.d, "w.sh"); open(w, "w").write(HOSTILE_WRAPPER); os.chmod(w, 0o755); return w
    def go(self, mod=rawrun):
        return mod.run_raw(["-n", "hrdiso-x"], ["true"], env=hostile_env(), wrapper=self.mkwrapper(), tmp=self.d)
    def test_no_str_surface_of_raw_can_carry_an_escape_or_control(self):
        r = self.go()
        self.assertIsInstance(r.raw_stdout, bytes); self.assertIn(b"\x1b]52;c;", r.raw_stdout)        # non-vacuous: the container really did emit escapes and the bytes are kept
        self.assertIn(b"ERRMARK", r.raw_stderr); self.assertNotIn(b"ERRMARK", r.raw_stdout)          # unmerged
        s = surfaces(r); self.assertGreaterEqual(len(s), 8)
        for k, v in s.items(): self.assertTrue(clean(v.encode()), k)
        self.assertIn("ERRMARK", r.stderr); self.assertIn("[container:stderr] ", r.stderr); self.assertTrue(r.has_token("ADMIN:")); self.assertEqual(r.json_after("ADMIN:"), {"k": 1})
    def test_a_failing_assertion_built_from_a_raw_surface_carries_no_escape(self):
        r = self.go()
        for k in ("stdout", "stderr", "wrapper_stderr"):
            try: self.assertTrue(False, getattr(r, k))
            except AssertionError as e: self.assertTrue(clean(str(e).encode()), k)
        try: self.assertEqual(r, 1)
        except AssertionError as e: self.assertTrue(clean(str(e).encode()), "assertEqual(r, 1)")
    def mutated_rawrun(self, old, new):
        t = newdir("rr6m-"); self.addCleanup(shutil.rmtree, t, True)
        for n in ("rawrun.py", "sanitize.py"): shutil.copy(ISO / n, t)
        s = open(os.path.join(t, "rawrun.py")).read(); self.assertIn(old, s); open(os.path.join(t, "rawrun.py"), "w").write(s.replace(old, new, 1))
        return load(os.path.join(t, "rawrun.py"), "rr6_rawrun_mut_" + str(abs(hash(old + new))))
    def test_mutant_raw_text_back_in_stderr_leaks(self):
        m = self.mutated_rawrun('def stderr(self): return _clean(self.raw_stderr, "[container:stderr] ")', 'def stderr(self): return self.raw_stderr.decode("utf-8", "replace")')
        r = self.go(m); self.assertFalse(clean(r.stderr.encode()))
    def test_mutant_unsanitized_str_leaks(self):
        m = self.mutated_rawrun('def __str__(self): return f"Raw(rc={self.returncode}, stdout={len(self.raw_stdout)}B, stderr={len(self.raw_stderr)}B)"',
                                'def __str__(self): return self.raw_stdout.decode("utf-8", "replace")')
        r = self.go(m); self.assertFalse(clean(str(r).encode()))
    def test_mutant_wrapper_stderr_unsanitized_leaks(self):
        m = self.mutated_rawrun('def wrapper_stderr(self): return _clean(self._wrapper_stderr, "[wrapper] ")', 'def wrapper_stderr(self): return self._wrapper_stderr.decode("utf-8", "replace")')
        r = self.go(m); self.assertFalse(clean(r.wrapper_stderr.encode()))
    def test_json_after_is_strict(self):
        def raw(line): return rawrun.Raw(0, line.encode(), b"", b"", None)
        self.assertEqual(raw('x\nADMIN:{"a":[1,2]}\n').json_after("ADMIN:"), {"a": [1, 2]})
        for bad in ('ADMIN:{"a":NaN}', 'ADMIN:{"a":Infinity}', 'ADMIN:{"a":-Infinity}', 'ADMIN:{"a":"\\u001b]52;c;x"}', 'ADMIN:{"a":"\\u202e"}', 'ADMIN:{"\\u0007":1}', 'ADMIN:{', "ADMIN:" + "[" * 100000):
            self.assertIsNone(raw(bad + "\n").json_after("ADMIN:"), bad[:40])
        self.assertIsNone(raw("nothing\n").json_after("ADMIN:"))
    def test_mutant_json_after_accepts_controls_and_nan(self):
        m = self.mutated_rawrun("return _check_strings(json.loads(", "return (json.loads(")
        line = 'ADMIN:{"a":"\\u001b]52"}\n'.encode(); self.assertIsNotNone(m.Raw(0, line, b"", b"", None).json_after("ADMIN:"))
        m = self.mutated_rawrun("parse_constant=bad", "parse_constant=None")
        self.assertIsNotNone(m.Raw(0, b'ADMIN:{"a":NaN}\n', b"", b"", None).json_after("ADMIN:"))

class ConsumerSweep(Base):
    """drive the REAL consumers with hostile container output and check what a failing assertion would print"""
    def podman_stub(self, body):
        d = os.path.join(self.d, "bin"); os.makedirs(d, exist_ok=True); p = os.path.join(d, "podman"); open(p, "w").write("#!/bin/bash\n" + body); os.chmod(p, 0o755); return d
    def test_runtime_checks_failure_output_has_no_escape(self):
        hostile = ESCAPED(HOSTILE_ALL)
        d = self.podman_stub("printf '%%b' '%s' >&2; printf '%%b' '%s'; exit 9\n" % (hostile.replace("'", ""), hostile.replace("'", "")))
        env = dict(os.environ, PATH=d + ":" + os.environ["PATH"], PYTHONDONTWRITEBYTECODE="1")
        r = subprocess.run([sys.executable, "-I", "-W", "ignore", "-m", "unittest", "discover", "-s", str(ISO / "tests"), "-p", "test_runtime_checks.py"], capture_output=True, env=env, timeout=120)
        out = r.stdout + r.stderr
        self.assertNotEqual(r.returncode, 0); self.assertIn(b"[container:stderr]", out); self.assertTrue(clean(out), "a failing test_runtime_checks printed a raw container control character")
    def tree_with_runtime_checks(self, mutate):
        t = newdir("rr6t-"); self.addCleanup(shutil.rmtree, t, True)
        shutil.copytree(ISO, os.path.join(t, "tools/isolation"), ignore=shutil.ignore_patterns("__pycache__", "tests", "guard"))
        os.makedirs(os.path.join(t, "tools/isolation/tests")); shutil.copy(ISO / "tests/test_runtime_checks.py", os.path.join(t, "tools/isolation/tests"))
        os.makedirs(os.path.join(t, "qa/isolation/inside")); mutate(os.path.join(t, "tools/isolation/rawrun.py")); return t
    def test_mutant_direct_podman_text_in_the_runtime_check_would_leak(self):
        hostile = ESCAPED(HOSTILE_ALL)
        d = self.podman_stub("printf '%%b' '%s' >&2; exit 9\n" % hostile.replace("'", ""))
        def mut(p):
            s = open(p).read(); a = 'return Raw(r.returncode, r.stdout, r.stderr, b"", None)'; self.assertIn(a, s)
            s = s.replace(a, 'class T: pass\n    t = T(); t.returncode = r.returncode; t.stdout = r.stdout.decode("utf-8", "replace"); t.stderr = r.stderr.decode("utf-8", "replace"); return t'); open(p, "w").write(s)
        t = self.tree_with_runtime_checks(mut)
        env = dict(os.environ, PATH=d + ":" + os.environ["PATH"], PYTHONDONTWRITEBYTECODE="1")
        r = subprocess.run([sys.executable, "-I", "-W", "ignore", "-m", "unittest", "discover", "-s", os.path.join(t, "tools/isolation/tests"), "-p", "test_runtime_checks.py"], capture_output=True, env=env, timeout=120)
        self.assertFalse(clean(r.stdout + r.stderr))
    def test_canary_probe_and_network_failure_messages_are_clean(self):
        cp = load(ISO / "tests/canary_probe.py", "rr6_canary"); w = os.path.join(self.d, "w.sh"); open(w, "w").write(HOSTILE_WRAPPER); os.chmod(w, 0o755)
        real = cp.run_raw
        os.environ.update({k: v for k, v in hostile_env().items() if k.startswith("HQ_")}); self.addCleanup(lambda: [os.environ.pop(k, None) for k in ("HQ_OUT", "HQ_ERR", "HQ_WERR")])
        cp.run_raw = lambda wa, cmd, **kw: real(wa, cmd, wrapper=w, tmp=self.d, **kw)
        res, r = cp.probe(["-n", "hrdiso-x"], ["10.0.2.2:1"])
        self.assertEqual(res, {})                                                  # the hostile stdout has no JSON line starting with "{": nothing parsed, nothing invented
        for k, v in surfaces(r).items(): self.assertTrue(clean(v.encode()), k)
        try: self.assertTrue(res, r.stderr)                                        # the exact shape test_network_isolation.py uses
        except AssertionError as e: self.assertTrue(clean(str(e).encode()))
    def test_network_isolation_has_no_raw_access_and_no_assertIn_on_container_logs(self):
        src = (ISO / "tests/test_network_isolation.py").read_text()
        self.assertNotIn("raw_std", src); self.assertNotRegex(src, r"assertIn\([^\n]*read_text\(")
        self.assertIn("json_after", src); self.assertIn("has_token", src)
    def test_every_podman_capture_in_the_tree_goes_through_run_direct(self):
        g = load(ISO / "check_raw_consumers.py", "rr6_gate_tree"); self.assertEqual(g.scan(str(ROOT)), [])

class GateRawRules(Base):
    G = ISO / "check_raw_consumers.py"
    def gate(self, files, script=None):
        t = os.path.join(self.d, "t"); shutil.rmtree(t, ignore_errors=True)
        for rel, body in files.items():
            p = os.path.join(t, rel); os.makedirs(os.path.dirname(p), exist_ok=True); open(p, "w").write(body)
        return subprocess.run([sys.executable, "-I", str(script or self.G), t], capture_output=True, text=True, timeout=60)
    def test_raw_attributes_outside_rawrun_are_flagged(self):
        for c in ("def f(r):\n    assert r.raw_stderr, r.raw_stderr\n", "def f(r):\n    return getattr(r, 'raw_stdout')\n", "def f(r):\n    return r.raw_stdout.decode()\n"):
            r = self.gate({"tools/x.py": c}); self.assertEqual(r.returncode, 1, (c, r.stdout, r.stderr)); self.assertIn("raw_std", r.stderr)
        r = self.gate({"tools/x.py": "def f(r):\n    return r.raw_stdout  # raw-consumer-exempt: this is the one parse function\n"}); self.assertEqual(r.returncode, 0, r.stderr)
        r = self.gate({"tools/x.py": "def f(r):\n    return r.stdout, r.stderr, r.json_after('X:'), r.has_token('Y')\n"}); self.assertEqual(r.returncode, 0, r.stderr)
    def test_hand_captured_podman_output_is_flagged(self):
        bad = 'import subprocess\nr = subprocess.run(["podman", "run", "--rm", "img"], capture_output=True, text=True)\n'
        r = self.gate({"tools/x.py": bad}); self.assertEqual(r.returncode, 1, r.stderr); self.assertIn("run_direct", r.stderr)
        for ok in ('import subprocess\nsubprocess.run(["podman", "rm", "-f", "n"], capture_output=True)\n', 'import subprocess\nsubprocess.run(["podman", "run", "-d", "--rm", "img"], capture_output=True, text=True)\n',
                   'from rawrun import run_direct\nr = run_direct(["podman", "run", "img"])\n'):
            r = self.gate({"tools/x.py": ok}); self.assertEqual(r.returncode, 0, (ok, r.stderr))
    def test_mutant_gate_without_the_raw_attribute_rule_is_blind(self):
        t, f = mutated_copy("check_raw_consumers.py", "if not in_lib and ((isinstance(n, ast.Attribute)", "if False and ((isinstance(n, ast.Attribute)"); self.addCleanup(shutil.rmtree, t, True)
        r = self.gate({"tools/x.py": "def f(r):\n    return r.raw_stderr\n"}, script=f); self.assertEqual(r.returncode, 0)
    def test_mutant_gate_without_the_podman_rule_is_blind(self):
        t, f = mutated_copy("check_raw_consumers.py", 'if a == "podman" and b in RT_VERBS', 'if False and b in RT_VERBS'); self.addCleanup(shutil.rmtree, t, True)
        r = self.gate({"tools/x.py": 'import subprocess\nsubprocess.run(["podman", "exec", "n", "x"], capture_output=True, text=True)\n'}, script=f); self.assertEqual(r.returncode, 0)

# --------------------------------------------------------------------------------------------------------------------------------------- R6-I1
SHELL_SHAPES = {
    "redirect_and": '"$R/tools/isolation/run_container.sh" -n hrdiso-x -- cmd &> "$OUTF"',
    "process_subst": 'cat <("$R/tools/isolation/run_container.sh" -n hrdiso-x -- cmd) > "$OUTF"',
    "variable_indirection": 'W="$R/tools/isolation/run_container.sh"; X=$("$W" -n hrdiso-x -- cmd); printf %s "$X" > "$OUTF"',
    "exec_redirect": 'exec > "$OUTF"; "$R/tools/isolation/run_container.sh" -n hrdiso-x -- cmd',
    "pipe_grep_o": '"$R/tools/isolation/run_container.sh" -n hrdiso-x -- cmd | grep -o PASS > "$OUTF"',
    "dash_o_inside_container_command": '"$R/tools/isolation/run_container.sh" -n hrdiso-x -- sh -c "grep -o x /etc/hostname" > "$OUTF"',
    "command_substitution": 'out=$("$R/tools/isolation/run_container.sh" -n hrdiso-x -- cmd); printf %s "$out" > "$OUTF"',
}
PY_SHAPES = {
    "shell_true": 'import subprocess, os\nr = subprocess.run(os.environ["R"] + "/tools/isolation/run_container.sh -n hrdiso-x -- cmd", shell=True, capture_output=True)\n',
    "os_popen": 'import os\nt = os.popen(os.environ["R"] + "/tools/isolation/run_container.sh -n hrdiso-x -- cmd").read()\n',
    "list_without_o": 'import subprocess, os\nr = subprocess.run([os.environ["R"] + "/tools/isolation/run_container.sh", "-n", "hrdiso-x", "--", "cmd"], capture_output=True)\n',
}

class WrapperTerminalRefusal(Base):
    def go(self, args, tty, script_dir=None, extra_env=None):
        with Stub() as st:
            iso = os.path.join(self.d, "iso"); os.makedirs(os.path.join(iso, "run1"), exist_ok=True); st.env["ISO_WORK"] = iso
            pathlib.Path(st.d, "podman").write_text(R1.STUB.replace("  run) exit 0;;", "  run) printf 'CONTAINER-SAID-HELLO\\n'; exit 0;;"))
            st.env.update(extra_env or {})
            r = run_rc((script_dir or ISO) / "run_container.sh", ["-n", "hrdiso-p6", *args, "--", "true"], st, tty=tty)
            return r, st.log.read_text()
    def test_attached_mode_without_a_terminal_is_refused_before_podman(self):
        r, log = self.go([], tty=False); self.assertEqual(r.returncode, 67, r.stderr); self.assertIn("attached mode needs a terminal on stdout", r.stderr); self.assertIn("-o RAWDIR", r.stderr)
        self.assertEqual(log, ""); self.assertEqual(r.stdout, "")                    # podman was not even asked: nothing ran, nothing to capture
    def test_attached_mode_with_a_terminal_still_runs_and_is_sanitized(self):
        r, log = self.go([], tty=True); self.assertEqual(r.returncode, 0, r.stderr); self.assertIn("run ", log); self.assertIn("[container] CONTAINER-SAID-HELLO", r.stdout)
    def test_raw_mode_and_detached_mode_do_not_need_a_terminal(self):
        rd = os.path.join(self.d, "raw"); os.makedirs(rd, mode=0o700)
        r, log = self.go(["-o", rd], tty=False); self.assertEqual(r.returncode, 0, r.stderr); self.assertIn("run ", log)
        r, log = self.go(["-d"], tty=False); self.assertEqual(r.returncode, 0, r.stderr); self.assertIn("run -d", log)
    def test_validation_refusals_still_come_first_with_their_own_codes(self):
        r, _ = self.go(["-m", "banana"], tty=False); self.assertEqual(r.returncode, 64)       # not 67: the refusals before the terminal check keep their meaning
        r, _ = self.go(["-e", "WAYLAND" + "_DISPLAY=x"], tty=False); self.assertEqual(r.returncode, 64)
    def test_mutant_refusal_removed_lets_a_pipe_capture_the_terminal_stream(self):
        t, f = mutated_copy("run_container.sh", "[ ! -t 1 ]", "false"); self.addCleanup(shutil.rmtree, t, True)
        r, log = self.go([], tty=False, script_dir=f.parent); self.assertEqual(r.returncode, 0); self.assertIn("CONTAINER-SAID-HELLO", r.stdout); self.assertIn("run ", log)
    def test_mutant_refusal_that_also_blocks_raw_mode_is_caught(self):
        t, f = mutated_copy("run_container.sh", '[ -z "$RAWDIR" ] && [ ! -t 1 ]', '[ ! -t 1 ]'); self.addCleanup(shutil.rmtree, t, True)
        rd = os.path.join(self.d, "raw"); os.makedirs(rd, mode=0o700)
        r, log = self.go(["-o", rd], tty=False, script_dir=f.parent); self.assertEqual(r.returncode, 67)       # the mutant refuses -o: test_raw_mode_and_detached_mode... would FAIL on it
    def test_mutant_refusal_that_also_blocks_detached_is_caught(self):
        t, f = mutated_copy("run_container.sh", '[ ${#DETACH[@]} -eq 0 ] && [ -z "$RAWDIR" ] && [ ! -t 1 ]', '[ -z "$RAWDIR" ] && [ ! -t 1 ]'); self.addCleanup(shutil.rmtree, t, True)
        r, log = self.go(["-d"], tty=False, script_dir=f.parent); self.assertEqual(r.returncode, 67)

class EvasionShapes(Base):
    """the reviewer's evasions of the grep gate, run FOR REAL against the wrapper (refused: podman never asked) and through the gate (flagged), then against a wrapper without the refusal"""
    def run_shape(self, name, body, script_dir_root=None):
        with Stub() as st:
            pathlib.Path(st.d, "podman").write_text(R1.STUB.replace("  run) exit 0;;", "  run) printf 'CONTAINER-SAID-HELLO\\n'; exit 0;;"))
            f = os.path.join(self.d, name + (".py" if name in PY_SHAPES else ".sh")); outf = os.path.join(self.d, name + ".out")
            open(f, "w").write(("#!/bin/bash\n" if f.endswith(".sh") else "") + body + "\n"); os.chmod(f, 0o755)
            env = dict(st.env, R=str(script_dir_root or ROOT), OUTF=outf)
            r = subprocess.run(["bash", f] if f.endswith(".sh") else [sys.executable, "-I", f], capture_output=True, text=True, env=env, timeout=60, cwd=self.d)
            return r, st.log.read_text(), (open(outf).read() if os.path.exists(outf) else "")
    def gate(self, rel, body):
        t = os.path.join(self.d, "gt-" + os.path.basename(rel)); shutil.rmtree(t, ignore_errors=True); p = os.path.join(t, rel); os.makedirs(os.path.dirname(p)); open(p, "w").write(body)
        return subprocess.run([sys.executable, "-I", str(ISO / "check_raw_consumers.py"), t], capture_output=True, text=True, timeout=60)
    def test_every_shell_shape_is_refused_by_the_wrapper_and_flagged_by_the_gate(self):
        for name, body in SHELL_SHAPES.items():
            with self.subTest(shape=name):
                r, log, out = self.run_shape(name, body)
                self.assertNotIn("run ", log, (name, r.stderr)); self.assertNotIn("CONTAINER-SAID-HELLO", out + r.stdout)          # the container never ran, nothing was captured
                g = self.gate("qa/x.sh", "#!/bin/bash\n" + body + "\n"); self.assertEqual(g.returncode, 1, (name, g.stdout, g.stderr))
    def test_every_python_shape_is_refused_by_the_wrapper_and_flagged_by_the_gate(self):
        for name, body in PY_SHAPES.items():
            with self.subTest(shape=name):
                r, log, out = self.run_shape(name, body); self.assertNotIn("run ", log, (name, r.stderr)); self.assertNotIn("CONTAINER-SAID-HELLO", out + r.stdout)
                g = self.gate("tools/x.py", body); self.assertEqual(g.returncode, 1, (name, g.stdout, g.stderr))
    def test_mutant_wrapper_without_the_refusal_lets_each_shape_capture(self):
        t, f = mutated_copy("run_container.sh", "[ ! -t 1 ]", "false"); self.addCleanup(shutil.rmtree, t, True)
        for name, body in {**SHELL_SHAPES, **PY_SHAPES}.items():
            with self.subTest(shape=name):
                r, log, out = self.run_shape(name, body, script_dir_root=t); self.assertIn("run ", log, (name, r.stderr))          # on the mutant the shape reaches podman: the capture works
    def test_golden_good_shapes_are_neither_refused_nor_flagged(self):
        rd = os.path.join(self.d, "rawg"); os.makedirs(rd, mode=0o700)
        good = {"raw_mode": '"$R/tools/isolation/run_container.sh" -n hrdiso-x -o %s -- cmd > "$OUTF"' % rd, "detached": '"$R/tools/isolation/run_container.sh" -d -n hrdiso-x -- cmd > "$OUTF"'}
        for name, body in good.items():
            r, log, out = self.run_shape(name, body); self.assertIn("run ", log, (name, r.stderr))
            g = self.gate("qa/x.sh", "#!/bin/bash\n" + body + "\n"); self.assertEqual(g.returncode, 0, (name, g.stderr))
    def test_mutant_gate_that_trusts_a_dash_o_anywhere_misses_the_dash_o_inside_the_command(self):
        t, f = mutated_copy("check_raw_consumers.py", 'if not ({"o", "d"} & flags_of(toks[k + 1:])):', 'if not re.search(r"\\s-o[\\s=]", code):'); self.addCleanup(shutil.rmtree, t, True)
        d = os.path.join(self.d, "gm"); os.makedirs(os.path.join(d, "qa")); open(os.path.join(d, "qa/x.sh"), "w").write("#!/bin/bash\n" + SHELL_SHAPES["dash_o_inside_container_command"] + "\n")
        r = subprocess.run([sys.executable, "-I", str(f), d], capture_output=True, text=True); self.assertEqual(r.returncode, 0)          # the old proxy accepted it; the real gate (above) does not

# --------------------------------------------------------------------------------------------------------------------------------------- minors
class PublisherAndRunner(R5.HelixqaRunner):
    """round-5 runner tree, now with the REAL wrapper and a stub podman: the whole -o path into the publisher"""
    def real_wrapper(self):
        shutil.copy(ISO / "run_container.sh", os.path.join(self.T, "tools/isolation/run_container.sh"))
        os.makedirs(os.path.join(self.T, "qa/helixqa/banks"), exist_ok=True)
        with open(os.path.join(self.T, "tools/isolation/start_server.sh"), "w") as f: f.write('#!/bin/bash\ncase $1 in start) mkdir -p "$3/sock"; echo log > "$3/access.log";; esac\nexit 0\n')
        sd = os.path.join(self.d, "pbin"); os.makedirs(sd, exist_ok=True)
        with open(os.path.join(sd, "podman"), "w") as f: f.write(R1.STUB.replace("  run) exit 0;;", "  run) printf '%b' \"$HQ_OUT\"; printf '%b' \"$HQ_ERR\" >&2; exit \"${HQ_RC:-0}\";;").replace('echo "$*" >> "${STUB_LOG:?}"', 'echo "$*" >> "${STUB_LOG:-/dev/null}"'))
        os.chmod(os.path.join(sd, "podman"), 0o755); self.env = dict(self.env, PATH=sd + ":" + os.environ["PATH"], STUB_DIR=sd)
    def run_real(self, out, err="", rc=0):
        enc = lambda s: s.replace("\\", "\\\\").replace("\n", "\\n").replace("\x1b", "\\033").replace("\x07", "\\007")
        env = dict(self.env, HQ_OUT=enc(out), HQ_ERR=enc(err), HQ_RC=str(rc))
        return subprocess.run(["bash", os.path.join(self.T, "qa/isolation/run_helixqa_container.sh"), self.out], capture_output=True, text=True, env=env, timeout=120, stdin=subprocess.DEVNULL)
    COMPACT = 'bank: PASS\n{"passed":2,"failed":0}\n'.replace('{"passed"', "{\n\"passed\"").replace(',"failed":0}', ',"failed":0\n}')
    def test_real_wrapper_o_path_publishes_through_the_publisher(self):
        self.build(); self.real_wrapper(); r = self.run_real(self.COMPACT, "to-stderr\x1b]2;x\x07\n"); self.assertEqual(r.returncode, 0, r.stdout + r.stderr)
        self.assertEqual(self.published(), ["helixqa_http.exit.txt", "helixqa_http.json", "helixqa_http.stderr.txt", "helixqa_http.txt"])
        want = json.dumps({"passed": 2, "failed": 0}, indent=2, ensure_ascii=True) + "\n"
        self.assertEqual(open(os.path.join(self.out, "helixqa_http.json")).read(), want)            # re-serialised, not the container's text
        for n in self.published(): self.assertNotIn(b"\x1b", open(os.path.join(self.out, n), "rb").read(), n)
        self.assertNotIn("\x1b", r.stdout + r.stderr)
    def test_mutant_publisher_copies_the_raw_json_text(self):
        s = (QA / "helixqa_publish.py").read_text(); old = '"helixqa_http.json": body_json,'; self.assertIn(old, s)
        self.build(publisher=s.replace(old, '"helixqa_http.json": text[i + 1:],', 1)); self.real_wrapper(); r = self.run_real(self.COMPACT); self.assertEqual(r.returncode, 0)
        want = json.dumps({"passed": 2, "failed": 0}, indent=2, ensure_ascii=True) + "\n"
        self.assertNotEqual(open(os.path.join(self.out, "helixqa_http.json")).read(), want)         # the real-path test above would FAIL on this mutant
    def test_raw_dir_behind_a_parent_symlink_is_refused_by_the_real_wrapper(self):
        self.build(); self.real_wrapper(); real = os.path.join(self.d, "elsewhere"); os.makedirs(os.path.join(real, "raw"), mode=0o700); os.chmod(os.path.join(real, "raw"), 0o700)
        link = os.path.join(self.d, "lnk"); os.symlink(real, link)
        with Stub() as st:
            r = run_rc(ISO / "run_container.sh", ["-n", "hrdiso-p6s", "-o", os.path.join(link, "raw"), "--", "true"], st, tty=False)
            self.assertEqual(r.returncode, 64, r.stderr); self.assertIn("-o directory", r.stderr); self.assertEqual(os.listdir(os.path.join(real, "raw")), []); self.assertEqual(st.log.read_text(), "")
    def test_mutant_wrapper_without_the_realpath_check_accepts_the_symlinked_dir(self):
        t, f = mutated_copy("run_container.sh", '&& [ "$(realpath "$RAWDIR")" = "$RAWDIR" ] ||', "||"); self.addCleanup(shutil.rmtree, t, True)
        real = os.path.join(self.d, "elsewhere"); os.makedirs(os.path.join(real, "raw")); os.chmod(os.path.join(real, "raw"), 0o700); link = os.path.join(self.d, "lnk"); os.symlink(real, link)
        with Stub() as st:
            r = run_rc(f, ["-n", "hrdiso-p6s", "-o", os.path.join(link, "raw"), "--", "true"], st, tty=False)
            self.assertEqual(r.returncode, 0, r.stderr); self.assertTrue(os.path.exists(os.path.join(real, "raw", "stdout.raw")))          # the mutant writes through the symlink
    def test_nan_infinity_and_deep_nesting_fail_closed(self):
        for name, out in {"nan": 'bank\n{\n  "x": NaN\n}\n', "inf": 'bank\n{\n  "x": Infinity\n}\n', "neg_inf": 'bank\n{\n  "x": -Infinity\n}\n'}.items():
            self.build(); r = self.run_it(out); self.assertEqual(r.returncode, 1, (name, r.stdout[-200:], r.stderr[-300:])); self.assertEqual(self.published(), [], name); self.assertNotIn("Traceback", r.stderr, name)
            self.assertIn("nothing published", r.stderr, name)
    def publisher_with_recursion(self, src):
        t = newdir("rr6p-"); self.addCleanup(shutil.rmtree, t, True); p = os.path.join(t, "pub.py"); open(p, "w").write(src)
        raw = os.path.join(t, "o.raw"); open(raw, "wb").write(b'bank\n{\n  "a": 1\n}\n'); open(os.path.join(t, "e.raw"), "wb").write(b"")
        out = os.path.join(t, "out"); os.makedirs(out)
        code = ("import importlib.util, json, sys\nsys.path.insert(0, %r)\nspec = importlib.util.spec_from_file_location('pub', %r); m = importlib.util.module_from_spec(spec); spec.loader.exec_module(m)\n"
                "def boom(*a, **k): raise RecursionError('deep')\nm.json.loads = boom\nsys.exit(m.main(['x', %r, %r, %r]))\n") % (str(ISO), p, raw, os.path.join(t, "e.raw"), out)
        r = subprocess.run([sys.executable, "-I", "-c", code], capture_output=True, text=True, timeout=60); return r, os.listdir(out)
    def test_recursion_error_while_parsing_fails_closed(self):
        r, published = self.publisher_with_recursion((QA / "helixqa_publish.py").read_text()); self.assertEqual(r.returncode, 3, r.stderr); self.assertEqual(published, []); self.assertIn("nested too deeply", r.stderr); self.assertNotIn("Traceback", r.stderr)
    def test_mutant_publisher_without_the_recursion_handler_crashes(self):
        s = (QA / "helixqa_publish.py").read_text(); old = '    except RecursionError: fail("the JSON result block is nested too deeply")\n'; self.assertIn(old, s)
        r, published = self.publisher_with_recursion(s.replace(old, "", 1)); self.assertNotEqual(r.returncode, 3); self.assertIn("Traceback", r.stderr)
    def test_mutant_publisher_without_the_nan_refusal_publishes_it(self):
        s = (QA / "helixqa_publish.py").read_text(); old = "parse_constant=non_finite"; self.assertIn(old, s)
        self.build(publisher=s.replace(old, "parse_constant=None", 1).replace(", allow_nan=False", "")); r = self.run_it('bank\n{\n  "x": NaN\n}\n'); self.assertEqual(r.returncode, 0)
    def test_container_never_started_runner_exits_1_with_the_wrapper_status_and_publishes_nothing(self):
        self.build()
        with open(os.path.join(self.T, "tools/isolation/run_container.sh"), "w") as f: f.write("#!/bin/bash\necho 'image missing' >&2; exit 66\n")
        os.chmod(os.path.join(self.T, "tools/isolation/run_container.sh"), 0o755)
        r = self.run_it(self.GOOD); self.assertEqual(r.returncode, 1, r.stderr); self.assertEqual(self.published(), []); self.assertIn("exit 66", r.stderr); self.assertNotIn("helixqa exit=", r.stdout + r.stderr)
    def test_runner_documents_its_exit_status_contract(self):
        s = (QA / "run_helixqa_container.sh").read_text(); self.assertIn("EXIT STATUS", s); self.assertIn("67", s)

if __name__ == "__main__":
    unittest.main()
