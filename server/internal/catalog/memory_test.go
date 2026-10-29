package catalog

import (
	"context"
	"testing"
	"time"
)

func TestMemorySearchDeterministicAndDeepCopy(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, id := range []string{"b", "a", "c"} {
		w := Work{ID: id, CanonicalTitle: "Title " + id, UpdatedAt: now, Editions: []Edition{{ID: "e" + id, Title: "Title " + id, Authors: []string{"An Author"}, State: Identified, UpdatedAt: now}}}
		if err := m.UpsertWork(ctx, w); err != nil {
			t.Fatal(err)
		}
	}
	p, err := m.Search(ctx, Query{Text: "title", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if p.Total != 3 || len(p.Works) != 2 || p.Works[0].ID != "a" || p.Works[1].ID != "b" {
		t.Fatalf("unstable page: %+v", p)
	}
	p.Works[0].Editions[0].Authors[0] = "MUTATED"
	w, _ := m.GetWork(ctx, "a")
	if w.Editions[0].Authors[0] != "An Author" {
		t.Fatal("repository boundary leaked mutable slices")
	}
	next, _ := m.Search(ctx, Query{Text: "title", Offset: 2, Limit: 2})
	if len(next.Works) != 1 || next.Works[0].ID != "c" {
		t.Fatalf("bad second page: %+v", next)
	}
}

func TestMemoryRepeatMergeIsIdempotent(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	work := Work{ID: "w", CanonicalTitle: "Title", UpdatedAt: now, Editions: []Edition{{
		ID: "e", Title: "Title", State: Identified, UpdatedAt: now,
		Identifiers: []Identifier{{Scheme: "isbn", Value: "9780441478125", Source: "embedded", Confidence: 0.95}},
		Files:       []File{{ID: "f1", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}},
		Evidence:    []MetadataEvidence{{Field: "title", Provider: "embedded", Confidence: 0.9, ObservedAt: now}},
	}}}
	for i := 0; i < 3; i++ {
		if err := m.MergeWork(ctx, work); err != nil {
			t.Fatal(err)
		}
	}
	got, err := m.GetWork(ctx, "w")
	if err != nil {
		t.Fatal(err)
	}
	e := got.Editions[0]
	if len(got.Editions) != 1 || len(e.Files) != 1 || len(e.Identifiers) != 1 || len(e.Evidence) != 1 {
		t.Fatalf("repeat merge grew stored state: %+v", got)
	}
}

func TestFindEditionAndFindFileReturnDeepClones(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	work := Work{ID: "w", CanonicalTitle: "Title", UpdatedAt: now, Editions: []Edition{{
		ID: "e", Title: "Title", State: Identified, UpdatedAt: now,
		Authors:     []string{"An Author"},
		Identifiers: []Identifier{{Scheme: "isbn", Value: "9780441478125", Source: "embedded", Confidence: 0.95}},
		Files:       []File{{ID: "f1", Path: "books/title.epub", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}},
		Evidence:    []MetadataEvidence{{Field: "title", Provider: "embedded", Confidence: 0.9, ObservedAt: now}},
	}}}
	if err := m.UpsertWork(ctx, work); err != nil {
		t.Fatal(err)
	}

	_, edition, err := m.FindEdition(ctx, "e")
	if err != nil {
		t.Fatal(err)
	}
	// Mutate every mutable slice reachable from the returned edition.
	edition.Authors[0] = "MUTATED"
	edition.Identifiers[0].Value = "MUTATED"
	edition.Files[0].Path = "MUTATED"
	edition.Evidence[0].Field = "MUTATED"

	stored, err := m.GetWork(ctx, "w")
	if err != nil {
		t.Fatal(err)
	}
	se := stored.Editions[0]
	if se.Authors[0] != "An Author" || se.Identifiers[0].Value != "9780441478125" ||
		se.Files[0].Path != "books/title.epub" || se.Evidence[0].Field != "title" {
		t.Fatalf("FindEdition leaked stored slices: %+v", se)
	}

	_, fedition, file, err := m.FindFile(ctx, "f1")
	if err != nil {
		t.Fatal(err)
	}
	fedition.Authors[0] = "MUTATED"
	fedition.Identifiers[0].Value = "MUTATED"
	fedition.Files[0].Path = "MUTATED"
	fedition.Evidence[0].Field = "MUTATED"
	file.Path = "MUTATED"
	file.SHA256 = "MUTATED"

	stored, err = m.GetWork(ctx, "w")
	if err != nil {
		t.Fatal(err)
	}
	se = stored.Editions[0]
	if se.Authors[0] != "An Author" || se.Identifiers[0].Value != "9780441478125" ||
		se.Files[0].Path != "books/title.epub" || se.Files[0].SHA256 != work.Editions[0].Files[0].SHA256 ||
		se.Evidence[0].Field != "title" {
		t.Fatalf("FindFile leaked stored state: %+v", se)
	}
}

func TestMemoryRejectsInvalidIdentificationState(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	bad := Work{ID: "w", CanonicalTitle: "T", Editions: []Edition{{ID: "e", Title: "T", State: "invented"}}}
	if err := m.UpsertWork(ctx, bad); err == nil {
		t.Fatal("upsert accepted state outside the closed identification state machine")
	}
	if err := m.MergeWork(ctx, bad); err == nil {
		t.Fatal("merge accepted state outside the closed identification state machine")
	}
}

func TestMemoryValidationAndDelete(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	if err := m.UpsertWork(ctx, Work{ID: "empty"}); err == nil {
		t.Fatal("work without edition accepted")
	}
	if _, err := m.GetWork(ctx, "missing"); err != ErrNotFound {
		t.Fatalf("wrong not-found error: %v", err)
	}
	if err := m.DeleteWork(ctx, "missing"); err != ErrNotFound {
		t.Fatalf("wrong delete error: %v", err)
	}
}

func TestMemoryRejectsIncompleteFileIdentityLikePostgres(t *testing.T) {
	repo := NewMemory()
	work := Work{ID: "w", CanonicalTitle: "Title", Editions: []Edition{{
		ID: "e", Title: "Title", State: NeedsReview,
		Files: []File{{Path: "a.epub", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}},
	}}}
	if err := repo.MergeWork(context.Background(), work); err == nil {
		t.Fatal("memory repository accepted an empty file ID")
	}
	work.Editions[0].Files[0].ID = "f"
	work.Editions[0].Files[0].SHA256 = "short"
	if err := repo.UpsertWork(context.Background(), work); err == nil {
		t.Fatal("memory repository accepted a malformed content hash")
	}
}
