// Package mobi parses unencrypted PalmDB/MOBI (PRC) containers into HDM
// (decision C-2: custom pure-Go reader; KF8/AZW3 falls back to server-side
// conversion). Implements the PalmDB header + record table + MOBI header
// text reassembly for the "old MOBI" text format.
package mobi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	"github.com/helixreader/helix/core/pkg/hdm"
	"github.com/helixreader/helix/core/pkg/parse/parseutil"
)

type Parser struct{}

type palmHeader struct {
	Name       string
	NumRecords int
	Offsets    []uint32
}

func readPalmHeader(r io.ReaderAt) (*palmHeader, error) {
	buf := make([]byte, 78)
	if _, err := r.ReadAt(buf, 0); err != nil {
		return nil, fmt.Errorf("mobi: header: %w", err)
	}
	name := strings.TrimRight(string(buf[0:32]), "\x00")
	num := int(binary.BigEndian.Uint16(buf[76:78]))
	if num <= 0 || num > 100000 {
		return nil, fmt.Errorf("mobi: bad record count %d", num)
	}
	h := &palmHeader{Name: name, NumRecords: num}
	off := 78
	for i := 0; i < num; i++ {
		rec := make([]byte, 8)
		if _, err := r.ReadAt(rec, int64(off)); err != nil {
			return nil, fmt.Errorf("mobi: record table: %w", err)
		}
		h.Offsets = append(h.Offsets, binary.BigEndian.Uint32(rec[0:4]))
		off += 8
	}
	return h, nil
}

// record returns record i's bytes (clamped to file size via next offset or EOF).
func record(r io.ReaderAt, h *palmHeader, size int64, i int) ([]byte, error) {
	start := int64(h.Offsets[i])
	var end int64
	if i+1 < h.NumRecords {
		end = int64(h.Offsets[i+1])
	} else {
		end = size
	}
	if start < 0 || end < start || start > size || end > size {
		return nil, fmt.Errorf("mobi: bad record bounds %d", i)
	}
	if end-start > parseutil.MaxEntryBytes {
		return nil, fmt.Errorf("mobi: record %d exceeds %d byte limit", i, parseutil.MaxEntryBytes)
	}
	buf := make([]byte, end-start)
	_, err := r.ReadAt(buf, start)
	return buf, err
}

const palmdocCompression = 2

func (Parser) Parse(r io.ReaderAt, size int64) (*hdm.Document, error) {
	h, err := readPalmHeader(r)
	if err != nil {
		return nil, err
	}
	if h.NumRecords < 2 {
		return nil, fmt.Errorf("mobi: too few records")
	}
	rec0, err := record(r, h, size, 0)
	if err != nil {
		return nil, err
	}
	if len(rec0) < 16 {
		return nil, fmt.Errorf("mobi: record 0 too short")
	}
	compression := binary.BigEndian.Uint16(rec0[0:2])
	textLength := binary.BigEndian.Uint32(rec0[4:8])
	recordCount := int(binary.BigEndian.Uint16(rec0[8:10]))
	if recordCount <= 0 || recordCount >= h.NumRecords {
		return nil, fmt.Errorf("mobi: invalid text record count %d for %d container records", recordCount, h.NumRecords)
	}

	doc := &hdm.Document{Format: "mobi", Resources: map[string][]byte{}}
	doc.Meta.Title = h.Name
	doc.Meta.Identifiers = map[string]string{}

	// Reassemble text from records 1..recordCount with a hard aggregate bound;
	// compressed records must never expand into unbounded memory.
	var raw bytes.Buffer
	writeText := func(data []byte) error {
		if int64(raw.Len())+int64(len(data)) > parseutil.MaxTextBytes {
			return fmt.Errorf("mobi: decoded text exceeds %d byte limit", parseutil.MaxTextBytes)
		}
		raw.Write(data)
		return nil
	}
	for i := 1; i <= recordCount && i < h.NumRecords; i++ {
		rec, err := record(r, h, size, i)
		if err != nil {
			return nil, err
		}
		if compression == palmdocCompression {
			dec, err := decompressPalmDOC(rec, parseutil.MaxEntryBytes)
			if err != nil {
				return nil, fmt.Errorf("mobi: PalmDOC record %d: %w", i, err)
			}
			if err := writeText(dec); err != nil {
				return nil, err
			}
		} else if compression == 1 {
			if err := writeText(rec); err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("mobi: unsupported compression %d", compression)
		}
	}
	text := raw.Bytes()
	if int64(len(text)) > int64(textLength) {
		text = text[:textLength]
	}
	// MOBI text is HTML-ish; strip tags crudely into paragraphs (loader-layer
	// refinement replaces this for KF8).
	plain := stripTags(string(text))
	ch := &hdm.Chapter{ID: "text", Title: h.Name}
	for _, p := range strings.Split(plain, "\n") {
		p = strings.Join(strings.Fields(p), " ")
		if p != "" {
			ch.Blocks = append(ch.Blocks, hdm.Block{Kind: hdm.BlockParagraph, Text: p})
		}
	}
	if len(ch.Blocks) == 0 {
		return nil, fmt.Errorf("mobi: no text content")
	}
	doc.Chapters = []*hdm.Chapter{ch}
	return doc, doc.Validate()
}

func stripTags(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	// MOBI uses <p> as block breaks; approximate with newlines before strip
	return strings.ReplaceAll(b.String(), "\r", "\n")
}
