package catalog

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func hrdRepo(t *testing.T, workID string) (*Postgres, context.Context) {
	t.Helper()
	dsn := os.Getenv("HELIX_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("HELIX_TEST_POSTGRES_DSN is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	t.Cleanup(cancel)
	repo, err := NewPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	_ = repo.DeleteWork(ctx, workID)
	t.Cleanup(func() {
		_ = repo.DeleteWork(context.Background(), workID)
		repo.Close()
	})
	return repo, ctx
}

// HRD finding 1: two incoming files sharing (source_id, path) must collapse to
// one row, the current-ID row winning over the legacy-ID row, whatever order
// they arrive in (the old per-row loop made the last one win).
func TestLivePostgresHRDUpsertDuplicateSourcePathCurrentIDWins(t *testing.T) {
	const wid = "hrd-dup-path-work"
	repo, ctx := hrdRepo(t, wid)
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	curID := currentFileID("hrd-dup-src", "Book.epub")
	mk := func(id string) File {
		return File{ID: id, SourceID: "hrd-dup-src", Path: "Book.epub", Format: "epub", MIMEType: "application/epub+zip", Size: 1, SHA256: strings.Repeat("a", 64), ModifiedAt: now}
	}
	for _, order := range [][]string{{"hrd-legacy-file", curID}, {curID, "hrd-legacy-file"}} {
		w := Work{ID: wid, CanonicalTitle: "Dup", CreatedAt: now, UpdatedAt: now, Editions: []Edition{{
			ID: "hrd-dup-ed", Title: "Dup", State: Pending, UpdatedAt: now,
			Files: []File{mk(order[0]), mk(order[1])},
		}}}
		if err := repo.UpsertWork(ctx, w); err != nil {
			t.Fatalf("order %v: upsert failed: %v", order, err)
		}
		got, err := repo.GetWork(ctx, wid)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Editions) != 1 || len(got.Editions[0].Files) != 1 || got.Editions[0].Files[0].ID != curID {
			t.Fatalf("order %v: want exactly the current-ID file, got %+v", order, got.Editions)
		}
	}
}

// HRD finding 2: duplicate edition ids inside the incoming set are an error,
// not silently swallowed with their children attached to the survivor.
func TestLivePostgresHRDUpsertDuplicateEditionIDFailsLoud(t *testing.T) {
	const wid = "hrd-dup-ed-work"
	repo, ctx := hrdRepo(t, wid)
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	w := Work{ID: wid, CanonicalTitle: "DupEd", CreatedAt: now, UpdatedAt: now, Editions: []Edition{
		{ID: "hrd-dup-ed-1", Title: "A", State: Pending, UpdatedAt: now},
		{ID: "hrd-dup-ed-1", Title: "B", State: Pending, UpdatedAt: now,
			Files: []File{{ID: "hrd-dup-ed-file", SourceID: "hrd-s", Path: "x.epub", Format: "epub", SHA256: strings.Repeat("b", 64), ModifiedAt: now}}},
	}}
	err := repo.UpsertWork(ctx, w)
	if err == nil {
		t.Fatal("duplicate edition id in the incoming set was accepted silently")
	}
	// A bare PK violation would also error; only the guard names the work and
	// the edition, so pin its message.
	want := "catalog: work " + wid + " carries edition hrd-dup-ed-1 more than once"
	if err.Error() != want {
		t.Fatalf("error = %q, want the guard's %q", err.Error(), want)
	}
	if _, err := repo.GetWork(ctx, wid); err == nil {
		t.Fatal("failed upsert left the work behind (transaction not rolled back)")
	}
}

// HRD finding 3: a work larger than the 65535 bind-parameter limit of a single
// statement must still upsert. Unchunked, the editions insert (13/row) and the
// files insert (12/row) would exceed 65535 here; claimFiles (2/row + 1) peaks
// at 11201 for this size and is not the limiting statement.
func TestLivePostgresHRDUpsertBeyondBindParameterLimit(t *testing.T) {
	const wid = "hrd-big-work"
	repo, ctx := hrdRepo(t, wid)
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	const nEd = 5200
	// 5200 editions x 13 = 67600 params; plus 5600 files x 12 = 67200 params.
	const nFiles = 5600
	eds := make([]Edition, nEd)
	for i := range eds {
		eds[i] = Edition{ID: fmt.Sprintf("hrd-big-ed-%05d", i), Title: "E", State: Pending, UpdatedAt: now}
	}
	for i := 0; i < nFiles; i++ {
		e := &eds[i%nEd]
		e.Files = append(e.Files, File{ID: fmt.Sprintf("hrd-big-f-%05d", i), SourceID: "hrd-big-src", Path: fmt.Sprintf("p/%05d.epub", i), Format: "epub", SHA256: strings.Repeat("c", 64), ModifiedAt: now})
	}
	w := Work{ID: wid, CanonicalTitle: "Big", CreatedAt: now, UpdatedAt: now, Editions: eds}
	if err := repo.UpsertWork(ctx, w); err != nil {
		t.Fatalf("large upsert failed: %v", err)
	}
	if n := countCatalogRows(ctx, t, repo, "SELECT count(*) FROM editions WHERE work_id=$1", wid); n != nEd {
		t.Fatalf("editions stored = %d, want %d", n, nEd)
	}
	if n := countCatalogRows(ctx, t, repo, "SELECT count(*) FROM files WHERE source_id='hrd-big-src'"); n != nFiles {
		t.Fatalf("files stored = %d, want %d", n, nFiles)
	}
	// Replay is idempotent at this size too.
	if err := repo.UpsertWork(ctx, w); err != nil {
		t.Fatalf("large repeat upsert failed: %v", err)
	}
}

// Pins currentFileID to ingest.fileID; the 17 vectors are copied verbatim from
// ingest.TestFileIDGoldenVectors (catalog cannot import ingest).
func TestHRDCurrentFileIDGoldenVector(t *testing.T) {
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
		if got := currentFileID(v[0], v[1]); got != v[2] {
			t.Errorf("currentFileID(%q,%q)=%s want %s", v[0], v[1], got, v[2])
		}
	}
}

// DB-free: the guard itself, independent of any database constraint.
func TestRejectDuplicateEditionIDsUnit(t *testing.T) {
	ok := Work{ID: "w", Editions: []Edition{{ID: "a"}, {ID: "b"}}}
	if err := rejectDuplicateEditionIDs(ok); err != nil {
		t.Fatalf("distinct ids rejected: %v", err)
	}
	if err := rejectDuplicateEditionIDs(Work{ID: "w"}); err != nil {
		t.Fatalf("empty work rejected: %v", err)
	}
	dup := Work{ID: "w", Editions: []Edition{{ID: "a"}, {ID: "b"}, {ID: "a"}}}
	err := rejectDuplicateEditionIDs(dup)
	if err == nil || err.Error() != "catalog: work w carries edition a more than once" {
		t.Fatalf("got %v", err)
	}
}

// Claiming files of ANOTHER work across a bindChunkRows boundary: every file is
// re-homed, and the other work's editions that end up empty are pruned, while
// an edition that keeps an unclaimed file survives.
func TestLivePostgresHRDClaimFilesAcrossChunkBoundary(t *testing.T) {
	const victim, thief = "hrd-chunk-victim", "hrd-chunk-thief"
	repo, ctx := hrdRepo(t, thief)
	_ = repo.DeleteWork(ctx, victim)
	t.Cleanup(func() { _ = repo.DeleteWork(context.Background(), victim) })
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	const n = 2500 // chunks: 0-999, 1000-1999, 2000-2499
	const src = "hrd-chunk-src"
	mk := func(i int) File {
		return File{ID: fmt.Sprintf("hrd-chunk-f-%05d", i), SourceID: src, Path: fmt.Sprintf("p/%05d.epub", i), Format: "epub", SHA256: strings.Repeat("d", 64), ModifiedAt: now}
	}
	// Victim: edition A holds files 0..999, B holds 1000..1999 (both fully
	// claimed, in different chunks), C holds 2000..2499 plus one extra file
	// the thief does not claim, so C must survive.
	va := Edition{ID: "hrd-chunk-ed-a", Title: "A", State: Pending, UpdatedAt: now}
	vb := Edition{ID: "hrd-chunk-ed-b", Title: "B", State: Pending, UpdatedAt: now}
	vc := Edition{ID: "hrd-chunk-ed-c", Title: "C", State: Pending, UpdatedAt: now}
	for i := 0; i < n; i++ {
		switch {
		case i < 1000:
			va.Files = append(va.Files, mk(i))
		case i < 2000:
			vb.Files = append(vb.Files, mk(i))
		default:
			vc.Files = append(vc.Files, mk(i))
		}
	}
	extra := File{ID: "hrd-chunk-f-extra", SourceID: src, Path: "p/extra.epub", Format: "epub", SHA256: strings.Repeat("e", 64), ModifiedAt: now}
	vc.Files = append(vc.Files, extra)
	vw := Work{ID: victim, CanonicalTitle: "Victim", CreatedAt: now, UpdatedAt: now, Editions: []Edition{va, vb, vc}}
	if err := repo.UpsertWork(ctx, vw); err != nil {
		t.Fatalf("victim upsert: %v", err)
	}
	// Thief claims all n files (same ids and paths) in one edition, spanning
	// all three chunks of claimFiles/insertFiles.
	te := Edition{ID: "hrd-chunk-ed-thief", Title: "T", State: Pending, UpdatedAt: now}
	for i := 0; i < n; i++ {
		te.Files = append(te.Files, mk(i))
	}
	tw := Work{ID: thief, CanonicalTitle: "Thief", CreatedAt: now, UpdatedAt: now, Editions: []Edition{te}}
	if err := repo.UpsertWork(ctx, tw); err != nil {
		t.Fatalf("thief upsert: %v", err)
	}
	if got := countCatalogRows(ctx, t, repo, "SELECT count(*) FROM files WHERE edition_id=$1", "hrd-chunk-ed-thief"); got != n {
		t.Fatalf("thief owns %d files, want %d", got, n)
	}
	for _, id := range []string{"hrd-chunk-ed-a", "hrd-chunk-ed-b"} {
		if got := countCatalogRows(ctx, t, repo, "SELECT count(*) FROM editions WHERE id=$1", id); got != 0 {
			t.Fatalf("emptied edition %s not pruned", id)
		}
	}
	if got := countCatalogRows(ctx, t, repo, "SELECT count(*) FROM files WHERE edition_id=$1", "hrd-chunk-ed-c"); got != 1 {
		t.Fatalf("surviving edition C has %d files, want exactly the 1 unclaimed", got)
	}
	if got := countCatalogRows(ctx, t, repo, "SELECT count(*) FROM works WHERE id=$1", victim); got != 1 {
		t.Fatalf("victim work (still owns C) was pruned")
	}
	if got := countCatalogRows(ctx, t, repo, "SELECT count(*) FROM files WHERE source_id=$1", src); got != n+1 {
		t.Fatalf("total files = %d, want %d (no duplicates, none lost)", got, n+1)
	}
}
