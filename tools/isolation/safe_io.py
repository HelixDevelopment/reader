#!/usr/bin/env python3
"""safe_io.py - symlink-safe host-side file access to directories a container could write to (review round 2, class R2-B1).

MODEL: once a container has had a rw bind of a directory, EVERYTHING in it is attacker-controlled - names, symlinks, hardlinks, fifos,
directories swapped for links. A host process that then runs `cp`, `tee -a`, `> f`, `sed -i`, `ls`, or `open(path + ".tmp", "w")` inside
that directory follows such a link and reads or overwrites a host file. Rules this helper enforces (every component below the trusted
base is opened with O_NOFOLLOW relative to a directory fd, so no path component can be redirected):
  read   regular file only (never symlink / fifo / device / directory), link count 1, size <= --max (default 64 MiB), no blocking open.
  write  the existing entry is unlinked (a link is removed, never followed) and the file is created with O_EXCL|O_NOFOLLOW.
  scan   every entry below DIR must be a regular file (link count 1) or a plain directory; anything else is reported and rc=1.
  first  first regular file in BASE/REL whose name matches GLOB (and the safe-name charset), printed as a bare name.
  deny   (review round 3, class "post-run trust of container-written trees") the SINGLE deny table below (DENY_EXACT / DENY_PREFIX) names every
         git / agent / tool control artifact (.git dir OR file, .gitattributes, .gitmodules, .gitignore, .claude*, CLAUDE.md, AGENTS.md, GEMINI.md,
         QWEN.md, .mcp.json, .envrc, .direnv, .vscode, .idea, .cursor*, .husky, .githooks, .specify, hook dirs ...): scan refuses it at ANY depth,
         and bindcheck.py (pre-run) imports the SAME table, so the pre-run and post-run checks cannot drift apart. A refused entry is never
         descended into, hashed, copied or handed to any host tool (no host git/claude/make runs on a tree that holds one).
  allow  `scan DIR --allow` additionally refuses every regular file whose extension is not in ALLOW_EXT (publish allow-list: png json jsonl log
         txt md html csv mp4 webm gif webp); the runners use it on the evidence dir before anything is sealed.
  caps   scan refuses a tree with more than MAX_FILES files, more than MAX_TOTAL bytes in total, one file above MAX_FILE bytes or more than
         MAX_DEPTH levels (env SAFE_IO_MAX_FILES / _TOTAL / _FILE / _DEPTH override; a malformed value is itself a refusal).
usage: safe_io.py scan DIR [--allow] | put DIR NAME (data on stdin) | copy SRCBASE SRCREL DSTDIR DSTNAME [--max N] | first BASE REL GLOB
BASE / DIR arguments are trusted host paths (not under container control); REL / NAME are names, never '..'.
exit: 0 ok, 1 refused (message on stderr), 2 usage."""
import fnmatch, os, re, stat, sys

NAME_OK = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._ -]{0,200}$")
MAX_DEFAULT = 64 * 1024 * 1024

# ---- the ONE deny table (imported by bindcheck.py: pre-run bind check and post-run scan share it) ----
DENY_EXACT = frozenset({".git", ".gitattributes", ".gitmodules", ".gitignore", ".gitconfig", ".git-credentials", ".git-hooks", ".githooks", ".husky",
                        ".hooks", "hooks", ".specify", "claude.md", "agents.md", "gemini.md", "qwen.md", ".mcp.json", ".envrc", ".direnv",
                        ".vscode", ".idea", ".cursorrules", ".pre-commit-config.yaml", ".npmrc", ".pypirc", ".netrc"})
DENY_PREFIX = (".claude", ".cursor", ".aider", ".gemini", ".qwen", ".codex", ".windsurf")

def deny_name(n):
    """True for a git / agent / tool control artifact name (case-insensitive: a case-folding file system must not hide one)."""
    ln = n.lower()
    return ln in DENY_EXACT or ln.startswith(DENY_PREFIX)

ALLOW_EXT = frozenset({"png", "json", "jsonl", "log", "txt", "md", "html", "csv", "mp4", "webm", "gif", "webp"})
CAP_DEFAULTS = {"FILES": 20000, "TOTAL": 4 * 1024 ** 3, "FILE": 2 * 1024 ** 3, "DEPTH": 12}

def caps(env=None):
    """-> (dict of caps, [config problems]); a malformed override is a refusal, never a silent default."""
    env = os.environ if env is None else env
    out, bad = {}, []
    for k, d in CAP_DEFAULTS.items():
        v = env.get("SAFE_IO_MAX_" + k)
        if v is None:
            out[k] = d
        elif re.fullmatch(r"[1-9][0-9]{0,15}", v):
            out[k] = int(v)
        else:
            out[k] = d; bad.append(f"SAFE_IO_MAX_{k}={v!r} is not a positive integer")
    return out, bad


class Refused(Exception):
    pass


def _comps(rel):
    parts = [p for p in rel.split("/") if p != ""]
    if not parts or any(p in (".", "..") or "\0" in p for p in parts):
        raise Refused(f"bad relative path {rel!r}")
    return parts


def _opendir(base_fd, name):
    try:
        return os.open(name, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW | os.O_CLOEXEC, dir_fd=base_fd)
    except OSError as e:
        raise Refused(f"{name}: not a plain directory ({e.strerror})")


def _root(path):
    try:
        return os.open(path, os.O_RDONLY | os.O_DIRECTORY | os.O_CLOEXEC)   # the trusted base itself is the host's own path
    except OSError as e:
        raise Refused(f"{path}: {e.strerror}")


def read_regular(base, rel, max_bytes=MAX_DEFAULT):
    """-> bytes of base/rel; refuses links, non-regular files, hard-linked files and oversize files."""
    parts = _comps(rel)
    fds = [_root(base)]
    try:
        for p in parts[:-1]:
            fds.append(_opendir(fds[-1], p))
        try:
            fd = os.open(parts[-1], os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK | os.O_CLOEXEC, dir_fd=fds[-1])
        except OSError as e:
            raise Refused(f"{rel}: cannot open without following links ({e.strerror})")
        try:
            st = os.fstat(fd)
            if not stat.S_ISREG(st.st_mode):
                raise Refused(f"{rel}: not a regular file")
            if st.st_nlink != 1:
                raise Refused(f"{rel}: hard-linked file (link count {st.st_nlink})")
            if st.st_size > max_bytes:
                raise Refused(f"{rel}: larger than {max_bytes} bytes")
            data = b""
            while True:
                chunk = os.read(fd, 1 << 20)
                if not chunk:
                    break
                data += chunk
                if len(data) > max_bytes:
                    raise Refused(f"{rel}: larger than {max_bytes} bytes")
            return data
        finally:
            os.close(fd)
    finally:
        for f in fds:
            os.close(f)


def put(dirpath, name, data, mode=0o644):
    """Create dirpath/name (one component) holding `data`, replacing - never following - whatever is there."""
    if "/" in name or name in ("", ".", "..") or "\0" in name:
        raise Refused(f"bad file name {name!r}")
    d = _root(dirpath)
    try:
        try:
            st = os.lstat(name, dir_fd=d)
            if stat.S_ISDIR(st.st_mode):
                raise Refused(f"{name}: is a directory")
            os.unlink(name, dir_fd=d)            # removes a symlink itself, never its target
        except FileNotFoundError:
            pass
        try:
            fd = os.open(name, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW | os.O_CLOEXEC, mode, dir_fd=d)
        except OSError as e:
            raise Refused(f"{name}: cannot create exclusively ({e.strerror})")
        try:
            view = memoryview(data)
            n = 0
            while n < len(data):
                n += os.write(fd, view[n:])
        finally:
            os.close(fd)
    finally:
        os.close(d)


def scan(dirpath, allow=False, env=None):
    """-> list of problems below dirpath (deny-table names, links, special files, hard-linked files, over-cap volume; with allow=True also
    non-allow-listed extensions); [] when clean. Nothing below dirpath is ever opened for reading, hashed or executed here."""
    cp, bad = caps(env)
    count = [0, 0]                       # files, bytes

    def walk(fd, prefix, depth):
        if depth > cp["DEPTH"]:
            bad.append(f"{prefix}: deeper than {cp['DEPTH']} levels"); return
        for e in os.listdir(fd):
            if len(bad) > 200:
                return
            try:
                st = os.lstat(e, dir_fd=fd)
            except OSError as ex:
                bad.append(f"{prefix}{e}: {ex.strerror}")
                continue
            if deny_name(e):
                bad.append(f"{prefix}{e}: git/agent/tool control artifact (deny table)")
                continue
            if stat.S_ISDIR(st.st_mode):
                try:
                    sub = _opendir(fd, e)
                except Refused as r:
                    bad.append(f"{prefix}{e}: {r}")
                    continue
                try:
                    walk(sub, prefix + e + "/", depth + 1)
                finally:
                    os.close(sub)
            elif stat.S_ISREG(st.st_mode):
                count[0] += 1; count[1] += st.st_size
                if st.st_nlink != 1:
                    bad.append(f"{prefix}{e}: hard link (count {st.st_nlink})")
                if st.st_size > cp["FILE"]:
                    bad.append(f"{prefix}{e}: file larger than {cp['FILE']} bytes")
                if allow and (("." not in e) or e.rsplit(".", 1)[-1].lower() not in ALLOW_EXT):
                    bad.append(f"{prefix}{e}: not an allow-listed publish type")
            else:
                bad.append(f"{prefix}{e}: " + ("symlink" if stat.S_ISLNK(st.st_mode) else "special file"))

    r = _root(dirpath)
    try:
        walk(r, "", 0)
    finally:
        os.close(r)
    if count[0] > cp["FILES"]:
        bad.append(f"{count[0]} files exceed the cap of {cp['FILES']}")
    if count[1] > cp["TOTAL"]:
        bad.append(f"{count[1]} bytes exceed the cap of {cp['TOTAL']}")
    return bad


def first(base, rel, pattern):
    fds = [_root(base)]
    try:
        for c in _comps(rel):
            fds.append(_opendir(fds[-1], c))
        r = fds[-1]
        for e in sorted(os.listdir(r)):
            if not fnmatch.fnmatch(e, pattern) or not NAME_OK.match(e):
                continue
            st = os.lstat(e, dir_fd=r)
            if stat.S_ISREG(st.st_mode) and st.st_nlink == 1:
                return e
    finally:
        for f in fds:
            os.close(f)
    return None


def main(argv):
    try:
        if argv[1:2] == ["scan"] and (len(argv) == 3 or (len(argv) == 4 and argv[3] == "--allow")):
            bad = scan(argv[2], allow=len(argv) == 4)
            for b in bad:
                print(f"safe_io: refused {b}", file=sys.stderr)
            return 1 if bad else 0
        if len(argv) == 4 and argv[1] == "put":
            put(argv[2], argv[3], sys.stdin.buffer.read())
            return 0
        if len(argv) in (6, 8) and argv[1] == "copy":
            mx = int(argv[7]) if len(argv) == 8 and argv[6] == "--max" else MAX_DEFAULT
            put(argv[4], argv[5], read_regular(argv[2], argv[3], mx))
            return 0
        if len(argv) == 5 and argv[1] == "first":
            n = first(argv[2], argv[3], argv[4])
            if n is None:
                print("safe_io: no matching regular file", file=sys.stderr)
                return 1
            print(n)
            return 0
    except Refused as r:
        print(f"safe_io: refused: {r}", file=sys.stderr)
        return 1
    print(__doc__, file=sys.stderr)
    return 2


if __name__ == "__main__":
    sys.exit(main(sys.argv))
