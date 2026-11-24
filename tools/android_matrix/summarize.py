#!/usr/bin/env python3
"""summarize.py <evidence_dir> - read the per-row result files written by run_matrix.sh and write results.json + summary.txt.
A row is PASS only when its instrumentation output says `OK (N tests)` with N>=1 AND the screenshot is non-blank AND the
HRDMATRIX line proves the configured display; FAIL when instrumentation ran and did not; SKIP/BLOCKED rows keep their reason."""
import json, os, re, sys, struct, zlib

def png_colors(path):
    """number of distinct RGB(A) pixel values in an 8-bit non-interlaced PNG (stdlib only); 0 when not decodable."""
    d = open(path, "rb").read()
    if d[:8] != b"\x89PNG\r\n\x1a\n":
        return 0
    i, idat, w, h, ct, bd, il = 8, b"", 0, 0, 0, 0, 0
    while i < len(d):
        n, t = struct.unpack(">I4s", d[i:i + 8]); c = d[i + 8:i + 8 + n]; i += 12 + n
        if t == b"IHDR": w, h, bd, ct, _, _, il = struct.unpack(">IIBBBBB", c)
        elif t == b"IDAT": idat += c
    if bd != 8 or il != 0 or ct not in (2, 6): return 0
    bpp = 3 if ct == 2 else 4; raw = zlib.decompress(idat); st = w * bpp; prev = bytearray(st); seen = set(); p = 0
    for _ in range(h):
        f = raw[p]; line = bytearray(raw[p + 1:p + 1 + st]); p += 1 + st
        for x in range(st):
            a = line[x - bpp] if x >= bpp else 0; b = prev[x]; c2 = prev[x - bpp] if x >= bpp else 0
            if f == 1: line[x] = (line[x] + a) & 255
            elif f == 2: line[x] = (line[x] + b) & 255
            elif f == 3: line[x] = (line[x] + ((a + b) >> 1)) & 255
            elif f == 4:
                pa, pb, pc = abs(b - c2), abs(a - c2), abs(a + b - 2 * c2)
                line[x] = (line[x] + (a if pa <= pb and pa <= pc else b if pb <= pc else c2)) & 255
        for x in range(0, st, bpp * 7): seen.add(bytes(line[x:x + bpp]))
        prev = line
    return len(seen)

ev = sys.argv[1]; rows = []
for name in sorted(os.listdir(ev)):
    rd = os.path.join(ev, name)
    if not os.path.isdir(rd) or not os.path.exists(os.path.join(rd, "row.json")): continue
    r = json.load(open(os.path.join(rd, "row.json"))); r["checks"] = {}
    if r["state"] == "RAN":
        ins = open(os.path.join(rd, "instrument.txt"), errors="replace").read()
        m = re.search(r"^OK \((\d+) tests?\)", ins, re.M); r["checks"]["instrumentation_ok"] = bool(m) and int(m.group(1)) >= 1
        r["instrumentation_tests"] = int(m.group(1)) if m else 0
        h = re.search(r"HRDMATRIX (sdk=\d+ px=(\d+)x(\d+) dpi=(\d+).*)", ins)
        r["hrdmatrix"] = h.group(1) if h else None
        r["checks"]["display_matches_row"] = bool(h) and (int(h.group(2)), int(h.group(3)), int(h.group(4))) == (r["width"], r["height"], r["dpi"]) \
            and ("sdk=%d " % r["api"]) in h.group(1)
        shot = os.path.join(rd, "screen.png"); n = png_colors(shot) if os.path.exists(shot) else 0
        r["screenshot_distinct_colors"] = n; r["checks"]["screenshot_nonblank"] = n >= 8
        r["verdict"] = "PASS" if all(r["checks"].values()) else "FAIL"
    else:
        r["verdict"] = r["state"]
    rows.append(r)
json.dump({"rows": rows}, open(os.path.join(ev, "results.json"), "w"), indent=1)
with open(os.path.join(ev, "summary.txt"), "w") as f:
    for r in rows: f.write("%-14s %-8s %s\n" % (r["row_id"], r["verdict"], r.get("reason") or r.get("hrdmatrix") or ""))
print(open(os.path.join(ev, "summary.txt")).read(), end="")
sys.exit(1 if any(r["verdict"] == "FAIL" for r in rows) else 0)
