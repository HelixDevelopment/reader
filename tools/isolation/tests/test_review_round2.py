"""Review round 2 (PR #5) regression tests - the CLASS "container-writable state that the host later acts on or follows".
Hermetic: scratch dirs under ~/.cache/claude-tmp only, a PATH stub stands in for podman (NEVER the real one); no container, GUI or display.
Symlink targets are scratch "victim" files, never real host paths. Each fix has a paired mutant: ONE text mutation of the tool in a temp
copy, and the matching test must then fail (the mutant is the vulnerable one)."""
import importlib.util, json, os, pathlib, re, shutil, stat, subprocess, sys, tempfile, textwrap, time, unittest
ROOT = pathlib.Path(__file__).resolve().parents[3]
ISO = ROOT / "tools/isolation"
CT = os.path.expanduser("~/.cache/claude-tmp")

def load(path, name):
    spec = importlib.util.spec_from_file_location(name, str(path)); m = importlib.util.module_from_spec(spec); spec.loader.exec_module(m); return m

R1 = load(ISO / "tests/test_review_round1.py", "rr1_helpers")
mutated_copy, Stub, run_rc = R1.mutated_copy, R1.Stub, R1.run_rc

def py(script, *args, cwd=None, inp=None):
    return subprocess.run([sys.executable, "-I", str(script), *args], capture_output=True, text=True, cwd=cwd, input=inp, timeout=60)

class Scratch(unittest.TestCase):
    def setUp(self):
        self.d = tempfile.mkdtemp(prefix="rr2-", dir=CT)
        self.victim = os.path.join(self.d, "victim.txt"); open(self.victim, "w").write("ORIG")
        self.out = os.path.join(self.d, "out"); os.makedirs(self.out)
    def tearDown(self): shutil.rmtree(self.d, ignore_errors=True)
    def victim_intact(self): return open(self.victim).read() == "ORIG"

class B1SafeIo(Scratch):
    """every host-side access to a container-writable directory goes through safe_io: links are never followed."""
    SIO = ISO / "safe_io.py"
    def test_put_over_symlink_replaces_link_not_target(self):
        os.symlink(self.victim, os.path.join(self.out, "f.txt"))
        r = py(self.SIO, "put", self.out, "f.txt", inp="NEW")
        self.assertEqual(r.returncode, 0, r.stderr); self.assertTrue(self.victim_intact())
        p = os.path.join(self.out, "f.txt"); self.assertFalse(os.path.islink(p)); self.assertEqual(open(p).read(), "NEW")
    def test_put_over_dangling_symlink_and_tmp_name(self):
        os.symlink(self.victim + ".nonexistent", os.path.join(self.out, "g.txt"))
        self.assertEqual(py(self.SIO, "put", self.out, "g.txt", inp="X").returncode, 0); self.assertFalse(os.path.exists(self.victim + ".nonexistent"))
    def test_put_refuses_directory_and_path_components(self):
        os.mkdir(os.path.join(self.out, "d"))
        self.assertEqual(py(self.SIO, "put", self.out, "d", inp="X").returncode, 1)
        for n in ("../x", "a/b", "..", ""): self.assertEqual(py(self.SIO, "put", self.out, n, inp="X").returncode, 1, n)
    def test_copy_refuses_symlink_source_link_dir_hardlink_fifo(self):
        src = os.path.join(self.d, "src"); os.makedirs(src); secret = os.path.join(self.d, "secret.txt"); open(secret, "w").write("SECRET")
        os.symlink(secret, os.path.join(src, "ln.log"))                                  # file symlink
        os.symlink(self.d, os.path.join(src, "dirlink"))                                 # directory symlink in the middle of the path
        os.link(secret, os.path.join(src, "hard.log"))                                   # hard link to a foreign inode
        os.mkfifo(os.path.join(src, "fifo.log"))                                         # would block a plain open()
        open(os.path.join(src, "ok.log"), "w").write("fine")
        for rel in ("ln.log", "dirlink/secret.txt", "hard.log", "fifo.log", "../etc"):
            r = py(self.SIO, "copy", src, rel, self.out, "o.txt"); self.assertEqual(r.returncode, 1, (rel, r.stderr)); self.assertFalse(os.path.exists(os.path.join(self.out, "o.txt")), rel)
        self.assertEqual(py(self.SIO, "copy", src, "ok.log", self.out, "o.txt").returncode, 0); self.assertEqual(open(os.path.join(self.out, "o.txt")).read(), "fine")
    def test_copy_size_cap(self):
        open(os.path.join(self.d, "big"), "wb").write(b"x" * 100)
        self.assertEqual(py(self.SIO, "copy", self.d, "big", self.out, "b", "--max", "10").returncode, 1)
    def test_scan_reports_every_planted_kind(self):
        self.assertEqual(py(self.SIO, "scan", self.out).returncode, 0)
        os.symlink(self.victim, os.path.join(self.out, "s")); os.mkfifo(os.path.join(self.out, "q")); os.link(self.victim, os.path.join(self.out, "h"))
        os.makedirs(os.path.join(self.out, "sub")); os.symlink(self.d, os.path.join(self.out, "sub", "deep"))
        r = py(self.SIO, "scan", self.out); self.assertEqual(r.returncode, 1)
        for k in ("s: symlink", "q: special", "h: hard link", "sub/deep: symlink"): self.assertIn(k, r.stderr)
    def test_first_skips_links_and_bad_names(self):
        b = os.path.join(self.out, "A", "Books"); os.makedirs(b)
        os.symlink(self.victim, os.path.join(b, "a.epub")); open(os.path.join(b, "$(x).epub"), "w").write("x"); open(os.path.join(b, "z.epub"), "w").write("x")
        r = py(self.SIO, "first", self.out, "A/Books", "*.epub"); self.assertEqual((r.returncode, r.stdout.strip()), (0, "z.epub"))
        shutil.rmtree(os.path.join(self.out, "A")); os.makedirs(os.path.join(self.d, "elsewhere", "Books")); open(os.path.join(self.d, "elsewhere", "Books", "q.epub"), "w").write("x")
        os.symlink(os.path.join(self.d, "elsewhere"), os.path.join(self.out, "A"))        # directory in the path swapped for a link
        self.assertEqual(py(self.SIO, "first", self.out, "A/Books", "*.epub").returncode, 1)
    def test_container_proof_never_follows_planted_links(self):
        stub = os.path.join(self.d, "bin"); os.makedirs(stub)
        proof = {"Name": "hrdiso-t", "ImageName": "i", "Created": "c", "HostConfig": {"NetworkMode": "none", "Privileged": False, "CapDrop": ["ALL"], "CapAdd": None,
                 "SecurityOpt": ["no-new-privileges"], "Memory": 1, "MemorySwap": 1, "PidsLimit": 1, "UsernsMode": "keep-id", "Binds": [], "Devices": None, "PortBindings": {}},
                 "Config": {"Timeout": 5, "Env": ["A=1"], "Labels": {}, "Entrypoint": None, "Cmd": ["x"]}}
        open(os.path.join(stub, "podman"), "w").write("#!/bin/sh\ncat <<'EOF'\n" + json.dumps([proof]) + "\nEOF\n"); os.chmod(os.path.join(stub, "podman"), 0o755)
        env = dict(os.environ, PATH=stub + ":" + os.environ["PATH"])
        out = os.path.join(self.out, "container_proof.json")
        os.symlink(self.victim, out); os.symlink(self.victim, out + ".tmp")
        r = subprocess.run([sys.executable, "-I", str(ISO / "container_proof.py"), "hrdiso-t", out], env=env, capture_output=True, text=True, timeout=60)
        self.assertEqual(r.returncode, 0, r.stderr); self.assertTrue(self.victim_intact())
        self.assertFalse(os.path.islink(out)); self.assertEqual(json.load(open(out))["name"], "hrdiso-t")
    def test_seal_refuses_dir_with_planted_link(self):
        open(os.path.join(self.out, "a.txt"), "w").write("a")
        self.assertEqual(subprocess.run([str(ISO / "seal_evidence.sh"), self.out], capture_output=True, text=True).returncode, 0)
        os.unlink(os.path.join(self.out, "SHA256SUMS")); os.symlink(self.victim, os.path.join(self.out, "evil"))
        r = subprocess.run([str(ISO / "seal_evidence.sh"), self.out], capture_output=True, text=True); self.assertEqual(r.returncode, 3, r.stderr)
        self.assertFalse(os.path.exists(os.path.join(self.out, "SHA256SUMS")))
    # ---- paired mutants: each reverts ONE protection and the very same scenario then succeeds in the attack ----
    def test_mutant_put_following_links(self):
        t, f = mutated_copy("safe_io.py", "os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW | os.O_CLOEXEC, mode", "os.O_WRONLY | os.O_CREAT | os.O_TRUNC | os.O_CLOEXEC, mode")
        try:
            f.write_text(f.read_text().replace("os.unlink(name, dir_fd=d)", "pass"))      # plain truncating open, no unlink: the classic `> file`
            os.symlink(self.victim, os.path.join(self.out, "f.txt")); self.assertEqual(py(f, "put", self.out, "f.txt", inp="PWNED").returncode, 0)
            self.assertFalse(self.victim_intact())
        finally: shutil.rmtree(t, ignore_errors=True)
    def test_mutant_read_following_links_and_hardlinks(self):
        secret = os.path.join(self.d, "s.txt"); open(secret, "w").write("SECRET"); os.symlink(secret, os.path.join(self.d, "ln"))
        t, f = mutated_copy("safe_io.py", "os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK", "os.O_RDONLY | os.O_NONBLOCK")
        try:
            self.assertEqual(py(f, "copy", self.d, "ln", self.out, "o").returncode, 0); self.assertEqual(open(os.path.join(self.out, "o")).read(), "SECRET")
        finally: shutil.rmtree(t, ignore_errors=True)
        os.link(secret, os.path.join(self.d, "hl"))
        t, f = mutated_copy("safe_io.py", "if st.st_nlink != 1:\n                raise Refused(f\"{rel}: hard-linked", "if False:\n                raise Refused(f\"{rel}: hard-linked")
        try: self.assertEqual(py(f, "copy", self.d, "hl", self.out, "o2").returncode, 0)
        finally: shutil.rmtree(t, ignore_errors=True)
    def test_mutant_dir_component_followed(self):
        os.makedirs(os.path.join(self.d, "real")); open(os.path.join(self.d, "real", "k"), "w").write("K"); os.symlink(os.path.join(self.d, "real"), os.path.join(self.d, "lk"))
        t, f = mutated_copy("safe_io.py", "os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW | os.O_CLOEXEC, dir_fd=base_fd", "os.O_RDONLY | os.O_DIRECTORY | os.O_CLOEXEC, dir_fd=base_fd")
        try: self.assertEqual(py(f, "copy", self.d, "lk/k", self.out, "o").returncode, 0)
        finally: shutil.rmtree(t, ignore_errors=True)
        self.assertEqual(py(self.SIO, "copy", self.d, "lk/k", self.out, "o3").returncode, 1)
    def test_mutant_scan_and_seal_blind(self):
        t, f = mutated_copy("safe_io.py", "else:\n                bad.append(f\"{prefix}{e}: \" + (\"symlink\"", "elif False:\n                bad.append(f\"{prefix}{e}: \" + (\"symlink\"")
        try:
            os.symlink(self.victim, os.path.join(self.out, "s")); self.assertEqual(py(f, "scan", self.out).returncode, 0)
        finally: shutil.rmtree(t, ignore_errors=True)
        t, f = mutated_copy("seal_evidence.sh", 'python3 -I "$HERE/safe_io.py" "${SCAN[@]}" || {', "true || {")
        try: self.assertEqual(subprocess.run([str(f), self.out], capture_output=True, text=True).returncode, 0)
        finally: shutil.rmtree(t, ignore_errors=True)
    def test_mutant_container_proof_plain_open(self):
        t, f = mutated_copy("container_proof.py", "    safe_io.put(os.path.dirname(os.path.abspath(out)), os.path.basename(out), (json.dumps(proof, indent=2) + \"\\n\").encode())",
                            "    open(out, \"w\").write(json.dumps(proof, indent=2) + \"\\n\")")
        try:
            stub = os.path.join(self.d, "bin"); os.makedirs(stub)
            open(os.path.join(stub, "podman"), "w").write("#!/bin/sh\ncat <<'EOF'\n" + json.dumps([{"Name": "n", "ImageName": "i", "Created": "c", "HostConfig": {"NetworkMode": "none", "Privileged": False, "CapDrop": [], "CapAdd": None, "SecurityOpt": [], "Memory": 1, "MemorySwap": 1, "PidsLimit": 1, "UsernsMode": "", "Binds": [], "PortBindings": {}}, "Config": {"Timeout": 1, "Env": [], "Labels": {}, "Entrypoint": None, "Cmd": []}}]) + "\nEOF\n")
            os.chmod(os.path.join(stub, "podman"), 0o755); out = os.path.join(self.out, "container_proof.json"); os.symlink(self.victim, out)
            subprocess.run([sys.executable, "-I", str(f), "n", out], env=dict(os.environ, PATH=stub + ":" + os.environ["PATH"]), capture_output=True, timeout=60)
            self.assertFalse(self.victim_intact())
        finally: shutil.rmtree(t, ignore_errors=True)

def unsafe_refs(text):
    """Static rule for the host scripts that hand a directory to a container with -w: after the run_container.sh line every reference to a bound
    variable must go through safe_io ($SIO) - a bare cp/tee/sed -i/redirect there follows links the container planted."""
    bound = set(re.findall(r'-w\s+"\$(\w+)[:"]', text)); lines = text.splitlines(); bad = []
    start = next((i for i, l in enumerate(lines) if "run_container.sh" in l), None)
    if start is None or not bound: return bad
    for n, l in enumerate(lines[start + 1:], start + 2):
        if l.lstrip().startswith("#") or "$SIO" in l or "safe_io" in l: continue
        l = re.sub(r'-[wp]\s+"\$\w+[:/][^"]*"', "", l)        # the bind / proof-path arguments of run_container.sh itself (host paths it validates)
        for v in bound:
            if re.search(r"\$\{?%s\}?(?![A-Za-z0-9_])" % v, l): bad.append((n, l.strip()[:100]))
    return bad

class B1HostScriptsEnumerated(unittest.TestCase):
    def test_every_script_with_a_rw_bind_is_clean(self):
        for f in sorted((ROOT / "qa/isolation").glob("run_*.sh")):
            self.assertEqual(unsafe_refs(f.read_text()), [], str(f))
        self.assertTrue(unsafe_refs('"$R/tools/isolation/run_container.sh" -w "$OUT:/out" -- x\ncp "$ST/a" "$OUT/b"\n'))
    def test_mutants_each_old_access_pattern_is_flagged(self):
        base = (ROOT / "qa/isolation/run_koreader_gui.sh").read_text()
        for add in ('cp "$ST/access.log" "$OUT/server_access.log"', 'echo x | tee -a "$OUT/steps.log"', 'sed -i "s/a/b/" "$OUT/redis_sink.txt"', 'cp "$PR/A/ko.log" "$OUT/ko.log"',
                    'BOOK=$(ls "$PR/A/Books"/*.epub | head -1)', 'echo x > "$OUT/f"'):
            self.assertTrue(unsafe_refs(base + "\n" + add + "\n"), add)
    def test_koreader_uses_host_only_log_dir_and_scans_before_done(self):
        t = (ROOT / "qa/isolation/run_koreader_gui.sh").read_text()
        self.assertIn("LOG=$HW/steps.log", t); self.assertIn('$SIO scan "$OUT"', t); self.assertIn("podman rm -f $N", t)
        self.assertLess(t.index("podman rm -f $N"), t.index('cp "$ST/access.log" "$HW/'))

class B2BindRules(Scratch):
    """rule table for ISO_WORK / rw sources: no git checkout/worktree/pointer file, no .claude*/.specify/.husky/.githooks, no symlink."""
    def setUp(self):
        super().setUp(); self.b = load(ISO / "bindcheck.py", "bc_r2"); self.repo = os.path.join(self.d, "repo")
        for x in ("tools/isolation", "qa", "evidence"): os.makedirs(os.path.join(self.repo, x))
        self.w = os.path.join(self.d, "w"); os.makedirs(self.w)
    def chk(self, iso, src, mode="rw", b=None): return (b or self.b).check(mode, src, "/work/t", self.repo, {"ISO_WORK": iso})
    def test_iso_work_that_is_another_checkout_or_inside_one(self):
        co = os.path.join(self.d, "co"); os.makedirs(os.path.join(co, ".git")); os.makedirs(os.path.join(co, "run"))
        self.assertIsNotNone(self.chk(co, co + "/run")); self.assertIsNotNone(self.chk(co + "/run", co + "/run"))
        wt = os.path.join(self.d, "wt"); os.makedirs(os.path.join(wt, "run")); open(os.path.join(wt, ".git"), "w").write("gitdir: /x\n")   # worktree pointer FILE
        self.assertIsNotNone(self.chk(wt, wt + "/run")); self.assertIsNotNone(self.chk(wt + "/run", wt + "/run"))
        os.makedirs(self.w + "/run1"); self.assertIsNone(self.chk(self.w, self.w + "/run1"))                                 # golden: a plain dedicated dir
    def test_iso_work_holding_agent_or_hook_dirs(self):
        for n in (".claude", ".claude-x", ".specify", ".husky", ".githooks"):
            d = os.path.join(self.d, "iso" + n.replace(".", "_")); os.makedirs(os.path.join(d, n)); os.makedirs(os.path.join(d, "run"))
            self.assertIsNotNone(self.chk(d, d + "/run"), n)
    def test_rw_source_tree_rules(self):
        os.makedirs(self.w + "/ok"); self.assertIsNone(self.chk(self.w, self.w + "/ok"))
        cases = {"nested_git_dir": (".git", "d"), "nested_git_pointer": (".git", "f"), "claude_cfg": (".claude/settings.json", "f"), "specify": (".specify/x", "f"),
                 "husky": (".husky/pre-commit", "f"), "githooks": (".githooks/h", "f"), "symlink": ("lnk", "l")}
        for k, (rel, kind) in cases.items():
            s = os.path.join(self.w, "r_" + k); os.makedirs(os.path.join(s, "deep", "er"))
            p = os.path.join(s, "deep", "er", rel); os.makedirs(os.path.dirname(p), exist_ok=True)
            if kind == "d": os.makedirs(p)
            elif kind == "f": open(p, "w").write("x")
            else: os.symlink(self.victim, p)
            self.assertIsNotNone(self.chk(self.w, s), k)
    def test_rw_source_below_a_checkout(self):
        s = os.path.join(self.w, "a", "b"); os.makedirs(s); os.makedirs(os.path.join(self.w, "a", ".git"))         # ancestor holds .git, ISO_WORK itself is clean
        self.assertIsNotNone(self.chk(self.w, s))
    def test_ro_binds_are_not_affected(self):
        s = os.path.join(self.w, "r"); os.makedirs(os.path.join(s, ".git")); open(os.path.join(s, "f"), "w").write("x")
        self.assertIsNone(self.chk(self.w, s, "ro"))
    def test_evidence_fresh_dir_still_ok_and_hostile_content_there_refused(self):
        e = os.path.join(self.repo, "evidence", "run1"); os.makedirs(e); self.assertIsNone(self.b.check("rw", e, "/out", self.repo, {}))
        os.symlink(self.victim, os.path.join(e, "planted")); self.assertIsNotNone(self.b.check("rw", e, "/out", self.repo, {}))
    def test_cli_exit_codes(self):
        co = os.path.join(self.d, "co"); os.makedirs(os.path.join(co, ".git")); os.makedirs(os.path.join(co, "run"))
        r = subprocess.run([sys.executable, "-I", str(ISO / "bindcheck.py"), "rw", co + "/run", "/work/t"], env=dict(os.environ, ISO_WORK=co), capture_output=True, text=True)
        self.assertEqual(r.returncode, 64, r.stderr)
    # ---- paired mutants ----
    def mut(self, old, new, also=None):
        t, f = mutated_copy("bindcheck.py", old, new)
        if also:   # a second text mutation in the same copy (a rule backed by a defence-in-depth sibling)
            f = pathlib.Path(f); f.write_text(f.read_text().replace(also[0], also[1]))
        self.addCleanup(shutil.rmtree, t, True); return load(f, "bcm" + str(abs(hash(old))))
    def test_mutants_each_rule_is_load_bearing(self):
        co = os.path.join(self.d, "co"); os.makedirs(os.path.join(co, ".git")); os.makedirs(os.path.join(co, "run"))
        os.makedirs(co + "/run/sub"); self.assertIsNotNone(self.chk(co + "/run", co + "/run/sub", "ro"))      # ancestor rule alone refuses this ro bind
        self.assertIsNone(self.chk(co + "/run", co + "/run/sub", "ro", b=self.mut("c = checkout_ancestor(iso)\n    if c:", "c = None\n    if c:")))
        d = os.path.join(self.d, "iso_c"); os.makedirs(os.path.join(d, ".claude")); os.makedirs(os.path.join(d, "run"))
        self.assertIsNone(self.chk(d, d + "/run", b=self.mut("top = [n for n in os.listdir(iso) if hostile_name(n)]", "top = []")))
        s = os.path.join(self.w, "nested"); os.makedirs(s + "/x/.git")
        self.assertIsNone(self.chk(self.w, s, b=self.mut("if hostile_name(name):", "if False:")))
        s2 = os.path.join(self.w, "lnkdir"); os.makedirs(s2); os.symlink(self.victim, s2 + "/l")
        self.assertIsNone(self.chk(self.w, s2, b=self.mut("if stat.S_ISLNK(st.st_mode):", "if False:", also=("if not (stat.S_ISDIR(st.st_mode) or stat.S_ISREG(st.st_mode)):", "if False:"))))
        s3 = os.path.join(self.w, "a", "b"); os.makedirs(s3); os.makedirs(os.path.join(self.w, "a", ".git"))
        self.assertIsNone(self.chk(self.w, s3, b=self.mut("c = None if inside(real, ev) else checkout_ancestor(real)", "c = None")))
        self.assertIsNone(self.chk(self.w, s, b=self.mut('h = hostile_tree_error(real, os.path.join(real, "h.sock") if bridge and dst == "/run/helixsock" else None) if os.path.isdir(real) else None', "h = None")))

class I1Ranges(unittest.TestCase):
    def rc(self, args, script=None):
        with Stub() as st: return run_rc(script or ISO / "run_container.sh", ["-n", "hrdiso-r2"] + args + ["--", "true"], st).returncode
    def test_refused(self):
        for a in (["-m", "0"], ["-m", "0m"], ["-m", "0k"], ["-m", "63m"], ["-m", "17g"], ["-m", "1000000000"], ["-m", "0100m"], ["-m", "1.5g"], ["-m", "5k"],
                  ["-c", "0"], ["-c", "0."], ["-c", "."], ["-c", "1.2.3"], ["-c", ".5"], ["-c", "0.05"], ["-c", "9"], ["-c", "100"],
                  ["-t", "0"], ["-t", "14401"], ["-t", "007"], ["-t", "99999999"]):
            self.assertEqual(self.rc(a), 64, a)
    def test_golden_good(self):
        for a in (["-m", "64m"], ["-m", "16g"], ["-m", "2g"], ["-m", "512M"], ["-c", "0.10"], ["-c", "0.5"], ["-c", "1.50"], ["-c", "2"], ["-c", "8"], ["-t", "1"], ["-t", "14400"]):
            self.assertEqual(self.rc(a), 0, a)
    def mutant(self, old, new, args):
        t, f = mutated_copy("run_container.sh", old, new)
        try: return self.rc(args, f) == 0
        finally: shutil.rmtree(t, ignore_errors=True)
    def test_mutants_each_check_is_load_bearing(self):
        self.assertTrue(self.mutant('{ [ "$mem_mib" -ge 64 ] && [ "$mem_mib" -le 16384 ]; }', "true", ["-m", "63m"]))
        self.assertTrue(self.mutant('{ [ "$mem_mib" -ge 64 ] && [ "$mem_mib" -le 16384 ]; }', "true", ["-m", "17g"]))
        self.assertTrue(self.mutant("^([1-9][0-9]{0,5})([mMgG])$", "^([0-9]{1,6})([mMgG])$", ["-m", "0100m"]))
        self.assertTrue(self.mutant('{ [ "$cpu_centi" -ge 10 ] && [ "$cpu_centi" -le 800 ]; }', "true", ["-c", "0.05"]))
        self.assertTrue(self.mutant('{ [ "$cpu_centi" -ge 10 ] && [ "$cpu_centi" -le 800 ]; }', "true", ["-c", "9"]))
        self.assertTrue(self.mutant("(\\.([0-9]{1,2}))?$ ]]", "(\\.([0-9]{1,2}))?(\\.[0-9]+)?$ ]]", ["-c", "1.2.3"]))
        self.assertTrue(self.mutant('[[ $LIFE =~ ^[1-9][0-9]{0,4}$ ]] && [ "$LIFE" -le 14400 ]', '[ "$LIFE" -le 14400 ]', ["-t", "0"]))
        self.assertTrue(self.mutant('[[ $LIFE =~ ^[1-9][0-9]{0,4}$ ]] && [ "$LIFE" -le 14400 ]', "true", ["-t", "14401"]))
    def test_iceauthority_refused_and_mutant(self):
        self.assertEqual(self.rc(["-e", "ICEAUTHORITY=/x"]), 64)
        self.assertTrue(self.mutant("ICEAUTHORITY*|", "", ["-e", "ICEAUTHORITY=/x"]))

class I3Hardening(unittest.TestCase):
    """the argv handed to podman carries every isolation flag; the in-container launcher refuses a foreign DISPLAY; the reaper honours its grace."""
    NEED = ("--network none", "--cap-drop ALL", "--security-opt no-new-privileges", "--memory-swap", "--pids-limit", "--userns=keep-id", "--rm", "--label org.helix.isolation=gui-test")
    def argv(self, script=None):
        with Stub() as st:
            run_rc(script or ISO / "run_container.sh", ["-n", "hrdiso-h", "--", "true"], st)
            return " ".join(l for l in st.log.read_text().splitlines() if l.startswith("run "))
    def test_all_flags_present(self):
        a = self.argv()
        for n in self.NEED: self.assertIn(n, a)
        self.assertNotIn("--privileged", a); self.assertNotIn("--cap-add", a); self.assertRegex(a, r"--memory (\S+) --memory-swap \1\b")
    def test_mutants_dropping_a_flag_are_detected(self):
        for old in ("--cap-drop ALL ", "--security-opt no-new-privileges ", "\"$NAME\" --network none ", "--pids-limit 512 ", "--memory-swap \"$MEM\" "):
            t, f = mutated_copy("run_container.sh", old, '"$NAME" ' if "NAME" in old else "")
            try:
                a = self.argv(f); self.assertTrue(any(n not in a for n in self.NEED) or not re.search(r"--memory (\S+) --memory-swap \1\b", a), old)
            finally: shutil.rmtree(t, ignore_errors=True)
    LAUNCH = ROOT / "qa/isolation/inside/launch.sh"
    def launch(self, env, f=None):
        e = {"PATH": os.environ["PATH"]}; e.update(env)
        return subprocess.run(["sh", str(f or self.LAUNCH), "--", "true"], env=e, capture_output=True, text=True, timeout=60)
    def test_launch_display_check(self):
        r = self.launch({"DISPLAY": ":0"}); self.assertEqual(r.returncode, 65); self.assertIn("DISPLAY is not the container-private :99", r.stderr)
        r = self.launch({"DISPLAY": "localhost:10.0"}); self.assertIn("DISPLAY is not the container-private :99", r.stderr)
        self.assertNotIn("DISPLAY is not", self.launch({"DISPLAY": ":99"}).stderr)                 # the container-private value is the only one accepted
        r = self.launch({"WAYLAND_DISPLAY": "wayland-0"}); self.assertEqual(r.returncode, 65); self.assertIn("WAYLAND_DISPLAY is set", r.stderr)
        r = self.launch({"SSH_AUTH_SOCK": "/x"}); self.assertIn("SSH_AUTH_SOCK is set", r.stderr)
    def test_launch_mutants(self):
        d = tempfile.mkdtemp(prefix="rr2l-", dir=CT)
        try:
            s = self.LAUNCH.read_text(); old = '[ -z "${DISPLAY:-}" ] || [ "$DISPLAY" = ":99" ] || bad "DISPLAY is not the container-private :99"'
            self.assertIn(old, s); p = pathlib.Path(d, "launch.sh"); p.write_text(s.replace(old, "true"))
            self.assertNotIn("DISPLAY is not", self.launch({"DISPLAY": ":0"}, p).stderr)
            old2 = '[ -z "$val" ] || bad "$v is set"'; self.assertIn(old2, s); p.write_text(s.replace(old2, "true"))
            self.assertNotIn("is set", self.launch({"WAYLAND_DISPLAY": "w"}, p).stderr)
        finally: shutil.rmtree(d, ignore_errors=True)
    def reaped(self, script, lease, grace="120", pid=None):
        with Stub() as st:
            d = subprocess.Popen(["true"]); d.wait()
            pathlib.Path(st.d, "ps").write_text("hrdiso-a\n"); pathlib.Path(st.d, "hrdiso-a.pid").write_text(str(pid or d.pid))
            if lease is not None: pathlib.Path(st.d, "hrdiso-a.lease").write_text(str(lease))
            subprocess.run([str(script)], capture_output=True, text=True, env=dict(st.env, REAP_GRACE=grace), timeout=60)
            return "rm -f hrdiso-a" in st.log.read_text()
    def test_reaper_startup_grace(self):
        now = int(time.time())
        self.assertFalse(self.reaped(ISO / "reap.sh", now))                  # dead owner but younger than the grace: its starter may not be done
        self.assertFalse(self.reaped(ISO / "reap.sh", now - 30))
        self.assertTrue(self.reaped(ISO / "reap.sh", now - 400))             # older than the grace and owner dead: reaped
        self.assertFalse(self.reaped(ISO / "reap.sh", None))                 # unknown age: fail safe
    def test_reaper_grace_mutants(self):
        now = int(time.time())
        t, f = mutated_copy("reap.sh", '[ $((now - lease)) -lt "$GRACE" ] && continue', "true")
        try: self.assertTrue(self.reaped(f, now))
        finally: shutil.rmtree(t, ignore_errors=True)
        t, f = mutated_copy("reap.sh", "case $lease in ''|*[!0-9]*) continue ;; esac", "lease=0")
        try: self.assertTrue(self.reaped(f, None))
        finally: shutil.rmtree(t, ignore_errors=True)

class I2LintClass(unittest.TestCase):
    G = ISO / "guard/assert_host_untouched.py"
    def scan(self, g, name, body):
        d = tempfile.mkdtemp(prefix="i2-", dir=CT)
        try:
            p = pathlib.Path(d, name); p.parent.mkdir(parents=True, exist_ok=True); p.write_text(body); return g.scan(d)
        finally: shutil.rmtree(d, ignore_errors=True)
    BAD = {"env-host": "podman run --rm --env-host img sh\n", "rootfs": "podman run --rm --rootfs /h sh\n", "cwd-mount": "podman run --rm -v .:/h img sh\n",
           "pwd-mount": 'podman run --rm -v "$PWD":/h img sh\n', "pod": "podman run --rm --pod p img sh\n", "devcg": "podman run --rm --device-cgroup-rule='a *:* rwm' img sh\n",
           "sysctl": "podman run --rm --sysctl net.x=1 img sh\n", "cp-home": 'podman cp c:/etc/passwd "$HOME/.bashrc"\n', "docker-create": "docker create img sh\n",
           "nerdctl-exec": "nerdctl exec c sh\n", "var-cp": "P=podman; $P cp c:/a /h\n", "py-list": 'import subprocess\nsubprocess.run(["podman", "run", "img"])\n'}
    def test_any_runtime_call_outside_isolation_dir_is_a_finding(self):
        g = R1.load(self.G, "g_i2")
        for k, body in self.BAD.items():
            name = "tools/a.py" if k == "py-list" else "qa/a.sh"; hits = self.scan(g, name, body)
            self.assertTrue(any(h[2] in ("container-outside-isolation", "container-escape") for h in hits), k)
    def test_inside_isolation_dir_is_exempt_and_waiver_is_pinned(self):
        g = R1.load(self.G, "g_i2b")
        self.assertEqual([h[2] for h in self.scan(g, "tools/isolation/x.sh", "podman run --rm --network none img sh\n")], ["container-outside-isolation"])   # round 3: no blanket exemption - only a pinned row exempts
        self.assertEqual(self.scan(g, "tools/a.sh", "echo podman is a tool; podman ps\n"), [])           # listing / non-run verbs are not the class
        d = tempfile.mkdtemp(prefix="i2w-", dir=CT)
        try:
            body = "podman exec c redis-cli ping\n"; pathlib.Path(d, "qa").mkdir(); pathlib.Path(d, "qa/a.sh").write_text(body)
            self.assertTrue(g.scan(d))
            subprocess.run([sys.executable, "-I", str(ISO / "guard/pin_waivers.py"), "--root", d, "add", "waive", "qa/a.sh", "container-outside-isolation", "test", "2027-06-30"], capture_output=True, timeout=60)
            pathlib.Path(d, "qa/a.sh").write_text(body + "podman run --rm --env-host i s\n")
            self.assertTrue(g.scan(d))                                                                    # content changed after the pin: void
        finally: shutil.rmtree(d, ignore_errors=True)
    def test_mutant_without_class_rule_misses_the_odd_flags(self):
        src = self.G.read_text(); old = 'if kind == "code" and any(CONTAINER_USE.search(b) for b in cviews):'
        self.assertIn(old, src); t = tempfile.mkdtemp(prefix="i2m-", dir=CT)
        try:
            p = pathlib.Path(t, "m.py"); p.write_text(src.replace(old, "if False:"))
            gm = R1.load(p, "g_i2m"); g = R1.load(self.G, "g_i2ok")
            missed = [k for k in ("env-host", "rootfs", "pod", "devcg", "sysctl", "cp-home", "docker-create", "nerdctl-exec", "var-cp") if not self.scan(gm, "qa/a.sh", self.BAD[k])]
            self.assertGreaterEqual(len(missed), 2, missed)                                              # the per-flag list alone cannot be complete
            for k in missed: self.assertTrue(self.scan(g, "qa/a.sh", self.BAD[k]), k)
        finally: shutil.rmtree(t, ignore_errors=True)

if __name__ == "__main__":
    unittest.main()
