package lcp

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/helixreader/helix/core/pkg/parse/parseutil"
)

// The producer side of LCP: Issue builds and signs a License Document, Protect
// rewrites a publication container so the Content Key seals its resources and
// declares them in META-INF/encryption.xml. Both are meant for licensing
// Helix-owned content under an operator-provided Provider Certificate, and
// enforce nothing beyond what LCP itself enforces.
var (
	// ErrAlreadyProtected refuses to wrap a container that already carries a
	// License Document, which would leave two licenses in one publication.
	ErrAlreadyProtected = errors.New("lcp: publication is already LCP protected")
	// ErrNoResourcesToProtect refuses to emit a container whose encryption.xml
	// would declare nothing, which is a publication that only looks protected.
	ErrNoResourcesToProtect = errors.New("lcp: no protectable resources in the container")
	// ErrNotPublication refuses a container with no META-INF/container.xml, i.e.
	// not an EPUB, so nothing here claims to have protected one.
	ErrNotPublication = errors.New("lcp: container is not a publication")
)

// codecExtensions lists the content types §2.2 says should be stored without
// recompression: deflating an already-compressed image or media stream spends
// production time to save nothing, and costs decode time at playback.
var codecExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
	".bmp": true, ".tif": true, ".tiff": true, ".jxl": true, ".avif": true,
	".mp3": true, ".m4a": true, ".aac": true, ".ogg": true, ".oga": true,
	".opus": true, ".mp4": true, ".m4v": true, ".mov": true, ".webm": true,
	".mpg": true, ".mpeg": true, ".wav": true,
}

func isCodec(name string) bool {
	return codecExtensions[strings.ToLower(path.Ext(name))]
}

// deflate encrypts a stream the way §2.2 recommends for non-codec content:
// compressed first, then sealed, so the ciphertext carries the smaller form and
// the declaration records the plaintext length the inflater must produce.
func deflateEncrypt(key, plain []byte) (ciphertext []byte, err error) {
	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, flate.DefaultCompression)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(plain); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return EncryptCBC(key, buf.Bytes())
}

// manifestExclusions resolves the §2.1 prohibitions that only the Package
// Document can express: nav documents, NCX files, and cover images. A missing or
// unreadable Package Document contributes nothing rather than failing, because
// the container.xml checks have already run by then and a malformed manifest is
// the parser's problem to report, not the exclusion list's.
func (c *Container) manifestExclusions() []string {
	out := []string{}
	for _, rf := range c.rootfiles() {
		f := c.files[rf]
		if f == nil {
			continue
		}
		doc, err := c.raw(f, parseutil.MaxEntryBytes)
		if err != nil {
			continue
		}
		var opf packageManifest
		if err := xml.Unmarshal(doc, &opf); err != nil {
			continue
		}
		base := path.Dir(rf)
		byID := map[string]string{}
		for _, it := range opf.Items {
			byID[it.ID] = it.Href
		}
		add := func(href string) {
			if href == "" {
				return
			}
			out = append(out, resolveOPFHref(base, href))
		}
		for _, it := range opf.Items {
			// The media type identifies an NCX without any property to read,
			// which is how EPUB 2 publications mark it.
			if it.MediaType == "application/x-dtbncx+xml" {
				add(it.Href)
				continue
			}
			for _, token := range strings.Fields(it.Properties) {
				if token == "nav" || token == "cover-image" {
					add(it.Href)
					break
				}
			}
		}
		// EPUB 2 carries no cover-image property; its cover is an item id named
		// by meta, and guide[type=cover] is the older spelling of the same thing.
		for _, m := range opf.Metadata.Metas {
			if !strings.EqualFold(m.Name, "cover") && m.Property != "cover" {
				continue
			}
			id := strings.TrimSpace(m.Value)
			if id == "" {
				id = strings.TrimSpace(m.Text)
			}
			add(byID[id])
		}
		for _, g := range opf.Guide {
			if strings.EqualFold(g.Type, "cover") {
				add(g.Href)
			}
		}
	}
	return out
}

// resolveOPFHref turns a manifest href into a container path. Manifest hrefs are
// relative to the Package Document, and a fragment or query addresses a view of a
// resource rather than a second resource.
func resolveOPFHref(base, href string) string {
	if i := strings.IndexAny(href, "#?"); i >= 0 {
		href = href[:i]
	}
	href = strings.ReplaceAll(href, "\\", "/")
	if strings.HasPrefix(href, "/") {
		return path.Clean(href[1:])
	}
	if base == "." || base == "/" {
		base = ""
	}
	return path.Clean(base + "/" + href)
}

// packageManifest is the subset of the Package Document naming what must stay in
// the clear: the exclusion rules are written in terms of manifest properties, so
// a producer that skipped the manifest would publish a container the reader's
// §2.1 check refuses.
type packageManifest struct {
	Metadata struct {
		Metas []struct {
			Name     string `xml:"name,attr"`
			Property string `xml:"property,attr"`
			// OPF2 spells the value as a content attribute (OPF 2.0.1 section 2.2:
			// <meta name="price" content="USD 19.99"/>); the EPUB 3 meta carries it as
			// text content. name="cover" naming a cover item is community practice
			// rather than normative, and publishing an encrypted cover costs the
			// reader a page image, not the book, so both spellings are accepted.
			Value string `xml:"content,attr"`
			Text  string `xml:",chardata"`
		} `xml:"meta"`
	} `xml:"metadata"`
	Items []struct {
		ID         string `xml:"id,attr"`
		Href       string `xml:"href,attr"`
		MediaType  string `xml:"media-type,attr"`
		Properties string `xml:"properties,attr"`
	} `xml:"manifest>item"`
	Guide []struct {
		Type  string `xml:"type,attr"`
		Title string `xml:"title,attr"`
		Href  string `xml:"href,attr"`
	} `xml:"guide>reference"`
}

// Protect seals a publication's resources under contentKey and returns a new
// container carrying them, the License Document at META-INF/license.lcpl, and a
// §2.2 META-INF/encryption.xml declaring each sealed resource.
//
// Entry order and storage method are preserved for everything left in the clear,
// because OCF requires the mimetype member to stay first and uncompressed.
// Encrypted members are stored rather than deflated by the zip layer: §2.2 puts
// any compression in the declaration, where a reader can act on it without
// guessing whether the container also compressed.
func Protect(r io.ReaderAt, size int64, contentKey, license []byte) ([]byte, error) {
	if len(contentKey) != ContentKeyBytes {
		return nil, fmt.Errorf("%w: got %d", ErrContentKeySize, len(contentKey))
	}
	c, err := OpenContainer(r, size)
	if err != nil {
		return nil, err
	}
	if c.License != nil {
		return nil, ErrAlreadyProtected
	}
	// OCF makes META-INF/container.xml what identifies a publication. Sealing a
	// zip that has none would report a protected EPUB no reading system can open.
	if c.files["META-INF/container.xml"] == nil {
		return nil, ErrNotPublication
	}
	excluded, err := c.excludedResources()
	if err != nil {
		return nil, err
	}

	var decls []string
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, f := range c.zr.File {
		if f.Name == LicensePath || f.Name == EncryptionXMLPath {
			continue // rewritten from the material handed here
		}
		plain, err := c.raw(f, parseutil.MaxEntryBytes)
		if err != nil {
			return nil, fmt.Errorf("lcp: %s: %w", f.Name, err)
		}
		h := &zip.FileHeader{Name: f.Name, Method: f.Method, Modified: f.Modified}
		body := plain
		if !excluded[f.Name] {
			// Whatever the source container did to this member, the ciphertext is
			// stored: §2.2 puts compression in the declaration, where a reader can
			// act on it, and a zip-layer deflator would be invisible to one.
			h.Method = zip.Store
			method := storedMethod
			if isCodec(f.Name) {
				body, err = EncryptCBC(contentKey, plain)
			} else {
				// §2.2 compresses non-codec content before sealing it.
				method = deflateMethod
				body, err = deflateEncrypt(contentKey, plain)
			}
			if err != nil {
				return nil, fmt.Errorf("lcp: %s: %w", f.Name, err)
			}
			decls = append(decls, encryptedData(len(decls)+1, f.Name, method, len(plain)))
		}
		w, err := zw.CreateHeader(h)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(body); err != nil {
			return nil, err
		}
	}
	if len(decls) == 0 {
		return nil, ErrNoResourcesToProtect
	}
	for _, e := range []struct {
		name  string
		bytes []byte
	}{{LicensePath, license}, {EncryptionXMLPath, []byte(encryptionList(decls))}} {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: e.name, Method: zip.Store})
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(e.bytes); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// encryptionList writes the §2.2 document element around EncryptedData members,
// declaring the namespaces the LCP syntax is written in.
func encryptionList(bodies []string) string {
	return `<encryption xmlns="urn:oasis:names:tc:opendocument:xmlns:container"` +
		` xmlns:enc="http://www.w3.org/2001/04/xmlenc#"` +
		` xmlns:ds="http://www.w3.org/2000/09/xmldsig#">` +
		strings.Join(bodies, "") + `</encryption>`
}

// encryptedData declares one sealed resource. The Compression element is written
// for stored streams too: §2.2 requires it for compressed ones and recommends the
// same metadata for the rest, and the original length is the only per-resource
// integrity signal the profile offers.
func encryptedData(index int, target string, method, originalLength int) string {
	return fmt.Sprintf(
		`<enc:EncryptedData Id="ED%d"><enc:EncryptionMethod Algorithm="%s"/>`+
			`<ds:KeyInfo><ds:RetrievalMethod URI="%s" Type="%s"/></ds:KeyInfo>`+
			`<enc:CipherData><enc:CipherReference URI="%s"/></enc:CipherData>`+
			`<enc:EncryptionProperties><enc:EncryptionProperty xmlns:ns="%s">`+
			`<ns:Compression Method="%d" OriginalLength="%d"/>`+
			`</enc:EncryptionProperty></enc:EncryptionProperties></enc:EncryptedData>`,
		index, AlgAES256CBC, contentKeyRetrieval, ContentKeyRetType, xmlAttribute(target),
		compressionNS, method, originalLength)
}

var xmlEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")

func xmlAttribute(s string) string { return xmlEscaper.Replace(s) }

// IssueRequest is what an issuer decides; everything the profile fixes is
// filled in by Issue, and everything §3 requires is refused before anything is
// handed back.
type IssueRequest struct {
	// ID is the license identifier, also what key_check proves a passphrase
	// unlocks. Empty generates a random UUID, which is the usual case.
	ID string
	// Provider is the issuing organization's URI (§3.3).
	Provider string
	// Passphrase is the User Passphrase this license is sealed for. It is never
	// written into the document; only SHA-256 of it is (§4.2).
	Passphrase string
	// TextHint helps a user remember the passphrase (§3.4). The member is
	// required even when there is nothing to say, so an empty string is allowed
	// and omitted hints are not.
	TextHint string
	Links    []Link
	Rights   Rights
	User     User
	// Issued defaults to now.
	Issued time.Time
}

// issuedLicense is the §3 wire form. Dates are written as strings by the issuer
// rather than marshalled from time.Time, because §3.3 requires RFC 3339 and
// time.Time's own form carries a monotonic-clock-free but still varying layout.
type issuedLicense struct {
	ID         string     `json:"id"`
	Issued     string     `json:"issued"`
	Updated    string     `json:"updated,omitempty"`
	Provider   string     `json:"provider"`
	Encryption Encryption `json:"encryption"`
	Links      []Link     `json:"links"`
	// Pointers because omitempty does not apply to a struct value: an absent
	// rights object means unrestricted (§3.6), and "rights": {} would be the
	// document claiming something it does not.
	Rights    *Rights    `json:"rights,omitempty"`
	User      *User      `json:"user,omitempty"`
	Signature *Signature `json:"signature,omitempty"`
}

func randomID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = b[6]&0x0f | 0x40 // version 4
	b[8] = b[8]&0x3f | 0x80 // variant 10
	s := hex.EncodeToString(b[:])
	return s[0:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:32], nil
}

// Issue builds a License Document for the Basic Encryption Profile, seals a fresh
// Content Key for the passphrase in req, and signs the result with the Provider
// Certificate. It returns the license bytes and the Content Key the protected
// publication must be encrypted under, which Protect consumes.
//
// The Content Key is generated here rather than supplied: it is the one value in
// this flow that has no reason to be chosen by a caller, and generating it keeps
// the property that a license and a container produced together always agree.
func Issue(req IssueRequest, cert *x509.Certificate, key *rsa.PrivateKey) (doc []byte, contentKey []byte, err error) {
	if req.Provider == "" {
		return nil, nil, fmt.Errorf("%w: provider", ErrIncomplete)
	}
	if len(req.Links) == 0 {
		return nil, nil, fmt.Errorf("%w: links", ErrIncomplete)
	}
	for i, l := range req.Links {
		if l.Href == "" {
			return nil, nil, fmt.Errorf("%w: links[%d]/href", ErrIncomplete, i)
		}
	}
	if req.ID == "" {
		if req.ID, err = randomID(); err != nil {
			return nil, nil, err
		}
	}
	issued := req.Issued.UTC().Truncate(time.Second)
	if req.Issued.IsZero() {
		issued = time.Now().UTC().Truncate(time.Second)
	}

	contentKey = make([]byte, ContentKeyBytes)
	if _, err := rand.Read(contentKey); err != nil {
		return nil, nil, err
	}
	userKey := UserKey(req.Passphrase)
	encryptedValue, keyCheck, err := SealKeys(userKey, contentKey, req.ID)
	if err != nil {
		return nil, nil, err
	}

	lic := issuedLicense{
		ID:       req.ID,
		Issued:   issued.Format(time.RFC3339),
		Provider: req.Provider,
		Encryption: Encryption{
			Profile: ProfileBasic,
			ContentKey: ContentKeyInfo{
				EncryptedValue: encryptedValue,
				Algorithm:      AlgAES256CBC,
			},
			UserKey: UserKeyInfo{
				TextHint:  req.TextHint,
				Algorithm: AlgSHA256,
				KeyCheck:  keyCheck,
			},
		},
		Links: req.Links,
	}
	if req.Rights != (Rights{}) {
		lic.Rights = &req.Rights
	}
	if req.User.ID != "" || req.User.Email != "" || req.User.Name != "" || len(req.User.Encrypted) > 0 {
		u := req.User
		lic.User = &u
	}
	if lic.User != nil {
		if *lic.User, err = sealUser(*lic.User, userKey); err != nil {
			return nil, nil, err
		}
	}

	unsigned, err := json.MarshalIndent(lic, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	sig, err := Sign(unsigned, cert, key)
	if err != nil {
		return nil, nil, err
	}
	lic.Signature = &sig
	final, err := json.MarshalIndent(lic, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	// Self-check, not ceremony: a document this function hands back is one the
	// reader in this same package has to accept, and finding out otherwise at
	// fulfillment time would be a producer bug reported as a user's problem.
	if _, err := Parse(final); err != nil {
		return nil, nil, fmt.Errorf("lcp: issued license does not parse: %w", err)
	}
	return final, contentKey, nil
}

// sealUser encrypts the user members §3.7 lists in user/encrypted, and narrows
// that list to the members actually carrying a value, so the document it produces
// never claims a member is encrypted that isn't there.
func sealUser(u User, userKey []byte) (User, error) {
	out := User{ID: u.ID, Email: u.Email, Name: u.Name}
	fields := map[string]*string{"id": &out.ID, "email": &out.Email, "name": &out.Name}
	for _, name := range u.Encrypted {
		f, ok := fields[name]
		if !ok || *f == "" {
			continue
		}
		ct, err := EncryptCBC(userKey, []byte(*f))
		if err != nil {
			return out, err
		}
		*f = base64.StdEncoding.EncodeToString(ct)
		out.Encrypted = append(out.Encrypted, name)
	}
	sort.Strings(out.Encrypted)
	return out, nil
}

// excludedResources returns every path §2.1 prohibits encrypting: the literal
// list, the EPUB rootfiles, and the manifest-derived nav documents, NCX files
// and cover images. A container without META-INF/container.xml is not a
// publication, and protecting it would report success for something no reading
// system can open.
func (c *Container) excludedResources() (map[string]bool, error) {
	out := map[string]bool{}
	for name := range neverEncrypted {
		out[name] = true
	}
	for _, name := range c.rootfiles() {
		out[name] = true
	}
	for _, name := range c.manifestExclusions() {
		out[name] = true
	}
	return out, nil
}

// rootfiles lists the Package Documents from META-INF/container.xml, resolved to
// container paths. container.xml spells them relative to the container root, so
// no base-URI resolution is involved.
func (c *Container) rootfiles() []string {
	cf := c.files["META-INF/container.xml"]
	if cf == nil {
		return nil
	}
	doc, err := c.raw(cf, parseutil.MaxEntryBytes)
	if err != nil {
		return nil
	}
	var cont struct {
		Rootfiles []struct {
			FullPath string `xml:"full-path,attr"`
		} `xml:"rootfiles>rootfile"`
	}
	if err := xml.Unmarshal(doc, &cont); err != nil {
		return nil
	}
	out := make([]string, 0, len(cont.Rootfiles))
	for _, rf := range cont.Rootfiles {
		if rf.FullPath != "" {
			out = append(out, rf.FullPath)
		}
	}
	return out
}
