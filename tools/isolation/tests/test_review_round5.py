"""Review round 5 (PR #5) regression tests.
R5-B1  the sanitized terminal stream of run_container.sh (merged stdout+stderr, "[container] " prefix, 4000-char line cap) must never be PARSED. Consumers of container output
       use the wrapper's RAW-capture mode (-o): stdout and stderr unmerged and uncapped in host-only 0700 files, only a sanitized copy on the terminal. Consumer table
       (every script / test that reads what a container printed; grep of the whole tree for run_container.sh):
         qa/isolation/run_helixqa_container.sh      PARSES (splits text/JSON)   -> -o + helixqa_publish.py, fails closed        tests: HelixqaRunner*, PublisherFailsClosed
         tools/isolation/tests/canary_probe.py      PARSES (json line)          -> rawrun.run_raw                                tests: CanaryProbeRaw
         tools/isolation/tests/test_network_isolation.py  PARSES (GOT:/ADMIN:) -> run_raw (3 call sites)                         tests: CanaryProbeRaw (shared helper) + gate
         tools/isolation/tests/test_runtime_checks.py     PARSES (env names)    -> run_raw                                       tests: gate (RawConsumersGate)
         qa/isolation/run_web_live.sh / run_desktop_live.sh   do not parse (terminal copy only; evidence from a -w dir via scan --allow)
         qa/isolation/run_koreader_gui.sh           detached (-d), the output is read through podman exec + sanitize.py          tests: round 4
         tools/isolation/tests/test_reaper.py       stdout to DEVNULL                                                           gate exempt by construction
         tools/isolation/tests/test_bindcheck.py, test_review_round*.py   test the wrapper itself (refusals / the sanitized stream on purpose); skipped by name in the gate
       The gate tools/isolation/check_raw_consumers.py enforces the table mechanically (any capture of run_container.sh output without -o FAILS); each consumer has a
       behavioural test and a paired mutant that restores the old behaviour.
R5-I1  lint: quoted global-option values with spaces, Python list / JS array forms with global options, docker compose / docker-compose / podman-compose (matrix entries r5_*).
R5-I2  seal_evidence / scan --allow accept the HelixQA exit record (now helixqa_http.exit.txt).
Minors bidi controls are stripped (mutant S4), a lone ESC never survives, the line cap applies to the terminal copy only (raw files are uncapped).
Hermetic: scratch dirs under ~/.cache/claude-tmp, PATH stubs (NEVER the real podman), no container, no GUI."""
import importlib.util, json, os, pathlib, re, shutil, stat, subprocess, sys, tempfile, textwrap, unittest
ROOT = pathlib.Path(__file__).resolve().parents[3]
ISO = ROOT / "tools/isolation"
QA = ROOT / "qa/isolation"
CT = os.path.expanduser("~/.cache/claude-tmp")

def load(path, name):
    spec = importlib.util.spec_from_file_location(name, str(path)); m = importlib.util.module_from_spec(spec); spec.loader.exec_module(m); return m

R1 = load(ISO / "tests/test_review_round1.py", "rr5_helpers1")
R4 = load(ISO / "tests/test_review_round4.py", "rr5_helpers4")
mutated_copy, Stub, run_rc = R1.mutated_copy, R1.Stub, R1.run_rc
HOSTILE, clean = R4.HOSTILE, R4.clean
LONG = "J" * 9000          # longer than the sanitizer's 4000-character terminal line cap

def newdir(prefix="rr5-"):
    return os.path.realpath(tempfile.mkdtemp(prefix=prefix, dir=CT))

class Base(unittest.TestCase):
    def setUp(self): self.d = newdir(); self.addCleanup(shutil.rmtree, self.d, True)
    def raw(self, mode=0o700):
        p = os.path.join(self.d, "raw"); os.makedirs(p, exist_ok=True); os.chmod(p, mode); return p

# ------------------------------------------------------------------------------------------------------------------------------------------ the wrapper's raw mode
class RawMode(Base):
    STDOUT = 'ADMIN:{"a":1}\\n' + '{"127.0.0.1:22":"closed"}\\n'
    def go(self, extra=None, script_dir=None, rc=0, rawdir=None, wargs=()):
        with Stub() as st:
            iso = os.path.join(self.d, "iso"); os.makedirs(os.path.join(iso, "run1"), exist_ok=True); st.env["ISO_WORK"] = iso
            body = R1.STUB.replace("  run) exit 0;;", "  run) printf '%s\\n' \"$P_OUT\"; printf '%s\\n' \"$LONGLINE\"; printf '%s\\n' \"$P_ERR\" >&2; exit \"${STUB_RC:-0}\";;")
            pathlib.Path(st.d, "podman").write_text(body)
            st.env.update(P_OUT=HOSTILE.decode("latin-1").replace("\n", " ").replace("\x00", "") + ' {"k":"v"}', P_ERR="to-stderr \x1b]52;c;x\x07 ERRMARK", LONGLINE=LONG, STUB_RC=str(rc))
            rd = rawdir or self.raw()
            r = run_rc((script_dir or ISO) / "run_container.sh", ["-n", "hrdiso-p5", "-o", rd, *wargs, *(extra or []), "--", "true"], st)
            return r, rd
    def test_raw_streams_are_unmerged_exact_uncapped_private_and_the_terminal_copy_is_sanitized(self):
        r, rd = self.go(rc=7); self.assertEqual(r.returncode, 7, r.stderr)
        so = open(os.path.join(rd, "stdout.raw"), "rb").read(); se = open(os.path.join(rd, "stderr.raw"), "rb").read()
        self.assertIn(b"\x1b]52;c;cHduZWQ=", so)                       # byte exact: the raw file is NOT sanitized
        self.assertIn(LONG.encode(), so)                                # NOT capped at the terminal copy's 4000 characters
        self.assertIn(b"ERRMARK", se); self.assertNotIn(b"ERRMARK", so)  # UNMERGED
        for n in ("stdout.raw", "stderr.raw"): self.assertEqual(stat.S_IMODE(os.stat(os.path.join(rd, n)).st_mode), 0o600)
        self.assertEqual(r.stdout, "")                                  # nothing of the container on stdout: a consumer cannot parse the wrong stream by accident
        self.assertNotIn("\x1b", r.stderr); self.assertTrue(clean(r.stderr.encode())); self.assertIn("[container] ", r.stderr); self.assertIn("[container:stderr] ", r.stderr)
        self.assertNotIn(LONG, r.stderr)
    def test_refusals(self):
        for bad in ("relative/dir", "/tmp/a b", "/nonexistent-dir-xyz"):
            r, _ = self.go(rawdir=bad); self.assertEqual(r.returncode, 64, (bad, r.stderr))
        loose = os.path.join(self.d, "loose"); os.makedirs(loose); os.chmod(loose, 0o755)
        r, _ = self.go(rawdir=loose); self.assertEqual(r.returncode, 64); self.assertIn("0700", r.stderr)
        real = self.raw(); link = os.path.join(self.d, "lnk"); os.symlink(real, link)
        r, _ = self.go(rawdir=link); self.assertEqual(r.returncode, 64)
        r, _ = self.go(extra=["-d"]); self.assertEqual(r.returncode, 64); self.assertIn("-d", r.stderr)
    def test_raw_dir_inside_a_container_writable_source_is_refused(self):
        iso = os.path.join(self.d, "iso"); inner = os.path.join(iso, "run1", "rawin"); os.makedirs(inner); os.chmod(inner, 0o700)
        r, _ = self.go(rawdir=inner, wargs=["-w", f"{iso}/run1:/out"]); self.assertEqual(r.returncode, 64, r.stderr); self.assertIn("lies inside", r.stderr)
    def mut(self, old, new):
        t, f = mutated_copy("run_container.sh", old, new); self.addCleanup(shutil.rmtree, t, True); return f.parent
    def test_mutant_raw_bytes_to_the_terminal(self):
        r, _ = self.go(script_dir=self.mut('python3 -I "$HERE/sanitize.py" < "$RAWDIR/stdout.raw" >&2', 'cat "$RAWDIR/stdout.raw" >&2'))
        self.assertIn("\x1b", r.stderr)
    def test_mutant_streams_merged(self):
        r, rd = self.go(script_dir=self.mut('2> "$RAWDIR/stderr.raw"', '2>&1'))
        self.assertIn(b"ERRMARK", open(os.path.join(rd, "stdout.raw"), "rb").read())
    def test_mutant_lost_exit_status(self):
        r, _ = self.go(script_dir=self.mut('exit "$RC"', "exit 0"), rc=7); self.assertEqual(r.returncode, 0)
    def test_mutant_no_mode_check(self):
        loose = os.path.join(self.d, "loose"); os.makedirs(loose); os.chmod(loose, 0o755)
        r, _ = self.go(script_dir=self.mut('[ "$(stat -c \'%a %u\' "$RAWDIR")" = "700 $(id -u)" ]', "true"), rawdir=loose); self.assertEqual(r.returncode, 0)

# ------------------------------------------------------------------------------------------------------------------------------------------ Python consumers
class CanaryProbeRaw(Base):
    """canary_probe.probe (shared by the network-isolation gate) goes through rawrun.run_raw: the JSON line is found although the terminal stream would be prefixed/merged."""
    def stubbed(self, rawrun_dir=None):
        st = Stub().__enter__(); self.addCleanup(st.__exit__)
        body = R1.STUB.replace("  run) exit 0;;", "  run) printf '%s\\n' '{\"127.0.0.1:22\":\"closed\",\"127.0.0.1:38191\":\"open\"}'; printf 'ADMIN:{\"read_ok\":200}\\n'; printf 'warn\\x1b]2;t\\x07\\n' >&2; exit 0;;")
        pathlib.Path(st.d, "podman").write_text(body); iso = os.path.join(self.d, "iso"); os.makedirs(iso); st.env["ISO_WORK"] = iso
        return st
    def probe(self, st, rawrun_dir=None):
        src = (ISO / "tests/canary_probe.py").read_text()
        sys_path = [str(rawrun_dir or ISO)]
        env = dict(os.environ, **{k: v for k, v in st.env.items()})
        code = ("import sys, json; sys.path.insert(0, %r); import importlib.util as u; s = u.spec_from_file_location('cp', %r); m = u.module_from_spec(s); s.loader.exec_module(m);"
                " res, r = m.probe(['-n', 'hrdiso-p5c'], ['127.0.0.1:22']); print(json.dumps({'res': res, 'err': r.stderr}))") % (sys_path[0], str(ISO / "tests/canary_probe.py"))
        return subprocess.run([sys.executable, "-I", "-c", code], capture_output=True, text=True, env=env, timeout=60)
    def test_probe_finds_the_json_through_raw_mode(self):
        st = self.stubbed(); r = self.probe(st); self.assertEqual(r.returncode, 0, r.stderr)
        out = json.loads(r.stdout); self.assertEqual(out["res"].get("127.0.0.1:38191"), "open", out); self.assertIn("warn", out["err"])
    def test_mutant_attached_parsing_restores_the_false_red(self):
        t, f = mutated_copy("rawrun.py", '"-o", d, ', ""); self.addCleanup(shutil.rmtree, t, True)        # the copy's rawrun no longer asks for raw mode
        tdir = os.path.join(t, "tools/isolation/tests"); os.makedirs(tdir); shutil.copy(ISO / "tests/canary_probe.py", tdir)    # canary_probe of the copy: ROOT = t
        st = self.stubbed(); code = ("import sys, json, importlib.util as u; s = u.spec_from_file_location('cp', %r); m = u.module_from_spec(s); s.loader.exec_module(m);"
                " res, r = m.probe(['-n', 'hrdiso-p5c'], ['127.0.0.1:22']); print(json.dumps({'res': res}))") % os.path.join(tdir, "canary_probe.py")
        r = subprocess.run([sys.executable, "-I", "-c", code], capture_output=True, text=True, env=dict(os.environ, **st.env), timeout=60)
        self.assertEqual(r.returncode, 0, r.stderr); self.assertEqual(json.loads(r.stdout)["res"], {})          # nothing to parse without -o: the gate would fail closed
    def test_consumers_use_run_raw(self):
        for n in ("test_network_isolation.py", "test_runtime_checks.py", "canary_probe.py"):
            s = (ISO / "tests" / n).read_text(); self.assertIn("run_raw", s, n); self.assertNotRegex(s, r'run_container\.sh"\)\s*,\s*"-n"', n)

# ------------------------------------------------------------------------------------------------------------------------------------------ HelixQA runner
class HelixqaRunner(Base):
    """the real run_helixqa_container.sh + helixqa_publish.py in a repo-shaped temp tree; run_container.sh is a stub that follows the -o contract (writes the raw files)."""
    GOOD = 'bank http-health: PASS\nsummary: 2 passed\n{\n  "passed": 2,\n  "failed": 0\n}\n'
    def build(self, runner=None, publisher=None):
        T = os.path.join(self.d, "repo"); self.T = T; shutil.rmtree(T, ignore_errors=True); shutil.rmtree(os.path.join(self.d, "iso"), ignore_errors=True)
        shutil.copytree(ISO, os.path.join(T, "tools/isolation"), ignore=shutil.ignore_patterns("__pycache__", "tests", "guard"))
        os.makedirs(os.path.join(T, "qa/isolation")); os.makedirs(os.path.join(T, "qa/helixqa/fixtures")); os.makedirs(os.path.join(T, "evidence"))
        shutil.copy(QA / "run_helixqa_container.sh", os.path.join(T, "qa/isolation")); shutil.copy(QA / "helixqa_publish.py", os.path.join(T, "qa/isolation"))
        if runner is not None: open(os.path.join(T, "qa/isolation/run_helixqa_container.sh"), "w").write(runner)
        if publisher is not None: open(os.path.join(T, "qa/isolation/helixqa_publish.py"), "w").write(publisher)
        open(os.path.join(T, "qa/helixqa/fixtures/make_fixtures.py"), "w").write("import os, sys\nos.makedirs(sys.argv[1], exist_ok=True)\n")
        for n, body in (("start_server.sh", '#!/bin/bash\ncase $1 in start) mkdir -p "$3"; echo log > "$3/access.log";; esac\nexit 0\n'),
                        ("run_container.sh", textwrap.dedent('''\
                            #!/bin/bash
                            R=; while [ $# -gt 0 ]; do case $1 in -o) R=$2; shift;; esac; shift; done
                            [ -n "$R" ] || exit 64
                            printf '%b' "$HQ_OUT" > "$R/stdout.raw"; printf '%b' "$HQ_ERR" > "$R/stderr.raw"; exit "${HQ_RC:-0}"
                        '''))):
            p = os.path.join(T, "tools/isolation", n); open(p, "w").write(body); os.chmod(p, 0o755)
        self.work = os.path.join(self.d, "iso"); os.makedirs(self.work); self.out = os.path.join(T, "evidence", "hq1")
        hb = os.path.join(self.d, "helixqa-bin"); open(hb, "w").write("x"); self.env = dict(os.environ, HELIX_SERVER_BIN="/bin/true", HELIXQA_BIN=hb, ISO_WORK=self.work)
    def run_it(self, out, err="", rc=0):
        env = dict(self.env, HQ_OUT=out.replace("\\", "\\\\").replace("\n", "\\n").replace("\x1b", "\\033").replace("\x07", "\\007"), HQ_ERR=err.replace("\\", "\\\\").replace("\n", "\\n").replace("\x1b", "\\033").replace("\x07", "\\007"), HQ_RC=str(rc))
        return subprocess.run(["bash", os.path.join(self.T, "qa/isolation/run_helixqa_container.sh"), self.out], capture_output=True, text=True, env=env, timeout=120)
    def published(self): return sorted(f for f in os.listdir(self.out) if f.startswith("helixqa_http"))
    def test_good_run_publishes_text_json_stderr_and_exit_record(self):
        self.build(); r = self.run_it(self.GOOD, "note\x1b]2;x\x07\n"); self.assertEqual(r.returncode, 0, r.stdout + r.stderr)
        self.assertEqual(self.published(), ["helixqa_http.exit.txt", "helixqa_http.json", "helixqa_http.stderr.txt", "helixqa_http.txt"])
        self.assertEqual(json.load(open(os.path.join(self.out, "helixqa_http.json"))), {"passed": 2, "failed": 0}); self.assertIn("helixqa exit=0", r.stdout)
        for n in self.published(): self.assertNotIn(b"\x1b", open(os.path.join(self.out, n), "rb").read(), n)
        self.assertIn("[container:stderr] note", open(os.path.join(self.out, "helixqa_http.stderr.txt")).read())
    def test_helixqa_exit_status_is_still_the_runners_exit_status_when_evidence_is_fine(self):
        self.build(); r = self.run_it(self.GOOD, rc=3); self.assertEqual(r.returncode, 3); self.assertIn("helixqa exit=3", r.stdout)
    def test_hostile_json_text_is_reserialised_not_copied(self):
        self.build(); r = self.run_it('t\x1b]2;x\x07\n{\n  "n": "a\\u001b]52;c;x\\u0007b"\n}\n'); self.assertEqual(r.returncode, 0, r.stderr)
        raw = open(os.path.join(self.out, "helixqa_http.json"), "rb").read(); self.assertNotIn(b"\x1b", raw); self.assertEqual(json.loads(raw)["n"], "a\x1b]52;c;x\x07b")
        self.assertNotIn(b"\x1b", open(os.path.join(self.out, "helixqa_http.txt"), "rb").read())
    BROKEN = {"no_split": 'bank: PASS\nsummary: 2 passed\n', "bad_json": 'bank: PASS\n{\n  "passed": 2,\n', "trailing_garbage": 'bank\n{\n}\n[1]\n', "empty": ""}
    def test_parse_failure_fails_closed_publishes_nothing_keeps_the_raw_file_and_never_prints_success(self):
        for name, out in self.BROKEN.items():
            self.build(); r = self.run_it(out); self.assertNotEqual(r.returncode, 0, (name, r.stdout, r.stderr))
            self.assertEqual([f for f in self.published() if f != "helixqa_http.exit.txt" or True], [], (name, self.published()))
            self.assertNotIn("helixqa exit=", r.stdout + r.stderr, name)
            kept = os.path.join(self.work, "hq-raw", "stdout.raw"); self.assertTrue(os.path.exists(kept), name); self.assertEqual(open(kept).read(), out, name)
            self.assertIn("nothing published", r.stderr)
    def test_parse_failure_even_when_the_container_exited_zero_is_not_a_green(self):
        self.build(); r = self.run_it("no json here\n", rc=0); self.assertEqual(r.returncode, 1)
    def test_runner_uses_raw_mode(self):
        s = (QA / "run_helixqa_container.sh").read_text(); self.assertRegex(s, r'run_container\.sh" [^\n]* -o "\$RAW"'); self.assertIn("set -eu -o pipefail", s)
    # paired mutants: each restores a bad behaviour and the matching test above must then fail (we assert the bad behaviour is visible)
    def test_mutant_swallowed_publish_failure_is_a_false_green(self):
        s = (QA / "run_helixqa_container.sh").read_text(); old = '"$OUT" || { echo "run_helixqa_container: could not turn'; self.assertIn(old, s)
        self.build(runner=s.replace(old, '"$OUT" || { :; true; } || { echo "run_helixqa_container: could not turn', 1)); r = self.run_it(self.BROKEN["no_split"])
        self.assertEqual(r.returncode, 0); self.assertIn("helixqa exit=0", r.stdout)           # the original false green: success printed, nothing published
    def test_mutant_raw_output_deleted_before_the_check(self):
        s = (QA / "run_helixqa_container.sh").read_text(); old = 'python3 -I "$R/qa/isolation/helixqa_publish.py"'; self.assertIn(old, s)
        self.build(runner=s.replace(old, 'find "$RAW" -type f -delete; ' + old, 1)); self.run_it(self.BROKEN["no_split"])
        self.assertFalse(os.path.exists(os.path.join(self.work, "hq-raw", "stdout.raw")))
    def test_mutant_publisher_ignores_a_bad_json_block(self):
        s = (QA / "helixqa_publish.py").read_text(); old = "doc = json.loads(text[i + 1:])"; self.assertIn(old, s)
        self.build(publisher=s.replace(old, "doc = {}", 1)); r = self.run_it(self.BROKEN["bad_json"]); self.assertEqual(r.returncode, 0)

# ------------------------------------------------------------------------------------------------------------------------------------------ gate (d)
class RawConsumersGate(Base):
    G = ISO / "check_raw_consumers.py"
    def tree(self, files):
        t = os.path.join(self.d, "t"); shutil.rmtree(t, ignore_errors=True)
        for rel, body in files.items():
            p = os.path.join(t, rel); os.makedirs(os.path.dirname(p), exist_ok=True); open(p, "w").write(body)
        return t
    def gate(self, files, script=None):
        return subprocess.run([sys.executable, "-I", str(script or self.G), self.tree(files)], capture_output=True, text=True, timeout=60)
    def test_the_repo_tree_is_clean(self):
        r = subprocess.run([sys.executable, "-I", str(self.G), str(ROOT)], capture_output=True, text=True); self.assertEqual(r.returncode, 0, r.stderr + r.stdout)
    SH_BAD = ['out=$("$R/tools/isolation/run_container.sh" -n hrdiso-x -- cmd)', '"$R/tools/isolation/run_container.sh" -n hrdiso-x -- cmd > "$OUT/o.txt"',
              '"$R/tools/isolation/run_container.sh" -n hrdiso-x -- cmd | grep PASS', '"$R/tools/isolation/run_container.sh" -n hrdiso-x \\\n  -- cmd 2> err.txt',
              '"$R/tools/isolation/run_container.sh" -n hrdiso-x -- cmd 2>&1 | tee log']
    SH_OK = ['"$R/tools/isolation/run_container.sh" -n hrdiso-x -o "$RAW" -- cmd > "$OUT/o.txt"', '"$R/tools/isolation/run_container.sh" -n hrdiso-x -- cmd > /dev/null 2>&1',
             '"$R/tools/isolation/run_container.sh" -n hrdiso-x -- cmd || rc=$?', '# "$R/tools/isolation/run_container.sh" -n x -- cmd | grep PASS',
             '"$R/tools/isolation/run_container.sh" -d -n hrdiso-x -- cmd | head -1  # raw-consumer-exempt: detached, prints only the container id']
    def test_shell_captures_without_o_fail_and_raw_or_uncaptured_pass(self):
        for c in self.SH_BAD:
            r = self.gate({"qa/x.sh": "#!/bin/bash\n" + c + "\n"}); self.assertEqual(r.returncode, 1, (c, r.stdout, r.stderr))
        for c in self.SH_OK:
            r = self.gate({"qa/x.sh": "#!/bin/bash\n" + c + "\n"}); self.assertEqual(r.returncode, 0, (c, r.stdout, r.stderr))
    PY_BAD = ['import subprocess\ndef f():\n    cmd = ["tools/isolation/run_container.sh", "-n", "x", "--", "true"]\n    return subprocess.run(cmd, capture_output=True, text=True)\n',
              'import subprocess\nsubprocess.check_output(["tools/isolation/run_container.sh", "--", "true"])\n',
              'import subprocess\nr = subprocess.run(["x/run_container.sh", "--", "true"], stdout=subprocess.PIPE)\n']
    PY_OK = ['import subprocess\nr = subprocess.run(["x/run_container.sh", "-o", "/d", "--", "true"], capture_output=True)\n',
             'import subprocess\nsubprocess.Popen(["x/run_container.sh", "--", "sleep", "1"], stdout=subprocess.DEVNULL)\n',
             'import subprocess\ndef f():\n    # raw-consumer-exempt: refused before any container exists\n    return subprocess.run(["x/run_container.sh", "-n", "bad"], capture_output=True)\n',
             'from rawrun import run_raw\nr = run_raw(["-n", "x"], ["true"])\n']
    def test_python_captures_without_o_fail_and_raw_or_uncaptured_pass(self):
        for c in self.PY_BAD:
            r = self.gate({"tools/x.py": c}); self.assertEqual(r.returncode, 1, (c, r.stdout, r.stderr))
        for c in self.PY_OK:
            r = self.gate({"tools/x.py": c}); self.assertEqual(r.returncode, 0, (c, r.stdout, r.stderr))
    def test_mutant_runner_without_o_is_caught_by_the_gate(self):
        shutil.copy(QA / "run_helixqa_container.sh", self.tree({"qa/isolation/.keep": ""}) + "/qa/isolation/")
        p = os.path.join(self.d, "t/qa/isolation/run_helixqa_container.sh"); s = open(p).read(); open(p, "w").write(s.replace(' -o "$RAW"', "", 1).replace("--json || RC=$?", '--json > "$RAW/stdout.raw" 2> "$RAW/stderr.raw" || RC=$?', 1))
        r = subprocess.run([sys.executable, "-I", str(self.G), os.path.join(self.d, "t")], capture_output=True, text=True); self.assertEqual(r.returncode, 1, r.stdout + r.stderr)
        self.assertIn("run_helixqa_container.sh", r.stderr)
    def test_mutant_gate_that_never_flags(self):
        t, f = mutated_copy("check_raw_consumers.py", "return bool(re.search(", "return False and bool(re.search("); self.addCleanup(shutil.rmtree, t, True)
        r = self.gate({"qa/x.sh": "#!/bin/bash\n" + self.SH_BAD[1] + "\n"}, script=f); self.assertEqual(r.returncode, 0)          # the mutant is blind: the real gate (test above) is not
    def test_the_existing_consumers_are_covered_by_the_gate_inputs(self):
        for rel in ("qa/isolation/run_helixqa_container.sh", "tools/isolation/tests/canary_probe.py", "tools/isolation/tests/test_network_isolation.py", "tools/isolation/tests/test_runtime_checks.py"):
            self.assertTrue((ROOT / rel).exists(), rel)

# ------------------------------------------------------------------------------------------------------------------------------------------ R5-I2 / minors
class SealAndSanitizer(Base):
    def test_helixqa_exit_record_name_is_publishable_and_the_old_name_is_refused(self):
        out = os.path.join(self.d, "o"); os.makedirs(out)
        for n in ("helixqa_http.exit.txt", "helixqa_http.json", "helixqa_http.txt", "helixqa_http.stderr.txt"): open(os.path.join(out, n), "w").write("x")
        r = subprocess.run([sys.executable, "-I", str(ISO / "safe_io.py"), "scan", out, "--allow", "--host-seal"], capture_output=True, text=True); self.assertEqual(r.returncode, 0, r.stderr)
        open(os.path.join(out, "helixqa_http.exit"), "w").write("x")
        r = subprocess.run([sys.executable, "-I", str(ISO / "safe_io.py"), "scan", out, "--allow", "--host-seal"], capture_output=True, text=True); self.assertEqual(r.returncode, 1)
    BIDI = "a\u202eb\u2066c\u2028d\u200ee\u2029f"
    def san(self, text, script=None):
        return subprocess.run([sys.executable, "-I", str(script or ISO / "sanitize.py")], input=text.encode("utf-8"), capture_output=True, timeout=30).stdout.decode("utf-8")
    def test_bidi_and_line_separator_controls_are_stripped(self):
        out = self.san(self.BIDI + "\n")
        self.assertEqual(sum(1 for ch in out if ch in "\u202e\u2066\u2028\u200e\u2029"), 0, repr(out)); self.assertIn("abcdef", out.replace("\n", ""))
    def test_mutant_S4_bidi_not_stripped_survives_without_this_test(self):
        t, f = mutated_copy("sanitize.py", 's = _CTL.sub("", s)', 's = re.sub("[\\x00-\\x08\\x0b-\\x1f\\x7f-\\x9f]", "", s)'); self.addCleanup(shutil.rmtree, t, True)
        out = self.san(self.BIDI + "\n", script=f); self.assertTrue(any(ch in out for ch in "\u202e\u2066\u2028\u200e\u2029"))
    def test_lone_escape_never_survives_and_may_consume_one_following_byte(self):
        for s in ("\x1b", "a\x1b", "\x1b\x1b\x1b[31mred", "\x1b\x1b", "x\x1b]", "\x1b[", "\x1bP"):
            out = self.san(s + "\n"); self.assertNotIn("\x1b", out, repr(s)); self.assertTrue(out.startswith("[container] "))
    def test_line_cap_applies_to_the_terminal_copy_only(self):
        out = self.san("J" * 9000 + "\n"); self.assertTrue(all(len(l) <= len("[container] ") + 4003 for l in out.splitlines()))
        self.assertIn("MAXLINE = 4000", (ISO / "sanitize.py").read_text())

if __name__ == "__main__":
    unittest.main()
