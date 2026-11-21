// End-to-end conversion chains: parse(format A) -> HDM -> export(format B)
// -> parse(B) with exact expected values at every hop, plus pagination of
// parsed documents through sync.Paginate (positions compatible with the sync
// protocol). These are the executable proofs behind docs/formats.md.
package convert_test

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"image"
	"image/png"
	"strings"
	"testing"

	"github.com/helixreader/helix/core/pkg/convert"
	"github.com/helixreader/helix/core/pkg/hdm"
	"github.com/helixreader/helix/core/pkg/parse/cbz"
	"github.com/helixreader/helix/core/pkg/parse/epub"
	"github.com/helixreader/helix/core/pkg/parse/fb2"
	"github.com/helixreader/helix/core/pkg/parse/mobi"
	"github.com/helixreader/helix/core/pkg/parse/txt"
	psync "github.com/helixreader/helix/core/pkg/sync"
)

const txtSample = `THE WIDGET BOOK

Chapter 1

Alice wrote the widget book in a small town by the sea.
It was not a long book, but it was hers.

Chapter 2

The second chapter opens on a rainy morning.
Alice counts her words and finds them wanting.
`

// minimal uncompressed PalmDOC container with one text record.
func mobiBytes(t *testing.T, text string) []byte {
	t.Helper()
	rec0 := make([]byte, 16)
	binary.BigEndian.PutUint16(rec0[0:2], 1) // no compression
	binary.BigEndian.PutUint32(rec0[4:8], uint32(len(text)))
	binary.BigEndian.PutUint16(rec0[8:10], 1) // one text record
	records := [][]byte{rec0, []byte(text)}
	var out bytes.Buffer
	hdr := make([]byte, 78)
	copy(hdr, "Chain MOBI")
	binary.BigEndian.PutUint16(hdr[76:78], uint16(len(records)))
	out.Write(hdr)
	off := 78 + len(records)*8
	for _, r := range records {
		var b [8]byte
		binary.BigEndian.PutUint32(b[0:4], uint32(off))
		out.Write(b[:])
		off += len(r)
	}
	for _, r := range records {
		out.Write(r)
	}
	return out.Bytes()
}

func parseTxt(t *testing.T) *hdm.Document {
	t.Helper()
	doc, err := txt.Parser{}.Parse(strings.NewReader(txtSample))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

// TestTxtToEPUBFullChain walks txt -> fb2 -> epub -> txt and asserts the
// exact final text, proving each hop is a real interconversion.
func TestTxtToEPUBFullChain(t *testing.T) {
	doc := parseTxt(t)

	fb2Out, err := convert.Convert(doc, convert.FormatFB2, convert.DefaultOptions)
	if err != nil {
		t.Fatal(err)
	}
	doc2, err := fb2.Parser{}.Parse(bytes.NewReader(fb2Out))
	if err != nil {
		t.Fatalf("exported fb2 must re-parse: %v", err)
	}
	if doc2.Meta.Title != "THE WIDGET BOOK" || len(doc2.Chapters) != 3 {
		t.Fatalf("fb2 hop meta/chapters: %q %d", doc2.Meta.Title, len(doc2.Chapters))
	}

	epubOut, err := convert.Convert(doc2, convert.FormatEPUB, convert.DefaultOptions)
	if err != nil {
		t.Fatalf("fb2->epub: %v", err)
	}
	if !bytes.HasPrefix(epubOut, []byte("PK\x03\x04")) {
		t.Fatal("epub export is not a zip")
	}
	r := bytes.NewReader(epubOut)
	doc3, err := epub.Parser{}.Parse(r, int64(r.Len()))
	if err != nil {
		t.Fatalf("exported epub must re-parse: %v", err)
	}

	finalTxt, err := convert.Convert(doc3, convert.FormatTXT, convert.DefaultOptions)
	if err != nil {
		t.Fatal(err)
	}
	want := `THE WIDGET BOOK

# THE WIDGET BOOK

# Chapter 1

Alice wrote the widget book in a small town by the sea. It was not a long book, but it was hers.

# Chapter 2

The second chapter opens on a rainy morning. Alice counts her words and finds them wanting.

`
	if string(finalTxt) != want {
		t.Fatalf("chain drift, got:\n%s\nwant:\n%s", finalTxt, want)
	}
}

// TestMobiToEPUBThenPaginate: MOBI parses to one spine chapter; export to
// EPUB and back must preserve every paragraph, and pagination of the
// re-parsed document must agree with sync positions exactly.
func TestMobiToEPUBThenPaginate(t *testing.T) {
	text := "<p>One two three four five six seven.</p><p>Eight nine ten.</p><p>Eleven twelve.</p>"
	raw := mobiBytes(t, text)
	mdoc, err := mobi.Parser{}.Parse(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	epubOut, err := convert.Convert(mdoc, convert.FormatEPUB, convert.DefaultOptions)
	if err != nil {
		t.Fatalf("mobi->epub: %v", err)
	}
	r := bytes.NewReader(epubOut)
	edoc, err := epub.Parser{}.Parse(r, int64(r.Len()))
	if err != nil {
		t.Fatalf("epub re-parse: %v", err)
	}
	var texts []string
	for _, b := range edoc.Chapters[0].Blocks {
		texts = append(texts, b.Text)
	}
	want := []string{"Chain MOBI", "One two three four five six seven.", "Eight nine ten.", "Eleven twelve."}
	if len(texts) != len(want) {
		t.Fatalf("texts: %v", texts)
	}
	for i := range want {
		if texts[i] != want[i] {
			t.Fatalf("block %d: %q want %q", i, texts[i], want[i])
		}
	}

	pages, err := psync.Paginate(edoc, 40)
	if err != nil {
		t.Fatal(err)
	}
	// Packing (rune counts 10, 34, 15, 14 at budget 40): page0={heading},
	// page1={first paragraph}, page2={last two paragraphs}.
	if len(pages) != 3 {
		t.Fatalf("pages: %+v", pages)
	}
	if pages[0].Start != (psync.Position{Chapter: 0, Block: 0}) || pages[1].Start != (psync.Position{Chapter: 0, Block: 1}) ||
		pages[2].Start != (psync.Position{Chapter: 0, Block: 2}) {
		t.Fatalf("page starts: %+v", pages)
	}
	// Progress sync across pages uses the same Position ordering.
	resumed := psync.Furthest(pages[1].Start, pages[0].Start)
	if resumed != pages[1].Start {
		t.Fatal("page positions must be ordered by sync.Furthest")
	}
	if got := psync.PageOf(pages, psync.Position{Chapter: 0, Block: 2}); got != 2 {
		t.Fatalf("PageOf: %d", got)
	}
}

// TestEPUBToCBZRepack extracts the image pages of an EPUB into a comic
// archive whose enumeration matches the source payloads byte for byte.
func TestEPUBToCBZRepack(t *testing.T) {
	img := onePixelPNG(t)
	doc := &hdm.Document{
		Format: "epub",
		Meta:   hdm.Metadata{Title: "Gallery", Identifiers: map[string]string{}},
		Chapters: []*hdm.Chapter{{ID: "plates", Blocks: []hdm.Block{
			{Kind: hdm.BlockImage, ImageRef: "p1"},
			{Kind: hdm.BlockImage, ImageRef: "p2"},
		}}},
		Resources: map[string][]byte{"p1": img, "p2": img},
	}
	pack, err := convert.Convert(doc, convert.FormatCBZ, convert.DefaultOptions)
	if err != nil {
		t.Fatal(err)
	}
	refs, err := cbz.Pages(bytes.NewReader(pack), int64(len(pack)))
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 2 || refs[0].ID != "page-0001" || refs[1].ID != "page-0002" {
		t.Fatalf("refs: %+v", refs)
	}
	cdoc, err := cbz.Parser{}.Parse(bytes.NewReader(pack), int64(len(pack)))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(cdoc.Resources["page-0001"], img) || !bytes.Equal(cdoc.Resources["page-0002"], img) {
		t.Fatal("cbz repack must preserve payloads exactly")
	}
}

// TestFB2BinariesToEPUBImages: decoded FB2 binaries survive an EPUB export
// with byte-identical payloads.
func TestFB2BinariesToEPUBImages(t *testing.T) {
	img := onePixelPNG(t)
	src := `<?xml version="1.0"?><FictionBook>` +
		`<description><title-info><book-title>Ill</book-title></title-info></description>` +
		`<body><section><title><p>S</p></title><p>Text.</p><image href="#art"/></section></body>` +
		`<binary id="art" content-type="image/png">` + base64.StdEncoding.EncodeToString(img) + `</binary></FictionBook>`
	fdoc, err := fb2.Parser{}.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(fdoc.Chapters[0].Blocks) != 3 || fdoc.Chapters[0].Blocks[2].Kind != hdm.BlockImage {
		t.Fatalf("fb2 blocks: %+v", fdoc.Chapters[0].Blocks)
	}
	epubOut, err := convert.Convert(fdoc, convert.FormatEPUB, convert.DefaultOptions)
	if err != nil {
		t.Fatalf("fb2->epub with image: %v", err)
	}
	r := bytes.NewReader(epubOut)
	edoc, err := epub.Parser{}.Parse(r, int64(r.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(edoc.Resources["res-art"], img) {
		t.Fatal("image payload drifted through fb2->epub")
	}
	// The exported zip must contain the image entry.
	zr, err := zip.NewReader(bytes.NewReader(epubOut), int64(len(epubOut)))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range zr.File {
		if f.Name == "OEBPS/res/art.png" {
			found = true
		}
	}
	if !found {
		t.Fatal("image entry missing from exported epub")
	}
}

// TestTxtPaginateExact asserts exact page geometry for a parsed TXT book.
func TestTxtPaginateExact(t *testing.T) {
	doc := parseTxt(t)
	pages, err := psync.Paginate(doc, 120)
	if err != nil {
		t.Fatal(err)
	}
	// Deterministic greedy packing over the exact parsed block texts.
	if len(pages) == 0 || pages[0].Start != (psync.Position{Chapter: 0, Block: 0}) {
		t.Fatalf("pages: %+v", pages)
	}
	total := 0
	for _, p := range pages {
		total += p.Blocks
		if p.Chars > 120 && p.Blocks != 1 {
			t.Fatalf("page over budget without being a single oversized block: %+v", p)
		}
	}
	flat := doc.FlatTOC()
	wantBlocks := 0
	for _, ch := range flat {
		wantBlocks += len(ch.Blocks)
	}
	if total != wantBlocks {
		t.Fatalf("pagination dropped blocks: %d vs %d", total, wantBlocks)
	}
	// Exact first-page geometry: heading "THE WIDGET BOOK" (15) +
	// "Chapter 1" (9) + long paragraph (96) = 120 <= 120; the next
	// heading (9) does not fit. Page 2: "Chapter 2" (9) + paragraph (91).
	if pages[0].Blocks != 3 || pages[0].Chars != 120 {
		t.Fatalf("first page: %+v", pages[0])
	}
	if len(pages) != 2 {
		t.Fatalf("page count: %+v", pages)
	}
	if pages[1].Blocks != 2 || pages[1].Chars != 100 {
		t.Fatalf("second page: %+v", pages[1])
	}
}

func onePixelPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewGray(image.Rect(0, 0, 1, 1))
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	if _, err := png.Decode(bytes.NewReader(b.Bytes())); err != nil {
		t.Fatalf("png fixture invalid: %v", err)
	}
	return b.Bytes()
}
