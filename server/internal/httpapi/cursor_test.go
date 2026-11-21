package httpapi

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/helixreader/helix/server/internal/catalog"
	"github.com/helixreader/helix/server/internal/opds"
)

// walkTime is the instant four of the five seeded works share, and walkAt the
// only one that differs. The collision is the point: a cursor that mishandles
// equal instants repeats or drops a row over HTTP instead of in a unit test.
var walkAt = time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
var walkTime = walkAt.Add(time.Minute)

// seedWalk installs the fixture and returns the order every pagination mode must
// reproduce: newest first, ties broken by ascending identifier.
func seedWalk(t *testing.T, repo *catalog.Memory) []string {
	t.Helper()
	ctx := context.Background()
	at := walkAt
	ids := []string{"walk-5", "walk-1", "walk-3", "walk-2", "walk-4"}
	for _, id := range ids {
		updated := at
		if id == "walk-4" {
			updated = walkTime
		}
		work := catalog.Work{ID: id, CanonicalTitle: "Walk " + id, CreatedAt: at, UpdatedAt: updated, Editions: []catalog.Edition{{ID: "e-" + id, Title: "Walk " + id, State: catalog.Identified, UpdatedAt: updated}}}
		if err := repo.UpsertWork(ctx, work); err != nil {
			t.Fatalf("seed %s: %v", id, err)
		}
	}
	return []string{"walk-4", "walk-1", "walk-2", "walk-3", "walk-5"}
}

func decodePage(t *testing.T, body []byte) catalog.Page {
	t.Helper()
	var page catalog.Page
	if err := json.Unmarshal(body, &page); err != nil {
		t.Fatalf("page is not JSON: %v\n%s", err, body)
	}
	return page
}

func TestWorksCursorWalkOverHTTP(t *testing.T) {
	s, repo, _ := testServer(t)
	defer s.Close()
	r := s.Router()
	want := append(seedWalk(t, repo), "w1")

	target := "/v1/works?limit=2"
	var got []string
	for hops := 0; ; hops++ {
		if hops > len(want) {
			t.Fatalf("cursor walk over HTTP did not terminate after %d pages", hops)
		}
		w := request(t, r, http.MethodGet, target, "secret", nil, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("GET %s: %d %s", target, w.Code, w.Body.String())
		}
		page := decodePage(t, w.Body.Bytes())
		for _, work := range page.Works {
			got = append(got, work.ID)
		}
		if !page.HasNext {
			break
		}
		cursor := catalog.CursorAfter(page.Works)
		if cursor == nil {
			t.Fatal("has_next true with an empty page")
		}
		target = "/v1/works?limit=2&after=" + cursor.Encode()
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("cursor walk = %v, want %v", got, want)
	}
	// The same set must come out of the offset walk the API already publishes.
	var viaOffset []string
	for off := 0; ; off += 2 {
		w := request(t, r, http.MethodGet, "/v1/works?limit=2&offset="+strconv.Itoa(off), "secret", nil, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("offset page: %d", w.Code)
		}
		page := decodePage(t, w.Body.Bytes())
		for _, work := range page.Works {
			viaOffset = append(viaOffset, work.ID)
		}
		if !page.HasNext {
			break
		}
	}
	if strings.Join(viaOffset, ",") != strings.Join(want, ",") {
		t.Fatalf("offset walk = %v, want %v", viaOffset, want)
	}
	// Cursor mode still reports the filtered total, because the paging method does
	// not change what a client reading "total" expects.
	w := request(t, r, http.MethodGet, "/v1/works?limit=2&after="+catalog.Cursor{UpdatedAt: walkAt, ID: "walk-1"}.Encode(), "secret", nil, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("cursor page: %d %s", w.Code, w.Body.String())
	}
	page := decodePage(t, w.Body.Bytes())
	if page.Total != len(want) {
		t.Fatalf("total in cursor mode = %d, want %d", page.Total, len(want))
	}
	if len(page.Works) != 2 || page.Works[0].ID != "walk-2" || page.Works[1].ID != "walk-3" {
		t.Fatalf("cursor page contents = %+v", page.Works)
	}
	if !page.HasNext {
		t.Fatal("rows remain after walk-3 but has_next is false")
	}
	// A cursor at the last row is an empty page, not an error and not a repeat.
	tail := request(t, r, http.MethodGet, "/v1/works?limit=2&after="+catalog.Cursor{UpdatedAt: walkAt, ID: "walk-5"}.Encode(), "secret", nil, nil)
	tailPage := decodePage(t, tail.Body.Bytes())
	if len(tailPage.Works) != 1 || tailPage.Works[0].ID != "w1" {
		t.Fatalf("tail page = %+v, want only w1", tailPage.Works)
	}
	if tailPage.HasNext {
		t.Fatal("last page claims another follows")
	}
}

func TestWorksRejectsMalformedCursor(t *testing.T) {
	s, repo, _ := testServer(t)
	defer s.Close()
	r := s.Router()
	seedWalk(t, repo)
	for _, value := range []string{"not-a-cursor", strings.Repeat("A", 400), "MQ%7C"} {
		w := request(t, r, http.MethodGet, "/v1/works?after="+value, "secret", nil, nil)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("after=%q: %d %s, want 400", value, w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "cursor") {
			t.Fatalf("after=%q: error must name the parameter: %s", value, w.Body.String())
		}
	}
	// An escape the query string cannot be decoded from is not a missing
	// parameter: answering it with the first page would report success to a client
	// whose cursor arrived corrupted.
	w := request(t, r, http.MethodGet, "/v1/works?after=%", "secret", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("undecodable query string: %d %s, want 400", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "query string") {
		t.Fatalf("undecodable query string must be named: %s", w.Body.String())
	}
	// Both OPDS feeds share the same parser and must reject it the same way.
	for _, path := range []string{"/opds/1.2/publications", "/opds/2.0/publications"} {
		w := request(t, r, http.MethodGet, path+"?after=not-a-cursor", "secret", nil, nil)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("%s: %d %s, want 400", path, w.Code, w.Body.String())
		}
	}
}

// feedLinks is the pair of projections the two OPDS versions expose; both are
// walked the same way, because a reader application follows whichever it speaks.
type feedLinks struct {
	path  string
	links func(*testing.T, []byte) []opds.Link2
	ids   func(*testing.T, []byte) []string
}

func TestOPDSFeedLinksAreFollowable(t *testing.T) {
	feeds := []feedLinks{
		{
			path: "/opds/1.2/publications",
			links: func(t *testing.T, raw []byte) []opds.Link2 {
				t.Helper()
				var f opds.Feed12
				if err := xml.Unmarshal(raw, &f); err != nil {
					t.Fatalf("1.2 feed is not well formed: %v\n%s", err, raw)
				}
				out := make([]opds.Link2, 0, len(f.Links))
				for _, l := range f.Links {
					out = append(out, opds.Link2{HREF: l.HREF, Rel: l.Rel, Type: l.Type})
				}
				return out
			},
			ids: func(t *testing.T, raw []byte) []string {
				t.Helper()
				var f opds.Feed12
				if err := xml.Unmarshal(raw, &f); err != nil {
					t.Fatal(err)
				}
				out := []string{}
				for _, e := range f.Entries {
					out = append(out, strings.TrimPrefix(e.ID, "urn:helix:edition:"))
				}
				return out
			},
		},
		{
			path: "/opds/2.0/publications",
			links: func(t *testing.T, raw []byte) []opds.Link2 {
				t.Helper()
				var f opds.Feed2
				if err := json.Unmarshal(raw, &f); err != nil {
					t.Fatalf("2.0 feed is not JSON: %v\n%s", err, raw)
				}
				return f.Links
			},
			ids: func(t *testing.T, raw []byte) []string {
				t.Helper()
				var f opds.Feed2
				if err := json.Unmarshal(raw, &f); err != nil {
					t.Fatal(err)
				}
				out := []string{}
				for _, p := range f.Publications {
					out = append(out, strings.TrimPrefix(p.Metadata.Identifier, "urn:helix:edition:"))
				}
				return out
			},
		},
	}
	for _, feed := range feeds {
		// A fresh server per walk: the seeded catalog must not be disturbed by the
		// other pagination mode.
		s, repo, _ := testServer(t)
		want := append(seedWalk(t, repo), "w1")
		wantEditions := []string{}
		for _, id := range want {
			if id == "w1" {
				wantEditions = append(wantEditions, "e1")
				continue
			}
			wantEditions = append(wantEditions, "e-"+id)
		}
		for _, mode := range []string{"offset", "cursor"} {
			target := feed.path + "?limit=2"
			if mode == "cursor" {
				target += "&after=" + catalog.Cursor{UpdatedAt: walkAt, ID: "walk-1"}.Encode()
			}
			seen := []string{}
			for hops := 0; hops <= len(want); hops++ {
				w := request(t, s.Router(), http.MethodGet, target, "secret", nil, nil)
				if w.Code != http.StatusOK {
					t.Fatalf("%s %s: %d %s", feed.path, mode, w.Code, w.Body.String())
				}
				body := w.Body.Bytes()
				seen = append(seen, feed.ids(t, body)...)
				next := ""
				for _, l := range feed.links(t, body) {
					if l.Rel == "next" {
						next = l.HREF
					}
				}
				if next == "" {
					break
				}
				if mode == "cursor" && !strings.Contains(next, "after=") {
					t.Fatalf("%s cursor mode must continue by cursor, got %s", feed.path, next)
				}
				if mode == "cursor" && strings.Contains(next, "offset=") {
					t.Fatalf("%s cursor link carries an offset: %s", feed.path, next)
				}
				target = next
				if hops == len(want) {
					t.Fatalf("%s %s: next-link walk did not terminate", feed.path, mode)
				}
			}
			expect := wantEditions
			if mode == "cursor" {
				// The walk started after the first page's two works.
				expect = wantEditions[2:]
			}
			if strings.Join(seen, ",") != strings.Join(expect, ",") {
				t.Fatalf("%s %s walk produced %v, want %v", feed.path, mode, seen, expect)
			}
		}
		s.Close()
	}
}
