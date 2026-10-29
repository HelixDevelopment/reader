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
