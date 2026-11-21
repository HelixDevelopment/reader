package epub

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	"github.com/helixreader/helix/core/pkg/parse/parseutil"
)

func TestSpineEnumeration(t *testing.T) {
	r := buildTestEPUB(t)
	pages, err := Spine(r, int64(r.Len()))
	if err != nil {
		t.Fatal(err)
	}
	want := []SpinePage{
		{Index: 0, IDRef: "ch1", Entry: "OEBPS/ch1.xhtml", MediaType: "application/xhtml+xml", DuplicateOf: -1},
		{Index: 1, IDRef: "ch2", Entry: "OEBPS/ch2.xhtml", MediaType: "application/xhtml+xml", DuplicateOf: -1},
	}
	if len(pages) != len(want) {
		t.Fatalf("pages: %+v", pages)
	}
	for i := range want {
		g := pages[i]
		if g.IDRef != want[i].IDRef || g.Entry != want[i].Entry || g.MediaType != want[i].MediaType || g.DuplicateOf != want[i].DuplicateOf {
			t.Fatalf("page %d: %+v want %+v", i, g, want[i])
		}
	}
	// Sizes come from the central directory without inflating payloads and
	// must equal the decompressed length the full parser would read.
	if pages[0].Size != uint64(len(buildTestEPUBChapter1)) {
		t.Fatalf("page 0 size: %d want %d", pages[0].Size, len(buildTestEPUBChapter1))
	}
}

// buildTestEPUBChapter1 mirrors the ch1.xhtml payload in buildTestEPUB;
// TestSpineEnumeration asserts the central-directory size against it.
const buildTestEPUBChapter1 = `<html><body><h1>Chapter   One</h1><p>Hello   world.
Second line.</p><p>Another para.</p></body></html>`

func TestSpineDetectsRepeatsAndMissingManifestItems(t *testing.T) {
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	add := func(name, content string) {
		w, _ := zw.Create(name)
		w.Write([]byte(content))
	}
	add("META-INF/container.xml", `<container><rootfiles><rootfile full-path="content.opf"/></rootfiles></container>`)
	add("content.opf", `<package><manifest><item id="a" href="a.xhtml" media-type="application/xhtml+xml"/></manifest>
<spine><itemref idref="a"/><itemref idref="ghost"/><itemref idref="a"/></spine></package>`)
	add("a.xhtml", `<html><body><p>x</p></body></html>`)
	zw.Close()
	r := bytes.NewReader(buf.Bytes())
	pages, err := Spine(r, int64(r.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 3 || pages[1].Entry != "" || pages[2].DuplicateOf != 0 {
		t.Fatalf("repeat/ghost detection: %+v", pages)
	}
}

func spineZip(t *testing.T, entries ...[2]string) *bytes.Reader {
	t.Helper()
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	for _, e := range entries {
		w, err := zw.Create(e[0])
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(e[1])); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(buf.Bytes())
}

// TestSpineFailsClosed proves the bounded-input guards on the enumeration
// path: oversized directory documents, malformed XML, and hostile hrefs all
// error, never panic.
func TestSpineFailsClosed(t *testing.T) {
	container := `<container><rootfiles><rootfile full-path="c.opf"/></rootfiles></container>`
	opf := `<package><manifest><item id="a" href="a.xhtml" media-type="application/xhtml+xml"/></manifest><spine><itemref idref="a"/></spine></package>`
	tests := []struct {
		name    string
		data    *bytes.Reader
		wantErr string
	}{
		{"not a zip", bytes.NewReader([]byte("junk")), "open zip"},
		{"missing container", spineZip(t, [2]string{"c.opf", opf}), "missing META-INF/container.xml"},
		{"oversized container", spineZip(t, [2]string{"META-INF/container.xml", strings.Repeat("x", int(parseutil.MaxEntryBytes)+1)}), "exceeds"},
		{"malformed container xml", spineZip(t, [2]string{"META-INF/container.xml", "<container><rootfiles>"}), "container.xml"},
		{"no rootfile", spineZip(t, [2]string{"META-INF/container.xml", "<container><rootfiles/></container>"}, [2]string{"c.opf", opf}), "no rootfile"},
		{"missing opf", spineZip(t, [2]string{"META-INF/container.xml", container}), "missing c.opf"},
		{"empty spine", spineZip(t, [2]string{"META-INF/container.xml", container}, [2]string{"c.opf", "<package><manifest/></package>"}), "empty spine"},
		{"absolute href", spineZip(t, [2]string{"META-INF/container.xml", container},
			[2]string{"c.opf", `<package><manifest><item id="a" href="http://evil/x.xhtml" media-type="application/xhtml+xml"/></manifest><spine><itemref idref="a"/></spine></package>`}), "non-local"},
		{"root-path href", spineZip(t, [2]string{"META-INF/container.xml", container},
			[2]string{"c.opf", `<package><manifest><item id="a" href="/escape.xhtml" media-type="application/xhtml+xml"/></manifest><spine><itemref idref="a"/></spine></package>`}), "non-local"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Spine(tc.data, int64(tc.data.Len()))
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("want error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

// TestSpineCompressedBombIsBounded: a container.xml that deflates to more
// than MaxEntryBytes must be rejected while reading; the on-disk archive
// stays tiny, so only the streaming guard can stop the bomb.
func TestSpineCompressedBombIsBounded(t *testing.T) {
	bomb := strings.Repeat("a", int(parseutil.MaxEntryBytes)+1024)
	r := spineZip(t,
		[2]string{"META-INF/container.xml", bomb},
		[2]string{"c.opf", `<package><spine><itemref idref="a"/></spine></package>`},
	)
	if r.Len() >= len(bomb)/4 {
		t.Fatalf("bomb fixture should compress hard, got %d raw bytes", r.Len())
	}
	_, err := Spine(r, int64(r.Len()))
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("compressed bomb must fail closed, got %v", err)
	}
}

// TestSpineRejectsExcessSpineItems: spine length is capped by MaxPages.
func TestSpineRejectsExcessSpineItems(t *testing.T) {
	var sb strings.Builder
	sb.WriteString(`<package><manifest>`)
	var spine strings.Builder
	for i := 0; i < parseutil.MaxPages+1; i++ {
		// unique entries would also need unique ZIP files; referencing the
		// same item repeatedly exercises both the page cap and dedupe map
		// without growing the archive.
		if i == 0 {
			sb.WriteString(`<item id="a" href="a.xhtml" media-type="application/xhtml+xml"/>`)
		}
		spine.WriteString(`<itemref idref="a"/>`)
	}
	sb.WriteString(`</manifest><spine>`)
	sb.WriteString(spine.String())
	sb.WriteString(`</spine></package>`)
	r := spineZip(t,
		[2]string{"META-INF/container.xml", `<container><rootfiles><rootfile full-path="c.opf"/></rootfiles></container>`},
		[2]string{"c.opf", sb.String()},
		[2]string{"a.xhtml", `<html><body><p>x</p></body></html>`},
	)
	_, err := Spine(r, int64(r.Len()))
	if err == nil || !strings.Contains(err.Error(), "spine page count exceeds") {
		t.Fatalf("excess spine items must fail closed, got %v", err)
	}
}

func FuzzSpine(f *testing.F) {
	f.Add([]byte("not a zip"))
	f.Add([]byte("PK\x03\x04"))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			t.Skip("bound fuzz input memory")
		}
		_, _ = Spine(bytes.NewReader(data), int64(len(data)))
	})
}
