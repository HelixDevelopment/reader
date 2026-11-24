#!/usr/bin/env python3
"""check_raw_consumers.py [ROOT] - gate (review round 5, R5-B1(d)): every script that PARSES what run_container.sh printed MUST use its raw-capture mode (-o).
Without -o the wrapper's attached output is the sanitized, merged, "[container] "-prefixed, line-capped TERMINAL stream; parsing it is the false-green class round 5 found
(a HelixQA runner that exited 0 with nothing published, a network gate that could never pass).
A "consumer" is an invocation of run_container.sh whose output is captured:
  shell    : a logical line (continuations joined) naming run_container.sh that has a pipe, a command substitution, or a redirection to anything but /dev/null;
  python   : a function / module scope that builds a command list naming run_container.sh and also calls subprocess with capture_output / stdout=PIPE / check_output.
It must contain "-o" (shell: a ` -o ` word; python: the constant "-o"), or the line/scope carries "raw-consumer-exempt: <non-empty reason>" (a call that is refused before any
container exists, or detached). Files that TEST the wrapper itself (test_review_round*.py, test_bindcheck.py) and this gate's own files are skipped by name.
Python consumers should go through tools/isolation/rawrun.py (no literal run_container.sh at all). exit 0 PASS / 1 FAIL (lists file:line) / 2 usage."""
import ast, os, re, sys

SKIP = re.compile(r"(^|/)(test_review_round\d+\.py|test_bindcheck\.py|check_raw_consumers\.py|test_raw_consumers\.py|rawrun\.py|run_container\.sh)$")
EXEMPT = re.compile(r"raw-consumer-exempt:\s*\S")
DIRS = ("qa", "tools", "scripts", "tests")


def shell_logical(text):
    out, cur, start = [], "", 0
    for n, ln in enumerate(text.splitlines(), 1):
        if not cur: start = n
        if ln.rstrip().endswith("\\"): cur += ln.rstrip()[:-1] + " "; continue
        out.append((start, cur + ln)); cur = ""
    return out


def shell_captures(line):
    code = line.split(" #")[0]
    code = code.replace("||", " ").replace("&&", " ").replace("2>&1", " ")
    if "$(" in code or "`" in code or "|" in code: return True
    return bool(re.search(r"(?<![0-9&<])\d?>>?\s*(?!/dev/null)[^\s&]", code.replace("2>/dev/null", " ").replace(">/dev/null", " ").replace("> /dev/null", " ")))


def check_shell(path, text):
    bad = []
    for n, ln in shell_logical(text):
        if "run_container.sh" not in ln or ln.lstrip().startswith("#"): continue
        if not re.search(r'run_container\.sh"?\s', ln) and not ln.rstrip().endswith("run_container.sh"): continue
        if not shell_captures(ln) or re.search(r"\s-o[\s=]", ln) or EXEMPT.search(ln): continue
        bad.append((path, n, "run_container.sh output is captured but the call has no -o (raw-capture) option"))
    return bad


def _strs(node):
    return [c.value for c in ast.walk(node) if isinstance(c, ast.Constant) and isinstance(c.value, str)]


def _captures(call):
    if ast.unparse(call.func).endswith("check_output"): return True
    return any(k.arg == "capture_output" or (k.arg == "stdout" and "PIPE" in ast.unparse(k.value)) for k in call.keywords)


def check_python(path, text):
    """a subprocess call whose COMMAND (a list literal, or a name assigned from one) names run_container.sh and which captures its stdout; the list must hold "-o"."""
    try: tree = ast.parse(text)
    except SyntaxError: return []
    lines = text.splitlines(); bad = []
    scopes = [tree] + [n for n in ast.walk(tree) if isinstance(n, (ast.FunctionDef, ast.AsyncFunctionDef))]
    for sc in scopes:
        own, todo = [], list(ast.iter_child_nodes(sc))      # the scope's own nodes: nested functions are scopes of their own
        while todo:
            n = todo.pop(); own.append(n)
            if not isinstance(n, (ast.FunctionDef, ast.AsyncFunctionDef)): todo.extend(ast.iter_child_nodes(n))
        named = {}
        for n in own:
            if isinstance(n, ast.Assign) and isinstance(n.value, (ast.List, ast.Tuple)) and any("run_container.sh" in x for x in _strs(n.value)):
                for t in n.targets:
                    if isinstance(t, ast.Name): named[t.id] = n.value
        seg = "\n".join(lines[getattr(sc, "lineno", 1) - 1: getattr(sc, "end_lineno", len(lines))])
        for n in own:
            if not isinstance(n, ast.Call) or not n.args or not _captures(n): continue
            cmd = n.args[0]
            if isinstance(cmd, ast.Name): cmd = named.get(cmd.id)
            if not isinstance(cmd, (ast.List, ast.Tuple)) or not any("run_container.sh" in x for x in _strs(cmd)): continue
            if "-o" in _strs(cmd) or EXEMPT.search(seg): continue
            bad.append((path, cmd.lineno, "run_container.sh output is captured but the command has no \"-o\" (use tools/isolation/rawrun.py run_raw)"))
    return sorted(set(bad))


def scan(root):
    bad = []
    for d in DIRS:
        for dp, dn, fn in os.walk(os.path.join(root, d)):
            dn[:] = [x for x in dn if x not in (".git", "node_modules", "__pycache__")]
            for f in fn:
                p = os.path.join(dp, f); rel = os.path.relpath(p, root)
                if SKIP.search(rel) or not f.endswith((".sh", ".py")): continue
                try: text = open(p, encoding="utf-8").read()
                except (OSError, UnicodeDecodeError): continue
                if "run_container.sh" not in text: continue
                bad += (check_shell if f.endswith(".sh") else check_python)(rel, text)
    return bad


def main(argv):
    if len(argv) > 2: print(__doc__, file=sys.stderr); return 2
    root = os.path.abspath(argv[1]) if len(argv) == 2 else os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
    bad = scan(root)
    for p, n, m in bad: print(f"check_raw_consumers: {p}:{n}: {m}", file=sys.stderr)
    print(f"check_raw_consumers: {'FAIL' if bad else 'PASS'} ({len(bad)} finding(s))"); return 1 if bad else 0


if __name__ == "__main__": sys.exit(main(sys.argv))
