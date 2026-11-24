#!/usr/bin/env python3
"""Check that every evidence path stored in the workable-items registry and every repo-relative
docs/ or evidence/ link in the derived Markdown documents resolves in the working tree.

Usage: check_registry_references.py [--root DIR] [--db PATH] [--json]
Exit 0 = zero missing, 1 = missing references, 2 = cannot read the registry (BLIND, never a pass).
Control needle: a path known to exist (the DB itself) must resolve, or the instrument is blind.
"""
import argparse, json, os, re, sqlite3, sys

DOC_LINK = re.compile(r"(?<![\w/.-])((?:docs|evidence|scripts|tests|deploy|config|tools)/[\w./@+-]+)")

def evidence_paths(db):
    out = []
    for table, key in (("items", "atm_id"), ("item_history", "atm_id"), ("reopen_details", "atm_id"),
                       ("obsolete_details", "atm_id"), ("operator_block_details", "atm_id")):
        cols = [r[1] for r in db.execute(f"PRAGMA table_info({table})")]
        if key not in cols:
            continue
        for c in cols:
            if "evidence" not in c:
                continue
            for k, v in db.execute(f"SELECT {key}, {c} FROM {table}"):
                for part in (v or "").split(";"):
                    part = re.split(r"\s*\(|\s+and\s+", part.strip())[0].strip().rstrip("/")
                    if part and part.lower() not in ("none", "none recorded"):
                        out.append((table, k, part))
    return out

def main(argv=None):
    ap = argparse.ArgumentParser()
    ap.add_argument("--root", default=".")
    ap.add_argument("--db", default="docs/workable_items.db")
    ap.add_argument("--docs", nargs="*", default=["docs/Issues.md", "docs/Fixed.md", "docs/Issues_Summary.md", "docs/Fixed_Summary.md"])
    ap.add_argument("--json", action="store_true")
    a = ap.parse_args(argv)
    root = os.path.abspath(a.root)
    dbp = os.path.join(root, a.db)
    if not os.path.exists(dbp):
        print("BLIND: registry database not found", file=sys.stderr); return 2
    db = sqlite3.connect(f"file:{dbp}?mode=ro", uri=True)
    if not os.path.exists(os.path.join(root, a.db)):
        print("BLIND: control needle failed", file=sys.stderr); return 2
    missing = []
    seen = set()
    for table, k, p in evidence_paths(db):
        if (k, p) in seen: continue
        seen.add((k, p))
        if not os.path.exists(os.path.join(root, p.split("#")[0])):
            missing.append({"kind": "registry-evidence", "where": f"{table}:{k}", "path": p})
    for d in a.docs:
        f = os.path.join(root, d)
        if not os.path.exists(f):
            continue
        text = open(f, encoding="utf-8").read()
        found = {x.group(1) for x in DOC_LINK.finditer(text) if text[x.end(1):x.end(1) + 1] not in ("*", "{", "[")}
        for m in sorted(found):
            m = m.rstrip(".,;:)")
            if not os.path.exists(os.path.join(root, m.split("#")[0])):
                missing.append({"kind": "doc-link", "where": d, "path": m})
    res = {"status": "PASS" if not missing else "FAIL", "missing": len(missing), "checked_evidence": len(seen)}
    if a.json:
        res["items"] = missing
        print(json.dumps(res, indent=1))
    else:
        print(json.dumps(res))
        for m in missing: print(f"MISSING {m['kind']} {m['where']} {m['path']}")
    return 0 if not missing else 1

if __name__ == "__main__":
    sys.exit(main())
