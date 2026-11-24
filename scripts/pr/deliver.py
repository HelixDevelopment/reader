#!/usr/bin/env python3
"""deliver.py - pull-request delivery tool of Helix Reader (constitution section 11.4.297, clauses A-F).

Reads config/delivery.conf (the ONE tracked declaration). Subcommands:
  plan           read-only: config check, branch name, trailers, base tip, ahead/behind, mirror parity table
  start          create the task branch at the remote base tip and switch to it
  open           verify trailers/evidence, push ff-only + ls-remote read-back, open or update the PR, request reviewers
  self-review    post the author's self-review (names the head it read) on the PR; never the GO
  review-record  attach the INDEPENDENT source-correctness GO for exactly one head sha
  merge          merge ONLY when every condition of clause C(2) reads back; merge commit only; then read back the form
  mirror         fast-forward the mirror remotes to the merged main (ff-only, never force), PRESENT/MIRROR-LAG/UNREACHABLE
  retire         delete the task branch only when it is an ancestor of main on every declared upstream
Global --dry-run prints every mutating git/gh command and runs none of them.

Status words (11.4.297(G)): PROVISIONAL - exercised only against a stub platform (tests/test_pr_deliver.py) until a real
PR has been read back. No force, no squash, no rebase, no admin override exists in this tool; run() refuses them.
Only the python3 standard library, git and gh are used. No secret is read, printed or stored (gh holds its own auth).
"""
import argparse
import datetime
import hashlib
import json
import os
import re
import subprocess
import sys

GH = os.environ.get("HELIX_GH", "gh")
EXIT_REVIEWER_REPORT = 3  # PR opened/updated, reviewer step could not request anyone (11.4.297(D)(2), (D)(6))

CLOSED = {
    "mode": {"direct", "pr"},
    "landing_form": {"ff-only", "merge-commit"},
    "merger": {"author-agent", "human"},
    "direct_push_guard": {"off", "expected", "enforced"},
    "author_only_reviewer_policy": {"report-and-merge", "report-and-stop"},
    "go_record_location": {"pr-comment"},
}
REQUIRED = ["decision_id", "mode", "base", "pr_remote", "pr_repository", "branch_prefix", "pr_reviewers", "landing_form",
            "merger", "protected_branches", "direct_push_guard", "shared_single_files", "tool_written_paths",
            "evidence_dir", "merge_window_hours", "author_only_reviewer_policy", "roadmap_item_pattern"]
OPTIONAL = ["decision_date", "decision_id_round2", "pr_assignees", "mirror_remotes", "multi_task_reason_field",
            "go_record_location", "own_repositories"]
LIST_KEYS = {"pr_reviewers", "pr_assignees", "mirror_remotes", "protected_branches", "shared_single_files",
             "tool_written_paths", "own_repositories"}
FORBIDDEN_ARGS = {"--force", "-f", "--force-with-lease", "--squash", "--rebase", "--admin", "-D", "--mirror", "--auto",
                  "--delete-branch", "--no-verify"}  # MUT:guard
LOGIN_RE = re.compile(r"^[A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?$")
SHA_RE = re.compile(r"^[0-9a-f]{40}$")
EVIDENCE_RE = re.compile(r"(?:^|\s)(?:red|green|Evidence:)\s+(\S+)\s+sha256\s+([0-9a-fA-F]{64})\b", re.M)

DRY = False


class Refuse(Exception):
    """A rule of the delivery refused; nothing further is done."""


def say(msg=""):
    print(msg, flush=True)


# ----------------------------------------------------------------------------------------------- process layer
def _check_args(args):
    for a in args:
        if a in FORBIDDEN_ARGS or (a.startswith("+") and ":" in a) or a.startswith("--force"):
            raise Refuse("forbidden argument %r in %s (no force, squash, rebase, admin or delete-at-merge exists here)" % (a, args))


def run(args, mutating=False, cwd=None, check=True):
    """Run argv; return stdout. In --dry-run a mutating command is printed and not run (returns '')."""
    _check_args(args)  # MUT:guardcall
    if mutating and DRY:
        say("DRY-RUN: " + " ".join(args))
        return ""
    p = subprocess.run(args, cwd=cwd, capture_output=True)
    out = p.stdout.decode("utf-8", "replace")
    if check and p.returncode != 0:
        raise Refuse("%s failed (rc=%d): %s" % (" ".join(args[:4]), p.returncode, p.stderr.decode("utf-8", "replace").strip()[:600]))
    return out


def git(ctx, *a, mutating=False, check=True):
    return run(["git", "-C", ctx.repo] + list(a), mutating=mutating, check=check)


def gh(_ctx, *a, mutating=False, check=True):
    return run([GH] + list(a), mutating=mutating, check=check)


def ghjson(ctx, *a):
    out = gh(ctx, *a)
    try:
        return json.loads(out)
    except ValueError:
        raise Refuse("gh returned non-JSON for %s" % " ".join(a[:3]))


# ----------------------------------------------------------------------------------------------- configuration
def parse_config(path):
    if not os.path.isfile(path):
        raise Refuse("delivery configuration missing: %s" % path)
    cfg = {}
    for n, line in enumerate(open(path, encoding="utf-8"), 1):
        line = re.sub(r"#.*", "", line).strip()
        if not line:
            continue
        if "=" not in line:
            raise Refuse("%s:%d not key=value" % (path, n))
        k, v = line.split("=", 1)
        k, v = k.strip(), v.strip()
        if k not in REQUIRED and k not in OPTIONAL:
            raise Refuse("unknown key %r in %s (an unknown key makes the delivery refuse, 11.4.297(A))" % (k, path))
        if k in cfg:
            raise Refuse("duplicate key %r" % k)
        cfg[k] = [x for x in re.split(r"[ \t,]+", v) if x] if k in LIST_KEYS else v
    for k in REQUIRED:
        if k not in cfg:
            raise Refuse("required key %r absent from %s" % (k, path))
    for k, allowed in CLOSED.items():
        if k in cfg and cfg[k] not in allowed:
            raise Refuse("%s=%r outside its closed set %s" % (k, cfg[k], sorted(allowed)))
    if not cfg["pr_reviewers"] or any(not LOGIN_RE.match(x) for x in cfg["pr_reviewers"]):
        raise Refuse("pr_reviewers empty or malformed (11.4.297(D)(1))")
    if cfg["merge_window_hours"] != "none" and not cfg["merge_window_hours"].isdigit():
        raise Refuse("merge_window_hours must be a whole number or 'none'")
    try:
        re.compile(cfg["roadmap_item_pattern"])
    except re.error as e:
        raise Refuse("roadmap_item_pattern invalid: %s" % e)
    cfg.setdefault("mirror_remotes", [])
    cfg.setdefault("multi_task_reason_field", "Multi-Task-Reason")
    cfg.setdefault("go_record_location", "pr-comment")
    return cfg


class Ctx:
    def __init__(self, repo, config, task=None, slug=None):
        self.repo = os.path.abspath(repo)
        self.cfg = parse_config(config or os.path.join(self.repo, "config", "delivery.conf"))
        self.task, self.slug = task, slug
        if task is not None:
            if not re.fullmatch(self.cfg["roadmap_item_pattern"], task):
                raise Refuse("task id %r does not match roadmap_item_pattern" % task)
            if not slug or not re.fullmatch(r"[a-z0-9][a-z0-9-]{0,60}", slug):
                raise Refuse("--slug must be lowercase [a-z0-9-]")
        gd = run(["git", "-C", self.repo, "rev-parse", "--git-common-dir"]).strip()
        self.state = os.path.join(gd if os.path.isabs(gd) else os.path.join(self.repo, gd), "helix-pr")

    @property
    def branch(self):
        return "%s/%s-%s" % (self.cfg["branch_prefix"], self.task, self.slug)

    @property
    def remote(self):
        return self.cfg["pr_remote"]

    @property
    def base(self):
        return self.cfg["base"]

    @property
    def repo_slug(self):
        return self.cfg["pr_repository"]


def require_pr_mode(ctx):
    if ctx.cfg["mode"] != "pr":
        raise Refuse("mode=%s: pull-request delivery is not declared; section 11.4.288 applies" % ctx.cfg["mode"])


# ----------------------------------------------------------------------------------------------- git helpers
def ls_remote(ctx, remote, ref):
    out = git(ctx, "ls-remote", remote, ref, check=False)
    for line in out.splitlines():
        sha, _, name = line.partition("\t")
        if name == ref and SHA_RE.match(sha):
            return sha
    return None


def fetch_base(ctx):
    """Fetch the base into its remote-tracking ref (no '+', no force); return the remote tip read by ls-remote."""
    tip = ls_remote(ctx, ctx.remote, "refs/heads/" + ctx.base)
    if tip is None:
        raise Refuse("cannot read %s/%s with ls-remote" % (ctx.remote, ctx.base))
    if not DRY:
        git(ctx, "fetch", "--no-tags", ctx.remote, "refs/heads/%s:refs/remotes/%s/%s" % (ctx.base, ctx.remote, ctx.base))
        local = git(ctx, "rev-parse", "refs/remotes/%s/%s" % (ctx.remote, ctx.base)).strip()
        if local != tip:
            raise Refuse("remote-tracking base %s differs from ls-remote %s" % (local[:12], tip[:12]))
    return tip


def has_obj(ctx, sha):
    return subprocess.run(["git", "-C", ctx.repo, "cat-file", "-e", sha + "^{commit}"], capture_output=True).returncode == 0


def is_anc(ctx, a, b):
    return subprocess.run(["git", "-C", ctx.repo, "merge-base", "--is-ancestor", a, b], capture_output=True).returncode == 0


def commits(ctx, base_tip, head):
    out = git(ctx, "log", "--topo-order", "--format=%H%x00%B%x1e", "%s..%s" % (base_tip, head))
    res = []
    for rec in out.split("\x1e"):
        rec = rec.strip("\n")
        if rec.strip():
            h, _, body = rec.partition("\x00")
            res.append((h.strip(), body))
    return res


# ----------------------------------------------------------------------------------------------- trailers and evidence
def check_trailers(ctx, base_tip, head):
    """11.4.297(B)(1),(2): plain Roadmap-Item on EVERY commit; two ids need a reason trailer. Returns list of problems."""
    idre = ctx.cfg["roadmap_item_pattern"]
    valid = re.compile(r"^%s([ ,]+%s)*$" % (idre, idre))
    reason = ctx.cfg["multi_task_reason_field"]
    problems = []
    cs = commits(ctx, base_tip, head)
    if not cs:
        problems.append("no commits between base %s and head %s" % (base_tip[:12], head[:12]))
    for h, body in cs:
        vals = re.findall(r"^Roadmap-Item:[ \t]*(.*?)[ \t]*$", body, re.M)
        if len(vals) != 1:
            problems.append("%s: needs exactly one Roadmap-Item trailer (found %d)" % (h[:12], len(vals)))  # MUT:trailer
            continue
        if not valid.match(vals[0]):
            problems.append("%s: Roadmap-Item value %r is not plain ids" % (h[:12], vals[0]))
            continue
        ids = re.split(r"[ ,]+", vals[0])
        if ctx.task and ctx.task not in ids:
            problems.append("%s: Roadmap-Item %s does not name the task %s" % (h[:12], vals[0], ctx.task))
        if len(set(ids)) > 1 and not re.search(r"^%s:[ \t]*\S" % re.escape(reason), body, re.M):
            problems.append("%s: two ids without a %s trailer" % (h[:12], reason))
    return problems


def check_evidence(ctx, base_tip, head):
    """11.4.297(E): every cited evidence path is repository-relative and under evidence_dir for EVERY citation; the cited sha256 must
    match the head tree for the LATEST citation (newest commit first in log order) of each path. An earlier citation of the same
    path in another commit is SUPERSEDED (a later commit legitimately regenerated the file): it must still match the path's content in
    ITS OWN citing commit's tree (absent there = refused), and is reported as a NOTE."""
    problems, cited = [], []
    evdir = ctx.cfg["evidence_dir"].strip("/") + "/"
    latest = {}      # path -> commit that holds the latest citation (the one checked against the head tree)
    superseded = {}  # path -> [short hashes of earlier citing commits]
    for h, body in commits(ctx, base_tip, head):  # git log order: newest first
        for path, sha in EVIDENCE_RE.findall(body):
            cited.append(path)
            if path.startswith("/") or ".." in path.split("/") or not path.startswith(evdir):
                problems.append("%s: evidence %r is absolute/outside/not under %s (INCONCLUSIVE, never PASS)" % (h[:12], path, evdir))
                continue
            is_latest = latest.setdefault(path, h) == h  # MUT:supersede
            if not is_latest and h[:12] not in superseded.setdefault(path, []):
                superseded[path].append(h[:12])
            # latest citation: head tree; earlier (superseded) citation: the path's content in ITS OWN citing commit
            tree = head if is_latest else h  # MUT:owntree
            p = subprocess.run(["git", "-C", ctx.repo, "show", "%s:%s" % (tree, path)], capture_output=True)
            if p.returncode != 0:
                problems.append("%s: evidence %s absent from %s tree %s" % (h[:12], path, "head" if is_latest else "its own commit", tree[:12]))
                continue
            if hashlib.sha256(p.stdout).hexdigest() != sha.lower():  # MUT:evhash
                problems.append("%s: evidence %s sha256 differs from the cited %s" % (h[:12], path, sha[:12]))
    for path, hs in superseded.items():
        say("NOTE: evidence %s: earlier citation(s) superseded by the latest in commit %s (each verified against its own commit tree): %s" % (path, latest[path][:12], ", ".join(hs)))
    return problems, cited


# ----------------------------------------------------------------------------------------------- platform helpers
def whoami(ctx):
    return gh(ctx, "api", "user", "--jq", ".login").strip()


def find_pr(ctx, branch):
    prs = ghjson(ctx, "pr", "list", "--repo", ctx.repo_slug, "--head", branch, "--state", "open",
                 "--json", "number,headRefName,headRefOid,url")
    if len(prs) > 1:
        raise Refuse("two open PRs for %s (one task, one PR)" % branch)
    return prs[0] if prs else None


def view_pr(ctx, number):
    return ghjson(ctx, "pr", "view", str(number), "--repo", ctx.repo_slug, "--json",
                  "number,state,headRefOid,baseRefName,reviewRequests,assignees,url,mergeCommit,comments,reviews,body,files")


def marker_in(pr, regex):
    texts = [c.get("body", "") for c in pr.get("comments", [])] + [r.get("body", "") for r in pr.get("reviews", [])]
    return [t for t in texts if re.search(regex, t, re.M)]


def sha256_file(path):
    return hashlib.sha256(open(path, "rb").read()).hexdigest()


# ----------------------------------------------------------------------------------------------- mirror parity
def parity(ctx, merged_tip):
    """Return rows (name, tip, verdict) per declared upstream; verdict PRESENT | MIRROR-LAG | UNREACHABLE."""
    rows = []
    for r in [ctx.remote] + ctx.cfg["mirror_remotes"]:
        tip = ls_remote(ctx, r, "refs/heads/" + ctx.base)
        if tip is None:
            rows.append((r, "-", "UNREACHABLE"))
            continue
        if tip == merged_tip:
            rows.append((r, tip, "PRESENT"))
            continue
        if not has_obj(ctx, tip) and not DRY:
            git(ctx, "fetch", "--no-tags", r, "refs/heads/%s:refs/remotes/%s/%s" % (ctx.base, r, ctx.base), check=False)
        if has_obj(ctx, tip) and is_anc(ctx, merged_tip, tip):
            rows.append((r, tip, "PRESENT"))
        else:
            rows.append((r, tip, "MIRROR-LAG"))
    return rows


def print_parity(rows, pinned):
    say("PARITY of %s on declared upstreams:" % pinned[:12])
    for n, t, v in rows:
        say("  %-10s %-14s %s" % (n, t[:12] if t != "-" else "-", v if v != "MIRROR-LAG" else "MIRROR-LAG " + t[:12]))


# ----------------------------------------------------------------------------------------------- subcommands
def cmd_plan(ctx, a):
    require_pr_mode(ctx)
    say("CONFIG OK: decision %s mode=%s base=%s remote=%s landing=%s merger=%s guard=%s" % (
        ctx.cfg["decision_id"], ctx.cfg["mode"], ctx.base, ctx.remote, ctx.cfg["landing_form"], ctx.cfg["merger"], ctx.cfg["direct_push_guard"]))
    say("BRANCH: %s" % ctx.branch)
    base_tip = ls_remote(ctx, ctx.remote, "refs/heads/" + ctx.base)
    say("BASE TIP (ls-remote %s/%s): %s" % (ctx.remote, ctx.base, base_tip or "UNREACHABLE"))
    exists = subprocess.run(["git", "-C", ctx.repo, "rev-parse", "--verify", "-q", "refs/heads/" + ctx.branch], capture_output=True)
    if exists.returncode != 0:
        say("LOCAL BRANCH: absent (run `deliver.py start`)")
    else:
        head = git(ctx, "rev-parse", "refs/heads/" + ctx.branch).strip()
        say("LOCAL BRANCH TIP: %s" % head)
        if base_tip and has_obj(ctx, base_tip):
            ahead = git(ctx, "rev-list", "--count", "%s..%s" % (base_tip, head)).strip()
            behind = git(ctx, "rev-list", "--count", "%s..%s" % (head, base_tip)).strip()
            say("AHEAD/BEHIND vs remote base tip: ahead=%s behind=%s" % (ahead, behind))
            probs = check_trailers(ctx, base_tip, head)
            ev, cited = check_evidence(ctx, base_tip, head)
            say("TRAILERS: %s" % ("OK" if not probs else "REFUSED: " + "; ".join(probs)))
            say("EVIDENCE: %s (%d cited)" % ("OK" if not ev else "REFUSED: " + "; ".join(ev), len(cited)))
            say("CONTAINS REMOTE BASE TIP: %s" % ("yes" if is_anc(ctx, base_tip, head) else "NO (merge the base into the branch; never rebase)"))
        else:
            say("AHEAD/BEHIND: UNKNOWN (remote base tip object not local; run `git fetch %s %s`)" % (ctx.remote, ctx.base))
    if base_tip:
        print_parity(parity(ctx, base_tip), base_tip)
    say("ENFORCEMENT: protected branch on platform UNCONFIRMED; direct-push guard '%s' not installed locally: UNCONFIRMED" % ctx.cfg["direct_push_guard"])
    return 0


def foreign_commits(ctx, base_tip):
    head = git(ctx, "rev-parse", "HEAD").strip()
    cur = subprocess.run(["git", "-C", ctx.repo, "symbolic-ref", "-q", "--short", "HEAD"], capture_output=True)
    if cur.returncode != 0:
        raise Refuse("detached HEAD refused (11.4.297(B)(3))")
    if cur.stdout.decode().strip() == ctx.branch:
        return []
    if not has_obj(ctx, base_tip):
        raise Refuse("base tip %s not local; fetch first" % base_tip[:12])
    return [h for h, _ in commits(ctx, base_tip, head)]


def cmd_start(ctx, a):
    require_pr_mode(ctx)
    base_tip = fetch_base(ctx)
    if subprocess.run(["git", "-C", ctx.repo, "rev-parse", "--verify", "-q", "refs/heads/" + ctx.branch], capture_output=True).returncode == 0:
        raise Refuse("branch %s already exists: reuse it (one task, one branch)" % ctx.branch)
    if ls_remote(ctx, ctx.remote, "refs/heads/" + ctx.branch):
        raise Refuse("branch %s exists on %s and not locally: fetch it, do not create a second branch" % (ctx.branch, ctx.remote))
    fc = foreign_commits(ctx, base_tip)
    if fc:
        raise Refuse("HEAD holds %d commit(s) the remote base lacks, they would ride into this PR: %s" % (len(fc), " ".join(x[:12] for x in fc)))
    git(ctx, "switch", "-c", ctx.branch, base_tip, mutating=True)
    say("STARTED %s at remote base tip %s" % (ctx.branch, base_tip))
    return 0


def pr_body(ctx, a, base_tip, head, cited, rows):
    lines = ["## Task", "", "Roadmap item: %s" % ctx.task, "Delivery decision: %s (config/delivery.conf)" % ctx.cfg["decision_id"], "",
             "## Evidence (repository-relative; latest citation of each path verified in the head tree, earlier citations in their own commit tree)", ""]
    lines += ["- `%s`" % p for p in sorted(set(cited))] or ["- none cited by the commits"]
    lines += ["", "## Test types run", "", a.test_types or "NOT STATED", "",
              "## Review record", "",
              "Independent source-correctness GO: PENDING (recorded by `deliver.py review-record` as a comment naming the head sha).",
              "Self-review: PENDING.", "",
              "## Mirror parity at body-writing time", ""]
    lines += ["- %s %s %s" % (n, t[:12], v) for n, t, v in rows]
    lines += ["", "Head at open: %s. Base tip at open: %s. Enforcement words: delivery tool PROVISIONAL; platform branch protection UNCONFIRMED." % (head, base_tip)]
    return "\n".join(lines)


def cmd_open(ctx, a):
    require_pr_mode(ctx)
    base_tip = fetch_base(ctx)
    ref = "refs/heads/" + ctx.branch
    if subprocess.run(["git", "-C", ctx.repo, "rev-parse", "--verify", "-q", ref], capture_output=True).returncode != 0:
        raise Refuse("local branch %s absent: run `deliver.py start` and commit on it" % ctx.branch)
    head = git(ctx, "rev-parse", ref).strip()
    if not is_anc(ctx, base_tip, head):  # MUT:basecontains
        raise Refuse("head %s does not contain the remote base tip %s: merge the base into the branch (never rebase)" % (head[:12], base_tip[:12]))
    probs = check_trailers(ctx, base_tip, head)
    ev, cited = check_evidence(ctx, base_tip, head)
    if probs or ev:
        raise Refuse("refused before anything is published:\n  " + "\n  ".join(probs + ev))
    # push: plain fast-forward-only push of the branch, then read it back with ls-remote
    git(ctx, "push", "--porcelain", ctx.remote, "%s:%s" % (ref, ref), mutating=True)  # MUT:push
    if not DRY:
        remote_tip = ls_remote(ctx, ctx.remote, ref)
        if remote_tip != head:
            raise Refuse("push read-back failed: remote tip %s != local %s" % (remote_tip, head))
        say("PUSHED %s -> %s/%s read back %s" % (ctx.branch, ctx.remote, ctx.branch, head))
    rows = parity(ctx, base_tip)
    title = a.title or "%s: %s" % (ctx.task, ctx.slug.replace("-", " "))
    body = pr_body(ctx, a, base_tip, head, cited, rows)
    if DRY:
        gh(ctx, "pr", "create", "--repo", ctx.repo_slug, "--base", ctx.base, "--head", ctx.branch, "--title", title, "--body", body, mutating=True)
        say("DRY-RUN: reviewers per config %s, then re-read" % ctx.cfg["pr_reviewers"])
        return 0
    pr = find_pr(ctx, ctx.branch)
    if pr is None:
        gh(ctx, "pr", "create", "--repo", ctx.repo_slug, "--base", ctx.base, "--head", ctx.branch, "--title", title, "--body", body, mutating=True)  # MUT:base
        pr = find_pr(ctx, ctx.branch)
        if pr is None:
            raise Refuse("PR not found after create")
    else:
        gh(ctx, "pr", "edit", str(pr["number"]), "--repo", ctx.repo_slug, "--body", body, mutating=True)
    author = whoami(ctx)
    rc = 0
    others = [r for r in ctx.cfg["pr_reviewers"] if r.lower() != author.lower()]
    refusal = ""
    if not others:
        say("REVIEWERS: author %s is the only listed account: no reviewer possible (11.4.297(D)(2)); policy=%s" % (author, ctx.cfg["author_only_reviewer_policy"]))
        refusal = "no reviewer possible: the only listed reviewer %s is the PR author; the platform refuses a review request from the author" % author
        rc = EXIT_REVIEWER_REPORT
    for r in others:
        out = subprocess.run([GH, "pr", "edit", str(pr["number"]), "--repo", ctx.repo_slug, "--add-reviewer", r], capture_output=True)
        if out.returncode != 0:
            refusal += "reviewer %s refused: %s; " % (r, out.stderr.decode().strip())
            rc = EXIT_REVIEWER_REPORT
    for asg in ctx.cfg.get("pr_assignees", []):
        subprocess.run([GH, "pr", "edit", str(pr["number"]), "--repo", ctx.repo_slug, "--add-assignee", asg], capture_output=True)
    if refusal:
        gh(ctx, "pr", "comment", str(pr["number"]), "--repo", ctx.repo_slug, "--body", "HELIX-REVIEWER-REPORT head=%s\n%s" % (head, refusal), mutating=True)
    back = view_pr(ctx, pr["number"])
    say("PR #%s %s state=%s head=%s base=%s" % (back["number"], back["url"], back["state"], back["headRefOid"], back["baseRefName"]))
    say("REVIEW REQUESTS (as the platform shows): %s" % ([x.get("login") for x in back.get("reviewRequests", [])] or "none"))
    say("ASSIGNEES (as the platform shows): %s" % ([x.get("login") for x in back.get("assignees", [])] or "none"))
    if back["headRefOid"] != head or back["baseRefName"] != ctx.base:
        raise Refuse("PR read-back disagrees: head %s base %s" % (back["headRefOid"], back["baseRefName"]))
    return rc


def pr_for(ctx):
    pr = find_pr(ctx, ctx.branch)
    if pr is None:
        raise Refuse("no open PR for %s" % ctx.branch)
    return pr


def cmd_self_review(ctx, a):
    require_pr_mode(ctx)
    pr = pr_for(ctx)
    head = pr["headRefOid"]
    if a.head != head:
        raise Refuse("self-review names head %s but the PR head is %s" % (a.head[:12], head[:12]))
    body = open(a.body_file, encoding="utf-8").read().strip()
    if len(body) < 80:
        raise Refuse("a self-review with no reading record is not a self-review (11.4.297(C)(1))")
    text = "HELIX-SELF-REVIEW head=%s findings-open=%d\n%s" % (head, a.findings_open, body)
    gh(ctx, "pr", "review", str(pr["number"]), "--repo", ctx.repo_slug, "--comment", "--body", text, mutating=True)
    say("SELF-REVIEW posted for head %s (this is NOT the GO)" % head)
    return 0


def go_path(ctx, head):
    return os.path.join(ctx.state, "go", head + ".json")


def cmd_review_record(ctx, a):
    require_pr_mode(ctx)
    pr = pr_for(ctx)
    head = pr["headRefOid"]
    if not SHA_RE.match(a.head):
        raise Refuse("--head must be a full 40-hex sha")
    if a.head != head:  # MUT:recordhead
        raise Refuse("GO is for head %s but the PR head is %s: a GO for another head is no GO" % (a.head[:12], head[:12]))
    if a.reviewer_label.strip().lower() == a.author_label.strip().lower():
        raise Refuse("producer != verifier (11.4.240): reviewer label equals author label")
    vf = os.path.abspath(a.verdict_file)
    rel = os.path.relpath(vf, ctx.repo)
    evdir = ctx.cfg["evidence_dir"].strip("/")
    if rel.startswith("..") or not (rel == evdir or rel.startswith(evdir + "/")):
        raise Refuse("verdict file must be under %s/ of the repository" % evdir)
    if not os.path.isfile(vf):
        raise Refuse("verdict file %s missing" % rel)
    txt = open(vf, encoding="utf-8", errors="replace").read()
    if not re.search(r"^VERDICT:[ \t]*GO[ \t]*$", txt, re.M) or not re.search(r"^HEAD:[ \t]*%s[ \t]*$" % a.head, txt, re.M):
        raise Refuse("verdict file must contain 'VERDICT: GO' and 'HEAD: %s'" % a.head)
    digest = sha256_file(vf)
    rec = {"head": a.head, "reviewer": a.reviewer_label, "author": a.author_label, "verdict_file": rel, "sha256": digest,
           "recorded_at": datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"), "pr": pr["number"]}
    if not DRY:
        os.makedirs(os.path.dirname(go_path(ctx, a.head)), exist_ok=True)
        with open(go_path(ctx, a.head), "w") as f:
            json.dump(rec, f, indent=1)
    gh(ctx, "pr", "comment", str(pr["number"]), "--repo", ctx.repo_slug, "--body",
       "HELIX-GO head=%s verdict=%s sha256=%s reviewer=%s" % (a.head, rel, digest, a.reviewer_label), mutating=True)
    say("GO RECORDED for head %s by %s (verdict %s sha256 %s)" % (a.head, a.reviewer_label, rel, digest))
    return 0


def cmd_merge(ctx, a):
    require_pr_mode(ctx)
    if ctx.cfg["merger"] != "author-agent":
        raise Refuse("merger=%s: under a human merge gate no agent merges (11.4.297(C)(2))" % ctx.cfg["merger"])
    if ctx.cfg["landing_form"] != "merge-commit":
        raise Refuse("landing_form=%s is not implemented by this tool (only merge-commit); squash and rebase are forbidden" % ctx.cfg["landing_form"])
    base_tip = fetch_base(ctx)
    pr = pr_for(ctx)
    view = view_pr(ctx, pr["number"])
    head = view["headRefOid"]
    need = []
    if view["state"] != "OPEN" or view["baseRefName"] != ctx.base:
        need.append("PR is %s against %s" % (view["state"], view["baseRefName"]))
    rtip = ls_remote(ctx, ctx.remote, "refs/heads/" + ctx.branch)
    if rtip != head:
        need.append("remote branch tip %s differs from PR head %s" % (rtip, head))
    if not DRY:
        git(ctx, "fetch", "--no-tags", ctx.remote, "refs/heads/%s:refs/remotes/%s/%s" % (ctx.branch, ctx.remote, ctx.branch), check=False)
    if not has_obj(ctx, head):
        need.append("head %s not fetched" % head[:12])
    else:
        if not is_anc(ctx, base_tip, head):  # MUT:mergebase
            need.append("(g) remote base tip %s is not an ancestor of head %s: merge the base into the branch, then re-review" % (base_tip[:12], head[:12]))
        need += check_trailers(ctx, base_tip, head)
        ev, _ = check_evidence(ctx, base_tip, head)
        need += ev
    # (a) the independent GO for exactly this head
    go = None
    if os.path.isfile(go_path(ctx, head)):
        go = json.load(open(go_path(ctx, head)))
    if not go:  # MUT:nogo
        need.append("(a) no recorded independent GO for head %s" % head[:12])
    else:
        if go["head"] != head:  # MUT:gohead
            need.append("(a) GO is for head %s, merging %s" % (go["head"][:12], head[:12]))
        vf = os.path.join(ctx.repo, go["verdict_file"])
        if not os.path.isfile(vf) or sha256_file(vf) != go["sha256"]:
            need.append("(a) GO verdict file changed or missing since it was recorded")
        if not marker_in(view, r"^HELIX-GO head=%s " % re.escape(head)):
            need.append("(a) the GO comment for this head is not on the platform")
    # (b) self-review of this head with every finding answered
    sr = marker_in(view, r"^HELIX-SELF-REVIEW head=%s findings-open=0\b" % re.escape(head))
    if not sr:
        need.append("(b) no self-review of head %s with findings-open=0 on the platform" % head[:12])
    # (c) reviewers
    others = [r for r in ctx.cfg["pr_reviewers"] if r.lower() != whoami(ctx).lower()]
    if others:
        shown = {x.get("login", "").lower() for x in view.get("reviewRequests", [])} | {x.get("author", {}).get("login", "").lower() for x in view.get("reviews", [])}
        miss = [r for r in others if r.lower() not in shown and not marker_in(view, r"^HELIX-REVIEWER-REPORT\b.*\b%s\b" % re.escape(r))]
        if miss:
            need.append("(c) reviewers not shown by the platform and not reported: %s" % miss)
    elif ctx.cfg["author_only_reviewer_policy"] != "report-and-merge" or not marker_in(view, r"^HELIX-REVIEWER-REPORT\b"):
        need.append("(c) author-only list: policy report-and-merge and a read-back HELIX-REVIEWER-REPORT comment are required")
    # (e) serial order on shared single files
    mine = {f["path"] for f in view.get("files", [])} & set(ctx.cfg["shared_single_files"])
    if mine:
        for other in ghjson(ctx, "pr", "list", "--repo", ctx.repo_slug, "--state", "open", "--json", "number,headRefName,files"):
            if other["number"] != view["number"] and {f["path"] for f in other.get("files", [])} & mine:
                need.append("(e) PR #%s also changes a shared single file %s: land them one at a time" % (other["number"], sorted(mine)))
    if need:
        raise Refuse("MERGE REFUSED:\n  " + "\n  ".join(need))
    margs = ["pr", "merge", str(view["number"]), "--repo", ctx.repo_slug, "--merge", "--match-head-commit", head]  # MUT:mergeflag
    gh(ctx, *margs, mutating=True)
    if DRY:
        return 0
    return readback_merge(ctx, view["number"], base_tip, head)


def readback_merge(ctx, number, prev_base_tip, head):
    after = view_pr(ctx, number)
    if after["state"] != "MERGED":
        raise Refuse("READ-BACK: PR #%s state %s, not MERGED" % (number, after["state"]))
    mc = (after.get("mergeCommit") or {}).get("oid")
    new_tip = fetch_base(ctx)
    if not mc:
        raise Refuse("READ-BACK: no merge commit id on the platform")
    parents = git(ctx, "rev-list", "--parents", "-n", "1", mc).split()[1:]
    form_ok = len(parents) == 2 and parents[0] == prev_base_tip and parents[1] == head
    anc = is_anc(ctx, head, "refs/remotes/%s/%s" % (ctx.remote, ctx.base))
    say("READ-BACK PR #%s state=MERGED merge-commit=%s parents=%s" % (number, mc, parents))
    say("FORM: %s (first parent = previous base tip, second = reviewed head); reviewed head ancestor of %s/%s: %s" % (
        "merge-commit OK" if form_ok else "VIOLATION", ctx.remote, ctx.base, anc))
    if not (form_ok and anc):
        raise Refuse("LANDING FORM VIOLATION (squash/rebase or stale parent): record it, file an item; the task branch is KEPT (11.4.297(C)(3))")
    say("LANDED on %s/%s tip %s. Next: `deliver.py mirror`, then `deliver.py retire`." % (ctx.remote, ctx.base, new_tip))
    return 0


def cmd_mirror(ctx, a):
    require_pr_mode(ctx)
    tip = fetch_base(ctx)
    rc = 0
    for m in ctx.cfg["mirror_remotes"]:
        mt = ls_remote(ctx, m, "refs/heads/" + ctx.base)
        if mt is None:
            say("MIRROR %s UNREACHABLE: parity UNCONFIRMED" % m)
            rc = 4
            continue
        if mt != tip:
            if not has_obj(ctx, mt) and not DRY:
                git(ctx, "fetch", "--no-tags", m, "refs/heads/%s:refs/remotes/%s/%s" % (ctx.base, m, ctx.base), check=False)
            if has_obj(ctx, mt) and not is_anc(ctx, mt, tip):
                say("MIRROR %s DIVERGED (its tip %s is not an ancestor of %s): refused, no force exists" % (m, mt[:12], tip[:12]))
                rc = 4
                continue
            git(ctx, "push", "--porcelain", m, "%s:refs/heads/%s" % (tip, ctx.base), mutating=True)  # MUT:mirrorpush
    if DRY:
        return 0
    rows = parity(ctx, tip)
    print_parity(rows, tip)
    if any(v != "PRESENT" for _, _, v in rows):
        rc = rc or 4
    return rc


def cmd_retire(ctx, a):
    require_pr_mode(ctx)
    _ = fetch_base(ctx)
    ref = "refs/heads/" + ctx.branch
    btip = ls_remote(ctx, ctx.remote, ref) or (git(ctx, "rev-parse", "-q", "--verify", ref, check=False).strip() or None)
    if not btip:
        raise Refuse("branch %s not found locally or remotely" % ctx.branch)
    if not has_obj(ctx, btip):
        raise Refuse("branch tip %s not local: cannot prove ancestry" % btip[:12])
    problems = []
    for r in [ctx.remote] + ctx.cfg["mirror_remotes"]:
        mt = ls_remote(ctx, r, "refs/heads/" + ctx.base)
        if mt is None:
            problems.append("%s UNREACHABLE" % r)
            continue
        if not has_obj(ctx, mt):
            git(ctx, "fetch", "--no-tags", r, "refs/heads/%s:refs/remotes/%s/%s" % (ctx.base, r, ctx.base), check=False)
        if not (has_obj(ctx, mt) and is_anc(ctx, btip, mt)):
            problems.append("%s main %s does not contain branch tip %s" % (r, mt[:12], btip[:12]))
    if problems:
        raise Refuse("RETIRE REFUSED (branch kept):\n  " + "\n  ".join(problems))
    if ls_remote(ctx, ctx.remote, ref):
        git(ctx, "push", ctx.remote, "--delete", ref, mutating=True)  # MUT:retiredelete
    if subprocess.run(["git", "-C", ctx.repo, "rev-parse", "--verify", "-q", ref], capture_output=True).returncode == 0:
        cur = subprocess.run(["git", "-C", ctx.repo, "symbolic-ref", "-q", "--short", "HEAD"], capture_output=True, text=True).stdout.strip()
        if cur == ctx.branch:  # a checked-out branch cannot be deleted: move to the base, fast-forward it only
            git(ctx, "switch", ctx.base, mutating=True)
            git(ctx, "merge", "--ff-only", "refs/remotes/%s/%s" % (ctx.remote, ctx.base), mutating=True)
        git(ctx, "branch", "-d", ctx.branch, mutating=True)
    git(ctx, "fetch", "--prune", ctx.remote, mutating=True)
    say("RETIRED %s (ancestor of %s on every declared upstream)" % (ctx.branch, ctx.base))
    return 0


def main(argv=None):
    global DRY
    ap = argparse.ArgumentParser(prog="deliver.py", description=(__doc__ or "").split("\n")[0])
    ap.add_argument("--repo", default=".")
    ap.add_argument("--config")
    ap.add_argument("--dry-run", action="store_true")
    sub = ap.add_subparsers(dest="cmd", required=True)

    def task(p):
        p.add_argument("--task", required=True)
        p.add_argument("--slug", required=True)
    for name in ("plan", "start", "retire", "mirror"):
        p = sub.add_parser(name)
        task(p)
    p = sub.add_parser("open")
    task(p)
    p.add_argument("--title")
    p.add_argument("--test-types")
    p = sub.add_parser("self-review")
    task(p)
    p.add_argument("--head", required=True)
    p.add_argument("--body-file", required=True)
    p.add_argument("--findings-open", type=int, default=0)
    p = sub.add_parser("review-record")
    task(p)
    p.add_argument("--head", required=True)
    p.add_argument("--reviewer-label", required=True)
    p.add_argument("--author-label", required=True)
    p.add_argument("--verdict-file", required=True)
    p = sub.add_parser("merge")
    task(p)
    a = ap.parse_args(argv)
    DRY = a.dry_run
    fn = {"plan": cmd_plan, "start": cmd_start, "open": cmd_open, "self-review": cmd_self_review, "review-record": cmd_review_record,
          "merge": cmd_merge, "mirror": cmd_mirror, "retire": cmd_retire}[a.cmd]
    try:
        ctx = Ctx(a.repo, a.config, a.task, a.slug)
        return fn(ctx, a)
    except Refuse as e:
        print("REFUSED: %s" % e, file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
