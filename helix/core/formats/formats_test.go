package formats

import "testing"

func TestSameFormatIsDirect(t *testing.T) {
	for _, f := range []Format{EPUB, PDF, MOBI, AZW3, FB2, DOCX, TXT, HTMLZ, CBZ} {
		if Convertible(f, f) != Direct {
			t.Errorf("%s->%s must be Direct", f, f)
		}
	}
}
func TestDocumentedMatrix(t *testing.T) {
	cases := []struct {
		from, to Format
		want     Verdict
	}{
		{EPUB, PDF, Blocked}, {EPUB, AZW3, Blocked}, {EPUB, CBZ, Blocked},
		{PDF, EPUB, Blocked}, {PDF, CBZ, Blocked}, {PDF, TXT, Blocked},
		{CBZ, TXT, Blocked}, {CBZ, EPUB, Blocked}, {TXT, CBZ, Blocked},
		{MOBI, FB2, Direct}, {EPUB, XHTML, Direct},
	}
	for _, c := range cases {
		if got := Convertible(c.from, c.to); got != c.want {
			t.Errorf("%s->%s: got %v want %v", c.from, c.to, got, c.want)
		}
	}
}
func TestTargetsNeverBlocked(t *testing.T) {
	for _, from := range []Format{EPUB, PDF, MOBI, FB2, TXT} {
		for _, to := range Targets(from) {
			if Convertible(from, to) == Blocked {
				t.Errorf("Targets(%s) lists blocked %s", from, to)
			}
		}
	}
}
