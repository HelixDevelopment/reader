package opds

import (
	"encoding/json"

	"github.com/helixreader/helix/server/internal/catalog"
)

func Navigation20() ([]byte, error) {
	f := Feed2{Context: "https://readium.org/webpub-manifest/context.jsonld", Metadata: Metadata2{Title: "Helix Catalog"}, Links: []Link2{{HREF: "/opds/2.0", Rel: "self", Type: MIME2Feed}, {HREF: "/opds/2.0/publications?q={searchTerms}", Rel: "search", Type: MIME2Feed, Templated: true}}, Navigation: []Navigation2{{HREF: "/opds/2.0/publications", Title: "All publications", Type: MIME2Feed}, {HREF: "/opds/2.0/publications?sort=new", Title: "Newest publications", Type: MIME2Feed, Rel: "http://opds-spec.org/sort/new"}, {HREF: "/opds/2.0/publications?state=needs_review", Title: "Needs identification review", Type: MIME2Feed}}}
	return json.MarshalIndent(f, "", "  ")
}

func Acquisition20(path string, page catalog.Page, q catalog.Query) ([]byte, error) {
	f := Feed2{Context: "https://readium.org/webpub-manifest/context.jsonld", Metadata: Metadata2{Title: "Helix Publications", NumberOfItems: page.Total}, Links: []Link2{{HREF: pageURL(path, q, q.Offset), Rel: "self", Type: MIME2Feed}, {HREF: "/opds/2.0", Rel: "start", Type: MIME2Feed}, {HREF: "/opds/2.0/publications?q={searchTerms}", Rel: "search", Type: MIME2Feed, Templated: true}}, Publications: []Publication2{}}
	if q.Offset > 0 {
		prev := q.Offset - q.Limit
		if prev < 0 {
			prev = 0
		}
		f.Links = append(f.Links, Link2{HREF: pageURL(path, q, prev), Rel: "previous", Type: MIME2Feed})
	}
	if q.Offset+len(page.Works) < page.Total {
		f.Links = append(f.Links, Link2{HREF: pageURL(path, q, q.Offset+q.Limit), Rel: "next", Type: MIME2Feed})
	}
	for _, w := range page.Works {
		for _, e := range w.Editions {
			f.Publications = append(f.Publications, publication20(w, e))
		}
	}
	return json.MarshalIndent(f, "", "  ")
}
