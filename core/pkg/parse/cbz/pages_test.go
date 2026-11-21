package cbz

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	"github.com/helixreader/helix/core/pkg/parse/parseutil"
)

func TestPagesEnumeration(t *testing.T) {
	r := makeCBZ(t, []string{"p10.png", "p02.png", "p01.png", "notes.txt"})
	refs, err := Pages(r, int64(r.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 3 {
		t.Fatalf("refs: %+v", refs)
	}
	wantNames := []string{"p01.png", "p02.png", "p10.png"}
	for i, want := range wantNames {
		got := refs[i]
		if got.Index != i || got.ID != fmt.Sprintf("page-%04d", i+1) || got.Name != want {
			t.Fatalf("ref %d: %+v want %s", i, got, want)
		}
		if got.Size != uint64(len("img:"+want)) {
			t.Fatalf("ref %d size: %d", i, got.Size)
		}
		if got.CompressedSize == 0 {
			t.Fatalf("ref %d compressed size missing", i)
		}
	}
}

// TestPagesIDsAgreeWithParse asserts the enumeration contract: the ids and
// order Pages reports are exactly what the full parser materialises.
func TestPagesIDsAgreeWithParse(t *testing.T) {
	names := []string{"b/10.jpeg", "b/2.jpeg", "a.png", "upper.PNG", "cover.webp"}
	r := makeCBZ(t, names)
	refs, err := Pages(bytes.NewReader(toBytes(t, r)), int64(r.Len()))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := (Parser{}).Parse(bytes.NewReader(toBytes(t, r)), int64(r.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != len(doc.Resources) {
		t.Fatalf("pages %d vs parse %d", len(refs), len(doc.Resources))
	}
	for _, ref := range refs {
		data, ok := doc.Resources[ref.ID]
		if !ok || string(data) != "img:"+ref.Name {
			t.Fatalf("page id %s does not match parsed payload for %s", ref.ID, ref.Name)
		}
	}
}

func toBytes(t *testing.T, r *bytes.Reader) []byte {
	t.Helper()
	data := make([]byte, r.Size())
	if _, err := r.ReadAt(data, 0); err != nil {
		t.Fatal(err)
	}
	return data
}

func TestPagesFailsClosed(t *testing.T) {
	if _, err := Pages(bytes.NewReader([]byte("not a zip")), 9); err == nil || !strings.Contains(err.Error(), "open zip") {
		t.Fatalf("corrupt archive must error, got %v", err)
	}
	r := makeCBZ(t, []string{"notes.txt"})
	if _, err := Pages(r, int64(r.Len())); err == nil || !strings.Contains(err.Error(), "no images") {
		t.Fatalf("image-less archive must error, got %v", err)
	}
	names := make([]string, parseutil.MaxPages+1)
	for i := range names {
		names[i] = fmt.Sprintf("page-%05d.png", i)
	}
	r = makeCBZ(t, names)
	if _, err := Pages(r, int64(r.Len())); err == nil || !strings.Contains(err.Error(), "page count exceeds") {
		t.Fatalf("excess pages must error, got %v", err)
	}
}

// TestPagesDoesNotInflate proves Pages reads only the central directory: an
// archive whose local file headers are corrupted is still enumerable, while
// the full parser (which must inflate payloads) rejects the same bytes.
func TestPagesDoesNotInflate(t *testing.T) {
	raw := toBytes(t, makeCBZ(t, []string{"a.png", "b.png"}))
	// Corrupt the CRC-32 field of the first *central directory* header
	// ('PK\x01\x02', CRC at offset 16). Go's zip reader ignores the local
	// header's copy of these fields, so only the central directory drives
	// payload verification; enumeration itself never consults it.
	idx := bytes.Index(raw, []byte("PK\x01\x02"))
	if idx < 0 {
		t.Fatal("fixture is not a zip")
	}
	binary.LittleEndian.PutUint32(raw[idx+16:], 0xDEADBEEF)
	r := bytes.NewReader(raw)
	refs, err := Pages(r, int64(len(raw)))
	if err != nil {
		t.Fatalf("directory enumeration must survive payload corruption: %v", err)
	}
	if len(refs) != 2 || refs[0].Name != "a.png" || refs[1].Name != "b.png" {
		t.Fatalf("refs: %+v", refs)
	}
	if _, err := (Parser{}).Parse(bytes.NewReader(raw), int64(len(raw))); err == nil {
		t.Fatal("full parser must reject corrupted payload")
	}
}

func FuzzPages(f *testing.F) {
	f.Add([]byte("not a zip"))
	f.Add([]byte("PK\x03\x04"))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			t.Skip("bound fuzz input memory")
		}
		_, _ = Pages(bytes.NewReader(data), int64(len(data)))
	})
}
