package mobi

import (
	"testing"

	syn "github.com/helixreader/helix/core/pkg/sync"
)

// TestPalmDOCPagination proves end-to-end pagination of PalmDOC-compressed
// MOBI text with sync-compatible positions: decompress -> strip -> blocks ->
// pages. Paragraph rune lengths below are exact, not sampled.
func TestPalmDOCPagination(t *testing.T) {
	text := "<p>Alpha bravo charlie.</p>\r<p>Delta echo.</p>\r<p>Foxtrot golf.</p>\r"
	r := buildTestMOBI(t, 2, text)
	doc, err := Parser{}.Parse(r, int64(r.Len()))
	if err != nil {
		t.Fatal(err)
	}
	blocks := doc.Chapters[0].Blocks
	wantTexts := []string{"Alpha bravo charlie.", "Delta echo.", "Foxtrot golf."}
	if len(blocks) != len(wantTexts) {
		t.Fatalf("blocks: %+v", blocks)
	}
	for i, w := range wantTexts {
		if blocks[i].Text != w {
			t.Fatalf("block %d: %q want %q", i, blocks[i].Text, w)
		}
	}
	// Exact rune lengths: 20, 11, 13. Budget 22: 20+11>22 and 11+13>22,
	// so every paragraph is a page.
	pages, err := syn.Paginate(doc, 22)
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 3 {
		t.Fatalf("pages: %+v", pages)
	}
	for i, pos := range []syn.Position{{Chapter: 0, Block: 0}, {Chapter: 0, Block: 1}, {Chapter: 0, Block: 2}} {
		if pages[i].Start != pos {
			t.Fatalf("page %d start %+v want %+v", i, pages[i].Start, pos)
		}
		if got := syn.PageOf(pages, pos); got != i {
			t.Fatalf("PageOf(%+v) = %d want %d", pos, got, i)
		}
	}
	if pages[0].Chars != 20 || pages[1].Chars != 11 || pages[2].Chars != 13 {
		t.Fatalf("page char totals: %+v", pages)
	}
	// Budget 24: paragraphs 2+3 (11+13=24) share page 1.
	pages, err = syn.Paginate(doc, 24)
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 2 || pages[1].Blocks != 2 || pages[1].Chars != 24 {
		t.Fatalf("merged page: %+v", pages)
	}
}
