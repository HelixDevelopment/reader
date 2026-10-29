package txt

import (
	"strings"
	"testing"

	"github.com/helixreader/helix/core/pkg/parse/parseutil"
)

func TestChaptersAndHeadings(t *testing.T) {
	in := `Chapter 1

It was a bright cold day in April,
and the clocks were striking thirteen.

Chapter 2

The hallway smelt of boiled cabbage.`
	doc, err := Parser{}.Parse(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Chapters) != 2 {
		t.Fatalf("chapters: %d", len(doc.Chapters))
	}
	if doc.Chapters[0].Title != "Chapter 1" {
		t.Fatalf("title: %q", doc.Chapters[0].Title)
	}
	// multi-line paragraph joined
	p := doc.Chapters[0].Blocks[1].Text
	if p != "It was a bright cold day in April, and the clocks were striking thirteen." {
		t.Fatalf("para join: %q", p)
	}
	if doc.Meta.Title != "Chapter 1" {
		t.Fatalf("fallback title from first heading: %q", doc.Meta.Title)
	}
}

func TestAllCapsHeading(t *testing.T) {
	doc, err := Parser{}.Parse(strings.NewReader("THE BEGINNING\n\nSome text here."))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Chapters[0].Blocks[0].Kind.String() != "heading" {
		t.Fatal("ALL CAPS line must become heading")
	}
}

func TestNoHeadingsSingleChapter(t *testing.T) {
	doc, err := Parser{FallbackTitle: "Notes"}.Parse(strings.NewReader("just\nsome\nlines"))
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Chapters) != 1 || doc.Meta.Title != "Notes" {
		t.Fatalf("doc: %+v", doc.Meta)
	}
	if doc.Chapters[0].Blocks[0].Text != "just some lines" {
		t.Fatalf("blocks: %+v", doc.Chapters[0].Blocks)
	}
}

func TestEmpty(t *testing.T) {
	if _, err := (Parser{}).Parse(strings.NewReader("  \n \n")); err == nil {
		t.Fatal("empty text must fail validation")
	}
}

func TestLongUnwrappedLineWithinInputBudget(t *testing.T) {
	line := strings.Repeat("x", 2<<20)
	doc, err := (Parser{}).Parse(strings.NewReader(line))
	if err != nil || len(doc.Chapters) != 1 || doc.Chapters[0].Blocks[0].Text != line {
		t.Fatalf("long valid line must survive: %v", err)
	}
}

func TestTXTRejectsOversizedInput(t *testing.T) {
	r := strings.NewReader(strings.Repeat("x", int(parseutil.MaxTextBytes+1)))
	if _, err := (Parser{}).Parse(r); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversized TXT must be rejected, got %v", err)
	}
}

func FuzzParse(f *testing.F) {
	f.Add("Chapter 1\n\nA paragraph.")
	f.Add("\x00")
	f.Fuzz(func(t *testing.T, data string) {
		if len(data) > 1<<20 {
			t.Skip("bound fuzz input memory")
		}
		_, _ = (Parser{}).Parse(strings.NewReader(data))
	})
}
