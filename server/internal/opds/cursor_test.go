package opds

import (
	"encoding/xml"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/helixreader/helix/server/internal/catalog"
)

// cursorPage mirrors what a repository returns for a middle page of a three-work
// catalog: the page itself carries HasNext, and the total is the whole set.
func cursorPage(at time.Time) catalog.Page {
	work := func(id string) catalog.Work {
		return catalog.Work{ID: id, CanonicalTitle: "Walk " + id, UpdatedAt: at, Editions: []catalog.Edition{{ID: "e" + id, Title: "Walk " + id, State: catalog.Identified, UpdatedAt: at, Files: []catalog.File{{ID: "f" + id, Format: "epub", MIMEType: "application/epub+zip"}}}}}
	}
	return catalog.Page{Works: []catalog.Work{work("w2"), work("w3")}, Total: 3, Limit: 2, HasNext: true}
}

func queryValues(target string) url.Values {
	raw := target[strings.Index(target, "?")+1:]
	v, err := url.ParseQuery(raw)
	if err != nil {
		panic(err)
	}
	return v
}

func linkFor(t *testing.T, raw []byte, rel string) string {
	t.Helper()
	var f Feed12
	if err := xml.Unmarshal(raw, &f); err != nil {
		t.Fatalf("not well formed: %v\n%s", err, raw)
	}
	for _, l := range f.Links {
		if l.Rel == rel {
			return l.HREF
		}
	}
	return ""
}

func TestAcquisition12CursorLinksRoundTrip(t *testing.T) {
	at := time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC)
	page := cursorPage(at)
	anchor := catalog.CursorAfter(page.Works)
	q := catalog.Query{Limit: 2, After: anchor}
	raw, err := Acquisition12("/opds/1.2/publications", page, q)
	if err != nil {
		t.Fatal(err)
	}
	self := linkFor(t, raw, "self")
	next := linkFor(t, raw, "next")
	if self == "" || next == "" {
		t.Fatalf("self and next must both be present: %q %q", self, next)
	}
	if got := queryValues(self).Get("after"); got != anchor.Encode() {
		t.Fatalf("self link must reproduce the requested cursor: %q", got)
	}
	if got := queryValues(next).Get("after"); got != catalog.CursorAfter(page.Works).Encode() {
		t.Fatalf("next link must point past the last work: %q", got)
	}
	// The offsets are meaningless once a cursor is in play, and emitting both
	// would let a client that copies the link get the cursor it asked for and an
	// offset it did not.
	if queryValues(next).Has("offset") {
		t.Fatalf("cursor link must not carry an offset: %s", next)
	}
	if linkFor(t, raw, "previous") != "" {
		t.Fatal("a cursor page cannot address the page before it")
	}
	// The emitted cursor must be the very value the API accepts again.
	parsed, err := catalog.ParseCursor(queryValues(next).Get("after"))
	if err != nil {
		t.Fatalf("next link carries an unusable cursor: %v", err)
	}
	if parsed.ID != "w3" || !parsed.UpdatedAt.Equal(at) {
		t.Fatalf("next cursor decoded to %+v", parsed)
	}
}

func TestAcquisition20CursorLinksRoundTrip(t *testing.T) {
	at := time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC)
	page := cursorPage(at)
	q := catalog.Query{Limit: 2, After: catalog.CursorAfter(page.Works)}
	raw, err := Acquisition20("/opds/2.0/publications", page, q)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, `"rel": "next"`) {
		t.Fatalf("next link missing: %s", text)
	}
	if strings.Contains(text, "offset=") {
		t.Fatalf("cursor feed must not advertise offsets: %s", text)
	}
	// numberOfItems is the filtered total, which the repository still reports, so
	// a client that reads it is not misled by the absence of an offset.
	if !strings.Contains(text, `"numberOfItems": 3`) {
		t.Fatalf("total missing: %s", text)
	}
}

func TestNextLinkFollowsHasNextNotTheTotal(t *testing.T) {
	at := time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC)
	// The count claims more rows than exist. Trusting it would advertise a page
	// that cannot be served, which is the failure this rule exists to prevent.
	page := cursorPage(at)
	page.HasNext = false
	for _, q := range []catalog.Query{
		{Limit: 2},
		{Limit: 2, Offset: 0},
		{Limit: 1, Offset: 9},
	} {
		raw, err := Acquisition12("/opds/1.2/publications", page, q)
		if err != nil {
			t.Fatal(err)
		}
		if linkFor(t, raw, "next") != "" {
			t.Fatalf("HasNext false must not emit a next link (query %+v)", q)
		}
		raw2, err := Acquisition20("/opds/2.0/publications", page, q)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw2), `"rel": "next"`) {
			t.Fatalf("2.0 emitted a next link with HasNext false (query %+v)", q)
		}
	}
	// And the converse: HasNext true in offset mode must advance by exactly one
	// page width.
	page.HasNext = true
	raw, err := Acquisition12("/opds/1.2/publications", page, catalog.Query{Limit: 2, Offset: 4})
	if err != nil {
		t.Fatal(err)
	}
	if got := queryValues(linkFor(t, raw, "next")).Get("offset"); got != "6" {
		t.Fatalf("offset next link = %s, want 6", got)
	}
	if got := linkFor(t, raw, "previous"); got == "" || queryValues(got).Get("offset") != "2" {
		t.Fatalf("offset previous link = %q, want offset 2", got)
	}
}
