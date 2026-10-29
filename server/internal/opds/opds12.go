package opds

import (
	"encoding/xml"
	"net/url"
	"strconv"
	"time"

	"github.com/helixreader/helix/server/internal/catalog"
)

func stamp(t time.Time) string {
	if t.IsZero() {
		return time.Unix(0, 0).UTC().Format(time.RFC3339)
	}
	return t.UTC().Format(time.RFC3339)
}

func feed12(id, title string, updated time.Time, kind string) Feed12 {
	return Feed12{
		XMLNS: "http://www.w3.org/2005/Atom", XMLNSDC: "http://purl.org/dc/terms/",
		XMLNSOPDS: "http://opds-spec.org/2010/catalog", XMLNSTHR: "http://purl.org/syndication/thread/1.0",
		XMLNSOpenSearch: "http://a9.com/-/spec/opensearch/1.1/", ID: id, Title: title,
		Updated: stamp(updated), Author: Author12{Name: "Helix Reader"},
		Links: []Link12{{Rel: "self", HREF: id, Type: kind}, {Rel: "start", HREF: "/opds/1.2", Type: MIME12Navigation}},
	}
}

func Navigation12() ([]byte, error) {
	f := feed12("/opds/1.2", "Helix Catalog", time.Unix(0, 0), MIME12Navigation)
	f.Links = append(f.Links, Link12{Rel: "search", HREF: "/opds/1.2/opensearch.xml", Type: MIMEOpenSearch})
	nav := []struct{ title, href, rel string }{
		{"All publications", "/opds/1.2/publications", "subsection"},
		{"Newest publications", "/opds/1.2/publications?sort=new", "http://opds-spec.org/sort/new"},
		{"Needs identification review", "/opds/1.2/publications?state=needs_review", "subsection"},
	}
	for _, n := range nav {
		f.Entries = append(f.Entries, Entry12{Title: n.title, ID: n.href, Updated: stamp(time.Time{}), Content: &Text12{Type: "text", Value: n.title}, Links: []Link12{{Rel: n.rel, HREF: n.href, Type: MIME12Acquisition}}})
	}
	return xml.MarshalIndent(f, "", "  ")
}

func OpenSearch12(base string) ([]byte, error) {
	if base == "" {
		base = ""
	}
	type osURL struct {
		Type     string `xml:"type,attr"`
		Template string `xml:"template,attr"`
	}
	type doc struct {
		XMLName        xml.Name `xml:"OpenSearchDescription"`
		XMLNS          string   `xml:"xmlns,attr"`
		ShortName      string   `xml:"ShortName"`
		Description    string   `xml:"Description"`
		InputEncoding  string   `xml:"InputEncoding"`
		OutputEncoding string   `xml:"OutputEncoding"`
		URL            osURL    `xml:"Url"`
	}
	d := doc{XMLNS: "http://a9.com/-/spec/opensearch/1.1/", ShortName: "Helix", Description: "Search the Helix publication catalog", InputEncoding: "UTF-8", OutputEncoding: "UTF-8", URL: osURL{Type: MIME12Acquisition, Template: base + "/opds/1.2/publications?q={searchTerms}&author={atom:author}"}}
	return xml.MarshalIndent(d, "", "  ")
}

func pageURL(path string, q catalog.Query, offset int) string {
	v := url.Values{}
	if q.Text != "" {
		v.Set("q", q.Text)
	}
	if q.Author != "" {
		v.Set("author", q.Author)
	}
	if q.State != "" {
		v.Set("state", string(q.State))
	}
	v.Set("offset", strconv.Itoa(offset))
	v.Set("limit", strconv.Itoa(q.Limit))
	return path + "?" + v.Encode()
}

func Acquisition12(path string, page catalog.Page, q catalog.Query) ([]byte, error) {
	updated := time.Time{}
	if len(page.Works) > 0 {
		updated = page.Works[0].UpdatedAt
	}
	f := feed12(pageURL(path, q, q.Offset), "Helix Publications", updated, MIME12Acquisition)
	f.TotalResults, f.ItemsPerPage = page.Total, page.Limit
	f.Links = append(f.Links, Link12{Rel: "search", HREF: "/opds/1.2/opensearch.xml", Type: MIMEOpenSearch})
	if q.Offset > 0 {
		prev := q.Offset - q.Limit
		if prev < 0 {
			prev = 0
		}
		f.Links = append(f.Links, Link12{Rel: "previous", HREF: pageURL(path, q, prev), Type: MIME12Acquisition})
	}
	if q.Offset+len(page.Works) < page.Total {
		f.Links = append(f.Links, Link12{Rel: "next", HREF: pageURL(path, q, q.Offset+q.Limit), Type: MIME12Acquisition})
	}
	for _, w := range page.Works {
		for _, e := range w.Editions {
			f.Entries = append(f.Entries, entry12(w, e))
		}
	}
	return xml.MarshalIndent(f, "", "  ")
}
