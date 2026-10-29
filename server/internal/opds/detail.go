package opds

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"

	"github.com/helixreader/helix/server/internal/catalog"
)

func entry12(w catalog.Work, e catalog.Edition) Entry12 {
	entry := Entry12{Title: e.Title, ID: "urn:helix:edition:" + e.ID, Updated: stamp(e.UpdatedAt), Language: e.Language, Issued: e.Published, Publisher: e.Publisher}
	for _, a := range e.Authors {
		entry.Authors = append(entry.Authors, Author12{Name: a})
	}
	for _, id := range e.Identifiers {
		entry.Identifiers = append(entry.Identifiers, "urn:"+id.Scheme+":"+id.Value)
	}
	if e.Description != "" {
		entry.Summary = &Text12{Type: "text", Value: e.Description}
	}
	for _, s := range w.Subjects {
		entry.Categories = append(entry.Categories, Category12{Term: s, Label: s})
	}
	if e.CoverHREF != "" {
		entry.Links = append(entry.Links, Link12{Rel: RelImage, HREF: e.CoverHREF, Type: "image/jpeg"}, Link12{Rel: RelThumbnail, HREF: e.CoverHREF + "?size=thumb", Type: "image/jpeg"})
	}
	entry.Links = append(entry.Links, Link12{Rel: "alternate", HREF: "/opds/1.2/editions/" + url.PathEscape(e.ID), Type: MIME12Entry, Title: "Complete catalog entry"})
	for _, file := range e.Files {
		entry.Links = append(entry.Links, Link12{Rel: RelAcquisition, HREF: "/v1/files/" + url.PathEscape(file.ID) + "/content", Type: file.MIMEType, Title: fmt.Sprintf("%s (%s)", e.Title, file.Format)})
	}
	return entry
}
func Edition12(w catalog.Work, e catalog.Edition) ([]byte, error) {
	entry := entry12(w, e)
	entry.XMLNS = "http://www.w3.org/2005/Atom"
	entry.XMLNSDC = "http://purl.org/dc/terms/"
	entry.XMLNSOPDS = "http://opds-spec.org/2010/catalog"
	entry.XMLNSTHR = "http://purl.org/syndication/thread/1.0"
	return xml.MarshalIndent(entry, "", "  ")
}

func publication20(w catalog.Work, e catalog.Edition) Publication2 {
	p := Publication2{Metadata: Metadata2{Title: e.Title, Identifier: "urn:helix:edition:" + e.ID, Modified: stamp(e.UpdatedAt), Published: e.Published, Description: e.Description, Subjects: append([]string(nil), w.Subjects...)}, Links: []Link2{{HREF: "/opds/2.0/editions/" + url.PathEscape(e.ID), Rel: "self", Type: MIME2Publication}}}
	if e.Language != "" {
		p.Metadata.Language = e.Language
	}
	for _, a := range e.Authors {
		p.Metadata.Authors = append(p.Metadata.Authors, Contributor2{Name: a})
	}
	if e.Publisher != "" {
		p.Metadata.Publisher = &Contributor2{Name: e.Publisher}
	}
	if e.CoverHREF != "" {
		p.Images = []Link2{{HREF: e.CoverHREF, Type: "image/jpeg", Rel: "cover"}}
	}
	for _, file := range e.Files {
		p.Links = append(p.Links, Link2{HREF: "/v1/files/" + url.PathEscape(file.ID) + "/content", Rel: RelAcquisition, Type: file.MIMEType, Title: file.Format})
	}
	return p
}
func Edition20(w catalog.Work, e catalog.Edition) ([]byte, error) {
	p := publication20(w, e)
	p.Context = "https://readium.org/webpub-manifest/context.jsonld"
	return json.MarshalIndent(p, "", "  ")
}
