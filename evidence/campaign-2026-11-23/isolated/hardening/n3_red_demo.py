#!/usr/bin/env python3
"""N3 RED/GREEN demonstration. OLD predicate = round-1 line `if DENYLINE.search(text) and in_tools: continue` with in_tools =
rel.startswith("tools/isolation/") (reproduced from the review finding; the round-1 source itself was not kept), applied to an attacker
file that merely ends a malicious line with the marker. NEW = the current lint on the same tree."""
import re, sys, tempfile, pathlib
R = pathlib.Path(sys.argv[1]); sys.path.insert(0, str(R / "tools/isolation/guard"))
import assert_host_untouched as g
lit = "".join(["xdo", "tool"]); mark = "".join(["# isolation: ", "deny", "list"])
rel = "tools/isolation/evil.sh"; line = f"{lit} key Return {mark}"
old_denyline = re.compile(r"isolation:\s*denylist"); in_tools = rel.startswith("tools/isolation/")
print(f"OLD predicate on {rel!r}: line exempt = {bool(old_denyline.search(line) and in_tools)}   <- RED: any file under tools/isolation/ is exempt by a trailing comment")
d = tempfile.mkdtemp(prefix="n3-"); p = pathlib.Path(d, rel); p.parent.mkdir(parents=True); p.write_text(line + "\n")
hits = g.scan(d)
print("NEW lint on the same tree:", sorted({h[2] for h in hits}), "-> FAIL" if hits else "-> PASS (unexpected)")
sys.exit(0 if hits else 1)
