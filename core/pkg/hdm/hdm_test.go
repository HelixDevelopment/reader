package hdm

import (
	"strings"
	"testing"
)

func sampleDoc() *Document {
	return &Document{
		Meta: Metadata{
			Title:       "The Left Hand of Darkness",
			Creators:    []string{"Ursula K. Le Guin"},
			Language:    "en",
			Identifiers: map[string]string{"isbn": "9780441478125"},
		},
		Chapters: []*Chapter{
			{ID: "c1", Title: "A Parade in Erhenrang", Href: "ch1.xhtml", Blocks: []Block{
				{Kind: BlockHeading, Level: 1, Text: "Chapter 1"},
				{Kind: BlockParagraph, Text: "From the archives of Hain. A parade in Erhenrang."},
				{Kind: BlockImage, ImageRef: "img1"},
			}},
			{ID: "c2", Title: "The Place Inside the Blizzard", Href: "ch2.xhtml", Blocks: []Block{
				{Kind: BlockParagraph, Text: "Snow. Ice. The glacier speaks."},
			}},
		},
		Resources: map[string][]byte{"img1": {0x89, 0x50}},
	}
}

func TestValidateOK(t *testing.T) {
	if err := sampleDoc().Validate(); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestValidateNoChapters(t *testing.T) {
	d := &Document{}
	if err := d.Validate(); err != ErrNoContent {
		t.Fatalf("want ErrNoContent, got %v", err)
	}
}

func TestValidateDuplicateChapterID(t *testing.T) {
	d := sampleDoc()
	d.Chapters = append(d.Chapters, d.Chapters[0])
	if err := d.Validate(); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("want duplicate error, got %v", err)
	}
}

func TestValidateMissingImageRef(t *testing.T) {
	d := sampleDoc()
	delete(d.Resources, "img1")
	if err := d.Validate(); err == nil || !strings.Contains(err.Error(), "missing image") {
		t.Fatalf("want missing image error, got %v", err)
	}
}

func TestWordCount(t *testing.T) {
	// c1: heading 2 words + paragraph 9 words = 11; image skipped.
	// c2: 5 words. Total 16.
	if got := sampleDoc().WordCount(); got != 16 {
		t.Fatalf("want 17, got %d", got)
	}
}

func TestFingerprintStableAndSensitive(t *testing.T) {
	a := sampleDoc()
	b := sampleDoc()
	if a.Fingerprint() != b.Fingerprint() {
		t.Fatal("identical docs must fingerprint identically")
	}
	b.Meta.Title = "Different Title"
	if a.Fingerprint() == b.Fingerprint() {
		t.Fatal("different titles must produce different fingerprints")
	}
	c := sampleDoc()
	c.Chapters[1].Blocks[0].Text = "Completely changed content here."
	if a.Fingerprint() == c.Fingerprint() {
		t.Fatal("changed chapter text must change fingerprint")
	}
}

func TestFingerprintCaseAndWhitespaceInsensitive(t *testing.T) {
	a := sampleDoc()
	b := sampleDoc()
	b.Chapters[0].Blocks[1].Text = "  FROM  the archives   of Hain.\nA parade in Erhenrang. "
	if a.Fingerprint() != b.Fingerprint() {
		t.Fatal("whitespace/case-only differences must not change fingerprint")
	}
}

func TestFlatTOC(t *testing.T) {
	d := sampleDoc()
	d.Chapters[0].Children = []*Chapter{{ID: "c1.1", Title: "Sub", Href: "s.xhtml"}}
	flat := d.FlatTOC()
	if len(flat) != 3 || flat[1].ID != "c1.1" {
		t.Fatalf("unexpected flat TOC: %v", flat)
	}
}
