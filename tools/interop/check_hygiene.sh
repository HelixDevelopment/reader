#!/usr/bin/env bash
# check_hygiene.sh - static 11.4.287(C) hygiene check for the interop scripts, with a control needle.
# usage: check_hygiene.sh            (checks tools/interop/hrd026_*.sh next to this file)
# FAIL patterns: HOME reassignment, bare rm -rf of a variable, positional arg taken without ${N:?}, ${N:-} defaults.
# The needle is a scratch file holding all four bad shapes; the check MUST flag it, else the instrument is blind.
set -u
here=$(cd "$(dirname "$0")" && pwd)
bad() { # file -> prints offending lines, returns 0 when any found
  local f=$1 n=0
  grep -nE '(^|[[:space:];])(export )?HOME=' "$f" && n=1
  grep -nE 'rm -rf[^;|&]*"?\$\{?[A-Za-z_]+\}?"?' "$f" | grep -vE '\$\{[A-Za-z_]+:\?' && n=1
  grep -nE '=\$[1-9]($|[[:space:];])' "$f" && n=1
  grep -nE '=\$\{[1-9]:-' "$f" && n=1
  [ $n -eq 1 ]
}
needle=$(mktemp) || exit 2
trap 'rm -f -- "${needle:?}"' EXIT
printf '%s\n' 'W=$(mktemp -d)' 'export HOME=$W/home' 'rm -rf "$W"' 'ST=$3' 'BIN=${2:-}' > "$needle"
if ! bad "$needle" >/dev/null; then echo "CONTROL NEEDLE NOT DETECTED: instrument is blind" >&2; exit 3; fi
echo "control needle: detected (instrument can see)"
rc=0
for f in "$here"/hrd026_*.sh; do
  if bad "$f"; then echo "FAIL $f"; rc=1; else echo "PASS $f"; fi
done
exit $rc
