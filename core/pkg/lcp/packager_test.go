package lcp

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/helixreader/helix/core/pkg/parse/parseutil"
)

// manifestOPF is a Package Document exercising every §2.1 exclusion expressible
// in a manifest: an EPUB 3 nav document, an EPUB 2 NCX recognised by media type,
// a cover-image property, the OPF2 meta cover form, the legacy guide reference,
// and an item href carrying both a traversal and a fragment so resolution is
// covered alongside exclusion.
const manifestOPF = `<?xml version="1.0"?><package xmlns="http://www.idpf.org/2007/opf" version="3.0">` +
	`<metadata><meta property="dcterms:identifier">urn:isbn:97800000000001</meta>` +
	`<meta name="cover" content="cover-img"/></metadata><manifest>` +
	`<item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>` +
	`<item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>` +
	`<item id="cover-img" href="images/cover.jpeg" media-type="image/jpeg"/>` +
	`<item id="ch1" href="Text/../chapter1.xhtml#start" media-type="application/xhtml+xml"/>` +
	`<item id="ch2" href="chapter2.xhtml" media-type="application/xhtml+xml"/>` +
	`</manifest><spine><itemref idref="ch1"/><itemref idref="ch2"/></spine>` +
	`<guide><reference type="cover" href="titlepage.xhtml"/></guide></package>`

// manifestPublication returns a plain (unprotected) EPUB plus the names Protect
// must leave in the clear and the names it must seal. The track is deliberately
// deflated in the source container: a codec member's ciphertext still has to be
// stored, because §2.2 records compression in the declaration rather than leaving
// a reader to infer it from the zip layer.
func manifestPublication() (entries []zipEntry, clear, sealed []string) {
	text := func(n int) []byte {
		return []byte("<html><body>" + strings.Repeat("<p>Body.</p>", n) + "</body></html>")
	}
	entries = []zipEntry{
		{name: "mimetype", plain: []byte("application/epub+zip")},
		{name: "META-INF/container.xml", plain: []byte(
			`<?xml version="1.0"?><container xmlns="urn:oasis:names:tc:opendocument:xmlns:container">` +
				`<rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>` +
				`</container>`)},
		{name: "OEBPS/content.opf", plain: []byte(manifestOPF)},
		{name: "OEBPS/nav.xhtml", plain: text(30)},
		{name: "OEBPS/titlepage.xhtml", plain: text(8)},
		{name: "OEBPS/toc.ncx", plain: []byte(`<?xml version="1.0"?><ncx xmlns="http://www.daisy.org/z3986/2005/ncx/"><navMap/></ncx>`)},
		{name: "OEBPS/images/cover.jpeg", plain: bytes.Repeat([]byte{0xff, 0xd9, 0x10, 0x20}, 400)},
		{name: "OEBPS/chapter1.xhtml", deflate: true, plain: text(60)},
		{name: "OEBPS/chapter2.xhtml", plain: text(20)},
		{name: "OEBPS/media/track.mp3", deflate: true, plain: bytes.Repeat([]byte("mp3frame"), 900)},
	}
	clear = []string{
		"mimetype", "META-INF/container.xml", "OEBPS/content.opf",
		"OEBPS/nav.xhtml", "OEBPS/toc.ncx", "OEBPS/images/cover.jpeg", "OEBPS/titlepage.xhtml",
	}
	sealed = []string{"OEBPS/chapter1.xhtml", "OEBPS/chapter2.xhtml", "OEBPS/media/track.mp3"}
	return
}

// issueProtect runs the producer half of the flow the way the fulfillment route
// will: Issue the license, Protect the container with the key Issue returned.
func issueProtect(tb testing.TB, c *chain, passphrase string, plain []byte) ([]byte, []byte) {
	tb.Helper()
	doc, key, err := Issue(IssueRequest{
		Provider:   "https://helix.example.org",
		Passphrase: passphrase,
		TextHint:   "The phrase on the receipt",
		Links: []Link{{Rel: "hint", Href: "https://helix.example.org/hint", Type: "text/html"},
			{Rel: "self", Href: "https://helix.example.org/license.lcpl", Type: "application/vnd.readium.lcp.license.v1.0+json"}},
		Issued: mustTime(tb, "2026-11-21T00:00:00Z"),
	}, c.provider, c.providerKey)
	if err != nil {
		tb.Fatalf("Issue: %v", err)
	}
	out, err := Protect(bytes.NewReader(plain), int64(len(plain)), key, doc)
	if err != nil {
		tb.Fatalf("Protect: %v", err)
	}
	return out, doc
}

// zipMember returns one member's stored bytes and its zip-layer method.
func zipMember(tb testing.TB, raw []byte, name string) ([]byte, uint16) {
	tb.Helper()
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		tb.Fatal(err)
	}
	for _, f := range zr.File {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			tb.Fatal(err)
		}
		defer rc.Close()
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(rc); err != nil {
			tb.Fatal(err)
		}
		return buf.Bytes(), f.Method
	}
	tb.Fatalf("no %s in the protected container", name)
	return nil, 0
}

// declaration returns the §2.2 EncryptedData block naming target, so a test can
// read what the producer told readers about one resource.
func declaration(tb testing.TB, protected []byte, target string) string {
	tb.Helper()
	doc, _ := zipMember(tb, protected, EncryptionXMLPath)
	for _, chunk := range strings.Split(string(doc), "</enc:EncryptedData>") {
		if strings.Contains(chunk, `URI="`+target+`"`) {
			return chunk
		}
	}
	tb.Fatalf("encryption.xml declares no resource %s:\n%s", target, doc)
	return ""
}

// TestIssueProtectUnlockRoundTrip is the producer half of HRD-031: a container
// built here must be readable by the reader in this same package, which is the
// only honest way to claim Helix can both issue and serve LCP content.
func TestIssueProtectUnlockRoundTrip(t *testing.T) {
	c := defaultChain(t)
	entries, clear, sealed := manifestPublication()
	plain, want := buildContainer(t, nil, nil, entries...)
	if n := countProtected(entries); n != 0 {
		t.Fatalf("fixture marks %d entries protected; Protect would be testing nothing", n)
	}
	protected, _ := issueProtect(t, c, fixturePassphrase, plain)

	got := open(t, protected)
	if !got.Protected() {
		t.Fatal("the container Protect produced does not report itself protected")
	}
	if err := got.EncryptionErr(); err != nil {
		t.Fatalf("EncryptionErr on a container this package produced: %v", err)
	}
	if miss := got.MissingResources(); len(miss) != 0 {
		t.Errorf("MissingResources = %v; the producer declared resources it did not seal", miss)
	}
	if ps := got.ProtectedResources(); !reflect.DeepEqual(ps, sealed) {
		t.Errorf("ProtectedResources = %v, want %v", ps, sealed)
	}

	// §2.1 is a promise to the reading system, not to the licensee: everything on
	// the exclusion list has to be inspectable before a passphrase exists.
	for _, name := range clear {
		b, err := got.Resource(name, parseutil.MaxEntryBytes)
		if err != nil {
			t.Errorf("Resource(%s) before unlock: %v", name, err)
			continue
		}
		if !bytes.Equal(b, want[name]) {
			t.Errorf("Resource(%s) changed the bytes of an excluded resource", name)
		}
	}
	for _, name := range sealed {
		if _, err := got.Resource(name, parseutil.MaxTextBytes); !errors.Is(err, ErrLocked) {
			t.Errorf("Resource(%s) before unlock error = %v, want ErrLocked", name, err)
		}
	}

	rep, err := got.Unlock(fixturePassphrase, c.policy())
	if err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	if rep.Mode != CanonicalReference {
		t.Errorf("signature verified under mode %v, want the reference convention", rep.Mode)
	}
	if err := func() error { _, e := got.Resource("OEBPS/chapter1.xhtml", parseutil.MaxTextBytes); return e }(); err != nil {
		t.Fatalf("Resource after unlock: %v", err)
	}
	for _, name := range append(append([]string{}, clear...), sealed...) {
		b, err := got.Resource(name, parseutil.MaxTextBytes)
		if err != nil {
			t.Errorf("Resource(%s) after unlock: %v", name, err)
			continue
		}
		if !bytes.Equal(b, want[name]) {
			t.Errorf("Resource(%s) returned %d bytes that differ from the %d Protect sealed",
				name, len(b), len(want[name]))
		}
	}
	// The chapter is the case that separates the compression paths: it was
	// deflated before sealing, so a reader ignoring the Compression element would
	// hand back an inflate stream instead of markup.
	if !bytes.HasPrefix(want["OEBPS/chapter1.xhtml"], []byte("<html>")) {
		t.Error("fixture chapter no longer starts with markup; the round trip proves nothing")
	}
}

// TestProtectDeclaresCompressionAndStoresCiphertext: §2.2 records compression per
// resource, so the zip layer must not also compress. The track arrives deflated
// and is a codec, so it is the member that can only pass by being re-stored.
func TestProtectDeclaresCompressionAndStoresCiphertext(t *testing.T) {
	c := defaultChain(t)
	entries, _, _ := manifestPublication()
	plain, want := buildContainer(t, nil, nil, entries...)
	if _, m := zipMember(t, plain, "OEBPS/media/track.mp3"); m != zip.Deflate {
		t.Fatalf("fixture track is %v in the source container, want Deflate so this test has a method to change", m)
	}
	protected, _ := issueProtect(t, c, fixturePassphrase, plain)

	// OCF requires the mimetype member first and stored, before any change of
	// method is legal.
	zr, err := zip.NewReader(bytes.NewReader(protected), int64(len(protected)))
	if err != nil {
		t.Fatal(err)
	}
	if len(zr.File) == 0 || zr.File[0].Name != "mimetype" || zr.File[0].Method != zip.Store {
		t.Fatalf("first member = %+v, want stored mimetype", zr.File)
	}

	for _, tc := range []struct {
		name   string
		method int
	}{
		{"OEBPS/chapter1.xhtml", deflateMethod},
		{"OEBPS/chapter2.xhtml", deflateMethod},
		{"OEBPS/media/track.mp3", storedMethod},
	} {
		body, zm := zipMember(t, protected, tc.name)
		if zm != zip.Store {
			t.Errorf("%s: ciphertext is stored with zip method %v, want Store", tc.name, zm)
		}
		decl := declaration(t, protected, tc.name)
		if !strings.Contains(decl, `Method="`+strconv.Itoa(tc.method)+`"`) {
			t.Errorf("%s: declaration %s omits Compression Method=%d", tc.name, decl, tc.method)
		}
		if !strings.Contains(decl, `OriginalLength="`+strconv.Itoa(len(want[tc.name]))+`"`) {
			t.Errorf("%s: declaration omits the %d-byte OriginalLength: %s", tc.name, len(want[tc.name]), decl)
		}
		// Ciphertext is an IV, whole blocks, and the declaration is what a reader
		// uses to size the plaintext, so it cannot equal the plaintext length.
		if len(body) == len(want[tc.name]) {
			t.Errorf("%s: sealed body is the same length as the plaintext; nothing was encrypted", tc.name)
		}
	}
}

// TestProtectRefusesContainersItCannotHonestlyReturn: each of these would leave
// the caller holding bytes labelled a protected publication that no reading
// system can open, which is the failure mode §7.4 tells a producer not to ship.
func TestProtectRefusesContainersItCannotHonestlyReturn(t *testing.T) {
	c := defaultChain(t)
	entries, clear, _ := manifestPublication()
	plain, _ := buildContainer(t, nil, nil, entries...)
	_, doc := issueProtect(t, c, fixturePassphrase, plain)
	licensed, _ := buildContainer(t, nil, doc, entries...)
	key := make([]byte, ContentKeyBytes)

	// Every entry of an all-excluded container is a resource §2.1 forbids
	// sealing, so a successful return here would be an encryption.xml declaring
	// nothing.
	onlyClear := make([]zipEntry, 0, len(clear))
	for _, e := range entries {
		for _, n := range clear {
			if e.name == n {
				onlyClear = append(onlyClear, e)
			}
		}
	}
	cleared, _ := buildContainer(t, nil, nil, onlyClear...)
	if len(cleared) == 0 {
		t.Fatal("the all-excluded fixture is empty; the refusal below would prove nothing")
	}

	for _, tc := range []struct {
		name  string
		raw   []byte
		key   []byte
		want  error
		sweep string
	}{
		{"a container that already carries a license", licensed, key, ErrAlreadyProtected, "already"},
		{"a zip that is not a publication", zipWith(t, map[string]string{"readme.txt": "no container.xml here"}), key, ErrNotPublication, "not a publication"},
		{"a publication with nothing left to seal", cleared, key, ErrNoResourcesToProtect, "no protectable"},
		{"a key of the wrong size", plain, key[:ContentKeyBytes-1], ErrContentKeySize, "content key"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := Protect(bytes.NewReader(tc.raw), int64(len(tc.raw)), tc.key, doc)
			if !errors.Is(err, tc.want) {
				t.Fatalf("Protect error = %v, want %v", err, tc.want)
			}
			if out != nil {
				t.Errorf("Protect returned %d bytes alongside the error", len(out))
			}
			if !strings.Contains(strings.ToLower(err.Error()), tc.sweep) {
				t.Errorf("error = %v, want it to mention %q", err, tc.sweep)
			}
		})
	}
}

// TestIssueRefusesIncompleteRequests covers §3's required members. A license
// missing one of these is not a license a reading system can act on, and Issue
// must not discover that at fulfillment time.
func TestIssueRefusesIncompleteRequests(t *testing.T) {
	c := defaultChain(t)
	link := Link{Rel: "hint", Href: "https://helix.example.org/hint"}
	for _, tc := range []struct {
		name string
		req  IssueRequest
		want string
	}{
		{"no provider", IssueRequest{Passphrase: "x", Links: []Link{link}}, "provider"},
		{"no links", IssueRequest{Provider: "https://helix.example.org", Passphrase: "x"}, "links"},
		{"a link without an href", IssueRequest{Provider: "https://helix.example.org", Passphrase: "x",
			Links: []Link{link, {Rel: "self"}}}, "links[1]/href"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, key, err := Issue(tc.req, c.provider, c.providerKey)
			if !errors.Is(err, ErrIncomplete) {
				t.Fatalf("Issue error = %v, want ErrIncomplete", err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %v, want it to name %q", err, tc.want)
			}
			if doc != nil || key != nil {
				t.Error("Issue returned material alongside the refusal")
			}
		})
	}
}

// TestIssueAcceptsAnyPassphrase is §4.2 read literally: "There are no
// restrictions on the length or content of the User Passphrase." An issuer that
// enforced strength would refuse licenses it is required to be able to issue, so
// the empty passphrase and a long non-ASCII one must both produce a working pair.
func TestIssueAcceptsAnyPassphrase(t *testing.T) {
	entries, _, _ := manifestPublication()
	plain, want := buildContainer(t, nil, nil, entries...)
	for _, passphrase := range []string{"", "ᛉ ᚠ ᛚ passphrase with spaces and ✓", strings.Repeat("k", 1024)} {
		t.Run(fmt.Sprintf("len=%d", len([]rune(passphrase))), func(t *testing.T) {
			c := defaultChain(t)
			doc, key, err := Issue(IssueRequest{
				Provider: "https://helix.example.org", Passphrase: passphrase, TextHint: "",
				Links:  []Link{{Rel: "hint", Href: "https://helix.example.org/hint"}},
				Issued: mustTime(t, "2026-11-21T00:00:00Z"),
			}, c.provider, c.providerKey)
			if err != nil {
				t.Fatalf("Issue: %v", err)
			}
			var m map[string]any
			if err := json.Unmarshal(doc, &m); err != nil {
				t.Fatal(err)
			}
			// text_hint is required by §3.4 even when there is nothing to say.
			uk, _ := m["encryption"].(map[string]any)["user_key"].(map[string]any)
			if _, ok := uk["text_hint"]; !ok {
				t.Error("issued license omits user_key/text_hint")
			}
			if _, ok := m["id"].(string); !ok {
				t.Error("issued license has no id")
			}
			protected, err := Protect(bytes.NewReader(plain), int64(len(plain)), key, doc)
			if err != nil {
				t.Fatalf("Protect: %v", err)
			}
			got := open(t, protected)
			if _, err := got.Unlock(passphrase, c.policy()); err != nil {
				t.Fatalf("Unlock with the passphrase Issue sealed for: %v", err)
			}
			if b, err := got.Resource("OEBPS/chapter1.xhtml", parseutil.MaxTextBytes); err != nil || !bytes.Equal(b, want["OEBPS/chapter1.xhtml"]) {
				t.Errorf("Resource after unlock = %v, %v", b, err)
			}
			// A second Container object: Unlock on an already-unlocked one is a
			// no-op, so this would otherwise measure nothing.
			other := open(t, protected)
			if _, err := other.Unlock(passphrase+"x", c.policy()); !errors.Is(err, ErrWrongUserKey) {
				t.Errorf("Unlock with a different passphrase error = %v, want ErrWrongUserKey", err)
			}
		})
	}
}

// TestIssueSealsOnlyRequestedUserFields is §3.7: user/encrypted names the members
// carrying ciphertext, so a document that listed a member it left in the clear
// would misreport what it protects.
func TestIssueSealsOnlyRequestedUserFields(t *testing.T) {
	c := defaultChain(t)
	doc, _, err := Issue(IssueRequest{
		Provider: "https://helix.example.org", Passphrase: fixturePassphrase,
		Links:  []Link{{Rel: "hint", Href: "https://helix.example.org/hint"}},
		User:   User{ID: "user-42", Email: "reader@example.org", Name: "Reader", Encrypted: []string{"email", "name"}},
		Issued: mustTime(t, "2026-11-21T00:00:00Z"),
	}, c.provider, c.providerKey)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	var m struct {
		User struct {
			ID        string   `json:"id"`
			Email     string   `json:"email"`
			Name      string   `json:"name"`
			Encrypted []string `json:"encrypted"`
		} `json:"user"`
	}
	if err := json.Unmarshal(doc, &m); err != nil {
		t.Fatal(err)
	}
	if m.User.ID != "user-42" {
		t.Errorf("user/id = %q, want it left in the clear as requested", m.User.ID)
	}
	for _, f := range []struct{ label, value string }{{"email", m.User.Email}, {"name", m.User.Name}} {
		if f.value == "reader@example.org" || f.value == "Reader" {
			t.Errorf("user/%s was left plaintext although it was listed for encryption", f.label)
		}
		if _, err := base64.StdEncoding.DecodeString(f.value); err != nil {
			t.Errorf("user/%s is not base64 ciphertext: %v", f.label, err)
		}
	}
	if !reflect.DeepEqual(m.User.Encrypted, []string{"email", "name"}) {
		t.Errorf("user/encrypted = %v, want [email name] sorted", m.User.Encrypted)
	}
}
