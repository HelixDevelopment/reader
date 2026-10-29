package fts

import "testing"

func TestSearchRanking(t *testing.T) {
	ix := NewMemoryIndex()
	ix.Add(DocRef{BookID: "b1", Chapter: "c1", Block: 0, Text: "the wall was there"})
	ix.Add(DocRef{BookID: "b1", Chapter: "c1", Block: 1, Text: "the wall the wall the wall"})
	ix.Add(DocRef{BookID: "b1", Chapter: "c2", Block: 0, Text: "nothing relevant"})
	hits := ix.Search("wall", 10)
	if len(hits) != 2 {
		t.Fatalf("hits: %d", len(hits))
	}
	if hits[0].Block != 1 || hits[0].Score <= hits[1].Score {
		t.Fatalf("higher tf must rank first: %+v", hits)
	}
}

func TestSearchDeterministicTies(t *testing.T) {
	ix := NewMemoryIndex()
	ix.Add(DocRef{BookID: "b", Chapter: "c", Block: 0, Text: "alpha"})
	ix.Add(DocRef{BookID: "b", Chapter: "c", Block: 1, Text: "alpha"})
	h := ix.Search("alpha", 10)
	if len(h) != 2 || h[0].Block != 0 || h[1].Block != 1 {
		t.Fatalf("ties must break by insertion order: %+v", h)
	}
}

func TestCaseAndPunctuation(t *testing.T) {
	ix := NewMemoryIndex()
	ix.Add(DocRef{BookID: "b", Chapter: "c", Block: 0, Text: "Erhenrang, capital!"})
	if len(ix.Search("ERHENRANG", 5)) != 1 {
		t.Fatal("case/punct insensitive match failed")
	}
}

func TestEmptyAndLimits(t *testing.T) {
	ix := NewMemoryIndex()
	ix.Add(DocRef{Text: "x y"})
	if ix.Search("", 10) != nil || ix.Search("x", 0) != nil {
		t.Fatal("empty query/limit must return nil")
	}
	if ix.Size() != 1 {
		t.Fatal("size")
	}
	if len(ix.Search("absent", 10)) != 0 {
		t.Fatal("no-match must be empty")
	}
}
