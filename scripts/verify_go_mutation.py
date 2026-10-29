#!/usr/bin/env python3
"""Prove targeted Go tests reject deliberate contract-breaking mutations.

Each mutant runs in an isolated copy. The original files are hashed before and
after the campaign; a mutant that passes, a failed mutation edit, or any original
tree modification fails the verifier. Logs contain command output only.
"""
from __future__ import annotations

import hashlib
import json
import os
import shutil
import subprocess
from pathlib import Path

ROOT = Path("/mnt/agents/output/helix")
WORK = Path("/mnt/agents/research/go-mutation")
EVIDENCE = ROOT / "evidence" / "go-mutation"
GO = "/tmp/helix-go-toolchain/bin/go"

MUTANTS = [
    {
        "id": "equal-clock-regresses-to-local",
        "module": "core",
        "file": "pkg/sync/sync.go",
        "old": "case Equal, Concurrent:",
        "new": "case Concurrent:",
        "package": "./pkg/sync",
        "test": "TestResolveEqualClocksFurthestAndCommutative",
    },
    {
        "id": "compressed-length-not-removed-before-write",
        "module": "server",
        "file": "internal/httpapi/compression.go",
        "old": 'if w.encoding != "identity" {',
        "new": "if false {",
        "package": "./internal/httpapi",
        "test": "TestCompressionNetworkContentLength",
    },
    {
        "id": "opds-private-destination-policy-bypassed",
        "module": "server",
        "file": "internal/sources/http.go",
        "old": "if h.allowPrivateNetwork {",
        "new": "if true {",
        "package": "./internal/sources",
        "test": "TestOPDSRejectsCrossOriginLoopbackAcquisitionByDefault",
    },
    {
        "id": "if-range-validator-ignored",
        "module": "server",
        "file": "internal/httpapi/router.go",
        "old": '(c.GetHeader("If-Range") == "" || c.GetHeader("If-Range") == etag)',
        "new": "(true)",
        "package": "./internal/httpapi",
        "test": "TestContentIfRangeHonorsETag",
    },
    {
        "id": "active-source-deletion-guard-removed",
        "module": "server",
        "file": "internal/httpapi/router.go",
        "old": "if ok && s.jobs.sourceBusy(id) {",
        "new": "if false {",
        "package": "./internal/httpapi",
        "test": "TestSourceDeletionWaitsForScanAndScanAdmission",
    },
    {
        "id": "redis-watch-never-executed",
        "module": "server",
        "file": "internal/kosync/redis.go",
        "old": '''_, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.Set(ctx, key, payload, 0)
				return nil
			})
			return err''',
        "new": "return tx.Set(ctx, key, payload, 0).Err()",
        "package": "./internal/kosync",
        "test": "TestRedisPutProgressRetriesWatchConflict",
    },
    {
        "id": "cross-origin-redirect-rejection-bypassed",
        "module": "server",
        "file": "internal/sources/http.go",
        "old": "if !allowCrossOrigin {",
        "new": "if false {",
        "package": "./internal/sources",
        "test": "TestHTTPRejectsCrossOriginRedirectBeforeRequest",
    },
    {
        "id": "changed-file-current-bytes-ignored",
        "module": "server",
        "file": "internal/catalog/merge.go",
        "old": "out[i] = f",
        "new": "out[i] = out[i]",
        "package": "./internal/catalog",
        "test": "TestMergeWorkReplacesChangedFileAtSameSourcePath",
    },
    {
        "id": "source-reconciliation-disabled",
        "module": "server",
        "file": "internal/ingest/service.go",
        "old": "removed, err := s.Repository.ReconcileSource(ctx, sourceID, present)",
        "new": "removed, err := 0, error(nil)",
        "package": "./internal/ingest",
        "test": "TestMemoryFullScanReconcilesDeletedSourceFiles",
    },
    {
        "id": "opds-pagination-limit-ignored",
        "module": "server",
        "file": "internal/sources/opds.go",
        "old": "if next != \"\" {",
        "new": "if false {",
        "package": "./internal/sources",
        "test": "TestOPDSRejectsTruncatedPaginationInsteadOfReturningPartialSnapshot",
    },
    {
        "id": "stale-etag-range-gating-removed",
        "module": "server",
        "file": "internal/httpapi/router.go",
        "old": "serveRange := current && sizeKnown && rangeHeader != \"\"",
        "new": "serveRange := sizeKnown && rangeHeader != \"\"",
        "package": "./internal/httpapi",
        "test": "TestContentStaleETagCannotSpliceChangedEqualSizeFile",
    },
    {
        "id": "stable-source-id-ignored",
        "module": "server",
        "file": "internal/httpapi/router.go",
        "old": "id := strings.TrimSpace(cfg.ID)",
        "new": "id := \"\"",
        "package": "./internal/httpapi",
        "test": "TestStableSourceIDReattachesCatalogAfterRestart",
    },
    {
        "id": "embedded-isbn-normalization-discarded",
        "module": "server",
        "file": "internal/identify/pipeline.go",
        "old": "\t\tid.Value = v\n\t\tif best < 0.98 {",
        "new": "\t\t_ = v\n\t\tif best < 0.98 {",
        "package": "./internal/identify",
        "test": "TestValidEmbeddedISBNAccepted",
    },
    {
        "id": "isbn-identity-canonicalization-disabled",
        "module": "server",
        "file": "internal/ingest/identity.go",
        "old": "value, ok := isbn.Canonical(id.Value)",
        "new": "value, ok := id.Value, true",
        "package": "./internal/ingest",
        "test": "TestISBNIdentityCanonicalizesSchemePairAndOrder",
    },
    {
        "id": "opds-query-collision-disambiguation-removed",
        "module": "server",
        "file": "internal/sources/opds.go",
        "old": "key += \"#helix-x-\" + hex.EncodeToString(digest[:])",
        "new": "key += \"\"",
        "package": "./internal/sources",
        "test": "TestOPDSQueryAddressedAcquisitionsKeepDistinctPublicIdentities",
    },
    {
        "id": "memory-file-validation-removed",
        "module": "server",
        "file": "internal/catalog/memory.go",
        "old": "func (m *Memory) MergeWork(_ context.Context, w Work) error {\n\tif err := validateWork(w); err != nil {",
        "new": "func (m *Memory) MergeWork(_ context.Context, w Work) error {\n\tif err := error(nil); err != nil {",
        "package": "./internal/catalog",
        "test": "TestMemoryRejectsIncompleteFileIdentityLikePostgres",
    },
    {
        "id": "opds-stable-collision-grouping-removed",
        "module": "server",
        "file": "internal/sources/opds.go",
        "old": "if len(groups[candidate.base]) > 1 {",
        "new": "if false {",
        "package": "./internal/sources",
        "test": "TestOPDSCollisionKeysSurviveReorderAndSiblingRemoval",
    },
    {
        "id": "local-delete-not-found-translation-removed",
        "module": "server",
        "file": "internal/sources/local.go",
        "old": "resolved, err := filepath.EvalSymlinks(p)\n\tif err != nil {\n\t\tif os.IsNotExist(err) {\n\t\t\treturn \"\", ErrNotFound",
        "new": "resolved, err := filepath.EvalSymlinks(p)\n\tif err != nil {\n\t\tif os.IsNotExist(err) {\n\t\t\treturn \"\", err",
        "package": "./internal/httpapi",
        "test": "TestContentDeletedLocalFileReturns404",
    },
    {
        "id": "opds-previous-generation-keys-ignored",
        "module": "server",
        "file": "internal/sources/opds.go",
        "old": "if validPrevious {",
        "new": "if false {",
        "package": "./internal/sources",
        "test": "TestOPDSCollisionKeysSurviveReorderAndSiblingRemoval",
    },
    {
        "id": "content-single-handle-validation-bypassed",
        "module": "server",
        "file": "internal/httpapi/router.go",
        "old": "if !head {",
        "new": "if false {",
        "package": "./internal/httpapi",
        "test": "TestContentGETValidatesAndReadsSameOpenedHandle",
    },
    {
        "id": "identity-response-closes-unused-compressor",
        "module": "server",
        "file": "internal/httpapi/compression.go",
        "old": "w.closer = nil",
        "new": "w.closer = w.closer",
        "package": "./internal/httpapi",
        "test": "TestCompressionIdentityFallbackPreservesKeepAlive",
    },
]


def digest(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main() -> int:
    EVIDENCE.mkdir(parents=True, exist_ok=True)
    if WORK.exists():
        shutil.rmtree(WORK)
    WORK.mkdir(parents=True)
    originals = {(m["module"], m["file"]): digest(ROOT / m["module"] / m["file"]) for m in MUTANTS}
    results = []
    for mutant in MUTANTS:
        case = WORK / mutant["id"]
        module_copy = case / mutant["module"]
        shutil.copytree(ROOT / mutant["module"], module_copy)
        target = module_copy / mutant["file"]
        text = target.read_text()
        occurrences = text.count(mutant["old"])
        if occurrences != 1:
            results.append({"id": mutant["id"], "status": "FAIL", "reason": f"mutation anchor count {occurrences}"})
            continue
        target.write_text(text.replace(mutant["old"], mutant["new"], 1))
        command = [GO, "test", mutant["package"], "-run", "^" + mutant["test"] + "$", "-count=1"]
        env = os.environ.copy()
        env["GOPROXY"] = "off"
        completed = subprocess.run(command, cwd=module_copy, env=env, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=120)
        log = "$ " + " ".join(command) + "\n" + completed.stdout + f"\nexit={completed.returncode}\n"
        (EVIDENCE / (mutant["id"] + ".log")).write_text(log)
        killed = completed.returncode != 0
        results.append({
            "id": mutant["id"],
            "status": "PASS" if killed else "FAIL",
            "expected": "targeted test rejects mutant",
            "test": mutant["test"],
            "exit": completed.returncode,
            "log": mutant["id"] + ".log",
        })
    unchanged = all(digest(ROOT / module / name) == before for (module, name), before in originals.items())
    if not unchanged:
        results.append({"id": "original-tree-unchanged", "status": "FAIL"})
    status = "PASS" if len(results) == len(MUTANTS) and all(r["status"] == "PASS" for r in results) and unchanged else "FAIL"
    summary = {
        "status": status,
        "pass": sum(r["status"] == "PASS" for r in results),
        "fail": sum(r["status"] == "FAIL" for r in results),
        "original_tree_unchanged": unchanged,
        "results": results,
    }
    (EVIDENCE / "results.json").write_text(json.dumps(summary, indent=2) + "\n")
    shutil.rmtree(WORK)
    print(json.dumps({"status": status, "pass": summary["pass"], "fail": summary["fail"]}))
    return 0 if status == "PASS" else 1


if __name__ == "__main__":
    raise SystemExit(main())
