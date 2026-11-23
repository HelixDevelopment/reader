package lcp

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"strings"
)

var (
	// ErrNoPublicationLink reports a License Document with no link carrying the
	// "publication" relation (§3.5), so there is nothing to fulfill.
	ErrNoPublicationLink = errors.New("lcp: license has no publication link")
	// ErrPublicationUnverifiable reports a publication link without the length
	// or hash that would let a reader prove the bytes it fetched are the ones the
	// provider named. Fulfillment fails closed rather than trusting the transport.
	ErrPublicationUnverifiable = errors.New("lcp: publication link states no length or hash to verify against")
	// ErrPublicationMismatch reports bytes that do not match the link's length or
	// SHA-256.
	ErrPublicationMismatch = errors.New("lcp: publication does not match the license's publication link")
	// ErrUnsupportedFormat reports a publication link whose media type is not
	// EPUB. PDF, audiobook and Divina LCP packages carry their license at the
	// package root (license.lcpl) rather than META-INF/license.lcpl; Helix does
	// not place licenses there and refuses rather than produce a package that
	// would not unlock.
	ErrUnsupportedFormat = errors.New("lcp: unsupported publication format")
	// ErrTemplatedHref reports a publication link that is a URI Template
	// (templated: true). Helix does not expand templates, so it cannot name the
	// bytes the link describes.
	ErrTemplatedHref = errors.New("lcp: publication link is a URI template")
)

// PublicationLink returns the link whose relation is "publication".
func (l *License) PublicationLink() (Link, bool) { return l.Link("publication") }

const epubType = "application/epub+zip"

type fulfillOptions struct{ allowUnverified bool }

// FulfillOption adjusts Fulfill and FulfillTo.
type FulfillOption func(*fulfillOptions)

// AllowUnverified lets fulfillment proceed when the publication link states no
// length and no hash. This DIVERGES from the fail-closed default: the spec
// makes both optional ("should verify if provided"), so a conforming provider
// may omit them, but without them nothing proves the bytes are the ones the
// provider named. Whatever the link does state is still enforced: a stated
// length must equal the size, a stated hash must match, and a stated hash that
// cannot be decoded is a mismatch. A link stating only one of the two passes
// when that one checks out.
func AllowUnverified() FulfillOption { return func(o *fulfillOptions) { o.allowUnverified = true } }

// decodeLinkHash reads a link hash. The §3.5 table and JSON schema specify
// base64; spec Example 3 and the reference server (readium-lcp-server) emit
// lowercase hex SHA-256. Hex is tried first and must decode to 32 bytes
// (a 32-byte digest is 64 hex chars but 44 base64 chars, so the forms cannot be
// confused), otherwise standard base64 decoding to 32 bytes.
func decodeLinkHash(s string) ([]byte, error) {
	if b, err := hex.DecodeString(s); err == nil && len(b) == sha256.Size {
		return b, nil
	}
	if b, err := base64.StdEncoding.DecodeString(s); err == nil && len(b) == sha256.Size {
		return b, nil
	}
	return nil, fmt.Errorf("%w: hash is neither a hex nor a base64 SHA-256", ErrPublicationMismatch)
}

// unreadableErr explains why a freshly assembled container is not usable, or
// returns nil when it is.
func unreadableErr(c *Container) error {
	if c.License == nil {
		return errors.New("lcp: fulfilled container carries no license")
	}
	if err := c.EncryptionErr(); err != nil {
		return fmt.Errorf("lcp: fulfilled container is not readable: %w", err)
	}
	return nil
}

// Fulfill completes the consumer half of LCP delivery. A provider hands out a
// License Document whose "publication" link names an encrypted publication
// that carries no license (§3.5); the reading system downloads it, checks it,
// and puts the license at META-INF/license.lcpl so a Container can unlock it.
//
// The caller performs the download (network policy, redirects and credentials
// are not this package's business) and passes the bytes here. By default
// Fulfill refuses unless the link states a length and a SHA-256 (hex, either
// case, or base64) and the bytes match both; AllowUnverified relaxes that. It
// also refuses non-EPUB link types (ErrUnsupportedFormat), templated links
// (ErrTemplatedHref), and containers that do not declare LCP protection. The
// license is embedded byte-for-byte, because its signature is over those bytes.
//
// WARNING: Fulfill does NOT verify the license signature, the certificate
// chain, or the passphrase, and does not unlock anything. A fulfilled container
// is only as trustworthy as the License Document it was given; use
// Container.Unlock, where the trust root and passphrase are supplied, before
// treating any content as authentic.
//
// Fulfill buffers the result in memory and re-opens it to confirm a reader will
// see the embedded license. Use FulfillTo to stream.
func Fulfill(licenseDoc []byte, publication io.ReaderAt, size int64, opts ...FulfillOption) ([]byte, error) {
	var out bytes.Buffer
	if err := FulfillTo(&out, licenseDoc, publication, size, opts...); err != nil {
		return nil, err
	}
	// Prove the result is what a reader will see, not what was intended.
	res, err := OpenContainer(bytes.NewReader(out.Bytes()), int64(out.Len()))
	if err != nil {
		return nil, err
	}
	if err := unreadableErr(res); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// FulfillTo is Fulfill streaming the fulfilled container to w. Every check on
// the link, the length and the hash completes before the first byte is written,
// so a refused publication leaves w untouched.
//
// Hashing cannot happen while copying: entries are copied as raw compressed
// streams from the central directory, which does not touch every source byte
// (local headers' gaps, trailing data), so the digest is computed in a
// dedicated pass first. The publication must therefore be immutable for the
// duration of the call. That is checked, not assumed: the digest is computed
// again after the copy, a source shorter than size is an error, and a second
// digest that differs returns ErrPublicationMismatch. In that case, and on any
// error after writing began, w holds a partial or untrustworthy result and the
// caller must discard it.
//
// FulfillTo does not read w back: a bare io.Writer cannot be re-read. Callers
// whose destination supports it can run OpenContainer on it; Fulfill does so on
// its in-memory buffer. Like Fulfill, it does not verify the license signature.
func FulfillTo(w io.Writer, licenseDoc []byte, publication io.ReaderAt, size int64, opts ...FulfillOption) error {
	var cfg fulfillOptions
	for _, o := range opts {
		o(&cfg)
	}
	lic, err := Parse(licenseDoc)
	if err != nil {
		return err
	}
	link, ok := lic.PublicationLink()
	if !ok {
		return ErrNoPublicationLink
	}
	if link.Templated {
		// The href is deliberately not echoed: it may carry a bearer token.
		return ErrTemplatedHref
	}
	if link.Type != "" {
		mt, _, err := mime.ParseMediaType(link.Type)
		if err != nil || !strings.EqualFold(mt, epubType) {
			return fmt.Errorf("%w: %q (only %s)", ErrUnsupportedFormat, link.Type, epubType)
		}
	}
	hasLength, hasHash := link.Length > 0, link.Hash != ""
	if (!hasLength || !hasHash) && !cfg.allowUnverified {
		return ErrPublicationUnverifiable
	}
	if hasLength && link.Length != size {
		return fmt.Errorf("%w: length %d, link states %d", ErrPublicationMismatch, size, link.Length)
	}
	var want []byte
	if hasHash {
		if want, err = decodeLinkHash(strings.TrimSpace(link.Hash)); err != nil {
			return err
		}
		got, err := sha256Of(publication, size)
		if err != nil {
			return err
		}
		if !bytes.Equal(got, want) {
			return fmt.Errorf("%w: SHA-256 differs", ErrPublicationMismatch)
		}
	}

	c, err := OpenContainer(publication, size)
	if err != nil {
		return err
	}
	if !c.Protected() && !c.MissingLicense() {
		return ErrNotProtected
	}
	// The link's hash describes the publication as served, without a license, so
	// a container that already has one is not that file and is never rewritten.
	if c.License != nil {
		return ErrAlreadyProtected
	}

	zw := zip.NewWriter(w)
	for _, f := range c.zr.File {
		if err := zw.Copy(f); err != nil {
			return fmt.Errorf("lcp: copy %s: %w", f.Name, err)
		}
	}
	lw, err := zw.CreateHeader(&zip.FileHeader{Name: LicensePath, Method: zip.Store})
	if err != nil {
		return err
	}
	if _, err := lw.Write(licenseDoc); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	if want != nil {
		again, err := sha256Of(publication, size)
		if err != nil {
			return err
		}
		if !bytes.Equal(again, want) {
			return fmt.Errorf("%w: publication changed while it was being fulfilled", ErrPublicationMismatch)
		}
	}
	return nil
}

// sha256Of hashes exactly size bytes of r, failing if the source is shorter.
func sha256Of(r io.ReaderAt, size int64) ([]byte, error) {
	h := sha256.New()
	n, err := io.Copy(h, io.NewSectionReader(r, 0, size))
	if err != nil {
		return nil, fmt.Errorf("lcp: read publication: %w", err)
	}
	if n != size {
		return nil, fmt.Errorf("%w: source yielded %d of %d bytes", ErrPublicationMismatch, n, size)
	}
	return h.Sum(nil), nil
}
