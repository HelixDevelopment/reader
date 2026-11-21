#!/usr/bin/env python3
"""Needle gate for the vector-search load harness (campaign 2026-11-20).

Constitution 1.1: a test that cannot fail is not a test. Every measurement in
evidence/campaign-2026-11-20/vector/ rests on needles in
server/internal/aimemory/vector_load_test.go, so this script breaks one property
at a time, rebuilds the test binary, runs the harness, and requires the run to
FAIL for the reason that breakage should cause. A mutation the suite survives is
a blind needle: reported here, and the gate exits non-zero.

Each mutation is applied to a pristine copy of the two Go files and reverted
before the next one, so no two results share a cause. The originals are stashed
in /tmp/helix-vector-go-files.json and restored in a finally block; the script
refuses to start if it cannot write that backup, and refuses to restore into a
tree that differs from what it backed up plus one known mutation.

Runs small (5k rows, 64 dims) on purpose: this checks detection power, it does
not produce measurements.

Usage: ./needle_gate.py [M1 M7 ...]      (no arguments: run every mutation)
"""
import json
import os
import subprocess
import sys
import time
from pathlib import Path

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[2]
SERVER = ROOT / "server"
TARGET = SERVER / "internal/aimemory/vector_load_test.go"
POSTGRES_GO = SERVER / "internal/aimemory/postgres.go"
BACKUP = Path("/tmp/helix-vector-go-files.json")
LOG = HERE / "needle-gate.log"
RESULTS = HERE / "needle_gate_results.json"
BINARY = "/tmp/helix-vector-needle.test"

# 5k rows is small enough for a fast gate and large enough that the planner's
# choices here match the ones at measurement cardinality (the sweep in a real run
# records that from 5k upward).
ENV = {
    "HELIX_TEST_POSTGRES_DSN": os.environ.get(
        "HELIX_TEST_POSTGRES_DSN",
        "postgres://helixqual@127.0.0.1:34320/helix_test?sslmode=disable"),
    "HELIX_LOAD_RUN": "1",
    "HELIX_MEASUREMENT_PATH": "needle_gate",
    "HELIX_VECTOR_ROWS": "5000",
    "HELIX_VECTOR_DIMS": "64",
    "HELIX_VECTOR_QUERIES": "20",
    "HELIX_VECTOR_K": "10",
    "HELIX_LOAD_LEVELS": "1,8",
    "HELIX_LOAD_TIMEOUT_SECONDS": "90",
}

V = str(TARGET)
MUTATIONS = [
    {"id": "M1_hnsw_index_never_created", "file": V,
     "expect": ["create bench index", "drifted from production", "did not read the HNSW index"],
     "what": "the bench schema is created without the HNSW index at all",
     "find": '\t\tfmt.Sprintf("CREATE INDEX %s ON %s.%s USING hnsw(embedding vector_cosine_ops)", benchHNSWIndex, benchSchema, benchTable),\n',
     "replace": ""},
    {"id": "M2_index_forced_profile_neutralized", "file": V,
     "expect": ["did not read the HNSW index", "but the profile requires", "not reaching the session"],
     "what": "the index_forced profile keeps its label but loses its planner settings",
     "find": '{name: "index_forced", gucs: []string{pathGUC, "enable_seqscan=off", "enable_bitmapscan=off"}, ladder: true,',
     "replace": '{name: "index_forced", gucs: []string{pathGUC}, ladder: true,'},
    {"id": "M3_ladder_secretly_serial", "file": V,
     "expect": ["never had a second backend active", "throughput needle"],
     "what": "the concurrent ladder runs one client at a time but still labels the level by its client count",
     "find": "lats, errs, err := benchRun(ctx, clients, ops,",
     "replace": "lats, errs, err := benchRun(ctx, 1, ops,"},
    {"id": "M4_queries_ask_for_k_minus_1", "file": V,
     "expect": ["want 10 (cardinality"],
     "what": "every query silently asks for one fewer row than the configured k",
     "find": "q.QueryContext(ctx, sh.query, scope, VectorLiteral(vec), k)",
     "replace": "q.QueryContext(ctx, sh.query, scope, VectorLiteral(vec), k-1)"},
    {"id": "M5_fixture_collapsed_to_one_vector", "file": V,
     "expect": ["returned the probe row itself", "recall@10 is"],
     "what": "the synthetic corpus is all one repeated vector, so nothing has a nearest neighbour",
     "find": "x := next()*2 - 1",
     "replace": "x := next()*0 + 0.5"},
    {"id": "M6_search_path_never_applied", "file": V,
     "expect": ["want 10 (cardinality", "drifted from production", "unclassified", "measuring production's table rather than this fixture"],
     "what": "profiles connect without the bench search_path, so the unqualified statement runs against production's 3-row table",
     "find": 'arguments = append(arguments, "-c "+setting)',
     "replace": 'if !strings.Contains(setting, "search_path") {\n\t\t\t\targuments = append(arguments, "-c "+setting)\n\t\t\t}'},
    {"id": "M7_column_list_drifts", "file": V,
     "expect": ["drifted from production"],
     "what": "the bench table is created without the model column production selects",
     "find": "metadata JSONB NOT NULL DEFAULT '{}'::jsonb,model TEXT NOT NULL,",
     "replace": "metadata JSONB NOT NULL DEFAULT '{}'::jsonb,"},
    {"id": "M9_serial_ladder_with_blind_sampler", "file": V,
     "expect": ["throughput needle"],
     "what": "the ladder runs one client while the sampler also counts idle backends, so the peak-backend needle can no longer see the serialization",
     "find": "lats, errs, err := benchRun(ctx, clients, ops,",
     "replace": "lats, errs, err := benchRun(ctx, 1, ops,",
     "also": [{"find": "AND state = 'active'", "replace": "AND state IN ('active','idle')"}]},
    {"id": "M8_index_opclass_drifts", "file": V,
     "expect": ["drifted from production"],
     "what": "the bench HNSW index is built on L2 distance instead of cosine",
     "find": "USING hnsw(embedding vector_cosine_ops)",
     "replace": "USING hnsw(embedding vector_l2_ops)"},
]


def log(line):
    stamp = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
    text = f"[{stamp}] {line}"
    print(text, flush=True)
    with LOG.open("a") as handle:
        handle.write(text + "\n")


def sh(cmd, **kw):
    return subprocess.run(cmd, shell=isinstance(cmd, str), cwd=kw.pop("cwd", None),
                          env=kw.pop("env", None), timeout=kw.pop("timeout", 1200),
                          stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)


def main():
    wanted = set(sys.argv[1:])
    mutations = [m for m in MUTATIONS if not wanted or m["id"].split("_")[0] in wanted or m["id"] in wanted]
    if BACKUP.exists():
        sys.exit(f"refusing to run: {BACKUP} already exists, a previous run may not have restored the tree")
    originals = {p: p.read_text() for p in (TARGET, POSTGRES_GO)}
    BACKUP.write_text(json.dumps({str(p): t for p, t in originals.items()}))
    log(f"backed up {[str(p) for p in originals]} to {BACKUP}")
    results, dirty = [], False
    try:
        env = dict(os.environ)
        env["CGO_ENABLED"] = "0"
        for mutation in mutations:
            path = Path(mutation["file"])
            text = originals[path]
            pairs = [(mutation["find"], mutation["replace"])]
            pairs += [(extra["find"], extra["replace"]) for extra in mutation.get("also", [])]
            if any(text.count(find) != 1 for find, _ in pairs):
                counts = ", ".join(str(text.count(find)) for find, _ in pairs)
                log(f"{mutation['id']}: ANCHOR NOT UNIQUE ({counts} matches), mutation not applied")
                results.append({**mutation, "outcome": "anchor_error", "detected": False})
                continue
            for find, replace in pairs:
                text = text.replace(find, replace, 1)
            path.write_text(text)
            build = sh(["go", "test", "-c", "-o", BINARY, "./internal/aimemory"], cwd=str(SERVER), env=env)
            if build.returncode != 0:
                log(f"{mutation['id']}: build failed after mutation, which is itself a signal but not a needle result\n{build.stdout[-1500:]}")
                results.append({**mutation, "outcome": "build_failed", "detected": None})
                path.write_text(text)
                continue
            run = sh([BINARY, "-test.run", "TestLiveLoadVectorSearch", "-test.v", "-test.count=1",
                      "-test.timeout", "8m"], env={**env, **ENV}, timeout=900)
            output = run.stdout
            hits = [e for e in mutation["expect"] if e in output]
            detected = run.returncode != 0 and bool(hits)
            reason = ""
            for line in output.splitlines():
                if "vector_load_test.go:" in line and "LOADJSON" not in line and "FINDING" not in line:
                    reason = line.strip()[:400]
                    break
            log(f"{mutation['id']}: {'DETECTED' if detected else 'BLIND'} exit={run.returncode} matched={hits}")
            log(f"    first failure: {reason}")
            results.append({**mutation, "outcome": "detected" if detected else "blind",
                            "detected": detected, "exit": run.returncode,
                            "matched": hits, "first_failure": reason})
            path.write_text(text)
            if not detected:
                dirty = True
    finally:
        for path, text in originals.items():
            path.write_text(text)
        verify = {str(p): p.read_text() for p in originals}
        if verify != {str(p): t for p, t in originals.items()}:
            log("RESTORE VERIFICATION FAILED: the tree does not match the backup")
            dirty = True
        else:
            log("tree restored byte-for-byte from the backup")
        rebuild = sh(["go", "build", "./internal/aimemory"], cwd=str(SERVER))
        log(f"post-restore build exit={rebuild.returncode}")
        RESULTS.write_text(json.dumps({
            "generated_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
            "harness": "server/internal/aimemory/vector_load_test.go",
            "environment": ENV, "mutations": results,
            "all_detected": all(r["detected"] for r in results) and bool(results),
        }, indent=2) + "\n")
        BACKUP.unlink()
        log(f"results: {RESULTS}")
    if dirty or not all(r["detected"] for r in results):
        sys.exit(1)


if __name__ == "__main__":
    main()
