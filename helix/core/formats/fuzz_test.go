package formats

import "testing"

func FuzzDetect(f *testing.F) {
	for _, seed := range []string{"book.epub", "book.lcp.epub", "NCC.HTML", "../../book.fb2.zip", "", "\x00.epub"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, name string) {
		if len(name) > 4096 {
			t.Skip("bound path input")
		}
		d, ok := Detect(name)
		if !ok {
			return
		}
		byFormat, found := ByFormat(d.Format)
		if !found || byFormat.Format != d.Format {
			t.Fatalf("detected descriptor is not registered: %+v", d)
		}
	})
}
