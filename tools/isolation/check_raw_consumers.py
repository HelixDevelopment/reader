#!/usr/bin/env python3
"""check_raw_consumers.py [ROOT] - gate (review round 5 R5-B1(d), widened in round 6 R6-B1/R6-I1): DEFENCE IN DEPTH for the rule "container output is read only through the raw-capture mode".
THE STRUCTURAL FIX is in run_container.sh itself: attached mode (no -o, no -d) REFUSES, exit 67, when its stdout is not a terminal, so no redirection / pipe / substitution /
process-substitution / exec-redirect / variable-indirection shape can capture the terminal stream. This gate is a LINT on top of that: it is a text scan and has blind spots
(see docs/test-isolation.md section 8.4: a command assembled at run time, a script in a language it does not parse, a file outside qa/ tools/ scripts/ tests/); it does NOT
claim to catch "any capture", the wrapper's refusal does.
It flags, in .sh / .bash / .py files under qa, tools, scripts, tests (marker "raw-consumer-exempt: <reason>" on the line / in the function silences one finding):
  shell  : an invocation of run_container.sh (a logical line, continuations joined) whose OPTIONS (parsed like getopts, up to the first "--": so a "-o" inside the container
           command does not count) hold neither -o nor -d; an assignment that stores run_container.sh in a variable or array (indirection); process substitution / "exec >" /
           "&>" / "| grep -o" shapes need no rule of their own, they are invocations without -o and are flagged by the first rule.
  python : a call (subprocess.*, os.popen, os.system, asyncio.create_subprocess_*) whose command (a list/tuple literal, a name assigned from one, or a string) names
           run_container.sh and whose options hold neither "-o" nor "-d"; ANY such call with shell=True / os.popen / os.system; a hand-made call of the container runtime's run / exec / logs / cp / attach verbs
           that captures output (use rawrun.run_direct, whose str surfaces are sanitized); and any use of the UNSAFE-TO-PRINT attributes raw_stdout / raw_stderr outside
           tools/isolation/rawrun.py (a parser uses Raw.json_after / Raw.has_token).
Files that TEST the wrapper itself (test_review_round*.py, test_bindcheck.py) and this gate's own files are skipped by name.
exit 0 PASS / 1 FAIL (lists file:line) / 2 usage."""
import ast, os, re, shlex, sys

SKIP = re.compile(r"(^|/)(test_review_round\d+\.py|test_bindcheck\.py|check_raw_consumers\.py|test_raw_consumers\.py|rawrun\.py|run_container\.sh)$")
RAWLIB = re.compile(r"(^|/)rawrun\.py$")
EXEMPT = re.compile(r"raw-consumer-exempt:\s*\S")
DIRS = ("qa", "tools", "scripts", "tests")
EXTS = (".sh", ".bash", ".py")
WITH_ARG = set("nmctibprwe")          # run_container.sh getopts "dn:m:c:t:i:b:p:o:r:w:e:"
UNSAFE_ATTRS = ("raw_stdout", "raw_stderr")
RT_VERBS = {"run", "exec", "logs", "cp", "attach", "start"}


def flags_of(tokens):
    """the option letters a run_container.sh invocation really carries: getopts-style, stops at the first "--" or the first non-option word."""
    got, i = set(), 0
    while i < len(tokens):
        t = tokens[i]
        if t == "--" or not t.startswith("-") or t == "-": break
        letters = t[1:]
        for j, ch in enumerate(letters):
            got.add(ch)
            if ch in WITH_ARG:
                if j == len(letters) - 1: i += 1          # the argument is the next word
                break
        i += 1
    return got


def shell_logical(text):
    out, cur, start = [], "", 0
    for n, ln in enumerate(text.splitlines(), 1):
        if not cur: start = n
        if ln.rstrip().endswith("\\"): cur += ln.rstrip()[:-1] + " "; continue
        out.append((start, cur + ln)); cur = ""
    return out


def check_shell(path, text):
    bad = []
    for n, ln in shell_logical(text):
        if re.search(r"(^|[\s;|&(`$])(podman|docker)\s+(?:\S+\s+)*?(logs|attach)\b", ln) and not ln.lstrip().startswith("#") and not EXEMPT.search(ln):
            bad.append((path, n, "podman/docker logs|attach prints the container stream raw: pipe it through tools/isolation/sanitize.py or use run_container.sh -o")); continue
        if "run_container.sh" not in ln or ln.lstrip().startswith("#") or EXEMPT.search(ln): continue
        code = ln.split(" #")[0]
        try: toks = shlex.split(code, posix=True)
        except ValueError: bad.append((path, n, "unparsable line that names run_container.sh (cannot prove it has -o)")); continue
        idx = [k for k, t in enumerate(toks) if t.endswith("run_container.sh") and not t.startswith("-")]
        if not idx:                                              # named, but not a command word: a path in an echo / a test / a variable
            if re.match(r"\s*(export\s+|local\s+|readonly\s+|declare\s+\S+\s+)?[A-Za-z_]\w*=\(?[^ ]*run_container\.sh", code) or re.search(r"\w=\(?\"?[^ ]*run_container\.sh", code):
                bad.append((path, n, "run_container.sh stored in a variable / array: an indirect call cannot be linted, call it directly with -o or -d"));
            continue
        for k in idx:
            if k > 0 and not toks[k].startswith(("<(", ">(", "$(", "(")) and toks[k - 1] in ("echo", "printf", "cat", "ls", "test", "[", "[[", "grep", "stat", "sha256sum", "chmod"): continue
            if not ({"o", "d"} & flags_of(toks[k + 1:])):
                bad.append((path, n, "run_container.sh invoked in attached mode (no -o / -d among its options): its stdout must be a terminal; a machine consumer uses -o RAWDIR"))
    return bad


def _strs(node):
    return [c.value for c in ast.walk(node) if isinstance(c, ast.Constant) and isinstance(c.value, str)]


def _is_proc_call(call):
    f = ast.unparse(call.func)
    return bool(re.search(r"(^|\.)(run|Popen|call|check_call|check_output|getoutput|getstatusoutput|popen|system|create_subprocess_exec|create_subprocess_shell)$", f))


def _captures(call):
    f = ast.unparse(call.func)
    if f.endswith(("check_output", "popen", "getoutput", "getstatusoutput")): return True
    return any(k.arg == "capture_output" or (k.arg == "stdout" and "PIPE" in ast.unparse(k.value)) for k in call.keywords)


def _py_flags(cmd):
    """option letters of a literal command: the strings of a list/tuple after the run_container.sh element."""
    if isinstance(cmd, (ast.List, ast.Tuple)):
        toks, seen = [], False
        for e in cmd.elts:
            if isinstance(e, ast.Constant) and isinstance(e.value, str):
                if seen: toks.append(e.value)
                elif e.value.endswith("run_container.sh"): seen = True
            elif isinstance(e, ast.Starred):
                toks.append("*")                              # unknown extras: stop here, what we saw is all we can prove
                break
            else:
                if seen: toks.append("?")
                elif "run_container.sh" in ast.unparse(e): seen = True
        return flags_of(toks)
    if isinstance(cmd, ast.Constant) and isinstance(cmd.value, str):
        try: toks = shlex.split(cmd.value)
        except ValueError: return set()
        ix = [k for k, t in enumerate(toks) if t.endswith("run_container.sh")]
        return flags_of(toks[ix[0] + 1:]) if ix else set()
    return set()


def check_python(path, text):
    try: tree = ast.parse(text)
    except SyntaxError: return []
    lines = text.splitlines(); bad = []
    in_lib = bool(RAWLIB.search(path))
    for n in ast.walk(tree):                                      # the UNSAFE-TO-PRINT raw streams outside the library
        if not in_lib and ((isinstance(n, ast.Attribute) and n.attr in UNSAFE_ATTRS) or (isinstance(n, ast.Constant) and isinstance(n.value, str) and n.value in UNSAFE_ATTRS)):
            if not EXEMPT.search(lines[n.lineno - 1]):
                bad.append((path, n.lineno, "uses the UNSAFE-TO-PRINT raw_stdout / raw_stderr outside rawrun.py (parse with Raw.json_after / Raw.has_token, show with .stdout / .stderr)"))
    scopes = [tree] + [n for n in ast.walk(tree) if isinstance(n, (ast.FunctionDef, ast.AsyncFunctionDef))]
    for sc in scopes:
        own, todo = [], list(ast.iter_child_nodes(sc))      # the scope's own nodes: nested functions are scopes of their own
        while todo:
            n = todo.pop(); own.append(n)
            if not isinstance(n, (ast.FunctionDef, ast.AsyncFunctionDef)): todo.extend(ast.iter_child_nodes(n))
        named = {}
        for n in own:
            if isinstance(n, ast.Assign) and any("run_container.sh" in x for x in _strs(n.value)) and isinstance(n.value, (ast.List, ast.Tuple, ast.Constant)):
                for t in n.targets:
                    if isinstance(t, ast.Name): named[t.id] = n.value
        seg = "\n".join(lines[getattr(sc, "lineno", 1) - 1: getattr(sc, "end_lineno", len(lines))])
        for n in own:
            if not isinstance(n, ast.Call) or not _is_proc_call(n) or EXEMPT.search(seg): continue
            cmd = n.args[0] if n.args else None
            if isinstance(cmd, ast.Name): cmd = named.get(cmd.id, cmd)
            if cmd is None: continue
            text_ = " ".join(_strs(cmd))
            shell_form = any(k.arg == "shell" and ast.unparse(k.value) == "True" for k in n.keywords) or re.search(r"\.(popen|system|getoutput|getstatusoutput|create_subprocess_shell)$", ast.unparse(n.func))
            if "run_container.sh" in text_:
                if shell_form:
                    bad.append((path, n.lineno, "run_container.sh run through a shell string (shell=True / os.popen / os.system): use a list with -o, or tools/isolation/rawrun.py"))
                elif not ({"o", "d"} & _py_flags(cmd)):
                    bad.append((path, n.lineno, "run_container.sh invoked in attached mode (no \"-o\" / \"-d\" among its options): use tools/isolation/rawrun.py run_raw"))
            elif isinstance(cmd, (ast.List, ast.Tuple)) and cmd.elts and isinstance(cmd.elts[0], ast.Constant) and os.path.basename(str(cmd.elts[0].value)) in ("podman", "docker"):
                words = [e.value for e in cmd.elts[1:] if isinstance(e, ast.Constant) and isinstance(e.value, str)]
                verbs = [w for w in words if w in RT_VERBS]            # first runtime verb among the words (global options, their values and `container` are skipped; an image named like a verb can fool this: a lint)
                b = verbs[0] if verbs else None
                if b in RT_VERBS and (_captures(n) or b in ("logs", "attach")) and not (b == "run" and "-d" in words):   # `run -d` prints only the container id; logs/attach print the stream even with inherited stdout
                    bad.append((path, n.lineno, f"podman {b} output is captured by hand or shown raw: use tools/isolation/rawrun.py run_direct (sanitized str surfaces)"))
    return sorted(set(bad))


def check_messages(path, text):
    """round 8 (B1): a tool that prints a host message embedding an f-string value must pass it through escape_text (a container can plant file names with OSC 52 / CSI / bidi).
    Scope: tools/isolation/*.py and qa/isolation/*.py; print(f"...{x}...", file=sys.stderr/stdout) where some {x} is not an escape_text(...) call, unless the whole print argument is one."""
    try: tree = ast.parse(text)
    except SyntaxError: return []
    lines, bad = text.splitlines(), []
    is_esc = lambda e: isinstance(e, ast.Call) and ast.unparse(e.func).endswith("escape_text")
    for n in ast.walk(tree):
        if not (isinstance(n, ast.Call) and ast.unparse(n.func) == "print" and n.args): continue
        a = n.args[0]
        if is_esc(a) or EXEMPT.search(lines[n.lineno - 1]): continue
        parts = [a] if isinstance(a, ast.JoinedStr) else [x for x in ast.walk(a) if isinstance(x, ast.JoinedStr)]
        for js in parts:
            for fv in js.values:
                if isinstance(fv, ast.FormattedValue) and not is_esc(fv.value) and not isinstance(fv.value, ast.Constant) and not (isinstance(fv.value, ast.Call) and ast.unparse(fv.value.func) in ("len", "int")):
                    bad.append((path, n.lineno, "print() interpolates a value that is not wrapped in sanitize.escape_text (a container-planted name could carry terminal escapes)")); break
            else: continue
            break
    return bad


def scan(root):
    bad = []
    for d in DIRS:
        for dp, dn, fn in os.walk(os.path.join(root, d)):
            dn[:] = [x for x in dn if x not in (".git", "node_modules", "__pycache__")]
            for f in fn:
                p = os.path.join(dp, f); rel = os.path.relpath(p, root)
                if SKIP.search(rel) or not f.endswith(EXTS): continue
                try: text = open(p, encoding="utf-8").read()
                except (OSError, UnicodeDecodeError): continue
                if not f.endswith(".py"):
                    if "run_container.sh" in text or re.search(r"(podman|docker)\s.*\b(logs|attach)\b", text): bad += check_shell(rel, text)
                    continue
                if "run_container.sh" in text or "raw_std" in text or "podman" in text: bad += check_python(rel, text)
                if re.match(r"(tools|qa)/isolation/[^/]+\.py$", rel): bad += check_messages(rel, text)
    return bad


def main(argv):
    if len(argv) > 2: print(__doc__, file=sys.stderr); return 2
    root = os.path.abspath(argv[1]) if len(argv) == 2 else os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
    bad = scan(root)
    for p, n, m in bad: print(f"check_raw_consumers: {p}:{n}: {m}", file=sys.stderr)
    print(f"check_raw_consumers: {'FAIL' if bad else 'PASS'} ({len(bad)} finding(s))"); return 1 if bad else 0


if __name__ == "__main__": sys.exit(main(sys.argv))
