#!/usr/bin/env python3
"""summarize.py <evidence_dir> - read the per-row result files written by run_matrix.sh and write results.json + summary.txt; exit 1 on any FAIL.
A RAN row is PASS only when ALL hold:
  instrumentation_ok      `OK (N tests)` with N>=1 in instrument.txt
  display_matches_row     HRDMATRIX logcat line: sdk, width, dpi exact; height within 10 percent below the configured one (system bars)
  webview_fills_display   the same line: webview width == display width and webview height >= 85 percent of the display height
  screenshot_size_matches the PNG is exactly the row's width x height
  screenshot_content      >= 8 distinct sampled colours in the CONTENT region (top and bottom 8 percent, the system bars, are excluded)
  installed_apk_matches   installed.txt (package path + sha256 read back FROM THE DEVICE) equals the build recorded in apk.sha256.txt, for the app and the test APK
  source_rev_recorded     source_rev.txt names the revision the APKs were built from
If unit/exit.txt exists the JVM result is part of the verdict (`gradle exit 0` required); unit_mutant/exit.txt must say a NONZERO exit (the mutant is killed)."""
import json, os, re, sys, struct, zlib

def png_read(path):
    """(width, height, [set of sampled pixel values per row]) of an 8-bit non-interlaced RGB(A) PNG; None when not decodable."""
    try: d = open(path, "rb").read()
    except OSError: return None
    if d[:8] != b"\x89PNG\r\n\x1a\n": return None
    i, idat, w, h, ct, bd, il = 8, b"", 0, 0, 0, 0, 0
    try:
        while i < len(d):
            n, t = struct.unpack(">I4s", d[i:i + 8]); c = d[i + 8:i + 8 + n]; i += 12 + n
            if t == b"IHDR": w, h, bd, ct, _, _, il = struct.unpack(">IIBBBBB", c)
            elif t == b"IDAT": idat += c
        if bd != 8 or il != 0 or ct not in (2, 6): return None
        bpp = 3 if ct == 2 else 4; raw = zlib.decompress(idat); st = w * bpp; prev = bytearray(st); p = 0; rows = []
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
            rows.append({bytes(line[x:x + bpp]) for x in range(0, st, bpp * 7)}); prev = line
        return w, h, rows
    except (struct.error, zlib.error, IndexError): return None

def content_colors(rows, h):
    seen = set()
    for r in rows[-(-h * 8 // 100):h * 92 // 100]: seen |= r
    return len(seen)

def read(p):
    try: return open(p, errors="replace").read()
    except OSError: return ""

def sums(path):
    out = {}
    for ln in read(path).splitlines():
        m = re.match(r"^([0-9a-f]{64})\s+\*?(.+)$", ln)
        if m: out[os.path.basename(m.group(2))] = m.group(1)
    return out

ev = sys.argv[1]; rows = []; built = sums(os.path.join(ev, "apk.sha256.txt")); srcrev = read(os.path.join(ev, "source_rev.txt")).strip()
for name in sorted(os.listdir(ev)):
    rd = os.path.join(ev, name)
    if not os.path.isdir(rd) or not os.path.exists(os.path.join(rd, "row.json")): continue
    r = json.load(open(os.path.join(rd, "row.json"))); r["checks"] = {}
    if r["state"] == "RAN":
        m = re.search(r"^OK \((\d+) tests?\)", read(os.path.join(rd, "instrument.txt")), re.M)
        r["checks"]["instrumentation_ok"] = bool(m) and int(m.group(1)) >= 1; r["instrumentation_tests"] = int(m.group(1)) if m else 0
        h = re.search(r"HRDMATRIX: (sdk=(\d+) px=(\d+)x(\d+) dpi=(\d+) .*?webview=(\d+)x(\d+).*)", read(os.path.join(rd, "logcat_hrdmatrix.txt")))
        r["hrdmatrix"] = h.group(1) if h else None
        if h:
            sdk, pw, ph, dpi, ww, wh = (int(h.group(k)) for k in (2, 3, 4, 5, 6, 7))
            r["checks"]["display_matches_row"] = (sdk, pw, dpi) == (r["api"], r["width"], r["dpi"]) and 0.9 * r["height"] <= ph <= r["height"]
            r["checks"]["webview_fills_display"] = ww == pw and wh >= 0.85 * ph
        else:
            r["checks"]["display_matches_row"] = r["checks"]["webview_fills_display"] = False
        pr = png_read(os.path.join(rd, "screen.png"))
        r["checks"]["screenshot_size_matches"] = bool(pr) and (pr[0], pr[1]) == (r["width"], r["height"])
        r["screenshot_content_colors"] = content_colors(pr[2], pr[1]) if pr else 0
        r["checks"]["screenshot_content"] = r["screenshot_content_colors"] >= 8
        inst = {}
        for ln in read(os.path.join(rd, "installed.txt")).splitlines():
            p = ln.split()
            if len(p) == 3: inst[p[0]] = p[2]
        r["installed_sha256"] = inst
        r["checks"]["installed_apk_matches"] = bool(built.get("app-debug.apk")) and inst.get("app") == built.get("app-debug.apk") \
            and bool(built.get("app-debug-androidTest.apk")) and inst.get("test") == built.get("app-debug-androidTest.apk")
        r["checks"]["source_rev_recorded"] = bool(re.fullmatch(r"[0-9a-f]{40}", srcrev))
        r["verdict"] = "PASS" if all(r["checks"].values()) else "FAIL"
    else:
        r["verdict"] = r["state"]
    rows.append(r)
unit = {}
for k, want_zero in (("unit", True), ("unit_mutant", False)):
    t = read(os.path.join(ev, k, "exit.txt"))
    if t:
        m = re.search(r"gradle exit (\d+)", t); code = int(m.group(1)) if m else None
        unit[k] = {"gradle_exit": code, "verdict": "PASS" if code is not None and ((code == 0) == want_zero) else "FAIL"}
json.dump({"source_rev": srcrev, "apk_sha256": built, "unit": unit, "rows": rows}, open(os.path.join(ev, "results.json"), "w"), indent=1)
with open(os.path.join(ev, "summary.txt"), "w") as f:
    for k, v in unit.items(): f.write("%-14s %-8s gradle exit %s\n" % (k, v["verdict"], v["gradle_exit"]))
    for r in rows: f.write("%-14s %-8s %s\n" % (r["row_id"], r["verdict"], r.get("reason") or r.get("hrdmatrix") or ""))
print(open(os.path.join(ev, "summary.txt")).read(), end="")
sys.exit(1 if any(r["verdict"] == "FAIL" for r in rows) or any(u["verdict"] == "FAIL" for u in unit.values()) else 0)
