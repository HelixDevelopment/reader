#!/usr/bin/env python3
"""mutation_kill_table.py - §1.1 paired-mutant proof for scripts/gates (HRD-051/052, PR #9 review B2).

Each mutant is ONE exact-string replacement in a scratch copy of the gate scripts that disables one
check; the suite that must notice it is then run. KILLED = the suite exits non-zero; SURVIVED = it still
passes (a missing test). A replacement that does not apply is an ERROR (exit 2), never a silent skip.
The real tree is never modified.
Usage: scripts/gates/mutation_kill_table.py [--only M03,N01] [--baseline FILE]
  --baseline FILE   run the manifest suite against FILE as build_evidence_manifest.sh (RED baseline: pre-fix script)
Exit: 0 all killed, 1 a mutant survived, 2 tooling error.
"""
import os, shutil, subprocess, sys, tempfile

HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.dirname(os.path.dirname(HERE))
B = 'build_evidence_manifest.sh'
# (id, file, suite, description, old, new)
MUTANTS = [
 ('M03', B, 'manifest', 'VERIFIED/PARTIAL/BLOCKED need an evidence file',
  'case "$st" in VERIFIED|PARTIAL|BLOCKED) echo "FAIL $who status $st requires an evidence file (found \'none\')"; fail=1;; esac', ':'),
 ('M04', B, 'manifest', 'ledger sha != real sha compare', '[ "$mode" = check ] && [ "$sha" != "$newsha" ]', 'false'),
 ('M05', B, 'manifest', 'manifest differs from ledger-implied (cmp)', '! cmp -s "$man" "$newman"', 'false'),
 ('M08', B, 'manifest', 'PROVENANCE sha-mismatch check', 'elif [ "$ps" != "$newsha" ]; then', 'elif false; then'),
 ('M10', B, 'manifest', 'source file:line existence check',
  'echo "FAIL $who source \'$src\' is not an existing file:line (line must be 1..last)"; fail=1;', ':;'),
 ('M11', B, 'manifest', 'claim_status closed set', "*) echo \"FAIL $who bad claim_status '$st'\"; fail=1;;", '*) :;;'),
 ('N01', B, 'manifest', 'control-byte refusal in ledger + PROVENANCE (B1)',
  'ctl_check "$led" "ledger"; ctl_check "$root/$SNAP/PROVENANCE.tsv" "PROVENANCE.tsv"', ':'),
 ('N02', B, 'manifest', 'control-byte refusal in PROVENANCE only', 'ctl_check "$root/$SNAP/PROVENANCE.tsv" "PROVENANCE.tsv"', ':'),
 ('N03', B, 'manifest', 'source path escape (C14)', 'case "/$sf/" in //*|*/../*) echo', 'case "/$sf/" in //*|*/NOPE/*) echo'),
 ('N03b', B, 'manifest', 'source realpath-outside-root check (symlinked dir)', 'resolves outside the root"; fail=1; srcbad=1;; esac', 'resolves outside the root"; srcbad=1;; esac'),
 ('N04', B, 'manifest', 'source line 0 (C15)', ' || [ "$sl" -lt 1 ]', ''),
 ('N05', B, 'manifest', 'last_verified date validation (C24)', 'then echo "FAIL $who BAD_DATE', 'then : "FAIL $who BAD_DATE'),
 ('N06', B, 'manifest', 'snapshot must be git-tracked (C16)', 'if ! git --literal-pathspecs -C "$root" ls-files --error-unmatch -- "$sfile" >/dev/null 2>&1; then', 'if false; then'),
 ('N07', B, 'manifest', 'snapshot must be unmodified', 'elif ! git --literal-pathspecs -C "$root" diff --quiet HEAD -- "$sfile" 2>/dev/null || [ -n "$(hidden_flag "$sfile")" ]; then', 'elif false; then'),
 ('B01', B, 'manifest', 'NUL byte member of the control-byte class', "CTL='[\\x00-\\x08", "CTL='[\\x01-\\x08"),
 ('B02', B, 'manifest', 'symlinked PARENT directory not detected (leaf -L only)',
  'if [ -L "$root/$ev" ] || [ "$(realpath -m "$root/$ev")" != "$rr/$ev" ]; then', 'if [ -L "$root/$ev" ]; then'),
 ('B03', B, 'manifest', 'PROVENANCE plain-file-name rule', 'case "$pn" in ""|*/*|*..*)', 'case "$pn" in "")'),
 ('B04', B, 'manifest', 'evidence file must be NON-EMPTY (-s)', 'elif [ -s "$root/$ev" ]; then', 'elif [ -e "$root/$ev" ]; then'),
 ('B05', B, 'manifest', 'byte-exact header check', 'if [ "$hdrline" != "$hdrexp" ]; then', 'if false; then'),
 ('B10', B, 'manifest', 'snapshot file itself must be git-tracked',
  'for sfile in "$SNAP/$b" "$SNAP/PROVENANCE.tsv"; do', 'for sfile in "$SNAP/PROVENANCE.tsv"; do'),
 ('N08', B, 'manifest', 'literal pathspecs (glob char in evidence path)',
  'git --literal-pathspecs -C "$root" ls-files --error-unmatch -- "$ev"', 'git -C "$root" ls-files --error-unmatch -- "$ev"'),
 ('N09', B, 'manifest', 'assume-unchanged / skip-worktree evidence', ' || [ -n "$(hidden_flag "$ev")" ]', ''),
 ('N10', B, 'manifest', 'id grammar [A-Z]+-[0-9]+', 'if [ -n "$id" ] && ! printf', 'if false && ! printf'),
 ('R03', 'run_local_gates.sh', 'runner', 'gate 6 not applied to the --constitution override',
  '"$cg/cm_pr_delivery_account_name_free.sh" --root "$const"', '"$cg/cm_pr_delivery_account_name_free.sh" --root "$repo/submodules/constitution"'),
 ('M14', 'run_local_gates.sh', 'runner', 'gate 5 cm_pr_delivery_anchor_wired wiring replaced by true',
  'bash "$cg/cm_pr_delivery_anchor_wired.sh" --root "$const"', 'true'),
 ('M15', 'run_local_gates.sh', 'runner', 'gate 1 gofmt_clean wiring replaced by true', 'bash "$here/gofmt_clean.sh" "$root"', 'true'),
]

def scratch(tmp, name):
    s = os.path.join(tmp, name)
    os.makedirs(os.path.join(s, 'scripts'))
    shutil.copytree(HERE, os.path.join(s, 'scripts', 'gates'), ignore=shutil.ignore_patterns('__pycache__'))
    shutil.copytree(os.path.join(REPO, 'config'), os.path.join(s, 'config'))
    os.symlink(os.path.join(REPO, 'submodules'), os.path.join(s, 'submodules'))
    os.symlink(os.path.join(REPO, 'scripts', 'pr'), os.path.join(s, 'scripts', 'pr'))
    subprocess.run(['git', 'init', '-q', '.'], cwd=s, check=True, capture_output=True)
    return s

def run_suite(s, suite):
    f = 'test_build_evidence_manifest.sh' if suite == 'manifest' else 'test_run_local_gates.sh'
    p = subprocess.run(['bash', os.path.join(s, 'scripts', 'gates', f)], capture_output=True, text=True)
    out = p.stdout + p.stderr
    fails = [l for l in out.splitlines() if l.startswith('FAIL')]
    return p.returncode, fails, (out.strip().splitlines() or [''])[-1]

def main():
    only, baseline, a = None, None, sys.argv[1:]
    while a:
        if a[0] == '--only': only = set(a[1].split(',')); a = a[2:]
        elif a[0] == '--baseline': baseline = a[1]; a = a[2:]
        else: print('unknown arg', a[0], file=sys.stderr); return 2
    with tempfile.TemporaryDirectory(prefix='mutkill_') as tmp:
        if baseline:
            s = scratch(tmp, 'baseline'); shutil.copy(baseline, os.path.join(s, 'scripts', 'gates', B)); os.chmod(os.path.join(s, 'scripts', 'gates', B), 0o755)
            rc, fails, last = run_suite(s, 'manifest')
            print(f'BASELINE pre-fix script vs new manifest suite: rc={rc} {last}')
            for f in fails: print('  ' + f[:150])
            return 0
        surv = err = 0; killed = total = 0
        if not only:  # positive control: the UNMUTATED scratch copy must pass both suites, else every KILLED below is meaningless
            s = scratch(tmp, 'control')
            for suite in ('manifest', 'runner'):
                rc, fails, last = run_suite(s, suite)
                print(f'CONTROL  unmutated {suite} suite: rc={rc} {last}')
                if rc != 0: print('ERROR    unmutated copy does not pass; kill table invalid'); return 2
        for mid, fn, suite, desc, old, new in MUTANTS:
            if only and mid not in only: continue
            s = scratch(tmp, mid); p = os.path.join(s, 'scripts', 'gates', fn); t = open(p).read()
            if old not in t: print(f'ERROR    {mid} {desc}: mutation does not apply'); err = 1; continue
            open(p, 'w').write(t.replace(old, new, 1))
            rc, fails, last = run_suite(s, suite); total += 1
            if rc != 0:
                killed += 1
                print(f'KILLED   {mid} {desc}  [{suite} suite: {len(fails)} failing case(s); first: {fails[0][:100] if fails else "?"}]')
            else:
                print(f'SURVIVED {mid} {desc}  [{suite} suite still: {last}]'); surv = 1
        print(f'SUMMARY  {killed} killed of {total} mutants defined and run (table defines {len(MUTANTS)})')
        return 2 if err else (1 if surv else 0)

if __name__ == '__main__': sys.exit(main())
