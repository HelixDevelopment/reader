package formats

import "testing"

// TestSameFormatRepackIsHonest: repack is Direct only where the core has
// both a parser and an exporter; formats without one of the two must not
// claim even same-format conversion.
func TestSameFormatRepackIsHonest(t *testing.T) {
	for _, f := range []Format{EPUB, FB2, TXT, CBZ} {
		if Convertible(f, f) != Direct {
			t.Errorf("%s->%s must be Direct (parser+exporter implemented)", f, f)
		}
	}
	for _, f := range []Format{PDF, MOBI, AZW3, DOCX, HTMLZ, XHTML} {
		if Convertible(f, f) != Blocked {
			t.Errorf("%s->%s must be Blocked (no parser+exporter pair in core)", f, f)
		}
	}
}

func TestDocumentedMatrix(t *testing.T) {
	cases := []struct {
		from, to Format
		want     Verdict
	}{
		{EPUB, PDF, Blocked}, {EPUB, AZW3, Blocked},
		{PDF, EPUB, Blocked}, {PDF, CBZ, Blocked}, {PDF, TXT, Blocked},
		{CBZ, TXT, Blocked}, {CBZ, EPUB, Blocked}, {TXT, CBZ, Blocked},
		{MOBI, FB2, Direct}, {EPUB, XHTML, Direct},
		{TXT, EPUB, Direct}, {MOBI, EPUB, Direct}, {EPUB, CBZ, Direct},
		{AZW3, EPUB, Blocked}, {DOCX, TXT, Blocked},
	}
	for _, c := range cases {
		if got := Convertible(c.from, c.to); got != c.want {
			t.Errorf("%s->%s: got %v want %v", c.from, c.to, got, c.want)
		}
	}
}

// TestMatrixNeverOutrunsTheCode guards the matrix against future bluffs:
// every claimed (source, target) pair must have a parser and an exporter
// implemented in core. The executable end-to-end proof of the same claim
// lives in pkg/convert chain tests; this guard keeps new rows honest even
// when only this package changes.
func TestMatrixNeverOutrunsTheCode(t *testing.T) {
	parsed := map[Format]bool{EPUB: true, FB2: true, MOBI: true, TXT: true, CBZ: true}
	exported := map[Format]bool{TXT: true, XHTML: true, FB2: true, EPUB: true, CBZ: true}
	for _, from := range []Format{EPUB, FB2, MOBI, TXT, CBZ, PDF, AZW3, DOCX, HTMLZ, XHTML} {
		for _, to := range Targets(from) {
			if !parsed[from] || !exported[to] {
				t.Errorf("matrix claims %s->%s but core lacks parser (%v) or exporter (%v)", from, to, parsed[from], exported[to])
			}
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
