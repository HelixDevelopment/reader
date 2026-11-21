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
//
// Implemented on-device: parse of epub/fb2/mobi/txt/cbz into HDM
// (pkg/parse/*) and export from HDM to txt/xhtml/fb2/epub/cbz
// (pkg/convert/*). Same-format Direct means repack via the HDM pivot and is
// listed per format only where both a parser and an exporter exist — MOBI
// has no PalmDOC writer, so MOBI->MOBI is deliberately NOT Direct.
// EPUB->CBZ repacks the document's image payloads; text is dropped, which is
// the definition of the comic target, not a silent loss of image content.
// All image-bearing claims cover PNG/JPEG payloads only (sniffed, never
// inferred from extensions, constitution §7.1).
var matrix = map[Format]map[Format]Verdict{
	EPUB: {FB2: Direct, TXT: Direct, XHTML: Direct, CBZ: Direct},
	FB2:  {EPUB: Direct, TXT: Direct, XHTML: Direct},
	MOBI: {FB2: Direct, TXT: Direct, XHTML: Direct, EPUB: Direct},
	TXT:  {FB2: Direct, XHTML: Direct, EPUB: Direct},
}

// repackDirect lists formats with BOTH a HDM parser and a HDM exporter in
// the core, i.e. a truthful same-format Convertible verdict. XHTML has no
// standalone parser (only the EPUB spine path reads XHTML) and MOBI has no
// PalmDOC writer, so both stay out.
var repackDirect = map[Format]bool{
	EPUB: true, FB2: true, TXT: true, CBZ: true,
}

// Convertible reports the verdict for from->to. Same-format is Direct only
// for formats in repackDirect (repack through HDM). Unknown pairs are Blocked.
func Convertible(from, to Format) Verdict {
	if from == to {
		if repackDirect[from] {
			return Direct
		}
		return Blocked
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
