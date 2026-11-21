// Page enumeration: ordered page list for a CBZ archive derived from the ZIP
// central directory only. No image payload is ever inflated here, so cost is
// proportional to the entry count, not the (compressible) payload size.
// This supports page-count metadata and pre-fetch planning; it is directory
// inspection, not rendering or parsing of page content.
package cbz

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"

	"github.com/helixreader/helix/core/pkg/parse/parseutil"
)

// PageRef is one image entry in deterministic (lexicographic by name, the
// same ordering Parse applies) reading order.
type PageRef struct {
	Index          int    // 0-based page sequence number
	ID             string // HDM resource id Parse will assign: page-%04d
	Name           string // ZIP entry path
	Size           uint64 // uncompressed size from the central directory
	CompressedSize uint64 // stored payload size
}

// Pages lists the image pages of a comic archive without inflating any
// entry. It enforces the same entry-count and page-count bounds as Parse
// and fails closed on corrupt archives.
func Pages(r io.ReaderAt, size int64) ([]PageRef, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("cbz: open zip: %w", err)
	}
	if len(zr.File) > parseutil.MaxArchiveEntries {
		return nil, fmt.Errorf("cbz: archive entry count exceeds %d", parseutil.MaxArchiveEntries)
	}
	var names []string
	byName := map[string]*zip.File{}
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if imageExts[strings.ToLower(path.Ext(f.Name))] {
			names = append(names, f.Name)
			byName[f.Name] = f
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil, fmt.Errorf("cbz: no images in archive")
	}
	if len(names) > parseutil.MaxPages {
		return nil, fmt.Errorf("cbz: page count exceeds %d", parseutil.MaxPages)
	}
	refs := make([]PageRef, 0, len(names))
	for i, name := range names {
		f := byName[name]
		refs = append(refs, PageRef{
			Index:          i,
			ID:             fmt.Sprintf("page-%04d", i+1),
			Name:           name,
			Size:           f.UncompressedSize64,
			CompressedSize: f.CompressedSize64,
		})
	}
	return refs, nil
}

// pageSignature is the leading byte run that proves what a page payload
// actually is. Only image types a page-stream link is allowed to declare are
// recognized; a payload matching none of them proves nothing, so the archive
// gets no advertised type rather than a guessed one.
var pageSignature = []struct {
	prefix    []byte
	mediaType string
}{
	{[]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}, "image/png"},
	{[]byte{0xff, 0xd8, 0xff}, "image/jpeg"},
	{[]byte("GIF87a"), "image/gif"},
	{[]byte("GIF89a"), "image/gif"},
}

const signatureBytes = 8

// SniffMediaType names the image type a payload prefix proves, or "" when the
// prefix proves nothing streamable.
func SniffMediaType(prefix []byte) string {
	for _, s := range pageSignature {
		if bytes.HasPrefix(prefix, s.prefix) {
			return s.mediaType
		}
	}
	return ""
}

// StreamMediaType reports the image type shared by every page payload in the
// archive. It inflates only each entry's first bytes, so its cost follows the
// entry count and not the page sizes. It fails closed: a page that proves
// nothing, or a mix of types, leaves a stream link with no single honest type
// to declare.
func StreamMediaType(r io.ReaderAt, size int64) (string, error) {
	refs, err := Pages(r, size)
	if err != nil {
		return "", err
	}
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return "", fmt.Errorf("cbz: open zip: %w", err)
	}
	entries := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		entries[f.Name] = f
	}
	media := ""
	for _, ref := range refs {
		f, ok := entries[ref.Name]
		if !ok {
			return "", fmt.Errorf("cbz: enumerated page %q is not in the archive", ref.Name)
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		head := make([]byte, signatureBytes)
		n, err := io.ReadFull(rc, head)
		rc.Close()
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			return "", err
		}
		t := SniffMediaType(head[:n])
		if t == "" {
			return "", fmt.Errorf("cbz: page %q is not a jpeg, png or gif image", ref.Name)
		}
		if media == "" {
			media = t
		} else if media != t {
			return "", fmt.Errorf("cbz: pages mix %s and %s", media, t)
		}
	}
	if media == "" {
		return "", fmt.Errorf("cbz: no pages to type")
	}
	return media, nil
}

// Page returns the payload of one enumerated page. Ordering and bounds are
// those Pages defines, and inflation is capped at one page's limit, so
// serving a page can never cost more than parsing one.
func Page(r io.ReaderAt, size int64, index int) (PageRef, []byte, error) {
	refs, err := Pages(r, size)
	if err != nil {
		return PageRef{}, nil, err
	}
	if index < 0 || index >= len(refs) {
		return PageRef{}, nil, fmt.Errorf("cbz: page index %d out of range 0..%d", index, len(refs)-1)
	}
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return PageRef{}, nil, fmt.Errorf("cbz: open zip: %w", err)
	}
	ref := refs[index]
	for _, f := range zr.File {
		if f.Name != ref.Name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return PageRef{}, nil, err
		}
		data, err := parseutil.ReadAll(rc, parseutil.MaxImageBytes)
		closeErr := rc.Close()
		if err != nil {
			return PageRef{}, nil, err
		}
		if closeErr != nil {
			return PageRef{}, nil, closeErr
		}
		return ref, data, nil
	}
	return PageRef{}, nil, fmt.Errorf("cbz: enumerated page %q is not in the archive", ref.Name)
}
