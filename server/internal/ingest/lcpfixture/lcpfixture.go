// Package lcpfixture builds throwaway LCP provider artifacts for tests: a
// signed License Document and the encrypted EPUB its publication link names.
// It is test support only; nothing in the server imports it outside tests.
package lcpfixture

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/helixreader/helix/core/pkg/lcp"
)

// Passphrase is the User Passphrase every fixture license is sealed for.
const Passphrase = "pasSED1!"

// Chapter is the plaintext of the one encrypted content document.
const Chapter = "<html><body><p>Fulfilled and still sealed.</p></body></html>"

// Fixture is one provider's pair of artifacts.
type Fixture struct {
	Root        *x509.Certificate // trust anchor a reader would embed
	License     []byte            // signed License Document (.lcpl bytes)
	Publication []byte            // encrypted EPUB WITHOUT a license (what the link serves)
	Link        map[string]any    // the publication link as written into License
}

// Build issues and protects a tiny EPUB. href is the publication link target;
// mutate may change the link (length, hash, type, templated ...) before the
// license is signed.
func Build(tb testing.TB, href string, mutate func(link map[string]any)) Fixture {
	tb.Helper()
	root, rootKey := cert(tb, &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Helix Test License Authority"},
		NotBefore: time.Now().AddDate(-1, 0, 0), NotAfter: time.Now().AddDate(10, 0, 0),
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign, BasicConstraintsValid: true, IsCA: true,
	}, nil, nil)
	pkey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		tb.Fatal(err)
	}
	prov, _ := cert(tb, &x509.Certificate{
		SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: "https://helix.example.org"},
		NotBefore: time.Now().AddDate(-1, 0, 0), NotAfter: time.Now().AddDate(5, 0, 0),
		KeyUsage: x509.KeyUsageDigitalSignature,
	}, root, rootKey, pkey)

	doc, key, err := lcp.Issue(lcp.IssueRequest{
		Provider: "https://helix.example.org", Passphrase: Passphrase, TextHint: "The phrase on the receipt",
		Links: []lcp.Link{{Rel: "hint", Href: "https://helix.example.org/hint", Type: "text/html"}},
	}, prov, pkey)
	if err != nil {
		tb.Fatalf("Issue: %v", err)
	}
	plain := plainEPUB(tb)
	protected, err := lcp.Protect(bytes.NewReader(plain), int64(len(plain)), key, doc)
	if err != nil {
		tb.Fatalf("Protect: %v", err)
	}
	pub := stripLicense(tb, protected)
	sum := sha256.Sum256(pub)
	link := map[string]any{"rel": "publication", "href": href, "type": "application/epub+zip", "length": len(pub), "hash": hex.EncodeToString(sum[:])}
	if mutate != nil {
		mutate(link)
	}
	var m map[string]any
	if err := json.Unmarshal(doc, &m); err != nil {
		tb.Fatal(err)
	}
	delete(m, "signature")
	m["links"] = append(m["links"].([]any), link)
	unsigned, _ := json.Marshal(m)
	sig, err := lcp.Sign(unsigned, prov, pkey)
	if err != nil {
		tb.Fatalf("Sign: %v", err)
	}
	m["signature"] = sig
	signed, _ := json.Marshal(m)
	return Fixture{Root: root, License: signed, Publication: pub, Link: link}
}

func cert(tb testing.TB, tmpl, issuer *x509.Certificate, issuerKey *rsa.PrivateKey, subjectKey ...*rsa.PrivateKey) (*x509.Certificate, *rsa.PrivateKey) {
	tb.Helper()
	var key *rsa.PrivateKey
	if len(subjectKey) > 0 {
		key = subjectKey[0]
	} else {
		var err error
		if key, err = rsa.GenerateKey(rand.Reader, 2048); err != nil {
			tb.Fatal(err)
		}
	}
	if issuer == nil {
		issuer, issuerKey = tmpl, key
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, issuer, &key.PublicKey, issuerKey)
	if err != nil {
		tb.Fatal(err)
	}
	c, err := x509.ParseCertificate(der)
	if err != nil {
		tb.Fatal(err)
	}
	return c, key
}

func plainEPUB(tb testing.TB) []byte {
	tb.Helper()
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	add := func(name, body string, method uint16) {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: method})
		if err != nil {
			tb.Fatal(err)
		}
		w.Write([]byte(body))
	}
	add("mimetype", "application/epub+zip", zip.Store)
	add("META-INF/container.xml", `<?xml version="1.0"?><container xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`, zip.Deflate)
	add("OEBPS/content.opf", `<?xml version="1.0"?><package xmlns="http://www.idpf.org/2007/opf" version="3.0"><metadata/><manifest><item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/><item id="c1" href="chapter1.xhtml" media-type="application/xhtml+xml"/></manifest><spine><itemref idref="c1"/></spine></package>`, zip.Deflate)
	add("OEBPS/nav.xhtml", "<html><body><nav/></body></html>", zip.Deflate)
	add("OEBPS/chapter1.xhtml", Chapter, zip.Deflate)
	if err := zw.Close(); err != nil {
		tb.Fatal(err)
	}
	return out.Bytes()
}

func stripLicense(tb testing.TB, raw []byte) []byte {
	tb.Helper()
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		tb.Fatal(err)
	}
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, f := range zr.File {
		if f.Name == lcp.LicensePath {
			continue
		}
		if err := zw.Copy(f); err != nil {
			tb.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		tb.Fatal(err)
	}
	return out.Bytes()
}
