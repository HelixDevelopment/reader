#!/usr/bin/env python3
"""assert_host_untouched: LINT that fails if scripts/tools/tests/Dockerfiles/agent prompts contain host-GUI/portal/input patterns.

HONEST SCOPE: this is a static lint. THE CONTAINER IS THE BOUNDARY (tools/isolation/run_container.sh: --network none, one
unix-socket path guarded by a route allow-list, bind allow-list, runtime assertions in qa/isolation/inside/launch.sh).
Residual classes the lint cannot catch: command names computed at run time (chr/hex/base64 held in data, environment- or
file-driven command lines), commands in binaries/third-party code (node_modules, .git objects), tools not yet in the rule
list, and behaviour of code that is merely imported. Third-party dependency trees (node_modules) are skipped; build/out/
dist/.cache/evidence trees are scanned for script-like files; .git/hooks are scanned.

Allowlist (code that runs INSIDE a container image only):
  * qa/isolation/inside/** and submodules/containers/images/** may carry the marker `isolation: runs-inside-container`
    in their first 15 lines (ignored elsewhere). Such files must not be executable and are never referenced from host
    files except through the directory mount.
Exemptions are PINNED in tools/isolation/guard/waivers.txt as `kind|path|sha256|rule|justification|expires` and are void
(finding `waiver-pin-mismatch`) the moment the file's sha256 changes; expired or malformed entries fail too; `kind` is
self (lint's own sources, skipped), denylist (the trailing `# isolation: denylist` marker is honoured in that file only) or
waive (findings of `rule` in that file are dropped). Re-pin deliberately with pin_waivers.py (writes waivers.history).
usage: assert_host_untouched.py [root] [--submodules]   (exit 0 clean, 1 violations, 2 usage; --submodules: report-only)
"""
import os, re, subprocess, sys, hashlib, datetime, fnmatch
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))      # python -I does not add the script dir
from sanitize import escape_text      # scanned names / lines / symlink targets are tree-derived: escape them (review round 9, N1)

SCAN_EXT = {".sh", ".bash", ".zsh", ".py", ".js", ".mjs", ".cjs", ".ts", ".tsx", ".jsx", ".lua", ".go", ".rb", ".pl", ".php", ".ps1",
            ".bat", ".cmd", ".yaml", ".yml", ".json", ".toml", ".mk", ".service", ".desktop", ".kt", ".kts", ".gradle", ".java", ".c", ".cc",
            ".cpp", ".rs", ".dockerfile", ".txt", ".html", ".htm", ".env"}
DATA_EXT = {".json", ".yaml", ".yml", ".toml", ".txt", ".html", ".htm", ".env"}          # prose-ish: only the data-safe rule subset applies
SCRIPT_EXT = {".sh", ".bash", ".zsh", ".py", ".js", ".mjs", ".cjs", ".rb", ".pl", ".php", ".ps1", ".bat", ".cmd", ".gradle", ".kts"}
SCAN_NAMES = {"Makefile", "makefile", "GNUmakefile", "justfile", "Taskfile", "Dockerfile", "Containerfile", "Vagrantfile", "Procfile"}
PROMPT_DIRS = (".claude/agents/", ".claude/commands/", ".claude/skills/", "prompts/")
INSIDE_DIRS = ("qa/isolation/inside/", "submodules/containers/images/")
THIRD_PARTY = {"node_modules"}          # third-party trees: script-like files get only the HARD subset (documented residual)
SCRIPT_ONLY = {"build", "out", "dist", ".cache", ".gradle", "__pycache__", ".venv", "venv", "target", "evidence"}   # only script-like files, full rules
HARD = {"xdotool-family", "uinput", "portal-or-dbus", "container-escape", "gui-automation-lib", "clipboard"}
SHELL_EXT = {".sh", ".bash", ".zsh", ".mk", ".service"}
SHELL_ONLY = {"dynamic-command-sh"}
WAIVER_FILE = "tools/isolation/guard/waivers.txt"
MARKER = re.compile(r"isolation:\s*runs-inside-container")
DENYLINE = re.compile(r"isolation:\s*denylist")
MAXSIZE = 2_000_000

CMD = r"(?:^|[\s;&|(`'\"=/{}])"
def w(name): return re.compile(CMD + name + r"(?![\w-])")
RULES = [
    # input injection / window & clipboard control / screenshots (host display)
    ("xdotool-family", re.compile(r"\b(xdotool|ydotool|dotool|kdotool|wtype|wlrctl|wmctrl|xvkbd|xinput|xrandr|xbindkeys)\b|" + CMD + r"xte\s")),
    ("clipboard", re.compile(r"\b(xclip|xsel|wl-copy|wl-paste|clipman|copyq)\b|pyperclip")),
    ("screenshot", re.compile(r"\b(scrot|maim|flameshot|spectacle|gnome-screenshot|grim|wf-recorder|xwd|xwininfo|xdpyinfo|import\s+(?:\S+\s+)*-window\s+root|screencapture)\b")),
    ("desktop-launch", re.compile(r"\b(xdg-open|gio\s+open|notify-send|kde-open|gnome-open|webbrowser\.open|open_new_tab|swaymsg|hyprctl|i3-msg|gsettings|dconf)\b")),
    ("uinput", re.compile(r"/dev/(?:input/|uinput)")),
    ("uinput-word", re.compile(r"\buinput\b|\bevdev\b|\blibei\b")),                 # code only: bare words are harmless prose in data files
    ("scrcpy", re.compile(r"\bscrcpy\b")),
    ("adb-input", re.compile(r"\badb\b[^\n]*\binput\s+(?:tap|swipe|keyevent|text|touchscreen|roll|press|motionevent)|\badb\b[^\n]*\bscreencap\b[^\n]*\|\s*(?:xclip|display|eog|feh)")),
    ("gui-automation-lib", re.compile(r"\b(pyautogui|pynput|pyscreenshot|ImageGrab|robotjs|autopy|node-key-sender|keybd_event|SendKeys|pywinauto|uiautomation)\b|@nut-tree|nut-js|\bmss\.mss\b|pygetwindow")),
    # portals / session bus / compositor IPC
    ("portal-or-dbus", re.compile(r"\b(dbus-send|dbus-monitor|busctl|gdbus|qdbus|dbus-run-session|pydbus|dbus_next|sd_bus|dbus-python)\b|\bdbus\.(?:Session|System)Bus|Gio\.DBus|org\.freedesktop\.(?:impl\.)?portal|\b(RemoteDesktop|ScreenCast|InputCapture)\b|org\.gnome\.Shell|org\.kde\.KWin|org\.freedesktop\.ScreenSaver")),
    ("screen-capture-device", re.compile(r"x11grab|kmsgrab|pipewire|gst-launch[^\n]*(?:ximagesrc|pipewiresrc)|ximagesrc")),
    # display / session environment
    ("display-env", re.compile(r"\b(DISPLAY|WAYLAND_DISPLAY|XAUTHORITY|XDG_RUNTIME_DIR|XDG_SESSION_TYPE|XDG_SESSION_ID|DBUS_SESSION_BUS_ADDRESS|DBUS_SYSTEM_BUS_ADDRESS|SSH_AUTH_SOCK|WAYLAND_SOCKET)\b")),
    ("x11-socket", re.compile(r"\.X11-unix|/tmp/\.ICE-unix|wayland-\d|/run/user/\d*")),
    # visible browsers / electron / app launches
    ("headed-browser", re.compile(r"headless\s*[:=]\s*(?:false|False|0)\b|--headed\b|--no-headless|headless\s*=\s*false|[\"']headless[\"']\s*:\s*false|HEADED\s*=\s*1|headless:\s*false")),
    ("electron-launch", re.compile(r"\bnpx\s+electron\b|" + CMD + r"electron\s+(?:\.(?=[\s\"']|$)|[\w./-]+\.[cm]?js\b)|node_modules/\.bin/electron\b|(?:spawn|exec|execFile|fork|run|Popen|call|check_call|check_output|launch)\w*\(\s*\[?\s*[\"'](?:\S*/)?electron[\"']")),
    ("browser-launch", re.compile(r"(?:^|[;&|(`]\s*|\b(?:exec|nohup|sudo|env|command|setsid|start|open)\s+)(?:google-chrome(?:-stable|-beta)?|chromium(?:-browser)?|chrome|firefox|brave(?:-browser)?|msedge|microsoft-edge|epiphany|vivaldi|opera)(?:\s|$)|(?:spawn|exec|execFile|fork|run|Popen|call|check_call|check_output|launch)\w*\(\s*\[?\s*[\"'](?:\S*/)?(?:google-chrome\S*|chromium\S*|chrome|firefox|brave\S*|msedge)[\"']")),
    ("android-emulator-windowed", None),   # handled specially below
    # container escapes (checked on continuation-joined lines of podman/docker commands)
    ("container-escape", None),
    # obfuscation / indirection
    ("obfuscated-exec", re.compile(r"base64\s+(?:-d|--decode|-D)\b[^\n|]*\|\s*(?:\w+/)?(?:ba|z|da)?sh\b|b64decode\s*\([^)]*\)[^\n]*(?:exec|eval|system|Popen)|\b(?:exec|eval|system|popen|run|call|check_output|spawn|execSync)\w*\s*\([^\n]*b64decode|\|\s*(?:sudo\s+)?(?:ba|z|da)?sh\s*(?:-s\b|$|\s)|\beval\s+[\"']?\$|\beval\s*\(\s*(?:atob|base64)|\bFunction\s*\(\s*atob")),
    ("dynamic-command", re.compile(r"(?:^|[;&|]\s*|^\s*)\$\{?\w+\}?\$\{?\w+\}?\s")),
    ("dynamic-command-sh", re.compile(r"(?:^|[;&|(]\s*)(?:\$\{\w+\}[\w.-]|\$\([^)]*\)[\w.-])")),
    ("obfuscated-exec", re.compile(r"base64\s+(?:-d|--decode|-D)\b|(?:^|[\s;&|(])(?:source|\.)\s+<\(|<\(\s*(?:curl|wget)\b|\b(?:ba|z|da)?sh\b[^\n]*<<<|\b(?:ba|z|da)?sh\s+-c\s+[\"']?\$|\b(?:curl|wget)\b[^\n]*\|\s*(?:sudo\s+)?(?:ba|z|da)?sh\b")),
    ("host-service-probe", re.compile(r"(?:localhost|127\.\d+\.\d+\.\d+|\[::1\]|0\.0\.0\.0|::1):5037\b|\bANDROID_ADB_SERVER_\w+\b")),
    ("env-recon", re.compile(r"\b(?:env|printenv|set|compgen\s+-e|export\s+-p)\b[^\n|]*\|\s*(?:e?grep|rg|awk|sed|cut)\b[^\n]*(?:wayl|display|xdg|dbus|x11|ssh_auth|session)", re.I)),
    ("qemu-windowed", None),
    ("glob-command", None),
]
EMU = re.compile(CMD + r"emulator\s+(?:@\S+|-avd\b)")
QEMU = re.compile(CMD + r"(?:\S*/)?qemu(?:-system-[\w]+|-kvm)?\s")
TOOLS = ["xdotool", "ydotool", "dotool", "wtype", "wmctrl", "xclip", "xsel", "wl-copy", "wl-paste", "scrot", "maim", "grim", "import", "xdg-open",
         "notify-send", "scrcpy", "gnome-screenshot", "flameshot", "spectacle", "xte", "xinput", "xrandr", "dbus-send", "gdbus", "busctl", "qdbus",
         "chromium", "chromium-browser", "google-chrome", "firefox", "electron", "gio", "xwininfo", "xdpyinfo", "xvkbd"]
# R3-I4: the runtime family beyond podman/docker/nerdctl: every other way of starting a process in a namespace / sandbox / container is the same class
# ("a runtime call that bypasses run_container.sh"). Command-position anchored so prose and identifiers do not match.
RUNTIME_EXTRA = (r"(?:^|[\s;&|(`'\"=/{}])(?:(?:docker|podman|nerdctl)\s+compose\s+(?:run|up|start|exec|create)\b|(?:docker|podman|nerdctl)-compose\s+(?:run|up|start|exec|create)\b|podman-remote\s+\S|buildah\s+(?:run|bud|build)\b|runc\s+(?:run|exec|create|start)\b|lxc\s+(?:exec|launch|start|shell|run)\b|"
                 r"lxc-(?:execute|attach|start|create)\b|(?:apptainer|singularity)\s+(?:run|exec|shell|instance)\b|distrobox(?:-enter|-create|\s+(?:enter|create|run))\b|"
                 r"toolbox\s+(?:run|enter|create)\b|flatpak\s+run\b|systemd-run\b|bwrap\b|nsenter\b|unshare\b|chroot\b|"
                 r"crun\s+(?:run|exec|create|start)\b|incus\s+(?:exec|launch|start|shell|run)\b|ctr\s+(?:\S+\s+)*?(?:run|exec|start)\b|kubectl\s+(?:run|exec|debug|attach|cp)\b|"
                 r"firejail\b|toolbx\s+(?:run|enter|create)\b|systemd-nspawn\b|machinectl\s+(?:shell|login|start)\b|flatpak-spawn\b|multipass\s+(?:exec|shell|launch)\b|"
                 r"limactl\s+(?:shell|start)\b|vagrant\s+(?:ssh|up)\b|host-spawn\b|ch-run\b|"
                 r"(?:podman|docker|nerdctl)\s+(?:container\s+runlabel|pod\s+create|start\b[^\n]*\s(?:-a|--attach)\b))|"
                 # Python / JS list form: the runtime word as the first list element ("distrobox", "enter", ...)
                 r"[\"']\s*(?:podman-remote|buildah|runc|crun|lxc|incus|ctr|kubectl|apptainer|singularity|distrobox|toolbox|toolbx|flatpak|flatpak-spawn|systemd-run|systemd-nspawn|machinectl|"
                 r"bwrap|nsenter|unshare|chroot|firejail|multipass|limactl|vagrant|host-spawn|ch-run)\s*[\"']\s*[,\]]")
# R4-I1: a GLOBAL option between the CLI and the subcommand (`podman --root X run`, `docker --context c run`, `docker -H h run`, `podman --remote exec`) must not hide
# the subcommand: tokenize-ish normalisation drops leading `-x`/`--x[=v]`/`--x v` words before run/create/exec/cp/start (an option's value is the next word unless that word is
# itself a subcommand), the original view is still checked as well.
_GLOBOPT = re.compile(r"\b(podman|docker|nerdctl|buildah)((?:\s+--?[A-Za-z][\w-]*(?:=\S+|\s+(?!(?:container|pod|compose|run|create|exec|cp|start|build|runlabel)\b)[^\s-]\S*)?)+)(?=\s+(?:container\s+|pod\s+)?(?:compose\s+(?:run|up|start|exec|create)|run|create|exec|cp|start|runlabel)\b)")
def _normrt(b):
    return _GLOBOPT.sub(lambda m: m.group(1), b)
def _tokview(b):
    """review round 5 (R5-I1): a TOKEN view - quoted strings become one word (inner whitespace -> "_", quotes dropped), list / array separators become spaces - so the same
    shape written as a shell word list, a quoted value with spaces, a Python list ["podman","--root",...] or a JS spawnSync("podman",[...]) array normalises identically."""
    t = re.sub(r'"([^"]*)"|\'([^\']*)\'', lambda m: re.sub(r"\s+", "_", m.group(1) if m.group(1) is not None else m.group(2)), b)
    return re.sub(r"\s+", " ", re.sub(r"[\[\],()]", " ", t))

CONTAINER_CMD = re.compile(r"\b(?:podman|docker|nerdctl|buildah)\s+(?:container\s+)?(?:run|create|exec)\b|\$\{?\w+\}?\s+(?:container\s+)?(?:run|create|exec)\b|" + RUNTIME_EXTRA)
# R2-I2: ANY container-runtime run/create/exec/cp outside tools/isolation/ (the one place that owns the hardened launch path) is a finding unless
# a pinned `waive` row names it - it is how --env-host / --rootfs / -v .:/h / "$PWD" / --pod / --sysctl / --device-cgroup-rule / `podman cp ... $HOME`
# escapes would be written, and the per-flag list can never be complete (the class is "a runtime call that bypasses run_container.sh").
CONTAINER_USE = re.compile(r"\b(?:podman|docker|nerdctl)\s+(?:container\s+)?(?:run|create|exec|cp)\b|[\"']\s*(?:podman|docker|nerdctl)\s*[\"']\s*,\s*[\"'](?:run|create|exec|cp)[\"']|\$\{?\w+\}?\s+(?:container\s+)?(?:run|create|exec|cp)\b|" + RUNTIME_EXTRA)
VARCMD = re.compile(r"\$\{?\w+\}?\s+(?:container\s+)?(?:run|create|exec)\b")
ESCAPES = re.compile(
    r"--(?:net|network)(?:=|\s+)(?!none\b)[^\s-]|--(?:net|network)(?:=|\s+)--|--privileged\b|--(?:pid|ipc|userns|uts|cgroupns)(?:=|\s+)host\b|"
    r"--cap-add\b|--security-opt(?:=|\s+)(?!no-new-privileges)[^\s-]|--device(?:=|\s+)(?!/dev/kvm\b)[^\s-]|map-host-loopback|allow_host_loopback|"
    r"--(?:volumes-from|mount)\b(?![^\n]*type=(?:tmpfs|volume))|--(?:group-add|uidmap|gidmap|ulimit\s+memlock|sysctl)\b[^\n]*host|"
    r"(?:^|\s)(?:-v|--volume)(?:=|\s*)(?:/|~|\$HOME|\$\{HOME\}|\$XDG|\$\{XDG|\$PWD/\.\.|\$\(pwd\)/\.\.|\$\{PWD\}/\.\.|\.\.)|"
    r"(?:-v|--volume|--mount)(?:=|\s*)\S*(?:/\.\.(?:/|:|,|\s|$))|"
    r"(?:--mount)(?:=|\s*)\S*(?:source|src)=(?:/|~|\$)|"
    r"--rootfs\b|--env-host\b|--pod(?:=|\s+)|--security-opt(?:=|\s+)(?:unconfined|seccomp=unconfined|label=disable|apparmor=unconfined)|(?:^|\s)(?:-v|--volume)(?:=|\s*)\.(?:/|:|\s)|--userns(?:=|\s+)host\b|"
    r"volume-opt\b|--env-file\b|--gpus\b|--hooks-dir\b|--annotation\b|host-gateway|"
    r"(?:^|\s)(?:-p|--publish)(?:=|\s*)(?:0\.0\.0\.0:|\[::\]:|\d+:\d+(?:/\w+)?(?:\s|$))|--publish-all\b")
KUBE = re.compile(r"\b(?:podman|docker)\s+volume\s+create\b[^\n]*(?:--opt\b|\s-o\b)|\b(?:podman|docker)\s+(?:kube\s+play|play\s+kube|compose\s+up|machine\s+ssh|system\s+service|--remote|-H\s|--host\s)|\bpodman\s+--(?:url|connection)\b")

def _union(labels):
    return re.compile("|".join(("(?i:%s)" if rx.flags & re.I else "(?:%s)") % rx.pattern for l, rx in RULES if rx is not None and (labels is None or l in labels)))
ANY = _union(None)
_SPLICE1 = re.compile(r"[\"']\s*\+\s*[\"']"); _SPLICE2 = re.compile(r"(?<=\w)(?:''|\"\")(?=\w)"); _SPLICE3 = re.compile(r"(?<=[A-Za-z])\\(?=[A-Za-z])")
_QUOTES = re.compile(r"[\"'\\]")
_GLOBCH = re.compile(r"[*?\[]")
_CMDWORDS = ("compose", "podman", "docker", "nerdctl", "buildah", "crun", "incus", "ctr", "kubectl", "firejail", "toolbx", "systemd-nspawn", "machinectl", "flatpak-spawn", "multipass", "limactl", "vagrant", "host-spawn", "ch-run", "runc", "lxc", "apptainer", "singularity", "distrobox", "toolbox", "flatpak", "systemd-run", "bwrap", "nsenter", "unshare", "chroot")

JSON_RULES = {"headed-browser", "electron-launch", "container-escape", "display-env", "xdotool-family", "uinput", "gui-automation-lib", "portal-or-dbus", "x11-socket"}
ANY_DATA = _union(JSON_RULES)
ANY_HARD = _union(HARD)

# Cheap literal superset of every rule above (lower-case). A line/file with none of these cannot match any RULE, so the slow
# regexes only run on candidates. Keep in sync when adding a rule: tests/test_guard_matrix.py proves every mutant is still caught.
LITS = ("dotool", "wtype", "wlrctl", "wmctrl", "xvkbd", "xinput", "xrandr", "xbindkeys", "xte", "xclip", "xsel", "wl-copy", "wl-paste", "clipman",
        "copyq", "pyperclip", "scrot", "maim", "flameshot", "spectacle", "grim", "wf-recorder", "xwd", "xwininfo", "xdpyinfo", "import", "screencapture",
        "xdg-open", "gio", "notify-send", "kde-open", "gnome-open", "webbrowser", "open_new_tab", "swaymsg", "hyprctl", "i3-msg", "gsettings", "dconf",
        "/dev/", "uinput", "evdev", "libei", "scrcpy", "adb", "pyautogui", "pynput", "pyscreenshot", "imagegrab", "robotjs", "autopy", "node-key-sender",
        "keybd_event", "sendkeys", "pywinauto", "uiautomation", "nut", "mss", "pygetwindow", "dbus", "busctl", "gdbus", "qdbus", "pydbus", "sd_bus",
        "portal", "remotedesktop", "screencast", "inputcapture", "org.gnome", "org.kde", "org.freedesktop", "x11grab", "kmsgrab", "pipewire", "ximagesrc",
        "display", "xauthority", "xdg_", "ssh_auth", "wayland", "x11", ".ice-unix", "/run/user", "headless", "headed", "electron", "chrome", "chromium",
        "firefox", "brave", "msedge", "epiphany", "vivaldi", "opera", "base64", "b64decode", "atob", "eval", "sh", "$", "5037", "grep", "rg", "awk",
        "sed", "cut", "source", " . ", "<(")
HARD_LITS = ("dotool", "wtype", "wlrctl", "wmctrl", "xvkbd", "xinput", "xrandr", "xbindkeys", "xte ", "uinput", "evdev", "libei", "/dev/input", "dbus",
             "busctl", "gdbus", "qdbus", "portal", "remotedesktop", "screencast", "inputcapture", "org.gnome", "org.kde", "pyautogui", "pynput", "robotjs",
             "autopy", "xclip", "xsel", "wl-copy", "wl-paste", "clipman", "copyq", "pyperclip", "podman", "docker", "nerdctl", "buildah", "keybd_event",
             "sendkeys", "pywinauto", "uiautomation", "nut-js", "@nut-tree", "mss.", "pygetwindow", "imagegrab", "pyscreenshot", "node-key-sender", "sd_bus", "pydbus")
HARD_RX = re.compile("|".join(re.escape(l) for l in HARD_LITS))
LIT_RX = re.compile("|".join(re.escape(l) for l in LITS))

def normalise(line):
    """Undo cheap string splicing before matching: x''dotool, "xdo"+"tool", 'xdo' 'tool', backslash-split words."""
    l = _SPLICE1.sub("", line)          # "a"+"b"
    l = _SPLICE2.sub("", l)             # xdo''tool
    return _SPLICE3.sub("", l)          # xdo\tool

def dequote(line):
    """Second matching view: every quote and backslash removed (x"d"otool, --net="host", -v "/":/h)."""
    return _QUOTES.sub("", normalise(line))

def glob_hit(body):
    """A word that contains glob/charclass metacharacters and expands to a known GUI/escape tool name."""
    for tok in re.findall(r"[\w./*?\[\]!^-]+", body):
        base = tok.rsplit("/", 1)[-1]
        if not _GLOBCH.search(base) or len(re.sub(r"\[[^\]]*\]|[*?]", "", base)) < 3:
            continue
        if any(fnmatch.fnmatchcase(t, base) for t in TOOLS):
            return base
    return None

def is_shell(rel, path):
    base = os.path.basename(rel)
    if os.path.splitext(base)[1].lower() in SHELL_EXT or base in SCAN_NAMES or base.startswith(("Dockerfile", "Containerfile")):
        return True
    try:
        with open(path, "rb") as fh:
            first = fh.readline(200)
    except OSError:
        return False
    return first.startswith(b"#!") and (b"sh" in first or b"bash" in first or b"zsh" in first)

def is_binary(path):
    try:
        with open(path, "rb") as fh:
            return b"\0" in fh.read(8192)
    except OSError:
        return True

def sha256(p):
    h = hashlib.sha256()
    with open(p, "rb") as fh:
        for ch in iter(lambda: fh.read(1 << 16), b""):
            h.update(ch)
    return h.hexdigest()

# ---------------------------------------------------------------- file discovery
def git_files(root):
    try:
        top = subprocess.run(["git", "-C", root, "rev-parse", "--show-toplevel"], capture_output=True, check=True).stdout.decode().strip()
        if os.path.realpath(top) != os.path.realpath(root):
            return None
        # tracked + untracked + IGNORED files all count
        out = subprocess.run(["git", "-C", root, "ls-files", "-z", "--cached", "--others"], capture_output=True, check=True).stdout.decode().split("\0")
        return [f for f in out if f]
    except Exception:
        return None

def walk_files(root, skip_submodules=True):
    res = []
    for dp, dn, fn in os.walk(root):
        dn[:] = [d for d in dn if d != ".git" and not (skip_submodules and os.path.relpath(os.path.join(dp, d), root) == "submodules")]
        res += [os.path.relpath(os.path.join(dp, f), root) for f in fn]
    return res

def git_hooks(root):
    d = os.path.join(root, ".git", "hooks")
    if not os.path.isdir(d):
        return []
    return [f".git/hooks/{f}" for f in sorted(os.listdir(d)) if not f.endswith(".sample") and os.path.isfile(os.path.join(d, f))]

def has_shebang(path):
    try:
        with open(path, "rb") as fh:
            return fh.read(2) == b"#!"
    except OSError:
        return False

def is_script_like(rel, path):
    return os.path.splitext(rel)[1].lower() in SCRIPT_EXT or has_shebang(path) or os.access(path, os.X_OK)

def classify(rel, path):
    """-> None (not scanned) | "code" (all rules) | "data" (data-safe subset)."""
    base = os.path.basename(rel)
    ext = os.path.splitext(base)[1].lower()
    parts = rel.split("/")[:-1]
    if any(p in THIRD_PARTY for p in parts):
        return "hard" if is_script_like(rel, path) else None
    if any(p in SCRIPT_ONLY for p in parts):
        return "code" if is_script_like(rel, path) else None
    if rel.startswith(PROMPT_DIRS):
        return "data" if ext in DATA_EXT else "code"
    if os.access(path, os.X_OK) and not rel.startswith(".git/"):
        return "code"
    if ext in DATA_EXT:
        return "data"
    if ext in SCAN_EXT or base in SCAN_NAMES or base.startswith(("Dockerfile", "Containerfile")):
        return "code"
    if has_shebang(path):
        return "code"                                          # any extension: a shebang makes it a script
    if ext == "":
        return "data"                                          # extension-less, no shebang: data-safe subset (never silently skipped)
    return None

def candidate_files(root):
    fs = git_files(root)
    if fs is None:
        fs = walk_files(root)
    return [f for f in fs if not f.startswith("submodules/") and not f.startswith(".git/")] + git_hooks(root)

def read_lines(p, maxsize=MAXSIZE):
    try:
        if os.path.getsize(p) > maxsize:
            return None
        with open(p, encoding="utf-8", errors="replace") as fh:
            return fh.read().splitlines()
    except OSError:
        return None

def joined(lines):
    """Yield (first_lineno, text) with backslash/continuation lines joined (shell, Dockerfile)."""
    buf, start = "", 0
    for n, l in enumerate(lines, 1):
        if not buf:
            start = n
        if l.rstrip().endswith("\\"):
            buf += l.rstrip()[:-1] + " "; continue
        yield start, buf + l; buf = ""
    if buf:
        yield start, buf

# ---------------------------------------------------------------- pinned waivers (N3)
KINDS = {"self", "denylist", "waive"}
MAX_DAYS = 366

def load_waivers(root, today=None):
    """-> (entries, problems). Entry: dict(kind,path,sha,rule,why,expires). A line is
    `kind|path|sha256|rule|justification|expires(YYYY-MM-DD)`; exact paths only (no globs); pinned by content hash."""
    today = today or datetime.date.today()
    ents, probs = [], []
    wf = os.path.join(root, WAIVER_FILE)
    if not os.path.isfile(wf):
        return ents, probs
    with open(wf, encoding="utf-8") as fh:
        for n, l in enumerate(fh, 1):
            l = l.rstrip("\n")
            if not l.strip() or l.lstrip().startswith("#"):
                continue
            f = [x.strip() for x in l.split("|")]
            where = f"{WAIVER_FILE}:{n}"
            if len(f) != 6 or f[0] not in KINDS or not re.fullmatch(r"[0-9a-f]{64}", f[2]) or not f[4] or any(c in f[1] for c in "*?[") or f[1].startswith("/") or ".." in f[1].split("/"):
                probs.append((WAIVER_FILE, n, "waiver-malformed", l[:120])); continue
            try:
                exp = datetime.date.fromisoformat(f[5])
            except ValueError:
                probs.append((WAIVER_FILE, n, "waiver-malformed", "expiry is not YYYY-MM-DD")); continue
            if exp < today:
                probs.append((WAIVER_FILE, n, "waiver-expired", f"{f[1]} expired {f[5]}")); continue
            if (exp - today).days > MAX_DAYS:
                probs.append((WAIVER_FILE, n, "waiver-malformed", f"{f[1]}: expiry more than {MAX_DAYS} days away")); continue
            ents.append(dict(kind=f[0], path=f[1], sha=f[2], rule=f[3], why=f[4], expires=f[5], line=n))
    return ents, probs

def pinned(root, ents, probs):
    """Split entries into those whose file exists with the pinned hash (active) and report mismatch/orphan findings."""
    active = {}
    for e in ents:
        p = os.path.join(root, e["path"])
        if not os.path.isfile(p):
            probs.append((WAIVER_FILE, e["line"], "waiver-orphan", f"{e['path']} does not exist")); continue
        if sha256(p) != e["sha"]:
            probs.append((e["path"], 0, "waiver-pin-mismatch", f"content changed since pinned ({e['kind']}); re-pin deliberately with pin_waivers.py")); continue
        if not _recorded(root, e):
            probs.append((WAIVER_FILE, e["line"], "waiver-unrecorded", f"{e['path']} pin {e['sha'][:12]} has no record in waivers.history (use pin_waivers.py)")); continue
        active.setdefault(e["path"], []).append(e)
    return active

def _recorded(root, e):
    hf = os.path.join(root, "tools/isolation/guard/waivers.history")
    if not os.path.isfile(hf):
        return False
    with open(hf, encoding="utf-8") as fh:
        for l in fh:
            f = l.rstrip("\n").split("|")
            if len(f) >= 6 and f[2] == e["path"] and f[4] == e["sha"] and f[1] in ("add", "repin"):
                return True
    return False

# ---------------------------------------------------------------- scan
def scan(root, include_submodules=False, stats=None):
    hits = []
    st = stats if stats is not None else {}
    st.update({"files_scanned": 0, "symlinks": 0, "symlinks_outside": 0, "third_party_scripts": 0, "waivers_active": 0, "waivers_self": 0,
               "waivers_denylist": 0, "waivers_waive": 0})
    ents, probs = load_waivers(root)
    active = pinned(root, ents, probs)
    hits += probs
    for es in active.values():
        for e in es:
            st["waivers_active"] += 1; st["waivers_" + e["kind"]] += 1
    rels = candidate_files(root)
    if include_submodules:
        rels = []
        for sm in os.listdir(os.path.join(root, "submodules")) if os.path.isdir(os.path.join(root, "submodules")) else []:
            d = os.path.join(root, "submodules", sm)
            fs = git_files(d) or walk_files(d)
            rels += [f"submodules/{sm}/{f}" for f in fs]
    for rel in rels:
        p = os.path.join(root, rel)
        if os.path.islink(p):
            st["symlinks"] += 1
            tgt = os.path.realpath(p)
            if not tgt.startswith(os.path.realpath(root) + os.sep):
                st["symlinks_outside"] += 1
                if classify(rel, tgt):
                    hits.append((rel, 0, "symlink-outside-repo", f"-> {tgt}"))
                continue
            p = tgt
        if not os.path.isfile(p):
            continue
        mine = active.get(rel, [])
        if any(e["kind"] == "self" for e in mine):
            continue
        inside_ok = rel.startswith(INSIDE_DIRS) and not include_submodules
        if inside_ok and os.access(p, os.X_OK):
            hits.append((rel, 0, "inside-file-executable", "container-only file has the exec bit"))
        kind = classify(rel, p)
        if kind is None or is_binary(p):
            continue
        lines = read_lines(p, 400_000 if kind == "hard" else MAXSIZE)
        if lines is None:
            continue
        if kind == "hard":
            whole = "\n".join(lines)
            if not HARD_RX.search(whole.lower()) or (not ANY_HARD.search(whole) and not (CONTAINER_CMD.search(whole) and ESCAPES.search(whole))):
                st["third_party_scripts"] += 1
                continue
        shell = is_shell(rel, p)
        st["files_scanned"] += 1
        if kind == "hard":
            st["third_party_scripts"] += 1
        if inside_ok and any(MARKER.search(l) for l in lines[:15]):
            continue
        deny_pinned = any(e["kind"] == "denylist" for e in mine)
        waived = {e["rule"] for e in mine if e["kind"] == "waive"}
        fh = []
        for n, text in joined(lines):
            if DENYLINE.search(text):
                if deny_pinned and rel.startswith("tools/isolation/"):
                    continue
                fh.append((rel, n, "denylist-marker-unpinned", text.strip()[:160]))
            if len(text) < 3:
                continue
            nb = normalise(text)
            views = (nb, dequote(text)) if _QUOTES.search(nb) else (nb,)
            quick = ANY_DATA if kind == "data" else ANY_HARD if kind == "hard" else ANY
            any_rule = LIT_RX.search(views[-1].lower()) is not None and any(quick.search(v) for v in views)
            if any_rule:
                for label, rx in RULES:
                    if rx is None:
                        continue
                    if kind == "data" and label not in JSON_RULES:
                        continue
                    if kind == "hard" and label not in HARD:
                        continue
                    if label in SHELL_ONLY and not shell:
                        continue
                    if any(rx.search(b) and not _benign(label, b) for b in views):
                        fh.append((rel, n, label, text.strip()[:160]))
            low = nb.lower()
            if not any(w in low for w in _CMDWORDS) and not VARCMD.search(nb) and "kube" not in low and "qemu" not in low and "emulator" not in low and "isolation" not in low and not (shell and _GLOBCH.search(nb)):
                continue
            cviews = tuple(dict.fromkeys(views + tuple(_normrt(v) for v in views) + tuple(_normrt(_tokview(v)) for v in views)))
            if any(CONTAINER_CMD.search(b) and ESCAPES.search(b) for b in cviews) or (kind == "code" and any(KUBE.search(b) for b in views)):
                fh.append((rel, n, "container-escape", text.strip()[:160]))
            if kind == "code" and any(CONTAINER_USE.search(b) for b in cviews):   # no blanket tools/isolation/ exemption (review round 3): each legitimate file carries its own pinned waiver row
                fh.append((rel, n, "container-outside-isolation", text.strip()[:160]))
            if kind != "code":
                continue
            if any(EMU.search(b) for b in views) and not any("-no-window" in b for b in views):
                fh.append((rel, n, "emulator-without--no-window", text.strip()[:160]))
            if any(QEMU.search(b) for b in views) and not any(re.search(r"-display\s+none|-nographic|-display=none", b) for b in views):
                fh.append((rel, n, "qemu-windowed", text.strip()[:160]))
            if shell and not rel.startswith(PROMPT_DIRS) and _GLOBCH.search(nb):
                g = next((glob_hit(b) for b in views if glob_hit(b)), None)
                if g:
                    fh.append((rel, n, "glob-command", f"{g}: {text.strip()[:140]}"))
            if "isolation" in low and re.search(r"isolation/inside/[\w*?\[{$]|cd\s+\S*isolation/inside", views[-1]) and not rel.startswith(INSIDE_DIRS):
                fh.append((rel, n, "inside-script-referenced-from-host", text.strip()[:160]))
        hits += [h for h in fh if h[2] not in waived]
    return hits

def _benign(label, body):
    # the container-private display value is the only DISPLAY use that is fine
    if label == "display-env" and re.search(r"DISPLAY\W{0,3}:99\b", body) and not re.search(r"WAYLAND|XAUTHORITY|XDG_|DBUS|SSH_AUTH", body):
        return True
    return False

def main(argv):
    args = [a for a in argv[1:] if not a.startswith("--")]
    sub = "--submodules" in argv
    root = args[0] if args else os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))
    if not os.path.isdir(root):
        print("usage: assert_host_untouched.py [root] [--submodules]", file=sys.stderr); return 2
    st = {}
    hits = scan(root, include_submodules=sub, stats=st)
    for rel, n, label, text in hits:
        print(f"{escape_text(rel)}:{int(n)}: FORBIDDEN[{escape_text(label)}] {escape_text(text)}")
    print(f"NOTICE pinned waivers active: {int(st['waivers_active'])} (self {int(st['waivers_self'])}, denylist {int(st['waivers_denylist'])}, waive {int(st['waivers_waive'])}); "
          f"third-party script files given the HARD subset only: {int(st['third_party_scripts'])}")
    print(f"assert_host_untouched: {'REPORT' if sub else ('FAIL' if hits else 'PASS')} ({len(hits)} finding(s); files scanned {int(st['files_scanned'])}, "
          f"symlinks {int(st['symlinks'])} (outside repo {int(st['symlinks_outside'])}))")
    return 0 if sub else (1 if hits else 0)

if __name__ == "__main__":
    sys.exit(main(sys.argv))
