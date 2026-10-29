// Package convert implements the HDM-pivot conversion engine (docs/31):
// any parsed format -> HDM -> exportable formats. On-device exporters are
// pure Go; heavy targets (PDF/AZW3) are server-side per plan P-2.
package convert

import (
	"fmt"
	"strings"

	"github.com/helixreader/helix/core/pkg/hdm"
)

// Format is an export target.
type Format string

const (
	FormatTXT   Format = "txt"
	FormatXHTML Format = "xhtml"
	FormatFB2   Format = "fb2"
)

// Options controls conversion (docs/31 options/presets).
type Options struct {
	IncludeHeadings bool
	HeadingMarker   string // e.g. "#" for markdown-ish txt
	LineWidth       int    // 0 = no wrapping
}

// DefaultOptions matches the "Balanced" preset from docs/31.
var DefaultOptions = Options{IncludeHeadings: true, HeadingMarker: "#", LineWidth: 0}

// Convert transforms HDM into the target format.
func Convert(doc *hdm.Document, target Format, opt Options) ([]byte, error) {
	if err := doc.Validate(); err != nil {
		return nil, err
	}
	switch target {
	case FormatTXT:
		return toTXT(doc, opt), nil
	case FormatXHTML:
		return toXHTML(doc, opt), nil
	case FormatFB2:
		return toFB2(doc, opt), nil
	}
	return nil, fmt.Errorf("convert: unsupported target %q", target)
}

func blockText(b hdm.Block, opt Options) string {
	if b.Kind == hdm.BlockHeading && opt.IncludeHeadings {
		return strings.Repeat(opt.HeadingMarker, max(1, b.Level)) + " " + b.Text
	}
	return b.Text
}

func toTXT(doc *hdm.Document, opt Options) []byte {
	var b strings.Builder
	if doc.Meta.Title != "" {
		b.WriteString(doc.Meta.Title + "\n\n")
	}
	for _, ch := range doc.FlatTOC() {
		for _, blk := range ch.Blocks {
			switch blk.Kind {
			case hdm.BlockImage, hdm.BlockSeparator:
				continue
			}
			t := blockText(blk, opt)
			if opt.LineWidth > 0 {
				t = wrap(t, opt.LineWidth)
			}
			b.WriteString(t + "\n\n")
		}
	}
	return []byte(b.String())
}

func esc(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

func toXHTML(doc *hdm.Document, opt Options) []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n<html xmlns=\"http://www.w3.org/1999/xhtml\"><head><title>")
	b.WriteString(esc(doc.Meta.Title))
	b.WriteString("</title></head><body>\n")
	for _, ch := range doc.FlatTOC() {
		b.WriteString("<section id=\"" + esc(ch.ID) + "\">\n")
		for _, blk := range ch.Blocks {
			switch blk.Kind {
			case hdm.BlockHeading:
				lvl := max(1, min(6, blk.Level))
				fmt.Fprintf(&b, "<h%d>%s</h%d>\n", lvl, esc(blk.Text), lvl)
			case hdm.BlockParagraph:
				b.WriteString("<p>" + esc(blk.Text) + "</p>\n")
			case hdm.BlockQuote:
				b.WriteString("<blockquote>" + esc(blk.Text) + "</blockquote>\n")
			case hdm.BlockImage:
				b.WriteString("<img src=\"res/" + esc(blk.ImageRef) + "\" alt=\"\"/>\n")
			case hdm.BlockCode:
				b.WriteString("<pre><code>" + esc(blk.Text) + "</code></pre>\n")
			}
		}
		b.WriteString("</section>\n")
	}
	b.WriteString("</body></html>\n")
	return []byte(b.String())
}

func toFB2(doc *hdm.Document, opt Options) []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n<FictionBook xmlns=\"http://www.gribuser.ru/xml/fictionbook/2.0\">\n<description><title-info>\n")
	for _, c := range doc.Meta.Creators {
		b.WriteString("<author><first-name>" + esc(c) + "</first-name></author>\n")
	}
	b.WriteString("<book-title>" + esc(doc.Meta.Title) + "</book-title>\n<lang>" + esc(doc.Meta.Language) + "</lang>\n</title-info></description>\n<body>\n")
	for _, ch := range doc.FlatTOC() {
		b.WriteString("<section>\n")
		if ch.Title != "" {
			b.WriteString("<title><p>" + esc(ch.Title) + "</p></title>\n")
		}
		for _, blk := range ch.Blocks {
			if blk.Kind == hdm.BlockParagraph || blk.Kind == hdm.BlockQuote {
				b.WriteString("<p>" + esc(blk.Text) + "</p>\n")
			}
		}
		b.WriteString("</section>\n")
	}
	b.WriteString("</body>\n</FictionBook>\n")
	return []byte(b.String())
}

// wrap performs deterministic greedy word wrapping.
func wrap(s string, width int) string {
	words := strings.Fields(s)
	var lines []string
	var cur string
	for _, w := range words {
		if cur == "" {
			cur = w
			continue
		}
		if len(cur)+1+len(w) > width {
			lines = append(lines, cur)
			cur = w
		} else {
			cur += " " + w
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return strings.Join(lines, "\n")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
