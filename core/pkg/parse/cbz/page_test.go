package cbz

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/helixreader/helix/core/pkg/parse/parseutil"
)

var (
	pngSig  = append([]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}, []byte("-png")...)
	jpgSig  = append([]byte{0xff, 0xd8, 0xff}, []byte("-jpg")...)
	gifSig  = append([]byte("GIF87a"), []byte("-gi")...)
	webpSig = append([]byte("RIFF\x02\x00\x00\x00WEBP"), []byte("x")...)
)

type entry struct {
	name    string
	payload []byte
}

// makeEntries writes the archive with entries in the given order, which tests
// keep different from lexicographic order so reading order is not an artifact
// of archive layout.
func makeEntries(t *testing.T, entries []entry) *bytes.Reader {
	t.Helper()
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	for _, e := range entries {
		w, err := zw.Create(e.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(e.payload); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(buf.Bytes())
}

func pages(entries ...entry) []entry { return entries }

func TestPagePayloadsFollowEnumeratedOrder(t *testing.T) {
	r := makeEntries(t, pages(
		entry{"p10.png", pngSig},
		entry{"notes.txt", []byte("not a page")},
		entry{"cover.jpeg", jpgSig},
		entry{"p01.png", append(append([]byte{}, pngSig...), 'A')},
		entry{"p02.png", append(append([]byte{}, pngSig...), 'B')},
	))
	size := int64(r.Len())
	refs, err := Pages(r, size)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 4 {
		t.Fatalf("page count = %d, want 4 (%+v)", len(refs), refs)
	}
	want := []string{"cover.jpeg", "p01.png", "p02.png", "p10.png"}
	for i, name := range want {
		ref, data, err := Page(r, size, i)
		if err != nil {
			t.Fatalf("page %d: %v", i, err)
		}
		if ref.Name != name || ref.Index != i || ref.ID != fmt.Sprintf("page-%04d", i+1) {
			t.Errorf("page %d = %+v, want name %q", i, ref, name)
		}
		if int64(len(data)) != int64(ref.Size) {
			t.Errorf("page %d payload %d bytes, central directory says %d", i, len(data), ref.Size)
		}
		if i == 0 {
			if !bytes.Equal(data, jpgSig) {
				t.Errorf("cover page payload = %v, want the stored jpeg", data)
			}
			continue
		}
		if !bytes.HasPrefix(data, pngSig) {
			t.Errorf("page %d payload %q is not the stored png", i, data)
		}
	}
}

func TestPageIndexOutOfRange(t *testing.T) {
	r := makeEntries(t, pages(entry{"p01.png", pngSig}, entry{"p02.png", pngSig}))
	size := int64(r.Len())
	for _, i := range []int{-1, 2, 3} {
		_, _, err := Page(r, size, i)
		if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("page index %d out of range 0..1", i)) {
			t.Errorf("Page(%d) error = %v, want out of range 0..1", i, err)
		}
	}
}

// A page larger than the per-image limit must not be inflated whole.
func TestPageInflationIsBounded(t *testing.T) {
	r := makeEntries(t, pages(entry{"p01.png", append(append([]byte{}, pngSig...), bytes.Repeat([]byte{0}, int(parseutil.MaxImageBytes))...)}))
	if _, _, err := Page(r, int64(r.Len()), 0); err == nil {
		t.Fatal("page larger than the per-image limit must not be inflated")
	}
}

// Typing a stream reads each entry's signature only, so it stays honest and
// cheap on an archive whose pages individually exceed the inflate limit —
// exactly the archive a page-by-page reader exists for.
func TestStreamMediaTypeReadsOnlySignatures(t *testing.T) {
	big := append(append([]byte{}, pngSig...), bytes.Repeat([]byte{0}, int(parseutil.MaxImageBytes))...)
	r := makeEntries(t, pages(entry{"p01.png", big}, entry{"p02.png", big}))
	got, err := StreamMediaType(r, int64(r.Len()))
	if err != nil {
		t.Fatalf("StreamMediaType: %v", err)
	}
	if got != "image/png" {
		t.Fatalf("StreamMediaType = %q, want image/png", got)
	}
	if _, _, err := Page(r, int64(r.Len()), 0); err == nil {
		t.Fatal("fixture stopped proving the two costs differ: Page inflated an over-limit entry")
	}
}

func TestStreamMediaTypeCases(t *testing.T) {
	tests := []struct {
		name    string
		entries []entry
		want    string
		wantErr string
	}{
		{name: "png", entries: pages(entry{"a.png", pngSig}, entry{"b.png", pngSig}), want: "image/png"},
		{name: "jpeg", entries: pages(entry{"a.jpg", jpgSig}), want: "image/jpeg"},
		{name: "gif", entries: pages(entry{"a.gif", gifSig}), want: "image/gif"},
		{
			name:    "mixed types have no single honest type",
			entries: pages(entry{"a.png", pngSig}, entry{"b.jpg", jpgSig}),
			wantErr: "pages mix image/png and image/jpeg",
		},
		{
			name:    "extension is not proof",
			entries: pages(entry{"a.png", []byte("PNG but really text")}),
			wantErr: `page "a.png" is not a jpeg, png or gif image`,
		},
		{
			name:    "webp is not streamable",
			entries: pages(entry{"a.webp", webpSig}),
			wantErr: `page "a.webp" is not a jpeg, png or gif image`,
		},
		{
			name:    "payload shorter than any signature",
			entries: pages(entry{"a.gif", []byte("GI")}),
			wantErr: `page "a.gif" is not a jpeg, png or gif image`,
		},
		{
			name:    "empty payload",
			entries: pages(entry{"a.png", nil}),
			wantErr: `page "a.png" is not a jpeg, png or gif image`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := makeEntries(t, tc.entries)
			got, err := StreamMediaType(r, int64(r.Len()))
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want one containing %q", err, tc.wantErr)
				}
				if got != "" {
					t.Errorf("a refused stream must report no type, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("StreamMediaType = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSniffMediaType(t *testing.T) {
	tests := map[string]string{
		"":                   "",
		"GI":                 "",
		"GIF8":               "",
		"GIF87a":             "image/gif",
		"GIF89a!":            "image/gif",
		"\x89PNG\r\n\x1a\nx": "image/png",
		"\x89PNG\r\n\x1a!":   "",
		"\xff\xd8\xff\xe0":   "image/jpeg",
		"\xff\xd8\x00":       "",
	}
	for payload, want := range tests {
		if got := SniffMediaType([]byte(payload)); got != want {
			t.Errorf("SniffMediaType(%q) = %q, want %q", payload, got, want)
		}
	}
}
