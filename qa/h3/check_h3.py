#!/usr/bin/env python3
"""Judge one h3-lab scenario from the SERVER's structured access log (slog JSON lines), never from a client claim.
usage: check_h3.py <server-log-slice> <path> <expect-proto> <expect-encoding|any> [min-count]
Prints one JSON verdict line; exit 0 PASS, 1 FAIL, 2 BLIND (no matching log lines: the instrument saw nothing, never a pass)."""
import json, sys
log, path, want_proto, want_enc = sys.argv[1:5]
min_count = int(sys.argv[5]) if len(sys.argv) > 5 else 1
rows = []
for line in open(log, encoding="utf-8", errors="replace"):
    line = line.strip()
    if not line.startswith("{"):
        continue
    try:
        o = json.loads(line)
    except ValueError:
        continue
    if o.get("msg") == "access" and o.get("path") == path:
        rows.append(o)
protos = sorted({r["proto"] for r in rows}); encs = sorted({r["encoding"] for r in rows})
v = {"path": path, "requests": len(rows), "protocols": protos, "encodings": encs, "expect_proto": want_proto, "expect_encoding": want_enc}
if len(rows) < min_count:
    v["verdict"] = "BLIND"; print(json.dumps(v)); sys.exit(2)
ok = protos == [want_proto] and (want_enc == "any" or encs == [want_enc])
v["verdict"] = "PASS" if ok else "FAIL"; print(json.dumps(v)); sys.exit(0 if ok else 1)
