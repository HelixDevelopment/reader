#!/usr/bin/env python3
"""Re-run every BLOCKED reproducer in clients/supported-platforms.json and record real output.

Writes evidence/clients/blocked_reproducers.json. Run from anywhere:
    python3 clients/tools/capture_blocked.py
Each reproducer is executed with `bash -c`, 30 s timeout. Nothing is installed or modified.
"""
from __future__ import annotations

import datetime
import json
from pathlib import Path
import platform
import subprocess

ROOT = Path(__file__).resolve().parents[2]
MATRIX = ROOT / "clients" / "supported-platforms.json"
OUT = ROOT / "evidence" / "clients" / "blocked_reproducers.json"


def main() -> int:
    platforms = json.loads(MATRIX.read_text())["platforms"]
    results = []
    for p in platforms:
        if p["status"] != "BLOCKED":
            continue
        command = p["reproducer"]["command"]
        try:
            r = subprocess.run(["bash", "-c", command], capture_output=True, text=True, timeout=30)
            code, output = r.returncode, (r.stdout + r.stderr).strip()
        except subprocess.TimeoutExpired:
            code, output = 124, "timeout after 30s"
        results.append({"id": p["id"], "command": command, "exit_code": code, "output": output})
        print(f"{p['id']:<11} exit={code:<4} {output.splitlines()[-1] if output else ''}")
    doc = {"captured_at": datetime.datetime.now(datetime.timezone.utc).isoformat(timespec="seconds"),
           "host": platform.platform(), "reproducers": results}
    OUT.parent.mkdir(parents=True, exist_ok=True)
    OUT.write_text(json.dumps(doc, indent=2) + "\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
