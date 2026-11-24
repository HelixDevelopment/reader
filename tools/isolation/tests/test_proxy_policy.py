"""N1 gate (host side, no container): the proxy's route allow-list and robustness. Uses the real access_proxy.py on a unix socket."""
import http.server, json, os, pathlib, socket, subprocess, sys, tempfile, threading, time, unittest
ROOT = pathlib.Path(__file__).resolve().parents[3]

class Rec(http.server.BaseHTTPRequestHandler):
    seen = []
    def _go(s):
        n = int(s.headers.get("Content-Length") or 0); body = s.rfile.read(n).decode() if n else ""
        Rec.seen.append((s.command, s.path, body, s.headers.get("Authorization")))
        s.send_response(200); s.send_header("Content-Length", "2"); s.end_headers(); s.wfile.write(b"{}")
    do_GET = do_POST = do_PUT = do_HEAD = _go
    def log_message(s, *a): pass

def uds_request(path, raw):
    c = socket.socket(socket.AF_UNIX); c.settimeout(10); c.connect(path); c.sendall(raw)
    data = b""
    try:
        while True:
            d = c.recv(65536)
            if not d: break
            data += d
    except socket.timeout:
        pass
    c.close(); return data

def status(resp):
    return int(resp.split(b" ", 2)[1]) if resp.startswith(b"HTTP/") else 0

class QuietUp(http.server.ThreadingHTTPServer):
    def handle_error(self, request, client_address):      # the proxy aborts some upstream calls on purpose (oversize/aborted bodies)
        pass

class Policy(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.up = QuietUp(("127.0.0.1", 0), Rec); threading.Thread(target=cls.up.serve_forever, daemon=True).start()
        cls.d = tempfile.mkdtemp(prefix="pp-", dir=os.path.expanduser("~/.cache/claude-tmp"))
        cls.procs = {}
        for name, extra in (("common", ["--policy", "common"]), ("helixqa", ["--policy", "helixqa", "--rewrite-bearer", "hq-lit"])):
            sp = f"{cls.d}/{name}.sock"
            env = dict(os.environ, PROXY_ADMIN_TOKEN="REAL-RANDOM")
            cls.procs[name] = (subprocess.Popen([sys.executable, "-I", str(ROOT / "tools/isolation/access_proxy.py"), *extra, f"unix:{sp}",
                                                 "127.0.0.1", str(cls.up.server_address[1]), f"{cls.d}/{name}.log"], env=env), sp)
        for _, sp in cls.procs.values():
            for _ in range(50):
                if os.path.exists(sp): break
                time.sleep(0.1)
    @classmethod
    def tearDownClass(cls):
        for p, _ in cls.procs.values(): p.terminate()
        cls.up.shutdown()
        import shutil; shutil.rmtree(cls.d, ignore_errors=True)
    def req(self, pol, method, path, body=b"", headers=""):
        Rec.seen.clear()
        raw = f"{method} {path} HTTP/1.1\r\nHost: x\r\nContent-Length: {len(body)}\r\n{headers}Connection: close\r\n\r\n".encode() + body
        r = uds_request(self.procs[pol][1], raw); return status(r), list(Rec.seen)

    def test_read_routes_allowed(self):
        for m, p in (("GET", "/opds/1.2"), ("GET", "/v1/works?q=a&limit=2"), ("GET", "/v1/files/file-1/content"), ("GET", "/healthz"),
                     ("GET", "/syncs/progress/doc"), ("PUT", "/syncs/progress"), ("POST", "/users/create"), ("GET", "/users/auth"),
                     ("HEAD", "/v1/covers/e1"), ("GET", "/opds-pse/stream/f/0")):
            with self.subTest(route=p):
                st, seen = self.req("common", m, p, b"{}" if m in ("PUT", "POST") else b"")
                self.assertEqual(st, 200, (m, p)); self.assertEqual(len(seen), 1)
    def test_admin_and_mutating_routes_denied_in_every_policy(self):
        webdav = json.dumps({"id": "x", "protocol": "webdav", "endpoint": "http://127.0.0.1:9/"}).encode()
        local = json.dumps({"id": "y", "protocol": "local", "root": "/home"}).encode()
        for pol in ("common", "helixqa"):
            for m, p, b in (("POST", "/v1/sources", webdav), ("POST", "/v1/sources", local), ("POST", "/v1/scans", b"{}"), ("DELETE", "/v1/sources/x", b""),
                            ("GET", "/v1/sources", b""), ("GET", "/v1/admin/x", b""), ("POST", "/v1/admin/reset", b"{}"), ("GET", "/v1/jobs", b""),
                            ("PUT", "/v1/works/w", b"{}"), ("POST", "/v1/works", b"{}"), ("GET", "/debug/pprof/", b""), ("GET", "/metrics", b"")):
                with self.subTest(pol=pol, route=f"{m} {p} {b[:30]}"):
                    st, seen = self.req(pol, m, p, b)
                    self.assertEqual(st, 403); self.assertEqual(seen, [], "request reached the upstream")
    def test_path_tricks_denied(self):
        for p in ("/v1/works/../sources", "/v1/works/%2e%2e/sources", "//v1/sources", "/v1/works%2fx", "/opds/../v1/scans", "/v1/works/x/../../scans"):
            with self.subTest(path=p):
                st, seen = self.req("common", "GET", p); self.assertIn(st, (403, 400)); self.assertEqual(seen, [])
    def test_helixqa_policy_is_narrow(self):
        st, seen = self.req("helixqa", "GET", "/v1/scans"); self.assertEqual(st, 200)
        st, seen = self.req("helixqa", "POST", "/v1/sources", b"{}", "Authorization: Bearer hq-lit\r\n")
        self.assertEqual(st, 200); self.assertEqual(seen[0][3], "Bearer REAL-RANDOM")      # literal mapped to the real token, host-side only
        st, seen = self.req("helixqa", "POST", "/v1/sources", b'{"protocol":"local","root":"/"}', "Authorization: Bearer hq-lit\r\n")
        self.assertEqual(st, 403); self.assertEqual(seen, [])
        st, seen = self.req("helixqa", "POST", "/v1/sources", b"[]"); self.assertEqual(st, 403)
        st, seen = self.req("common", "POST", "/v1/sources", b"{}"); self.assertEqual(st, 403)          # not even empty in the default policy
        st, seen = self.req("common", "GET", "/v1/scans"); self.assertEqual(st, 403)
        st, seen = self.req("helixqa", "GET", "/v1/works", headers="Authorization: Bearer hq-lit\r\n")
        self.assertEqual(seen[0][3], "Bearer hq-lit")                                                   # no mapping on other routes
    def test_chunked_body_rejected(self):
        raw = b"POST /users/create HTTP/1.1\r\nHost: x\r\nTransfer-Encoding: chunked\r\nConnection: close\r\n\r\n2\r\n{}\r\n0\r\n\r\n"
        self.assertEqual(status(uds_request(self.procs["common"][1], raw)), 411)
    def test_bad_content_length(self):
        for cl, want in (("-5", 400), ("abc", 400), (str(9 * 1024 * 1024), 413), ("99999999999999999999", 413)):
            with self.subTest(cl=cl):
                raw = f"PUT /syncs/progress HTTP/1.1\r\nHost: x\r\nContent-Length: {cl}\r\nConnection: close\r\n\r\n".encode()
                self.assertEqual(status(uds_request(self.procs["common"][1], raw)), want)
    def test_proxy_survives_garbage_and_aborts(self):
        sp = self.procs["common"][1]
        uds_request(sp, b"\x00\xff\xfe garbage\r\n\r\n"); uds_request(sp, b"GET / HTTP/9.9\r\n\r\n")
        c = socket.socket(socket.AF_UNIX); c.connect(sp); c.sendall(b"PUT /syncs/progress HTTP/1.1\r\nContent-Length: 100\r\n\r\nab"); c.close()
        time.sleep(0.5)
        self.assertIsNone(self.procs["common"][0].poll(), "proxy died")
        st, _ = self.req("common", "GET", "/healthz"); self.assertEqual(st, 200)

if __name__ == "__main__":
    unittest.main()
