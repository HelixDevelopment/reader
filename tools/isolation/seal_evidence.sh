#!/usr/bin/env bash
# seal_evidence.sh <dir> - write <dir>/SHA256SUMS atomically (temp file + rename) over every file except itself.
set -eu
D=$1; [ -d "$D" ] || { echo "no such dir $D" >&2; exit 2; }
T=$(mktemp "$D/.SHA256SUMS.XXXXXX")
( cd "$D" && find . -type f ! -name SHA256SUMS ! -name '.SHA256SUMS.*' -print0 | sort -z | xargs -0 sha256sum ) > "$T"
chmod 0644 "$T"; mv -f "$T" "$D/SHA256SUMS"; echo "sealed $(wc -l < "$D/SHA256SUMS") files in $D"
