package opds

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"net/url"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/helixreader/helix/server/internal/catalog"
)

// xmlSafe reports whether every rune of s survives an XML round trip unchanged
// (valid UTF-8, inside the XML 1.0 character range, no carriage return).
func xmlSafe(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		ok := r == 0x9 || r == 0xA || r >= 0x20 && r <= 0xD7FF || r >= 0xE000 && r <= 0xFFFD || r >= 0x10000 && r <= 0x10FFFF
		if !ok {
			return false
		}
	}
	return true
}

type atomLink struct {
	Rel  string `xml:"rel,attr"`
	HREF string `xml:"href,attr"`
	Type string `xml:"type,attr"`
}
type atomEntry struct {
	Title string     `xml:"title"`
	ID    string     `xml:"id"`
	Links []atomLink `xml:"link"`
}
type atomFeed struct {
	Title   string      `xml:"title"`
	Entries []atomEntry `xml:"entry"`
}
type jsonLink struct {
	HREF string `json:"href"`
	Rel  any    `json:"rel"`
	Type string `json:"type"`
}
type jsonMeta struct {
	Title         string `json:"title"`
	Identifier    string `json:"identifier"`
	NumberOfItems int    `json:"numberOfItems"`
}
type jsonPub struct {
	Context  string     `json:"@context"`
	Metadata jsonMeta   `json:"metadata"`
	Links    []jsonLink `json:"links"`
}
type jsonFeed struct {
	Context      string    `json:"@context"`
	Metadata     jsonMeta  `json:"metadata"`
	Publications []jsonPub `json:"publications"`
}

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
	f.Add("A title", "Au", "d", "f/1?x", "application/epub+zip", "a&b", "a/b?c", 5, 0, 1, false)
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
		acqHref := "/v1/files/" + url.PathEscape(fileID) + "/content"
		// Atom contract (RULES.md table O): feed title, one entry whose title,
		// id and acquisition link round-trip through XML escaping.
		checkEntry := func(name string, e atomEntry) {
			if xmlSafe(title) && e.Title != title {
				t.Fatalf("%s: entry title %q != %q", name, e.Title, title)
			}
			if xmlSafe(id) && e.ID != "urn:helix:edition:"+id {
				t.Fatalf("%s: entry id %q", name, e.ID)
			}
			n := 0
			for _, l := range e.Links {
				if l.Rel == RelAcquisition {
					n++
					if l.HREF != acqHref {
						t.Fatalf("%s: acquisition href %q != %q", name, l.HREF, acqHref)
					}
					if xmlSafe(mime) && l.Type != mime {
						t.Fatalf("%s: acquisition type %q != %q", name, l.Type, mime)
					}
				}
			}
			if n != 1 {
				t.Fatalf("%s: %d acquisition links, want 1", name, n)
			}
		}
		raw, err := Acquisition12("/opds/1.2/publications", page, q, streams)
		check12("Acquisition12", raw, err)
		if err == nil {
			var fd atomFeed
			if e := xml.Unmarshal(raw, &fd); e != nil || fd.Title != "Helix Publications" || len(fd.Entries) != 1 {
				t.Fatalf("Acquisition12 feed: title %q entries %d (%v)", fd.Title, len(fd.Entries), e)
			}
			checkEntry("Acquisition12", fd.Entries[0])
		}
		raw, err = Acquisition12("/opds/1.2/publications", page, q, nil)
		check12("Acquisition12/nostream", raw, err)
		raw, err = Edition12(page.Works[0], page.Works[0].Editions[0], streams)
		check12("Edition12", raw, err)
		if err == nil {
			var e atomEntry
			if x := xml.Unmarshal(raw, &e); x != nil {
				t.Fatalf("Edition12: %v", x)
			}
			checkEntry("Edition12", e)
		}
		raw, err = OpenSearch12(text)
		check12("OpenSearch12", raw, err)
		if err == nil && xmlSafe(text) {
			var d struct {
				ShortName string `xml:"ShortName"`
				URL       struct {
					Template string `xml:"template,attr"`
				} `xml:"Url"`
			}
			if x := xml.Unmarshal(raw, &d); x != nil || d.ShortName != "Helix" || d.URL.Template != text+"/opds/1.2/publications?q={searchTerms}&author={atom:author}" {
				t.Fatalf("OpenSearch12: template %q (%v)", d.URL.Template, x)
			}
		}
		// OPDS 2.0 contract: required members present and round-tripping.
		checkPub := func(name string, p jsonPub) {
			if utf8.ValidString(title) && p.Metadata.Title != title {
				t.Fatalf("%s: metadata.title %q != %q", name, p.Metadata.Title, title)
			}
			if utf8.ValidString(id) && p.Metadata.Identifier != "urn:helix:edition:"+id {
				t.Fatalf("%s: identifier %q", name, p.Metadata.Identifier)
			}
			selfs, acqs := 0, 0
			for _, l := range p.Links {
				switch l.Rel {
				case "self":
					selfs++
					if l.HREF != "/opds/2.0/editions/"+url.PathEscape(id) {
						t.Fatalf("%s: self href %q", name, l.HREF)
					}
				case RelAcquisition:
					acqs++
					if l.HREF != acqHref || (utf8.ValidString(mime) && l.Type != mime) {
						t.Fatalf("%s: acquisition link %+v", name, l)
					}
				}
			}
			if selfs != 1 || acqs != 1 {
				t.Fatalf("%s: self=%d acquisition=%d links, want 1/1", name, selfs, acqs)
			}
		}
		raw, err = Acquisition20("/opds/2.0/publications", page, q)
		check2("Acquisition20", raw, err)
		if err == nil {
			var fd jsonFeed
			if e := json.Unmarshal(raw, &fd); e != nil || fd.Metadata.Title != "Helix Publications" || fd.Context == "" ||
				fd.Metadata.NumberOfItems != total || len(fd.Publications) != 1 {
				t.Fatalf("Acquisition20 feed: %+v (%v)", fd.Metadata, e)
			}
			checkPub("Acquisition20", fd.Publications[0])
		}
		raw, err = Edition20(page.Works[0], page.Works[0].Editions[0])
		check2("Edition20", raw, err)
		if err == nil {
			var p jsonPub
			if e := json.Unmarshal(raw, &p); e != nil || p.Context != "https://readium.org/webpub-manifest/context.jsonld" {
				t.Fatalf("Edition20: context %q (%v)", p.Context, e)
			}
			checkPub("Edition20", p)
		}
		if raw, err = Navigation12(); err == nil && !bytes.HasPrefix(bytes.TrimSpace(raw), []byte("<")) {
			t.Fatalf("Navigation12 not XML")
		}
	})
}
