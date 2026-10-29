// Package opds implements OPDS 1.2 (Atom) and OPDS 2.0 (RWPM/JSON)
// projections of the same catalog repository.
package opds

import "encoding/xml"

const (
	MIME12Navigation  = "application/atom+xml;profile=opds-catalog;kind=navigation"
	MIME12Acquisition = "application/atom+xml;profile=opds-catalog;kind=acquisition"
	MIME12Entry       = "application/atom+xml;type=entry;profile=opds-catalog"
	MIME2Feed         = "application/opds+json"
	MIME2Publication  = "application/opds-publication+json"
	MIMEOpenSearch    = "application/opensearchdescription+xml"
	RelAcquisition    = "http://opds-spec.org/acquisition"
	RelImage          = "http://opds-spec.org/image"
	RelThumbnail      = "http://opds-spec.org/image/thumbnail"
)

// Link12 is an Atom link with OPDS facet hints.
type Link12 struct {
	Rel         string `xml:"rel,attr,omitempty"`
	HREF        string `xml:"href,attr"`
	Type        string `xml:"type,attr,omitempty"`
	Title       string `xml:"title,attr,omitempty"`
	FacetGroup  string `xml:"opds:facetGroup,attr,omitempty"`
	ActiveFacet string `xml:"opds:activeFacet,attr,omitempty"`
	Count       int    `xml:"thr:count,attr,omitempty"`
}

type Author12 struct {
	Name string `xml:"name"`
	URI  string `xml:"uri,omitempty"`
}

type Category12 struct {
	Term   string `xml:"term,attr"`
	Scheme string `xml:"scheme,attr,omitempty"`
	Label  string `xml:"label,attr,omitempty"`
}

type Text12 struct {
	Type  string `xml:"type,attr,omitempty"`
	Value string `xml:",chardata"`
}

type Entry12 struct {
	XMLName     xml.Name     `xml:"entry"`
	XMLNS       string       `xml:"xmlns,attr,omitempty"`
	XMLNSDC     string       `xml:"xmlns:dc,attr,omitempty"`
	XMLNSOPDS   string       `xml:"xmlns:opds,attr,omitempty"`
	XMLNSTHR    string       `xml:"xmlns:thr,attr,omitempty"`
	Title       string       `xml:"title"`
	ID          string       `xml:"id"`
	Updated     string       `xml:"updated"`
	Authors     []Author12   `xml:"author,omitempty"`
	Language    string       `xml:"dc:language,omitempty"`
	Issued      string       `xml:"dc:issued,omitempty"`
	Publisher   string       `xml:"dc:publisher,omitempty"`
	Identifiers []string     `xml:"dc:identifier,omitempty"`
	Summary     *Text12      `xml:"summary,omitempty"`
	Content     *Text12      `xml:"content,omitempty"`
	Categories  []Category12 `xml:"category,omitempty"`
	Links       []Link12     `xml:"link"`
}

type Feed12 struct {
	XMLName         xml.Name  `xml:"feed"`
	XMLNS           string    `xml:"xmlns,attr"`
	XMLNSDC         string    `xml:"xmlns:dc,attr"`
	XMLNSOPDS       string    `xml:"xmlns:opds,attr"`
	XMLNSTHR        string    `xml:"xmlns:thr,attr"`
	XMLNSOpenSearch string    `xml:"xmlns:opensearch,attr,omitempty"`
	ID              string    `xml:"id"`
	Title           string    `xml:"title"`
	Updated         string    `xml:"updated"`
	Author          Author12  `xml:"author"`
	Links           []Link12  `xml:"link"`
	Entries         []Entry12 `xml:"entry,omitempty"`
	TotalResults    int       `xml:"http://a9.com/-/spec/opensearch/1.1/ totalResults,omitempty"`
	ItemsPerPage    int       `xml:"http://a9.com/-/spec/opensearch/1.1/ itemsPerPage,omitempty"`
}

// OPDS 2 / Readium Web Publication Manifest shared models.
type Link2 struct {
	HREF       string `json:"href"`
	Type       string `json:"type,omitempty"`
	Rel        any    `json:"rel,omitempty"` // string or []string per RWPM
	Title      string `json:"title,omitempty"`
	Templated  bool   `json:"templated,omitempty"`
	Properties any    `json:"properties,omitempty"`
}

type Contributor2 struct {
	Name       string `json:"name"`
	Identifier string `json:"identifier,omitempty"`
	SortAs     string `json:"sortAs,omitempty"`
}

type Metadata2 struct {
	Title         string         `json:"title"`
	Identifier    string         `json:"identifier,omitempty"`
	Type          string         `json:"@type,omitempty"`
	Modified      string         `json:"modified,omitempty"`
	Published     string         `json:"published,omitempty"`
	Language      any            `json:"language,omitempty"`
	Authors       []Contributor2 `json:"author,omitempty"`
	Publisher     *Contributor2  `json:"publisher,omitempty"`
	Description   string         `json:"description,omitempty"`
	Subjects      []string       `json:"subject,omitempty"`
	NumberOfItems int            `json:"numberOfItems,omitempty"`
}

type Publication2 struct {
	Context  string    `json:"@context,omitempty"`
	Metadata Metadata2 `json:"metadata"`
	Links    []Link2   `json:"links"`
	Images   []Link2   `json:"images,omitempty"`
}

type Navigation2 struct {
	HREF  string `json:"href"`
	Title string `json:"title"`
	Type  string `json:"type,omitempty"`
	Rel   string `json:"rel,omitempty"`
}

type Feed2 struct {
	Context      string         `json:"@context"`
	Metadata     Metadata2      `json:"metadata"`
	Links        []Link2        `json:"links"`
	Navigation   []Navigation2  `json:"navigation,omitempty"`
	Publications []Publication2 `json:"publications,omitempty"`
	Facets       any            `json:"facets,omitempty"`
}
