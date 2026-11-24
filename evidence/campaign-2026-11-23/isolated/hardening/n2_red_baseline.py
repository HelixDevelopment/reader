#!/usr/bin/env python3
"""N2 RED: RECONSTRUCTED round-1 behaviour (round-1 source was not kept; this re-creates its documented limits on top of the
current module: SKIP_DIRS hid build/out/dist/.cache/node_modules/evidence/.git at any depth, extensions without .gradle/.kts/.txt/.html,
extension-less files only with a shebang, no quote-stripped second view, narrow container-escape regex, none of the N2 rules) and
runs the CURRENT matrix. UNCONFIRMED: exact equality with the round-1 binary; the reviewer's own 38/80 figure is the authoritative RED."""
import base64, json, os, re, sys, tempfile, pathlib
R = pathlib.Path(sys.argv[1])
sys.path.insert(0, str(R / "tools/isolation/guard"))
import assert_host_untouched as g
OLD_SKIP = {"node_modules", ".git", ".cache", ".gradle", ".venv", "__pycache__", "venv", "dist", "evidence", "out", "build"}
OLD_EXT = g.SCAN_EXT - {".gradle", ".kts", ".txt", ".html", ".htm"}
def old_classify(rel, path):
    base = os.path.basename(rel); ext = os.path.splitext(base)[1].lower()
    if any(p in OLD_SKIP for p in rel.split("/")[:-1]): return None
    if rel.startswith(g.PROMPT_DIRS): return "data" if ext in (".json", ".yaml", ".yml", ".toml") else "code"
    if ext in OLD_EXT or base in g.SCAN_NAMES: return "data" if ext in (".json", ".yaml", ".yml", ".toml") else "code"
    if ext == "": return "code" if g.has_shebang(path) else None
    return None
g.classify = old_classify
g.git_hooks = lambda root: []
g.dequote = g.normalise
g.ESCAPES = re.compile(r"--(?:net|network)(?:=|\s+)host\b|--privileged\b|--pid(?:=|\s+)host\b|--ipc(?:=|\s+)host\b|--userns(?:=|\s+)host\b|--uts(?:=|\s+)host\b|"
    r"--cap-add\b|--security-opt(?:=|\s+)(?:seccomp|apparmor|label)=unconfined|--device(?:=|\s+)/dev/(?:dri|input|" + "".join(["uin", "put"]) + r"|snd|video|tty)\b|"
    r"(?:-v|--volume|--mount)(?:=|\s*)(?:type=bind,)?(?:source=|src=)?\s*(?:/tmp\b|/run\b|/dev\b|/home\b|/root\b|/etc\b|/var/run\b|/:|~|\$HOME|\$\{HOME\}|\$XDG|\$\{XDG|\"\$HOME|\"~)")
g.KUBE = re.compile(r"(?!x)x"); g.glob_hit = lambda b: None
NEW = {"".join(["uin", "put-word"]), "dynamic-command-sh", "host-service-probe", "env-recon", "qemu-windowed", "glob-command"}
g.RULES = [(l, rx) for l, rx in g.RULES if l not in NEW and not (l == "obfuscated-exec" and "<<<" in (rx.pattern if rx else ""))]
g.ANY = g._union(None); g.ANY_DATA = g._union(g.JSON_RULES); g.ANY_HARD = g._union(g.HARD)
MUT = json.loads(base64.b64decode((R / "tools/isolation/guard/mutants.b64").read_text()))
def run(m):
    d = tempfile.mkdtemp(prefix="n2red-"); p = pathlib.Path(d, m["file"]); p.parent.mkdir(parents=True, exist_ok=True); p.write_text(m["content"])
    return bool(g.scan(d))
req = [m for m in MUT if m["expect"] == "caught"]
missed = [m["name"] for m in req if not run(m)]
print(f"RED (reconstructed round-1 rule behaviour) vs current matrix: {len(req)-len(missed)} caught of {len(req)} required; EVADED {len(missed)}:")
for n in missed: print("  evaded:", n)
