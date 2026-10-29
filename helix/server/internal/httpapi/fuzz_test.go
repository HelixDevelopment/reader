package httpapi

import "testing"

func FuzzSingleRange(f *testing.F) {
	for _, seed := range []string{"bytes=0-0", "bytes=8-", "bytes=-3", "bytes=1-2,4-5", "items=1-2", "bytes=--"} {
		f.Add(seed, int64(10))
	}
	f.Fuzz(func(t *testing.T, value string, size int64) {
		if len(value) > 4096 || size < 0 || size > 1<<40 {
			t.Skip("bound fuzz input")
		}
		start, length, err := singleRange(value, size)
		if err != nil {
			return
		}
		if size <= 0 || start < 0 || length <= 0 || start >= size || length > size-start {
			t.Fatalf("invalid successful range: value=%q size=%d start=%d length=%d", value, size, start, length)
		}
	})
}
