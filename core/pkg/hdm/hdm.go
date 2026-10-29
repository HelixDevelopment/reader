// Package hdm implements the Helix Document Model (HDM), the unified
// intermediate representation that every supported input format is parsed
// into and every renderer consumes. See docs/27 (Format Matrix & Unified
// Rendering). HDM is the pivot of the conversion engine (docs/31):
// any format -> HDM -> any renderable/exportable format.
package hdm

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// BlockKind enumerates semantic block types in reading order.
type BlockKind int

const (
	BlockParagraph BlockKind = iota
	BlockHeading
	BlockQuote
	BlockList
	BlockImage
	BlockTable
	BlockCode
	BlockSeparator
	BlockFootnoteRef
)

func (k BlockKind) String() string {
	switch k {
	case BlockParagraph:
		return "paragraph"
	case BlockHeading:
		return "heading"
	case BlockQuote:
		return "quote"
	case BlockList:
		return "list"
	case BlockImage:
		return "image"
	case BlockTable:
		return "table"
	case BlockCode:
		return "code"
	case BlockSeparator:
		return "separator"
	case BlockFootnoteRef:
		return "footnote_ref"
	}
	return "unknown"
}

// Block is one semantic unit of content.
type Block struct {
	Kind     BlockKind
	Level    int    // heading level 1-6, list nesting depth
	Text     string // plain text payload (inline marks resolved by renderers)
	ImageRef string // HDM-internal resource id when Kind==BlockImage
	Language string // for BlockCode
}

// Chapter is a spine item: one navigable unit of the book.
type Chapter struct {
	ID       string
	Title    string
	Href     string // source href inside the original container
	Blocks   []Block
	Children []*Chapter // nested TOC entries
}

// Metadata carries normalised bibliographic data (drives docs/26
// identification cascade inputs and cover pipeline).
type Metadata struct {
	Title       string
	Creators    []string
	Language    string
	Identifiers map[string]string // scheme -> value: isbn, asin, oclc, lccn, doi, uuid
	Publisher   string
	Date        string
	Description string
	Subjects    []string
	Series      string
	SeriesIndex float64
}

// Document is the full HDM root.
type Document struct {
	Meta      Metadata
	Chapters  []*Chapter
	Resources map[string][]byte // images, fonts, css: id -> payload
	Footnotes map[string][]Block
	Format    string // originating format tag: epub, pdf, fb2, cbz...
}

// ErrNoContent is returned by Validate when the document has no spine.
var ErrNoContent = errors.New("hdm: document has no chapters")

// Validate enforces the minimal HDM invariants required by renderers.
func (d *Document) Validate() error {
	if len(d.Chapters) == 0 {
		return ErrNoContent
	}
	ids := map[string]bool{}
	var walk func(cs []*Chapter) error
	walk = func(cs []*Chapter) error {
		for _, c := range cs {
			if c.ID == "" {
				return errors.New("hdm: chapter with empty id")
			}
			if ids[c.ID] {
				return fmt.Errorf("hdm: duplicate chapter id %q", c.ID)
			}
			ids[c.ID] = true
			for _, b := range c.Blocks {
				if b.Kind == BlockImage && b.ImageRef != "" {
					if _, ok := d.Resources[b.ImageRef]; !ok {
						return fmt.Errorf("hdm: chapter %q references missing image %q", c.ID, b.ImageRef)
					}
				}
			}
			if err := walk(c.Children); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(d.Chapters)
}

// WordCount returns the total word count across all chapters.
func (d *Document) WordCount() int {
	n := 0
	var walk func(cs []*Chapter)
	walk = func(cs []*Chapter) {
		for _, c := range cs {
			for _, b := range c.Blocks {
				if b.Kind == BlockImage || b.Kind == BlockSeparator {
					continue
				}
				n += len(strings.Fields(b.Text))
			}
			walk(c.Children)
		}
	}
	walk(d.Chapters)
	return n
}

// Fingerprint returns a stable content fingerprint used by the
// identification cascade (docs/26) as a last-resort matcher when no
// ISBN/ASIN/OCLC is present.
func (d *Document) Fingerprint() string {
	h := sha256.New()
	h.Write([]byte(strings.ToLower(strings.TrimSpace(d.Meta.Title))))
	var flat []*Chapter
	var walk func(cs []*Chapter)
	walk = func(cs []*Chapter) {
		for _, c := range cs {
			flat = append(flat, c)
			walk(c.Children)
		}
	}
	walk(d.Chapters)
	// First 10 chapters contribute their first 500 runes — mirrors the
	// "first 5-10 pages" analysis from docs/26.
	for i, c := range flat {
		if i >= 10 {
			break
		}
		for _, b := range c.Blocks {
			t := strings.ToLower(strings.Join(strings.Fields(b.Text), " "))
			r := []rune(t)
			if len(r) > 500 {
				t = string(r[:500])
			}
			h.Write([]byte(t))
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

// FlatTOC returns the table of contents flattened in reading order.
func (d *Document) FlatTOC() []*Chapter {
	var out []*Chapter
	var walk func(cs []*Chapter)
	walk = func(cs []*Chapter) {
		for _, c := range cs {
			out = append(out, c)
			walk(c.Children)
		}
	}
	walk(d.Chapters)
	return out
}

// SortKeys is a helper for deterministic serialisation of identifier maps.
func (m Metadata) SortKeys() []string {
	keys := make([]string, 0, len(m.Identifiers))
	for k := range m.Identifiers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
