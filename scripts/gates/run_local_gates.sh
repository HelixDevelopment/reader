#!/usr/bin/env bash
# run_local_gates.sh - the ONE local entry point for the standing gates (HRD-051/052/055 review fix 2).
# There is no CI/CD in this project (§11.4.156); enforcement is local: run this by hand, from the
# optional pre-commit hook (scripts/gates/install_hooks.sh), or from any release script.
# Gates (all run; none short-circuits; exit 1 if ANY fails, 2 on tooling error):
#   1 gofmt_clean                              gofmt -l over core/ and server/ must be empty
#   2 build_evidence_manifest_check            ledger <-> manifest <-> evidence hashes, frozen-evidence
#                                              rule, vacuous/blank-cell guards
#   3 cm_machine_evidence_at_every_gate        constitution §11.4.262 gate over the manifest
#   4 cm_ledger_row_typed_from_closed_vocabulary  constitution §11.4.266(B) gate over the ledger
#   5 cm_pr_delivery_anchor_wired              constitution §11.4.297 anchor wired (runs on the pinned constitution)
#   6 cm_pr_delivery_account_name_free         §11.4.297(D)(1): accounts of config/delivery.conf absent from the rule corpus
#   7 pr_delivery_config_valid                 config/delivery.conf parses under scripts/pr/deliver.py (closed sets, no unknown key)
# Output: one line per gate "GATE <name> PASS|FAIL" (gate detail printed above it), then a summary.
# Usage: scripts/gates/run_local_gates.sh [--root DIR] [--constitution DIR]
#        --root          tree under test (default: this repository)
#        --constitution  constitution tree gates 5 and 6 inspect (default: submodules/constitution; the
#                        override exists so the wiring of gate 5/6 can be mutation-tested)
set -u
here="$(cd "$(dirname "$0")" && pwd)"; repo="$(cd "$here/../.." && pwd)"; root="$repo"; const="$repo/submodules/constitution"
while [ $# -gt 0 ]; do case "$1" in --root) root="$(cd "$2" && pwd)" || exit 2; shift 2;; --constitution) const="$(cd "$2" && pwd)" || exit 2; shift 2;; *) echo "unknown arg $1" >&2; exit 2;; esac; done
cg="$repo/submodules/constitution/scripts/gates"
[ -f "$cg/cm_machine_evidence_at_every_gate.sh" ] && [ -f "$cg/cm_ledger_row_typed_from_closed_vocabulary.sh" ] \
  || { echo "constitution gates not found under $cg (git submodule update --init submodules/constitution)" >&2; exit 2; }
failed=0; total=0
gate() { # gate <name> <cmd...>
  local n="$1"; shift; total=$((total+1))
  echo "--- $n"; "$@"; local rc=$?
  if [ $rc = 0 ]; then echo "GATE $n PASS"; else echo "GATE $n FAIL (exit $rc)"; failed=$((failed+1)); fi
}
gate gofmt_clean                                bash "$here/gofmt_clean.sh" "$root"
gate build_evidence_manifest_check              bash "$here/build_evidence_manifest.sh" --check --root "$root"
gate cm_machine_evidence_at_every_gate          bash "$cg/cm_machine_evidence_at_every_gate.sh" --root "$root"
gate cm_ledger_row_typed_from_closed_vocabulary bash "$cg/cm_ledger_row_typed_from_closed_vocabulary.sh" --root "$root"
gate cm_pr_delivery_anchor_wired                bash "$cg/cm_pr_delivery_anchor_wired.sh" --root "$const"
gate cm_pr_delivery_account_name_free           bash "$cg/cm_pr_delivery_account_name_free.sh" --root "$const" --consumer-config "$root/config/delivery.conf"
gate pr_delivery_config_valid                   python3 -I -c 'import importlib.util,sys;s=importlib.util.spec_from_file_location("d",sys.argv[1]);m=importlib.util.module_from_spec(s);s.loader.exec_module(m);m.parse_config(sys.argv[2]);print("config ok")' "$repo/scripts/pr/deliver.py" "$root/config/delivery.conf"
echo "=== local gates: $((total-failed))/$total passed"
[ $failed = 0 ]
