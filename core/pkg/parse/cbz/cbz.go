// Package cbz parses comic book archives (CBZ = zip of images) into HDM:
// each image becomes a chapter-less page sequence (docs/27 Tier-1).
package cbz

import (
	"archive/zip"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"

	"github.com/helixreader/helix/core/pkg/hdm"
	"github.com/helixreader/helix/core/pkg/parse/parseutil"
)

type Parser struct{}

var imageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true, ".avif": true,
}

func (Parser) Parse(r io.ReaderAt, size int64) (*hdm.Document, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("cbz: open zip: %w", err)
	}
	if len(zr.File) > parseutil.MaxArchiveEntries {
		return nil, fmt.Errorf("cbz: archive entry count exceeds %d", parseutil.MaxArchiveEntries)
	}
	var pages []*zip.File
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if imageExts[strings.ToLower(path.Ext(f.Name))] {
			pages = append(pages, f)
		}
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("cbz: no images in archive")
	}
	if len(pages) > parseutil.MaxPages {
		return nil, fmt.Errorf("cbz: page count exceeds %d", parseutil.MaxPages)
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i].Name < pages[j].Name })
	doc := &hdm.Document{Format: "cbz", Resources: map[string][]byte{}}
	doc.Meta.Identifiers = map[string]string{}
	ch := &hdm.Chapter{ID: "pages", Title: "Pages"}
	var aggregate int64
	for i, page := range pages {
		id := fmt.Sprintf("page-%04d", i+1)
		rc, err := page.Open()
		if err != nil {
			return nil, err
		}
		limit := min(parseutil.MaxImageBytes, parseutil.MaxAggregateBytes-aggregate)
		data, err := parseutil.ReadAll(rc, limit)
		closeErr := rc.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
		aggregate += int64(len(data))
		doc.Resources[id] = data
		ch.Blocks = append(ch.Blocks, hdm.Block{Kind: hdm.BlockImage, ImageRef: id})
	}
	doc.Chapters = []*hdm.Chapter{ch}
	if err := doc.Validate(); err != nil {
		return nil, err
	}
	return doc, nil
}
