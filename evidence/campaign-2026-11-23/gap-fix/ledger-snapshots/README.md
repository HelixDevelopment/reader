# Ledger snapshots

Rule: **ledger rows cite only frozen evidence** (docs/claim_reality_ledger.tsv, enforced by
`scripts/gates/build_evidence_manifest.sh --check`). Frozen means either a file git tracks and has not
modified, or a file in this directory whose sha256 matches its row in `PROVENANCE.tsv`.
Still-changing trees (`evidence/clients/**`, `clients/*.json`, untracked campaign output) are rewritten by
other work; citing them made the gate flip with EVIDENCE_HASH_MISMATCH. Snapshots are made with
`scripts/gates/snapshot_evidence.sh <path>`; each records source path, capture time (source mtime),
snapshot time and sha256. A snapshot is a point-in-time copy: the live source may have moved on.
