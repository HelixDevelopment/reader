#!/usr/bin/env python3
"""pin_waivers.py - the ONLY sanctioned way to add or change an exemption in waivers.txt.

  pin_waivers.py add  KIND PATH RULE "justification" YYYY-MM-DD     pin PATH at its current sha256
  pin_waivers.py repin PATH "reason"                                 deliberately accept PATH's new content (every entry for PATH)
  pin_waivers.py remove PATH "reason"                                drop every entry for PATH
  pin_waivers.py list

KIND is self | denylist | waive. Every change is appended to waivers.history (`utc|action|path|old_sha|new_sha|reason`); the lint
reports `waiver-unrecorded` for any pinned hash that has no history record, so a hand-edited hash cannot pass the gate.
Expiry must be within 366 days. Run from the repository root (or pass --root)."""
import datetime, hashlib, os, re, sys

WF = "tools/isolation/guard/waivers.txt"
HF = "tools/isolation/guard/waivers.history"
HEADER = ("# kind|path|sha256|rule|justification|expires  -  exact paths, content-pinned; edit ONLY via tools/isolation/guard/pin_waivers.py\n"
          "# kind: self (lint's own source, not scanned) | denylist (the trailing marker is honoured in that file) | waive (findings of `rule` in that file)\n")

def sha(p):
    return hashlib.sha256(open(p, "rb").read()).hexdigest()

def read(root):
    ents = []
    p = os.path.join(root, WF)
    if os.path.isfile(p):
        for l in open(p, encoding="utf-8"):
            if l.strip() and not l.lstrip().startswith("#"):
                ents.append([x.strip() for x in l.rstrip("\n").split("|")])
    return ents

def write(root, ents):
    os.makedirs(os.path.dirname(os.path.join(root, WF)), exist_ok=True)
    with open(os.path.join(root, WF), "w", encoding="utf-8") as fh:
        fh.write(HEADER)
        for e in sorted(ents, key=lambda e: (e[0], e[1], e[3])):
            fh.write("|".join(e) + "\n")

def hist(root, action, path, old, new, reason):
    ts = datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    with open(os.path.join(root, HF), "a", encoding="utf-8") as fh:
        fh.write(f"{ts}|{action}|{path}|{old}|{new}|{reason.replace('|', '/')}\n")

def main(a):
    root = os.getcwd()
    if a and a[0] == "--root":
        root, a = a[1], a[2:]
    if not a:
        print(__doc__); return 2
    cmd, args = a[0], a[1:]
    ents = read(root)
    if cmd == "list":
        for e in ents:
            print("|".join(e))
        return 0
    if cmd == "add" and len(args) == 5:
        kind, path, rule, why, exp = args
        if kind not in ("self", "denylist", "waive") or not os.path.isfile(os.path.join(root, path)) or re.search(r"[*?\[]", path) or path.startswith("/") or ".." in path.split("/"):
            print("pin_waivers: bad KIND/PATH", file=sys.stderr); return 2
        d = datetime.date.fromisoformat(exp)
        if (d - datetime.date.today()).days > 366 or d < datetime.date.today():
            print("pin_waivers: expiry must be in the future and within 366 days", file=sys.stderr); return 2
        if not why.strip():
            print("pin_waivers: justification required", file=sys.stderr); return 2
        ents = [e for e in ents if not (e[0] == kind and e[1] == path and e[3] == rule)]
        h = sha(os.path.join(root, path))
        ents.append([kind, path, h, rule, why, exp]); write(root, ents); hist(root, "add", path, "-", h, why)
        print(f"pinned {kind} {path} {h[:12]} (expires {exp})"); return 0
    if cmd == "repin" and len(args) == 2:
        path, reason = args; n = 0
        for e in ents:
            if e[1] == path:
                old = e[2]; e[2] = sha(os.path.join(root, path)); n += 1
                if old != e[2]:
                    hist(root, "repin", path, old, e[2], reason)
                    print(f"RE-PINNED {path}: {old[:12]} -> {e[2][:12]}  ({reason})")
        if not n:
            print("pin_waivers: no entry for that path", file=sys.stderr); return 2
        write(root, ents); return 0
    if cmd == "remove" and len(args) == 2:
        path, reason = args; keep = [e for e in ents if e[1] != path]
        if len(keep) == len(ents):
            print("pin_waivers: no entry for that path", file=sys.stderr); return 2
        write(root, keep); hist(root, "remove", path, "-", "-", reason); print(f"removed {path}"); return 0
    print(__doc__); return 2

if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
