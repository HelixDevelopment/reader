#!/usr/bin/env python3
"""Anti-bluff qualification for the real SQLite workable-items registry."""
from __future__ import annotations

import argparse
import importlib.util
import json
import os
from pathlib import Path
import resource
import shutil
import signal
import sqlite3
import subprocess
import sys
import tempfile
import time

ROOT = Path(__file__).resolve().parents[1]
EVIDENCE = ROOT / "evidence" / "workable-items"
SPEC = importlib.util.spec_from_file_location("wi", ROOT / "scripts" / "workable_items.py")
wi = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(wi)
RESULTS = []


def record(name, status, detail, **extra):
    row = {"name": name, "status": status, "detail": detail, **extra}
    RESULTS.append(row)
    print(json.dumps(row, sort_keys=True), flush=True)


def run(name, command, cwd=ROOT):
    completed = subprocess.run(command, cwd=cwd, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    record(name, "PASS" if completed.returncode == 0 else "FAIL", completed.stdout[-6000:], returncode=completed.returncode, command=command)
    return completed.returncode == 0


def stress(root: Path):
    path = root / "stress.db"
    db = wi.connect(path)
    db.execute("BEGIN IMMEDIATE")
    now = "2026-10-29T00:00:00Z"
    for number in range(1, 1001):
        item = {
            "atm_id": f"ATM-{number:03d}", "type": "Task", "status": "Queued", "severity": "Low",
            "title": f"Stress registry row {number}", "description": "A complete stress fixture description with enough meaningful words and characters.",
            "affected_scope": "One thousand real SQLite registry rows.", "reproduction": "Insert one thousand rows in a real transaction.",
            "acceptance": "All rows persist with unique gapless identifiers and valid lifecycle fields.", "created_at": now, "last_modified": now,
            "current_location": "Issues", "forensic_anchor": "Stress fixture", "closure_criteria": "One thousand rows validate.", "composes_with": "[]",
            "evidence_path": "", "created_by": "AI", "assigned_to": "Operator", "reopens_count": 0,
        }
        db.execute(f"INSERT INTO items({','.join(wi.ITEM_COLUMNS)}) VALUES({','.join('?' for _ in wi.ITEM_COLUMNS)})", [item[c] for c in wi.ITEM_COLUMNS])
        db.execute("INSERT INTO item_history(atm_id,event_type,actor,on_date,reason,evidence_path) VALUES(?,?,?,?,?,?)", (item["atm_id"], "Opened", "AI", "2026-10-29", "stress-fixture", ""))
    db.execute("COMMIT")
    count = db.execute("SELECT count(*) FROM items").fetchone()[0]
    problems = wi.validate(db)
    check = db.execute("PRAGMA integrity_check").fetchone()[0]
    db.close()
    record("stress-1000-real-sqlite-rows", "PASS" if count == 1000 and not problems and check == "ok" else "FAIL", f"rows={count} integrity={check} validation_errors={problems}")


def kill_and_recovery(root: Path):
    path = root / "kill.db"
    db = wi.connect(path)
    db.close()
    child_code = '''
import importlib.util,os,sys,time
spec=importlib.util.spec_from_file_location("wi",sys.argv[1]);wi=importlib.util.module_from_spec(spec);spec.loader.exec_module(wi)
db=wi.connect(wi.Path(sys.argv[2]))
db.execute("CREATE TABLE committed_marker(id INTEGER PRIMARY KEY,value TEXT)")
db.execute("INSERT INTO committed_marker VALUES(1,'committed')")
db.execute("BEGIN IMMEDIATE")
for n in range(2,10000): db.execute("INSERT INTO committed_marker VALUES(?,?)",(n,"uncommitted"))
print("READY",flush=True)
time.sleep(60)
'''
    child = subprocess.Popen([sys.executable, "-c", child_code, str(ROOT / "scripts" / "workable_items.py"), str(path)], stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    ready = child.stdout.readline().strip()
    if ready == "READY":
        child.kill()
    child.wait(timeout=10)
    db = wi.connect(path)
    count = db.execute("SELECT count(*) FROM committed_marker").fetchone()[0]
    integrity = db.execute("PRAGMA integrity_check").fetchone()[0]
    db.close()
    record("sigkill-uncommitted-transaction-recovery", "PASS" if ready == "READY" and count == 1 and integrity == "ok" else "FAIL", f"ready={ready} committed_rows={count} integrity={integrity} child_returncode={child.returncode}")


def corruption_and_md_recovery(root: Path):
    source = ROOT / "docs" / "workable_items.db"
    corrupt = root / "corrupt.db"
    shutil.copyfile(source, corrupt)
    with corrupt.open("r+b") as handle:
        handle.seek(100)
        handle.write(b"CORRUPT-HELIX-TEST")
    detected = False
    message = ""
    try:
        db = sqlite3.connect(corrupt)
        db.execute("PRAGMA integrity_check").fetchall()
        db.close()
    except sqlite3.DatabaseError as exc:
        detected = True
        message = str(exc)
    recovered = wi.connect(root / "recovered-after-corruption.db")
    count = wi.import_markdown(recovered, ROOT / "docs")
    valid = wi.validate(recovered)
    recovered.close()
    record("corrupt-db-detection-and-md-recovery", "PASS" if detected and count == len(wi.items(wi.connect(source))) and not valid else "FAIL", f"corruption_detected={detected} error={message!r} recovered_rows={count} validation_errors={valid}")


def mutations(root: Path):
    cases = [
        ("closure-vocabulary-gate", 'if item["status"] in TERMINAL and item["status"] != "Obsolete (→ Fixed.md)" and item["status"] != CLOSED.get(item["type"]):', 'if False and item["status"] in TERMINAL:', "test_type_aware_closure_vocabulary"),
        ("markdown-drift-gate", 'if not path.exists() or path.read_bytes() != content.encode("utf-8"):', 'if False:', "test_export_and_drift_detection"),
        ("remote-push-anti-bluff", '"payload": payload, "remote_status": "NOT RUN",', '"payload": payload, "remote_status": "pushed",', "test_outbox_never_claims_remote_push"),
        ("attribution-roundtrip", 'created_by = "" if created_by == "UNKNOWN" else created_by', 'created_by = ""', "test_markdown_database_markdown_round_trip"),
    ]
    for name, old, new, test in cases:
        case = root / name
        (case / "scripts").mkdir(parents=True)
        (case / "tests").mkdir()
        shutil.copyfile(ROOT / "scripts" / "workable_items.py", case / "scripts" / "workable_items.py")
        shutil.copyfile(ROOT / "tests" / "test_workable_items.py", case / "tests" / "test_workable_items.py")
        (case / "tests" / "__init__.py").write_text("")
        path = case / "scripts" / "workable_items.py"
        text = path.read_text()
        if text.count(old) < 1:
            record("mutation-" + name, "FAIL", "mutation anchor missing")
            continue
        path.write_text(text.replace(old, new, 1))
        completed = subprocess.run([sys.executable, "-m", "unittest", "-v", f"tests.test_workable_items.WorkableItemsTest.{test}"], cwd=case, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        record("mutation-" + name, "PASS" if completed.returncode != 0 else "FAIL", completed.stdout[-3000:], mutant_rejected=completed.returncode != 0)


def main():
    EVIDENCE.mkdir(parents=True, exist_ok=True)
    run("registry-unit-integration", [sys.executable, "-m", "unittest", "-v", "tests.test_workable_items"])
    run("authoritative-registry-validation", [sys.executable, "scripts/workable_items.py", "validate"])
    run("authoritative-export-drift", [sys.executable, "scripts/workable_items.py", "diff"])
    with tempfile.TemporaryDirectory(prefix="helix-wi-qualification-") as temp:
        root = Path(temp)
        stress(root)
        kill_and_recovery(root)
        corruption_and_md_recovery(root)
        mutations(root)
    status = "PASS" if all(row["status"] == "PASS" for row in RESULTS) else "FAIL"
    envelope = {"schema": "helix.workable-items.qualification.v1", "status": status, "checks": RESULTS, "summary": {"pass": sum(r["status"] == "PASS" for r in RESULTS), "fail": sum(r["status"] != "PASS" for r in RESULTS)}}
    output = EVIDENCE / "results.json"
    output.write_text(json.dumps(envelope, indent=2, sort_keys=True) + "\n")
    print(json.dumps({"status": status, "output": str(output), **envelope["summary"]}, sort_keys=True))
    return status != "PASS"


if __name__ == "__main__":
    raise SystemExit(main())
