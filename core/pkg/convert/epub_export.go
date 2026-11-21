// EPUB container exporter: HDM -> valid EPUB 3 zip (docs/31 pivot). This
// makes the FB2/TXT/MOBI -> EPUB rows of the formats matrix real on-device
// conversions instead of aspirations. Determinism (constitution F-A3):
// fixed MS-DOS timestamp, FlatTOC reading order, sorted resource ids.
//
// Fidelity policy (constitution §7.1 NO BLUFF): the exporter FAILS rather
// than silently dropping information. Blocks whose semantics the XHTML
// serializer cannot express exactly when re-parsed (code, tables, lists,
// footnote refs) and referenced resources that are not images are rejected;
// callers keep the txt/xhtml/fb2 targets for text-only projections.
package convert

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/helixreader/helix/core/pkg/hdm"
)

const epubMIME = "application/epub+zip"

// epubDate is the fixed epoch for all zip entries (MS-DOS minimum). Real
// modification times would make byte-identical output impossible.
var epubDate = time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)

type epubPart struct {
	id, entry, mt string
	body          []byte
}

// imageExt maps sniffed image encodings to file extensions and media types.
func imageExt(data []byte) (ext, mt string, ok bool) {
	switch {
	case bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")):
		return ".png", "image/png", true
	case bytes.HasPrefix(data, []byte{0xff, 0xd8, 0xff}):
		return ".jpg", "image/jpeg", true
	}
	return "", "", false
}

// toEPUB serialises the whole HDM document into an EPUB container.
func toEPUB(doc *hdm.Document, opt Options) ([]byte, error) {
	chapters := doc.FlatTOC()
	used := map[string]bool{}
	for _, ch := range chapters {
		if used[ch.ID] {
			return nil, fmt.Errorf("convert: epub: duplicate chapter id %q", ch.ID)
		}
		used[ch.ID] = true
	}
	// Images referenced anywhere in the document must exist as resources
	// (hdm.Validate already guarantees this) and be encodable.
	referenced := map[string]bool{}
	for _, ch := range chapters {
		for _, b := range ch.Blocks {
			if b.Kind == hdm.BlockImage {
				if b.ImageRef == "" || strings.ContainsAny(b.ImageRef, `/%\`) {
					return nil, fmt.Errorf("convert: epub: invalid image resource id %q", b.ImageRef)
				}
				referenced[b.ImageRef] = true
			}
		}
	}
	for id, data := range doc.Resources {
		if !referenced[id] {
			continue
		}
		if _, _, ok := imageExt(data); !ok {
			return nil, fmt.Errorf("convert: epub: image resource %q is neither PNG nor JPEG; refusing lossy export", id)
		}
	}

	var parts []epubPart
	for _, ch := range chapters {
		// Chapter titles have no dedicated slot in spine XHTML. When the
		// first block already is a heading with the title text the body
		// represents it faithfully; otherwise the title is injected as a
		// leading h1 so it survives the container hop (EPUB readers and our
		// own parser derive chapter titles from the first heading).
		blocks := ch.Blocks
		if ch.Title != "" && (len(blocks) == 0 || blocks[0].Kind != hdm.BlockHeading || blocks[0].Text != ch.Title) {
			blocks = append([]hdm.Block{{Kind: hdm.BlockHeading, Level: 1, Text: ch.Title}}, blocks...)
		}
		body, err := epubXHTML(ch, blocks, doc)
		if err != nil {
			return nil, err
		}
		parts = append(parts, epubPart{id: "ch-" + ch.ID, entry: "OEBPS/" + ch.ID + ".xhtml",
			mt: "application/xhtml+xml", body: body})
	}
	ids := make([]string, 0, len(doc.Resources))
	for id := range doc.Resources {
		if referenced[id] {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		ext, mt, _ := imageExt(doc.Resources[id])
		parts = append(parts, epubPart{id: "res-" + id, entry: "OEBPS/res/" + id + ext,
			mt: mt, body: doc.Resources[id]})
	}

	opf, err := epubOPF(doc, parts)
	if err != nil {
		return nil, err
	}
	ncx, err := epubNCX(chapters)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	// The mimetype entry must be first and stored (EPUB 3 §4.1).
	mw, err := zw.CreateHeader(&zip.FileHeader{Name: "mimetype", Method: zip.Store, Modified: epubDate})
	if err != nil {
		return nil, err
	}
	if _, err := mw.Write([]byte(epubMIME)); err != nil {
		return nil, err
	}
	add := func(name string, body []byte) error {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate, Modified: epubDate})
		if err != nil {
			return err
		}
		_, err = w.Write(body)
		return err
	}
	if err := add("META-INF/container.xml", []byte(epubContainerXML)); err != nil {
		return nil, err
	}
	if err := add("OEBPS/content.opf", opf); err != nil {
		return nil, err
	}
	if err := add("OEBPS/toc.ncx", ncx); err != nil {
		return nil, err
	}
	for _, p := range parts {
		if err := add(p.entry, p.body); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

const epubContainerXML = `<?xml version="1.0"?>` + "\n" +
	`<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">` +
	`<rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>` + "\n"

// writeBlocks renders HDM blocks as XHTML tokens using encoding/xml so text
// payloads are always escaped (never markup-injected). It is the lossless
// serializer used by the EPUB exporter; the standalone single-document
// XHTML target below stays a lossy text projection for historic payloads.
func writeBlocks(b *strings.Builder, blocks []hdm.Block, doc *hdm.Document) error {
	for _, blk := range blocks {
		switch blk.Kind {
		case hdm.BlockHeading:
			lvl := blk.Level
			if lvl < 1 {
				lvl = 1
			}
			if lvl > 6 {
				return fmt.Errorf("convert: heading level %d out of range 1-6", blk.Level)
			}
			tag := fmt.Sprintf("h%d", lvl)
			if err := writeElement(b, tag, blk.Text); err != nil {
				return err
			}
		case hdm.BlockParagraph:
			if err := writeElement(b, "p", blk.Text); err != nil {
				return err
			}
		case hdm.BlockQuote:
			if err := writeElement(b, "blockquote", blk.Text); err != nil {
				return err
			}
		case hdm.BlockSeparator:
			b.WriteString("<hr/>\n")
		case hdm.BlockImage:
			if _, ok := doc.Resources[blk.ImageRef]; !ok {
				return fmt.Errorf("convert: missing image resource %q", blk.ImageRef)
			}
			ext, _, ok := imageExt(doc.Resources[blk.ImageRef])
			if !ok {
				return fmt.Errorf("convert: image resource %q is neither PNG nor JPEG", blk.ImageRef)
			}
			b.WriteString(`<img src="res/` + xmlEscapeAttr(blk.ImageRef+ext) + `" alt="` + xmlEscapeAttr("") + `"/>` + "\n")
		case hdm.BlockList, hdm.BlockTable, hdm.BlockCode, hdm.BlockFootnoteRef:
			return fmt.Errorf("convert: block kind %s has no lossless XHTML serialisation; refusing lossy export", blk.Kind)
		default:
			return fmt.Errorf("convert: unknown block kind %d", int(blk.Kind))
		}
	}
	return nil
}

func writeElement(b *strings.Builder, tag, text string) error {
	if text == "" {
		return fmt.Errorf("convert: %s: empty text block has no lossless serialisation", tag)
	}
	if err := validateRoundTripText(text); err != nil {
		return fmt.Errorf("convert: %s: %w", tag, err)
	}
	b.WriteString("<" + tag + ">")
	if err := xml.EscapeText(b, []byte(text)); err != nil {
		return err
	}
	b.WriteString("</" + tag + ">\n")
	return nil
}

// validateRoundTripText rejects payloads that would not survive a
// parse->export->parse cycle unchanged: whitespace is normalised at parse
// time and control characters are dropped by the XML layer.
func validateRoundTripText(s string) error {
	if s == "" {
		return nil
	}
	if strings.Join(strings.Fields(s), " ") != s {
		return fmt.Errorf("text %q is not whitespace-normalised; export would alter it", s)
	}
	for _, r := range s {
		if r < 0x20 && r != '\t' {
			return fmt.Errorf("control character U+%04X cannot be exported without loss", r)
		}
		if r == 0x7f {
			return fmt.Errorf("DEL character cannot be exported without loss")
		}
	}
	return nil
}

func xmlEscapeAttr(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// epubXHTML renders one spine document for chapter ch using the given
// (possibly title-injected) block sequence.
func epubXHTML(ch *hdm.Chapter, blocks []hdm.Block, doc *hdm.Document) ([]byte, error) {
	if err := validateRoundTripText(ch.Title); err != nil {
		return nil, fmt.Errorf("convert: chapter %q title: %w", ch.ID, err)
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	b.WriteString(`<html xmlns="http://www.w3.org/1999/xhtml"><head><title>`)
	if err := xml.EscapeText(&b, []byte(doc.Meta.Title)); err != nil {
		return nil, err
	}
	b.WriteString("</title></head><body>\n")
	if err := writeBlocks(&b, blocks, doc); err != nil {
		return nil, fmt.Errorf("convert: chapter %q: %w", ch.ID, err)
	}
	b.WriteString("</body></html>\n")
	return []byte(b.String()), nil
}

func titleFromBlocks(ch *hdm.Chapter) string {
	if len(ch.Blocks) > 0 && ch.Blocks[0].Kind == hdm.BlockHeading {
		return ch.Blocks[0].Text
	}
	return ch.Title
}

// epubOPF writes package.opf: manifest ids are "ch-"/"res-"-prefixed so they
// can never collide with each other.
func epubOPF(doc *hdm.Document, parts []epubPart) ([]byte, error) {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	b.WriteString(`<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="uid">` + "\n")
	b.WriteString("<metadata xmlns:dc=\"http://purl.org/dc/elements/1.1/\">\n")
	b.WriteString(`<dc:identifier id="uid">urn:uuid:` + doc.Fingerprint() + "</dc:identifier>\n")
	if doc.Meta.Title != "" {
		if err := validateRoundTripText(doc.Meta.Title); err != nil {
			return nil, fmt.Errorf("convert: epub title: %w", err)
		}
		b.WriteString("<dc:title>")
		if err := xml.EscapeText(&b, []byte(doc.Meta.Title)); err != nil {
			return nil, err
		}
		b.WriteString("</dc:title>\n")
	}
	for _, c := range doc.Meta.Creators {
		if err := validateRoundTripText(c); err != nil {
			return nil, fmt.Errorf("convert: epub creator: %w", err)
		}
		b.WriteString("<dc:creator>")
		if err := xml.EscapeText(&b, []byte(c)); err != nil {
			return nil, err
		}
		b.WriteString("</dc:creator>\n")
	}
	if doc.Meta.Language != "" {
		b.WriteString("<dc:language>" + xmlEscapeAttr(doc.Meta.Language) + "</dc:language>\n")
	}
	if doc.Meta.Publisher != "" {
		b.WriteString("<dc:publisher>" + xmlEscapeAttr(doc.Meta.Publisher) + "</dc:publisher>\n")
	}
	if doc.Meta.Date != "" {
		b.WriteString("<dc:date>" + xmlEscapeAttr(doc.Meta.Date) + "</dc:date>\n")
	}
	b.WriteString("</metadata>\n<manifest>\n")
	for _, p := range parts {
		b.WriteString(`<item id="` + xmlEscapeAttr(p.id) + `" href="` + xmlEscapeAttr(p.entry[len("OEBPS/"):]) +
			`" media-type="` + xmlEscapeAttr(p.mt) + `"/>` + "\n")
	}
	b.WriteString(`<item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>` + "\n</manifest>\n")
	b.WriteString(`<spine toc="ncx">` + "\n")
	for _, p := range parts {
		if p.mt == "application/xhtml+xml" {
			b.WriteString(`<itemref idref="` + xmlEscapeAttr(p.id) + `"/>` + "\n")
		}
	}
	b.WriteString("</spine>\n</package>\n")
	return []byte(b.String()), nil
}

func epubNCX(chapters []*hdm.Chapter) ([]byte, error) {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	b.WriteString(`<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1"><navMap>`)
	for i, ch := range chapters {
		b.WriteString(fmt.Sprintf(`<navPoint id="np%d" playOrder="%d"><navLabel><text>%s</text></navLabel><content src="%s"/></navPoint>`,
			i, i+1, xmlEscapeAttr(titleFromBlocks(ch)), xmlEscapeAttr(ch.ID+".xhtml")))
	}
	b.WriteString("</navMap></ncx>\n")
	return []byte(b.String()), nil
}
