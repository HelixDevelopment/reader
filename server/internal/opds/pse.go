package opds

import (
	"net/url"

	"github.com/helixreader/helix/core/pkg/pse"
)

// PageStreamPrefix is where page images are served. The router registers its
// route against this same constant, so a feed can never point at a path that
// is not served.
const PageStreamPrefix = "/opds-pse/stream/"

// PageStream is one file's page-stream advertisement, resolved by the caller
// from the archive itself.
type PageStream struct {
	Count int
	Type  string
}

// PageStreamHREF is the templated href the extension requires: it carries the
// literal {pageNumber} token the client replaces with 0..Count-1.
func PageStreamHREF(fileID string) string {
	return PageStreamPrefix + url.PathEscape(fileID) + "/" + pse.PageNumberTemplate
}

// streamLink projects an already-resolved page stream into an Atom link.
// A stream is advertised only when its count and image type are values the
// extension permits: the caller may know the archive, but this is where the
// feed either keeps the spec honest or breaks it.
func streamLink(fileID string, ps PageStream) (Link12, bool) {
	if ps.Count < 1 || !pse.IsImageType(ps.Type) {
		return Link12{}, false
	}
	return Link12{Rel: pse.StreamRel, HREF: PageStreamHREF(fileID), Type: ps.Type, PSECount: ps.Count}, true
}
