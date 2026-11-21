package lcp

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/helixreader/helix/core/pkg/parse/parseutil"
)

// encDataDeflated declares a resource that was deflated before encryption, so
// §2.2 requires the OCF Compression element carrying its original length.
func encDataDeflated(id, target string, originalLength int) string {
	return `<enc:EncryptedData Id="` + id + `"><enc:EncryptionMethod Algorithm="` + AlgAES256CBC + `"/>` +
		`<ds:KeyInfo><ds:RetrievalMethod URI="` + contentKeyRetrieval + `" Type="` + ContentKeyRetType + `"/></ds:KeyInfo>` +
		`<enc:CipherData><enc:CipherReference URI="` + target + `"/></enc:CipherData>` +
		`<enc:EncryptionProperties><enc:EncryptionProperty xmlns:ns="` + compressionNSName + `">` +
		`<ns:Compression Method="8" OriginalLength="` + strconv.Itoa(originalLength) + `"/>` +
		`</enc:EncryptionProperty></enc:EncryptionProperties></enc:EncryptedData>`
}

// TestContainerRefusesEncryptedExcludedResources covers §2.1. These are not
// passphrase problems: a producer that encrypted them has built a container no
// reading system can open, so the failure belongs at open time.
func TestContainerRefusesEncryptedExcludedResources(t *testing.T) {
	for _, tc := range []struct {
		name       string
		target     string
		wantTarget string
	}{
		{"mimetype is how the container type is read", "mimetype", "mimetype"},
		{"container.xml is how the rootfile is found", "META-INF/container.xml", "META-INF/container.xml"},
		{"the license itself must stay readable", LicensePath, LicensePath},
		// The rootfile is not on the literal list; it is found by reading
		// container.xml, which is what makes this case worth separating.
		{"the package document is an EPUB rootfile", "OEBPS/content.opf", "OEBPS/content.opf"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := encData("ED1", AlgAES256CBC, ContentKeyRetType, contentKeyRetrieval, tc.target)
			// A real license, because OpenContainer correctly refuses to parse a
			// stub one: an unreadable license is a failure, not a convenience.
			f := newFixture(t, nil, publication()...)
			raw := rewriteZipEntry(t, f.raw, EncryptionXMLPath, encXML(doc))
			c := open(t, raw)
			err := c.EncryptionErr()
			if !errors.Is(err, ErrExcludedResourceEncrypted) {
				t.Fatalf("EncryptionErr = %v, want ErrExcludedResourceEncrypted", err)
			}
			if !strings.Contains(err.Error(), tc.wantTarget) {
				t.Errorf("error = %v, want it to name %s", err, tc.wantTarget)
			}
			// Unlock must not paper over a structurally broken container.
			if _, err := c.Unlock(fixturePassphrase, CertificatePolicy{}); err == nil {
				t.Fatal("Unlock succeeded on a container that failed §2.1")
			}
		})
	}
}

// TestContainerRejectsMalformedEncryptionXML: detection already told the caller
// this is an LCP publication, so a broken declaration must be reported rather
// than leaving the container looking unprotected or readable.
func TestContainerRejectsMalformedEncryptionXML(t *testing.T) {
	for _, tc := range []struct {
		name string
		doc  string
	}{
		{"truncated document", encXML(`<enc:EncryptedData Id="ED1">` +
			`<ds:KeyInfo><ds:RetrievalMethod URI="` + contentKeyRetrieval + `" Type="` + ContentKeyRetType + `"/>`)},
		{"wrong root element", `<package xmlns="http://www.idpf.org/2007/opf"><manifest/></package>`},
		{"key uri is not the LCP content key pointer", encXML(encData("ED1", AlgAES256CBC, ContentKeyRetType,
			"license.lcpl#/encryption/user_key", "OEBPS/ch1.html"))},
		{"cipher reference without a target", encXML(encData("ED1", AlgAES256CBC, ContentKeyRetType, contentKeyRetrieval, ""))},
		{"compression method is not a number", encXML(`<enc:EncryptedData Id="ED1">` +
			`<ds:KeyInfo><ds:RetrievalMethod URI="` + contentKeyRetrieval + `" Type="` + ContentKeyRetType + `"/></ds:KeyInfo>` +
			`<enc:CipherData><enc:CipherReference URI="OEBPS/ch1.html"/></enc:CipherData>` +
			`<enc:EncryptionProperties><enc:EncryptionProperty xmlns:ns="` + compressionNSName + `">` +
			`<ns:Compression Method="gzip" OriginalLength="10"/></enc:EncryptionProperty>` +
			`</enc:EncryptionProperties></enc:EncryptedData>`)},
		{"original length is not a number", encXML(`<enc:EncryptedData Id="ED1">` +
			`<ds:KeyInfo><ds:RetrievalMethod URI="` + contentKeyRetrieval + `" Type="` + ContentKeyRetType + `"/></ds:KeyInfo>` +
			`<enc:CipherData><enc:CipherReference URI="OEBPS/ch1.html"/></enc:CipherData>` +
			`<enc:EncryptionProperties><enc:EncryptionProperty xmlns:ns="` + compressionNSName + `">` +
			`<ns:Compression Method="8" OriginalLength="many"/></enc:EncryptionProperty>` +
			`</enc:EncryptionProperties></enc:EncryptedData>`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t, nil, publication()...)
			raw := rewriteZipEntry(t, f.raw, EncryptionXMLPath, tc.doc)
			c := open(t, raw)
			if err := c.EncryptionErr(); err == nil {
				t.Fatal("malformed encryption.xml reported no error")
			} else if !strings.Contains(err.Error(), EncryptionXMLPath) {
				t.Errorf("error = %v, want it to name %s", err, EncryptionXMLPath)
			}
			if _, err := c.Unlock(fixturePassphrase, CertificatePolicy{}); !errors.Is(err, c.EncryptionErr()) {
				t.Errorf("Unlock error = %v, want the encryption.xml failure %v", err, c.EncryptionErr())
			}
		})
	}
}

// TestContainerKeepsCipherReferencesInsideTheContainer: a CipherReference is
// data from an untrusted file. Cleaning it under a virtual root means a traversal
// names something inside the container that does not exist, and the only
// filesystem this package can reach is the zip it was handed.
func TestContainerKeepsCipherReferencesInsideTheContainer(t *testing.T) {
	f := newFixture(t, nil, publication()...)
	for _, tc := range []struct{ written, resolved string }{
		{"../../etc/passwd", "etc/passwd"},
		{"/OEBPS/secret.txt", "OEBPS/secret.txt"},
		{"..\\..\\windows\\win.ini", "windows/win.ini"},
	} {
		doc := encData("ED1", AlgAES256CBC, ContentKeyRetType, contentKeyRetrieval, tc.written)
		raw := rewriteZipEntry(t, f.raw, EncryptionXMLPath, encXML(doc))
		c := open(t, raw)
		if err := c.EncryptionErr(); err != nil {
			t.Fatalf("CipherReference %q: %v", tc.written, err)
		}
		if got := c.MissingResources(); len(got) != 1 || got[0] != tc.resolved {
			t.Errorf("CipherReference %q resolved to %v, want [%s]", tc.written, got, tc.resolved)
		}
		if _, err := c.Unlock(fixturePassphrase, f.chain.policy()); err != nil {
			t.Fatalf("Unlock after rewriting encryption.xml: %v", err)
		}
		if _, err := c.Resource(tc.resolved, parseutil.MaxTextBytes); !errors.Is(err, ErrMissingResource) {
			t.Errorf("Resource(%q) error = %v, want ErrMissingResource", tc.resolved, err)
		}
	}

	// A reference that wanders and comes back is untidy rather than hostile: it
	// must resolve to the resource it names, not be refused or read from
	// somewhere else. The chapter is stored deflated-then-encrypted, so the
	// rewritten declaration has to say so or the comparison below would measure
	// a deflate stream against plaintext.
	const chapter = "OEBPS/chapter1.xhtml"
	benign := encDataDeflated("ED1", "OEBPS/../OEBPS/chapter1.xhtml", len(f.want[chapter]))
	raw := rewriteZipEntry(t, f.raw, EncryptionXMLPath, encXML(benign))
	c := open(t, raw)
	if got := c.MissingResources(); len(got) != 0 {
		t.Fatalf("normalizing a self-returning reference lost a resource that exists: %v", got)
	}
	if _, err := c.Unlock(fixturePassphrase, f.chain.policy()); err != nil {
		t.Fatal(err)
	}
	got, err := c.Resource(chapter, parseutil.MaxTextBytes)
	if err != nil {
		t.Fatalf("Resource: %v", err)
	}
	if !bytes.Equal(got, f.want[chapter]) {
		t.Error("the normalized reference read a different resource")
	}

	// A reference to another origin names nothing this package can open: there is
	// no fetcher here, and inventing a container path out of a URL would read a
	// resource the producer never declared.
	for _, uri := range []string{"http://example.org/ch1.html", "file:///etc/passwd"} {
		doc := encData("ED1", AlgAES256CBC, ContentKeyRetType, contentKeyRetrieval, uri)
		raw := rewriteZipEntry(t, f.raw, EncryptionXMLPath, encXML(doc))
		c := open(t, raw)
		if err := c.EncryptionErr(); !errors.Is(err, ErrLicenseSyntax) {
			t.Errorf("CipherReference %q error = %v, want ErrLicenseSyntax", uri, err)
		}
	}
}

// TestContainerHonoursDeclaredOriginalLength exercises the one integrity signal
// §2.2 provides per resource: the length recorded before compression and
// encryption.
func TestContainerHonoursDeclaredOriginalLength(t *testing.T) {
	f := newFixture(t, nil, publication()...)
	const chapter = "OEBPS/chapter1.xhtml"
	right := len(f.want[chapter])

	// The fixture's own declaration must be the true length, or the mismatch
	// case below could pass because the fixture was already wrong.
	c := f.container
	f.unlock(t)
	if _, err := c.Resource(chapter, parseutil.MaxTextBytes); err != nil {
		t.Fatalf("Resource with a correct declaration: %v", err)
	}

	for _, off := range []int{-1, 1, 4096} {
		doc := encDataDeflated("ED1", chapter, right+off)
		raw := rewriteZipEntry(t, f.raw, EncryptionXMLPath, encXML(doc))
		c := open(t, raw)
		fx := *f
		fx.raw, fx.container = raw, c
		fx.unlock(t)
		_, err := c.Resource(chapter, parseutil.MaxTextBytes)
		if !errors.Is(err, ErrOriginalLengthMismatch) {
			t.Fatalf("OriginalLength %+d: error = %v, want ErrOriginalLengthMismatch", off, err)
		}
		if !strings.Contains(err.Error(), chapter) {
			t.Errorf("error = %v, want it to name the resource", err)
		}
	}
}

func TestContainerRejectsAlgorithmDisagreement(t *testing.T) {
	f := newFixture(t, nil, publication()...)
	// §3.4 binds the two documents to the profile's algorithm; a declaration
	// naming another cipher must not be decrypted under this one.
	doc := encData("ED1", "http://www.w3.org/2001/04/xmlenc#aes128-cbc",
		ContentKeyRetType, contentKeyRetrieval, "OEBPS/chapter1.xhtml")
	raw := rewriteZipEntry(t, f.raw, EncryptionXMLPath, encXML(doc))
	c := open(t, raw)
	_, err := c.Unlock(fixturePassphrase, f.chain.policy())
	if !errors.Is(err, ErrAlgorithmMismatch) {
		t.Fatalf("error = %v, want ErrAlgorithmMismatch", err)
	}
	if c.Unlocked() {
		t.Error("container reports itself unlocked after a rejected algorithm")
	}
}

// TestContainerRefusesATamperedLicenseEndToEnd is §7.4 stated where it is
// enforced: not that Verify would reject, but that a protected publication with
// one altered byte in its signature never yields content.
func TestContainerRefusesATamperedLicenseEndToEnd(t *testing.T) {
	f := newFixture(t, nil, publication()...)
	lic := mustParseTB(t, f.container.License.Raw)
	var m map[string]any
	if err := json.Unmarshal(lic.Raw, &m); err != nil {
		t.Fatal(err)
	}
	sig := m["signature"].(map[string]any)
	rawValue, err := base64.StdEncoding.DecodeString(sig["value"].(string))
	if err != nil {
		t.Fatal(err)
	}
	// Flip one bit of the signature value and re-encode, so the document stays
	// valid JSON and valid base64 and only the cryptographic check can notice.
	rawValue[len(rawValue)-1] ^= 0x01
	sig["value"] = base64.StdEncoding.EncodeToString(rawValue)
	tampered, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(tampered, lic.Raw) {
		t.Fatal("the tamper produced no change; this test would prove nothing")
	}
	raw := rewriteZipEntry(t, f.raw, LicensePath, string(tampered))
	c := open(t, raw)
	if _, err := c.Unlock(fixturePassphrase, f.chain.policy()); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("Unlock error = %v, want ErrBadSignature", err)
	}
	if _, err := c.Resource("OEBPS/chapter1.xhtml", parseutil.MaxTextBytes); !errors.Is(err, ErrLocked) {
		t.Errorf("Resource error = %v, want ErrLocked", err)
	}
}

// TestContainerFirstLicenseEntryWins: a zip may carry duplicate names, and the
// reader hands back both. Keeping the last would let a trailing entry decide
// which license the publication is checked against.
func TestContainerFirstLicenseEntryWins(t *testing.T) {
	f := newFixture(t, nil, publication()...)
	other := defaultChain(t)
	otherDoc, otherKey := buildLicenseTB(t, fixturePassphrase, nil)
	otherSigned := resignTB(t, other, otherDoc)
	// The imposter must be a well-formed license from a different authority,
	// otherwise the test would pass for the wrong reason.
	otherLic := mustParseTB(t, otherSigned)
	if otherLic.ID == f.container.License.ID {
		// buildLicenseTB is deterministic about the id, so distinguish the two
		// chains by key material instead.
		if bytes.Equal(otherKey, f.key) {
			t.Fatal("the two fixtures share a content key; the imposter is not a second license")
		}
	}

	zr, err := zip.NewReader(bytes.NewReader(f.raw), int64(len(f.raw)))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	duplicated := false
	for _, zf := range zr.File {
		rc, err := zf.Open()
		if err != nil {
			t.Fatal(err)
		}
		w, err := zw.CreateHeader(&zip.FileHeader{Name: zf.Name, Method: zf.Method})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.Copy(w, rc); err != nil {
			t.Fatal(err)
		}
		rc.Close()
		if zf.Name == LicensePath {
			w, err := zw.CreateHeader(&zip.FileHeader{Name: LicensePath, Method: zip.Store})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write(otherSigned); err != nil {
				t.Fatal(err)
			}
			duplicated = true
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if !duplicated {
		t.Fatal("no license entry to duplicate; the test proved nothing")
	}

	c := open(t, out.Bytes())
	if c.License == nil {
		t.Fatal("container with two license entries parsed none")
	}
	if err := c.License.CheckPassphrase(fixturePassphrase); err != nil {
		t.Errorf("the license chosen is not the container's own: %v", err)
	}
	if _, err := c.Unlock(fixturePassphrase, f.chain.policy()); err != nil {
		t.Errorf("Unlock with a trailing imposter license: %v", err)
	}
	// Under the imposter's authority the same document must fail, which is how
	// we know the first entry -- not the last -- is the one in force. A fresh
	// container, because Unlock on an already-unlocked one is a no-op: the key
	// is held, and there is nothing left for a second policy to establish.
	if _, err := open(t, out.Bytes()).Unlock(fixturePassphrase, other.policy()); !errors.Is(err, ErrUntrustedRoot) {
		t.Errorf("Unlock under the imposter chain error = %v, want ErrUntrustedRoot", err)
	}
}
