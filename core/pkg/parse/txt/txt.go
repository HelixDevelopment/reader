// Package txt parses plain text and Markdown-ish text into HDM.
// Heuristics (docs/27 Tier-1): blank-line separated paragraphs; ALL-CAPS or
// "Chapter N" lines become headings; UTF-8 assumed after loader-layer
// charset transcoding.
package txt

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/helixreader/helix/core/pkg/hdm"
	"github.com/helixreader/helix/core/pkg/parse/parseutil"
)

type Parser struct {
	// Title used when no heading is found in the first block.
	FallbackTitle string
}

var chapterRe = regexp.MustCompile(`(?i)^(chapter|part|book|prologue|epilogue)\b[\s\dIVXLCDM.:—-]*`)

func isHeading(line string) bool {
	t := strings.TrimSpace(line)
	if t == "" || len([]rune(t)) > 80 {
		return false
	}
	if chapterRe.MatchString(t) {
		return true
	}
	letters := 0
	upper := 0
	for _, r := range t {
		if r >= 'A' && r <= 'Z' {
			letters++
			upper++
		} else if r >= 'a' && r <= 'z' {
			letters++
		}
	}
	return letters >= 4 && upper == letters
}

func (p Parser) Parse(r io.Reader) (*hdm.Document, error) {
	data, err := parseutil.ReadAll(r, parseutil.MaxTextBytes)
	if err != nil {
		return nil, err
	}
	doc := &hdm.Document{Format: "txt", Resources: map[string][]byte{}, Meta: hdm.Metadata{
		Title: p.FallbackTitle, Identifiers: map[string]string{},
	}}
	var cur *hdm.Chapter
	var para []string
	n := 0
	flushPara := func() {
		if len(para) == 0 {
			return
		}
		t := strings.Join(para, " ")
		t = strings.Join(strings.Fields(t), " ")
		if t != "" {
			cur.Blocks = append(cur.Blocks, hdm.Block{Kind: hdm.BlockParagraph, Text: t})
		}
		para = nil
	}
	newChapter := func(title string) {
		flushPara()
		if cur != nil && len(cur.Blocks) > 0 {
			doc.Chapters = append(doc.Chapters, cur)
		}
		n++
		cur = &hdm.Chapter{ID: fmt.Sprintf("c%d", n), Title: title}
		if title != "" {
			cur.Blocks = append(cur.Blocks, hdm.Block{Kind: hdm.BlockHeading, Level: 1, Text: title})
		}
	}
	newChapter("")
	sc := bufio.NewScanner(bytes.NewReader(data))
	// The complete input is already bounded in memory; permit long unwrapped
	// lines up to that size instead of imposing an unrelated 1 MiB line cap.
	sc.Buffer(make([]byte, 64<<10), len(data)+1)
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			flushPara()
			continue
		}
		if isHeading(line) {
			newChapter(strings.TrimSpace(line))
			continue
		}
		para = append(para, strings.TrimSpace(line))
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	flushPara()
	if cur != nil && len(cur.Blocks) > 0 {
		doc.Chapters = append(doc.Chapters, cur)
	}
	if doc.Meta.Title == "" && len(doc.Chapters) > 0 && doc.Chapters[0].Title != "" {
		doc.Meta.Title = doc.Chapters[0].Title
	}
	if err := doc.Validate(); err != nil {
		return nil, err
	}
	return doc, nil
}
