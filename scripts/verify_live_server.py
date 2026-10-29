#!/usr/bin/env python3
"""Live server verification against real PostgreSQL, pgvector and Redis.

The test uses non-production loopback credentials, never prints them, and writes
only assertion names/statuses. Run the database-backed live Go tests separately;
this script exercises the actual server process, API protocols and restart.
"""
from __future__ import annotations

import getpass
import json
import os
import signal
import subprocess
import sys
import time
from pathlib import Path

import requests

ROOT = Path("/mnt/agents/output/helix")
EVIDENCE = ROOT / "evidence" / "live-server"
BINARY = Path("/mnt/agents/research/helix-server-live")
GO = os.environ.get("HELIX_GO", "/tmp/helix-go-toolchain/bin/go" if Path("/tmp/helix-go-toolchain/bin/go").exists() else "go")
LIBRARY = Path("/mnt/agents/research/live-server-library")
BASE = "http://127.0.0.1:28080"
# Match the operating-system peer identity used to launch the disposable server.
# Both root and kimi have local test roles in the qualification environment; no
# password or production credential is embedded in this verifier.
DSN = f"postgres:///helix_test?host=/var/run/postgresql&user={getpass.getuser()}"
checks: list[dict] = []
RESTART_DAEMONS = "--restart-daemons" in sys.argv[1:]


def require(condition: bool, name: str) -> None:
    if not condition:
        raise AssertionError(name)
    checks.append({"name": name, "status": "PASS"})


def server_environment() -> dict:
    env = os.environ.copy()
    env.update({
        "HELIX_ADDRESS": "127.0.0.1:28080",
        "HELIX_POSTGRES_DSN": DSN,
        "HELIX_REDIS_URL": "redis://127.0.0.1:6379",
        "HELIX_AI_MEMORY": "true",
        "HELIX_EMBEDDING_DIMENSIONS": "3",
        "HELIX_ADMIN_TOKEN": "live-admin-token",
        "HELIX_OPDS_USERNAME": "live-reader",
        "HELIX_OPDS_PASSWORD": "live-password",
        "HELIX_KOSYNC_REGISTRATION": "true",
        "HELIX_HTTP3": "false",
        "HELIX_COVER_CACHE": "/mnt/agents/research/live-server-covers",
        "GIN_MODE": "release",
    })
    return env


def start_server(log):
    process = subprocess.Popen([str(BINARY)], env=server_environment(), stdout=log, stderr=subprocess.STDOUT)
    session = requests.Session()
    session.trust_env = False
    # Cold startup may run PostgreSQL migrations plus pgvector and Redis health
    # checks. Their configured budgets exceed ten seconds, so the harness must
    # not kill a healthy initializing process before those budgets elapse.
    for _ in range(900):
        if process.poll() is not None:
            raise RuntimeError("server exited before readiness")
        try:
            response = session.get(BASE + "/readyz", timeout=0.25)
            if response.status_code == 200:
                return process, session
        except requests.RequestException:
            pass
        time.sleep(0.1)
    process.send_signal(signal.SIGTERM)
    raise RuntimeError("server readiness timeout")


def stop_server(process) -> None:
    if process.poll() is None:
        process.send_signal(signal.SIGTERM)
    process.wait(timeout=20)


def redis_ready() -> bool:
    return subprocess.run(["redis-cli", "ping"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=2).returncode == 0


def start_redis() -> None:
    if redis_ready():
        return
    # Match the disposable qualification daemon's current data directory and
    # explicit-save policy. No application or user credentials are involved.
    subprocess.run([
        "redis-server", "--daemonize", "yes", "--dir", "/mnt/agents",
        "--dbfilename", "dump.rdb", "--appendonly", "no", "--save", "",
    ], check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=10)
    for _ in range(100):
        if redis_ready():
            return
        time.sleep(0.1)
    raise RuntimeError("Redis readiness after daemon restart")


def restart_daemons() -> None:
    subprocess.run(["redis-cli", "SAVE"], check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=15)
    subprocess.run(["redis-cli", "SHUTDOWN", "SAVE"], check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=15)
    for _ in range(100):
        if not redis_ready():
            break
        time.sleep(0.1)
    start_redis()
    subprocess.run(["pg_ctlcluster", "15", "main", "restart"], check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=45)
    for _ in range(150):
        if subprocess.run(["pg_isready"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=2).returncode == 0:
            return
        time.sleep(0.1)
    raise RuntimeError("PostgreSQL readiness after daemon restart")


def build_server() -> None:
    """Build the exact tree under test; never reuse a stale qualification binary."""
    BINARY.parent.mkdir(parents=True, exist_ok=True)
    with (EVIDENCE / "build.log").open("w") as log:
        subprocess.run([GO, "build", "-trimpath", "-o", str(BINARY), "./cmd/helix-server"], cwd=ROOT / "server", env=os.environ.copy(), check=True, stdout=log, stderr=subprocess.STDOUT, timeout=180)


def main() -> int:
    EVIDENCE.mkdir(parents=True, exist_ok=True)
    LIBRARY.mkdir(parents=True, exist_ok=True)
    (LIBRARY / "live-server-book.txt").write_text("live-server-book\n\nA deterministic paragraph for live ingestion.\n")
    status = "PASS"
    process = None
    try:
        build_server()
        with (EVIDENCE / "server.log").open("w") as log:
            process, session = start_server(log)
            admin = {"Authorization": "Bearer live-admin-token"}
            basic = requests.auth.HTTPBasicAuth("live-reader", "live-password")
            require(session.get(BASE + "/healthz", timeout=3).status_code == 200, "health endpoint")
            require(session.post(BASE + "/v1/sources", json={}, timeout=3).status_code == 401, "admin authentication boundary")
            require(session.get(BASE + "/v1/works", timeout=3).status_code == 401, "catalog authentication boundary")

            existing = session.get(BASE + "/v1/works", params={"q": "live-server-book", "limit": 100}, auth=basic, timeout=3)
            existing.raise_for_status()
            for work in existing.json().get("works", []):
                cleanup = session.delete(BASE + "/v1/works/" + work["id"], headers=admin, timeout=3)
                if cleanup.status_code != 204:
                    raise AssertionError("pre-test catalog cleanup")

            created = session.post(BASE + "/v1/sources", headers=admin, json={"id": "live-server-source", "protocol": "local", "root": str(LIBRARY)}, timeout=3)
            require(created.status_code == 201, "live local source creation")
            source_id = created.json()["id"]
            scan = session.post(BASE + "/v1/scans", headers=admin, json={"source_id": source_id, "prefix": ""}, timeout=3)
            require(scan.status_code == 202, "live scan accepted")
            job_id = scan.json()["id"]
            job = None
            for _ in range(100):
                job = session.get(BASE + "/v1/scans/" + job_id, headers=admin, timeout=3).json()
                if job.get("state") in {"succeeded", "failed", "cancelled"}:
                    break
                time.sleep(0.05)
            require(job is not None and job.get("state") == "succeeded" and job.get("report", {}).get("files") == 1, "live in-place ingestion")

            works = session.get(BASE + "/v1/works", params={"q": "live-server-book"}, auth=basic, timeout=3).json()
            require(len(works.get("works", [])) == 1, "live catalog query")
            work = works["works"][0]
            edition = work["editions"][0]
            file = edition["files"][0]
            require(file["id"] and file["source_id"] == source_id, "file identity and source association")
            require(file["sha256"] and file["size"] > 0, "content identity and size")

            atom = session.get(BASE + "/opds/1.2/publications", auth=basic, timeout=3)
            require(atom.status_code == 200 and "application/atom+xml" in atom.headers.get("Content-Type", "") and b"live-server-book" in atom.content, "OPDS 1.2 acquisition feed")
            feed = session.get(BASE + "/opds/2.0/publications", auth=basic, timeout=3)
            require(feed.status_code == 200 and "live-server-book" in feed.text and "publications" in feed.json(), "OPDS 2.0 publication feed")
            require(session.get(BASE + "/opds/1.2/opensearch.xml", auth=basic, timeout=3).status_code == 200, "OpenSearch description")

            content_url = BASE + "/v1/files/" + file["id"] + "/content"
            ranged = session.get(content_url, headers={"Range": "bytes=0-3"}, auth=basic, timeout=3)
            require(ranged.status_code == 206 and ranged.content == b"live" and ranged.headers.get("Content-Range", "").startswith("bytes 0-3/"), "single-range content download")
            head = session.head(content_url, auth=basic, timeout=3)
            require(head.status_code == 200 and not head.content and int(head.headers.get("Content-Length", "0")) == file["size"], "HEAD content metadata without body")
            stale = session.get(content_url, headers={"Range": "bytes=0-3", "If-Range": '"sha256-stale"'}, auth=basic, timeout=3)
            require(stale.status_code == 200 and stale.content.startswith(b"live-server-book"), "stale If-Range returns complete representation")
            multiple = session.get(content_url, headers={"Range": "bytes=0-3,5-8"}, auth=basic, timeout=3)
            require(multiple.status_code == 200 and multiple.content.startswith(b"live-server-book"), "multi-range request uses complete-response fallback")
            binary = session.get(content_url, headers={"Accept-Encoding": "br, gzip"}, auth=basic, timeout=3)
            require(binary.status_code == 200 and not binary.headers.get("Content-Encoding") and binary.content.startswith(b"live-server-book"), "attachment remains identity encoded")
            compressed = session.get(BASE + "/v1/formats", headers={"Accept-Encoding": "gzip"}, timeout=3)
            require(compressed.status_code == 200 and compressed.headers.get("Content-Encoding") == "gzip" and len(compressed.json()["formats"]) > 0, "gzip content negotiation")

            credentials = {"x-auth-user": "live-server-reader", "x-auth-key": "0123456789abcdef0123456789abcdef"}
            registration = session.post(BASE + "/users/create", json={"username": credentials["x-auth-user"], "password": credentials["x-auth-key"]}, timeout=5)
            require(registration.status_code in {201, 409}, "KOSync registration or deterministic rerun")
            require(session.get(BASE + "/users/auth", headers=credentials, timeout=5).status_code == 200, "KOSync authentication")
            progress = {"document": "live-server-document", "progress": "42", "percentage": 0.25, "device": "live-device", "device_id": "live-device"}
            require(session.put(BASE + "/syncs/progress", headers=credentials, json=progress, timeout=5).status_code == 200, "KOSync progress write")
            synced = session.get(BASE + "/syncs/progress/live-server-document", headers=credentials, timeout=5)
            require(synced.status_code == 200 and synced.json()["progress"] == "42" and synced.json()["timestamp"] > 0, "KOSync progress read")

            memory = {"record": {"id": "live-server-memory", "scope": "live-server", "kind": "note", "content": "deterministic memory", "model": "live-model"}, "embedding": [1, 0, 0]}
            require(session.put(BASE + "/v1/ai/memories", headers=admin, json=memory, timeout=5).status_code == 200, "pgvector memory write")
            found = session.post(BASE + "/v1/ai/memories/search", headers=admin, json={"scope": "live-server", "embedding": [1, 0, 0], "limit": 5}, timeout=5)
            require(found.status_code == 200 and any(item["record"]["id"] == "live-server-memory" for item in found.json()["results"]), "pgvector scoped search")

            internal = {"BookID": "ignored", "DeviceID": "live-device", "Pos": {"Chapter": 2, "Block": 1, "Offset": 0}, "Clock": {"V": {"live-device": 1}}}
            require(session.put(BASE + "/v1/sync/progress/live-user/live-book", headers=admin, json=internal, timeout=5).status_code == 200, "internal progress write")

            stop_server(process)
            process = None
            if RESTART_DAEMONS:
                restart_daemons()
                require(redis_ready(), "Redis and PostgreSQL daemons restarted")
            process, session = start_server(log)
            persisted = session.get(BASE + "/v1/works", params={"q": "live-server-book"}, auth=basic, timeout=3).json()
            require(len(persisted.get("works", [])) == 1, "PostgreSQL catalog survives server restart")
            require(session.get(BASE + "/users/auth", headers=credentials, timeout=5).status_code == 200, "Redis KOSync account survives server restart")
            require(session.get(BASE + "/syncs/progress/live-server-document", headers=credentials, timeout=5).json()["progress"] == "42", "Redis KOSync progress survives server restart")
            internal_saved = session.get(BASE + "/v1/sync/progress/live-user/live-book", headers=admin, timeout=5)
            require(internal_saved.status_code == 200 and internal_saved.json()["Pos"]["Chapter"] == 2, "Redis internal progress survives server restart")
            reattached = session.post(BASE + "/v1/sources", headers=admin, json={"id": source_id, "protocol": "local", "root": str(LIBRARY)}, timeout=3)
            require(reattached.status_code == 201 and reattached.json()["id"] == source_id, "stable source identity reattaches after restart")
            restored_content = session.get(content_url, auth=basic, timeout=3)
            require(restored_content.status_code == 200 and restored_content.content.startswith(b"live-server-book"), "existing file identity serves after source re-registration")
            rescan = session.post(BASE + "/v1/scans", headers=admin, json={"source_id": source_id, "prefix": ""}, timeout=3)
            require(rescan.status_code == 202, "post-restart source rescan accepted")
            rescan_job = None
            for _ in range(100):
                rescan_job = session.get(BASE + "/v1/scans/" + rescan.json()["id"], headers=admin, timeout=3).json()
                if rescan_job.get("state") in {"succeeded", "failed", "cancelled"}:
                    break
                time.sleep(0.05)
            require(rescan_job is not None and rescan_job.get("state") == "succeeded" and rescan_job.get("report", {}).get("removed") == 0, "post-restart reconciliation preserves present file")
            persisted = session.get(BASE + "/v1/works", params={"q": "live-server-book"}, auth=basic, timeout=3).json()
            restored_files = persisted["works"][0]["editions"][0]["files"]
            require(len(persisted["works"]) == 1 and len(restored_files) == 1 and restored_files[0]["id"] == file["id"], "stable source rescan does not duplicate files")
            require(session.delete(BASE + "/v1/works/" + work["id"], headers=admin, timeout=3).status_code == 204, "post-test catalog cleanup")
            require(session.delete(BASE + "/v1/ai/memories/live-server/live-server-memory", headers=admin, timeout=3).status_code == 204, "post-test memory cleanup")
            require(session.delete(BASE + "/v1/sync/progress/live-user/live-book", headers=admin, timeout=3).status_code == 204, "post-test progress cleanup")
    except Exception as exc:
        status = "FAIL"
        checks.append({"name": str(exc), "status": "FAIL"})
    finally:
        if RESTART_DAEMONS:
            try:
                start_redis()
            except Exception as exc:
                status = "FAIL"
                checks.append({"name": "Redis restoration: " + str(exc), "status": "FAIL"})
        if process is not None:
            try:
                stop_server(process)
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
