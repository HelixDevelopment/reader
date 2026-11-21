// Package fb2 parses FictionBook 2 XML into HDM (docs/27 Tier-1).
// FB2 is plain XML; a thin stdlib parser beats any external dependency.
package fb2

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/helixreader/helix/core/isbn"
	"github.com/helixreader/helix/core/pkg/hdm"
	"github.com/helixreader/helix/core/pkg/parse/parseutil"
)

type Parser struct{}

type fb2Book struct {
	TitleInfo struct {
		Title   string `xml:"book-title"`
		Authors []struct {
			First string `xml:"first-name"`
			Last  string `xml:"last-name"`
		} `xml:"author"`
		Lang string `xml:"lang"`
		ISBN string `xml:"isbn"`
	} `xml:"description>title-info"`
	Bodies []struct {
		Sections []fb2Section `xml:"section"`
	} `xml:"body"`
	Binaries []struct {
		ID   string `xml:"id,attr"`
		Data string `xml:",chardata"`
	} `xml:"binary"`
}

type fb2Section struct {
	Title    string       `xml:"title>p"`
	Parags   []fb2Para    `xml:"p"`
	Images   []fb2Image   `xml:"image"`
	Sections []fb2Section `xml:"section"`
}

// fb2Para keeps the paragraph chardata plus any inline <image> children so
// block order survives; FB2 places images either inline in <p> or directly
// in <section>.
type fb2Para struct {
	Text   string     `xml:",chardata"`
	Images []fb2Image `xml:"image"`
}

type fb2Image struct {
	Href string `xml:"href,attr"`
	Type string `xml:"type,attr"`
}

func (Parser) Parse(r io.Reader) (*hdm.Document, error) {
	data, err := parseutil.ReadAll(r, parseutil.MaxTextBytes)
	if err != nil {
		return nil, err
	}
	// Reject pathological nesting before encoding/xml's recursive struct
	// decoder or chapter conversion can grow the goroutine stack without bound.
	depthDecoder := xml.NewDecoder(bytes.NewReader(data))
	depthDecoder.Strict = false
	depth := 0
	for {
		token, err := depthDecoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("fb2: xml depth scan: %w", err)
		}
		switch token.(type) {
		case xml.StartElement:
			depth++
			if depth > parseutil.MaxXMLDepth {
				return nil, fmt.Errorf("fb2: XML depth exceeds %d", parseutil.MaxXMLDepth)
			}
		case xml.EndElement:
			depth--
		}
	}
	// FB2 files are often windows-1251; we require UTF-8/declared-XML here —
	// charset transcoding happens in the loader layer (golang.org/x/text).
	var book fb2Book
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	if err := dec.Decode(&book); err != nil {
		return nil, fmt.Errorf("fb2: xml: %w", err)
	}
	doc := &hdm.Document{Format: "fb2", Resources: map[string][]byte{}}
	doc.Meta.Title = strings.TrimSpace(book.TitleInfo.Title)
	doc.Meta.Language = book.TitleInfo.Lang
	doc.Meta.Identifiers = map[string]string{}
	if values := isbn.Extract(book.TitleInfo.ISBN); len(values) > 0 {
		doc.Meta.Identifiers["isbn"] = values[0]
	}
	for _, a := range book.TitleInfo.Authors {
		name := strings.TrimSpace(strings.TrimSpace(a.First) + " " + strings.TrimSpace(a.Last))
		if name != "" {
			doc.Meta.Creators = append(doc.Meta.Creators, name)
		}
	}
	// Binaries: FB2 embeds images/fonts as base64. Decode with per-entry and
	// aggregate bounds; malformed base64 fails the parse (fail closed — a
	// silently dropped binary would later surface as a broken image block).
	// The 64 MiB input cap already bounds the decoded total (<=48 MiB), but
	// the aggregate check stays because the cap constant may tighten.
	var aggregate int64
	for _, bn := range book.Binaries {
		if bn.ID == "" {
			return nil, fmt.Errorf("fb2: binary with empty id")
		}
		if _, dup := doc.Resources[bn.ID]; dup {
			return nil, fmt.Errorf("fb2: duplicate binary id %q", bn.ID)
		}
		data, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(bn.Data), ""))
		if err != nil {
			return nil, fmt.Errorf("fb2: binary %s: %w", bn.ID, err)
		}
		if int64(len(data)) > parseutil.MaxImageBytes {
			return nil, fmt.Errorf("fb2: binary %s expands to %d bytes over %d limit", bn.ID, len(data), parseutil.MaxImageBytes)
		}
		if aggregate+int64(len(data)) > parseutil.MaxAggregateBytes {
			return nil, fmt.Errorf("fb2: binaries exceed %d byte aggregate limit", parseutil.MaxAggregateBytes)
		}
		aggregate += int64(len(data))
		doc.Resources[bn.ID] = data
	}
	n := 0
	for _, body := range book.Bodies {
		for _, s := range body.Sections {
			if ch := sectionToChapter(&s, &n, doc.Resources); ch != nil {
				doc.Chapters = append(doc.Chapters, ch)
			}
		}
	}
	if err := doc.Validate(); err != nil {
		return nil, err
	}
	return doc, nil
}

func sectionToChapter(s *fb2Section, n *int, resources map[string][]byte) *hdm.Chapter {
	*n++
	ch := &hdm.Chapter{ID: fmt.Sprintf("s%d", *n), Title: strings.TrimSpace(s.Title)}
	if ch.Title != "" {
		ch.Blocks = append(ch.Blocks, hdm.Block{Kind: hdm.BlockHeading, Level: 1, Text: ch.Title})
	}
	addImage := func(img fb2Image) {
		// FB2 image references are fragments ("#cover") naming a binary id;
		// bare ids are accepted too. Unresolved references are skipped
		// (dangling image hrefs are common in the wild and carry no text to
		// preserve). Only PNG/JPEG payload sniffing happens downstream: the
		// parser stores bytes, it does not decode or render them.
		if img.Type == "x-container" || img.Href == "" {
			return
		}
		id := strings.TrimPrefix(img.Href, "#")
		if _, ok := resources[id]; ok {
			ch.Blocks = append(ch.Blocks, hdm.Block{Kind: hdm.BlockImage, ImageRef: id})
		}
	}
	for _, p := range s.Parags {
		t := strings.Join(strings.Fields(p.Text), " ")
		if t != "" {
			ch.Blocks = append(ch.Blocks, hdm.Block{Kind: hdm.BlockParagraph, Text: t})
		}
		for _, img := range p.Images {
			addImage(img)
		}
	}
	for _, img := range s.Images {
		addImage(img)
	}
	for i := range s.Sections {
		if child := sectionToChapter(&s.Sections[i], n, resources); child != nil {
			ch.Children = append(ch.Children, child)
		}
	}
	if len(ch.Blocks) == 0 && len(ch.Children) == 0 {
		return nil
	}
	return ch
}
