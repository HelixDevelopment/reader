"""Review round 4 (PR #5) regression tests.
R4-B1  the post-run scan of a container-written tree is an ALLOW-LIST (safe name charset + publishable extension), not a name list: embedded bare git
       repositories, agent files outside any list, names with escape bytes / newlines / leading dash and planted checksum manifests are refused by construction;
       seal_evidence.sh refuses (exit 3, seals nothing) and no host git ever runs.
R4-B2  no container-controlled byte reaches the operator's terminal or a host-written log unfiltered: one sanitizer on every path (attached run_container,
       koreader podman exec, proxy-log sink, redis sink, published ko logs) and the proxy logs request lines percent-escaped.
R4-I1  lint: global runtime options before the subcommand and the wider runtime family / list forms.   R4-I2: a failing KOSync proof fails the run.
Hermetic: scratch dirs under ~/.cache/claude-tmp, PATH stubs (NEVER the real podman / git), no container, no GUI. Every fix has a paired mutant: ONE text mutation of
the tool in a temp copy and the matching test must then show the vulnerable behaviour."""
import importlib.util, os, pathlib, re, shutil, socket, subprocess, sys, tempfile, textwrap, threading, time, unittest
ROOT = pathlib.Path(__file__).resolve().parents[3]
ISO = ROOT / "tools/isolation"
CT = os.path.expanduser("~/.cache/claude-tmp")

def load(path, name):
    spec = importlib.util.spec_from_file_location(name, str(path)); m = importlib.util.module_from_spec(spec); spec.loader.exec_module(m); return m

R1 = load(ISO / "tests/test_review_round1.py", "rr4_helpers1")
R3 = load(ISO / "tests/test_review_round3.py", "rr4_helpers3")
mutated_copy, Stub, run_rc = R1.mutated_copy, R1.Stub, R1.run_rc
SIO = load(ISO / "safe_io.py", "safe_io_rr4")
BC = load(ISO / "bindcheck.py", "bindcheck_rr4")
SAN = load(ISO / "sanitize.py", "sanitize_rr4")
py = R3.py
ESC = b"\x1b"

class Scratch(unittest.TestCase):
    def setUp(self):
        self.d = tempfile.mkdtemp(prefix="rr4-", dir=CT); self.out = os.path.join(self.d, "out"); os.makedirs(os.path.join(self.out, "deep", "er"))
    def tearDown(self): shutil.rmtree(self.d, ignore_errors=True)
    def plant(self, name, kind="f", where="deep/er"):
        p = os.path.join(self.out, where, name); os.makedirs(os.path.dirname(p), exist_ok=True)
        if kind == "d": os.makedirs(os.path.join(p, "x")); open(os.path.join(p, "x", "x.txt"), "w").write("x")
        else: open(p, "w").write("x")
        return p
    def scan(self, *a): return py(ISO / "safe_io.py", "scan", self.out, *a)
    def bare_repo(self, where="run1"):
        b = os.path.join(self.out, where); os.makedirs(os.path.join(b, "objects")); os.makedirs(os.path.join(b, "refs"))
        open(os.path.join(b, "HEAD"), "w").write("ref: refs/heads/main\n")
        open(os.path.join(b, "config"), "w").write('[core]\n\tbare = false\n\tworktree = .\n\tfsmonitor = "touch PWNED; false"\n'); open(os.path.join(b, "shot.png"), "w").write("png")
        return b

# ----------------------------------------------------------------------------------------------------------------------------------------------- R4-B1
class B1AllowList(Scratch):
    AGENT_FILES = ["CLAUDE.local.md", "claude.local.md", "AGENT.md", "CRUSH.md", "CONVENTIONS.md", ".crush", ".opencode", "opencode.json", ".kimi", ".pi", ".roo",
                   ".continue", ".github", ".junie", ".amazonq", ".kiro", "copilot-instructions.md", "Makefile", "package.json", "conftest.py"]
    BAD_NAMES = ["\x1b]52;c;cHduZWQ=\x07.png", "a\nb.png", "-rf.png", "a b.png", ".hidden.png", "x.html", "notes.md", "run.py", "run.sh", "noext", "HEAD", "config",
                 ".SHA256SUMS.json", "SHA256SUMS", "sha256sums.txt", "x" * 200 + ".png", "a;b.png"]
    GOOD = ["shot.png", "A01_filemanager.png", "steps.log", "ko_A.log", "kosync_bodies.jsonl", "result.json", "video.mp4", "a.b.c.txt", "s.webm", "p.JPG", "t.csv", "_x.gif"]
    def test_agent_files_outside_the_deny_names_are_refused_as_file_and_dir_at_depth(self):
        for n in self.AGENT_FILES:
            for kind in ("f", "d"):
                p = self.plant(n, kind); r = self.scan("--allow")
                self.assertEqual(r.returncode, 1, (n, kind, r.stderr)); (shutil.rmtree if kind == "d" else os.unlink)(p)
    def test_unsafe_names_and_unlisted_types_are_refused(self):
        for n in self.BAD_NAMES:
            p = self.plant(n); r = self.scan("--allow"); self.assertEqual(r.returncode, 1, (n, r.stderr)); self.assertNotIn("\x1b", r.stderr)   # the refusal message never echoes the raw bytes
            os.unlink(p)
        for n in self.GOOD: self.plant(n)
        r = self.scan("--allow"); self.assertEqual(r.returncode, 0, r.stderr)
    def test_dot_directories_are_refused_wholesale_and_dirs_have_no_dots(self):
        for n in (".cache", ".x", "a.b", "v1.2"):
            p = self.plant(n, "d"); self.assertEqual(self.scan("--allow").returncode, 1, n); shutil.rmtree(p)
        self.plant("shots", "d"); self.assertEqual(self.scan("--allow").returncode, 0)
    def test_embedded_bare_git_repo_is_refused_by_every_scan_and_by_the_pre_run_bindcheck(self):
        self.bare_repo()
        for flag in ((), ("--allow",)):
            r = self.scan(*flag); self.assertEqual(r.returncode, 1, flag); self.assertIn("git directory", r.stderr)
        self.assertIn("git directory", BC.hostile_tree_error(self.out) or "")
        shutil.rmtree(self.out); os.makedirs(os.path.join(self.out, "worktree-like")); open(os.path.join(self.out, "worktree-like", "gitdir"), "w").write("x"); os.makedirs(os.path.join(self.out, "worktree-like", "refs"))
        self.assertEqual(self.scan("--allow").returncode, 1)
    def test_bare_repo_with_fsmonitor_is_never_sealed_and_no_host_git_runs(self):
        marker = os.path.join(self.d, "git-ran"); bd = os.path.join(self.d, "bin"); os.makedirs(bd)
        open(os.path.join(bd, "git"), "w").write(f'#!/bin/bash\necho "$*" >> {marker}\n'); os.chmod(os.path.join(bd, "git"), 0o755)
        evd = os.path.join(self.out, "run1"); self.bare_repo()
        env = dict(os.environ, PATH=bd + ":" + os.environ["PATH"])
        r = subprocess.run(["bash", str(ISO / "seal_evidence.sh"), evd], capture_output=True, text=True, env=env, timeout=60)
        self.assertEqual(r.returncode, 3, r.stdout + r.stderr); self.assertFalse(os.path.exists(os.path.join(evd, "SHA256SUMS")))
        self.assertEqual([f for f in os.listdir(evd) if f.startswith(".SHA256SUMS")], [])                      # not even the temp file
        self.assertFalse(os.path.exists(marker), "a host git ran"); self.assertFalse(os.path.exists(os.path.join(evd, "PWNED")))
    def test_planted_checksum_manifests_are_refused_never_left_unsealed(self):
        for name, where in ((".SHA256SUMS.json", "."), ("SHA256SUMS", "deep"), ("sha256sums.txt", "."), (".SHA256SUMS.abc", ".")):
            p = self.plant(name, where=where); r = subprocess.run(["bash", str(ISO / "seal_evidence.sh"), self.out], capture_output=True, text=True, timeout=60)
            self.assertEqual(r.returncode, 3, (name, r.stdout + r.stderr)); os.unlink(p)
    def test_seal_accepts_a_clean_tree_hashes_every_file_and_resealing_replaces_the_manifest(self):
        for n in self.GOOD: self.plant(n)
        for _ in range(2):
            r = subprocess.run(["bash", str(ISO / "seal_evidence.sh"), self.out], capture_output=True, text=True, timeout=60); self.assertEqual(r.returncode, 0, r.stdout + r.stderr)
        sums = open(os.path.join(self.out, "SHA256SUMS")).read().splitlines(); self.assertEqual(len(sums), len(self.GOOD))
        self.assertEqual(subprocess.run(["sha256sum", "-c", "--quiet", "SHA256SUMS"], cwd=self.out).returncode, 0)
        self.assertEqual([f for f in os.listdir(self.out) if f.startswith(".SHA256SUMS")], [])
    def test_seal_refuses_unlisted_types_but_host_authored_mode_is_explicit(self):
        self.plant("run.py"); self.plant("notes.md")
        r = subprocess.run(["bash", str(ISO / "seal_evidence.sh"), self.out], capture_output=True, text=True, timeout=60); self.assertEqual(r.returncode, 3)
        r = subprocess.run(["bash", str(ISO / "seal_evidence.sh"), "--host-authored", self.out], capture_output=True, text=True, timeout=60); self.assertEqual(r.returncode, 0, r.stderr)
        self.plant("CLAUDE.md"); r = subprocess.run(["bash", str(ISO / "seal_evidence.sh"), "--host-authored", self.out], capture_output=True, text=True, timeout=60); self.assertEqual(r.returncode, 3)   # deny table still applies
    def test_scan_stops_at_the_file_cap(self):
        for i in range(30): self.plant(f"f{i}.txt")
        env = dict(os.environ, SAFE_IO_MAX_FILES="5"); r = py(ISO / "safe_io.py", "scan", self.out, "--allow", env=env); self.assertEqual(r.returncode, 1); self.assertIn("exceed the cap", r.stderr)
        self.assertLess(r.stderr.count("\n"), 5)
    # ---- paired mutants: each ONE text change must re-open the hole
    def mut_scan(self, old, new, *flag):
        t, f = mutated_copy("safe_io.py", old, new); self.addCleanup(shutil.rmtree, t, True); return py(f, "scan", self.out, *flag)
    def test_mutant_gitdir_check_dropped_accepts_the_bare_repo_in_a_plain_scan(self):
        self.bare_repo(); self.assertEqual(self.mut_scan("if gitdir_like(names):", "if False:").returncode, 0)
    def test_mutant_name_charset_dropped_accepts_unsafe_names(self):
        for n in ("-rf.png", "a\nb.png", "\x1b]52;c;x\x07.png"): self.plant(n)
        self.assertEqual(self.mut_scan("not (SAFE_DIR if stat.S_ISDIR(st.st_mode) else SAFE_FILE).match(e)", "False", "--allow").returncode, 0)
    def test_mutant_checksum_name_refusal_dropped_accepts_a_planted_manifest(self):
        self.plant("sha256sums.txt"); self.assertEqual(self.mut_scan('if allow and e.lower().lstrip(".").startswith("sha256sums"):', "if False:", "--allow").returncode, 0)
    def test_mutant_md_on_the_publish_list_accepts_an_agent_instruction_file(self):
        self.plant("notes.md"); self.assertEqual(self.mut_scan('"txt", "csv"', '"txt", "md", "csv"', "--allow").returncode, 0)
    def test_mutant_extension_check_dropped_accepts_scripts(self):
        self.plant("run.py"); self.assertEqual(self.mut_scan('if allow and (("." not in e)', "if False and (('.' not in e)", "--allow").returncode, 0)
    def test_mutant_seal_without_allow_list_seals_scripts_and_unsafe_names(self):
        self.plant("run.py"); t, f = mutated_copy("seal_evidence.sh", "MODE=--allow", "MODE="); self.addCleanup(shutil.rmtree, t, True)
        r = subprocess.run(["bash", str(f), self.out], capture_output=True, text=True, timeout=60); self.assertEqual(r.returncode, 0, r.stderr)
    def test_mutant_bindcheck_without_gitdir_check_accepts_the_bare_repo(self):
        self.bare_repo(); t, f = mutated_copy("bindcheck.py", "if safe_io.gitdir_like(dn + fn):", "if False:"); self.addCleanup(shutil.rmtree, t, True)
        self.assertIsNone(R3.load(f, "bcm_gitdir").hostile_tree_error(self.out))
    def test_runners_use_the_allow_list_scan_and_seal_uses_it_by_default(self):
        for n in ("run_koreader_gui.sh", "run_web_live.sh", "run_desktop_live.sh"): self.assertIn('scan "$OUT" --allow', (ROOT / "qa/isolation" / n).read_text(), n)
        self.assertIn("scan \"$D\" --allow --host-seal", (ISO / "seal_evidence.sh").read_text())

# ----------------------------------------------------------------------------------------------------------------------------------------------- R4-B2
HOSTILE = b"ok\x1b]2;OWNED-TITLE\x07 \x1b]52;c;cHduZWQ=\x07 \x1b]0;t\x1b\\ \x1bP1;x\x1b\\ \x1b[2K\x1b[1;1H \x9d]52;c;x\x07 \r2026-11-24T00:00:00Z KOSYNC PROOF PASS (forged)\x08\x00\x7f\xe2\x80\xae end\n"

def clean(b):
    t = b.decode("utf-8", "replace")
    return not any((ord(c) < 32 and c not in "\n\t") or 0x7f <= ord(c) <= 0x9f or c in "‮  ‎" for c in t)

class B2Sanitizer(unittest.TestCase):
    def run_san(self, b, *a): return subprocess.run([sys.executable, "-I", str(ISO / "sanitize.py"), *a], input=b, capture_output=True, timeout=60)
    def test_hostile_stream_comes_out_free_of_terminal_controls_and_every_line_is_prefixed(self):
        r = self.run_san(HOSTILE); self.assertEqual(r.returncode, 0); self.assertNotIn(ESC, r.stdout); self.assertNotIn(b"52;c;cHdu", r.stdout); self.assertTrue(clean(r.stdout), r.stdout)
        lines = r.stdout.decode().splitlines(); self.assertTrue(lines and all(l.startswith("[container] ") for l in lines), lines)
        forged = [l for l in lines if "KOSYNC PROOF PASS (forged)" in l]; self.assertTrue(forged and all(l.startswith("[container] ") for l in forged))
    def test_random_bytes_never_yield_a_control_byte(self):
        import random
        rnd = random.Random(4)
        for _ in range(50):
            b = bytes(rnd.randrange(256) for _ in range(rnd.randrange(1, 400))); o = self.run_san(b).stdout
            self.assertTrue(clean(o) and ESC not in o, o)
    def test_prefix_option_and_usage(self):
        self.assertEqual(self.run_san(b"a\n", "--prefix", "[p] ").stdout, b"[p] a\n"); self.assertEqual(self.run_san(b"a\n", "--prefix", "").stdout, b"a\n")
        self.assertEqual(self.run_san(b"x", "--nonsense").returncode, 2)
    def test_long_line_is_bounded_and_unterminated_osc_is_dropped(self):
        o = self.run_san(b"a" * 300000 + b"\n").stdout; self.assertTrue(len(o) < 300000 + 200)
        self.assertEqual(self.run_san(b"x\x1b]52;c;AAAA").stdout, b"[container] x\n")
    def test_mutant_without_osc_or_c0_removal_leaks(self):
        s = (ISO / "sanitize.py").read_text(); t = tempfile.mkdtemp(prefix="san-", dir=CT); self.addCleanup(shutil.rmtree, t, True)
        for old, new in (("for rx in (_OSC, _STR, _CSI, _ESC):", "for rx in ():"), ("s = _CTL.sub(\"\", s)", "pass")):
            self.assertIn(old, s); p = pathlib.Path(t, "m.py"); p.write_text(s.replace(old, new, 1))
            o = subprocess.run([sys.executable, "-I", str(p)], input=HOSTILE, capture_output=True, timeout=60).stdout; self.assertTrue(b"52;c;cHdu" in o or ESC in o or not clean(o), (old, o))
        p = pathlib.Path(t, "m.py"); p.write_text(s.replace("out.append(prefix + sanitize_line(ln))", "out.append(sanitize_line(ln))", 1))
        o = subprocess.run([sys.executable, "-I", str(p)], input=HOSTILE, capture_output=True, timeout=60).stdout
        self.assertTrue(any(l.startswith("2026") or l.startswith("ok") for l in o.decode().splitlines()))                      # unprefixed: a forged host-looking line survives

class B2Proxy(unittest.TestCase):
    """the real access_proxy.py on a unix socket: a request line with ESC bytes is logged percent-escaped (and cannot forge a log line)."""
    def serve(self, src):
        d = tempfile.mkdtemp(prefix="px-", dir=CT); self.addCleanup(shutil.rmtree, d, True)
        class H(__import__("http.server").server.BaseHTTPRequestHandler):
            def do_GET(s): s.send_response(200); s.send_header("Content-Length", "2"); s.end_headers(); s.wfile.write(b"{}")
            def log_message(s, *a): pass
        up = __import__("http.server").server.ThreadingHTTPServer(("127.0.0.1", 0), H); threading.Thread(target=up.serve_forever, daemon=True).start(); self.addCleanup(up.shutdown)
        sp, lg = f"{d}/p.sock", f"{d}/a.log"
        pr = subprocess.Popen([sys.executable, "-I", str(src), "--policy", "common", f"unix:{sp}", "127.0.0.1", str(up.server_address[1]), lg]); self.addCleanup(pr.terminate)
        for _ in range(50):
            if os.path.exists(sp): break
            time.sleep(0.1)
        return sp, lg
    def hit(self, sp, raw):
        c = socket.socket(socket.AF_UNIX); c.settimeout(5); c.connect(sp); c.sendall(raw)
        try:
            while c.recv(65536): pass
        except socket.timeout: pass
        c.close(); time.sleep(0.3)
    RAW = b"GET /syncs/progress/\x1b]52;c;cHduZWQ=\x07\x1b]2;OWNED\x07\nFORGED KOSYNC PROOF PASS HTTP/1.1\r\nHost: x\r\nConnection: close\r\n\r\n"
    RAW2 = b"GET /opds/1.2/\x1b]2;OWNED\x07 HTTP/1.1\r\nHost: x\r\nConnection: close\r\n\r\n"
    def test_request_path_with_escape_bytes_is_logged_escaped(self):
        sp, lg = self.serve(ISO / "access_proxy.py")
        for raw in (self.RAW, self.RAW2): self.hit(sp, raw)
        data = open(lg, "rb").read(); self.assertTrue(data, "nothing logged"); self.assertNotIn(ESC, data); self.assertTrue(clean(data), data)
        self.assertIn(b"%1B", data); self.assertTrue(all(b" -> " in l for l in data.splitlines()), data)               # no forged extra line
    def test_mutant_raw_path_logging_leaks_the_escape_bytes(self):
        t, f = mutated_copy("access_proxy.py", "{esc(self.command)} {esc(self.path.split('?')[0])} -> {status}", "{self.command} {self.path.split('?')[0]} -> {status}"); self.addCleanup(shutil.rmtree, t, True)
        sp, lg = self.serve(f); self.hit(sp, self.RAW2); self.assertIn(ESC, open(lg, "rb").read())

class Run(Scratch):
    """run_container.sh in ATTACHED mode: the container's output passes the sanitizer, podman's exit status survives."""
    def go(self, script_dir=None, code=0):
        with Stub() as st:
            iso = os.path.join(self.d, "iso"); os.makedirs(os.path.join(iso, "run1"), exist_ok=True); st.env["ISO_WORK"] = iso
            body = (R1.STUB).replace("  run) exit 0;;", "  run) printf '%s\\n' \"$PAYLOAD\"; printf 'to-stderr\\x1b]52;c;x\\x07\\n' >&2; exit \"${STUB_RC:-0}\";;")
            pathlib.Path(st.d, "podman").write_text(body); st.env["PAYLOAD"] = HOSTILE.decode("latin-1").replace("\n", " ").replace("\x00", ""); st.env["STUB_RC"] = str(code)
            script = (script_dir or ISO) / "run_container.sh"
            return run_rc(script, ["-n", "hrdiso-p4", "-w", f"{iso}/run1:/out", "--", "true"], st)
    def test_attached_output_is_sanitized_and_exit_status_preserved(self):
        r = self.go(code=7); self.assertEqual(r.returncode, 7, r.stderr); self.assertNotIn("\x1b", r.stdout + r.stderr); self.assertIn("[container] ", r.stdout)
        self.assertTrue(clean((r.stdout + r.stderr).encode()))
        r = self.go(code=0); self.assertEqual(r.returncode, 0)
    def test_mutant_unsanitized_attached_run_leaks(self):
        t, f = mutated_copy("run_container.sh", '2>&1 | python3 -I "$HERE/sanitize.py"', "2>&1"); self.addCleanup(shutil.rmtree, t, True)
        r = self.go(script_dir=f.parent, code=0); self.assertIn("\x1b", r.stdout + r.stderr)
    def test_mutant_lost_exit_status(self):
        t, f = mutated_copy("run_container.sh", 'exit "${PIPESTATUS[0]}"', "exit 0"); self.addCleanup(shutil.rmtree, t, True)
        self.assertEqual(self.go(script_dir=f.parent, code=7).returncode, 0)

class KoHarness(R3.Harness):
    """the real run_koreader_gui.sh with a podman stub whose `exec` prints hostile bytes; kosync_proof stubbed to pass / fail / crash."""
    PODMAN = textwrap.dedent('''\
        echo "podman $*" >> "$EVLOG"
        case $1 in exec) printf '%s\\n' "$PAYLOAD"; printf 'x\\033]52;c;x\\007\\n' >&2;; esac
        exit 0
    ''')
    def kobuild(self, ko=None, proof="import sys\nopen(sys.argv[2],'w').write('{}')\n"):
        self.build(ko=ko); bd = os.path.join(self.d, "bin"); p = os.path.join(bd, "podman"); open(p, "w").write("#!/bin/bash\n" + self.PODMAN); os.chmod(p, 0o755)
        open(os.path.join(self.T, "qa/isolation/kosync_proof.py"), "w").write(proof)
        self.env["PAYLOAD"] = HOSTILE.decode("latin-1").replace("\n", " ").replace("\x00", "")

class B2Koreader(KoHarness):
    def test_no_escape_byte_reaches_stdout_stderr_steps_log_or_published_evidence(self):
        self.kobuild(); r = self.run_script("run_koreader_gui.sh", [self.out]); self.assertEqual(r.returncode, 0, r.stdout + r.stderr)
        for name, data in (("stdout", r.stdout.encode("latin-1", "replace")), ("stderr", r.stderr.encode("latin-1", "replace"))): self.assertNotIn(ESC, data, name)
        self.assertGreater(len(r.stdout), 100)
        files = [os.path.join(dp, f) for dp, _, fs in os.walk(self.out) for f in fs]; self.assertTrue(any(f.endswith("steps.log") for f in files), files)
        for f in files: self.assertNotIn(ESC, open(f, "rb").read(), f)
        steps = open(os.path.join(self.out, "steps.log")).read()
        forged = [l for l in steps.splitlines() if "KOSYNC PROOF PASS (forged)" in l]; self.assertTrue(forged)
        self.assertTrue(all("[container] 2026-11-24T00:00:00Z KOSYNC PROOF PASS (forged)" in l for l in forged), forged)       # the forged text is container-marked, never a bare host line
        self.assertFalse([l for l in steps.splitlines() if l.startswith("2026-11-24T00:00:00Z KOSYNC")])
    def test_every_podman_exec_and_echoed_file_goes_through_the_sanitizer(self):
        s = (ROOT / "qa/isolation/run_koreader_gui.sh").read_text()
        self.assertIn('2>&1 | $SAN', s); self.assertIn('| $SAN --prefix "[proxy-log] "', s); self.assertIn('2>&1 | $SAN > "$HW/redis_sink.txt"', s); self.assertIn('$SAN < "$HW/ko_$P.raw"', s)
        self.assertNotRegex(s, r'\n\s*podman exec [^|\n]*ko_drive\.sh[^|\n]*\n')
    def test_mutants_each_sanitizer_stage_is_load_bearing(self):
        s = (ROOT / "qa/isolation/run_koreader_gui.sh").read_text()
        for old, new in (('"$@" 2>&1 | $SAN; }', '"$@"; }'),):
            self.assertIn(old, s); self.kobuild(ko=s.replace(old, new, 1)); r = self.run_script("run_koreader_gui.sh", [self.out]); self.assertIn("\x1b", r.stdout + r.stderr, old)
        old = '| $SAN --prefix "[proxy-log] " | tee'; self.assertIn(old, s)
        self.kobuild(ko=s.replace(old, "| tee", 1)); open(os.path.join(self.st, "access.log"), "w").close()
        self.assertNotIn("[proxy-log] ", self.run_script("run_koreader_gui.sh", [self.out]).stdout)

class I2KosyncVerdict(KoHarness):
    FAIL = "import sys, json\nopen(sys.argv[2],'w').write(json.dumps({'verdict':'FAIL'}))\nprint('verdict FAIL')\nsys.exit(1)\n"
    CRASH = "import sys\nraise ValueError('malformed container-shaped body')\n"
    def test_failing_or_crashing_proof_fails_the_run(self):
        for body in (self.FAIL, self.CRASH):
            self.kobuild(proof=body); r = self.run_script("run_koreader_gui.sh", [self.out]); self.assertNotEqual(r.returncode, 0, r.stdout + r.stderr)
            self.assertIn("KOSYNC PROOF FAILED", open(os.path.join(self.out, "steps.log")).read())
        self.kobuild(); self.assertEqual(self.run_script("run_koreader_gui.sh", [self.out]).returncode, 0)
    def test_mutant_final_status_check_dropped_is_a_false_green(self):
        s = (ROOT / "qa/isolation/run_koreader_gui.sh").read_text(); old = '[ "$PRC" -eq 0 ] || { echo "run_koreader_gui: the KOSync proof did not pass'; self.assertIn(old, s)
        self.kobuild(ko=s.replace(old, ': || { echo "run_koreader_gui: the KOSync proof did not pass', 1), proof=self.FAIL)
        self.assertEqual(self.run_script("run_koreader_gui.sh", [self.out]).returncode, 0)
    def test_the_runner_has_pipefail(self): self.assertRegex((ROOT / "qa/isolation/run_koreader_gui.sh").read_text(), r"set -eu -o pipefail")

# ----------------------------------------------------------------------------------------------------------------------------------------------- R4-I1
class I1Lint(unittest.TestCase):
    G = ISO / "guard/assert_host_untouched.py"
    SHAPES = ["podman --root /tmp/r run -v /:/h img", "podman --log-level=debug run --rm img sh", "docker --context x run img", "docker -H unix:///x run img",
              "podman --remote exec c ls", "nerdctl --namespace k run img", "podman -r run img", "crun run c", "incus exec c -- sh", "ctr run x y", "kubectl exec p -- sh", "multipass exec v -- ls",
              "limactl shell x", "vagrant ssh", "firejail ls", "toolbx run x", "systemd-nspawn -D /x", "machinectl shell x", "host-spawn ls", "flatpak-spawn --host ls", "ch-run img",
              "podman start -a c", "podman container runlabel x y", "podman pod create",
              'subprocess.run(["distrobox","enter","x"])', 'subprocess.run(["flatpak","run","x"])', 'subprocess.run(["runc","run","x"])', 'subprocess.run(["podman-remote","run","x"])']
    BENIGN = ["podman ps", "docker images", "podman --root /tmp/r ps", "echo crun is a runtime", "docker --context x ps"]
    def tree(self, line, ext="sh"):
        d = tempfile.mkdtemp(prefix="i1-", dir=CT); self.addCleanup(shutil.rmtree, d, True); os.makedirs(d + "/qa"); open(f"{d}/qa/a.{ext}", "w").write(("#!/bin/bash\n" if ext == "sh" else "import subprocess\n") + line + "\n"); return d
    def scan(self, g, line, ext=None): return [h[2] for h in g.scan(self.tree(line, ext or ("py" if line.startswith("subprocess") else "sh")))]
    def test_every_shape_is_a_finding_and_benign_commands_are_not(self):
        g = load(self.G, "g_r4i1")
        for c in self.SHAPES: self.assertIn("container-outside-isolation", self.scan(g, c), c)
        for c in self.BENIGN: self.assertEqual(self.scan(g, c), [], c)
    def test_global_option_shape_still_gets_the_escape_check(self):
        g = load(self.G, "g_r4i1b"); self.assertIn("container-escape", self.scan(g, "podman --root /tmp/r run --privileged img"))
    def mutant(self, old, new):
        s = self.G.read_text(); self.assertIn(old, s); t = tempfile.mkdtemp(prefix="i1m-", dir=CT); self.addCleanup(shutil.rmtree, t, True)
        p = pathlib.Path(t, "m.py"); p.write_text(s.replace(old, new, 1)); return load(p, "g_i1m" + str(abs(hash(old))))
    def test_mutant_without_global_option_normalisation_misses_the_global_shapes(self):
        gm = self.mutant("return _GLOBOPT.sub(lambda m: m.group(1), b)", "return b")
        missed = [c for c in self.SHAPES[:6] if "container-outside-isolation" not in self.scan(gm, c)]; self.assertGreaterEqual(len(missed), 3, missed)
    def test_mutant_without_the_list_form_misses_python_lists(self):
        gm = self.mutant('(?:podman-remote|buildah|runc|crun', '(?:zzzx|buildah|runc|crun')
        self.assertNotIn("container-outside-isolation", self.scan(gm, 'subprocess.run(["podman-remote","run","x"])'))
    def test_mutant_without_the_extra_family_misses_incus_and_kubectl(self):
        gm = self.mutant('incus\\s+(?:exec|launch|start|shell|run)\\b|', "")
        self.assertNotIn("container-outside-isolation", self.scan(gm, "incus exec c -- sh"))

if __name__ == "__main__":
    unittest.main()
