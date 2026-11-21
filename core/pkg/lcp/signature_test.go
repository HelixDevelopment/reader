package lcp

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"
)

// chain is a throwaway License Authority: a self-signed root plus one Provider
// Certificate it signed. §5.2.2 makes the root the reading system's trust anchor
// and §5.2.1 makes the provider certificate the signing key's holder, so every
// §5.5 rule is exercised against a chain that actually verifies.
type chain struct {
	root        *x509.Certificate
	rootKey     *rsa.PrivateKey
	provider    *x509.Certificate
	providerKey *rsa.PrivateKey
}

func mustTime(tb testing.TB, s string) time.Time {
	tb.Helper()
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		tb.Fatal(err)
	}
	return t
}

func newChainTB(tb testing.TB, providerNotBefore, providerNotAfter string) *chain {
	tb.Helper()
	c := &chain{}
	c.root, c.rootKey = selfSigned(tb, &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Helix Test License Authority", Organization: []string{"Helix"}},
		NotBefore:             mustTime(tb, "2020-01-01T00:00:00Z"),
		NotAfter:              mustTime(tb, "2040-01-01T00:00:00Z"),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	})

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		tb.Fatal(err)
	}
	c.providerKey = key
	c.provider = signedBy(tb, &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "https://helix.example.org", Organization: []string{"Helix Example Store"}},
		NotBefore:    mustTime(tb, providerNotBefore),
		NotAfter:     mustTime(tb, providerNotAfter),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}, c.root, key, c.rootKey)
	return c
}

// defaultChain spans the fixture license's issued/updated instants, so tests
// about the signature itself are not accidentally tests about validity windows.
func defaultChain(tb testing.TB) *chain {
	tb.Helper()
	return newChainTB(tb, "2026-01-01T00:00:00Z", "2031-01-01T00:00:00Z")
}

func selfSigned(tb testing.TB, tmpl *x509.Certificate) (*x509.Certificate, *rsa.PrivateKey) {
	tb.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		tb.Fatal(err)
	}
	return signCert(tb, tmpl, tmpl, key, key), key
}

func signedBy(tb testing.TB, tmpl, issuer *x509.Certificate, subjectKey, issuerKey *rsa.PrivateKey) *x509.Certificate {
	tb.Helper()
	return signCert(tb, tmpl, issuer, subjectKey, issuerKey)
}

func signCert(tb testing.TB, tmpl, issuer *x509.Certificate, subjectKey, issuerKey *rsa.PrivateKey) *x509.Certificate {
	tb.Helper()
	der, err := x509.CreateCertificate(rand.Reader, tmpl, issuer, &subjectKey.PublicKey, issuerKey)
	if err != nil {
		tb.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		tb.Fatal(err)
	}
	return cert
}

// policyFor trusts only c's root, which is what §5.5.1 point 1 asks of a
// reading system: the anchor is embedded, not inferred from the wire.
func (c *chain) policy() CertificatePolicy {
	return CertificatePolicy{Roots: []*x509.Certificate{c.root}}
}

func TestVerifyAcceptsARealChain(t *testing.T) {
	c := defaultChain(t)
	lic, _ := signLicense(t, c, nil)
	rep, err := lic.Verify(c.policy())
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if rep.Mode != CanonicalReference {
		t.Errorf("signature verified under mode %v, expected the reference convention", rep.Mode)
	}
	if rep.CRLChecked {
		t.Error("CRLChecked is true without a CRL being supplied")
	}
	if rep.ProviderCert == nil || rep.ProviderCert.SerialNumber.Int64() != 2 {
		t.Errorf("ProviderCert = %v", rep.ProviderCert)
	}
}

// TestVerifyRejectsOneChangedByte is the property the signature exists for
// (§7.4: must not open a publication with an invalid signature). The rights are
// edited in the raw document, leaving every other byte and the signature value
// untouched.
func TestVerifyRejectsOneChangedByte(t *testing.T) {
	c := defaultChain(t)
	_, doc := signLicense(t, c, nil)
	tampered := append([]byte(nil), doc...)
	i := indexOf(t, tampered, `"print": 10`)
	copy(tampered[i+len(`"print": `):], `9`)
	if bytes.Equal(tampered, doc) {
		t.Fatal("the fixture carries no print right; the tamper did not happen")
	}
	lic := mustParseTB(t, tampered)
	if _, err := lic.Verify(c.policy()); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("tampered rights error = %v, want ErrBadSignature", err)
	}
}

// TestVerifyRejectsForeignChain covers two distinct rejections: a certificate
// no embedded root signed, and a signature value that the embedded certificate's
// key did not produce.
func TestVerifyRejectsForeignChain(t *testing.T) {
	c := defaultChain(t)
	other := defaultChain(t)

	lic, _ := signLicense(t, other, nil)
	if _, err := lic.Verify(c.policy()); !errors.Is(err, ErrUntrustedRoot) {
		t.Fatalf("foreign chain error = %v, want ErrUntrustedRoot", err)
	}
	// Same content and algorithm, signature value swapped for one produced by a
	// different key. The chain check passes because the certificate is c's, so
	// only the signature check can refuse this.
	signed, doc := signLicense(t, c, nil)
	foreign, _ := signLicense(t, other, nil)
	replaced := swapMember(t, doc, []string{"signature", "value"}, foreign.Signature.Value)
	if bytes.Equal(replaced, doc) {
		t.Fatal("the two signatures are identical; the swap tested nothing")
	}
	if mustParseTB(t, replaced).Signature.Certificate != signed.Signature.Certificate {
		t.Fatal("the swap changed the certificate as well as the value")
	}
	if _, err := mustParseTB(t, replaced).Verify(c.policy()); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("swapped signature value error = %v, want ErrBadSignature", err)
	}
}

// TestSignRefusesKeyCertificateMismatch: Sign would otherwise produce a
// signature object whose certificate does not correspond to the key that signed,
// which reads as a valid license until someone tries to verify it.
func TestSignRefusesKeyCertificateMismatch(t *testing.T) {
	c := defaultChain(t)
	other := defaultChain(t)
	doc, _ := buildLicenseTB(t, "", nil)
	if _, err := Sign(doc, c.provider, other.providerKey); !errors.Is(err, ErrCertificate) {
		t.Fatalf("Sign(mismatched key) error = %v, want ErrCertificate", err)
	}
}

// TestSignExcisesAPlaceholderSignature: a producer re-signing a document that
// already carries a signature member must sign the canonical form with that
// member removed (§5.5.2 step 1 is the same operation on both sides). Signing
// the raw bytes instead would produce a value no verifier can reproduce.
func TestSignExcisesAPlaceholderSignature(t *testing.T) {
	c := defaultChain(t)
	// buildLicenseTB always emits a syntactically valid but cryptographically
	// meaningless signature object, which is exactly a placeholder.
	doc, _ := buildLicenseTB(t, "pasSED1!", nil)
	placeholder := base64.StdEncoding.EncodeToString([]byte("not-a-real-signature"))
	if !bytes.Contains(doc, []byte(placeholder)) {
		t.Fatal("the fixture no longer carries a placeholder signature; this test proves nothing")
	}
	sig, err := Sign(doc, c.provider, c.providerKey)
	if err != nil {
		t.Fatalf("Sign over a signed document: %v", err)
	}
	replaced := replaceSignature(t, doc, sig)
	lic := mustParseTB(t, replaced)
	if _, err := lic.Verify(c.policy()); err != nil {
		t.Fatalf("Verify after re-signing in place: %v", err)
	}
	if bytes.Equal(replaced, doc) {
		t.Error("the signature members were not replaced")
	}
}

// TestVerifyRejectsNonProfileSignatureAlgorithm: §3.8 requires signature/algorithm
// to match the algorithm named by the Encryption Profile, and §6.3 names RSA with
// SHA-256. A license claiming an ECDSA signature is outside the profile Helix
// fulfills and must be refused before any verification is attempted.
func TestVerifyRejectsNonProfileSignatureAlgorithm(t *testing.T) {
	c := defaultChain(t)
	lic, doc := signLicense(t, c, nil)
	const ecdsa = "http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha256"
	swapped := mustParseTB(t, swapMember(t, doc, []string{"signature", "algorithm"}, ecdsa))
	if swapped.Signature.Value != lic.Signature.Value {
		t.Fatal("swapMember altered more than the algorithm")
	}
	_, err := swapped.Verify(c.policy())
	if !errors.Is(err, ErrUnsupportedAlgorithm) {
		t.Fatalf("error = %v, want ErrUnsupportedAlgorithm", err)
	}
	if !strings.Contains(err.Error(), "signature") {
		t.Errorf("error = %v, want it to name the signature algorithm", err)
	}
}

func replaceSignature(tb testing.TB, doc []byte, sig Signature) []byte {
	tb.Helper()
	var m map[string]any
	if err := json.Unmarshal(doc, &m); err != nil {
		tb.Fatal(err)
	}
	m["signature"] = map[string]any{
		"algorithm":   sig.Algorithm,
		"certificate": sig.Certificate,
		"value":       sig.Value,
	}
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		tb.Fatal(err)
	}
	return out
}

// TestVerifyWithoutRootsConfigured states the honest failure: Helix ships no
// Root Certificate of its own, so until an operator supplies one no license can
// be opened, and the error says so instead of reporting a bad signature.
func TestVerifyWithoutRootsConfigured(t *testing.T) {
	c := defaultChain(t)
	lic, _ := signLicense(t, c, nil)
	_, err := lic.Verify(CertificatePolicy{})
	if !errors.Is(err, ErrUntrustedRoot) {
		t.Fatalf("no-roots error = %v, want ErrUntrustedRoot", err)
	}
	if want := "no root certificate configured"; !strings.Contains(err.Error(), want) {
		t.Errorf("error = %v, want it to say %q", err, want)
	}
}

// TestRevokedProviderCertificateIsRefused covers §5.5.1 point 3.
func TestRevokedProviderCertificateIsRefused(t *testing.T) {
	c := defaultChain(t)
	lic, _ := signLicense(t, c, nil)
	policy := c.policy()
	policy.CRL = &x509.RevocationList{
		RevokedCertificateEntries: []x509.RevocationListEntry{
			{SerialNumber: big.NewInt(2), RevocationTime: time.Now().UTC()},
		},
	}
	if _, err := lic.Verify(policy); !errors.Is(err, ErrRevoked) {
		t.Fatalf("revoked error = %v, want ErrRevoked", err)
	}
	// An unrelated serial must not trip it.
	policy.CRL.RevokedCertificateEntries[0].SerialNumber = big.NewInt(99)
	rep, err := lic.Verify(policy)
	if err != nil {
		t.Fatalf("license with an unlisted serial failed: %v", err)
	}
	if !rep.CRLChecked {
		t.Error("CRLChecked is false although a CRL was supplied")
	}
}

// TestCertificateValidityWindows pins §5.2.1 (the certificate must have been
// valid when the license was issued) and §5.5.1 point 4 (it must not have
// expired by the last update). Both are anchored on license fields, not on
// wall-clock time, so a license keeps opening after its certificate rolls over.
func TestCertificateValidityWindows(t *testing.T) {
	cases := []struct {
		name      string
		notBefore string
		notAfter  string
		mutate    func(map[string]any)
		wantErr   error
	}{
		{
			name:      "expires before issue",
			notBefore: "2020-01-01T00:00:00Z", notAfter: "2026-06-01T00:00:00Z",
			wantErr: ErrCertOutsideValidity,
		},
		{
			name:      "begins after issue",
			notBefore: "2027-01-01T00:00:00Z", notAfter: "2031-01-01T00:00:00Z",
			wantErr: ErrCertOutsideValidity,
		},
		{
			name:      "valid at issue, expired by the last update",
			notBefore: "2026-01-01T00:00:00Z", notAfter: "2026-06-01T00:00:00Z",
			mutate: func(m map[string]any) {
				m["issued"] = "2026-02-01T00:00:00+00:00"
				m["updated"] = "2026-11-21T00:00:00+00:00"
			},
			wantErr: ErrCertOutsideValidity,
		},
		{
			name:      "valid across issue and last update",
			notBefore: "2026-01-01T00:00:00Z", notAfter: "2027-01-01T00:00:00Z",
			mutate: func(m map[string]any) {
				m["issued"] = "2026-02-01T00:00:00+00:00"
				m["updated"] = "2026-11-21T00:00:00+00:00"
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newChainTB(t, tc.notBefore, tc.notAfter)
			lic, _ := signLicense(t, c, tc.mutate)
			_, err := lic.Verify(c.policy())
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("%s: Verify: %v", tc.name, err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("%s: error = %v, want %v", tc.name, err, tc.wantErr)
			}
		})
	}
}

func indexOf(tb testing.TB, haystack []byte, needle string) int {
	tb.Helper()
	i := bytes.Index(haystack, []byte(needle))
	if i < 0 {
		tb.Fatalf("fixture does not contain %s", needle)
	}
	return i
}

func swapMember(tb testing.TB, doc []byte, path []string, value string) []byte {
	tb.Helper()
	var m map[string]any
	if err := json.Unmarshal(doc, &m); err != nil {
		tb.Fatal(err)
	}
	cur := m
	for _, k := range path[:len(path)-1] {
		next, ok := cur[k].(map[string]any)
		if !ok {
			tb.Fatalf("%s is not an object", k)
		}
		cur = next
	}
	cur[path[len(path)-1]] = value
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		tb.Fatal(err)
	}
	return out
}

// signLicense returns a complete License Document signed by c's provider key,
// plus its parsed form. mutate runs before signing, so a test can alter content
// and still get a valid signature over it; tampering tests change the returned
// document instead.
func signLicense(tb testing.TB, c *chain, mutate func(map[string]any)) (*License, []byte) {
	tb.Helper()
	doc, _ := buildLicenseTB(tb, "pasSED1!", mutate)
	var m map[string]any
	if err := json.Unmarshal(doc, &m); err != nil {
		tb.Fatal(err)
	}
	delete(m, "signature")
	unsigned, err := json.Marshal(m)
	if err != nil {
		tb.Fatal(err)
	}
	sig, err := Sign(unsigned, c.provider, c.providerKey)
	if err != nil {
		tb.Fatalf("Sign: %v", err)
	}
	m["signature"] = sig
	final, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		tb.Fatal(err)
	}
	return mustParseTB(tb, final), final
}
