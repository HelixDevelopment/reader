#!/usr/bin/env python3
"""rawrun.py - the ONE way a Python consumer PARSES what a container printed (review round 5, R5-B1).
run_container.sh without -o gives the operator's terminal a sanitized, merged, "[container] "-prefixed, line-capped stream: fine to read, wrong to parse.
run_raw() uses the wrapper's -o mode: container stdout and stderr come back UNMERGED and byte-exact from host-only files (private mode-0700 dir that is
removed afterwards, unless keep=True), and only a sanitized copy reached the terminal. The text returned is DATA FROM THE CONTAINER: parse it with a strict
parser (json.loads), never print or publish it without sanitize.sanitize_bytes / re-serialisation.
usage (library): from rawrun import run_raw; r = run_raw(["-n", "hrdiso-x"], ["node", "x.js"]); r.stdout, r.stderr, r.returncode, r.wrapper_stderr"""
import os, pathlib, shutil, subprocess, tempfile
HERE = pathlib.Path(__file__).resolve().parent
WRAPPER = HERE / "run_container.sh"

class Raw:
    def __init__(self, rc, out, err, wrapper_stderr, rawdir):
        self.returncode, self.stdout, self.stderr, self.wrapper_stderr, self.rawdir = rc, out, err, wrapper_stderr, rawdir

def run_raw(wrapper_args, cmd, timeout=120, env=None, keep=False, tmp=None, wrapper=None):
    base = tmp or os.environ.get("ISO_RAW_TMP") or os.path.expanduser("~/.cache/claude-tmp")
    d = os.path.realpath(tempfile.mkdtemp(prefix="rawrun-", dir=base)); os.chmod(d, 0o700)
    try:
        r = subprocess.run([str(wrapper or WRAPPER), *wrapper_args, "-o", d, "--", *cmd], capture_output=True, text=True, timeout=timeout, env=env)
        def rd(n):
            try: return pathlib.Path(d, n).read_bytes().decode("utf-8", "replace")
            except OSError: return ""
        return Raw(r.returncode, rd("stdout.raw"), rd("stderr.raw"), r.stderr, d if keep else None)
    finally:
        if not keep: shutil.rmtree(d, ignore_errors=True)
