#!/usr/bin/env python3
"""Prove targeted Go tests reject deliberate contract-breaking mutations.

Each mutant runs in an isolated copy. The original files are hashed before and
after the campaign; a mutant that passes, a failed mutation edit, or any original
tree modification fails the verifier. Logs contain command output only.

A mutant that fails for an unrelated reason proves nothing about the assertion it
targets, so an entry may carry "expect": a substring the test output must contain
for the mutant to count as rejected.
"""
from __future__ import annotations

import hashlib
import json
import os
import shutil
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
WORK = ROOT / ".helix-verify" / "go-mutation"
EVIDENCE = ROOT / "evidence" / "go-mutation"
GO = os.environ.get("HELIX_GO", "go")
# Mutants are edited into a copy that mirrors the repo layout, because a module
# may reach its sibling through a relative replace directive. Copying only the
# target module leaves that replace dangling, the build fails, and a failed
# build would otherwise be counted as a rejected mutant.
GO_MODULES = tuple(sorted(p.parent.name for p in ROOT.glob("*/go.mod")))
# Only the Go toolchain prints these; a test asserting its own error text
# cannot produce them, so their presence means the mutant never ran.
BUILD_FAILURES = (
    "[build failed]",
    "[setup failed]",
    "replacement directory",
    "no required module provides package",
    "cannot find module",
    "missing go.sum entry",
)

MUTANTS = [
    {
        "id": "equal-clock-regresses-to-local",
        "module": "core",
        "file": "pkg/sync/sync.go",
        "old": "case Equal, Concurrent:",
        "new": "case Concurrent:",
        "package": "./pkg/sync",
        "test": "TestResolveEqualClocksFurthestAndCommutative",
        "expect": "furthest equal-clock position must win",
    },
    {
        "id": "compressed-length-not-removed-before-write",
        "module": "server",
        "file": "internal/httpapi/compression.go",
        "old": 'if w.encoding != "identity" {',
        "new": "if false {",
        "package": "./internal/httpapi",
        "test": "TestCompressionNetworkContentLength",
        "expect": "br truncated network response",
    },
    {
        "id": "opds-private-destination-policy-bypassed",
        "module": "server",
        "file": "internal/sources/http.go",
        "old": "if h.allowPrivateNetwork {",
        "new": "if true {",
        "package": "./internal/sources",
        "test": "TestOPDSRejectsCrossOriginLoopbackAcquisitionByDefault",
        "expect": "cross-origin loopback acquisition must be rejected",
    },
    {
        "id": "if-range-validator-ignored",
        "module": "server",
        "file": "internal/httpapi/router.go",
        "old": '(c.GetHeader("If-Range") == "" || c.GetHeader("If-Range") == etag)',
        "new": "(true)",
        "package": "./internal/httpapi",
        "test": "TestContentIfRangeHonorsETag",
        "expect": "must force full 200",
    },
    {
        "id": "active-source-deletion-guard-removed",
        "module": "server",
        "file": "internal/httpapi/router.go",
        "old": "if ok && s.jobs.sourceBusy(id) {",
        "new": "if false {",
        "package": "./internal/httpapi",
        "test": "TestSourceDeletionWaitsForScanAndScanAdmission",
        "expect": "delete during active scan",
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
        "expect": "stale lower-percentage update rewound progress",
    },
    {
        "id": "cross-origin-redirect-rejection-bypassed",
        "module": "server",
        "file": "internal/sources/http.go",
        "old": "if !allowCrossOrigin {",
        "new": "if false {",
        "package": "./internal/sources",
        "test": "TestHTTPRejectsCrossOriginRedirectBeforeRequest",
        "expect": "before rejection",
    },
    {
        "id": "changed-file-current-bytes-ignored",
        "module": "server",
        "file": "internal/catalog/merge.go",
        "old": "out[i] = f",
        "new": "out[i] = out[i]",
        "package": "./internal/catalog",
        "test": "TestMergeWorkReplacesChangedFileAtSameSourcePath",
        "expect": "changed file did not replace stale byte evidence",
    },
    {
        "id": "source-reconciliation-disabled",
        "module": "server",
        "file": "internal/ingest/service.go",
        "old": "removed, err := s.Repository.ReconcileSource(ctx, sourceID, present)",
        "new": "removed, err := 0, error(nil)",
        "package": "./internal/ingest",
        "test": "TestMemoryFullScanReconcilesDeletedSourceFiles",
        "expect": "deletion rescan: report={Works:1 Files:1 Removed:0",
    },
    {
        "id": "opds-pagination-limit-ignored",
        "module": "server",
        "file": "internal/sources/opds.go",
        "old": "if next != \"\" {",
        "new": "if false {",
        "package": "./internal/sources",
        "test": "TestOPDSRejectsTruncatedPaginationInsteadOfReturningPartialSnapshot",
        "expect": "truncated OPDS listing must fail closed",
    },
    {
        "id": "stale-etag-range-gating-removed",
        "module": "server",
        "file": "internal/httpapi/router.go",
        "old": "serveRange := current && sizeKnown && rangeHeader != \"\"",
        "new": "serveRange := sizeKnown && rangeHeader != \"\"",
        "package": "./internal/httpapi",
        "test": "TestContentStaleETagCannotSpliceChangedEqualSizeFile",
        "expect": "stale validator produced unsafe response",
    },
    {
        "id": "stable-source-id-ignored",
        "module": "server",
        "file": "internal/httpapi/router.go",
        "old": "id := strings.TrimSpace(cfg.ID)",
        "new": "id := \"\"",
        "package": "./internal/httpapi",
        "test": "TestStableSourceIDReattachesCatalogAfterRestart",
        "expect": "stable source creation",
    },
    {
        "id": "embedded-isbn-normalization-discarded",
        "module": "server",
        "file": "internal/identify/pipeline.go",
        "old": "\t\tid.Value = v\n\t\tif best < 0.98 {",
        "new": "\t\t_ = v\n\t\tif best < 0.98 {",
        "package": "./internal/identify",
        "test": "TestValidEmbeddedISBNAccepted",
        "expect": "embedded ISBN normalization was discarded",
    },
    {
        "id": "isbn-identity-canonicalization-disabled",
        "module": "server",
        "file": "internal/ingest/identity.go",
        "old": "value, ok := isbn.Canonical(id.Value)",
        "new": "value, ok := id.Value, isbn.Valid10(id.Value) || isbn.Valid13(id.Value)",
        "package": "./internal/ingest",
        "test": "TestISBNIdentityCanonicalizesSchemePairAndOrder",
        "expect": "ISBN-10 and equivalent ISBN-13 forked identity",
    },
    {
        "id": "opds-query-collision-disambiguation-removed",
        "module": "server",
        "file": "internal/sources/opds.go",
        "old": "key += \"#helix-x-\" + hex.EncodeToString(digest[:])",
        "new": "key += \"#helix-x-\" + hex.EncodeToString(digest[:0])",
        "package": "./internal/sources",
        "test": "TestOPDSQueryAddressedAcquisitionsKeepDistinctPublicIdentities",
        "expect": "OPDS acquisition identity collision",
    },
    {
        "id": "memory-file-validation-removed",
        "module": "server",
        "file": "internal/catalog/memory.go",
        "old": "func (m *Memory) MergeWork(_ context.Context, w Work) error {\n\tif err := validateWork(w); err != nil {",
        "new": "func (m *Memory) MergeWork(_ context.Context, w Work) error {\n\tif err := error(nil); err != nil {",
        "package": "./internal/catalog",
        "test": "TestMemoryRejectsIncompleteFileIdentityLikePostgres",
        "expect": "memory repository accepted an empty file ID",
    },
    {
        "id": "opds-stable-collision-grouping-removed",
        "module": "server",
        "file": "internal/sources/opds.go",
        "old": "if len(groups[candidate.base]) > 1 {",
        "new": "if false {",
        "package": "./internal/sources",
        "test": "TestOPDSCollisionKeysSurviveReorderAndSiblingRemoval",
        "expect": "OPDS acquisition identity collision",
    },
    {
        "id": "local-delete-not-found-translation-removed",
        "module": "server",
        "file": "internal/sources/local.go",
        "old": "resolved, err := filepath.EvalSymlinks(p)\n\tif err != nil {\n\t\tif os.IsNotExist(err) {\n\t\t\treturn \"\", ErrNotFound",
        "new": "resolved, err := filepath.EvalSymlinks(p)\n\tif err != nil {\n\t\tif os.IsNotExist(err) {\n\t\t\treturn \"\", err",
        "package": "./internal/httpapi",
        "test": "TestContentDeletedLocalFileReturns404",
        "expect": "deleted local file returned 502, want 404",
    },
    {
        "id": "opds-previous-generation-keys-ignored",
        "module": "server",
        "file": "internal/sources/opds.go",
        "old": "if validPrevious {",
        "new": "if validPrevious && false {",
        "package": "./internal/sources",
        "test": "TestOPDSCollisionKeysSurviveReorderAndSiblingRemoval",
        "expect": "colliding member lacks stable synthetic key",
    },
    {
        "id": "content-single-handle-validation-bypassed",
        "module": "server",
        "file": "internal/httpapi/router.go",
        "old": "if !head {",
        "new": "if false {",
        "package": "./internal/httpapi",
        "test": "TestContentGETValidatesAndReadsSameOpenedHandle",
        "expect": "source does not support range reads",
    },
    {
        "id": "identity-response-closes-unused-compressor",
        "module": "server",
        "file": "internal/httpapi/compression.go",
        "old": "w.closer = nil",
        "new": "w.closer = w.closer",
        "package": "./internal/httpapi",
        "test": "TestCompressionIdentityFallbackPreservesKeepAlive",
        "expect": "read /data after identity fallback",
    },
    {
        "id": "lcp-cbc-minimum-size-loosened",
        "module": "core",
        "file": "pkg/lcp/keys.go",
        "old": "len(ciphertext) < 2*aes.BlockSize",
        "new": "len(ciphertext) < aes.BlockSize",
        "package": "./pkg/lcp",
        "test": "TestDecryptCBCRejectsMalformedCiphertext",
        "expect": "index out of range [-1]",
    },
    {
        "id": "lcp-cbc-minimum-size-over-strict",
        "module": "core",
        "file": "pkg/lcp/keys.go",
        "old": "len(ciphertext) < 2*aes.BlockSize",
        "new": "len(ciphertext) <= 2*aes.BlockSize",
        "package": "./pkg/lcp",
        "test": "TestEncryptDecryptRoundTripEveryLength",
        "expect": "n=0 decrypt",
    },
    {
        "id": "lcp-cbc-block-multiple-guard-removed",
        "module": "core",
        "file": "pkg/lcp/keys.go",
        "old": "|| len(ciphertext)%aes.BlockSize != 0",
        "new": "|| false",
        "package": "./pkg/lcp",
        "test": "TestDecryptCBCRejectsMalformedCiphertext",
        "expect": "input not full blocks",
    },
    {
        "id": "lcp-pad-length-zero-accepted",
        "module": "core",
        "file": "pkg/lcp/keys.go",
        "old": "if n < 1 || n > aes.BlockSize",
        "new": "if n > aes.BlockSize",
        "package": "./pkg/lcp",
        "test": "TestDecryptCBCRejectsMalformedCiphertext",
        "expect": "pad length 0: error = <nil>",
    },
    {
        "id": "lcp-pad-length-bound-widened",
        "module": "core",
        "file": "pkg/lcp/keys.go",
        "old": "if n < 1 || n > aes.BlockSize",
        "new": "if n < 1 || n > 2*aes.BlockSize",
        "package": "./pkg/lcp",
        "test": "TestDecryptCBCRejectsMalformedCiphertext",
        "expect": "pad length 25 over two blocks: error = <nil>",
    },
    {
        "id": "lcp-user-key-not-raw-utf8",
        "module": "core",
        "file": "pkg/lcp/keys.go",
        "old": "sum := sha256.Sum256([]byte(passphrase))",
        "new": "sum := sha256.Sum256([]byte(fmt.Sprintf(\"%q\", passphrase)))",
        "package": "./pkg/lcp",
        "test": "TestUserKeyIsRawSHA256OfPassphrase",
        "expect": "UserKey(",
    },
    {
        "id": "lcp-key-check-does-not-bind-license-id",
        "module": "core",
        "file": "pkg/lcp/keys.go",
        "old": "if string(got) != l.ID {",
        "new": "if string(got) != string(got) {",
        "package": "./pkg/lcp",
        "test": "TestKeyCheckDetectsTamperedLicenseID",
        "expect": "tampered id error = <nil>",
    },
    {
        "id": "lcp-content-key-length-unchecked",
        "module": "core",
        "file": "pkg/lcp/keys.go",
        "old": "if len(key) != ContentKeyBytes {",
        "new": "if false {",
        "package": "./pkg/lcp",
        "test": "TestContentKeyRejectsWrongLength",
        "expect": "ContentKey error = <nil>",
    },
    {
        "id": "lcp-duplicate-member-guard-removed",
        "module": "core",
        "file": "pkg/lcp/canonical.go",
        "old": "if seen[key] {",
        "new": "if false {",
        "package": "./pkg/lcp",
        "test": "TestCanonicalRejectsDuplicateMembers",
        "expect": "want ErrAmbiguousLicense",
    },
    {
        "id": "lcp-duplicate-scan-skipped-before-signing",
        "module": "core",
        "file": "pkg/lcp/canonical.go",
        "old": "if err := noDuplicateMembers(doc); err != nil {",
        "new": "if err := noDuplicateMembers(doc); false && err != nil {",
        "package": "./pkg/lcp",
        "test": "TestCanonicalRejectsDuplicateMembers",
        "expect": "CanonicalForSigning error = <nil>",
    },
    {
        "id": "lcp-member-sorting-removed",
        "module": "core",
        "file": "pkg/lcp/canonical.go",
        "old": "sort.Strings(keys) // UTF-8 byte order is code point order",
        "new": "sort.SliceStable(keys, func(i, j int) bool { return false })",
        "package": "./pkg/lcp",
        "test": "TestCanonicalIsDeterministicAcrossMapOrder",
        "expect": "depends on map iteration order",
    },
    {
        "id": "lcp-array-order-not-preserved",
        "module": "core",
        "file": "pkg/lcp/canonical.go",
        "old": "case []any:\n\t\tb.WriteByte('[')",
        "new": "case []any:\n\t\tsort.SliceStable(t, func(i, j int) bool { return fmt.Sprint(t[i]) < fmt.Sprint(t[j]) })\n\t\tb.WriteByte('[')",
        "package": "./pkg/lcp",
        "test": "TestCanonicalSortingAndArrayOrder",
        "expect": "Canonical({\"k\":[3,{\"x\":1},2]})",
    },
    {
        "id": "lcp-reference-hex-case-drift",
        "module": "core",
        "file": "pkg/lcp/canonical.go",
        "old": "format = `\\u%04x`",
        "new": "format = `\\u%04X`",
        "package": "./pkg/lcp",
        "test": "TestCanonicalSpecModeEscaping",
        "expect": "reference mode should use lower-case hex",
    },
    {
        "id": "lcp-reference-line-separator-unescaped",
        "module": "core",
        "file": "pkg/lcp/canonical.go",
        "old": "if mode == CanonicalReference && (r == 0x2028 || r == 0x2029) {",
        "new": "if false && (r == 0x2028 || r == 0x2029) {",
        "package": "./pkg/lcp",
        "test": "TestCanonicalSpecModeEscaping",
        "expect": "reference mode should escape U+2028/U+2029",
    },
    {
        "id": "lcp-trailing-zero-scale-lost",
        "module": "core",
        "file": "pkg/lcp/canonical.go",
        "old": "power += len(digits) - len(trimmed)",
        "new": "power += 0",
        "package": "./pkg/lcp",
        "test": "TestCanonicalNumbers",
        "expect": "1E-1",
    },
    {
        "id": "lcp-negative-zero-preserved",
        "module": "core",
        "file": "pkg/lcp/canonical.go",
        "old": "if s == \"-0\" {",
        "new": "if false {",
        "package": "./pkg/lcp",
        "test": "TestCanonicalNumbers",
        "expect": "Canonical({\"n\":-0}) = {\"n\":-0}",
    },
    {
        "id": "lcp-reference-number-rewritten",
        "module": "core",
        "file": "pkg/lcp/canonical.go",
        "old": "if mode == CanonicalReference {\n\t\treturn s, nil\n\t}",
        "new": "if false {\n\t\treturn s, nil\n\t}",
        "package": "./pkg/lcp",
        "test": "TestCanonicalMatchesReferenceProducer",
        "expect": "reference-mode divergence",
    },
    {
        "id": "lcp-missing-signature-unreported",
        "module": "core",
        "file": "pkg/lcp/canonical.go",
        "old": "\tif !found {\n\t\treturn nil, ErrSignatureMissing\n\t}",
        "new": "\tif !found {\n\t\treturn nil, nil\n\t}",
        "package": "./pkg/lcp",
        "test": "TestCanonicalForSigningRemovesOnlyTopLevelSignature",
        "expect": "missing signature object must be reported",
    },
    {
        "id": "lcp-trailing-data-accepted",
        "module": "core",
        "file": "pkg/lcp/canonical.go",
        "old": "if _, err := dec.Token(); !errors.Is(err, io.EOF) {",
        "new": "if _, err := dec.Token(); !errors.Is(err, io.EOF) && false {",
        "package": "./pkg/lcp",
        "test": "TestCanonicalRejectsMalformedInput",
        "expect": "error = <nil>, want ErrLicenseSyntax",
    },
    {
        "id": "lcp-chain-trust-check-removed",
        "module": "core",
        "file": "pkg/lcp/signature.go",
        "old": "if !signed {",
        "new": "if !signed && signed {",
        "package": "./pkg/lcp",
        "test": "TestVerifyRejectsForeignChain",
        "expect": "foreign chain error = <nil>",
    },
    {
        "id": "lcp-cert-validity-window-removed",
        "module": "core",
        "file": "pkg/lcp/signature.go",
        "old": "if cert.NotBefore.After(l.Issued) || cert.NotAfter.Before(at) {",
        "new": "if false {",
        "package": "./pkg/lcp",
        "test": "TestCertificateValidityWindows",
        "expect": "expires before issue: error = <nil>",
    },
    {
        "id": "lcp-crl-never-consulted",
        "module": "core",
        "file": "pkg/lcp/signature.go",
        "old": "for _, revoked := range policy.CRL.RevokedCertificateEntries {",
        "new": "for _, revoked := range policy.CRL.RevokedCertificateEntries[:0] {",
        "package": "./pkg/lcp",
        "test": "TestRevokedProviderCertificateIsRefused",
        "expect": "revoked error = <nil>",
    },
    {
        "id": "lcp-crl-status-overstated",
        "module": "core",
        "file": "pkg/lcp/signature.go",
        "old": "rep.CRLChecked = policy.CRL != nil",
        "new": "rep.CRLChecked = true",
        "package": "./pkg/lcp",
        "test": "TestVerifyAcceptsARealChain",
        "expect": "CRLChecked is true without a CRL being supplied",
    },
    {
        "id": "lcp-unconfigured-root-generic-error",
        "module": "core",
        "file": "pkg/lcp/signature.go",
        "old": "if len(policy.Roots) == 0 {",
        "new": "if false {",
        "package": "./pkg/lcp",
        "test": "TestVerifyWithoutRootsConfigured",
        "expect": "want it to say",
    },
    {
        "id": "lcp-sign-does-not-excise-placeholder",
        "module": "core",
        "file": "pkg/lcp/signature.go",
        "old": "canonical, err := canonicalForNewSignature(doc, CanonicalReference)",
        "new": "canonical, err := canonicalWithMode(doc, CanonicalReference)",
        "package": "./pkg/lcp",
        "test": "TestSignExcisesAPlaceholderSignature",
        "expect": "Verify after re-signing in place",
    },
    {
        "id": "lcp-sign-key-certificate-match-unchecked",
        "module": "core",
        "file": "pkg/lcp/signature.go",
        "old": "if !publicKeyEquals(cert.PublicKey, &key.PublicKey) {",
        "new": "if !publicKeyEquals(cert.PublicKey, &key.PublicKey) && false {",
        "package": "./pkg/lcp",
        "test": "TestSignRefusesKeyCertificateMismatch",
        "expect": "mismatched key) error = <nil>",
    },
    {
        "id": "lcp-verify-skips-profile-check",
        "module": "core",
        "file": "pkg/lcp/signature.go",
        "old": "if err := l.CheckProfile(); err != nil {",
        "new": "if err := l.CheckProfile(); false && err != nil {",
        "package": "./pkg/lcp",
        "test": "TestVerifyRejectsNonProfileSignatureAlgorithm",
        "expect": "error = <nil>, want ErrUnsupportedAlgorithm",
    },
    {
        "id": "lcp-cert-expiry-at-update-ignored",
        "module": "core",
        "file": "pkg/lcp/signature.go",
        "old": "\tif l.Updated.After(at) {\n\t\tat = l.Updated\n\t}",
        "new": "\tif false {\n\t\tat = l.Updated\n\t}",
        "package": "./pkg/lcp",
        "test": "TestCertificateValidityWindows",
        "expect": "valid at issue, expired by the last update: error = <nil>",
    },
    {
        "id": "lcp-absent-issued-reported-as-syntax",
        "module": "core",
        "file": "pkg/lcp/license.go",
        "old": "\tif w.Issued == \"\" {\n\t\treturn nil, fmt.Errorf(\"%w: issued\", ErrIncomplete)\n\t}",
        "new": "\tif false {\n\t\treturn nil, fmt.Errorf(\"%w: issued\", ErrIncomplete)\n\t}",
        "package": "./pkg/lcp",
        "test": "TestParseRejectsIncompleteLicenses",
        "expect": "Parse without issued error = lcp: license document is not valid JSON",
    },
    {
        "id": "lcp-rights-dropped-when-present",
        "module": "core",
        "file": "pkg/lcp/license.go",
        "old": "if len(w.Rights) > 0 {",
        "new": "if false {",
        "package": "./pkg/lcp",
        "test": "TestParseRightsAbsentIsNotZero",
        "expect": "print:0 read as <nil>",
    },
    {
        "id": "lcp-unknown-members-not-reported",
        "module": "core",
        "file": "pkg/lcp/license.go",
        "old": "if !knownMembers[k] {",
        "new": "if false {",
        "package": "./pkg/lcp",
        "test": "TestParseAcceptsAndReportsUnknownMembers",
        "expect": "UnknownMembers = [], want",
    },
    {
        "id": "lcp-unknown-members-order-nondeterministic",
        "module": "core",
        "file": "pkg/lcp/license.go",
        "old": "sort.Strings(lic.UnknownMembers)",
        "new": "sort.SliceStable(lic.UnknownMembers, func(i, j int) bool { return false })",
        "package": "./pkg/lcp",
        "test": "TestParseAcceptsAndReportsUnknownMembers",
        "expect": "UnknownMembers order is not deterministic",
    },
    {
        "id": "lcp-signature-algorithm-unchecked",
        "module": "core",
        "file": "pkg/lcp/license.go",
        "old": "if l.Signature.Algorithm != AlgRSASHA256 {",
        "new": "if false {",
        "package": "./pkg/lcp",
        "test": "TestCheckProfileMatrix",
        "expect": "CheckProfile error = <nil>",
    },
    {
        "id": "lcp-profile-uri-unchecked",
        "module": "core",
        "file": "pkg/lcp/license.go",
        "old": "if !l.IsBasicProfile() {",
        "new": "if false {",
        "package": "./pkg/lcp",
        "test": "TestCheckProfileMatrix",
        "expect": "CheckProfile error = <nil>",
    },
    {
        "id": "lcp-empty-link-href-accepted",
        "module": "core",
        "file": "pkg/lcp/license.go",
        "old": "if l.Href == \"\" {",
        "new": "if l.Href == \"\" && false {",
        "package": "./pkg/lcp",
        "test": "TestParseRejectsEmptyLinkHref",
        "expect": "error = <nil>, want ErrIncomplete",
    },
    {
        "id": "lcp-empty-links-accepted",
        "module": "core",
        "file": "pkg/lcp/license.go",
        "old": "if len(w.Links) == 0 {",
        "new": "if false {",
        "package": "./pkg/lcp",
        "test": "TestParseRejectsIncompleteLicenses",
        "expect": "Parse without links error = <nil>",
    },
    {
        "id": "lcp-signature-members-unchecked",
        "module": "core",
        "file": "pkg/lcp/license.go",
        "old": "if w.Signature.Algorithm == \"\" || w.Signature.Certificate == \"\" || w.Signature.Value == \"\" {",
        "new": "if false {",
        "package": "./pkg/lcp",
        "test": "TestParseRejectsIncompleteLicenses",
        "expect": "Parse without signature/algorithm error = <nil>",
    },
    # text_hint is the one required member whose value may be empty, so presence
    # and emptiness are two separate contracts and need two separate mutants.
    {
        "id": "lcp-texthint-absent-accepted",
        "module": "core",
        "file": "pkg/lcp/license.go",
        "old": "\tif hint.UserKey.TextHint == nil {\n\t\treturn nil, fmt.Errorf(\"%w: encryption/user_key/text_hint\", ErrIncomplete)\n\t}",
        "new": "\tif false {\n\t\treturn nil, fmt.Errorf(\"%w: encryption/user_key/text_hint\", ErrIncomplete)\n\t}",
        "package": "./pkg/lcp",
        "test": "TestParseRejectsIncompleteLicenses",
        "expect": "Parse without encryption/user_key/text_hint error = <nil>",
    },
    {
        "id": "lcp-texthint-emptiness-refused",
        "module": "core",
        "file": "pkg/lcp/license.go",
        "old": "if hint.UserKey.TextHint == nil {",
        "new": "if hint.UserKey.TextHint == nil || *hint.UserKey.TextHint == \"\" {",
        "package": "./pkg/lcp",
        "test": "TestParseAcceptsAnEmptyTextHint",
        "expect": "incomplete: encryption/user_key/text_hint",
    },
    {
        "id": "lcp-updated-defaults-to-zero-time",
        "module": "core",
        "file": "pkg/lcp/license.go",
        "old": "updated := issued",
        "new": "updated := time.Time{}",
        "package": "./pkg/lcp",
        "test": "TestParseDefaultsUpdatedToIssued",
        "expect": "want it to default to Issued",
    },
    {
        "id": "lcp-malformed-date-read-as-zero-time",
        "module": "core",
        "file": "pkg/lcp/license.go",
        "old": "t, err := time.Parse(time.RFC3339, s)\n\tif err != nil {",
        "new": "t, err := time.Parse(time.RFC3339, s)\n\tif false && err != nil {",
        "package": "./pkg/lcp",
        "test": "TestParseRejectsMalformedDates",
        "expect": "want ErrLicenseSyntax",
    },
    {
        "id": "lcp-user-name-not-decryptable",
        "module": "core",
        "file": "pkg/lcp/keys.go",
        "old": '"id": &out.ID, "email": &out.Email, "name": &out.Name,',
        "new": '"id": &out.ID, "email": &out.Email,',
        "package": "./pkg/lcp",
        "test": "TestDecryptUserFields",
        "expect": "decrypted user = ",
    },
    {
        "id": "lcp-unknown-encrypted-member-fatal",
        "module": "core",
        "file": "pkg/lcp/keys.go",
        "old": "continue // §7.2: ignore what we do not understand",
        "new": "return out, fmt.Errorf(\"lcp: unsupported encrypted member %s\", name)",
        "package": "./pkg/lcp",
        "test": "TestDecryptUserIgnoresUnknownEncryptedMembers",
        "expect": "should be ignored",
    },
    {
        "id": "lcp-passphrase-checks-key-before-profile",
        "module": "core",
        "file": "pkg/lcp/keys.go",
        "old": "if err := l.CheckProfile(); err != nil {\n\t\treturn err\n\t}\n\treturn l.CheckUserKey(UserKey(passphrase))",
        "new": "if err := l.CheckProfile(); false && err != nil {\n\t\treturn err\n\t}\n\treturn l.CheckUserKey(UserKey(passphrase))",
        "package": "./pkg/lcp",
        "test": "TestCheckPassphraseRefusesUnsupportedProfile",
        "expect": "error = <nil>, want ErrUnsupportedProfile",
    },
    {
        "id": "lcp-contentkey-decrypts-before-profile",
        "module": "core",
        "file": "pkg/lcp/keys.go",
        "old": "if err := l.CheckProfile(); err != nil {\n\t\treturn nil, err\n\t}\n\tenc, err := base64",
        "new": "if err := l.CheckProfile(); false && err != nil {\n\t\treturn nil, err\n\t}\n\tenc, err := base64",
        "package": "./pkg/lcp",
        "test": "TestCheckPassphraseRefusesUnsupportedProfile",
        "expect": "ContentKey error = <nil>, want ErrUnsupportedProfile",
    },
    {
        "id": "lcp-container-duplicate-entry-last-wins",
        "module": "core",
        "file": "pkg/lcp/container.go",
        "old": 'if _, dup := c.files[f.Name]; !dup {\n\t\t\tc.files[f.Name] = f\n\t\t}',
        "new": 'c.files[f.Name] = f\n\t\tif _, dup := c.files[f.Name]; !dup {\n\t\t\tc.files[f.Name] = f\n\t\t}',
        "package": "./pkg/lcp",
        "test": "TestContainerFirstLicenseEntryWins",
        "expect": "Unlock with a trailing imposter license",
    },
    {
        "id": "lcp-container-exclusion-list-unchecked",
        "module": "core",
        "file": "pkg/lcp/container.go",
        "old": "if neverEncrypted[name] {",
        "new": "if false && neverEncrypted[name] {",
        "package": "./pkg/lcp",
        "test": "TestContainerRefusesEncryptedExcludedResources",
        "expect": "want ErrExcludedResourceEncrypted",
    },
    {
        "id": "lcp-container-rootfile-exclusion-unchecked",
        "module": "core",
        "file": "pkg/lcp/container.go",
        "old": "if _, protected := c.protected[rf.FullPath]; protected {",
        "new": "if _, protected := c.protected[rf.FullPath]; false && protected {",
        "package": "./pkg/lcp",
        "test": "TestContainerRefusesEncryptedExcludedResources",
        "expect": "want ErrExcludedResourceEncrypted",
    },
    {
        "id": "lcp-container-malformed-encryption-xml-swallowed",
        "module": "core",
        "file": "pkg/lcp/container.go",
        "old": 'if err := c.readEncryptionXML(raw); err != nil {\n\t\t\tc.encryptionErr = err\n\t\t}',
        "new": 'if err := c.readEncryptionXML(raw); err != nil && false {\n\t\t\tc.encryptionErr = err\n\t\t}',
        "package": "./pkg/lcp",
        "test": "TestContainerRejectsMalformedEncryptionXML",
        "expect": "malformed encryption.xml reported no error",
    },
    {
        "id": "lcp-container-cipher-reference-not-cleaned",
        "module": "core",
        "file": "pkg/lcp/container.go",
        "old": 'return path.Clean("/" + strings.ReplaceAll(uri, "\\\\", "/"))[1:], nil',
        "new": 'return strings.TrimPrefix(path.Clean(uri), "/"), nil',
        "package": "./pkg/lcp",
        "test": "TestContainerKeepsCipherReferencesInsideTheContainer",
        "expect": "want [etc/passwd]",
    },
    {
        "id": "lcp-container-cipher-reference-origin-unchecked",
        "module": "core",
        "file": "pkg/lcp/container.go",
        "old": 'if u, err := url.Parse(uri); err == nil && (u.Scheme != "" || u.Host != "") {',
        "new": "if _, err := url.Parse(uri); err == nil && false {",
        "package": "./pkg/lcp",
        "test": "TestContainerKeepsCipherReferencesInsideTheContainer",
        "expect": "want ErrLicenseSyntax",
    },
    {
        "id": "lcp-container-protection-requires-license",
        "module": "core",
        "file": "pkg/lcp/container.go",
        "old": "return c.License != nil || c.keyRefPresent",
        "new": "return c.License != nil",
        "package": "./pkg/lcp",
        "test": "TestContainerDetectsByEachMeans",
        "expect": "did not register as protection",
    },
    {
        "id": "lcp-container-missing-license-reported-as-unprotected",
        "module": "core",
        "file": "pkg/lcp/container.go",
        "old": 'if c.keyRefPresent {\n\t\t\treturn rep, fmt.Errorf("%w: %s is referenced but absent", ErrMissingLicense, LicensePath)',
        "new": 'if false && c.keyRefPresent {\n\t\t\treturn rep, fmt.Errorf("%w: %s is referenced but absent", ErrMissingLicense, LicensePath)',
        "package": "./pkg/lcp",
        "test": "TestContainerDetectsByEachMeans",
        "expect": "want ErrMissingLicense",
    },
    {
        "id": "lcp-container-deflate-never-inflated",
        "module": "core",
        "file": "pkg/lcp/container.go",
        "old": "if res.method == deflateMethod {",
        "new": "if false && res.method == deflateMethod {",
        "package": "./pkg/lcp",
        "test": "TestContainerUnlocksAndReadsEveryResource",
        "expect": "lcp: decrypted length does not match encryption.xml",
    },
    {
        "id": "lcp-container-original-length-unchecked",
        "module": "core",
        "file": "pkg/lcp/container.go",
        "old": "if res.originalLength > 0 && int64(len(plain)) != res.originalLength {",
        "new": "if false && res.originalLength > 0 && int64(len(plain)) != res.originalLength {",
        "package": "./pkg/lcp",
        "test": "TestContainerHonoursDeclaredOriginalLength",
        "expect": "want ErrOriginalLengthMismatch",
    },
    {
        "id": "lcp-container-algorithm-agreement-unchecked",
        "module": "core",
        "file": "pkg/lcp/container.go",
        "old": 'if res.algorithm != "" && res.algorithm != c.License.Encryption.ContentKey.Algorithm {',
        "new": 'if false && res.algorithm != "" && res.algorithm != c.License.Encryption.ContentKey.Algorithm {',
        "package": "./pkg/lcp",
        "test": "TestContainerRejectsAlgorithmDisagreement",
        "expect": "want ErrAlgorithmMismatch",
    },
    {
        "id": "lcp-container-encrypted-readable-before-unlock",
        "module": "core",
        "file": "pkg/lcp/container.go",
        "old": 'if !c.unlocked {\n\t\treturn nil, fmt.Errorf("%w: %s", ErrLocked, name)\n\t}',
        "new": 'if false && !c.unlocked {\n\t\treturn nil, fmt.Errorf("%w: %s", ErrLocked, name)\n\t}',
        "package": "./pkg/lcp",
        "test": "TestContainerWrongPassphraseLeavesResourcesLocked",
        "expect": "want ErrLocked",
    },
    {
        "id": "lcp-container-ciphertext-budget-excludes-overhead",
        "module": "core",
        "file": "pkg/lcp/container.go",
        "old": "ct, err := c.raw(f, limit+2*BlockSize)",
        "new": "ct, err := c.raw(f, limit)",
        "package": "./pkg/lcp",
        "test": "TestContainerCiphertextBudgetCoversEncryptionOverhead",
        "expect": "entry exceeds",
    },
    {
        "id": "lcp-container-plaintext-limit-unenforced",
        "module": "core",
        "file": "pkg/lcp/container.go",
        "old": "if int64(len(plain)) > limit {",
        "new": "if false && int64(len(plain)) > limit {",
        "package": "./pkg/lcp",
        "test": "TestContainerCiphertextBudgetCoversEncryptionOverhead",
        "expect": "Resource returned 1997 bytes against a 1996 byte limit",
    },
    # Producer side (§2.1 exclusions, §2.2 declarations). Each of these was a live
    # defect: the properties check read an element where the Package Document
    # carries an attribute, so nav documents were encrypted on every run, and the
    # cover meta read text content where OPF2 carries a content attribute.
    {
        "id": "lcp-producer-manifest-properties-read-as-element",
        "module": "core",
        "file": "pkg/lcp/packager.go",
        "old": 'Properties string `xml:"properties,attr"`',
        "new": 'Properties string `xml:"properties"`',
        "package": "./pkg/lcp",
        "test": "TestIssueProtectUnlockRoundTrip",
        "expect": "ProtectedResources = [OEBPS/nav.xhtml",
    },
    {
        "id": "lcp-producer-nav-property-ignored",
        "module": "core",
        "file": "pkg/lcp/packager.go",
        "old": 'if token == "nav" || token == "cover-image" {',
        "new": 'if token == "cover-image" {',
        "package": "./pkg/lcp",
        "test": "TestIssueProtectUnlockRoundTrip",
        "expect": "OEBPS/nav.xhtml",
    },
    {
        "id": "lcp-producer-ncx-media-type-ignored",
        "module": "core",
        "file": "pkg/lcp/packager.go",
        "old": 'if it.MediaType == "application/x-dtbncx+xml" {',
        "new": "if false {",
        "package": "./pkg/lcp",
        "test": "TestIssueProtectUnlockRoundTrip",
        "expect": "OEBPS/toc.ncx",
    },
    {
        "id": "lcp-producer-opf2-cover-meta-attribute-ignored",
        "module": "core",
        "file": "pkg/lcp/packager.go",
        "old": "id := strings.TrimSpace(m.Value)",
        "new": "id := strings.TrimSpace(m.Text)",
        "package": "./pkg/lcp",
        "test": "TestIssueProtectUnlockRoundTrip",
        "expect": "OEBPS/images/cover.jpeg",
    },
    {
        "id": "lcp-producer-guide-cover-reference-ignored",
        "module": "core",
        "file": "pkg/lcp/packager.go",
        "old": 'if strings.EqualFold(g.Type, "cover") {',
        "new": "if false {",
        "package": "./pkg/lcp",
        "test": "TestIssueProtectUnlockRoundTrip",
        "expect": "OEBPS/titlepage.xhtml",
    },
    {
        "id": "lcp-producer-ciphertext-keeps-source-method",
        "module": "core",
        "file": "pkg/lcp/packager.go",
        "old": "\t\t\th.Method = zip.Store",
        "new": "\t\t\t// the declaration records compression; the zip layer does not add its own",
        "package": "./pkg/lcp",
        "test": "TestProtectDeclaresCompressionAndStoresCiphertext",
        "expect": "want Store",
    },
    {
        "id": "lcp-producer-protects-a-non-publication",
        "module": "core",
        "file": "pkg/lcp/packager.go",
        "old": 'if c.files["META-INF/container.xml"] == nil {',
        "new": "if false {",
        "package": "./pkg/lcp",
        "test": "TestProtectRefusesContainersItCannotHonestlyReturn",
        "expect": "want lcp: container is not a publication",
    },
    {
        "id": "pse-image-type-check-removed",
        "module": "core",
        "file": "pkg/pse/link.go",
        "old": "if !IsImageType(l.Type) {",
        "new": "if false {",
        "package": "./pkg/pse",
        "test": "TestParseAcquisitionLinkErrors",
        "expect": "is not an image type allowed by OPDS-PSE",
    },
    {
        "id": "pse-page-number-template-check-removed",
        "module": "core",
        "file": "pkg/pse/link.go",
        "old": "if !strings.Contains(l.Href, PageNumberTemplate) {",
        "new": "if false {",
        "package": "./pkg/pse",
        "test": "TestParseAcquisitionLinkErrors",
        "expect": "must contain the {pageNumber} template",
    },
    {
        "id": "pse-count-zero-accepted",
        "module": "core",
        "file": "pkg/pse/link.go",
        "old": "if err != nil || n < 1 {",
        "new": "if err != nil {",
        "package": "./pkg/pse",
        "test": "TestParseAcquisitionLinkErrors",
        "expect": r'positive integer, got \"0\"',
    },
    {
        "id": "pse-last-read-zero-accepted",
        "module": "core",
        "file": "pkg/pse/link.go",
        "old": "if err != nil || v < 1 {",
        "new": "if err != nil {",
        "package": "./pkg/pse",
        "test": "TestParseAcquisitionLinkErrors",
        "expect": r'numbering starts at 1), got \"0\"',
    },
    {
        "id": "pse-page-upper-bound-removed",
        "module": "core",
        "file": "pkg/pse/link.go",
        "old": "if page < 0 || page > l.Count-1 {",
        "new": "if page < 0 {",
        "package": "./pkg/pse",
        "test": "TestPageURL",
        "expect": "page 2 out of range 0..1",
    },
    {
        "id": "pse-max-width-zero-accepted",
        "module": "core",
        "file": "pkg/pse/link.go",
        "old": "if maxWidth <= 0 {",
        "new": "if maxWidth < 0 {",
        "package": "./pkg/pse",
        "test": "TestPageURL",
        "expect": "maxWidth 0 is not positive",
    },
    {
        "id": "pse-last-read-date-unvalidated",
        "module": "core",
        "file": "pkg/pse/link.go",
        "old": "if !isAtomDate(d) {",
        "new": "if false {",
        "package": "./pkg/pse",
        "test": "TestParseAcquisitionLinkErrors",
        "expect": "must conform to the Atom date construct",
    },
    {
        "id": "opds-stream-advertises-illegal-link",
        "module": "server",
        "file": "internal/opds/pse.go",
        "old": '\tif ps.Count < 1 || !pse.IsImageType(ps.Type) {',
        "new": '\tif false {',
        "package": "./internal/opds",
        "test": "TestIllegalStreamsAreNeverAdvertised",
        "expect": 'feed advertises a stream it cannot describe',
    },
    {
        "id": "opds-stream-href-drops-page-template",
        "module": "server",
        "file": "internal/opds/pse.go",
        "old": '+ "/" + pse.PageNumberTemplate',
        "new": '+ "/page"',
        "package": "./internal/opds",
        "test": "TestStreamLinkRoundTripsThroughTheParser",
        # The feed goes through core/pkg/pse before any producer-side assertion
        # runs, so the parser's wording is what proves this mutant dead.
        "expect": 'must contain the {pageNumber} template',
    },
    {
        "id": "opds-feed-omits-pse-namespace",
        "module": "server",
        "file": "internal/opds/opds12.go",
        "old": '\t\tXMLPSE:          pse.NamespaceURI,',
        "new": '\t\tXMLPSE:          pse.NamespaceURI[:0],',
        "package": "./internal/opds",
        "test": "TestFeedDeclaresThePSENamespace",
        "expect": 'must declare the namespace',
    },
    {
        "id": "opds-entry-omits-pse-namespace",
        "module": "server",
        "file": "internal/opds/detail.go",
        "old": '\tentry.XMLPSE = pse.NamespaceURI',
        "new": '\tentry.XMLPSE = pse.NamespaceURI[:0]',
        "package": "./internal/opds",
        "test": "TestEditionEntryStandaloneCarriesNamespaceAndLink",
        "expect": 'must declare the namespace itself',
    },
    {
        "id": "cbz-stream-type-ignores-mixed-payloads",
        "module": "core",
        "file": "pkg/parse/cbz/pages.go",
        "old": '\t\t} else if media != t {',
        "new": '\t\t} else if false {',
        "package": "./pkg/parse/cbz",
        "test": "TestStreamMediaTypeCases",
        "expect": 'pages mix image/png and image/jpeg',
    },
    {
        "id": "cbz-stream-type-ignores-unproven-payload",
        "module": "core",
        "file": "pkg/parse/cbz/pages.go",
        "old": '\t\tif t == "" {',
        "new": '\t\tif false {',
        "package": "./pkg/parse/cbz",
        "test": "TestStreamMediaTypeCases",
        "expect": 'is not a jpeg, png or gif image',
    },
    {
        "id": "cbz-page-index-bound-removed",
        "module": "core",
        "file": "pkg/parse/cbz/pages.go",
        "old": '\tif index < 0 || index >= len(refs) {',
        "new": '\tif index < 0 {',
        "package": "./pkg/parse/cbz",
        "test": "TestPageIndexOutOfRange",
        "expect": 'index out of range [2]',
    },
    {
        "id": "http-stream-serves-stale-catalog-record",
        "module": "server",
        "file": "internal/httpapi/pse.go",
        "old": '\tif size != f.Size {',
        "new": '\tif false {',
        "package": "./internal/httpapi",
        "test": "TestPageStreamRefusesFileChangedAfterScan",
        "expect": 'stale catalog record was advertised',
    },
    {
        "id": "http-stream-serves-protected-archive",
        "module": "server",
        "file": "internal/httpapi/pse.go",
        "old": '\tif f.Protected {',
        "new": '\tif false {',
        "package": "./internal/httpapi",
        "test": "TestPageStreamRefusesProtectedAndUnknownFormats",
        "expect": 'protected archive was advertised',
    },
    {
        "id": "http-stream-page-bound-removed",
        "module": "server",
        "file": "internal/httpapi/pse.go",
        "old": '\tif page < 0 || page >= stream.Count {',
        "new": '\tif page < 0 {',
        "package": "./internal/httpapi",
        "test": "TestPageStreamFeedAndServe",
        "expect": 'beyond-count page: 502',
    },
]


def digest(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def go_test(module_dir: Path, pkg: str, test: str, wall_seconds: int = 300):
    """Run one targeted test in a copied module, returning (command, output, exit).

    -trimpath matters more than it looks: every mutant runs in a directory with
    a unique absolute path, and without it each path is a distinct build-cache
    key, so the whole dependency graph recompiles per mutant and slow builds
    were being mistaken for verdicts. Go's own -timeout produces a goroutine
    dump on a hang; the harness kill below would only leave silence.
    """
    command = [GO, "test", "-trimpath", pkg, "-run", "^" + test + "$", "-count=1",
               "-timeout", f"{max(30, wall_seconds - 60)}s"]
    env = os.environ.copy()
    env["GOPROXY"] = "off"
    try:
        completed = subprocess.run(command, cwd=module_dir, env=env, text=True, errors="replace",
                                   stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=wall_seconds)
        return command, completed.stdout, completed.returncode
    except subprocess.TimeoutExpired as exc:
        partial = exc.stdout or b""
        if isinstance(partial, bytes):
            partial = partial.decode("utf-8", "replace")
        return command, partial + f"\nkilled: no verdict after {wall_seconds}s\n", 124


def main() -> int:
    EVIDENCE.mkdir(parents=True, exist_ok=True)
    # Every mutant must name the assertion that proves its contract was caught.
    # A mutant without one could be "killed" by any unrelated failure, which is
    # how every server mutant in this campaign once passed without running.
    unpredicted = [m["id"] for m in MUTANTS if not m.get("expect")]
    if unpredicted:
        (EVIDENCE / "results.json").write_text(json.dumps(
            {"status": "FAIL", "pass": 0, "fail": len(unpredicted),
             "reason": "mutants with no predicted assertion", "ids": unpredicted}, indent=2) + "\n")
        print(json.dumps({"status": "FAIL", "reason": "mutants with no predicted assertion",
                          "ids": unpredicted}))
        return 1
    if WORK.exists():
        shutil.rmtree(WORK)
    WORK.mkdir(parents=True)
    originals = {(m["module"], m["file"]): digest(ROOT / m["module"] / m["file"]) for m in MUTANTS}
    results = []
    # Positive control: a targeted test that already fails on unmutated code
    # would "reject" every mutant thrown at it. Each distinct test is run once
    # against a pristine copy first, and the campaign cannot proceed if one
    # fails, because every later result would be meaningless.
    control_case = WORK / "_control"
    for name in GO_MODULES:
        shutil.copytree(ROOT / name, control_case / name)
    controls = sorted({(m["module"], m["package"], m["test"]) for m in MUTANTS})
    for module, pkg, test in controls:
        command, output, code = go_test(control_case / module, pkg, test)
        (EVIDENCE / (f"control-{module}-{test}.log")).write_text(
            "$ " + " ".join(command) + "\n" + output + f"\nexit={code}\n")
        if code != 0:
            results.append({"id": f"control-{module}-{test}", "status": "FAIL",
                            "reason": "targeted test does not pass on unmutated code, so it cannot reject anything"})
    for mutant in MUTANTS:
        case = WORK / mutant["id"]
        for name in GO_MODULES:
            shutil.copytree(ROOT / name, case / name)
        module_copy = case / mutant["module"]
        target = module_copy / mutant["file"]
        text = target.read_text()
        occurrences = text.count(mutant["old"])
        if occurrences != 1:
            results.append({"id": mutant["id"], "status": "FAIL", "reason": f"mutation anchor count {occurrences}"})
            continue
        target.write_text(text.replace(mutant["old"], mutant["new"], 1))
        command, output, code = go_test(module_copy, mutant["package"], mutant["test"])
        log = "$ " + " ".join(command) + "\n" + output + f"\nexit={code}\n"
        (EVIDENCE / (mutant["id"] + ".log")).write_text(log)
        setup = next((mark for mark in BUILD_FAILURES if mark in output), "")
        killed = code != 0 and code != 124
        reason = "" if killed else "targeted test accepted the mutant"
        expect = mutant.get("expect", "")
        if code == 124:
            killed = False
            reason = "no verdict: the targeted test did not finish, which proves nothing about the mutant"
        elif setup:
            killed = False
            reason = f"mutant never ran: the harness could not build the package ({setup})"
        elif killed and expect and expect not in output:
            killed = False
            reason = f"test failed, but not with the predicted assertion {expect!r}"
        elif killed and expect:
            reason = "rejected with the predicted assertion"
        results.append({
            "id": mutant["id"],
            "status": "PASS" if killed else "FAIL",
            "reason": reason,
            "expected": expect or "targeted test rejects mutant",
            "test": mutant["test"],
            "exit": code,
            "log": mutant["id"] + ".log",
        })
        print(f"{mutant['id']}: {'PASS' if killed else 'FAIL'} {reason}", file=sys.stderr, flush=True)
    unchanged = all(digest(ROOT / module / name) == before for (module, name), before in originals.items())
    if not unchanged:
        results.append({"id": "original-tree-unchanged", "status": "FAIL"})
    status = "PASS" if len(results) == len(MUTANTS) and all(r["status"] == "PASS" for r in results) and unchanged else "FAIL"
    summary = {
        "status": status,
        "pass": sum(r["status"] == "PASS" for r in results),
        "fail": sum(r["status"] == "FAIL" for r in results),
        "mutants": len(MUTANTS),
        "controls": len(controls),
        "with_predicted_assertion": sum(1 for m in MUTANTS if m.get("expect")),
        "original_tree_unchanged": unchanged,
        "results": results,
    }
    (EVIDENCE / "results.json").write_text(json.dumps(summary, indent=2) + "\n")
    shutil.rmtree(WORK)
    print(json.dumps({"status": status, "pass": summary["pass"], "fail": summary["fail"],
                      "mutants": summary["mutants"], "controls": summary["controls"],
                      "with_predicted_assertion": summary["with_predicted_assertion"]}))
    return 0 if status == "PASS" else 1


if __name__ == "__main__":
    raise SystemExit(main())
