package httpapi

import "testing"

func BenchmarkSingleRange(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		start, length, err := singleRange("bytes=1024-8191", 1<<20)
		if err != nil || start != 1024 || length != 7168 {
			b.Fatalf("unexpected range: %d %d %v", start, length, err)
		}
	}
}
