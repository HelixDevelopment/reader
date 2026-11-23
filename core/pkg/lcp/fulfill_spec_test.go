package lcp

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

// rehash replaces the link's hash with the given encoding of its own digest.
func rehash(enc func([]byte) string) func(map[string]any) {
	return func(l map[string]any) {
		raw, _ := hex.DecodeString(l["hash"].(string))
		l["hash"] = enc(raw)
	}
}

func TestFulfillAcceptsHexAndBase64Hash(t *testing.T) {
	cases := map[string]func(map[string]any){
		"lowercase hex (spec Example 3, reference server)": nil,
		"uppercase hex":                      rehash(func(b []byte) string { return strings.ToUpper(hex.EncodeToString(b)) }),
		"std base64 (spec table and schema)": rehash(func(b []byte) string { return base64.StdEncoding.EncodeToString(b) }),
	}
	for name, mut := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFulfillFixture(t, mut)
			if _, err := f.fulfill(); err != nil {
				t.Fatalf("Fulfill: %v", err)
			}
		})
	}
}

func TestFulfillRejectsBadHashEncodings(t *testing.T) {
	cases := map[string]func(map[string]any){
		"base64 of 16 bytes": func(l map[string]any) { l["hash"] = base64.StdEncoding.EncodeToString(make([]byte, 16)) },
		"base64 of other digest": func(l map[string]any) {
			s := sha256.Sum256([]byte("other"))
			l["hash"] = base64.StdEncoding.EncodeToString(s[:])
		},
		"short hex":  func(l map[string]any) { l["hash"] = strings.Repeat("ab", 16) },
		"url base64": func(l map[string]any) { l["hash"] = strings.Repeat("_", 43) + "=" },
		"empty-ish":  func(l map[string]any) { l["hash"] = "  " },
	}
	for name, mut := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := newFulfillFixture(t, mut).fulfill()
			if !errors.Is(err, ErrPublicationMismatch) {
				t.Fatalf("err = %v, want ErrPublicationMismatch", err)
			}
		})
	}
}

func TestFulfillToleratesStringLength(t *testing.T) {
	f := newFulfillFixture(t, func(l map[string]any) { l["length"] = strconv.Itoa(l["length"].(int)) })
	if _, err := f.fulfill(); err != nil {
		t.Fatalf("string length (spec Example 3): %v", err)
	}
	lic, err := Parse(f.license)
	if err != nil {
		t.Fatal(err)
	}
	l, _ := lic.PublicationLink()
	if l.Length != int64(len(f.pub)) {
		t.Errorf("Length = %d, want %d", l.Length, len(f.pub))
	}
}

func TestParseRejectsBadLengths(t *testing.T) {
	for name, v := range map[string]any{
		"negative number": -5, "negative string": "-5", "non-numeric": "abc",
		"float": 1.5, "empty string": "", "bool": true, "object": map[string]any{},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFulfillFixture(t, func(l map[string]any) { l["length"] = v })
			if _, err := Parse(f.license); !errors.Is(err, ErrLicenseSyntax) {
				t.Fatalf("Parse err = %v, want ErrLicenseSyntax", err)
			}
		})
	}
}

func TestFulfillRefusesNonEPUBTypes(t *testing.T) {
	for _, typ := range []string{"application/pdf+lcp", "application/audiobook+lcp", "application/divina+lcp", "application/pdf"} {
		t.Run(typ, func(t *testing.T) {
			_, err := newFulfillFixture(t, func(l map[string]any) { l["type"] = typ }).fulfill()
			if !errors.Is(err, ErrUnsupportedFormat) {
				t.Fatalf("err = %v, want ErrUnsupportedFormat", err)
			}
		})
	}
	for name, mut := range map[string]func(map[string]any){
		"parameters and case": func(l map[string]any) { l["type"] = "Application/EPUB+zip; charset=binary" },
		"type absent":         func(l map[string]any) { delete(l, "type") },
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := newFulfillFixture(t, mut).fulfill(); err != nil {
				t.Fatalf("Fulfill: %v", err)
			}
		})
	}
}

func TestFulfillAllowUnverified(t *testing.T) {
	both := func(l map[string]any) { delete(l, "hash"); delete(l, "length") }
	if _, err := newFulfillFixture(t, both).fulfill(); !errors.Is(err, ErrPublicationUnverifiable) {
		t.Fatalf("default: err = %v, want ErrPublicationUnverifiable", err)
	}
	call := func(f fulfillFixture) error {
		_, err := Fulfill(f.license, bytes.NewReader(f.pub), int64(len(f.pub)), AllowUnverified())
		return err
	}
	if err := call(newFulfillFixture(t, both)); err != nil {
		t.Fatalf("AllowUnverified with neither: %v", err)
	}
	if err := call(newFulfillFixture(t, func(l map[string]any) { delete(l, "hash") })); err != nil {
		t.Fatalf("AllowUnverified, length only: %v", err)
	}
	if err := call(newFulfillFixture(t, func(l map[string]any) { delete(l, "length") })); err != nil {
		t.Fatalf("AllowUnverified, hash only: %v", err)
	}
	// What is stated is always enforced.
	if err := call(newFulfillFixture(t, func(l map[string]any) { delete(l, "hash"); l["length"] = 7 })); !errors.Is(err, ErrPublicationMismatch) {
		t.Errorf("wrong length: %v", err)
	}
	if err := call(newFulfillFixture(t, func(l map[string]any) { delete(l, "length"); l["hash"] = strings.Repeat("0", 64) })); !errors.Is(err, ErrPublicationMismatch) {
		t.Errorf("wrong hash: %v", err)
	}
	if err := call(newFulfillFixture(t, func(l map[string]any) { delete(l, "length"); l["hash"] = "zz" })); !errors.Is(err, ErrPublicationMismatch) {
		t.Errorf("undecodable hash: %v", err)
	}
}

func TestFulfillToMatchesFulfill(t *testing.T) {
	f := newFulfillFixture(t, nil)
	want, err := f.fulfill()
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := FulfillTo(&buf, f.license, bytes.NewReader(f.pub), int64(len(f.pub))); err != nil {
		t.Fatalf("FulfillTo: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Error("FulfillTo output differs from Fulfill")
	}
	var bad bytes.Buffer
	pub := append([]byte{}, f.pub...)
	pub[len(pub)/2] ^= 1
	if err := FulfillTo(&bad, f.license, bytes.NewReader(pub), int64(len(pub))); !errors.Is(err, ErrPublicationMismatch) {
		t.Errorf("tampered: %v", err)
	}
	if bad.Len() != 0 {
		t.Errorf("wrote %d bytes to the destination before verification passed", bad.Len())
	}
}

// shifty serves honest bytes for the first pass and altered ones afterwards:
// a source that is not immutable between FulfillTo's passes.
type shifty struct {
	good, alt []byte
	reads     atomic.Int64
	after     int64
}

func (s *shifty) ReadAt(p []byte, off int64) (int, error) {
	src := s.good
	if s.reads.Add(1) > s.after {
		src = s.alt
	}
	if off >= int64(len(src)) {
		return 0, io.EOF
	}
	n := copy(p, src[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func TestFulfillToDetectsSourceChangingBetweenPasses(t *testing.T) {
	f := newFulfillFixture(t, nil)
	alt := append([]byte{}, f.pub...)
	alt[len(alt)-30] ^= 1 // inside the central directory/EOCD region: structure still opens
	// Count the reads a clean run performs, then switch part-way through.
	probe := &shifty{good: f.pub, alt: f.pub, after: 1 << 40}
	if err := FulfillTo(io.Discard, f.license, probe, int64(len(f.pub))); err != nil {
		t.Fatal(err)
	}
	total := probe.reads.Load()
	s := &shifty{good: f.pub, alt: alt, after: total - 2}
	err := FulfillTo(io.Discard, f.license, s, int64(len(f.pub)))
	if !errors.Is(err, ErrPublicationMismatch) {
		t.Fatalf("err = %v, want ErrPublicationMismatch (reads=%d)", err, total)
	}
}

func TestFulfillToSourceShorterThanDeclared(t *testing.T) {
	f := newFulfillFixture(t, nil)
	short := f.pub[:len(f.pub)-10]
	err := FulfillTo(io.Discard, f.license, bytes.NewReader(short), int64(len(f.pub)))
	if err == nil {
		t.Fatal("short source accepted")
	}
}

func TestFulfillRefusesTemplatedHref(t *testing.T) {
	f := newFulfillFixture(t, func(l map[string]any) {
		l["href"] = "https://helix.example.org/pub{?id}"
		l["templated"] = true
	})
	lic, err := Parse(f.license)
	if err != nil {
		t.Fatal(err)
	}
	if l, _ := lic.PublicationLink(); !l.Templated {
		t.Error("Link.Templated not parsed")
	}
	if _, err := f.fulfill(); !errors.Is(err, ErrTemplatedHref) {
		t.Fatalf("err = %v, want ErrTemplatedHref", err)
	}
}

func TestFulfillTemplatedErrorDoesNotEchoHref(t *testing.T) {
	f := newFulfillFixture(t, func(l map[string]any) {
		l["href"] = "https://helix.example.org/pub{?id}&token=SECRETTOKEN"
		l["templated"] = true
	})
	_, err := f.fulfill()
	if !errors.Is(err, ErrTemplatedHref) || strings.Contains(err.Error(), "SECRETTOKEN") {
		t.Fatalf("err = %v", err)
	}
}

func TestUnreadableErrorNeverPrintsNil(t *testing.T) {
	err := unreadableErr(&Container{})
	if err == nil || strings.Contains(err.Error(), "nil") {
		t.Fatalf("message = %v", err)
	}
}
