#!/usr/bin/env python3
"""mutants_codec.py decode|encode FILE - the evader matrix is stored base64-encoded so the lint's own tree stays free of
forbidden literals. decode prints JSON; encode reads JSON on stdin and rewrites FILE."""
import base64, sys
mode, f = sys.argv[1:3]
if mode == "decode":
    sys.stdout.write(base64.b64decode(open(f).read()).decode())
else:
    open(f, "w").write(base64.b64encode(sys.stdin.read().encode()).decode() + "\n")
