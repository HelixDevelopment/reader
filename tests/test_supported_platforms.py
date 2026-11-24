#!/usr/bin/env python3
"""Tests for clients/supported-platforms.json (HRD-022, section 11.4.81: no silent skip).

The matrix is the single statement of which client platforms exist. These tests make
three promises: every platform carries a status from the closed vocabulary; every
BLOCKED platform carries an exact reason and a reproducer command; and the captured
output of those reproducers (evidence/clients/blocked_reproducers.json, written by
clients/tools/capture_blocked.py) agrees with the matrix and with the live host. A
tool that later appears on the host makes the live re-run succeed, which fails the
test and forces the BLOCKED entry to be revisited instead of rotting.
"""
from __future__ import annotations

import json
import os
from pathlib import Path
import shutil
import subprocess
import unittest

ROOT = Path(__file__).resolve().parents[1]
MATRIX = ROOT / "clients" / "supported-platforms.json"
CAPTURED = ROOT / "evidence" / "clients" / "blocked_reproducers.json"
STATUSES = {"BUILT", "VERIFIED", "BLOCKED", "NOT_STARTED"}
REQUIRED = {"linux", "windows", "macos", "android", "ios", "android-tv"}


# --- pure helpers (hermetic tests below exercise them with plain strings) -----------------------
SKIP_DISABLED = "feature_disabled_by_config"
SKIP_NO_HW = "hardware_not_present"


def is_tv_characteristics(value: str) -> bool:
    """True when `getprop ro.build.characteristics` output names the tv characteristic.

    The output is comma separated and ends in a line end ("tv\n", "nosdcard,tv\r\n"), so strip first and
    compare each trimmed item exactly (so "television" does not match).
    """
    return "tv" in [c.strip() for c in value.strip().lower().split(",")]


def live_probe_skip_reason(environ) -> str | None:
    """Closed skip reason for the opt-in live probe, or None when it is enabled."""
    if environ.get("HELIX_LIVE_PLATFORM_PROBE") != "1":
        return SKIP_DISABLED + ": set HELIX_LIVE_PLATFORM_PROBE=1 to run the live host probes"
    return None


def find_adb(environ, which=shutil.which) -> str | None:
    """adb from PATH or $ANDROID_HOME/platform-tools; None (never a relative path) when absent."""
    found = which("adb")
    if found:
        return found
    home = environ.get("ANDROID_HOME", "")
    if home and Path(home).is_absolute():
        cand = Path(home) / "platform-tools" / "adb"
        if cand.exists():
            return str(cand)
    return None


def stale_pending(doc: dict, root: Path) -> list:
    """pending_evidence paths that now exist on disk (their PR has landed): the row should be promoted."""
    paths = (doc.get("pending_evidence") or {}).get("paths", {})
    return sorted(rel for rel in paths if (root / rel).exists())


def load(path: Path):
    return json.loads(path.read_text())


class SupportedPlatformsTest(unittest.TestCase):
    def setUp(self) -> None:
        self.assertTrue(MATRIX.is_file(), f"missing {MATRIX}")
        self.doc = load(MATRIX)
        self.platforms = self.doc["platforms"]

    def test_schema_and_required_platforms(self) -> None:
        self.assertEqual(self.doc["item"], "HRD-022")
        ids = [p["id"] for p in self.platforms]
        self.assertEqual(len(ids), len(set(ids)), "duplicate platform id")
        self.assertTrue(REQUIRED <= set(ids), f"missing {REQUIRED - set(ids)}")
        for p in self.platforms:
            for key in ("id", "name", "status", "technology", "notes"):
                self.assertIsInstance(p.get(key), str, f"{p.get('id')}: {key}")
                self.assertTrue(p[key].strip(), f"{p['id']}: empty {key}")

    def test_every_platform_has_a_known_status(self) -> None:
        for p in self.platforms:
            self.assertIn(p.get("status"), STATUSES, p["id"])

    def test_blocked_entries_carry_reason_and_reproducer(self) -> None:
        blocked = [p for p in self.platforms if p["status"] == "BLOCKED"]
        self.assertTrue(blocked, "expected at least one BLOCKED platform on this host")
        for p in blocked:
            self.assertGreater(len(p.get("reason", "")), 20, p["id"])
            repro = p.get("reproducer") or {}
            self.assertTrue(repro.get("command", "").strip(), p["id"])
            self.assertTrue(repro.get("expect_output_contains", "").strip(), p["id"])

    def test_non_blocked_entries_have_no_reproducer(self) -> None:
        for p in self.platforms:
            if p["status"] != "BLOCKED":
                self.assertNotIn("reproducer", p, p["id"])

    def pending(self) -> dict:
        return (self.doc.get("pending_evidence") or {}).get("paths", {})

    def test_verified_and_built_entries_name_their_evidence(self) -> None:
        pending = self.pending()
        for p in self.platforms:
            if p["status"] in {"BUILT", "VERIFIED"}:
                self.assertTrue(p.get("evidence"), p["id"])
                for rel in p["evidence"]:
                    if rel in pending:
                        # documented, closed exception: the file ships in another open PR
                        self.assertNotEqual(p["status"], "VERIFIED", f"{p['id']}: VERIFIED may not cite pending evidence {rel}")
                        continue
                    self.assertTrue((ROOT / rel).exists(), f"{p['id']}: evidence {rel} absent")

    def test_pending_evidence_is_declared_with_its_pr(self) -> None:
        for rel, where in self.pending().items():
            self.assertTrue(where.startswith("PR #"), rel)
            self.assertTrue(any(rel in p.get("evidence", []) for p in self.platforms), f"{rel}: pending but cited by no row")

    def test_evidence_paths_are_not_gitignored(self) -> None:
        # A clean checkout only has tracked files; evidence that .gitignore excludes
        # (e.g. *.apk, sect. 11.4.30) would make the previous test fail there.
        for p in self.platforms:
            for rel in p.get("evidence", []):
                if rel in self.pending():
                    continue
                r = subprocess.run(["git", "check-ignore", "-q", "--no-index", rel], cwd=ROOT)
                self.assertNotEqual(r.returncode, 0, f"{p['id']}: evidence {rel} is gitignored")

    def test_apk_claim_is_backed_by_sha256_record(self) -> None:
        android = next(p for p in self.platforms if p["id"] == "android")
        rec = ROOT / "evidence/clients/android/helix-reader-debug.apk.sha256"
        self.assertIn("evidence/clients/android/helix-reader-debug.apk.sha256", android["evidence"])
        self.assertIn("evidence/clients/android/aapt2_badging.txt", android["evidence"])
        digest = rec.read_text().split()[0]
        self.assertEqual(len(digest), 64)
        self.assertIn(digest[:8], android["notes"])
        self.assertIn(digest[-4:], android["notes"])

    def test_captured_reproducers_match_matrix(self) -> None:
        self.assertTrue(CAPTURED.is_file(), f"missing {CAPTURED}; run clients/tools/capture_blocked.py")
        captured = {c["id"]: c for c in load(CAPTURED)["reproducers"]}
        blocked = {p["id"]: p for p in self.platforms if p["status"] == "BLOCKED"}
        self.assertEqual(set(captured), set(blocked))
        for pid, p in blocked.items():
            c = captured[pid]
            self.assertEqual(c["command"], p["reproducer"]["command"], pid)
            self.assertNotEqual(c["exit_code"], 0, f"{pid}: reproducer succeeded, platform is not blocked")
            self.assertIn(p["reproducer"]["expect_output_contains"], c["output"], pid)

    def test_stale_pending_evidence_is_reported(self) -> None:
        # Not a failure (the merge order #5, #10, #18 makes some paths exist at merge time): a warning that the
        # row should be promoted or the entry removed.
        stale = stale_pending(self.doc, ROOT)
        if stale:
            import warnings
            warnings.warn(f"pending_evidence now present on disk, promote or remove: {stale}")

    def test_reproducers_still_fail_on_this_host(self) -> None:
        """Opt-in LIVE probe (HELIX_LIVE_PLATFORM_PROBE=1); not part of the hermetic unit suite.

        Skips with a closed reason code (11.4.69): feature_disabled_by_config when not enabled. The android-tv
        sub-check is skipped (hardware_not_present, printed) when adb is absent, but the other BLOCKED
        reproducers still run. It looks for a device whose ro.build.characteristics has the tv item, not for
        any attached Android device.
        """
        reason = live_probe_skip_reason(os.environ)
        if reason:
            self.skipTest(reason)
        for p in self.platforms:
            if p["status"] != "BLOCKED":
                continue
            if p["id"] == "android-tv":
                adb = find_adb(os.environ)
                if adb is None:
                    print(f"\n{SKIP_NO_HW}: adb not found, android-tv sub-check skipped")
                    continue
                out = subprocess.run([adb, "devices"], capture_output=True, text=True, timeout=30).stdout
                serials = [l.split()[0] for l in out.splitlines()[1:] if len(l.split()) == 2 and l.split()[1] == "device"]
                tv = []
                for sn in serials:
                    ch = subprocess.run([adb, "-s", sn, "shell", "getprop", "ro.build.characteristics"],
                                        capture_output=True, text=True, timeout=30).stdout
                    if is_tv_characteristics(ch):
                        tv.append(sn)
                self.assertEqual(tv, [], f"android-tv: TV-class device now attached {tv}; revisit BLOCKED")
                continue
            r = subprocess.run(["bash", "-c", p["reproducer"]["command"]], capture_output=True, text=True, timeout=30)
            self.assertNotEqual(r.returncode, 0, f"{p['id']}: now succeeds; revisit BLOCKED")
            self.assertIn(p["reproducer"]["expect_output_contains"], r.stdout + r.stderr, p["id"])


class HermeticHelpersTest(unittest.TestCase):
    """Pure-function tests (no host access) with paired mutants in tests/mutate_supported_platforms.py."""

    def test_tv_characteristics_parsing(self) -> None:
        for v in ("tv\n", "tv", "nosdcard,tv\n", "tv,nosdcard\n", "tv\r\n", " Tv , nosdcard \n"):
            self.assertTrue(is_tv_characteristics(v), repr(v))
        for v in ("default\n", "", "\n", "television\n", "nosdcard\n", "tablet,nosdcard\n", "tvbox\n", "mytv,x\n"):
            self.assertFalse(is_tv_characteristics(v), repr(v))

    def test_live_probe_stays_opt_in(self) -> None:
        self.assertTrue(live_probe_skip_reason({}).startswith(SKIP_DISABLED))
        self.assertTrue(live_probe_skip_reason({"HELIX_LIVE_PLATFORM_PROBE": "0"}).startswith(SKIP_DISABLED))
        self.assertTrue(live_probe_skip_reason({"HELIX_LIVE_PLATFORM_PROBE": "yes"}).startswith(SKIP_DISABLED))
        self.assertIsNone(live_probe_skip_reason({"HELIX_LIVE_PLATFORM_PROBE": "1"}))
        self.assertEqual((SKIP_DISABLED, SKIP_NO_HW), ("feature_disabled_by_config", "hardware_not_present"))

    def test_find_adb_never_returns_a_relative_path(self) -> None:
        none = lambda _n: None
        self.assertIsNone(find_adb({}, none))
        self.assertIsNone(find_adb({"ANDROID_HOME": ""}, none))
        self.assertIsNone(find_adb({"ANDROID_HOME": "relative/sdk"}, none))
        self.assertEqual(find_adb({}, lambda _n: "/usr/bin/adb"), "/usr/bin/adb")
        import tempfile
        old = os.getcwd()
        with tempfile.TemporaryDirectory() as d:
            try:
                os.chdir(d)
                (Path(d) / "sdk" / "platform-tools").mkdir(parents=True)
                (Path(d) / "sdk" / "platform-tools" / "adb").write_text("")
                # a relative ANDROID_HOME that resolves from the cwd must still be refused
                self.assertIsNone(find_adb({"ANDROID_HOME": "sdk"}, none))
                self.assertEqual(find_adb({"ANDROID_HOME": str(Path(d) / "sdk")}, none), str(Path(d) / "sdk" / "platform-tools" / "adb"))
            finally:
                os.chdir(old)

    def test_stale_pending_detection(self) -> None:
        import tempfile
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            (root / "here.txt").write_text("x")
            doc = {"pending_evidence": {"paths": {"here.txt": "PR #5", "gone.txt": "PR #5"}}}
            self.assertEqual(stale_pending(doc, root), ["here.txt"])
            self.assertEqual(stale_pending({}, root), [])


if __name__ == "__main__":
    unittest.main()
