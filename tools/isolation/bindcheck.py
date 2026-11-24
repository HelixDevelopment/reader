#!/usr/bin/env python3
"""Bind-mount allow-list for run_container.sh.   usage: bindcheck.py ro|rw SRC DST [--bridge]
exit 0 and print the RESOLVED (symlink-free) source on stdout - run_container.sh hands THAT path to podman, so a symlink swapped after
the check cannot redirect the mount (TOCTOU); exit 64 and explain on stderr when refused.

Rules (all must hold):
  source   resolved path strictly inside an allowed root; never `/`, a system dir, $HOME or an ancestor of it; never inside, or an
           ancestor of, a credential/config dir ($HOME/.ssh .config .gnupg .aws .kube .docker .mozilla .local .password-store .claude* ...);
           no ':' ',' or control characters (they would inject mount options).
  rw       only the run's own FRESH subdirectory: under evidence/ it must contain no SHA256SUMS seal anywhere below it and no file older than
           ISO_FRESH_SECS (default 6 h) - a sealed or stale sibling is never writable; under ISO_WORK a sub-directory, never ISO_WORK itself.
  ISO_WORK absolute, canonical (no '..', no symlink in the chain), strictly inside a dedicated root (~/.cache/claude-tmp or ~/.cache/helix-iso),
           not inside/over a sensitive dir.
  hostile  (review round 2, class "container-writable state the host later acts on or follows") a rw source - and ISO_WORK - must not be, lie
           under, or contain a git checkout/worktree/gitdir (a `.git` dir OR pointer file: `core.hooksPath`, `.git/hooks`, `gitdir:` redirection),
           a `.claude*` / `.specify` / `.husky` / `.githooks` entry (host-run hooks/skills), or any symlink: all of these are host-executed or
           host-followed the moment a host tool (git, claude, make, cp) runs inside the tree after the container wrote to it.
  dest     absolute, canonical (normpath-equal: no '..', '//', '.', trailing '/'), charset [A-Za-z0-9_./-], below an allowed prefix, and it may
           never be, lie inside or cover the reserved mounts /opt/isoinside and /run/helixsock (the latter only via --bridge); run_container.sh
           additionally refuses duplicate destinations."""
import os, posixpath, re, sys, time, pathlib

DEDICATED = ("~/.cache/claude-tmp", "~/.cache/helix-iso")
SENSITIVE = (".ssh", ".config", ".gnupg", ".aws", ".kube", ".docker", ".mozilla", ".local", ".password-store", ".pki", ".var", ".thunderbird",
             ".cert", ".netrc", ".gitconfig", ".git-credentials", ".bash_history", ".claude", ".claude.json", ".claude-shared", ".claude-claude4",
             ".npmrc", ".pypirc", ".cargo/credentials.toml", "snap")
FORBIDDEN = {"/", "/tmp", "/dev/shm", "/var/tmp", "/home", "/root", "/etc", "/usr", "/var", "/run", "/proc", "/sys", "/dev", "/mnt", "/media", "/opt", "/srv", "/boot", "/lib", "/bin", "/sbin"}
DST_OK = ("/work", "/opt/", "/out", "/prof", "/fx", "/pw", "/ms-playwright", "/run/helixsock")
RESERVED = ("/opt/isoinside", "/run/helixsock")
DST_CHARS = re.compile(r"^/[A-Za-z0-9_./-]*$")

def inside(p, root):
    return p == root or p.startswith(root.rstrip("/") + "/")

def home_of():
    return os.path.realpath(os.path.expanduser("~"))

def sensitive_dirs(home):
    out = []
    for s in SENSITIVE:
        out.append(os.path.join(home, s))
    try:
        out += [os.path.join(home, n) for n in os.listdir(home) if n.startswith(".claude")]
    except OSError:
        pass
    return sorted(set(out))

def touches_sensitive(real, home):
    for s in sensitive_dirs(home):
        rs = os.path.realpath(s)
        if inside(real, rs) or inside(rs, real):          # inside it, or an ancestor that would expose it
            return rs
    return None

HOSTILE_EXACT = (".git", ".specify", ".husky", ".githooks")
SCAN_LIMIT = 50000

def hostile_name(n):
    return n in HOSTILE_EXACT or n.startswith(".claude")

def dedicated_roots():
    return [os.path.realpath(os.path.expanduser(r)) for r in DEDICATED]

def checkout_ancestor(path):
    """-> the nearest ancestor-or-self of `path`, below the dedicated roots, that holds a `.git` entry (dir, file or link), else None."""
    roots = dedicated_roots(); p = path
    while p not in roots and p != "/" and not any(p == os.path.dirname(r) for r in roots):
        if os.path.lexists(os.path.join(p, ".git")):
            return p
        p = os.path.dirname(p)
    return None

def hostile_tree_error(real):
    """-> reason when the tree below `real` holds a hostile entry (git pointer/gitdir, .claude*, hook dirs) or any symlink; None when clean."""
    n = 0
    for dp, dn, fn in os.walk(real, followlinks=False):
        for name in dn + fn:
            n += 1
            full = os.path.join(dp, name)
            if hostile_name(name):
                return f"{full} is a git/hook/agent-config entry ({name}): host tools would act on it"
            if os.path.islink(full):
                return f"{full} is a symlink: the host would follow it"
        if n > SCAN_LIMIT:
            return f"more than {SCAN_LIMIT} entries below {real}: refusing to certify it"
    return None

def iso_error(iso, home, repo=None):
    if not os.path.isabs(iso):
        return "ISO_WORK must be an absolute path"
    if posixpath.normpath(iso) != iso:
        return "ISO_WORK must be canonical (no '..', '//', '.', trailing '/')"
    if os.path.realpath(iso) != iso:
        return "ISO_WORK must not contain a symlink"
    roots = [os.path.realpath(os.path.expanduser(r)) for r in DEDICATED]
    if not any(inside(iso, r) and iso != r for r in roots):
        return f"ISO_WORK must be a directory strictly inside one of {', '.join(DEDICATED)}"
    t = touches_sensitive(iso, home)
    if t:
        return f"ISO_WORK overlaps the sensitive path {t}"
    c = checkout_ancestor(iso)
    if c:
        return f"ISO_WORK {iso} is, or lies inside, the git checkout/worktree {c}: the container could plant hooks or pointer files the host would run"
    try:
        top = [n for n in os.listdir(iso) if hostile_name(n)]
    except OSError:
        top = []
    if top:
        return f"ISO_WORK {iso} contains {top[0]} (git/hook/agent-config entry): refuse"
    if repo:   # the repo's host-executed tooling must never become writable: ISO_WORK may be neither the repo, inside it, nor an ancestor of it
        rr = os.path.realpath(repo)
        if inside(iso, rr) or inside(rr, iso):
            return f"ISO_WORK {iso} is the repository {rr}, inside it, or an ancestor of it: the container could rewrite host-executed tooling"
    return None

def fresh_error(real, now, fresh_secs):
    """rw evidence dir: nothing sealed below it and nothing old in it."""
    if not os.path.isdir(real):
        return f"{real} is not an existing directory (create the run's fresh directory first)"
    for dp, dn, fn in os.walk(real):
        for f in fn:
            if f == "SHA256SUMS" or f.startswith(".SHA256SUMS."):
                return f"{os.path.join(dp, f)} is a seal: sealed evidence is never writable"
            try:
                if now - os.lstat(os.path.join(dp, f)).st_mtime > fresh_secs:
                    return f"{os.path.join(dp, f)} is older than {fresh_secs}s: not this run's fresh directory"
            except OSError:
                pass
    return None

def check(mode, src, dst, repo, env=os.environ, bridge=False, now=None):
    """-> error string, or None when the bind is allowed (resolved path: resolve(src))."""
    home = home_of()
    repo = os.path.realpath(repo)
    iso = env.get("ISO_WORK")
    if iso:
        e = iso_error(iso, home, repo)
        if e:
            return e
    if any(c in src for c in ":,\n\r\0") or not src:
        return "source contains ':' ',' or a control character"
    real = os.path.realpath(src)
    if any(c in real for c in ":,\n\r\0"):
        return "resolved source contains ':' ',' or a control character (symlink target would inject mount options)"
    ro = [f"{repo}/clients", f"{repo}/qa", f"{repo}/tools/isolation", f"{repo}/submodules/containers/images",
          os.path.join(home, ".cache/ms-playwright"), "/usr/lib/koreader"]
    ev = os.path.realpath(f"{repo}/evidence")
    rw = [ev]
    if iso:
        ro.append(iso); rw.append(iso)
    ro, rw = [os.path.realpath(x) for x in ro], [os.path.realpath(x) for x in rw]
    if real in FORBIDDEN or real == home or inside(home, real):
        return f"source {real} is too broad (home/system path or an ancestor of $HOME)"
    t = touches_sensitive(real, home)
    if t:
        return f"source {real} overlaps the sensitive path {t}"
    allowed = rw if mode == "rw" else ro + rw   # rw roots are also readable
    if not any(inside(real, r) for r in allowed):
        return f"source {real} is outside the {mode} allow-list"
    if mode == "rw":
        if real in rw:
            return f"rw bind of a root ({real}) refused; bind a per-run subdirectory"
        c = None if inside(real, ev) else checkout_ancestor(real)
        if c:
            return f"rw source {real} is, or lies inside, the git checkout/worktree {c}"
        h = hostile_tree_error(real) if os.path.isdir(real) else None
        if h:
            return f"rw source refused: {h}"
        if inside(real, ev):
            e = fresh_error(real, now if now is not None else time.time(), int(env.get("ISO_FRESH_SECS", "21600")))
            if e:
                return e
    # destination
    if not dst.startswith("/") or not DST_CHARS.match(dst) or posixpath.normpath(dst) != dst or ".." in dst.split("/"):
        return f"destination {dst!r} is not an absolute canonical path"
    if not any(dst == d or dst.startswith(d if d.endswith('/') else d + "/") for d in DST_OK):
        return f"destination {dst} not permitted"
    for r in RESERVED:
        if inside(dst, r) or inside(r, dst):
            if bridge and dst == "/run/helixsock":
                continue
            return f"destination {dst} would overmount or cover the reserved mount {r}"
    return None

def resolve(src):
    return os.path.realpath(src)

if __name__ == "__main__":
    a = [x for x in sys.argv[1:] if x != "--bridge"]
    mode, src, dst = a[:3]
    repo = os.path.realpath(pathlib.Path(__file__).resolve().parents[2])
    err = check(mode, src, dst, repo, bridge="--bridge" in sys.argv)
    if err:
        print(f"run_container: bind refused ({mode} {src}:{dst}): {err}", file=sys.stderr); sys.exit(64)
    print(resolve(src))
