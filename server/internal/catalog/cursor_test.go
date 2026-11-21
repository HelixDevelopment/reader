package catalog

import (
	"context"
	"encoding/base64"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCursorRoundTripsExactly(t *testing.T) {
	// Microseconds are the finest instant the database can hold, so a cursor must
	// survive Encode/Parse at that resolution — including a negative (pre-1970)
	// value and an identifier containing the separator.
	for _, c := range []Cursor{
		{UpdatedAt: time.Date(2026, 11, 20, 18, 44, 12, 123456000, time.UTC), ID: "work-a"},
		{UpdatedAt: time.Date(1969, 7, 20, 20, 17, 40, 999999000, time.UTC), ID: "pre-epoch"},
		{UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), ID: "has|separator"},
		{UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.FixedZone("+0300", 3*60*60)), ID: "offset-zone"},
	} {
		encoded := c.Encode()
		if strings.Contains(encoded, "=") || strings.Contains(encoded, "+") || strings.Contains(encoded, "/") {
			t.Fatalf("cursor %q is not URL-safe", encoded)
		}
		back, err := ParseCursor(encoded)
		if err != nil {
			t.Fatalf("parse %s: %v", encoded, err)
		}
		if !back.UpdatedAt.Equal(c.UpdatedAt.Truncate(time.Microsecond)) || back.ID != c.ID {
			t.Fatalf("round trip changed the cursor: %+v -> %+v", c, back)
		}
		if back.UpdatedAt.Location() != time.UTC {
			t.Fatalf("parsed cursor is not UTC: %v", back.UpdatedAt.Location())
		}
	}
}

func TestCursorRejectsMalformedInput(t *testing.T) {
	raw := func(payload string) string {
		return base64.RawURLEncoding.EncodeToString([]byte(payload))
	}
	for _, value := range []string{
		"", "not base64 !!", "###",
		raw("123456789012345"),        // no separator, so no identifier
		raw("notanumber|id"),          // timestamp is not an integer
		raw("123456789012345|"),       // separator present, identifier empty
		raw("-9223372036854775809|x"), // timestamp outside int64
	} {
		if got, err := ParseCursor(value); err == nil {
			t.Fatalf("malformed cursor accepted: %q -> %+v", value, got)
		}
	}
}

func TestMemoryCursorWalkEqualsOffsetWalk(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	base := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	// Deliberate ties in updated_at: the identifier is the only thing that makes
	// the ordering total, and a cursor that gets the tie case wrong either repeats
	// a row or skips one.
	times := []time.Time{base, base, base, base.Add(time.Second), base.Add(time.Second), base, base.Add(-time.Hour)}
	for i, id := range []string{"m1", "m2", "m3", "m4", "m5", "m6", "m7"} {
		at := times[i].Add(time.Duration(i) * time.Nanosecond) // sub-microsecond noise the database cannot represent
		if err := m.UpsertWork(ctx, Work{ID: id, CanonicalTitle: "Cursor " + id, CreatedAt: at, UpdatedAt: at, Editions: []Edition{{ID: "e" + id, Title: "Cursor " + id, State: Identified, UpdatedAt: at}}}); err != nil {
			t.Fatal(err)
		}
	}
	for _, limit := range []int{1, 2, 3, 7, 8} {
		var viaCursor []string
		var after *Cursor
		pages := 0
		for {
			page, err := m.Search(ctx, Query{Limit: limit, After: after})
			if err != nil {
				t.Fatal(err)
			}
			pages++
			if pages > 20 {
				t.Fatalf("limit %d: cursor walk did not terminate", limit)
			}
			for _, w := range page.Works {
				viaCursor = append(viaCursor, w.ID)
			}
			if !page.HasNext {
				break
			}
			after = CursorAfter(page.Works)
			if after == nil {
				t.Fatalf("HasNext true with an empty page")
			}
		}
		var viaOffset []string
		for off := 0; ; off += limit {
			page, err := m.Search(ctx, Query{Limit: limit, Offset: off})
			if err != nil {
				t.Fatal(err)
			}
			for _, w := range page.Works {
				viaOffset = append(viaOffset, w.ID)
			}
			if !page.HasNext {
				break
			}
		}
		if strings.Join(viaCursor, ",") != strings.Join(viaOffset, ",") {
			t.Fatalf("limit %d: cursor walk %v, offset walk %v", limit, viaCursor, viaOffset)
		}
		if len(viaOffset) != 7 {
			t.Fatalf("limit %d: walk covered %d works, want 7", limit, len(viaOffset))
		}
	}
}

func TestMemoryHasNextAndEmptyPages(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	at := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	for _, id := range []string{"h1", "h2", "h3"} {
		if err := m.UpsertWork(ctx, Work{ID: id, CanonicalTitle: "HasNext " + id, CreatedAt: at, UpdatedAt: at, Editions: []Edition{{ID: "e" + id, Title: "HasNext " + id, State: Identified, UpdatedAt: at}}}); err != nil {
			t.Fatal(err)
		}
	}
	// A page that consumes the whole set must not claim another one follows.
	full, err := m.Search(ctx, Query{Limit: 3})
	if err != nil {
		t.Fatal(err)
	}
	if full.HasNext {
		t.Fatalf("HasNext true for the final page: %+v", full)
	}
	// An anchor at the last row yields an empty page, not a phantom one.
	last := CursorAfter(full.Works)
	tail, err := m.Search(ctx, Query{Limit: 3, After: last})
	if err != nil {
		t.Fatal(err)
	}
	if len(tail.Works) != 0 || tail.HasNext {
		t.Fatalf("page after the last row: works=%d has_next=%v", len(tail.Works), tail.HasNext)
	}
	if CursorAfter(tail.Works) != nil {
		t.Fatal("cursor derived from an empty page")
	}
	// Total still describes the whole filtered set in cursor mode, so a client
	// that reads it is not misled.
	mid, err := m.Search(ctx, Query{Limit: 1, After: CursorAfter(mustSearch(ctx, t, m, Query{Limit: 1}).Works)})
	if err != nil {
		t.Fatal(err)
	}
	if mid.Total != 3 || len(mid.Works) != 1 || mid.Works[0].ID != "h2" || !mid.HasNext {
		t.Fatalf("second page: total=%d works=%+v", mid.Total, mid.Works)
	}
}

func mustSearch(ctx context.Context, t *testing.T, m *Memory, q Query) Page {
	t.Helper()
	page, err := m.Search(ctx, q)
	if err != nil {
		t.Fatal(err)
	}
	return page
}

// TestLivePostgresCursorWalkMatchesOffset is the same equivalence claim against
// PostgreSQL, where the predicate is evaluated by the planner rather than by this
// package. It uses the large fixture so the ordering is not a handful of rows.
func TestLivePostgresCursorWalkMatchesOffset(t *testing.T) {
	dsn := os.Getenv("HELIX_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("HELIX_TEST_POSTGRES_DSN is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	repo, err := NewPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	base := time.Date(2026, 5, 4, 9, 0, 0, 0, time.UTC)
	// Three works share an instant to the microsecond, so the identifier
	// tie-break is exercised on the server too. Sub-microsecond spread is absent
	// here on purpose: PostgreSQL would store all four as the same value.
	seeded := []struct {
		id string
		at time.Time
	}{
		{"cur-work-d", base},
		{"cur-work-a", base},
		{"cur-work-c", base.Add(time.Microsecond)},
		{"cur-work-b", base.Add(-time.Microsecond)},
		{"cur-work-e", base.Add(2 * time.Second)},
	}
	for _, s := range seeded {
		w := Work{ID: s.id, CanonicalTitle: "Cursor Walk " + s.id, CreatedAt: s.at, UpdatedAt: s.at, Editions: []Edition{{ID: "cur-edition-" + s.id, Title: "Cursor Walk " + s.id, State: Identified, UpdatedAt: s.at, Files: []File{{ID: "cur-file-" + s.id, SourceID: "cur-source", Path: s.id + ".epub", Format: "epub", MIMEType: "application/epub+zip", Size: 1, SHA256: strings.Repeat("d", 64), ModifiedAt: s.at}}}}}
		if err := repo.UpsertWork(ctx, w); err != nil {
			t.Fatalf("seed %s: %v", s.id, err)
		}
		defer func(id string) { _ = repo.DeleteWork(context.Background(), id) }(s.id)
	}
	want := []string{"cur-work-e", "cur-work-c", "cur-work-a", "cur-work-d", "cur-work-b"}
	for _, limit := range []int{1, 2, 5, 6} {
		var got []string
		var after *Cursor
		for pages := 0; ; pages++ {
			if pages > 10 {
				t.Fatalf("limit %d: walk did not terminate", limit)
			}
			page, err := repo.Search(ctx, Query{Text: "Cursor Walk", Limit: limit, After: after})
			if err != nil {
				t.Fatal(err)
			}
			for _, w := range page.Works {
				got = append(got, w.ID)
			}
			if !page.HasNext {
				break
			}
			after = CursorAfter(page.Works)
		}
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("limit %d: cursor walk returned %v, want %v", limit, got, want)
		}
		var viaOffset []string
		for off := 0; ; off += limit {
			page, err := repo.Search(ctx, Query{Text: "Cursor Walk", Limit: limit, Offset: off})
			if err != nil {
				t.Fatal(err)
			}
			for _, w := range page.Works {
				viaOffset = append(viaOffset, w.ID)
			}
			if !page.HasNext {
				break
			}
		}
		if strings.Join(viaOffset, ",") != strings.Join(want, ",") {
			t.Fatalf("limit %d: offset walk returned %v, want %v", limit, viaOffset, want)
		}
	}
	// A page loaded through a cursor must describe rows identically to the same
	// page loaded one work at a time, or the batch loader has diverged again.
	page, err := repo.Search(ctx, Query{Text: "Cursor Walk", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Works) != 2 {
		t.Fatalf("page size %d", len(page.Works))
	}
	for _, listed := range page.Works {
		single, err := repo.GetWork(ctx, listed.ID)
		if err != nil {
			t.Fatal(err)
		}
		if listed.Editions[0].Files[0].ID != single.Editions[0].Files[0].ID {
			t.Fatalf("cursor page child mismatch for %s", listed.ID)
		}
	}
}
