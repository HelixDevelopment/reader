#!/usr/bin/env python3
"""helixqa_publish.py RAW_STDOUT RAW_STDERR OUT_DIR - turn the RAW (unmerged, uncapped) streams of a HelixQA container run into evidence, FAIL CLOSED (review round 5, R5-B1).
The raw stdout is "<text report>\\n{\\n  json...}\\n" (helixqa --verbose --json). This script
  * splits at the first "\\n{\\n", parses the JSON part strictly (json.loads, must be an object) and RE-SERIALISES it (container bytes never reach evidence as-is),
  * sanitizes the text report and the stderr (sanitize.py: terminal controls removed, lines prefixed),
  * writes helixqa_http.txt / helixqa_http.json / helixqa_http.stderr.txt into OUT_DIR atomically - all three or none.
Any failure (missing file, no split point, invalid JSON, NaN/Infinity, nesting too deep, not an object) => message on stderr, exit 3, NOTHING published; the caller keeps the raw files for the post-mortem.
It never prints a success line itself: the exit status is the verdict."""
import json, os, sys
HERE = os.path.dirname(os.path.abspath(__file__)); sys.path.insert(0, os.path.join(HERE, "..", "..", "tools", "isolation"))
import sanitize


def fail(msg):
    print("helixqa_publish: FAILED (nothing published): " + msg, file=sys.stderr); sys.exit(3)


def main(argv):
    if len(argv) != 4: print(__doc__, file=sys.stderr); return 2
    rawout, rawerr, out = argv[1:]
    try: data = open(rawout, "rb").read(); err = open(rawerr, "rb").read()
    except OSError as e: fail(f"cannot read the raw streams: {e.strerror}")
    text = data.decode("utf-8", "replace"); i = text.find("\n{\n")
    if i < 0: fail("no JSON result block (newline, open brace, newline) in the container's stdout")
    def non_finite(c): raise ValueError("non-finite number " + c)     # round 6: NaN / Infinity / -Infinity are not JSON; a result that carries one is refused
    try: doc = json.loads(text[i + 1:], parse_constant=non_finite)
    except RecursionError: fail("the JSON result block is nested too deeply")
    except ValueError as e: fail(f"the JSON result block does not parse ({type(e).__name__})")
    if not isinstance(doc, dict): fail("the JSON result block is not an object")
    try: body_json = json.dumps(doc, indent=2, ensure_ascii=True, allow_nan=False) + "\n"
    except (RecursionError, ValueError): fail("the JSON result block cannot be re-serialised")
    files = {"helixqa_http.txt": sanitize.sanitize_bytes(text[:i + 1].encode("utf-8"), ""), "helixqa_http.json": body_json,
             "helixqa_http.stderr.txt": sanitize.sanitize_bytes(err, "[container:stderr] ")}
    tmp = []
    try:
        for n, body in files.items():
            t = os.path.join(out, "." + n + ".tmp"); tmp.append((t, os.path.join(out, n)))
            fd = os.open(t, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o644)
            with os.fdopen(fd, "w", encoding="utf-8") as f: f.write(body)
        for t, n in tmp: os.replace(t, n)
    except OSError as e:
        for t, _ in tmp:
            try: os.unlink(t)
            except OSError: pass
        fail(f"cannot write the evidence: {e.strerror}")
    return 0


if __name__ == "__main__": sys.exit(main(sys.argv))
