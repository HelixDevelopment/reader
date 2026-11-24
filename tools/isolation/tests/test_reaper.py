"""F5 gate: a kill -9 of the runner must not leave an immortal container (lifetime cap) and the reaper removes the orphan.
Only touches containers named hrdisorev-*."""
import os, pathlib, shutil, signal, subprocess, tempfile, time, unittest
ROOT = pathlib.Path(__file__).resolve().parents[3]
NAME = "hrdisorev-test"

def rawdir():
    """round 6 (R6-I1): attached mode refuses a non-terminal stdout, and these runs have none - they use the raw-capture mode with a private host dir"""
    d = os.path.realpath(tempfile.mkdtemp(prefix="reaper-raw-", dir=os.path.expanduser("~/.cache/claude-tmp"))); os.chmod(d, 0o700); return d

def ps():
    return subprocess.run(["podman", "ps", "-a", "--filter", f"name={NAME}", "--format", "{{.Names}}"], capture_output=True, text=True).stdout.split()

class Reaper(unittest.TestCase):
    def setUp(self): self.raw = rawdir(); self.addCleanup(shutil.rmtree, self.raw, True)
    def tearDown(self): subprocess.run(["podman", "rm", "-f", NAME], capture_output=True)
    def test_killed_runner_orphan_is_reaped_and_capped(self):
        p = subprocess.Popen([str(ROOT / "tools/isolation/run_container.sh"), "-n", NAME, "-t", "120", "-o", self.raw, "--", "sleep", "300"],
                             stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        for _ in range(60):
            if ps(): break
            time.sleep(0.5)
        self.assertEqual(ps(), [NAME])
        life = subprocess.run(["podman", "inspect", NAME, "--format", "{{.Config.Timeout}}"], capture_output=True, text=True).stdout.strip()
        self.assertEqual(life, "120", "container has no lifetime cap")
        os.kill(p.pid, signal.SIGKILL); p.wait(); time.sleep(2)
        self.assertEqual(ps(), [NAME], "precondition: orphan survives kill -9 of the runner")
        r = subprocess.run([str(ROOT / "tools/isolation/reap.sh")], capture_output=True, text=True)
        self.assertNotIn(f"reaped {NAME}", r.stdout, "N4: a fresh container is inside the startup grace window")
        r = subprocess.run([str(ROOT / "tools/isolation/reap.sh")], capture_output=True, text=True, env=dict(os.environ, REAP_GRACE="0"))
        self.assertIn(f"reaped {NAME}", r.stdout); self.assertEqual(ps(), [])
    def test_reaper_ignores_foreign_names(self):
        # a container that is NOT ours must never be touched: the reaper only lists label org.helix.isolation + hrdiso names
        src = (ROOT / "tools/isolation/reap.sh").read_text()
        self.assertIn("hrdiso-*|hrdisorev-*", src); self.assertIn("label=org.helix.isolation", src)
    def test_live_runner_container_not_reaped(self):
        p = subprocess.Popen([str(ROOT / "tools/isolation/run_container.sh"), "-n", NAME, "-t", "60", "-o", self.raw, "--", "sleep", "30"],
                             stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        for _ in range(60):
            if ps(): break
            time.sleep(0.5)
        subprocess.run([str(ROOT / "tools/isolation/reap.sh")], capture_output=True)
        self.assertEqual(ps(), [NAME], "reaper killed a live run")
        p.kill(); p.wait()

class StartupRace(unittest.TestCase):
    """N4: a container whose owner file is not (yet) written must survive the startup grace window; once the lease is older than the
    window and the owner is dead/missing it is reaped. Dummy containers use the hrdisorev- prefix and a tiny local image."""
    NAME = "hrdisorev-lease"
    def tearDown(self):
        subprocess.run(["podman", "rm", "-f", self.NAME], capture_output=True)
    def start(self, lease, ownerfile):
        img = subprocess.run(["podman", "images", "--format", "{{.Repository}}:{{.Tag}}", "--filter", "reference=*gui-test*"], capture_output=True, text=True).stdout.split()
        self.assertTrue(img, "gui-test image missing")
        subprocess.run(["podman", "run", "-d", "--rm", "--name", self.NAME, "--network", "none", "--memory", "32m", "--timeout", "90",
                        "--label", "org.helix.isolation=redis", "--label", f"org.helix.owner-file={ownerfile}", "--label", f"org.helix.lease-ts={lease}",
                        "--entrypoint", "sleep", img[0], "80"], capture_output=True, check=True)
    def reap(self, grace=None):
        env = dict(os.environ, **({"REAP_GRACE": str(grace)} if grace is not None else {}))
        return subprocess.run([str(ROOT / "tools/isolation/reap.sh")], capture_output=True, text=True, env=env).stdout
    def test_owner_file_missing_within_grace_is_not_reaped_then_is(self):
        missing = f"/nonexistent/{self.NAME}.pid"
        self.start(int(time.time()), missing)                                    # starter has not written the owner file yet
        self.assertNotIn(f"reaped {self.NAME}", self.reap()); self.assertEqual(ps_name(self.NAME), [self.NAME], "reaper killed a starting container (race)")
        self.assertIn(f"reaped {self.NAME}", self.reap(grace=0)); self.assertEqual(ps_name(self.NAME), [])
    def test_expired_lease_with_missing_owner_is_reaped(self):
        self.start(int(time.time()) - 1000, f"/nonexistent/{self.NAME}.pid")
        self.assertIn(f"reaped {self.NAME}", self.reap())
    def test_owner_file_naming_live_process_is_kept_even_when_old(self):
        import tempfile
        d = tempfile.mkdtemp(prefix="rl-"); f = os.path.join(d, "owner.pid"); open(f, "w").write(str(os.getpid()))
        self.start(int(time.time()) - 1000, f)
        self.assertNotIn(f"reaped {self.NAME}", self.reap()); self.assertEqual(ps_name(self.NAME), [self.NAME])
    def test_start_server_writes_owner_file_before_creating_redis_and_uses_per_run_names(self):
        src = (ROOT / "tools/isolation/start_server.sh").read_text()
        self.assertLess(src.index('echo $$ > "$ST/server.pid"'), src.index("podman run -d"))      # owner file first
        self.assertIn("org.helix.lease-ts", src); self.assertNotIn("rname=hrdiso-redis\n", src); self.assertNotIn("36479", src)
        a = subprocess.run([str(ROOT / "tools/isolation/start_server.sh"), "ids"], capture_output=True, text=True).stdout.split()
        b = subprocess.run([str(ROOT / "tools/isolation/start_server.sh"), "ids"], capture_output=True, text=True).stdout.split()
        import re
        for x in (a, b):
            self.assertRegex(x[1], r"^hrdiso-redis-[0-9a-f]{8}$"); self.assertEqual(len(x), 4); self.assertNotEqual(x[2], x[3])
        self.assertNotEqual(a[1], b[1])                                                         # two runs never share a redis name

def ps_name(n):
    return subprocess.run(["podman", "ps", "-a", "--filter", f"name={n}", "--format", "{{.Names}}"], capture_output=True, text=True).stdout.split()

if __name__ == "__main__":
    unittest.main()
