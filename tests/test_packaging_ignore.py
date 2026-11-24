"""Packaging hygiene: local-only paths are ignored; the large bundle is referenced by hash, not committed."""
import hashlib, json, os, pathlib, shutil, subprocess, tempfile, unittest

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
MANIFEST = os.path.join(ROOT, "docs", "bundles", "koodo-reader.manifest.json")
SKILLS = ["action-prefix-system", "constitution", "jev-decision-layer", "label-sync", "media-validator",
          "multitrack", "reporting-workable-items", "scheduled-work-queue", "session-sync", "skill-catalog"]
IGNORED = [".claude/skills/" + s for s in SKILLS] + [
    ".claude/settings.json.bak", ".claude/settings.json.label-sync.bak.20261123T224932Z", "notes.bak",
    "evidence/campaign-2026-11-23/research/bundles/koodo-reader-fe1abf4b8960.bundle",
    "docs/requests/background_queue.md", "clients/android/.env.sh"]
NOT_IGNORED = [".claude/settings.json", ".claude/skills/speckit-plan/SKILL.md", "docs/requests/history.md",
               "evidence/campaign-2026-11-23/research/bundles/SHA256SUMS", "docs/bundles/koodo-reader.manifest.json",
               "clients/desktop/test/transport.test.js",
               "clients/android/android/app/src/test/java/org/helixreader/app/ApiRequestPolicyTest.java"]


def check_ignore(path, root=ROOT):
    return subprocess.run(["git", "-C", root, "check-ignore", "-q", "--no-index", path]).returncode == 0


def sha256_of(path):
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(1 << 20), b""):
            h.update(chunk)
    return h.hexdigest()


def manifest_matches(manifest, file_path):
    return (os.path.getsize(file_path) == manifest["size"] and sha256_of(file_path) == manifest["sha256"])


class IgnoreTests(unittest.TestCase):
    def test_ignored(self):
        for p in IGNORED:
            with self.subTest(p=p):
                self.assertTrue(check_ignore(p), p)

    def test_not_ignored(self):
        for p in NOT_IGNORED:
            with self.subTest(p=p):
                self.assertFalse(check_ignore(p), p)

    def test_mutant_gitignore_without_rules_is_detected(self):
        # paired mutant: a repo with an empty .gitignore must fail the ignored-set check
        with tempfile.TemporaryDirectory() as d:
            subprocess.run(["git", "init", "-q", d], check=True)
            pathlib.Path(d, ".gitignore").write_text("")
            self.assertFalse(all(check_ignore(p, d) for p in IGNORED))


class ManifestTests(unittest.TestCase):
    def setUp(self):
        with open(MANIFEST) as f:
            self.m = json.load(f)

    def test_fields(self):
        for k in ("path", "size", "sha256", "reproduce", "kind"):
            self.assertIn(k, self.m)
        self.assertRegex(self.m["sha256"], r"^[0-9a-f]{64}$")

    def test_hash_matches_file_on_disk_if_present(self):
        cands = [os.path.join(ROOT, self.m["path"]), os.path.join(os.environ.get("HRD_BUNDLE_ROOT", "/nonexistent"), self.m["path"])]
        f = next((c for c in cands if os.path.isfile(c)), None)
        if f is None:
            self.skipTest("bundle not present in this checkout (SKIP, not PASS): set HRD_BUNDLE_ROOT to the worktree holding it")
        self.assertTrue(manifest_matches(self.m, f))

    def test_matcher_accepts_good_and_rejects_mutant(self):
        with tempfile.TemporaryDirectory() as d:
            f = os.path.join(d, "x.bundle")
            data = b"abc" * 1000
            pathlib.Path(f).write_bytes(data)
            m = {"size": len(data), "sha256": hashlib.sha256(data).hexdigest()}
            self.assertTrue(manifest_matches(m, f))
            pathlib.Path(f).write_bytes(b"abd" + data[3:])  # same size, one byte mutated
            self.assertFalse(manifest_matches(m, f))


if __name__ == "__main__":
    unittest.main()
