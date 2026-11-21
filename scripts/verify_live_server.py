#!/usr/bin/env python3
"""Live server verification against real PostgreSQL, pgvector and Redis.

The test uses non-production loopback credentials, never prints them, and writes
only assertion names/statuses. Run the database-backed live Go tests separately;
this script exercises the actual server process, API protocols and restart.
"""
from __future__ import annotations

import getpass
import hashlib
import json
import os
import shutil
import signal
import subprocess
import sys
import time
import urllib.parse
import xml.etree.ElementTree as ET
import zipfile
from pathlib import Path

import requests

ROOT = Path(__file__).resolve().parents[1]
EVIDENCE = ROOT / "evidence" / "live-server"
SCRATCH = ROOT / ".helix-verify"
BINARY = SCRATCH / "bin" / "helix-server-live"
PSE_CHECK = SCRATCH / "bin" / "helix-pse-check"
GO = os.environ.get("HELIX_GO", "go")
LIBRARY = SCRATCH / "live-server-library"
STREAM_LIBRARY = SCRATCH / "live-server-streams"
BASE = "http://127.0.0.1:28080"
# Match the operating-system peer identity used to launch the disposable server.
# Both root and kimi have local test roles in the qualification environment; no
# password or production credential is embedded in this verifier. When the
# qualification environment exposes its test database elsewhere, HELIX_TEST_POSTGRES_DSN
# wins over the local peer-auth default.
DSN = os.environ.get("HELIX_TEST_POSTGRES_DSN") or f"postgres:///helix_test?host=/var/run/postgresql&user={getpass.getuser()}"
# Redis endpoint for the disposable test daemon. HELIX_TEST_REDIS_ADDR (host:port)
# points the verifier at an externally managed loopback service; the fallback is
# the historical local default.
REDIS_ADDR = os.environ.get("HELIX_TEST_REDIS_ADDR", "127.0.0.1:6379")
_redis_host, _redis_port = REDIS_ADDR.rsplit(":", 1)
checks: list[dict] = []
RESTART_DAEMONS = "--restart-daemons" in sys.argv[1:]


def require(condition: bool, name: str) -> None:
    if not condition:
        raise AssertionError(name)
    checks.append({"name": name, "status": "PASS"})


ATOM = "{http://www.w3.org/2005/Atom}"
PSE_NS = "http://vaemendis.net/opds-pse/ns"
PSE_REL = "http://vaemendis.net/opds-pse/stream"
PNG_SIG = b"\x89PNG\r\n\x1a\n"
JPEG_SIG = b"\xff\xd8\xff"
# The streamable fixture and the pages it must promise, in the order the
# archive enumerates them. Payloads carry their own signature so the advertised
# type is read from bytes, exactly as the server reads it.
PSE_PAGES = [("p01.png", PNG_SIG + b"page-one"), ("p02.png", PNG_SIG + b"page-two"), ("p03.png", PNG_SIG + b"page-three")]


def write_archive(path: Path, members: list[tuple[str, bytes]]) -> None:
    with zipfile.ZipFile(path, "w", zipfile.ZIP_DEFLATED) as archive:
        for name, payload in members:
            archive.writestr(name, payload)


def stream_fixtures() -> dict[str, str]:
    """Lay down one archive a page stream is promised for, plus the files it must
    be refused for, and map each fixture's sha256 to its file name.

    Digests, not titles or paths, identify the fixtures later: the catalog does
    not expose a file path, and a digest match proves the server streamed the
    bytes this harness wrote."""
    STREAM_LIBRARY.mkdir(parents=True, exist_ok=True)
    # Start from an empty directory: a fixture left by an earlier version of this
    # harness would silently change the file count the scan is asserted against.
    for stale in STREAM_LIBRARY.iterdir():
        stale.unlink()
    write_archive(STREAM_LIBRARY / "live-server-pse.cbz", PSE_PAGES)
    # Pages of two types have no single honest stream type, so they get no link.
    write_archive(STREAM_LIBRARY / "live-server-mixed.cbz", [("p01.png", PNG_SIG + b"a"), ("p02.jpg", JPEG_SIG + b"b")])
    # An extension is not proof of a type.
    write_archive(STREAM_LIBRARY / "live-server-forge.cbz", [("cover.png", b"an image by extension only")])
    # A registered container this server cannot enumerate pages for.
    (STREAM_LIBRARY / "live-server-cbr-stub.cbr").write_bytes(b"Rar!\x1a\x07\x01\x00not a zip archive")
    (STREAM_LIBRARY / "live-server-plain.txt").write_text("not pageable\n")
    digests = {}
    for path in sorted(STREAM_LIBRARY.iterdir()):
        digests[hashlib.sha256(path.read_bytes()).hexdigest()] = path.name
    return digests


def catalog_by_digest(session, basic, digests: dict[str, str]) -> dict[str, dict]:
    """Resolve fixture digests to the catalog records that describe them.

    Each fixture is looked up by its own title stem rather than by one broad
    query, because a missed search match would otherwise read as a missing
    file."""
    found: dict[str, dict] = {}
    for digest, name in digests.items():
        response = session.get(BASE + "/v1/works", params={"q": name.rsplit(".", 1)[0], "limit": 100}, auth=basic, timeout=3)
        response.raise_for_status()
        for work in response.json().get("works", []):
            for edition in work.get("editions", []):
                for file in edition.get("files", []):
                    if file.get("sha256") == digest:
                        found[name] = {"work_id": work["id"], "edition_id": edition["id"], "file": file}
    return found


def stream_links(entry) -> list[dict]:
    """The PSE links of one Atom entry element, as a client reads them."""
    links = []
    for link in entry.findall(ATOM + "link"):
        if (link.get("rel") or "") == PSE_REL:
            links.append({"href": link.get("href") or "", "type": link.get("type"), "count": link.get("{" + PSE_NS + "}count"), "attrs": sorted(link.attrib)})
    return links


def edition_streams(session, basic, edition_id: str) -> list[dict]:
    """Stream links advertised on the standalone OPDS 1.2 entry for an edition.

    The entry endpoint names the edition exactly, so a link appearing or
    disappearing is a statement about this file rather than an artifact of search
    or pagination."""
    response = session.get(BASE + "/opds/1.2/editions/" + urllib.parse.quote(edition_id, safe=""), auth=basic, timeout=3)
    response.raise_for_status()
    return stream_links(ET.fromstring(response.content))


def feed_streams(body: bytes) -> dict[str, list[dict]]:
    """Map edition id to the stream links its entry advertises in a feed."""
    by_edition: dict[str, list[dict]] = {}
    for entry in ET.fromstring(body).findall(ATOM + "entry"):
        edition = ""
        for link in entry.findall(ATOM + "link"):
            href = link.get("href") or ""
            if (link.get("rel") or "") == "alternate" and href.startswith("/opds/1.2/editions/"):
                edition = urllib.parse.unquote(href.rsplit("/", 1)[-1])
        by_edition[edition] = stream_links(entry)
    return by_edition


def run_pse_client(env: dict, name: str = "OPDS-PSE client round trip", evidence: str = "pse-client.json") -> dict:
    """Drive the live feed with the checker built from this tree, which parses
    the advertisement with core/pkg/pse and then fetches every promised page."""
    output = subprocess.run([str(PSE_CHECK), "-base", BASE, "-q", "live-server-pse"], capture_output=True, text=True, timeout=120, env=env)
    payload = {}
    try:
        payload = json.loads(output.stdout)
    except json.JSONDecodeError:
        payload = {"status": "FAIL", "failures": ["checker produced no JSON: " + output.stdout[:400] + output.stderr[:400]]}
    (EVIDENCE / evidence).write_text(json.dumps(payload, indent=2) + "\n")
    # The child's verdict, not the harness's reading of it, decides: a checker
    # that exited non-zero is a failure even if its JSON looks tidy.
    require(output.returncode == 0 and payload.get("status") == "PASS", name + ": " + str(payload.get("failures")))
    return payload


def server_environment() -> dict:
    env = os.environ.copy()
    env.update({
        "HELIX_ADDRESS": "127.0.0.1:28080",
        "HELIX_POSTGRES_DSN": DSN,
        "HELIX_REDIS_URL": "redis://" + REDIS_ADDR,
        "HELIX_AI_MEMORY": "true",
        "HELIX_EMBEDDING_DIMENSIONS": "3",
        "HELIX_ADMIN_TOKEN": "live-admin-token",
        "HELIX_OPDS_USERNAME": "live-reader",
        "HELIX_OPDS_PASSWORD": "live-password",
        "HELIX_KOSYNC_REGISTRATION": "true",
        "HELIX_HTTP3": "false",
        "HELIX_COVER_CACHE": str(SCRATCH / "live-server-covers"),
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


def redis_cli_arguments() -> list[str]:
    return ["redis-cli", "-h", _redis_host, "-p", _redis_port]


def redis_ready() -> bool:
    return subprocess.run(redis_cli_arguments() + ["ping"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=2).returncode == 0


def start_redis() -> None:
    if redis_ready():
        return
    if os.environ.get("HELIX_TEST_REDIS_ADDR"):
        raise RuntimeError("Redis not reachable at HELIX_TEST_REDIS_ADDR=" + REDIS_ADDR)
    # An externally managed daemon (for example in a container) is addressed via
    # HELIX_TEST_REDIS_ADDR. Without it we may launch a disposable local daemon
    # whose data directory stays inside the repository scratch tree. No
    # application or user credentials are involved. When redis-server is not
    # installed, record an honest SKIP; never simulate a pass.
    if shutil.which("redis-server") is None:
        checks.append({"name": "redis daemon start: NOT RUN (redis-server not installed; set HELIX_TEST_REDIS_ADDR to an externally managed loopback Redis)", "status": "SKIP"})
        return
    SCRATCH.mkdir(parents=True, exist_ok=True)
    subprocess.run([
        "redis-server", "--daemonize", "yes", "--dir", str(SCRATCH),
        "--dbfilename", "dump.rdb", "--appendonly", "no", "--save", "",
    ], check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=10)
    for _ in range(100):
        if redis_ready():
            return
        time.sleep(0.1)
    raise RuntimeError("Redis readiness after daemon restart")


def restart_daemons() -> None:
    subprocess.run(redis_cli_arguments() + ["SAVE"], check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=15)
    subprocess.run(redis_cli_arguments() + ["SHUTDOWN", "SAVE"], check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=15)
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
        for package, output in (("./cmd/helix-server", BINARY), ("./cmd/helix-pse-check", PSE_CHECK)):
            subprocess.run([GO, "build", "-trimpath", "-o", str(output), package], cwd=ROOT / "server", env=os.environ.copy(), check=True, stdout=log, stderr=subprocess.STDOUT, timeout=180)


def main() -> int:
    EVIDENCE.mkdir(parents=True, exist_ok=True)
    LIBRARY.mkdir(parents=True, exist_ok=True)
    (LIBRARY / "live-server-book.txt").write_text("live-server-book\n\nA deterministic paragraph for live ingestion.\n")
    digests = stream_fixtures()
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

            for query in ("live-server-book", "live-server-pse", "live-server-mixed", "live-server-forge", "live-server-cbr-stub", "live-server-plain"):
                existing = session.get(BASE + "/v1/works", params={"q": query, "limit": 100}, auth=basic, timeout=3)
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

            # --- OPDS-PSE page streaming ------------------------------------
            streams_source = session.post(BASE + "/v1/sources", headers=admin, json={"id": "live-server-streams", "protocol": "local", "root": str(STREAM_LIBRARY)}, timeout=3)
            require(streams_source.status_code == 201, "live page-stream source creation")
            stream_scan = session.post(BASE + "/v1/scans", headers=admin, json={"source_id": streams_source.json()["id"], "prefix": ""}, timeout=3)
            require(stream_scan.status_code == 202, "page-stream scan accepted")
            stream_job = None
            for _ in range(200):
                stream_job = session.get(BASE + "/v1/scans/" + stream_scan.json()["id"], headers=admin, timeout=3).json()
                if stream_job.get("state") in {"succeeded", "failed", "cancelled"}:
                    break
                time.sleep(0.05)
            require(stream_job is not None and stream_job.get("state") == "succeeded" and stream_job.get("report", {}).get("files") == len(digests),
                    "page-stream fixtures ingested: " + json.dumps(stream_job and stream_job.get("report")))

            catalogued = catalog_by_digest(session, basic, digests)
            require(len(catalogued) == len(digests), "every page-stream fixture is catalogued by digest: " + str(sorted(catalogued)))
            comic = catalogued["live-server-pse.cbz"]["file"]

            advertised = edition_streams(session, basic, catalogued["live-server-pse.cbz"]["edition_id"])
            require(len(advertised) == 1, "a CBZ edition advertises exactly one page stream")
            stream = advertised[0]
            require(stream["count"] == str(len(PSE_PAGES)), "pse:count equals the pages in the archive")
            require(stream["type"] == "image/png", "the advertised stream type comes from the page bytes, not the file name")
            require(stream["href"] == "/opds-pse/stream/" + comic["id"] + "/{pageNumber}", "the stream href names this file and the page template")
            require(not any("lastRead" in attribute for attribute in stream["attrs"]), "a stream the server cannot position carries no pse:lastRead")

            for refused in ("live-server-mixed.cbz", "live-server-forge.cbz", "live-server-cbr-stub.cbr", "live-server-plain.txt"):
                record = catalogued[refused]
                require(not edition_streams(session, basic, record["edition_id"]), refused + " is not advertised as pageable")
                # Refusing a stream must not cost access to the file itself.
                whole = session.get(BASE + "/v1/files/" + record["file"]["id"] + "/content", auth=basic, timeout=3)
                require(whole.status_code == 200 and whole.content, refused + " still ships whole")
                refused_page = session.get(BASE + "/opds-pse/stream/" + record["file"]["id"] + "/0", auth=basic, timeout=3)
                require(refused_page.status_code == 404, refused + " refuses page requests as well as the advertisement")

            for index, (_, payload) in enumerate(PSE_PAGES):
                page = session.get(BASE + "/opds-pse/stream/" + comic["id"] + "/" + str(index), auth=basic, timeout=3)
                require(page.status_code == 200 and page.content == payload and page.headers.get("Content-Type") == "image/png",
                        "page " + str(index) + " is served byte-identical to the archive member")
                require(page.headers.get("ETag") == '"$sha256-' + comic["sha256"] + '$page-' + str(index) + '"',
                        "page " + str(index) + " is tagged with the catalogued file digest")

            beyond = session.get(BASE + "/opds-pse/stream/" + comic["id"] + "/" + str(len(PSE_PAGES)), auth=basic, timeout=3)
            require(beyond.status_code == 404 and "0.." + str(len(PSE_PAGES) - 1) in beyond.text, "page past the last returns the valid range")
            require(session.get(BASE + "/opds-pse/stream/" + comic["id"] + "/notanumber", auth=basic, timeout=3).status_code == 400, "a non-integer page number is rejected")
            require(session.get(BASE + "/opds-pse/stream/no-such-file/0", auth=basic, timeout=3).status_code == 404, "an unknown file has no page stream")

            listed = session.get(BASE + "/opds/1.2/publications", params={"q": "live-server-pse", "limit": 100}, auth=basic, timeout=3)
            require(listed.status_code == 200 and len(feed_streams(listed.content).get(catalogued["live-server-pse.cbz"]["edition_id"], [])) == 1,
                    "the acquisition feed carries the stream link")
            publications20 = session.get(BASE + "/opds/2.0/publications", params={"q": "live-server-pse", "limit": 100}, auth=basic, timeout=3)
            require(publications20.status_code == 200 and "vaemendis" not in publications20.text, "OPDS 2.0 invents no page-stream extension")

            client = run_pse_client(server_environment())
            require(client["links"] >= 1 and client["pages"] == len(PSE_PAGES), "the PSE client fetched every advertised page")

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
            # The scan finished before the process stopped, so its record must be
            # readable by the new one. With --restart-daemons this also proves the
            # record survived a Redis save and reload, not only a client restart.
            restored_job = session.get(BASE + "/v1/scans/" + job_id, headers=admin, timeout=3)
            require(restored_job.status_code == 200 and restored_job.json().get("state") == "succeeded", "completed scan job survives server restart")
            listed = session.get(BASE + "/v1/scans", headers=admin, timeout=3).json().get("jobs", [])
            require(any(entry.get("id") == job_id for entry in listed), "restored scan job appears in the scan list")
            require(session.get(BASE + "/users/auth", headers=credentials, timeout=5).status_code == 200, "Redis KOSync account survives server restart")
            require(session.get(BASE + "/syncs/progress/live-server-document", headers=credentials, timeout=5).json()["progress"] == "42", "Redis KOSync progress survives server restart")
            internal_saved = session.get(BASE + "/v1/sync/progress/live-user/live-book", headers=admin, timeout=5)
            require(internal_saved.status_code == 200 and internal_saved.json()["Pos"]["Chapter"] == 2, "Redis internal progress survives server restart")
            reattached = session.post(BASE + "/v1/sources", headers=admin, json={"id": source_id, "protocol": "local", "root": str(LIBRARY)}, timeout=3)
            require(reattached.status_code == 201 and reattached.json()["id"] == source_id, "stable source identity reattaches after restart")
            restream = session.post(BASE + "/v1/sources", headers=admin, json={"id": "live-server-streams", "protocol": "local", "root": str(STREAM_LIBRARY)}, timeout=3)
            require(restream.status_code == 201, "page-stream source reattaches after restart")
            # The catalog record, the connector and the archive are all re-derived
            # by a new process: the advertisement is a property of the bytes, not
            # of state the first process happened to hold open.
            run_pse_client(server_environment(), "OPDS-PSE client round trip after restart", "pse-client-after-restart.json")
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
            left = [name for name, record in sorted(catalogued.items()) if session.delete(BASE + "/v1/works/" + record["work_id"], headers=admin, timeout=3).status_code != 204]
            require(not left, "post-test page-stream catalog cleanup failed for: " + str(left))
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
