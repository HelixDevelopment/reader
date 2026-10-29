package mobi

import (
	"bytes"
	"testing"
)

func BenchmarkPalmDOCLiterals(b *testing.B) {
	encoded := bytes.Repeat([]byte("Helix Reader PalmDOC benchmark text. "), 2048)
	b.SetBytes(int64(len(encoded)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		decoded, err := decompressPalmDOC(encoded, maxEntryBytesForBenchmark)
		if err != nil {
			b.Fatal(err)
		}
		if len(decoded) != len(encoded) {
			b.Fatal("decoded length mismatch")
		}
	}
}

const maxEntryBytesForBenchmark = 16 << 20
