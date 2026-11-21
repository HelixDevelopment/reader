#!/usr/bin/env python3
"""Live backup/restore rehearsal with isolated restored PostgreSQL and Redis.

Qualification-only: uses the local peer-authenticated test database, disposable
loopback credentials, a separate restore database, and Redis on port 26379.
No credentials, DSNs, or backup contents are written to evidence.
"""
from __future__ import annotations

import base64
import getpass
import json
import os
import shutil
import signal
import subprocess
import sys
import time
from pathlib import Path
from urllib.parse import parse_qsl, unquote, urlsplit, urlunsplit

import requests

ROOT = Path(__file__).resolve().parents[1]
SCRATCH = ROOT / ".helix-verify"
WORK = SCRATCH / "backup-restore"
EVIDENCE = ROOT / "evidence" / "backup-restore"
BINARY = SCRATCH / "bin" / "helix-server-live"
GO = os.environ.get("HELIX_GO", "go")
LIBRARY = WORK / "library"
USER = getpass.getuser()
MAIN_DB = "helix_test"
RESTORE_DB = "helix_restore_test"
RESTORE_REDIS_PORT = 26379
# Optional env overrides keep the verifier host-portable. HELIX_TEST_POSTGRES_DSN
# points at the disposable test database (peer-auth socket default below); the
# restore database is derived from it by swapping the database name. Credentials
# from the DSN are passed to child processes via environment only and are never
# written to any file or log.
TEST_DSN = os.environ.get("HELIX_TEST_POSTGRES_DSN")
# Redis endpoint of the running source daemon. HELIX_TEST_REDIS_ADDR (host:port)
# wins; the fallback is the historical local default.
REDIS_ADDR = os.environ.get("HELIX_TEST_REDIS_ADDR", "127.0.0.1:6379")
REDIS_HOST, REDIS_PORT = REDIS_ADDR.rsplit(":", 1)
checks: list[dict] = []
processes: list[subprocess.Popen] = []
work_id = ""


def require(condition: bool, name: str) -> None:
    if not condition:
        raise AssertionError(name)
    checks.append({"name": name, "status": "PASS"})


def run(command: list[str], timeout: int = 60, env: dict | None = None) -> None:
    subprocess.run(command, check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=timeout, env=env)


def dsn(database: str) -> str:
    if TEST_DSN:
        parsed = urlsplit(TEST_DSN)
        return urlunsplit((parsed.scheme, parsed.netloc, "/" + database, parsed.query, parsed.fragment))
    return f"postgres:///{database}?host=/var/run/postgresql&user={USER}"


def pg_environment(database: str) -> dict:
    """Environment for psql/pg_dump/createdb/dropdb/pg_restore so maintenance
    commands reach the same endpoint selected by HELIX_TEST_POSTGRES_DSN."""
    env = os.environ.copy()
    env["PGDATABASE"] = database
    if TEST_DSN:
        parsed = urlsplit(TEST_DSN)
        if parsed.username:
            env["PGUSER"] = unquote(parsed.username)
        if parsed.password:
            env["PGPASSWORD"] = unquote(parsed.password)
        if parsed.hostname:
            env["PGHOST"] = parsed.hostname
        if parsed.port:
            env["PGPORT"] = str(parsed.port)
        for key, value in parse_qsl(parsed.query, keep_blank_values=True):
            if key == "host":
                env["PGHOST"] = value
            elif key == "port":
                env["PGPORT"] = value
            elif key == "user":
                env["PGUSER"] = value
            elif key == "password":
                env["PGPASSWORD"] = value
    else:
        env["PGHOST"] = "/var/run/postgresql"
        env["PGUSER"] = USER
    return env


def redis_cli(port: str | int | None = None) -> list[str]:
    if port in (None, ""):
        return ["redis-cli", "-h", REDIS_HOST, "-p", REDIS_PORT]
    return ["redis-cli", "-h", "127.0.0.1", "-p", str(port)]


def start_server(port: int, database: str, redis_url: str, log_name: str):
    env = os.environ.copy()
    env.update({
        "HELIX_ADDRESS": f"127.0.0.1:{port}",
        "HELIX_POSTGRES_DSN": dsn(database),
        "HELIX_REDIS_URL": redis_url,
        "HELIX_AI_MEMORY": "true",
        "HELIX_EMBEDDING_DIMENSIONS": "3",
        "HELIX_ADMIN_TOKEN": "backup-admin-token",
        "HELIX_OPDS_USERNAME": "backup-reader",
        "HELIX_OPDS_PASSWORD": "backup-password",
        "HELIX_KOSYNC_REGISTRATION": "false",
        "HELIX_HTTP3": "false",
        "HELIX_COVER_CACHE": str(WORK / ("covers-" + str(port))),
        "GIN_MODE": "release",
    })
    log = (EVIDENCE / log_name).open("w")
    process = subprocess.Popen([str(BINARY)], env=env, stdout=log, stderr=subprocess.STDOUT)
    process.helix_log = log
    processes.append(process)
    session = requests.Session()
    session.trust_env = False
    base = f"http://127.0.0.1:{port}"
    # Cover the configured PostgreSQL, pgvector and Redis initialization budgets
    # instead of imposing a shorter, artificial cold-start deadline.
    for _ in range(900):
        if process.poll() is not None:
            raise RuntimeError(f"server {port} exited before readiness")
        try:
            if session.get(base + "/readyz", timeout=0.25).status_code == 200:
                return process, session, base
        except requests.RequestException:
            pass
        time.sleep(0.1)
    raise RuntimeError(f"server {port} readiness timeout")


def stop_server(process) -> None:
    if process.poll() is None:
        process.send_signal(signal.SIGTERM)
    process.wait(timeout=25)
    process.helix_log.close()
    if process in processes:
        processes.remove(process)


def redis_ready(port: int) -> bool:
    return subprocess.run(redis_cli(port) + ["ping"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=2).returncode == 0


def stop_restore_redis() -> None:
    if redis_ready(RESTORE_REDIS_PORT):
        subprocess.run(redis_cli(RESTORE_REDIS_PORT) + ["SHUTDOWN", "NOSAVE"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=10)


def cleanup_main_data() -> None:
    env = pg_environment(MAIN_DB)
    if work_id:
        subprocess.run([
            "psql", "-d", MAIN_DB,
            "-v", "ON_ERROR_STOP=1", "-v", "work_id=" + work_id,
            "-c", "DELETE FROM works WHERE id=:'work_id'",
        ], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=15, env=env)
    subprocess.run([
        "psql", "-d", MAIN_DB,
        "-v", "ON_ERROR_STOP=1", "-c", "DELETE FROM ai_memories WHERE id='backup-restore-memory'",
    ], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=15, env=env)
    part = lambda value: base64.urlsafe_b64encode(value.encode()).decode().rstrip("=")
    key = "helix:v1:progress:" + part("backup-restore-user") + ":" + part("backup-restore-book")
    subprocess.run(redis_cli() + ["DEL", key], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=10)


def drop_restore_db() -> None:
    subprocess.run(["dropdb", "--if-exists", RESTORE_DB], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=30, env=pg_environment("postgres"))


def build_server() -> None:
    """Build the exact tree under test; never reuse a stale qualification binary."""
    BINARY.parent.mkdir(parents=True, exist_ok=True)
    with (EVIDENCE / "build.log").open("w") as log:
        subprocess.run([GO, "build", "-trimpath", "-o", str(BINARY), "./cmd/helix-server"], cwd=ROOT / "server", env=os.environ.copy(), check=True, stdout=log, stderr=subprocess.STDOUT, timeout=180)


def main() -> int:
    global work_id
    EVIDENCE.mkdir(parents=True, exist_ok=True)
    if WORK.exists():
        shutil.rmtree(WORK)
    LIBRARY.mkdir(parents=True)
    (LIBRARY / "backup-restore-book.txt").write_text("backup-restore-book\n\nDeterministic recovery fixture.\n")
    status = "PASS"
    try:
        build_server()
        stop_restore_redis()
        drop_restore_db()
        process, session, base = start_server(28080, MAIN_DB, "redis://" + REDIS_ADDR, "source-server.log")
        admin = {"Authorization": "Bearer backup-admin-token"}
        basic = requests.auth.HTTPBasicAuth("backup-reader", "backup-password")
        # Deterministic rerun cleanup through the public management surface.
        for old in session.get(base + "/v1/works", params={"q": "backup-restore-book", "limit": 100}, auth=basic, timeout=3).json().get("works", []):
            session.delete(base + "/v1/works/" + old["id"], headers=admin, timeout=3).raise_for_status()
        session.delete(base + "/v1/ai/memories/backup-restore/backup-restore-memory", headers=admin, timeout=3)
        session.delete(base + "/v1/sync/progress/backup-restore-user/backup-restore-book", headers=admin, timeout=3)

        created = session.post(base + "/v1/sources", headers=admin, json={"protocol": "local", "root": str(LIBRARY)}, timeout=3)
        require(created.status_code == 201, "source fixture creation")
        scan = session.post(base + "/v1/scans", headers=admin, json={"source_id": created.json()["id"]}, timeout=3)
        require(scan.status_code == 202, "source fixture scan accepted")
        job_id = scan.json()["id"]
        for _ in range(150):
            job = session.get(base + "/v1/scans/" + job_id, headers=admin, timeout=3).json()
            if job.get("state") in {"succeeded", "failed", "canceled"}:
                break
            time.sleep(0.05)
        require(job.get("state") == "succeeded" and job.get("report", {}).get("files") == 1, "source fixture ingestion")
        works = session.get(base + "/v1/works", params={"q": "backup-restore-book"}, auth=basic, timeout=3).json().get("works", [])
        require(len(works) == 1, "source catalog fixture")
        work_id = works[0]["id"]
        memory = {"record": {"id": "backup-restore-memory", "scope": "backup-restore", "kind": "note", "content": "recovery memory", "model": "backup-model"}, "embedding": [1, 0, 0]}
        require(session.put(base + "/v1/ai/memories", headers=admin, json=memory, timeout=5).status_code == 200, "source vector fixture")
        progress = {"BookID": "ignored", "DeviceID": "backup-device", "Pos": {"Chapter": 9, "Block": 2, "Offset": 1}, "Clock": {"V": {"backup-device": 1}}}
        require(session.put(base + "/v1/sync/progress/backup-restore-user/backup-restore-book", headers=admin, json=progress, timeout=5).status_code == 200, "source progress fixture")
        stop_server(process)

        run(["pg_dump", "--format=custom", "--file=" + str(WORK / "catalog.dump"), "-d", MAIN_DB], 90, env=pg_environment(MAIN_DB))
        require((WORK / "catalog.dump").stat().st_size > 0, "PostgreSQL backup created")
        run(["createdb", RESTORE_DB], 30, env=pg_environment("postgres"))
        run(["pg_restore", "--exit-on-error", "--no-owner", "--no-privileges", "--dbname=" + RESTORE_DB, str(WORK / "catalog.dump")], 90, env=pg_environment(RESTORE_DB))
        require(True, "PostgreSQL restore into isolated database")

        run(redis_cli() + ["SAVE"], 20)
        # Copy the snapshot the source daemon just flushed. Its data directory
        # is discovered from the running daemon instead of assuming a fixed
        # path from a retired qualification machine.
        config_reply = subprocess.check_output(redis_cli() + ["CONFIG", "GET", "dir"], text=True, timeout=15)
        source_rdb = Path(config_reply.splitlines()[-1].strip()) / "dump.rdb"
        shutil.copy2(source_rdb, WORK / "restore.rdb")
        require((WORK / "restore.rdb").stat().st_size > 0, "Redis backup created")
        if shutil.which("redis-server") is None:
            raise RuntimeError("redis-server binary unavailable for the isolated restore daemon")
        run([
            "redis-server", "--bind", "127.0.0.1", "--port", str(RESTORE_REDIS_PORT),
            "--daemonize", "yes", "--dir", str(WORK), "--dbfilename", "restore.rdb",
            "--appendonly", "no", "--save", "", "--pidfile", str(WORK / "redis.pid"),
            "--logfile", str(WORK / "redis.log"),
        ], 15)
        for _ in range(100):
            if redis_ready(RESTORE_REDIS_PORT):
                break
            time.sleep(0.1)
        require(redis_ready(RESTORE_REDIS_PORT), "isolated Redis restore started")

        restored, session, base = start_server(28081, RESTORE_DB, f"redis://127.0.0.1:{RESTORE_REDIS_PORT}", "restored-server.log")
        restored_works = session.get(base + "/v1/works", params={"q": "backup-restore-book"}, auth=basic, timeout=3).json().get("works", [])
        require(len(restored_works) == 1 and restored_works[0]["id"] == work_id, "restored catalog verified through fresh server")
        found = session.post(base + "/v1/ai/memories/search", headers=admin, json={"scope": "backup-restore", "embedding": [1, 0, 0], "limit": 5}, timeout=5)
        require(found.status_code == 200 and any(item["record"]["id"] == "backup-restore-memory" for item in found.json()["results"]), "restored vector memory verified")
        saved = session.get(base + "/v1/sync/progress/backup-restore-user/backup-restore-book", headers=admin, timeout=5)
        require(saved.status_code == 200 and saved.json()["Pos"]["Chapter"] == 9, "restored reading progress verified")
        stop_server(restored)
    except Exception as exc:
        status = "FAIL"
        checks.append({"name": str(exc), "status": "FAIL"})
    finally:
        for process in list(processes):
            try:
                stop_server(process)
            except Exception as exc:
                status = "FAIL"
                checks.append({"name": "server cleanup: " + str(exc), "status": "FAIL"})
        try:
            stop_restore_redis()
            drop_restore_db()
            cleanup_main_data()
        except Exception as exc:
            status = "FAIL"
            checks.append({"name": "test-data cleanup: " + str(exc), "status": "FAIL"})
        passed = sum(item["status"] == "PASS" for item in checks)
        failed = sum(item["status"] == "FAIL" for item in checks)
        (EVIDENCE / "results.json").write_text(json.dumps({"status": status, "pass": passed, "fail": failed, "checks": checks}, indent=2) + "\n")
    print(json.dumps({"status": status, "pass": sum(x["status"] == "PASS" for x in checks), "fail": sum(x["status"] == "FAIL" for x in checks)}))
    return 0 if status == "PASS" else 1


if __name__ == "__main__":
    sys.exit(main())
