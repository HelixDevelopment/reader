// Package lcp implements Readium LCP 1.0 with the Basic Encryption Profile,
// following https://readium.org/lcp-specs/releases/lcp-1.0.html: License
// Document parsing, the canonical form used for signing, RSA-SHA-256
// signature and Provider Certificate validation, User Key derivation,
// Content Key unwrap, and AES-256-CBC decryption of publication resources.
//
// Two scope limits must not be overstated by callers:
//
//   - Trust comes only from the Root Certificate passed to
//     CertificatePolicy. Helix embeds no root, so licenses issued under the
//     Readium License Authority chain do not validate until an operator
//     supplies that root.
//   - Protect and Issue in packager.go are the provider side, meant for
//     licensing Helix-owned content under an operator-provided CA. They
//     enforce nothing beyond what LCP itself enforces.
package lcp

// Encryption profile and algorithm identifiers from §6.3 (Basic Encryption
// Profile 1.0).
const (
	ProfileBasic = "http://readium.org/lcp/basic-profile"

	AlgAES256CBC = "http://www.w3.org/2001/04/xmlenc#aes256-cbc"
	AlgSHA256    = "http://www.w3.org/2001/04/xmlenc#sha256"
	AlgRSASHA256 = "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"
)

// Well-known names inside an LCP-protected container (§2.1, §2.2, §7.1).
const (
	LicensePath         = "META-INF/license.lcpl"
	EncryptionXMLPath   = "META-INF/encryption.xml"
	ContentKeyRetType   = "http://readium.org/2014/01/lcp#EncryptedContentKey"
	contentKeyRetrieval = "license.lcpl#/encryption/content_key"

	// compressionNS is the namespace OCF assigns the Compression element that
	// §2.2 requires for streams deflated before encryption.
	compressionNS = "http://www.idpf.org/2016/encryption#compression"
)

// Sizes fixed by the Basic Encryption Profile.
const (
	ContentKeyBytes = 32 // AES-256
	BlockSize       = 16 // AES block, also the prepended IV length
)
