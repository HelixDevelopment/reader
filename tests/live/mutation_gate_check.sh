#!/usr/bin/env bash
# mutation_gate_check.sh - HRD-050: paired mutants of the skip gate. Each mutant weakens ONE check in a temp
# copy of tests/live/{run_gated_tests.sh,parse_gotest.awk}; test_run_gated_tests.sh (GATE=<copy>) must then
# FAIL (mutant KILLED). A mutant that survives means that check is not load-bearing. Exit 0 iff all killed.
set -uo pipefail
ROOT=$(cd "$(dirname "$0")/../.." && pwd) || exit 2
W=$(mktemp -d); trap 'rm -rf "$W"' EXIT
# name@file@sed expression (separator @; sed uses | internally)
M=(
"M1-F1-no-unusable-check@run_gated_tests.sh@s/\[ -n \"\$LOGBAD\" \] && return 2$/:/"
"M2-F2-bare-name-key@parse_gotest.awk@s/print bk\\[i\\] \"\\\\t\" p \"\\\\t\" bn\\[i\\]/print bk[i] \"\\\\tX\\\\t\" bn[i]/"
"M3-F3-blank-reason-accepted@run_gated_tests.sh@s/! \[\[ \$rr =~ \[^\[:space:\]\] \]\]/false/"
"M4-F3-unanchored-item@run_gated_tests.sh@s/\^HRD-\[0-9\]+\$ \]\]/HRD-[0-9]+ ]]/"
"M5-F4-ignore-FAIL-tests@run_gated_tests.sh@s/FAIL) echo \"FAILED-TEST: \\\$pk|\\\$n\"; bad=1;;/FAIL) :;;/"
"M6-F4-ignore-build-failed@parse_gotest.awk@s/print \"BUILDFAIL/print \"IGNORED/"
"M7-F5-ignore-timeout@parse_gotest.awk@s/print \"TIMEOUT/print \"IGNORED/"
"M8-F5-ignore-panic@parse_gotest.awk@s/print \"PANIC/print \"IGNORED/"
"M9-F5-ignore-no-results@run_gated_tests.sh@s/NORESULTS) echo \"NO-RESULTS: \\\$pk (package FAIL with no test results)\"; bad=1;;/NORESULTS) :;;/"
"M10-F5-ignore-pkgfail@run_gated_tests.sh@s/PKGFAIL) echo \"PACKAGE-FAILED: \\\$pk\"; bad=1;;/PKGFAIL) :;;/"
"M11-F8-no-unused-report@run_gated_tests.sh@s/echo \"UNUSED-ALLOW/echo \"IGNORED/"
"M12-I1-ignore-incomplete@parse_gotest.awk@s/print \"INCOMPLETE/print \"IGNORED/"
"M13-I4-no-allow-check@run_gated_tests.sh@s/if \[ ! -f \"\$ALLOW\" \] || \[ ! -r \"\$ALLOW\" \]/if false/"
"M14-B1-phase-b-helper-missing@run_gated_tests.sh@s/^phase_b_sets() {/phase_b_setz() {/"
"M15-B2-marker-relative@parse_gotest.awk@s/def = \$3; open = 1/def = \".\/\" \$3; open = 1/"
"M16-m2-no-trim@run_gated_tests.sh@s/^trim() { local v=\$1;.*\$/trim() { printf '%s' \"\$1\"; }/"
"M17-m2-drop-last-line@run_gated_tests.sh@s/ || \[ -n \"\$pk\$n\$r\$i\" \]//"
"M18-B1-regex-excludes-nothing@run_gated_tests.sh@s/EXCL='TestLiveScaleCatalog|/EXCL='TestNeverMatches|/"
"M19-R5-seal-call-removed@run_gated_tests.sh@s/^seal_dir \"\$OUT\" ||.*$/:/"
"M20-R3-markers-removed@run_gated_tests.sh@s/echo \"# PKG \$IMP\" >> \"\$LB\"/:/"
"M21-R2-trap-wrong-code@run_gated_tests.sh@s/\[ \$rc -eq 127 \]/[ \$rc -eq 128 ]/"
"M22-B3-pkg-end-ignored@parse_gotest.awk@s/^\/\^# PKG-END\/ {.*$/\/^# PKG-END\/ { next }/"
"M23-R7-no-python3-preflight@run_gated_tests.sh@s/need python3 go podman/need go podman/"
"M24-R1-untracked-seal-tool@run_gated_tests.sh@s/^seal_dir \"\$OUT\" ||/tools\/isolation\/seal_evidence.sh \"\$OUT\" ||/"
"M25-m1-no-ansi-strip@parse_gotest.awk@s/^{ gsub.*$/{ sub(\/\\r$\/, \"\") }/"
"M26-m3-contradiction-allowed@run_gated_tests.sh@s/different reason\/item\"; bad=1; fi/different reason\/item\"; fi/"
"M27-R6-workers-interleave@run_gated_tests.sh@s/ > \"\$TMPD\/worker\$i.log\" 2>&1 &/ >> \"\$LB\" 2>\&1 \&/"
"M28-R8-no-port-check@run_gated_tests.sh@s/SETUP-ERROR: could not allocate ports/IGNORED/"
"M29-B1-grep-q-in-pipe@run_gated_tests.sh@s/| grep -E '\^\[\[:space:\]\]\*--- (PASS|SKIP|FAIL): ' >\/dev\/null; then/| grep -qE '^[[:space:]]*--- (PASS|SKIP|FAIL): '; then/"
"M30-m1-no-dashdash@run_gated_tests.sh@s/sha256sum -- >/sha256sum >/"
"M31-m3-no-sort@run_gated_tests.sh@s/| LC_ALL=C sort | sed 's|^/| sed 's|^/"
"M32-m4-worker-failure-ignored@run_gated_tests.sh@s/wait \"\$p\" || RC=1; done/wait \"\$p\"; done/"
"M33-m6-end-mismatch-ignored@parse_gotest.awk@s/else if (\$3 != def) print/else if (0) print/"
"M34-m5-open-at-eof-ignored@parse_gotest.awk@s/if (open) print \"INCOMPLETE.t\" def \".tmarker-open/if (0) print \"INCOMPLETE.t\" def \".tmarker-open/"
"M35-m5-pkgline-while-open-ignored@parse_gotest.awk@s/if (open) { print \"INCOMPLETE.t\" def \".tpackage-line/if (0) { print \"INCOMPLETE.t\" def \".tpackage-line/"
"M36-m8-seq-not-needed@run_gated_tests.sh@s/xargs seq cp/xargs cp/"
"M37-X1-panic-not-bad@run_gated_tests.sh@s/\(PANIC).*\)bad=1;;/\1:;;/"
"M38-X2-timeout-not-bad@run_gated_tests.sh@s/\(TIMEOUT).*\)bad=1;;/\1:;;/"
"M39-X4-substring-allow-match@run_gated_tests.sh@s/grep -qxF \"\$rec\"/grep -qF \"\$rec\"/"
"M40-X5-empty-item-digits@run_gated_tests.sh@s/\^HRD-\[0-9\]+\$ \]\]/^HRD-[0-9]*$ ]]/"
"M41-X6-nbsp-not-stripped@run_gated_tests.sh@s/rr=\${r\/\/.*   # NBSP/rr=\$r   # NBSP/"
"M42-X7-space-separated-ok-accepted@parse_gotest.awk@s/\^(ok|FAIL) \*.t\[\^ .t\]\//^(ok|FAIL)[ \\t]+[^ \\t]\//"
"M43-X8-bare-pass-eof-not-incomplete@parse_gotest.awk@s/if (nb > 0) print \"INCOMPLETE/if (0) print \"INCOMPLETE/"
"M44-X9-empty-marker-pair-ok@parse_gotest.awk@s/if (open \&\& nb == 0)/if (0)/"
"M45-R6-report-reverts-tracked-files@run_gated_tests.sh@s/LC_ALL=C sort > \"\$3\"$/LC_ALL=C sort > \"\$3\"; git checkout -- evidence/"
"M46-R6-snapshot-not-nul-separated@run_gated_tests.sh@s/--porcelain -z /--porcelain /"
"M47-R7-bare-fail-in-marker-ignored@parse_gotest.awk@s/if (open) print \"PKGFAIL/if (0) print \"PKGFAIL/"
"M48-R8-hash-anywhere-is-comment@run_gated_tests.sh@s/case \$pk in \\\\#\*) continue;; esac/case \$pk\$n in *\\\\#*) continue;; esac/"
"M49-m5-scratch-in-out-dir@run_gated_tests.sh@s/^SCR=\$(mktemp -d) || exit 2; \[ -n \"\$SCR\" \] || exit 2$/SCR=\$OUT/;s/rm -rf \"\${SCR:?}\"; //"
"M51-F2-default-out-is-sealed-dir@run_gated_tests.sh@s/^if \[ -z \"\$OUT\" \]; then OUT=\$(mktemp -d [^;]*;/if [ -z \"\$OUT\" ]; then OUT=evidence\/campaign-2026-11-23\/gap-fix\/hrd050; OUTDEF=0;/"
"M52-F2-nonempty-out-not-refused@run_gated_tests.sh@s/\&\& \[ \$OUTDEF = 0 \] \&\& \[ -d \"\$OUT\" \]/\&\& false \&\& [ -d \"\$OUT\" ]/"
"M53-F5-empty-package-silently-ignored@run_gated_tests.sh@s/echo \"ALLOWLIST-MALFORMED: empty package field[^;]*; bad=1; continue; fi/continue; fi/"
"M50-R6-report-own-output-not-skipped@run_gated_tests.sh@s/index(p, skip \"\/\") == 1/0/g"
)
# Baseline control: the UNMUTATED copy must pass every case, else a mutant's "kill" could be an artefact of the copy.
mkcopy() { rm -rf "$W/c"; mkdir -p "$W/c/tests/live"; cp "$ROOT"/tests/live/run_gated_tests.sh "$ROOT"/tests/live/parse_gotest.awk "$ROOT"/tests/live/allowed_skips.txt "$W/c/tests/live/"; }
mkcopy
if ! GATE="$W/c/tests/live/run_gated_tests.sh" bash "$ROOT/tests/live/test_run_gated_tests.sh" >"$W/base.txt" 2>&1; then
  echo "BASELINE-FAILED: unmutated copy does not pass all cases; no mutant result is valid"; grep '^FAIL ' "$W/base.txt"; exit 2; fi
echo "baseline: unmutated copy passes ($(tail -1 "$W/base.txt"))"
rc=0
for m in "${M[@]}"; do
  [ -z "${ONLY:-}" ] || case " $ONLY " in *" ${m%%-*} "*) ;; *) continue;; esac
  IFS='@' read -r id file sedx <<<"$m"
  mkcopy
  sed -i "$sedx" "$W/c/tests/live/$file"
  if cmp -s "$W/c/tests/live/$file" "$ROOT/tests/live/$file"; then echo "$id: MUTATION-NOT-APPLIED (sed matched nothing)"; rc=1; continue; fi
  if GATE="$W/c/tests/live/run_gated_tests.sh" bash "$ROOT/tests/live/test_run_gated_tests.sh" >"$W/o.txt" 2>&1; then
    echo "$id: SURVIVED (check not load-bearing)"; rc=1
  else echo "$id: KILLED ($(grep -c '^FAIL ' "$W/o.txt") case(s) failed)"; fi
done
exit $rc
