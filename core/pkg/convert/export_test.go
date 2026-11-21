package convert

import (
	"archive/zip"
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"

	"github.com/helixreader/helix/core/pkg/hdm"
	"github.com/helixreader/helix/core/pkg/parse/cbz"
	"github.com/helixreader/helix/core/pkg/parse/epub"
)

// Valid single-pixel images encoded with the stdlib: fake payloads would
// prove nothing about the sniffing path.
func fixturePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 10, G: 120, B: 220, A: 255})
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	if _, err := png.Decode(bytes.NewReader(b.Bytes())); err != nil {
		t.Fatalf("png fixture invalid: %v", err)
	}
	return b.Bytes()
}

func fixtureJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 200, G: 30, B: 30, A: 255})
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := jpeg.Decode(bytes.NewReader(b.Bytes())); err != nil {
		t.Fatalf("jpeg fixture invalid: %v", err)
	}
	return b.Bytes()
}

func exportDoc() *hdm.Document {
	return &hdm.Document{
		Format: "txt",
		Meta: hdm.Metadata{
			Title: "Round Trip", Creators: []string{"Ada & Co"}, Language: "en",
			Identifiers: map[string]string{},
		},
		Chapters: []*hdm.Chapter{
			{ID: "c1", Title: "Chapter One", Blocks: []hdm.Block{
				{Kind: hdm.BlockHeading, Level: 1, Text: "Chapter One"},
				{Kind: hdm.BlockParagraph, Text: "Hello <world> & friends."},
				{Kind: hdm.BlockQuote, Text: "A quote."},
				{Kind: hdm.BlockSeparator},
			}},
			{ID: "c2", Title: "Chapter Two", Blocks: []hdm.Block{
				{Kind: hdm.BlockHeading, Level: 2, Text: "Chapter Two"},
				{Kind: hdm.BlockParagraph, Text: "Second body."},
			}},
		},
		Resources: map[string][]byte{},
	}
}

func TestExportEPUBStructure(t *testing.T) {
	out, err := Convert(exportDoc(), FormatEPUB, DefaultOptions)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(out), int64(len(out)))
	if err != nil {
		t.Fatal(err)
	}
	if zr.File[0].Name != "mimetype" || zr.File[0].Method != zip.Store {
		t.Fatalf("first entry must be stored mimetype, got %q method %d", zr.File[0].Name, zr.File[0].Method)
	}
	rc, _ := zr.File[0].Open()
	var mt bytes.Buffer
	mt.ReadFrom(rc)
	rc.Close()
	if mt.String() != "application/epub+zip" {
		t.Fatalf("mimetype: %q", mt.String())
	}
	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
	}
	for _, want := range []string{"META-INF/container.xml", "OEBPS/content.opf", "OEBPS/toc.ncx", "OEBPS/c1.xhtml", "OEBPS/c2.xhtml"} {
		if !names[want] {
			t.Fatalf("missing entry %s in %v", want, names)
		}
	}
}

// TestExportEPUBRoundTripExact proves the exporter/parser pair is a real
// interconversion: every block kind, escaped text, separator, metadata value
// and fingerprint must come back byte-exact.
func TestExportEPUBRoundTripExact(t *testing.T) {
	doc := exportDoc()
	out, err := Convert(doc, FormatEPUB, DefaultOptions)
	if err != nil {
		t.Fatal(err)
	}
	r := bytes.NewReader(out)
	got, err := epub.Parser{}.Parse(r, int64(r.Len()))
	if err != nil {
		t.Fatalf("re-parse of exported epub failed: %v", err)
	}
	if got.Format != "epub" || got.Meta.Title != "Round Trip" || got.Meta.Language != "en" {
		t.Fatalf("meta: %+v", got.Meta)
	}
	if len(got.Meta.Creators) != 1 || got.Meta.Creators[0] != "Ada & Co" {
		t.Fatalf("creators: %v", got.Meta.Creators)
	}
	if got.Meta.Identifiers["uuid"] != doc.Fingerprint() {
		t.Fatalf("identifier: %v", got.Meta.Identifiers)
	}
	if len(got.Chapters) != 2 {
		t.Fatalf("chapters: %d", len(got.Chapters))
	}
	wantBlocks := [][]hdm.Block{
		doc.Chapters[0].Blocks,
		doc.Chapters[1].Blocks,
	}
	for ci, want := range wantBlocks {
		gotCh := got.Chapters[ci]
		if gotCh.Title != doc.Chapters[ci].Title {
			t.Fatalf("chapter %d title: %q", ci, gotCh.Title)
		}
		if len(gotCh.Blocks) != len(want) {
			t.Fatalf("chapter %d block count: want %d got %+v", ci, len(want), gotCh.Blocks)
		}
		for bi := range want {
			if gotCh.Blocks[bi].Kind != want[bi].Kind || gotCh.Blocks[bi].Text != want[bi].Text || gotCh.Blocks[bi].Level != want[bi].Level {
				t.Fatalf("chapter %d block %d: got %+v want %+v", ci, bi, gotCh.Blocks[bi], want[bi])
			}
		}
	}
	// Fingerprint is content-derived: an exact round trip reproduces it.
	if got.Fingerprint() != doc.Fingerprint() {
		t.Fatal("fingerprint drift after round trip")
	}
}

func TestExportEPUBDeterministicBytes(t *testing.T) {
	a, err := Convert(exportDoc(), FormatEPUB, DefaultOptions)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Convert(exportDoc(), FormatEPUB, DefaultOptions)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("epub export not byte-deterministic")
	}
}

func TestExportEPUBWithImagesRoundTripsResources(t *testing.T) {
	pngBytes, jpgBytes := fixturePNG(t), fixtureJPEG(t)
	doc := exportDoc()
	doc.Chapters[0].Blocks = append(doc.Chapters[0].Blocks,
		hdm.Block{Kind: hdm.BlockImage, ImageRef: "art1"},
		hdm.Block{Kind: hdm.BlockImage, ImageRef: "art2"})
	doc.Resources["art1"] = pngBytes
	doc.Resources["art2"] = jpgBytes
	out, err := Convert(doc, FormatEPUB, DefaultOptions)
	if err != nil {
		t.Fatal(err)
	}
	r := bytes.NewReader(out)
	got, err := epub.Parser{}.Parse(r, int64(r.Len()))
	if err != nil {
		t.Fatal(err)
	}
	// Manifest ids are "res-<id>"; payloads must be byte-identical.
	if !bytes.Equal(got.Resources["res-art1"], pngBytes) || !bytes.Equal(got.Resources["res-art2"], jpgBytes) {
		t.Fatalf("resources: %v", keys(got.Resources))
	}
	// The same HDM document packaged as CBZ must carry identical page bytes.
	pack, err := Convert(doc, FormatCBZ, DefaultOptions)
	if err != nil {
		t.Fatal(err)
	}
	cr := bytes.NewReader(pack)
	cdoc, err := cbz.Parser{}.Parse(cr, int64(cr.Len()))
	if err != nil {
		t.Fatalf("exported cbz must re-parse: %v", err)
	}
	if !bytes.Equal(cdoc.Resources["page-0001"], pngBytes) || !bytes.Equal(cdoc.Resources["page-0002"], jpgBytes) {
		t.Fatalf("cbz page payloads: %v", keys(cdoc.Resources))
	}
}

func keys(m map[string][]byte) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestExportEPUBFailClosed(t *testing.T) {
	pngBytes := fixturePNG(t)
	imageDoc := func(data []byte, id string) *hdm.Document {
		d := exportDoc()
		d.Chapters[1].Blocks = append(d.Chapters[1].Blocks, hdm.Block{Kind: hdm.BlockImage, ImageRef: id})
		d.Resources[id] = data
		return d
	}
	tests := []struct {
		name    string
		doc     *hdm.Document
		wantErr string
	}{
		{"code block", withBlock(hdm.Block{Kind: hdm.BlockCode, Text: "x := 1"}), "lossless"},
		{"table block", withBlock(hdm.Block{Kind: hdm.BlockTable, Text: "a"}), "lossless"},
		{"list block", withBlock(hdm.Block{Kind: hdm.BlockList, Text: "item"}), "lossless"},
		{"footnote ref", withBlock(hdm.Block{Kind: hdm.BlockFootnoteRef, Text: "[1]"}), "lossless"},
		{"heading level 9", withBlock(hdm.Block{Kind: hdm.BlockHeading, Level: 9, Text: "deep"}), "out of range"},
		{"unnormalised text", withBlock(hdm.Block{Kind: hdm.BlockParagraph, Text: "two  spaces"}), "whitespace-normalised"},
		{"trailing space", withBlock(hdm.Block{Kind: hdm.BlockParagraph, Text: "trailing "}), "whitespace-normalised"},
		{"control char", withBlock(hdm.Block{Kind: hdm.BlockParagraph, Text: "bad\x01char"}), "control character"},
		{"empty paragraph", withBlock(hdm.Block{Kind: hdm.BlockParagraph, Text: ""}), "empty text"},
		{"non-image payload", imageDoc([]byte("\x89PNG-fake-synthetic"), "art"), "neither PNG nor JPEG"},
		{"webp payload", imageDoc([]byte("RIFFxxxxWEBVPayload"), "art"), "neither PNG nor JPEG"},
		{"path separator in ref", func() *hdm.Document {
			d := exportDoc()
			d.Chapters[1].Blocks = append(d.Chapters[1].Blocks, hdm.Block{Kind: hdm.BlockImage, ImageRef: "a/b"})
			d.Resources["a/b"] = pngBytes
			return d
		}(), "invalid image resource id"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Convert(tc.doc, FormatEPUB, DefaultOptions)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("want error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

// TestExportEPUBTitleInjection: a chapter title not represented by a leading
// heading is injected as h1 so it survives the container round trip.
func TestExportEPUBTitleInjection(t *testing.T) {
	d := exportDoc()
	d.Chapters[1].Title = "Different"
	out, err := Convert(d, FormatEPUB, DefaultOptions)
	if err != nil {
		t.Fatal(err)
	}
	r := bytes.NewReader(out)
	got, err := epub.Parser{}.Parse(r, int64(r.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if got.Chapters[1].Title != "Different" {
		t.Fatalf("title lost: %q", got.Chapters[1].Title)
	}
	if got.Chapters[1].Blocks[0].Kind != hdm.BlockHeading || got.Chapters[1].Blocks[0].Text != "Different" {
		t.Fatalf("injected heading: %+v", got.Chapters[1].Blocks[0])
	}
	if got.Chapters[1].Blocks[1].Text != "Chapter Two" {
		t.Fatalf("original heading lost: %+v", got.Chapters[1].Blocks)
	}
}

func withBlock(b hdm.Block) *hdm.Document {
	d := exportDoc()
	d.Chapters[1].Blocks = append(d.Chapters[1].Blocks, b)
	return d
}

func TestExportCBZFailClosed(t *testing.T) {
	noImages := exportDoc()
	if _, err := Convert(noImages, FormatCBZ, DefaultOptions); err == nil || !strings.Contains(err.Error(), "no image blocks") {
		t.Fatalf("want no image blocks error, got %v", err)
	}
	fake := exportDoc()
	fake.Chapters[0].Blocks = append(fake.Chapters[0].Blocks, hdm.Block{Kind: hdm.BlockImage, ImageRef: "im"})
	fake.Resources["im"] = []byte("x")
	if _, err := Convert(fake, FormatCBZ, DefaultOptions); err == nil || !strings.Contains(err.Error(), "neither PNG nor JPEG") {
		t.Fatalf("want encoding rejection, got %v", err)
	}
}

// FuzzExportEPUB follows the existing FuzzParse naming convention: arbitrary
// HDM documents must never panic the exporter; refusals are legal outcomes.
func FuzzExportEPUB(f *testing.F) {
	f.Add("title", "body text", 4)
	f.Add("T", "a  b", 0)
	f.Add("<esc>&amp;", "quote", 2)
	f.Fuzz(func(t *testing.T, title, text string, budget int) {
		doc := &hdm.Document{
			Format: "fuzz",
			Meta:   hdm.Metadata{Title: title, Identifiers: map[string]string{}},
			Chapters: []*hdm.Chapter{{ID: "c1", Blocks: []hdm.Block{
				{Kind: hdm.BlockParagraph, Text: text},
			}}},
			Resources: map[string][]byte{},
		}
		_, _ = Convert(doc, FormatEPUB, DefaultOptions)
	})
}
