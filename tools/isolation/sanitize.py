#!/usr/bin/env python3
"""sanitize.py - the ONE filter between container-controlled bytes and the operator's terminal / host-written logs (review round 4, class R4-B2).

MODEL: everything a container prints (the output of a run or exec of the container, files it wrote that a runner echoes, a request path in the proxy log) is attacker-controlled.
Raw, it can set the terminal title (OSC 0/2), write the operator's clipboard through a terminal multiplexer or terminal (OSC 52), move the cursor / erase
lines to hide or rewrite earlier output (CSI, CR), or forge host-looking verdict lines ("KOSYNC PROOF PASS"). This filter, applied on EVERY path where such
bytes reach stdout / tee / steps.log / evidence:
  * decodes as UTF-8 with replacement (an invalid byte can never smuggle a C1 control),
  * removes OSC (ESC ] ... BEL | ESC \\), DCS / APC / PM / SOS strings (ESC P _ ^ X ... ST), CSI (ESC [ ... final), any other ESC sequence, even when unterminated,
  * removes every other C0 control except \\n and \\t (so \\r, BEL, BS, ESC, NUL, VT, FF are gone), DEL, the C1 range U+0080-U+009F,
    and bidirectional / line-separator controls (U+200E/F, U+202A-E, U+2028/9, U+2066-9),
  * prefixes EVERY output line with PREFIX (default "[container] "), so a forged host line is visibly container-origin and never equals a host marker line,
  * caps one line at MAXLINE characters (the rest of that line is dropped, marked "...").
usage: sanitize.py [--prefix TEXT] < raw > clean     exit 0 (never fails on content). Library: sanitize_line(str) / sanitize_bytes(bytes).
Honest boundary: it removes terminal control bytes and marks origin; it cannot make container TEXT true - the text is still evidence of what the container
printed, not of what happened (the host-written verdicts are separate, host-only files)."""
import re, sys

PREFIX = "[container] "
MAXLINE = 4000
_OSC = re.compile(r"\x1b\][^\x07\x1b\n]*(?:\x07|\x1b\\)?")
_STR = re.compile(r"\x1b[P_^X][^\x1b\n]*(?:\x1b\\)?")
_CSI = re.compile(r"\x1b\[[0-?]*[ -/]*[@-~]?")
_ESC = re.compile(r"\x1b[ -/]*[0-~]?")
_CTL = re.compile("[\x00-\x08\x0b-\x1f\x7f-\x9f‎‏  ‪-‮⁦-⁩]")


def sanitize_line(s):
    """one logical line (no newline) -> printable text without any terminal control or sequence."""
    for rx in (_OSC, _STR, _CSI, _ESC):
        s = rx.sub("", s)
    s = _CTL.sub("", s)
    return s if len(s) <= MAXLINE else s[:MAXLINE] + "..."


def sanitize_bytes(b, prefix=PREFIX):
    text = b.decode("utf-8", errors="replace").replace("\r\n", "\n").replace("\r", "\n")
    out = []
    for ln in text.split("\n"):
        out.append(prefix + sanitize_line(ln))
    if out and out[-1] == prefix:           # text ended with a newline: no phantom prefixed empty line
        out.pop()
    return "\n".join(out) + ("\n" if out else "")


def main(argv):
    prefix = PREFIX
    if len(argv) == 3 and argv[1] == "--prefix":
        prefix = argv[2]
    elif len(argv) != 1:
        print(__doc__, file=sys.stderr)
        return 2
    inp, out = sys.stdin.buffer, sys.stdout
    while True:
        raw = inp.readline(1 << 16)       # a line is read in bounded pieces; a longer one is simply continued as the next prefixed line
        if not raw:
            break
        # \r is a line break for the filter: a carriage return can overwrite the visible line, so it must not survive
        chunk = sanitize_bytes(raw, prefix)
        out.write(chunk)
        out.flush()
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main(sys.argv))
    except BrokenPipeError:
        sys.exit(0)
