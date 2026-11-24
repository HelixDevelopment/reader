#!/usr/bin/env python3
"""Paired mutants for the hermetic helpers of tests/test_supported_platforms.py.

Each mutant edits a scratch copy of the test module (never the original); HermeticHelpersTest must FAIL for it.
The unmutated copy runs first as the control and must PASS. Prints a kill table; exit 1 on any survivor.
"""
import shutil, subprocess, sys, tempfile
from pathlib import Path

SRC = Path(__file__).resolve().parent / "test_supported_platforms.py"
MUTANTS = [
    ("M1 no strip of trailing newline", 'return "tv" in [c.strip() for c in value.strip().lower().split(",")]', 'return "tv" in value.lower().split(",")'),
    ("M2 no per-item strip", 'return "tv" in [c.strip() for c in value.strip().lower().split(",")]', 'return "tv" in value.strip().lower().split(",")'),
    ("M3 substring match", 'return "tv" in [c.strip() for c in value.strip().lower().split(",")]', 'return "tv" in value.lower()'),
    ("M4 case sensitive", 'return "tv" in [c.strip() for c in value.strip().lower().split(",")]', 'return "tv" in [c.strip() for c in value.strip().split(",")]'),
    ("M5 opt-in gate inverted", 'environ.get("HELIX_LIVE_PLATFORM_PROBE") != "1"', 'environ.get("HELIX_LIVE_PLATFORM_PROBE") == "1"'),
    ("M6 gate accepts any value", 'environ.get("HELIX_LIVE_PLATFORM_PROBE") != "1"', 'not environ.get("HELIX_LIVE_PLATFORM_PROBE")'),
    ("M7 skip code changed", 'SKIP_DISABLED = "feature_disabled_by_config"', 'SKIP_DISABLED = "skipped"'),
    ("M8 no-hw code changed", 'SKIP_NO_HW = "hardware_not_present"', 'SKIP_NO_HW = "no_adb"'),
    ("M9 relative adb fallback", 'if home and Path(home).is_absolute():', 'if True:'),
    ("M10 stale detection inverted", 'if (root / rel).exists())', 'if not (root / rel).exists())'),
]


def run(text: str) -> int:
    with tempfile.TemporaryDirectory() as d:
        t = Path(d) / "tests"
        t.mkdir()
        (t / "test_supported_platforms.py").write_text(text)
        (Path(d) / "clients").mkdir()
        r = subprocess.run([sys.executable, "-m", "unittest", "tests.test_supported_platforms.HermeticHelpersTest"],
                           cwd=d, capture_output=True, text=True)
        return r.returncode


def main() -> int:
    text = SRC.read_text()
    if run(text) != 0:
        print("CONTROL FAILED: unmutated copy does not pass"); return 2
    print("control (unmutated): PASS")
    bad = 0
    for name, old, new in MUTANTS:
        assert text.count(old) == 1, f"{name}: needle not unique ({text.count(old)})"
        killed = run(text.replace(old, new)) != 0
        print(f"{name}: {'KILLED' if killed else 'SURVIVED'}")
        bad += not killed
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main())
