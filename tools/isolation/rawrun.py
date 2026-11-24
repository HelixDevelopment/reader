#!/usr/bin/env python3
"""rawrun.py - the ONE way a Python consumer PARSES what a container printed (review round 5 R5-B1, hardened in round 6 R6-B1).
run_container.sh without -o gives the operator's terminal a sanitized, merged, "[container] "-prefixed, line-capped stream: fine to read, wrong to parse.
run_raw() uses the wrapper's -o mode: container stdout and stderr come back UNMERGED and byte-exact from host-only files (private mode-0700 dir that is
removed afterwards, unless keep=True), and only a sanitized copy reached the terminal.

Round 6 (R6-B1): the byte-exact streams are held ONLY in Raw.raw_stdout / Raw.raw_stderr (bytes, UNSAFE TO PRINT: they may carry ESC/OSC/CSI, C0/C1, bidi).
Every str-ish surface of Raw is sanitized-only: .stdout, .stderr, .wrapper_stderr, __str__, __repr__, and the parsers below return parsed DATA, never text.
A consumer that needs to PARSE uses json_after() / has_token() (the allow-listed readers of raw_*); a consumer that wants to SHOW something uses .stdout / .stderr.
check_raw_consumers.py flags any use of raw_stdout / raw_stderr outside this file.
usage: from rawrun import run_raw; r = run_raw(["-n", "hrdiso-x"], ["node", "x.js"]); r.json_after("ADMIN:"), r.has_token("GOT:"), r.returncode, r.stderr (sanitized)"""
import json, os, pathlib, shutil, subprocess, sys, tempfile
HERE = pathlib.Path(__file__).resolve().parent
WRAPPER = HERE / "run_container.sh"
sys.path.insert(0, str(HERE))
from sanitize import sanitize_bytes, _CTL     # noqa: E402


def _clean(b, prefix):
    return sanitize_bytes(b, prefix)


def _check_strings(o):
    """a parsed value is DATA FROM THE CONTAINER: refuse (ValueError) any string holding a control / bidi / ESC character, so a failure message that formats it cannot carry one."""
    if isinstance(o, str):
        if _CTL.search(o) or "\x1b" in o: raise ValueError("container data holds a terminal control character")
    elif isinstance(o, dict):
        for k, v in o.items(): _check_strings(k); _check_strings(v)
    elif isinstance(o, list):
        for v in o: _check_strings(v)
    return o


def _b(x): return x if isinstance(x, bytes) else str(x).encode("utf-8", "replace")


class Raw:
    """raw_stdout / raw_stderr are BYTES and UNSAFE TO PRINT; everything else a human or an assertion message may see is sanitized."""
    __slots__ = ("returncode", "raw_stdout", "raw_stderr", "_wrapper_stderr", "rawdir")

    def __init__(self, rc, out, err, wrapper_stderr, rawdir):
        self.returncode, self.rawdir = rc, rawdir
        self.raw_stdout, self.raw_stderr, self._wrapper_stderr = _b(out), _b(err), _b(wrapper_stderr)

    @property
    def stdout(self): return _clean(self.raw_stdout, "[container] ")

    @property
    def stderr(self): return _clean(self.raw_stderr, "[container:stderr] ")

    @property
    def wrapper_stderr(self): return _clean(self._wrapper_stderr, "[wrapper] ")

    def __str__(self): return f"Raw(rc={self.returncode}, stdout={len(self.raw_stdout)}B, stderr={len(self.raw_stderr)}B)"
    __repr__ = __str__

    def has_token(self, token):
        """True if the container's stdout holds the token (a literal the CALLER wrote). Returns a bool, never container text."""
        return _b(token) in self.raw_stdout

    def json_after(self, prefix, keep_prefix=False):
        """the last stdout line starting with prefix, parsed as strict JSON (NaN/Infinity refused, strings control-free). None if there is none or it does not parse."""
        p = _b(prefix)
        lines = [ln for ln in self.raw_stdout.split(b"\n") if ln.startswith(p)]
        if not lines: return None
        def bad(c): raise ValueError("non-finite number")
        try: return _check_strings(json.loads(lines[-1][0 if keep_prefix else len(p):].decode("utf-8"), parse_constant=bad))
        except (ValueError, RecursionError): return None


def run_raw(wrapper_args, cmd, timeout=120, env=None, keep=False, tmp=None, wrapper=None):
    base = tmp or os.environ.get("ISO_RAW_TMP") or os.path.expanduser("~/.cache/claude-tmp")
    d = os.path.realpath(tempfile.mkdtemp(prefix="rawrun-", dir=base)); os.chmod(d, 0o700)
    try:
        # bytes, never text=True: nothing here decodes container bytes into a str that could travel in an exception
        r = subprocess.run([str(wrapper or WRAPPER), *wrapper_args, "-o", d, "--", *cmd], capture_output=True, timeout=timeout, env=env)
        def rd(n):
            try: return pathlib.Path(d, n).read_bytes()
            except OSError: return b""
        return Raw(r.returncode, rd("stdout.raw"), rd("stderr.raw"), r.stderr, d if keep else None)
    finally:
        if not keep: shutil.rmtree(d, ignore_errors=True)


def run_direct(argv, timeout=120, env=None):
    """run a command that is NOT run_container.sh (a test that must bypass the wrapper and call the container runtime directly) and hand its output back as a Raw: bytes in raw_*,
    sanitized text everywhere else, so a failure message built from .stderr / .stdout can never carry a container escape sequence."""
    r = subprocess.run([str(a) for a in argv], capture_output=True, timeout=timeout, env=env)
    return Raw(r.returncode, r.stdout, r.stderr, b"", None)
