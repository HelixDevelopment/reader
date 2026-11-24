#!/usr/bin/env bash
# test_run_local_gates.sh - run_local_gates.sh must (a) pass a good scratch tree printing one verdict per
# gate, (b) exit non-zero when ANY single gate fails (mutation per gate), (c) install_hooks.sh must
# write a pre-commit hook only into the target hooks dir it is given and never into this repo's .git.
set -u
here="$(cd "$(dirname "$0")" && pwd)"; repo="$(cd "$here/../.." && pwd)"
base="$(mktemp -d -t ledfix_XXXXXX)"; trap 'rm -rf "$base"' EXIT
pass=0; fail=0; ok(){ pass=$((pass+1)); echo "PASS $1"; }; bad(){ fail=$((fail+1)); echo "FAIL $1"; }
HDR=$'id\tcapability\tbluff_type\tseverity\tclaim_status\tlayer\tsource\tevidence_path\tevidence_sha256\tproducer\tlast_verified\tnotes'
mk() { d="$1"; mkdir -p "$d/docs/evidence" "$d/evidence" "$d/core"
  mkdir -p "$d/config"; cp "$repo/config/delivery.conf" "$d/config/delivery.conf"
  printf 'l1\nl2\nl3\n' > "$d/README.md"; echo ok > "$d/evidence/a.log"; printf 'package x\n\nfunc F() int { return 1 }\n' > "$d/core/a.go"
  { echo "$HDR"; printf 'LED-001\tcap\tdoc-vs-code-drift\thigh\tVERIFIED\tSOURCE\tREADME.md:2\tevidence/a.log\tnone\tcmd\t2026-11-23\tn\n'; } > "$d/docs/claim_reality_ledger.tsv"
  ( cd "$d" && git init -q . && git add -A && git -c user.email=t@t -c user.name=t commit -qm i ) >/dev/null 2>&1
  "$here/build_evidence_manifest.sh" --root "$d" >/dev/null 2>&1; }
g="$here/run_local_gates.sh"
[ -x "$g" ] || { echo "FAIL run_local_gates.sh missing or not executable"; exit 1; }
d="$base/good"; mk "$d"; out="$("$g" --root "$d" 2>&1)"; rc=$?
[ $rc = 0 ] && ok "good tree exits 0" || bad "good tree rc=$rc: $out"
n="$(echo "$out" | grep -c '^GATE .* PASS$')"; [ "$n" = 7 ] && ok "7 per-gate PASS verdict lines" || bad "verdict lines=$n: $out"
# per-gate mutations
d="$base/m1"; mk "$d"; printf 'package x\n\ntype T struct {\n\tA  int\n\tBBB int\n}\n' > "$d/core/bad.go"; out="$("$g" --root "$d" 2>&1)"; rc=$?
{ [ $rc != 0 ] && echo "$out" | grep -q '^GATE gofmt_clean FAIL'; } && ok "gofmt drift fails run" || bad "gofmt mutation rc=$rc"
d="$base/m2"; mk "$d"; echo tamper >> "$d/evidence/a.log"; out="$("$g" --root "$d" 2>&1)"; rc=$?
{ [ $rc != 0 ] && echo "$out" | grep -q '^GATE build_evidence_manifest_check FAIL'; } && ok "evidence tamper fails run" || bad "manifest mutation rc=$rc"
d="$base/m3"; mk "$d"; sed -i 's/doc-vs-code-drift/made-up-type/' "$d/docs/claim_reality_ledger.tsv"; out="$("$g" --root "$d" 2>&1)"; rc=$?
{ [ $rc != 0 ] && echo "$out" | grep -q '^GATE cm_ledger_row_typed_from_closed_vocabulary FAIL'; } && ok "ad-hoc bluff type fails run" || bad "typed mutation rc=$rc"
d="$base/m4"; mk "$d"; sed -i 's/SOURCE/NOT_A_LAYER/' "$d/docs/evidence/gate_evidence_manifest.tsv"; out="$("$g" --root "$d" 2>&1)"; rc=$?
{ [ $rc != 0 ] && echo "$out" | grep -q '^GATE cm_machine_evidence_at_every_gate FAIL'; } && ok "bad layer in manifest fails run" || bad "layer mutation rc=$rc"
d="$base/m5"; mk "$d"; sed -i 's/^landing_form=.*/landing_form=squash/' "$d/config/delivery.conf"; out="$("$g" --root "$d" 2>&1)"; rc=$?
{ [ $rc != 0 ] && echo "$out" | grep -q '^GATE pr_delivery_config_valid FAIL'; } && ok "squash landing form in config fails run" || bad "config mutation rc=$rc"
d="$base/m6"; mk "$d"; rm "$d/config/delivery.conf"; out="$("$g" --root "$d" 2>&1)"; rc=$?
{ [ $rc != 0 ] && echo "$out" | grep -q '^GATE cm_pr_delivery_account_name_free FAIL'; } && ok "missing delivery config fails run" || bad "missing config rc=$rc"
d="$base/m7"; mk "$d"; echo "see milos85vasic here" >> "$d/README.md"; out="$("$g" --root "$d" 2>&1)"; rc=$?
[ $rc = 0 ] && ok "account in a consumer file outside the canon corpus does not trip the canon-corpus gate (scope as documented)" || bad "m7 rc=$rc"
# installer
i="$here/install_hooks.sh"; [ -x "$i" ] || { bad "install_hooks.sh missing"; echo "== $pass passed, $fail failed"; exit 1; }
before="$(ls "$repo/.git/hooks" 2>/dev/null | sort | tr '\n' ' ')"
hd="$base/hooks"; mkdir -p "$hd"; "$i" --hooks-dir "$hd" >/dev/null 2>&1; rc=$?
{ [ $rc = 0 ] && [ -x "$hd/pre-commit" ] && grep -q run_local_gates "$hd/pre-commit"; } && ok "installer writes executable pre-commit into given dir" || bad "installer rc=$rc"
after="$(ls "$repo/.git/hooks" 2>/dev/null | sort | tr '\n' ' ')"; [ "$before" = "$after" ] && ok "repo .git/hooks untouched" || bad "repo hooks changed"
"$i" --hooks-dir "$hd" >/dev/null 2>&1; rc=$?; [ $rc != 0 ] && ok "installer refuses to overwrite foreign/existing hook" || bad "installer overwrote"
"$i" >/dev/null 2>&1; rc=$?; [ $rc = 0 ] && [ "$(ls "$repo/.git/hooks" | sort | tr '\n' ' ')" = "$before" ] && ok "installer without --hooks-dir only prints (dry run)" || bad "installer default mutated or failed"
# round 2: dangling symlink + missing value
ln -s /nonexistent/target "$hd/pre-push-dangling"; mkdir -p "$base/h2"; ln -s /nonexistent/x "$base/h2/pre-commit"
"$i" --hooks-dir "$base/h2" >/dev/null 2>&1; rc=$?; { [ $rc != 0 ] && [ -L "$base/h2/pre-commit" ] && [ "$(readlink "$base/h2/pre-commit")" = /nonexistent/x ]; } && ok "dangling symlink treated as existing, not overwritten" || bad "dangling symlink rc=$rc"
out="$("$i" --hooks-dir 2>&1)"; rc=$?; { [ $rc = 2 ] && echo "$out" | grep -qi usage && ! echo "$out" | grep -qi unbound; } && ok "--hooks-dir without value: usage, rc 2" || bad "missing value rc=$rc: $out"
echo "== $pass passed, $fail failed"; [ $fail = 0 ]
