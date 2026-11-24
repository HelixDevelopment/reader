"""Review round 1 (PR #5) regression tests. Hermetic: a PATH stub stands in for podman (NEVER the real one); no container, GUI or display.
Each fix has a paired mutant: the tool's source text is mutated in a temp copy and the matching test must then FAIL (check_fails)."""
import os, pathlib, shutil, stat, subprocess, sys, tempfile, textwrap, time, unittest, importlib.util
ROOT = pathlib.Path(__file__).resolve().parents[3]
ISO = ROOT / "tools/isolation"
CT = os.path.expanduser("~/.cache/claude-tmp")
sys.path.insert(0, str(ISO / "guard"))

def load(path, name):
    spec = importlib.util.spec_from_file_location(name, str(path)); m = importlib.util.module_from_spec(spec); spec.loader.exec_module(m); return m

def mutated_copy(sub, old, new):
    """copy tools/isolation to a temp tree (repo-shaped: <t>/tools/isolation, <t>/qa/isolation/inside) with ONE text mutation; -> (tmp, file)"""
    t = tempfile.mkdtemp(prefix="rr1-", dir=CT)
    shutil.copytree(ISO, pathlib.Path(t, "tools/isolation"), ignore=shutil.ignore_patterns("__pycache__", "tests", "guard"))
    pathlib.Path(t, "qa/isolation/inside").mkdir(parents=True)
    f = pathlib.Path(t, "tools/isolation", sub); s = f.read_text()
    assert old in s, f"mutation anchor not found in {sub}: {old!r}"
    f.write_text(s.replace(old, new, 1)); return t, f

STUB = textwrap.dedent('''\
    #!/usr/bin/env bash
    # podman STUB: records argv, never touches containers.
    echo "$*" >> "${STUB_LOG:?}"
    case $1 in
      image) exit 0;;
      ps) cat "${STUB_PS:-/dev/null}" 2>/dev/null; exit 0;;
      inspect) n=$2; f=$4
        case $f in *owner-pid*) cat "$STUB_DIR/$n.pid" 2>/dev/null;; *owner-file*) cat "$STUB_DIR/$n.file" 2>/dev/null;; *lease-ts*) cat "$STUB_DIR/$n.lease" 2>/dev/null;; esac; exit 0;;
      rm) exit 0;;
      run) exit 0;;
    esac; exit 0
''')

class Stub:
    def __enter__(self):
        self.d = tempfile.mkdtemp(prefix="stub-", dir=CT); p = pathlib.Path(self.d, "podman"); p.write_text(STUB); p.chmod(0o755)
        self.log = pathlib.Path(self.d, "log"); self.log.write_text("")
        self.env = dict(os.environ, PATH=self.d + ":" + os.environ["PATH"], STUB_LOG=str(self.log), STUB_DIR=self.d, STUB_PS=str(pathlib.Path(self.d, "ps")))
        return self
    def __exit__(self, *a): shutil.rmtree(self.d, ignore_errors=True)

def run_rc(script, args, st, cwd=None):
    return subprocess.run([str(script)] + args, capture_output=True, text=True, env=st.env, cwd=cwd, timeout=60)

class B2IsoWork(unittest.TestCase):
    """ISO_WORK may not be the repo, inside it, an ancestor of it, or a symlink to it - wherever the repo lives."""
    def setUp(self):
        self.base = tempfile.mkdtemp(prefix="b2-", dir=CT); self.repo = os.path.join(self.base, "x", "repo")
        for d in ("tools/isolation", "qa", "evidence"): os.makedirs(os.path.join(self.repo, d))
        self.b = load(ISO / "bindcheck.py", "bc_b2")
    def tearDown(self): shutil.rmtree(self.base, ignore_errors=True)
    def chk(self, b, iso, src, mode="rw", dst="/work/t"): return b.check(mode, src, dst, self.repo, {"ISO_WORK": iso})
    def cases(self, b):
        r = self.repo; r_ = os.path.join(self.base, "x"); sib = os.path.join(self.base, "x", "sib"); os.makedirs(sib)
        link = os.path.join(self.base, "lnk"); os.symlink(r, link)
        return {"equal": self.chk(b, r, r + "/tools/isolation"), "equal_qa": self.chk(b, r, r + "/qa"),
                "inside": self.chk(b, r + "/evidence", r + "/evidence/run1"), "ancestor": self.chk(b, r_, r + "/tools/isolation"),
                "symlink_to_repo": self.chk(b, link, r + "/tools/isolation"), "golden_sibling": self.chk(b, sib, sib + "/run1")}
    def test_refusals_and_golden_good(self):
        c = self.cases(self.b)
        for k in ("equal", "equal_qa", "inside", "ancestor", "symlink_to_repo"): self.assertIsNotNone(c[k], k)
        self.assertIsNone(c["golden_sibling"], c["golden_sibling"])
    def test_cli_rw_of_host_tooling_refused_when_iso_work_is_repo(self):
        e = dict(os.environ, ISO_WORK=self.repo)
        self.assertEqual(subprocess.run([sys.executable, "-I", str(ISO / "bindcheck.py"), "rw", ROOT / "tools/isolation", "/work/t"], env=e, capture_output=True).returncode, 64)
    def test_mutant_without_repo_check_accepts_repo_as_iso_work(self):
        t, f = mutated_copy("bindcheck.py", "if inside(iso, rr) or inside(rr, iso):", "if False:")
        try:
            c = self.cases(load(f, "bc_mut"))
            self.assertIsNone(c["equal"]); self.assertIsNone(c["ancestor"])      # the mutant is the vulnerable one
        finally: shutil.rmtree(t, ignore_errors=True)
    def test_m1_resolved_path_with_colon_or_comma_refused(self):
        w = os.path.join(self.base, "x", "sib"); os.makedirs(w + "/weird:dir"); os.makedirs(w + "/ok")
        os.symlink(w + "/weird:dir", w + "/lnk"); os.makedirs(w + "/a,b"); os.symlink(w + "/a,b", w + "/lnk2")
        self.assertIsNotNone(self.chk(self.b, w, w + "/lnk")); self.assertIsNotNone(self.chk(self.b, w, w + "/lnk2"))
        self.assertIsNone(self.chk(self.b, w, w + "/ok"))
    def test_m1_mutant(self):
        t, f = mutated_copy("bindcheck.py", 'if any(c in real for c in ":,\\n\\r\\0"):', "if False:")
        try:
            w = os.path.join(self.base, "x", "sib"); os.makedirs(w + "/weird:dir"); os.symlink(w + "/weird:dir", w + "/lnk")
            self.assertIsNone(self.chk(load(f, "bc_m1"), w, w + "/lnk"))
        finally: shutil.rmtree(t, ignore_errors=True)

class I1RunnerEnv(unittest.TestCase):
    BAD = ["DISPLAY", "DISPLAY=:0", "WAYLAND_DISPLAY", "WAYLAND_DISPLAY=w", "XAUTHORITY", "PULSE_SERVER", "PULSE_SERVER=x", "PIPEWIRE_RUNTIME_DIR", "PIPEWIRE_REMOTE=x",
           "XDG_RUNTIME_DIR", "XDG_SESSION_TYPE=x", "DBUS_SESSION_BUS_ADDRESS", "DBUS_SESSION_BUS_ADDRESS=u", "SSH_AUTH_SOCK", "FOO", "1X=1", "A-B=1", "display=:0", "gnome_x=1"]
    def test_env_by_name_refused(self):
        with Stub() as st:
            for e in self.BAD:
                with self.subTest(env=e):
                    r = run_rc(ISO / "run_container.sh", ["-n", "hrdiso-t1", "-e", e, "--", "true"], st)
                    self.assertEqual(r.returncode, 64, (e, r.stderr)); self.assertNotIn("run ", st.log.read_text())
    def test_long_options_and_env_file_refused(self):
        with Stub() as st:
            for a in (["--env", "DISPLAY"], ["--env-file", "/x"], ["--env=DISPLAY"], ["--privileged"], ["--network", "host"]):
                r = run_rc(ISO / "run_container.sh", ["-n", "hrdiso-t2"] + a + ["--", "true"], st)
                self.assertNotEqual(r.returncode, 0, a); self.assertNotIn("run ", st.log.read_text())
    def test_golden_good_explicit_env_reaches_podman(self):
        with Stub() as st:
            r = run_rc(ISO / "run_container.sh", ["-n", "hrdiso-t3", "-e", "HELIX_X=1", "--", "true"], st)
            self.assertEqual(r.returncode, 0, r.stderr); log = st.log.read_text()
            self.assertIn("--network none", log); self.assertIn("-e HELIX_X=1", log); self.assertNotIn("--privileged", log)
    def test_m3_bad_image_and_numeric_args_refused(self):
        with Stub() as st:
            for a in (["-i", "--privileged"], ["-i", "a b"], ["-m", "--x"], ["-c", "x"], ["-t", "1;2"]):
                self.assertEqual(run_rc(ISO / "run_container.sh", ["-n", "hrdiso-t4"] + a + ["--", "true"], st).returncode, 64, a)
            self.assertNotIn("run ", st.log.read_text())
    def mutant_accepts(self, old, new, args):
        t, f = mutated_copy("run_container.sh", old, new)
        try:
            with Stub() as st:
                return run_rc(f, ["-n", "hrdiso-m"] + args + ["--", "true"], st).returncode == 0
        finally: shutil.rmtree(t, ignore_errors=True)
    def test_mutants(self):
        self.assertTrue(self.mutant_accepts("PULSE*|", "", ["-e", "PULSE_SERVER=x"]))
        self.assertTrue(self.mutant_accepts("case $b in *=*) ;; *)", "case $b in *) ;; *)", ["-e", "FOO"]))
        self.assertTrue(self.mutant_accepts("case $IMG in -*|", "case $IMG in ZZZ|", ["-i", "-x"]))
        self.assertTrue(self.mutant_accepts("DISPLAY*|*_DISPLAY|", "", ["-e", "DISPLAY=:0"]))

class M4Reaper(unittest.TestCase):
    def reaped(self, script, pid):
        with Stub() as st:
            pathlib.Path(st.d, "ps").write_text("hrdiso-a\n"); pathlib.Path(st.d, "hrdiso-a.pid").write_text(str(pid)); pathlib.Path(st.d, "hrdiso-a.lease").write_text("1")
            r = subprocess.run([str(script)], capture_output=True, text=True, env=dict(st.env, REAP_GRACE="1"), timeout=60)
            return "rm -f hrdiso-a" in st.log.read_text(), r
    def dead_pid(self):
        p = subprocess.Popen(["true"]); p.wait(); return p.pid
    def test_dead_owner_is_reaped_eperm_owner_is_not(self):
        self.assertTrue(self.reaped(ISO / "reap.sh", self.dead_pid())[0])
        self.assertFalse(self.reaped(ISO / "reap.sh", 1)[0])                          # pid 1: kill -0 -> EPERM for a non-root user (exists)
        self.assertFalse(self.reaped(ISO / "reap.sh", os.getpid())[0])                # live own pid
    def test_mutant_old_logic_reaps_eperm_owner(self):
        t, f = mutated_copy("reap.sh", 'dead "$pid" && stale=1', 'kill -0 "$pid" 2>/dev/null || stale=1')
        try: self.assertTrue(self.reaped(f, 1)[0])
        finally: shutil.rmtree(t, ignore_errors=True)

class I3LintEscapes(unittest.TestCase):
    CASES = {"volume-opt": "podman run --rm --mount type=volume,src=x,dst=/a,volume-opt=device=/h,volume-opt=o=bind img sh\n",
             "env-file": "podman run --rm --env-file host.env img sh\n", "gpus": "docker run --rm --gpus all img sh\n",
             "hooks-dir": "podman run --rm --hooks-dir /x img sh\n", "var-command": "P=podman; $P run --privileged img sh\n",
             "publish-0000": "podman run --rm -p 0.0.0.0:6000:6000 img sh\n", "add-host": "podman run --rm --add-host x:host-gateway img sh\n",
             "annotation": "podman run --rm --annotation a=b img sh\n", "volume-create": "podman volume create --opt device=/h --opt o=bind v\n"}
    GOOD = ["podman run --rm --network none -p 127.0.0.1:5590:5555 --mount type=tmpfs,dst=/t img sh\n", "echo run the tests\n", "podman run --rm -e A=1 img sh\n"]
    def scan(self, g, name, body):
        d = tempfile.mkdtemp(prefix="i3-", dir=CT)
        try:
            pathlib.Path(d, name).parent.mkdir(parents=True, exist_ok=True); pathlib.Path(d, name).write_text(body); return g.scan(d)
        finally: shutil.rmtree(d, ignore_errors=True)
    def test_each_gap_caught_and_good_not(self):
        g = load(ISO / "guard/assert_host_untouched.py", "g_i3")
        for k, body in self.CASES.items(): self.assertTrue(self.scan(g, "tools/a.sh", body), k)
        self.assertTrue(self.scan(g, "tools/host.env", "DISPLAY=:0\n"), "env file content")
        for body in self.GOOD: self.assertEqual(self.scan(g, "tools/a.sh", body), [], body)
        self.assertEqual(self.scan(g, "tools/ok.env", "HELIX_PORT=1\n"), [])
    def test_paired_mutants_each_alternative_is_load_bearing(self):
        src = (ISO / "guard/assert_host_untouched.py").read_text()
        for k, old in (("volume-opt", "volume-opt\\b|"), ("env-file", "--env-file\\b|"), ("gpus", "--gpus\\b|"), ("hooks-dir", "--hooks-dir\\b|"),
                       ("annotation", "--annotation\\b|"), ("add-host", "host-gateway|")):
            self.assertIn(old, src, k)
            t = tempfile.mkdtemp(prefix="i3m-", dir=CT); p = pathlib.Path(t, "m.py"); p.write_text(src.replace(old, "", 1))
            try: self.assertEqual(self.scan(load(p, "g_m_" + k.replace("-", "_")), "tools/a.sh", self.CASES[k]), [], k)
            finally: shutil.rmtree(t, ignore_errors=True)
        for k, old in (("var-command", "CONTAINER_CMD = re.compile(r\"\\b(?:podman|docker|nerdctl|buildah)\\s+(?:container\\s+)?(?:run|create|exec)\\b|\\$\\{?\\w+\\}?\\s+(?:container\\s+)?(?:run|create|exec)\\b\")"),):
            self.assertIn(old, src)
            t = tempfile.mkdtemp(prefix="i3m-", dir=CT); p = pathlib.Path(t, "m.py")
            p.write_text(src.replace(old, "CONTAINER_CMD = re.compile(r\"\\b(?:podman|docker|nerdctl|buildah)\\s+(?:container\\s+)?(?:run|create|exec)\\b\")", 1))
            try: self.assertEqual(self.scan(load(p, "g_m_var"), "tools/a.sh", self.CASES[k]), [], k)
            finally: shutil.rmtree(t, ignore_errors=True)

class B1CleanTree(unittest.TestCase):
    def test_no_waiver_row_names_a_path_absent_from_this_tree(self):
        g = load(ISO / "guard/assert_host_untouched.py", "g_b1")
        ents, probs = g.load_waivers(str(ROOT))
        self.assertEqual([e["path"] for e in ents if not (ROOT / e["path"]).is_file()], [], "orphan waiver rows must stay out of this PR (and stay fatal)")
        self.assertEqual([e for e in ents if e["path"].startswith("clients/")], [])
    def test_orphan_stays_fatal(self):
        g = load(ISO / "guard/assert_host_untouched.py", "g_b1b"); d = tempfile.mkdtemp(prefix="b1-", dir=CT)
        try:
            pathlib.Path(d, "tools/isolation/guard").mkdir(parents=True)
            pathlib.Path(d, "tools/isolation/guard/waivers.txt").write_text("waive|clients/x/xdg-open|" + "a" * 64 + "|portal-or-dbus|third party|2027-06-30\n")
            self.assertTrue(any(h[2] == "waiver-orphan" for h in g.scan(d)))
        finally: shutil.rmtree(d, ignore_errors=True)

if __name__ == "__main__":
    unittest.main()
