// Package fb2 parses FictionBook 2 XML into HDM (docs/27 Tier-1).
// FB2 is plain XML; a thin stdlib parser beats any external dependency.
package fb2

import (
	"bytes"
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
	Para     []string     `xml:"p"`
	Sections []fb2Section `xml:"section"`
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
	n := 0
	for _, body := range book.Bodies {
		for _, s := range body.Sections {
			if ch := sectionToChapter(&s, &n); ch != nil {
				doc.Chapters = append(doc.Chapters, ch)
			}
		}
	}
	if err := doc.Validate(); err != nil {
		return nil, err
	}
	return doc, nil
}

func sectionToChapter(s *fb2Section, n *int) *hdm.Chapter {
	*n++
	ch := &hdm.Chapter{ID: fmt.Sprintf("s%d", *n), Title: strings.TrimSpace(s.Title)}
	if ch.Title != "" {
		ch.Blocks = append(ch.Blocks, hdm.Block{Kind: hdm.BlockHeading, Level: 1, Text: ch.Title})
	}
	for _, p := range s.Para {
		t := strings.Join(strings.Fields(p), " ")
		if t != "" {
			ch.Blocks = append(ch.Blocks, hdm.Block{Kind: hdm.BlockParagraph, Text: t})
		}
	}
	for i := range s.Sections {
		ch.Children = append(ch.Children, sectionToChapter(&s.Sections[i], n))
	}
	if len(ch.Blocks) == 0 && len(ch.Children) == 0 {
		return nil
	}
	return ch
}
