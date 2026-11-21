package opds

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/helixreader/helix/core/pkg/pse"
	"github.com/helixreader/helix/server/internal/catalog"
)

func comicPage() catalog.Page {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	return catalog.Page{Total: 1, Limit: 1, Works: []catalog.Work{{ID: "w1", CanonicalTitle: "Comic", UpdatedAt: now, Editions: []catalog.Edition{{
		ID: "e1", Title: "Comic", UpdatedAt: now, State: catalog.Identified,
		Files: []catalog.File{{ID: "f1", Format: "cbz", MIMEType: "application/vnd.comicbook+zip"}, {ID: "f2", Format: "txt", MIMEType: "text/plain"}},
	}}}}}
}

var comicStreams = map[string]PageStream{"f1": {Count: 35, Type: "image/jpeg"}}

func TestStreamLinkRoundTripsThroughTheParser(t *testing.T) {
	raw, err := Acquisition12("/opds/1.2/publications", comicPage(), catalog.Query{Limit: 1}, comicStreams)
	if err != nil {
		t.Fatal(err)
	}
	links, err := pse.ParseAcquisitionLinks(raw)
	if err != nil {
		t.Fatalf("the feed this package produces must satisfy core/pkg/pse: %v\n%s", err, raw)
	}
	if len(links) != 1 {
		t.Fatalf("got %d stream links, want 1: %+v", len(links), links)
	}
	l := links[0]
	if l.Count != 35 || l.Type != "image/jpeg" || l.Rel != pse.StreamRel {
		t.Errorf("parsed link = %+v, want count 35, image/jpeg, stream rel", l)
	}
	if !strings.Contains(l.Href, pse.PageNumberTemplate) {
		t.Errorf("href %q must carry the %s template", l.Href, pse.PageNumberTemplate)
	}
	// Every page the advertised count promises must be addressable, and no
	// page beyond it.
	first, err := l.PageURL(0, 0)
	if err != nil {
		t.Fatalf("page 0: %v", err)
	}
	if !strings.HasSuffix(first, "/0") {
		t.Errorf("page 0 URL = %q, want it to end in the page number", first)
	}
	if _, err := l.PageURL(34, 0); err != nil {
		t.Errorf("last advertised page: %v", err)
	}
	if _, err := l.PageURL(35, 0); err == nil {
		t.Error("a page beyond the advertised count must be refused by the template")
	}
}

func TestFeedDeclaresThePSENamespace(t *testing.T) {
	raw, err := Acquisition12("/opds/1.2/publications", comicPage(), catalog.Query{Limit: 1}, comicStreams)
	if err != nil {
		t.Fatal(err)
	}
	feedTag := string(raw)[:strings.Index(string(raw), ">")+1]
	if !strings.Contains(feedTag, `xmlns:pse="`+pse.NamespaceURI+`"`) {
		t.Errorf("the feed element must declare the namespace, got %s", feedTag)
	}
	if strings.Count(string(raw), `xmlns:pse=`) != 1 {
		t.Errorf("the namespace should be declared once, on the feed:\n%s", raw)
	}
}

// The producer enforces the same MUSTs the parser enforces, so a resolver bug
// upstream cannot put an illegal link on the wire.
func TestIllegalStreamsAreNeverAdvertised(t *testing.T) {
	tests := []struct {
		name    string
		streams map[string]PageStream
	}{
		{name: "no pages", streams: map[string]PageStream{"f1": {Count: 0, Type: "image/jpeg"}}},
		{name: "negative pages", streams: map[string]PageStream{"f1": {Count: -3, Type: "image/jpeg"}}},
		{name: "not an image type", streams: map[string]PageStream{"f1": {Count: 3, Type: "application/zip"}}},
		{name: "empty type", streams: map[string]PageStream{"f1": {Count: 3, Type: ""}}},
		{name: "no stream resolved", streams: nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := Acquisition12("/opds/1.2/publications", comicPage(), catalog.Query{Limit: 1}, tc.streams)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(raw), pse.StreamRel) {
				t.Errorf("feed advertises a stream it cannot describe:\n%s", raw)
			}
			// The ordinary acquisition link must survive: refusing to
			// advertise pages must not make the file undeliverable.
			if !strings.Contains(string(raw), `/v1/files/f1/content`) {
				t.Error("whole-file acquisition link went missing")
			}
		})
	}
}

// A stream is advertised for the files the caller resolved and for no others;
// unresolved files keep their whole-file acquisition link.
func TestOnlyResolvedFilesGetLinks(t *testing.T) {
	raw, err := Acquisition12("/opds/1.2/publications", comicPage(), catalog.Query{Limit: 1},
		map[string]PageStream{"f1": {Count: 9, Type: "image/png"}})
	if err != nil {
		t.Fatal(err)
	}
	var f Feed12
	if err := xml.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	streamed := 0
	for _, e := range f.Entries {
		for _, l := range e.Links {
			if l.Rel != pse.StreamRel {
				continue
			}
			streamed++
			if !strings.Contains(l.HREF, "/f1/") {
				t.Errorf("stream link %q was advertised for a file with no resolved stream", l.HREF)
			}
		}
	}
	if streamed != 1 {
		t.Fatalf("got %d stream links, want exactly 1", streamed)
	}
}

func TestEditionEntryStandaloneCarriesNamespaceAndLink(t *testing.T) {
	page := comicPage()
	w := page.Works[0]
	raw, err := Edition12(w, w.Editions[0], comicStreams)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `xmlns:pse="`+pse.NamespaceURI+`"`) {
		t.Errorf("a standalone entry must declare the namespace itself:\n%s", raw)
	}
	links, err := pse.ParseAcquisitionLinks(raw)
	if err != nil {
		t.Fatalf("standalone entry does not satisfy core/pkg/pse: %v\n%s", err, raw)
	}
	if len(links) != 1 || links[0].Count != 35 {
		t.Fatalf("parsed %+v, want one 35-page link", links)
	}
}

// The extension defines an Atom acquisition link inside an OPDS entry; nothing
// in v1.0-v1.2 describes an OPDS 2.0 JSON form, so none is invented here.
func TestOPDS20InventsNoPageStream(t *testing.T) {
	page := comicPage()
	raw, err := Acquisition20("/opds/2.0/publications", page, catalog.Query{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "vaemendis") {
		t.Errorf("OPDS 2.0 output carries a PSE member the specification does not define:\n%s", raw)
	}
	entry, err := Edition20(page.Works[0], page.Works[0].Editions[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(entry), "vaemendis") {
		t.Errorf("OPDS 2.0 entry carries a PSE member:\n%s", entry)
	}
}

func TestPageStreamHREFShape(t *testing.T) {
	got := PageStreamHREF("f/1 x")
	if !strings.HasPrefix(got, PageStreamPrefix) || !strings.HasSuffix(got, pse.PageNumberTemplate) {
		t.Fatalf("href = %q, want %s...%s", got, PageStreamPrefix, pse.PageNumberTemplate)
	}
	if strings.Contains(got, pse.MaxWidthTemplate) {
		t.Errorf("href %q advertises {maxWidth}, which would promise resizing this server does not do", got)
	}
	if strings.Contains(got, "f/1 x") {
		t.Errorf("file id must be escaped into the path: %q", got)
	}
	if !strings.Contains(got, "f%2F1%20x") {
		t.Errorf("escaped file id missing from %q", got)
	}
}
