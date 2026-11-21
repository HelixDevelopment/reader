package lcp

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"time"
)

var (
	// ErrBadSignature reports a signature value that does not match the
	// canonical form of the License Document under either escaping
	// convention.
	ErrBadSignature = errors.New("lcp: signature does not match the license document")
	// ErrUntrustedRoot reports a Provider Certificate that no configured Root
	// Certificate signed (§5.5.1 point 1).
	ErrUntrustedRoot = errors.New("lcp: provider certificate is not signed by a trusted root")
	// ErrCertOutsideValidity reports a Provider Certificate that was not valid
	// when the license was issued, or had expired by its last update
	// (§5.2.1, §5.5.1 point 4).
	ErrCertOutsideValidity = errors.New("lcp: provider certificate was not valid for this license period")
	// ErrRevoked reports a Provider Certificate listed in a supplied
	// Certificate Revocation List (§5.5.1 point 3).
	ErrRevoked = errors.New("lcp: provider certificate is revoked")
	// ErrCertificate reports an undecodable signature/certificate member.
	ErrCertificate = errors.New("lcp: cannot parse provider certificate")
)

// CertificatePolicy carries the trust material §5.5.1 needs. Helix embeds no
// Root Certificate of its own: an operator supplies the Readium root to accept
// licenses from the Readium License Authority, or their own root to accept
// licenses Helix issues.
//
// CRL is optional on purpose. §7.4 forbids blocking a user because a
// revocation list could not be reached, so an absent CRL downgrades to
// CRLChecked=false on the result rather than to a verification failure.
type CertificatePolicy struct {
	Roots []*x509.Certificate
	CRL   *x509.RevocationList
}

// VerifyReport says what validation actually consulted, so capability
// reporting can state it instead of implying it.
type VerifyReport struct {
	// Mode is the escaping convention whose canonical form the signature
	// matched under.
	Mode CanonicalMode
	// CRLChecked is true only when a revocation list was available.
	CRLChecked bool
	// ProviderCert is the parsed Provider Certificate.
	ProviderCert *x509.Certificate
}

// ProviderCertificate decodes the base64 DER Provider Certificate.
func (l *License) ProviderCertificate() (*x509.Certificate, error) {
	der, err := base64.StdEncoding.DecodeString(l.Signature.Certificate)
	if err != nil {
		return nil, fmt.Errorf("%w: certificate is not valid base64: %v", ErrCertificate, err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCertificate, err)
	}
	return cert, nil
}

// Verify checks the Provider Certificate chain and the document signature,
// following §5.5.1 then §5.5.2 in that order.
//
// The profile is checked first: §3.8 requires signature/algorithm to match the
// algorithm named by encryption/profile, and the Basic Encryption Profile names
// RSA with SHA-256 (§6.3). A license pairing the Basic Profile with another
// signature algorithm is refused rather than verified under it.
func (l *License) Verify(policy CertificatePolicy) (VerifyReport, error) {
	var rep VerifyReport
	if err := l.CheckProfile(); err != nil {
		return rep, err
	}
	cert, err := l.ProviderCertificate()
	if err != nil {
		return rep, err
	}
	if err := l.validateCertificate(cert, policy); err != nil {
		return rep, err
	}
	rep.ProviderCert = cert
	rep.CRLChecked = policy.CRL != nil

	sig, err := base64.StdEncoding.DecodeString(l.Signature.Value)
	if err != nil {
		return rep, fmt.Errorf("%w: signature value is not valid base64: %v", ErrBadSignature, err)
	}
	// §5.5.2 step 4 compares the recalculated signature with the extracted
	// one. Go's RSA and ECDSA verify primitives check that equivalence without
	// the intermediate signature bytes, which is the same predicate and avoids
	// a second, hand-rolled comparison of cryptographic material.
	for _, mode := range []CanonicalMode{CanonicalReference, CanonicalSpec} {
		canonical, err := CanonicalForSigningMode(l.Raw, mode)
		if err != nil {
			return rep, err
		}
		if err := verifySignature(cert, canonical, sig); err == nil {
			rep.Mode = mode
			return rep, nil
		}
	}
	return rep, ErrBadSignature
}

func (l *License) validateCertificate(cert *x509.Certificate, policy CertificatePolicy) error {
	if len(policy.Roots) == 0 {
		return fmt.Errorf("%w: no root certificate configured", ErrUntrustedRoot)
	}
	if policy.CRL != nil {
		for _, revoked := range policy.CRL.RevokedCertificateEntries {
			if revoked.SerialNumber.Cmp(cert.SerialNumber) == 0 {
				return fmt.Errorf("%w: serial %s", ErrRevoked, cert.SerialNumber.Text(16))
			}
		}
	}
	// §5.2.1 and §5.5.1 point 4 are separate windows: valid at issue time, and
	// still unexpired at the last update. Both are anchored on license fields,
	// not on wall-clock time, so a license keeps opening after its provider
	// certificate rolls over.
	signed := false
	for _, root := range policy.Roots {
		if cert.CheckSignatureFrom(root) == nil {
			signed = true
			break
		}
	}
	if !signed {
		return fmt.Errorf("%w: %s", ErrUntrustedRoot, cert.Issuer.CommonName)
	}
	at := l.Issued
	if l.Updated.After(at) {
		at = l.Updated
	}
	if cert.NotBefore.After(l.Issued) || cert.NotAfter.Before(at) {
		return fmt.Errorf("%w: certificate %s .. %s, license %s .. %s",
			ErrCertOutsideValidity,
			cert.NotBefore.Format(time.RFC3339), cert.NotAfter.Format(time.RFC3339),
			l.Issued.Format(time.RFC3339), at.Format(time.RFC3339))
	}
	return nil
}

// verifySignature applies the signature algorithm the Basic Encryption Profile
// names: RSASSA-PKCS1-v1_5 with SHA-256 (§6.3). CheckProfile has already ruled
// out any other algorithm URI before this is reached.
func verifySignature(cert *x509.Certificate, canonical, sig []byte) error {
	pub, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("%w: certificate key is %T, the Basic Profile requires RSA", ErrUnsupportedAlgorithm, cert.PublicKey)
	}
	digest := sha256.Sum256(canonical)
	return rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest[:], sig)
}

// Sign produces the §3.8 signature object for a License Document, over its
// canonical form in the reference convention, with the RSA key that matches
// cert. doc must not already carry a signature member.
func Sign(doc []byte, cert *x509.Certificate, key *rsa.PrivateKey) (Signature, error) {
	if len(cert.Raw) == 0 || cert.PublicKey == nil {
		return Signature{}, fmt.Errorf("%w: certificate carries no DER or public key", ErrCertificate)
	}
	if !publicKeyEquals(cert.PublicKey, &key.PublicKey) {
		return Signature{}, fmt.Errorf("%w: certificate does not belong to the signing key", ErrCertificate)
	}
	canonical, err := canonicalForNewSignature(doc, CanonicalReference)
	if err != nil {
		return Signature{}, err
	}
	digest := sha256.Sum256(canonical)
	raw, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return Signature{}, err
	}
	return Signature{
		Algorithm:   AlgRSASHA256,
		Certificate: base64.StdEncoding.EncodeToString(cert.Raw),
		Value:       base64.StdEncoding.EncodeToString(raw),
	}, nil
}

func publicKeyEquals(a, b crypto.PublicKey) bool {
	ar, aok := a.(*rsa.PublicKey)
	br, bok := b.(*rsa.PublicKey)
	return aok && bok && ar.N.Cmp(br.N) == 0 && ar.E == br.E
}
