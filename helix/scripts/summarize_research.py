#!/usr/bin/env python3
"""Rebuild the research coverage summary from preserved, revision-pinned analyses.

A valid current checkout is recorded separately from preserved static evidence. This
prevents a lost or still-running clone from either erasing prior evidence or being
misreported as currently available. Keyword indexing and manual code review remain
distinct states.
"""
from __future__ import annotations

import argparse
import gzip
import json
import re
import subprocess
from datetime import datetime, timezone
from pathlib import Path

BACKEND_ALIASES = {
    "stumpapp-stump": "stump",
    "advplyr-audiobookshelf": "audiobookshelf",
    "bookorbit-bookorbit": "bookorbit",
    "grimmory-tools-grimmory": "grimmory",
    "kareadita-kavita": "kavita",
    "seblucas-cops": "cops",
}


def slug_for(kind: str, record: dict) -> str:
    if kind == "client":
        return record["slug"]
    return re.sub(r"[^a-z0-9]+", "-", record["owner_repo"].lower()).strip("-")


def checkout_revision(path: Path) -> str | None:
    result = subprocess.run(
        ["git", "-C", str(path), "rev-parse", "HEAD"],
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.DEVNULL,
        check=False,
    )
    revision = result.stdout.strip()
    if result.returncode or not re.fullmatch(r"[0-9a-f]{40}", revision):
        return None
    return revision


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", default="/mnt/agents/output/helix/research")
    parser.add_argument("--clones", default="/mnt/agents/research/git-repos")
    args = parser.parse_args()
    out = Path(args.output)
    clones = Path(args.clones)

    repositories = []
    unavailable = []
    counts = {
        "expected": 0,
        "analysis_present": 0,
        "analysis_missing": 0,
        "file_ledger_present": 0,
        "current_checkout_present": 0,
        "current_checkout_matches_analysis": 0,
    }
    for kind, inventory_name in (("client", "repository_inventory.json"), ("backend", "backend_inventory.json")):
        inventory = json.loads((out / inventory_name).read_text())
        for record in inventory["repositories"]:
            counts["expected"] += 1
            slug = slug_for(kind, record)
            url = record.get("canonical") or record["url"]
            analysis_path = out / f"{kind}_analysis" / f"{slug}.json"
            item = {"kind": kind, "slug": slug, "url": url}
            reasons = []
            analysis = None
            if analysis_path.exists():
                analysis = json.loads(analysis_path.read_text())
                if analysis.get("kind") != kind or analysis.get("slug") != slug:
                    raise ValueError(f"analysis identity mismatch: {analysis_path}")
                for field in ("revision", "tree"):
                    if not re.fullmatch(r"[0-9a-f]{40}", analysis.get(field, "")):
                        raise ValueError(f"invalid {field} in {analysis_path}")
                counts["analysis_present"] += 1
                item.update({
                    "analysis_status": analysis["analysis_status"],
                    "manual_review_status": analysis["manual_review_status"],
                    "support_claims_allowed": analysis["support_claims_allowed"],
                    "revision": analysis["revision"],
                    "tree": analysis["tree"],
                    "tracked_files": analysis["files"]["tracked"],
                    "analysis_file": str(analysis_path.relative_to(out)),
                })
            else:
                counts["analysis_missing"] += 1
                item.update({"analysis_status": "unavailable", "manual_review_status": "pending", "support_claims_allowed": False})
                reasons.append("static_analysis_unavailable")

            if analysis is not None:
                ledger = out / analysis["file_evidence_gzip"]
                item["file_evidence_gzip"] = analysis["file_evidence_gzip"]
                if ledger.exists():
                    with gzip.open(ledger, "rt", encoding="utf-8") as stream:
                        ledger_records = sum(1 for _ in stream)
                    if ledger_records != analysis["files"]["tracked"]:
                        raise ValueError(f"ledger count mismatch: {ledger}")
                    counts["file_ledger_present"] += 1
                    item["file_ledger_records"] = ledger_records
                else:
                    reasons.append("file_evidence_ledger_unavailable")

            checkout = clones / kind / slug
            if kind == "backend" and slug in BACKEND_ALIASES:
                checkout = clones / "client" / BACKEND_ALIASES[slug]
            revision = checkout_revision(checkout)
            item["current_checkout_status"] = "valid" if revision else "unavailable"
            if revision:
                counts["current_checkout_present"] += 1
                item["current_checkout_revision"] = revision
                matches = analysis is not None and revision == analysis["revision"]
                item["current_checkout_matches_analysis"] = matches
                if matches:
                    counts["current_checkout_matches_analysis"] += 1
            if reasons:
                unavailable.append({"kind": kind, "slug": slug, "url": url, "reasons": reasons})
            repositories.append(item)

    repositories.sort(key=lambda item: (item["kind"], item["slug"]))
    unavailable.sort(key=lambda item: (item["kind"], item["slug"]))
    complete = counts["analysis_present"] == counts["expected"] and counts["file_ledger_present"] == counts["expected"]
    summary = {
        "schema_version": 2,
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "status": "automated_index_complete_manual_review_pending" if complete else "partial",
        "manual_review_status": "pending",
        "keyword_evidence_is_not_support_proof": True,
        "current_checkout_availability_is_not_analysis_coverage": True,
        "coverage": counts,
        "repositories": repositories,
        "unavailable": unavailable,
    }
    (out / "analysis_summary.json").write_text(json.dumps(summary, indent=2, sort_keys=True) + "\n")
    print(json.dumps({"status": summary["status"], "coverage": counts, "unavailable": len(unavailable)}, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
