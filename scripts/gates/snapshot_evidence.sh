#!/usr/bin/env bash
# snapshot_evidence.sh - freeze a still-changing evidence file for ledger citation (HRD-051 review fix 1).
# Rule: "ledger rows cite only frozen evidence" (enforced by build_evidence_manifest.sh --check).
# Copies <src> (path relative to repo root) into evidence/campaign-2026-11-23/gap-fix/ledger-snapshots/
# as <path with / -> __> and records source path, capture time (source mtime, UTC), snapshot time and
# sha256 in that directory's PROVENANCE.tsv. Re-snapshotting an already-recorded name is refused
# (snapshots are immutable; use a new name or remove the row deliberately).
# Usage: scripts/gates/snapshot_evidence.sh <relative-src> [...]
set -eu
root="$(cd "$(dirname "$0")/../.." && pwd)"; d="$root/evidence/campaign-2026-11-23/gap-fix/ledger-snapshots"
mkdir -p "$d"; p="$d/PROVENANCE.tsv"
[ -f "$p" ] || printf 'snapshot\tsource_path\tcaptured_at\tsnapshotted_at\tsha256\n' > "$p"
for src in "$@"; do
  [ -f "$root/$src" ] || { echo "no such file: $src" >&2; exit 1; }
  n="$(printf '%s' "$src" | sed 's#/#__#g')"
  if awk -F'\t' -v n="$n" 'NR>1&&$1==n{f=1} END{exit !f}' "$p"; then echo "already snapshotted: $n" >&2; exit 1; fi
  cp -p "$root/$src" "$d/$n"
  printf '%s\t%s\t%s\t%s\t%s\n' "$n" "$src" "$(date -u -d "@$(stat -c %Y "$root/$src")" +%FT%TZ)" "$(date -u +%FT%TZ)" "$(sha256sum "$d/$n" | cut -d' ' -f1)" >> "$p"
  echo "snapshot $n"
done
