package catalog

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLivePostgresCatalogRoundTrip(t *testing.T) {
	dsn := os.Getenv("HELIX_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("HELIX_TEST_POSTGRES_DSN is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	repo, err := NewPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	_ = repo.DeleteWork(ctx, "live-work")
	repo.Close()

	repo, err = NewPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	fileA := File{ID: "live-file-a", SourceID: "live-source", Path: "book.epub", Format: "epub", MIMEType: "application/epub+zip", Size: 123, SHA256: strings.Repeat("a", 64), ModifiedAt: now}
	fileB := File{ID: "live-file-b", SourceID: "live-source", Path: "book.pdf", Format: "pdf", MIMEType: "application/pdf", Size: 456, SHA256: strings.Repeat("b", 64), ModifiedAt: now}
	evidence := MetadataEvidence{Field: "title", Provider: "embedded", Confidence: 0.9, ObservedAt: now}
	edition := Edition{ID: "live-edition", Title: "Live Book", Authors: []string{"Author"}, State: Identified, UpdatedAt: now, Identifiers: []Identifier{{Scheme: "isbn", Value: "9780441478125", Source: "embedded", Confidence: 1}}, Files: []File{fileA}, Evidence: []MetadataEvidence{evidence}}
	work := Work{ID: "live-work", CanonicalTitle: "Live Book", CreatedAt: now, UpdatedAt: now, Editions: []Edition{edition}}
	// Exact duplicate evidence rows must be tolerated idempotently.
	dup := work
	dup.Editions = append([]Edition(nil), work.Editions...)
	dup.Editions[0].Evidence = []MetadataEvidence{evidence, evidence}
	if err := repo.MergeWork(ctx, dup); err != nil {
		repo.Close()
		t.Fatalf("duplicate evidence rejected: %v", err)
	}
	if err := repo.MergeWork(ctx, work); err != nil {
		repo.Close()
		t.Fatal(err)
	}
	second := work
	second.Editions = append([]Edition(nil), work.Editions...)
	second.Editions[0].Files = []File{fileB}
	second.UpdatedAt = now.Add(time.Hour)
	second.Editions[0].UpdatedAt = now.Add(time.Hour)
	if err := repo.MergeWork(ctx, second); err != nil {
		repo.Close()
		t.Fatal(err)
	}
	// Repeating the same merge must not accumulate evidence or files.
	if err := repo.MergeWork(ctx, second); err != nil {
		repo.Close()
		t.Fatal(err)
	}
	if err := repo.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := NewPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	defer reopened.DeleteWork(context.Background(), "live-work")
	got, err := reopened.GetWork(ctx, "live-work")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Editions) != 1 || len(got.Editions[0].Files) != 2 {
		t.Fatalf("edition/file identity did not survive reconnect: %+v", got)
	}
	if len(got.Editions[0].Identifiers) != 1 || got.Editions[0].Identifiers[0].Value != "9780441478125" {
		t.Fatalf("edition identifier lost: %+v", got.Editions[0].Identifiers)
	}
	page, err := reopened.Search(ctx, Query{Text: "Live Book", Limit: 10})
	if err != nil || page.Total != 1 {
		t.Fatalf("durable search failed: %+v %v", page, err)
	}
}

// TestLivePostgresUpsertReplacesEditionSet pins the set-based edition write on
// the UpsertWork path: the work's whole old edition set is deleted by one
// statement and the new set reinserted by one multi-row statement, so the
// stored set is exactly the incoming set — stale editions, files, identifiers
// and evidence of the replaced set are gone — and a repeat upsert is
// idempotent. (TestLivePostgresCatalogRoundTrip covers the same property on
// the MergeWork path; this test pins the UpsertWork path so a regression of
// the set-based write is caught.)
func TestLivePostgresUpsertReplacesEditionSet(t *testing.T) {
	dsn := os.Getenv("HELIX_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("HELIX_TEST_POSTGRES_DSN is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	repo, err := NewPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	_ = repo.DeleteWork(ctx, "set-live-work")
	defer repo.DeleteWork(context.Background(), "set-live-work")

	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	file := func(id, name, sha string) File {
		return File{ID: id, SourceID: "set-live-src", Path: name, Format: "epub", MIMEType: "application/epub+zip", Size: 10, SHA256: strings.Repeat(sha, 64), ModifiedAt: now}
	}
	first := Work{
		ID: "set-live-work", CanonicalTitle: "Set Live", CreatedAt: now, UpdatedAt: now,
		Editions: []Edition{
			{
				ID: "set-live-ed-1", Title: "Edition One", Authors: []string{"A"}, Publisher: "Old Press",
				State: Identified, Confidence: 1, UpdatedAt: now,
				Identifiers: []Identifier{{Scheme: "isbn", Value: "9780441478125", Source: "s", Confidence: 1}},
				Files:       []File{file("set-live-file-1", "one.epub", "a")},
				Evidence:    []MetadataEvidence{{Field: "title", Provider: "p", Confidence: 1, ObservedAt: now}},
			},
			{
				ID: "set-live-ed-2", Title: "Edition Two", State: Pending, UpdatedAt: now,
				Files: []File{file("set-live-file-2", "two.epub", "b")},
			},
		},
	}
	if err := repo.UpsertWork(ctx, first); err != nil {
		t.Fatalf("seed: %v", err)
	}
	// The second upsert carries a different edition set; the old set must be
	// replaced, not merged: upsertTx deletes the work's whole edition set.
	second := Work{
		ID: "set-live-work", CanonicalTitle: "Set Live Revised", CreatedAt: now, UpdatedAt: now.Add(time.Hour),
		Editions: []Edition{{
			ID: "set-live-ed-3", Title: "Edition Three", State: Identified, UpdatedAt: now.Add(time.Hour),
			Identifiers: []Identifier{{Scheme: "isbn", Value: "9780441478133", Source: "s", Confidence: 1}},
			Files:       []File{file("set-live-file-3", "three.epub", "c")},
			Evidence:    []MetadataEvidence{{Field: "title", Provider: "p", Confidence: 1, ObservedAt: now}},
		}},
	}
	if err := repo.UpsertWork(ctx, second); err != nil {
		t.Fatalf("replace: %v", err)
	}
	got, err := repo.GetWork(ctx, "set-live-work")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Editions) != 1 || got.Editions[0].ID != "set-live-ed-3" {
		t.Fatalf("edition set was not replaced: %+v", got.Editions)
	}
	e := got.Editions[0]
	if got.CanonicalTitle != "Set Live Revised" || e.Title != "Edition Three" || e.Publisher != "" {
		t.Fatalf("work columns did not follow the new set: %+v", got)
	}
	if len(e.Files) != 1 || e.Files[0].ID != "set-live-file-3" || e.Files[0].SHA256 != strings.Repeat("c", 64) {
		t.Fatalf("files did not follow the new set: %+v", e.Files)
	}
	if len(e.Identifiers) != 1 || e.Identifiers[0].Value != "9780441478133" {
		t.Fatalf("identifiers did not follow the new set: %+v", e.Identifiers)
	}
	if len(e.Evidence) != 1 || e.Evidence[0].Field != "title" {
		t.Fatalf("evidence did not follow the new set: %+v", e.Evidence)
	}
	for _, gone := range []string{"set-live-ed-1", "set-live-ed-2"} {
		if n := countCatalogRows(ctx, t, repo, "SELECT count(*) FROM editions WHERE id=$1", gone); n != 0 {
			t.Errorf("replaced edition %s still has %d row(s)", gone, n)
		}
	}
	for _, gone := range []string{"set-live-file-1", "set-live-file-2"} {
		if n := countCatalogRows(ctx, t, repo, "SELECT count(*) FROM files WHERE id=$1", gone); n != 0 {
			t.Errorf("replaced file %s still has %d row(s)", gone, n)
		}
	}
	if n := countCatalogRows(ctx, t, repo, "SELECT count(*) FROM files WHERE source_id='set-live-src'"); n != 1 {
		t.Errorf("source set-live-src has %d file rows, want 1 (no stale rows)", n)
	}
	if n := countCatalogRows(ctx, t, repo, "SELECT count(*) FROM files WHERE edition_id IN (SELECT id FROM editions WHERE work_id=$1)", "set-live-work"); n != 1 {
		t.Fatalf("work has %d file rows, want 1 (only the new edition's file)", n)
	}
	// Repeating the same upsert must be idempotent.
	if err := repo.UpsertWork(ctx, second); err != nil {
		t.Fatalf("repeat upsert: %v", err)
	}
	if got, err = repo.GetWork(ctx, "set-live-work"); err != nil || len(got.Editions) != 1 {
		t.Fatalf("repeat upsert changed the stored set: %+v %v", got, err)
	}
}
