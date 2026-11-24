"""Helper for the network gate: run a container with the given wrapper args and probe host-loopback canaries."""
import json, socket, subprocess, threading, os, sys, pathlib
ROOT = pathlib.Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / "tools/isolation")); from rawrun import run_raw   # R5-B1: parse the RAW container streams, never the sanitized terminal copy

def listening_ports():
    """Every TCP port a host process is listening on (loopback or wildcard)."""
    out = subprocess.run(["ss", "-ltnH"], capture_output=True, text=True).stdout
    ports = set()
    for line in out.splitlines():
        a = line.split()[3]
        host, _, port = a.rpartition(":")
        if host.strip("[]") in ("127.0.0.1", "0.0.0.0", "*", "::", "::1") or host.startswith("127."):
            ports.add(int(port))
    return sorted(ports)

class Canary:
    def __init__(self):
        self.s = socket.socket(); self.s.bind(("127.0.0.1", 0)); self.s.listen(16); self.port = self.s.getsockname()[1]
        self.hits = 0; self.t = threading.Thread(target=self._run, daemon=True); self.t.start()
    def _run(self):
        while True:
            try:
                c, _ = self.s.accept(); self.hits += 1; c.close()
            except OSError:
                return
    def close(self): self.s.close()

def probe(wrapper_args, targets):
    r = run_raw(wrapper_args, ["node", "/opt/isoinside/net_probe.js", ",".join(dict.fromkeys(targets))])
    return (r.json_after("{", keep_prefix=True) or {}), r
