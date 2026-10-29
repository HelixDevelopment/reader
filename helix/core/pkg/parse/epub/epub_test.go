package epub

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	"github.com/helixreader/helix/core/pkg/parse/parseutil"
)

// buildTestEPUB creates a minimal valid EPUB in memory.
func buildTestEPUB(t *testing.T) *bytes.Reader {
	t.Helper()
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	add := func(name, content string) {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(content))
	}
	add("META-INF/container.xml", `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`)
	add("OEBPS/content.opf", `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="id">
 <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
  <dc:identifier id="id" opf:scheme="ISBN" xmlns:opf="http://www.idpf.org/2007/opf">9780441478125</dc:identifier>
  <dc:title>Test   Book</dc:title>
  <dc:creator>Test Author</dc:creator>
  <dc:language>en</dc:language>
 </metadata>
 <manifest>
  <item id="ch1" href="ch1.xhtml" media-type="application/xhtml+xml"/>
  <item id="ch2" href="ch2.xhtml" media-type="application/xhtml+xml"/>
  <item id="img1" href="cover.png" media-type="image/png"/>
 </manifest>
 <spine><itemref idref="ch1"/><itemref idref="ch2"/></spine>
</package>`)
	add("OEBPS/ch1.xhtml", `<html><body><h1>Chapter   One</h1><p>Hello   world.
Second line.</p><p>Another para.</p></body></html>`)
	add("OEBPS/ch2.xhtml", `<html><body><h2>Part Two</h2><blockquote>A quote.</blockquote></body></html>`)
	add("OEBPS/cover.png", "\x89PNG-fake")
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(buf.Bytes())
}

func TestParseEPUB(t *testing.T) {
	r := buildTestEPUB(t)
	doc, err := Parser{}.Parse(r, int64(r.Len()))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if doc.Format != "epub" {
		t.Fatal("format tag")
	}
	if doc.Meta.Title != "Test   Book" { // metadata is NOT whitespace-normalised by design
		t.Fatalf("title: %q", doc.Meta.Title)
	}
	if doc.Meta.Identifiers["isbn"] != "9780441478125" {
		t.Fatalf("isbn: %v", doc.Meta.Identifiers)
	}
	if len(doc.Chapters) != 2 {
		t.Fatalf("chapters: %d", len(doc.Chapters))
	}
	c1 := doc.Chapters[0]
	if c1.Title != "Chapter One" {
		t.Fatalf("chapter title from heading: %q", c1.Title)
	}
	// whitespace normalised (F-A4)
	if c1.Blocks[1].Text != "Hello world. Second line." {
		t.Fatalf("normalised text: %q", c1.Blocks[1].Text)
	}
	if doc.Chapters[1].Blocks[1].Kind.String() != "quote" {
		t.Fatalf("blockquote kind: %v", doc.Chapters[1].Blocks[1].Kind)
	}
	if len(doc.Resources) != 1 {
		t.Fatalf("resources: %d", len(doc.Resources))
	}
	if got := doc.WordCount(); got != 12 { // ch1: 2+4+2, ch2: 2+2
		t.Fatalf("word count: %d", got)
	}
}

func TestXHTMLRetainsListTableAndDivContent(t *testing.T) {
	blocks := xhtmlToBlocks(`<html><body><div>bare text</div><ul><li>one</li><li>two</li></ul><table><tr><td>A</td><td>B</td></tr></table></body></html>`)
	expected := []string{"bare text", "one", "two", "A B"}
	if len(blocks) != len(expected) {
		t.Fatalf("blocks: %+v", blocks)
	}
	for i := range expected {
		if blocks[i].Text != expected[i] {
			t.Fatalf("block %d = %q, expected %q", i, blocks[i].Text, expected[i])
		}
	}
}

func TestParsePercentEncodedManifestHrefs(t *testing.T) {
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	add := func(name, content string) {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	add("META-INF/container.xml", `<container><rootfiles><rootfile full-path="OEBPS/content.opf"/></rootfiles></container>`)
	add("OEBPS/content.opf", `<package><manifest><item id="ch" href="ch%201.xhtml" media-type="application/xhtml+xml"/><item id="cover" href="cover%20one.png" media-type="image/png"/></manifest><spine><itemref idref="ch"/></spine></package>`)
	add("OEBPS/ch 1.xhtml", `<html><body><p>decoded path</p></body></html>`)
	add("OEBPS/cover one.png", "image")
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	r := bytes.NewReader(buf.Bytes())
	doc, err := (Parser{}).Parse(r, int64(r.Len()))
	if err != nil || len(doc.Chapters) != 1 || doc.Chapters[0].Blocks[0].Text != "decoded path" || string(doc.Resources["cover"]) != "image" {
		t.Fatalf("percent-encoded href resolution failed: %+v %v", doc, err)
	}
}

func TestParseCorruptZip(t *testing.T) {
	r := bytes.NewReader([]byte("not a zip"))
	if _, err := (Parser{}).Parse(r, 9); err == nil {
		t.Fatal("corrupt zip must error")
	}
}

func TestParseMissingRootfile(t *testing.T) {
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	w, _ := zw.Create("META-INF/container.xml")
	w.Write([]byte(`<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles/></container>`))
	zw.Close()
	r := bytes.NewReader(buf.Bytes())
	if _, err := (Parser{}).Parse(r, int64(r.Len())); err == nil {
		t.Fatal("missing rootfile must error")
	}
}

func TestParseRejectsOversizedRequiredEntries(t *testing.T) {
	for _, target := range []string{"META-INF/container.xml", "OEBPS/ch1.xhtml"} {
		t.Run(target, func(t *testing.T) {
			buf := &bytes.Buffer{}
			zw := zip.NewWriter(buf)
			add := func(name, content string) {
				w, err := zw.Create(name)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := w.Write([]byte(content)); err != nil {
					t.Fatal(err)
				}
			}
			container := `<container><rootfiles><rootfile full-path="OEBPS/content.opf"/></rootfiles></container>`
			chapter := `<html><body><p>small</p></body></html>`
			if target == "META-INF/container.xml" {
				container = strings.Repeat("x", int(parseutil.MaxEntryBytes+1))
			}
			if target == "OEBPS/ch1.xhtml" {
				chapter = strings.Repeat("x", int(parseutil.MaxEntryBytes+1))
			}
			add("META-INF/container.xml", container)
			add("OEBPS/content.opf", `<package><manifest><item id="ch1" href="ch1.xhtml" media-type="application/xhtml+xml"/></manifest><spine><itemref idref="ch1"/></spine></package>`)
			add("OEBPS/ch1.xhtml", chapter)
			if err := zw.Close(); err != nil {
				t.Fatal(err)
			}
			r := bytes.NewReader(buf.Bytes())
			_, err := (Parser{}).Parse(r, int64(r.Len()))
			if err == nil || !strings.Contains(err.Error(), "exceeds") {
				t.Fatalf("oversized required entry must be rejected, got %v", err)
			}
		})
	}
}

func FuzzParse(f *testing.F) {
	f.Add([]byte("not a zip"))
	f.Add([]byte("PK\x03\x04"))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			t.Skip("bound fuzz input memory")
		}
		_, _ = (Parser{}).Parse(bytes.NewReader(data), int64(len(data)))
	})
}
