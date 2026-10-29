package catalog

import (
	"strings"
	"testing"
	"time"
)

func sha(c byte) string { return strings.Repeat(string(c), 64) }

func TestMergeWorkDedupsEvidenceAndIsIdempotent(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ev := MetadataEvidence{Field: "title", Provider: "embedded", Confidence: 0.9, ObservedAt: now}
	existing := Work{ID: "w", CanonicalTitle: "Title", UpdatedAt: now, Editions: []Edition{{ID: "e", Title: "Title", State: Identified, UpdatedAt: now, Evidence: []MetadataEvidence{ev}, Files: []File{{ID: "f1", SHA256: sha('a')}}}}}
	// Incoming carries the same observation twice plus one new observation;
	// exact duplicates must collapse instead of growing the evidence log.
	dup := ev
	newer := MetadataEvidence{Field: "title", Provider: "openlibrary", Confidence: 0.8, ObservedAt: now.Add(time.Hour)}
	incoming := Work{ID: "w", UpdatedAt: now.Add(time.Hour), Editions: []Edition{{ID: "e", Title: "Title — Revised", State: Identified, UpdatedAt: now.Add(time.Hour), Evidence: []MetadataEvidence{ev, dup, newer}, Files: []File{{ID: "f2", SHA256: sha('b')}}}}}
	merged := MergeWork(existing, incoming)
	if got := len(merged.Editions[0].Evidence); got != 2 {
		t.Fatalf("duplicate evidence not deduped: %+v", merged.Editions[0].Evidence)
	}
	// Repeat merges must be idempotent for evidence, files and identifiers.
	for i := 0; i < 3; i++ {
		merged = MergeWork(merged, incoming)
	}
	e := merged.Editions[0]
	if len(e.Evidence) != 2 || len(e.Files) != 2 {
		t.Fatalf("repeat merge grew state: evidence=%d files=%d", len(e.Evidence), len(e.Files))
	}
	if merged.Editions[0].Evidence[0].Confidence != 0.9 {
		t.Fatalf("existing confidence must win on duplicate observation: %+v", merged.Editions[0].Evidence)
	}
}

func TestMergeWorkDoesNotAliasCallerSlices(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	existing := Work{ID: "w", CanonicalTitle: "Title", UpdatedAt: now, Editions: []Edition{{ID: "e", Title: "Title", State: Identified, UpdatedAt: now}}}
	incoming := Work{ID: "w", UpdatedAt: now.Add(time.Hour), Editions: []Edition{{
		ID: "e", Title: "Title — Revised", State: Identified, UpdatedAt: now.Add(time.Hour),
		Authors:  []string{"Original Author"},
		Evidence: []MetadataEvidence{{Field: "title", Provider: "embedded", Confidence: 0.9, ObservedAt: now}},
		Files:    []File{{ID: "f1", SHA256: sha('a')}},
	}}}
	merged := MergeWork(existing, incoming)
	// Mutating the caller's slices after the merge must not leak into the
	// merged result.
	incoming.Editions[0].Authors[0] = "MUTATED"
	incoming.Editions[0].Evidence[0].Confidence = 0.01
	incoming.Editions[0].Files[0].SHA256 = sha('z')
	e := merged.Editions[0]
	if e.Authors[0] != "Original Author" || e.Evidence[0].Confidence != 0.9 || e.Files[0].SHA256 != sha('a') {
		t.Fatalf("merge aliased caller slices: %+v", e)
	}
	// Mutating the existing work afterwards must not leak either.
	existing.Editions[0].Title = "MUTATED"
	if merged.Editions[0].Title == "MUTATED" {
		t.Fatal("merge aliased existing edition")
	}
}

func TestMergeWorkUnionsEditionsFilesIdentifiers(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a := Work{ID: "w", CanonicalTitle: "Title", UpdatedAt: now, Editions: []Edition{{ID: "e", Title: "Title", UpdatedAt: now, Identifiers: []Identifier{{Scheme: "isbn", Value: "9780441478125"}}, Files: []File{{ID: "f1"}}}}}
	b := Work{ID: "w", UpdatedAt: now.Add(time.Hour), Subjects: []string{"Fiction"}, Editions: []Edition{{ID: "e", Title: "Title — Revised", UpdatedAt: now.Add(time.Hour), Identifiers: []Identifier{{Scheme: "isbn", Value: "9780441478125"}, {Scheme: "oclc", Value: "123"}}, Files: []File{{ID: "f2"}}}, {ID: "e2", Title: "Translation", UpdatedAt: now}}}
	got := MergeWork(a, b)
	if len(got.Editions) != 2 || len(got.Editions[0].Files) != 2 || len(got.Editions[0].Identifiers) != 2 {
		t.Fatalf("merge lost data: %+v", got)
	}
	if got.Editions[0].Title != "Title — Revised" {
		t.Fatalf("newer edition did not win: %+v", got.Editions[0])
	}
	again := MergeWork(got, b)
	if len(again.Editions[0].Files) != 2 || len(again.Editions[0].Identifiers) != 2 {
		t.Fatal("merge is not idempotent")
	}
}

func TestMergeWorkReplacesChangedFileAtSameSourcePath(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	existing := Work{ID: "w", Editions: []Edition{{ID: "e", Files: []File{{ID: "file-stable", SourceID: "src", Path: "book.txt", Size: 3, SHA256: sha('a'), ModifiedAt: now}}}}}
	incoming := Work{ID: "w", Editions: []Edition{{ID: "e", Files: []File{{ID: "file-stable", SourceID: "src", Path: "book.txt", Size: 5, SHA256: sha('b'), ModifiedAt: now.Add(time.Hour)}}}}}
	got := MergeWork(existing, incoming)
	files := got.Editions[0].Files
	if len(files) != 1 || files[0].SHA256 != sha('b') || files[0].Size != 5 {
		t.Fatalf("changed file did not replace stale byte evidence: %+v", files)
	}
}
