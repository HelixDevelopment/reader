#!/usr/bin/env python3
"""Loopback fault-injecting forwarder for the HRD-029 live run (test tool, no secrets stored, bodies never logged).
Rules are re-read from --rules (JSON) on every request; counters reset when the rules text changes:
  {"fail_create_nth": N}          -> 503 (not forwarded) on the Nth `create ticket` since the rules changed
  {"fail_status_modify_nth": N}   -> 503 (not forwarded) on the Nth real status `modify` (version probes excluded)
  {"redirect_authed": true}       -> every authenticated request is answered 302 to the sink instead of forwarded
A second listener (--sink-port) is the redirect target; every request it receives is appended to --sink-log
(method, path, whether an Authorization header was present - never its value)."""
import argparse, http.server, json, threading, urllib.request, urllib.error

ap = argparse.ArgumentParser()
ap.add_argument("--listen", type=int, required=True)
ap.add_argument("--upstream", required=True)
ap.add_argument("--rules", required=True)
ap.add_argument("--sink-port", type=int, required=True)
ap.add_argument("--sink-log", required=True)
ap.add_argument("--log", required=True)
a = ap.parse_args()
LOCK = threading.Lock()
STATE = {"text": None, "creates": 0, "status_modifies": 0}
OPENER = urllib.request.build_opener(urllib.request.ProxyHandler({}))


def log(path, line):
    with LOCK, open(path, "a") as fh:
        fh.write(line + "\n")


class Sink(http.server.BaseHTTPRequestHandler):
    def _hit(self):
        self.rfile.read(int(self.headers.get("Content-Length") or 0))
        log(a.sink_log, "SINK-HIT %s %s authorization_header_present=%s" % (
            self.command, self.path, self.headers.get("Authorization") is not None))
        self.send_response(200); self.end_headers(); self.wfile.write(b"{}")
    do_POST = do_GET = _hit

    def log_message(self, *x):
        pass


class Proxy(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        raw = self.rfile.read(int(self.headers.get("Content-Length") or 0))
        try:
            body = json.loads(raw or b"{}")
        except ValueError:
            body = {}
        data = body.get("data") or {}
        action, obj = body.get("action"), body.get("object")
        try:
            text = open(a.rules).read()
        except OSError:
            text = "{}"
        with LOCK:
            if text != STATE["text"]:
                STATE.update(text=text, creates=0, status_modifies=0)
            rules = json.loads(text or "{}")
            fail = None
            if (action, obj) == ("create", "ticket"):
                STATE["creates"] += 1
                if rules.get("fail_create_nth") == STATE["creates"]:
                    fail = "create#%d" % STATE["creates"]
            if action == "modify" and "status" in data and data.get("version") != 2147483647:
                STATE["status_modifies"] += 1
                if rules.get("fail_status_modify_nth") == STATE["status_modifies"]:
                    fail = "status-modify#%d" % STATE["status_modifies"]
        if fail:
            log(a.log, "INJECT-503 %s" % fail)
            return self._send(503, json.dumps({"errorCode": 5000, "errorMessage": "injected unavailable"}).encode())
        if rules.get("redirect_authed") and action != "authenticate" and (body.get("jwt") or self.headers.get("Authorization")):
            log(a.log, "INJECT-302 action=%s" % action)
            self.send_response(302)
            self.send_header("Location", "http://127.0.0.1:%d/do" % a.sink_port)
            self.send_header("Content-Length", "0")
            self.end_headers()
            return
        req = urllib.request.Request(a.upstream + self.path, raw, {k: v for k, v in self.headers.items()
                                                                    if k.lower() in ("content-type", "authorization")}, method="POST")
        try:
            with OPENER.open(req, timeout=60) as r:
                code, out = r.status, r.read()
        except urllib.error.HTTPError as e:
            code, out = e.code, e.read()
        log(a.log, "FWD action=%s object=%s status=%d" % (action, obj, code))
        self._send(code, out)

    def do_GET(self):
        try:
            with OPENER.open(a.upstream + self.path, timeout=30) as r:
                self._send(r.status, r.read())
        except urllib.error.HTTPError as e:
            self._send(e.code, e.read())

    def _send(self, code, out):
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(out)))
        self.end_headers()
        self.wfile.write(out)

    def log_message(self, *x):
        pass


for port, h in ((a.sink_port, Sink),):
    threading.Thread(target=http.server.ThreadingHTTPServer(("127.0.0.1", port), h).serve_forever, daemon=True).start()
http.server.ThreadingHTTPServer(("127.0.0.1", a.listen), Proxy).serve_forever()
