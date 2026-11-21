#!/usr/bin/env python3
"""Rebuild results.json from the append-only vector.jsonl for this directory.

vector.jsonl is the record: every run appends, nothing is edited in place, so an
earlier measurement stays visible after a later one supersedes it. results.json is
derived and is rewritten from scratch on every run of this script.
"""
import json
import re
from collections import defaultdict
from pathlib import Path

HERE = Path(__file__).resolve().parent
OUT = HERE / "results.json"


def load_rows():
    rows = []
    if (HERE / "vector.jsonl").exists():
        with (HERE / "vector.jsonl").open() as handle:
            for line in handle:
                line = line.strip()
                if line:
                    rows.append(json.loads(line))
    # Runs whose jsonl copy-back failed still left their payloads in the captured
    # go test output, so recover them rather than losing the measurement.
    for log in sorted(HERE.glob("raw-*.log")):
        tag = log.name[len("raw-"):-len(".log")]
        if any(r.get("run_tag") == tag for r in rows):
            continue
        recovered = 0
        with log.open() as handle:
            for line in handle:
                match = re.search(r"LOADJSON (\{.*\})\s*$", line)
                if match:
                    row = json.loads(match.group(1))
                    row["run_tag"] = tag
                    row["recovered_from_log"] = True
                    rows.append(row)
                    recovered += 1
        if recovered:
            print(f"recovered {recovered} rows for {tag} from {log.name}")
    return rows


def main():
    rows = load_rows()
    by_run = defaultdict(lambda: defaultdict(list))
    for row in rows:
        by_run[row.get("run_tag", "untagged")][row.get("role", "unknown")].append(row)

    runs = {}
    for tag, roles in sorted(by_run.items()):
        summaries = roles.get("summary", [])
        run = {
            "roles": {role: len(items) for role, items in sorted(roles.items())},
            "measurement_path": sorted({r.get("measurement_path") for r in rows if r.get("run_tag") == tag and r.get("measurement_path")}),
            "plan_at_cardinality": [
                {"cardinality": r.get("cardinality"), "statement": r.get("statement"),
                 "plan_class": r.get("plan_class"), "dimensions": r.get("dimensions")}
                for r in roles.get("plan_at_cardinality", [])
            ],
            "summaries": summaries,
        }
        # One run can hold several cardinalities (main + high); keep the largest
        # summary as the headline so the table generator has a stable target.
        if summaries:
            run["headline"] = max(summaries, key=lambda s: s.get("cardinality", 0))
        runs[tag] = run

    payload = {
        "test": "vector_search",
        "source_rows": len(rows),
        "runs": runs,
        "needle_gate": json.loads((HERE / "needle_gate_results.json").read_text())
        if (HERE / "needle_gate_results.json").exists() else None,
    }
    OUT.write_text(json.dumps(payload, indent=2) + "\n")
    print(f"{OUT} written: {len(rows)} rows across {len(runs)} runs")
    for tag, run in runs.items():
        headline = run.get("headline")
        if headline:
            print(f"  {tag}: {headline.get('cardinality')} rows x {headline.get('dimensions')} dims, "
                  f"production plan by default = {headline.get('production_plan_default')}")


if __name__ == "__main__":
    main()
