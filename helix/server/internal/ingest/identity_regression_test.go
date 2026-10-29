package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/helixreader/helix/server/internal/catalog"
	"github.com/helixreader/helix/server/internal/sources"
)

func TestFileIdentityIsCaseSensitive(t *testing.T) {
	if fileID("source", "Library/Book.epub") == fileID("source", "Library/book.epub") {
		t.Fatal("distinct case-sensitive source paths collapsed to one file identity")
	}
	if fileID("source", "Library/Book.epub") != fileID("source", "Library/Book.epub") {
		t.Fatal("identical source/path identity is not deterministic")
	}
}

func fb2WithISBN(body string) []byte {
	return []byte(`<?xml version="1.0" encoding="utf-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
<description><title-info><book-title>Shared ISBN Edition</book-title><isbn>9780441478125</isbn></title-info></description>
<body><section><p>` + body + `</p></section></body></FictionBook>`)
}

func TestScanDifferentBytesWithSameISBNShareWorkAndEdition(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.fb2"), fb2WithISBN("First file"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "b.fb2"), fb2WithISBN("Second file with different bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	connector, err := sources.NewLocal(sources.Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	repo := catalog.NewMemory()
	service := Service{Repository: repo, Now: func() time.Time { return time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC) }}
	report, err := service.ScanSource(context.Background(), "isbn-source", connector, "")
	if err != nil || report.Files != 2 || len(report.Skipped) != 0 {
		t.Fatalf("same-ISBN scan failed: report=%+v err=%v", report, err)
	}
	workID := stableID("work", "isbn", "9780441478125")
	work, err := repo.GetWork(context.Background(), workID)
	if err != nil {
		t.Fatal(err)
	}
	if len(work.Editions) != 1 || len(work.Editions[0].Files) != 2 {
		t.Fatalf("same ISBN did not share one edition with two files: %+v", work)
	}
	if work.Editions[0].Files[0].SHA256 == work.Editions[0].Files[1].SHA256 {
		t.Fatal("distinct content hashes were not retained")
	}
}

func TestMemoryRescanRehomesChangedFileAndRemovesEmptyWork(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "changed.txt")
	oldBytes := []byte("Memory rehome regression: old bytes.\n")
	newBytes := []byte("Memory rehome regression: new bytes with a new hash.\n")
	if err := os.WriteFile(path, oldBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	connector, err := sources.NewLocal(sources.Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	repo := catalog.NewMemory()
	service := Service{Repository: repo}
	if report, err := service.ScanSource(context.Background(), "memory-rehome-source", connector, ""); err != nil || report.Files != 1 {
		t.Fatalf("first scan: report=%+v err=%v", report, err)
	}
	oldHash := sha256.Sum256(oldBytes)
	newHash := sha256.Sum256(newBytes)
	oldWorkID := stableID("work", hex.EncodeToString(oldHash[:]))
	newWorkID := stableID("work", hex.EncodeToString(newHash[:]))
	if err := os.WriteFile(path, newBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	if report, err := service.ScanSource(context.Background(), "memory-rehome-source", connector, ""); err != nil || report.Files != 1 || len(report.Skipped) != 0 {
		t.Fatalf("changed rescan: report=%+v err=%v", report, err)
	}
	if _, err := repo.GetWork(context.Background(), oldWorkID); !errors.Is(err, catalog.ErrNotFound) {
		t.Fatalf("empty old memory work was not removed: %v", err)
	}
	work, err := repo.GetWork(context.Background(), newWorkID)
	if err != nil {
		t.Fatal(err)
	}
	if len(work.Editions) != 1 || len(work.Editions[0].Files) != 1 || work.Editions[0].Files[0].SHA256 != hex.EncodeToString(newHash[:]) {
		t.Fatalf("memory file was not re-homed with current bytes: %+v", work)
	}
}

func verifyFullScanSourceReconciliation(t *testing.T, repo catalog.Repository, sourceID string) {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	content := []byte("Full-source reconciliation for " + sourceID + ".\n")
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	connector, err := sources.NewLocal(sources.Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(content)
	workID := stableID("work", hex.EncodeToString(hash[:]))
	if err := repo.DeleteWork(ctx, workID); err != nil && !errors.Is(err, catalog.ErrNotFound) {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := repo.DeleteWork(context.Background(), workID); err != nil && !errors.Is(err, catalog.ErrNotFound) {
			t.Errorf("cleanup work %s: %v", workID, err)
		}
	})
	service := Service{Repository: repo}
	first, err := service.ScanSource(ctx, sourceID, connector, "")
	if err != nil || first.Files != 2 || first.Removed != 0 || len(first.Skipped) != 0 {
		t.Fatalf("initial scan: report=%+v err=%v", first, err)
	}
	work, err := repo.GetWork(ctx, workID)
	if err != nil || len(work.Editions) != 1 || len(work.Editions[0].Files) != 2 {
		t.Fatalf("initial work shape: %+v err=%v", work, err)
	}
	removedID := ""
	for _, file := range work.Editions[0].Files {
		if file.Path == "a.txt" {
			removedID = file.ID
		}
	}
	if removedID == "" {
		t.Fatal("a.txt file identity missing")
	}
	if err := os.Remove(filepath.Join(root, "a.txt")); err != nil {
		t.Fatal(err)
	}
	second, err := service.ScanSource(ctx, sourceID, connector, "")
	if err != nil || second.Files != 1 || second.Removed != 1 || len(second.Skipped) != 0 {
		t.Fatalf("deletion rescan: report=%+v err=%v", second, err)
	}
	if _, _, _, err := repo.FindFile(ctx, removedID); !errors.Is(err, catalog.ErrNotFound) {
		t.Fatalf("deleted source file remains in catalog: %v", err)
	}
	work, err = repo.GetWork(ctx, workID)
	if err != nil || len(work.Editions) != 1 || len(work.Editions[0].Files) != 1 || work.Editions[0].Files[0].Path != "b.txt" {
		t.Fatalf("remaining shared-edition file was not preserved: %+v err=%v", work, err)
	}
	if err := os.Remove(filepath.Join(root, "b.txt")); err != nil {
		t.Fatal(err)
	}
	third, err := service.ScanSource(ctx, sourceID, connector, "")
	if err != nil || third.Files != 0 || third.Removed != 1 || len(third.Skipped) != 0 {
		t.Fatalf("empty-source rescan: report=%+v err=%v", third, err)
	}
	if _, err := repo.GetWork(ctx, workID); !errors.Is(err, catalog.ErrNotFound) {
		t.Fatalf("empty work remains after final source file deletion: %v", err)
	}
}

func TestMemoryFullScanReconcilesDeletedSourceFiles(t *testing.T) {
	verifyFullScanSourceReconciliation(t, catalog.NewMemory(), "memory-reconcile-source")
}

func TestLivePostgresFullScanReconcilesDeletedSourceFiles(t *testing.T) {
	repo, _ := livePostgresRepo(t)
	verifyFullScanSourceReconciliation(t, repo, "live-reconcile-source")
}

func livePostgresRepo(t *testing.T) (*catalog.Postgres, context.Context) {
	t.Helper()
	dsn := os.Getenv("HELIX_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("HELIX_TEST_POSTGRES_DSN is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	repo, err := catalog.NewPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { repo.Close() })
	return repo, ctx
}

func deleteWorkIfPresent(ctx context.Context, repo *catalog.Postgres, id string) {
	if err := repo.DeleteWork(ctx, id); err != nil && !errors.Is(err, catalog.ErrNotFound) {
		panic(err)
	}
}

func TestLivePostgresRescanRehomesChangedFileAndRemovesEmptyWork(t *testing.T) {
	repo, ctx := livePostgresRepo(t)
	root := t.TempDir()
	path := filepath.Join(root, "changed.txt")
	oldBytes := []byte("Helix PostgreSQL rehome regression: old bytes.\n")
	newBytes := []byte("Helix PostgreSQL rehome regression: new and longer bytes.\n")
	oldHash := sha256.Sum256(oldBytes)
	newHash := sha256.Sum256(newBytes)
	oldWorkID := stableID("work", hex.EncodeToString(oldHash[:]))
	newWorkID := stableID("work", hex.EncodeToString(newHash[:]))
	for _, id := range []string{oldWorkID, newWorkID} {
		deleteWorkIfPresent(ctx, repo, id)
		id := id
		t.Cleanup(func() { deleteWorkIfPresent(context.Background(), repo, id) })
	}
	if err := os.WriteFile(path, oldBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	connector, err := sources.NewLocal(sources.Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	service := Service{Repository: repo}
	first, err := service.ScanSource(ctx, "live-rehome-source", connector, "")
	if err != nil || first.Files != 1 || len(first.Skipped) != 0 {
		t.Fatalf("first scan: report=%+v err=%v", first, err)
	}
	if err := os.WriteFile(path, newBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := service.ScanSource(ctx, "live-rehome-source", connector, "")
	if err != nil || second.Files != 1 || len(second.Skipped) != 0 {
		t.Fatalf("changed rescan: report=%+v err=%v", second, err)
	}
	if _, err := repo.GetWork(ctx, oldWorkID); !errors.Is(err, catalog.ErrNotFound) {
		t.Fatalf("empty old work was not removed: %v", err)
	}
	work, err := repo.GetWork(ctx, newWorkID)
	if err != nil {
		t.Fatal(err)
	}
	if len(work.Editions) != 1 || len(work.Editions[0].Files) != 1 {
		t.Fatalf("re-homed work shape: %+v", work)
	}
	file := work.Editions[0].Files[0]
	if file.SHA256 != hex.EncodeToString(newHash[:]) || file.Size != int64(len(newBytes)) {
		t.Fatalf("stale file bytes metadata: %+v", file)
	}
	_, _, found, err := repo.FindFile(ctx, file.ID)
	if err != nil || found.SHA256 != file.SHA256 {
		t.Fatalf("FindFile did not resolve re-homed identity: file=%+v err=%v", found, err)
	}
}

func verifyLegacyCaseFoldedFileRehome(t *testing.T, repo catalog.Repository) {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	sourceID := "legacy-case-folded-source"
	entryPath := "book.txt"
	oldBytes := []byte("Legacy case-folded identity: old bytes.\n")
	newBytes := []byte("Legacy case-folded identity: current bytes.\n")
	oldHash := sha256.Sum256(oldBytes)
	newHash := sha256.Sum256(newBytes)
	oldWorkID := stableID("work", hex.EncodeToString(oldHash[:]))
	newWorkID := stableID("work", hex.EncodeToString(newHash[:]))
	for _, id := range []string{oldWorkID, newWorkID} {
		if err := repo.DeleteWork(ctx, id); err != nil && !errors.Is(err, catalog.ErrNotFound) {
			t.Fatal(err)
		}
		id := id
		t.Cleanup(func() {
			if err := repo.DeleteWork(context.Background(), id); err != nil && !errors.Is(err, catalog.ErrNotFound) {
				t.Errorf("cleanup work %s: %v", id, err)
			}
		})
	}
	legacyFileID := stableID("file", sourceID, entryPath)
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	oldWork := catalog.Work{ID: oldWorkID, CanonicalTitle: "Legacy", CreatedAt: now, UpdatedAt: now, Editions: []catalog.Edition{{
		ID: stableID("edition", hex.EncodeToString(oldHash[:])), Title: "Legacy", State: catalog.NeedsReview, UpdatedAt: now,
		Files: []catalog.File{{ID: legacyFileID, SourceID: sourceID, Path: entryPath, Format: "txt", Size: int64(len(oldBytes)), SHA256: hex.EncodeToString(oldHash[:]), ModifiedAt: now}},
	}}}
	if err := repo.UpsertWork(ctx, oldWork); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, entryPath), newBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	connector, err := sources.NewLocal(sources.Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	service := Service{Repository: repo}
	report, err := service.ScanSource(ctx, sourceID, connector, "")
	if err != nil || report.Files != 1 || len(report.Skipped) != 0 {
		t.Fatalf("legacy-ID rescan: report=%+v err=%v", report, err)
	}
	if _, err := repo.GetWork(ctx, oldWorkID); !errors.Is(err, catalog.ErrNotFound) {
		t.Fatalf("legacy empty work remains: %v", err)
	}
	work, err := repo.GetWork(ctx, newWorkID)
	if err != nil {
		t.Fatal(err)
	}
	files := work.Editions[0].Files
	if len(files) != 1 || files[0].ID != fileID(sourceID, entryPath) || files[0].ID == legacyFileID || files[0].SHA256 != hex.EncodeToString(newHash[:]) {
		t.Fatalf("legacy source/path was not re-homed to current ID: %+v", files)
	}
}

func TestMemoryRescanRehomesLegacyCaseFoldedFileID(t *testing.T) {
	verifyLegacyCaseFoldedFileRehome(t, catalog.NewMemory())
}

func TestLivePostgresRescanRehomesLegacyCaseFoldedFileID(t *testing.T) {
	repo, _ := livePostgresRepo(t)
	verifyLegacyCaseFoldedFileRehome(t, repo)
}

func TestLivePostgresDifferentBytesWithSameISBNShareEdition(t *testing.T) {
	repo, ctx := livePostgresRepo(t)
	workID := stableID("work", "isbn", "9780441478125")
	deleteWorkIfPresent(ctx, repo, workID)
	t.Cleanup(func() { deleteWorkIfPresent(context.Background(), repo, workID) })
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.fb2"), fb2WithISBN("Live first file"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "b.fb2"), fb2WithISBN("Live second file with different bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	connector, err := sources.NewLocal(sources.Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	service := Service{Repository: repo}
	report, err := service.ScanSource(ctx, "live-isbn-source", connector, "")
	if err != nil || report.Files != 2 || len(report.Skipped) != 0 {
		t.Fatalf("same-ISBN live scan failed: report=%+v err=%v", report, err)
	}
	work, err := repo.GetWork(ctx, workID)
	if err != nil {
		t.Fatal(err)
	}
	if len(work.Editions) != 1 || len(work.Editions[0].Files) != 2 {
		t.Fatalf("live same-ISBN work did not merge: %+v", work)
	}
	hashes := map[string]bool{}
	for _, file := range work.Editions[0].Files {
		if len(file.SHA256) != 64 || strings.TrimSpace(file.Path) == "" {
			t.Fatalf("incomplete file identity: %+v", file)
		}
		hashes[file.SHA256] = true
	}
	if len(hashes) != 2 {
		t.Fatalf("distinct byte streams collapsed: %+v", work.Editions[0].Files)
	}
}
