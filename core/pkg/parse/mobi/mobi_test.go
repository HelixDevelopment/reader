package mobi

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/helixreader/helix/core/pkg/parse/parseutil"
)

// buildTestMOBI synthesises a minimal PalmDOC-compressed MOBI container.
func buildTestMOBI(t *testing.T, compression uint16, text string) *bytes.Reader {
	t.Helper()
	half := len(text) / 2
	return buildMOBIRecords(t, compression, [][]byte{[]byte(text[:half]), []byte(text[half:])})
}

func buildMOBIRecords(t *testing.T, compression uint16, records [][]byte) *bytes.Reader {
	t.Helper()
	var recs [][]byte
	var textLength int
	for _, record := range records {
		textLength += len(record)
		if compression == 2 {
			var encoded []byte
			for len(record) > 0 {
				count := min(len(record), 8)
				encoded = append(encoded, byte(count))
				encoded = append(encoded, record[:count]...)
				record = record[count:]
			}
			recs = append(recs, encoded)
		} else {
			recs = append(recs, record)
		}
	}
	rec0 := make([]byte, 16)
	binary.BigEndian.PutUint16(rec0[0:2], compression)
	binary.BigEndian.PutUint32(rec0[4:8], uint32(textLength))
	binary.BigEndian.PutUint16(rec0[8:10], uint16(len(recs)))

	numRecords := 1 + len(recs)
	headerLen := 78 + numRecords*8
	offsets := []uint32{}
	off := uint32(headerLen)
	recsAll := append([][]byte{rec0}, recs...)
	for _, r := range recsAll {
		offsets = append(offsets, off)
		off += uint32(len(r))
	}

	var out bytes.Buffer
	hdr := make([]byte, 78)
	copy(hdr, "Test MOBI Book")
	binary.BigEndian.PutUint16(hdr[76:78], uint16(numRecords))
	out.Write(hdr)
	for _, o := range offsets {
		var b [8]byte
		binary.BigEndian.PutUint32(b[0:4], o)
		out.Write(b[:])
	}
	for _, r := range recsAll {
		out.Write(r)
	}
	return bytes.NewReader(out.Bytes())
}

func TestParseMOBINoCompression(t *testing.T) {
	text := "<p>First paragraph of the book.</p>\n<p>Second paragraph here.</p>\n"
	r := buildTestMOBI(t, 1, text)
	doc, err := Parser{}.Parse(r, int64(r.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Format != "mobi" || doc.Meta.Title != "Test MOBI Book" {
		t.Fatalf("meta: %+v", doc.Meta)
	}
	if len(doc.Chapters) != 1 || len(doc.Chapters[0].Blocks) != 2 {
		t.Fatalf("blocks: %+v", doc.Chapters)
	}
	if doc.Chapters[0].Blocks[0].Text != "First paragraph of the book." {
		t.Fatalf("tag strip: %q", doc.Chapters[0].Blocks[0].Text)
	}
}

func TestParseMOBIPalmDOCCompression(t *testing.T) {
	text := "<p>Compressed text content works.</p>\n"
	r := buildTestMOBI(t, 2, text)
	doc, err := Parser{}.Parse(r, int64(r.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Chapters[0].Blocks[0].Text != "Compressed text content works." {
		t.Fatalf("got %q", doc.Chapters[0].Blocks[0].Text)
	}
}

func TestBadRecordCount(t *testing.T) {
	buf := make([]byte, 78)
	binary.BigEndian.PutUint16(buf[76:78], 0)
	if _, err := (Parser{}).Parse(bytes.NewReader(buf), int64(len(buf))); err == nil {
		t.Fatal("zero records must error")
	}
}

func TestTruncated(t *testing.T) {
	if _, err := (Parser{}).Parse(bytes.NewReader([]byte("xx")), 2); err == nil {
		t.Fatal("truncated file must error")
	}
}

func TestPalmDOCTokenClasses(t *testing.T) {
	tests := []struct {
		name     string
		encoded  []byte
		expected []byte
	}{
		{name: "literals and byte pair", encoded: []byte{0x00, 0x03, 'a', 'b', 'c', 0x09, 0xc1}, expected: []byte{0x00, 'a', 'b', 'c', 0x09, ' ', 'A'}},
		{name: "back reference", encoded: []byte{0x03, 'a', 'b', 'c', 0x80, 0x18}, expected: []byte("abcabc")},
		{name: "overlapping back reference", encoded: []byte{'x', 0x80, 0x0f}, expected: bytes.Repeat([]byte{'x'}, 11)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := decompressPalmDOC(test.encoded, 1024)
			if err != nil || !bytes.Equal(actual, test.expected) {
				t.Fatalf("decoded %q, expected %q: %v", actual, test.expected, err)
			}
		})
	}
}

func TestPalmDOCRejectsMalformedAndOverflow(t *testing.T) {
	for _, encoded := range [][]byte{{0x02, 'x'}, {0x80}, {0x80, 0x08}} {
		if _, err := decompressPalmDOC(encoded, 1024); err == nil {
			t.Fatalf("malformed stream %v must fail", encoded)
		}
	}
	if _, err := decompressPalmDOC([]byte{'x', 0x80, 0x0f}, 10); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expanded record must be rejected, got %v", err)
	}
}

func TestMOBIRejectsAggregateTextOverflow(t *testing.T) {
	block := bytes.Repeat([]byte{'x'}, int(parseutil.MaxEntryBytes))
	r := buildMOBIRecords(t, 1, [][]byte{block, block, block, block, []byte("overflow")})
	_, err := (Parser{}).Parse(r, int64(r.Len()))
	if err == nil || !strings.Contains(err.Error(), "decoded text exceeds") {
		t.Fatalf("aggregate text must be rejected, got %v", err)
	}
}

func FuzzParse(f *testing.F) {
	f.Add([]byte("xx"))
	header := make([]byte, 78)
	binary.BigEndian.PutUint16(header[76:78], 2)
	f.Add(header)
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			t.Skip("bound fuzz input memory")
		}
		_, _ = (Parser{}).Parse(bytes.NewReader(data), int64(len(data)))
	})
}
