// Page enumeration: ordered page list for a CBZ archive derived from the ZIP
// central directory only. No image payload is ever inflated here, so cost is
// proportional to the entry count, not the (compressible) payload size.
// This supports page-count metadata and pre-fetch planning; it is directory
// inspection, not rendering or parsing of page content.
package cbz

import (
	"archive/zip"
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
