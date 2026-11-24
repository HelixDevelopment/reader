#!/usr/bin/env bash
# gofmt_clean.sh - standing gate (HRD-055): `gofmt -l` over core/ and server/
# must list nothing. Exit 0 = clean, 1 = drift (files listed), 2 = tooling error.
# Usage: scripts/gates/gofmt_clean.sh [root]      (default: repo root)
#        scripts/gates/gofmt_clean.sh --self-test (mutation: misformat in scratch copy must FAIL)
set -u
here="$(cd "$(dirname "$0")" && pwd)"
if [ "${1:-}" = "--self-test" ]; then
  tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
  mkdir -p "$tmp/core" "$tmp/server"
  printf 'package x\n\nfunc F() int { return 1 }\n' > "$tmp/core/ok.go"
  "$0" "$tmp" >/dev/null || { echo "SELF-TEST FAIL: clean tree rejected"; exit 1; }
  printf 'package x\n\ntype T struct {\n\tA  int\n\tBBB int\n}\n' > "$tmp/server/bad.go"   # misaligned
  if "$0" "$tmp" >/dev/null; then echo "SELF-TEST FAIL: mutation not detected"; exit 1; fi
  echo "SELF-TEST PASS: clean accepted, misformat mutation rejected"; exit 0
fi
root="${1:-$(cd "$here/../.." && pwd)}"
command -v gofmt >/dev/null || { echo "gofmt not found" >&2; exit 2; }
out=""
for d in core server; do
  [ -d "$root/$d" ] || continue
  o="$(cd "$root/$d" && gofmt -l . 2>&1)" || true
  [ -n "$o" ] && out+="$(printf '%s\n' "$o" | sed "s#^#$d/#")"$'\n'
done
if [ -n "$out" ]; then printf 'gofmt drift:\n%s' "$out"; exit 1; fi
echo "gofmt clean (core, server)"; exit 0
