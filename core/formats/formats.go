// Package formats is the registry of readable formats and the conversion
// matrix (docs/27, docs/31). The matrix is data; tests prove the documented
// guarantees.
package formats

import "sort"

type Format string

const (
	EPUB  Format = "epub"
	PDF   Format = "pdf"
	MOBI  Format = "mobi"
	AZW3  Format = "azw3"
	FB2   Format = "fb2"
	DOCX  Format = "docx"
	TXT   Format = "txt"
	HTMLZ Format = "htmlz"
	CBZ   Format = "cbz"
	XHTML Format = "xhtml"
)

type Verdict int

const (
	Direct Verdict = iota
	Degraded
	Blocked
)

// matrix contains only conversions implemented by the pure-Go HDM pivot.
// External Calibre/MuPDF engines must be registered and tested before adding targets.
var matrix = map[Format]map[Format]Verdict{
	EPUB: {FB2: Direct, TXT: Direct, XHTML: Direct},
	FB2:  {EPUB: Direct, TXT: Direct, XHTML: Direct},
	MOBI: {FB2: Direct, TXT: Direct, XHTML: Direct},
	TXT:  {FB2: Direct, XHTML: Direct},
}

// Convertible reports the verdict for from->to. Same-format is Direct
// (re-package). Unknown pairs are Blocked.
func Convertible(from, to Format) Verdict {
	if from == to {
		return Direct
	}
	if row, ok := matrix[from]; ok {
		if v, ok := row[to]; ok {
			return v
		}
	}
	return Blocked
}

// Targets lists reachable formats for a source format.
func Targets(from Format) []Format {
	var out []Format
	for to, v := range matrix[from] {
		if v != Blocked {
			out = append(out, to)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
