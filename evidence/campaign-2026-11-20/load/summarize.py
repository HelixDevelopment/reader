#!/usr/bin/env python3
"""Rebuild results.json from the *.jsonl measurement payloads in this
directory. Re-runnable: every LOADJSON line collected so far is included,
each tagged with the log file and measurement path it came from. Raw per-run
logs stay beside it; nothing here transforms a number."""
import glob
import json
import os
import sys
import time

HERE = os.path.dirname(os.path.abspath(__file__))

def main():
    runs = []
    for path in sorted(glob.glob(os.path.join(HERE, "*.jsonl"))):
        source = os.path.basename(path)
        with open(path) as handle:
            for line in handle:
                line = line.strip()
                if not line:
                    continue
                try:
                    payload = json.loads(line)
                except json.JSONDecodeError as exc:
                    print(f"{source}: skipping unparseable line: {exc}", file=sys.stderr)
                    continue
                payload["_source_file"] = source
                runs.append(payload)
    out = {
        "generated_at": time.strftime("%Y-%m-%dT%H:%M:%S%z"),
        "campaign": "2026-11-20 phase 1 / ATM-027 load evidence",
        "note": "candidate observations only; no service-level objective is ratified by the project owner and none is asserted here",
        "runs": runs,
    }
    target = os.path.join(HERE, "results.json")
    with open(target, "w") as handle:
        json.dump(out, handle, indent=2)
        handle.write("\n")
    print(f"results.json: {len(runs)} payload(s)")

if __name__ == "__main__":
    main()
