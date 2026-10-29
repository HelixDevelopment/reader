package sync

import "testing"

func BenchmarkResolveProgress(b *testing.B) {
	left, right := New(), New()
	left.Increment("device-a")
	right.Increment("device-b")
	local := ProgressRecord{BookID: "book", DeviceID: "device-a", Pos: Position{Chapter: 7, Block: 2}, Clock: left}
	remote := ProgressRecord{BookID: "book", DeviceID: "device-b", Pos: Position{Chapter: 8, Block: 1}, Clock: right}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := ResolveProgress(local, remote); err != nil {
			b.Fatal(err)
		}
	}
}
