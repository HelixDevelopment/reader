#!/usr/bin/env python3
"""make_ui_mutants.py BASE_DIST OUT_ROOT   - build the product-negative-control dists for tools/verify uiverify.

Each mutant is BASE_DIST with exact-once string replacements applied to the built bundle (or one font file removed), so every
negative control is replayable: the replacements and the sha256 of every changed file are written to OUT_ROOT/<id>/mutation.json.
No browser is started here (host-safe, non-GUI). Run each mutant with qa/isolation/run_uiverify.sh inside the container and compare
the failing check ids with EXPECT (printed by `expect`).

  make_ui_mutants.py BASE_DIST OUT_ROOT    build all mutants
  make_ui_mutants.py expect                print id -> expected failing check ids (JSON)
"""
import hashlib, json, os, shutil, sys

MUTANTS = {
    "next-noop": dict(ops=[("js", "onClick:()=>k(Math.min(fl.length-1,B+1)),disabled:B===fl.length-1", "onClick:()=>k(B),disabled:B===fl.length-1")], fail=["reader-next-delta"]),
    "font-removed": dict(ops=[("rm", "fonts/NotoSans-SemiBold.woff")], fail=["console-errors", "missing-assets"]),
    "disclosure-text-changed": dict(ops=[("js", 'children:"No backend connected"', 'children:"Backend state unknown"'),
                                         ("js", "Sample data only · no backend connected", "Sample data only")], fail=["sample-data-disclosure"]),
    "source-validation-removed": dict(ops=[("js", "Enter a descriptive source name.", "Required.")], fail=["source-validation"]),
    "scan-state-text-changed": dict(ops=[("js", "Scan queued in prototype", "Scan done")], fail=["source-scan-state"]),
    "conflict-text-changed": dict(ops=[("js", "Conflict resolved", "Resolved")], fail=["sync-conflict-delta"]),
}

def sha(p):
    return hashlib.sha256(open(p, "rb").read()).hexdigest()

def tree_hash(root):
    h = hashlib.sha256()
    for d, _, fs in sorted(os.walk(root)):
        for f in sorted(fs):
            p = os.path.join(d, f)
            h.update(os.path.relpath(p, root).encode() + b"\0" + sha(p).encode() + b"\n")
    return h.hexdigest()

def main(argv):
    if argv[1:] == ["expect"]:
        print(json.dumps({k: v["fail"] for k, v in MUTANTS.items()}, indent=1)); return 0
    if len(argv) != 3:
        print(__doc__, file=sys.stderr); return 64
    base, out = os.path.realpath(argv[1]), os.path.realpath(argv[2])
    if "/.cache/claude-tmp/" not in out + "/" and "/.cache/helix-iso/" not in out + "/":
        print("make_ui_mutants: OUT_ROOT must be under ~/.cache/claude-tmp or ~/.cache/helix-iso", file=sys.stderr); return 64
    js = [f for f in os.listdir(os.path.join(base, "assets")) if f.endswith(".js")]
    assert len(js) == 1, js
    jsrel = "assets/" + js[0]
    os.makedirs(out, exist_ok=True)
    base_hash = tree_hash(base)
    for mid, m in MUTANTS.items():
        dst = os.path.join(out, mid, "dist")
        if os.path.isdir(os.path.join(out, mid)):
            if not os.path.isfile(os.path.join(out, mid, "mutation.json")):
                print(f"make_ui_mutants: {os.path.join(out, mid)} exists and was not made by this tool", file=sys.stderr); return 64
            shutil.rmtree(os.path.join(out, mid))
        shutil.copytree(base, dst)
        rec = []
        for op in m["ops"]:
            if op[0] == "js":
                p = os.path.join(dst, jsrel); text = open(p, encoding="utf-8").read()
                if text.count(op[1]) != 1:
                    print(f"make_ui_mutants: {mid}: anchor occurs {text.count(op[1])} times: {op[1]!r}", file=sys.stderr); return 1
                open(p, "w", encoding="utf-8").write(text.replace(op[1], op[2]))
                rec.append({"file": jsrel, "old": op[1], "new": op[2], "sha256_before": sha(os.path.join(base, jsrel)), "sha256_after": sha(p)})
            else:
                p = os.path.join(dst, op[1]); rec.append({"file": op[1], "removed": True, "sha256_before": sha(p)}); os.remove(p)
        json.dump({"id": mid, "base_tree_sha256": base_hash, "dist_tree_sha256": tree_hash(dst), "expect_failing_checks": m["fail"], "operations": rec},
                  open(os.path.join(out, mid, "mutation.json"), "w"), indent=1, ensure_ascii=False)
        print(mid, tree_hash(dst))
    print("base", base_hash)
    return 0

sys.exit(main(sys.argv))
