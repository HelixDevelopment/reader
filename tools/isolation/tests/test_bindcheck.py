"""F3 gate: bind-source allow-list."""
import os, pathlib, subprocess, sys, tempfile, time, unittest
HERE = pathlib.Path(__file__).resolve().parent
ROOT = HERE.parents[2]
sys.path.insert(0, str(ROOT / "tools/isolation"))
import importlib.util
_p = os.environ.get("BINDCHECK_UNDER_TEST", str(ROOT / "tools/isolation/bindcheck.py"))      # RED runs point this at the round-2 module
_spec = importlib.util.spec_from_file_location("bindcheck_ut", _p); b = importlib.util.module_from_spec(_spec); _spec.loader.exec_module(b)
CT = os.path.expanduser("~/.cache/claude-tmp")      # dedicated scratch root

class Bind(unittest.TestCase):
    def chk(self, mode, src, dst="/work/x", env=None):
        return b.check(mode, src, dst, str(ROOT), env or {})
    def test_refused_sources(self):
        for s in (os.path.expanduser("~"), "/", "/tmp", "/dev/shm", "/var/tmp", "/etc", "/home", os.path.expanduser("~/.ssh"),
                  os.path.expanduser("~/Projects"), str(ROOT), str(ROOT / "server"), str(ROOT / ".git")):
            with self.subTest(src=s):
                self.assertIsNotNone(self.chk("ro", s), s)
                self.assertIsNotNone(self.chk("rw", s), s)
    def test_allowed_ro(self):
        for s in (ROOT / "clients/web", ROOT / "qa/isolation/inside", ROOT / "qa/helixqa/banks", ROOT / "clients/desktop/e2e"):
            with self.subTest(src=str(s)):
                self.assertIsNone(self.chk("ro", str(s)))
    def test_playwright_and_koreader_ro_only(self):
        pw = os.path.expanduser("~/.cache/ms-playwright")
        self.assertIsNone(self.chk("ro", pw, "/ms-playwright")); self.assertIsNotNone(self.chk("rw", pw, "/ms-playwright"))
        self.assertIsNone(self.chk("ro", "/usr/lib/koreader", "/opt/koreader")); self.assertIsNotNone(self.chk("rw", "/usr/lib/koreader", "/opt/koreader"))
    def test_rw_only_under_evidence_or_iso_work(self):
        self.assertIsNotNone(self.chk("rw", str(ROOT / "clients/web")))
        self.assertIsNotNone(self.chk("rw", str(ROOT / "evidence")))          # root itself
        fresh = pathlib.Path(tempfile.mkdtemp(prefix="n5fresh-", dir=str(ROOT / "evidence")))
        try:
            self.assertIsNone(self.chk("rw", str(fresh)))
        finally:
            fresh.rmdir()
        with tempfile.TemporaryDirectory(dir=CT) as d:
            self.assertIsNone(self.chk("rw", d + "/run1", "/prof", {"ISO_WORK": d}))
            self.assertIsNotNone(self.chk("rw", d, "/prof", {"ISO_WORK": d}))      # the root itself: per-run subdir only
            self.assertIsNotNone(self.chk("rw", d + "/run1", "/prof", {}))        # not allowed without ISO_WORK
    def test_broad_iso_work_refused(self):
        for bad in ("/", "/tmp", os.path.expanduser("~")):
            self.assertIsNotNone(self.chk("rw", str(ROOT / "evidence/x"), "/out", {"ISO_WORK": bad}))
    def test_symlink_escape_refused(self):
        with tempfile.TemporaryDirectory(dir=CT) as d:
            link = pathlib.Path(d, "lnk"); link.symlink_to(os.path.expanduser("~"))
            self.assertIsNotNone(self.chk("ro", str(link), "/work/x", {"ISO_WORK": d}))
    def test_bad_destination(self):
        self.assertIsNotNone(self.chk("ro", str(ROOT / "clients/web"), "/etc"))
        self.assertIsNotNone(self.chk("ro", str(ROOT / "clients/web"), "/usr/bin"))
    def test_wrapper_end_to_end_refuses(self):
        r = subprocess.run([str(ROOT / "tools/isolation/run_container.sh"), "-n", "hrdiso-bindtest", "-r", f"{os.path.expanduser('~')}:/work/h", "--", "ls"],
                           capture_output=True, text=True)
        self.assertEqual(r.returncode, 64); self.assertIn("bind refused", r.stderr)
    def test_wrapper_refuses_bad_name_and_display_env(self):
        base = [str(ROOT / "tools/isolation/run_container.sh")]
        self.assertEqual(subprocess.run(base + ["-n", "evil", "--", "true"], capture_output=True).returncode, 64)
        env = "WAYLAND" + "_DISPLAY=x"
        self.assertEqual(subprocess.run(base + ["-n", "hrdiso-x", "-e", env, "--", "true"], capture_output=True).returncode, 64)

class N5(unittest.TestCase):
    """N5 hardening: destination normalisation, overmount ban, ISO_WORK constraints, fresh-only evidence rw, resolved path to podman."""
    def chk(self, mode, src, dst="/work/x", env=None, **kw):
        return b.check(mode, src, dst, str(ROOT), env if env is not None else {}, **kw)
    WEB = str(ROOT / "clients/web")
    def test_non_canonical_destinations_refused(self):
        for d in ("/work/../etc", "/work/x/../../etc", "/work//x", "/work/./x", "/work/x/", "work/x", "/work/x:rw", "/work/x,suid", "/work/a b", "/work/..", "/opt/../etc"):
            with self.subTest(dst=d):
                self.assertIsNotNone(self.chk("ro", self.WEB, d), d)
    def test_overmount_of_reserved_paths_refused(self):
        for d in ("/opt/isoinside", "/opt/isoinside/launch.sh", "/opt/isoinside/x/y", "/run/helixsock", "/run/helixsock/h.sock", "/opt", "/run", "/"):
            with self.subTest(dst=d):
                self.assertIsNotNone(self.chk("ro", self.WEB, d), d)
    def test_bridge_may_use_its_own_reserved_mount_only(self):
        with tempfile.TemporaryDirectory(dir=CT) as d:
            sock = d + "/sock"; os.mkdir(sock)
            self.assertIsNone(self.chk("rw", sock, "/run/helixsock", {"ISO_WORK": d}, bridge=True))
            self.assertIsNotNone(self.chk("rw", sock, "/run/helixsock", {"ISO_WORK": d}))
            self.assertIsNotNone(self.chk("rw", sock, "/opt/isoinside", {"ISO_WORK": d}, bridge=True))
    def test_bridge_dir_may_hold_its_one_real_unix_socket_only(self):
        # HRD-076: a real run has <dir>/h.sock (a unix socket); the bridge bind must accept exactly that and nothing else
        import socket
        with tempfile.TemporaryDirectory(dir=CT) as d:
            sock = d + "/sock"; os.mkdir(sock)
            s = socket.socket(socket.AF_UNIX); s.bind(sock + "/h.sock")
            try:
                self.assertIsNone(self.chk("rw", sock, "/run/helixsock", {"ISO_WORK": d}, bridge=True))
                self.assertIsNotNone(self.chk("rw", sock, "/run/helixsock", {"ISO_WORK": d}), "socket refused without --bridge")
                s2 = socket.socket(socket.AF_UNIX); s2.bind(sock + "/other.sock")
                self.assertIsNotNone(self.chk("rw", sock, "/run/helixsock", {"ISO_WORK": d}, bridge=True), "a second socket is refused")
                s2.close()
            finally:
                s.close()
    def test_bridge_exemption_needs_a_real_socket_not_a_fifo_or_hardlink_named_h_sock(self):
        # HRD-076 review I-3: the exemption is for a SOCKET named h.sock only; a fifo or a hard-linked regular file with that name stays refused
        with tempfile.TemporaryDirectory(dir=CT) as d:
            sock = d + "/sock"; os.mkdir(sock)
            os.mkfifo(sock + "/h.sock")
            self.assertIsNotNone(self.chk("rw", sock, "/run/helixsock", {"ISO_WORK": d}, bridge=True), "fifo named h.sock must be refused")
            os.unlink(sock + "/h.sock")
            with open(d + "/orig", "w") as f: f.write("x")
            os.link(d + "/orig", sock + "/h.sock")
            self.assertIsNotNone(self.chk("rw", sock, "/run/helixsock", {"ISO_WORK": d}, bridge=True), "hard-linked file named h.sock must be refused")
    def test_bridge_exemption_is_the_exact_top_level_path_not_any_file_named_h_sock(self):
        # HRD-076 review I-5: a socket named h.sock in a SUBDIRECTORY of the bridge dir is refused, with and without the top-level h.sock
        import socket
        with tempfile.TemporaryDirectory(dir=CT) as d:
            sock = d + "/sock"; os.mkdir(sock); os.mkdir(sock + "/sub")
            inner = socket.socket(socket.AF_UNIX); inner.bind(sock + "/sub/h.sock")
            try:
                self.assertIsNotNone(self.chk("rw", sock, "/run/helixsock", {"ISO_WORK": d}, bridge=True), "nested h.sock alone must be refused")
                outer = socket.socket(socket.AF_UNIX); outer.bind(sock + "/h.sock")
                try:
                    self.assertIsNotNone(self.chk("rw", sock, "/run/helixsock", {"ISO_WORK": d}, bridge=True), "nested h.sock beside the top-level one must be refused")
                finally:
                    outer.close()
            finally:
                inner.close()
    def test_bridge_exemption_applies_to_the_helixsock_destination_only(self):
        # HRD-076 review M2: a real h.sock is accepted for dst /run/helixsock only, never for another destination even with --bridge
        import socket
        with tempfile.TemporaryDirectory(dir=CT) as d:
            sock = d + "/sock"; os.mkdir(sock)
            s = socket.socket(socket.AF_UNIX); s.bind(sock + "/h.sock")
            try:
                self.assertIsNone(self.chk("rw", sock, "/run/helixsock", {"ISO_WORK": d}, bridge=True))
                self.assertIsNotNone(self.chk("rw", sock, "/opt/other", {"ISO_WORK": d}, bridge=True))
            finally:
                s.close()
    def test_iso_work_must_be_absolute_canonical_dedicated_and_not_sensitive(self):
        home = os.path.expanduser("~")
        bad = ["relative/dir", "~/.cache/claude-tmp/x", CT, CT + "/../../.ssh", CT + "/a/../b", CT + "//x", home + "/.ssh", home + "/.config/x", home + "/.gnupg",
               home + "/.cache", home + "/Projects", str(ROOT / "evidence"), "/var/tmp/x", "/tmp/x"]
        for iso in bad:
            with self.subTest(iso=iso):
                self.assertIsNotNone(self.chk("ro", self.WEB, "/work/x", {"ISO_WORK": iso}), iso)
        with tempfile.TemporaryDirectory(dir=CT) as d:
            self.assertIsNone(self.chk("ro", self.WEB, "/work/x", {"ISO_WORK": d}))
            link = d + "-lnk"; os.symlink(home + "/.ssh", link)
            try:
                self.assertIsNotNone(self.chk("ro", self.WEB, "/work/x", {"ISO_WORK": link}))        # symlink in the ISO_WORK chain
            finally:
                os.unlink(link)
    def test_sensitive_dirs_and_their_ancestors_refused_as_sources(self):
        home = os.path.expanduser("~")
        for s in (home + "/.ssh", home + "/.config", home + "/.gnupg", home + "/.config/x", home + "/.local", home + "/.claude-claude4", home + "/.cache"):
            with self.subTest(src=s):
                self.assertIsNotNone(self.chk("ro", s, "/work/x", {"ISO_WORK": CT + "/iso-work"}), s)
    def test_source_with_option_injection_chars_refused(self):
        with tempfile.TemporaryDirectory(dir=CT) as d:
            for name in ("a:rw", "a,b"):
                p = pathlib.Path(d, name); p.mkdir()
                with self.subTest(name=name):
                    self.assertIsNotNone(self.chk("ro", str(p), "/work/x", {"ISO_WORK": d}))
    def test_rw_evidence_only_fresh_unsealed_dirs(self):
        ev = ROOT / "evidence"
        sealed = pathlib.Path(tempfile.mkdtemp(prefix="n5sealed-", dir=str(ev)))
        fresh = pathlib.Path(tempfile.mkdtemp(prefix="n5fresh-", dir=str(ev)))
        try:
            (sealed / "SHA256SUMS").write_text("x\n"); (sealed / "a.png").write_text("x")
            self.assertIsNotNone(self.chk("rw", str(sealed)), "sealed dir writable")
            parent = pathlib.Path(tempfile.mkdtemp(prefix="n5parent-", dir=str(ev)))
            try:
                import shutil
                shutil.copytree(sealed, parent / "web")          # a sealed SIBLING below the bound dir
                self.assertIsNotNone(self.chk("rw", str(parent)), "ancestor of a sealed dir writable")
            finally:
                import shutil as sh2; sh2.rmtree(parent, ignore_errors=True)
            (fresh / "new.txt").write_text("x")
            self.assertIsNone(self.chk("rw", str(fresh)))
            old = time.time() - 30 * 86400; os.utime(fresh / "new.txt", (old, old))
            self.assertIsNotNone(self.chk("rw", str(fresh)), "stale content accepted as fresh")
            self.assertIsNotNone(self.chk("rw", str(ev / "does-not-exist-n5")))
        finally:
            import shutil as sh3; sh3.rmtree(sealed, ignore_errors=True); sh3.rmtree(fresh, ignore_errors=True)
    def test_cli_prints_resolved_path_and_wrapper_passes_it_to_podman(self):
        with tempfile.TemporaryDirectory(dir=CT) as d:
            real = pathlib.Path(d, "real"); real.mkdir(); link = pathlib.Path(d, "lnk"); link.symlink_to(real)
            r = subprocess.run([sys.executable, "-I", str(ROOT / "tools/isolation/bindcheck.py"), "ro", str(link), "/work/x"], capture_output=True, text=True,
                               env=dict(os.environ, ISO_WORK=d))
            self.assertEqual(r.returncode, 0, r.stderr); self.assertEqual(r.stdout.strip(), os.path.realpath(real))
        src = (ROOT / "tools/isolation/run_container.sh").read_text()
        self.assertIn('-v "$real:$d:ro"', src); self.assertIn('-v "$real:$d:rw"', src); self.assertNotIn('-v "$s:$d', src)
    def test_wrapper_refuses_duplicate_and_overmount_destinations(self):
        base = [str(ROOT / "tools/isolation/run_container.sh"), "-n", "hrdiso-n5dup"]
        web = str(ROOT / "clients/web")
        r = subprocess.run(base + ["-r", f"{web}:/work/a", "-r", f"{web}:/work/a", "--", "true"], capture_output=True, text=True)
        self.assertEqual(r.returncode, 64); self.assertIn("duplicate destination", r.stderr)
        r = subprocess.run(base + ["-r", f"{web}:/opt/isoinside", "--", "true"], capture_output=True, text=True)
        self.assertEqual(r.returncode, 64); self.assertIn("bind refused", r.stderr)

if __name__ == "__main__":
    unittest.main()
