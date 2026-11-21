package lcp

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"testing"

	"github.com/helixreader/helix/core/pkg/parse/parseutil"
)

// compressionNSName is the namespace OCF assigns the Compression element that
// §2.2 requires for streams deflated before encryption.
const compressionNSName = "http://www.idpf.org/2016/encryption#compression"

// zipEntry is one container member. plain is what the resource must read back
// as; when protected it is deflated (if deflate), encrypted with the Content
// Key, and stored uncompressed.
type zipEntry struct {
	name      string
	plain     []byte
	deflate   bool
	protected bool
}

// buildContainer writes a publication container: entries, the License Document
// at META-INF/license.lcpl when non-nil, and a §2.2 encryption.xml declaring
// every protected entry. It returns the container bytes and a name-to-plaintext
// map for the caller to assert against.
//
// contentKey must be the same key the license wraps, which protectedFixture
// arranges. Writing encryption.xml here rather than accepting a string keeps the
// fixture structurally what a producer emits, including the Compression element.
func buildContainer(tb testing.TB, contentKey, licenseDoc []byte, entries ...zipEntry) ([]byte, map[string][]byte) {
	tb.Helper()
	want := map[string][]byte{}
	body := map[string][]byte{}
	var decl bytes.Buffer
	decl.WriteString(`<encryption xmlns="urn:oasis:names:tc:opendocument:xmlns:container"` +
		` xmlns:enc="http://www.w3.org/2001/04/xmlenc#"` +
		` xmlns:ds="http://www.w3.org/2000/09/xmldsig#">`)
	any := false
	for i, e := range entries {
		want[e.name] = e.plain
		data := e.plain
		if e.protected {
			if e.deflate {
				var buf bytes.Buffer
				w, err := flate.NewWriter(&buf, flate.DefaultCompression)
				if err != nil {
					tb.Fatal(err)
				}
				if _, err := w.Write(e.plain); err != nil {
					tb.Fatal(err)
				}
				if err := w.Close(); err != nil {
					tb.Fatal(err)
				}
				data = buf.Bytes()
			}
			enc, err := EncryptCBC(contentKey, data)
			if err != nil {
				tb.Fatal(err)
			}
			data = enc
			any = true
			fmt.Fprintf(&decl, `<enc:EncryptedData Id="ED%d"><enc:EncryptionMethod Algorithm="%s"/>`+
				`<ds:KeyInfo><ds:RetrievalMethod URI="%s" Type="%s"/></ds:KeyInfo>`+
				`<enc:CipherData><enc:CipherReference URI="%s"/></enc:CipherData>`,
				i+1, AlgAES256CBC, contentKeyRetrieval, ContentKeyRetType, e.name)
			if e.deflate {
				fmt.Fprintf(&decl, `<enc:EncryptionProperties><enc:EncryptionProperty xmlns:ns="%s">`+
					`<ns:Compression Method="8" OriginalLength="%d"/></enc:EncryptionProperty>`+
					`</enc:EncryptionProperties>`, compressionNSName, len(e.plain))
			}
			decl.WriteString(`</enc:EncryptedData>`)
		}
		body[e.name] = data
	}
	decl.WriteString(`</encryption>`)

	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	// ordered so a duplicate-name test can rely on which entry the reader keeps
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		method := zip.Store
		if !e.protected && e.deflate {
			method = zip.Deflate
		}
		w, err := zw.CreateHeader(&zip.FileHeader{Name: e.name, Method: method})
		if err != nil {
			tb.Fatal(err)
		}
		if _, err := w.Write(body[e.name]); err != nil {
			tb.Fatal(err)
		}
		names = append(names, e.name)
	}
	if licenseDoc != nil {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: LicensePath, Method: zip.Store})
		if err != nil {
			tb.Fatal(err)
		}
		if _, err := w.Write(licenseDoc); err != nil {
			tb.Fatal(err)
		}
	}
	if any {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: EncryptionXMLPath, Method: zip.Store})
		if err != nil {
			tb.Fatal(err)
		}
		if _, err := w.Write(decl.Bytes()); err != nil {
			tb.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		tb.Fatal(err)
	}
	return out.Bytes(), want
}

// resignTB signs an already-built License Document with c's provider key,
// replacing the placeholder signature buildLicenseTB leaves in place. It cannot
// simply rebuild the fixture: the Content Key is generated per build, so the
// bytes a test encrypts resources under must come from the same build.
func resignTB(tb testing.TB, c *chain, doc []byte) []byte {
	tb.Helper()
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
	return final
}

const fixturePassphrase = "pasSED1!"

// fixture is a protected publication together with everything needed to unlock
// it. raw is kept so a test can mutate the container bytes and reopen.
type fixture struct {
	raw        []byte
	want       map[string][]byte
	key        []byte
	passphrase string
	chain      *chain
	container  *Container
}

func newFixture(tb testing.TB, mutate func(map[string]any), entries ...zipEntry) *fixture {
	tb.Helper()
	c := defaultChain(tb)
	doc, key := buildLicenseTB(tb, fixturePassphrase, mutate)
	lic := mustParseTB(tb, resignTB(tb, c, doc))
	raw, want := buildContainer(tb, key, lic.Raw, entries...)
	f := &fixture{raw: raw, want: want, key: key, passphrase: fixturePassphrase, chain: c}
	f.reopen(tb)
	return f
}

// reopen re-reads the container bytes, which is also how a test picks up a
// mutation applied to f.raw.
func (f *fixture) reopen(tb testing.TB) *Container {
	tb.Helper()
	c, err := OpenContainer(bytes.NewReader(f.raw), int64(len(f.raw)))
	if err != nil {
		tb.Fatalf("OpenContainer: %v", err)
	}
	f.container = c
	return c
}

func (f *fixture) unlock(tb testing.TB) VerifyReport {
	tb.Helper()
	rep, err := f.container.Unlock(f.passphrase, f.chain.policy())
	if err != nil {
		tb.Fatalf("Unlock: %v", err)
	}
	return rep
}

// publication returns the entry set an LCP-protected EPUB carries: the §2.1
// never-encrypted members in the clear, a chapter deflated then encrypted, and a
// codec image encrypted without recompression.
func publication() []zipEntry {
	return []zipEntry{
		{name: "mimetype", plain: []byte("application/epub+zip")},
		{name: "META-INF/container.xml", plain: []byte(
			`<?xml version="1.0"?><container xmlns="urn:oasis:names:tc:opendocument:xmlns:container">` +
				`<rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>` +
				`</container>`)},
		{name: "OEBPS/content.opf", plain: []byte(
			`<?xml version="1.0"?><package xmlns="http://www.idpf.org/2007/opf"><manifest/></package>`)},
		{name: "OEBPS/chapter1.xhtml", deflate: true, protected: true, plain: []byte(
			"<html><body>" + strings.Repeat("<p>In which the container layer is exercised.</p>", 40) + "</body></html>")},
		{name: "OEBPS/image.jpeg", protected: true, plain: bytes.Repeat([]byte{0xff, 0xd8, 0x00, 0x42}, 500)},
	}
}

// zipWith writes a container from literal bytes, for cases buildContainer will
// not produce: malformed metadata, hostile paths, wrong algorithms.
func zipWith(tb testing.TB, files map[string]string) []byte {
	tb.Helper()
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	// Writer order decides which duplicate name the reader keeps, so it must not
	// come from a map range.
	sort.Strings(names)
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, n := range names {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: n, Method: zip.Store})
		if err != nil {
			tb.Fatal(err)
		}
		if _, err := w.Write([]byte(files[n])); err != nil {
			tb.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		tb.Fatal(err)
	}
	return out.Bytes()
}

// dropZipEntry rebuilds a container without one entry, leaving every other
// member's bytes and storage method intact.
func dropZipEntry(tb testing.TB, raw []byte, name string) []byte {
	tb.Helper()
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		tb.Fatal(err)
	}
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	kept := 0
	for _, f := range zr.File {
		if f.Name == name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			tb.Fatal(err)
		}
		w, err := zw.CreateHeader(&zip.FileHeader{Name: f.Name, Method: f.Method})
		if err != nil {
			rc.Close()
			tb.Fatal(err)
		}
		_, cpErr := io.Copy(w, rc)
		rc.Close()
		if cpErr != nil {
			tb.Fatal(cpErr)
		}
		kept++
	}
	if kept == len(zr.File) {
		tb.Fatalf("%s is not in the container; dropping it tested nothing", name)
	}
	if err := zw.Close(); err != nil {
		tb.Fatal(err)
	}
	return out.Bytes()
}

func open(tb testing.TB, raw []byte) *Container {
	tb.Helper()
	c, err := OpenContainer(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		tb.Fatalf("OpenContainer: %v", err)
	}
	return c
}

func TestContainerUnlocksAndReadsEveryResource(t *testing.T) {
	f := newFixture(t, nil, publication()...)
	c := f.container
	if !c.Protected() {
		t.Fatal("a container carrying a license and protected resources is not protected")
	}
	if c.EncryptionErr() != nil {
		t.Fatalf("EncryptionErr on a well-formed container: %v", c.EncryptionErr())
	}
	if got := c.ProtectedResources(); len(got) != 2 {
		t.Errorf("ProtectedResources = %v, want the chapter and the image", got)
	}
	if got := c.MissingResources(); len(got) != 0 {
		t.Errorf("MissingResources = %v, want none", got)
	}

	// §2.1 exists so a reading system can inspect these before asking for a
	// passphrase, so what matters is that they read while locked.
	for _, name := range []string{"mimetype", "META-INF/container.xml", "OEBPS/content.opf"} {
		got, err := c.Resource(name, parseutil.MaxEntryBytes)
		if err != nil {
			t.Fatalf("Resource(%s) before unlock: %v", name, err)
		}
		if !bytes.Equal(got, f.want[name]) {
			t.Errorf("Resource(%s) = %q", name, got)
		}
	}
	for _, name := range []string{"OEBPS/chapter1.xhtml", "OEBPS/image.jpeg"} {
		if _, err := c.Resource(name, parseutil.MaxTextBytes); !errors.Is(err, ErrLocked) {
			t.Errorf("Resource(%s) before unlock error = %v, want ErrLocked", name, err)
		}
	}

	rep := f.unlock(t)
	if rep.Mode != CanonicalReference {
		t.Errorf("signature verified under mode %v, want the reference convention", rep.Mode)
	}
	if !c.Unlocked() {
		t.Fatal("Unlocked() is false after a successful Unlock")
	}
	for name, want := range f.want {
		got, err := c.Resource(name, parseutil.MaxTextBytes)
		if err != nil {
			t.Fatalf("Resource(%s): %v", name, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("Resource(%s) returned %d bytes that differ from the %d stored", name, len(got), len(want))
		}
	}
	// The chapter is the case that separates the two compression paths: it was
	// deflated before encryption, so a reader that ignored §2.2's Compression
	// element would hand back an inflate stream instead of markup.
	if !bytes.HasPrefix(f.want["OEBPS/chapter1.xhtml"], []byte("<html>")) {
		t.Error("fixture chapter no longer starts with markup; the round trip proves nothing")
	}
}

// TestContainerCiphertextBudgetCoversEncryptionOverhead: §2.2 puts an IV ahead
// of the body and pads the end, so the ciphertext of a stored resource is longer
// than the plaintext a caller asked to bound. Budgeting the read at the caller's
// limit would refuse a legal resource for being a block or two bigger than the
// thing it decrypts to; ignoring the limit entirely would hand back more than the
// caller said it could hold.
//
// The two assertions need different resources. A plaintext whose length is a
// multiple of the block size gains a full padding block, which is already enough
// to overrun the ciphertext budget, so the overshoot would be reported by the
// reader rather than by the plaintext check. The dedicated entry is three bytes
// short of a block boundary, leaving padding small enough that only the
// plaintext measure can notice.
func TestContainerCiphertextBudgetCoversEncryptionOverhead(t *testing.T) {
	const image = "OEBPS/image.jpeg"
	const short = "OEBPS/payload.bin"
	plain := bytes.Repeat([]byte{0x7f}, 1997)
	f := newFixture(t, nil, append(publication(), zipEntry{name: short, protected: true, plain: plain})...)
	f.unlock(t)

	if _, err := f.container.Resource(image, int64(len(f.want[image]))); err != nil {
		t.Errorf("Resource(%s) at exactly %d bytes: %v", image, len(f.want[image]), err)
	}
	if got, err := f.container.Resource(short, int64(len(plain))); err != nil {
		t.Fatalf("Resource(%s) at exactly %d bytes: %v", short, len(plain), err)
	} else if !bytes.Equal(got, plain) {
		t.Errorf("Resource(%s) returned different bytes than the container stored", short)
	}
	if got, err := f.container.Resource(short, int64(len(plain))-1); err == nil {
		t.Errorf("Resource returned %d bytes against a %d byte limit", len(got), int64(len(plain))-1)
	}
}

func TestContainerUnprotectedPublicationIsNotAnError(t *testing.T) {
	entries := unprotected(publication())
	if n := countProtected(entries); n != 0 {
		t.Fatalf("fixture still marks %d entries protected", n)
	}
	raw, want := buildContainer(t, nil, nil, entries...)
	c := open(t, raw)
	if c.Protected() {
		t.Error("an unprotected container reports itself as protected")
	}
	got, err := c.Resource("OEBPS/chapter1.xhtml", parseutil.MaxTextBytes)
	if err != nil {
		t.Fatalf("Resource on an unprotected entry: %v", err)
	}
	if !bytes.Equal(got, want["OEBPS/chapter1.xhtml"]) {
		t.Error("unprotected entry did not round-trip through the zip deflator")
	}
	if _, err := c.Unlock("x", CertificatePolicy{}); !errors.Is(err, ErrNotProtected) {
		t.Errorf("Unlock error = %v, want ErrNotProtected", err)
	}
}

// TestContainerDetectsByEachMeans covers §7.1: either the License Document or an
// LCP Content Key reference is sufficient, and the second without the first is
// the case the specification asks be reported to the user.
func TestContainerDetectsByEachMeans(t *testing.T) {
	f := newFixture(t, nil, publication()...)
	if !f.container.Protected() || f.container.MissingLicense() {
		t.Fatalf("complete container: protected=%v missingLicense=%v",
			f.container.Protected(), f.container.MissingLicense())
	}

	withoutLicense := dropZipEntry(t, f.raw, LicensePath)
	c := open(t, withoutLicense)
	if !c.Protected() {
		t.Error("encryption.xml referencing an LCP Content Key did not register as protection")
	}
	if !c.MissingLicense() {
		t.Error("MissingLicense() is false although the license is absent")
	}
	if _, err := c.Unlock(fixturePassphrase, f.chain.policy()); !errors.Is(err, ErrMissingLicense) {
		t.Errorf("Unlock error = %v, want ErrMissingLicense", err)
	}

	// A key reference naming some other mechanism is not LCP, and must not be
	// reported as a license problem a passphrase could fix.
	foreign := zipWith(t, map[string]string{
		EncryptionXMLPath: encXML(encData("ED1", AlgAES256CBC,
			"http://example.org/2014/01/other#ContentKey", contentKeyRetrieval, "OEBPS/ch1.html")),
	})
	c = open(t, foreign)
	if c.Protected() {
		t.Error("a non-LCP RetrievalMethod Type registered as LCP protection")
	}
}

func TestContainerWrongPassphraseLeavesResourcesLocked(t *testing.T) {
	f := newFixture(t, nil, publication()...)
	if _, err := f.container.Unlock("wrong", f.chain.policy()); !errors.Is(err, ErrWrongUserKey) {
		t.Fatalf("Unlock error = %v, want ErrWrongUserKey", err)
	}
	if f.container.Unlocked() {
		t.Fatal("Unlocked() is true after a rejected passphrase")
	}
	if _, err := f.container.Resource("OEBPS/chapter1.xhtml", parseutil.MaxTextBytes); !errors.Is(err, ErrLocked) {
		t.Errorf("Resource error = %v, want ErrLocked", err)
	}
	// With no root configured, verification fails before the passphrase is
	// consulted, and that must not be reported as a bad passphrase.
	if _, err := f.container.Unlock(fixturePassphrase, CertificatePolicy{}); !errors.Is(err, ErrUntrustedRoot) {
		t.Errorf("Unlock with no trust anchor error = %v, want ErrUntrustedRoot", err)
	}
}
