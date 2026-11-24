package opds

import (
	"encoding/json"
	"encoding/xml"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/helixreader/helix/server/internal/catalog"
)

// Deterministic, fixture-driven tests of every link and field the OPDS
// generators produce (RULES.md table O). Each assertion decodes the generated
// bytes with an independent decoder and compares against the fixture, so a
// mutant that drops, renames or mis-computes a link fails the plain seed run.

type xLink struct {
	Rel   string `xml:"rel,attr"`
	HREF  string `xml:"href,attr"`
	Type  string `xml:"type,attr"`
	Title string `xml:"title,attr"`
}
type xEntry struct {
	Identifiers []string `xml:"identifier"`
	Title       string   `xml:"title"`
	ID          string   `xml:"id"`
	Authors     []string `xml:"author>name"`
	Language    string   `xml:"language"`
	Publisher   string   `xml:"publisher"`
	Summary     string   `xml:"summary"`
	Categories  []struct {
		Term string `xml:"term,attr"`
	} `xml:"category"`
	Links []xLink `xml:"link"`
}
type xFeed struct {
	Updated string   `xml:"updated"`
	ID      string   `xml:"id"`
	Title   string   `xml:"title"`
	Links   []xLink  `xml:"link"`
	Entries []xEntry `xml:"entry"`
}

func richEdition() (catalog.Work, catalog.Edition) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	ed := catalog.Edition{ID: "e/1 x", Title: "Title & Co", Authors: []string{"Ann Author", "Bob Writer"}, Publisher: "Pub House", Published: "2020-05-06",
		Language: "en", Description: "A description", CoverHREF: "/v1/covers/e1", UpdatedAt: now,
		Identifiers: []catalog.Identifier{{Scheme: "isbn", Value: "9780000000002"}},
		Files: []catalog.File{
			{ID: "f/lcp", Format: "epub", MIMEType: "application/epub+zip", DRMScheme: "lcp", LicenseHREF: "/v1/licenses/f1"},
			{ID: "f-plain", Format: "cbz", MIMEType: "application/vnd.comicbook+zip"},
			{ID: "f-lcp-nohref", Format: "epub", MIMEType: "application/epub+zip", DRMScheme: "lcp"},
			{ID: "f-other-drm", Format: "epub", MIMEType: "application/epub+zip", DRMScheme: "adept", LicenseHREF: "/v1/licenses/x"},
		}}
	return catalog.Work{ID: "w1", Subjects: []string{"Fiction", "Poetry"}, UpdatedAt: now, Editions: []catalog.Edition{ed}}, ed
}

func linksByRel(ls []xLink, rel string) []xLink {
	var out []xLink
	for _, l := range ls {
		if l.Rel == rel {
			out = append(out, l)
		}
	}
	return out
}

func query(t *testing.T, href string) (string, url.Values) {
	t.Helper()
	u, err := url.Parse(href)
	if err != nil {
		t.Fatalf("href %q: %v", href, err)
	}
	return u.Path, u.Query()
}

func wantQ(t *testing.T, what, href string, want map[string]string) {
	t.Helper()
	path, q := query(t, href)
	if path != "/opds/p" {
		t.Errorf("%s: path %q", what, path)
	}
	if len(q) != len(want) {
		t.Errorf("%s: query %v, want %v", what, q, want)
	}
	for k, v := range want {
		if q.Get(k) != v {
			t.Errorf("%s: %s=%q, want %q (href %s)", what, k, q.Get(k), v, href)
		}
	}
}

func TestHRD070OPDSPaging(t *testing.T) {
	w, _ := richEdition()
	page := func(offset, limit int, hasNext bool) catalog.Page {
		return catalog.Page{Total: 55, Offset: offset, Limit: limit, HasNext: hasNext, Works: []catalog.Work{w}}
	}
	cur := &catalog.Cursor{UpdatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), ID: "c1"}
	type linkSet map[string]string // rel -> expected "k=v&k=v" via wantQ maps encoded below
	cases := []struct {
		name            string
		q               catalog.Query
		page            catalog.Page
		self, prev, nxt map[string]string // nil = link absent
	}{
		{"offset mode, middle page", catalog.Query{Text: "go lang", Author: "A B", State: catalog.NeedsReview, Offset: 20, Limit: 10}, page(20, 10, true),
			map[string]string{"q": "go lang", "author": "A B", "state": "needs_review", "offset": "20", "limit": "10"},
			map[string]string{"q": "go lang", "author": "A B", "state": "needs_review", "offset": "10", "limit": "10"},
			map[string]string{"q": "go lang", "author": "A B", "state": "needs_review", "offset": "30", "limit": "10"}},
		{"previous clamps to 0", catalog.Query{Offset: 5, Limit: 10}, page(5, 10, true),
			map[string]string{"offset": "5", "limit": "10"}, map[string]string{"offset": "0", "limit": "10"}, map[string]string{"offset": "15", "limit": "10"}},
		{"first page has no previous", catalog.Query{Offset: 0, Limit: 10}, page(0, 10, true),
			map[string]string{"offset": "0", "limit": "10"}, nil, map[string]string{"offset": "10", "limit": "10"}},
		{"last page has no next", catalog.Query{Offset: 50, Limit: 10}, page(50, 10, false),
			map[string]string{"offset": "50", "limit": "10"}, map[string]string{"offset": "40", "limit": "10"}, nil},
		{"next follows HasNext, not the total", catalog.Query{Offset: 0, Limit: 100}, page(0, 100, false),
			map[string]string{"offset": "0", "limit": "100"}, nil, nil},
		{"cursor mode: after, no offset, no previous", catalog.Query{Offset: 40, Limit: 10, After: cur}, page(40, 10, true),
			map[string]string{"after": cur.Encode(), "limit": "10"}, nil, map[string]string{"after": catalog.CursorAfter(page(0, 0, false).Works).Encode(), "limit": "10"}},
		{"cursor mode, HasNext but an empty page offers no next cursor", catalog.Query{Limit: 10, After: cur}, catalog.Page{Total: 55, Limit: 10, HasNext: true},
			map[string]string{"after": cur.Encode(), "limit": "10"}, nil, nil},
		{"cursor mode, last page", catalog.Query{Limit: 10, After: cur}, page(0, 10, false),
			map[string]string{"after": cur.Encode(), "limit": "10"}, nil, nil},
	}
	_ = linkSet(nil)
	for _, c := range cases {
		// OPDS 1.2
		raw, err := Acquisition12("/opds/p", c.page, c.q, nil)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		var fd xFeed
		if err := xml.Unmarshal(raw, &fd); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		check := func(kind string, ls []xLink, rel string, want map[string]string) {
			got := linksByRel(ls, rel)
			if want == nil {
				if len(got) != 0 {
					t.Errorf("%s %s: unexpected %s link %v", c.name, kind, rel, got)
				}
				return
			}
			if len(got) != 1 {
				t.Errorf("%s %s: %d %s links, want 1", c.name, kind, len(got), rel)
				return
			}
			wantQ(t, c.name+" "+kind+" "+rel, got[0].HREF, want)
		}
		wantUpdated := "2026-01-02T03:04:05Z"
		if len(c.page.Works) == 0 {
			wantUpdated = "1970-01-01T00:00:00Z"
		}
		if fd.Updated != wantUpdated {
			t.Errorf("%s: feed updated %q, want %q", c.name, fd.Updated, wantUpdated)
		}
		if len(fd.Entries) != len(c.page.Works) {
			t.Errorf("%s: %d entries for %d works", c.name, len(fd.Entries), len(c.page.Works))
		}
		check("1.2", fd.Links, "self", c.self)
		check("1.2", fd.Links, "previous", c.prev)
		check("1.2", fd.Links, "next", c.nxt)
		if len(linksByRel(fd.Links, "start")) != 1 || linksByRel(fd.Links, "start")[0].HREF != "/opds/1.2" {
			t.Errorf("%s: 1.2 start link %v", c.name, linksByRel(fd.Links, "start"))
		}
		if s := linksByRel(fd.Links, "search"); len(s) != 1 || s[0].HREF != "/opds/1.2/opensearch.xml" || s[0].Type != MIMEOpenSearch {
			t.Errorf("%s: 1.2 search link %v", c.name, s)
		}
		// OPDS 2.0
		raw, err = Acquisition20("/opds/p", c.page, c.q)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		var jf struct {
			Publications []json.RawMessage `json:"publications"`
			Links        []struct {
				HREF string `json:"href"`
				Rel  string `json:"rel"`
				Type string `json:"type"`
			} `json:"links"`
		}
		if err := json.Unmarshal(raw, &jf); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if len(jf.Publications) != len(c.page.Works) {
			t.Errorf("%s: 2.0 %d publications for %d works", c.name, len(jf.Publications), len(c.page.Works))
		}
		var jl []xLink
		for _, l := range jf.Links {
			jl = append(jl, xLink{Rel: l.Rel, HREF: l.HREF, Type: l.Type})
		}
		check("2.0", jl, "self", c.self)
		check("2.0", jl, "previous", c.prev)
		check("2.0", jl, "next", c.nxt)
		if st := linksByRel(jl, "start"); len(st) != 1 || st[0].HREF != "/opds/2.0" {
			t.Errorf("%s: 2.0 start link %v", c.name, st)
		}
		for _, rel := range []string{"previous", "next", "self"} {
			for _, l := range linksByRel(jl, rel) {
				if l.Type != MIME2Feed {
					t.Errorf("%s: 2.0 %s link type %q", c.name, rel, l.Type)
				}
			}
		}
	}
}

func TestHRD070OPDSEntry12(t *testing.T) {
	w, e := richEdition()
	streams := map[string]PageStream{"f-plain": {Count: 3, Type: "image/jpeg"}}
	for name, gen := range map[string]func() ([]byte, error){
		"Edition12": func() ([]byte, error) { return Edition12(w, e, streams) },
		"Acquisition12": func() ([]byte, error) {
			return Acquisition12("/opds/p", catalog.Page{Total: 1, Limit: 10, Works: []catalog.Work{w}}, catalog.Query{Limit: 10}, streams)
		},
	} {
		raw, err := gen()
		if err != nil {
			t.Fatal(err)
		}
		var ent xEntry
		if name == "Edition12" {
			err = xml.Unmarshal(raw, &ent)
		} else {
			var fd xFeed
			if err = xml.Unmarshal(raw, &fd); err == nil {
				if len(fd.Entries) != 1 {
					t.Fatalf("%s: %d entries", name, len(fd.Entries))
				}
				ent = fd.Entries[0]
			}
		}
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(ent.Identifiers) != 1 || ent.Identifiers[0] != "urn:isbn:9780000000002" {
			t.Errorf("%s: identifiers %v", name, ent.Identifiers)
		}
		if ent.Title != "Title & Co" || ent.ID != "urn:helix:edition:e/1 x" || ent.Language != "en" || ent.Publisher != "Pub House" || ent.Summary != "A description" {
			t.Errorf("%s: scalar fields %+v", name, ent)
		}
		if len(ent.Authors) != 2 || ent.Authors[0] != "Ann Author" || ent.Authors[1] != "Bob Writer" {
			t.Errorf("%s: authors %v", name, ent.Authors)
		}
		if len(ent.Categories) != 2 || ent.Categories[0].Term != "Fiction" || ent.Categories[1].Term != "Poetry" {
			t.Errorf("%s: categories %v", name, ent.Categories)
		}
		if alt := linksByRel(ent.Links, "alternate"); len(alt) != 1 || alt[0].HREF != "/opds/1.2/editions/e%2F1%20x" || alt[0].Type != MIME12Entry {
			t.Errorf("%s: alternate %v", name, alt)
		}
		if img := linksByRel(ent.Links, RelImage); len(img) != 1 || img[0].HREF != "/v1/covers/e1" || img[0].Type != "image/jpeg" {
			t.Errorf("%s: cover image %v", name, img)
		}
		if th := linksByRel(ent.Links, RelThumbnail); len(th) != 1 || th[0].HREF != "/v1/covers/e1?size=thumb" {
			t.Errorf("%s: thumbnail %v", name, th)
		}
		acq := linksByRel(ent.Links, RelAcquisition)
		wantAcq := []string{"/v1/files/f%2Flcp/content", "/v1/files/f-plain/content", "/v1/files/f-lcp-nohref/content", "/v1/files/f-other-drm/content"}
		if len(acq) != len(wantAcq) {
			t.Fatalf("%s: %d acquisition links", name, len(acq))
		}
		for i, a := range acq {
			if a.HREF != wantAcq[i] || a.Type != e.Files[i].MIMEType {
				t.Errorf("%s: acquisition %d %+v", name, i, a)
			}
		}
		rel := linksByRel(ent.Links, RelRelated)
		if len(rel) != 1 || rel[0].HREF != "/v1/licenses/f1" || rel[0].Type != LCPLicenseMIME || rel[0].Title != "LCP license" {
			t.Errorf("%s: LCP license link must exist exactly for the lcp file with a license href: %v", name, rel)
		}
		if st := linksByRel(ent.Links, "http://vaemendis.net/opds-pse/stream"); len(st) != 1 || st[0].HREF != PageStreamHREF("f-plain") {
			t.Errorf("%s: stream link %v", name, st)
		}
	}
}

func TestHRD070OPDSPublication20(t *testing.T) {
	w, e := richEdition()
	raw, err := Edition20(w, e)
	if err != nil {
		t.Fatal(err)
	}
	var p struct {
		Context  string `json:"@context"`
		Metadata struct {
			Title       string   `json:"title"`
			Identifier  string   `json:"identifier"`
			Language    string   `json:"language"`
			Published   string   `json:"published"`
			Description string   `json:"description"`
			Subjects    []string `json:"subject"`
			Authors     []struct {
				Name string `json:"name"`
			} `json:"author"`
			Publisher struct {
				Name string `json:"name"`
			} `json:"publisher"`
		} `json:"metadata"`
		Links []struct {
			HREF  string `json:"href"`
			Rel   string `json:"rel"`
			Type  string `json:"type"`
			Title string `json:"title"`
		} `json:"links"`
		Images []struct {
			HREF string `json:"href"`
			Rel  string `json:"rel"`
			Type string `json:"type"`
		} `json:"images"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	m := p.Metadata
	if p.Context != "https://readium.org/webpub-manifest/context.jsonld" || m.Title != "Title & Co" || m.Identifier != "urn:helix:edition:e/1 x" || m.Language != "en" ||
		m.Published != "2020-05-06" || m.Description != "A description" || m.Publisher.Name != "Pub House" {
		t.Errorf("scalar fields %+v ctx=%q", m, p.Context)
	}
	if len(m.Authors) != 2 || m.Authors[0].Name != "Ann Author" || m.Authors[1].Name != "Bob Writer" {
		t.Errorf("authors %v", m.Authors)
	}
	if len(m.Subjects) != 2 || m.Subjects[0] != "Fiction" || m.Subjects[1] != "Poetry" {
		t.Errorf("subjects %v", m.Subjects)
	}
	if len(p.Images) != 1 || p.Images[0].HREF != "/v1/covers/e1" || p.Images[0].Rel != "cover" || p.Images[0].Type != "image/jpeg" {
		t.Errorf("cover %+v", p.Images)
	}
	var self, acq, rel int
	for _, l := range p.Links {
		switch l.Rel {
		case "self":
			self++
			if l.HREF != "/opds/2.0/editions/e%2F1%20x" || l.Type != MIME2Publication {
				t.Errorf("self %+v", l)
			}
		case RelAcquisition:
			acq++
		case RelRelated:
			rel++
			if l.HREF != "/v1/licenses/f1" || l.Type != LCPLicenseMIME || l.Title != "LCP license" {
				t.Errorf("license link %+v", l)
			}
		}
	}
	if self != 1 || acq != 4 || rel != 1 {
		t.Errorf("links: self=%d acquisition=%d related=%d (want 1/4/1: the LCP link exists only for the lcp file that has a license href)", self, acq, rel)
	}
}

func TestHRD070OPDSNavigation(t *testing.T) {
	raw, err := Navigation20()
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Context  string `json:"@context"`
		Metadata struct {
			Title string `json:"title"`
		} `json:"metadata"`
		Links []struct {
			HREF      string `json:"href"`
			Rel       string `json:"rel"`
			Type      string `json:"type"`
			Templated bool   `json:"templated"`
		} `json:"links"`
		Navigation []struct {
			HREF  string `json:"href"`
			Title string `json:"title"`
			Type  string `json:"type"`
			Rel   string `json:"rel"`
		} `json:"navigation"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	if f.Context != "https://readium.org/webpub-manifest/context.jsonld" || f.Metadata.Title != "Helix Catalog" {
		t.Errorf("navigation 2.0 header %+v", f)
	}
	if len(f.Links) != 2 || f.Links[0].Rel != "self" || f.Links[0].HREF != "/opds/2.0" || f.Links[0].Type != MIME2Feed ||
		f.Links[1].Rel != "search" || f.Links[1].HREF != "/opds/2.0/publications?q={searchTerms}" || !f.Links[1].Templated {
		t.Errorf("navigation 2.0 links %+v", f.Links)
	}
	wantNav := [][3]string{{"/opds/2.0/publications", "All publications", ""}, {"/opds/2.0/publications?sort=new", "Newest publications", "http://opds-spec.org/sort/new"}, {"/opds/2.0/publications?state=needs_review", "Needs identification review", ""}}
	if len(f.Navigation) != 3 {
		t.Fatalf("navigation items %d", len(f.Navigation))
	}
	for i, n := range f.Navigation {
		if n.HREF != wantNav[i][0] || n.Title != wantNav[i][1] || n.Rel != wantNav[i][2] || n.Type != MIME2Feed {
			t.Errorf("navigation %d: %+v", i, n)
		}
	}
	raw, err = Navigation12()
	if err != nil {
		t.Fatal(err)
	}
	var x xFeed
	if err := xml.Unmarshal(raw, &x); err != nil {
		t.Fatal(err)
	}
	if x.ID != "/opds/1.2" || x.Title != "Helix Catalog" || len(x.Entries) != 3 {
		t.Fatalf("navigation 1.2 %q %q %d entries", x.ID, x.Title, len(x.Entries))
	}
	if s := linksByRel(x.Links, "self"); len(s) != 1 || s[0].HREF != "/opds/1.2" || s[0].Type != MIME12Navigation {
		t.Errorf("navigation 1.2 self %v", s)
	}
	if s := linksByRel(x.Links, "search"); len(s) != 1 || s[0].HREF != "/opds/1.2/opensearch.xml" {
		t.Errorf("navigation 1.2 search %v", s)
	}
	for i, e := range x.Entries {
		if e.ID != wantNav[i][0][:len("/opds/")]+"1.2"+wantNav[i][0][len("/opds/2.0"):] || e.Title != wantNav[i][1] || len(e.Links) != 1 {
			t.Errorf("navigation 1.2 entry %d: %+v", i, e)
		}
	}
	_ = strconv.Itoa
}
