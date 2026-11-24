#!/usr/bin/env python3
"""HRD-076 capture manifest, written HOST-SIDE after the capture container is gone (nothing here trusts a container-written hash).
usage: make_capture_manifest.py --dir <evidence/screenshots/X> --client NAME --version STR --command "producing command" [--scenes FILE] [--image-id ID] [--server-sha SHA]
Reads <dir>/scenes.json (or --scenes) = {"scenes":[{"file","scenario","viewport"?,"color_scheme"?,"note"?}]}; every file in the manifest must exist and
be a regular file; every png/webm/jpg in <dir> must be listed (nothing unlisted ships). Writes <dir>/capture_manifest.json (sha256, bytes, pixel
resolution read from the PNG header). Exit 1 on any mismatch."""
import argparse, hashlib, json, os, struct, sys, datetime
from pathlib import Path

def png_size(p):
    with open(p, "rb") as f:
        h = f.read(24)
    if h[:8] != b"\x89PNG\r\n\x1a\n" or h[12:16] != b"IHDR":
        raise SystemExit(f"{p}: not a PNG")
    return struct.unpack(">II", h[16:24])

def main():
    a = argparse.ArgumentParser()
    a.add_argument("--dir", required=True); a.add_argument("--client", required=True); a.add_argument("--version", required=True)
    a.add_argument("--command", required=True); a.add_argument("--scenes"); a.add_argument("--image-id", default="UNCONFIRMED"); a.add_argument("--server-sha", default="UNCONFIRMED")
    a.add_argument("--git-head", default="UNCONFIRMED"); a.add_argument("--note", default="")
    o = a.parse_args()
    d = Path(o.dir)
    sc = json.loads(Path(o.scenes or d / "scenes.json").read_text())["scenes"]
    listed = {s["file"] for s in sc}
    media = {p.name for p in d.iterdir() if p.suffix in (".png", ".webm", ".jpg", ".mp4")}
    if listed != media:
        print("manifest mismatch: listed-not-present", sorted(listed - media), "present-not-listed", sorted(media - listed)); return 1
    items = []
    for s in sorted(sc, key=lambda s: s["file"]):
        p = d / s["file"]
        if not p.is_file() or p.is_symlink():
            print("not a regular file", p); return 1
        b = p.read_bytes()
        e = {"file": s["file"], "client": o.client, "client_version": o.version, "scenario": s["scenario"],
             "sha256": hashlib.sha256(b).hexdigest(), "bytes": len(b), "producing_command": o.command}
        if p.suffix == ".png":
            w, h = png_size(p); e["resolution_px"] = f"{w}x{h}"
        else:
            e["resolution_px"] = s.get("viewport", "UNCONFIRMED"); e["kind"] = "video"
        for k in ("viewport", "color_scheme", "note"):
            if s.get(k): e[k] = s[k]
        items.append(e)
    doc = {"task": "HRD-076", "client": o.client, "client_version": o.version, "captured_utc": datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
           "container_image_id": o.image_id, "server_binary_sha256": o.server_sha, "git_head": o.git_head,
           "isolation": "rootless podman, --network none, private Xvfb/headless browser inside the container, only the unix-socket bridge to the host test server; see docs/test-isolation.md",
           "note": o.note, "items": items}
    (d / "capture_manifest.json").write_text(json.dumps(doc, indent=2) + "\n")
    print(f"{o.client}: {len(items)} items")
    return 0
sys.exit(main())
