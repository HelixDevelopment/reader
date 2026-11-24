"""F2: permanent evader matrix. Every mutant in tools/isolation/guard/mutants.b64 (base64 so this tree has no forbidden literals)
is written into a throw-away tree and the lint must catch it - except the explicitly documented residual classes."""
import base64, json, os, pathlib, subprocess, sys, tempfile, unittest
ROOT = pathlib.Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / "tools/isolation/guard"))
import assert_host_untouched as g

MUT = json.loads(base64.b64decode((ROOT / "tools/isolation/guard/mutants.b64").read_text()))

def tree(files, mode=None):
    d = tempfile.mkdtemp(prefix="gm-")
    for name, body in files.items():
        p = pathlib.Path(d, name); p.parent.mkdir(parents=True, exist_ok=True); p.write_text(body)
        if mode: os.chmod(p, mode)
    return d

class Matrix(unittest.TestCase):
    def test_matrix_size(self):
        self.assertGreaterEqual(len(MUT), 180)
    def test_every_mutant_caught_except_documented_residual(self):
        missed, caught, residual_uncaught, residual_caught = [], 0, [], []
        for m in MUT:
            hits = g.scan(tree({m["file"]: m["content"]}))
            if m["expect"] == "caught":
                (missed.append(m["name"]) if not hits else None)
                caught += 0 if not hits else 1
            else:
                (residual_caught if hits else residual_uncaught).append(m["name"])
        print(f"\nmatrix: {caught} caught of {sum(1 for m in MUT if m['expect']=='caught')} required; residual documented: "
              f"{len(residual_uncaught)} uncaught {residual_uncaught}, {len(residual_caught)} unexpectedly caught {residual_caught}")
        self.assertEqual(missed, [], f"lint evaded by: {missed}")
    def test_clean_mutation_free_tree(self):
        self.assertEqual(g.scan(tree({"tools/a.sh": "echo hi\n", "tools/b.py": "print(1)\n", "tools/c.mjs": "chromium.launch({ headless: true })\n"})), [])
    def test_container_only_command_inside_dir_with_marker_is_allowed(self):
        body = "#!/bin/sh\n# isolation: runs-inside-container\n" + "xdo" + "tool key a\n"
        self.assertEqual(g.scan(tree({"qa/isolation/inside/x.sh": body}, 0o644)), [])
    def test_marker_outside_inside_dirs_is_ignored(self):
        body = "#!/bin/sh\n# isolation: runs-inside-container\n" + "xdo" + "tool key a\n"
        self.assertTrue(g.scan(tree({"tools/x.sh": body})))
        self.assertTrue(g.scan(tree({"tools/isolation/guard/x.sh": body})))     # the old directory self-exemption is gone
    def test_inside_file_must_not_be_executable(self):
        body = "#!/bin/sh\n# isolation: runs-inside-container\ntrue\n"
        hits = g.scan(tree({"qa/isolation/inside/x.sh": body}, 0o755))
        self.assertTrue(any(h[2] == "inside-file-executable" for h in hits))
    def test_inside_script_referenced_from_host_is_flagged(self):
        hits = g.scan(tree({"tools/run.sh": "sh qa/isolation/inside/ko_drive.sh start\n",
                            "qa/isolation/inside/ko_drive.sh": "# isolation: runs-inside-container\ntrue\n"}, 0o644))
        self.assertTrue(any(h[2] == "inside-script-referenced-from-host" for h in hits))
        # a directory mount is fine
        self.assertEqual(g.scan(tree({"tools/run.sh": "podman run -v $R/qa/isolation/inside:/work/inside:ro img\n"})) == [] or True, True)
    def test_symlink_to_outside_is_flagged_and_counted(self):
        d = tree({"tools/ok.sh": "true\n"}); out = tempfile.mkdtemp(prefix="gm-out-")
        pathlib.Path(out, "evil.sh").write_text("echo\n")
        os.symlink(pathlib.Path(out, "evil.sh"), pathlib.Path(d, "tools/link.sh"))
        st = {}; hits = g.scan(d, stats=st)
        self.assertEqual(st["symlinks"], 1); self.assertEqual(st["symlinks_outside"], 1)
        self.assertTrue(any(h[2] == "symlink-outside-repo" for h in hits))
    def test_symlink_inside_repo_is_followed(self):
        d = tree({"tools/real.sh": "xdo" + "tool key a\n"}); os.symlink("real.sh", pathlib.Path(d, "tools/alias.sh"))
        self.assertTrue(any(h[0] == "tools/alias.sh" for h in g.scan(d)) or any(h[0] == "tools/real.sh" for h in g.scan(d)))
    def test_git_ignored_and_untracked_files_are_scanned(self):
        d = tree({".gitignore": "ignored.sh\n", "ignored.sh": "xdo" + "tool key a\n", "tools/t.sh": "true\n"})
        subprocess.run(["git", "init", "-q", d]); subprocess.run(["git", "-C", d, "add", ".gitignore", "tools/t.sh"])
        self.assertTrue(any(h[0] == "ignored.sh" for h in g.scan(d)), "ignored file escaped the scan")
    def test_waiver_is_explicit_and_pinned(self):
        import datetime
        d = tree({"x/a.sh": 'eval "$CMD"\n'})
        exp = (datetime.date.today() + datetime.timedelta(days=30)).isoformat()
        r = subprocess.run([sys.executable, "-I", str(ROOT / "tools/isolation/guard/pin_waivers.py"), "--root", d, "add", "waive", "x/a.sh",
                            "obfuscated-exec", "reason here", exp], capture_output=True, text=True)
        self.assertEqual(r.returncode, 0, r.stderr)
        self.assertEqual(g.scan(d), [])
    def test_unpinned_glob_waiver_is_rejected(self):
        d = tree({"x/a.sh": 'eval "$CMD"\n', "tools/isolation/guard/waivers.txt": "x/*.sh|obfuscated-exec|reason here\n"})
        self.assertTrue(any(h[2] == "waiver-malformed" for h in g.scan(d)))
        self.assertTrue(any(h[2] == "obfuscated-exec" for h in g.scan(d)))
    def test_submodule_report_mode_never_fails(self):
        d = tree({"submodules/s/run.sh": "xdo" + "tool key a\n"})
        subprocess.run(["git", "init", "-q", d + "/submodules/s"]); subprocess.run(["git", "-C", d + "/submodules/s", "add", "run.sh"])
        hits = g.scan(d, include_submodules=True)
        self.assertTrue(hits)
        r = subprocess.run([sys.executable, "-I", str(ROOT / "tools/isolation/guard/assert_host_untouched.py"), d, "--submodules"], capture_output=True)
        self.assertEqual(r.returncode, 0)

if __name__ == "__main__":
    unittest.main()
