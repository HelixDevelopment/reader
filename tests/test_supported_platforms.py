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

    def test_reproducers_still_fail_on_this_host(self) -> None:
        """Opt-in LIVE probe (HELIX_LIVE_PLATFORM_PROBE=1); not part of the hermetic unit suite.

        Skips with a closed reason code (11.4.69): feature_disabled_by_config when not enabled,
        hardware_not_present for android-tv when no TV-class device is attached. The android-tv
        check looks for a device whose ro.build.characteristics contains 'tv' (adb from PATH or
        $ANDROID_HOME/platform-tools), not for any attached Android device.
        """
        if os.environ.get("HELIX_LIVE_PLATFORM_PROBE") != "1":
            self.skipTest("feature_disabled_by_config: set HELIX_LIVE_PLATFORM_PROBE=1 to run the live host probes")
        for p in self.platforms:
            if p["status"] != "BLOCKED":
                continue
            if p["id"] == "android-tv":
                adb = shutil.which("adb") or str(Path(os.environ.get("ANDROID_HOME", "")) / "platform-tools" / "adb")
                if not Path(adb).exists():
                    self.skipTest("hardware_not_present: adb not found")
                out = subprocess.run([adb, "devices"], capture_output=True, text=True, timeout=30).stdout
                serials = [l.split()[0] for l in out.splitlines()[1:] if l.strip().endswith("device")]
                tv = []
                for sn in serials:
                    ch = subprocess.run([adb, "-s", sn, "shell", "getprop", "ro.build.characteristics"],
                                        capture_output=True, text=True, timeout=30).stdout
                    if "tv" in ch.lower().split(","):
                        tv.append(sn)
                self.assertEqual(tv, [], f"android-tv: TV-class device now attached {tv}; revisit BLOCKED")
                continue
            r = subprocess.run(["bash", "-c", p["reproducer"]["command"]], capture_output=True, text=True, timeout=30)
            self.assertNotEqual(r.returncode, 0, f"{p['id']}: now succeeds; revisit BLOCKED")
            self.assertIn(p["reproducer"]["expect_output_contains"], r.stdout + r.stderr, p["id"])


if __name__ == "__main__":
    unittest.main()
