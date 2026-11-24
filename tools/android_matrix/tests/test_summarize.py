#!/usr/bin/env python3
"""test_summarize.py - HRD-040 F2/F3/F4: summarize.py must FAIL a row/unit result for: content-less screenshot whose only colours are system bars (F2),
a screenshot of the wrong size (F2), a webview that does not fill the display (F3), a unit run whose gradle exit is nonzero or a mutant run that stayed green (F4),
a stale installed package hash, a screen where only the header strip is left (R4), a run with zero rows or only SKIP rows (R3: exit 1, not a quiet 0), a "killed" mutant
whose gradle failed without any failing test (R2). A fully good row must PASS (golden-FALSE guard)."""
import json, os, struct, subprocess, sys, tempfile, zlib, hashlib, unittest
HERE = os.path.dirname(os.path.abspath(__file__)); SUM = os.environ.get("HRD_SUMMARIZE") or os.path.join(HERE, "..", "summarize.py")

def png(w, h, pixel):
    raw = b"".join(b"\x00" + b"".join(bytes(pixel(x, y)) for x in range(w)) for y in range(h))
    def ch(t, d): return struct.pack(">I", len(d)) + t + d + struct.pack(">I", zlib.crc32(t + d) & 0xffffffff)
    return b"\x89PNG\r\n\x1a\n" + ch(b"IHDR", struct.pack(">IIBBBBB", w, h, 8, 2, 0, 0, 0)) + ch(b"IDAT", zlib.compress(raw)) + ch(b"IEND", b"")

W, H = 120, 240
def busy(x, y): return ((x * 7) % 256, (y * 5) % 256, ((x + y) * 3) % 256)
def header_only(x, y): return busy(x, y) if y < H * 0.13 else (255, 255, 255)
def bars_only(x, y): return (x % 256, 40, 40) if (y < H * 0.08 or y >= H * 0.92) else (255, 255, 255)

def make(d, shot=None, size=(W, H), webview="120x230", unit=("0", "1"), installed_ok=True, ok="OK (2 tests)", state="RAN", norows=False, mutant_junit="failure"):
    os.makedirs(d + "/r1")
    app, test = "a" * 64, "b" * 64
    open(d + "/apk.sha256.txt", "w").write("%s  app-debug.apk\n%s  app-debug-androidTest.apk\n" % (app, test))
    open(d + "/source_rev.txt", "w").write("0123456789abcdef0123456789abcdef01234567\n")
    json.dump({"row_id": "r1", "api": 34, "image": "i", "width": W, "height": H, "dpi": 420, "class": "phone", "state": state, "reason": "x"}, open(d + "/r1/row.json", "w"))
    if norows: os.remove(d + "/r1/row.json")
    open(d + "/r1/instrument.txt", "w").write(ok + "\n")
    open(d + "/r1/logcat_hrdmatrix.txt", "w").write("I/HRDMATRIX: HRDMATRIX: sdk=34 px=%dx%d dpi=420 widthDp=1 heightDp=1 webview=%s\n" % (W, 230, webview))
    open(d + "/r1/screen.png", "wb").write(png(size[0], size[1], shot or busy))
    open(d + "/r1/installed.txt", "w").write("app /p/app %s\ntest /p/test %s\n" % (app if installed_ok else "c" * 64, test))
    if unit[0] is not None:
        os.makedirs(d + "/unit"); open(d + "/unit/exit.txt", "w").write("gradle exit %s\n" % unit[0])
    os.makedirs(d + "/unit_mutant/junit"); open(d + "/unit_mutant/exit.txt", "w").write("gradle exit %s (nonzero expected)\n" % unit[1])
    if mutant_junit: open(d + "/unit_mutant/junit/TEST-x.xml.txt", "w").write('<testsuite tests="1" failures="1"><testcase name="write method"><%s message="expected false"/></testcase></testsuite>' % mutant_junit)

def run(**kw):
    with tempfile.TemporaryDirectory() as t:
        make(t, **kw); p = subprocess.run([sys.executable, "-I", SUM, t], capture_output=True, text=True)
        return p.returncode, p.stdout

class T(unittest.TestCase):
    def test_good_row_passes(self): rc, o = run(); self.assertEqual(rc, 0, o)
    def test_f2_system_bars_only_fails(self): rc, o = run(shot=bars_only); self.assertEqual(rc, 1, o)
    def test_f2_wrong_screenshot_size_fails(self): rc, o = run(size=(W // 2, H // 2)); self.assertEqual(rc, 1, o)
    def test_f3_tiny_webview_fails(self): rc, o = run(webview="1x1"); self.assertEqual(rc, 1, o)
    def test_f4_unit_failure_fails(self): rc, o = run(unit=("1", "1")); self.assertEqual(rc, 1, o)
    def test_f4_mutant_survives_fails(self): rc, o = run(unit=("0", "0")); self.assertEqual(rc, 1, o)
    def test_stale_installed_hash_fails(self): rc, o = run(installed_ok=False); self.assertEqual(rc, 1, o)
    def test_r4_header_only_fails(self): rc, o = run(shot=header_only); self.assertEqual(rc, 1, o)
    def test_r3_zero_rows_exits_nonzero(self): rc, o = run(norows=True); self.assertEqual(rc, 1, o)
    def test_r3_only_skip_rows_exit_nonzero(self): rc, o = run(state="SKIP"); self.assertEqual(rc, 1, o)
    def test_r2_mutant_killed_by_build_failure_only_fails(self): rc, o = run(mutant_junit=None); self.assertEqual(rc, 1, o)
    def test_r2_mutant_error_in_junit_counts(self): rc, o = run(mutant_junit="error"); self.assertEqual(rc, 0, o)
    def test_no_tests_run_fails(self): rc, o = run(ok="OK (0 tests)"); self.assertEqual(rc, 1, o)

if __name__ == "__main__": unittest.main()
