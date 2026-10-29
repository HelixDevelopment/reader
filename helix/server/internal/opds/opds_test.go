package opds

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/helixreader/helix/server/internal/catalog"
)

func fixturePage() catalog.Page {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	return catalog.Page{Total: 3, Offset: 0, Limit: 1, Works: []catalog.Work{{ID: "w1", CanonicalTitle: "A work", Subjects: []string{"Fiction"}, UpdatedAt: now, Editions: []catalog.Edition{{ID: "e1", Title: "A work — first edition", Authors: []string{"A. Author"}, Language: "en", Description: "Description", State: catalog.Identified, UpdatedAt: now, Identifiers: []catalog.Identifier{{Scheme: "isbn", Value: "9780441478125"}}, CoverHREF: "/v1/covers/e1", Files: []catalog.File{{ID: "f1", Format: "epub", MIMEType: "application/epub+zip"}}}, {ID: "e2", Title: "A work — revised edition", Authors: []string{"A. Author"}, UpdatedAt: now, Files: []catalog.File{{ID: "f2", Format: "pdf", MIMEType: "application/pdf"}}}}}}}
}

func TestOPDS12WellFormedMultiEditionAndPaging(t *testing.T) {
	raw, err := Acquisition12("/opds/1.2/publications", fixturePage(), catalog.Query{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	var f Feed12
	if err := xml.Unmarshal(raw, &f); err != nil {
		t.Fatalf("not well formed: %v\n%s", err, raw)
	}
	if len(f.Entries) != 2 {
		t.Fatalf("two editions must remain distinct: %d", len(f.Entries))
	}
	if f.TotalResults != 3 || f.ItemsPerPage != 1 {
		t.Fatalf("wrong paging metadata: %+v", f)
	}
	if !bytes.Contains(raw, []byte(`rel="next"`)) {
		t.Fatal("next link missing")
	}
	for _, e := range f.Entries {
		acquire := false
		for _, l := range e.Links {
			if l.Rel == RelAcquisition && l.Type != "" {
				acquire = true
			}
		}
		if !acquire {
			t.Errorf("entry %s lacks typed acquisition", e.ID)
		}
	}
}

func TestOPDS20WellFormedAndTyped(t *testing.T) {
	raw, err := Acquisition20("/opds/2.0/publications", fixturePage(), catalog.Query{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	var f Feed2
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	if len(f.Publications) != 2 {
		t.Fatalf("editions collapsed: %d", len(f.Publications))
	}
	if f.Metadata.NumberOfItems != 3 {
		t.Fatalf("total missing: %+v", f.Metadata)
	}
	for _, p := range f.Publications {
		if p.Metadata.Identifier == "" || len(p.Links) < 2 {
			t.Errorf("incomplete publication: %+v", p)
		}
		acquire := false
		for _, l := range p.Links {
			if l.Rel == RelAcquisition && l.Type != "" {
				acquire = true
			}
		}
		if !acquire {
			t.Errorf("%s lacks acquisition", p.Metadata.Identifier)
		}
	}
}

func TestStandaloneEditionDocuments(t *testing.T) {
	w := fixturePage().Works[0]
	raw12, err := Edition12(w, w.Editions[0])
	if err != nil {
		t.Fatal(err)
	}
	var entry Entry12
	if err := xml.Unmarshal(raw12, &entry); err != nil {
		t.Fatal(err)
	}
	if entry.XMLName.Local != "entry" || entry.ID != "urn:helix:edition:e1" {
		t.Fatalf("invalid OPDS entry: %s", raw12)
	}
	raw20, err := Edition20(w, w.Editions[0])
	if err != nil {
		t.Fatal(err)
	}
	var publication Publication2
	if err := json.Unmarshal(raw20, &publication); err != nil {
		t.Fatal(err)
	}
	if publication.Context == "" || publication.Metadata.Identifier != "urn:helix:edition:e1" {
		t.Fatalf("invalid RWPM: %s", raw20)
	}
}

func TestOpenSearchTemplateEscaped(t *testing.T) {
	raw, err := OpenSearch12("")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "{searchTerms}") || !strings.Contains(string(raw), "&amp;author=") {
		t.Fatalf("invalid OpenSearch template: %s", raw)
	}
	var document struct {
		XMLName xml.Name
		URL     struct {
			Template string `xml:"template,attr"`
		} `xml:"Url"`
	}
	if err := xml.Unmarshal(raw, &document); err != nil {
		t.Fatalf("OpenSearch XML is not well formed: %v", err)
	}
	if document.XMLName.Local != "OpenSearchDescription" || document.URL.Template == "" {
		t.Fatalf("incomplete OpenSearch document: %s", raw)
	}
}
