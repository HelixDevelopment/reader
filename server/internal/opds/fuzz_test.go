package opds

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"testing"
	"time"

	"github.com/helixreader/helix/server/internal/catalog"
)

func fuzzPage(title, author, desc, fileID, mime, text, id string, total, offset, limit int, hasNext bool) (catalog.Page, catalog.Query) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	ed := catalog.Edition{ID: id, Title: title, Authors: []string{author}, Language: "en", Description: desc,
		State: catalog.Identified, UpdatedAt: now, CoverHREF: "/v1/covers/" + id,
		Files: []catalog.File{{ID: fileID, Format: "epub", MIMEType: mime}}}
	w := catalog.Work{ID: id, CanonicalTitle: title, Subjects: []string{desc}, UpdatedAt: now, Editions: []catalog.Edition{ed}}
	return catalog.Page{Total: total, Offset: offset, Limit: limit, HasNext: hasNext, Works: []catalog.Work{w}},
		catalog.Query{Text: text, Author: author, Offset: offset, Limit: limit}
}

// FuzzOPDSGeneration (HRD-070): for arbitrary text in every catalog string
// field, the OPDS 1.2 (Atom/XML) and 2.0 (JSON) projections must never panic
// and every successful output must be well formed XML / valid JSON.
func FuzzOPDSGeneration(f *testing.F) {
	f.Add("A work", "A. Author", "Description", "f1", "application/epub+zip", "q", "e1", 3, 0, 1, true)
	f.Add("<&>\"'", "\x00\x01\x1f", "]]>", "../x?y#z", "a/b;c=d", "%00&x=<", "e&1", -1, -5, 0, false)
	f.Add("\xff\xfe", " ", "\U0010FFFF", "{pageNumber}", "", "é", "", 1<<30, 1<<30, 1<<30, true)
	f.Fuzz(func(t *testing.T, title, author, desc, fileID, mime, text, id string, total, offset, limit int, hasNext bool) {
		if len(title)+len(author)+len(desc)+len(fileID)+len(mime)+len(text)+len(id) > 1<<16 {
			t.Skip("bound input")
		}
		page, q := fuzzPage(title, author, desc, fileID, mime, text, id, total, offset, limit, hasNext)
		streams := map[string]PageStream{fileID: {Count: limit, Type: mime}}
		check12 := func(name string, raw []byte, err error) {
			if err != nil {
				return
			}
			var v struct{ XMLName xml.Name }
			if e := xml.Unmarshal(raw, &v); e != nil {
				t.Fatalf("%s: not well formed XML: %v\n%q", name, e, raw)
			}
		}
		check2 := func(name string, raw []byte, err error) {
			if err != nil {
				return
			}
			if !json.Valid(raw) {
				t.Fatalf("%s: invalid JSON: %q", name, raw)
			}
		}
		raw, err := Acquisition12("/opds/1.2/publications", page, q, streams)
		check12("Acquisition12", raw, err)
		raw, err = Acquisition12("/opds/1.2/publications", page, q, nil)
		check12("Acquisition12/nostream", raw, err)
		raw, err = Edition12(page.Works[0], page.Works[0].Editions[0], streams)
		check12("Edition12", raw, err)
		raw, err = OpenSearch12(text)
		check12("OpenSearch12", raw, err)
		raw, err = Acquisition20("/opds/2.0/publications", page, q)
		check2("Acquisition20", raw, err)
		raw, err = Edition20(page.Works[0], page.Works[0].Editions[0])
		check2("Edition20", raw, err)
		if raw, err = Navigation12(); err == nil && !bytes.HasPrefix(bytes.TrimSpace(raw), []byte("<")) {
			t.Fatalf("Navigation12 not XML")
		}
	})
}
