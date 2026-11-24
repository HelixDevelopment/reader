package catalog

// Regression test for the harness-design question raised while building the
// HRD-027 ingest-concurrent-with-reads run: when a source listing omits a file
// of a multi-file edition, ReconcileSource must delete exactly the omitted
// file row, keep the listed one, keep the edition and work alive, and report
// the removed count. (Concurrent runs showed works vanishing mid-run; the cause
// was the harness re-merging the SAME (source, path) stable file onto each new
// work so the previous work legitimately lost its last file and was pruned —
// not a ReconcileSource defect. This test pins the single-threaded contract so
// that distinction stays provable.)
//
// Skips cleanly unless HELIX_TEST_POSTGRES_DSN is set.

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestLiveReconcileSourceKeepsListedFileAndEdition(t *testing.T) {
	dsn := os.Getenv("HELIX_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("HELIX_TEST_POSTGRES_DSN is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	repo, err := NewPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { repo.Close() })

	runTag := "reconcilelive" + time.Now().UTC().Format("20060102T150405.000000000")
	source := runTag + "-src"
	now := time.Now().UTC().Truncate(time.Microsecond)
	w := Work{
		ID: runTag + "-w", CanonicalTitle: "Reconcile contract", CreatedAt: now, UpdatedAt: now,
		Editions: []Edition{{
			ID: runTag + "-ed", Title: "Reconcile contract", Authors: []string{"x"}, State: Identified, Confidence: 1, UpdatedAt: now,
			Files: []File{
				{ID: runTag + "-stable", SourceID: source, Path: "load/book.txt", Format: "txt", MIMEType: "text/plain", Size: 64, SHA256: fmt.Sprintf("%064x", 1), ModifiedAt: now},
				{ID: runTag + "-q", SourceID: source, Path: "load/q-1.txt", Format: "txt", MIMEType: "text/plain", Size: 64, SHA256: fmt.Sprintf("%064x", 2), ModifiedAt: now},
			},
		}},
	}
	if err := repo.MergeWork(ctx, w); err != nil {
		t.Fatalf("merge: %v", err)
	}
	t.Cleanup(func() {
		cctx, c := context.WithTimeout(context.Background(), 30*time.Second)
		defer c()
		_ = repo.DeleteWork(cctx, w.ID)
	})

	before, err := repo.GetWork(ctx, w.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Editions) != 1 || len(before.Editions[0].Files) != 2 {
		t.Fatalf("precondition: want 1 edition with 2 files, got %+v", before.Editions)
	}

	removed, err := repo.ReconcileSource(ctx, source, []string{"load/book.txt"})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if removed != 1 {
		t.Fatalf("reconcile removed %d file rows, want exactly 1 (the omitted q-1.txt)", removed)
	}

	after, err := repo.GetWork(ctx, w.ID)
	if err != nil {
		t.Fatalf("work must survive reconcile while a listed file remains: %v", err)
	}
	if len(after.Editions) != 1 {
		t.Fatalf("edition must survive, got %d editions", len(after.Editions))
	}
	files := after.Editions[0].Files
	if len(files) != 1 || files[0].Path != "load/book.txt" || files[0].ID != runTag+"-stable" {
		t.Fatalf("want only the listed file load/book.txt to remain, got %+v", files)
	}

	// Listing nothing prunes the rest and, with it, the emptied edition and work.
	removed, err = repo.ReconcileSource(ctx, source, nil)
	if err != nil || removed != 1 {
		t.Fatalf("empty listing: removed=%d err=%v, want 1,nil", removed, err)
	}
	if _, err := repo.GetWork(ctx, w.ID); err == nil {
		t.Fatal("work with no remaining files must be pruned by an empty listing")
	}
}
