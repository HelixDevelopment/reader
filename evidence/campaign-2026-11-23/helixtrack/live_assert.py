#!/usr/bin/env python3
"""live_assert.py LABEL FILE.json 'python expression over d' EXPECTED_JSON  -> appends PASS/FAIL to $ASSERT_LOG and prints it."""
import json, os, sys
label, path, expr, expected = sys.argv[1:5]
d = json.load(open(path))
actual = eval(expr, {"d": d, "len": len, "sorted": sorted, "set": set, "sum": sum, "any": any, "all": all})
n = int(os.environ.get("N", "0"))
for tok, val in (("@N10", n - 10), ("@N2", n - 2), ("@N1", n - 1), ("@N", n)):
    expected = expected.replace(tok, str(val))
ok = actual == json.loads(expected)
line = "%s %s actual=%s expected=%s" % ("PASS" if ok else "FAIL", label, json.dumps(actual), expected)
print(line)
open(os.environ.get("ASSERT_LOG", "/dev/null"), "a").write(line + "\n")
sys.exit(0 if ok else 1)
