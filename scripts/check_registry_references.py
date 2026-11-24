#!/usr/bin/env python3
"""Check that every evidence path stored in the workable-items registry and every repo-relative
link in the documents resolves in the working tree, and classify each missing path.

Usage: check_registry_references.py [--root DIR] [--db PATH] [--docs GLOB ...] [--ownership FILE] [--json]

Classes of a missing path:
  OWNED-BY-PENDING-PR       listed in the ownership map with a pull request number
  OWNED-BY-UNDELIVERED-WORK listed in the ownership map with pr null (no pull request yet)
  NOWHERE                   not in the tree and in no map entry (a real defect)
Exit 0 = nothing missing; 1 = something missing (see classes); 2 = BLIND (registry unreadable, the ownership
map malformed, or the control needle failed: no stored evidence path resolves at all), never a pass.
"""
import argparse, subprocess, glob, json, os, re, sqlite3, sys

DOC_LINK = re.compile(r"(?<![\w/.-])((?:docs|evidence|scripts|tests|deploy|config|tools|clients|core|server|README|CHANGELOG)[\w./@+-]*)")
BACKTICK = re.compile(r"`([^`\s]+)`")
REL_TOKEN = re.compile(r"[\w@+.-]+(?:/[\w@+.-]+)*/?")
DEFAULT_DOCS = ["docs/Issues*.md", "docs/Fixed*.md", "docs/upstream-issues/*.md", "docs/gap-register*.md", "docs/helixtrack-sync.md",
                "docs/constitution-update-plan-*.md", "docs/submodule-pins-*.md", "docs/workable-items.md", "README.md", "deploy/README.md"]
TABLES = ("items", "item_history", "reopen_details", "obsolete_details", "operator_block_details")


def evidence_paths(db):
    out = []
    for table in TABLES:
        cols = [r[1] for r in db.execute(f"PRAGMA table_info({table})")]
        if "atm_id" not in cols:
            continue
        for c in cols:
            if "evidence" not in c:
                continue
            for k, v in db.execute(f"SELECT atm_id, {c} FROM {table}"):
                for part in (v or "").split(";"):
                    part = re.split(r"\s*\(|\s+and\s+", part.strip())[0].strip().rstrip("/")
                    if part and part.lower() not in ("none", "none recorded"):
                        out.append((table, k, part))
    return out


def classify(path, entries):
    for e in entries:
        pre = e["prefix"].rstrip("/")
        if path == pre or path.startswith(pre + "/") or (pre.endswith("_") and path.startswith(pre)):
            return ("OWNED-BY-PENDING-PR" if e.get("pr") else "OWNED-BY-UNDELIVERED-WORK"), e
    return "NOWHERE", None


def main(argv=None):
    ap = argparse.ArgumentParser()
    ap.add_argument("--root", default=".")
    ap.add_argument("--db", default="docs/workable_items.db")
    ap.add_argument("--docs", nargs="*", default=DEFAULT_DOCS)
    ap.add_argument("--ownership", default="docs/tracking/pending_ownership.json")
    ap.add_argument("--json", action="store_true")
    a = ap.parse_args(argv)
    root = os.path.abspath(a.root)
    dbp = os.path.join(root, a.db)
    if not os.path.exists(dbp):
        print("BLIND: registry database not found", file=sys.stderr); return 2
    db = sqlite3.connect(f"file:{dbp}?mode=ro", uri=True)
    paths = evidence_paths(db)
    # Presence means TRACKED (git index), never merely present in the working tree: an ignored or
    # untracked file or an empty directory does not exist for a clean clone (11.4.215).
    try:
        listed = subprocess.run(["git", "-C", root, "ls-files", "-z"], capture_output=True, check=True).stdout.decode("utf-8", "replace")
    except (OSError, subprocess.CalledProcessError) as exc:
        print(f"BLIND: not a git checkout, tracked files cannot be listed: {exc}", file=sys.stderr); return 2
    tracked = {t for t in listed.split("\0") if t}
    tracked_dirs = set()
    for t in tracked:
        while "/" in t:
            t = t.rsplit("/", 1)[0]; tracked_dirs.add(t)
    def exists(p):
        q = os.path.normpath(p.split("#")[0]).rstrip("/")
        return q in tracked or q in tracked_dirs
    # Control needle: a stored path known to be present must resolve through the SAME instrument.
    if not any(exists(p) for _, _, p in paths):
        print("BLIND: control needle failed (no stored evidence path resolves)", file=sys.stderr); return 2
    entries = []
    omap = os.path.join(root, a.ownership)
    if os.path.exists(omap):
        try:
            entries = json.load(open(omap, encoding="utf-8"))["entries"]
            if not (isinstance(entries, list) and all(isinstance(e, dict) and isinstance(e.get("prefix"), str) and e["prefix"] and e.get("task") for e in entries)):
                raise ValueError("entries must be a list of objects with a non-empty prefix and a task")
        except (ValueError, KeyError, TypeError, OSError) as exc:
            print(f"BLIND: ownership map unreadable or malformed: {exc}", file=sys.stderr); return 2
    missing, seen = [], set()
    for table, k, p in paths:
        if (k, p) in seen: continue
        seen.add((k, p))
        if not exists(p):
            missing.append({"kind": "registry-evidence", "where": f"{table}:{k}", "path": p})
    docs = sorted({f for g in a.docs for f in glob.glob(os.path.join(root, g))})
    for f in docs:
        rel = os.path.relpath(f, root)
        text = open(f, encoding="utf-8", errors="replace").read()
        found = {x.group(1) for x in DOC_LINK.finditer(text) if text[x.end(1):x.end(1) + 1] not in ("*", "{", "[")}
        for m in sorted(found):
            m = m.rstrip(".,;:)")
            if "/" in m and not exists(m):
                missing.append({"kind": "doc-link", "where": rel, "path": m})
        # Paths cited RELATIVE to a directory the same document cites (for example `live/x.json` after
        # `evidence/c/h/`, an existing directory under evidence/): resolved against every cited directory and against the document's own directory; if it resolves under
        # none, it is reported under the first one. Limit: only backticked tokens with a slash or a file
        # extension are examined; a bare name such as `latest` cannot be told from prose.
        cited_dirs = [m.rstrip(".,;:)") for m in sorted(found) if m.rstrip(".,;:)").endswith("/") and m.startswith("evidence/") and m.count("/") >= 3 and exists(m)]
        if cited_dirs:
            for tok in sorted({t.group(1).rstrip(".,;:)") for t in BACKTICK.finditer(text)}):
                if not REL_TOKEN.fullmatch(tok) or not ("/" in tok or re.search(r"\.\w{2,5}$", tok)):
                    continue
                if any(tok.startswith(r) for r in ("docs/", "evidence/", "scripts/", "tests/", "deploy/", "config/", "tools/", "clients/", "core/", "server/", "README", "CHANGELOG")):
                    continue
                if not any(exists(d + tok) for d in cited_dirs) and not exists(os.path.join(os.path.dirname(rel), tok)):
                    missing.append({"kind": "doc-relative-link", "where": rel, "path": cited_dirs[0] + tok})
    for m in missing:
        m["class"], e = classify(m["path"], entries)
        if e: m["owner"] = {"task": e["task"], "pr": e.get("pr")}
    counts = {}
    for m in missing: counts[m["class"]] = counts.get(m["class"], 0) + 1
    res = {"status": "PASS" if not missing else "FAIL", "missing": len(missing), "by_class": counts,
           "checked_evidence": len(seen), "docs_scanned": len(docs)}
    if a.json:
        res["items"] = missing
        print(json.dumps(res, indent=1))
    else:
        print(json.dumps(res))
        for m in missing: print(f"MISSING {m['class']} {m['kind']} {m['where']} {m['path']}")
    return 0 if not missing else 1


if __name__ == "__main__":
    sys.exit(main())
