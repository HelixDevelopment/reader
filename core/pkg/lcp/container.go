package lcp

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/helixreader/helix/core/pkg/parse/parseutil"
)

// Compression method numbers reported by the OCF Compression element (§2.2): 0
// is stored, 8 is deflate. The license and encryption.xml paths and the LCP
// Content Key retrieval constants live in lcp.go.
const (
	storedMethod  = 0
	deflateMethod = 8
)

// encryptionXML is the §2.2 META-INF/encryption.xml document. Only the members
// LCP constrains are decoded; the namespaces are part of the match, so an
// unrelated ds:RetrievalMethod cannot be mistaken for an LCP key reference.
type encryptionXML struct {
	XMLName xml.Name `xml:"urn:oasis:names:tc:opendocument:xmlns:container encryption"`
	Data    []struct {
		ID               string `xml:"Id,attr"`
		EncryptionMethod struct {
			Algorithm string `xml:"Algorithm,attr"`
		} `xml:"http://www.w3.org/2001/04/xmlenc# EncryptionMethod"`
		KeyInfo struct {
			RetrievalMethod struct {
				URI  string `xml:"URI,attr"`
				Type string `xml:"Type,attr"`
			} `xml:"http://www.w3.org/2000/09/xmldsig# RetrievalMethod"`
		} `xml:"http://www.w3.org/2000/09/xmldsig# KeyInfo"`
		CipherData struct {
			CipherReference struct {
				URI string `xml:"URI,attr"`
			} `xml:"http://www.w3.org/2001/04/xmlenc# CipherReference"`
		} `xml:"http://www.w3.org/2001/04/xmlenc# CipherData"`
		Props struct {
			Properties []struct {
				Compression struct {
					Method         string `xml:"Method,attr"`
					OriginalLength string `xml:"OriginalLength,attr"`
				} `xml:"http://www.idpf.org/2016/encryption#compression Compression"`
			} `xml:"http://www.w3.org/2001/04/xmlenc# EncryptionProperty"`
		} `xml:"http://www.w3.org/2001/04/xmlenc# EncryptionProperties"`
	} `xml:"http://www.w3.org/2001/04/xmlenc# EncryptedData"`
}

// resource states what one protected resource's EncryptedData entry declares.
type resource struct {
	// algorithm is EncryptionMethod/@Algorithm, which §3.4 requires agree with
	// the license's content_key/algorithm.
	algorithm string
	// method and originalLength come from the OCF Compression element. A stream
	// compressed before encryption must declare them (§2.2).
	method         int
	hasCompression bool
	originalLength int64
}

// Container is an LCP-protected publication read from a zip container.
//
// The unwrapped Content Key lives in this process's memory only, and only after
// Unlock: §7.2 forbids storing it or the decrypted user fields beside the
// license. Nothing here writes to disk.
type Container struct {
	zr      *zip.Reader
	files   map[string]*zip.File
	License *License

	// protected maps a container path to its declaration. Paths are recorded
	// exactly as CipherReference spells them, resolved against the container
	// root.
	protected map[string]resource
	// keyRefPresent is true when encryption.xml carries an LCP Content Key
	// reference, which is §7.1's second detection means and stays true when the
	// License Document itself is missing.
	keyRefPresent bool
	// encryptionErr records a malformed encryption.xml. Detection has already
	// succeeded at that point, so the container opens and reports the problem
	// rather than looking like an unprotected publication.
	encryptionErr error

	contentKey []byte
	unlocked   bool
}

// OpenContainer reads a publication container and detects LCP protection by
// either means §7.1 allows. An unprotected container is not an error: the caller
// gets a Container with no license and Protected() false, so one entry point
// serves both cases.
func OpenContainer(r io.ReaderAt, size int64) (*Container, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("lcp: open container: %w", err)
	}
	if len(zr.File) > parseutil.MaxArchiveEntries {
		return nil, fmt.Errorf("%w: %d entries", ErrTooManyEntries, len(zr.File))
	}
	c := &Container{zr: zr, files: make(map[string]*zip.File, len(zr.File)), protected: map[string]resource{}}
	for _, f := range zr.File {
		// A zip may carry duplicate names; the reader returns the first, so
		// recording the last would let a later entry shadow the one opened.
		if _, dup := c.files[f.Name]; !dup {
			c.files[f.Name] = f
		}
	}

	if lf, ok := c.files[LicensePath]; ok {
		doc, err := c.raw(lf, parseutil.MaxEntryBytes)
		if err != nil {
			return nil, fmt.Errorf("lcp: read %s: %w", LicensePath, err)
		}
		lic, err := Parse(doc)
		if err != nil {
			return nil, fmt.Errorf("lcp: %s: %w", LicensePath, err)
		}
		c.License = lic
	}
	if ef, ok := c.files[EncryptionXMLPath]; ok {
		raw, err := c.raw(ef, parseutil.MaxEntryBytes)
		if err != nil {
			return nil, fmt.Errorf("lcp: read %s: %w", EncryptionXMLPath, err)
		}
		if err := c.readEncryptionXML(raw); err != nil {
			c.encryptionErr = err
		}
		if c.encryptionErr == nil && c.keyRefPresent {
			// Structural, not cryptographic: a §2.1 violation cannot be repaired
			// with the right passphrase, so it is reported when the container is
			// opened rather than when the key arrives.
			if err := c.checkExclusions(); err != nil {
				c.encryptionErr = err
			}
		}
	}
	return c, nil
}

// readEncryptionXML records which resources §2.2 declares encrypted.
func (c *Container) readEncryptionXML(raw []byte) error {
	var doc encryptionXML
	if err := xml.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("%w: %s: %v", ErrLicenseSyntax, EncryptionXMLPath, err)
	}
	if doc.XMLName.Local != "encryption" {
		return fmt.Errorf("%w: %s: root element is %q", ErrLicenseSyntax, EncryptionXMLPath, doc.XMLName.Local)
	}
	for _, ed := range doc.Data {
		km := ed.KeyInfo.RetrievalMethod
		if km.Type != ContentKeyRetType {
			// Another key mechanism protects this resource; §7.2 says ignore
			// what we do not understand, but not pretend it is readable.
			continue
		}
		if km.URI != contentKeyRetrieval {
			return fmt.Errorf("%w: %s: RetrievalMethod URI %q", ErrLicenseSyntax, EncryptionXMLPath, km.URI)
		}
		target, err := resolveCipherReference(ed.CipherData.CipherReference.URI)
		if err != nil {
			return err
		}
		res := resource{algorithm: ed.EncryptionMethod.Algorithm, method: storedMethod}
		for _, p := range ed.Props.Properties {
			comp := p.Compression
			if comp.Method == "" {
				continue
			}
			m, err := strconv.Atoi(comp.Method)
			if err != nil {
				return fmt.Errorf("%w: %s: Compression Method %q for %s", ErrLicenseSyntax, EncryptionXMLPath, comp.Method, target)
			}
			res.method = m
			res.hasCompression = true
			if comp.OriginalLength != "" {
				n, err := strconv.ParseInt(comp.OriginalLength, 10, 64)
				if err != nil || n < 0 {
					return fmt.Errorf("%w: %s: OriginalLength %q for %s", ErrLicenseSyntax, EncryptionXMLPath, comp.OriginalLength, target)
				}
				res.originalLength = n
			}
		}
		c.protected[target] = res
	}
	c.keyRefPresent = len(c.protected) > 0
	return nil
}

// resolveCipherReference turns a CipherReference URI into a container path.
// §2.2 spells these as plain relative paths; an absolute path or a traversal
// would name something outside the container, which is not a resource to open.
func resolveCipherReference(uri string) (string, error) {
	if uri == "" {
		return "", fmt.Errorf("%w: %s: CipherReference without URI", ErrLicenseSyntax, EncryptionXMLPath)
	}
	if u, err := url.Parse(uri); err == nil && (u.Scheme != "" || u.Host != "") {
		return "", fmt.Errorf("%w: %s: CipherReference %q is not a container path", ErrLicenseSyntax, EncryptionXMLPath, uri)
	}
	// Cleaning under a virtual root is what neutralizes "../": it cannot rise
	// above the root, so the result names a resource inside the container or
	// nothing at all.
	return path.Clean("/" + strings.ReplaceAll(uri, "\\", "/"))[1:], nil
}

// neverEncrypted lists the resources §2.1 and the OCF specification prohibit
// from being encrypted, in the form they appear as container paths. The
// wildcarded entries (EPUB rootfiles, nav documents, NCX, cover images) need the
// Package Document to resolve, so they are checked separately against the
// manifest.
var neverEncrypted = map[string]bool{
	"mimetype":                true,
	"META-INF/container.xml":  true,
	EncryptionXMLPath:         true,
	"META-INF/manifest.xml":   true,
	"META-INF/metadata.xml":   true,
	"META-INF/rights.xml":     true,
	"META-INF/signatures.xml": true,
	LicensePath:               true,
}

var (
	// ErrTooManyEntries reports a container exceeding the archive entry budget the
	// parsers share, checked before any entry is opened.
	ErrTooManyEntries = errors.New("lcp: container has too many entries")
	// ErrNotProtected reports an operation that requires a license on a
	// container that carries none.
	ErrNotProtected = errors.New("lcp: publication is not LCP protected")
	// ErrLocked reports a protected resource requested before Unlock succeeded.
	ErrLocked = errors.New("lcp: resource is encrypted and the license is not unlocked")
	// ErrMissingResource reports an encryption.xml declaration with no matching
	// entry in the container.
	ErrMissingResource = errors.New("lcp: protected resource is absent from the container")
	// ErrAlgorithmMismatch reports encryption.xml naming a different content key
	// algorithm than the License Document (§3.4: both must match the profile).
	ErrAlgorithmMismatch = errors.New("lcp: encryption.xml algorithm does not match the license")
	// ErrExcludedResourceEncrypted reports a §2.1 prohibited resource declared
	// encrypted, which leaves the container unreadable by construction.
	ErrExcludedResourceEncrypted = errors.New("lcp: a resource that must never be encrypted is declared encrypted")
	// ErrMissingLicense reports a container whose encryption.xml references an LCP
	// Content Key but which carries no License Document (§7.1).
	ErrMissingLicense = errors.New("lcp: license document is missing")
	// ErrOriginalLengthMismatch reports a decrypted resource whose length differs
	// from the OriginalLength encryption.xml declared for it.
	ErrOriginalLengthMismatch = errors.New("lcp: decrypted length does not match encryption.xml")
)

// Protected reports LCP protection by either means §7.1 allows: a License
// Document, or an encryption.xml that references an LCP Content Key.
func (c *Container) Protected() bool {
	return c.License != nil || c.keyRefPresent
}

// MissingLicense is §7.1's reportable case: encryption.xml points at an LCP
// Content Key but META-INF/license.lcpl is absent. The user must be told the
// license is missing rather than that the passphrase is wrong.
func (c *Container) MissingLicense() bool { return c.keyRefPresent && c.License == nil }

// EncryptionErr reports a malformed encryption.xml. Detection already
// succeeded, so OpenContainer returns the container; callers surface this
// instead of treating the publication as unprotected.
func (c *Container) EncryptionErr() error { return c.encryptionErr }

// Unlocked reports whether the Content Key is held.
func (c *Container) Unlocked() bool { return c.unlocked }

// ProtectedResources lists the declared encrypted paths in container order.
func (c *Container) ProtectedResources() []string {
	out := make([]string, 0, len(c.protected))
	for _, f := range c.zr.File {
		if _, ok := c.protected[f.Name]; ok {
			out = append(out, f.Name)
		}
	}
	return out
}

// MissingResources lists declared protected resources the container does not
// carry -- a publication that cannot be completed even with the right key.
func (c *Container) MissingResources() []string {
	out := []string{}
	for name := range c.protected {
		if _, ok := c.files[name]; !ok {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// raw reads an entry without decrypting it, under a byte budget. Encrypted
// entries are stored by the producer, but a container that deflates them is
// still read correctly because zip.OpenReader inflates transparently.
func (c *Container) raw(f *zip.File, limit int64) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return parseutil.ReadAll(rc, limit)
}

// checkExclusions verifies the §2.1 prohibition list against what encryption.xml
// declares. The literal names are checked directly; the EPUB rootfiles are read
// out of container.xml, which is itself required to stay plaintext, so a
// container that encrypted it fails here rather than later in the parser.
//
// Nav documents, NCX files and cover images are also on the §2.1 list, and
// packager.go's manifestExclusions resolves the full set for the producer side.
// The reader refuses only what it cannot do without: those three are identified
// through the Package Document, and a publication that encrypted one still
// decrypts and reads, it merely cannot be inspected before a passphrase is
// entered. Rejecting it outright would put a conformance detail between a user
// and a book they own.
func (c *Container) checkExclusions() error {
	for name := range c.protected {
		if neverEncrypted[name] {
			return fmt.Errorf("%w: %s", ErrExcludedResourceEncrypted, name)
		}
	}
	cdata, err := c.files["META-INF/container.xml"].Open()
	if err != nil {
		return nil // Not an EPUB container; nothing further to resolve.
	}
	defer cdata.Close()
	doc, err := parseutil.ReadAll(cdata, parseutil.MaxEntryBytes)
	if err != nil {
		return fmt.Errorf("lcp: META-INF/container.xml: %w", err)
	}
	var container struct {
		Rootfiles []struct {
			FullPath string `xml:"full-path,attr"`
		} `xml:"rootfiles>rootfile"`
	}
	if err := xml.Unmarshal(doc, &container); err != nil {
		return fmt.Errorf("lcp: META-INF/container.xml: %w", err)
	}
	for _, rf := range container.Rootfiles {
		if _, protected := c.protected[rf.FullPath]; protected {
			return fmt.Errorf("%w: %s (EPUB rootfile)", ErrExcludedResourceEncrypted, rf.FullPath)
		}
	}
	return nil
}

// Unlock verifies the License Document and derives the Content Key, which it
// then holds in memory only.
//
// The signature is checked before the passphrase, because §7.4 forbids opening a
// publication with an invalid signature: a wrong key on a forged license must
// not be distinguishable from a right key by reaching the content at all. The
// report says what verification actually consulted, which is what capability
// reporting states rather than implies.
func (c *Container) Unlock(passphrase string, policy CertificatePolicy) (VerifyReport, error) {
	var rep VerifyReport
	if c.encryptionErr != nil {
		return rep, c.encryptionErr
	}
	if c.License == nil {
		if c.keyRefPresent {
			return rep, fmt.Errorf("%w: %s is referenced but absent", ErrMissingLicense, LicensePath)
		}
		return rep, ErrNotProtected
	}
	if c.unlocked {
		return rep, nil
	}
	rep, err := c.License.Verify(policy)
	if err != nil {
		return rep, err
	}
	if err := c.License.CheckPassphrase(passphrase); err != nil {
		return rep, err
	}
	key, err := c.License.ContentKey(UserKey(passphrase))
	if err != nil {
		return rep, err
	}
	for name, res := range c.protected {
		if res.algorithm != "" && res.algorithm != c.License.Encryption.ContentKey.Algorithm {
			return rep, fmt.Errorf("%w: %s declares %s, license declares %s",
				ErrAlgorithmMismatch, name, res.algorithm, c.License.Encryption.ContentKey.Algorithm)
		}
	}
	c.contentKey, c.unlocked = key, true
	return rep, nil
}

// Resource returns a publication resource as plaintext. An unprotected entry is
// read as stored; a declared-encrypted one is decrypted, and inflated when
// §2.2's Compression metadata says it was deflated before encryption. limit
// bounds the returned bytes, so a container cannot ask for more than the caller
// intends to hold.
func (c *Container) Resource(name string, limit int64) ([]byte, error) {
	f, ok := c.files[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrMissingResource, name)
	}
	res, protected := c.protected[name]
	if !protected {
		return c.raw(f, limit)
	}
	if !c.unlocked {
		return nil, fmt.Errorf("%w: %s", ErrLocked, name)
	}
	// Ciphertext is the IV, the body, and at most one padding block.
	ct, err := c.raw(f, limit+2*BlockSize)
	if err != nil {
		return nil, err
	}
	plain, err := DecryptCBC(c.contentKey, ct)
	if err != nil {
		return nil, fmt.Errorf("lcp: %s: %w", name, err)
	}
	if res.method == deflateMethod {
		rc := flate.NewReader(bytes.NewReader(plain))
		defer rc.Close()
		if plain, err = parseutil.ReadAll(rc, limit); err != nil {
			return nil, fmt.Errorf("lcp: %s: inflate: %w", name, err)
		}
	}
	// The caller's budget covers plaintext. Only this path can overshoot it: the
	// zip reader and the inflater are bounded as they read, while decryption
	// yields the whole plaintext before anything counts it.
	if int64(len(plain)) > limit {
		return nil, fmt.Errorf("lcp: %s: plaintext exceeds %d byte limit", name, limit)
	}
	// The declared length is the one integrity signal available without a
	// digest; OCF requires it for streams compressed before encryption.
	if res.originalLength > 0 && int64(len(plain)) != res.originalLength {
		return nil, fmt.Errorf("%w: %s: %d bytes, encryption.xml declares %d",
			ErrOriginalLengthMismatch, name, len(plain), res.originalLength)
	}
	return plain, nil
}
