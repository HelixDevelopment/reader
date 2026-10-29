package aimemory

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestLivePostgresVectorMemory(t *testing.T) {
	dsn := os.Getenv("HELIX_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("HELIX_TEST_POSTGRES_DSN is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	memory, err := NewPostgres(ctx, dsn, 3)
	if err != nil {
		t.Fatal(err)
	}
	defer memory.Close()
	defer memory.Delete(context.Background(), "live-scope", "live-near")
	defer memory.Delete(context.Background(), "live-scope", "live-far")
	defer memory.Delete(context.Background(), "other-scope", "live-other")
	_ = memory.Delete(ctx, "live-scope", "live-near")
	_ = memory.Delete(ctx, "live-scope", "live-far")
	_ = memory.Delete(ctx, "other-scope", "live-other")

	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	put := func(record Record, vector []float32) {
		t.Helper()
		record.Kind = "note"
		record.Model = "live-model"
		record.CreatedAt = now
		if _, err := memory.Put(ctx, record, vector); err != nil {
			t.Fatal(err)
		}
	}
	put(Record{ID: "live-near", Scope: "live-scope", Content: "near"}, []float32{1, 0, 0})
	put(Record{ID: "live-far", Scope: "live-scope", Content: "far"}, []float32{0, 1, 0})
	put(Record{ID: "live-other", Scope: "other-scope", Content: "other"}, []float32{1, 0, 0})
	if _, err := memory.Put(ctx, Record{ID: "bad", Scope: "live-scope", Kind: "note", Content: "bad", Model: "live-model"}, []float32{1}); err == nil {
		t.Fatal("dimension mismatch accepted")
	}
	results, err := memory.Search(ctx, "live-scope", []float32{0.9, 0.1, 0}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || results[0].Record.ID != "live-near" || results[0].Similarity <= results[1].Similarity {
		t.Fatalf("cosine ordering or scope isolation failed: %+v", results)
	}
	// Re-putting the same ID must advance content, content_sha256 and the
	// embedding together; stale content paired with a fresh vector is a
	// provenance violation.
	updated, err := memory.Put(ctx, Record{ID: "live-near", Scope: "live-scope", Kind: "note", Model: "live-model", Content: "near v2", CreatedAt: now}, []float32{0, 0, 1})
	if err != nil {
		t.Fatal(err)
	}
	moved, err := memory.Search(ctx, "live-scope", []float32{0, 0, 1}, 1)
	if err != nil || len(moved) != 1 {
		t.Fatalf("search after re-put failed: %+v %v", moved, err)
	}
	if moved[0].Record.ID != "live-near" || moved[0].Record.Content != "near v2" {
		t.Fatalf("upsert did not update content with embedding: %+v", moved[0].Record)
	}
	if moved[0].Record.ContentSHA256 != updated.ContentSHA256 {
		t.Fatalf("content_sha256 diverged from content: %+v vs %+v", moved[0].Record, updated)
	}
	if moved[0].Similarity < 0.99 {
		t.Fatalf("embedding was not updated on conflict: %+v", moved[0])
	}
	// Restore live-near so the reconnect assertions below stay valid.
	put(Record{ID: "live-near", Scope: "live-scope", Content: "near"}, []float32{1, 0, 0})
	if err := memory.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := NewPostgres(ctx, dsn, 3)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	defer reopened.Delete(context.Background(), "live-scope", "live-near")
	defer reopened.Delete(context.Background(), "live-scope", "live-far")
	defer reopened.Delete(context.Background(), "other-scope", "live-other")
	persisted, err := reopened.Search(ctx, "live-scope", []float32{1, 0, 0}, 1)
	if err != nil || len(persisted) != 1 || persisted[0].Record.ID != "live-near" {
		t.Fatalf("vector memory did not survive reconnect: %+v %v", persisted, err)
	}
	mismatched, err := NewPostgres(ctx, dsn, 4)
	if err == nil {
		mismatched.Close()
		t.Fatal("changed embedding dimensions must fail against the migrated schema")
	}
}
