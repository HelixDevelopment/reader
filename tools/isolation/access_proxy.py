#!/usr/bin/env python3
"""Logging pass-through HTTP proxy (host side). Listens on a UNIX socket (default for isolated runs: only the one
container that is handed the socket directory can reach it) or on a loopback TCP port.
usage: access_proxy.py [--policy common|helixqa] [--rewrite-bearer LITERAL] unix:/path/h.sock up_host up_port logfile
       access_proxy.py [...] bind_host bind_port up_host up_port logfile            (legacy TCP form)
ROUTE ALLOW-LIST (default deny): the container may only use read routes of the catalogue/OPDS, KOSync and health; every
other method/path (admin /v1/sources, /v1/scans, /v1/admin*, jobs, any mutation) is answered 403 here and never reaches
the server. Policy `helixqa` additionally allows GET /v1/scans (read-only), POST /v1/sources ONLY with an empty JSON object
(the server answers 400; no source can be created), and GET /no/such/route. --rewrite-bearer: on those two admin routes
only, `Authorization: Bearer LITERAL` (the banks' hard-coded token) is replaced by the real per-run admin token taken from
env PROXY_ADMIN_TOKEN, which never enters the container. Request bodies: chunked -> 411, bad/negative Content-Length -> 400,
> 8 MiB -> 413; handler exceptions -> 502, never a crash.
logfile        one line per request: ts METHOD path -> status bytes auth=yes|no ua=...
logfile.bodies JSON lines for KOSync routes (/syncs/*, /users/*): request + response bodies. Authorization and
               x-auth-* headers are never logged; JSON fields named password/userkey/key/token are replaced by "<redacted>"."""
import sys, os, re, json, socket, posixpath, http.client, threading, datetime, traceback
from urllib.parse import unquote

def esc(s):
    """container-controlled text (request method / path) as it is written to the host logs: every byte outside printable ASCII (ESC, CR, LF, NUL, DEL, high bytes)
    becomes %XX, so a request line can neither carry a terminal sequence into a log the operator later `cat`s or tees, nor forge a log line (review round 4)."""
    return "".join(c if "!" <= c <= "~" else "".join("%%%02X" % b for b in c.encode("utf-8", "replace")) for c in str(s))
from http.server import ThreadingHTTPServer, BaseHTTPRequestHandler

MAX_BODY = 8 * 1024 * 1024
POLICY, REWRITE = "common", None
_a = sys.argv[1:]
while _a and _a[0].startswith("--"):
    if _a[0] == "--policy": POLICY = _a[1]; _a = _a[2:]
    elif _a[0] == "--rewrite-bearer": REWRITE = _a[1]; _a = _a[2:]
    else: raise SystemExit("unknown option " + _a[0])
if _a[0].startswith("unix:"):
    LISTEN = ("unix", _a[0][5:]); uh, up, logf = _a[1], int(_a[2]), _a[3]
else:
    LISTEN = ("tcp", (_a[0], int(_a[1]))); uh, up, logf = _a[2], int(_a[3]), _a[4]
ADMIN_REAL = os.environ.pop("PROXY_ADMIN_TOKEN", "")

READ = [r"/healthz", r"/readyz", r"/opds/.*", r"/opds-pse/stream/[^/]+/[^/]+", r"/v1/works(?:/[^/]+)?", r"/v1/editions/[^/]+",
        r"/v1/files/[^/]+/(?:content|license)", r"/v1/covers/[^/]+", r"/v1/formats", r"/v1/protocols", r"/syncs/progress/[^/]+", r"/users/auth"]
ALLOW = [("GET", r) for r in READ] + [("HEAD", r) for r in READ] + [("PUT", r"/syncs/progress"), ("POST", r"/users/create")]
ADMIN_ROUTES = []
if POLICY == "helixqa":
    ALLOW += [("GET", r"/no/such/route"), ("GET", r"/v1/scans"), ("POST", r"/v1/sources")]
    ADMIN_ROUTES = [("GET", "/v1/scans"), ("POST", "/v1/sources")]
ALLOW_RX = [(m, re.compile(r)) for m, r in ALLOW]

def normal_path(raw):
    """Decoded, canonical path or None. Encoded slashes, dot segments, double slashes, NULs are refused."""
    pth = raw.split("?", 1)[0].split("#", 1)[0]
    if not pth.startswith("/") or "%2f" in pth.lower() or "%5c" in pth.lower() or "\\" in pth or "\x00" in unquote(pth):
        return None
    d = unquote(pth)
    if "//" in d or posixpath.normpath(d) != (d if d == "/" else d.rstrip("/")):
        return None
    return d

def allowed(method, raw, body):
    d = normal_path(raw)
    if d is None:
        return False
    if not any(m == method and rx.fullmatch(d) for m, rx in ALLOW_RX):
        return False
    if method == "POST" and d == "/v1/sources":          # helixqa: only an EMPTY object (server answers 400, creates nothing)
        try:
            return json.loads(body or b"null") == {}
        except Exception:
            return False
    return True
BODYLOG = logf + ".bodies"
lock = threading.Lock()
HOP = {"connection", "keep-alive", "transfer-encoding", "te", "trailer", "upgrade", "proxy-authorization"}
SECRET = {"password", "userkey", "key", "token", "authorization"}

def redact(raw):
    txt = raw.decode("utf-8", "replace")[:4096]
    try:
        o = json.loads(txt)
        def walk(x):
            if isinstance(x, dict):
                return {k: ("<redacted>" if k.lower() in SECRET else walk(v)) for k, v in x.items()}
            if isinstance(x, list):
                return [walk(v) for v in x]
            return x
        return walk(o)
    except Exception:
        return txt if len(txt) < 200 else txt[:200] + "..."

class H(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"
    timeout = 30                      # slow/stalled clients cannot pin a thread forever
    def log_message(self, *a): pass
    def address_string(self): return "unix"
    def _reply(self, status, msg):
        data = json.dumps({"error": msg}).encode()
        self.close_connection = True
        self.send_response(status); self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data))); self.send_header("Connection", "close"); self.end_headers()
        if self.command != "HEAD": self.wfile.write(data)
        with lock, open(logf, "a") as f:
            f.write(f"{datetime.datetime.now(datetime.timezone.utc).isoformat()} {esc(self.command)} {esc(self.path.split('?')[0])} -> {status} DENIED:{esc(msg)}\n")
    def handle_any(self):
        try:
            self._handle()
        except Exception:
            try: self._reply(502, "isolation proxy: upstream/handler error")
            except Exception: pass
            with lock, open(logf + ".errors", "a") as f:
                f.write(traceback.format_exc() + "\n")
    def _handle(self):
        if self.headers.get("Transfer-Encoding"):
            return self._reply(411, "chunked request bodies are not accepted")
        cl = self.headers.get("Content-Length")
        try:
            n = int(cl) if cl is not None else 0
        except ValueError:
            return self._reply(400, "bad Content-Length")
        if n < 0: return self._reply(400, "negative Content-Length")
        if n > MAX_BODY: return self._reply(413, "request body too large")
        body = self.rfile.read(n) if n else None
        if not allowed(self.command, self.path, body):
            return self._reply(403, "isolation proxy: route not allowed")
        c = http.client.HTTPConnection(uh, up, timeout=30)
        hdrs = {k: v for k, v in self.headers.items() if k.lower() not in HOP}
        if REWRITE and ADMIN_REAL and (self.command, self.path.split("?")[0]) in ADMIN_ROUTES and hdrs.get("Authorization") == "Bearer " + REWRITE:
            hdrs["Authorization"] = "Bearer " + ADMIN_REAL
        c.request(self.command, self.path, body, hdrs)
        r = c.getresponse(); data = r.read()
        self.send_response(r.status)
        for k, v in r.getheaders():
            if k.lower() not in HOP and k.lower() != "content-length": self.send_header(k, v)
        self.send_header("Content-Length", str(len(data))); self.end_headers()
        if self.command != "HEAD": self.wfile.write(data)
        path = esc(self.path.split("?")[0])
        ts = datetime.datetime.now(datetime.timezone.utc).isoformat()
        auth = "yes" if (self.headers.get("Authorization") or self.headers.get("x-auth-user")) else "no"
        with lock:
            with open(logf, "a") as f:
                f.write(f"{ts} {esc(self.command)} {path} -> {r.status} bytes={len(data)} auth={auth} ua={esc(self.headers.get('User-Agent'))!r}\n")
            if path.startswith(("/syncs", "/users")):
                with open(BODYLOG, "a") as f:
                    f.write(json.dumps({"ts": ts, "method": esc(self.command), "path": path, "status": r.status,
                                        "request_body": redact(body) if body else None, "response_body": redact(data) if data else None}) + "\n")
    do_GET = do_POST = do_PUT = do_DELETE = do_HEAD = do_OPTIONS = handle_any

class _Quiet(ThreadingHTTPServer):
    daemon_threads = True
    def handle_error(self, request, client_address):
        with lock, open(logf + ".errors", "a") as f:
            f.write(traceback.format_exc() + "\n")

class UServer(_Quiet):
    address_family = socket.AF_UNIX
    def server_bind(self):
        if os.path.exists(self.server_address): os.unlink(self.server_address)
        self.socket.bind(self.server_address); os.chmod(self.server_address, 0o600)
        self.server_name = "unix"; self.server_port = 0
    def get_request(self):
        c, _ = self.socket.accept(); return c, ("unix", 0)

if LISTEN[0] == "unix":
    UServer(LISTEN[1], H).serve_forever()
else:
    _Quiet(LISTEN[1], H).serve_forever()
