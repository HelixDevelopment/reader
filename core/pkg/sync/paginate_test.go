package sync

import (
	"strings"
	"testing"

	"github.com/helixreader/helix/core/pkg/hdm"
)

func pageDoc(blocksPerChapter ...[]string) *hdm.Document {
	doc := &hdm.Document{Format: "txt", Resources: map[string][]byte{}, Meta: hdm.Metadata{Identifiers: map[string]string{}}}
	for ci, blocks := range blocksPerChapter {
		ch := &hdm.Chapter{ID: string(rune('a' + ci))}
		for _, b := range blocks {
			ch.Blocks = append(ch.Blocks, hdm.Block{Kind: hdm.BlockParagraph, Text: b})
		}
		doc.Chapters = append(doc.Chapters, ch)
	}
	return doc
}

func TestPaginateExactPacking(t *testing.T) {
	// Block rune lengths: 10, 10, 10 with budget 25 => pages {10,10}, {10}.
	doc := pageDoc([]string{"aaaaaaaaaa", "bbbbbbbbbb", "cccccccccc"})
	pages, err := Paginate(doc, 25)
	if err != nil {
		t.Fatal(err)
	}
	want := []Page{
		{Index: 0, Start: Position{Chapter: 0, Block: 0}, End: Position{Chapter: 0, Block: 1}, Blocks: 2, Chars: 20},
		{Index: 1, Start: Position{Chapter: 0, Block: 2}, End: Position{Chapter: 0, Block: 2}, Blocks: 1, Chars: 10},
	}
	if len(pages) != len(want) {
		t.Fatalf("pages: %+v", pages)
	}
	for i := range want {
		if pages[i] != want[i] {
			t.Fatalf("page %d = %+v, want %+v", i, pages[i], want[i])
		}
	}
}

func TestPaginateSpansChapters(t *testing.T) {
	doc := pageDoc([]string{"aaaa", "bbbb"}, []string{"cccc"})
	pages, err := Paginate(doc, 12)
	if err != nil {
		t.Fatal(err)
	}
	// Budget 12 fits all three 4-rune blocks; one page spans two chapters.
	want := []Page{
		{Index: 0, Start: Position{Chapter: 0, Block: 0}, End: Position{Chapter: 1, Block: 0}, Blocks: 3, Chars: 12},
	}
	if len(pages) != 1 || pages[0] != want[0] {
		t.Fatalf("pages: %+v", pages)
	}
	// Budget 8: the third block starts a new page in the second chapter.
	pages, err = Paginate(doc, 8)
	if err != nil {
		t.Fatal(err)
	}
	want = []Page{
		{Index: 0, Start: Position{Chapter: 0, Block: 0}, End: Position{Chapter: 0, Block: 1}, Blocks: 2, Chars: 8},
		{Index: 1, Start: Position{Chapter: 1, Block: 0}, End: Position{Chapter: 1, Block: 0}, Blocks: 1, Chars: 4},
	}
	if len(pages) != 2 || pages[0] != want[0] || pages[1] != want[1] {
		t.Fatalf("pages: %+v", pages)
	}
}

func TestPaginateOversizedBlockGetsOwnPage(t *testing.T) {
	doc := pageDoc([]string{"xx", strings.Repeat("y", 100), "zz"})
	pages, err := Paginate(doc, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 3 {
		t.Fatalf("expected 3 pages, got %+v", pages)
	}
	if pages[1].Blocks != 1 || pages[1].Chars != 100 || pages[1].Start != (Position{Chapter: 0, Block: 1}) {
		t.Fatalf("oversized block page: %+v", pages[1])
	}
}

func TestPaginateImageBlocksAnchorWithoutChars(t *testing.T) {
	doc := pageDoc([]string{"text"})
	doc.Chapters[0].Blocks = append(doc.Chapters[0].Blocks,
		hdm.Block{Kind: hdm.BlockImage, ImageRef: "im"},
		hdm.Block{Kind: hdm.BlockParagraph, Text: "more"})
	doc.Resources["im"] = []byte{1}
	pages, err := Paginate(doc, 4)
	if err != nil {
		t.Fatal(err)
	}
	// "text"=4 chars fills page 0 (budget 4); the image block joins it at
	// zero cost; "more" overflows onto page 1.
	if len(pages) != 2 ||
		pages[0] != (Page{Index: 0, Start: Position{Chapter: 0, Block: 0}, End: Position{Chapter: 0, Block: 1}, Blocks: 2, Chars: 4}) ||
		pages[1] != (Page{Index: 1, Start: Position{Chapter: 0, Block: 2}, End: Position{Chapter: 0, Block: 2}, Blocks: 1, Chars: 4}) {
		t.Fatalf("pages: %+v", pages)
	}
}

func TestPaginateUsesFlatTOCOrderIncludingChildren(t *testing.T) {
	doc := pageDoc([]string{"parent"})
	doc.Chapters[0].Children = []*hdm.Chapter{{ID: "child", Blocks: []hdm.Block{{Kind: hdm.BlockParagraph, Text: "childblock"}}}}
	pages, err := Paginate(doc, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 1 || pages[0].Blocks != 2 {
		t.Fatalf("flat toc packing: %+v", pages)
	}
}

func TestPaginateFailClosed(t *testing.T) {
	for _, test := range []struct {
		name    string
		doc     *hdm.Document
		budget  int
		wantErr string
	}{
		{"zero budget", pageDoc([]string{"x"}), 0, "invalid character budget"},
		{"negative budget", pageDoc([]string{"x"}), -3, "invalid character budget"},
		{"no chapters", &hdm.Document{Resources: map[string][]byte{}}, 10, "no chapters"},
		{"chapter without blocks", pageDoc([]string{}), 10, "no blocks"},
		{"duplicate ids", func() *hdm.Document {
			d := pageDoc([]string{"a"}, []string{"b"})
			d.Chapters[1].ID = d.Chapters[0].ID
			return d
		}(), 10, "duplicate chapter id"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := Paginate(test.doc, test.budget)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("want error containing %q, got %v", test.wantErr, err)
			}
		})
	}
}

func TestPageOfBoundariesAndOrdering(t *testing.T) {
	doc := pageDoc([]string{"aaaaaaaaaa", "bbbbbbbbbb", "cccccccccc", "dddddddddd"})
	pages, err := Paginate(doc, 15)
	if err != nil {
		t.Fatal(err)
	}
	// Budget 15: pages {a},{b},{c},{d}.
	if len(pages) != 4 {
		t.Fatalf("pages: %+v", pages)
	}
	for i := range pages {
		if got := PageOf(pages, pages[i].Start); got != i {
			t.Fatalf("PageOf(start %d) = %d", i, got)
		}
		if got := PageOf(pages, pages[i].End); got != i {
			t.Fatalf("PageOf(end %d) = %d", i, got)
		}
	}
	if got := PageOf(pages, Position{Chapter: -1}); got != 0 {
		t.Fatalf("before first page: %d", got)
	}
	if got := PageOf(pages, Position{Chapter: 99, Block: 99}); got != len(pages)-1 {
		t.Fatalf("after last page: %d", got)
	}
	// Page starts are strictly ascending, so the sync furthest-position rule
	// orders page boundaries consistently.
	for i := 1; i < len(pages); i++ {
		if Furthest(pages[i-1].Start, pages[i].Start) != pages[i].Start {
			t.Fatalf("page starts not ascending: %+v then %+v", pages[i-1].Start, pages[i].Start)
		}
	}
}

func FuzzPaginate(f *testing.F) {
	f.Add("abc def ghi", 4)
	f.Add("xxxxxxxxxxxxxxxxxxxx", 1)
	f.Add("", 10)
	f.Fuzz(func(t *testing.T, text string, budget int) {
		if len(text) > 1<<16 || budget > 1<<16 {
			t.Skip("bound fuzz input memory")
		}
		doc := pageDoc(strings.Split(text, " "))
		pages, err := Paginate(doc, budget)
		if err != nil {
			return
		}
		// Invariant: pages are contiguous over the flat block sequence and
		// the first page starts at the very first block.
		if pages[0].Start != (Position{Chapter: 0, Block: 0}) {
			t.Fatalf("first page start: %+v", pages[0].Start)
		}
		for i := 1; i < len(pages); i++ {
			prev := pages[i-1].End
			if pages[i].Start.Chapter != prev.Chapter || pages[i].Start.Block != prev.Block+1 {
				t.Fatalf("page gap between %d and %d", i-1, i)
			}
		}
		last := pages[len(pages)-1]
		if PageOf(pages, last.End) != len(pages)-1 {
			t.Fatalf("PageOf lost the last page")
		}
	})
}
