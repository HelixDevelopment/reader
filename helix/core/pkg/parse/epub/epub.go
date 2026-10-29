// Package epub parses EPUB 2/3 containers into HDM using only the stdlib
// (archive/zip + encoding/xml). Production builds may delegate to
// readium/go-toolkit behind the same parser.Parser contract; this
// dependency-free implementation guarantees a baseline parser on every
// platform including WASM and restricted mobile builds.
package epub

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/helixreader/helix/core/pkg/hdm"
	"github.com/helixreader/helix/core/pkg/parse/parseutil"
)

// Parser implements parsing of EPUB containers.
type Parser struct{}

// opf structures (subset).
type container struct {
	Rootfiles []struct {
		FullPath string `xml:"full-path,attr"`
	} `xml:"rootfiles>rootfile"`
}

type opfPackage struct {
	Metadata struct {
		Titles      []string `xml:"title"`
		Creators    []string `xml:"creator"`
		Language    string   `xml:"language"`
		Identifiers []struct {
			Scheme string `xml:"scheme,attr"`
			Value  string `xml:",chardata"`
		} `xml:"identifier"`
		Publisher string `xml:"publisher"`
		Date      string `xml:"date"`
	} `xml:"metadata"`
	Manifest []struct {
		ID        string `xml:"id,attr"`
		Href      string `xml:"href,attr"`
		MediaType string `xml:"media-type,attr"`
	} `xml:"manifest>item"`
	Spine []struct {
		IDRef string `xml:"idref,attr"`
	} `xml:"spine>itemref"`
}

// Parse reads an EPUB from r (zip container) into HDM.
func (Parser) Parse(r io.ReaderAt, size int64) (*hdm.Document, error) {
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
	read := func(name string, entryLimit int64) ([]byte, error) {
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
		limit := min(entryLimit, remaining)
		data, err := parseutil.ReadAll(rc, limit)
		if err == nil {
			remaining -= int64(len(data))
		}
		return data, err
	}

	cdata, err := read("META-INF/container.xml", parseutil.MaxEntryBytes)
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
	resolve := func(href string) (string, error) {
		u, err := url.Parse(href)
		if err != nil {
			return "", fmt.Errorf("epub: invalid manifest href %q: %w", href, err)
		}
		if u.IsAbs() || u.Host != "" || strings.HasPrefix(u.Path, "/") {
			return "", fmt.Errorf("epub: non-local manifest href %q", href)
		}
		// url.Parse decodes percent-escaped path octets into Path. Query and
		// fragment components identify views, not additional ZIP entries.
		return path.Join(opfDir, u.Path), nil
	}

	var opf opfPackage
	odata, err := read(opfPath, parseutil.MaxEntryBytes)
	if err != nil {
		return nil, err
	}
	if err := xml.Unmarshal(odata, &opf); err != nil {
		return nil, fmt.Errorf("epub: opf: %w", err)
	}

	doc := &hdm.Document{Format: "epub", Resources: map[string][]byte{}}
	if len(opf.Metadata.Titles) > 0 {
		doc.Meta.Title = strings.TrimSpace(opf.Metadata.Titles[0])
	}
	doc.Meta.Creators = opf.Metadata.Creators
	doc.Meta.Language = opf.Metadata.Language
	doc.Meta.Publisher = opf.Metadata.Publisher
	doc.Meta.Date = opf.Metadata.Date
	doc.Meta.Identifiers = map[string]string{}
	for _, id := range opf.Metadata.Identifiers {
		v := strings.TrimSpace(id.Value)
		scheme := strings.ToLower(id.Scheme)
		if scheme == "" {
			if strings.HasPrefix(v, "urn:isbn:") {
				scheme, v = "isbn", strings.TrimPrefix(v, "urn:isbn:")
			} else if strings.HasPrefix(v, "urn:uuid:") {
				scheme, v = "uuid", strings.TrimPrefix(v, "urn:uuid:")
			} else {
				scheme = "other"
			}
		}
		doc.Meta.Identifiers[scheme] = v
	}

	byID := map[string]struct{ href, mt string }{}
	for _, m := range opf.Manifest {
		byID[m.ID] = struct{ href, mt string }{m.Href, m.MediaType}
	}
	for i, item := range opf.Spine {
		m, ok := byID[item.IDRef]
		if !ok {
			continue
		}
		ch := &hdm.Chapter{
			ID:    item.IDRef,
			Title: fmt.Sprintf("Section %d", i+1),
			Href:  m.href,
		}
		entry, err := resolve(m.href)
		if err != nil {
			return nil, fmt.Errorf("epub: spine item %s: %w", item.IDRef, err)
		}
		raw, err := read(entry, parseutil.MaxEntryBytes)
		if err != nil {
			return nil, fmt.Errorf("epub: spine item %s: %w", item.IDRef, err)
		}
		ch.Blocks = xhtmlToBlocks(string(raw))
		if len(ch.Blocks) > 0 && ch.Blocks[0].Kind == hdm.BlockHeading {
			ch.Title = ch.Blocks[0].Text
		}
		doc.Chapters = append(doc.Chapters, ch)
	}
	// Resources: images/fonts from the manifest. Sort IDs so aggregate-budget
	// decisions are deterministic across runs.
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		m := byID[id]
		if strings.HasPrefix(m.mt, "image/") || strings.HasPrefix(m.mt, "font/") ||
			strings.Contains(m.mt, "font") {
			entry, err := resolve(m.href)
			if err != nil {
				return nil, fmt.Errorf("epub: manifest resource %s: %w", id, err)
			}
			limit := parseutil.MaxEntryBytes
			if strings.HasPrefix(m.mt, "image/") {
				limit = parseutil.MaxImageBytes
			}
			data, err := read(entry, limit)
			if err != nil {
				return nil, fmt.Errorf("epub: manifest resource %s: %w", id, err)
			}
			doc.Resources[id] = data
		}
	}
	if err := doc.Validate(); err != nil {
		return nil, err
	}
	return doc, nil
}

// xhtmlToBlocks extracts block-level content from XHTML, whitespace-
// normalised (constitution F-A4: parsers emit normalised text).
func xhtmlToBlocks(xhtml string) []hdm.Block {
	d := xml.NewDecoder(strings.NewReader(xhtml))
	var blocks []hdm.Block
	var cur *hdm.Block
	flush := func() {
		if cur == nil {
			return
		}
		cur.Text = strings.Join(strings.Fields(cur.Text), " ")
		if cur.Text != "" {
			blocks = append(blocks, *cur)
		}
		cur = nil
	}
	for {
		tok, err := d.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "p", "li", "dt", "dd", "div", "tr", "figcaption", "pre":
				flush()
				cur = &hdm.Block{Kind: hdm.BlockParagraph}
			case "h1", "h2", "h3", "h4", "h5", "h6":
				flush()
				cur = &hdm.Block{Kind: hdm.BlockHeading, Level: int(t.Name.Local[1] - '0')}
			case "blockquote":
				flush()
				cur = &hdm.Block{Kind: hdm.BlockQuote}
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "p", "li", "dt", "dd", "div", "tr", "figcaption", "pre", "h1", "h2", "h3", "h4", "h5", "h6", "blockquote":
				flush()
			}
		case xml.CharData:
			if cur != nil {
				cur.Text += string(t) + " "
			}
		}
	}
	flush()
	return blocks
}
