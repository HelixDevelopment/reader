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
usage: safe_io.py scan DIR | put DIR NAME (data on stdin) | copy SRCBASE SRCREL DSTDIR DSTNAME [--max N] | first BASE REL GLOB
BASE / DIR arguments are trusted host paths (not under container control); REL / NAME are names, never '..'.
exit: 0 ok, 1 refused (message on stderr), 2 usage."""
import fnmatch, os, re, stat, sys

NAME_OK = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._ -]{0,200}$")
MAX_DEFAULT = 64 * 1024 * 1024


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


def scan(dirpath):
    """-> list of problems below dirpath (links, special files, hard-linked files); [] when clean."""
    bad = []

    def walk(fd, prefix):
        for e in os.listdir(fd):
            try:
                st = os.lstat(e, dir_fd=fd)
            except OSError as ex:
                bad.append(f"{prefix}{e}: {ex.strerror}")
                continue
            if stat.S_ISDIR(st.st_mode):
                try:
                    sub = _opendir(fd, e)
                except Refused as r:
                    bad.append(f"{prefix}{e}: {r}")
                    continue
                try:
                    walk(sub, prefix + e + "/")
                finally:
                    os.close(sub)
            elif stat.S_ISREG(st.st_mode):
                if st.st_nlink != 1:
                    bad.append(f"{prefix}{e}: hard link (count {st.st_nlink})")
            else:
                bad.append(f"{prefix}{e}: " + ("symlink" if stat.S_ISLNK(st.st_mode) else "special file"))

    r = _root(dirpath)
    try:
        walk(r, "")
    finally:
        os.close(r)
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
        if len(argv) == 3 and argv[1] == "scan":
            bad = scan(argv[2])
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
