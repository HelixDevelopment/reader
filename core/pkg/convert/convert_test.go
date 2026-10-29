package convert

import (
	"strings"
	"testing"

	"github.com/helixreader/helix/core/pkg/hdm"
)

func docFixture() *hdm.Document {
	return &hdm.Document{
		Meta: hdm.Metadata{Title: "T & <Co>", Creators: []string{"A&B"}, Language: "en",
			Identifiers: map[string]string{}},
		Chapters: []*hdm.Chapter{{ID: "c1", Title: "Ch1", Blocks: []hdm.Block{
			{Kind: hdm.BlockHeading, Level: 1, Text: "Intro"},
			{Kind: hdm.BlockParagraph, Text: "One <two> & three."},
			{Kind: hdm.BlockQuote, Text: "Said \"she\"."},
			{Kind: hdm.BlockImage, ImageRef: "im"},
		}}},
		Resources: map[string][]byte{"im": {1}},
	}
}

func TestTXT(t *testing.T) {
	out, err := Convert(docFixture(), FormatTXT, DefaultOptions)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, "T & <Co>") || !strings.Contains(s, "# Intro") || !strings.Contains(s, "One <two> & three.") {
		t.Fatalf("txt out:\n%s", s)
	}
	if strings.Contains(s, "im") && strings.Contains(s, "BlockImage") {
		t.Fatal("images must be skipped in txt")
	}
}

func TestTXTWrapDeterministic(t *testing.T) {
	o1, _ := Convert(docFixture(), FormatTXT, Options{LineWidth: 10})
	o2, _ := Convert(docFixture(), FormatTXT, Options{LineWidth: 10})
	if string(o1) != string(o2) {
		t.Fatal("wrapping not deterministic")
	}
	for _, line := range strings.Split(string(o1), "\n") {
		if len(line) > 10 && strings.Contains(line, " ") {
			t.Fatalf("line over width: %q", line)
		}
	}
}

func TestXHTMLEscaping(t *testing.T) {
	out, err := Convert(docFixture(), FormatXHTML, DefaultOptions)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, "T &amp; &lt;Co&gt;") {
		t.Fatalf("title not escaped: %s", s)
	}
	if !strings.Contains(s, "One &lt;two&gt; &amp; three.") {
		t.Fatalf("paragraph not escaped: %s", s)
	}
	if !strings.Contains(s, "Said &quot;she&quot;.") {
		t.Fatalf("quote not escaped: %s", s)
	}
	if strings.Contains(s, "One <two>") {
		t.Fatal("raw html leaked")
	}
}

func TestFB2RoundTrip(t *testing.T) {
	out, err := Convert(docFixture(), FormatFB2, DefaultOptions)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, "<book-title>T &amp; &lt;Co&gt;</book-title>") {
		t.Fatalf("fb2 out:\n%s", s)
	}
	if !strings.Contains(s, "<section>") || !strings.Contains(s, "</FictionBook>") {
		t.Fatal("fb2 structure")
	}
}

func TestUnsupported(t *testing.T) {
	if _, err := Convert(docFixture(), "azw3", DefaultOptions); err == nil {
		t.Fatal("azw3 on-device must be unsupported (server-side per C-2)")
	}
}

func TestInvalidDoc(t *testing.T) {
	if _, err := Convert(&hdm.Document{}, FormatTXT, DefaultOptions); err == nil {
		t.Fatal("invalid doc must error")
	}
}
