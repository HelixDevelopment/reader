package fb2

import (
	"strings"
	"testing"

	"github.com/helixreader/helix/core/pkg/parse/parseutil"
)

const sample = `<?xml version="1.0" encoding="utf-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
 <description><title-info>
  <author><first-name>Ursula</first-name><last-name>Le Guin</last-name></author>
  <book-title>The Dispossessed</book-title>
    <isbn>978-0-441-47812-5</isbn>
  <lang>en</lang>
 </title-info></description>
 <body>
  <section>
   <title><p>Chapter 1</p></title>
   <p>There was a wall.</p>
   <p>It did not look important.</p>
  </section>
  <section>
   <title><p>Chapter 2</p></title>
   <p>Some   spaced   text.</p>
  </section>
 </body>
</FictionBook>`

func TestParseFB2(t *testing.T) {
	doc, err := Parser{}.Parse(strings.NewReader(sample))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Format != "fb2" || doc.Meta.Title != "The Dispossessed" {
		t.Fatalf("meta: %+v", doc.Meta)
	}
	if doc.Meta.Identifiers["isbn"] != "9780441478125" {
		t.Fatalf("validated normalized ISBN missing: %v", doc.Meta.Identifiers)
	}
	if len(doc.Meta.Creators) != 1 || doc.Meta.Creators[0] != "Ursula Le Guin" {
		t.Fatalf("creators: %v", doc.Meta.Creators)
	}
	if len(doc.Chapters) != 2 {
		t.Fatalf("chapters: %d", len(doc.Chapters))
	}
	c1 := doc.Chapters[0]
	if c1.Title != "Chapter 1" || c1.Blocks[0].Kind.String() != "heading" {
		t.Fatalf("ch1: %+v", c1.Blocks[0])
	}
	if got := doc.WordCount(); got != 16 {
		t.Fatalf("words: %d", got) // ch1: 2+4+5=11, ch2: 2+3=5
	}
	// whitespace normalised
	if doc.Chapters[1].Blocks[1].Text != "Some spaced text." {
		t.Fatalf("normalisation: %q", doc.Chapters[1].Blocks[1].Text)
	}
}

func TestParseGarbage(t *testing.T) {
	if _, err := (Parser{}).Parse(strings.NewReader("\x00\x01not xml")); err == nil {
		t.Fatal("garbage must error")
	}
}

func TestEmptyBody(t *testing.T) {
	empty := `<?xml version="1.0"?><FictionBook><description><title-info><book-title>X</book-title></title-info></description><body></body></FictionBook>`
	if _, err := (Parser{}).Parse(strings.NewReader(empty)); err == nil {
		t.Fatal("no chapters must fail HDM validation")
	}
}

func TestFB2RejectsExcessiveNesting(t *testing.T) {
	input := `<FictionBook><body>` + strings.Repeat("<section>", parseutil.MaxXMLDepth+1) + "<p>text</p>" + strings.Repeat("</section>", parseutil.MaxXMLDepth+1) + `</body></FictionBook>`
	if _, err := (Parser{}).Parse(strings.NewReader(input)); err == nil || !strings.Contains(err.Error(), "depth exceeds") {
		t.Fatalf("excessive nesting must be rejected, got %v", err)
	}
}

func TestFB2RejectsOversizedInput(t *testing.T) {
	r := strings.NewReader(strings.Repeat("x", int(parseutil.MaxTextBytes+1)))
	if _, err := (Parser{}).Parse(r); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversized FB2 must be rejected, got %v", err)
	}
}

func FuzzParse(f *testing.F) {
	f.Add(sample)
	f.Add("<FictionBook>")
	f.Add("\x00")
	f.Fuzz(func(t *testing.T, data string) {
		if len(data) > 1<<20 {
			t.Skip("bound fuzz input memory")
		}
		_, _ = (Parser{}).Parse(strings.NewReader(data))
	})
}

func TestFB2RejectsInvalidISBNMetadata(t *testing.T) {
	xml := strings.Replace(sample, "978-0-441-47812-5", "978-0-441-47812-4", 1)
	doc, err := Parser{}.Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := doc.Meta.Identifiers["isbn"]; ok {
		t.Fatalf("invalid ISBN accepted as deterministic evidence: %v", doc.Meta.Identifiers)
	}
}
