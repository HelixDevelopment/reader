package ingest

import "testing"

// Drift guard for catalog.currentFileID, a copy of fileID (catalog cannot
// import ingest). The first two vectors are byte-for-byte the ones pinned in
// catalog.TestHRDCurrentFileIDGoldenVector; the rest were computed with an
// independent implementation (Python hashlib) and pin case-sensitivity,
// length-prefix unambiguity and unicode handling. If fileID changes, this
// test fails; update catalog.currentFileID and its vectors in the same change.
func TestFileIDGoldenVectors(t *testing.T) {
	for _, v := range [][3]string{
		{"src-a", "Dir/Book.EPUB", "file-18da8b2a51747e0aef982d29"},
		{"hrd-dup-src", "Book.epub", "file-1f18d6411b488763cf60cfa6"},
		{"src-a", "dir/book.epub", "file-0073aa62a6094ebc31931505"},
		{"SRC-A", "Dir/Book.EPUB", "file-0b1dd5b05ba85f718c5b61b7"},
		{"", "", "file-f894ad68de109cc7431f44b5"},
		{"a", "bc", "file-3b69afb707a403127c82e649"},
		{"ab", "c", "file-537e7baea2e22444c6b19727"},
		{"src", "Ünïcode/Книга.epub", "file-730d90cb590752d87a9302c7"},
		{"src", "ünïcode/книга.epub", "file-8240a1c3c5d8ea90db8f5899"},
		{"src", "日本語/本.epub", "file-fa910c702400336c0d0c0e5b"},
		{"src", "İstanbul/ISBN.epub", "file-f2900e4f909ad961e2c49649"},
		{"src", "istanbul/isbn.epub", "file-8ed7cd9faf8479b835857918"},
		{"src", "with space/a b.epub", "file-e9b9118e1ce04594db744387"},
		{"src", "a\\b.epub", "file-8a2212c2aca060d4911353d9"},
		{"src", "é.epub", "file-da16039381750fc8f05a7ca5"},
		{"src", "é.epub", "file-7dd025d5f205a797cfd5c219"},
		{"s", "/abs/Path/../x.EPUB", "file-42aac06125403b9398f05b1e"},
	} {
		if got := fileID(v[0], v[1]); got != v[2] {
			t.Errorf("fileID(%q,%q)=%s want %s", v[0], v[1], got, v[2])
		}
	}
}
