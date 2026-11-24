#!/usr/bin/env bash
# install_hooks.sh - installs a LOCAL git pre-commit hook that runs scripts/gates/run_local_gates.sh.
# The operator decides whether to install; nothing runs this automatically. No CI/CD (§11.4.156).
# Usage: scripts/gates/install_hooks.sh                       dry run: prints what would be installed
#        scripts/gates/install_hooks.sh --hooks-dir DIR       write DIR/pre-commit (refuses to overwrite)
#        scripts/gates/install_hooks.sh --hooks-dir "$(git rev-parse --git-path hooks)"   real install
# Uninstall: delete the pre-commit file. Bypass once: git commit --no-verify (the operator's call).
set -eu
here="$(cd "$(dirname "$0")" && pwd)"; hd=""
while [ $# -gt 0 ]; do case "$1" in --hooks-dir) [ $# -ge 2 ] && [ -n "$2" ] || { echo "usage: install_hooks.sh [--hooks-dir DIR]  (--hooks-dir needs a value)" >&2; exit 2; }; hd="$2"; shift 2;; *) echo "usage: install_hooks.sh [--hooks-dir DIR]; unknown arg $1" >&2; exit 2;; esac; done
body="#!/usr/bin/env bash
# installed by scripts/gates/install_hooks.sh - local gate runner (no CI/CD, §11.4.156)
exec \"$here/run_local_gates.sh\"
"
if [ -z "$hd" ]; then echo "DRY RUN - would write pre-commit:"; printf '%s' "$body"; echo "re-run with --hooks-dir DIR to install"; exit 0; fi
[ -d "$hd" ] || { echo "no such hooks dir: $hd" >&2; exit 2; }
{ [ ! -e "$hd/pre-commit" ] && [ ! -L "$hd/pre-commit" ]; } || { echo "refusing to overwrite existing $hd/pre-commit" >&2; exit 1; }
printf '%s' "$body" > "$hd/pre-commit"; chmod +x "$hd/pre-commit"; echo "installed $hd/pre-commit"
