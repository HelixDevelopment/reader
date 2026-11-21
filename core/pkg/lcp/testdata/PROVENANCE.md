# Fixtures extracted from the Readium LCP 1.0 specification

Source: https://raw.githubusercontent.com/readium/lcp-specs/812dd91d6b90087e657a5c94d04e6f16abd8873b/releases/lcp/latest.md
Retrieved: 2026-11-21 (UTC), rendered markdown of the published specification.
Extraction: fenced JSON blocks, line ranges recorded below, no edits to content.

| file | section | lines in latest.md |
|---|---|---|
| spec-5.3.1-pretty.json | 5.3.1 example input | 694-715 |
| spec-5.3.1-sorted.json | 5.3.1 "sort this document" step | 722-747 |
| spec-5.3.1-canonical.json | 5.3.1 whitespace-stripped canonical form | 754 |
| spec-5.4.1-signed-license.json | 5.4.1 license with signature and certificate | 795-823 |

Fidelity check: SHA-256 of `spec-5.3.1-canonical.json` (the bytes on disk, minus the
trailing newline this repo adds) equals the digest the specification itself publishes
in 5.4.1:

    23c68442c7214ba294ddd1a2902756e9fe575116a88f36e55baf94590a90c2ad

That equality is asserted by TestCanonicalSpecDigestVector, so a corrupted fixture
fails the suite rather than passing silently.

Note: spec-5.3.1-canonical.json keeps the members of `links[0]` in document order,
which contradicts the recursive sorting required by rule 2 of the same section. See
TestCanonicalSpecExample and canonical.go for the behaviour Helix implements.
