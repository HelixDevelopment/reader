"""Permanent gate (F1): a container started by run_container.sh can reach NOTHING on the host except the one proxy socket.
Needs rootless podman + localhost/helix-gui-test:1. Run: python3 -I -m unittest discover -s tools/isolation/tests -v"""
import urllib.error, http.server, json, os, pathlib, socket, subprocess, sys, tempfile, threading, unittest
sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import canary_probe as c

ROOT = c.ROOT
IMG_OK = subprocess.run(["podman", "image", "exists", "localhost/helix-gui-test:1"]).returncode == 0

def host_ips():
    out = subprocess.run(["hostname", "-I"], capture_output=True, text=True).stdout.split()
    return [i for i in out if "." in i]

@unittest.skipUnless(IMG_OK, "gui-test image not built")
class NetworkIsolation(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.work = tempfile.mkdtemp(prefix="hiso-", dir=os.path.expanduser("~/.cache/claude-tmp"))
        os.environ["ISO_WORK"] = cls.work
        cls.canary = c.Canary()
        cls.ports = c.listening_ports()
        # a tiny upstream "server" + the real logging proxy on a unix socket
        cls.seen = []
        outer = cls
        class Up(http.server.BaseHTTPRequestHandler):
            def do_GET(s): outer.seen.append(("GET", s.path)); s.send_response(200); s.send_header("Content-Length", "6"); s.end_headers(); s.wfile.write(b"UPSTRM")
            def do_POST(s):
                n = int(s.headers.get("Content-Length") or 0); s.rfile.read(n); outer.seen.append(("POST", s.path))
                s.send_response(201); s.send_header("Content-Length", "2"); s.end_headers(); s.wfile.write(b"{}")
            def log_message(s, *a): pass
        cls.up = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Up)
        threading.Thread(target=cls.up.serve_forever, daemon=True).start()
        cls.sock = pathlib.Path(cls.work, "sock"); cls.sock.mkdir(mode=0o700)
        cls.proxy = subprocess.Popen([sys.executable, "-I", str(ROOT / "tools/isolation/access_proxy.py"), f"unix:{cls.sock}/h.sock",
                                      "127.0.0.1", str(cls.up.server_address[1]), f"{cls.work}/access.log"])
        for _ in range(50):
            if (cls.sock / "h.sock").exists(): break
            import time; time.sleep(0.1)

    @classmethod
    def tearDownClass(cls):
        cls.proxy.terminate(); cls.up.shutdown(); cls.canary.close()
        import shutil; shutil.rmtree(cls.work, ignore_errors=True)

    def targets(self):
        t = [f"10.0.2.2:{self.canary.port}", f"127.0.0.1:{self.canary.port}", "1.1.1.1:443", "10.0.2.3:53"]
        for p in set(self.ports) | {5037, 8432, 55432, 11434, 22, 631, 8095, 45395}:
            t += [f"10.0.2.2:{p}", f"127.0.0.1:{p}"]
        for ip in host_ips():
            t += [f"{ip}:{self.canary.port}", f"{ip}:22"]
        return t

    def test_no_bridge_nothing_open(self):
        res, r = c.probe(["-n", "hrdiso-nettest-a"], self.targets())
        self.assertTrue(res, r.stderr)
        self.assertEqual([k for k, v in res.items() if v == "open"], [], f"reachable from container: {res}")
        self.assertEqual(self.canary.hits, 0)

    def test_with_bridge_only_the_proxy_port_open(self):
        res, r = c.probe(["-n", "hrdiso-nettest-b", "-b", str(self.sock)], self.targets() + ["127.0.0.1:38191"])
        self.assertTrue(res, r.stderr)
        opened = sorted(k for k, v in res.items() if v == "open")
        self.assertEqual(opened, ["127.0.0.1:38191"], f"unexpected reachable targets: {res}")
        self.assertEqual(self.canary.hits, 0)

    def test_bridge_really_reaches_upstream(self):
        r = c.run_raw(["-n", "hrdiso-nettest-c", "-b", str(self.sock)], ["node", "-e",
               "fetch('http://127.0.0.1:38191/healthz').then(r=>r.text()).then(t=>console.log('GOT:'+t))"])
        self.assertIn("GOT:UPSTRM", r.stdout, r.stderr)
        self.assertIn("GET /healthz -> 200", pathlib.Path(self.work, "access.log").read_text())

    ADMIN_JS = """
const post = (p, body) => fetch('http://127.0.0.1:38191' + p, {method: 'POST', body: JSON.stringify(body), headers: {Authorization: 'Bearer anything', 'Content-Type': 'application/json'}}).then(r => r.status);
(async () => {
  const out = {};
  out.webdav_to_canary = await post('/v1/sources', {id: 'x', protocol: 'webdav', endpoint: 'http://127.0.0.1:%(port)d/'});
  out.scan = await post('/v1/scans', {source_id: 'x', prefix: ''});
  out.local_root = await post('/v1/sources', {id: 'y', protocol: 'local', root: '/home'});
  out.list_sources = (await fetch('http://127.0.0.1:38191/v1/sources')).status;
  out.admin_prefix = (await fetch('http://127.0.0.1:38191/v1/admin/x')).status;
  out.read_ok = (await fetch('http://127.0.0.1:38191/v1/works')).status;
  console.log('ADMIN:' + JSON.stringify(out));
})();
"""
    def run_admin_attempts(self):
        r = c.run_raw(["-n", "hrdiso-nettest-d", "-b", str(self.sock)], ["node", "-e", self.ADMIN_JS % {"port": self.canary.port}])
        line = [l for l in r.stdout.splitlines() if l.startswith("ADMIN:")]
        self.assertTrue(line, r.stderr); return json.loads(line[0][6:])

    def test_admin_routes_refused_from_inside_the_container(self):
        out = self.run_admin_attempts()
        for k in ("webdav_to_canary", "scan", "local_root", "list_sources", "admin_prefix"):
            self.assertIn(out[k], (403, 404), f"{k} was not refused: {out}")
        self.assertEqual(self.canary.hits, 0, "host loopback canary was hit via SSRF")
        self.assertEqual([x for x in self.seen if x[0] == "POST"], [], "a mutating request reached the server")
        self.assertEqual(out["read_ok"], 200)

    @unittest.skipUnless(os.environ.get("HELIX_SERVER_BIN"), "set HELIX_SERVER_BIN to also run against the REAL helix-server")
    def test_admin_routes_refused_against_real_server_and_no_source_created(self):
        import secrets, time, urllib.request
        admin = secrets.token_hex(12); port = 38199; st = pathlib.Path(self.work, "real"); (st / "sock").mkdir(parents=True, mode=0o700)
        env = dict(os.environ, HELIX_ADDRESS=f"127.0.0.1:{port}", HELIX_ADMIN_TOKEN=admin, HELIX_HTTP3="false", GIN_MODE="release",
                   HELIX_COVER_CACHE=str(st / "covers"), PROXY_ADMIN_TOKEN=admin)
        srv = subprocess.Popen([os.environ["HELIX_SERVER_BIN"]], env={k: v for k, v in env.items() if k != "PROXY_ADMIN_TOKEN"}, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        prx = subprocess.Popen([sys.executable, "-I", str(ROOT / "tools/isolation/access_proxy.py"), "--policy", "common", f"unix:{st}/sock/h.sock", "127.0.0.1", str(port), f"{st}/a.log"], env=env)
        try:
            for _ in range(60):
                try: urllib.request.urlopen(f"http://127.0.0.1:{port}/readyz", timeout=1); break
                except Exception: time.sleep(0.5)
            r = c.run_raw(["-n", "hrdiso-nettest-e", "-b", f"{st}/sock"], ["node", "-e", self.ADMIN_JS % {"port": self.canary.port}])
            out = json.loads([l for l in r.stdout.splitlines() if l.startswith("ADMIN:")][0][6:])
            for k in ("webdav_to_canary", "scan", "local_root", "list_sources", "admin_prefix"):
                self.assertIn(out[k], (403, 404), out)
            self.assertEqual(self.canary.hits, 0)
            for sid in ("x", "y"):    # host side, REAL admin token: a scan of a source that must not exist is refused
                q = urllib.request.Request(f"http://127.0.0.1:{port}/v1/scans", data=json.dumps({"source_id": sid, "prefix": ""}).encode(),
                                           headers={"Authorization": "Bearer " + admin, "Content-Type": "application/json"}, method="POST")
                try:
                    code = urllib.request.urlopen(q, timeout=5).status
                except urllib.error.HTTPError as e:
                    code = e.code
                self.assertIn(code, (400, 404), f"source {sid} exists on the server after the container's attempts")
        finally:
            prx.terminate(); srv.terminate()

if __name__ == "__main__":
    unittest.main()
