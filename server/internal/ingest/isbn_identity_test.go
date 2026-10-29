package ingest

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/helixreader/helix/server/internal/catalog"
	"github.com/helixreader/helix/server/internal/identify"
	"github.com/helixreader/helix/server/internal/sources"
)

func TestISBNIdentityCanonicalizesSchemePairAndOrder(t *testing.T) {
	isbn10 := catalog.Edition{Identifiers: []catalog.Identifier{{Scheme: "isbn10", Value: "0-306-40615-2"}}}
	isbn13 := catalog.Edition{Identifiers: []catalog.Identifier{{Scheme: "isbn13", Value: "978-0-306-40615-7"}}}
	if identityID(isbn10, "a") != identityID(isbn13, "b") || workIdentityID(isbn10, "a") != workIdentityID(isbn13, "b") {
		t.Fatal("ISBN-10 and equivalent ISBN-13 forked identity")
	}
	forward := catalog.Edition{Identifiers: []catalog.Identifier{
		{Scheme: "isbn", Value: "9780441478125"},
		{Scheme: "isbn", Value: "9780306406157"},
	}}
	reverse := catalog.Edition{Identifiers: []catalog.Identifier{forward.Identifiers[1], forward.Identifiers[0]}}
	if identityID(forward, "a") != identityID(reverse, "a") || workIdentityID(forward, "a") != workIdentityID(reverse, "a") {
		t.Fatal("identifier order changed identity")
	}
	invalid := catalog.Edition{Identifiers: []catalog.Identifier{{Scheme: "isbn", Value: "9780441478126"}}}
	if identityID(invalid, "content") != stableID("edition", "content") {
		t.Fatal("invalid ISBN displaced content identity")
	}
}

func epubWithCanonicalISBN(t *testing.T, value string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	files := []struct{ name, body string }{
		{"mimetype", "application/epub+zip"},
		{"META-INF/container.xml", `<?xml version="1.0"?><container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`},
		{"OEBPS/content.opf", `<?xml version="1.0"?><package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="id"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>EPUB ISBN</dc:title><dc:identifier id="id" opf:scheme="ISBN" xmlns:opf="http://www.idpf.org/2007/opf">` + value + `</dc:identifier></metadata><manifest><item id="c1" href="c1.xhtml" media-type="application/xhtml+xml"/></manifest><spine><itemref idref="c1"/></spine></package>`},
		{"OEBPS/c1.xhtml", `<html xmlns="http://www.w3.org/1999/xhtml"><body><p>hello chapter</p></body></html>`},
	}
	for _, file := range files {
		w, err := zw.Create(file.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(file.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestEPUBISBNHyphenationDoesNotForkIdentity(t *testing.T) {
	for _, withPipeline := range []bool{false, true} {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "a.epub"), epubWithCanonicalISBN(t, "9780441478125"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "b.epub"), epubWithCanonicalISBN(t, "978-0-441-47812-5"), 0o644); err != nil {
			t.Fatal(err)
		}
		connector, err := sources.NewLocal(sources.Config{Root: root})
		if err != nil {
			t.Fatal(err)
		}
		repo := catalog.NewMemory()
		service := Service{Repository: repo}
		if withPipeline {
			service.Identifier = &identify.Pipeline{}
		}
		report, err := service.ScanSource(context.Background(), "src", connector, "")
		if err != nil || report.Files != 2 {
			t.Fatalf("pipeline=%v report=%+v err=%v", withPipeline, report, err)
		}
		page, err := repo.Search(context.Background(), catalog.Query{Limit: 100})
		if err != nil {
			t.Fatal(err)
		}
		if page.Total != 1 || len(page.Works[0].Editions) != 1 || len(page.Works[0].Editions[0].Files) != 2 {
			t.Fatalf("pipeline=%v: equivalent ISBNs forked catalog identity: %+v", withPipeline, page)
		}
	}
}
