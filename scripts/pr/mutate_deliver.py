#!/usr/bin/env python3
"""mutate_deliver.py - paired section 1.1 mutants of scripts/pr/deliver.py (constitution 11.4.297(H)(1)).

Each mutant is applied to a TEMP COPY (the real file is never touched) and tests/test_pr_deliver.py is run against it
through PR_DELIVER_PATH with --failfast. A mutant is KILLED when the suite fails. A control run on the unmutated copy must
pass first (11.4.273); a failing control makes every verdict INVALID. Exit 0 only when the control passes and all die.
Usage: python3 scripts/pr/mutate_deliver.py [--jobs N] [--only NAME]
"""
import argparse
import concurrent.futures
import os
import shutil
import subprocess
import sys
import tempfile

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
SRC = os.path.join(ROOT, "scripts", "pr", "deliver.py")
# name -> list of (old, new); every old must occur exactly once in the source
MUTANTS = {
    "squash-flag": [('"--merge", "--match-head-commit", head]', '"--squash", "--match-head-commit", head]')],
    "force-push-guard-on": [('"push", "--porcelain", ctx.remote, "%s:%s" % (ref, ref)', '"push", "--force", "--porcelain", ctx.remote, "%s:%s" % (ref, ref)')],
    "force-push-guard-off": [('"push", "--porcelain", ctx.remote, "%s:%s" % (ref, ref)', '"push", "--force", "--porcelain", ctx.remote, "%s:%s" % (ref, ref)'),
                             ("    _check_args(args)  # MUT:guardcall", "    pass  # MUT:guardcall")],
    "plus-refspec-guard-off": [('git(ctx, "push", "--porcelain", ctx.remote, "%s:%s" % (ref, ref), mutating=True)', 'git(ctx, "push", "--porcelain", ctx.remote, "+%s:%s" % (ref, ref), mutating=True)'),
                               ("    _check_args(args)  # MUT:guardcall", "    pass  # MUT:guardcall")],
    "missing-go": [("    if not go:  # MUT:nogo", "    if False:  # MUT:nogo"), ('        if go["head"] != head:  # MUT:gohead', "        if False:")],
    "head-mismatch-in-record": [("    if a.head != head:  # MUT:recordhead", "    if False:  # MUT:recordhead")],
    "missing-trailer": [("        if len(vals) != 1:", "        if False:")],
    "wrong-base": [('"--base", ctx.base, "--head", ctx.branch, "--title", title, "--body", body, mutating=True)  # MUT:base', '"--base", "develop", "--head", ctx.branch, "--title", title, "--body", body, mutating=True)  # MUT:base')],
    "evidence-hash-mismatch": [("hashlib.sha256(p.stdout).hexdigest() != sha.lower()", "False")],
    "evidence-all-commits-checked": [("            is_latest = latest.setdefault(path, h) == h  # MUT:supersede", "            is_latest = True  # MUT:supersede")],
    "evidence-check-superseded-not-latest": [("            is_latest = latest.setdefault(path, h) == h  # MUT:supersede", "            is_latest = latest.setdefault(path, h) != h  # MUT:supersede")],
    "evidence-drop-own-tree-check": [("            tree = head if is_latest else h  # MUT:owntree", "            tree = head  # MUT:owntree")],
    "evidence-superseded-unchecked": [("            if p.returncode != 0:\n                problems.append(\"%s: evidence %s absent from %s", "            if not is_latest:\n                continue\n            if p.returncode != 0:\n                problems.append(\"%s: evidence %s absent from %s")],
    "evidence-same-commit-duplicate-superseded": [("            is_latest = latest.setdefault(path, h) == h  # MUT:supersede", "            is_latest = latest.setdefault(path, h) == h and path not in seen_in_commit  # MUT:supersede\n            seen_in_commit.add(path)"),
        ("    for h, body in commits(ctx, base_tip, head):  # git log order: newest first", "    for h, body in commits(ctx, base_tip, head):  # git log order: newest first\n        seen_in_commit = set()")],
    "evidence-validity-only-for-latest": [('            if path.startswith("/") or ".." in path.split("/") or not path.startswith(evdir):', '            first = latest.setdefault(path, h) == h\n            if first and (path.startswith("/") or ".." in path.split("/") or not path.startswith(evdir)):')],
    "evidence-absent-not-reported": [("            if p.returncode != 0:\n                problems.append(\"%s: evidence %s absent", "            if False:\n                problems.append(\"%s: evidence %s absent")],
    "evidence-superseded-note-dropped": [('        say("NOTE: evidence %s: earlier', '        (lambda *a: None)("NOTE: evidence %s: earlier')],
    "stale-base-at-open": [("    if not is_anc(ctx, base_tip, head):  # MUT:basecontains", "    if False:  # MUT:basecontains")],
    "stale-base-at-merge": [("        if not is_anc(ctx, base_tip, head):  # MUT:mergebase", "        if False:  # MUT:mergebase")],
    "merge-without-head-pin": [('"--merge", "--match-head-commit", head]', '"--merge"]')],
    "no-self-review-needed": [("    if not sr:", "    if False:")],
    "no-reviewer-report-needed": [('or not marker_in(view, r"^HELIX-REVIEWER-REPORT\\b"):', "or False:")],
    "form-readback-off": [("    if not (form_ok and anc):", "    if False:")],
    "human-gate-ignored": [('    if ctx.cfg["merger"] != "author-agent":', "    if False:")],
    "mirror-force": [('"push", "--porcelain", m, "%s:refs/heads/%s"', '"push", "--force", "--porcelain", m, "%s:refs/heads/%s"')],
    "retire-without-ancestry": [("        if not (has_obj(ctx, mt) and is_anc(ctx, btip, mt)):", "        if False:")],
}
# Not testable cheaply (needs date-skewed parallel branches), no mutant: `log --topo-order` in commits().
# Equivalent mutant, documented and not counted: merge's `go["head"] != head` duplicates the head-keyed record file name.


def run_suite(path):
    env = dict(os.environ, PR_DELIVER_PATH=path)
    p = subprocess.run([sys.executable, "-m", "unittest", "--failfast", "tests.test_pr_deliver"], cwd=ROOT, env=env, capture_output=True, text=True)
    return p.returncode, (p.stderr or "")[-400:]


def make(name, edits, tmp):
    src = open(SRC).read()
    for old, new in edits:
        if src.count(old) != 1:
            raise SystemExit("mutant %s: pattern not found exactly once: %r" % (name, old[:60]))
        src = src.replace(old, new)
    d = os.path.join(tmp, name)
    os.makedirs(d)
    path = os.path.join(d, "deliver.py")
    open(path, "w").write(src)
    return path


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--jobs", type=int, default=3)
    ap.add_argument("--only")
    a = ap.parse_args()
    tmp = tempfile.mkdtemp(prefix="prflow_mut_")
    try:
        ctl = os.path.join(tmp, "control.py")
        shutil.copy(SRC, ctl)
        rc, tail = run_suite(ctl)
        print("CONTROL (unmutated copy): %s" % ("PASS" if rc == 0 else "FAIL -> all verdicts INVALID\n" + tail))
        if rc != 0:
            return 2
        names = [n for n in MUTANTS if not a.only or n == a.only]
        paths = {n: make(n, MUTANTS[n], tmp) for n in names}
        with concurrent.futures.ThreadPoolExecutor(a.jobs) as ex:
            res = dict(zip(names, ex.map(lambda n: run_suite(paths[n]), names)))
        survived = []
        for n in names:
            killed = res[n][0] != 0
            print("MUTANT %-26s %s" % (n, "KILLED" if killed else "SURVIVED"))
            if not killed:
                survived.append(n)
        print("MUTANTS: %d killed, %d survived" % (len(names) - len(survived), len(survived)))
        return 1 if survived else 0
    finally:
        shutil.rmtree(tmp, ignore_errors=True)


if __name__ == "__main__":
    sys.exit(main())
