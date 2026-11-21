package catalog

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

// TestLivePostgresBatchLoadMatchesRowByRow pins the ATM-027 read-path rewrite:
// the constant-round-trip loader must describe exactly what the per-work loader
// describes, including ordering and empty-versus-nil child semantics.
//
// Children are deliberately seeded in an order that differs from every canonical
// ordering the loaders apply. Inserting them already sorted let a missing ORDER BY
// pass by coincidence, which is how the first version of this test was proven weak.
func TestLivePostgresBatchLoadMatchesRowByRow(t *testing.T) {
	dsn := os.Getenv("HELIX_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("HELIX_TEST_POSTGRES_DSN is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	repo, err := NewPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	base := time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC)
	file := func(id, name string, minute int) File {
		return File{ID: id, SourceID: "batch-source", Path: name, Format: "epub", MIMEType: "application/epub+zip", Size: int64(len(name) * 100), SHA256: strings.Repeat("c", 64), ModifiedAt: base.Add(time.Duration(minute) * time.Minute)}
	}
	works := []Work{
		{
			ID: "batch-work-a", CanonicalTitle: "Batch Alpha", Series: "Batch", SeriesIndex: 1,
			Subjects: []string{"fiction", "test"}, CreatedAt: base, UpdatedAt: base.Add(3 * time.Hour),
			// Seeded a2 first; canonical edition order is by id.
			Editions: []Edition{
				{
					ID: "batch-edition-a2", Title: "Alpha Two", Authors: []string{"A Author"}, State: Pending,
					UpdatedAt: base.Add(2 * time.Hour),
					Files:     []File{file("batch-file-a3", "alpha/three.epub", 3)},
				},
				{
					ID: "batch-edition-a1", Title: "Alpha One", Authors: []string{"A Author", "B Author"},
					Publisher: "Helix Press", Language: "en", State: Identified, Confidence: 0.9,
					UpdatedAt: base.Add(3 * time.Hour),
					// Canonical identifier order is scheme then value: doi, isbn.
					Identifiers: []Identifier{
						{Scheme: "isbn", Value: "9780000000017", Source: "embedded", Confidence: 1},
						{Scheme: "doi", Value: "10.1/alpha", Source: "sidecar", Confidence: 0.7},
					},
					// Canonical file order is by id: a1 then a2.
					Files: []File{file("batch-file-a2", "alpha/two.epub", 2), file("batch-file-a1", "alpha/one.epub", 1)},
					// Canonical evidence order is field then provider: author, title.
					Evidence: []MetadataEvidence{
						{Field: "title", Provider: "embedded", Confidence: 0.95, ObservedAt: base},
						{Field: "author", Provider: "sidecar", Confidence: 0.6, ObservedAt: base.Add(time.Minute)},
					},
				},
			},
		},
		{
			ID: "batch-work-b", CanonicalTitle: "Batch Beta", CreatedAt: base, UpdatedAt: base.Add(2 * time.Hour),
			Editions: []Edition{{ID: "batch-edition-b1", Title: "Beta One", State: Identified, UpdatedAt: base.Add(2 * time.Hour)}},
		},
		{
			ID: "batch-work-c", CanonicalTitle: "Batch Gamma", CreatedAt: base, UpdatedAt: base.Add(1 * time.Hour),
			Editions: []Edition{{ID: "batch-edition-c1", Title: "Gamma One", State: Pending, UpdatedAt: base.Add(1 * time.Hour)}},
		},
	}
	ids := make([]string, 0, len(works))
	for _, w := range works {
		if err := repo.UpsertWork(ctx, w); err != nil {
			t.Fatalf("seed %s: %v", w.ID, err)
		}
		ids = append(ids, w.ID)
		defer func(id string) { _ = repo.DeleteWork(context.Background(), id) }(w.ID)
	}

	batch, err := repo.getWorks(ctx, ids)
	if err != nil {
		t.Fatal(err)
	}
	// A fixture that loaded nothing would compare equal to an empty result set.
	if len(batch) != len(ids) {
		t.Fatalf("batch loaded %d works, want %d", len(batch), len(ids))
	}
	if batch[0].Editions[0].ID != "batch-edition-a1" || batch[0].Editions[1].ID != "batch-edition-a2" {
		t.Fatalf("edition ordering is not canonical: %s, %s", batch[0].Editions[0].ID, batch[0].Editions[1].ID)
	}
	if batch[0].Editions[0].Identifiers[0].Scheme != "doi" || batch[0].Editions[0].Identifiers[1].Scheme != "isbn" {
		t.Fatalf("identifier ordering is not canonical: %+v", batch[0].Editions[0].Identifiers)
	}
	if batch[0].Editions[0].Evidence[0].Field != "author" || batch[0].Editions[0].Evidence[1].Field != "title" {
		t.Fatalf("evidence ordering is not canonical: %+v", batch[0].Editions[0].Evidence)
	}
	gotFiles := []string{}
	for _, f := range batch[0].Editions[0].Files {
		gotFiles = append(gotFiles, f.ID+"@"+f.Path)
	}
	if strings.Join(gotFiles, ",") != "batch-file-a1@alpha/one.epub,batch-file-a2@alpha/two.epub" {
		t.Fatalf("file ordering is not canonical: %v", gotFiles)
	}

	for i, id := range ids {
		if batch[i].ID != id {
			t.Fatalf("batch order = %s at %d, want %s", batch[i].ID, i, id)
		}
		single, err := repo.GetWork(ctx, id)
		if err != nil {
			t.Fatalf("GetWork(%s): %v", id, err)
		}
		if !reflect.DeepEqual(batch[i], single) {
			t.Fatalf("batch and per-work loaders disagree for %s:\n batch  %+v\n single %+v", id, batch[i], single)
		}
	}
	// Timestamps are compared as instants, not structs: the driver returns them in
	// the session timezone, so a loaded time.Time equals the written one in time
	// but not under DeepEqual.
	for _, f := range batch[0].Editions[0].Files {
		minute := map[string]int{"batch-file-a1": 1, "batch-file-a2": 2}[f.ID]
		if !f.ModifiedAt.Equal(base.Add(time.Duration(minute) * time.Minute)) {
			t.Fatalf("file %s modified_at = %v", f.ID, f.ModifiedAt)
		}
	}
	// Child collections must stay nil when absent, matching the old path exactly.
	if batch[1].Editions[0].Identifiers != nil || batch[1].Editions[0].Evidence != nil {
		t.Fatalf("empty children materialized: %+v", batch[1].Editions[0])
	}
	if batch[2].Editions[0].Files != nil {
		t.Fatalf("edition without files got some: %+v", batch[2].Editions[0].Files)
	}

	page, err := repo.Search(ctx, Query{Text: "Batch", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 3 || len(page.Works) != 3 {
		t.Fatalf("search page = total %d works %d, want 3 and 3", page.Total, len(page.Works))
	}
	// Newest update first, ties broken by id.
	if page.Works[0].ID != "batch-work-a" || page.Works[1].ID != "batch-work-b" || page.Works[2].ID != "batch-work-c" {
		t.Fatalf("search ordering = %s, %s, %s", page.Works[0].ID, page.Works[1].ID, page.Works[2].ID)
	}
	for _, listed := range page.Works {
		single, err := repo.GetWork(ctx, listed.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(listed, single) {
			t.Fatalf("search result %s disagrees with GetWork: %+v", listed.ID, listed)
		}
	}
	if got, err := repo.getWorks(ctx, nil); err != nil || len(got) != 0 {
		t.Fatalf("empty id set = %+v, %v", got, err)
	}
	// An identifier that no longer resolves must be skipped, not an error.
	missing, err := repo.getWorks(ctx, []string{"batch-does-not-exist", "batch-work-a"})
	if err != nil {
		t.Fatal(err)
	}
	if len(missing) != 1 || missing[0].ID != "batch-work-a" {
		t.Fatalf("missing id handling = %+v", missing)
	}
}
