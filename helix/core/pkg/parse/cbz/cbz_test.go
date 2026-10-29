package cbz

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/helixreader/helix/core/pkg/parse/parseutil"
)

func makeCBZ(t *testing.T, names []string) *bytes.Reader {
	t.Helper()
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	for _, n := range names {
		w, _ := zw.Create(n)
		w.Write([]byte("img:" + n))
	}
	zw.Close()
	return bytes.NewReader(buf.Bytes())
}

func TestCBZPageOrder(t *testing.T) {
	// intentionally unsorted + a non-image file + a directory entry
	r := makeCBZ(t, []string{"p10.png", "p02.png", "p01.png", "notes.txt"})
	doc, err := Parser{}.Parse(r, int64(r.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Resources) != 3 {
		t.Fatalf("resources: %d", len(doc.Resources))
	}
	// deterministic lexicographic order: p01, p02, p10
	data := doc.Resources["page-0001"]
	if string(data) != "img:p01.png" {
		t.Fatalf("page 1 = %q", data)
	}
	if string(doc.Resources["page-0003"]) != "img:p10.png" {
		t.Fatalf("page 3 = %q", doc.Resources["page-0003"])
	}
	if len(doc.Chapters[0].Blocks) != 3 {
		t.Fatal("blocks per page")
	}
}

func TestCBZNoImages(t *testing.T) {
	r := makeCBZ(t, []string{"readme.txt"})
	if _, err := (Parser{}).Parse(r, int64(r.Len())); err == nil {
		t.Fatal("no images must error")
	}
}

func TestCBZRejectsOversizedImage(t *testing.T) {
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	w, err := zw.Create("page.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(bytes.Repeat([]byte{'x'}, int(parseutil.MaxImageBytes+1))); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	r := bytes.NewReader(buf.Bytes())
	_, err = (Parser{}).Parse(r, int64(r.Len()))
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversized image must be rejected, got %v", err)
	}
}

func TestCBZRejectsExcessPages(t *testing.T) {
	names := make([]string, parseutil.MaxPages+1)
	for i := range names {
		names[i] = fmt.Sprintf("page-%05d.png", i)
	}
	r := makeCBZ(t, names)
	_, err := (Parser{}).Parse(r, int64(r.Len()))
	if err == nil || !strings.Contains(err.Error(), "page count exceeds") {
		t.Fatalf("excess pages must be rejected, got %v", err)
	}
}

func FuzzParse(f *testing.F) {
	f.Add([]byte("not a zip"))
	f.Add([]byte("PK\x03\x04"))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			t.Skip("bound fuzz input memory")
		}
		_, _ = (Parser{}).Parse(bytes.NewReader(data), int64(len(data)))
	})
}
