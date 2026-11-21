#!/usr/bin/env python3
"""Aggregate the multi-process lock-scope payloads in this directory.

Reads every *.jsonl file and every LOADJSON line in every *.log file, keeps one
record per (run_tag, role, process_index), and prints per-arm throughput plus the
verify outcome for each run. Nothing here re-measures anything: it only reshapes
artifacts the runner already produced.
"""
import glob
import json
import os
from collections import OrderedDict

ARMS = ("disjoint", "contested_work", "shared_file_row")


def records(here):
    out = OrderedDict()
    for path in sorted(glob.glob(os.path.join(here, "*.jsonl"))):
        with open(path) as fh:
            for line in fh:
                line = line.strip()
                if not line:
                    continue
                try:
                    rec = json.loads(line)
                except ValueError:
                    continue
                out[(rec.get("run_tag"), rec.get("role"), rec.get("process_index"))] = rec
    for path in sorted(glob.glob(os.path.join(here, "*.log"))):
        with open(path, errors="replace") as fh:
            for line in fh:
                i = line.find("LOADJSON ")
                if i < 0:
                    continue
                try:
                    rec = json.loads(line[i + len("LOADJSON "):].strip())
                except ValueError:
                    continue
                key = (rec.get("run_tag"), rec.get("role"), rec.get("process_index"))
                out.setdefault(key, rec)
    return out


def main():
    here = os.path.dirname(os.path.abspath(__file__))
    runs = OrderedDict()
    for rec in records(here).values():
        tag = rec.get("run_tag")
        if not tag:
            continue
        runs.setdefault(tag, {"worker": [], "verify": []})
        runs[tag][rec.get("role", "worker")].append(rec)

    summary = []
    for tag, roles in sorted(runs.items()):
        workers = sorted(roles["worker"], key=lambda r: r.get("process_index") or 0)
        if not workers:
            continue
        procs = workers[0].get("processes")
        ops = workers[0].get("ops_per_arm")
        arms = {}
        for arm in ARMS:
            # Run 207361 predates the third arm and names the contended arm
            # "contested"; report what the payload actually holds instead of 0.
            key = arm if any(arm in w for w in workers) else next(
                (alt for alt, is_arm in (("contested", arm == "contested_work"),) if is_arm and any(alt in w for w in workers)),
                arm)
            rates = [w[key]["ops_per_second"] for w in workers if key in w]
            errs = sum(w[key].get("errors", 0) for w in workers if key in w)
            before = [w[key].get("db_deadlocks_before") for w in workers
                      if key in w and isinstance(w[key].get("db_deadlocks_before"), int)
                      and w[key]["db_deadlocks_before"] >= 0]
            after = [w[key].get("db_deadlocks_after") for w in workers
                     if key in w and isinstance(w[key].get("db_deadlocks_after"), int)
                     and w[key]["db_deadlocks_after"] >= 0]
            bound = (max(after) - min(before)) if before and after else None
            arms[arm] = {
                "payload_key": key,
                "sum_ops_per_second": round(sum(rates), 2) if rates else None,
                "processes_reporting": len(rates),
                "errors": errs,
                # pg_stat_database.deadlocks is database-wide and cumulative, and
                # concurrent runs share the container, so only the span across all
                # sampled processes is interpretable: a lower bound, never an exact
                # count for this run alone.
                "db_deadlocks_lower_bound": bound,
                "per_process_deadlock_deltas": [
                    w[key]["db_deadlocks_after"] - w[key]["db_deadlocks_before"]
                    for w in workers if key in w
                    and isinstance(w[key].get("db_deadlocks_after"), int)
                    and isinstance(w[key].get("db_deadlocks_before"), int)
                ],
            }
        row = {
            "run_tag": tag,
            "variant": tag.rstrip("0123456789"),
            "processes": procs,
            "ops_per_arm_per_process": ops,
            "total_merges": procs * ops * len(ARMS),
            "arms": arms,
        }
        for v in roles["verify"]:
            row["verify"] = {
                "missing_disjoint_works": v.get("missing_disjoint_works"),
                "contested_works_lost": v.get("contested_works_lost"),
                "editions_expected": v.get("editions_expected"),
                "editions_found": v.get("editions_found"),
                "duplicate_file_rows": v.get("duplicate_file_rows"),
                "orphan_file_rows": v.get("orphan_file_rows"),
                "shared_source_file_rows": v.get("shared_source_file_rows"),
                "shared_source_expected": v.get("shared_source_expected"),
            }
        summary.append(row)

    print(json.dumps({"runs": summary}, indent=2, sort_keys=True))
    wall = {}
    for row in summary:
        wall[row["run_tag"]] = row["arms"]["disjoint"]["sum_ops_per_second"]
    print("// disjoint-arm sum is the only arm that should scale with processes;",
          "contended arms are bounded by per-identity serialization by design.",
          file=__import__("sys").stderr)


if __name__ == "__main__":
    main()
