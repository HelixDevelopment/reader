#!/usr/bin/env python3
"""Publish linked, non-promotional test outcomes from preserved evidence."""
from __future__ import annotations

import hashlib
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "docs" / "testing" / "results"


def artifact(path: str):
    p = ROOT / path
    return {"path": path, "exists": p.is_file(), "sha256": hashlib.sha256(p.read_bytes()).hexdigest() if p.is_file() else None}


def result_json(path: str):
    p = ROOT / path
    return json.loads(p.read_text()) if p.is_file() else {}


def json_entry(identifier, category, path, note):
    data = result_json(path)
    return {"id": identifier, "category": category, "status": data.get("status", "FAIL"), "summary": f'{data.get("pass", data.get("summary", {}).get("pass", "?"))} pass / {data.get("fail", data.get("summary", {}).get("fail", "?"))} fail. {note}', "evidence": [artifact(path)]}


def log_entry(identifier, category, paths, note):
    artifacts = [artifact(path) for path in paths]
    missing = [a["path"] for a in artifacts if not a["exists"]]
    failed = []
    for path in paths:
        p = ROOT / path
        if p.is_file() and ("\nFAIL" in p.read_text(errors="replace") or "FAIL\t" in p.read_text(errors="replace")):
            failed.append(path)
    status = "FAIL" if missing or failed else "PASS"
    return {"id": identifier, "category": category, "status": status, "summary": note + (f" Missing: {missing}." if missing else "") + (f" Failure markers: {failed}." if failed else ""), "evidence": artifacts}


def manual(identifier, category, status, summary, paths=()):
    return {"id": identifier, "category": category, "status": status, "summary": summary, "evidence": [artifact(p) for p in paths]}


def collect():
    entries = [
        log_entry("core-unit-integration", "Core", ["evidence/core/final-go-test.log"], "Complete core Go test log."),
        log_entry("core-race", "Core", ["evidence/core/final-go-race.log"], "Core race-detector campaign."),
        log_entry("core-vet", "Core", ["evidence/core/final-go-vet.log"], "Core static vet campaign."),
        log_entry("core-fuzz", "Fuzz", [f"evidence/core/final-go-fuzz-{name}.log" for name in ("formats", "cbz", "epub", "fb2", "mobi", "txt")], "Six core format fuzz targets."),
        log_entry("server-unit-integration", "Server", ["evidence/server/final-go-test.log", "evidence/server/final-repeat-1.log", "evidence/server/final-repeat-2.log", "evidence/server/final-repeat-3.log"], "Server test and three repeat logs."),
        log_entry("server-race-vet", "Server", ["evidence/server/final-go-race.log", "evidence/server/final-go-vet.log"], "Server race and vet campaigns."),
        log_entry("server-range-fuzz", "Fuzz", ["evidence/server/final-go-fuzz-single-range.log"], "Single-range parser fuzz target."),
        log_entry("live-postgresql-redis", "Live services", ["evidence/server/final-go-live-services.log"], "Real PostgreSQL, pgvector, Redis KOSync, and internal progress tests."),
        log_entry("web-static-and-build", "Web", ["evidence/ui/final-npm-ci-dry-run.log", "evidence/ui/final-npm-lint.log", "evidence/ui/final-npm-build.log"], "Dependency dry run, lint, and production build."),
        log_entry("web-four-viewports", "Web", ["evidence/ui/final-run.log"], "64 browser checks across four viewports; sample-only interface, not native-client evidence."),
        json_entry("go-mutation", "Mutation", "evidence/go-mutation/results.json", "Twenty-one targeted mutants; original-tree hash check is included in the artifact."),
        json_entry("workable-items", "Governance", "evidence/workable-items/results.json", "Real SQLite lifecycle, stress, crash recovery, corruption recovery, and four tracker mutations."),
        manual("live-server-post-fix", "Live API", "PASS", "Three ordinary 37/37 runs and one daemon/PostgreSQL/Redis restart 38/38 run passed after the keepalive fix.", ["evidence/live-server/post-fix-repeat-01/results.json", "evidence/live-server/post-fix-repeat-02/results.json", "evidence/live-server/post-fix-repeat-03/results.json", "evidence/live-server/post-fix-daemon-restart/results.json"]),
        manual("live-server-earlier-failure", "Live API", "FAIL", "Preserved pre-fix daemon-restart failure: connection reset after the identity attachment response.", ["evidence/live-server-daemon-restart/results.json"]),
        manual("transport-post-fix", "Transport", "PASS", "Three consecutive instrumented 5/5 HTTP/1.1, HTTP/2, HTTP/3, TLS, and Alt-Svc reruns passed.", ["evidence/live-transport/post-compression-repeat-02/results.json", "evidence/live-transport/post-compression-repeat-03/results.json", "evidence/live-transport/post-compression-repeat-04/results.json"]),
        manual("transport-readiness-failure", "Transport", "FAIL", "One post-fix readiness attempt failed without diagnostics; direct reproduction passed and the verifier was instrumented before three passing reruns. The initial failure is not explained away.", ["evidence/live-transport/post-compression-failure-01/results.json"]),
        json_entry("backup-restore-post-fix", "Recovery", "evidence/backup-restore/post-compression/results.json", "Live backup/restore rerun after the compression change."),
        manual("helixqa-build", "HelixQA", "FAIL", "Actual CLI build failed because declared own-org dependency checkouts are incomplete. No HelixQA test pass is claimed.", ["evidence/helixqa-build-test.log", "submodules/qa/helix-deps.yaml"]),
        manual("repository-research", "Research", "PARTIAL", "106/111 static analyses are indexed; five targets remain unavailable and manual architecture/license review is incomplete.", ["research/analysis_summary.json"]),
        manual("enterprise-load", "Scalability", "NOT RUN", "No representative enterprise load, query-plan, saturation, or multi-node qualification has run.", ["docs/scalability-review.md"]),
        manual("native-clients", "Clients", "NOT DELIVERED", "No native Windows, macOS, Linux, Android, or iOS client artifacts or real-device validation exist.", ["docs/architecture.md"]),
        manual("device-demos", "Clients", "PARTIAL", "Four browser viewports are verified; native-device and low-end-hardware interactive demonstrations are not delivered.", ["evidence/ui/final-run.log"]),
        manual("container-runtime", "Deployment", "NOT RUN", "Container recipes exist, but Docker/Podman cold start, restart, upgrade, and rollback have not run.", ["deploy/docker-compose.yml", "server/Dockerfile"]),
        manual("firebase-distribution", "Deployment", "NOT RUN", "No project-bound Firebase distribution run or CLI evidence exists."),
        manual("external-interoperability", "Interoperability", "NOT RUN", "No physical KOReader, external OPDS client, or deployment-specific rclone provider validation is recorded."),
        manual("helixtrack-remote-sync", "Governance", "NOT RUN", "Only an offline outbox exists. Exact remote API compatibility, credentials, and push/pull receipts are unverified.", ["docs/tracking/helixtrack-outbox.json", "docs/workable-items.md"]),
        manual("final-archives", "Release", "NOT RUN", "Fresh final ZIP and tar.gz archives are withheld while requested scope and qualification remain open."),
    ]
    return entries


def main():
    OUT.mkdir(parents=True, exist_ok=True)
    entries = collect()
    counts = {}
    for entry in entries:
        counts[entry["status"]] = counts.get(entry["status"], 0) + 1
    envelope = {"schema": "helix.test-results.index.v1", "release_qualification": "NOT READY", "status_counts": counts, "entries": entries}
    (OUT / "summary.json").write_text(json.dumps(envelope, indent=2, sort_keys=True) + "\n")
    lines = ["# Helix test and qualification results", "", "This index is generated from preserved artifacts. A suite PASS is scoped to that suite; it is not a production-readiness or final-release claim. Historical failures remain visible.", "", "**Release qualification: NOT READY.**", "", "## Status rollup", ""]
    lines += [f"- {status}: {count}" for status, count in sorted(counts.items())]
    lines += ["", "| Result | Category | Status | Outcome | Evidence |", "|---|---|---|---|---|"]
    for entry in entries:
        evidence = "<br>".join(f'`{a["path"]}`' + ("" if a["exists"] else " (missing)") for a in entry["evidence"]) or "No artifact; non-run statement"
        lines.append(f'| {entry["id"]} | {entry["category"]} | **{entry["status"]}** | {entry["summary"].replace("|", "\\|")} | {evidence} |')
    lines += ["", "## Interpretation", "", "- `FAIL` entries are preserved defects or failed qualification attempts, not hidden by later passes.", "- `PARTIAL` means some evidence exists but the requested scope is incomplete.", "- `NOT RUN` means no qualifying execution occurred.", "- `NOT DELIVERED` means the requested implementation or artifact does not exist.", "- No result in this index authorizes final archives or a production-ready claim.", ""]
    (OUT / "index.md").write_text("\n".join(lines))
    print(json.dumps({"status": "PASS", "release_qualification": "NOT READY", "results": len(entries), "status_counts": counts}, sort_keys=True))


if __name__ == "__main__":
    main()
