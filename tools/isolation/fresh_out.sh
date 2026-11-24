#!/usr/bin/env bash
# fresh_out.sh DIR - give one run its own FRESH evidence directory (N5: bindcheck only lets a container write to an unsealed, fresh
# directory under evidence/). An existing non-empty DIR (possibly sealed) is MOVED aside into the scratch archive - never deleted -
# and DIR is recreated empty. DIR must lie below <repo>/evidence/.
set -eu
HERE=$(cd "$(dirname "$0")" && pwd); REPO=$(cd "$HERE/../.." && pwd)
D=$(realpath -m -- "${1:?usage: fresh_out.sh DIR}")
case $D in "$REPO"/evidence/?*) ;; *) echo "fresh_out: $D is not below $REPO/evidence/" >&2; exit 64;; esac
if [ -e "$D" ] && [ -n "$(ls -A -- "$D" 2>/dev/null)" ]; then
  A=${ISO_ARCHIVE:-$HOME/.cache/claude-tmp/evidence-archive}; mkdir -p -- "$A"
  mv -- "$D" "$A/$(basename -- "$D").$(date -u +%Y%m%dT%H%M%SZ)"
fi
mkdir -p -- "$D"
