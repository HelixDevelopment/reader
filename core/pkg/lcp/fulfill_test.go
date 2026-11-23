package lcp

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/helixreader/helix/core/pkg/parse/parseutil"
)

// stripLicense returns the container without META-INF/license.lcpl: the file a
// provider serves at the license's publication link (LCP §3.5), which is why
// fulfillment has to put the license back before any reader can unlock it.
func stripLicense(tb testing.TB, raw []byte) []byte {
	tb.Helper()
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		tb.Fatal(err)
	}
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, f := range zr.File {
		if f.Name == LicensePath {
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

// fulfillFixture is a provider's two artifacts: the license document naming the
// publication (with the length and SHA-256 of exactly those bytes) and the
// encrypted publication that lacks a license.
type fulfillFixture struct {
	chain   *chain
	license []byte
	pub     []byte
	want    map[string][]byte
}

// newFulfillFixture builds them through Issue and Protect, then re-signs the
// license with a publication link, since the hash can only be known once the
// ciphertext exists.
func newFulfillFixture(tb testing.TB, mutate func(link map[string]any)) fulfillFixture {
	tb.Helper()
	c := defaultChain(tb)
	entries, _, _ := manifestPublication()
	plain, want := buildContainer(tb, nil, nil, entries...)
	protected, doc := issueProtect(tb, c, fixturePassphrase, plain)
	pub := stripLicense(tb, protected)
	sum := sha256.Sum256(pub)
	link := map[string]any{
		"rel": "publication", "href": "https://helix.example.org/pub.epub",
		"type": "application/epub+zip", "length": len(pub), "hash": hex.EncodeToString(sum[:]),
	}
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
	sig, err := Sign(unsigned, c.provider, c.providerKey)
	if err != nil {
		tb.Fatal(err)
	}
	m["signature"] = sig
	signed, _ := json.Marshal(m)
	return fulfillFixture{chain: c, license: signed, pub: pub, want: want}
}

func (f fulfillFixture) fulfill() ([]byte, error) {
	return Fulfill(f.license, bytes.NewReader(f.pub), int64(len(f.pub)))
}

// TestFulfillEmbedsLicenseAndUnlocks is the consumer half of HRD-031: a license
// plus the publication it names must become a container this package reads.
func TestFulfillEmbedsLicenseAndUnlocks(t *testing.T) {
	f := newFulfillFixture(t, nil)
	if open(t, f.pub).License != nil {
		t.Fatal("fixture publication already carries a license; fulfillment would prove nothing")
	}
	out, err := f.fulfill()
	if err != nil {
		t.Fatalf("Fulfill: %v", err)
	}
	got := open(t, out)
	if got.License == nil {
		t.Fatal("fulfilled container has no license")
	}
	if !bytes.Equal(got.License.Raw, f.license) {
		t.Error("embedded license is not byte-identical to the one received; the signature would no longer verify")
	}
	if err := got.EncryptionErr(); err != nil {
		t.Fatalf("EncryptionErr: %v", err)
	}
	if got.MissingLicense() {
		t.Error("MissingLicense still true after fulfillment")
	}
	if _, err := got.Unlock(fixturePassphrase, f.chain.policy()); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	for name, w := range f.want {
		b, err := got.Resource(name, parseutil.MaxTextBytes)
		if err != nil {
			t.Errorf("Resource(%s): %v", name, err)
			continue
		}
		if !bytes.Equal(b, w) {
			t.Errorf("Resource(%s) differs from the original after fulfillment", name)
		}
	}
	// Everything else must travel untouched: first entry still mimetype.
	zr, _ := zip.NewReader(bytes.NewReader(out), int64(len(out)))
	if zr.File[0].Name != "mimetype" || zr.File[0].Method != zip.Store {
		t.Errorf("first entry = %s/%d, OCF requires an uncompressed mimetype first", zr.File[0].Name, zr.File[0].Method)
	}
}

func TestFulfillRefusesWhatItCannotVouchFor(t *testing.T) {
	good := newFulfillFixture(t, nil)
	flip := func(b []byte) []byte { c := append([]byte{}, b...); c[len(c)/2] ^= 0xff; return c }
	cases := []struct {
		name string
		f    fulfillFixture
		pub  []byte
		want error
	}{
		{"tampered bytes", good, flip(good.pub), ErrPublicationMismatch},
		{"wrong length", newFulfillFixture(t, func(l map[string]any) { l["length"] = 7 }), nil, ErrPublicationMismatch},
		{"wrong hash", newFulfillFixture(t, func(l map[string]any) { l["hash"] = strings.Repeat("0", 64) }), nil, ErrPublicationMismatch},
		{"non-hex hash", newFulfillFixture(t, func(l map[string]any) { l["hash"] = "not-a-hash" }), nil, ErrPublicationMismatch},
		{"no hash", newFulfillFixture(t, func(l map[string]any) { delete(l, "hash") }), nil, ErrPublicationUnverifiable},
		{"no length", newFulfillFixture(t, func(l map[string]any) { delete(l, "length") }), nil, ErrPublicationUnverifiable},
		{"no publication link", newFulfillFixture(t, func(l map[string]any) { l["rel"] = "support" }), nil, ErrNoPublicationLink},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pub := tc.pub
			if pub == nil {
				pub = tc.f.pub
			}
			out, err := Fulfill(tc.f.license, bytes.NewReader(pub), int64(len(pub)))
			if !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
			if out != nil {
				t.Error("a refused fulfillment must return no bytes")
			}
		})
	}
}

func TestFulfillRefusesNonPublications(t *testing.T) {
	f := newFulfillFixture(t, nil)
	// An unprotected EPUB whose hash the license happens to state is still not
	// something this license can unlock.
	entries, _, _ := manifestPublication()
	plain, _ := buildContainer(t, nil, nil, entries...)
	sum := sha256.Sum256(plain)
	g := newFulfillFixture(t, func(l map[string]any) {
		l["hash"] = hex.EncodeToString(sum[:])
		l["length"] = len(plain)
	})
	if _, err := Fulfill(g.license, bytes.NewReader(plain), int64(len(plain))); !errors.Is(err, ErrNotProtected) {
		t.Errorf("unprotected publication: error = %v, want ErrNotProtected", err)
	}
	if _, err := Fulfill([]byte(`{"id":`), bytes.NewReader(f.pub), int64(len(f.pub))); err == nil {
		t.Error("malformed license accepted")
	}
	if _, err := Fulfill(f.license, bytes.NewReader([]byte("PK")), 2); err == nil {
		t.Error("non-zip publication accepted")
	}
}

// TestFulfillNeverRewritesALicensedContainer: the link's hash describes the
// licenseless file the provider serves, so an already-licensed container is not
// that file and no license, same id or not, may be written over its own.
func TestFulfillNeverRewritesALicensedContainer(t *testing.T) {
	f := newFulfillFixture(t, nil)
	once, err := f.fulfill()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Fulfill(f.license, bytes.NewReader(once), int64(len(once))); !errors.Is(err, ErrPublicationMismatch) {
		t.Errorf("licensed container fails the served-bytes hash: error = %v, want ErrPublicationMismatch", err)
	}
	// Even a license whose link vouches for the licensed bytes is refused.
	sum := sha256.Sum256(once)
	g := newFulfillFixture(t, func(l map[string]any) {
		l["hash"] = hex.EncodeToString(sum[:])
		l["length"] = len(once)
	})
	if _, err := Fulfill(g.license, bytes.NewReader(once), int64(len(once))); !errors.Is(err, ErrAlreadyProtected) {
		t.Errorf("error = %v, want ErrAlreadyProtected", err)
	}
}
