#!/usr/bin/env python3
"""Validate the Firebase distribution CONFIGURATION (HRD-025). Stdlib only.

This tool never talks to Firebase, never reads credential values and never
deploys. Its report always states "DEPLOYMENT: NOT RUN". A clean report means
"configuration is well-formed", never "something was distributed".

Exit codes: 0 config well-formed; 2 config errors; 3 --require-deployable and
blockers exist (placeholder ids, missing artifact/notes).
"""
import argparse, hashlib, json, re, subprocess, sys
from pathlib import Path

PLACEHOLDER = re.compile(r"REPLACE_ME", re.I)
APP_ID = re.compile(r"^1:\d{6,}:android:[0-9a-f]{8,}$")
PROJECT_ID = re.compile(r"^[a-z][a-z0-9-]{4,28}[a-z0-9]$")
# Patterns are matched, never echoed.
CRED = [re.compile(p) for p in (
    r"AIza[0-9A-Za-z_\-]{35}", r"ya29\.[0-9A-Za-z_\-]{20,}", r"1//[0-9A-Za-z_\-]{30,}",
    r"-----BEGIN [A-Z ]*PRIVATE KEY-----", r'"type"\s*:\s*"service_account"',
    r'"private_key"\s*:', r"FIREBASE_TOKEN\s*=\s*\S+")]


def _json(path):
    try:
        return json.loads(Path(path).read_text()), None
    except (OSError, ValueError) as e:
        return None, type(e).__name__


def check_firebase_json(doc):
    if not isinstance(doc, dict):
        return False
    h = doc.get("hosting")
    hs = h if isinstance(h, list) else [h]
    for x in hs:
        if not isinstance(x, dict) or not isinstance(x.get("public"), str) or not x["public"]:
            return False
        for r in x.get("rewrites", []):
            if not (isinstance(r, dict) and isinstance(r.get("source"), str) and
                    any(isinstance(r.get(k), str) for k in ("destination", "function", "run", "dynamicLinks"))):
                return False
    return True


def sha256(path):
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for b in iter(lambda: f.read(1 << 20), b""):
            h.update(b)
    return h.hexdigest()


def scan_files(root):
    """Scan git-tracked files plus everything under deploy/firebase and tools/firebase."""
    files = set()
    try:
        out = subprocess.run(["git", "-C", str(root), "ls-files", "-z", "--cached", "--others", "--exclude-standard",
                              "--", "deploy", "tools/firebase", "docs/firebase-distribution.md"],
                             capture_output=True, text=True, timeout=30).stdout
        files |= {root / f for f in out.split("\0") if f}
    except (OSError, subprocess.SubprocessError):
        pass
    for sub in ("deploy/firebase", "tools/firebase"):
        files |= {p for p in (root / sub).rglob("*") if p.is_file()}
    return sorted(p for p in files if p.is_file() and p.stat().st_size < 2_000_000
                  and p.name != "test_validate_distribution.py" and p.name != "validate_distribution.py")


def validate(root):
    root = Path(root)
    fd = root / "deploy/firebase"
    cfg_err, blockers, notes = [], [], []
    fb, e = _json(fd / "firebase.json")
    if fb is None or not check_firebase_json(fb):
        cfg_err.append("FIREBASE_JSON_INVALID")
    rc, e = _json(fd / ".firebaserc")
    pid = (rc or {}).get("projects", {}).get("default") if isinstance(rc, dict) else None
    if not isinstance(pid, str):
        cfg_err.append("FIREBASERC_INVALID")
    elif PLACEHOLDER.search(pid):
        blockers.append("PLACEHOLDER_PROJECT_ID")
    elif not PROJECT_ID.match(pid):
        cfg_err.append("PROJECT_ID_MALFORMED")
    ad, e = _json(fd / "app-distribution.json")
    artifact = {"path": None, "exists": False, "sha256": None}
    if not isinstance(ad, dict):
        cfg_err.append("APP_DISTRIBUTION_JSON_INVALID"); ad = {}
    app = ad.get("app_id")
    if not isinstance(app, str):
        cfg_err.append("APP_ID_MISSING")
    elif PLACEHOLDER.search(app):
        blockers.append("PLACEHOLDER_APP_ID")
    elif not APP_ID.match(app):
        cfg_err.append("APP_ID_MALFORMED")
    if ad.get("artifact"):
        ap = root / ad["artifact"]
        artifact["path"] = ad["artifact"]
        if ap.is_file():
            artifact.update(exists=True, sha256=sha256(ap), bytes=ap.stat().st_size)
        else:
            blockers.append("ARTIFACT_MISSING")
    else:
        cfg_err.append("ARTIFACT_NOT_CONFIGURED")
    for key, code in (("release_notes_file", "RELEASE_NOTES_MISSING"), ("groups_file", "GROUPS_FILE_MISSING")):
        p = ad.get(key)
        if not p or not (root / p).is_file() or not (root / p).read_text().strip():
            blockers.append(code)
    hp = ad.get("hosting_public")
    if hp and not (root / hp / "index.html").is_file():
        blockers.append("HOSTING_BUILD_MISSING")
    if not isinstance(ad.get("credentials"), dict) or not ad["credentials"].get("env"):
        cfg_err.append("CREDENTIAL_ENV_NOT_NAMED")
    hits = []
    for f in scan_files(root):
        try:
            t = f.read_text(errors="ignore")
        except OSError:
            continue
        if any(c.search(t) for c in CRED):
            hits.append(str(f.relative_to(root)))
    if hits:
        cfg_err.append("CREDENTIAL_IN_TRACKED_FILE")
    return {
        "deployment": "NOT RUN",
        "config_valid": not cfg_err,
        "config_errors": cfg_err,
        "blockers": blockers,
        "deployability": "NOT_DEPLOYABLE" if (blockers or cfg_err) else "DEPLOYABLE_PENDING_OPERATOR",
        "artifact": artifact,
        "credential_files": hits,  # file names only, never values
        "notes": ["Credentials are read by the firebase CLI from the environment variable named in app-distribution.json; this tool never reads them."],
    }


def plan(root):
    ad = json.loads((Path(root) / "deploy/firebase/app-distribution.json").read_text())
    return ["firebase appdistribution:distribute " + ad["artifact"] + " --app " + ad["app_id"]
            + " --groups-file " + ad["groups_file"] + " --release-notes-file " + ad["release_notes_file"]
            + " --project <deploy/firebase/.firebaserc default>   # env: " + ad["credentials"]["env"] + " (value never read)",
            "firebase deploy --only hosting --config deploy/firebase/firebase.json --project <default>"]


def render(r, with_plan=None):
    L = ["config_valid: %s" % r["config_valid"], "deployability: %s" % r["deployability"],
         "config_errors: %s" % (r["config_errors"] or "none"), "blockers: %s" % (r["blockers"] or "none"),
         "artifact: %s sha256=%s" % (r["artifact"]["path"], r["artifact"]["sha256"] or "n/a")]
    if r["credential_files"]:
        L.append("credential patterns in: %s" % ", ".join(r["credential_files"]))
    if with_plan:
        L.append("planned commands (NOT executed):"); L += ["  " + c for c in with_plan]
    L.append("DEPLOYMENT: NOT RUN")
    return "\n".join(L)


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("--root", default=str(Path(__file__).resolve().parents[2]))
    ap.add_argument("--plan", action="store_true")
    ap.add_argument("--require-deployable", action="store_true")
    ap.add_argument("--json", action="store_true")
    a = ap.parse_args(argv)
    r = validate(a.root)
    p = plan(a.root) if a.plan and r["config_valid"] else None
    print(json.dumps(r, indent=2) + "\nDEPLOYMENT: NOT RUN" if a.json else render(r, p))
    if not r["config_valid"]:
        return 2
    if a.require_deployable and r["blockers"]:
        return 3
    return 0


if __name__ == "__main__":
    sys.exit(main())
