// Spine enumeration: ordered page extraction metadata without decompressing
// any content stream (constant-memory directory walk). This is detection +
// structure only: it reads container.xml and the OPF (both bounded by
// parseutil limits) and returns the spine order with resolved entry names
// and sizes; page payloads are delivered by Parser.Parse, never here, and
// nothing in this file renders or converts anything.
package epub

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"

	"github.com/helixreader/helix/core/pkg/parse/parseutil"
)

// SpinePage is one spine slot in reading order. Index is the 0-based page
// sequence position; DuplicateOf is the Index of the first spine item that
// resolved to the same ZIP entry (-1 for first occurrences; items missing
// from the manifest keep an empty Entry).
type SpinePage struct {
	Index       int
	IDRef       string
	Entry       string // resolved ZIP entry path, "" when unresolved
	MediaType   string
	Size        uint64 // uncompressed size from the ZIP central directory
	DuplicateOf int    // -1 when unique, otherwise first Index with same Entry
}

// Spine returns the ordered spine page list of an EPUB container. It fails
// closed on every resource bound: entry count, aggregate read budget, and
// per-entry size limits are enforced exactly as in Parse, and no entry
// content beyond container.xml and the OPF is ever inflated.
func Spine(r io.ReaderAt, size int64) ([]SpinePage, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("epub: open zip: %w", err)
	}
	if len(zr.File) > parseutil.MaxArchiveEntries {
		return nil, fmt.Errorf("epub: archive entry count exceeds %d", parseutil.MaxArchiveEntries)
	}
	files := map[string]*zip.File{}
	for _, f := range zr.File {
		files[f.Name] = f
	}
	remaining := parseutil.MaxAggregateBytes
	read := func(name string) ([]byte, error) {
		f, ok := files[name]
		if !ok {
			return nil, fmt.Errorf("epub: missing %s", name)
		}
		if remaining <= 0 {
			return nil, fmt.Errorf("epub: aggregate extraction limit reached")
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		limit := min(parseutil.MaxEntryBytes, remaining)
		data, err := parseutil.ReadAll(rc, limit)
		if err == nil {
			remaining -= int64(len(data))
		}
		return data, err
	}
	cdata, err := read("META-INF/container.xml")
	if err != nil {
		return nil, err
	}
	var cont container
	if err := xml.Unmarshal(cdata, &cont); err != nil {
		return nil, fmt.Errorf("epub: container.xml: %w", err)
	}
	if len(cont.Rootfiles) == 0 {
		return nil, fmt.Errorf("epub: no rootfile")
	}
	opfPath := cont.Rootfiles[0].FullPath
	opfDir := path.Dir(opfPath)
	odata, err := read(opfPath)
	if err != nil {
		return nil, err
	}
	var opf opfPackage
	if err := xml.Unmarshal(odata, &opf); err != nil {
		return nil, fmt.Errorf("epub: opf: %w", err)
	}
	type item struct{ href, mt string }
	byID := map[string]item{}
	for _, m := range opf.Manifest {
		byID[m.ID] = item{m.Href, m.MediaType}
	}
	seen := map[string]int{}
	pages := make([]SpinePage, 0, len(opf.Spine))
	for i, s := range opf.Spine {
		if len(pages) >= parseutil.MaxPages {
			return nil, fmt.Errorf("epub: spine page count exceeds %d", parseutil.MaxPages)
		}
		p := SpinePage{Index: i, IDRef: s.IDRef, DuplicateOf: -1}
		m, ok := byID[s.IDRef]
		if ok {
			entry, err := func() (string, error) {
				u, err := url.Parse(m.href)
				if err != nil {
					return "", fmt.Errorf("epub: invalid manifest href %q: %w", m.href, err)
				}
				if u.IsAbs() || u.Host != "" || strings.HasPrefix(u.Path, "/") {
					return "", fmt.Errorf("epub: non-local manifest href %q", m.href)
				}
				return path.Join(opfDir, u.Path), nil
			}()
			if err != nil {
				return nil, fmt.Errorf("epub: spine item %s: %w", s.IDRef, err)
			}
			p.Entry, p.MediaType = entry, m.mt
			if f, ok := files[entry]; ok {
				p.Size = f.UncompressedSize64
			}
			if first, dup := seen[entry]; dup {
				p.DuplicateOf = first
			} else {
				seen[entry] = i
			}
		}
		pages = append(pages, p)
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("epub: empty spine")
	}
	return pages, nil
}
