#!/usr/bin/env python3
"""HRD-070 rule enumerator (round 6).

Scope is DERIVED, not hand-listed: `covered` runs the seed run of every fuzz
target's package with a coverage profile (-coverpkg for the lcp package) and
writes covered_functions.txt = every production function with a covered
statement. `check` then enumerates EVERY branch of EVERY such function and
requires a rulemap.tsv row for it, so a function the seed run executes cannot be
left out of the rule table.

Branch = a line starting if / else / case / switch / for / default, a
multi-line condition counted from its first to its last line, and any other line
holding && or || (assignment/return booleans).  Row span = the statement's block.
A row's mutant must have EDITED ONLY lines inside the row's own span (a mutant that edits lines of several rows counts for none of them) (mutation.log
carries "edits <file> <a>-<b>" per mutant); the outcome must match the
disposition (KILL -> KILLED, EQUIV/UNREACHABLE -> SURVIVED).
usage:
  enumerate_rules.py covered <repo-root> <out covered_functions.txt>
  enumerate_rules.py check   <repo-root> <covered_functions.txt> <mutation.log> <rulemap.tsv> <out RULES.md>
  enumerate_rules.py list    <repo-root> <covered_functions.txt>
  enumerate_rules.py map     <repo-root> <covered_functions.txt> <mutation.log> <old rulemap (why source)> <equiv.tsv> <out rulemap.tsv>
"""
import re, sys, os, subprocess, tempfile, collections

SEED = {
 'core': [('./pkg/lcpfuzz', 'Fuzz|TestHRD070|TestSeed', ['./pkg/lcp'])],
 'server': [('./internal/kosync', 'FuzzPutProgressBody|FuzzRegisterInput|TestHRD070', ['./internal/kosync']),
            ('./internal/opds', 'Fuzz|TestHRD070', ['./internal/opds'])],
}
FUNC = re.compile(r'^func (?:\((?:\w+ )?\*?(\w+)\) )?(\w+)')
START = re.compile(r'^(if |\} else|else|case |for |switch |default:)')

def funcs_of(path):
    L = open(path).read().split('\n'); out = []; i = 0
    while i < len(L):
        m = FUNC.match(L[i])
        if m:
            name = (m.group(1) + '.' if m.group(1) else '') + m.group(2)
            j = i
            if not (L[i].rstrip().endswith('}') and L[i].count('{') == L[i].count('}')):
                j = i + 1
                while j < len(L) and L[j] != '}': j += 1
            out.append((name, i + 1, j + 1)); i = j
        i += 1
    return out

def covered(root, outp):
    res = set()
    for mod, specs in SEED.items():
        for pkg, run, cpkgs in specs:
            with tempfile.TemporaryDirectory() as td:
                prof = os.path.join(td, 'c.out')
                cmd = ['go', 'test', pkg, '-run', run, '-count=1', '-coverprofile=' + prof, '-coverpkg=' + ','.join(cpkgs)]
                r = subprocess.run(cmd, cwd=os.path.join(root, mod), capture_output=True, text=True)
                if r.returncode: sys.exit('seed run failed: ' + r.stdout + r.stderr)
                for l in open(prof).read().split('\n')[1:]:
                    m = re.match(r'(.+?):(\d+)\.\d+,(\d+)\.\d+ (\d+) (\d+)$', l)
                    if not m or int(m.group(5)) == 0: continue
                    f = m.group(1).split('helixreader/helix/', 1)[1]
                    a, b = int(m.group(2)), int(m.group(3))
                    for name, s, e in funcs_of(os.path.join(root, f)):
                        if s <= a and b <= e: res.add((f, name))
    open(outp, 'w').write('\n'.join(f'{f}\t{n}' for f, n in sorted(res)) + '\n')
    print(f'{len(res)} covered production functions')

def indent(l): return len(l) - len(l.lstrip('\t '))

def branches(root, cov):
    byfile = collections.defaultdict(set)
    for l in open(cov):
        if l.strip(): f, n = l.rstrip('\n').split('\t'); byfile[f].add(n)
    out = []
    for f, names in sorted(byfile.items()):
        L = open(os.path.join(root, f)).read().split('\n')
        for name, s, e in funcs_of(os.path.join(root, f)):
            if name not in names: continue
            nbefore = len(out)
            seen = {}; i = s  # L[i] is the line after the func line (0-based index s)
            consumed = set()
            while i < e - 1:
                line = L[i]; t = line.strip()
                if t.startswith('//') or not t: i += 1; continue
                if START.match(t):
                    k = i; text = t
                    while not re.sub(r'\s*//.*$', '', text).rstrip().endswith('{') and not t.endswith(':') and k + 1 < e:
                        k += 1; text += ' ' + L[k].strip()
                        if re.sub(r'\s*//.*$', '', text).rstrip().endswith('{'): break
                    text = re.sub(r'\s*//.*$', '', text)
                    ind = indent(line); end = k
                    for q in range(k + 1, e):
                        lq = L[q]
                        if not lq.strip(): continue
                        iq = indent(lq)
                        if iq < ind or (iq == ind and not lq.strip().startswith('}')): break
                        if iq == ind and lq.strip().startswith('} else'): break
                        end = q
                    key = (f, name, text); seen[key] = seen.get(key, 0) + 1
                    out.append((f, name, i + 1, end + 1, text, seen[key]))
                    for q in range(i, k + 1): consumed.add(q)
                    i = k + 1; continue
                if ('&&' in t or '||' in t) and i not in consumed:
                    text = re.sub(r'\s*//.*$', '', t)
                    key = (f, name, text); seen[key] = seen.get(key, 0) + 1
                    out.append((f, name, i + 1, i + 1, text, seen[key]))
                i += 1
            if len(out) == nbefore:  # branch-less covered function: one FUNC row over the whole function
                out.append((f, name, s, e, 'func ' + name, 1))
    return out

def check(root, covp, logp, mapp, outp):
    br = branches(root, covp)
    log = {}; edits = {}; cur = None
    for l in open(logp):
        m = re.match(r'MUTANT (\S+): (.*)', l)
        if m: cur = m.group(1); log[cur] = m.group(2).strip(); continue
        m = re.match(r'  edits (\S+) (.+)$', l)
        if m and cur:
            for r in m.group(2).split(','):
                a, b = r.split('-'); edits.setdefault(cur, []).append((m.group(1), int(a), int(b)))
    rows = {}
    for l in open(mapp):
        if l.startswith('#') or not l.strip(): continue
        f, fn, occ, text, disp, why = l.rstrip('\n').split('\t')
        rows[(f, fn, text, int(occ))] = (disp, why)
    errs = []; used = set(); out = []
    for f, fn, a, b, t, occ in br:
        key = (f, fn, t, occ)
        if key not in rows: errs.append(f"UNMAPPED {f}:{a} {fn}: {t}"); continue
        used.add(key); disp, why = rows[key]
        kind = disp.split(':')[0]; ms = disp.split(':', 1)[1].split(',') if ':' in disp else []
        want = 'SURVIVED' if kind in ('EQUIV', 'UNREACHABLE') else 'KILLED'
        status = []; hit = False
        for mu in ms:
            r = log.get(mu)
            if r is None: errs.append(f"NO-LOG {mu} (for {f}:{a})"); status.append(f"{mu}=?"); continue
            if not r.startswith(want) or 'BAD' in r: errs.append(f"WRONG {mu}: log says '{r}' want {want} ({f}:{a})")
            ed = edits.get(mu, [])
            own = bool(ed) and all(ef.endswith(f) and a <= ea and eb <= b for ef, ea, eb in ed)
            hit = hit or own
            status.append(f"{mu}={r.split()[0]}{'' if own else '*'}")
        if kind not in ('KILL', 'EQUIV', 'UNREACHABLE'): errs.append(f"BAD-DISPOSITION {f}:{a}")
        elif not ms: errs.append(f"NO-MUTANT {f}:{a}")
        elif not hit: errs.append(f"NO-OWN-LINE-MUTANT {f}:{a}-{b} {fn}: no listed mutant edits a line inside this branch (edits: {[(m, edits.get(m)) for m in ms]})")
        out.append((f, a, b, fn, t, kind, ', '.join(status), why))
    for k in rows:
        if k not in used: errs.append(f"STALE map row {k}")
    nfun = len(set((f, n) for f, n, *_ in br))
    with open(outp, 'w') as o:
        o.write("# HRD-070 rule table (generated by enumerate_rules.py)\n\n")
        o.write("SCOPE (exact): every production function with a statement executed by the seed run of the six fuzz targets' packages\n"
                "(core/pkg/lcpfuzz -> coverpkg core/pkg/lcp; server/internal/kosync; server/internal/opds), as listed in covered_functions.txt\n"
                "(derived by `enumerate_rules.py covered`). NOT in scope: other packages (e.g. catalog), functions the seed run does not execute,\n"
                "and the HTTP handler layer server/internal/httpapi/kosync.go (not fuzzed; tracked follow-up, see NOTES.txt).\n\n")
        o.write(f"{len(br)} branches in {nfun} branch-bearing covered functions; {len(errs)} problems.\n"
                "A `*` after a mutant outcome means that mutant edits a line outside this row (extra evidence, not the row's own binding).\n\n")
        o.write("| source | function | branch (text) | disposition | mutant(s) = outcome | oracle / why |\n|---|---|---|---|---|---|\n")
        for f, a, b, fn, t, kind, st, why in out:
            tt = t.replace('|', '\\|')
            o.write(f"| {f}:{a}-{b} | {fn} | `{tt}` | {kind} | {st} | {why} |\n")
    if errs: print('\n'.join(errs)); sys.exit(1)
    print(f"OK: {len(br)} branches in {nfun} covered functions, all mapped; each row has an own-line mutant with the expected outcome")

def mapmode(root, covp, logp, oldp, equivp, outp):
    """Write a rulemap.tsv from the mutation log: a row is KILL:<own-line mutants that were KILLED>, or the
    EQUIV/UNREACHABLE entry from equiv.tsv (key text -> disposition + justification) when every own-line mutant
    SURVIVED. Rows the log cannot support are written with disposition NONE so `check` fails on them."""
    br = branches(root, covp)
    log = {}; edits = {}; cur = None
    for l in open(logp):
        m = re.match(r'MUTANT (\S+): (.*)', l)
        if m: cur = m.group(1); log[cur] = m.group(2).strip(); continue
        m = re.match(r'  edits (\S+) (.+)$', l)
        if m and cur:
            for r in m.group(2).split(','):
                a, b = r.split('-'); edits.setdefault(cur, []).append((m.group(1), int(a), int(b)))
    old = {}
    for l in open(oldp):
        if l.startswith('#') or not l.strip(): continue
        f, fn, occ, text, disp, why = l.rstrip('\n').split('\t'); old[(f, fn, text, int(occ))] = why
    eq = {}
    for l in open(equivp):
        if l.startswith('#') or not l.strip(): continue
        f, fn, occ, text, disp, why = l.rstrip('\n').split('\t'); eq[(f, fn, text, int(occ))] = (disp, why)
    orc = {}
    for l in open(os.path.join(os.path.dirname(os.path.abspath(__file__)), 'oracle.tsv')):
        if l.startswith('#') or not l.strip(): continue
        f, fn, why = l.rstrip('\n').split('\t'); orc[(f, fn)] = why
    rows = []
    for f, fn, a, b, t, occ in br:
        key = (f, fn, t, occ)
        own = [mu for mu, ed in edits.items() if ed and all(ef.endswith(f) and a <= ea and eb <= b for ef, ea, eb in ed)]
        killed = [mu for mu in own if log[mu].startswith('KILLED') and 'BAD' not in log[mu]]
        surv = [mu for mu in own if log[mu].startswith('SURVIVED') and 'BAD' not in log[mu]]
        why = old.get(key) or orc.get((f, fn)) or 'seed-run oracle for this function (see oracle.tsv)'
        if key in eq:
            disp, w = eq[key]
            mus = [mu for mu in surv]
            rows.append((f, fn, occ, t, disp.split(':')[0] + ':' + ','.join(mus), w))
        elif killed:
            rows.append((f, fn, occ, t, 'KILL:' + ','.join(killed), why))
        else:
            rows.append((f, fn, occ, t, 'NONE:' + ','.join(own), 'no killed own-line mutant and no equiv.tsv entry'))
    with open(outp, 'w') as o:
        o.write('# file\tfunc\toccurrence\tbranch text\tdisposition\toracle / justification (generated by enumerate_rules.py map from mutation.log + equiv.tsv + oracle.tsv)\n')
        for r in rows: o.write('\t'.join(str(x) for x in r) + '\n')
    print(f'{len(rows)} rows written; {sum(1 for r in rows if r[4].startswith("NONE"))} unsupported')

if __name__ == '__main__':
    m = sys.argv[1]
    if m == 'covered': covered(sys.argv[2], sys.argv[3])
    elif m == 'check': check(*sys.argv[2:7])
    elif m == 'map': mapmode(*sys.argv[2:8])
    elif m == 'list':
        for f, fn, a, b, t, occ in branches(sys.argv[2], sys.argv[3]): print(f"{f}\t{fn}\t{occ}\t{t}\t{a}-{b}")
