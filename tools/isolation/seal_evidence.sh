#!/usr/bin/env bash
# seal_evidence.sh <dir> - write <dir>/SHA256SUMS atomically (temp file + rename) over every file except itself.
#   Review round 4 (R4-B1): the directory must pass the ALLOW-LIST scan (`safe_io.py scan --allow --host-seal`): regular files only, link count 1, safe
#   names (no leading dot/dash, no escape bytes or newlines), publishable extensions only (png jpg json jsonl log txt csv mp4 webm gif webp), no dot-dir,
#   no git-dir-shaped directory, no planted checksum manifest, volume caps. Anything else = exit 3 and NOTHING is sealed (no temp file, no SHA256SUMS).
#   --host-authored <dir>: for a directory only the host ever wrote (docs, scripts, reports): the kinds-and-deny-table scan (default) instead of the allow-list.
#   No runner uses it; never use it on a directory a container could write.
set -eu
HERE=$(cd "$(dirname "$0")" && pwd)
MODE=--allow
if [ "${1:-}" = "--host-authored" ]; then MODE=; shift; fi
D=${1:?usage: seal_evidence.sh [--host-authored] <dir>}; [ -d "$D" ] || { echo "no such dir $D" >&2; exit 2; }
# a container-writable directory may hold links / special files / hard links / control artifacts: never hash (= follow) them, never run a host tool in them
if [ -n "$MODE" ]; then SCAN=(scan "$D" --allow --host-seal); else SCAN=(scan "$D"); fi
python3 -I "$HERE/safe_io.py" "${SCAN[@]}" || { echo "seal_evidence: $D holds an entry outside the publish allow-list (link, special file, hard link, control artifact, unsafe name or type): refusing to seal" >&2; exit 3; }
T=$(mktemp "$D/.SHA256SUMS.XXXXXX")
( cd "$D" && find . -type f ! -path ./SHA256SUMS ! -path "./${T##*/}" -print0 | sort -z | xargs -0 sha256sum ) > "$T"
chmod 0644 "$T"; mv -f "$T" "$D/SHA256SUMS"; echo "sealed $(wc -l < "$D/SHA256SUMS") files in $D"
