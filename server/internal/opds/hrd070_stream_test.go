package opds

import "testing"

// TestHRD070StreamLink: streamLink advertises a page stream only for count>=1
// and an image type the OPDS-PSE extension permits. Oracle: the extension text
// (positive integer count; image/jpeg, image/gif, image/png), by hand.
func TestHRD070StreamLink(t *testing.T) {
	for _, c := range []struct {
		name string
		ps   PageStream
		ok   bool
	}{
		{"jpeg one page", PageStream{Count: 1, Type: "image/jpeg"}, true},
		{"png many", PageStream{Count: 500, Type: "image/png"}, true},
		{"gif", PageStream{Count: 2, Type: "image/gif"}, true},
		{"count zero", PageStream{Count: 0, Type: "image/jpeg"}, false},
		{"count negative", PageStream{Count: -3, Type: "image/jpeg"}, false},
		{"empty type", PageStream{Count: 5, Type: ""}, false},
		{"non image type", PageStream{Count: 5, Type: "application/pdf"}, false},
		{"case differs", PageStream{Count: 5, Type: "IMAGE/JPEG"}, false},
		{"both bad", PageStream{Count: 0, Type: "text/html"}, false},
	} {
		l, ok := streamLink("f1", c.ps)
		if ok != c.ok {
			t.Errorf("%s: ok=%v want %v", c.name, ok, c.ok)
			continue
		}
		if !ok {
			if l != (Link12{}) {
				t.Errorf("%s: refused stream returned a link %+v", c.name, l)
			}
			continue
		}
		if l.HREF != PageStreamHREF("f1") || l.Type != c.ps.Type || l.PSECount != c.ps.Count || l.Rel == "" {
			t.Errorf("%s: link %+v", c.name, l)
		}
	}
}

// TestHRD070PageStreamHREF: the href carries the literal {pageNumber} token
// last and path-escapes the file id (OPDS-PSE text, expected string by hand).
func TestHRD070PageStreamHREF(t *testing.T) {
	for id, want := range map[string]string{
		"x":     "/opds-pse/stream/x/{pageNumber}",
		"a b/c": "/opds-pse/stream/a%20b%2Fc/{pageNumber}",
	} {
		if got := PageStreamHREF(id); got != want {
			t.Errorf("PageStreamHREF(%q) = %q, want %q", id, got, want)
		}
	}
}
