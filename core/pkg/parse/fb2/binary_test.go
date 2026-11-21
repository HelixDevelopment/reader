package fb2

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/helixreader/helix/core/pkg/parse/parseutil"
)

const binarySample = `<?xml version="1.0" encoding="utf-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink">
 <description><title-info><book-title>Ill</book-title><lang>en</lang></title-info></description>
 <body>
  <section>
   <title><p>Chapter 1</p></title>
   <p>Text before image.</p>
   <p>Inline <image l:href="#cover"/> here.</p>
   <image l:href="#plate"/>
  </section>
 </body>
 <binary id="cover" content-type="image/png">aGk=</binary>
 <binary id="plate" content-type="image/jpeg">aGk=</binary>
</FictionBook>`

func TestFB2BinariesBecomeResources(t *testing.T) {
	doc, err := Parser{}.Parse(strings.NewReader(binarySample))
	if err != nil {
		t.Fatal(err)
	}
	if string(doc.Resources["cover"]) != "hi" || string(doc.Resources["plate"]) != "hi" {
		t.Fatalf("resources: %v", doc.Resources)
	}
	blocks := doc.Chapters[0].Blocks
	want := []struct {
		kind string
		text string
		ref  string
	}{
		{"heading", "Chapter 1", ""},
		{"paragraph", "Text before image.", ""},
		{"paragraph", "Inline here.", ""},
		{"image", "", "cover"},
		{"image", "", "plate"},
	}
	if len(blocks) != len(want) {
		t.Fatalf("blocks: %+v", blocks)
	}
	for i, w := range want {
		if blocks[i].Kind.String() != w.kind || blocks[i].Text != w.text || blocks[i].ImageRef != w.ref {
			t.Fatalf("block %d: %+v want %+v", i, blocks[i], w)
		}
	}
}

func TestFB2DanglingImageHrefSkipped(t *testing.T) {
	xml := strings.Replace(binarySample, `<image l:href="#plate"/>`, `<image l:href="#missing"/>`, 1)
	doc, err := Parser{}.Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range doc.Chapters[0].Blocks {
		if b.ImageRef == "missing" {
			t.Fatal("dangling reference must not produce a block")
		}
	}
}

func TestFB2BinariesFailClosed(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		wantErr string
	}{
		{"malformed base64", `<FictionBook><body><section><p>t</p></section></body><binary id="x" content-type="image/png">!!!</binary></FictionBook>`, "binary x"},
		{"empty id", `<FictionBook><body><section><p>t</p></section></body><binary id="" content-type="image/png">aGk=</binary></FictionBook>`, "empty id"},
		{"duplicate id", `<FictionBook><body><section><p>t</p></section></body><binary id="x">aGk=</binary><binary id="x">aGk=</binary></FictionBook>`, "duplicate binary id"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := (Parser{}).Parse(strings.NewReader(tc.fixture))
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("want error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

// TestFB2RejectsOversizedBinary: base64 inside the 64 MiB input budget that
// decodes beyond MaxImageBytes must be rejected per-entry (decompression
// bomb analogue for XML-embedded payloads).
func TestFB2RejectsOversizedBinary(t *testing.T) {
	bomb := base64.StdEncoding.EncodeToString(bytesRepeatA(int(parseutil.MaxImageBytes) + 1))
	input := fmt.Sprintf(`<FictionBook><body><section><p>t</p></section></body><binary id="x" content-type="image/png">%s</binary></FictionBook>`, bomb)
	if len(input) > int(parseutil.MaxTextBytes) {
		t.Fatalf("fixture must stay inside the input cap: %d", len(input))
	}
	_, err := (Parser{}).Parse(strings.NewReader(input))
	if err == nil || !strings.Contains(err.Error(), "over") {
		t.Fatalf("oversized binary must fail closed, got %v", err)
	}
}

// TestFB2BinaryAggregateBoundIsArithmeticallySound documents why no input
// inside the parse budget can overflow the aggregate cap: a 64 MiB capped
// input decodes to at most 48 MiB of binaries (< 128 MiB aggregate), so the
// per-entry 32 MiB check is the binding constraint with current constants.
// If either cap changes, this test fails and the parser's checks must be
// re-audited.
func TestFB2BinaryAggregateBoundIsArithmeticallySound(t *testing.T) {
	maxDecoded := parseutil.MaxTextBytes * 3 / 4
	if maxDecoded > parseutil.MaxAggregateBytes {
		t.Fatalf("input cap %d can overflow aggregate cap %d after base64 decode; per-entry checks are insufficient", parseutil.MaxTextBytes, parseutil.MaxAggregateBytes)
	}
}

func bytesRepeatA(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'a'
	}
	return b
}

// TestFB2UnclosedNestedSectionsNoPanic is the regression guard for the fuzz
// crash seed in testdata/fuzz/FuzzParse: an unclosed inner <section> made
// the recursive decoder yield a zero section that entered Children as nil.
func TestFB2UnclosedNestedSectionsNoPanic(t *testing.T) {
	doc, err := (Parser{}).Parse(strings.NewReader("<FictionBook><body><section><section></FictionBook>"))
	if err == nil {
		for _, ch := range doc.Chapters {
			for _, c := range ch.Children {
				if c == nil {
					t.Fatal("nil child chapter accepted")
				}
			}
		}
	}
}
