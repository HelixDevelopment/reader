// Pagination over HDM, expressed in sync.Position coordinates (docs/10,
// docs/27): a page is a deterministic greedy packing of the flattened block
// sequence into a per-page character budget. Positions are block-anchored
// (Chapter = index into Document.FlatTOC order, Block = index within the
// chapter, Offset is reserved and always 0 at page boundaries), so page
// boundaries participate in the same furthest-position ordering the sync
// protocol uses for progress. This is reflowable pagination only: it performs
// no glyph layout and is therefore not a rendering implementation.
package sync

import (
	"fmt"
	"unicode/utf8"

	"github.com/helixreader/helix/core/pkg/hdm"
)

// Page is one deterministic page of the flattened document.
type Page struct {
	Index  int      // 0-based page number
	Start  Position // position of the first block on the page (inclusive)
	End    Position // position of the last block on the page (inclusive)
	Blocks int      // number of blocks packed on the page
	Chars  int      // summed rune length of block texts (images count 0)
}

// Paginate packs doc's flattened blocks into pages of at most charsPerPage
// runes. A single block longer than the budget occupies a page of its own
// (blocks are never split, so page content is exact and deterministic).
// Empty-image and separator blocks are anchored on pages with zero character
// cost so image positions survive pagination. A document whose chapters carry
// no blocks at all is rejected rather than reported as a zero-page book.
func Paginate(doc *hdm.Document, charsPerPage int) ([]Page, error) {
	if charsPerPage <= 0 {
		return nil, fmt.Errorf("sync: paginate: invalid character budget %d", charsPerPage)
	}
	if err := doc.Validate(); err != nil {
		return nil, err
	}
	flat := doc.FlatTOC()
	var pages []Page
	cur := Page{Start: Position{Chapter: -1}}
	flush := func() {
		if cur.Blocks > 0 {
			cur.Index = len(pages)
			pages = append(pages, cur)
			cur = Page{Start: Position{Chapter: -1}}
		}
	}
	for ci, ch := range flat {
		for bi, b := range ch.Blocks {
			chars := utf8.RuneCountInString(b.Text)
			if cur.Blocks > 0 && cur.Chars+chars > charsPerPage {
				flush()
			}
			if cur.Blocks == 0 {
				cur.Start = Position{Chapter: ci, Block: bi}
			}
			cur.End = Position{Chapter: ci, Block: bi}
			cur.Blocks++
			cur.Chars += chars
		}
	}
	flush()
	if len(pages) == 0 {
		return nil, fmt.Errorf("sync: paginate: document has no blocks")
	}
	return pages, nil
}

// PageOf returns the index of the page containing pos. A position before the
// first page maps to page 0; a position at or after the final block maps to
// the last page. Pages produced by Paginate are strictly ascending in
// Position order, so the binary search is well defined.
func PageOf(pages []Page, pos Position) int {
	lo, hi := 0, len(pages)-1
	if hi < 0 {
		return 0
	}
	if compare(pos, pages[0].Start) < 0 {
		return 0
	}
	idx := 0
	for lo <= hi {
		mid := (lo + hi) / 2
		if compare(pos, pages[mid].Start) >= 0 {
			idx = mid
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	return idx
}

func compare(a, b Position) int {
	if a.Chapter != b.Chapter {
		return a.Chapter - b.Chapter
	}
	if a.Block != b.Block {
		return a.Block - b.Block
	}
	return a.Offset - b.Offset
}
