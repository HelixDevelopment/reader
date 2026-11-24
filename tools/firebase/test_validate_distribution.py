import json, os, shutil, subprocess, sys, tempfile, unittest
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent))
import validate_distribution as v

GOOD_FB = {"hosting": {"public": "web", "rewrites": [{"source": "**", "destination": "/index.html"}]}}

def make_tree(placeholder=True, apk=True, notes=True, fb=None, extra=None):
    d = Path(tempfile.mkdtemp())
    fd = d / "deploy/firebase"; fd.mkdir(parents=True)
    (fd / "firebase.json").write_text(json.dumps(fb or GOOD_FB))
    pid = "helix-reader-REPLACE_ME" if placeholder else "helix-reader-prod1"
    (fd / ".firebaserc").write_text(json.dumps({"projects": {"default": pid}}))
    app = "1:000000000000:android:REPLACE_ME" if placeholder else "1:123456789012:android:abcdef0123456789"
    (fd / "app-distribution.json").write_text(json.dumps({
        "app_id": app, "artifact": "out/app.apk", "release_notes_file": "deploy/firebase/release-notes.md",
        "groups_file": "deploy/firebase/groups.txt", "credentials": {"env": "GOOGLE_APPLICATION_CREDENTIALS"},
        "hosting_public": "web"}))
    (fd / "groups.txt").write_text("helix-internal\n")
    if notes: (fd / "release-notes.md").write_text("notes\n")
    (d / "web").mkdir(); (d / "web/index.html").write_text("<html></html>")
    if apk:
        (d / "out").mkdir(); (d / "out/app.apk").write_bytes(b"apk")
    for k, val in (extra or {}).items():
        (d / k).parent.mkdir(parents=True, exist_ok=True); (d / k).write_text(val)
    return d

class T(unittest.TestCase):
    def test_placeholder_project_is_not_deployable(self):
        r = v.validate(make_tree(placeholder=True))
        self.assertEqual(r["deployability"], "NOT_DEPLOYABLE")
        self.assertIn("PLACEHOLDER_PROJECT_ID", r["blockers"])
        self.assertIn("PLACEHOLDER_APP_ID", r["blockers"])

    def test_real_ids_and_artifacts_are_deployable_but_still_not_run(self):
        r = v.validate(make_tree(placeholder=False))
        self.assertEqual(r["deployability"], "DEPLOYABLE_PENDING_OPERATOR")
        self.assertEqual(r["deployment"], "NOT RUN")
        self.assertEqual(r["blockers"], [])

    def test_config_valid_never_implies_deployment(self):
        r = v.validate(make_tree(placeholder=False))
        self.assertIn("DEPLOYMENT: NOT RUN", v.render(r))
        r2 = v.validate(make_tree(placeholder=True))
        self.assertIn("DEPLOYMENT: NOT RUN", v.render(r2))
        self.assertNotIn("DEPLOYED", v.render(r2).replace("NOT DEPLOYED", ""))

    def test_missing_artifact_blocks(self):
        r = v.validate(make_tree(placeholder=False, apk=False))
        self.assertIn("ARTIFACT_MISSING", r["blockers"])

    def test_artifact_sha256_reported(self):
        r = v.validate(make_tree(placeholder=False))
        import hashlib
        self.assertEqual(r["artifact"]["sha256"], hashlib.sha256(b"apk").hexdigest())

    def test_missing_release_notes_blocks(self):
        r = v.validate(make_tree(placeholder=False, notes=False))
        self.assertIn("RELEASE_NOTES_MISSING", r["blockers"])

    def test_bad_firebase_json(self):
        for bad in ({"hosting": {"rewrites": []}}, {"hosting": {"public": "x", "rewrites": [{"source": "**"}]}}, {"hosting": {"public": 5}}):
            r = v.validate(make_tree(placeholder=False, fb=bad))
            self.assertIn("FIREBASE_JSON_INVALID", r["config_errors"], bad)

    def test_unparseable_firebase_json(self):
        d = make_tree(placeholder=False); (d / "deploy/firebase/firebase.json").write_text("{nope")
        self.assertIn("FIREBASE_JSON_INVALID", v.validate(d)["config_errors"])

    def test_credentials_detected_without_echo(self):
        secret = "AIza" + "A" * 35
        d = make_tree(placeholder=False, extra={"deploy/firebase/oops.txt": "key=" + secret})
        r = v.validate(d)
        self.assertIn("CREDENTIAL_IN_TRACKED_FILE", r["config_errors"])
        self.assertNotIn(secret, json.dumps(r)); self.assertNotIn(secret, v.render(r))

    def test_service_account_and_private_key_detected(self):
        for body in ('{"type": "service_account"}', "-----BEGIN PRIVATE KEY-----\nx", "1//0" + "a" * 40, "ya29." + "b" * 30):
            d = make_tree(placeholder=False, extra={"deploy/firebase/c.json": body})
            self.assertIn("CREDENTIAL_IN_TRACKED_FILE", v.validate(d)["config_errors"], body[:12])

    def test_cli_exit_codes_and_plan(self):
        d = make_tree(placeholder=True)
        p = subprocess.run([sys.executable, v.__file__, "--root", str(d)], capture_output=True, text=True)
        self.assertEqual(p.returncode, 0, p.stdout + p.stderr)  # config valid
        self.assertIn("DEPLOYMENT: NOT RUN", p.stdout)
        p = subprocess.run([sys.executable, v.__file__, "--root", str(d), "--require-deployable"], capture_output=True, text=True)
        self.assertEqual(p.returncode, 3)
        p = subprocess.run([sys.executable, v.__file__, "--root", str(d), "--plan"], capture_output=True, text=True)
        self.assertIn("appdistribution:distribute", p.stdout); self.assertIn("DEPLOYMENT: NOT RUN", p.stdout)
        self.assertIn("NOT_DEPLOYABLE", p.stdout)

if __name__ == "__main__":
    unittest.main()
