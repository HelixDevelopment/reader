package lcp

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"testing"
)

// buildLicense returns a complete, parseable License Document for the Basic
// Encryption Profile plus the Content Key it carries. The signature members are
// syntactically present but cryptographically meaningless; tests exercising
// Verify re-sign with a real key.
//
// sealPassphrase may be "", which leaves the encryption members absent for
// syntax-only tests. mutate runs after sealing, so a test that changes the
// license id also invalidates key_check — which is the point of that test.
func buildLicenseTB(tb testing.TB, sealPassphrase string, mutate func(map[string]any)) (doc []byte, contentKey []byte) {
	tb.Helper()
	lic := map[string]any{
		"id":       "3a1f6e2c-0000-4000-8000-abcdefabcdef",
		"issued":   "2026-11-21T00:00:00+00:00",
		"updated":  "2026-11-21T00:00:00+00:00",
		"provider": "https://helix.example.org",
		"encryption": map[string]any{
			"profile": ProfileBasic,
			"content_key": map[string]any{
				"algorithm": AlgAES256CBC,
			},
			"user_key": map[string]any{
				"algorithm": AlgSHA256,
				"text_hint": "Enter your passphrase",
			},
		},
		"links": []any{
			map[string]any{"rel": "hint", "href": "https://helix.example.org/hint", "type": "text/html"},
			map[string]any{"rel": "publication", "href": "https://helix.example.org/pub.epub", "type": "application/epub+zip"},
		},
		"rights": map[string]any{"print": 10, "copy": 200},
		"user":   map[string]any{"id": "user-42"},
		"signature": map[string]any{
			"algorithm":   AlgRSASHA256,
			"certificate": base64.StdEncoding.EncodeToString([]byte("not-a-real-certificate")),
			"value":       base64.StdEncoding.EncodeToString([]byte("not-a-real-signature")),
		},
	}

	enc := lic["encryption"].(map[string]any)
	ck := enc["content_key"].(map[string]any)
	uk := enc["user_key"].(map[string]any)
	if sealPassphrase != "" {
		contentKey = make([]byte, ContentKeyBytes)
		if _, err := rand.Read(contentKey); err != nil {
			tb.Fatal(err)
		}
		sealed, check, err := SealKeys(UserKey(sealPassphrase), contentKey, lic["id"].(string))
		if err != nil {
			tb.Fatal(err)
		}
		ck["encrypted_value"] = sealed
		uk["key_check"] = check
	}
	if mutate != nil {
		mutate(lic)
	}
	out, err := json.MarshalIndent(lic, "", "  ")
	if err != nil {
		tb.Fatal(err)
	}
	return out, contentKey
}

func mustParseTB(tb testing.TB, doc []byte) *License {
	tb.Helper()
	lic, err := Parse(doc)
	if err != nil {
		tb.Fatalf("Parse: %v", err)
	}
	return lic
}

// specDigest is the SHA-256 of the canonical form published in §5.4.1 of the
// LCP 1.0 specification. Provenance: lcp-specs@812dd91d6b90087e657a5c94d04e6f16abd8873b,
// releases/lcp/latest.md, which is byte-identical to the latest.md the fetched
// copy was diffed against. The testdata/ fixtures are fenced blocks of that file.
const specDigest = "23c68442c7214ba294ddd1a2902756e9fe575116a88f36e55baf94590a90c2ad"
