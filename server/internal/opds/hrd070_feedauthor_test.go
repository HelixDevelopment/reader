package opds

import (
	"encoding/xml"
	"testing"

	"github.com/helixreader/helix/server/internal/catalog"
)

// TestHRD070FeedAuthor12 pins the feed-level atom:author that OPDS 1.2 clients
// show as the catalog owner (RFC 4287 §4.1.1 requires an author on every feed
// that has no author on all its entries). Oracle: the product name written by
// hand from the project README ("Helix Reader"); no value comes from the code.
func TestHRD070FeedAuthor12(t *testing.T) {
	type authored struct {
		Authors []string `xml:"author>name"`
	}
	nav, err := Navigation12()
	if err != nil {
		t.Fatal(err)
	}
	acq, err := Acquisition12("/opds/p", catalog.Page{Total: 0, Limit: 10}, catalog.Query{Limit: 10}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for name, raw := range map[string][]byte{"Navigation12": nav, "Acquisition12": acq} {
		var f authored
		if err := xml.Unmarshal(raw, &f); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(f.Authors) != 1 || f.Authors[0] != "Helix Reader" {
			t.Errorf("%s: feed author %v, want [Helix Reader]", name, f.Authors)
		}
	}
}
