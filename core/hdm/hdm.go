// Package hdm defines the Helix Document Model (docs/27) — the canonical
// intermediate representation every parser emits and every consumer
// (renderer, TTS, AI, search, converter) reads. One model = consistent
// presentation across all formats.
package hdm

// BlockKind enumerates normalized block types.
type BlockKind string

const (
	Paragraph BlockKind = "paragraph"
	Heading   BlockKind = "heading"
	Code      BlockKind = "code"
	Quote     BlockKind = "quote"
	Image     BlockKind = "image"
	Table     BlockKind = "table"
	Footnote  BlockKind = "footnote"
	Math      BlockKind = "math"
)

type Inline struct {
	Text  string
	Bold  bool
	Italic bool
	Mono  bool
	Link  string
}

type Block struct {
	Kind     BlockKind
	Level    int // heading level
	Inlines  []Inline
	Language string // code blocks: syntax id for syntect
	AssetID  string // images/tables
}

type SpineItem struct {
	ID       string
	Title    string
	Blocks   []Block
	AnchorBase string // CFI base for annotations
}

type Metadata struct {
	Title     string
	Authors   []string
	Language  string
	ISBN      []string
	Publisher string
	Series    string
}

// Publication is the canonical book. Converters are pure functions
// Publication -> bytes (writer per format).
type Publication struct {
	Meta  Metadata
	Toc   []TocEntry
	Spine []SpineItem
}

type TocEntry struct {
	Title    string
	SpineID  string
	Children []TocEntry
}

// WordCount approximates reading-time inputs deterministically.
func (p *Publication) WordCount() int {
	n := 0
	for _, s := range p.Spine {
		for _, b := range s.Blocks {
			for _, in := range b.Inlines {
				w := 0
				inWord := false
				for _, r := range in.Text {
					if r == ' ' || r == '\t' || r == '\n' {
						inWord = false
					} else if !inWord {
						inWord = true
						w++
					}
				}
				n += w
			}
		}
	}
	return n
}
