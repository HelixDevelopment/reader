#!/usr/bin/env python3
"""Verify the actual server over TLS HTTP/1.1, HTTP/2 and HTTP/3."""
from __future__ import annotations

import json
import os
import signal
import subprocess
import sys
import time
from pathlib import Path

import requests

ROOT = Path("/mnt/agents/output/helix")
WORK = Path("/mnt/agents/research/helix-transport-tls")
EVIDENCE = ROOT / "evidence" / "live-transport"
BINARY = Path("/mnt/agents/research/helix-server-live")
H3CHECK = Path("/mnt/agents/research/helix-http3-check")
GO = os.environ.get("HELIX_GO", "/tmp/helix-go-toolchain/bin/go" if Path("/tmp/helix-go-toolchain/bin/go").exists() else "go")
BASE = "https://127.0.0.1:28443"
checks: list[dict] = []


def require(condition: bool, name: str) -> None:
    if not condition:
        raise AssertionError(name)
    checks.append({"name": name, "status": "PASS"})


def curl(version: str, output: Path, *extra: str) -> str:
    command = ["curl", "--fail", "--silent", "--show-error", "--cacert", str(WORK / "cert.pem"), version, "--output", str(output), "--write-out", "%{http_version}", *extra, BASE + "/healthz"]
    return subprocess.check_output(command, text=True, timeout=20).strip()


def build_binaries() -> None:
    """Rebuild server and verified HTTP/3 client from the current source tree."""
    BINARY.parent.mkdir(parents=True, exist_ok=True)
    with (EVIDENCE / "build.log").open("w") as log:
        for output, package in ((BINARY, "./cmd/helix-server"), (H3CHECK, "./cmd/helix-http3-check")):
            subprocess.run([GO, "build", "-trimpath", "-o", str(output), package], cwd=ROOT / "server", env=os.environ.copy(), check=True, stdout=log, stderr=subprocess.STDOUT, timeout=180)


def main() -> int:
    EVIDENCE.mkdir(parents=True, exist_ok=True)
    WORK.mkdir(parents=True, exist_ok=True)
    if not (WORK / "cert.pem").exists():
        subprocess.run([
            "openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes", "-days", "1",
            "-subj", "/CN=127.0.0.1", "-addext", "subjectAltName=IP:127.0.0.1",
            "-keyout", str(WORK / "key.pem"), "-out", str(WORK / "cert.pem"),
        ], check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=30)
        os.chmod(WORK / "key.pem", 0o600)
    env = os.environ.copy()
    env.update({
        "HELIX_ADDRESS": "127.0.0.1:28443",
        "HELIX_TLS_CERT": str(WORK / "cert.pem"),
        "HELIX_TLS_KEY": str(WORK / "key.pem"),
        "HELIX_HTTP3": "true",
        "HELIX_ADMIN_TOKEN": "transport-admin-token",
        "HELIX_OPDS_USERNAME": "transport-reader",
        "HELIX_OPDS_PASSWORD": "transport-password",
        "HELIX_POSTGRES_DSN": "",
        "HELIX_REDIS_URL": "",
        "HELIX_AI_MEMORY": "false",
        "GIN_MODE": "release",
    })
    process = None
    status = "PASS"
    try:
        build_binaries()
        with (EVIDENCE / "server.log").open("w") as log:
            process = subprocess.Popen([str(BINARY)], env=env, stdout=log, stderr=subprocess.STDOUT)
            session = requests.Session()
            session.trust_env = False
            response = None
            last_error = "no request completed"
            for _ in range(300):
                if process.poll() is not None:
                    raise RuntimeError("TLS server exited before readiness")
                try:
                    response = session.get(BASE + "/healthz", verify=str(WORK / "cert.pem"), timeout=1)
                    if response.status_code == 200:
                        break
                    last_error = f"HTTP {response.status_code}"
                except requests.RequestException as exc:
                    last_error = f"{type(exc).__name__}: {exc}"
                    time.sleep(0.1)
            require(response is not None and response.status_code == 200, f"verified TLS readiness (last error: {last_error})")
            require("h3=" in response.headers.get("Alt-Svc", ""), "HTTP/3 Alt-Svc advertisement")
            require(curl("--http1.1", WORK / "http1-body.json", "--tlsv1.2", "--tls-max", "1.2") == "1.1", "TLS 1.2 HTTP/1.1 request")
            require(curl("--http2", WORK / "http2-body.json") == "2", "TLS HTTP/2 request")
            h3 = json.loads(subprocess.check_output([str(H3CHECK), "-url", BASE + "/healthz", "-ca", str(WORK / "cert.pem")], text=True, timeout=25))
            require(h3.get("status") == "PASS" and h3.get("http_status") == 200, "TLS 1.3 HTTP/3 QUIC request")
    except Exception as exc:
        status = "FAIL"
        checks.append({"name": str(exc), "status": "FAIL"})
    finally:
        if process is not None:
            try:
                if process.poll() is None:
                    process.send_signal(signal.SIGTERM)
                process.wait(timeout=20)
            except Exception as exc:
                status = "FAIL"
                checks.append({"name": "server shutdown: " + str(exc), "status": "FAIL"})
        passed = sum(item["status"] == "PASS" for item in checks)
        failed = sum(item["status"] == "FAIL" for item in checks)
        (EVIDENCE / "results.json").write_text(json.dumps({"status": status, "pass": passed, "fail": failed, "checks": checks}, indent=2) + "\n")
    print(json.dumps({"status": status, "pass": sum(x["status"] == "PASS" for x in checks), "fail": sum(x["status"] == "FAIL" for x in checks)}))
    return 0 if status == "PASS" else 1


if __name__ == "__main__":
    sys.exit(main())
