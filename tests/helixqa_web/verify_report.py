#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
"""verify_report.py - HRD-038 post-run cross-check of the webrunner's report.json against the files it claims.

usage: verify_report.py <shots-dir> [--viewport 1920x937]
Exit 0 only when: report.json is a real file (not a symlink) with unique case ids; counts add up; the pinned
expected_cases / expected_steps are present and equal the real counts and every pinned step PASSed; every PASS step has
attempts >= 1, a page_state (with the pinned viewport) and a screenshot that is a regular non-symlink file with a safe name
inside the directory whose sha256 equals the recorded one; no PNG in the directory is unreferenced.
It re-derives everything from the files, so a hand-edited or truncated report.json cannot pass.
"""
import hashlib, json, os, re, sys

SAFE = re.compile(r"^[A-Za-z0-9._-]+-step\d\d\.png$")


def sha(p):
    h = hashlib.sha256()
    with open(p, "rb") as f:
        for b in iter(lambda: f.read(1 << 16), b""):
            h.update(b)
    return h.hexdigest()


def verify(d, viewport=(1920, 937)):
    errs = []
    rp = os.path.join(d, "report.json")
    if os.path.islink(rp) or not os.path.isfile(rp):
        return ["report.json missing or a symlink"]
    try:
        rep = json.load(open(rp))
    except ValueError as e:
        return [f"report.json unreadable: {e}"]
    cases = rep.get("cases")
    if not isinstance(cases, list) or not cases:
        return ["report has no cases"]
    ids = [c.get("id") for c in cases]
    if len(set(ids)) != len(ids):
        errs.append("duplicate case ids in report")
    n_pass = n_fail = n_skip = n_steps = 0
    referenced = set()
    for c in cases:
        for s in c.get("steps", []):
            n_steps += 1
            v = s.get("verdict")
            if v == "PASS":
                n_pass += 1
                if not isinstance(s.get("attempts"), int) or isinstance(s.get("attempts"), bool) or s["attempts"] < 1:
                    errs.append(f"{c.get('id')}/{s.get('name')}: attempts missing")
                pg = s.get("page_state")
                if not pg or [pg.get("viewport_width"), pg.get("viewport_height")] != list(viewport) or not pg.get("dom_text_sha256"):
                    errs.append(f"{c.get('id')}/{s.get('name')}: page_state missing or viewport != {viewport}")
                f = s.get("screenshot")
                if not f or not SAFE.match(f) or "/" in f:
                    errs.append(f"{c.get('id')}/{s.get('name')}: unsafe or missing screenshot name {f!r}")
                    continue
                p = os.path.join(d, f)
                if os.path.islink(p) or not os.path.isfile(p):
                    errs.append(f"{f}: not a regular file"); continue
                if sha(p) != s.get("screenshot_sha256"):
                    errs.append(f"{f}: sha256 differs from the report")
                referenced.add(f)
            elif v == "FAIL":
                n_fail += 1
            else:
                n_skip += 1
    if (rep.get("pass"), rep.get("fail"), rep.get("skip")) != (n_pass, n_fail, n_skip):
        errs.append(f"report totals {(rep.get('pass'), rep.get('fail'), rep.get('skip'))} != steps {(n_pass, n_fail, n_skip)}")
    if n_fail or n_skip:
        errs.append(f"{n_fail} FAIL / {n_skip} SKIP steps")
    ec, es = rep.get("expected_cases"), rep.get("expected_steps")
    if not isinstance(ec, int) or not isinstance(es, int):
        errs.append("report does not carry pinned expected_cases/expected_steps")
    else:
        if len(cases) != ec:
            errs.append(f"{len(cases)} cases != pinned {ec}")
        if n_steps != es or n_pass != es:
            errs.append(f"{n_steps} steps / {n_pass} PASS != pinned {es}")
    if rep.get("pin_errors"):
        errs.append(f"pin_errors present: {rep['pin_errors']}")
    for f in os.listdir(d):
        if f.endswith(".png") and f not in referenced:
            errs.append(f"unreferenced screenshot {f}")
    return errs


def main(argv):
    if not argv:
        print("usage: verify_report.py <shots-dir> [--viewport WxH]", file=sys.stderr); return 2
    vp = (1920, 937)
    if "--viewport" in argv:
        vp = tuple(int(x) for x in argv[argv.index("--viewport") + 1].split("x"))
    errs = verify(argv[0], vp)
    for e in errs:
        print("FAIL", e)
    print(f"verify_report: {len(errs)} defect(s)")
    return 1 if errs else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
